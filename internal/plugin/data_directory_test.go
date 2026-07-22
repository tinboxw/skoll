package plugin

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPluginDataDirectoriesKeepStableIdentityAcrossPackageChanges(t *testing.T) {
	directories := mustPluginDataDirectories(t)
	first, err := directories.Prepare("equipment_maintenance")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(first, "repository.db")
	if err := os.WriteFile(marker, []byte("v1-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	second, err := directories.Prepare("equipment_maintenance")
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("data directory changed across package lifecycle: %q != %q", second, first)
	}
	if raw, err := os.ReadFile(filepath.Join(second, "repository.db")); err != nil || string(raw) != "v1-data" {
		t.Fatalf("plugin-owned data was not retained: value=%q err=%v", raw, err)
	}
}

func TestPluginDataDirectoriesApplyDeclaredUninstallPolicy(t *testing.T) {
	for _, policy := range []DataUninstallPolicy{DataUninstallRetain, DataUninstallArchive, DataUninstallDrop} {
		t.Run(string(policy), func(t *testing.T) {
			directories := mustPluginDataDirectories(t)
			directories.nowFn = func() time.Time { return time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC) }
			active, err := directories.Prepare("policy_test")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(active, "owned-data"), []byte("data"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := directories.Uninstall("policy_test", policy); err != nil {
				t.Fatal(err)
			}
			_, activeErr := os.Stat(filepath.Join(active, "owned-data"))
			if got := activeErr == nil; got != (policy == DataUninstallRetain) {
				t.Fatalf("active data exists=%v policy=%s", got, policy)
			}
			archive := filepath.Join(directories.root, pluginDataArchiveDirectory, "policy_test", "20260722T080000.000000000Z", "owned-data")
			_, archiveErr := os.Stat(archive)
			if got := archiveErr == nil; got != (policy == DataUninstallArchive) {
				t.Fatalf("archived data exists=%v policy=%s", got, policy)
			}
		})
	}
}

func TestPluginDataDirectoriesRejectInvalidIdentityAndImplicitPolicy(t *testing.T) {
	directories := mustPluginDataDirectories(t)
	if _, err := directories.Prepare("../escape"); err == nil {
		t.Fatal("expected invalid plugin identity to fail")
	}
	if err := directories.Uninstall("valid_plugin", ""); err == nil {
		t.Fatal("expected implicit uninstall policy to fail")
	}
}
