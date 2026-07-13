package pharmaoa

import (
	"context"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
)

func TestPurchaseInboundIncreasesStockWithBatchAndExpiry(t *testing.T) {
	ctx := context.Background()
	suppliers := NewSupplierService(nil)
	supplier, _ := suppliers.Create(ctx, SupplierWriteInput{Code: "SUP-IN", Name: "Inbound Supplier", Qualifications: []domainpharma.SupplierQualification{{ID: "q", Name: "License", Number: "1", ExpiresAt: time.Now().AddDate(1, 0, 0)}}})
	purchases := NewPurchaseService(suppliers, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), nil)
	request, err := purchases.CreateRequest(ctx, PurchaseRequestCreateInput{Number: "PR-IN", SupplierID: supplier.ID.String(), RequesterID: "buyer", ApproverID: "manager", Lines: []domainpharma.PurchaseLine{{ProductID: "product-1", Quantity: 5, UnitPrice: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	order, err := purchases.ApproveRequest(ctx, request.ID.String(), PurchaseApprovalInput{ActorID: "manager"})
	if err != nil {
		t.Fatal(err)
	}
	warehouses := NewWarehouseService(nil)
	warehouse, err := warehouses.Create(ctx, WarehouseWriteInput{Code: "WH-IN", Name: "Inbound Warehouse", Region: "East", Areas: []domainpharma.WarehouseArea{{ID: "area-1", Code: "A1", Name: "Area", Locations: []domainpharma.WarehouseLocation{{ID: "loc-1", Code: "L1", Name: "Location"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	inventory := NewInventoryService(nil)
	service := NewPurchaseInboundService(purchases, warehouses, inventory, nil)
	now := time.Now().UTC()
	inbound, err := service.Create(ctx, PurchaseInboundCreateInput{Number: "IN-001", PurchaseOrderID: order.ID.String(), WarehouseID: warehouse.ID.String(), AreaID: "area-1", LocationID: "loc-1", ActorID: "receiver", Lines: []domainpharma.PurchaseInboundLine{{ProductID: "product-1", Quantity: 5, BatchNo: "B-001", ProductionDate: now.AddDate(0, -1, 0), ExpiresAt: now.AddDate(1, 0, 0)}}, Attachments: []domainpharma.InboundAttachment{{FileID: "file-1", FileName: "inspection.pdf", Size: 10}}})
	if err != nil {
		t.Fatal(err)
	}
	if inbound.Status != domainpharma.PurchaseInboundCompleted || inbound.Lines[0].BatchID == "" || inbound.Lines[0].LedgerID == "" {
		t.Fatalf("unexpected inbound: %+v", inbound)
	}
	balances, _ := inventory.ListBalances(ctx)
	ledger, _ := inventory.ListLedger(ctx)
	if len(balances) != 1 || balances[0].Quantity != 5 || len(ledger) != 1 || ledger[0].ReferenceID != inbound.ID.String() {
		t.Fatalf("unexpected inventory: %+v %+v", balances, ledger)
	}
}
