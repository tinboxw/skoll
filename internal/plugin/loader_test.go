package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileLoaderLoad(t *testing.T) {
	dir := t.TempDir()
	manifest := `id: "sample-plugin"
name: "Sample Plugin"
version: "1.0.0"
description: "plugin for tests"
level: "app"
app_id: "crm"
mount_policy: "user"
dependencies:
  - id: "base-auth"
    version: ">=2.0.0"
permissions:
  - "user:read"
  - "role:manage"
ui_mode: "separated"
frontend_entry: "/plugins/sample-plugin"
`

	path := filepath.Join(dir, "plugin.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	loader := NewFileLoader()
	info, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("loader.Load error: %v", err)
	}

	if info.ID != "sample-plugin" {
		t.Fatalf("unexpected plugin id: %s", info.ID)
	}
	if len(info.Dependencies) != 1 || info.Dependencies[0].ID != "base-auth" {
		t.Fatalf("unexpected dependencies: %+v", info.Dependencies)
	}
	if len(info.Permissions) != 2 {
		t.Fatalf("unexpected permissions: %+v", info.Permissions)
	}
	if info.UIMode != UIModeSeparated {
		t.Fatalf("unexpected ui mode: %s", info.UIMode)
	}
	if info.Level != LevelApp {
		t.Fatalf("unexpected level: %s", info.Level)
	}
	if info.AppID != "crm" {
		t.Fatalf("unexpected app id: %s", info.AppID)
	}
	if info.MountPolicy != MountPolicyUser {
		t.Fatalf("unexpected mount policy: %s", info.MountPolicy)
	}
	if info.FrontendEntry != "/plugins/sample-plugin" {
		t.Fatalf("unexpected frontend entry: %s", info.FrontendEntry)
	}
}

func TestFileLoaderLoadFrontendOnly(t *testing.T) {
	dir := t.TempDir()
	manifest := `id: "frontend-only-plugin"
name: "Frontend Only Plugin"
version: "1.0.0"
ui_mode: "frontend_only"
frontend_entry: "/plugins/frontend-only-plugin"
`

	path := filepath.Join(dir, "plugin.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	loader := NewFileLoader()
	info, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("loader.Load error: %v", err)
	}

	if info.UIMode != UIModeFrontendOnly {
		t.Fatalf("unexpected ui mode: %s", info.UIMode)
	}
	if info.FrontendEntry != "/plugins/frontend-only-plugin" {
		t.Fatalf("unexpected frontend entry: %s", info.FrontendEntry)
	}
}

func TestFileLoaderLoadDerivesFrontendEntryByLevel(t *testing.T) {
	dir := t.TempDir()
	manifest := `id: "crm-report"
name: "CRM Report"
version: "1.0.0"
level: "app"
app_id: "crm"
ui_mode: "frontend_only"
`

	path := filepath.Join(dir, "plugin.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	loader := NewFileLoader()
	info, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("loader.Load error: %v", err)
	}

	if info.FrontendEntry != "/crm" {
		t.Fatalf("unexpected derived frontend entry: %s", info.FrontendEntry)
	}
}
