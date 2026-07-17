package pharmaoa

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type memoryInventoryState struct {
	balances    map[string]domainpharma.StockBalance
	batches     map[string]domainpharma.StockBatch
	locks       map[string]domainpharma.StockLock
	ledger      []domainpharma.StockLedgerEntry
	ledgerByKey map[string]domainpharma.StockLedgerEntry
	ledgerByID  map[string]struct{}
}

type MemoryInventoryRepository struct {
	mu    sync.RWMutex
	state memoryInventoryState
}

func NewMemoryInventoryRepository() *MemoryInventoryRepository {
	return &MemoryInventoryRepository{state: newMemoryInventoryState()}
}

func newMemoryInventoryState() memoryInventoryState {
	return memoryInventoryState{
		balances: make(map[string]domainpharma.StockBalance), batches: make(map[string]domainpharma.StockBatch),
		locks: make(map[string]domainpharma.StockLock), ledger: []domainpharma.StockLedgerEntry{},
		ledgerByKey: make(map[string]domainpharma.StockLedgerEntry), ledgerByID: make(map[string]struct{}),
	}
}

func (r *MemoryInventoryRepository) Transact(ctx context.Context, fn func(tx InventoryTransaction) error) error {
	if r == nil || fn == nil {
		return fmt.Errorf("inventory transaction is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	working := cloneMemoryInventoryState(r.state)
	if err := fn(&memoryInventoryTransaction{state: &working}); err != nil {
		return err
	}
	r.state = working
	return nil
}

func (r *MemoryInventoryRepository) ListBalances(_ context.Context, filter ListFilter) ([]domainpharma.StockBalance, error) {
	r.mu.RLock()
	items := make([]domainpharma.StockBalance, 0, len(r.state.balances))
	for _, item := range r.state.balances {
		if matchesInventoryKeyword(filter.Keyword, item.ID.String(), item.ProductID, item.WarehouseID, item.BatchID) {
			items = append(items, item)
		}
	}
	r.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return pageInventoryItems(items, filter), nil
}

func (r *MemoryInventoryRepository) ListBatches(_ context.Context, filter ListFilter) ([]domainpharma.StockBatch, error) {
	r.mu.RLock()
	items := make([]domainpharma.StockBatch, 0, len(r.state.batches))
	for _, item := range r.state.batches {
		if matchesInventoryKeyword(filter.Keyword, item.ID.String(), item.ProductID, item.BatchNo) {
			items = append(items, item)
		}
	}
	r.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return pageInventoryItems(items, filter), nil
}

func (r *MemoryInventoryRepository) ListLedger(_ context.Context, filter ListFilter) ([]domainpharma.StockLedgerEntry, error) {
	r.mu.RLock()
	items := make([]domainpharma.StockLedgerEntry, 0, len(r.state.ledger))
	for _, item := range r.state.ledger {
		if matchesInventoryKeyword(filter.Keyword, item.ID.String(), item.ReferenceID, item.ProductID, item.BatchID) {
			items = append(items, item)
		}
	}
	r.mu.RUnlock()
	return pageInventoryItems(items, filter), nil
}

type memoryInventoryTransaction struct{ state *memoryInventoryState }

func (tx *memoryInventoryTransaction) GetBalanceForUpdate(_ context.Context, position domainpharma.StockPosition) (*domainpharma.StockBalance, error) {
	item, ok := tx.state.balances[inventoryPositionKey(position)]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (tx *memoryInventoryTransaction) GetBatch(_ context.Context, id shared.ID) (*domainpharma.StockBatch, error) {
	item, ok := tx.state.batches[id.String()]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (tx *memoryInventoryTransaction) GetBatchByProductAndNumber(_ context.Context, productID, batchNo string) (*domainpharma.StockBatch, error) {
	for _, item := range tx.state.batches {
		if item.ProductID == strings.TrimSpace(productID) && strings.EqualFold(item.BatchNo, strings.TrimSpace(batchNo)) {
			return &item, nil
		}
	}
	return nil, nil
}

func (tx *memoryInventoryTransaction) GetLedgerByIdempotencyKey(_ context.Context, key string) (*domainpharma.StockLedgerEntry, error) {
	item, ok := tx.state.ledgerByKey[strings.TrimSpace(key)]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (tx *memoryInventoryTransaction) UpsertBatch(_ context.Context, batch domainpharma.StockBatch) error {
	if batch.ID.IsZero() {
		return fmt.Errorf("stock batch id is required")
	}
	for id, item := range tx.state.batches {
		if id != batch.ID.String() && item.ProductID == batch.ProductID && strings.EqualFold(item.BatchNo, batch.BatchNo) {
			return fmt.Errorf("stock batch already exists")
		}
	}
	tx.state.batches[batch.ID.String()] = batch
	return nil
}

func (tx *memoryInventoryTransaction) UpsertBalance(_ context.Context, balance domainpharma.StockBalance) error {
	if balance.ID.IsZero() {
		return fmt.Errorf("stock balance id is required")
	}
	tx.state.balances[inventoryPositionKey(domainpharma.StockPosition{ProductID: balance.ProductID, WarehouseID: balance.WarehouseID, AreaID: balance.AreaID, LocationID: balance.LocationID, BatchID: balance.BatchID})] = balance
	return nil
}

func (tx *memoryInventoryTransaction) AppendLedger(_ context.Context, entry domainpharma.StockLedgerEntry, key string) error {
	key = strings.TrimSpace(key)
	if key == "" || entry.ID.IsZero() {
		return fmt.Errorf("stock ledger id and idempotency key are required")
	}
	if _, exists := tx.state.ledgerByKey[key]; exists {
		return fmt.Errorf("stock ledger idempotency key already exists")
	}
	if _, exists := tx.state.ledgerByID[entry.ID.String()]; exists {
		return fmt.Errorf("stock ledger id already exists")
	}
	tx.state.ledger = append(tx.state.ledger, entry)
	tx.state.ledgerByKey[key] = entry
	tx.state.ledgerByID[entry.ID.String()] = struct{}{}
	return nil
}

func (tx *memoryInventoryTransaction) UpsertLock(_ context.Context, lock domainpharma.StockLock) error {
	if lock.ID.IsZero() {
		return fmt.Errorf("stock lock id is required")
	}
	tx.state.locks[lock.ID.String()] = lock
	return nil
}

func cloneMemoryInventoryState(in memoryInventoryState) memoryInventoryState {
	out := newMemoryInventoryState()
	for key, item := range in.balances {
		out.balances[key] = item
	}
	for key, item := range in.batches {
		out.batches[key] = item
	}
	for key, item := range in.locks {
		out.locks[key] = item
	}
	out.ledger = append(out.ledger, in.ledger...)
	for key, item := range in.ledgerByKey {
		out.ledgerByKey[key] = item
	}
	for key := range in.ledgerByID {
		out.ledgerByID[key] = struct{}{}
	}
	return out
}

func inventoryPositionKey(position domainpharma.StockPosition) string {
	return strings.Join([]string{strings.TrimSpace(position.ProductID), strings.TrimSpace(position.WarehouseID), strings.TrimSpace(position.AreaID), strings.TrimSpace(position.LocationID), strings.TrimSpace(position.BatchID)}, "|")
}

func matchesInventoryKeyword(keyword string, values ...string) bool {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return true
	}
	return strings.Contains(strings.ToLower(strings.Join(values, " ")), keyword)
}

func pageInventoryItems[T any](items []T, filter ListFilter) []T {
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []T{}
	}
	end := len(items)
	if filter.Limit > 0 && offset+filter.Limit < end {
		end = offset + filter.Limit
	}
	return items[offset:end]
}
