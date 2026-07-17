package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PharmaInventoryStore struct{ db *gorm.DB }

func NewPharmaInventoryStore(db *gorm.DB) *PharmaInventoryStore { return &PharmaInventoryStore{db: db} }

func (s *PharmaInventoryStore) Transact(ctx context.Context, fn func(tx pharmaoarepo.InventoryTransaction) error) error {
	if s == nil || s.db == nil || fn == nil {
		return fmt.Errorf("inventory database transaction is required")
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
			return fn(&pharmaInventoryTransaction{db: db, versions: map[string]int64{}})
		})
	})
}

func (s *PharmaInventoryStore) ListBalances(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.StockBalance, error) {
	var rows []PharmaStockBalanceModel
	query := inventoryListQuery(s.db.WithContext(ctx).Model(&PharmaStockBalanceModel{}), filter, []string{"id", "product_id", "warehouse_id", "batch_id"}).Order("id asc")
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domainpharma.StockBalance, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toDomain())
	}
	return out, nil
}

func (s *PharmaInventoryStore) ListBatches(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.StockBatch, error) {
	var rows []PharmaStockBatchModel
	query := inventoryListQuery(s.db.WithContext(ctx).Model(&PharmaStockBatchModel{}), filter, []string{"id", "product_id", "batch_no"}).Order("id asc")
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domainpharma.StockBatch, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toDomain())
	}
	return out, nil
}

func (s *PharmaInventoryStore) ListLedger(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.StockLedgerEntry, error) {
	var rows []PharmaStockLedgerModel
	query := inventoryListQuery(s.db.WithContext(ctx).Model(&PharmaStockLedgerModel{}), filter, []string{"id", "reference_id", "product_id", "batch_id"}).Order("occurred_at asc, id asc")
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domainpharma.StockLedgerEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toDomain())
	}
	return out, nil
}

type pharmaInventoryTransaction struct {
	db       *gorm.DB
	versions map[string]int64
}

func (tx *pharmaInventoryTransaction) GetBalanceForUpdate(ctx context.Context, position domainpharma.StockPosition) (*domainpharma.StockBalance, error) {
	var row PharmaStockBalanceModel
	err := tx.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("product_id = ? AND warehouse_id = ? AND area_id = ? AND location_id = ? AND batch_id = ?", strings.TrimSpace(position.ProductID), strings.TrimSpace(position.WarehouseID), strings.TrimSpace(position.AreaID), strings.TrimSpace(position.LocationID), strings.TrimSpace(position.BatchID)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tx.versions[row.ID] = row.Version
	item := row.toDomain()
	return &item, nil
}

func (tx *pharmaInventoryTransaction) GetBatch(ctx context.Context, id shared.ID) (*domainpharma.StockBatch, error) {
	var row PharmaStockBatchModel
	err := tx.db.WithContext(ctx).Where("id = ?", id.String()).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item := row.toDomain()
	return &item, nil
}

func (tx *pharmaInventoryTransaction) GetBatchByProductAndNumber(ctx context.Context, productID, batchNo string) (*domainpharma.StockBatch, error) {
	var row PharmaStockBatchModel
	err := tx.db.WithContext(ctx).Where("product_id = ? AND LOWER(batch_no) = ?", strings.TrimSpace(productID), strings.ToLower(strings.TrimSpace(batchNo))).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item := row.toDomain()
	return &item, nil
}

func (tx *pharmaInventoryTransaction) GetLedgerByIdempotencyKey(ctx context.Context, key string) (*domainpharma.StockLedgerEntry, error) {
	var row PharmaStockLedgerModel
	err := tx.db.WithContext(ctx).Where("idempotency_key = ?", strings.TrimSpace(key)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item := row.toDomain()
	return &item, nil
}

func (tx *pharmaInventoryTransaction) UpsertBatch(ctx context.Context, item domainpharma.StockBatch) error {
	row := stockBatchModelFromDomain(item)
	return tx.db.WithContext(ctx).Save(&row).Error
}

func (tx *pharmaInventoryTransaction) UpsertBalance(ctx context.Context, item domainpharma.StockBalance) error {
	version := tx.versions[item.ID.String()] + 1
	if version < 1 {
		version = 1
	}
	row := stockBalanceModelFromDomain(item, version)
	if tx.versions[item.ID.String()] == 0 {
		return tx.db.WithContext(ctx).Create(&row).Error
	}
	result := tx.db.WithContext(ctx).Model(&PharmaStockBalanceModel{}).Where("id = ? AND version = ?", item.ID.String(), tx.versions[item.ID.String()]).Updates(map[string]any{"quantity": item.Quantity, "locked_quantity": item.LockedQuantity, "updated_at": item.UpdatedAt, "version": version})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("stock balance concurrent update conflict")
	}
	tx.versions[item.ID.String()] = version
	return nil
}

func (tx *pharmaInventoryTransaction) AppendLedger(ctx context.Context, item domainpharma.StockLedgerEntry, key string) error {
	row := stockLedgerModelFromDomain(item, strings.TrimSpace(key))
	return tx.db.WithContext(ctx).Create(&row).Error
}

func (tx *pharmaInventoryTransaction) UpsertLock(ctx context.Context, item domainpharma.StockLock) error {
	row := stockLockModelFromDomain(item)
	return tx.db.WithContext(ctx).Save(&row).Error
}

func inventoryListQuery(query *gorm.DB, filter pharmaoarepo.ListFilter, columns []string) *gorm.DB {
	if keyword := strings.ToLower(strings.TrimSpace(filter.Keyword)); keyword != "" {
		parts := make([]string, 0, len(columns))
		args := make([]any, 0, len(columns))
		for _, column := range columns {
			parts = append(parts, "LOWER("+column+") LIKE ?")
			args = append(args, "%"+keyword+"%")
		}
		query = query.Where("("+strings.Join(parts, " OR ")+")", args...)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	return query
}
