package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
)

type fakePluginManager struct {
	items map[string]plugin.Info
}

func (m *fakePluginManager) Install(path string) (plugin.Info, error) {
	return plugin.Info{}, nil
}

func (m *fakePluginManager) Enable(pluginID string) error {
	return nil
}

func (m *fakePluginManager) Disable(pluginID string) error {
	return nil
}

func (m *fakePluginManager) Uninstall(pluginID string) error {
	return nil
}

func (m *fakePluginManager) List() []plugin.Info {
	result := make([]plugin.Info, 0, len(m.items))
	for _, item := range m.items {
		result = append(result, item)
	}
	return result
}

func (m *fakePluginManager) Get(pluginID string) (plugin.Info, error) {
	item, ok := m.items[pluginID]
	if !ok {
		return plugin.Info{}, plugin.ErrPluginNotFound
	}
	return item, nil
}

func TestPluginCommandListAndDebug(t *testing.T) {
	mgr := &fakePluginManager{
		items: map[string]plugin.Info{
			"auth": {
				ID:          "auth",
				Name:        "Auth",
				Version:     "1.0.0",
				State:       plugin.StateEnabled,
				Permissions: []string{"menu.read", "route.write"},
				Dependencies: []plugin.Dependency{
					{ID: "rbac", Version: "1.0.0"},
				},
				Source: "plugins/auth",
			},
		},
	}
	cmd := NewPluginCommand(mgr, nil, "")

	listOut, err := cmd.Handle(context.Background(), []string{"list"})
	if err != nil {
		t.Fatalf("list should succeed, got err: %v", err)
	}
	if !strings.Contains(listOut, "plugins=1") {
		t.Fatalf("unexpected list output: %s", listOut)
	}
	if !strings.Contains(listOut, "auth@1.0.0 state=enabled") {
		t.Fatalf("unexpected list row: %s", listOut)
	}

	debugOut, err := cmd.Handle(context.Background(), []string{"debug", "auth"})
	if err != nil {
		t.Fatalf("debug should succeed, got err: %v", err)
	}
	for _, fragment := range []string{
		"id=auth",
		"version=1.0.0",
		"state=enabled",
		"deps=rbac@1.0.0",
		"source=plugins/auth",
	} {
		if !strings.Contains(debugOut, fragment) {
			t.Fatalf("missing fragment %q in debug output: %s", fragment, debugOut)
		}
	}
}

func TestPluginCommandValidateAndLogs(t *testing.T) {
	tmp := t.TempDir()
	manifestDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := strings.Join([]string{
		"id: demo",
		"name: Demo Plugin",
		"version: 0.1.0",
		"dependencies:",
		"  - id: core",
		"    version: 1.0.0",
		"permissions:",
		"  - menu.read",
	}, "\n")
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	logsDir := filepath.Join(tmp, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("mkdir logs failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(logsDir, "demo.log"), []byte("line-1\nline-2\n"), 0o644); err != nil {
		t.Fatalf("write log failed: %v", err)
	}

	cmd := NewPluginCommand(&fakePluginManager{items: map[string]plugin.Info{}}, plugin.NewFileLoader(), logsDir)

	validateOut, err := cmd.Handle(context.Background(), []string{"validate", manifestDir})
	if err != nil {
		t.Fatalf("validate should succeed, got err: %v", err)
	}
	if !strings.Contains(validateOut, "valid id=demo version=0.1.0") {
		t.Fatalf("unexpected validate output: %s", validateOut)
	}

	logsOut, err := cmd.Handle(context.Background(), []string{"logs", "demo"})
	if err != nil {
		t.Fatalf("logs should succeed, got err: %v", err)
	}
	if logsOut != "line-1\nline-2" {
		t.Fatalf("unexpected logs output: %q", logsOut)
	}
}

func TestPluginCommandValidateAll(t *testing.T) {
	tmp := t.TempDir()
	pluginsRoot := filepath.Join(tmp, "plugins")
	if err := os.MkdirAll(pluginsRoot, 0o755); err != nil {
		t.Fatalf("mkdir plugins root failed: %v", err)
	}

	good := filepath.Join(pluginsRoot, "good")
	if err := os.MkdirAll(good, 0o755); err != nil {
		t.Fatalf("mkdir good plugin failed: %v", err)
	}
	goodManifest := strings.Join([]string{
		"id: good",
		"name: Good Plugin",
		"version: 1.0.0",
	}, "\n")
	if err := os.WriteFile(filepath.Join(good, "plugin.yaml"), []byte(goodManifest), 0o644); err != nil {
		t.Fatalf("write good manifest failed: %v", err)
	}

	bad := filepath.Join(pluginsRoot, "bad")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatalf("mkdir bad plugin failed: %v", err)
	}
	badManifest := strings.Join([]string{
		"id: bad",
		"name: Bad Plugin",
		"version: 1.0.0",
		"api_version: 1",
	}, "\n")
	if err := os.WriteFile(filepath.Join(bad, "plugin.yaml"), []byte(badManifest), 0o644); err != nil {
		t.Fatalf("write bad manifest failed: %v", err)
	}

	cmd := NewPluginCommand(&fakePluginManager{items: map[string]plugin.Info{}}, plugin.NewFileLoader(), "")
	out, err := cmd.Handle(context.Background(), []string{"validate-all", pluginsRoot})
	if err == nil {
		t.Fatalf("validate-all should fail with invalid plugin manifest")
	}
	if !strings.Contains(out, "validated=2") {
		t.Fatalf("unexpected validate-all output: %s", out)
	}
	if !strings.Contains(out, "- ok") {
		t.Fatalf("expected at least one success row, got: %s", out)
	}
	if !strings.Contains(err.Error(), "manifest validation failed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestPluginCommandUsageErrors(t *testing.T) {
	cmd := NewPluginCommand(nil, nil, "")

	_, err := cmd.Handle(context.Background(), []string{"list"})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected manager config error, got: %v", err)
	}

	_, err = cmd.Handle(context.Background(), []string{"validate"})
	if err == nil || !strings.Contains(err.Error(), "usage: validate") {
		t.Fatalf("expected validate usage error, got: %v", err)
	}

	_, err = cmd.Handle(context.Background(), []string{"validate-all"})
	if err == nil || !strings.Contains(err.Error(), "usage: validate-all") {
		t.Fatalf("expected validate-all usage error, got: %v", err)
	}

	_, err = cmd.Handle(context.Background(), []string{"unknown"})
	if err == nil || !strings.Contains(err.Error(), "unsupported plugin subcommand") {
		t.Fatalf("expected unsupported error, got: %v", err)
	}
}
