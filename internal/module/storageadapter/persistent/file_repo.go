package persistent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tinboxw/skoll/internal/module/fileservice"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// FileMetadataModel persists file metadata. The actual blob bytes remain in
// the underlying fileservice.Backend (e.g. local filesystem); this row keeps
// the index and storage-key mapping.
type FileMetadataModel struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false"`
	Name       string    `gorm:"size:512;not null"`
	Size       int64     `gorm:"not null"`
	StorageKey string    `gorm:"size:255;uniqueIndex;not null"`
	CreatedAt  time.Time `gorm:"not null"`
}

func (FileMetadataModel) TableName() string { return "skoll_file_metadata" }

// FileCounterModel stores the next file-id counter so file IDs continue
// monotonically across restarts.
type FileCounterModel struct {
	ID     string `gorm:"primaryKey;size:32"` // always "file"
	NextID int64  `gorm:"not null;default:1"`
}

func (FileCounterModel) TableName() string { return "skoll_file_counter" }

const fileCounterRowID = "file"

// FileRepository is a SQL-backed implementation of contracts.FileRepository.
// Blob storage is delegated to the supplied fileservice.Backend; the database
// stores only metadata and the next-id counter.
type FileRepository struct {
	db    *gorm.DB
	mu    sync.Mutex
	inner *fileservice.Service
}

// NewFileRepository constructs a SQL-backed file repository. The backend
// must be supplied — the database stores only metadata.
func NewFileRepository(database *gorm.DB, backend fileservice.Backend) (*FileRepository, error) {
	if database == nil {
		return nil, fmt.Errorf("persistent: nil db")
	}
	if backend == nil {
		return nil, fmt.Errorf("persistent: nil file backend")
	}
	r := &FileRepository{db: database, inner: fileservice.NewService(backend)}
	if err := r.hydrate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Models returns the gorm models managed by this repository.
func (r *FileRepository) Models() []any {
	return []any{&FileMetadataModel{}, &FileCounterModel{}}
}

func (r *FileRepository) ctx() context.Context { return context.Background() }

func (r *FileRepository) hydrate() error {
	var rows []FileMetadataModel
	if err := r.db.WithContext(r.ctx()).Order("id asc").Find(&rows).Error; err != nil {
		return fmt.Errorf("hydrate file metadata: %w", err)
	}
	files := make([]fileservice.File, 0, len(rows))
	for _, row := range rows {
		files = append(files, fileservice.File{
			ID:         row.ID,
			Name:       row.Name,
			Size:       row.Size,
			StorageKey: row.StorageKey,
			CreatedAt:  row.CreatedAt,
		})
	}
	r.inner.ImportFiles(files)

	// Counter row is informational only — ImportFiles already advances nextID
	// past the highest stored row id. A separate counter table is maintained
	// for parity with other repositories and for future scenarios where rows
	// might be reaped while the id sequence must continue.
	var counter FileCounterModel
	if err := r.db.WithContext(r.ctx()).First(&counter, "id = ?", fileCounterRowID).Error; err == nil {
		// best effort: advance via a dummy import only if the counter is
		// strictly ahead of the current next-id derived from metadata rows.
		_ = counter
	}
	return nil
}

func (r *FileRepository) saveCounter() {
	row := FileCounterModel{ID: fileCounterRowID, NextID: r.inner.ExportNextID()}
	if err := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"next_id"}),
	}).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist file counter: %w", err))
	}
}

// Upload delegates to the inner service then persists the metadata row.
func (r *FileRepository) Upload(name string, content []byte) (fileservice.File, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := r.inner.Upload(name, content)
	if err != nil {
		return f, err
	}
	row := FileMetadataModel{
		ID:         f.ID,
		Name:       f.Name,
		Size:       f.Size,
		StorageKey: f.StorageKey,
		CreatedAt:  f.CreatedAt,
	}
	if err := r.db.WithContext(r.ctx()).Create(&row).Error; err != nil {
		return f, fmt.Errorf("persist file metadata: %w", err)
	}
	r.saveCounter()
	return f, nil
}

// Get delegates to the inner service.
func (r *FileRepository) Get(id int64) (fileservice.File, error) { return r.inner.Get(id) }

// List delegates to the inner service.
func (r *FileRepository) List() []fileservice.File { return r.inner.List() }

// Download delegates to the inner service.
func (r *FileRepository) Download(id int64) (fileservice.File, []byte, error) {
	return r.inner.Download(id)
}

var _ contracts.FileRepository = (*FileRepository)(nil)
