package gormrepo

import (
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type PharmaStockBatchModel struct {
	ID             string    `gorm:"primaryKey;size:64"`
	ProductID      string    `gorm:"size:64;not null;uniqueIndex:uk_pharma_batches_product_no,priority:1;index:idx_pharma_batches_expiry,priority:2"`
	BatchNo        string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_batches_product_no,priority:2"`
	ProductionDate time.Time `gorm:"not null"`
	ExpiresAt      time.Time `gorm:"not null;index:idx_pharma_batches_expiry,priority:1"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      string `gorm:"size:64"`
	UpdatedBy      string `gorm:"size:64"`
}

func (PharmaStockBatchModel) TableName() string { return "pharma_oa_stock_batches" }

type PharmaStockBalanceModel struct {
	ID             string `gorm:"primaryKey;size:64"`
	ProductID      string `gorm:"size:64;not null;uniqueIndex:uk_pharma_balance_position,priority:1;index:idx_pharma_balance_warehouse_product,priority:2"`
	WarehouseID    string `gorm:"size:64;not null;uniqueIndex:uk_pharma_balance_position,priority:2;index:idx_pharma_balance_warehouse_product,priority:1"`
	AreaID         string `gorm:"size:64;not null;uniqueIndex:uk_pharma_balance_position,priority:3"`
	LocationID     string `gorm:"size:64;not null;uniqueIndex:uk_pharma_balance_position,priority:4"`
	BatchID        string `gorm:"size:64;not null;uniqueIndex:uk_pharma_balance_position,priority:5;index:idx_pharma_balance_batch"`
	Quantity       int    `gorm:"not null"`
	LockedQuantity int    `gorm:"not null"`
	Version        int64  `gorm:"not null;default:1"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      string `gorm:"size:64"`
	UpdatedBy      string `gorm:"size:64"`
}

func (PharmaStockBalanceModel) TableName() string { return "pharma_oa_stock_balances" }

type PharmaStockLedgerModel struct {
	ID             string    `gorm:"primaryKey;size:64"`
	Operation      string    `gorm:"size:32;not null"`
	ReferenceID    string    `gorm:"size:128;not null;index:idx_pharma_ledger_reference_time,priority:1"`
	ProductID      string    `gorm:"size:64;not null;index:idx_pharma_ledger_position_time,priority:1"`
	WarehouseID    string    `gorm:"size:64;not null;index:idx_pharma_ledger_position_time,priority:2"`
	AreaID         string    `gorm:"size:64;not null"`
	LocationID     string    `gorm:"size:64;not null"`
	BatchID        string    `gorm:"size:64;not null;index:idx_pharma_ledger_position_time,priority:3"`
	QuantityDelta  int       `gorm:"not null"`
	BalanceAfter   int       `gorm:"not null"`
	OccurredAt     time.Time `gorm:"not null;index:idx_pharma_ledger_reference_time,priority:2;index:idx_pharma_ledger_position_time,priority:4"`
	IdempotencyKey string    `gorm:"size:255;not null;uniqueIndex:uk_pharma_ledger_idempotency"`
	CreatedAt      time.Time
	CreatedBy      string `gorm:"size:64"`
}

func (PharmaStockLedgerModel) TableName() string { return "pharma_oa_stock_ledger" }

type PharmaStockLockModel struct {
	ID         string `gorm:"primaryKey;size:64"`
	BalanceID  string `gorm:"size:64;not null;index:idx_pharma_locks_balance_status,priority:1"`
	Quantity   int    `gorm:"not null"`
	Reason     string `gorm:"size:512"`
	Status     string `gorm:"size:32;not null;index:idx_pharma_locks_balance_status,priority:2"`
	ReleasedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	CreatedBy  string `gorm:"size:64"`
	UpdatedBy  string `gorm:"size:64"`
}

func (PharmaStockLockModel) TableName() string { return "pharma_oa_stock_locks" }

func stockBatchModelFromDomain(item domainpharma.StockBatch) PharmaStockBatchModel {
	return PharmaStockBatchModel{ID: item.ID.String(), ProductID: item.ProductID, BatchNo: item.BatchNo, ProductionDate: item.ProductionDate, ExpiresAt: item.ExpiresAt}
}

func (m PharmaStockBatchModel) toDomain() domainpharma.StockBatch {
	return domainpharma.StockBatch{ID: shared.ID(m.ID), ProductID: m.ProductID, BatchNo: m.BatchNo, ProductionDate: m.ProductionDate, ExpiresAt: m.ExpiresAt}
}

func stockBalanceModelFromDomain(item domainpharma.StockBalance, version int64) PharmaStockBalanceModel {
	return PharmaStockBalanceModel{ID: item.ID.String(), ProductID: item.ProductID, WarehouseID: item.WarehouseID, AreaID: item.AreaID, LocationID: item.LocationID, BatchID: item.BatchID, Quantity: item.Quantity, LockedQuantity: item.LockedQuantity, Version: version, UpdatedAt: item.UpdatedAt}
}

func (m PharmaStockBalanceModel) toDomain() domainpharma.StockBalance {
	return domainpharma.StockBalance{ID: shared.ID(m.ID), ProductID: m.ProductID, WarehouseID: m.WarehouseID, AreaID: m.AreaID, LocationID: m.LocationID, BatchID: m.BatchID, Quantity: m.Quantity, LockedQuantity: m.LockedQuantity, UpdatedAt: m.UpdatedAt}
}

func stockLedgerModelFromDomain(item domainpharma.StockLedgerEntry, key string) PharmaStockLedgerModel {
	return PharmaStockLedgerModel{ID: item.ID.String(), Operation: string(item.Operation), ReferenceID: item.ReferenceID, ProductID: item.ProductID, WarehouseID: item.WarehouseID, AreaID: item.AreaID, LocationID: item.LocationID, BatchID: item.BatchID, QuantityDelta: item.QuantityDelta, BalanceAfter: item.BalanceAfter, OccurredAt: item.OccurredAt, IdempotencyKey: key}
}

func (m PharmaStockLedgerModel) toDomain() domainpharma.StockLedgerEntry {
	return domainpharma.StockLedgerEntry{ID: shared.ID(m.ID), Operation: domainpharma.StockLedgerOperation(m.Operation), ReferenceID: m.ReferenceID, ProductID: m.ProductID, WarehouseID: m.WarehouseID, AreaID: m.AreaID, LocationID: m.LocationID, BatchID: m.BatchID, QuantityDelta: m.QuantityDelta, BalanceAfter: m.BalanceAfter, OccurredAt: m.OccurredAt}
}

func stockLockModelFromDomain(item domainpharma.StockLock) PharmaStockLockModel {
	var releasedAt *time.Time
	if !item.ReleasedAt.IsZero() {
		value := item.ReleasedAt
		releasedAt = &value
	}
	return PharmaStockLockModel{ID: item.ID.String(), BalanceID: item.BalanceID, Quantity: item.Quantity, Reason: item.Reason, Status: string(item.Status), ReleasedAt: releasedAt, CreatedAt: item.CreatedAt}
}
