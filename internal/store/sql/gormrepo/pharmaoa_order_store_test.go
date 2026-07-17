package gormrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	mysqldriver "gorm.io/driver/mysql"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPharmaOrderRepositoriesPersistAcrossSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orders.db")
	db := openOrderSQLite(t, path)
	suffix := "sqlite"
	seedOrderContract(t, db, suffix)
	closeTestDB(t, db)
	db = openOrderSQLite(t, path)
	defer closeTestDB(t, db)
	assertOrderContract(t, db, suffix)
}
func TestPharmaOrderRepositoriesMySQLContract(t *testing.T) {
	dsn := os.Getenv("SKOLL_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("SKOLL_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysqldriver.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	runExternalOrderContract(t, db, "mysql")
}
func TestPharmaOrderRepositoriesPostgreSQLContract(t *testing.T) {
	dsn := os.Getenv("SKOLL_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("SKOLL_TEST_POSTGRES_DSN is not set")
	}
	db, err := gorm.Open(postgresdriver.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	runExternalOrderContract(t, db, "postgres")
}

func TestPharmaTransactionModelsMatchSchemaAndMigrations(t *testing.T) {
	db := TestDB(t)
	implemented := map[string]bool{"pharma_oa_stock_batches": true, "pharma_oa_stock_balances": true, "pharma_oa_stock_ledger": true, "pharma_oa_stock_locks": true, "pharma_oa_purchase_requests": true, "pharma_oa_purchase_orders": true, "pharma_oa_purchase_inbounds": true, "pharma_oa_sales_orders": true, "pharma_oa_sales_outbounds": true, "pharma_oa_stocktakes": true, "pharma_oa_transfers": true}
	for _, table := range pharmaoarepo.SchemaBaseline() {
		if !implemented[table.Name] {
			continue
		}
		for _, column := range table.Columns {
			if !db.Migrator().HasColumn(table.Name, column) {
				t.Fatalf("GORM model %s missing schema column %s", table.Name, column)
			}
		}
		for _, index := range table.Indexes {
			if !db.Migrator().HasIndex(table.Name, index.Name) {
				t.Fatalf("GORM model %s missing schema index %s", table.Name, index.Name)
			}
		}
	}
	for _, path := range []string{filepath.Join("..", "..", "..", "..", "migrations", "mysql", "20260718_000020_create_pharma_oa_inventory_orders.sql"), filepath.Join("..", "..", "..", "..", "migrations", "postgres", "20260718_000020_create_pharma_oa_inventory_orders.sql")} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ToLower(string(raw))
		if strings.Contains(text, "drop table") {
			t.Fatalf("migration must retain data: %s", path)
		}
		for table := range implemented {
			if !strings.Contains(text, table) {
				t.Fatalf("migration %s missing %s", path, table)
			}
		}
	}
}

func orderModels() []any {
	return []any{&PharmaPurchaseRequestModel{}, &PharmaPurchaseOrderModel{}, &PharmaPurchaseInboundModel{}, &PharmaSalesOrderModel{}, &PharmaSalesOutboundModel{}, &PharmaStocktakeModel{}, &PharmaTransferModel{}}
}
func openOrderSQLite(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(orderModels()...); err != nil {
		t.Fatal(err)
	}
	return db
}
func runExternalOrderContract(t *testing.T, db *gorm.DB, dialect string) {
	t.Helper()
	if err := db.AutoMigrate(orderModels()...); err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%s-%d", dialect, time.Now().UnixNano())
	seedOrderContract(t, db, suffix)
	assertOrderContract(t, db, suffix)
	t.Cleanup(func() {
		for _, model := range orderModels() {
			_ = db.Where("id LIKE ?", "%"+suffix+"%").Delete(model).Error
		}
	})
}

func seedOrderContract(t *testing.T, db *gorm.DB, suffix string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	purchase := NewPharmaPurchaseStore(db)
	request := &domainpharma.PurchaseRequest{ID: shared.ID("request-" + suffix), Number: "PR-" + suffix, SupplierID: "supplier-1", RequesterID: "buyer", ApproverID: "manager", Reason: "restock", Lines: []domainpharma.PurchaseLine{{ProductID: "product-1", Quantity: 5, UnitPrice: 2, Amount: 10}}, TotalAmount: 10, Status: domainpharma.PurchaseRequestApproved, WorkflowInstanceID: "workflow-1", PurchaseOrderID: "order-" + suffix, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}}
	order := &domainpharma.PurchaseOrder{ID: shared.ID("order-" + suffix), Number: "PO-" + suffix, PurchaseRequestID: request.ID.String(), SupplierID: "supplier-1", Lines: request.Lines, TotalAmount: 10, Status: domainpharma.PurchaseOrderOpen, ApprovedBy: "manager", ApprovedAt: now, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}}
	if err := purchase.CreateRequest(ctx, request); err != nil {
		t.Fatal(err)
	}
	if err := purchase.CreateOrder(ctx, order); err != nil {
		t.Fatal(err)
	}
	inbound := &domainpharma.PurchaseInbound{ID: shared.ID("inbound-" + suffix), Number: "IN-" + suffix, PurchaseOrderID: order.ID.String(), WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", Lines: []domainpharma.PurchaseInboundLine{{ProductID: "product-1", Quantity: 5, BatchNo: "B-1", BatchID: "batch-1", LedgerID: "ledger-in"}}, Attachments: []domainpharma.InboundAttachment{{FileID: "file-1", FileName: "inspection.pdf", Size: 12}}, Status: domainpharma.PurchaseInboundCompleted, ReceivedBy: "receiver", ReceivedAt: now, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}}
	if err := NewPharmaPurchaseInboundStore(db).Create(ctx, inbound); err != nil {
		t.Fatal(err)
	}
	sales := NewPharmaSalesStore(db)
	salesOrder := &domainpharma.SalesOrder{ID: shared.ID("sales-order-" + suffix), Number: "SO-" + suffix, CustomerID: "customer-1", Lines: []domainpharma.SalesLine{{ProductID: "product-1", Quantity: 2, UnitPrice: 3, Amount: 6}}, TotalAmount: 6, Status: domainpharma.SalesOrderOpen, CreatedBy: "seller", CreatedAt: now, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}}
	outbound := &domainpharma.SalesOutbound{ID: shared.ID("outbound-" + suffix), Number: "OUT-" + suffix, SalesOrderID: salesOrder.ID.String(), CustomerID: "customer-1", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", Lines: []domainpharma.SalesOutboundLine{{ProductID: "product-1", Quantity: 2, BatchID: "batch-1", LedgerID: "ledger-out"}}, Status: domainpharma.SalesOutboundCompleted, ShippedBy: "seller", ShippedAt: now, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}}
	if err := sales.CreateOrder(ctx, salesOrder); err != nil {
		t.Fatal(err)
	}
	if err := sales.CreateOutbound(ctx, outbound); err != nil {
		t.Fatal(err)
	}
	completed := now
	stocktake := &domainpharma.StocktakeOrder{ID: shared.ID("stocktake-" + suffix), Number: "ST-" + suffix, ProductID: "product-1", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchID: "batch-1", SystemQuantity: 5, ActualQuantity: 4, Difference: -1, ApproverID: "manager", WorkflowInstanceID: "workflow-stocktake", LedgerID: "ledger-stocktake", Status: domainpharma.StocktakeApproved, CreatedBy: "counter", ApprovedBy: "manager", CreatedAt: now, CompletedAt: &completed, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}}
	if err := NewPharmaStocktakeStore(db).Create(ctx, stocktake); err != nil {
		t.Fatal(err)
	}
	transfer := &domainpharma.TransferOrder{ID: shared.ID("transfer-" + suffix), Number: "TR-" + suffix, ProductID: "product-1", BatchID: "batch-1", Quantity: 1, FromWarehouseID: "warehouse-1", FromAreaID: "area-1", FromLocationID: "location-1", ToWarehouseID: "warehouse-2", ToAreaID: "area-2", ToLocationID: "location-2", OutLedgerID: "ledger-transfer-out", InLedgerID: "ledger-transfer-in", Status: domainpharma.TransferCompleted, TransferredBy: "manager", TransferredAt: now, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}}
	if err := NewPharmaTransferStore(db).Create(ctx, transfer); err != nil {
		t.Fatal(err)
	}
}

func assertOrderContract(t *testing.T, db *gorm.DB, suffix string) {
	t.Helper()
	ctx := context.Background()
	request, err := NewPharmaPurchaseStore(db).GetRequest(ctx, shared.ID("request-"+suffix))
	if err != nil || request == nil || len(request.Lines) != 1 {
		t.Fatalf("purchase request restart: %+v %v", request, err)
	}
	inbound, err := NewPharmaPurchaseInboundStore(db).Get(ctx, shared.ID("inbound-"+suffix))
	if err != nil || inbound == nil || len(inbound.Attachments) != 1 || inbound.Lines[0].LedgerID != "ledger-in" {
		t.Fatalf("inbound restart: %+v %v", inbound, err)
	}
	outbound, err := NewPharmaSalesStore(db).GetOutbound(ctx, shared.ID("outbound-"+suffix))
	if err != nil || outbound == nil || outbound.Lines[0].LedgerID != "ledger-out" {
		t.Fatalf("outbound restart: %+v %v", outbound, err)
	}
	stocktake, err := NewPharmaStocktakeStore(db).Get(ctx, shared.ID("stocktake-"+suffix))
	if err != nil || stocktake == nil || stocktake.Status != domainpharma.StocktakeApproved {
		t.Fatalf("stocktake restart: %+v %v", stocktake, err)
	}
	transfer, err := NewPharmaTransferStore(db).Get(ctx, shared.ID("transfer-"+suffix))
	if err != nil || transfer == nil || transfer.InLedgerID != "ledger-transfer-in" {
		t.Fatalf("transfer restart: %+v %v", transfer, err)
	}
}
