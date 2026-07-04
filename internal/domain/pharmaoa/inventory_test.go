package pharmaoa

import (
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestStockBalanceApplyLockAndRelease(t *testing.T) {
	now := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	balance, err := NewStockBalance(shared.ID("balance-1"), inventoryTestPosition(), now)
	if err != nil {
		t.Fatalf("new balance: %v", err)
	}
	if err := balance.Apply(10, now); err != nil {
		t.Fatalf("apply inbound: %v", err)
	}
	if err := balance.Lock(4, now); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if balance.AvailableQuantity() != 6 {
		t.Fatalf("expected available 6, got %d", balance.AvailableQuantity())
	}
	if err := balance.Apply(-7, now); err == nil || !strings.Contains(err.Error(), "below locked") {
		t.Fatalf("expected locked quantity guard, got %v", err)
	}
	if err := balance.Release(2, now); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := balance.Apply(-7, now); err != nil {
		t.Fatalf("apply outbound after release: %v", err)
	}
	if balance.Quantity != 3 || balance.LockedQuantity != 2 {
		t.Fatalf("unexpected balance: %+v", balance)
	}
}

func TestStockLedgerEntryValidation(t *testing.T) {
	now := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	entry, err := NewStockLedgerEntry(shared.ID("ledger-1"), StockLedgerInbound, "inbound-1", inventoryTestPosition(), 5, 5, now)
	if err != nil {
		t.Fatalf("new ledger entry: %v", err)
	}
	if entry.Operation != StockLedgerInbound || entry.QuantityDelta != 5 || entry.BalanceAfter != 5 {
		t.Fatalf("unexpected ledger entry: %+v", entry)
	}
	if _, err := NewStockLedgerEntry(shared.ID("ledger-2"), StockLedgerOperation("bad"), "bad-1", inventoryTestPosition(), 1, 1, now); err == nil {
		t.Fatalf("expected invalid operation rejection")
	}
	if _, err := NewStockLedgerEntry(shared.ID("ledger-3"), StockLedgerInbound, "zero-1", inventoryTestPosition(), 0, 1, now); err == nil {
		t.Fatalf("expected zero delta rejection")
	}
}

func TestStockBatchRejectsInvalidExpiry(t *testing.T) {
	production := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	expires := production.AddDate(0, 0, -1)
	_, err := NewStockBatch(shared.ID("batch-1"), "product-1", "B-001", production, expires)
	if err == nil || !strings.Contains(err.Error(), "expiresAt") {
		t.Fatalf("expected invalid expiry rejection, got %v", err)
	}
}

func inventoryTestPosition() StockPosition {
	return StockPosition{
		ProductID:   "product-1",
		WarehouseID: "warehouse-1",
		AreaID:      "area-1",
		LocationID:  "location-1",
		BatchID:     "batch-1",
	}
}
