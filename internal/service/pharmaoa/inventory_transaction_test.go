package pharmaoa

import (
	"context"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestInventoryConcurrentOutboundCannotOversell(t *testing.T) {
	ctx := context.Background()
	service := NewInventoryService(nil)
	seed, err := service.Inbound(ctx, StockMovementInput{ReferenceID: "concurrent-seed", ProductID: "product-concurrent", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "batch-1", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 100})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 20
	var group sync.WaitGroup
	errors := make(chan error, workers)
	for index := 0; index < workers; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			_, moveErr := service.Outbound(ctx, StockMovementInput{ReferenceID: fmtID("concurrent-out", index), ProductID: "product-concurrent", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchID: seed.Balance.BatchID, Quantity: 10})
			errors <- moveErr
		}(index)
	}
	group.Wait()
	close(errors)
	success := 0
	for moveErr := range errors {
		if moveErr == nil {
			success++
		}
	}
	balances, _ := service.ListBalances(ctx)
	ledger, _ := service.ListLedger(ctx)
	if success != 10 || len(balances) != 1 || balances[0].Quantity != 0 || len(ledger) != 11 {
		t.Fatalf("oversell guard failed: success=%d balances=%+v ledger=%d", success, balances, len(ledger))
	}
}

func TestInventoryRetryIsIdempotent(t *testing.T) {
	ctx := context.Background()
	service := NewInventoryService(nil)
	seed, err := service.Inbound(ctx, StockMovementInput{ReferenceID: "idem-seed", ProductID: "product-idem", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "batch-idem", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 100})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 12
	var group sync.WaitGroup
	errors := make(chan error, workers)
	for index := 0; index < workers; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, moveErr := service.Outbound(ctx, StockMovementInput{ReferenceID: "same-outbound", ProductID: "product-idem", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchID: seed.Balance.BatchID, Quantity: 10})
			errors <- moveErr
		}()
	}
	group.Wait()
	close(errors)
	for moveErr := range errors {
		if moveErr != nil {
			t.Fatalf("idempotent retry failed: %v", moveErr)
		}
	}
	balances, _ := service.ListBalances(ctx)
	ledger, _ := service.ListLedger(ctx)
	if balances[0].Quantity != 90 || len(ledger) != 2 {
		t.Fatalf("retry duplicated side effects: balances=%+v ledger=%+v", balances, ledger)
	}
}

func TestInventoryTransferRetryUsesBusinessIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	service := NewInventoryService(nil)
	seed, err := service.Inbound(ctx, StockMovementInput{ReferenceID: "transfer-seed", ProductID: "product-transfer", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-source", BatchNo: "batch-transfer", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 100})
	if err != nil {
		t.Fatal(err)
	}
	input := StockTransferInput{ReferenceID: "transfer-attempt-1", IdempotencyKey: "transfer:business-001", ProductID: "product-transfer", FromWarehouseID: "warehouse-1", FromAreaID: "area-1", FromLocationID: "location-source", ToWarehouseID: "warehouse-1", ToAreaID: "area-1", ToLocationID: "location-target", BatchID: seed.Balance.BatchID, Quantity: 40}
	first, err := service.Transfer(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReferenceID = "transfer-attempt-2"
	second, err := service.Transfer(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Ledgers) != 2 || len(second.Ledgers) != 2 || first.Ledgers[0].ID != second.Ledgers[0].ID || first.Ledgers[1].ID != second.Ledgers[1].ID {
		t.Fatalf("transfer retry did not reuse ledger pair: first=%+v second=%+v", first.Ledgers, second.Ledgers)
	}
	balances, _ := service.ListBalances(ctx)
	ledger, _ := service.ListLedger(ctx)
	quantities := map[string]int{}
	for _, balance := range balances {
		quantities[balance.LocationID] = balance.Quantity
	}
	if quantities["location-source"] != 60 || quantities["location-target"] != 40 || len(ledger) != 3 {
		t.Fatalf("transfer retry duplicated side effects: quantities=%+v ledger=%+v", quantities, ledger)
	}
}

func TestInventoryBatchFailureRollsBackAllLines(t *testing.T) {
	ctx := context.Background()
	service := NewInventoryService(nil)
	first, _ := service.Inbound(ctx, StockMovementInput{ReferenceID: "rollback-seed-1", ProductID: "product-1", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "batch-1", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 5})
	second, _ := service.Inbound(ctx, StockMovementInput{ReferenceID: "rollback-seed-2", ProductID: "product-2", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "batch-2", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 5})
	_, err := service.OutboundMany(ctx, []StockMovementInput{{ReferenceID: "rollback-out", IdempotencyKey: "rollback:1", ProductID: "product-1", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchID: first.Balance.BatchID, Quantity: 2}, {ReferenceID: "rollback-out", IdempotencyKey: "rollback:2", ProductID: "product-2", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchID: second.Balance.BatchID, Quantity: 6}})
	if err == nil {
		t.Fatal("expected batch outbound failure")
	}
	balances, _ := service.ListBalances(ctx)
	ledger, _ := service.ListLedger(ctx)
	if len(balances) != 2 || balances[0].Quantity+balances[1].Quantity != 10 || len(ledger) != 2 {
		t.Fatalf("failed batch left partial writes: balances=%+v ledger=%+v", balances, ledger)
	}
}

func TestInventorySQLiteRestartPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.db")
	db := openInventorySQLite(t, path)
	service := NewInventoryService(nil, gormrepo.NewPharmaInventoryStore(db))
	seed, err := service.Inbound(context.Background(), StockMovementInput{ReferenceID: "restart-seed", ProductID: "product-restart", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "batch-restart", ProductionDate: time.Now().UTC().AddDate(0, -1, 0), ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 17})
	if err != nil {
		t.Fatal(err)
	}
	closeInventorySQLite(t, db)
	db = openInventorySQLite(t, path)
	defer closeInventorySQLite(t, db)
	service = NewInventoryService(nil, gormrepo.NewPharmaInventoryStore(db))
	balances, _ := service.ListBalances(context.Background())
	batches, _ := service.ListBatches(context.Background())
	ledger, _ := service.ListLedger(context.Background())
	if len(balances) != 1 || balances[0].Quantity != 17 || len(batches) != 1 || batches[0].ID.String() != seed.Balance.BatchID || len(ledger) != 1 {
		t.Fatalf("restart persistence failed: balances=%+v batches=%+v ledger=%+v", balances, batches, ledger)
	}
}

func openInventorySQLite(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&gormrepo.PharmaStockBatchModel{}, &gormrepo.PharmaStockBalanceModel{}, &gormrepo.PharmaStockLedgerModel{}, &gormrepo.PharmaStockLockModel{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func closeInventorySQLite(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err = sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
}
func fmtID(prefix string, index int) string { return prefix + "-" + strconv.Itoa(index) }
