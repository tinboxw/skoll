package pharmaoa

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type PurchaseInboundService interface {
	Create(ctx context.Context, in PurchaseInboundCreateInput) (*domainpharma.PurchaseInbound, error)
	List(ctx context.Context) ([]*domainpharma.PurchaseInbound, error)
	Get(ctx context.Context, id string) (*domainpharma.PurchaseInbound, error)
}

type PurchaseInboundCreateInput struct {
	Number          string
	PurchaseOrderID string
	WarehouseID     string
	AreaID          string
	LocationID      string
	Lines           []domainpharma.PurchaseInboundLine
	Attachments     []domainpharma.InboundAttachment
	ActorID         string
}

type purchaseInboundService struct {
	repo       pharmaoarepo.PurchaseInboundRepository
	purchases  PurchaseService
	warehouses WarehouseService
	inventory  InventoryService
	audit      auditsvc.Service
	nowFn      func() time.Time
	counter    atomic.Int64
}

func NewPurchaseInboundService(purchases PurchaseService, warehouses WarehouseService, inventory InventoryService, audit auditsvc.Service, repositories ...pharmaoarepo.PurchaseInboundRepository) PurchaseInboundService {
	repo := pharmaoarepo.PurchaseInboundRepository(pharmaoarepo.NewMemoryPurchaseInboundRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &purchaseInboundService{repo: repo, purchases: purchases, warehouses: warehouses, inventory: inventory, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *purchaseInboundService) Create(ctx context.Context, in PurchaseInboundCreateInput) (*domainpharma.PurchaseInbound, error) {
	if s == nil || s.purchases == nil || s.warehouses == nil || s.inventory == nil {
		return nil, fmt.Errorf("purchase inbound dependencies are required")
	}
	order, err := s.purchases.GetOrder(ctx, in.PurchaseOrderID)
	if err != nil {
		return nil, err
	}
	eligible, err := s.warehouses.ValidateMovementLocation(ctx, WarehouseMovementLocationInput{WarehouseID: in.WarehouseID, AreaID: in.AreaID, LocationID: in.LocationID})
	if err != nil {
		return nil, err
	}
	if !eligible.Allowed {
		return nil, fmt.Errorf("warehouse location is not eligible: %s", eligible.Reason)
	}
	if err := validateInboundAgainstOrder(in.Lines, order.Lines); err != nil {
		return nil, err
	}
	exists, err := s.numberExists(ctx, in.Number)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("purchase inbound number already exists")
	}
	id, err := s.nextAvailableInboundID(ctx)
	if err != nil {
		return nil, err
	}
	item, err := domainpharma.NewPurchaseInbound(id, in.Number, order.ID.String(), in.WarehouseID, in.AreaID, in.LocationID, in.ActorID, in.Lines, in.Attachments, s.nowFn())
	if err != nil {
		return nil, err
	}
	movements := make([]StockMovementInput, 0, len(item.Lines))
	for idx := range item.Lines {
		line := item.Lines[idx]
		movements = append(movements, StockMovementInput{ReferenceID: item.ID.String(), IdempotencyKey: fmt.Sprintf("inbound:%s:line:%d", strings.ToLower(strings.TrimSpace(item.Number)), idx+1), ProductID: line.ProductID, WarehouseID: item.WarehouseID, AreaID: item.AreaID, LocationID: item.LocationID, BatchNo: line.BatchNo, ProductionDate: line.ProductionDate, ExpiresAt: line.ExpiresAt, Quantity: line.Quantity, ActorID: in.ActorID})
	}
	results, movementErr := s.inventory.InboundMany(ctx, movements)
	if movementErr != nil {
		return nil, movementErr
	}
	for idx := range item.Lines {
		item.Lines[idx].BatchID = results[idx].Balance.BatchID
		item.Lines[idx].LedgerID = results[idx].Ledger.ID.String()
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	if s.audit != nil {
		_, _ = s.audit.Append(ctx, normalizeInboundActor(in.ActorID), "pharma_oa.inbound.create", "pharma_oa_purchase_inbound", item.ID.String(), map[string]any{"purchaseOrderId": order.ID.String(), "lines": len(item.Lines), "attachments": len(item.Attachments)})
	}
	return clonePurchaseInbound(item), nil
}

func (s *purchaseInboundService) List(ctx context.Context) ([]*domainpharma.PurchaseInbound, error) {
	rows, err := s.repo.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]*domainpharma.PurchaseInbound, 0, len(rows))
	for index := range rows {
		out = append(out, clonePurchaseInbound(&rows[index]))
	}
	return out, nil
}
func (s *purchaseInboundService) Get(ctx context.Context, id string) (*domainpharma.PurchaseInbound, error) {
	item, err := s.repo.Get(ctx, shared.ID(strings.TrimSpace(id)))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("purchase inbound not found")
	}
	return item, nil
}
func (s *purchaseInboundService) numberExists(ctx context.Context, number string) (bool, error) {
	items, err := s.repo.List(ctx, pharmaoarepo.ListFilter{Keyword: strings.TrimSpace(number)})
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true, nil
		}
	}
	return false, nil
}
func (s *purchaseInboundService) nextAvailableInboundID(ctx context.Context) (shared.ID, error) {
	for {
		id := shared.ID("purchase-inbound-" + strconv.FormatInt(s.counter.Add(1), 10))
		item, err := s.repo.Get(ctx, id)
		if err != nil {
			return "", err
		}
		if item == nil {
			return id, nil
		}
	}
}
func clonePurchaseInbound(item *domainpharma.PurchaseInbound) *domainpharma.PurchaseInbound {
	if item == nil {
		return nil
	}
	out := *item
	out.Lines = append([]domainpharma.PurchaseInboundLine(nil), item.Lines...)
	out.Attachments = append([]domainpharma.InboundAttachment(nil), item.Attachments...)
	return &out
}
func normalizeInboundActor(actor string) string {
	if strings.TrimSpace(actor) == "" {
		return "system"
	}
	return strings.TrimSpace(actor)
}
func validateInboundAgainstOrder(lines []domainpharma.PurchaseInboundLine, orderLines []domainpharma.PurchaseLine) error {
	ordered := map[string]float64{}
	for _, line := range orderLines {
		ordered[line.ProductID] += line.Quantity
	}
	received := map[string]int{}
	for _, line := range lines {
		received[strings.TrimSpace(line.ProductID)] += line.Quantity
	}
	for productID, quantity := range received {
		available, ok := ordered[productID]
		if !ok {
			return fmt.Errorf("product %s is not on purchase order", productID)
		}
		if math.Trunc(available) != available {
			return fmt.Errorf("purchase order quantity for %s must be a whole number", productID)
		}
		if float64(quantity) > available {
			return fmt.Errorf("inbound quantity exceeds purchase order quantity for %s", productID)
		}
	}
	return nil
}
