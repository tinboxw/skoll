package integration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestPharmaOAInventoryEndToEndSmoke(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	workflow := workflowsvc.NewService(workflowsvc.NewMemoryRepository())

	product := createInventorySmokeProduct(t, ctx, audit)
	inventory := pharmaoasvc.NewInventoryService(audit)
	suppliers := pharmaoasvc.NewSupplierService(audit)
	supplier, err := suppliers.Create(ctx, pharmaoasvc.SupplierWriteInput{
		Code: "SUP-E2E", Name: "Inventory Smoke Supplier", Rating: 5, ActorID: "master-data-admin",
		Qualifications: []domainpharma.SupplierQualification{{ID: "supplier-license", Name: "Drug Distribution License", Number: "SUP-LIC-E2E", ExpiresAt: now.AddDate(1, 0, 0)}},
	})
	if err != nil {
		t.Fatalf("create smoke supplier: %v", err)
	}
	purchases := pharmaoasvc.NewPurchaseService(suppliers, workflow, audit)

	request, err := purchases.CreateRequest(ctx, pharmaoasvc.PurchaseRequestCreateInput{
		Number: "PR-E2E-001", SupplierID: supplier.ID.String(), RequesterID: "buyer-e2e", ApproverID: "purchase-manager-e2e", Reason: "inventory smoke replenishment",
		Lines: []domainpharma.PurchaseLine{{ProductID: product.ID.String(), Quantity: 20, UnitPrice: 5}},
	})
	if err != nil {
		t.Fatalf("create purchase request: %v", err)
	}
	order, err := purchases.ApproveRequest(ctx, request.ID.String(), pharmaoasvc.PurchaseApprovalInput{ActorID: "purchase-manager-e2e", Comment: "approved by smoke"})
	if err != nil {
		t.Fatalf("approve purchase request: %v", err)
	}
	if order.PurchaseRequestID != request.ID.String() || order.TotalAmount != 100 {
		t.Fatalf("purchase approval did not create the expected order: %+v", order)
	}

	warehouses := pharmaoasvc.NewWarehouseService(audit)
	source, target := createInventorySmokeWarehouses(t, ctx, warehouses)
	inbounds := pharmaoasvc.NewPurchaseInboundService(purchases, warehouses, inventory, audit)
	inbound, err := inbounds.Create(ctx, pharmaoasvc.PurchaseInboundCreateInput{
		Number: "IN-E2E-001", PurchaseOrderID: order.ID.String(), WarehouseID: source.ID.String(), AreaID: "source-area", LocationID: "source-location", ActorID: "receiver-e2e",
		Lines:       []domainpharma.PurchaseInboundLine{{ProductID: product.ID.String(), Quantity: 20, BatchNo: "BATCH-E2E-001", ProductionDate: now.AddDate(0, -1, 0), ExpiresAt: now.AddDate(0, 0, 10)}},
		Attachments: []domainpharma.InboundAttachment{{FileID: "inspection-e2e", FileName: "inspection-e2e.pdf", Size: 1024}},
	})
	if err != nil {
		t.Fatalf("complete purchase inbound: %v", err)
	}
	if inbound.Status != domainpharma.PurchaseInboundCompleted || inbound.Lines[0].BatchID == "" || inbound.Lines[0].LedgerID == "" {
		t.Fatalf("purchase inbound did not produce batch inventory: %+v", inbound)
	}

	customers := pharmaoasvc.NewCustomerService(audit)
	customer, err := customers.Create(ctx, pharmaoasvc.CustomerWriteInput{
		Code: "CUST-E2E", Name: "Inventory Smoke Hospital", Region: "East", OrganizationID: "org-e2e", OwnerID: "sales-e2e", ActorID: "master-data-admin",
		Qualifications: []domainpharma.CustomerQualification{{Name: "Medical Institution License", Number: "CUST-LIC-E2E", ExpiresAt: now.AddDate(1, 0, 0)}},
		Scope:          pharmaoasvc.CustomerAccessScope{IncludeAll: true},
	})
	if err != nil {
		t.Fatalf("create smoke customer: %v", err)
	}
	sales := pharmaoasvc.NewSalesService(customers, warehouses, inventory, audit)
	salesOrder, err := sales.CreateOrder(ctx, pharmaoasvc.SalesOrderCreateInput{
		Number: "SO-E2E-001", CustomerID: customer.ID.String(), ActorID: "sales-e2e",
		Lines: []domainpharma.SalesLine{{ProductID: product.ID.String(), Quantity: 6, UnitPrice: 8}},
	})
	if err != nil {
		t.Fatalf("create sales order: %v", err)
	}
	outbound, err := sales.CreateOutbound(ctx, pharmaoasvc.SalesOutboundCreateInput{
		Number: "OUT-E2E-001", SalesOrderID: salesOrder.ID.String(), WarehouseID: source.ID.String(), AreaID: "source-area", LocationID: "source-location", ActorID: "sales-e2e",
		Lines: []domainpharma.SalesOutboundLine{{ProductID: product.ID.String(), BatchID: inbound.Lines[0].BatchID, Quantity: 6}},
	})
	if err != nil {
		t.Fatalf("complete sales outbound: %v", err)
	}
	if outbound.Status != domainpharma.SalesOutboundCompleted || outbound.Lines[0].LedgerID == "" {
		t.Fatalf("sales outbound did not produce ledger: %+v", outbound)
	}

	operations := pharmaoasvc.NewInventoryOperationService(inventory, warehouses, workflow, audit)
	stocktake, err := operations.CreateStocktake(ctx, pharmaoasvc.StocktakeOrderCreateInput{
		Number: "ST-E2E-001", ProductID: product.ID.String(), WarehouseID: source.ID.String(), AreaID: "source-area", LocationID: "source-location", BatchID: inbound.Lines[0].BatchID,
		ActualQuantity: 13, Reason: "smoke physical count", CreatorID: "counter-e2e", ApproverID: "inventory-manager-e2e",
	})
	if err != nil {
		t.Fatalf("create stocktake: %v", err)
	}
	stocktake, err = operations.ApproveStocktake(ctx, stocktake.ID.String(), pharmaoasvc.InventoryApprovalInput{ActorID: "inventory-manager-e2e", Comment: "count verified"})
	if err != nil || stocktake.Status != domainpharma.StocktakeApproved || stocktake.LedgerID == "" {
		t.Fatalf("approve stocktake: %+v %v", stocktake, err)
	}
	transfer, err := operations.CreateTransfer(ctx, pharmaoasvc.TransferOrderCreateInput{
		Number: "TR-E2E-001", ProductID: product.ID.String(), BatchID: inbound.Lines[0].BatchID, Quantity: 3,
		FromWarehouseID: source.ID.String(), FromAreaID: "source-area", FromLocationID: "source-location",
		ToWarehouseID: target.ID.String(), ToAreaID: "target-area", ToLocationID: "target-location", ActorID: "inventory-manager-e2e",
	})
	if err != nil || transfer.Status != domainpharma.TransferCompleted || transfer.OutLedgerID == "" || transfer.InLedgerID == "" {
		t.Fatalf("complete transfer: %+v %v", transfer, err)
	}

	assertInventorySmokeBalancesAndLedger(t, ctx, inventory, source.ID.String(), target.ID.String())

	notifications := notificationsvc.NewService(notificationsvc.NewMemoryRepository(), nil, nil)
	alerts := pharmaoasvc.NewInventoryAlertService(inventory, notifications, audit)
	job, err := alerts.Run(ctx, domainpharma.InventoryAlertPolicy{NearExpiryDays: 30, LowStockThreshold: 4, OverStockThreshold: 100, RecipientID: "inventory-manager-e2e"})
	if err != nil {
		t.Fatalf("run inventory alerts: %v", err)
	}
	if job.Status != domainpharma.InventoryAlertJobSucceeded || job.MatchedCount != 3 || job.CreatedCount != 3 {
		t.Fatalf("unexpected inventory alert job: %+v", job)
	}
	activeAlerts, err := alerts.ListAlerts(ctx, true)
	if err != nil || len(activeAlerts) != 3 {
		t.Fatalf("list inventory alerts: %+v %v", activeAlerts, err)
	}
	for _, alert := range activeAlerts {
		if alert.NotificationID == "" || !strings.Contains(alert.TargetPath, "balanceId=") || !strings.Contains(alert.TargetPath, "batchId=") {
			t.Fatalf("inventory alert is not actionable: %+v", alert)
		}
	}
	items, err := notifications.List(ctx, notificationsvc.Filter{ActorID: "inventory-manager-e2e", Category: notificationsvc.CategoryReminder, Status: notificationsvc.StatusPending})
	if err != nil || len(items) != 3 {
		t.Fatalf("inventory alerts did not enter notification center: %+v %v", items, err)
	}
}

func createInventorySmokeProduct(t *testing.T, ctx context.Context, audit auditsvc.Service) *domainpharma.Product {
	t.Helper()
	product, err := pharmaoasvc.NewProductService(audit).Create(ctx, pharmaoasvc.ProductWriteInput{
		Code: "DRUG-E2E", Name: "Inventory Smoke Drug", Spec: "10mg*20", DosageForm: "tablet", Manufacturer: "Skoll Pharma", ApprovalNumber: "NMPA-E2E-001", ActorID: "master-data-admin",
		Temperature: domainpharma.ProductTemperature{Required: true, MinCelsius: 2, MaxCelsius: 8},
	})
	if err != nil {
		t.Fatalf("create smoke product: %v", err)
	}
	return product
}

func createInventorySmokeWarehouses(t *testing.T, ctx context.Context, service pharmaoasvc.WarehouseService) (*domainpharma.Warehouse, *domainpharma.Warehouse) {
	t.Helper()
	source, err := service.Create(ctx, pharmaoasvc.WarehouseWriteInput{Code: "WH-E2E-SOURCE", Name: "Smoke Source Warehouse", Region: "East", ActorID: "master-data-admin", Areas: []domainpharma.WarehouseArea{{ID: "source-area", Code: "SRC", Name: "Source Area", Locations: []domainpharma.WarehouseLocation{{ID: "source-location", Code: "SRC-01", Name: "Source Location"}}}}})
	if err != nil {
		t.Fatalf("create source warehouse: %v", err)
	}
	target, err := service.Create(ctx, pharmaoasvc.WarehouseWriteInput{Code: "WH-E2E-TARGET", Name: "Smoke Target Warehouse", Region: "West", ActorID: "master-data-admin", Areas: []domainpharma.WarehouseArea{{ID: "target-area", Code: "DST", Name: "Target Area", Locations: []domainpharma.WarehouseLocation{{ID: "target-location", Code: "DST-01", Name: "Target Location"}}}}})
	if err != nil {
		t.Fatalf("create target warehouse: %v", err)
	}
	return source, target
}

func assertInventorySmokeBalancesAndLedger(t *testing.T, ctx context.Context, inventory pharmaoasvc.InventoryService, sourceWarehouseID, targetWarehouseID string) {
	t.Helper()
	balances, err := inventory.ListBalances(ctx)
	if err != nil {
		t.Fatalf("list inventory balances: %v", err)
	}
	quantityByWarehouse := map[string]int{}
	for _, balance := range balances {
		quantityByWarehouse[balance.WarehouseID] += balance.Quantity
	}
	if len(balances) != 2 || quantityByWarehouse[sourceWarehouseID] != 10 || quantityByWarehouse[targetWarehouseID] != 3 {
		t.Fatalf("unexpected end-to-end balances: %+v", balances)
	}
	ledger, err := inventory.ListLedger(ctx)
	if err != nil {
		t.Fatalf("list inventory ledger: %v", err)
	}
	wantOperations := []domainpharma.StockLedgerOperation{domainpharma.StockLedgerInbound, domainpharma.StockLedgerOutbound, domainpharma.StockLedgerStocktake, domainpharma.StockLedgerTransferOut, domainpharma.StockLedgerTransferIn}
	if len(ledger) != len(wantOperations) {
		t.Fatalf("expected %d immutable ledger entries, got %+v", len(wantOperations), ledger)
	}
	for index, want := range wantOperations {
		if ledger[index].Operation != want {
			t.Fatalf("ledger entry %d: want %s, got %+v", index, want, ledger[index])
		}
	}
}
