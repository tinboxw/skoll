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
	if len(plan.Pending) < 11 || plan.Pending[10].Version != 11 {
		t.Fatalf("pending migrations=%+v want contiguous versions through 011", plan.Pending)
	}
	applied, err := migrator.Apply(context.Background(), 9)
	if err != nil || len(applied) != 9 || applied[8].Version != 9 {
		t.Fatalf("apply through 009=%+v err=%v", applied, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, false, false)

	applied, err = migrator.Apply(context.Background(), 1)
	if err != nil || len(applied) != 1 || applied[0].Version != 10 {
		t.Fatalf("apply 010=%+v err=%v", applied, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, true, false)

	rolledBack, err := migrator.Rollback(context.Background(), 1)
	if err != nil || len(rolledBack) != 1 || rolledBack[0].Version != 10 {
		t.Fatalf("rollback 010=%+v err=%v", rolledBack, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, false, false)
	records, err := migrationStore.ListApplied(context.Background(), pharmaInventorySQLPluginID)
	if err != nil || len(records) != 9 || records[8].Version != 9 {
		t.Fatalf("migration ledger after rollback=%+v err=%v", records, err)
	}

	applied, err = migrator.Apply(context.Background(), 1)
	if err != nil || len(applied) != 1 || applied[0].Version != 10 {
		t.Fatalf("reapply 010=%+v err=%v", applied, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, true, false)

	applied, err = migrator.Apply(context.Background(), 1)
	if err != nil || len(applied) != 1 || applied[0].Version != 11 {
		t.Fatalf("apply 011=%+v err=%v", applied, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, true, true)

	rolledBack, err = migrator.Rollback(context.Background(), 1)
	if err != nil || len(rolledBack) != 1 || rolledBack[0].Version != 11 {
		t.Fatalf("rollback 011=%+v err=%v", rolledBack, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, true, false)
	records, err = migrationStore.ListApplied(context.Background(), pharmaInventorySQLPluginID)
	if err != nil || len(records) != 10 || records[9].Version != 10 {
		t.Fatalf("migration ledger after 011 rollback=%+v err=%v", records, err)
	}

	applied, err = migrator.Apply(context.Background(), 1)
	if err != nil || len(applied) != 1 || applied[0].Version != 11 {
		t.Fatalf("reapply 011=%+v err=%v", applied, err)
	}
	assertPharmaInventorySQLMigrationState(t, db, registry, true, true)
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

	transferLegs := `[` +
		`{"id":"transfer-a-out","legType":"transfer_out","productId":"product-a","lotId":"lot-a","batchNo":"BATCH-A",` +
		`"warehouseId":"warehouse-a","areaId":"area-a","locationId":"location-a","quantity":"-0.25"},` +
		`{"id":"transfer-a-in","legType":"transfer_in","productId":"product-a","lotId":"lot-a","batchNo":"BATCH-A",` +
		`"warehouseId":"warehouse-a","areaId":"area-a","locationId":"location-b","quantity":"0.25"}]`
	err = uow.Do(context.Background(), func(tx repository.Tx) error {
		mutations := []pluginsdk.DataMutation{
			pharmaInventorySQLMovementMutation(
				"movement-transfer-a", "ST-202607-0001", transferLegs, "movement-transfer-a.insert", intentA,
			),
			pharmaInventorySQLMovementLedgerMutation(
				"ledger-transfer-out", "transfer_out", "location-a", -250_000,
				"movement-transfer-a", "ST-202607-0001", "transfer-a-out", "ledger-transfer-out.insert", intentA,
			),
			pharmaInventorySQLMovementLedgerMutation(
				"ledger-transfer-in", "transfer_in", "location-b", 250_000,
				"movement-transfer-a", "ST-202607-0001", "transfer-a-in", "ledger-transfer-in.insert", intentA,
			),
			pharmaInventorySQLBalanceMutation(
				"balance-b", "product-a", "lot-a", "BATCH-A", "warehouse-a", "area-a", "location-b",
				"balance-b.insert", intentA,
			),
			pharmaInventorySQLBalanceAdjustment("balance-a", -250_000, "balance-a.transfer-out", intentA),
			pharmaInventorySQLBalanceAdjustment("balance-b", 250_000, "balance-b.transfer-in", intentA),
			pharmaInventorySQLReturnTotalMutation(
				"return-total-a", "ledger-a", "purchase", 100_000, "return-total-a.insert", intentA,
			),
		}
		for _, mutation := range mutations {
			if _, mutateErr := serviceA.Mutate(tx.Context(), mutation); mutateErr != nil {
				return mutateErr
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("commit stock transfer transaction: %v", err)
	}
	assertPharmaInventorySQLMovementState(t, serviceA, scopeA, "movement-transfer-a", "posted")
	assertPharmaInventorySQLLocationQuantity(t, serviceA, "stock_balances", scopeA, "location-a", 1_000_000)
	assertPharmaInventorySQLLocationQuantity(t, serviceA, "stock_balances", scopeA, "location-b", 250_000)
	assertPharmaInventorySQLReturnTotal(t, serviceA, scopeA, "return-total-a", 100_000)
	assertPharmaInventorySQLLedgerBalanceReconciled(t, serviceA, scopeA)

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
	if page := pharmaInventorySQLQuery(t, serviceA, "stock_ledger", scopeA, nil); len(page.Records) != 3 {
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
	rollbackTransferLegs := `[` +
		`{"id":"transfer-rollback-out","legType":"transfer_out","productId":"product-a","lotId":"lot-a","batchNo":"BATCH-A",` +
		`"warehouseId":"warehouse-a","areaId":"area-a","locationId":"location-a","quantity":"-0.1"},` +
		`{"id":"transfer-rollback-in","legType":"transfer_in","productId":"product-a","lotId":"lot-a","batchNo":"BATCH-A",` +
		`"warehouseId":"warehouse-a","areaId":"area-a","locationId":"location-b","quantity":"0.1"}]`
	err = uow.Do(context.Background(), func(tx repository.Tx) error {
		mutations := []pluginsdk.DataMutation{
			pharmaInventorySQLMovementMutation(
				"movement-transfer-rollback", "ST-202607-ROLLBACK", rollbackTransferLegs,
				"movement-transfer-rollback.insert", intentA,
			),
			pharmaInventorySQLMovementLedgerMutation(
				"ledger-transfer-rollback-out", "transfer_out", "location-a", -100_000,
				"movement-transfer-rollback", "ST-202607-ROLLBACK", "transfer-rollback-out",
				"ledger-transfer-rollback-out.insert", intentA,
			),
			pharmaInventorySQLMovementLedgerMutation(
				"ledger-transfer-rollback-in", "transfer_in", "location-b", 100_000,
				"movement-transfer-rollback", "ST-202607-ROLLBACK", "transfer-rollback-in",
				"ledger-transfer-rollback-in.insert", intentA,
			),
			pharmaInventorySQLBalanceAdjustment(
				"balance-a", -100_000, "balance-a.transfer-rollback-out", intentA,
			),
			pharmaInventorySQLBalanceAdjustment(
				"balance-b", 100_000, "balance-b.transfer-rollback-in", intentA,
			),
			pharmaInventorySQLReturnTotalAdjustment(
				"return-total-a", 50_000, 1_250_000, "return-total-a.rollback-adjust", intentA,
			),
		}
		for _, mutation := range mutations {
			if _, mutateErr := serviceA.Mutate(tx.Context(), mutation); mutateErr != nil {
				return mutateErr
			}
		}
		return rollbackSentinel
	})
	if !errors.Is(err, rollbackSentinel) {
		t.Fatalf("outer transaction error=%v", err)
	}
	rollbackFilter := pharmaInventorySQLFilter("id", pluginsdk.DataValueString, "movement-transfer-rollback")
	if page := pharmaInventorySQLQuery(t, serviceA, "inventory_movements", scopeA, &rollbackFilter); len(page.Records) != 0 {
		t.Fatalf("rolled-back movement persisted: %+v", page)
	}
	for _, ledgerID := range []string{"ledger-transfer-rollback-out", "ledger-transfer-rollback-in"} {
		rollbackFilter = pharmaInventorySQLFilter("id", pluginsdk.DataValueString, ledgerID)
		if page := pharmaInventorySQLQuery(t, serviceA, "stock_ledger", scopeA, &rollbackFilter); len(page.Records) != 0 {
			t.Fatalf("rolled-back ledger %s persisted: %+v", ledgerID, page)
		}
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
	assertPharmaInventorySQLLocationQuantity(t, serviceA, "stock_balances", scopeA, "location-a", 1_000_000)
	assertPharmaInventorySQLLocationQuantity(t, serviceA, "stock_balances", scopeA, "location-b", 250_000)
	assertPharmaInventorySQLReturnTotal(t, serviceA, scopeA, "return-total-a", 100_000)
	assertPharmaInventorySQLLedgerBalanceReconciled(t, serviceA, scopeA)
	for _, key := range []string{
		"movement-transfer-rollback.insert",
		"ledger-transfer-rollback-out.insert",
		"ledger-transfer-rollback-in.insert",
		"balance-a.transfer-rollback-out",
		"balance-b.transfer-rollback-in",
		"return-total-a.rollback-adjust",
	} {
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
	if err != nil || len(restartedPlan.Applied) < 11 || restartedPlan.Applied[10].Version != 11 {
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
	assertPharmaInventorySQLMovementState(
		t, restartedServiceA, scopeA, "movement-transfer-a", "posted",
	)
	assertPharmaInventorySQLLocationQuantity(
		t, restartedServiceA, "stock_balances", scopeA, "location-a", 1_000_000,
	)
	assertPharmaInventorySQLLocationQuantity(
		t, restartedServiceA, "stock_balances", scopeA, "location-b", 250_000,
	)
	assertPharmaInventorySQLReturnTotal(t, restartedServiceA, scopeA, "return-total-a", 100_000)
	assertPharmaInventorySQLLedgerBalanceReconciled(t, restartedServiceA, scopeA)
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
	wantMovements bool,
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
	for _, logical := range []string{"inventory_movements", "stock_return_totals"} {
		table, err := registry.Resolve(pharmaInventorySQLPluginID, logical)
		if err != nil {
			t.Fatal(err)
		}
		if got := db.Migrator().HasTable(table.PhysicalName); got != wantMovements {
			t.Fatalf("table %s exists=%v want=%v", logical, got, wantMovements)
		}
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

func pharmaInventorySQLMovementMutation(
	id, number, legs, idempotencyKey string,
	scope pluginsdk.DataScopeIntent,
) pluginsdk.DataMutation {
	return pluginsdk.DataMutation{
		Table: "inventory_movements", Operation: pluginsdk.DataMutationInsert, Scope: scope,
		Key: map[string]pluginsdk.DataValue{"id": pharmaInventorySQLValue(pluginsdk.DataValueString, id)},
		Values: map[string]pluginsdk.DataValue{
			"number":                   pharmaInventorySQLValue(pluginsdk.DataValueString, number),
			"movement_type":            pharmaInventorySQLValue(pluginsdk.DataValueString, "stock_transfer"),
			"reason":                   pharmaInventorySQLValue(pluginsdk.DataValueString, "real SQLite transfer acceptance"),
			"source_warehouse_id":      pharmaInventorySQLValue(pluginsdk.DataValueString, "warehouse-a"),
			"source_area_id":           pharmaInventorySQLValue(pluginsdk.DataValueString, "area-a"),
			"source_location_id":       pharmaInventorySQLValue(pluginsdk.DataValueString, "location-a"),
			"destination_warehouse_id": pharmaInventorySQLValue(pluginsdk.DataValueString, "warehouse-a"),
			"destination_area_id":      pharmaInventorySQLValue(pluginsdk.DataValueString, "area-a"),
			"destination_location_id":  pharmaInventorySQLValue(pluginsdk.DataValueString, "location-b"),
			"legs":                     pharmaInventorySQLValue(pluginsdk.DataValueJSON, legs),
			"requester_id":             pharmaInventorySQLValue(pluginsdk.DataValueString, "actor-a"),
			"requested_at":             pharmaInventorySQLValue(pluginsdk.DataValueTimestamp, "2026-07-28T08:30:00Z"),
			"status":                   pharmaInventorySQLValue(pluginsdk.DataValueString, "posted"),
			"create_operation_key":     pharmaInventorySQLValue(pluginsdk.DataValueString, idempotencyKey),
			"request_hash":             pharmaInventorySQLValue(pluginsdk.DataValueString, "request-hash-"+id),
			"last_operation_key":       pharmaInventorySQLValue(pluginsdk.DataValueString, idempotencyKey),
			"last_operation_hash":      pharmaInventorySQLValue(pluginsdk.DataValueString, "request-hash-"+id),
			"posted_at":                pharmaInventorySQLValue(pluginsdk.DataValueTimestamp, "2026-07-28T08:30:00Z"),
		},
		Returning:      []string{"id", "number", "movement_type", "legs", "status"},
		IdempotencyKey: idempotencyKey,
	}
}

func pharmaInventorySQLMovementLedgerMutation(
	id, entryType, locationID string,
	quantityMicros int64,
	sourceID, sourceNumber, sourceLineID, idempotencyKey string,
	scope pluginsdk.DataScopeIntent,
) pluginsdk.DataMutation {
	return pluginsdk.DataMutation{
		Table: "stock_ledger", Operation: pluginsdk.DataMutationInsert, Scope: scope,
		Key: map[string]pluginsdk.DataValue{"id": pharmaInventorySQLValue(pluginsdk.DataValueString, id)},
		Values: map[string]pluginsdk.DataValue{
			"entry_type":              pharmaInventorySQLValue(pluginsdk.DataValueString, entryType),
			"product_id":              pharmaInventorySQLValue(pluginsdk.DataValueString, "product-a"),
			"lot_id":                  pharmaInventorySQLValue(pluginsdk.DataValueString, "lot-a"),
			"batch_no":                pharmaInventorySQLValue(pluginsdk.DataValueString, "BATCH-A"),
			"warehouse_id":            pharmaInventorySQLValue(pluginsdk.DataValueString, "warehouse-a"),
			"area_id":                 pharmaInventorySQLValue(pluginsdk.DataValueString, "area-a"),
			"location_id":             pharmaInventorySQLValue(pluginsdk.DataValueString, locationID),
			"quantity_micros":         pharmaInventorySQLValue(pluginsdk.DataValueInteger, fmt.Sprintf("%d", quantityMicros)),
			"source_document_type":    pharmaInventorySQLValue(pluginsdk.DataValueString, "stock_transfer"),
			"source_document_id":      pharmaInventorySQLValue(pluginsdk.DataValueString, sourceID),
			"source_document_number":  pharmaInventorySQLValue(pluginsdk.DataValueString, sourceNumber),
			"source_document_line_id": pharmaInventorySQLValue(pluginsdk.DataValueString, sourceLineID),
			"occurred_at":             pharmaInventorySQLValue(pluginsdk.DataValueTimestamp, "2026-07-28T08:30:00Z"),
		},
		Returning: []string{
			"id", "entry_type", "location_id", "quantity_micros",
			"source_document_type", "source_document_id", "source_document_line_id",
		},
		IdempotencyKey: idempotencyKey,
	}
}

func pharmaInventorySQLReturnTotalMutation(
	id, referenceLedgerID, returnType string,
	quantityMicros int64,
	idempotencyKey string,
	scope pluginsdk.DataScopeIntent,
) pluginsdk.DataMutation {
	return pluginsdk.DataMutation{
		Table: "stock_return_totals", Operation: pluginsdk.DataMutationInsert, Scope: scope,
		Key: map[string]pluginsdk.DataValue{"id": pharmaInventorySQLValue(pluginsdk.DataValueString, id)},
		Values: map[string]pluginsdk.DataValue{
			"reference_ledger_entry_id": pharmaInventorySQLValue(pluginsdk.DataValueString, referenceLedgerID),
			"return_type":               pharmaInventorySQLValue(pluginsdk.DataValueString, returnType),
			"quantity_micros":           pharmaInventorySQLValue(pluginsdk.DataValueInteger, fmt.Sprintf("%d", quantityMicros)),
		},
		Returning:      []string{"id", "reference_ledger_entry_id", "return_type", "quantity_micros"},
		IdempotencyKey: idempotencyKey,
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

func pharmaInventorySQLReturnTotalAdjustment(
	id string,
	delta, maximumMicros int64,
	idempotencyKey string,
	scope pluginsdk.DataScopeIntent,
) pluginsdk.DataMutation {
	minimum := pharmaInventorySQLValue(pluginsdk.DataValueInteger, "0")
	maximum := pharmaInventorySQLValue(
		pluginsdk.DataValueInteger,
		fmt.Sprintf("%d", maximumMicros),
	)
	return pluginsdk.DataMutation{
		Table: "stock_return_totals", Operation: pluginsdk.DataMutationAdjust, Scope: scope,
		Key: map[string]pluginsdk.DataValue{"id": pharmaInventorySQLValue(pluginsdk.DataValueString, id)},
		Adjustment: &pluginsdk.DataAdjustment{
			Field: "quantity_micros", Delta: pharmaInventorySQLValue(pluginsdk.DataValueInteger, fmt.Sprintf("%d", delta)),
			Minimum: &minimum, Maximum: &maximum,
		},
		Returning:      []string{"id", "quantity_micros"},
		IdempotencyKey: idempotencyKey,
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
		"inventory_lots": {
			"id", "product_id", "batch_no", "production_date", "expires_at",
		},
		"stock_ledger": {
			"id", "entry_type", "product_id", "lot_id", "location_id", "quantity_micros",
			"source_document_type", "source_document_id", "source_document_number",
			"source_document_line_id",
		},
		"stock_balances": {
			"id", "product_id", "lot_id", "location_id", "quantity_micros",
		},
		"inventory_movements": {
			"id", "number", "movement_type", "source_location_id", "destination_location_id",
			"legs", "status", "posted_at",
		},
		"stock_return_totals": {
			"id", "reference_ledger_entry_id", "return_type", "quantity_micros",
		},
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

func assertPharmaInventorySQLMovementState(
	t *testing.T,
	service *datastore.Service,
	scope pluginsdk.ScopePredicate,
	id, status string,
) {
	t.Helper()
	filter := pharmaInventorySQLFilter("id", pluginsdk.DataValueString, id)
	page := pharmaInventorySQLQuery(t, service, "inventory_movements", scope, &filter)
	if len(page.Records) != 1 {
		t.Fatalf("movement %s page=%+v", id, page)
	}
	movement := page.Records[0]
	if movement.Values["movement_type"].Value != "stock_transfer" ||
		movement.Values["source_location_id"].Value != "location-a" ||
		movement.Values["destination_location_id"].Value != "location-b" ||
		movement.Values["status"].Value != status ||
		movement.Values["legs"].Value == "" ||
		movement.Values["posted_at"].Value == "" {
		t.Fatalf("movement %s facts=%+v", id, movement)
	}

	filter = pharmaInventorySQLFilter("source_document_id", pluginsdk.DataValueString, id)
	ledgerPage := pharmaInventorySQLQuery(t, service, "stock_ledger", scope, &filter)
	if len(ledgerPage.Records) != 2 {
		t.Fatalf("movement %s ledger entries=%+v want paired entries", id, ledgerPage)
	}
	quantities := make(map[string]int64, 2)
	var net int64
	for _, record := range ledgerPage.Records {
		if record.Values["source_document_type"].Value != "stock_transfer" ||
			record.Values["source_document_id"].Value != id ||
			record.Values["source_document_line_id"].Value == "" {
			t.Fatalf("movement %s ledger trace=%+v", id, record)
		}
		entryType := record.Values["entry_type"].Value
		quantity := pharmaInventorySQLRecordQuantity(t, record)
		quantities[entryType] = quantity
		net += quantity
	}
	if quantities["transfer_out"] != -250_000 ||
		quantities["transfer_in"] != 250_000 ||
		net != 0 {
		t.Fatalf("movement %s paired quantities=%+v net=%d", id, quantities, net)
	}
}

func assertPharmaInventorySQLLocationQuantity(
	t *testing.T,
	service *datastore.Service,
	table string,
	scope pluginsdk.ScopePredicate,
	locationID string,
	want int64,
) {
	t.Helper()
	filter := pharmaInventorySQLFilter("location_id", pluginsdk.DataValueString, locationID)
	page := pharmaInventorySQLQuery(t, service, table, scope, &filter)
	if len(page.Records) != 1 {
		t.Fatalf("%s location %s records=%+v", table, locationID, page)
	}
	if got := pharmaInventorySQLRecordQuantity(t, page.Records[0]); got != want {
		t.Fatalf("%s location %s quantity=%d want=%d", table, locationID, got, want)
	}
}

func assertPharmaInventorySQLReturnTotal(
	t *testing.T,
	service *datastore.Service,
	scope pluginsdk.ScopePredicate,
	id string,
	want int64,
) {
	t.Helper()
	filter := pharmaInventorySQLFilter("id", pluginsdk.DataValueString, id)
	page := pharmaInventorySQLQuery(t, service, "stock_return_totals", scope, &filter)
	if len(page.Records) != 1 {
		t.Fatalf("return total %s page=%+v", id, page)
	}
	record := page.Records[0]
	if record.Values["reference_ledger_entry_id"].Value != "ledger-a" ||
		record.Values["return_type"].Value != "purchase" ||
		pharmaInventorySQLRecordQuantity(t, record) != want {
		t.Fatalf("return total %s facts=%+v want quantity=%d", id, record, want)
	}
}

func assertPharmaInventorySQLLedgerBalanceReconciled(
	t *testing.T,
	service *datastore.Service,
	scope pluginsdk.ScopePredicate,
) {
	t.Helper()
	ledgerByLocation := make(map[string]int64)
	for _, record := range pharmaInventorySQLQuery(t, service, "stock_ledger", scope, nil).Records {
		locationID := record.Values["location_id"].Value
		ledgerByLocation[locationID] += pharmaInventorySQLRecordQuantity(t, record)
	}
	balanceByLocation := make(map[string]int64)
	for _, record := range pharmaInventorySQLQuery(t, service, "stock_balances", scope, nil).Records {
		locationID := record.Values["location_id"].Value
		balanceByLocation[locationID] += pharmaInventorySQLRecordQuantity(t, record)
	}
	if !reflect.DeepEqual(ledgerByLocation, balanceByLocation) {
		t.Fatalf("ledger by location=%+v balances=%+v", ledgerByLocation, balanceByLocation)
	}
}

func pharmaInventorySQLRecordQuantity(t *testing.T, record pluginsdk.DataRecord) int64 {
	t.Helper()
	var quantity int64
	value, ok := record.Values["quantity_micros"]
	if !ok {
		t.Fatalf("record %s has no quantity_micros: %+v", record.Values["id"].Value, record)
	}
	if _, err := fmt.Sscan(value.Value, &quantity); err != nil {
		t.Fatalf("record %s quantity=%q err=%v", record.Values["id"].Value, value.Value, err)
	}
	return quantity
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
