package plugin

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestPharmaOAPluginManifestCoversIndustrySkeleton(t *testing.T) {
	dir := pharmaOAPluginDir()
	info, err := NewFileLoader().Load(dir)
	if err != nil {
		t.Fatalf("load pharma oa plugin manifest: %v", err)
	}
	if err := info.ValidateManifest(); err != nil {
		t.Fatalf("validate pharma oa plugin manifest: %v", err)
	}

	if info.ID != "pharma_oa" || info.Level != LevelApp || info.AppID != "pharma_oa" || info.UIMode != UIModeFrontendOnly {
		t.Fatalf("unexpected pharma oa placement: %+v", info)
	}
	if info.FrontendEntry != "/skoll/pharma-oa/employees" {
		t.Fatalf("unexpected frontend entry: %s", info.FrontendEntry)
	}
	if info.UIMenu == nil || info.UIMenu.Key != "plugin.pharma_oa" || info.UIMenu.Path != "/skoll/pharma-oa/employees" {
		t.Fatalf("unexpected pharma oa menu: %+v", info.UIMenu)
	}
	if len(info.UIMenu.RequiredPermissions) != 1 || info.UIMenu.RequiredPermissions[0] != "pharma_oa.menu.read" {
		t.Fatalf("unexpected menu permissions: %+v", info.UIMenu.RequiredPermissions)
	}
	if info.ConfigSchema == nil || len(info.ConfigSchema.Fields) != 3 {
		t.Fatalf("unexpected config schema: %+v", info.ConfigSchema)
	}
	assertConfigField(t, info.ConfigSchema.Fields[0], "pharma_oa.seed.enabled", "boolean", false)
	assertConfigField(t, info.ConfigSchema.Fields[1], "pharma_oa.seed.scope", "string", true)
	assertConfigField(t, info.ConfigSchema.Fields[2], "pharma_oa.alert_days", "number", false)

	if len(info.PermissionResources) != 8 {
		t.Fatalf("unexpected permission resources: %+v", info.PermissionResources)
	}
	assertPermissionDeclaration(t, info.PermissionResources[0], "pharma_oa.menu.read", "menu", "low")
	assertPermissionDeclaration(t, info.PermissionResources[1], "pharma_oa.plugin.manage", "plugin", "high")
	assertPermissionDeclaration(t, info.PermissionResources[2], "pharma_oa.seed.apply", "button", "medium")
	assertPermissionDeclaration(t, info.PermissionResources[3], "pharma_oa.employee.read", "api", "low")
	assertPermissionDeclaration(t, info.PermissionResources[4], "pharma_oa.employee.create", "api", "medium")
	assertPermissionDeclaration(t, info.PermissionResources[5], "pharma_oa.employee.update", "api", "medium")
	assertPermissionDeclaration(t, info.PermissionResources[6], "pharma_oa.employee.leave", "button", "high")
	assertPermissionDeclaration(t, info.PermissionResources[7], "pharma_oa.employee.reminder", "api", "low")

	routes, err := info.RouteExtensions()
	if err != nil {
		t.Fatalf("route extensions: %v", err)
	}
	if len(routes) != 7 {
		t.Fatalf("unexpected route count: %+v", routes)
	}
	assertRoute(t, routes[0], "GET", "/v1/plugins/pharma_oa/api/employees", "pharma_oa.employee.read", "pharma_oa.employee.read")
	assertRoute(t, routes[1], "POST", "/v1/plugins/pharma_oa/api/employees", "pharma_oa.employee.create", "pharma_oa.employee.create")
	assertRoute(t, routes[2], "PUT", "/v1/plugins/pharma_oa/api/employees", "pharma_oa.employee.update", "pharma_oa.employee.update")
	assertRoute(t, routes[3], "POST", "/v1/plugins/pharma_oa/api/employees/leave", "pharma_oa.employee.leave", "pharma_oa.employee.leave")
	assertRoute(t, routes[4], "GET", "/v1/plugins/pharma_oa/api/employees/qualification-reminders", "pharma_oa.employee.reminder", "pharma_oa.employee.reminder")
	assertRoute(t, routes[5], "GET", "/v1/plugins/pharma_oa/api/demo-seed/status", "pharma_oa.seed.read", "pharma_oa.seed.read")
	assertRoute(t, routes[6], "POST", "/v1/plugins/pharma_oa/api/demo-seed/apply", "pharma_oa.seed.apply", "pharma_oa.seed.apply")
}

func TestPharmaOAPluginLifecycleSmoke(t *testing.T) {
	dir := pharmaOAPluginDir()
	preflight, err := NewInstallPreflightService(nil).Check(InstallPreflightInput{
		Path:        dir,
		CoreVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("preflight pharma oa plugin: %v", err)
	}
	if preflight.Status != InstallPreflightStatusPass {
		t.Fatalf("expected preflight pass, got %+v", preflight)
	}
	if len(preflight.Menus.Add) != 1 || preflight.Menus.Add[0].Key != "plugin.pharma_oa" {
		t.Fatalf("expected pharma oa menu preflight, got %+v", preflight.Menus)
	}
	if len(preflight.API.Routes) != 7 || len(preflight.API.AuditActions) != 7 {
		t.Fatalf("expected api and audit preflight, got %+v", preflight.API)
	}

	catalog := NewMemoryCatalogRegistry()
	routes := NewMemoryRegistry()
	audit := &fakeCatalogAuditSink{}
	manager := NewRuntimeManager(NewFileLoader(), NewTopologicalResolver())
	manager.SetCatalogRegistry(catalog)
	manager.SetExtensionRegistry(routes)
	manager.SetCatalogAuditSink(audit)

	installed, err := manager.Install(dir)
	if err != nil {
		t.Fatalf("install pharma oa plugin: %v", err)
	}
	if installed.ID != "pharma_oa" || installed.State != StateInstalled {
		t.Fatalf("unexpected installed plugin: %+v", installed)
	}
	if _, err := manager.Install(dir); !errors.Is(err, ErrPluginAlreadyExists) {
		t.Fatalf("expected duplicate install failure, got %v", err)
	}
	if err := manager.Enable("pharma_oa"); err != nil {
		t.Fatalf("enable pharma oa plugin: %v", err)
	}
	assertCatalogPermission(t, catalog, "pharma_oa.menu.read", true)
	assertCatalogPermission(t, catalog, "pharma_oa.plugin.manage", true)
	assertCatalogPermission(t, catalog, "pharma_oa.seed.apply", true)
	assertCatalogPermission(t, catalog, "pharma_oa.seed.read", true)
	assertCatalogPermission(t, catalog, "pharma_oa.employee.read", true)
	assertCatalogPermission(t, catalog, "pharma_oa.employee.create", true)
	assertCatalogPermission(t, catalog, "pharma_oa.employee.update", true)
	assertCatalogPermission(t, catalog, "pharma_oa.employee.leave", true)
	assertCatalogPermission(t, catalog, "pharma_oa.employee.reminder", true)
	assertCatalogMenu(t, catalog, "plugin.pharma_oa", "/skoll/pharma-oa/employees", true)

	snapshot := routes.Snapshot()
	if len(snapshot.Routes) != 7 {
		t.Fatalf("expected two registered pharma oa routes, got %+v", snapshot.Routes)
	}
	assertRoute(t, snapshot.Routes[0], "GET", "/v1/plugins/pharma_oa/api/employees", "pharma_oa.employee.read", "pharma_oa.employee.read")
	assertRoute(t, snapshot.Routes[1], "POST", "/v1/plugins/pharma_oa/api/employees", "pharma_oa.employee.create", "pharma_oa.employee.create")
	assertRoute(t, snapshot.Routes[2], "PUT", "/v1/plugins/pharma_oa/api/employees", "pharma_oa.employee.update", "pharma_oa.employee.update")
	assertRoute(t, snapshot.Routes[3], "POST", "/v1/plugins/pharma_oa/api/employees/leave", "pharma_oa.employee.leave", "pharma_oa.employee.leave")
	assertRoute(t, snapshot.Routes[4], "GET", "/v1/plugins/pharma_oa/api/employees/qualification-reminders", "pharma_oa.employee.reminder", "pharma_oa.employee.reminder")
	assertRoute(t, snapshot.Routes[5], "GET", "/v1/plugins/pharma_oa/api/demo-seed/status", "pharma_oa.seed.read", "pharma_oa.seed.read")
	assertRoute(t, snapshot.Routes[6], "POST", "/v1/plugins/pharma_oa/api/demo-seed/apply", "pharma_oa.seed.apply", "pharma_oa.seed.apply")

	if err := manager.Disable("pharma_oa"); err != nil {
		t.Fatalf("disable pharma oa plugin: %v", err)
	}
	assertCatalogPermission(t, catalog, "pharma_oa.menu.read", false)
	assertCatalogPermission(t, catalog, "pharma_oa.seed.apply", false)
	assertCatalogPermission(t, catalog, "pharma_oa.employee.read", false)
	assertCatalogMenu(t, catalog, "plugin.pharma_oa", "/skoll/pharma-oa/employees", false)

	if len(audit.events) != 3 {
		t.Fatalf("expected catalog, route, and disable audit events, got %+v", audit.events)
	}
	assertAuditEvent(t, audit.events[0], "pharma_oa", "plugin_catalog_import", "ok", 15, 1, 7, 7)
	assertAuditEvent(t, audit.events[1], "pharma_oa", "plugin_route_import", "ok", 15, 1, 7, 7)
	assertAuditEvent(t, audit.events[2], "pharma_oa", "plugin_catalog_disable", "ok", 15, 1, 7, 7)

	duplicate, err := NewInstallPreflightService(nil).Check(InstallPreflightInput{
		Path:      dir,
		Installed: []Info{installed},
	})
	if err != nil {
		t.Fatalf("duplicate preflight: %v", err)
	}
	if duplicate.Status != InstallPreflightStatusBlocked || len(duplicate.Blockers) == 0 {
		t.Fatalf("expected duplicate install to be blocked, got %+v", duplicate)
	}
}

func pharmaOAPluginDir() string {
	return filepath.Join("..", "..", "plugins", "pharma_oa")
}

func assertConfigField(t *testing.T, field ConfigField, key string, typ string, required bool) {
	t.Helper()
	if field.Key != key || field.Type != typ || field.Required != required {
		t.Fatalf("unexpected config field: %+v", field)
	}
}

func assertPermissionDeclaration(t *testing.T, permission PermissionDeclaration, key string, typ string, risk string) {
	t.Helper()
	if permission.Key != key || permission.Type != typ || permission.Module != "pharma_oa" || permission.Risk != risk {
		t.Fatalf("unexpected permission declaration: %+v", permission)
	}
}

func assertRoute(t *testing.T, route RouteExtension, method string, path string, permission string, auditAction string) {
	t.Helper()
	if route.Method != method || route.Path != path || route.Permission != permission || route.AuditAction != auditAction || route.Source != "plugin.pharma_oa" {
		t.Fatalf("unexpected route: %+v", route)
	}
}

func assertAuditEvent(t *testing.T, event CatalogAuditEvent, pluginID string, action string, result string, permissions int, menus int, routes int, auditActions int) {
	t.Helper()
	if event.PluginID != pluginID || event.Action != action || event.Result != result {
		t.Fatalf("unexpected audit event identity: %+v", event)
	}
	if event.Permissions != permissions || event.Menus != menus || event.Routes != routes || event.AuditActions != auditActions {
		t.Fatalf("unexpected audit event counts: %+v", event)
	}
}
