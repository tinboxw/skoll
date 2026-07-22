package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/plugin/datastore"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"gorm.io/gorm"
)

func TestPluginDataStoreLifecycleSurvivesRestartUpgradeAndDrop(t *testing.T) {
	db := gormrepo.TestDB(t)
	pluginID := "medical-lifecycle"
	namespace, err := datastore.PluginNamespace(pluginID)
	if err != nil {
		t.Fatal(err)
	}
	table := namespace + "_records"
	pluginDir := writeLifecycleMigrationPlugin(t, pluginID, plugin.DataUninstallDrop, map[string]string{
		"001_records.up.sql":   createLifecycleRecordsSQL(),
		"001_records.down.sql": "DROP TABLE {{table:records}};",
	})
	writeLifecycleDataStoreSchema(t, pluginDir, false)

	first := newMigrationTestPluginManager(t, db)
	if _, err = first.Install(pluginDir); err != nil {
		t.Fatal(err)
	}
	if err = first.Enable(pluginID); err != nil {
		t.Fatal(err)
	}
	assertLifecycleSchema(t, first, pluginID, false)
	if err = db.Exec("INSERT INTO "+table+" (id, name, tenant_id, organization_id, owner_id, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"record-1", "stable", "tenant-1", "org-1", "owner-1", 1, time.Now().UTC(), time.Now().UTC()).Error; err != nil {
		t.Fatal(err)
	}

	second := newMigrationTestPluginManager(t, db)
	if _, err = second.Install(pluginDir); err != nil {
		t.Fatal(err)
	}
	if err = second.Enable(pluginID); err != nil {
		t.Fatal(err)
	}
	assertLifecycleSchema(t, second, pluginID, false)
	assertLifecycleRecord(t, db, table, "stable")

	writeLifecycleMigrationFile(t, pluginDir, "002_status.up.sql", "ALTER TABLE {{table:records}} ADD COLUMN status TEXT;")
	writeLifecycleMigrationFile(t, pluginDir, "002_status.down.sql", "ALTER TABLE {{table:records}} DROP COLUMN status;")
	writeLifecycleDataStoreSchema(t, pluginDir, true)
	rewriteLifecycleMigrationVersion(t, pluginDir, "1.1.0", "v1.1.0")
	if err = second.ReloadPluginMetadata(pluginID); err != nil {
		t.Fatal(err)
	}
	assertLifecycleSchema(t, second, pluginID, true)
	assertLifecycleRecord(t, db, table, "stable")

	if err = db.Create(&gormrepo.PluginDataMutationModel{
		PluginID: pluginID, IdempotencyKey: "record-1.create", RequestHash: strings.Repeat("a", 64),
		ResultJSON: `{}`, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err = second.Uninstall(pluginID); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable(table) {
		t.Fatal("drop uninstall retained plugin datastore table")
	}
	if _, exists := second.dataLifecycle.Snapshot(pluginID); exists {
		t.Fatal("drop uninstall retained schema registration")
	}
	var mutations int64
	if err = db.Model(&gormrepo.PluginDataMutationModel{}).Where("plugin_id = ?", pluginID).Count(&mutations).Error; err != nil || mutations != 0 {
		t.Fatalf("drop uninstall mutation records=%d err=%v", mutations, err)
	}
}

func TestPluginDataStoreFailedUpgradeKeepsActiveSchemaAtomic(t *testing.T) {
	db := gormrepo.TestDB(t)
	pluginID := "atomic-lifecycle"
	namespace, _ := datastore.PluginNamespace(pluginID)
	table := namespace + "_records"
	pluginDir := writeLifecycleMigrationPlugin(t, pluginID, plugin.DataUninstallDrop, map[string]string{
		"001_records.up.sql":   createLifecycleRecordsSQL(),
		"001_records.down.sql": "DROP TABLE {{table:records}};",
	})
	writeLifecycleDataStoreSchema(t, pluginDir, false)
	manager := newMigrationTestPluginManager(t, db)
	if _, err := manager.Install(pluginDir); err != nil {
		t.Fatal(err)
	}
	if err := manager.Enable(pluginID); err != nil {
		t.Fatal(err)
	}

	writeLifecycleMigrationFile(t, pluginDir, "002_status.up.sql", "ALTER TABLE {{table:records}} ADD COLUMN status TEXT;")
	writeLifecycleMigrationFile(t, pluginDir, "002_status.down.sql", "ALTER TABLE {{table:records}} DROP COLUMN status;")
	writeLifecycleMigrationFile(t, pluginDir, "003_fail.up.sql", "INSERT INTO missing_atomic_table(id) VALUES (1);")
	writeLifecycleMigrationFile(t, pluginDir, "003_fail.down.sql", "DELETE FROM missing_atomic_table WHERE id = 1;")
	writeLifecycleDataStoreSchema(t, pluginDir, true)
	rewriteLifecycleMigrationVersion(t, pluginDir, "1.1.0", "v1.1.0")
	if err := manager.ReloadPluginMetadata(pluginID); err == nil {
		t.Fatal("expected failed datastore upgrade")
	}
	assertLifecycleSchema(t, manager, pluginID, false)
	if db.Migrator().HasColumn(table, "status") {
		t.Fatal("failed datastore upgrade leaked physical column")
	}
}

func TestPluginDataStoreRollbackIsExplicitAndFailClosed(t *testing.T) {
	db := gormrepo.TestDB(t)
	pluginID := "rollback-lifecycle"
	namespace, _ := datastore.PluginNamespace(pluginID)
	table := namespace + "_records"
	pluginDir := writeLifecycleMigrationPlugin(t, pluginID, plugin.DataUninstallDrop, map[string]string{
		"001_records.up.sql":   createLifecycleRecordsSQL(),
		"001_records.down.sql": "DROP TABLE {{table:records}};",
		"002_status.up.sql":    "ALTER TABLE {{table:records}} ADD COLUMN status TEXT;",
		"002_status.down.sql":  "ALTER TABLE {{table:records}} DROP COLUMN status;",
	})
	writeLifecycleDataStoreSchema(t, pluginDir, true)
	manager := newMigrationTestPluginManager(t, db)
	if _, err := manager.Install(pluginDir); err != nil {
		t.Fatal(err)
	}
	if err := manager.Enable(pluginID); err != nil {
		t.Fatal(err)
	}
	if err := manager.RollbackPluginData(pluginID, 1); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("enabled plugin rollback was not rejected: %v", err)
	}
	if err := manager.Disable(pluginID); err != nil {
		t.Fatal(err)
	}
	if err := manager.RollbackPluginData(pluginID, 1); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn(table, "status") {
		t.Fatal("explicit rollback retained rolled-back column")
	}
	if _, exists := manager.dataLifecycle.Snapshot(pluginID); exists {
		t.Fatal("explicit rollback left stale schema active")
	}
	if err := manager.Enable(pluginID); err != nil {
		t.Fatal(err)
	}
	assertLifecycleSchema(t, manager, pluginID, true)
}

func createLifecycleRecordsSQL() string {
	return `CREATE TABLE {{table:records}} (
id TEXT PRIMARY KEY,
name TEXT NOT NULL,
tenant_id TEXT NOT NULL,
organization_id TEXT NOT NULL,
owner_id TEXT NOT NULL,
version INTEGER NOT NULL,
created_at DATETIME NOT NULL,
updated_at DATETIME NOT NULL
);`
}

func writeLifecycleDataStoreSchema(t *testing.T, pluginDir string, withStatus bool) {
	t.Helper()
	status := ""
	index := ""
	if withStatus {
		status = "\n      - name: status\n        type: string\n        nullable: true\n        mutable: true\n        filterable: true\n        sortable: true"
		index = "\n    indexes:\n      - fields: [status]"
	}
	manifest := `version: 1
tables:
  - name: records
    primary_key: [id]
    fields:
      - name: id
        type: string
        filterable: true
        sortable: true
      - name: name
        type: string
        mutable: true
        filterable: true
        sortable: true` + status + index + "\n"
	if err := os.WriteFile(filepath.Join(pluginDir, datastore.SchemaManifestName), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertLifecycleSchema(t *testing.T, manager *pluginManagerWithExtensions, pluginID string, wantStatus bool) {
	t.Helper()
	schema, exists := manager.dataLifecycle.Snapshot(pluginID)
	if !exists || len(schema.Tables) != 1 {
		t.Fatalf("schema not active: %+v exists=%v", schema, exists)
	}
	_, hasStatus := schema.Tables[0].Fields["status"]
	if hasStatus != wantStatus {
		t.Fatalf("schema status field=%v want=%v: %+v", hasStatus, wantStatus, schema)
	}
}

func assertLifecycleRecord(t *testing.T, db *gorm.DB, table, want string) {
	t.Helper()
	var name string
	if err := db.Raw("SELECT name FROM "+table+" WHERE id = ?", "record-1").Scan(&name).Error; err != nil || name != want {
		t.Fatalf("record name=%q want=%q err=%v", name, want, err)
	}
}
