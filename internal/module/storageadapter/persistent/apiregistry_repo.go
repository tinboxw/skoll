package persistent

import (
	"context"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// APIRegistryEntryModel is the gorm representation of an api registry entry.
// Entries are normalized via apiregistry.Normalize before storage.
type APIRegistryEntryModel struct {
	Entry string `gorm:"primaryKey;size:255"`
}

func (APIRegistryEntryModel) TableName() string { return "skoll_api_registry" }

// APIRegistryRepository is a SQL-backed implementation of contracts.APIRegistryRepository.
type APIRegistryRepository struct {
	db *gorm.DB
}

// NewAPIRegistryRepository constructs a SQL-backed api registry repository.
func NewAPIRegistryRepository(db *gorm.DB) *APIRegistryRepository {
	return &APIRegistryRepository{db: db}
}

func (r *APIRegistryRepository) ctx() context.Context { return context.Background() }

// Models returns gorm models owned by this repo for migration registration.
func (r *APIRegistryRepository) Models() []any { return []any{&APIRegistryEntryModel{}} }

// RegisterMany inserts the supplied entries idempotently after normalization.
func (r *APIRegistryRepository) RegisterMany(entries []string) {
	rows := make([]APIRegistryEntryModel, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, raw := range entries {
		normalized := apiregistry.Normalize(raw)
		if normalized == "" {
			continue
		}
		if _, dup := seen[normalized]; dup {
			continue
		}
		seen[normalized] = struct{}{}
		rows = append(rows, APIRegistryEntryModel{Entry: normalized})
	}
	if len(rows) == 0 {
		return
	}
	// ON CONFLICT DO NOTHING for idempotent re-registration. gorm clause.OnConflict
	// is supported uniformly by the mysql/postgres/sqlite drivers in use.
	tx := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	if err := tx.Error; err != nil {
		panic(err)
	}
}

// Exists reports whether the supplied entry has been registered.
func (r *APIRegistryRepository) Exists(entry string) bool {
	normalized := apiregistry.Normalize(entry)
	if normalized == "" {
		return false
	}
	var count int64
	if err := r.db.WithContext(r.ctx()).Model(&APIRegistryEntryModel{}).
		Where("entry = ?", normalized).Count(&count).Error; err != nil {
		panic(err)
	}
	return count > 0
}

// List returns all registered entries sorted ascending.
func (r *APIRegistryRepository) List() []string {
	var rows []APIRegistryEntryModel
	if err := r.db.WithContext(r.ctx()).Find(&rows).Error; err != nil {
		panic(err)
	}
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.Entry
	}
	sort.Strings(out)
	return out
}

// Compile-time interface check.
var _ contracts.APIRegistryRepository = (*APIRegistryRepository)(nil)
