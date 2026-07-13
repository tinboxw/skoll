package pharmaoa

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
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
	mu         sync.RWMutex
	items      map[string]*domainpharma.PurchaseInbound
	purchases  PurchaseService
	warehouses WarehouseService
	inventory  InventoryService
	audit      auditsvc.Service
	nowFn      func() time.Time
	counter    int64
}

func NewPurchaseInboundService(purchases PurchaseService, warehouses WarehouseService, inventory InventoryService, audit auditsvc.Service) PurchaseInboundService {
	return &purchaseInboundService{items: map[string]*domainpharma.PurchaseInbound{}, purchases: purchases, warehouses: warehouses, inventory: inventory, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
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
	s.mu.Lock()
	if s.numberExistsLocked(in.Number) {
		s.mu.Unlock()
		return nil, fmt.Errorf("purchase inbound number already exists")
	}
	s.counter++
	id := shared.ID("purchase-inbound-" + strconv.FormatInt(s.counter, 10))
	s.mu.Unlock()
	item, err := domainpharma.NewPurchaseInbound(id, in.Number, order.ID.String(), in.WarehouseID, in.AreaID, in.LocationID, in.ActorID, in.Lines, in.Attachments, s.nowFn())
	if err != nil {
		return nil, err
	}
	for idx := range item.Lines {
		line := &item.Lines[idx]
		result, movementErr := s.inventory.Inbound(ctx, StockMovementInput{ReferenceID: item.ID.String(), ProductID: line.ProductID, WarehouseID: item.WarehouseID, AreaID: item.AreaID, LocationID: item.LocationID, BatchNo: line.BatchNo, ProductionDate: line.ProductionDate, ExpiresAt: line.ExpiresAt, Quantity: line.Quantity, ActorID: in.ActorID})
		if movementErr != nil {
			return nil, movementErr
		}
		line.BatchID = result.Balance.BatchID
		line.LedgerID = result.Ledger.ID.String()
	}
	s.mu.Lock()
	s.items[item.ID.String()] = clonePurchaseInbound(item)
	s.mu.Unlock()
	if s.audit != nil {
		_, _ = s.audit.Append(ctx, normalizeInboundActor(in.ActorID), "pharma_oa.inbound.create", "pharma_oa_purchase_inbound", item.ID.String(), map[string]any{"purchaseOrderId": order.ID.String(), "lines": len(item.Lines), "attachments": len(item.Attachments)})
	}
	return clonePurchaseInbound(item), nil
}

func (s *purchaseInboundService) List(context.Context) ([]*domainpharma.PurchaseInbound, error) {
	s.mu.RLock()
	out := make([]*domainpharma.PurchaseInbound, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, clonePurchaseInbound(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, nil
}
func (s *purchaseInboundService) Get(_ context.Context, id string) (*domainpharma.PurchaseInbound, error) {
	s.mu.RLock()
	item := clonePurchaseInbound(s.items[strings.TrimSpace(id)])
	s.mu.RUnlock()
	if item == nil {
		return nil, fmt.Errorf("purchase inbound not found")
	}
	return item, nil
}
func (s *purchaseInboundService) numberExistsLocked(number string) bool {
	for _, item := range s.items {
		if strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true
		}
	}
	return false
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
