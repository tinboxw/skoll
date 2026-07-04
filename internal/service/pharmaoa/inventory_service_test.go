package pharmaoa

import (
	"context"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
)

func TestInventoryServiceMovementsProduceImmutableLedger(t *testing.T) {
	service := NewInventoryService(nil).(*inventoryService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	inbound, err := service.Inbound(ctx, StockMovementInput{
		ReferenceID:    "inbound-1",
		ProductID:      "product-1",
		WarehouseID:    "warehouse-1",
		AreaID:         "area-1",
		LocationID:     "location-1",
		BatchNo:        "B-001",
		ProductionDate: service.nowFn().AddDate(0, -1, 0),
		ExpiresAt:      service.nowFn().AddDate(1, 0, 0),
		Quantity:       100,
		ActorID:        "inventory-admin",
	})
	if err != nil {
		t.Fatalf("inbound: %v", err)
	}
	if inbound.Balance.Quantity != 100 || inbound.Ledger.Operation != domainpharma.StockLedgerInbound {
		t.Fatalf("unexpected inbound result: %+v", inbound)
	}

	outbound, err := service.Outbound(ctx, StockMovementInput{
		ReferenceID: "outbound-1",
		ProductID:   "product-1",
		WarehouseID: "warehouse-1",
		AreaID:      "area-1",
		LocationID:  "location-1",
		BatchID:     inbound.Balance.BatchID,
		Quantity:    30,
		ActorID:     "inventory-admin",
	})
	if err != nil {
		t.Fatalf("outbound: %v", err)
	}
	if outbound.Balance.Quantity != 70 || outbound.Ledger.QuantityDelta != -30 {
		t.Fatalf("unexpected outbound result: %+v", outbound)
	}

	stocktake, err := service.Stocktake(ctx, StocktakeInput{
		ReferenceID:    "stocktake-1",
		ProductID:      "product-1",
		WarehouseID:    "warehouse-1",
		AreaID:         "area-1",
		LocationID:     "location-1",
		BatchID:        inbound.Balance.BatchID,
		ActualQuantity: 65,
		ActorID:        "inventory-admin",
	})
	if err != nil {
		t.Fatalf("stocktake: %v", err)
	}
	if stocktake.Balance.Quantity != 65 || stocktake.Ledger.Operation != domainpharma.StockLedgerStocktake || stocktake.Ledger.QuantityDelta != -5 {
		t.Fatalf("unexpected stocktake result: %+v", stocktake)
	}

	transfer, err := service.Transfer(ctx, StockTransferInput{
		ReferenceID:     "transfer-1",
		ProductID:       "product-1",
		FromWarehouseID: "warehouse-1",
		FromAreaID:      "area-1",
		FromLocationID:  "location-1",
		ToWarehouseID:   "warehouse-2",
		ToAreaID:        "area-2",
		ToLocationID:    "location-2",
		BatchID:         inbound.Balance.BatchID,
		Quantity:        20,
		ActorID:         "inventory-admin",
	})
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if transfer.FromBalance.Quantity != 45 || transfer.ToBalance.Quantity != 20 || len(transfer.Ledgers) != 2 {
		t.Fatalf("unexpected transfer result: %+v", transfer)
	}
	if transfer.Ledgers[0].Operation != domainpharma.StockLedgerTransferOut || transfer.Ledgers[1].Operation != domainpharma.StockLedgerTransferIn {
		t.Fatalf("expected transfer out/in ledgers, got %+v", transfer.Ledgers)
	}

	ledgers, err := service.ListLedger(ctx)
	if err != nil {
		t.Fatalf("list ledger: %v", err)
	}
	if len(ledgers) != 5 {
		t.Fatalf("expected 5 immutable ledger entries, got %+v", ledgers)
	}
	first := ledgers[0]
	first.QuantityDelta = 999
	again, _ := service.ListLedger(ctx)
	if again[0].QuantityDelta != 100 {
		t.Fatalf("ledger list should be immutable copy, got %+v", again[0])
	}
}

func TestInventoryServiceLocksBlockOutboundAndTransfer(t *testing.T) {
	service := NewInventoryService(nil).(*inventoryService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	inbound, err := service.Inbound(ctx, StockMovementInput{
		ReferenceID: "inbound-lock",
		ProductID:   "product-1",
		WarehouseID: "warehouse-1",
		AreaID:      "area-1",
		LocationID:  "location-1",
		BatchNo:     "B-LOCK",
		ExpiresAt:   service.nowFn().AddDate(1, 0, 0),
		Quantity:    10,
	})
	if err != nil {
		t.Fatalf("inbound: %v", err)
	}
	lock, err := service.LockStock(ctx, StockLockInput{
		ProductID:   "product-1",
		WarehouseID: "warehouse-1",
		AreaID:      "area-1",
		LocationID:  "location-1",
		BatchID:     inbound.Balance.BatchID,
		Quantity:    8,
		Reason:      "quality hold",
	})
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if lock.Status != domainpharma.StockLockActive {
		t.Fatalf("unexpected lock: %+v", lock)
	}
	_, err = service.Outbound(ctx, StockMovementInput{
		ReferenceID: "outbound-blocked",
		ProductID:   "product-1",
		WarehouseID: "warehouse-1",
		AreaID:      "area-1",
		LocationID:  "location-1",
		BatchID:     inbound.Balance.BatchID,
		Quantity:    3,
	})
	if err == nil || !strings.Contains(err.Error(), "insufficient available stock") {
		t.Fatalf("expected locked stock to block outbound, got %v", err)
	}
}
