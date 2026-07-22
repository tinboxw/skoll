package datastore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mutationAuditRow struct {
	ID         uint `gorm:"primaryKey"`
	Action     string
	Resource   string
	ResourceID string
}

type databaseMutationAudit struct {
	db   *gorm.DB
	fail bool
}

func (a *databaseMutationAudit) Record(ctx context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	if a.fail {
		return pluginsdk.AuditReceipt{}, errors.New("audit unavailable")
	}
	db := storesql.ResolveDB(ctx, a.db)
	row := mutationAuditRow{Action: entry.Action, Resource: entry.Resource, ResourceID: entry.ResourceID}
	if err := db.Create(&row).Error; err != nil {
		return pluginsdk.AuditReceipt{}, err
	}
	return pluginsdk.AuditReceipt{ID: fmt.Sprint(row.ID), OccurredAt: time.Now()}, nil
}

func TestMutationExecutorLifecycleAndIdempotency(t *testing.T) {
	executor, db, table, audit := newMutationFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	executor.now = func() time.Time { return now }

	insert := productMutation(pluginsdk.DataMutationInsert, "product-1", "insert-product-1")
	insert.Returning = []string{"id", "name", "status", FieldTenantID, FieldCreatedAt}
	created, err := executor.Mutate(ctx, "medical_oa", insert)
	if err != nil {
		t.Fatal(err)
	}
	if created.RowsAffected != 1 || created.Record == nil || created.Record.Version != 1 {
		t.Fatalf("created=%+v", created)
	}
	if created.Record.Values[FieldTenantID].Value != "tenant-a" || created.Record.Values[FieldCreatedAt].Value != now.Format(time.RFC3339Nano) {
		t.Fatalf("host fields=%+v", created.Record.Values)
	}

	replayed, err := executor.Mutate(ctx, "medical_oa", insert)
	if err != nil || !reflect.DeepEqual(replayed, created) {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
	assertMutationCounts(t, db, table.PhysicalName, 1, 1, 1)

	changedReplay := insert
	changedReplay.Values = map[string]pluginsdk.DataValue{
		"name":   {Type: pluginsdk.DataValueString, Value: "Changed"},
		"status": {Type: pluginsdk.DataValueString, Value: "active"},
	}
	_, err = executor.Mutate(ctx, "medical_oa", changedReplay)
	assertStoreError(t, err, pluginsdk.DataStoreErrorConflict, "idempotencyKey")

	version1 := int64(1)
	update := productMutation(pluginsdk.DataMutationUpdate, "product-1", "update-product-1")
	update.Values["name"] = pluginsdk.DataValue{Type: pluginsdk.DataValueString, Value: "Updated product"}
	update.ExpectedVersion = &version1
	update.Returning = []string{"id", "name", "status", FieldUpdatedAt}
	now = now.Add(time.Minute)
	updated, err := executor.Mutate(ctx, "medical_oa", update)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Record == nil || updated.Record.Version != 2 || updated.Record.Values["name"].Value != "Updated product" {
		t.Fatalf("updated=%+v", updated)
	}

	stale := update
	stale.IdempotencyKey = "stale-product-1"
	stale.Values["name"] = pluginsdk.DataValue{Type: pluginsdk.DataValueString, Value: "Stale write"}
	_, err = executor.Mutate(ctx, "medical_oa", stale)
	assertStoreError(t, err, pluginsdk.DataStoreErrorConflict, "expectedVersion")
	assertMutationCounts(t, db, table.PhysicalName, 1, 2, 2)

	version2 := int64(2)
	remove := productMutation(pluginsdk.DataMutationDelete, "product-1", "delete-product-1")
	remove.Values = nil
	remove.ExpectedVersion = &version2
	remove.Returning = []string{"id", "name"}
	deleted, err := executor.Mutate(ctx, "medical_oa", remove)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Record == nil || deleted.Record.Version != 2 || deleted.Record.Values["name"].Value != "Updated product" {
		t.Fatalf("deleted=%+v", deleted)
	}
	assertMutationCounts(t, db, table.PhysicalName, 0, 3, 3)
	if audit.fail {
		t.Fatal("audit fixture unexpectedly failed")
	}
}

func TestMutationExecutorUpsertCreatesThenUpdates(t *testing.T) {
	executor, db, table, _ := newMutationFixture(t)
	ctx := context.Background()
	upsert := productMutation(pluginsdk.DataMutationUpsert, "product-2", "upsert-product-2-create")
	upsert.Returning = []string{"id", "name"}
	created, err := executor.Mutate(ctx, "medical_oa", upsert)
	if err != nil || created.Record == nil || created.Record.Version != 1 {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	upsert.IdempotencyKey = "upsert-product-2-update"
	upsert.Values["name"] = pluginsdk.DataValue{Type: pluginsdk.DataValueString, Value: "Upsert updated"}
	updated, err := executor.Mutate(ctx, "medical_oa", upsert)
	if err != nil || updated.Record == nil || updated.Record.Version != 2 || updated.Record.Values["name"].Value != "Upsert updated" {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	assertMutationCounts(t, db, table.PhysicalName, 1, 2, 2)
}

func TestMutationExecutorRollsBackDataIdempotencyAndAudit(t *testing.T) {
	executor, db, table, audit := newMutationFixture(t)
	audit.fail = true
	_, err := executor.Mutate(context.Background(), "medical_oa", productMutation(pluginsdk.DataMutationInsert, "product-3", "audit-failure"))
	assertStoreError(t, err, pluginsdk.DataStoreErrorUnavailable, "audit")
	assertMutationCounts(t, db, table.PhysicalName, 0, 0, 0)

	audit.fail = false
	sentinel := errors.New("rollback outer transaction")
	err = executor.uow.Do(context.Background(), func(tx repository.Tx) error {
		if _, mutateErr := executor.Mutate(tx.Context(), "medical_oa", productMutation(pluginsdk.DataMutationInsert, "product-4", "outer-rollback")); mutateErr != nil {
			return mutateErr
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("outer transaction error=%v", err)
	}
	assertMutationCounts(t, db, table.PhysicalName, 0, 0, 0)
}

func TestMutationExecutorRejectsAmbiguousAndOutsideScope(t *testing.T) {
	executor, _, _, _ := newMutationFixture(t)
	executor.scopes = &recordingScopeService{predicate: allScope(t)}
	mutation := productMutation(pluginsdk.DataMutationInsert, "product-5", "ambiguous-scope")
	_, err := executor.Mutate(context.Background(), "medical_oa", mutation)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "scope.filter.tenantIds")

	mutation.Scope.Filter = pluginsdk.ScopeFilter{TenantIDs: []string{"tenant-a"}, OrganizationIDs: []string{"org-a"}, OwnerIDs: []string{"employee-1"}}
	if _, err = executor.Mutate(context.Background(), "medical_oa", mutation); err != nil {
		t.Fatalf("explicit super-admin scope failed: %v", err)
	}

	executor.scopes = &recordingScopeService{predicate: mustScope(t, pluginsdk.TrustedScope{
		SubjectID: "employee-1", TenantIDs: []string{"tenant-a"}, OwnerIDs: []string{"employee-1"}, OrganizationIDs: []string{"org-a"},
	})}
	outside := productMutation(pluginsdk.DataMutationInsert, "product-6", "outside-scope")
	outside.Scope.Filter.TenantIDs = []string{"tenant-b"}
	_, err = executor.Mutate(context.Background(), "medical_oa", outside)
	assertStoreError(t, err, pluginsdk.DataStoreErrorForbidden, "scope.filter")
}

func TestMutationExecutorRejectsHostAndSchemaFields(t *testing.T) {
	executor, _, _, _ := newMutationFixture(t)
	mutation := productMutation(pluginsdk.DataMutationInsert, "product-7", "host-field")
	mutation.Values[FieldVersion] = pluginsdk.DataValue{Type: pluginsdk.DataValueInteger, Value: "2"}
	_, err := executor.Mutate(context.Background(), "medical_oa", mutation)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "values.version")

	mutation = productMutation(pluginsdk.DataMutationInsert, "product-7", "unknown-returning")
	mutation.Returning = []string{"missing"}
	_, err = executor.Mutate(context.Background(), "medical_oa", mutation)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "returning[0]")
}

func TestMutationHashCanonicalizesScopeSetsAndAuditRedactsIdempotencyKey(t *testing.T) {
	mutation := productMutation(pluginsdk.DataMutationInsert, "product-8", "business-secret-like-key")
	mutation.Scope.Filter = pluginsdk.ScopeFilter{TenantIDs: []string{"tenant-b", "tenant-a"}, OwnerIDs: []string{"owner-b", "owner-a"}}
	left, err := mutationRequestHash("medical_oa", mutation)
	if err != nil {
		t.Fatal(err)
	}
	mutation.Scope.Filter.TenantIDs = []string{"tenant-a", "tenant-b"}
	mutation.Scope.Filter.OwnerIDs = []string{"owner-a", "owner-b"}
	right, err := mutationRequestHash("medical_oa", mutation)
	if err != nil || left != right {
		t.Fatalf("canonical hashes left=%q right=%q err=%v", left, right, err)
	}
	entry := mutationAuditEntry(ResolvedTable{LogicalName: "products"}, mutation, pluginsdk.DataMutationResult{RowsAffected: 1})
	if _, exposed := entry.Detail["idempotencyKey"]; exposed || entry.Detail["idempotencyHash"] == mutation.IdempotencyKey {
		t.Fatalf("audit exposed idempotency key: %+v", entry.Detail)
	}
}

func newMutationFixture(t *testing.T) (*MutationExecutor, *gorm.DB, ResolvedTable, *databaseMutationAudit) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mutation.db") + "?_busy_timeout=5000&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		t.Skipf("sqlite test requires cgo: %v", err)
	}
	if err = db.AutoMigrate(&gormrepo.PluginDataMutationModel{}, &mutationAuditRow{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	registry := NewSchemaRegistry()
	if _, err = registry.Register(validPluginSchema("medical_oa")); err != nil {
		t.Fatal(err)
	}
	table, err := registry.Resolve("medical_oa", "products")
	if err != nil {
		t.Fatal(err)
	}
	ddl := fmt.Sprintf(`CREATE TABLE "%s" (
"id" TEXT PRIMARY KEY, "name" TEXT NOT NULL, "status" TEXT NOT NULL,
"tenant_id" TEXT NOT NULL, "organization_id" TEXT NOT NULL, "owner_id" TEXT NOT NULL,
"version" INTEGER NOT NULL, "created_at" DATETIME NOT NULL, "updated_at" DATETIME NOT NULL
)`, table.PhysicalName)
	if err = db.Exec(ddl).Error; err != nil {
		t.Fatal(err)
	}
	predicate := mustScope(t, pluginsdk.TrustedScope{
		SubjectID: "employee-1", TenantIDs: []string{"tenant-a"}, OwnerIDs: []string{"employee-1"}, OrganizationIDs: []string{"org-a"},
	})
	audit := &databaseMutationAudit{db: db}
	executor, err := NewMutationExecutor(db, storesql.NewUnitOfWorkWithDB(db), registry, &recordingScopeService{predicate: predicate}, audit, DialectSQLite)
	if err != nil {
		t.Fatal(err)
	}
	return executor, db, table, audit
}

func productMutation(operation pluginsdk.DataMutationOperation, id, idempotencyKey string) pluginsdk.DataMutation {
	return pluginsdk.DataMutation{
		Table: "products", Operation: operation,
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "medical_oa.product", Action: "write"}},
		Key:   map[string]pluginsdk.DataValue{"id": {Type: pluginsdk.DataValueString, Value: id}},
		Values: map[string]pluginsdk.DataValue{
			"name":   {Type: pluginsdk.DataValueString, Value: "Product " + id},
			"status": {Type: pluginsdk.DataValueString, Value: "active"},
		},
		IdempotencyKey: idempotencyKey,
	}
}

func assertMutationCounts(t *testing.T, db *gorm.DB, table string, records, idempotency, audits int64) {
	t.Helper()
	assertTableCount(t, db.Table(table), records, "records")
	assertTableCount(t, db.Model(&gormrepo.PluginDataMutationModel{}), idempotency, "idempotency")
	assertTableCount(t, db.Model(&mutationAuditRow{}), audits, "audits")
}

func assertTableCount(t *testing.T, query *gorm.DB, expected int64, label string) {
	t.Helper()
	var count int64
	if err := query.Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != expected {
		t.Fatalf("%s count=%d want=%d", label, count, expected)
	}
}
