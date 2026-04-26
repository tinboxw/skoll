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
