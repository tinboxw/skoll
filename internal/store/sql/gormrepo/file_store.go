package gormrepo

import (
	"context"
	"strings"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FileStore struct {
	db *gorm.DB
}

func NewFileStore(db *gorm.DB) *FileStore {
	return &FileStore{db: db}
}

func (s *FileStore) Upsert(ctx context.Context, object *domainfile.FileObject) error {
	if object == nil {
		return nil
	}
	row := FileObjectModelFromDomain(*object)
	if strings.TrimSpace(row.ID) == "" {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"object_key",
				"name",
				"size_bytes",
				"mime",
				"hash",
				"owner_type",
				"owner_id",
				"visibility",
				"storage_driver",
				"status",
				"source_module",
				"source_plugin_id",
				"metadata_json",
				"updated_at",
			}),
		}).Create(&row).Error
	})
}

func (s *FileStore) Get(ctx context.Context, id shared.ID) (*domainfile.FileObject, error) {
	if id.IsZero() {
		return nil, nil
	}
	var row FileObjectModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("id = ?", id.String()).First(&row).Error
	})
	return fileObjectFromRow(row, err)
}

func (s *FileStore) GetByKey(ctx context.Context, key string) (*domainfile.FileObject, error) {
	key = domainfile.NormalizeKey(key)
	if key == "" {
		return nil, nil
	}
	var row FileObjectModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("object_key = ?", key).First(&row).Error
	})
	return fileObjectFromRow(row, err)
}

func (s *FileStore) List(ctx context.Context, filter filerepo.ListFilter, offset, limit int) ([]domainfile.FileObject, error) {
	q := s.db.WithContext(ctx).Model(&FileObjectModel{}).Order("object_key asc")
	if strings.TrimSpace(filter.OwnerType) != "" {
		q = q.Where("owner_type = ?", strings.TrimSpace(strings.ToLower(filter.OwnerType)))
	}
	if !filter.OwnerID.IsZero() {
		q = q.Where("owner_id = ?", filter.OwnerID.String())
	}
	if filter.Visibility != "" {
		q = q.Where("visibility = ?", string(filter.Visibility))
	}
	if strings.TrimSpace(filter.StorageDriver) != "" {
		q = q.Where("storage_driver = ?", strings.TrimSpace(strings.ToLower(filter.StorageDriver)))
	}
	if filter.Status != "" {
		q = q.Where("status = ?", string(filter.Status))
	}
	if strings.TrimSpace(filter.SourceModule) != "" {
		q = q.Where("source_module = ?", strings.TrimSpace(strings.ToLower(filter.SourceModule)))
	}
	if strings.TrimSpace(filter.SourcePluginID) != "" {
		q = q.Where("source_plugin_id = ?", strings.TrimSpace(strings.ToLower(filter.SourcePluginID)))
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}

	var rows []FileObjectModel
	if err := withDBRetry(func() error { return q.Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]domainfile.FileObject, 0, len(rows))
	for _, row := range rows {
		object, err := row.ToDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, *object)
	}
	return out, nil
}

func (s *FileStore) SetStatus(ctx context.Context, id shared.ID, status domainfile.Status, now time.Time) error {
	if id.IsZero() {
		return nil
	}
	if err := domainfile.ValidateStatus(status); err != nil {
		return err
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).
			Model(&FileObjectModel{}).
			Where("id = ?", id.String()).
			Updates(map[string]any{"status": string(status), "updated_at": now}).Error
	})
}

func (s *FileStore) Delete(ctx context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Delete(&FileObjectModel{}, "id = ?", id.String()).Error
	})
}

func fileObjectFromRow(row FileObjectModel, err error) (*domainfile.FileObject, error) {
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row.ToDomain()
}
