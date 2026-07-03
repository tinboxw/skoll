package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginLifecycleAcceptanceInstallEnableDisableUpgradeRollback(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "reports")
	writeLifecyclePluginManifest(t, pluginDir, lifecyclePluginFixture{
		version:           "1.0.0",
		permission:        "reports.read",
		menuPath:          "/skoll/plugins/reports",
		configDefault:     "10",
		migrationVersion:  "1.0.0",
		frontendAssetBody: "console.log('reports v1');\n",
	})

	catalog := NewMemoryCatalogRegistry()
	manager := NewRuntimeManager(NewFileLoader(), NewTopologicalResolver())
	manager.SetCatalogRegistry(catalog)

	installed, err := manager.Install(pluginDir)
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	if installed.Version != "1.0.0" || installed.State != StateInstalled {
		t.Fatalf("unexpected installed plugin: %+v", installed)
	}

	if err := manager.Enable("reports"); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}
	assertCatalogPermission(t, catalog, "reports.read", true)
	assertCatalogMenu(t, catalog, "plugin.reports", "/skoll/plugins/reports", true)

	if err := manager.Disable("reports"); err != nil {
		t.Fatalf("disable plugin: %v", err)
	}
	assertCatalogPermission(t, catalog, "reports.read", false)
	assertCatalogMenu(t, catalog, "plugin.reports", "/skoll/plugins/reports", false)

	if err := manager.Enable("reports"); err != nil {
		t.Fatalf("re-enable plugin: %v", err)
	}
	assertCatalogPermission(t, catalog, "reports.read", true)

	beforeUpgrade, err := manager.Get("reports")
	if err != nil {
		t.Fatalf("get before upgrade: %v", err)
	}
	writeLifecyclePluginManifest(t, pluginDir, lifecyclePluginFixture{
		version:           "1.1.0",
		permission:        "reports.export",
		menuPath:          "/skoll/plugins/reports/v2",
		configDefault:     "50",
		migrationVersion:  "1.1.0",
		frontendAssetBody: "console.log('reports v2');\n",
	})
	if err := manager.ReloadPluginMetadata("reports"); err != nil {
		t.Fatalf("upgrade plugin metadata: %v", err)
	}
	upgraded, err := manager.Get("reports")
	if err != nil {
		t.Fatalf("get upgraded plugin: %v", err)
	}
	if upgraded.Version != "1.1.0" || upgraded.State != StateEnabled {
		t.Fatalf("unexpected upgraded plugin: %+v", upgraded)
	}
	assertCatalogPermission(t, catalog, "reports.export", true)
	assertCatalogMenu(t, catalog, "plugin.reports", "/skoll/plugins/reports/v2", true)

	writeLifecyclePluginManifest(t, pluginDir, lifecyclePluginFixture{
		version:           "1.0.0",
		permission:        "reports.read",
		menuPath:          "/skoll/plugins/reports",
		configDefault:     "10",
		migrationVersion:  "1.0.0",
		frontendAssetBody: "console.log('reports v1 rollback');\n",
	})
	if err := manager.ReloadPluginMetadata("reports"); err != nil {
		t.Fatalf("rollback plugin metadata: %v", err)
	}
	rolledBack, err := manager.Get("reports")
	if err != nil {
		t.Fatalf("get rolled back plugin: %v", err)
	}
	if rolledBack.Version != "1.0.0" || rolledBack.State != StateEnabled {
		t.Fatalf("unexpected rolled back plugin: %+v", rolledBack)
	}
	assertCatalogPermission(t, catalog, "reports.read", true)
	assertCatalogMenu(t, catalog, "plugin.reports", "/skoll/plugins/reports", true)

	plan, err := NewPluginRollbackService().BuildPlan(PluginRollbackPlanInput{
		PluginID: "reports",
		From:     upgraded,
		Target:   rolledBack,
	})
	if err != nil {
		t.Fatalf("build rollback plan: %v", err)
	}
	if plan.Status != "passed" || plan.FromVersion != "1.1.0" || plan.ToVersion != "1.0.0" {
		t.Fatalf("unexpected rollback plan: %+v", plan)
	}
	if rollbackCheckpoint(plan, "permission").After != "reports.read" {
		t.Fatalf("rollback permission checkpoint did not restore v1 permission: %+v", plan)
	}
	if beforeUpgrade.Version != "1.0.0" {
		t.Fatalf("expected original snapshot to remain v1, got %+v", beforeUpgrade)
	}
}

type lifecyclePluginFixture struct {
	version           string
	permission        string
	menuPath          string
	configDefault     string
	migrationVersion  string
	frontendAssetBody string
}

func writeLifecyclePluginManifest(t *testing.T, dir string, fixture lifecyclePluginFixture) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "frontend", "dist", "assets"), 0o755); err != nil {
		t.Fatalf("mkdir lifecycle plugin: %v", err)
	}
	manifest := strings.Join([]string{
		`id: "reports"`,
		`name: "Reports"`,
		`version: "` + fixture.version + `"`,
		`api_version: "v1"`,
		`compatibility_skoll: ">=1.0.0 <2.0.0"`,
		`ui_mode: "monolith"`,
		`ui_nav_position: "sidebar"`,
		`migration_version: "` + fixture.migrationVersion + `"`,
		`permissions:`,
		`  - key: "` + fixture.permission + `"`,
		`    type: "api"`,
		`    module: "reports"`,
		`    name: "Reports permission"`,
		`ui_menu:`,
		`  key: "plugin.reports"`,
		`  label: "Reports"`,
		`  path: "` + fixture.menuPath + `"`,
		`  required_permissions:`,
		`    - "` + fixture.permission + `"`,
		`config_schema:`,
		`  title: "Reports Config"`,
		`  fields:`,
		`    - key: "rollout_percent"`,
		`      label: "Rollout Percent"`,
		`      type: "number"`,
		`      default: "` + fixture.configDefault + `"`,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write lifecycle manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "frontend", "dist", "assets", "app.js"), []byte(fixture.frontendAssetBody), 0o644); err != nil {
		t.Fatalf("write lifecycle asset: %v", err)
	}
}

func assertCatalogPermission(t *testing.T, catalog *MemoryCatalogRegistry, key string, enabled bool) {
	t.Helper()
	for _, item := range catalog.ListPermissions() {
		if item.Key() == key {
			if item.Enabled != enabled {
				t.Fatalf("permission %s enabled=%t, want %t", key, item.Enabled, enabled)
			}
			return
		}
	}
	t.Fatalf("permission %s not found in catalog: %+v", key, catalog.ListPermissions())
}

func assertCatalogMenu(t *testing.T, catalog *MemoryCatalogRegistry, key string, path string, visible bool) {
	t.Helper()
	for _, item := range catalog.ListMenuNodes() {
		if item.Key() == key {
			if item.Path() != path || item.Visible != visible {
				t.Fatalf("menu %s path=%s visible=%t, want path=%s visible=%t", key, item.Path(), item.Visible, path, visible)
			}
			return
		}
	}
	t.Fatalf("menu %s not found in catalog: %+v", key, catalog.ListMenuNodes())
}
