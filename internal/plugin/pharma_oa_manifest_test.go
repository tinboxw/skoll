package plugin

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestPharmaOAPluginManifestCoversCurrentIndustryBoundary(t *testing.T) {
	info := loadPharmaOAInfo(t)
	if info.ID != "pharma_oa" || info.AppID != info.ID || info.Level != LevelApp || info.UIMode != UIModeMonolith {
		t.Fatalf("unexpected Pharma OA placement: %+v", info)
	}
	if info.Version != "0.4.0" || info.APIVersion != "v1" || info.MigrationVersion != info.Version {
		t.Fatalf("unexpected current contract version: version=%q api=%q migration=%q", info.Version, info.APIVersion, info.MigrationVersion)
	}
	if info.ServiceBaseURL != "http://127.0.0.1:18093" || info.ServiceHealthURL != "http://127.0.0.1:18093/health" {
		t.Fatalf("managed service endpoints are incomplete: base=%q health=%q", info.ServiceBaseURL, info.ServiceHealthURL)
	}
	if info.NameZhCN != "医药 OA" || info.NameEnUS != "Pharma OA" || info.UIMenu == nil || info.UIMenu.LabelZhCN != "医药 OA" {
		t.Fatalf("localized plugin identity is incomplete: %+v", info.UIMenu)
	}
	if info.UIMenu.Path != info.FrontendEntry || len(info.UIMenu.RequiredPermissions) != 1 || info.UIMenu.RequiredPermissions[0] != "pharma_oa.menu.read" {
		t.Fatalf("unexpected plugin menu: %+v", info.UIMenu)
	}
	if info.ConfigSchema == nil || len(info.ConfigSchema.Fields) != 3 {
		t.Fatalf("unexpected config schema: %+v", info.ConfigSchema)
	}
	if info.DataManifest == nil || info.DataManifest.Namespace != info.ID || info.DataManifest.MigrationVersion != info.Version || info.DataManifest.MigrationDirectory != "migrations" || info.DataManifest.UninstallPolicy != DataUninstallDrop || info.DataManifest.RollbackPolicy != DataRollbackAutomatic || len(info.DataManifest.Tables) != 4 {
		t.Fatalf("unexpected plugin data lifecycle: %+v", info.DataManifest)
	}
	for _, table := range info.DataManifest.Tables {
		columns := make(map[string]struct{}, len(table.Columns))
		for _, column := range table.Columns {
			columns[column] = struct{}{}
		}
		for _, required := range []string{"tenant_id", "organization_id", "owner_id"} {
			if _, ok := columns[required]; !ok {
				t.Fatalf("table %s is missing scope column %s", table.Name, required)
			}
		}
	}
	if info.EventContract == nil || len(info.EventContract.Subscriptions) != 5 {
		t.Fatalf("unexpected event contract: %+v", info.EventContract)
	}

	permissions := make(map[string]struct{}, len(info.PermissionResources))
	for _, permission := range info.PermissionResources {
		if permission.Key == "" || permission.Module != info.ID || permission.Type == "" || permission.Risk == "" {
			t.Fatalf("incomplete permission declaration: %+v", permission)
		}
		if _, exists := permissions[permission.Key]; exists {
			t.Fatalf("duplicate permission declaration: %s", permission.Key)
		}
		permissions[permission.Key] = struct{}{}
	}
	for _, required := range []string{"pharma_oa.foundation.read", "pharma_oa.menu.read", "pharma_oa.seed.read", "pharma_oa.employee.read", "pharma_oa.qualification.read"} {
		if _, ok := permissions[required]; !ok {
			t.Fatalf("required permission %s is missing", required)
		}
	}

	routes, err := info.RouteExtensions()
	if err != nil {
		t.Fatalf("resolve Pharma OA routes: %v", err)
	}
	if len(routes) < 100 {
		t.Fatalf("Pharma OA route surface is unexpectedly small: %d", len(routes))
	}
	seenRoutes := make(map[string]struct{}, len(routes))
	foundationFound := false
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if !strings.HasPrefix(route.Path, "/v1/plugins/pharma_oa/api/") || route.Source != "plugin.pharma_oa" {
			t.Fatalf("route escapes current plugin namespace: %+v", route)
		}
		if _, exists := seenRoutes[key]; exists {
			t.Fatalf("duplicate route: %s", key)
		}
		seenRoutes[key] = struct{}{}
		if _, ok := permissions[route.Permission]; !ok {
			t.Fatalf("route %s references undeclared permission %s", key, route.Permission)
		}
		if strings.TrimSpace(route.AuditAction) == "" {
			t.Fatalf("route %s has no audit action", key)
		}
		if key == "GET /v1/plugins/pharma_oa/api/meta" && route.Permission == "pharma_oa.foundation.read" {
			foundationFound = true
		}
	}
	if !foundationFound {
		t.Fatal("independent plugin foundation route is missing")
	}
}

func TestPharmaOAPluginLifecycleUsesManifestAsSourceOfTruth(t *testing.T) {
	dir := pharmaOAPluginDir()
	info := loadPharmaOAInfo(t)
	wantRoutes, err := info.RouteExtensions()
	if err != nil {
		t.Fatal(err)
	}
	preflight, err := NewInstallPreflightService(nil).Check(InstallPreflightInput{Path: dir})
	if err != nil {
		t.Fatalf("preflight Pharma OA plugin: %v", err)
	}
	if preflight.Status != InstallPreflightStatusPass || len(preflight.Menus.Add) != 1 || len(preflight.API.Routes) != len(wantRoutes) {
		t.Fatalf("unexpected Pharma OA preflight: %+v", preflight)
	}

	catalog := NewMemoryCatalogRegistry()
	routeRegistry := NewMemoryRegistry()
	audit := &fakeCatalogAuditSink{}
	manager := NewRuntimeManager(NewFileLoader(), NewTopologicalResolver())
	manager.SetCatalogRegistry(catalog)
	manager.SetExtensionRegistry(routeRegistry)
	manager.SetCatalogAuditSink(audit)

	installed, err := manager.Install(dir)
	if err != nil {
		t.Fatalf("install Pharma OA plugin: %v", err)
	}
	if _, err := manager.Install(dir); !errors.Is(err, ErrPluginAlreadyExists) {
		t.Fatalf("expected duplicate install failure, got %v", err)
	}
	if err := manager.Enable(info.ID); err != nil {
		t.Fatalf("enable Pharma OA plugin: %v", err)
	}
	for _, permission := range info.PermissionResources {
		assertCatalogPermission(t, catalog, permission.Key, true)
	}
	assertCatalogMenu(t, catalog, info.UIMenu.Key, info.UIMenu.Path, true)
	snapshot := routeRegistry.Snapshot()
	if len(snapshot.Routes) != len(wantRoutes) {
		t.Fatalf("registered routes=%d want=%d", len(snapshot.Routes), len(wantRoutes))
	}
	registered := make(map[string]RouteExtension, len(snapshot.Routes))
	for _, route := range snapshot.Routes {
		registered[route.Method+" "+route.Path] = route
	}
	for _, route := range wantRoutes {
		key := route.Method + " " + route.Path
		if got, ok := registered[key]; !ok || got.Permission != route.Permission || got.AuditAction != route.AuditAction {
			t.Fatalf("registered route mismatch for %s: %+v", key, got)
		}
	}

	if err := manager.Disable(info.ID); err != nil {
		t.Fatalf("disable Pharma OA plugin: %v", err)
	}
	for _, permission := range info.PermissionResources {
		assertCatalogPermission(t, catalog, permission.Key, false)
	}
	assertCatalogMenu(t, catalog, info.UIMenu.Key, info.UIMenu.Path, false)
	if len(audit.events) != 4 {
		t.Fatalf("expected catalog, route, event, and disable audit events, got %+v", audit.events)
	}
	wantActions := map[string]struct{}{
		"plugin_catalog_import":  {},
		"plugin_route_import":    {},
		"plugin_event_import":    {},
		"plugin_catalog_disable": {},
	}
	for _, event := range audit.events {
		if _, ok := wantActions[event.Action]; !ok {
			t.Fatalf("unexpected lifecycle audit action: %+v", event)
		}
		delete(wantActions, event.Action)
		if event.PluginID != info.ID || event.Routes != len(wantRoutes) || event.Events != len(info.EventContract.Subscriptions) {
			t.Fatalf("unexpected lifecycle audit evidence: %+v", event)
		}
	}

	duplicate, err := NewInstallPreflightService(nil).Check(InstallPreflightInput{Path: dir, Installed: []Info{installed}})
	if err != nil {
		t.Fatalf("duplicate preflight: %v", err)
	}
	if duplicate.Status != InstallPreflightStatusBlocked || len(duplicate.Blockers) == 0 {
		t.Fatalf("expected duplicate install to be blocked, got %+v", duplicate)
	}
}

func loadPharmaOAInfo(t *testing.T) Info {
	t.Helper()
	info, err := NewFileLoader().Load(pharmaOAPluginDir())
	if err != nil {
		t.Fatalf("load Pharma OA plugin manifest: %v", err)
	}
	if err := info.ValidateManifest(); err != nil {
		t.Fatalf("validate Pharma OA plugin manifest: %v", err)
	}
	return info
}

func pharmaOAPluginDir() string {
	return filepath.Join("..", "..", "plugins", "pharma_oa")
}
