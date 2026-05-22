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
		ID:                 "oa-plugin",
		Name:               "OA Plugin",
		Version:            "1.0.0",
		APIVersion:         "v1",
		CompatibilitySkoll: ">=1.0.0 <2.0.0",
		ServiceBaseURL:     "https://oa.example.com",
		ServiceHealthURL:   "https://oa.example.com/health",
		MigrationVersion:   "v1.2.3",
	}
	if err := valid.ValidateManifest(); err != nil {
		t.Fatalf("expected valid contract fields, got %v", err)
	}

	badAPIVersion := valid
	badAPIVersion.APIVersion = "1"
	if err := badAPIVersion.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for invalid api_version")
	}

	missingCompat := valid
	missingCompat.CompatibilitySkoll = ""
	if err := missingCompat.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error when api_version is set without compatibility")
	}

	badURL := valid
	badURL.ServiceBaseURL = "oa.example.com"
	if err := badURL.ValidateManifest(); err == nil {
		t.Fatalf("expected validation error for invalid service_base_url")
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
