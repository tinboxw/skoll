package model

import (
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
)

type SystemSettingModel struct {
	ID        string `gorm:"primaryKey;size:128"`
	Key       string `gorm:"size:128;uniqueIndex"`
	Value     string `gorm:"type:text"`
	Encrypted bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (SystemSettingModel) TableName() string { return "sk_system_settings" }

func SystemSettingModelFromDomain(setting *domainsystem.Setting, normalizeKey Normalizer) SystemSettingModel {
	return SystemSettingModel{
		ID:        setting.ID.String(),
		Key:       normalizeKey(setting.Key),
		Value:     setting.Value,
		Encrypted: setting.Encrypted,
		CreatedAt: setting.Meta.CreatedAt,
		UpdatedAt: setting.Meta.UpdatedAt,
	}
}

func (m SystemSettingModel) ToDomain(normalizeKey Normalizer) *domainsystem.Setting {
	return &domainsystem.Setting{
		ID:        shared.ID(m.ID),
		Key:       normalizeKey(m.Key),
		Value:     m.Value,
		Encrypted: m.Encrypted,
		Meta: shared.AuditMeta{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	}
}
