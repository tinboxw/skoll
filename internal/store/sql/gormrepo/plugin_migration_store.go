package gormrepo

import (
	"context"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
	"gorm.io/gorm"
)

type PluginMigrationModel struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	PluginID  string    `gorm:"size:128;uniqueIndex:idx_plugin_migration_version,priority:1"`
	Version   int       `gorm:"uniqueIndex:idx_plugin_migration_version,priority:2"`
	Name      string    `gorm:"size:255"`
	Checksum  string    `gorm:"size:64"`
	AppliedAt time.Time `gorm:"index"`
}

func (PluginMigrationModel) TableName() string { return "sk_plugin_migrations" }

type PluginMigrationStore struct {
	db *gorm.DB
}

type pluginMigrationTransaction struct {
	db *gorm.DB
}

func NewPluginMigrationStore(db *gorm.DB) *PluginMigrationStore {
	return &PluginMigrationStore{db: db}
}

func (s *PluginMigrationStore) ListApplied(ctx context.Context, pluginID string) ([]plugin.MigrationRecord, error) {
	models := make([]PluginMigrationModel, 0)
	if err := s.db.WithContext(ctx).
		Where("plugin_id = ?", strings.TrimSpace(pluginID)).
		Order("version ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	records := make([]plugin.MigrationRecord, 0, len(models))
	for _, model := range models {
		records = append(records, plugin.MigrationRecord{
			PluginID:  model.PluginID,
			Version:   model.Version,
			Name:      model.Name,
			Checksum:  model.Checksum,
			AppliedAt: model.AppliedAt,
		})
	}
	return records, nil
}

func (s *PluginMigrationStore) WithTransaction(ctx context.Context, fn func(tx plugin.MigrationTransaction) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(pluginMigrationTransaction{db: tx})
	})
}

func (s *PluginMigrationStore) RequiresApplyCompensation() bool {
	return s != nil && s.db != nil && s.db.Dialector.Name() == "mysql"
}

func (tx pluginMigrationTransaction) ExecSQL(sql string) error {
	return tx.db.Exec(sql).Error
}

func (tx pluginMigrationTransaction) MarkApplied(record plugin.MigrationRecord) error {
	return tx.db.Create(&PluginMigrationModel{
		PluginID:  strings.TrimSpace(record.PluginID),
		Version:   record.Version,
		Name:      strings.TrimSpace(record.Name),
		Checksum:  strings.TrimSpace(record.Checksum),
		AppliedAt: record.AppliedAt.UTC(),
	}).Error
}

func (tx pluginMigrationTransaction) RemoveApplied(pluginID string, version int) error {
	return tx.db.
		Where("plugin_id = ? AND version = ?", strings.TrimSpace(pluginID), version).
		Delete(&PluginMigrationModel{}).Error
}

var _ plugin.MigrationStore = (*PluginMigrationStore)(nil)
var _ plugin.MigrationTransaction = pluginMigrationTransaction{}
