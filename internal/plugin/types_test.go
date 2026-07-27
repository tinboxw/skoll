package plugin

import "testing"

func TestResolveFrontendEntry(t *testing.T) {
	tests := []struct {
		name string
		in   Info
		want string
	}{
		{
			name: "system default",
			in:   Info{ID: "auth", Level: LevelSystem},
			want: "/skoll/plugins/auth",
		},
		{
			name: "app default",
			in:   Info{ID: "report", Level: LevelApp, AppID: "crm"},
			want: "/crm",
		},
		{
			name: "explicit entry keeps value",
			in:   Info{ID: "auth", FrontendEntry: "/plugins/custom-auth"},
			want: "/plugins/custom-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveFrontendEntry(tt.in)
			if got != tt.want {
				t.Fatalf("ResolveFrontendEntry()=%s, want %s", got, tt.want)
			}
		})
	}
}

func TestInfoValidateManifestLevelConstraints(t *testing.T) {
	valid := Info{ID: "crm-report", Name: "CRM Report", Version: "1.0.0", Level: LevelApp, AppID: "crm"}
	if err := valid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid manifest, got %v", err)
	}

	invalid := Info{ID: "crm-report", Name: "CRM Report", Version: "1.0.0", Level: LevelApp}
	if err := invalid.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for app level without app_id")
	}

	reserved := Info{ID: "crm-report", Name: "CRM Report", Version: "1.0.0", Level: LevelApp, AppID: "skoll"}
	if err := reserved.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for reserved app_id skoll")
	}

	systemWithAppID := Info{ID: "sys-plugin", Name: "System Plugin", Version: "1.0.0", Level: LevelSystem, AppID: "demo"}
	if err := systemWithAppID.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for system level with app_id")
	}
}

func TestInfoValidateManifestContractFields(t *testing.T) {
	valid := Info{
		ID:               "oa-plugin",
		Name:             "OA Plugin",
		Version:          "1.0.0",
		APIVersion:       "v1",
		ServiceBaseURL:   "http://127.0.0.1:18090",
		ServiceHealthURL: "http://127.0.0.1:18090/health",
		MigrationVersion: "v1.2.3",
	}
	if err := valid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid contract fields, got %v", err)
	}

	badAPIVersion := valid
	badAPIVersion.APIVersion = "1"
	if err := badAPIVersion.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for invalid api_version")
	}

	badURL := valid
	badURL.ServiceBaseURL = "oa.example.com"
	if err := badURL.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for invalid service_base_url")
	}

	missingHealth := valid
	missingHealth.ServiceHealthURL = ""
	if err := missingHealth.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error when service_health_url is missing")
	}

	missingBase := valid
	missingBase.ServiceBaseURL = ""
	if err := missingBase.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error when service_base_url is missing")
	}

	badMigration := valid
	badMigration.MigrationVersion = "v1"
	if err := badMigration.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for invalid migration_version")
	}
}

func TestInfoValidateManifestUIConfig(t *testing.T) {
	valid := Info{
		ID:            "crm-analytics",
		Name:          "CRM Analytics",
		Version:       "1.0.0",
		UIMode:        UIModeFrontendOnly,
		UINavPosition: UINavPositionSidebar,
		UIOpenMode:    UIOpenModeIntegrated,
		UITabMode:     UITabModeFixed,
		I18nLocales:   []string{"zh-CN", "en-US"},
	}
	if err := valid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid UI config, got %v", err)
	}

	badNav := valid
	badNav.UINavPosition = "left"
	if err := badNav.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for invalid ui_nav_position")
	}

	badOpenMode := valid
	badOpenMode.UIOpenMode = "popup"
	if err := badOpenMode.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for invalid ui_open_mode")
	}

	badTabMode := valid
	badTabMode.UITabMode = "always"
	if err := badTabMode.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for invalid ui_tab_mode")
	}

	missingLocales := valid
	missingLocales.I18nLocales = nil
	if err := missingLocales.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for missing i18n_locales on frontend plugin")
	}

	badLocale := valid
	badLocale.I18nLocales = []string{"zh_cn"}
	if err := badLocale.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for invalid locale format")
	}

	duplicatedLocale := valid
	duplicatedLocale.I18nLocales = []string{"en-US", "en-US"}
	if err := duplicatedLocale.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for duplicate locales")
	}

	standaloneValid := valid
	standaloneValid.UIOpenMode = UIOpenModeStandalone
	standaloneValid.UINavPosition = UINavPositionNone
	standaloneValid.UITabMode = UITabModeDisabled
	if err := standaloneValid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid standalone UI config, got %v", err)
	}

	standaloneWithSidebar := standaloneValid
	standaloneWithSidebar.UINavPosition = UINavPositionSidebar
	if err := standaloneWithSidebar.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for standalone plugin with non-none nav position")
	}

	standaloneWithTabs := standaloneValid
	standaloneWithTabs.UITabMode = UITabModeOptional
	if err := standaloneWithTabs.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for standalone plugin with enabled tab mode")
	}
}

func TestInfoValidateManifestUIMenu(t *testing.T) {
	valid := Info{
		ID:            "report-plugin",
		Name:          "Report Plugin",
		Version:       "1.0.0",
		UIMode:        UIModeFrontendOnly,
		UINavPosition: UINavPositionSidebar,
		I18nLocales:   []string{"zh-CN", "en-US"},
		UIMenu: &UIMenu{
			Label:               "Reports",
			Path:                "/skoll/plugins/report-plugin",
			Icon:                "plugins",
			Order:               120,
			RequiredRoles:       []string{"manager"},
			RequiredPermissions: []string{"report.read"},
		},
	}
	if err := valid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid ui menu, got %v", err)
	}

	badPath := valid
	badPath.UIMenu = &UIMenu{Path: "skoll/plugins/report-plugin"}
	if err := badPath.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for ui menu path without leading slash")
	}

	badPermission := valid
	badPermission.UIMenu = &UIMenu{RequiredPermissions: []string{""}}
	if err := badPermission.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for blank ui menu permission")
	}
}

func TestInfoValidateManifestPermissions(t *testing.T) {
	valid := Info{
		ID:          "report-plugin",
		Name:        "Report Plugin",
		Version:     "1.0.0",
		Permissions: []string{"report.read"},
		PermissionResources: []PermissionDeclaration{
			{Key: "report.read", Type: "api", Module: "report", Name: "Read reports", Risk: "low"},
		},
	}
	if err := valid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid permissions, got %v", err)
	}

	invalidKey := valid
	invalidKey.PermissionResources = []PermissionDeclaration{{Key: "", Type: "api"}}
	if err := invalidKey.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for blank permission key")
	}

	duplicate := valid
	duplicate.PermissionResources = []PermissionDeclaration{{Key: "report.read"}, {Key: "report.read"}}
	if err := duplicate.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for duplicate permission key")
	}

	invalidRisk := valid
	invalidRisk.PermissionResources = []PermissionDeclaration{{Key: "report.read", Risk: "warning"}}
	if err := invalidRisk.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for invalid permission risk")
	}
}

func TestInfoValidateManifestDataContract(t *testing.T) {
	valid := Info{
		ID:      "reports",
		Name:    "Reports",
		Version: "1.0.0",
		DataManifest: &DataManifest{
			Namespace:          "reports",
			MigrationVersion:   "v1.0.0",
			MigrationDirectory: "migrations",
			UninstallPolicy:    DataUninstallRetain,
			RollbackPolicy:     DataRollbackManual,
			Tables: []DataTable{
				{
					Name:       "orders",
					PrimaryKey: "id",
					Columns:    []string{"id", "code", "status"},
					Indexes: []DataIndex{
						{Name: "idx_reports_orders_code", Columns: []string{"code"}, Unique: true},
						{Name: "idx_reports_orders_status", Columns: []string{"status"}},
					},
				},
			},
		},
	}
	if err := valid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid data contract, got %v", err)
	}

	missingUninstallPolicy := valid
	missingUninstallPolicy.DataManifest = &DataManifest{Namespace: "reports", RollbackPolicy: DataRollbackManual}
	if err := missingUninstallPolicy.ValidateManifest(); err == nil {
		t.Fatal("expected data manifest without uninstall policy to fail")
	}

	missingRollbackPolicy := valid
	missingRollbackPolicy.DataManifest = &DataManifest{Namespace: "reports", UninstallPolicy: DataUninstallRetain}
	if err := missingRollbackPolicy.ValidateManifest(); err == nil {
		t.Fatal("expected data manifest without rollback policy to fail")
	}

	reservedNamespace := valid
	reservedNamespace.DataManifest = &DataManifest{Namespace: "sk_plugin", Tables: []DataTable{{Name: "sk_plugin_items", Columns: []string{"id"}}}}
	if err := reservedNamespace.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for reserved data namespace")
	}

	physicalTableName := valid
	physicalTableName.DataManifest = &DataManifest{Namespace: "reports", Tables: []DataTable{{Name: "reports_orders", Columns: []string{"id"}}}}
	if err := physicalTableName.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for physical table name")
	}

	unknownIndexColumn := valid
	unknownIndexColumn.DataManifest = &DataManifest{
		Namespace: "reports",
		Tables: []DataTable{
			{Name: "orders", Columns: []string{"id"}, Indexes: []DataIndex{{Name: "idx_reports_orders_status", Columns: []string{"status"}}}},
		},
	}
	if err := unknownIndexColumn.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for index column outside table columns")
	}
}

func TestInfoValidateManifestAPIContract(t *testing.T) {
	valid := Info{
		ID:      "reports",
		Name:    "Reports",
		Version: "1.0.0",
		APIContract: &APIContract{
			Routes: []APIRoute{
				{
					Method:      "GET",
					Path:        "/v1/plugins/reports/api/orders",
					Summary:     "List orders",
					Permission:  "reports.orders.read",
					AuditAction: "reports.orders.read",
				},
			},
		},
	}
	if err := valid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid api contract, got %v", err)
	}

	outsideNamespace := valid
	outsideNamespace.APIContract = &APIContract{Routes: []APIRoute{{Method: "GET", Path: "/v1/orders", Permission: "reports.orders.read"}}}
	if err := outsideNamespace.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for route outside plugin api namespace")
	}

	invalidPermission := valid
	invalidPermission.APIContract = &APIContract{Routes: []APIRoute{{Method: "GET", Path: "/v1/plugins/reports/api/orders", Permission: ""}}}
	if err := invalidPermission.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for missing route permission")
	}

	invalidAudit := valid
	invalidAudit.APIContract = &APIContract{Routes: []APIRoute{{Method: "GET", Path: "/v1/plugins/reports/api/orders", Permission: "reports.orders.read", AuditAction: "reports.read"}}}
	if err := invalidAudit.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for invalid audit action")
	}
}

func TestInfoValidateManifestEventContract(t *testing.T) {
	valid := Info{
		ID:      "reports",
		Name:    "Reports",
		Version: "1.0.0",
		EventContract: &EventContract{
			Publications: []EventPublication{
				{Name: "report-generated", SchemaVersion: 1, PayloadType: "report.generated", Scope: "tenant"},
			},
			Subscriptions: []EventSubscription{
				{Publisher: "skoll", Name: "approval-completed", SchemaVersions: []uint32{1}, Handler: "onApprovalCompleted", RetryPolicy: "standard"},
				{Publisher: "warehouse", Name: "inbound-completed", SchemaVersions: []uint32{1, 2}, Handler: "onInboundCompleted", RetryPolicy: "aggressive"},
				{Publisher: "compliance", Name: "qualification-expiring", SchemaVersions: []uint32{1}, Handler: "onQualificationExpiring", RetryPolicy: "none"},
			},
		},
	}
	if err := valid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid event contract, got %v", err)
	}
	if _, err := valid.AuthorizeEventPublication("report-generated", 1); err != nil {
		t.Fatalf("expected declared publication, got %v", err)
	}
	if _, err := valid.AuthorizeEventPublication("report-generated", 2); err == nil {
		t.Fatal("expected undeclared publication version to fail closed")
	}
	if _, err := valid.AuthorizeEventSubscription("warehouse", "inbound-completed", 2); err != nil {
		t.Fatalf("expected declared subscription, got %v", err)
	}
	if _, err := valid.AuthorizeEventSubscription("warehouse", "inbound-completed", 3); err == nil {
		t.Fatal("expected undeclared subscription version to fail closed")
	}

	invalidName := valid
	invalidName.EventContract = &EventContract{Subscriptions: []EventSubscription{{
		Publisher: "skoll", Name: "bad event", SchemaVersions: []uint32{1}, Handler: "onBadEvent", RetryPolicy: "standard",
	}}}
	if err := invalidName.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for invalid event name")
	}

	missingHandler := valid
	missingHandler.EventContract = &EventContract{Subscriptions: []EventSubscription{{
		Publisher: "skoll", Name: "approval-completed", SchemaVersions: []uint32{1}, RetryPolicy: "standard",
	}}}
	if err := missingHandler.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for missing event handler")
	}

	duplicate := valid
	duplicate.EventContract = &EventContract{Subscriptions: []EventSubscription{
		{Publisher: "skoll", Name: "approval-completed", SchemaVersions: []uint32{1}, Handler: "onApprovalCompleted", RetryPolicy: "standard"},
		{Publisher: "skoll", Name: "approval-completed", SchemaVersions: []uint32{1}, Handler: "onApprovalCompleted", RetryPolicy: "standard"},
	}}
	if err := duplicate.ValidateManifest(); err == nil {
		t.Fatal("expected validation error for duplicate event subscription")
	}
}
