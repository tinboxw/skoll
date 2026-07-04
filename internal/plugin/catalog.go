package plugin

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
)

type CatalogRegistry interface {
	ImportPlugin(info Info) error
	DisablePlugin(pluginID string) error
	RemovePlugin(pluginID string) error
}

type MemoryCatalogRegistry struct {
	mu          sync.RWMutex
	permissions map[string]domainpermission.PermissionResource
	menus       map[string]domainmenu.MenuNode
	byPlugin    map[string]catalogKeys
}

type catalogKeys struct {
	permissions []string
	menus       []string
}

func NewMemoryCatalogRegistry() *MemoryCatalogRegistry {
	return &MemoryCatalogRegistry{
		permissions: make(map[string]domainpermission.PermissionResource),
		menus:       make(map[string]domainmenu.MenuNode),
		byPlugin:    make(map[string]catalogKeys),
	}
}

func (r *MemoryCatalogRegistry) ImportPlugin(info Info) error {
	if r == nil {
		return nil
	}
	pluginID := strings.TrimSpace(strings.ToLower(info.ID))
	if pluginID == "" {
		return ErrPluginManifestBroken
	}
	permissions, err := info.CatalogPermissions()
	if err != nil {
		return err
	}
	menus, err := info.MenuNodes()
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.removePluginLocked(pluginID)

	keys := catalogKeys{
		permissions: make([]string, 0, len(permissions)),
		menus:       make([]string, 0, len(menus)),
	}
	for _, permission := range permissions {
		r.permissions[permission.Key()] = permission
		keys.permissions = append(keys.permissions, permission.Key())
	}
	for _, menu := range menus {
		r.menus[menu.Key()] = menu
		keys.menus = append(keys.menus, menu.Key())
	}
	r.byPlugin[pluginID] = keys
	return nil
}

func (r *MemoryCatalogRegistry) RemovePlugin(pluginID string) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.removePluginLocked(strings.TrimSpace(strings.ToLower(pluginID)))
	return nil
}

func (r *MemoryCatalogRegistry) DisablePlugin(pluginID string) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := r.byPlugin[strings.TrimSpace(strings.ToLower(pluginID))]
	for _, key := range keys.permissions {
		permission, ok := r.permissions[key]
		if !ok {
			continue
		}
		permission.Enabled = false
		r.permissions[key] = permission
	}
	for _, key := range keys.menus {
		menu, ok := r.menus[key]
		if !ok {
			continue
		}
		menu.Visible = false
		r.menus[key] = menu
	}
	return nil
}

func (r *MemoryCatalogRegistry) ListPermissions() []domainpermission.PermissionResource {
	if r == nil {
		return []domainpermission.PermissionResource{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.permissions))
	for key := range r.permissions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]domainpermission.PermissionResource, 0, len(keys))
	for _, key := range keys {
		out = append(out, r.permissions[key])
	}
	return out
}

func (r *MemoryCatalogRegistry) ListMenuNodes() []domainmenu.MenuNode {
	if r == nil {
		return []domainmenu.MenuNode{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.menus))
	for key := range r.menus {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]domainmenu.MenuNode, 0, len(keys))
	for _, key := range keys {
		out = append(out, r.menus[key])
	}
	return out
}

func (r *MemoryCatalogRegistry) removePluginLocked(pluginID string) {
	keys := r.byPlugin[pluginID]
	for _, key := range keys.permissions {
		delete(r.permissions, key)
	}
	for _, key := range keys.menus {
		delete(r.menus, key)
	}
	delete(r.byPlugin, pluginID)
}

func (i Info) CatalogPermissions() ([]domainpermission.PermissionResource, error) {
	declarations := i.PermissionResources
	if len(declarations) == 0 {
		declarations = make([]PermissionDeclaration, 0, len(i.Permissions))
		for _, key := range i.Permissions {
			declarations = append(declarations, PermissionDeclaration{Key: key})
		}
	}
	declarations = append(declarations, i.apiPermissionDeclarations()...)
	source := strings.TrimSpace(strings.ToLower(i.ID))
	if source == "" {
		return nil, ErrPluginManifestBroken
	}
	out := make([]domainpermission.PermissionResource, 0, len(declarations))
	for _, declaration := range declarations {
		declaration = normalizePermissionDeclaration(declaration)
		resource, err := domainpermission.NewResourceWithMetadata(domainpermission.ResourceIdentity{
			Key:    declaration.Key,
			Type:   domainpermission.ResourceType(declaration.Type),
			Module: declaration.Module,
			Source: fmt.Sprintf("plugin.%s", source),
		}, declaration.Name, domainpermission.RiskLevel(declaration.Risk), declaration.Metadata)
		if err != nil {
			return nil, err
		}
		out = append(out, resource)
	}
	return out, nil
}

func (i Info) RouteExtensions() ([]RouteExtension, error) {
	if i.APIContract == nil {
		return []RouteExtension{}, nil
	}
	source := strings.TrimSpace(strings.ToLower(i.ID))
	if source == "" {
		return nil, ErrPluginManifestBroken
	}
	out := make([]RouteExtension, 0, len(i.APIContract.Routes))
	for _, route := range i.APIContract.Routes {
		out = append(out, RouteExtension{
			Method:      strings.ToUpper(strings.TrimSpace(route.Method)),
			Path:        NormalizeEntryPath(route.Path),
			Summary:     strings.TrimSpace(route.Summary),
			Permission:  strings.TrimSpace(strings.ToLower(route.Permission)),
			AuditAction: strings.TrimSpace(strings.ToLower(route.AuditAction)),
			Source:      fmt.Sprintf("plugin.%s", source),
		})
	}
	return out, nil
}

func (i Info) AuditActions() []string {
	if i.APIContract == nil {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(i.APIContract.Routes))
	for _, route := range i.APIContract.Routes {
		action := strings.TrimSpace(strings.ToLower(route.AuditAction))
		if action == "" {
			continue
		}
		if _, ok := seen[action]; ok {
			continue
		}
		seen[action] = struct{}{}
		out = append(out, action)
	}
	sort.Strings(out)
	return out
}

func (i Info) apiPermissionDeclarations() []PermissionDeclaration {
	if i.APIContract == nil {
		return nil
	}
	out := make([]PermissionDeclaration, 0, len(i.APIContract.Routes))
	for _, route := range i.APIContract.Routes {
		key := strings.TrimSpace(strings.ToLower(route.Permission))
		if key == "" {
			continue
		}
		out = append(out, PermissionDeclaration{
			Key:    key,
			Type:   "api",
			Module: inferPermissionModule(key),
			Name:   strings.TrimSpace(route.Summary),
			Risk:   "medium",
			Metadata: map[string]string{
				"method": strings.ToUpper(strings.TrimSpace(route.Method)),
				"path":   NormalizeEntryPath(route.Path),
			},
		})
	}
	return out
}
