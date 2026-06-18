package gormrepo

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/menu"
	"github.com/tinboxw/skoll/internal/domain/permission"
)

type PermissionResourceModel struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	PermissionKey string `gorm:"size:128;uniqueIndex"`
	ResourceType  string `gorm:"size:32;index:idx_permission_resources_type;index:idx_permission_resources_module_type;index:idx_permission_resources_source_type"`
	Module        string `gorm:"size:64;index:idx_permission_resources_module_type"`
	Source        string `gorm:"size:64;index:idx_permission_resources_source_type"`
	Name          string `gorm:"size:128"`
	Risk          string `gorm:"size:32;index:idx_permission_resources_risk"`
	MetadataJSON  string `gorm:"type:text"`
	Enabled       bool   `gorm:"index:idx_permission_resources_enabled"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (PermissionResourceModel) TableName() string { return "sk_permission_resources" }

func PermissionResourceModelFromDomain(resource permission.PermissionResource) PermissionResourceModel {
	metadata, _ := json.Marshal(resource.Metadata)
	return PermissionResourceModel{
		PermissionKey: strings.TrimSpace(resource.Key()),
		ResourceType:  string(resource.Type()),
		Module:        strings.TrimSpace(resource.Module()),
		Source:        strings.TrimSpace(resource.Source()),
		Name:          strings.TrimSpace(resource.Name),
		Risk:          string(resource.Risk),
		MetadataJSON:  string(metadata),
		Enabled:       resource.Enabled,
	}
}

func (m PermissionResourceModel) ToDomain() (permission.PermissionResource, error) {
	metadata := map[string]string{}
	if strings.TrimSpace(m.MetadataJSON) != "" {
		if err := json.Unmarshal([]byte(m.MetadataJSON), &metadata); err != nil {
			return permission.PermissionResource{}, err
		}
	}
	resource, err := permission.NewResourceWithMetadata(permission.ResourceIdentity{
		Key:    strings.TrimSpace(m.PermissionKey),
		Type:   permission.ResourceType(strings.TrimSpace(m.ResourceType)),
		Module: strings.TrimSpace(m.Module),
		Source: strings.TrimSpace(m.Source),
	}, strings.TrimSpace(m.Name), permission.RiskLevel(strings.TrimSpace(m.Risk)), metadata)
	if err != nil {
		return permission.PermissionResource{}, err
	}
	resource.Enabled = m.Enabled
	return resource, nil
}

type MenuNodeModel struct {
	ID                      uint64 `gorm:"primaryKey;autoIncrement"`
	MenuKey                 string `gorm:"size:128;uniqueIndex"`
	ParentKey               string `gorm:"size:128;index:idx_menu_nodes_parent_sort"`
	Source                  string `gorm:"size:64;index:idx_menu_nodes_source"`
	Name                    string `gorm:"size:128"`
	Path                    string `gorm:"size:256;index:idx_menu_nodes_path"`
	Component               string `gorm:"size:256"`
	Icon                    string `gorm:"size:128"`
	Sort                    int    `gorm:"index:idx_menu_nodes_parent_sort;index:idx_menu_nodes_sort"`
	Visible                 bool   `gorm:"index:idx_menu_nodes_visible"`
	RequiredRolesJSON       string `gorm:"type:text"`
	RequiredPermissionsJSON string `gorm:"type:text"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (MenuNodeModel) TableName() string { return "sk_menu_nodes" }

func MenuNodeModelFromDomain(node menu.MenuNode) MenuNodeModel {
	roles, _ := json.Marshal(node.RequiredRoles)
	permissions, _ := json.Marshal(node.RequiredPermissions)
	return MenuNodeModel{
		MenuKey:                 strings.TrimSpace(node.Key()),
		ParentKey:               strings.TrimSpace(node.ParentKey()),
		Source:                  strings.TrimSpace(node.Source()),
		Name:                    strings.TrimSpace(node.Name()),
		Path:                    strings.TrimSpace(node.Path()),
		Component:               strings.TrimSpace(node.Component()),
		Icon:                    strings.TrimSpace(node.Icon()),
		Sort:                    node.Sort,
		Visible:                 node.Visible,
		RequiredRolesJSON:       string(roles),
		RequiredPermissionsJSON: string(permissions),
	}
}

func (m MenuNodeModel) ToDomain() (menu.MenuNode, error) {
	roles := []string{}
	permissions := []string{}
	if strings.TrimSpace(m.RequiredRolesJSON) != "" {
		if err := json.Unmarshal([]byte(m.RequiredRolesJSON), &roles); err != nil {
			return menu.MenuNode{}, err
		}
	}
	if strings.TrimSpace(m.RequiredPermissionsJSON) != "" {
		if err := json.Unmarshal([]byte(m.RequiredPermissionsJSON), &permissions); err != nil {
			return menu.MenuNode{}, err
		}
	}
	node, err := menu.NewNode(menu.NodeIdentity{
		Key:       strings.TrimSpace(m.MenuKey),
		ParentKey: strings.TrimSpace(m.ParentKey),
		Source:    strings.TrimSpace(m.Source),
	}, menu.NodeView{
		Name:      strings.TrimSpace(m.Name),
		Path:      strings.TrimSpace(m.Path),
		Component: strings.TrimSpace(m.Component),
		Icon:      strings.TrimSpace(m.Icon),
	}, m.Sort)
	if err != nil {
		return menu.MenuNode{}, err
	}
	node.Visible = m.Visible
	node.RequiredRoles = roles
	node.RequiredPermissions = permissions
	return node, nil
}
