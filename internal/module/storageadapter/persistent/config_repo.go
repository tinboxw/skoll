package persistent

import (
	"context"
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/tinboxw/skoll/internal/module/config"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// ConfigEntryModel is the gorm representation of config.Entry.
// The primary key column is named cfg_key to sidestep `key` being a reserved
// word across mysql/postgres while keeping a consistent identifier.
type ConfigEntryModel struct {
	Key         string `gorm:"primaryKey;column:cfg_key;size:191"`
	Value       string `gorm:"column:cfg_value;type:text"`
	Description string `gorm:"column:description;size:512"`
	UpdatedAt   time.Time
}

func (ConfigEntryModel) TableName() string { return "skoll_configs" }

func (m ConfigEntryModel) toDomain() config.Entry {
	return config.Entry{
		Key:         m.Key,
		Value:       m.Value,
		Description: m.Description,
		UpdatedAt:   m.UpdatedAt.UTC(),
	}
}

// ConfigRepository is a SQL-backed implementation of contracts.ConfigRepository.
type ConfigRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewConfigRepository constructs a SQL-backed config repository.
func NewConfigRepository(db *gorm.DB) *ConfigRepository {
	return &ConfigRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

func (r *ConfigRepository) ctx() context.Context { return context.Background() }

// Models returns gorm models owned by this repo for migration registration.
func (r *ConfigRepository) Models() []any { return []any{&ConfigEntryModel{}} }

// Set inserts or updates a config entry by key and returns the stored value.
func (r *ConfigRepository) Set(key, value, description string) config.Entry {
	row := ConfigEntryModel{
		Key:         key,
		Value:       value,
		Description: description,
		UpdatedAt:   r.now(),
	}
	// Upsert: rely on gorm Save which does INSERT...ON CONFLICT for known PKs
	// across mysql/postgres/sqlite via Save semantics.
	if err := r.db.WithContext(r.ctx()).Save(&row).Error; err != nil {
		panic(err)
	}
	return row.toDomain()
}

// Get returns the entry for the supplied key.
func (r *ConfigRepository) Get(key string) (config.Entry, error) {
	var row ConfigEntryModel
	if err := r.db.WithContext(r.ctx()).First(&row, "cfg_key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return config.Entry{}, config.ErrConfigNotFound
		}
		return config.Entry{}, err
	}
	return row.toDomain(), nil
}

// List returns all config entries sorted by key ascending.
func (r *ConfigRepository) List() []config.Entry {
	var rows []ConfigEntryModel
	if err := r.db.WithContext(r.ctx()).Find(&rows).Error; err != nil {
		panic(err)
	}
	out := make([]config.Entry, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Compile-time interface check.
var _ contracts.ConfigRepository = (*ConfigRepository)(nil)
