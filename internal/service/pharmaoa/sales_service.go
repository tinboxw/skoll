package pharmaoa

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
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
	repo            pharmaoarepo.SalesRepository
	customers       CustomerService
	warehouses      WarehouseService
	inventory       InventoryService
	audit           auditsvc.Service
	nowFn           func() time.Time
	orderCounter    atomic.Int64
	outboundCounter atomic.Int64
}

func NewSalesService(customers CustomerService, warehouses WarehouseService, inventory InventoryService, audit auditsvc.Service, repositories ...pharmaoarepo.SalesRepository) SalesService {
	repo := pharmaoarepo.SalesRepository(pharmaoarepo.NewMemorySalesRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &salesService{repo: repo, customers: customers, warehouses: warehouses, inventory: inventory, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *salesService) CreateOrder(ctx context.Context, in SalesOrderCreateInput) (*domainpharma.SalesOrder, error) {
	if err := s.ensureCustomerEligible(ctx, in.CustomerID, in.ActorID, "pharma_oa.sales.order_denied"); err != nil {
		return nil, err
	}
	exists, err := s.orderNumberExists(ctx, in.Number)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("sales order number already exists")
	}
	id, err := s.nextAvailableSalesOrderID(ctx)
	if err != nil {
		return nil, err
	}
	item, err := domainpharma.NewSalesOrder(id, in.Number, in.CustomerID, in.ActorID, in.Lines, s.nowFn())
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateOrder(ctx, item); err != nil {
		return nil, err
	}
	s.appendSalesAudit(ctx, in.ActorID, "pharma_oa.sales.order_create", "pharma_oa_sales_order", item.ID.String(), map[string]any{"customerId": item.CustomerID, "lines": len(item.Lines), "totalAmount": item.TotalAmount})
	return cloneSalesOrder(item), nil
}

func (s *salesService) ListOrders(ctx context.Context) ([]*domainpharma.SalesOrder, error) {
	rows, err := s.repo.ListOrders(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]*domainpharma.SalesOrder, 0, len(rows))
	for index := range rows {
		out = append(out, cloneSalesOrder(&rows[index]))
	}
	return out, nil
}

func (s *salesService) GetOrder(ctx context.Context, id string) (*domainpharma.SalesOrder, error) {
	item, err := s.repo.GetOrder(ctx, shared.ID(strings.TrimSpace(id)))
	if err != nil {
		return nil, err
	}
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
	exists, err := s.outboundNumberExists(ctx, in.Number)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("sales outbound number already exists")
	}
	id, err := s.nextAvailableSalesOutboundID(ctx)
	if err != nil {
		return nil, err
	}
	item, err := domainpharma.NewSalesOutbound(id, in.Number, order.ID.String(), order.CustomerID, in.WarehouseID, in.AreaID, in.LocationID, in.ActorID, in.Lines, s.nowFn())
	if err != nil {
		return nil, err
	}
	movements := make([]StockMovementInput, 0, len(item.Lines))
	for idx := range item.Lines {
		line := item.Lines[idx]
		movements = append(movements, StockMovementInput{ReferenceID: item.ID.String(), IdempotencyKey: fmt.Sprintf("outbound:%s:line:%d", strings.ToLower(strings.TrimSpace(item.Number)), idx+1), ProductID: line.ProductID, WarehouseID: item.WarehouseID, AreaID: item.AreaID, LocationID: item.LocationID, BatchID: line.BatchID, Quantity: line.Quantity, ActorID: in.ActorID})
	}
	results, movementErr := s.inventory.OutboundMany(ctx, movements)
	if movementErr != nil {
		return nil, movementErr
	}
	for idx := range item.Lines {
		item.Lines[idx].LedgerID = results[idx].Ledger.ID.String()
	}
	if err := s.repo.CreateOutbound(ctx, item); err != nil {
		return nil, err
	}
	s.appendSalesAudit(ctx, in.ActorID, "pharma_oa.sales.outbound_create", "pharma_oa_sales_outbound", item.ID.String(), map[string]any{"salesOrderId": order.ID.String(), "customerId": order.CustomerID, "lines": len(item.Lines)})
	return cloneSalesOutbound(item), nil
}

func (s *salesService) ListOutbounds(ctx context.Context) ([]*domainpharma.SalesOutbound, error) {
	rows, err := s.repo.ListOutbounds(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]*domainpharma.SalesOutbound, 0, len(rows))
	for index := range rows {
		out = append(out, cloneSalesOutbound(&rows[index]))
	}
	return out, nil
}

func (s *salesService) GetOutbound(ctx context.Context, id string) (*domainpharma.SalesOutbound, error) {
	item, err := s.repo.GetOutbound(ctx, shared.ID(strings.TrimSpace(id)))
	if err != nil {
		return nil, err
	}
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

func (s *salesService) orderNumberExists(ctx context.Context, number string) (bool, error) {
	items, err := s.repo.ListOrders(ctx, pharmaoarepo.ListFilter{Keyword: strings.TrimSpace(number)})
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
func (s *salesService) outboundNumberExists(ctx context.Context, number string) (bool, error) {
	items, err := s.repo.ListOutbounds(ctx, pharmaoarepo.ListFilter{Keyword: strings.TrimSpace(number)})
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
func (s *salesService) nextAvailableSalesOrderID(ctx context.Context) (shared.ID, error) {
	for {
		id := shared.ID("sales-order-" + strconv.FormatInt(s.orderCounter.Add(1), 10))
		item, err := s.repo.GetOrder(ctx, id)
		if err != nil {
			return "", err
		}
		if item == nil {
			return id, nil
		}
	}
}
func (s *salesService) nextAvailableSalesOutboundID(ctx context.Context) (shared.ID, error) {
	for {
		id := shared.ID("sales-outbound-" + strconv.FormatInt(s.outboundCounter.Add(1), 10))
		item, err := s.repo.GetOutbound(ctx, id)
		if err != nil {
			return "", err
		}
		if item == nil {
			return id, nil
		}
	}
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
