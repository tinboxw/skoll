package datastore

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestLifecycleInspectStorageReturnsRegisteredPhysicalTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inspect.db") + "?_busy_timeout=5000&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("sqlite test requires cgo: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	registry := NewSchemaRegistry()
	registered, err := registry.Register(validPluginSchema("medical_oa"))
	if err != nil {
		t.Fatal(err)
	}
	table := registered.Tables[0]
	if err = db.Exec(fmt.Sprintf(`CREATE TABLE "%s" ("id" TEXT PRIMARY KEY)`, table.PhysicalName)).Error; err != nil {
		t.Fatal(err)
	}
	lifecycle, err := NewLifecycle(db, registry)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, exists, err := lifecycle.InspectStorage(context.Background(), "medical_oa")
	if err != nil {
		t.Fatalf("inspect storage: %v", err)
	}
	if !exists || snapshot.Namespace != registered.Namespace || len(snapshot.Tables) != 1 {
		t.Fatalf("unexpected storage snapshot: %+v", snapshot)
	}
	if snapshot.Tables[0].LogicalName != "products" || snapshot.Tables[0].PhysicalName != table.PhysicalName || !snapshot.Tables[0].Exists {
		t.Fatalf("unexpected table snapshot: %+v", snapshot.Tables[0])
	}
	if len(snapshot.Tables[0].Fields) == 0 || snapshot.Tables[0].PrimaryKey[0] != "id" || snapshot.Tables[0].IndexCount != 1 {
		t.Fatalf("schema metadata missing: %+v", snapshot.Tables[0])
	}
	if snapshot.Tables[0].MutationPolicy != TableMutationMutable {
		t.Fatalf("mutation policy missing: %+v", snapshot.Tables[0])
	}
	registry.Unregister("medical_oa")
	candidateSnapshot, available, err := lifecycle.InspectCandidateStorage(context.Background(), SchemaCandidate{Schema: validPluginSchema("medical_oa"), Present: true})
	if err != nil || !available || len(candidateSnapshot.Tables) != 1 || !candidateSnapshot.Tables[0].Exists {
		t.Fatalf("inactive declared schema must remain inspectable: snapshot=%+v available=%v err=%v", candidateSnapshot, available, err)
	}
}
