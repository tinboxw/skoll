package gormrepo

import (
	"context"
	"strings"

	"github.com/tinboxw/skoll/internal/plugin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PluginStore struct {
	db *gorm.DB
}

func NewPluginStore(db *gorm.DB) *PluginStore {
	return &PluginStore{db: db}
}

func (s *PluginStore) Get(ctx context.Context, pluginID string) (*plugin.Info, error) {
	key := strings.TrimSpace(pluginID)
	if key == "" {
		return nil, nil
	}
	var row PluginModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("plugin_id = ?", key).First(&row).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	info := row.ToInfo()
	return &info, nil
}

func (s *PluginStore) List(ctx context.Context) ([]plugin.Info, error) {
	var rows []PluginModel
	if err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Order("plugin_id asc").Find(&rows).Error
	}); err != nil {
		return nil, err
	}
	out := make([]plugin.Info, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToInfo())
	}
	return out, nil
}

func (s *PluginStore) Save(ctx context.Context, info plugin.Info) error {
	row := PluginModelFromInfo(info)
	if strings.TrimSpace(row.PluginID) == "" {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "plugin_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "version", "migration_version", "description", "config_json", "data_manifest_json", "state", "source", "ui_mode", "plugin_level", "app_id", "mount_policy", "frontend_entry", "system_builtin", "host_capabilities_json", "permissions_json", "dependencies_json", "vendor", "vendor_url", "signature_json", "installed_at", "enabled_at", "updated_at"}),
		}).Create(&row).Error
	})
}

func (s *PluginStore) Delete(ctx context.Context, pluginID string) error {
	key := strings.TrimSpace(pluginID)
	if key == "" {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Delete(&PluginModel{}, "plugin_id = ?", key).Error
	})
}
