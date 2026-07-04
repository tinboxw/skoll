package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallPreflightServicePassesWithCompleteImpactSummary(t *testing.T) {
	root := writePreflightPlugin(t, preflightPluginFixture{
		id:             "reports",
		migration:      "v1.0.0",
		withMenu:       true,
		withConfig:     true,
		withNetwork:    true,
		withSignature:  true,
		withMigrations: true,
		withData:       true,
		withAPI:        true,
	})

	result, err := NewInstallPreflightService(nil).Check(InstallPreflightInput{
		Path:        root,
		CoreVersion: "1.2.0",
		Installed: []Info{
			{
				ID:      "audit",
				Name:    "Audit",
				Version: "1.0.0",
				PermissionResources: []PermissionDeclaration{
					{Key: "audit.read", Type: "api", Module: "audit", Name: "Read audit", Risk: "low"},
				},
				UIMenu: &UIMenu{Key: "plugin.audit", Label: "Audit", Path: "/skoll/audit"},
				State:  StateEnabled,
			},
		},
	})
	if err != nil {
		t.Fatalf("preflight check: %v", err)
	}
	if result.Status != InstallPreflightStatusPass {
		t.Fatalf("expected pass, got %+v", result)
	}
	if len(result.Permissions.Add) != 2 || result.Permissions.Add[0].Key != "reports.export" || result.Permissions.Add[1].Key != "reports.orders.read" {
		t.Fatalf("expected permission add diff, got %+v", result.Permissions)
	}
	if len(result.Menus.Add) != 1 || result.Menus.Add[0].Key != "plugin.reports" {
		t.Fatalf("expected menu add diff, got %+v", result.Menus)
	}
	if !result.Config.HasSchema || result.Config.FieldCount != 1 || len(result.Config.RequiredFields) != 1 {
		t.Fatalf("expected config schema summary, got %+v", result.Config)
	}
	if len(result.Migration.Pending) != 1 || result.Migration.Pending[0].Version != 1 {
		t.Fatalf("expected pending migration, got %+v", result.Migration)
	}
	if result.Data.Namespace != "reports" || len(result.Data.Tables) != 1 || len(result.Data.Tables[0].Indexes) != 1 {
		t.Fatalf("expected data contract summary, got %+v", result.Data)
	}
	if len(result.API.Routes) != 1 || result.API.Routes[0].Permission != "reports.orders.read" {
		t.Fatalf("expected api contract summary, got %+v", result.API)
	}
	if len(result.API.AuditActions) != 1 || result.API.AuditActions[0] != "reports.orders.read" || len(result.API.OpenAPIPaths) != 1 {
		t.Fatalf("expected api audit/openapi preview, got %+v", result.API)
	}
	if result.Signature.Status != "signed" {
		t.Fatalf("expected signed status, got %+v", result.Signature)
	}
	if result.Risk.Level != "high" {
		t.Fatalf("expected high risk from permission/network, got %+v", result.Risk)
	}
}

func TestInstallPreflightServiceBlocksPermissionAndMenuConflicts(t *testing.T) {
	root := writePreflightPlugin(t, preflightPluginFixture{
		id:       "reports",
		withMenu: true,
	})

	result, err := NewInstallPreflightService(nil).Check(InstallPreflightInput{
		Path: root,
		Installed: []Info{
			{
				ID:      "legacy",
				Name:    "Legacy",
				Version: "1.0.0",
				PermissionResources: []PermissionDeclaration{
					{Key: "reports.export", Type: "api", Module: "legacy", Name: "Legacy export", Risk: "low"},
				},
				UIMenu: &UIMenu{Key: "plugin.reports", Label: "Legacy Reports", Path: "/skoll/legacy-reports"},
				State:  StateEnabled,
			},
		},
	})
	if err != nil {
		t.Fatalf("preflight check: %v", err)
	}
	if result.Status != InstallPreflightStatusBlocked {
		t.Fatalf("expected blocked, got %+v", result)
	}
	if len(result.Permissions.Conflict) != 1 || len(result.Menus.Conflict) != 1 {
		t.Fatalf("expected permission and menu conflicts, got permissions=%+v menus=%+v", result.Permissions, result.Menus)
	}
	if result.Risk.Level != "critical" {
		t.Fatalf("expected critical risk for blockers, got %+v", result.Risk)
	}
}

func TestInstallPreflightServiceBlocksInvalidMigrationPlan(t *testing.T) {
	root := writePreflightPlugin(t, preflightPluginFixture{
		id:        "reports",
		migration: "v1.0.0",
	})

	result, err := NewInstallPreflightService(nil).Check(InstallPreflightInput{Path: root})
	if err != nil {
		t.Fatalf("preflight check: %v", err)
	}
	if result.Status != InstallPreflightStatusBlocked {
		t.Fatalf("expected blocked, got %+v", result)
	}
	if result.Migration.Error == "" {
		t.Fatalf("expected migration error, got %+v", result.Migration)
	}
}

type preflightPluginFixture struct {
	id             string
	migration      string
	withMenu       bool
	withConfig     bool
	withNetwork    bool
	withSignature  bool
	withMigrations bool
	withData       bool
	withAPI        bool
}

func writePreflightPlugin(t *testing.T, fixture preflightPluginFixture) string {
	t.Helper()
	if fixture.id == "" {
		fixture.id = "reports"
	}
	root := filepath.Join(t.TempDir(), fixture.id)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir plugin: %v", err)
	}
	manifest := "id: " + fixture.id + "\n" +
		"name: Reports\n" +
		"version: 1.0.0\n" +
		"api_version: v1\n" +
		"compatibility_skoll: \">=1.0.0 <2.0.0\"\n" +
		"ui_mode: frontend_only\n" +
		"i18n_locales:\n" +
		"  - zh-CN\n" +
		"permissions:\n" +
		"  - key: reports.export\n" +
		"    type: api\n" +
		"    module: reports\n" +
		"    name: Export reports\n" +
		"    risk: high\n"
	if fixture.migration != "" {
		manifest += "migration_version: " + fixture.migration + "\n"
	}
	if fixture.withNetwork {
		manifest += "service_base_url: https://reports.example.com\n"
	}
	if fixture.withMenu {
		manifest += "ui_menu:\n" +
			"  key: plugin.reports\n" +
			"  label: Reports\n" +
			"  path: /skoll/plugins/reports\n" +
			"  required_permissions:\n" +
			"    - reports.export\n"
	}
	if fixture.withConfig {
		manifest += "config_schema:\n" +
			"  title: Reports settings\n" +
			"  fields:\n" +
			"    - key: endpoint\n" +
			"      label: Endpoint\n" +
			"      type: string\n" +
			"      required: true\n" +
			"      default: https://reports.example.com\n"
	}
	if fixture.withData {
		manifest += "data:\n" +
			"  namespace: reports\n" +
			"  migration_version: v1.0.0\n" +
			"  migration_directory: migrations\n" +
			"  uninstall_policy: retain\n" +
			"  rollback_policy: manual\n" +
			"  tables:\n" +
			"    - name: reports_orders\n" +
			"      primary_key: id\n" +
			"      columns: id, code, status\n" +
			"      indexes: idx_reports_orders_status(status)\n"
	}
	if fixture.withAPI {
		manifest += "api:\n" +
			"  routes:\n" +
			"    - method: GET\n" +
			"      path: /v1/plugins/reports/api/orders\n" +
			"      summary: List orders\n" +
			"      permission: reports.orders.read\n" +
			"      audit_action: reports.orders.read\n"
	}
	if fixture.withSignature {
		manifest += "sign_algo: RSA-SHA256\n" +
			"sign_timestamp: 2026-06-29T00:00:00Z\n" +
			"sign_value: signed\n" +
			"vendor_id: skoll\n"
	}
	if err := os.WriteFile(filepath.Join(root, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if fixture.withMigrations {
		migrations := filepath.Join(root, "migrations")
		if err := os.MkdirAll(migrations, 0o755); err != nil {
			t.Fatalf("mkdir migrations: %v", err)
		}
		if err := os.WriteFile(filepath.Join(migrations, "001_create_reports.up.sql"), []byte("create table reports(id text);\n"), 0o644); err != nil {
			t.Fatalf("write up migration: %v", err)
		}
		if err := os.WriteFile(filepath.Join(migrations, "001_create_reports.down.sql"), []byte("drop table reports;\n"), 0o644); err != nil {
			t.Fatalf("write down migration: %v", err)
		}
	}
	return root
}
