package pluginmgr

import "testing"

func TestServiceInstallEnableDisable(t *testing.T) {
	svc := NewService()
	installed := svc.Install("audit-ext", "1.0.0", []string{"on_boot", "on_shutdown"})
	if !installed.Enabled {
		t.Fatalf("expected plugin enabled on install")
	}

	disabled, err := svc.Disable("audit-ext")
	if err != nil {
		t.Fatalf("disable plugin failed: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("expected plugin disabled")
	}

	enabled, err := svc.Enable("audit-ext")
	if err != nil {
		t.Fatalf("enable plugin failed: %v", err)
	}
	if !enabled.Enabled {
		t.Fatalf("expected plugin enabled")
	}
}

func TestServiceInstallPackageAndVersionCheck(t *testing.T) {
	svc := NewService()

	installed, err := svc.InstallPackage("billing-ext", "1.2.3", "https://example.com/plugins/billing-ext-1.2.3.tgz", "sha256:abc123", []string{"on_boot"})
	if err != nil {
		t.Fatalf("install package failed: %v", err)
	}
	if installed.PackageURL == "" || installed.PackageHash == "" {
		t.Fatalf("expected package metadata to be recorded")
	}

	result, err := svc.CheckVersion("billing-ext", "1.3.0")
	if err != nil {
		t.Fatalf("check version failed: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("expected update to be available")
	}

	upToDate, err := svc.CheckVersion("billing-ext", "1.2.3")
	if err != nil {
		t.Fatalf("check version failed: %v", err)
	}
	if upToDate.UpdateAvailable {
		t.Fatalf("expected no update for same version")
	}
}
