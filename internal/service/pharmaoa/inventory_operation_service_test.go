package pharmaoa

import (
	"context"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
)

func TestStocktakeDifferenceRequiresApprovalAndPostsOnce(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	inventory := NewInventoryService(nil)
	warehouses, source, _ := inventoryOperationWarehouses(t, ctx)
	inbound, err := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "seed-stocktake", ProductID: "product-1", WarehouseID: source.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchNo: "B-001", ProductionDate: now.AddDate(0, -1, 0), ExpiresAt: now.AddDate(1, 0, 0), Quantity: 10, ActorID: "inventory-admin"})
	if err != nil {
		t.Fatalf("seed inventory: %v", err)
	}
	service := NewInventoryOperationService(inventory, warehouses, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), nil)
	order, err := service.CreateStocktake(ctx, StocktakeOrderCreateInput{Number: "ST-001", ProductID: "product-1", WarehouseID: source.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchID: inbound.Balance.BatchID, ActualQuantity: 8, Reason: "physical count", CreatorID: "counter-1", ApproverID: "approver-1"})
	if err != nil {
		t.Fatalf("create stocktake: %v", err)
	}
	if order.Status != domainpharma.StocktakePendingApproval || order.Difference != -2 {
		t.Fatalf("unexpected stocktake: %+v", order)
	}
	balances, _ := inventory.ListBalances(ctx)
	if balances[0].Quantity != 10 {
		t.Fatalf("pending stocktake changed inventory: %+v", balances[0])
	}
	_, err = service.ApproveStocktake(ctx, order.ID.String(), InventoryApprovalInput{ActorID: "wrong-approver"})
	if err == nil || !strings.Contains(err.Error(), "assignee mismatch") {
		t.Fatalf("expected assignee denial, got %v", err)
	}
	approved, err := service.ApproveStocktake(ctx, order.ID.String(), InventoryApprovalInput{ActorID: "approver-1", Comment: "verified"})
	if err != nil {
		t.Fatalf("approve stocktake: %v", err)
	}
	if approved.Status != domainpharma.StocktakeApproved || approved.LedgerID == "" {
		t.Fatalf("unexpected approved stocktake: %+v", approved)
	}
	again, err := service.ApproveStocktake(ctx, order.ID.String(), InventoryApprovalInput{ActorID: "approver-1"})
	if err != nil || again.LedgerID != approved.LedgerID {
		t.Fatalf("duplicate approval must be idempotent: %+v %v", again, err)
	}
	ledger, _ := inventory.ListLedger(ctx)
	if len(ledger) != 2 || ledger[1].Operation != domainpharma.StockLedgerStocktake || ledger[1].QuantityDelta != -2 || ledger[1].BalanceAfter != 8 {
		t.Fatalf("unexpected stocktake ledger: %+v", ledger)
	}
}

func TestStocktakeApprovalRejectsStaleBalance(t *testing.T) {
	ctx := context.Background()
	inventory := NewInventoryService(nil)
	warehouses, source, _ := inventoryOperationWarehouses(t, ctx)
	inbound, _ := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "seed-stale", ProductID: "product-1", WarehouseID: source.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchNo: "B-STALE", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 10})
	service := NewInventoryOperationService(inventory, warehouses, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), nil)
	order, _ := service.CreateStocktake(ctx, StocktakeOrderCreateInput{Number: "ST-STALE", ProductID: "product-1", WarehouseID: source.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchID: inbound.Balance.BatchID, ActualQuantity: 9, CreatorID: "counter", ApproverID: "approver"})
	_, _ = inventory.Outbound(ctx, StockMovementInput{ReferenceID: "other-movement", ProductID: "product-1", WarehouseID: source.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchID: inbound.Balance.BatchID, Quantity: 1})
	_, err := service.ApproveStocktake(ctx, order.ID.String(), InventoryApprovalInput{ActorID: "approver"})
	if err == nil || !strings.Contains(err.Error(), "changed after stocktake creation") {
		t.Fatalf("expected stale balance rejection, got %v", err)
	}
	current, _ := service.GetStocktake(ctx, order.ID.String())
	if current.Status != domainpharma.StocktakePendingApproval {
		t.Fatalf("stale stocktake workflow must remain pending: %+v", current)
	}
}

func TestCrossWarehouseTransferCreatesPairedLedger(t *testing.T) {
	ctx := context.Background()
	inventory := NewInventoryService(nil)
	warehouses, source, target := inventoryOperationWarehouses(t, ctx)
	inbound, _ := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "seed-transfer", ProductID: "product-1", WarehouseID: source.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchNo: "B-TRANSFER", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 10})
	service := NewInventoryOperationService(inventory, warehouses, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), nil)
	order, err := service.CreateTransfer(ctx, TransferOrderCreateInput{Number: "TR-001", ProductID: "product-1", BatchID: inbound.Balance.BatchID, Quantity: 4, FromWarehouseID: source.ID.String(), FromAreaID: "area-1", FromLocationID: "location-1", ToWarehouseID: target.ID.String(), ToAreaID: "area-2", ToLocationID: "location-2", ActorID: "inventory-admin"})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if order.Status != domainpharma.TransferCompleted || order.OutLedgerID == "" || order.InLedgerID == "" || order.OutLedgerID == order.InLedgerID {
		t.Fatalf("unexpected transfer order: %+v", order)
	}
	ledger, _ := inventory.ListLedger(ctx)
	if len(ledger) != 3 || ledger[1].Operation != domainpharma.StockLedgerTransferOut || ledger[1].QuantityDelta != -4 || ledger[2].Operation != domainpharma.StockLedgerTransferIn || ledger[2].QuantityDelta != 4 {
		t.Fatalf("expected paired transfer ledger: %+v", ledger)
	}
	balances, _ := inventory.ListBalances(ctx)
	if len(balances) != 2 || balances[0].Quantity+balances[1].Quantity != 10 {
		t.Fatalf("transfer must preserve total stock: %+v", balances)
	}
}

func TestInvalidInventoryOperationInputHasNoSideEffects(t *testing.T) {
	ctx := context.Background()
	inventory := NewInventoryService(nil)
	warehouses, source, target := inventoryOperationWarehouses(t, ctx)
	inbound, err := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "seed-invalid", ProductID: "product-1", WarehouseID: source.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchNo: "B-INVALID", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 10})
	if err != nil {
		t.Fatalf("seed inventory: %v", err)
	}
	repository := workflowsvc.NewMemoryRepository()
	workflow := workflowsvc.NewService(repository)
	service := NewInventoryOperationService(inventory, warehouses, workflow, nil)

	_, err = service.CreateStocktake(ctx, StocktakeOrderCreateInput{Number: "ST-NO-DIFF", ProductID: "product-1", WarehouseID: source.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchID: inbound.Balance.BatchID, ActualQuantity: 10, CreatorID: "counter", ApproverID: "approver"})
	if err == nil || !strings.Contains(err.Error(), "non-zero difference") {
		t.Fatalf("expected zero-difference rejection, got %v", err)
	}
	if _, err = workflow.GetDefinition(ctx, shared.ID("stocktake-definition-1")); err == nil {
		t.Fatal("invalid stocktake created a workflow definition")
	}

	_, err = service.CreateTransfer(ctx, TransferOrderCreateInput{ProductID: "product-1", BatchID: inbound.Balance.BatchID, Quantity: 4, FromWarehouseID: source.ID.String(), FromAreaID: "area-1", FromLocationID: "location-1", ToWarehouseID: target.ID.String(), ToAreaID: "area-2", ToLocationID: "location-2", ActorID: "inventory-admin"})
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("expected incomplete transfer rejection, got %v", err)
	}
	ledger, _ := inventory.ListLedger(ctx)
	balances, _ := inventory.ListBalances(ctx)
	if len(ledger) != 1 || len(balances) != 1 || balances[0].Quantity != 10 {
		t.Fatalf("invalid transfer changed inventory: ledger=%+v balances=%+v", ledger, balances)
	}
}

func inventoryOperationWarehouses(t *testing.T, ctx context.Context) (WarehouseService, *domainpharma.Warehouse, *domainpharma.Warehouse) {
	t.Helper()
	service := NewWarehouseService(nil)
	source, err := service.Create(ctx, WarehouseWriteInput{Code: "WH-SOURCE", Name: "Source Warehouse", Region: "East", Areas: []domainpharma.WarehouseArea{{ID: "area-1", Code: "A1", Name: "Area 1", Locations: []domainpharma.WarehouseLocation{{ID: "location-1", Code: "L1", Name: "Location 1"}}}}})
	if err != nil {
		t.Fatalf("create source warehouse: %v", err)
	}
	target, err := service.Create(ctx, WarehouseWriteInput{Code: "WH-TARGET", Name: "Target Warehouse", Region: "West", Areas: []domainpharma.WarehouseArea{{ID: "area-2", Code: "A2", Name: "Area 2", Locations: []domainpharma.WarehouseLocation{{ID: "location-2", Code: "L2", Name: "Location 2"}}}}})
	if err != nil {
		t.Fatalf("create target warehouse: %v", err)
	}
	return service, source, target
}
