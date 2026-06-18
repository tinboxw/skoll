package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeManagerDependencyLifecycle(t *testing.T) {
	dir := t.TempDir()
	basePath := writePluginManifest(t, dir, "base", "Base", "1.0.0", "")
	appPath := writePluginManifest(t, dir, "app", "App", "1.0.0", "base")

	m := NewRuntimeManager(NewFileLoader(), NewTopologicalResolver())
	if _, err := m.Install(basePath); err != nil {
		t.Fatalf("install base error: %v", err)
	}
	if _, err := m.Install(appPath); err != nil {
		t.Fatalf("install app error: %v", err)
	}

	if err := m.Enable("app"); err != nil {
		t.Fatalf("enable app error: %v", err)
	}

	base, err := m.Get("base")
	if err != nil {
		t.Fatalf("get base error: %v", err)
	}
	if base.State != StateEnabled {
		t.Fatalf("base should be enabled, got %s", base.State)
	}

	if err := m.Disable("base"); err == nil {
		t.Fatal("expected disable base to fail while app depends on it")
	}

	if err := m.Disable("app"); err != nil {
		t.Fatalf("disable app error: %v", err)
	}
	if err := m.Disable("base"); err != nil {
		t.Fatalf("disable base error: %v", err)
	}
	if err := m.Uninstall("base"); err != nil {
		t.Fatalf("uninstall base error: %v", err)
	}

	base, err = m.Get("base")
	if err != nil {
		t.Fatalf("get base after uninstall error: %v", err)
	}
	if base.State != StateUninstalled {
		t.Fatalf("expected uninstalled state, got %s", base.State)
	}
}

func TestRuntimeManagerInstallDuplicate(t *testing.T) {
	dir := t.TempDir()
	path := writePluginManifest(t, dir, "dup", "Dup", "1.0.0", "")

	m := NewRuntimeManager(NewFileLoader(), NewTopologicalResolver())
	if _, err := m.Install(path); err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	if _, err := m.Install(path); !errors.Is(err, ErrPluginAlreadyExists) {
		t.Fatalf("expected ErrPluginAlreadyExists, got %v", err)
	}
}

func TestRuntimeManagerReloadPluginMetadata(t *testing.T) {
	dir := t.TempDir()
	path := writePluginManifest(t, dir, "portal", "Portal", "1.0.0", "")

	m := NewRuntimeManager(NewFileLoader(), NewTopologicalResolver())
	if _, err := m.Install(path); err != nil {
		t.Fatalf("install plugin failed: %v", err)
	}
	if err := m.Enable("portal"); err != nil {
		t.Fatalf("enable plugin failed: %v", err)
	}

	before, err := m.Get("portal")
	if err != nil {
		t.Fatalf("get plugin before reload failed: %v", err)
	}
	beforeInstalledAt := before.InstalledAt

	updatedManifest := "id: \"portal\"\n" +
		"name: \"Portal Updated\"\n" +
		"version: \"1.2.3\"\n" +
		"ui_nav_position: sidebar\n"
	if err := os.WriteFile(filepath.Join(path, "plugin.yaml"), []byte(updatedManifest), 0o600); err != nil {
		t.Fatalf("update manifest failed: %v", err)
	}

	if err := m.ReloadPluginMetadata("portal"); err != nil {
		t.Fatalf("reload plugin metadata failed: %v", err)
	}

	after, err := m.Get("portal")
	if err != nil {
		t.Fatalf("get plugin after reload failed: %v", err)
	}
	if after.Name != "Portal Updated" {
		t.Fatalf("expected updated name, got %q", after.Name)
	}
	if after.Version != "1.2.3" {
		t.Fatalf("expected updated version, got %q", after.Version)
	}
	if after.UINavPosition != UINavPositionSidebar {
		t.Fatalf("expected updated nav position, got %q", after.UINavPosition)
	}
	if after.State != StateEnabled {
		t.Fatalf("expected state to be preserved as enabled, got %q", after.State)
	}
	if after.EnabledAt == nil {
		t.Fatal("expected enabled timestamp to be preserved")
	}
	if !after.InstalledAt.Equal(beforeInstalledAt) {
		t.Fatal("expected installed timestamp to be preserved")
	}
}

func TestRuntimeManagerEnableImportsCatalogRegistry(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "reports")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	manifest := `id: "reports"
name: "Reports"
version: "1.0.0"
permissions:
  - key: "reports.read"
    type: "api"
    module: "reports"
    name: "Read reports"
ui_menu:
  key: "plugin.reports"
  label: "Reports"
  path: "/skoll/plugins/reports"
  required_permissions:
    - "reports.read"
`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	catalog := NewMemoryCatalogRegistry()
	m := NewRuntimeManager(NewFileLoader(), NewTopologicalResolver())
	m.SetCatalogRegistry(catalog)
	if _, err := m.Install(pluginDir); err != nil {
		t.Fatalf("install plugin failed: %v", err)
	}
	if err := m.Enable("reports"); err != nil {
		t.Fatalf("enable plugin failed: %v", err)
	}

	permissions := catalog.ListPermissions()
	if len(permissions) != 1 || permissions[0].Key() != "reports.read" || permissions[0].Source() != "plugin.reports" {
		t.Fatalf("unexpected imported permissions: %+v", permissions)
	}
	menus := catalog.ListMenuNodes()
	if len(menus) != 1 || menus[0].Key() != "plugin.reports" || menus[0].RequiredPermissions[0] != "reports.read" {
		t.Fatalf("unexpected imported menus: %+v", menus)
	}

	if err := m.Disable("reports"); err != nil {
		t.Fatalf("disable plugin failed: %v", err)
	}
	if len(catalog.ListPermissions()) != 0 || len(catalog.ListMenuNodes()) != 0 {
		t.Fatalf("expected catalog cleanup after disable")
	}
}

func writePluginManifest(t *testing.T, root, id, name, version, dep string) string {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}

	manifest := "id: \"" + id + "\"\n" +
		"name: \"" + name + "\"\n" +
		"version: \"" + version + "\"\n"
	if dep != "" {
		manifest += "dependencies:\n" +
			"  - id: \"" + dep + "\"\n"
	}

	path := filepath.Join(dir, "plugin.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	return dir
}
