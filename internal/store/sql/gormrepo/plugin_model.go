package gormrepo

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
)

type PluginModel struct {
	ID               uint64 `gorm:"primaryKey;autoIncrement"`
	PluginID         string `gorm:"size:128;uniqueIndex"`
	Name             string `gorm:"size:128"`
	Version          string `gorm:"size:64"`
	Description      string `gorm:"size:512"`
	ConfigJSON       string `gorm:"type:text"`
	State            string `gorm:"size:32;index"`
	Source           string `gorm:"size:512"`
	UIMode           string `gorm:"size:64"`
	PluginLevel      string `gorm:"size:32"`
	AppID            string `gorm:"size:128"`
	MountPolicy      string `gorm:"size:32"`
	FrontendEntry    string `gorm:"size:512"`
	SystemBuiltin    bool
	PermissionsJSON  string `gorm:"type:text"`
	DependenciesJSON string `gorm:"type:text"`
	Vendor           string `gorm:"size:128"`
	VendorURL        string `gorm:"size:512"`
	SignatureJSON    string `gorm:"type:text"`
	InstalledAt      time.Time
	EnabledAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (PluginModel) TableName() string { return "sk_plugins" }

func PluginModelFromInfo(info plugin.Info) PluginModel {
	permissions, _ := json.Marshal(info.Permissions)
	dependencies, _ := json.Marshal(info.Dependencies)
	signature, _ := json.Marshal(info.Signature)
	return PluginModel{
		PluginID:         strings.TrimSpace(info.ID),
		Name:             strings.TrimSpace(info.Name),
		Version:          strings.TrimSpace(info.Version),
		Description:      strings.TrimSpace(info.Description),
		ConfigJSON:       strings.TrimSpace(info.ConfigJSON),
		State:            string(info.State),
		Source:           strings.TrimSpace(info.Source),
		UIMode:           string(info.UIMode),
		PluginLevel:      string(info.Level),
		AppID:            strings.TrimSpace(info.AppID),
		MountPolicy:      string(info.MountPolicy),
		FrontendEntry:    strings.TrimSpace(info.FrontendEntry),
		SystemBuiltin:    info.SystemBuiltin,
		PermissionsJSON:  string(permissions),
		DependenciesJSON: string(dependencies),
		Vendor:           strings.TrimSpace(info.Vendor),
		VendorURL:        strings.TrimSpace(info.VendorURL),
		SignatureJSON:    string(signature),
		InstalledAt:      info.InstalledAt,
		EnabledAt:        info.EnabledAt,
	}
}

func (m PluginModel) ToInfo() plugin.Info {
	permissions := []string{}
	dependencies := []plugin.Dependency{}
	signature := (*plugin.Signature)(nil)

	if strings.TrimSpace(m.PermissionsJSON) != "" {
		_ = json.Unmarshal([]byte(m.PermissionsJSON), &permissions)
	}
	if strings.TrimSpace(m.DependenciesJSON) != "" {
		_ = json.Unmarshal([]byte(m.DependenciesJSON), &dependencies)
	}
	if strings.TrimSpace(m.SignatureJSON) != "" {
		sig := &plugin.Signature{}
		if err := json.Unmarshal([]byte(m.SignatureJSON), sig); err == nil {
			signature = sig
		}
	}

	return plugin.Info{
		ID:            strings.TrimSpace(m.PluginID),
		Name:          strings.TrimSpace(m.Name),
		Version:       strings.TrimSpace(m.Version),
		Description:   strings.TrimSpace(m.Description),
		ConfigJSON:    strings.TrimSpace(m.ConfigJSON),
		Dependencies:  dependencies,
		Permissions:   permissions,
		State:         plugin.State(strings.TrimSpace(m.State)),
		InstalledAt:   m.InstalledAt,
		EnabledAt:     m.EnabledAt,
		Source:        strings.TrimSpace(m.Source),
		UIMode:        plugin.UIMode(strings.TrimSpace(m.UIMode)),
		Level:         plugin.Level(strings.TrimSpace(m.PluginLevel)),
		AppID:         strings.TrimSpace(m.AppID),
		MountPolicy:   plugin.MountPolicy(strings.TrimSpace(m.MountPolicy)),
		FrontendEntry: strings.TrimSpace(m.FrontendEntry),
		SystemBuiltin: m.SystemBuiltin,
		Vendor:        strings.TrimSpace(m.Vendor),
		VendorURL:     strings.TrimSpace(m.VendorURL),
		Signature:     signature,
	}
}
