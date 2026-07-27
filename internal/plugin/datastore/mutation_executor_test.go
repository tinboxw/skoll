package datastore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

type fixedMutationScopeService struct {
	predicate pluginsdk.ScopePredicate
}

func (s fixedMutationScopeService) Resolve(context.Context, pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	return s.predicate, nil
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

func TestMutationExecutorEnforcesAppendOnlyPolicyBeforePersistence(t *testing.T) {
	executor, db, table, _ := newMutationFixtureWithPolicy(t, TableMutationAppendOnly)
	ctx := context.Background()
	insert := productMutation(pluginsdk.DataMutationInsert, "ledger-1", "append-ledger-1")
	if _, err := executor.Mutate(ctx, "medical_oa", insert); err != nil {
		t.Fatalf("append-only insert failed: %v", err)
	}
	assertMutationCounts(t, db, table.PhysicalName, 1, 1, 1)

	for _, operation := range []pluginsdk.DataMutationOperation{
		pluginsdk.DataMutationUpdate,
		pluginsdk.DataMutationUpsert,
		pluginsdk.DataMutationDelete,
		pluginsdk.DataMutationAdjust,
	} {
		mutation := productMutation(operation, "ledger-1", "reject-"+string(operation))
		if operation == pluginsdk.DataMutationDelete {
			mutation.Values = nil
		}
		if operation == pluginsdk.DataMutationAdjust {
			mutation = productAdjustment("ledger-1", "reject-adjust", "quantity", pluginsdk.DataValueInteger, "1")
		}
		_, err := executor.Mutate(ctx, "medical_oa", mutation)
		assertStoreError(t, err, pluginsdk.DataStoreErrorUnsupported, "operation")
		assertMutationCounts(t, db, table.PhysicalName, 1, 1, 1)
	}
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

	mutation = productAdjustment("product-7", "immutable-adjustment", "id", pluginsdk.DataValueInteger, "1")
	_, err = executor.Mutate(context.Background(), "medical_oa", mutation)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "adjustment.field")

	mutation = productAdjustment("product-7", "non-numeric-adjustment", "name", pluginsdk.DataValueInteger, "1")
	_, err = executor.Mutate(context.Background(), "medical_oa", mutation)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "adjustment.field")
}

func TestMutationExecutorAdjustsWithBoundsVersionsIdempotencyAndAudit(t *testing.T) {
	executor, db, table, audit := newMutationFixture(t)
	ctx := context.Background()
	if _, err := executor.Mutate(ctx, "medical_oa", productMutation(pluginsdk.DataMutationInsert, "stock-1", "insert-stock-1")); err != nil {
		t.Fatal(err)
	}

	adjust := productAdjustment("stock-1", "decrement-stock-1", "quantity", pluginsdk.DataValueInteger, "-3")
	adjust.Adjustment.Minimum = dataValuePointer(pluginsdk.DataValueInteger, "0")
	adjust.Returning = []string{"quantity"}
	result, err := executor.Mutate(ctx, "medical_oa", adjust)
	if err != nil {
		t.Fatal(err)
	}
	if result.Record == nil || result.Record.Version != 2 || result.Record.Values["quantity"].Value != "7" {
		t.Fatalf("adjusted=%+v", result)
	}
	replayed, err := executor.Mutate(ctx, "medical_oa", adjust)
	if err != nil || !reflect.DeepEqual(replayed, result) {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}

	blocked := productAdjustment("stock-1", "blocked-stock-1", "quantity", pluginsdk.DataValueInteger, "-8")
	blocked.Adjustment.Minimum = dataValuePointer(pluginsdk.DataValueInteger, "0")
	_, err = executor.Mutate(ctx, "medical_oa", blocked)
	assertStoreError(t, err, pluginsdk.DataStoreErrorConflict, pluginsdk.DataAdjustmentGuardConflictField)

	staleVersion := int64(1)
	stale := productAdjustment("stock-1", "stale-stock-1", "quantity", pluginsdk.DataValueInteger, "1")
	stale.ExpectedVersion = &staleVersion
	_, err = executor.Mutate(ctx, "medical_oa", stale)
	assertStoreError(t, err, pluginsdk.DataStoreErrorConflict, "expectedVersion")

	var quantity int64
	if err = db.Table(table.PhysicalName).Select("quantity").Where("id = ?", "stock-1").Scan(&quantity).Error; err != nil {
		t.Fatal(err)
	}
	if quantity != 7 {
		t.Fatalf("quantity=%d want=7", quantity)
	}
	assertMutationCounts(t, db, table.PhysicalName, 1, 2, 2)
	var adjustmentAudits int64
	if err = db.Model(&mutationAuditRow{}).Where("action = ?", "datastore.adjust").Count(&adjustmentAudits).Error; err != nil {
		t.Fatal(err)
	}
	if adjustmentAudits != 1 {
		t.Fatalf("adjustment audits=%d want=1", adjustmentAudits)
	}

	audit.fail = true
	rolledBack := productAdjustment("stock-1", "adjustment-audit-failure", "quantity", pluginsdk.DataValueInteger, "1")
	_, err = executor.Mutate(ctx, "medical_oa", rolledBack)
	assertStoreError(t, err, pluginsdk.DataStoreErrorUnavailable, "audit")
	if err = db.Table(table.PhysicalName).Select("quantity").Where("id = ?", "stock-1").Scan(&quantity).Error; err != nil {
		t.Fatal(err)
	}
	if quantity != 7 {
		t.Fatalf("rolled back quantity=%d want=7", quantity)
	}
	assertMutationCounts(t, db, table.PhysicalName, 1, 2, 2)
}

func TestMutationExecutorConcurrentAdjustmentsNeverLoseUpdates(t *testing.T) {
	executor, db, table, _ := newMutationFixture(t)
	ctx := context.Background()
	if _, err := executor.Mutate(ctx, "medical_oa", productMutation(pluginsdk.DataMutationInsert, "counter-1", "insert-counter-1")); err != nil {
		t.Fatal(err)
	}

	const workers = 12
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wait sync.WaitGroup
	wait.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func(index int) {
			defer wait.Done()
			<-start
			mutation := productAdjustment("counter-1", fmt.Sprintf("increment-counter-%02d", index), "quantity", pluginsdk.DataValueInteger, "1")
			mutation.Adjustment.Maximum = dataValuePointer(pluginsdk.DataValueInteger, "100")
			_, mutateErr := executor.Mutate(ctx, "medical_oa", mutation)
			errs <- mutateErr
		}(worker)
	}
	close(start)
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent adjustment failed: %v", err)
		}
	}

	var row struct {
		Quantity int64
		Version  int64
	}
	if err := db.Table(table.PhysicalName).Select("quantity", "version").Where("id = ?", "counter-1").Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Quantity != 10+workers || row.Version != 1+workers {
		t.Fatalf("row=%+v want quantity=%d version=%d", row, 10+workers, 1+workers)
	}
	assertMutationCounts(t, db, table.PhysicalName, 1, 1+workers, 1+workers)
}

func TestMutationExecutorSQLiteRejectsInexactDecimalAdjustment(t *testing.T) {
	executor, db, table, _ := newMutationFixture(t)
	ctx := context.Background()
	if _, err := executor.Mutate(ctx, "medical_oa", productMutation(pluginsdk.DataMutationInsert, "balance-1", "insert-balance-1")); err != nil {
		t.Fatal(err)
	}
	mutation := productAdjustment("balance-1", "decimal-adjustment", "amount", pluginsdk.DataValueDecimal, "0.10")
	mutation.Adjustment.Maximum = dataValuePointer(pluginsdk.DataValueDecimal, "10.30")
	_, err := executor.Mutate(ctx, "medical_oa", mutation)
	assertStoreError(t, err, pluginsdk.DataStoreErrorUnsupported, "adjustment.field")
	var amount string
	if err = db.Table(table.PhysicalName).Select("amount").Where("id = ?", "balance-1").Scan(&amount).Error; err != nil {
		t.Fatal(err)
	}
	if amount != "10" {
		t.Fatalf("amount=%s want=10", amount)
	}
	assertMutationCounts(t, db, table.PhysicalName, 1, 1, 1)
}

func TestBuildAdjustmentPlanQuotesAllCurrentDialects(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	table := ResolvedTable{
		Fields: map[string]FieldSchema{
			"id":       {Name: "id", Type: pluginsdk.DataValueString},
			"quantity": {Name: "quantity", Type: pluginsdk.DataValueInteger, Mutable: true},
		},
		PrimaryKey: []string{"id"},
	}
	scope := mutationScope{tenantID: "tenant-a", organizationID: "org-a", ownerID: "employee-1"}
	version := int64(4)
	mutation := productAdjustment("product-1", "dialect-plan", "quantity", pluginsdk.DataValueInteger, "-2")
	mutation.ExpectedVersion = &version
	mutation.Adjustment.Minimum = dataValuePointer(pluginsdk.DataValueInteger, "0")
	mutation.Adjustment.Maximum = dataValuePointer(pluginsdk.DataValueInteger, "100")
	for _, test := range []struct {
		dialect SQLDialect
		quote   string
	}{
		{dialect: DialectSQLite, quote: `"`},
		{dialect: DialectPostgreSQL, quote: `"`},
		{dialect: DialectMySQL, quote: "`"},
	} {
		t.Run(string(test.dialect), func(t *testing.T) {
			plan, err := buildAdjustmentPlan(test.dialect, table, scope, mutation, now)
			if err != nil {
				t.Fatal(err)
			}
			field := test.quote + "quantity" + test.quote
			for _, fragment := range []string{field + " IS NOT NULL", "(" + field + " + ?) >= ?", "(" + field + " + ?) <= ?", test.quote + "version" + test.quote + " = ?"} {
				if !strings.Contains(plan.where, fragment) {
					t.Fatalf("where=%s missing=%s", plan.where, fragment)
				}
			}
			expression, ok := plan.updates["quantity"].(clause.Expr)
			if !ok || expression.SQL != field+" + ?" || !reflect.DeepEqual(expression.Vars, []any{int64(-2)}) {
				t.Fatalf("expression=%+v", plan.updates["quantity"])
			}
			if len(plan.args) != 9 {
				t.Fatalf("args=%v", plan.args)
			}
		})
	}
}

func TestBuildAdjustmentPlanPreservesExactDecimalBindings(t *testing.T) {
	table := ResolvedTable{
		Fields: map[string]FieldSchema{
			"id":     {Name: "id", Type: pluginsdk.DataValueString},
			"amount": {Name: "amount", Type: pluginsdk.DataValueDecimal, Mutable: true},
		},
		PrimaryKey: []string{"id"},
	}
	scope := mutationScope{tenantID: "tenant-a", organizationID: "org-a", ownerID: "employee-1"}
	mutation := productAdjustment("product-1", "decimal-plan", "amount", pluginsdk.DataValueDecimal, "0.10")
	mutation.Adjustment.Minimum = dataValuePointer(pluginsdk.DataValueDecimal, "0.00")
	mutation.Adjustment.Maximum = dataValuePointer(pluginsdk.DataValueDecimal, "9999999999999999.99")
	for _, dialect := range []SQLDialect{DialectPostgreSQL, DialectMySQL} {
		t.Run(string(dialect), func(t *testing.T) {
			plan, err := buildAdjustmentPlan(dialect, table, scope, mutation, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			expression, ok := plan.updates["amount"].(clause.Expr)
			if !ok || !reflect.DeepEqual(expression.Vars, []any{"0.10"}) {
				t.Fatalf("expression=%+v", plan.updates["amount"])
			}
			for _, index := range []int{4, 5, 6, 7} {
				if _, ok := plan.args[index].(string); !ok {
					t.Fatalf("arg[%d]=%T want exact string: %v", index, plan.args[index], plan.args)
				}
			}
		})
	}
	if _, err := buildAdjustmentPlan(DialectSQLite, table, scope, mutation, time.Now()); err == nil {
		t.Fatal("sqlite decimal adjustment unexpectedly planned")
	}
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
	return newMutationFixtureWithPolicy(t, TableMutationMutable)
}

func newMutationFixtureWithPolicy(t *testing.T, policy TableMutationPolicy) (*MutationExecutor, *gorm.DB, ResolvedTable, *databaseMutationAudit) {
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
	schema := validPluginSchema("medical_oa")
	schema.Tables[0].MutationPolicy = policy
	schema.Tables[0].Fields = append(schema.Tables[0].Fields,
		FieldSchema{Name: "quantity", Type: pluginsdk.DataValueInteger, Mutable: true, Filterable: true, Sortable: true},
		FieldSchema{Name: "amount", Type: pluginsdk.DataValueDecimal, Mutable: true, Filterable: true, Sortable: true},
	)
	if _, err = registry.Register(schema); err != nil {
		t.Fatal(err)
	}
	table, err := registry.Resolve("medical_oa", "products")
	if err != nil {
		t.Fatal(err)
	}
	ddl := fmt.Sprintf(`CREATE TABLE "%s" (
"id" TEXT PRIMARY KEY, "name" TEXT NOT NULL, "status" TEXT NOT NULL, "quantity" INTEGER NOT NULL DEFAULT 10, "amount" NUMERIC NOT NULL DEFAULT 10.00,
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
	executor, err := NewMutationExecutor(db, storesql.NewUnitOfWorkWithDB(db), registry, fixedMutationScopeService{predicate: predicate}, audit, DialectSQLite)
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

func productAdjustment(id, idempotencyKey, field string, valueType pluginsdk.DataValueType, delta string) pluginsdk.DataMutation {
	mutation := productMutation(pluginsdk.DataMutationAdjust, id, idempotencyKey)
	mutation.Values = nil
	mutation.Adjustment = &pluginsdk.DataAdjustment{
		Field: field,
		Delta: pluginsdk.DataValue{Type: valueType, Value: delta},
	}
	return mutation
}

func dataValuePointer(valueType pluginsdk.DataValueType, value string) *pluginsdk.DataValue {
	return &pluginsdk.DataValue{Type: valueType, Value: value}
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
