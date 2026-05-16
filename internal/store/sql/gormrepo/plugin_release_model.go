package gormrepo

import "time"

type PluginReleaseModel struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	PluginID       string `gorm:"size:128;index:idx_plugin_release"`
	ReleaseVersion string `gorm:"size:64;index:idx_plugin_release"`
	PackageName    string `gorm:"size:256"`
	PackageHash    string `gorm:"size:128"`
	PackageSize    uint64
	Signature      string `gorm:"type:text"`
	Changelog      string `gorm:"type:text"`
	UploadedAt     time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (PluginReleaseModel) TableName() string { return "sk_plugin_releases" }
