package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPluginMigrationHookInstallAndUninstallTraceState(t *testing.T) {
	dir := setupMigrationHookPlugin(t)
	recorder := NewMemoryPluginMigrationRecorder()
	hook := NewPluginMigrationHook(newTestMigrationStore(), recorder)
	baseTime := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	ticks := 0
	hook.now = func() time.Time {
		ticks++
		return baseTime.Add(time.Duration(ticks) * time.Second)
	}

	installed, err := hook.Run(context.Background(), PluginMigrationHookInput{
		PluginID:  "demo",
		PluginDir: dir,
		Action:    PluginMigrationInstall,
		ToVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("install migration hook: %v", err)
	}
	if len(installed) != 2 {
		t.Fatalf("expected two install steps, got %+v", installed)
	}

	rolledBack, err := hook.Run(context.Background(), PluginMigrationHookInput{
		PluginID:        "demo",
		PluginDir:       dir,
		Action:          PluginMigrationUninstall,
		FromVersion:     "1.0.0",
		UninstallPolicy: DataUninstallDrop,
	})
	if err != nil {
		t.Fatalf("uninstall migration hook: %v", err)
	}
	if len(rolledBack) != 2 || rolledBack[0].Version != 2 || rolledBack[1].Version != 1 {
		t.Fatalf("expected rollback in reverse version order, got %+v", rolledBack)
	}

	events := recorder.Events()
	if len(events) != 4 {
		t.Fatalf("expected started/succeeded for install and uninstall, got %+v", events)
	}
	if events[0].Action != PluginMigrationInstall || events[0].Status != PluginMigrationStarted {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if events[1].Action != PluginMigrationInstall || events[1].Status != PluginMigrationSucceeded || len(events[1].Steps) != 2 {
		t.Fatalf("unexpected install success event: %+v", events[1])
	}
	if events[3].Action != PluginMigrationUninstall || events[3].Status != PluginMigrationSucceeded || len(events[3].Steps) != 2 {
		t.Fatalf("unexpected uninstall success event: %+v", events[3])
	}
}

func TestPluginMigrationHookUpgradeDowngradeLimit(t *testing.T) {
	dir := setupMigrationHookPlugin(t)
	recorder := NewMemoryPluginMigrationRecorder()
	hook := NewPluginMigrationHook(newTestMigrationStore(), recorder)

	upgraded, err := hook.Run(context.Background(), PluginMigrationHookInput{
		PluginID:    "demo",
		PluginDir:   dir,
		Action:      PluginMigrationUpgrade,
		FromVersion: "1.0.0",
		ToVersion:   "1.1.0",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("upgrade migration hook: %v", err)
	}
	if len(upgraded) != 1 || upgraded[0].Version != 1 {
		t.Fatalf("expected one upgrade step, got %+v", upgraded)
	}

	downgraded, err := hook.Run(context.Background(), PluginMigrationHookInput{
		PluginID:       "demo",
		PluginDir:      dir,
		Action:         PluginMigrationDowngrade,
		FromVersion:    "1.1.0",
		ToVersion:      "1.0.0",
		Limit:          1,
		RollbackPolicy: DataRollbackAutomatic,
	})
	if err != nil {
		t.Fatalf("downgrade migration hook: %v", err)
	}
	if len(downgraded) != 1 || downgraded[0].Version != 1 {
		t.Fatalf("expected one downgrade step, got %+v", downgraded)
	}
}

func TestPluginMigrationHookRecordsFailure(t *testing.T) {
	dir := t.TempDir()
	recorder := NewMemoryPluginMigrationRecorder()
	hook := NewPluginMigrationHook(newTestMigrationStore(), recorder)

	_, err := hook.Run(context.Background(), PluginMigrationHookInput{
		PluginID:  "broken",
		PluginDir: dir,
		Action:    PluginMigrationInstall,
	})
	if err == nil {
		t.Fatalf("expected migration hook failure")
	}
	events := recorder.Events()
	if len(events) != 2 {
		t.Fatalf("expected started and failed events, got %+v", events)
	}
	if events[1].Status != PluginMigrationFailed || !strings.Contains(events[1].Error, "migrations") {
		t.Fatalf("expected failed event with migrations error, got %+v", events[1])
	}
}

func setupMigrationHookPlugin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	migrations := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrations, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	files := map[string]string{
		"001_create_demo.up.sql":   "create table demo(id text);\n",
		"001_create_demo.down.sql": "drop table demo;\n",
		"002_add_name.up.sql":      "alter table demo add column name text;\n",
		"002_add_name.down.sql":    "alter table demo drop column name;\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(migrations, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write migration %s: %v", name, err)
		}
	}
	return dir
}
