package pharmaoa

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPharmaDatabaseAcceptanceSeedRestartAndBackupRestore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	livePath := filepath.Join(dir, "pharma-oa.db")
	backupPath := filepath.Join(dir, "pharma-oa.backup.db")

	db := openPharmaAcceptanceDB(t, livePath)
	first := newPersistentDemoSeedFixture(db)
	initial, err := first.seed.Apply(ctx, "seed-admin")
	if err != nil || !initial.Applied || initial.Reused {
		t.Fatalf("fresh seed: %+v %v", initial, err)
	}
	assertPersistentDemoSeed(t, ctx, first, initial)
	closePharmaAcceptanceDB(t, db)
	copyAcceptanceFile(t, livePath, backupPath)

	db = openPharmaAcceptanceDB(t, livePath)
	restarted := newPersistentDemoSeedFixture(db)
	reused, err := restarted.seed.Apply(ctx, "restart-admin")
	if err != nil || !reused.Applied || !reused.Reused || reused.Entities != initial.Entities {
		t.Fatalf("restart seed reuse: initial=%+v reused=%+v err=%v", initial, reused, err)
	}
	assertPersistentDemoSeed(t, ctx, restarted, reused)
	if _, err := restarted.products.Create(ctx, ProductWriteInput{
		Code: "BACKUP-MUTATION-001", Name: "备份后变更药品", Spec: "1mg", DosageForm: "tablet",
		Manufacturer: "Skoll", ApprovalNumber: "BACKUP-NMPA-001", ActorID: "restore-admin",
		Temperature: domainpharma.ProductTemperature{Required: false},
	}); err != nil {
		t.Fatal(err)
	}
	closePharmaAcceptanceDB(t, db)

	copyAcceptanceFile(t, backupPath, livePath)
	db = openPharmaAcceptanceDB(t, livePath)
	defer closePharmaAcceptanceDB(t, db)
	restored := newPersistentDemoSeedFixture(db)
	products, err := restored.products.List(ctx, ProductListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 1 || products[0].Code != "DEMO-DRUG-001" {
		t.Fatalf("backup restore did not return to snapshot: %+v", products)
	}
	restoredSeed, err := restored.seed.Apply(ctx, "restore-admin")
	if err != nil || !restoredSeed.Reused || restoredSeed.Entities != initial.Entities {
		t.Fatalf("restored seed: %+v %v", restoredSeed, err)
	}
	assertPersistentDemoSeed(t, ctx, restored, restoredSeed)
}

type persistentDemoSeedFixture struct {
	seed      DemoSeedService
	products  ProductService
	inventory InventoryService
	purchases PurchaseService
	inbounds  PurchaseInboundService
	sales     SalesService
}

func newPersistentDemoSeedFixture(db *gorm.DB) persistentDemoSeedFixture {
	employees := NewEmployeeService(nil, gormrepo.NewPharmaEmployeeStore(db))
	products := NewProductService(nil, gormrepo.NewPharmaProductStore(db))
	suppliers := NewSupplierService(nil, gormrepo.NewPharmaSupplierStore(db))
	customers := NewCustomerService(nil, gormrepo.NewPharmaCustomerStore(db))
	warehouses := NewWarehouseService(nil, gormrepo.NewPharmaWarehouseStore(db))
	inventory := NewInventoryService(nil, gormrepo.NewPharmaInventoryStore(db))
	workflow := workflowsvc.NewService(workflowsvc.NewMemoryRepository())
	purchases := NewPurchaseService(suppliers, workflow, nil, gormrepo.NewPharmaPurchaseStore(db))
	inbounds := NewPurchaseInboundService(purchases, warehouses, inventory, nil, gormrepo.NewPharmaPurchaseInboundStore(db))
	sales := NewSalesService(customers, warehouses, inventory, nil, gormrepo.NewPharmaSalesStore(db))
	followUps := NewCustomerFollowUpService(customers, nil, gormrepo.NewPharmaCustomerFollowUpStore(db))
	seed := NewDemoSeedService(DemoSeedDependencies{
		Employees: employees, Products: products, Suppliers: suppliers, Customers: customers, Warehouses: warehouses,
		Purchases: purchases, Inbounds: inbounds, Sales: sales, Inventory: inventory, FollowUps: followUps,
	})
	return persistentDemoSeedFixture{seed: seed, products: products, inventory: inventory, purchases: purchases, inbounds: inbounds, sales: sales}
}

func assertPersistentDemoSeed(t *testing.T, ctx context.Context, fixture persistentDemoSeedFixture, snapshot DemoSeedSnapshot) {
	t.Helper()
	balances, err := fixture.inventory.ListBalances(ctx)
	if err != nil || len(balances) != 1 || balances[0].Quantity != 88 || balances[0].BatchID != snapshot.Entities.BatchID {
		t.Fatalf("inventory after restart/restore: %+v %v", balances, err)
	}
	ledger, err := fixture.inventory.ListLedger(ctx)
	if err != nil || len(ledger) != 2 {
		t.Fatalf("ledger after restart/restore: %+v %v", ledger, err)
	}
	requests, requestErr := fixture.purchases.ListRequests(ctx)
	orders, orderErr := fixture.purchases.ListOrders(ctx)
	inbounds, inboundErr := fixture.inbounds.List(ctx)
	salesOrders, salesOrderErr := fixture.sales.ListOrders(ctx)
	outbounds, outboundErr := fixture.sales.ListOutbounds(ctx)
	if requestErr != nil || orderErr != nil || inboundErr != nil || salesOrderErr != nil || outboundErr != nil ||
		len(requests) != 1 || len(orders) != 1 || len(inbounds) != 1 || len(salesOrders) != 1 || len(outbounds) != 1 {
		t.Fatalf("business chain after restart/restore: requests=%d orders=%d inbounds=%d sales=%d outbounds=%d errors=%v/%v/%v/%v/%v", len(requests), len(orders), len(inbounds), len(salesOrders), len(outbounds), requestErr, orderErr, inboundErr, salesOrderErr, outboundErr)
	}
}

func openPharmaAcceptanceDB(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path+"?_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gormrepo.AllModels()...); err != nil {
		t.Fatal(err)
	}
	return db
}

func closePharmaAcceptanceDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
}

func copyAcceptanceFile(t *testing.T, source, target string) {
	t.Helper()
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}
