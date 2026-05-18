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
