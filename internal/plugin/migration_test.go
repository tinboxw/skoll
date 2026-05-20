package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigratorPlanApplyRollback(t *testing.T) {
	pluginDir := setupPluginMigrations(t)
	m := NewMigrator(pluginDir)

	plan, err := m.Plan()
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	if len(plan.Applied) != 0 || len(plan.Pending) != 2 {
		t.Fatalf("unexpected initial plan: applied=%d pending=%d", len(plan.Applied), len(plan.Pending))
	}

	applied, err := m.Apply(1)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(applied) != 1 || applied[0].Version != 1 {
		t.Fatalf("unexpected apply result: %+v", applied)
	}

	plan, err = m.Plan()
	if err != nil {
		t.Fatalf("plan error after apply: %v", err)
	}
	if len(plan.Applied) != 1 || len(plan.Pending) != 1 {
		t.Fatalf("unexpected plan after apply: applied=%d pending=%d", len(plan.Applied), len(plan.Pending))
	}

	rolled, err := m.Rollback(1)
	if err != nil {
		t.Fatalf("rollback error: %v", err)
	}
	if len(rolled) != 1 || rolled[0].Version != 1 {
		t.Fatalf("unexpected rollback result: %+v", rolled)
	}

	plan, err = m.Plan()
	if err != nil {
		t.Fatalf("plan error after rollback: %v", err)
	}
	if len(plan.Applied) != 0 || len(plan.Pending) != 2 {
		t.Fatalf("unexpected plan after rollback: applied=%d pending=%d", len(plan.Applied), len(plan.Pending))
	}
}

func TestMigratorRequiresUpDownPair(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "001_init.up.sql"), []byte("create table t(id int);"), 0o644); err != nil {
		t.Fatalf("write up sql: %v", err)
	}

	m := NewMigrator(dir)
	if _, err := m.Plan(); err == nil || !strings.Contains(err.Error(), "requires both up and down") {
		t.Fatalf("expected up/down pair error, got: %v", err)
	}
}

func setupPluginMigrations(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	files := map[string]string{
		"001_init.up.sql":    "create table t1(id int);",
		"001_init.down.sql":  "drop table t1;",
		"002_users.up.sql":   "create table users(id int);",
		"002_users.down.sql": "drop table users;",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(migrationsDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write migration %s: %v", name, err)
		}
	}
	return dir
}
