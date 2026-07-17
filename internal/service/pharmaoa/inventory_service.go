package pharmaoa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type InventoryService interface {
	Inbound(ctx context.Context, in StockMovementInput) (StockMovementResult, error)
	InboundMany(ctx context.Context, inputs []StockMovementInput) ([]StockMovementResult, error)
	Outbound(ctx context.Context, in StockMovementInput) (StockMovementResult, error)
	OutboundMany(ctx context.Context, inputs []StockMovementInput) ([]StockMovementResult, error)
	Stocktake(ctx context.Context, in StocktakeInput) (StockMovementResult, error)
	Transfer(ctx context.Context, in StockTransferInput) (StockTransferResult, error)
	LockStock(ctx context.Context, in StockLockInput) (*domainpharma.StockLock, error)
	ListBalances(ctx context.Context) ([]domainpharma.StockBalance, error)
	ListBatches(ctx context.Context) ([]domainpharma.StockBatch, error)
	ListLedger(ctx context.Context) ([]domainpharma.StockLedgerEntry, error)
}

type StockMovementInput struct {
	ReferenceID    string
	IdempotencyKey string
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
	IdempotencyKey  string
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
	repo         pharmaoarepo.InventoryRepository
	audit        auditsvc.Service
	nowFn        func() time.Time
	lockCounter  atomic.Uint64
	batchCounter atomic.Uint64
}

func NewInventoryService(audit auditsvc.Service, repositories ...pharmaoarepo.InventoryRepository) InventoryService {
	repo := pharmaoarepo.InventoryRepository(pharmaoarepo.NewMemoryInventoryRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &inventoryService{repo: repo, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *inventoryService) Inbound(ctx context.Context, in StockMovementInput) (StockMovementResult, error) {
	items, err := s.InboundMany(ctx, []StockMovementInput{in})
	if err != nil {
		return StockMovementResult{}, err
	}
	return items[0], nil
}

func (s *inventoryService) InboundMany(ctx context.Context, inputs []StockMovementInput) ([]StockMovementResult, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("stock movements are required")
	}
	results := make([]StockMovementResult, 0, len(inputs))
	err := s.repo.Transact(ctx, func(tx pharmaoarepo.InventoryTransaction) error {
		results = results[:0]
		for _, in := range inputs {
			result, err := s.applyInbound(ctx, tx, in)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for index, in := range inputs {
		s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.inbound", in.ReferenceID, map[string]any{"quantity": in.Quantity, "productId": in.ProductID, "batchId": results[index].Balance.BatchID})
	}
	return results, nil
}

func (s *inventoryService) Outbound(ctx context.Context, in StockMovementInput) (StockMovementResult, error) {
	items, err := s.OutboundMany(ctx, []StockMovementInput{in})
	if err != nil {
		return StockMovementResult{}, err
	}
	return items[0], nil
}

func (s *inventoryService) OutboundMany(ctx context.Context, inputs []StockMovementInput) ([]StockMovementResult, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("stock movements are required")
	}
	results := make([]StockMovementResult, 0, len(inputs))
	err := s.repo.Transact(ctx, func(tx pharmaoarepo.InventoryTransaction) error {
		results = results[:0]
		for _, in := range inputs {
			result, err := s.applyOutbound(ctx, tx, in)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, in := range inputs {
		s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.outbound", in.ReferenceID, map[string]any{"quantity": in.Quantity, "productId": in.ProductID, "batchId": in.BatchID})
	}
	return results, nil
}

func (s *inventoryService) Stocktake(ctx context.Context, in StocktakeInput) (result StockMovementResult, err error) {
	if in.ActualQuantity < 0 {
		return result, fmt.Errorf("actual quantity cannot be negative")
	}
	position := domainpharma.StockPosition{ProductID: in.ProductID, WarehouseID: in.WarehouseID, AreaID: in.AreaID, LocationID: in.LocationID, BatchID: in.BatchID}
	err = s.repo.Transact(ctx, func(tx pharmaoarepo.InventoryTransaction) error {
		balance, getErr := tx.GetBalanceForUpdate(ctx, position)
		if getErr != nil {
			return getErr
		}
		if balance == nil {
			return fmt.Errorf("stock balance not found")
		}
		delta := in.ActualQuantity - balance.Quantity
		if delta == 0 {
			return fmt.Errorf("stocktake delta cannot be zero")
		}
		key, keyErr := movementIdempotencyKey(domainpharma.StockLedgerStocktake, in.ReferenceID, "", position)
		if keyErr != nil {
			return keyErr
		}
		if existing, findErr := tx.GetLedgerByIdempotencyKey(ctx, key); findErr != nil {
			return findErr
		} else if existing != nil {
			result = resultFromLedger(*balance, *existing)
			return nil
		}
		if applyErr := balance.Apply(delta, s.nowFn()); applyErr != nil {
			return applyErr
		}
		entry, entryErr := newDeterministicLedger(domainpharma.StockLedgerStocktake, in.ReferenceID, position, delta, balance.Quantity, key, s.nowFn())
		if entryErr != nil {
			return entryErr
		}
		if saveErr := tx.UpsertBalance(ctx, *balance); saveErr != nil {
			return saveErr
		}
		if saveErr := tx.AppendLedger(ctx, entry, key); saveErr != nil {
			return saveErr
		}
		result = StockMovementResult{Balance: *balance, Ledger: entry}
		return nil
	})
	if err != nil {
		return StockMovementResult{}, err
	}
	s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.stocktake", in.ReferenceID, map[string]any{"delta": result.Ledger.QuantityDelta, "actualQuantity": in.ActualQuantity})
	return result, nil
}

func (s *inventoryService) Transfer(ctx context.Context, in StockTransferInput) (result StockTransferResult, err error) {
	if in.Quantity <= 0 {
		return result, fmt.Errorf("quantity must be positive")
	}
	from := domainpharma.StockPosition{ProductID: in.ProductID, WarehouseID: in.FromWarehouseID, AreaID: in.FromAreaID, LocationID: in.FromLocationID, BatchID: in.BatchID}
	to := domainpharma.StockPosition{ProductID: in.ProductID, WarehouseID: in.ToWarehouseID, AreaID: in.ToAreaID, LocationID: in.ToLocationID, BatchID: in.BatchID}
	if stockPositionKey(from) == stockPositionKey(to) {
		return result, fmt.Errorf("transfer source and target must differ")
	}
	err = s.repo.Transact(ctx, func(tx pharmaoarepo.InventoryTransaction) error {
		fromBalance, getErr := tx.GetBalanceForUpdate(ctx, from)
		if getErr != nil {
			return getErr
		}
		if fromBalance == nil {
			return fmt.Errorf("source stock balance not found")
		}
		toBalance, getErr := tx.GetBalanceForUpdate(ctx, to)
		if getErr != nil {
			return getErr
		}
		if toBalance == nil {
			item, createErr := domainpharma.NewStockBalance(deterministicID("pharma-stock-balance", stockPositionKey(to)), to, s.nowFn())
			if createErr != nil {
				return createErr
			}
			toBalance = &item
		}
		outKey, keyErr := movementIdempotencyKey(domainpharma.StockLedgerTransferOut, in.ReferenceID, in.IdempotencyKey, from)
		if keyErr != nil {
			return keyErr
		}
		inKey, keyErr := movementIdempotencyKey(domainpharma.StockLedgerTransferIn, in.ReferenceID, in.IdempotencyKey, to)
		if keyErr != nil {
			return keyErr
		}
		existingOut, findErr := tx.GetLedgerByIdempotencyKey(ctx, outKey)
		if findErr != nil {
			return findErr
		}
		existingIn, findErr := tx.GetLedgerByIdempotencyKey(ctx, inKey)
		if findErr != nil {
			return findErr
		}
		if existingOut != nil || existingIn != nil {
			if existingOut == nil || existingIn == nil {
				return fmt.Errorf("incomplete idempotent transfer state")
			}
			result = StockTransferResult{FromBalance: balanceAtLedger(*fromBalance, *existingOut), ToBalance: balanceAtLedger(*toBalance, *existingIn), Ledgers: []domainpharma.StockLedgerEntry{*existingOut, *existingIn}}
			return nil
		}
		if fromBalance.AvailableQuantity() < in.Quantity {
			return fmt.Errorf("insufficient available stock")
		}
		if applyErr := fromBalance.Apply(-in.Quantity, s.nowFn()); applyErr != nil {
			return applyErr
		}
		if applyErr := toBalance.Apply(in.Quantity, s.nowFn()); applyErr != nil {
			return applyErr
		}
		outEntry, entryErr := newDeterministicLedger(domainpharma.StockLedgerTransferOut, in.ReferenceID, from, -in.Quantity, fromBalance.Quantity, outKey, s.nowFn())
		if entryErr != nil {
			return entryErr
		}
		inEntry, entryErr := newDeterministicLedger(domainpharma.StockLedgerTransferIn, in.ReferenceID, to, in.Quantity, toBalance.Quantity, inKey, s.nowFn())
		if entryErr != nil {
			return entryErr
		}
		if saveErr := tx.UpsertBalance(ctx, *fromBalance); saveErr != nil {
			return saveErr
		}
		if saveErr := tx.UpsertBalance(ctx, *toBalance); saveErr != nil {
			return saveErr
		}
		if saveErr := tx.AppendLedger(ctx, outEntry, outKey); saveErr != nil {
			return saveErr
		}
		if saveErr := tx.AppendLedger(ctx, inEntry, inKey); saveErr != nil {
			return saveErr
		}
		result = StockTransferResult{FromBalance: *fromBalance, ToBalance: *toBalance, Ledgers: []domainpharma.StockLedgerEntry{outEntry, inEntry}}
		return nil
	})
	if err != nil {
		return StockTransferResult{}, err
	}
	s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.transfer", in.ReferenceID, map[string]any{"quantity": in.Quantity, "productId": in.ProductID, "batchId": in.BatchID})
	return result, nil
}

func (s *inventoryService) LockStock(ctx context.Context, in StockLockInput) (result *domainpharma.StockLock, err error) {
	position := domainpharma.StockPosition{ProductID: in.ProductID, WarehouseID: in.WarehouseID, AreaID: in.AreaID, LocationID: in.LocationID, BatchID: in.BatchID}
	err = s.repo.Transact(ctx, func(tx pharmaoarepo.InventoryTransaction) error {
		balance, getErr := tx.GetBalanceForUpdate(ctx, position)
		if getErr != nil {
			return getErr
		}
		if balance == nil {
			return fmt.Errorf("stock balance not found")
		}
		if lockErr := balance.Lock(in.Quantity, s.nowFn()); lockErr != nil {
			return lockErr
		}
		seed := fmt.Sprintf("%s|%s|%d|%d", balance.ID, strings.TrimSpace(in.Reason), s.nowFn().UnixNano(), s.lockCounter.Add(1))
		lock, createErr := domainpharma.NewStockLock(deterministicID("pharma-stock-lock", seed), balance.ID.String(), in.Quantity, in.Reason, s.nowFn())
		if createErr != nil {
			return createErr
		}
		if saveErr := tx.UpsertBalance(ctx, *balance); saveErr != nil {
			return saveErr
		}
		if saveErr := tx.UpsertLock(ctx, lock); saveErr != nil {
			return saveErr
		}
		result = cloneStockLock(lock)
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.appendInventoryAudit(ctx, in.ActorID, "pharma_oa.inventory.lock", result.ID.String(), map[string]any{"quantity": in.Quantity, "balanceId": result.BalanceID})
	return result, nil
}

func (s *inventoryService) ListBalances(ctx context.Context) ([]domainpharma.StockBalance, error) {
	return s.repo.ListBalances(ctx, pharmaoarepo.ListFilter{})
}
func (s *inventoryService) ListBatches(ctx context.Context) ([]domainpharma.StockBatch, error) {
	return s.repo.ListBatches(ctx, pharmaoarepo.ListFilter{})
}
func (s *inventoryService) ListLedger(ctx context.Context) ([]domainpharma.StockLedgerEntry, error) {
	return s.repo.ListLedger(ctx, pharmaoarepo.ListFilter{})
}

func (s *inventoryService) applyInbound(ctx context.Context, tx pharmaoarepo.InventoryTransaction, in StockMovementInput) (StockMovementResult, error) {
	if in.Quantity <= 0 {
		return StockMovementResult{}, fmt.Errorf("quantity must be positive")
	}
	var batch domainpharma.StockBatch
	if batchID := strings.TrimSpace(in.BatchID); batchID != "" {
		item, err := tx.GetBatch(ctx, shared.ID(batchID))
		if err != nil {
			return StockMovementResult{}, err
		}
		if item == nil {
			return StockMovementResult{}, fmt.Errorf("stock batch not found")
		}
		batch = *item
	} else {
		item, err := tx.GetBatchByProductAndNumber(ctx, in.ProductID, in.BatchNo)
		if err != nil {
			return StockMovementResult{}, err
		}
		if item != nil {
			batch = *item
		} else {
			var id shared.ID
			for {
				id = shared.ID(fmt.Sprintf("pharma-stock-batch-%d", s.batchCounter.Add(1)))
				existing, getErr := tx.GetBatch(ctx, id)
				if getErr != nil {
					return StockMovementResult{}, getErr
				}
				if existing == nil {
					break
				}
			}
			batch, err = domainpharma.NewStockBatch(id, in.ProductID, in.BatchNo, in.ProductionDate, in.ExpiresAt)
			if err != nil {
				return StockMovementResult{}, err
			}
			if err = tx.UpsertBatch(ctx, batch); err != nil {
				return StockMovementResult{}, err
			}
		}
	}
	position := stockPositionFromMovement(in, batch.ID.String())
	key, err := movementIdempotencyKey(domainpharma.StockLedgerInbound, in.ReferenceID, in.IdempotencyKey, position)
	if err != nil {
		return StockMovementResult{}, err
	}
	if existing, findErr := tx.GetLedgerByIdempotencyKey(ctx, key); findErr != nil {
		return StockMovementResult{}, findErr
	} else if existing != nil {
		balance, getErr := tx.GetBalanceForUpdate(ctx, position)
		if getErr != nil {
			return StockMovementResult{}, getErr
		}
		if balance == nil {
			return StockMovementResult{}, fmt.Errorf("idempotent stock balance not found")
		}
		return resultFromLedger(*balance, *existing), nil
	}
	balance, err := tx.GetBalanceForUpdate(ctx, position)
	if err != nil {
		return StockMovementResult{}, err
	}
	if balance == nil {
		item, createErr := domainpharma.NewStockBalance(deterministicID("pharma-stock-balance", stockPositionKey(position)), position, s.nowFn())
		if createErr != nil {
			return StockMovementResult{}, createErr
		}
		balance = &item
	}
	if err = balance.Apply(in.Quantity, s.nowFn()); err != nil {
		return StockMovementResult{}, err
	}
	entry, err := newDeterministicLedger(domainpharma.StockLedgerInbound, in.ReferenceID, position, in.Quantity, balance.Quantity, key, s.nowFn())
	if err != nil {
		return StockMovementResult{}, err
	}
	if err = tx.UpsertBalance(ctx, *balance); err != nil {
		return StockMovementResult{}, err
	}
	if err = tx.AppendLedger(ctx, entry, key); err != nil {
		return StockMovementResult{}, err
	}
	return StockMovementResult{Balance: *balance, Ledger: entry}, nil
}

func (s *inventoryService) applyOutbound(ctx context.Context, tx pharmaoarepo.InventoryTransaction, in StockMovementInput) (StockMovementResult, error) {
	if in.Quantity <= 0 {
		return StockMovementResult{}, fmt.Errorf("quantity must be positive")
	}
	position := stockPositionFromMovement(in, strings.TrimSpace(in.BatchID))
	key, err := movementIdempotencyKey(domainpharma.StockLedgerOutbound, in.ReferenceID, in.IdempotencyKey, position)
	if err != nil {
		return StockMovementResult{}, err
	}
	balance, err := tx.GetBalanceForUpdate(ctx, position)
	if err != nil {
		return StockMovementResult{}, err
	}
	if balance == nil {
		return StockMovementResult{}, fmt.Errorf("stock balance not found")
	}
	if existing, findErr := tx.GetLedgerByIdempotencyKey(ctx, key); findErr != nil {
		return StockMovementResult{}, findErr
	} else if existing != nil {
		return resultFromLedger(*balance, *existing), nil
	}
	if balance.AvailableQuantity() < in.Quantity {
		return StockMovementResult{}, fmt.Errorf("insufficient available stock")
	}
	if err = balance.Apply(-in.Quantity, s.nowFn()); err != nil {
		return StockMovementResult{}, err
	}
	entry, err := newDeterministicLedger(domainpharma.StockLedgerOutbound, in.ReferenceID, position, -in.Quantity, balance.Quantity, key, s.nowFn())
	if err != nil {
		return StockMovementResult{}, err
	}
	if err = tx.UpsertBalance(ctx, *balance); err != nil {
		return StockMovementResult{}, err
	}
	if err = tx.AppendLedger(ctx, entry, key); err != nil {
		return StockMovementResult{}, err
	}
	return StockMovementResult{Balance: *balance, Ledger: entry}, nil
}

func movementIdempotencyKey(operation domainpharma.StockLedgerOperation, referenceID, explicit string, position domainpharma.StockPosition) (string, error) {
	referenceID = strings.TrimSpace(referenceID)
	if referenceID == "" {
		return "", fmt.Errorf("stock movement reference id is required")
	}
	if err := position.Validate(); err != nil {
		return "", err
	}
	seed := strings.TrimSpace(explicit)
	if seed == "" {
		seed = referenceID
	}
	return strings.Join([]string{string(operation), seed, stockPositionKey(position)}, "|"), nil
}

func newDeterministicLedger(operation domainpharma.StockLedgerOperation, referenceID string, position domainpharma.StockPosition, delta, balanceAfter int, key string, now time.Time) (domainpharma.StockLedgerEntry, error) {
	return domainpharma.NewStockLedgerEntry(deterministicID("pharma-stock-ledger", key), operation, referenceID, position, delta, balanceAfter, now)
}

func deterministicID(prefix, seed string) shared.ID {
	sum := sha256.Sum256([]byte(seed))
	return shared.ID(prefix + "-" + hex.EncodeToString(sum[:12]))
}

func resultFromLedger(balance domainpharma.StockBalance, ledger domainpharma.StockLedgerEntry) StockMovementResult {
	return StockMovementResult{Balance: balanceAtLedger(balance, ledger), Ledger: ledger}
}
func balanceAtLedger(balance domainpharma.StockBalance, ledger domainpharma.StockLedgerEntry) domainpharma.StockBalance {
	balance.Quantity = ledger.BalanceAfter
	return balance
}
func stockPositionFromMovement(in StockMovementInput, batchID string) domainpharma.StockPosition {
	return domainpharma.StockPosition{ProductID: in.ProductID, WarehouseID: in.WarehouseID, AreaID: in.AreaID, LocationID: in.LocationID, BatchID: batchID}
}
func stockPositionKey(position domainpharma.StockPosition) string {
	return strings.Join([]string{strings.TrimSpace(position.ProductID), strings.TrimSpace(position.WarehouseID), strings.TrimSpace(position.AreaID), strings.TrimSpace(position.LocationID), strings.TrimSpace(position.BatchID)}, "|")
}
func balanceKey(position domainpharma.StockPosition) string              { return stockPositionKey(position) }
func cloneStockLock(lock domainpharma.StockLock) *domainpharma.StockLock { next := lock; return &next }

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
