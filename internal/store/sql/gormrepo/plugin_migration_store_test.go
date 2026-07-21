package gormrepo_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	mysqlstore "github.com/tinboxw/skoll/internal/store/sql/mysql"
	postgresstore "github.com/tinboxw/skoll/internal/store/sql/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPluginMigratorSQLiteExecutesAndRollsBack(t *testing.T) {
	db := openMigrationSQLite(t)
	exercisePluginMigrationLifecycle(t, db)
}

func TestPluginMigratorSQLiteFailureIsAtomic(t *testing.T) {
	db := openMigrationSQLite(t)
	pluginID := "broken_plugin"
	table := "broken_plugin_items"
	dir := writeMigrationFiles(t, map[string]string{
		"001_create.up.sql":   fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY);", table),
		"001_create.down.sql": fmt.Sprintf("DROP TABLE %s;", table),
		"002_fail.up.sql":     "INSERT INTO missing_plugin_table(id) VALUES (1);",
		"002_fail.down.sql":   "DELETE FROM missing_plugin_table WHERE id = 1;",
	})
	migrator := plugin.NewMigrator(pluginID, dir, "migrations", gormrepo.NewPluginMigrationStore(db))
	if _, err := migrator.Apply(context.Background(), 0); err == nil {
		t.Fatal("expected migration transaction failure")
	}
	if db.Migrator().HasTable(table) {
		t.Fatal("failed SQLite migration leaked a created table")
	}
	records, err := gormrepo.NewPluginMigrationStore(db).ListApplied(context.Background(), pluginID)
	if err != nil {
		t.Fatalf("list migration ledger: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("failed migration leaked ledger records: %+v", records)
	}
}

func TestPluginMigratorMySQLIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SKOLL_TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("SKOLL_TEST_MYSQL_DSN is not set")
	}
	adapter, err := mysqlstore.NewAdapter(dsn)
	if err != nil {
		t.Fatalf("open MySQL adapter: %v", err)
	}
	exercisePluginMigrationLifecycle(t, adapter.DB())
	exercisePluginMigrationFailureCleanup(t, adapter.DB())
}

func TestPluginMigratorPostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SKOLL_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("SKOLL_TEST_POSTGRES_DSN is not set")
	}
	adapter, err := postgresstore.NewAdapter(dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL adapter: %v", err)
	}
	exercisePluginMigrationLifecycle(t, adapter.DB())
	exercisePluginMigrationFailureCleanup(t, adapter.DB())
}

func exercisePluginMigrationLifecycle(t *testing.T, db *gorm.DB) {
	t.Helper()
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	pluginID := "migration_" + suffix
	table := pluginID + "_items"
	dir := writeMigrationFiles(t, map[string]string{
		"001_create.up.sql":   fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY);", table),
		"001_create.down.sql": fmt.Sprintf("DROP TABLE %s;", table),
		"002_seed.up.sql":     fmt.Sprintf("INSERT INTO %s (id) VALUES (1);", table),
		"002_seed.down.sql":   fmt.Sprintf("DELETE FROM %s WHERE id = 1;", table),
	})
	migrator := plugin.NewMigrator(pluginID, dir, "migrations", gormrepo.NewPluginMigrationStore(db))
	t.Cleanup(func() {
		_ = db.Exec("DROP TABLE IF EXISTS " + table).Error
		_ = db.Where("plugin_id = ?", pluginID).Delete(&gormrepo.PluginMigrationModel{}).Error
	})

	applied, err := migrator.Apply(context.Background(), 0)
	if err != nil {
		t.Fatalf("apply plugin migrations: %v", err)
	}
	if len(applied) != 2 || !db.Migrator().HasTable(table) {
		t.Fatalf("unexpected apply result: steps=%+v table=%v", applied, db.Migrator().HasTable(table))
	}
	var count int64
	if err := db.Table(table).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("query migrated data: count=%d err=%v", count, err)
	}
	repeated, err := migrator.Apply(context.Background(), 0)
	if err != nil || len(repeated) != 0 {
		t.Fatalf("repeated apply must be idempotent: steps=%+v err=%v", repeated, err)
	}
	rolledBack, err := migrator.Rollback(context.Background(), 0)
	if err != nil {
		t.Fatalf("rollback plugin migrations: %v", err)
	}
	if len(rolledBack) != 2 || db.Migrator().HasTable(table) {
		t.Fatalf("unexpected rollback result: steps=%+v table=%v", rolledBack, db.Migrator().HasTable(table))
	}
}

func exercisePluginMigrationFailureCleanup(t *testing.T, db *gorm.DB) {
	t.Helper()
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	pluginID := "migration_failure_" + suffix
	table := pluginID + "_items"
	dir := writeMigrationFiles(t, map[string]string{
		"001_create.up.sql":   fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY);", table),
		"001_create.down.sql": fmt.Sprintf("DROP TABLE %s;", table),
		"002_fail.up.sql":     "INSERT INTO missing_plugin_table(id) VALUES (1);",
		"002_fail.down.sql":   "DELETE FROM missing_plugin_table WHERE id = 1;",
	})
	store := gormrepo.NewPluginMigrationStore(db)
	migrator := plugin.NewMigrator(pluginID, dir, "migrations", store)
	t.Cleanup(func() {
		_ = db.Exec("DROP TABLE IF EXISTS " + table).Error
		_ = db.Where("plugin_id = ?", pluginID).Delete(&gormrepo.PluginMigrationModel{}).Error
	})

	if _, err := migrator.Apply(context.Background(), 0); err == nil {
		t.Fatal("expected failed migration batch")
	}
	if db.Migrator().HasTable(table) {
		t.Fatal("failed migration batch leaked schema")
	}
	records, err := store.ListApplied(context.Background(), pluginID)
	if err != nil {
		t.Fatalf("list failed migration ledger: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("failed migration batch leaked ledger records: %+v", records)
	}
}

func openMigrationSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:plugin-migration-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(&gormrepo.PluginMigrationModel{}); err != nil {
		t.Fatalf("migrate plugin ledger: %v", err)
	}
	return db
}

func writeMigrationFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	migrations := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrations, 0o755); err != nil {
		t.Fatalf("create migrations directory: %v", err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(migrations, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write migration %s: %v", name, err)
		}
	}
	return dir
}
