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
api_version: "v1"
compatibility_skoll: ">=1.0.0 <2.0.0"
service_base_url: "https://oa.example.com"
service_health_url: "https://oa.example.com/health"
migration_version: "v1.2.0"
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
ui_nav_position: "sidebar"
ui_open_mode: "integrated"
ui_tab_mode: "fixed"
i18n_locales:
	- "zh-CN"
	- "en-US"
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
	if info.APIVersion != "v1" {
		t.Fatalf("unexpected api version: %s", info.APIVersion)
	}
	if info.CompatibilitySkoll != ">=1.0.0 <2.0.0" {
		t.Fatalf("unexpected compatibility: %s", info.CompatibilitySkoll)
	}
	if info.ServiceBaseURL != "https://oa.example.com" {
		t.Fatalf("unexpected service base url: %s", info.ServiceBaseURL)
	}
	if info.ServiceHealthURL != "https://oa.example.com/health" {
		t.Fatalf("unexpected service health url: %s", info.ServiceHealthURL)
	}
	if info.MigrationVersion != "v1.2.0" {
		t.Fatalf("unexpected migration version: %s", info.MigrationVersion)
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
	if info.UINavPosition != UINavPositionSidebar {
		t.Fatalf("unexpected ui nav position: %s", info.UINavPosition)
	}
	if info.UIOpenMode != UIOpenModeIntegrated {
		t.Fatalf("unexpected ui open mode: %s", info.UIOpenMode)
	}
	if info.UITabMode != UITabModeFixed {
		t.Fatalf("unexpected ui tab mode: %s", info.UITabMode)
	}
	if len(info.I18nLocales) != 2 || info.I18nLocales[0] != "zh-CN" || info.I18nLocales[1] != "en-US" {
		t.Fatalf("unexpected i18n locales: %+v", info.I18nLocales)
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
	if info.UINavPosition != UINavPositionNone {
		t.Fatalf("unexpected default ui nav position: %s", info.UINavPosition)
	}
	if info.UIOpenMode != UIOpenModeIntegrated {
		t.Fatalf("unexpected default ui open mode: %s", info.UIOpenMode)
	}
	if info.UITabMode != UITabModeOptional {
		t.Fatalf("unexpected default ui tab mode: %s", info.UITabMode)
	}
	if len(info.I18nLocales) != 2 || info.I18nLocales[0] != "zh-CN" || info.I18nLocales[1] != "en-US" {
		t.Fatalf("unexpected default i18n locales: %+v", info.I18nLocales)
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

func TestFileLoaderLoadStandaloneDefaults(t *testing.T) {
	dir := t.TempDir()
	manifest := `id: "standalone-plugin"
name: "Standalone Plugin"
version: "1.0.0"
ui_mode: "frontend_only"
ui_open_mode: "standalone"
i18n_locales:
  - "zh-CN"
  - "en-US"
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

	if info.UIOpenMode != UIOpenModeStandalone {
		t.Fatalf("unexpected ui open mode: %s", info.UIOpenMode)
	}
	if info.UINavPosition != UINavPositionNone {
		t.Fatalf("unexpected ui nav position: %s", info.UINavPosition)
	}
	if info.UITabMode != UITabModeDisabled {
		t.Fatalf("unexpected ui tab mode: %s", info.UITabMode)
	}
}
