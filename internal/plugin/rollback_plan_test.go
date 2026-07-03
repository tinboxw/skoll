package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginRollbackServiceBuildsConsistencyPlan(t *testing.T) {
	targetDir := t.TempDir()
	writeRollbackAssetFixture(t, targetDir)
	visible := true
	service := NewPluginRollbackService()

	plan, err := service.BuildPlan(PluginRollbackPlanInput{
		PluginID: "reports",
		From: Info{
			ID:               "reports",
			Version:          "1.2.0",
			ConfigJSON:       `{"rollout_percent":50,"theme":"dark"}`,
			MigrationVersion: "1.2.0",
			Permissions:      []string{"reports.read", "reports.export"},
			UIMenu: &UIMenu{
				Key:                 "plugin.reports",
				Path:                "/skoll/plugins/reports",
				Visible:             &visible,
				RequiredPermissions: []string{"reports.read"},
			},
			UIMode: UIModeMonolith,
			Source: targetDir,
		},
		Target: Info{
			ID:               "reports",
			Version:          "1.1.0",
			ConfigJSON:       `{"rollout_percent":10,"theme":"light"}`,
			MigrationVersion: "1.1.0",
			Permissions:      []string{"reports.read"},
			UIMenu: &UIMenu{
				Key:                 "plugin.reports",
				Path:                "/skoll/plugins/reports",
				Visible:             &visible,
				RequiredPermissions: []string{"reports.read"},
			},
			UIMode: UIModeMonolith,
			Source: targetDir,
		},
		TargetPercent: intPtr(10),
	})
	if err != nil {
		t.Fatalf("build rollback plan: %v", err)
	}
	if plan.Status != "passed" || plan.FromVersion != "1.2.0" || plan.ToVersion != "1.1.0" {
		t.Fatalf("unexpected plan summary: %+v", plan)
	}
	for _, name := range []string{"version", "menu", "permission", "config", "asset", "migration"} {
		if checkpoint := rollbackCheckpoint(plan, name); checkpoint.Status != "passed" {
			t.Fatalf("expected %s checkpoint to pass, got %+v", name, checkpoint)
		}
	}
	if checkpoint := rollbackCheckpoint(plan, "asset"); checkpoint.After == "" {
		t.Fatalf("expected asset checkpoint to list assets, got %+v", checkpoint)
	}
}

func TestPluginRollbackServiceBlocksMismatchedRollbackPercent(t *testing.T) {
	service := NewPluginRollbackService()

	plan, err := service.BuildPlan(PluginRollbackPlanInput{
		PluginID:      "reports",
		From:          Info{ID: "reports", Version: "1.2.0"},
		Target:        Info{ID: "reports", Version: "1.1.0", ConfigJSON: `{"rollout_percent":30}`},
		TargetPercent: intPtr(10),
	})
	if err != nil {
		t.Fatalf("build rollback plan: %v", err)
	}
	if plan.Status != "blocked" {
		t.Fatalf("expected blocked plan, got %+v", plan)
	}
	if checkpoint := rollbackCheckpoint(plan, "config"); checkpoint.Status != "blocked" {
		t.Fatalf("expected config checkpoint to block, got %+v", checkpoint)
	}
}

func rollbackCheckpoint(plan PluginRollbackPlan, name string) PluginRollbackCheckpoint {
	for _, checkpoint := range plan.Checkpoints {
		if checkpoint.Name == name {
			return checkpoint
		}
	}
	return PluginRollbackCheckpoint{}
}

func writeRollbackAssetFixture(t *testing.T, root string) {
	t.Helper()
	dist := filepath.Join(root, "frontend", "dist", "assets")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dist, "app.js"), []byte("console.log('rollback');\n"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
}

func intPtr(v int) *int {
	return &v
}
