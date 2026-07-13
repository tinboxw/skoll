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

type SalesService interface {
	CreateOrder(ctx context.Context, in SalesOrderCreateInput) (*domainpharma.SalesOrder, error)
	ListOrders(ctx context.Context) ([]*domainpharma.SalesOrder, error)
	GetOrder(ctx context.Context, id string) (*domainpharma.SalesOrder, error)
	CreateOutbound(ctx context.Context, in SalesOutboundCreateInput) (*domainpharma.SalesOutbound, error)
	ListOutbounds(ctx context.Context) ([]*domainpharma.SalesOutbound, error)
	GetOutbound(ctx context.Context, id string) (*domainpharma.SalesOutbound, error)
}

type SalesOrderCreateInput struct {
	Number     string
	CustomerID string
	Lines      []domainpharma.SalesLine
	ActorID    string
}

type SalesOutboundCreateInput struct {
	Number       string
	SalesOrderID string
	WarehouseID  string
	AreaID       string
	LocationID   string
	Lines        []domainpharma.SalesOutboundLine
	ActorID      string
}

type salesService struct {
	mu              sync.RWMutex
	orders          map[string]*domainpharma.SalesOrder
	outbounds       map[string]*domainpharma.SalesOutbound
	customers       CustomerService
	warehouses      WarehouseService
	inventory       InventoryService
	audit           auditsvc.Service
	nowFn           func() time.Time
	orderCounter    int64
	outboundCounter int64
}

func NewSalesService(customers CustomerService, warehouses WarehouseService, inventory InventoryService, audit auditsvc.Service) SalesService {
	return &salesService{orders: map[string]*domainpharma.SalesOrder{}, outbounds: map[string]*domainpharma.SalesOutbound{}, customers: customers, warehouses: warehouses, inventory: inventory, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *salesService) CreateOrder(ctx context.Context, in SalesOrderCreateInput) (*domainpharma.SalesOrder, error) {
	if err := s.ensureCustomerEligible(ctx, in.CustomerID, in.ActorID, "pharma_oa.sales.order_denied"); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if salesNumberExists(s.orders, in.Number) {
		s.mu.Unlock()
		return nil, fmt.Errorf("sales order number already exists")
	}
	s.orderCounter++
	id := shared.ID("sales-order-" + strconv.FormatInt(s.orderCounter, 10))
	s.mu.Unlock()
	item, err := domainpharma.NewSalesOrder(id, in.Number, in.CustomerID, in.ActorID, in.Lines, s.nowFn())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.orders[item.ID.String()] = cloneSalesOrder(item)
	s.mu.Unlock()
	s.appendSalesAudit(ctx, in.ActorID, "pharma_oa.sales.order_create", "pharma_oa_sales_order", item.ID.String(), map[string]any{"customerId": item.CustomerID, "lines": len(item.Lines), "totalAmount": item.TotalAmount})
	return cloneSalesOrder(item), nil
}

func (s *salesService) ListOrders(context.Context) ([]*domainpharma.SalesOrder, error) {
	s.mu.RLock()
	out := make([]*domainpharma.SalesOrder, 0, len(s.orders))
	for _, item := range s.orders {
		out = append(out, cloneSalesOrder(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, nil
}

func (s *salesService) GetOrder(_ context.Context, id string) (*domainpharma.SalesOrder, error) {
	s.mu.RLock()
	item := cloneSalesOrder(s.orders[strings.TrimSpace(id)])
	s.mu.RUnlock()
	if item == nil {
		return nil, fmt.Errorf("sales order not found")
	}
	return item, nil
}

func (s *salesService) CreateOutbound(ctx context.Context, in SalesOutboundCreateInput) (*domainpharma.SalesOutbound, error) {
	if s == nil || s.customers == nil || s.warehouses == nil || s.inventory == nil {
		return nil, fmt.Errorf("sales outbound dependencies are required")
	}
	order, err := s.GetOrder(ctx, in.SalesOrderID)
	if err != nil {
		return nil, err
	}
	if err = s.ensureCustomerEligible(ctx, order.CustomerID, in.ActorID, "pharma_oa.sales.outbound_denied"); err != nil {
		return nil, err
	}
	location, err := s.warehouses.ValidateMovementLocation(ctx, WarehouseMovementLocationInput{WarehouseID: in.WarehouseID, AreaID: in.AreaID, LocationID: in.LocationID})
	if err != nil {
		return nil, err
	}
	if !location.Allowed {
		return nil, fmt.Errorf("warehouse location is not eligible: %s", location.Reason)
	}
	if err = validateOutboundAgainstOrder(in.Lines, order.Lines); err != nil {
		return nil, err
	}
	if err = s.validateAvailableStock(ctx, in); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if outboundNumberExists(s.outbounds, in.Number) {
		s.mu.Unlock()
		return nil, fmt.Errorf("sales outbound number already exists")
	}
	s.outboundCounter++
	id := shared.ID("sales-outbound-" + strconv.FormatInt(s.outboundCounter, 10))
	s.mu.Unlock()
	item, err := domainpharma.NewSalesOutbound(id, in.Number, order.ID.String(), order.CustomerID, in.WarehouseID, in.AreaID, in.LocationID, in.ActorID, in.Lines, s.nowFn())
	if err != nil {
		return nil, err
	}
	for idx := range item.Lines {
		line := &item.Lines[idx]
		result, movementErr := s.inventory.Outbound(ctx, StockMovementInput{ReferenceID: item.ID.String(), ProductID: line.ProductID, WarehouseID: item.WarehouseID, AreaID: item.AreaID, LocationID: item.LocationID, BatchID: line.BatchID, Quantity: line.Quantity, ActorID: in.ActorID})
		if movementErr != nil {
			return nil, movementErr
		}
		line.LedgerID = result.Ledger.ID.String()
	}
	s.mu.Lock()
	s.outbounds[item.ID.String()] = cloneSalesOutbound(item)
	s.mu.Unlock()
	s.appendSalesAudit(ctx, in.ActorID, "pharma_oa.sales.outbound_create", "pharma_oa_sales_outbound", item.ID.String(), map[string]any{"salesOrderId": order.ID.String(), "customerId": order.CustomerID, "lines": len(item.Lines)})
	return cloneSalesOutbound(item), nil
}

func (s *salesService) ListOutbounds(context.Context) ([]*domainpharma.SalesOutbound, error) {
	s.mu.RLock()
	out := make([]*domainpharma.SalesOutbound, 0, len(s.outbounds))
	for _, item := range s.outbounds {
		out = append(out, cloneSalesOutbound(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, nil
}

func (s *salesService) GetOutbound(_ context.Context, id string) (*domainpharma.SalesOutbound, error) {
	s.mu.RLock()
	item := cloneSalesOutbound(s.outbounds[strings.TrimSpace(id)])
	s.mu.RUnlock()
	if item == nil {
		return nil, fmt.Errorf("sales outbound not found")
	}
	return item, nil
}

func (s *salesService) ensureCustomerEligible(ctx context.Context, customerID, actorID, deniedAction string) error {
	if s == nil || s.customers == nil {
		return fmt.Errorf("customer service is required")
	}
	eligibility, err := s.customers.ValidateSalesCustomer(ctx, customerID, CustomerAccessScope{IncludeAll: true})
	if err != nil {
		s.appendSalesAudit(ctx, actorID, deniedAction, "pharma_oa_customer", customerID, map[string]any{"reason": err.Error()})
		s.appendSalesAudit(ctx, actorID, "pharma_oa.qualification.block", "pharma_oa_customer", customerID, map[string]any{"subjectType": "customer", "operation": deniedAction, "reason": err.Error()})
		return err
	}
	if !eligibility.Allowed {
		s.appendSalesAudit(ctx, actorID, deniedAction, "pharma_oa_customer", customerID, map[string]any{"reason": eligibility.Reason})
		s.appendSalesAudit(ctx, actorID, "pharma_oa.qualification.block", "pharma_oa_customer", customerID, map[string]any{"subjectType": "customer", "operation": deniedAction, "reason": eligibility.Reason})
		return fmt.Errorf("customer is not eligible for sales: %s", eligibility.Reason)
	}
	return nil
}

func (s *salesService) validateAvailableStock(ctx context.Context, in SalesOutboundCreateInput) error {
	balances, err := s.inventory.ListBalances(ctx)
	if err != nil {
		return err
	}
	available := map[string]int{}
	for _, balance := range balances {
		available[salesStockKey(balance.ProductID, balance.WarehouseID, balance.AreaID, balance.LocationID, balance.BatchID)] = balance.AvailableQuantity()
	}
	requested := map[string]int{}
	for _, line := range in.Lines {
		requested[salesStockKey(line.ProductID, in.WarehouseID, in.AreaID, in.LocationID, line.BatchID)] += line.Quantity
	}
	for key, quantity := range requested {
		if available[key] < quantity {
			return fmt.Errorf("insufficient available stock for %s", key)
		}
	}
	return nil
}

func validateOutboundAgainstOrder(lines []domainpharma.SalesOutboundLine, ordered []domainpharma.SalesLine) error {
	limits := map[string]int{}
	for _, line := range ordered {
		limits[line.ProductID] += line.Quantity
	}
	requested := map[string]int{}
	for _, line := range lines {
		requested[strings.TrimSpace(line.ProductID)] += line.Quantity
	}
	for productID, quantity := range requested {
		if limits[productID] == 0 {
			return fmt.Errorf("product %s is not on sales order", productID)
		}
		if quantity > limits[productID] {
			return fmt.Errorf("outbound quantity exceeds sales order quantity for %s", productID)
		}
	}
	return nil
}

func salesStockKey(productID, warehouseID, areaID, locationID, batchID string) string {
	return strings.Join([]string{strings.TrimSpace(productID), strings.TrimSpace(warehouseID), strings.TrimSpace(areaID), strings.TrimSpace(locationID), strings.TrimSpace(batchID)}, "|")
}
func salesNumberExists(items map[string]*domainpharma.SalesOrder, number string) bool {
	for _, item := range items {
		if strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true
		}
	}
	return false
}
func outboundNumberExists(items map[string]*domainpharma.SalesOutbound, number string) bool {
	for _, item := range items {
		if strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true
		}
	}
	return false
}
func cloneSalesOrder(item *domainpharma.SalesOrder) *domainpharma.SalesOrder {
	if item == nil {
		return nil
	}
	out := *item
	out.Lines = append([]domainpharma.SalesLine(nil), item.Lines...)
	return &out
}
func cloneSalesOutbound(item *domainpharma.SalesOutbound) *domainpharma.SalesOutbound {
	if item == nil {
		return nil
	}
	out := *item
	out.Lines = append([]domainpharma.SalesOutboundLine(nil), item.Lines...)
	return &out
}
func (s *salesService) appendSalesAudit(ctx context.Context, actorID, action, resource, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		actorID = "system"
	}
	_, _ = s.audit.Append(ctx, actorID, action, resource, resourceID, detail)
}
