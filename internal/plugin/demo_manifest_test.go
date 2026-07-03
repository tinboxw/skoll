package plugin

import (
	"path/filepath"
	"testing"
)

func TestDemoPluginManifestCoversPlatformContract(t *testing.T) {
	dir := filepath.Join("..", "..", "plugins", "demo")
	info, err := NewFileLoader().Load(dir)
	if err != nil {
		t.Fatalf("load demo plugin manifest: %v", err)
	}
	if err := info.ValidateManifest(); err != nil {
		t.Fatalf("validate demo plugin manifest: %v", err)
	}

	if info.UIMode != UIModeSeparated || info.Level != LevelApp || info.AppID != "demo" {
		t.Fatalf("unexpected plugin placement: mode=%s level=%s appID=%s", info.UIMode, info.Level, info.AppID)
	}
	if info.UIMenu == nil || info.UIMenu.Key != "plugin.demo" || info.UIMenu.Path != "/skoll/plugins/demo" || info.UIMenu.Component != "PluginRuntime" {
		t.Fatalf("unexpected ui menu: %+v", info.UIMenu)
	}
	if len(info.UIMenu.RequiredPermissions) != 1 || info.UIMenu.RequiredPermissions[0] != "demo.menu.read" {
		t.Fatalf("unexpected ui menu permissions: %+v", info.UIMenu.RequiredPermissions)
	}
	if info.ConfigSchema == nil || len(info.ConfigSchema.Fields) != 3 {
		t.Fatalf("unexpected config schema: %+v", info.ConfigSchema)
	}
	assertDemoConfigField(t, info.ConfigSchema.Fields[0], "demo.endpoint", "string", true)
	assertDemoConfigField(t, info.ConfigSchema.Fields[1], "demo.mode", "select", false)
	assertDemoConfigField(t, info.ConfigSchema.Fields[2], "demo.enabled", "boolean", false)

	if len(info.PermissionResources) != 3 {
		t.Fatalf("unexpected permission resources: %+v", info.PermissionResources)
	}
	assertDemoPermission(t, info.PermissionResources[0], "demo.menu.read", "menu", "low")
	assertDemoPermission(t, info.PermissionResources[1], "demo.route.read", "api", "low")
	assertDemoPermission(t, info.PermissionResources[2], "demo.report.export", "button", "medium")

	report, err := CheckSignatureCoverage(dir, info, SignatureCoveragePolicy{})
	if err != nil {
		t.Fatalf("signature coverage: %v report=%+v", err, report)
	}
	if report.BackendEntry != "backend/main.go" || len(report.FrontendAssets) != 5 || len(report.Missing) != 0 || len(report.Mismatched) != 0 {
		t.Fatalf("unexpected signature coverage report: %+v", report)
	}
}

func assertDemoConfigField(t *testing.T, field ConfigField, key string, typ string, required bool) {
	t.Helper()
	if field.Key != key || field.Type != typ || field.Required != required {
		t.Fatalf("unexpected config field: %+v", field)
	}
}

func assertDemoPermission(t *testing.T, permission PermissionDeclaration, key string, typ string, risk string) {
	t.Helper()
	if permission.Key != key || permission.Type != typ || permission.Module != "demo" || permission.Risk != risk {
		t.Fatalf("unexpected permission: %+v", permission)
	}
}
