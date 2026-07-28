package plugin_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/plugin/datastore"
	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const pharmaInventorySQLPluginID = "pharma_oa"

type pharmaInventorySQLScopes struct {
	predicate pluginsdk.ScopePredicate
}

func (s pharmaInventorySQLScopes) Resolve(context.Context, pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	return s.predicate, nil
}

type pharmaInventorySQLAudit struct {
	db *gorm.DB
}

type pharmaInventorySQLAuditRow struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	Action     string `gorm:"size:128;index"`
	Resource   string `gorm:"size:128"`
	ResourceID string `gorm:"size:128"`
	CreatedAt  time.Time
}

func (pharmaInventorySQLAuditRow) TableName() string {
	return "bf505b_inventory_sql_audits"
}

func (a pharmaInventorySQLAudit) Record(ctx context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	db := storesql.ResolveDB(ctx, a.db)
	if db == nil {
		return pluginsdk.AuditReceipt{}, errors.New("audit database is unavailable")
	}
	now := time.Now().UTC()
	row := pharmaInventorySQLAuditRow{
		Action: entry.Action, Resource: entry.Resource, ResourceID: entry.ResourceID, CreatedAt: now,
	}
	if err := db.Create(&row).Error; err != nil {
		return pluginsdk.AuditReceipt{}, err
	}
	return pluginsdk.AuditReceipt{ID: fmt.Sprintf("inventory-audit-%d", row.ID), OccurredAt: now}, nil
}

func TestPharmaOAInventoryLedgerRealSQLiteAcceptance(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pluginDir := filepath.Join(repoRoot, "plugins", pharmaInventorySQLPluginID)
	databasePath := filepath.Join(t.TempDir(), "pharma-inventory.db")
	db, sqlDB := openPharmaInventorySQLDB(t, databasePath)
	defer func() { _ = sqlDB.Close() }()

	if err = db.AutoMigrate(
		&gormrepo.PluginMigrationModel{},
		&gormrepo.PluginDataMutationModel{},
		&pharmaInventorySQLAuditRow{},
	); err != nil {
		t.Fatal(err)
	}

	registry, lifecycle, candidate, transformer := preparePharmaInventorySQLSchema(t, db, pluginDir)
	migrationStore := gormrepo.NewPluginMigrationStore(db)
	migrator := pluginruntime.NewMigratorWithTransformer(
		pharmaInventorySQLPluginID,
		pluginDir,
		"migrations",
		migrationStore,
		transformer,
	)
	plan, err := migrator.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Pending) < 10 || plan.Pending[9].Version != 10 {
		t.Fatalf("pending migrations=%+v want contiguous versions through 010", plan.Pending)
	}
	applied, err := migrator.Apply(context.Background(), 9)
	if err != nil || len(applied) != 9 || applied[8].Version != 9 {
		t.Fatalf("apply through 009=%+v err=%v", applied, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, false)

	applied, err = migrator.Apply(context.Background(), 1)
	if err != nil || len(applied) != 1 || applied[0].Version != 10 {
		t.Fatalf("apply 010=%+v err=%v", applied, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, true)

	rolledBack, err := migrator.Rollback(context.Background(), 1)
	if err != nil || len(rolledBack) != 1 || rolledBack[0].Version != 10 {
		t.Fatalf("rollback 010=%+v err=%v", rolledBack, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, false)
	records, err := migrationStore.ListApplied(context.Background(), pharmaInventorySQLPluginID)
	if err != nil || len(records) != 9 || records[8].Version != 9 {
		t.Fatalf("migration ledger after rollback=%+v err=%v", records, err)
	}

	applied, err = migrator.Apply(context.Background(), 1)
	if err != nil || len(applied) != 1 || applied[0].Version != 10 {
		t.Fatalf("reapply 010=%+v err=%v", applied, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, true)
	if err = lifecycle.Activate(candidate); err != nil {
		t.Fatal(err)
	}

	scopeA := pharmaInventorySQLScope(t, "tenant-a", "org-a", "actor-a")
	scopeB := pharmaInventorySQLScope(t, "tenant-b", "org-b", "actor-b")
	uow := storesql.NewUnitOfWorkWithDB(db)
	serviceA := newPharmaInventorySQLService(t, db, uow, registry, scopeA)
	serviceB := newPharmaInventorySQLService(t, db, uow, registry, scopeB)
	intentA := pharmaInventorySQLIntent("write", "tenant-a", "org-a", "actor-a")
	intentB := pharmaInventorySQLIntent("write", "tenant-b", "org-b", "actor-b")

	lotMutation := pharmaInventorySQLLotMutation(
		"lot-a", "product-a", "BATCH-A", "2026-07-01T00:00:00Z", "2027-07-01T00:00:00Z",
		"lot-a.insert", intentA,
	)
	if _, err = serviceA.Mutate(context.Background(), lotMutation); err != nil {
		t.Fatal(err)
	}
	ledgerMutation := pharmaInventorySQLLedgerMutation(
		"ledger-a", "product-a", "lot-a", "BATCH-A", "warehouse-a", "area-a", "location-a",
		1_250_000, "receipt-a", "IN-202607-0001", "line-a", "ledger-a.insert", intentA,
	)
	ledgerResult, err := serviceA.Mutate(context.Background(), ledgerMutation)
	if err != nil {
		t.Fatal(err)
	}
	replayedLedger, err := serviceA.Mutate(context.Background(), ledgerMutation)
	if err != nil || !reflect.DeepEqual(replayedLedger, ledgerResult) {
		t.Fatalf("ledger replay=%+v want=%+v err=%v", replayedLedger, ledgerResult, err)
	}
	_, err = serviceA.Mutate(context.Background(), pharmaInventorySQLLedgerMutation(
		"ledger-a", "product-a", "lot-a", "BATCH-A", "warehouse-a", "area-a", "location-a",
		1_500_000, "receipt-a", "IN-202607-0001", "line-a", "ledger-a.insert", intentA,
	))
	assertPharmaInventorySQLStoreError(t, err, pluginsdk.DataStoreErrorConflict, "idempotencyKey")
	if _, err = serviceA.Mutate(context.Background(), pharmaInventorySQLBalanceMutation(
		"balance-a", "product-a", "lot-a", "BATCH-A", "warehouse-a", "area-a", "location-a",
		"balance-a.insert", intentA,
	)); err != nil {
		t.Fatal(err)
	}
	adjustment := pharmaInventorySQLBalanceAdjustment("balance-a", 1_250_000, "balance-a.receive", intentA)
	adjusted, err := serviceA.Mutate(context.Background(), adjustment)
	if err != nil {
		t.Fatal(err)
	}
	replayedAdjustment, err := serviceA.Mutate(context.Background(), adjustment)
	if err != nil || !reflect.DeepEqual(replayedAdjustment, adjusted) {
		t.Fatalf("balance replay=%+v want=%+v err=%v", replayedAdjustment, adjusted, err)
	}
	if adjusted.Record == nil || adjusted.Record.Values["quantity_micros"].Value != "1250000" {
		t.Fatalf("adjusted balance=%+v", adjusted)
	}

	for _, test := range []struct {
		name      string
		table     string
		id        string
		operation pluginsdk.DataMutationOperation
		values    map[string]pluginsdk.DataValue
	}{
		{
			name: "lot update", table: "inventory_lots", id: "lot-a", operation: pluginsdk.DataMutationUpdate,
			values: map[string]pluginsdk.DataValue{"batch_no": pharmaInventorySQLValue(pluginsdk.DataValueString, "TAMPERED")},
		},
		{name: "lot delete", table: "inventory_lots", id: "lot-a", operation: pluginsdk.DataMutationDelete},
		{
			name: "ledger update", table: "stock_ledger", id: "ledger-a", operation: pluginsdk.DataMutationUpdate,
			values: map[string]pluginsdk.DataValue{"quantity_micros": pharmaInventorySQLValue(pluginsdk.DataValueInteger, "1")},
		},
		{name: "ledger delete", table: "stock_ledger", id: "ledger-a", operation: pluginsdk.DataMutationDelete},
	} {
		t.Run("append_only_"+test.name, func(t *testing.T) {
			version := int64(1)
			mutation := pluginsdk.DataMutation{
				Table: test.table, Operation: test.operation, Scope: intentA,
				Key:             map[string]pluginsdk.DataValue{"id": pharmaInventorySQLValue(pluginsdk.DataValueString, test.id)},
				Values:          test.values,
				IdempotencyKey:  "reject-" + test.table + "-" + string(test.operation),
				ExpectedVersion: &version,
			}
			_, mutateErr := serviceA.Mutate(context.Background(), mutation)
			assertPharmaInventorySQLStoreError(t, mutateErr, pluginsdk.DataStoreErrorUnsupported, "operation")
		})
	}

	if _, err = serviceB.Mutate(context.Background(), pharmaInventorySQLLotMutation(
		"lot-b", "product-a", "BATCH-B", "2026-07-02T00:00:00Z", "2027-07-02T00:00:00Z",
		"lot-b.insert", intentB,
	)); err != nil {
		t.Fatal(err)
	}
	if _, err = serviceB.Mutate(context.Background(), pharmaInventorySQLLedgerMutation(
		"ledger-b", "product-a", "lot-b", "BATCH-B", "warehouse-b", "area-b", "location-b",
		9_000_000, "receipt-b", "IN-202607-0002", "line-b", "ledger-b.insert", intentB,
	)); err != nil {
		t.Fatal(err)
	}
	if got := pharmaInventorySQLAggregateQuantity(t, serviceA, "stock_ledger", scopeA); got != 1_250_000 {
		t.Fatalf("scope A ledger total=%d want=1250000", got)
	}
	if got := pharmaInventorySQLAggregateQuantity(t, serviceA, "stock_balances", scopeA); got != 1_250_000 {
		t.Fatalf("scope A balance total=%d want=1250000", got)
	}
	if got := pharmaInventorySQLAggregateQuantity(t, serviceB, "stock_ledger", scopeB); got != 9_000_000 {
		t.Fatalf("scope B ledger total=%d want=9000000", got)
	}
	if page := pharmaInventorySQLQuery(t, serviceA, "stock_ledger", scopeA, nil); len(page.Records) != 1 {
		t.Fatalf("scope A ledger page=%+v", page)
	}
	outsideIntent := pharmaInventorySQLIntent("write", "tenant-b", "org-b", "actor-b")
	_, err = serviceA.Mutate(context.Background(), pharmaInventorySQLLotMutation(
		"lot-outside", "product-a", "BATCH-OUTSIDE", "2026-07-03T00:00:00Z", "2027-07-03T00:00:00Z",
		"lot-outside.insert", outsideIntent,
	))
	assertPharmaInventorySQLStoreError(t, err, pluginsdk.DataStoreErrorForbidden, "scope.filter")
	outsideQuery := pharmaInventorySQLQueryDefinition("stock_ledger", scopeB, nil)
	_, err = serviceA.Query(context.Background(), outsideQuery)
	assertPharmaInventorySQLStoreError(t, err, pluginsdk.DataStoreErrorForbidden, "scope.filter")

	auditsBeforeRollback := pharmaInventorySQLAuditCount(t, db)
	rollbackSentinel := errors.New("force inventory unit-of-work rollback")
	err = uow.Do(context.Background(), func(tx repository.Tx) error {
		if _, mutateErr := serviceA.Mutate(tx.Context(), pharmaInventorySQLLedgerMutation(
			"ledger-rollback", "product-a", "lot-a", "BATCH-A", "warehouse-a", "area-a", "location-a",
			500_000, "receipt-rollback", "IN-202607-ROLLBACK", "line-rollback", "ledger-rollback.insert", intentA,
		)); mutateErr != nil {
			return mutateErr
		}
		if _, mutateErr := serviceA.Mutate(tx.Context(), pharmaInventorySQLBalanceAdjustment(
			"balance-a", 500_000, "balance-rollback.adjust", intentA,
		)); mutateErr != nil {
			return mutateErr
		}
		return rollbackSentinel
	})
	if !errors.Is(err, rollbackSentinel) {
		t.Fatalf("outer transaction error=%v", err)
	}
	rollbackFilter := pharmaInventorySQLFilter("id", pluginsdk.DataValueString, "ledger-rollback")
	if page := pharmaInventorySQLQuery(t, serviceA, "stock_ledger", scopeA, &rollbackFilter); len(page.Records) != 0 {
		t.Fatalf("rolled-back ledger persisted: %+v", page)
	}
	if got := pharmaInventorySQLAggregateQuantity(t, serviceA, "stock_ledger", scopeA); got != 1_250_000 {
		t.Fatalf("ledger after rollback=%d want=1250000", got)
	}
	if got := pharmaInventorySQLAggregateQuantity(t, serviceA, "stock_balances", scopeA); got != 1_250_000 {
		t.Fatalf("balance after rollback=%d want=1250000", got)
	}
	if got := pharmaInventorySQLAuditCount(t, db); got != auditsBeforeRollback {
		t.Fatalf("audit count after rollback=%d want=%d", got, auditsBeforeRollback)
	}
	for _, key := range []string{"ledger-rollback.insert", "balance-rollback.adjust"} {
		var count int64
		if err = db.Model(&gormrepo.PluginDataMutationModel{}).
			Where("plugin_id = ? AND idempotency_key = ?", pharmaInventorySQLPluginID, key).
			Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("rolled-back idempotency %q count=%d err=%v", key, count, err)
		}
	}

	if err = sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	db, sqlDB = openPharmaInventorySQLDB(t, databasePath)
	registry, lifecycle, candidate, transformer = preparePharmaInventorySQLSchema(t, db, pluginDir)
	if err = lifecycle.Activate(candidate); err != nil {
		t.Fatal(err)
	}
	restartedMigrator := pluginruntime.NewMigratorWithTransformer(
		pharmaInventorySQLPluginID,
		pluginDir,
		"migrations",
		gormrepo.NewPluginMigrationStore(db),
		transformer,
	)
	restartedPlan, err := restartedMigrator.Plan(context.Background())
	if err != nil || len(restartedPlan.Applied) < 10 || restartedPlan.Applied[9].Version != 10 {
		t.Fatalf("restarted migration plan=%+v err=%v", restartedPlan, err)
	}
	restartedServiceA := newPharmaInventorySQLService(
		t, db, storesql.NewUnitOfWorkWithDB(db), registry, scopeA,
	)
	lotFilter := pharmaInventorySQLFilter("id", pluginsdk.DataValueString, "lot-a")
	lotPage := pharmaInventorySQLQuery(t, restartedServiceA, "inventory_lots", scopeA, &lotFilter)
	if len(lotPage.Records) != 1 ||
		lotPage.Records[0].Values["batch_no"].Value != "BATCH-A" ||
		lotPage.Records[0].Values["production_date"].Value != "2026-07-01T00:00:00Z" ||
		lotPage.Records[0].Values["expires_at"].Value != "2027-07-01T00:00:00Z" {
		t.Fatalf("lot facts after restart=%+v", lotPage)
	}
	if got := pharmaInventorySQLAggregateQuantity(t, restartedServiceA, "stock_ledger", scopeA); got != 1_250_000 {
		t.Fatalf("restarted ledger total=%d want=1250000", got)
	}
	if got := pharmaInventorySQLAggregateQuantity(t, restartedServiceA, "stock_balances", scopeA); got != 1_250_000 {
		t.Fatalf("restarted balance total=%d want=1250000", got)
	}
}

func openPharmaInventorySQLDB(t *testing.T, path string) (*gorm.DB, interface{ Close() error }) {
	t.Helper()
	dsn := path + "?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), TranslateError: true,
	})
	if err != nil {
		t.Skipf("sqlite test requires cgo: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	return db, sqlDB
}

func preparePharmaInventorySQLSchema(
	t *testing.T,
	db *gorm.DB,
	pluginDir string,
) (*datastore.SchemaRegistry, *datastore.Lifecycle, datastore.SchemaCandidate, func(string) (string, error)) {
	t.Helper()
	registry := datastore.NewSchemaRegistry()
	lifecycle, err := datastore.NewLifecycle(db, registry)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := lifecycle.Prepare(pharmaInventorySQLPluginID, pluginDir)
	if err != nil || !candidate.Present {
		t.Fatalf("prepare datastore schema candidate=%+v err=%v", candidate, err)
	}
	if _, err = registry.Register(candidate.Schema); err != nil {
		t.Fatal(err)
	}
	transformer, err := lifecycle.MigrationTransformer(candidate)
	if err != nil {
		t.Fatal(err)
	}
	return registry, lifecycle, candidate, transformer
}

func assertPharmaInventorySQLMigrationState(
	t *testing.T,
	db *gorm.DB,
	registry *datastore.SchemaRegistry,
	wantInventory bool,
) {
	t.Helper()
	for _, logical := range []string{"inventory_lots", "stock_ledger", "stock_balances"} {
		table, err := registry.Resolve(pharmaInventorySQLPluginID, logical)
		if err != nil {
			t.Fatal(err)
		}
		if got := db.Migrator().HasTable(table.PhysicalName); got != wantInventory {
			t.Fatalf("table %s exists=%v want=%v", logical, got, wantInventory)
		}
	}
	receipt, err := registry.Resolve(pharmaInventorySQLPluginID, "purchase_inbounds")
	if err != nil {
		t.Fatal(err)
	}
	if got := db.Migrator().HasColumn(receipt.PhysicalName, "request_hash"); got != wantInventory {
		t.Fatalf("purchase_inbounds.request_hash exists=%v want=%v", got, wantInventory)
	}
}

func newPharmaInventorySQLService(
	t *testing.T,
	db *gorm.DB,
	uow repository.UnitOfWork,
	registry *datastore.SchemaRegistry,
	scope pluginsdk.ScopePredicate,
) *datastore.Service {
	t.Helper()
	service, err := datastore.NewService(
		db,
		uow,
		registry,
		pharmaInventorySQLScopes{predicate: scope},
		pharmaInventorySQLAudit{db: db},
		datastore.DialectSQLite,
		pharmaInventorySQLPluginID,
	)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func pharmaInventorySQLScope(t *testing.T, tenant, organization, owner string) pluginsdk.ScopePredicate {
	t.Helper()
	scope, err := pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: owner, TenantIDs: []string{tenant}, OrganizationIDs: []string{organization}, OwnerIDs: []string{owner},
	})
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func pharmaInventorySQLIntent(action, tenant, organization, owner string) pluginsdk.DataScopeIntent {
	return pluginsdk.DataScopeIntent{
		Permission: pluginsdk.Permission{Resource: "pharma_oa.inventory", Action: action},
		Filter: pluginsdk.ScopeFilter{
			TenantIDs: []string{tenant}, OrganizationIDs: []string{organization}, OwnerIDs: []string{owner},
		},
	}
}

func pharmaInventorySQLLotMutation(
	id, productID, batchNo, productionDate, expiresAt, idempotencyKey string,
	scope pluginsdk.DataScopeIntent,
) pluginsdk.DataMutation {
	return pluginsdk.DataMutation{
		Table: "inventory_lots", Operation: pluginsdk.DataMutationInsert, Scope: scope,
		Key: map[string]pluginsdk.DataValue{"id": pharmaInventorySQLValue(pluginsdk.DataValueString, id)},
		Values: map[string]pluginsdk.DataValue{
			"product_id":      pharmaInventorySQLValue(pluginsdk.DataValueString, productID),
			"batch_no":        pharmaInventorySQLValue(pluginsdk.DataValueString, batchNo),
			"production_date": pharmaInventorySQLValue(pluginsdk.DataValueTimestamp, productionDate),
			"expires_at":      pharmaInventorySQLValue(pluginsdk.DataValueTimestamp, expiresAt),
		},
		Returning: []string{"id", "batch_no", "production_date", "expires_at"}, IdempotencyKey: idempotencyKey,
	}
}

func pharmaInventorySQLLedgerMutation(
	id, productID, lotID, batchNo, warehouseID, areaID, locationID string,
	quantityMicros int64,
	sourceID, sourceNumber, sourceLineID, idempotencyKey string,
	scope pluginsdk.DataScopeIntent,
) pluginsdk.DataMutation {
	return pluginsdk.DataMutation{
		Table: "stock_ledger", Operation: pluginsdk.DataMutationInsert, Scope: scope,
		Key: map[string]pluginsdk.DataValue{"id": pharmaInventorySQLValue(pluginsdk.DataValueString, id)},
		Values: map[string]pluginsdk.DataValue{
			"entry_type":              pharmaInventorySQLValue(pluginsdk.DataValueString, "purchase_receipt"),
			"product_id":              pharmaInventorySQLValue(pluginsdk.DataValueString, productID),
			"lot_id":                  pharmaInventorySQLValue(pluginsdk.DataValueString, lotID),
			"batch_no":                pharmaInventorySQLValue(pluginsdk.DataValueString, batchNo),
			"warehouse_id":            pharmaInventorySQLValue(pluginsdk.DataValueString, warehouseID),
			"area_id":                 pharmaInventorySQLValue(pluginsdk.DataValueString, areaID),
			"location_id":             pharmaInventorySQLValue(pluginsdk.DataValueString, locationID),
			"quantity_micros":         pharmaInventorySQLValue(pluginsdk.DataValueInteger, fmt.Sprintf("%d", quantityMicros)),
			"source_document_type":    pharmaInventorySQLValue(pluginsdk.DataValueString, "purchase_inbound"),
			"source_document_id":      pharmaInventorySQLValue(pluginsdk.DataValueString, sourceID),
			"source_document_number":  pharmaInventorySQLValue(pluginsdk.DataValueString, sourceNumber),
			"source_document_line_id": pharmaInventorySQLValue(pluginsdk.DataValueString, sourceLineID),
			"occurred_at":             pharmaInventorySQLValue(pluginsdk.DataValueTimestamp, "2026-07-28T08:00:00Z"),
		},
		Returning: []string{"id", "quantity_micros"}, IdempotencyKey: idempotencyKey,
	}
}

func pharmaInventorySQLBalanceMutation(
	id, productID, lotID, batchNo, warehouseID, areaID, locationID, idempotencyKey string,
	scope pluginsdk.DataScopeIntent,
) pluginsdk.DataMutation {
	return pluginsdk.DataMutation{
		Table: "stock_balances", Operation: pluginsdk.DataMutationInsert, Scope: scope,
		Key: map[string]pluginsdk.DataValue{"id": pharmaInventorySQLValue(pluginsdk.DataValueString, id)},
		Values: map[string]pluginsdk.DataValue{
			"product_id":      pharmaInventorySQLValue(pluginsdk.DataValueString, productID),
			"lot_id":          pharmaInventorySQLValue(pluginsdk.DataValueString, lotID),
			"batch_no":        pharmaInventorySQLValue(pluginsdk.DataValueString, batchNo),
			"warehouse_id":    pharmaInventorySQLValue(pluginsdk.DataValueString, warehouseID),
			"area_id":         pharmaInventorySQLValue(pluginsdk.DataValueString, areaID),
			"location_id":     pharmaInventorySQLValue(pluginsdk.DataValueString, locationID),
			"quantity_micros": pharmaInventorySQLValue(pluginsdk.DataValueInteger, "0"),
		},
		Returning: []string{"id", "quantity_micros"}, IdempotencyKey: idempotencyKey,
	}
}

func pharmaInventorySQLBalanceAdjustment(
	id string,
	delta int64,
	idempotencyKey string,
	scope pluginsdk.DataScopeIntent,
) pluginsdk.DataMutation {
	minimum := pharmaInventorySQLValue(pluginsdk.DataValueInteger, "0")
	maximum := pharmaInventorySQLValue(pluginsdk.DataValueInteger, fmt.Sprintf("%d", int64(math.MaxInt64)))
	return pluginsdk.DataMutation{
		Table: "stock_balances", Operation: pluginsdk.DataMutationAdjust, Scope: scope,
		Key: map[string]pluginsdk.DataValue{"id": pharmaInventorySQLValue(pluginsdk.DataValueString, id)},
		Adjustment: &pluginsdk.DataAdjustment{
			Field: "quantity_micros", Delta: pharmaInventorySQLValue(pluginsdk.DataValueInteger, fmt.Sprintf("%d", delta)),
			Minimum: &minimum, Maximum: &maximum,
		},
		Returning: []string{"id", "quantity_micros"}, IdempotencyKey: idempotencyKey,
	}
}

func pharmaInventorySQLQuery(
	t *testing.T,
	service *datastore.Service,
	table string,
	scope pluginsdk.ScopePredicate,
	filter *pluginsdk.DataFilter,
) pluginsdk.DataPage {
	t.Helper()
	page, err := service.Query(context.Background(), pharmaInventorySQLQueryDefinition(table, scope, filter))
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func pharmaInventorySQLQueryDefinition(
	table string,
	scope pluginsdk.ScopePredicate,
	filter *pluginsdk.DataFilter,
) pluginsdk.DataQuery {
	fields := map[string][]string{
		"inventory_lots": {"id", "product_id", "batch_no", "production_date", "expires_at"},
		"stock_ledger":   {"id", "product_id", "lot_id", "location_id", "quantity_micros"},
		"stock_balances": {"id", "product_id", "lot_id", "location_id", "quantity_micros"},
	}[table]
	return pluginsdk.DataQuery{
		Table: table, Fields: fields,
		Scope: pluginsdk.DataScopeIntent{
			Permission: pluginsdk.Permission{Resource: "pharma_oa.inventory", Action: "read"},
			Filter: pluginsdk.ScopeFilter{
				TenantIDs: scope.TenantIDs(), OrganizationIDs: scope.OrganizationIDs(), OwnerIDs: scope.OwnerIDs(),
			},
		},
		Filter: filter,
		Sort:   []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}},
		Page:   pluginsdk.DataPageRequest{Limit: 100},
	}
}

func pharmaInventorySQLAggregateQuantity(
	t *testing.T,
	service *datastore.Service,
	table string,
	scope pluginsdk.ScopePredicate,
) int64 {
	t.Helper()
	page, err := service.Aggregate(context.Background(), pluginsdk.DataAggregateQuery{
		Table: table,
		Scope: pluginsdk.DataScopeIntent{
			Permission: pluginsdk.Permission{Resource: "pharma_oa.inventory", Action: "read"},
			Filter: pluginsdk.ScopeFilter{
				TenantIDs: scope.TenantIDs(), OrganizationIDs: scope.OrganizationIDs(), OwnerIDs: scope.OwnerIDs(),
			},
		},
		Metrics: []pluginsdk.DataAggregateMetric{
			{Operation: pluginsdk.DataAggregateCount},
			{Operation: pluginsdk.DataAggregateSum, Field: "quantity_micros"},
		},
		Page: pluginsdk.DataPageRequest{Limit: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || len(page.Rows[0].Values) != 2 || page.Rows[0].Values[0].Type != pluginsdk.DataValueInteger {
		t.Fatalf("%s aggregate page=%+v", table, page)
	}
	var quantity int64
	if _, err = fmt.Sscan(page.Rows[0].Values[1].Value, &quantity); err != nil {
		t.Fatalf("%s aggregate quantity=%q err=%v", table, page.Rows[0].Values[1].Value, err)
	}
	return quantity
}

func pharmaInventorySQLFilter(field string, kind pluginsdk.DataValueType, value string) pluginsdk.DataFilter {
	typed := pharmaInventorySQLValue(kind, value)
	return pluginsdk.DataFilter{Field: field, Operator: pluginsdk.DataOperatorEqual, Value: &typed}
}

func pharmaInventorySQLValue(kind pluginsdk.DataValueType, value string) pluginsdk.DataValue {
	return pluginsdk.DataValue{Type: kind, Value: value}
}

func pharmaInventorySQLAuditCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&pharmaInventorySQLAuditRow{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func assertPharmaInventorySQLStoreError(
	t *testing.T,
	err error,
	code pluginsdk.DataStoreErrorCode,
	field string,
) {
	t.Helper()
	var storeErr *pluginsdk.DataStoreError
	if !errors.As(err, &storeErr) {
		t.Fatalf("error=%v want datastore error %s", err, code)
	}
	if storeErr.Code != code || field != "" && storeErr.Field != field {
		t.Fatalf("datastore error=%+v want code=%s field=%q", storeErr, code, field)
	}
}
