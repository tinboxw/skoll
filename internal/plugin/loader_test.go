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
ui_menu:
  label: "Reports"
  label_zh_cn: "报表"
  label_en_us: "Reports"
  path: "/skoll/plugins/sample-plugin/reports"
  icon: "plugins"
  order: 120
  required_roles:
    - "manager"
  required_permissions:
    - "report.read"
config_schema:
  title: "Report Config"
  fields:
    - key: "report.default_range"
      label: "Default Range"
      type: "select"
      default: "week"
      required: true
      min_length: 3
      max_length: 12
      pattern: "^[a-z]+$"
      options:
        - value: "day"
          label: "Day"
        - value: "week"
          label: "Week"
    - key: "report.enabled"
      label: "Enabled"
      type: "boolean"
      default: "true"
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
	if len(info.PermissionResources) != 2 || info.PermissionResources[0].Key != "user:read" || info.PermissionResources[0].Type != "api" {
		t.Fatalf("unexpected permission resources: %+v", info.PermissionResources)
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
	if info.UIMenu == nil {
		t.Fatalf("expected ui menu parsed")
	}
	if info.UIMenu.LabelZhCN != "报表" || info.UIMenu.Path != "/skoll/plugins/sample-plugin/reports" || info.UIMenu.Order != 120 {
		t.Fatalf("unexpected ui menu: %+v", info.UIMenu)
	}
	if len(info.UIMenu.RequiredPermissions) != 1 || info.UIMenu.RequiredPermissions[0] != "report.read" {
		t.Fatalf("unexpected ui menu permissions: %+v", info.UIMenu.RequiredPermissions)
	}
	if info.ConfigSchema == nil || len(info.ConfigSchema.Fields) != 2 {
		t.Fatalf("expected config schema fields parsed, got %+v", info.ConfigSchema)
	}
	if info.ConfigSchema.Fields[0].Key != "report.default_range" || len(info.ConfigSchema.Fields[0].Options) != 2 {
		t.Fatalf("unexpected config schema first field: %+v", info.ConfigSchema.Fields[0])
	}
	if info.ConfigSchema.Fields[0].MinLength == nil || *info.ConfigSchema.Fields[0].MinLength != 3 || info.ConfigSchema.Fields[0].Pattern != "^[a-z]+$" {
		t.Fatalf("unexpected config schema validation rules: %+v", info.ConfigSchema.Fields[0])
	}
	if info.ConfigSchema.Fields[1].Key != "report.enabled" || info.ConfigSchema.Fields[1].Type != "boolean" {
		t.Fatalf("unexpected config schema second field: %+v", info.ConfigSchema.Fields[1])
	}
}

func TestFileLoaderLoadStructuredPermissions(t *testing.T) {
	dir := t.TempDir()
	manifest := `id: "report-plugin"
name: "Report Plugin"
version: "1.0.0"
permissions:
  - key: "report.read"
    type: "api"
    module: "report"
    name: "Read reports"
    risk: "medium"
    metadata.owner: "analytics"
  - key: "report.export"
    type: "button"
    module: "report"
    name: "Export reports"
`

	path := filepath.Join(dir, "plugin.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	info, err := NewFileLoader().Load(dir)
	if err != nil {
		t.Fatalf("loader.Load error: %v", err)
	}
	if len(info.Permissions) != 2 || info.Permissions[0] != "report.read" || info.Permissions[1] != "report.export" {
		t.Fatalf("unexpected permissions: %+v", info.Permissions)
	}
	if len(info.PermissionResources) != 2 {
		t.Fatalf("unexpected permission resources: %+v", info.PermissionResources)
	}
	first := info.PermissionResources[0]
	if first.Key != "report.read" || first.Type != "api" || first.Module != "report" || first.Name != "Read reports" || first.Risk != "medium" {
		t.Fatalf("unexpected first permission resource: %+v", first)
	}
	if first.Metadata["owner"] != "analytics" {
		t.Fatalf("unexpected permission metadata: %+v", first.Metadata)
	}
	second := info.PermissionResources[1]
	if second.Type != "button" || second.Risk != "low" {
		t.Fatalf("expected default risk on second resource, got %+v", second)
	}
}

func TestFileLoaderRejectsInvalidPermissions(t *testing.T) {
	cases := map[string]string{
		"blank key": `permissions:
  - ""
`,
		"duplicate key": `permissions:
  - "report.read"
  - key: "report.read"
`,
		"invalid type": `permissions:
  - key: "report.read"
    type: "legacy"
`,
	}

	for name, permissionsBlock := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			manifest := `id: "bad-plugin"
name: "Bad Plugin"
version: "1.0.0"
` + permissionsBlock
			if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o600); err != nil {
				t.Fatalf("write manifest: %v", err)
			}
			if _, err := NewFileLoader().Load(dir); err == nil {
				t.Fatal("expected manifest validation error")
			}
		})
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
