package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type StockLedgerOperation string

const (
	StockLedgerInbound     StockLedgerOperation = "inbound"
	StockLedgerOutbound    StockLedgerOperation = "outbound"
	StockLedgerStocktake   StockLedgerOperation = "stocktake"
	StockLedgerTransferOut StockLedgerOperation = "transfer_out"
	StockLedgerTransferIn  StockLedgerOperation = "transfer_in"
)

type StockLockStatus string

const (
	StockLockActive   StockLockStatus = "active"
	StockLockReleased StockLockStatus = "released"
)

type StockBatch struct {
	ID             shared.ID `json:"id"`
	ProductID      string    `json:"productId"`
	BatchNo        string    `json:"batchNo"`
	ProductionDate time.Time `json:"productionDate"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

type StockBalance struct {
	ID             shared.ID `json:"id"`
	ProductID      string    `json:"productId"`
	WarehouseID    string    `json:"warehouseId"`
	AreaID         string    `json:"areaId"`
	LocationID     string    `json:"locationId"`
	BatchID        string    `json:"batchId"`
	Quantity       int       `json:"quantity"`
	LockedQuantity int       `json:"lockedQuantity"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type StockLedgerEntry struct {
	ID            shared.ID            `json:"id"`
	Operation     StockLedgerOperation `json:"operation"`
	ReferenceID   string               `json:"referenceId"`
	ProductID     string               `json:"productId"`
	WarehouseID   string               `json:"warehouseId"`
	AreaID        string               `json:"areaId"`
	LocationID    string               `json:"locationId"`
	BatchID       string               `json:"batchId"`
	QuantityDelta int                  `json:"quantityDelta"`
	BalanceAfter  int                  `json:"balanceAfter"`
	OccurredAt    time.Time            `json:"occurredAt"`
}

type StockLock struct {
	ID         shared.ID       `json:"id"`
	BalanceID  string          `json:"balanceId"`
	Quantity   int             `json:"quantity"`
	Reason     string          `json:"reason"`
	Status     StockLockStatus `json:"status"`
	CreatedAt  time.Time       `json:"createdAt"`
	ReleasedAt time.Time       `json:"releasedAt,omitempty"`
}

type StockPosition struct {
	ProductID   string
	WarehouseID string
	AreaID      string
	LocationID  string
	BatchID     string
}

func NewStockBatch(id shared.ID, productID string, batchNo string, productionDate time.Time, expiresAt time.Time) (StockBatch, error) {
	if id.IsZero() {
		return StockBatch{}, fmt.Errorf("batch id is required")
	}
	productID = strings.TrimSpace(productID)
	batchNo = strings.TrimSpace(batchNo)
	if productID == "" {
		return StockBatch{}, fmt.Errorf("productId is required")
	}
	if batchNo == "" {
		return StockBatch{}, fmt.Errorf("batchNo is required")
	}
	if !productionDate.IsZero() && !expiresAt.IsZero() && expiresAt.Before(productionDate) {
		return StockBatch{}, fmt.Errorf("expiresAt cannot be before productionDate")
	}
	return StockBatch{ID: id, ProductID: productID, BatchNo: batchNo, ProductionDate: productionDate.UTC(), ExpiresAt: expiresAt.UTC()}, nil
}

func NewStockBalance(id shared.ID, position StockPosition, now time.Time) (StockBalance, error) {
	if id.IsZero() {
		return StockBalance{}, fmt.Errorf("balance id is required")
	}
	if err := position.Validate(); err != nil {
		return StockBalance{}, err
	}
	return StockBalance{
		ID:          id,
		ProductID:   strings.TrimSpace(position.ProductID),
		WarehouseID: strings.TrimSpace(position.WarehouseID),
		AreaID:      strings.TrimSpace(position.AreaID),
		LocationID:  strings.TrimSpace(position.LocationID),
		BatchID:     strings.TrimSpace(position.BatchID),
		UpdatedAt:   now.UTC(),
	}, nil
}

func (b *StockBalance) Apply(delta int, now time.Time) error {
	if b == nil {
		return fmt.Errorf("stock balance is required")
	}
	next := b.Quantity + delta
	if next < 0 {
		return fmt.Errorf("stock quantity cannot be negative")
	}
	if next < b.LockedQuantity {
		return fmt.Errorf("stock quantity cannot be below locked quantity")
	}
	b.Quantity = next
	b.UpdatedAt = now.UTC()
	return nil
}

func (b *StockBalance) Lock(quantity int, now time.Time) error {
	if b == nil {
		return fmt.Errorf("stock balance is required")
	}
	if quantity <= 0 {
		return fmt.Errorf("lock quantity must be positive")
	}
	if b.AvailableQuantity() < quantity {
		return fmt.Errorf("insufficient available stock")
	}
	b.LockedQuantity += quantity
	b.UpdatedAt = now.UTC()
	return nil
}

func (b *StockBalance) Release(quantity int, now time.Time) error {
	if b == nil {
		return fmt.Errorf("stock balance is required")
	}
	if quantity <= 0 {
		return fmt.Errorf("release quantity must be positive")
	}
	if b.LockedQuantity < quantity {
		return fmt.Errorf("release quantity exceeds locked quantity")
	}
	b.LockedQuantity -= quantity
	b.UpdatedAt = now.UTC()
	return nil
}

func (b StockBalance) AvailableQuantity() int {
	return b.Quantity - b.LockedQuantity
}

func NewStockLedgerEntry(id shared.ID, operation StockLedgerOperation, referenceID string, position StockPosition, delta int, balanceAfter int, occurredAt time.Time) (StockLedgerEntry, error) {
	if id.IsZero() {
		return StockLedgerEntry{}, fmt.Errorf("ledger id is required")
	}
	if err := operation.Validate(); err != nil {
		return StockLedgerEntry{}, err
	}
	if err := position.Validate(); err != nil {
		return StockLedgerEntry{}, err
	}
	if delta == 0 {
		return StockLedgerEntry{}, fmt.Errorf("quantity delta cannot be zero")
	}
	return StockLedgerEntry{
		ID:            id,
		Operation:     operation,
		ReferenceID:   strings.TrimSpace(referenceID),
		ProductID:     strings.TrimSpace(position.ProductID),
		WarehouseID:   strings.TrimSpace(position.WarehouseID),
		AreaID:        strings.TrimSpace(position.AreaID),
		LocationID:    strings.TrimSpace(position.LocationID),
		BatchID:       strings.TrimSpace(position.BatchID),
		QuantityDelta: delta,
		BalanceAfter:  balanceAfter,
		OccurredAt:    occurredAt.UTC(),
	}, nil
}

func NewStockLock(id shared.ID, balanceID string, quantity int, reason string, now time.Time) (StockLock, error) {
	if id.IsZero() {
		return StockLock{}, fmt.Errorf("lock id is required")
	}
	balanceID = strings.TrimSpace(balanceID)
	if balanceID == "" {
		return StockLock{}, fmt.Errorf("balanceId is required")
	}
	if quantity <= 0 {
		return StockLock{}, fmt.Errorf("lock quantity must be positive")
	}
	return StockLock{ID: id, BalanceID: balanceID, Quantity: quantity, Reason: strings.TrimSpace(reason), Status: StockLockActive, CreatedAt: now.UTC()}, nil
}

func (l *StockLock) Release(now time.Time) error {
	if l == nil {
		return fmt.Errorf("stock lock is required")
	}
	if l.Status == StockLockReleased {
		return nil
	}
	l.Status = StockLockReleased
	l.ReleasedAt = now.UTC()
	return nil
}

func (p StockPosition) Validate() error {
	for field, value := range map[string]string{
		"productId":   p.ProductID,
		"warehouseId": p.WarehouseID,
		"areaId":      p.AreaID,
		"locationId":  p.LocationID,
		"batchId":     p.BatchID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}

func (op StockLedgerOperation) Validate() error {
	switch op {
	case StockLedgerInbound, StockLedgerOutbound, StockLedgerStocktake, StockLedgerTransferOut, StockLedgerTransferIn:
		return nil
	default:
		return fmt.Errorf("invalid stock ledger operation: %s", op)
	}
}
