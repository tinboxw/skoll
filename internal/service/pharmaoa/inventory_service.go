package pharmaoa

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type InventoryService interface {
	Inbound(ctx context.Context, in StockMovementInput) (StockMovementResult, error)
	Outbound(ctx context.Context, in StockMovementInput) (StockMovementResult, error)
	Stocktake(ctx context.Context, in StocktakeInput) (StockMovementResult, error)
	Transfer(ctx context.Context, in StockTransferInput) (StockTransferResult, error)
	LockStock(ctx context.Context, in StockLockInput) (*domainpharma.StockLock, error)
	ListBalances(ctx context.Context) ([]domainpharma.StockBalance, error)
	ListBatches(ctx context.Context) ([]domainpharma.StockBatch, error)
	ListLedger(ctx context.Context) ([]domainpharma.StockLedgerEntry, error)
}

type StockMovementInput struct {
	ReferenceID    string
	ProductID      string
	WarehouseID    string
	AreaID         string
	LocationID     string
	BatchNo        string
	BatchID        string
	ProductionDate time.Time
	ExpiresAt      time.Time
	Quantity       int
	ActorID        string
}

type StocktakeInput struct {
	ReferenceID    string
	ProductID      string
	WarehouseID    string
	AreaID         string
	LocationID     string
	BatchID        string
	ActualQuantity int
	ActorID        string
}

type StockTransferInput struct {
	ReferenceID     string
	ProductID       string
	FromWarehouseID string
	FromAreaID      string
	FromLocationID  string
	ToWarehouseID   string
	ToAreaID        string
	ToLocationID    string
	BatchID         string
	Quantity        int
	ActorID         string
}

type StockLockInput struct {
	ProductID   string
	WarehouseID string
	AreaID      string
	LocationID  string
	BatchID     string
	Quantity    int
	Reason      string
	ActorID     string
}

type StockMovementResult struct {
	Balance domainpharma.StockBalance     `json:"balance"`
	Ledger  domainpharma.StockLedgerEntry `json:"ledger"`
}

type StockTransferResult struct {
	FromBalance domainpharma.StockBalance       `json:"fromBalance"`
	ToBalance   domainpharma.StockBalance       `json:"toBalance"`
	Ledgers     []domainpharma.StockLedgerEntry `json:"ledgers"`
}

type inventoryService struct {
	mu             sync.RWMutex
	balances       map[string]domainpharma.StockBalance
	batches        map[string]domainpharma.StockBatch
	locks          map[string]domainpharma.StockLock
	ledger         []domainpharma.StockLedgerEntry
	audit          auditsvc.Service
	nowFn          func() time.Time
	balanceCounter int64
	batchCounter   int64
	lockCounter    int64
	ledgerCounter  int64
}

func NewInventoryService(audit auditsvc.Service) InventoryService {
	return &inventoryService{
		balances: map[string]domainpharma.StockBalance{},
		batches:  map[string]domainpharma.StockBatch{},
		locks:    map[string]domainpharma.StockLock{},
		ledger:   []domainpharma.StockLedgerEntry{},
		audit:    audit,
		nowFn:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *inventoryService) Inbound(ctx context.Context, in StockMovementInput) (StockMovementResult, error) {
	if in.Quantity <= 0 {
		return StockMovementResult{}, fmt.Errorf("quantity must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	batch, err := s.ensureBatchLocked(in)
	if err != nil {
		return StockMovementResult{}, err
	}
	position := stockPositionFromMovement(in, batch.ID.String())
	balance, err := s.ensureBalanceLocked(position)
	if err != nil {
		return StockMovementResult{}, err
	}
	if err := balance.Apply(in.Quantity, s.nowFn()); err != nil {
		return StockMovementResult{}, err
	}
	entry, err := s.appendLedgerLocked(domainpharma.StockLedgerInbound, in.ReferenceID, position, in.Quantity, balance.Quantity)
	if err != nil {
		return StockMovementResult{}, err
	}
	s.balances[balanceKey(position)] = balance
	s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.inbound", in.ReferenceID, map[string]any{"quantity": in.Quantity, "productId": in.ProductID, "batchId": batch.ID.String()})
	return StockMovementResult{Balance: balance, Ledger: entry}, nil
}

func (s *inventoryService) Outbound(ctx context.Context, in StockMovementInput) (StockMovementResult, error) {
	if in.Quantity <= 0 {
		return StockMovementResult{}, fmt.Errorf("quantity must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	position := stockPositionFromMovement(in, strings.TrimSpace(in.BatchID))
	balance, ok := s.balances[balanceKey(position)]
	if !ok {
		return StockMovementResult{}, fmt.Errorf("stock balance not found")
	}
	if balance.AvailableQuantity() < in.Quantity {
		return StockMovementResult{}, fmt.Errorf("insufficient available stock")
	}
	if err := balance.Apply(-in.Quantity, s.nowFn()); err != nil {
		return StockMovementResult{}, err
	}
	entry, err := s.appendLedgerLocked(domainpharma.StockLedgerOutbound, in.ReferenceID, position, -in.Quantity, balance.Quantity)
	if err != nil {
		return StockMovementResult{}, err
	}
	s.balances[balanceKey(position)] = balance
	s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.outbound", in.ReferenceID, map[string]any{"quantity": in.Quantity, "productId": in.ProductID, "batchId": in.BatchID})
	return StockMovementResult{Balance: balance, Ledger: entry}, nil
}

func (s *inventoryService) Stocktake(ctx context.Context, in StocktakeInput) (StockMovementResult, error) {
	if in.ActualQuantity < 0 {
		return StockMovementResult{}, fmt.Errorf("actual quantity cannot be negative")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	position := domainpharma.StockPosition{
		ProductID:   in.ProductID,
		WarehouseID: in.WarehouseID,
		AreaID:      in.AreaID,
		LocationID:  in.LocationID,
		BatchID:     in.BatchID,
	}
	balance, ok := s.balances[balanceKey(position)]
	if !ok {
		return StockMovementResult{}, fmt.Errorf("stock balance not found")
	}
	delta := in.ActualQuantity - balance.Quantity
	if delta == 0 {
		return StockMovementResult{}, fmt.Errorf("stocktake delta cannot be zero")
	}
	if err := balance.Apply(delta, s.nowFn()); err != nil {
		return StockMovementResult{}, err
	}
	entry, err := s.appendLedgerLocked(domainpharma.StockLedgerStocktake, in.ReferenceID, position, delta, balance.Quantity)
	if err != nil {
		return StockMovementResult{}, err
	}
	s.balances[balanceKey(position)] = balance
	s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.stocktake", in.ReferenceID, map[string]any{"delta": delta, "actualQuantity": in.ActualQuantity})
	return StockMovementResult{Balance: balance, Ledger: entry}, nil
}

func (s *inventoryService) Transfer(ctx context.Context, in StockTransferInput) (StockTransferResult, error) {
	if in.Quantity <= 0 {
		return StockTransferResult{}, fmt.Errorf("quantity must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	from := domainpharma.StockPosition{ProductID: in.ProductID, WarehouseID: in.FromWarehouseID, AreaID: in.FromAreaID, LocationID: in.FromLocationID, BatchID: in.BatchID}
	to := domainpharma.StockPosition{ProductID: in.ProductID, WarehouseID: in.ToWarehouseID, AreaID: in.ToAreaID, LocationID: in.ToLocationID, BatchID: in.BatchID}
	fromBalance, ok := s.balances[balanceKey(from)]
	if !ok {
		return StockTransferResult{}, fmt.Errorf("source stock balance not found")
	}
	if fromBalance.AvailableQuantity() < in.Quantity {
		return StockTransferResult{}, fmt.Errorf("insufficient available stock")
	}
	toBalance, err := s.ensureBalanceLocked(to)
	if err != nil {
		return StockTransferResult{}, err
	}
	if err := fromBalance.Apply(-in.Quantity, s.nowFn()); err != nil {
		return StockTransferResult{}, err
	}
	if err := toBalance.Apply(in.Quantity, s.nowFn()); err != nil {
		return StockTransferResult{}, err
	}
	outEntry, err := s.appendLedgerLocked(domainpharma.StockLedgerTransferOut, in.ReferenceID, from, -in.Quantity, fromBalance.Quantity)
	if err != nil {
		return StockTransferResult{}, err
	}
	inEntry, err := s.appendLedgerLocked(domainpharma.StockLedgerTransferIn, in.ReferenceID, to, in.Quantity, toBalance.Quantity)
	if err != nil {
		return StockTransferResult{}, err
	}
	s.balances[balanceKey(from)] = fromBalance
	s.balances[balanceKey(to)] = toBalance
	s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.transfer", in.ReferenceID, map[string]any{"quantity": in.Quantity, "productId": in.ProductID, "batchId": in.BatchID})
	return StockTransferResult{FromBalance: fromBalance, ToBalance: toBalance, Ledgers: []domainpharma.StockLedgerEntry{outEntry, inEntry}}, nil
}

func (s *inventoryService) LockStock(ctx context.Context, in StockLockInput) (*domainpharma.StockLock, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	position := domainpharma.StockPosition{ProductID: in.ProductID, WarehouseID: in.WarehouseID, AreaID: in.AreaID, LocationID: in.LocationID, BatchID: in.BatchID}
	balance, ok := s.balances[balanceKey(position)]
	if !ok {
		return nil, fmt.Errorf("stock balance not found")
	}
	if err := balance.Lock(in.Quantity, s.nowFn()); err != nil {
		return nil, err
	}
	lock, err := domainpharma.NewStockLock(s.nextLockID(), balance.ID.String(), in.Quantity, in.Reason, s.nowFn())
	if err != nil {
		return nil, err
	}
	s.balances[balanceKey(position)] = balance
	s.locks[lock.ID.String()] = lock
	s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.lock", lock.ID.String(), map[string]any{"quantity": in.Quantity, "balanceId": balance.ID.String()})
	return cloneStockLock(lock), nil
}

func (s *inventoryService) ListBalances(context.Context) ([]domainpharma.StockBalance, error) {
	s.mu.RLock()
	items := make([]domainpharma.StockBalance, 0, len(s.balances))
	for _, item := range s.balances {
		items = append(items, item)
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID.String() < items[j].ID.String()
	})
	return items, nil
}

func (s *inventoryService) ListBatches(context.Context) ([]domainpharma.StockBatch, error) {
	s.mu.RLock()
	items := make([]domainpharma.StockBatch, 0, len(s.batches))
	for _, item := range s.batches {
		items = append(items, item)
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })
	return items, nil
}

func (s *inventoryService) ListLedger(context.Context) ([]domainpharma.StockLedgerEntry, error) {
	s.mu.RLock()
	items := append([]domainpharma.StockLedgerEntry(nil), s.ledger...)
	s.mu.RUnlock()
	return items, nil
}

func (s *inventoryService) ensureBatchLocked(in StockMovementInput) (domainpharma.StockBatch, error) {
	if batchID := strings.TrimSpace(in.BatchID); batchID != "" {
		batch, ok := s.batches[batchID]
		if !ok {
			return domainpharma.StockBatch{}, fmt.Errorf("stock batch not found")
		}
		return batch, nil
	}
	batch, err := domainpharma.NewStockBatch(s.nextBatchID(), in.ProductID, in.BatchNo, in.ProductionDate, in.ExpiresAt)
	if err != nil {
		return domainpharma.StockBatch{}, err
	}
	s.batches[batch.ID.String()] = batch
	return batch, nil
}

func (s *inventoryService) ensureBalanceLocked(position domainpharma.StockPosition) (domainpharma.StockBalance, error) {
	key := balanceKey(position)
	if item, ok := s.balances[key]; ok {
		return item, nil
	}
	item, err := domainpharma.NewStockBalance(s.nextBalanceID(), position, s.nowFn())
	if err != nil {
		return domainpharma.StockBalance{}, err
	}
	s.balances[key] = item
	return item, nil
}

func (s *inventoryService) appendLedgerLocked(operation domainpharma.StockLedgerOperation, referenceID string, position domainpharma.StockPosition, delta int, balanceAfter int) (domainpharma.StockLedgerEntry, error) {
	entry, err := domainpharma.NewStockLedgerEntry(s.nextLedgerID(), operation, referenceID, position, delta, balanceAfter, s.nowFn())
	if err != nil {
		return domainpharma.StockLedgerEntry{}, err
	}
	s.ledger = append(s.ledger, entry)
	return entry, nil
}

func (s *inventoryService) nextBalanceID() shared.ID {
	s.balanceCounter++
	return shared.ID("pharma-stock-balance-" + strconv.FormatInt(s.balanceCounter, 10))
}

func (s *inventoryService) nextBatchID() shared.ID {
	s.batchCounter++
	return shared.ID("pharma-stock-batch-" + strconv.FormatInt(s.batchCounter, 10))
}

func (s *inventoryService) nextLockID() shared.ID {
	s.lockCounter++
	return shared.ID("pharma-stock-lock-" + strconv.FormatInt(s.lockCounter, 10))
}

func (s *inventoryService) nextLedgerID() shared.ID {
	s.ledgerCounter++
	return shared.ID("pharma-stock-ledger-" + strconv.FormatInt(s.ledgerCounter, 10))
}

func (s *inventoryService) appendInventoryAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actor := strings.TrimSpace(actorID)
	if actor == "" {
		actor = "system"
	}
	_, _ = s.audit.Append(ctx, actor, action, "pharma_oa_inventory", resourceID, detail)
}

func stockPositionFromMovement(in StockMovementInput, batchID string) domainpharma.StockPosition {
	return domainpharma.StockPosition{
		ProductID:   in.ProductID,
		WarehouseID: in.WarehouseID,
		AreaID:      in.AreaID,
		LocationID:  in.LocationID,
		BatchID:     batchID,
	}
}

func balanceKey(position domainpharma.StockPosition) string {
	return strings.Join([]string{
		strings.TrimSpace(position.ProductID),
		strings.TrimSpace(position.WarehouseID),
		strings.TrimSpace(position.AreaID),
		strings.TrimSpace(position.LocationID),
		strings.TrimSpace(position.BatchID),
	}, "|")
}

func cloneStockLock(lock domainpharma.StockLock) *domainpharma.StockLock {
	next := lock
	return &next
}
