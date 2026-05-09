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
