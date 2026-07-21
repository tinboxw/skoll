package plugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigratorPlanApplyRollbackIsIdempotent(t *testing.T) {
	pluginDir := setupPluginMigrations(t)
	store := newTestMigrationStore()
	m := NewMigrator("demo", pluginDir, "migrations", store)

	plan, err := m.Plan(context.Background())
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	if len(plan.Applied) != 0 || len(plan.Pending) != 2 {
		t.Fatalf("unexpected initial plan: applied=%d pending=%d", len(plan.Applied), len(plan.Pending))
	}

	applied, err := m.Apply(context.Background(), 0)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(applied) != 2 || len(store.records) != 2 || len(store.executed) != 2 {
		t.Fatalf("unexpected apply result: steps=%+v records=%+v sql=%+v", applied, store.records, store.executed)
	}
	repeated, err := m.Apply(context.Background(), 0)
	if err != nil || len(repeated) != 0 || len(store.executed) != 2 {
		t.Fatalf("repeated apply must be idempotent: steps=%+v err=%v sql=%+v", repeated, err, store.executed)
	}

	rolled, err := m.Rollback(context.Background(), 0)
	if err != nil {
		t.Fatalf("rollback error: %v", err)
	}
	if len(rolled) != 2 || rolled[0].Version != 2 || rolled[1].Version != 1 || len(store.records) != 0 {
		t.Fatalf("unexpected rollback result: %+v ledger=%+v", rolled, store.records)
	}
}

func TestMigratorApplyFailureRollsBackSQLAndLedger(t *testing.T) {
	pluginDir := setupPluginMigrations(t)
	store := newTestMigrationStore()
	store.failContains = "create table users"
	m := NewMigrator("demo", pluginDir, "migrations", store)

	if _, err := m.Apply(context.Background(), 0); err == nil {
		t.Fatal("expected migration failure")
	}
	if len(store.records) != 0 || len(store.executed) != 0 {
		t.Fatalf("failed transaction leaked state: ledger=%+v sql=%+v", store.records, store.executed)
	}
}

func TestMigratorCompensatesNonTransactionalDDLFailure(t *testing.T) {
	pluginDir := setupPluginMigrations(t)
	store := newTestMigrationStore()
	store.requiresCompensation = true
	store.failContains = "create table users"
	m := NewMigrator("demo", pluginDir, "migrations", store)

	if _, err := m.Apply(context.Background(), 0); err == nil || strings.Contains(err.Error(), "compensation failed") {
		t.Fatalf("migration must fail after successful compensation without reporting compensation failure: %v", err)
	}
	if len(store.records) != 0 {
		t.Fatalf("compensation leaked ledger records: %+v", store.records)
	}
	foundDrop := false
	for _, sql := range store.executed {
		if strings.Contains(strings.ToLower(sql), "drop table t1") {
			foundDrop = true
		}
	}
	if !foundDrop {
		t.Fatalf("compensation did not execute reverse down SQL: %+v", store.executed)
	}
}

func TestMigratorRejectsAppliedChecksumDrift(t *testing.T) {
	pluginDir := setupPluginMigrations(t)
	store := newTestMigrationStore()
	m := NewMigrator("demo", pluginDir, "migrations", store)
	if _, err := m.Apply(context.Background(), 1); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "migrations", "001_init.up.sql"), []byte("create table changed(id int);"), 0o644); err != nil {
		t.Fatalf("rewrite migration: %v", err)
	}
	if _, err := m.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum drift error, got %v", err)
	}
}

func TestMigrationPlannerRequiresUpDownPair(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "001_init.up.sql"), []byte("create table t(id int);"), 0o644); err != nil {
		t.Fatalf("write up sql: %v", err)
	}

	if _, err := NewMigrationPlanner(dir, "migrations").Plan(nil); err == nil || !strings.Contains(err.Error(), "requires both up and down") {
		t.Fatalf("expected up/down pair error, got: %v", err)
	}
}

type testMigrationStore struct {
	records              []MigrationRecord
	executed             []string
	failContains         string
	requiresCompensation bool
}

type testMigrationTransaction struct {
	store    *testMigrationStore
	records  []MigrationRecord
	executed []string
}

func newTestMigrationStore() *testMigrationStore {
	return &testMigrationStore{records: []MigrationRecord{}, executed: []string{}}
}

func (s *testMigrationStore) ListApplied(_ context.Context, pluginID string) ([]MigrationRecord, error) {
	out := make([]MigrationRecord, 0, len(s.records))
	for _, record := range s.records {
		if record.PluginID == pluginID {
			out = append(out, record)
		}
	}
	return out, nil
}

func (s *testMigrationStore) WithTransaction(_ context.Context, fn func(tx MigrationTransaction) error) error {
	tx := &testMigrationTransaction{
		store:    s,
		records:  append([]MigrationRecord(nil), s.records...),
		executed: append([]string(nil), s.executed...),
	}
	if err := fn(tx); err != nil {
		if s.requiresCompensation {
			s.records = tx.records
			s.executed = tx.executed
		}
		return err
	}
	s.records = tx.records
	s.executed = tx.executed
	return nil
}

func (s *testMigrationStore) RequiresApplyCompensation() bool { return s.requiresCompensation }

func (tx *testMigrationTransaction) ExecSQL(sql string) error {
	if tx.store.failContains != "" && strings.Contains(strings.ToLower(sql), strings.ToLower(tx.store.failContains)) {
		return errors.New("injected sql failure")
	}
	tx.executed = append(tx.executed, sql)
	return nil
}

func (tx *testMigrationTransaction) MarkApplied(record MigrationRecord) error {
	tx.records = append(tx.records, record)
	return nil
}

func (tx *testMigrationTransaction) RemoveApplied(pluginID string, version int) error {
	for i, record := range tx.records {
		if record.PluginID == pluginID && record.Version == version {
			tx.records = append(tx.records[:i], tx.records[i+1:]...)
			return nil
		}
	}
	return nil
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
