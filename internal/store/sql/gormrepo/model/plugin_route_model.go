package model

import "time"

type PluginRouteModel struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	PluginID      string `gorm:"size:128;index:idx_plugin_route"`
	RouteType     string `gorm:"size:32;index:idx_plugin_route"`
	RoutePath     string `gorm:"size:512"`
	OpenMode      string `gorm:"size:32"`
	PermissionKey string `gorm:"size:128"`
	IsEnabled     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (PluginRouteModel) TableName() string { return "sk_plugin_routes" }
