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

type MemoryPurchaseRepository struct {
	mu       sync.RWMutex
	requests map[string]domainpharma.PurchaseRequest
	orders   map[string]domainpharma.PurchaseOrder
}

func NewMemoryPurchaseRepository() *MemoryPurchaseRepository {
	return &MemoryPurchaseRepository{requests: map[string]domainpharma.PurchaseRequest{}, orders: map[string]domainpharma.PurchaseOrder{}}
}
func (r *MemoryPurchaseRepository) CreateRequest(_ context.Context, item *domainpharma.PurchaseRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return fmt.Errorf("purchase request is required")
	}
	if _, ok := r.requests[item.ID.String()]; ok || purchaseRequestNumberExists(r.requests, item.Number, "") {
		return fmt.Errorf("purchase request already exists")
	}
	r.requests[item.ID.String()] = cloneValue(*item)
	return nil
}
func (r *MemoryPurchaseRepository) UpsertRequest(_ context.Context, item *domainpharma.PurchaseRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return fmt.Errorf("purchase request is required")
	}
	if purchaseRequestNumberExists(r.requests, item.Number, item.ID.String()) {
		return fmt.Errorf("purchase request number already exists")
	}
	r.requests[item.ID.String()] = cloneValue(*item)
	return nil
}
func (r *MemoryPurchaseRepository) GetRequest(_ context.Context, id shared.ID) (*domainpharma.PurchaseRequest, error) {
	r.mu.RLock()
	item, ok := r.requests[id.String()]
	r.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	out := cloneValue(item)
	return &out, nil
}
func (r *MemoryPurchaseRepository) ListRequests(_ context.Context, filter ListFilter) ([]domainpharma.PurchaseRequest, error) {
	r.mu.RLock()
	out := make([]domainpharma.PurchaseRequest, 0, len(r.requests))
	for _, item := range r.requests {
		if matchesOrderFilter(filter, item.Number, item.SupplierID, string(item.Status)) {
			out = append(out, cloneValue(item))
		}
	}
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return pageInventoryItems(out, filter), nil
}
func (r *MemoryPurchaseRepository) CreateOrder(_ context.Context, item *domainpharma.PurchaseOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return fmt.Errorf("purchase order is required")
	}
	if _, ok := r.orders[item.ID.String()]; ok || purchaseOrderNumberExists(r.orders, item.Number, item.PurchaseRequestID, "") {
		return fmt.Errorf("purchase order already exists")
	}
	r.orders[item.ID.String()] = cloneValue(*item)
	return nil
}
func (r *MemoryPurchaseRepository) ApproveRequest(_ context.Context, request *domainpharma.PurchaseRequest, order *domainpharma.PurchaseOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if request == nil || order == nil {
		return fmt.Errorf("purchase approval aggregates are required")
	}
	if existing, ok := r.orders[order.ID.String()]; ok {
		if existing.PurchaseRequestID == request.ID.String() {
			return nil
		}
		return fmt.Errorf("purchase order already exists")
	}
	if purchaseOrderNumberExists(r.orders, order.Number, order.PurchaseRequestID, "") {
		return fmt.Errorf("purchase order already exists")
	}
	r.requests[request.ID.String()] = cloneValue(*request)
	r.orders[order.ID.String()] = cloneValue(*order)
	return nil
}
func (r *MemoryPurchaseRepository) UpsertOrder(_ context.Context, item *domainpharma.PurchaseOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return fmt.Errorf("purchase order is required")
	}
	if purchaseOrderNumberExists(r.orders, item.Number, item.PurchaseRequestID, item.ID.String()) {
		return fmt.Errorf("purchase order already exists")
	}
	r.orders[item.ID.String()] = cloneValue(*item)
	return nil
}
func (r *MemoryPurchaseRepository) GetOrder(_ context.Context, id shared.ID) (*domainpharma.PurchaseOrder, error) {
	r.mu.RLock()
	item, ok := r.orders[id.String()]
	r.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	out := cloneValue(item)
	return &out, nil
}
func (r *MemoryPurchaseRepository) ListOrders(_ context.Context, filter ListFilter) ([]domainpharma.PurchaseOrder, error) {
	r.mu.RLock()
	out := make([]domainpharma.PurchaseOrder, 0, len(r.orders))
	for _, item := range r.orders {
		if matchesOrderFilter(filter, item.Number, item.SupplierID, string(item.Status)) {
			out = append(out, cloneValue(item))
		}
	}
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return pageInventoryItems(out, filter), nil
}

type MemoryPurchaseInboundRepository struct {
	*memoryAggregate[domainpharma.PurchaseInbound]
}

func NewMemoryPurchaseInboundRepository() *MemoryPurchaseInboundRepository {
	return &MemoryPurchaseInboundRepository{newMemoryAggregate(func(v domainpharma.PurchaseInbound) string { return v.Number }, func(v domainpharma.PurchaseInbound) string { return string(v.Status) }, func(v domainpharma.PurchaseInbound) string {
		return strings.Join([]string{v.ID.String(), v.Number, v.PurchaseOrderID, v.WarehouseID}, " ")
	})}
}
func (r *MemoryPurchaseInboundRepository) Create(ctx context.Context, item *domainpharma.PurchaseInbound) error {
	if item == nil {
		return fmt.Errorf("purchase inbound is required")
	}
	return r.create(ctx, item.ID, *item)
}
func (r *MemoryPurchaseInboundRepository) Upsert(ctx context.Context, item *domainpharma.PurchaseInbound) error {
	if item == nil {
		return fmt.Errorf("purchase inbound is required")
	}
	return r.upsert(ctx, item.ID, *item)
}
func (r *MemoryPurchaseInboundRepository) Get(ctx context.Context, id shared.ID) (*domainpharma.PurchaseInbound, error) {
	return r.get(ctx, id)
}
func (r *MemoryPurchaseInboundRepository) List(ctx context.Context, filter ListFilter) ([]domainpharma.PurchaseInbound, error) {
	return r.list(ctx, filter)
}

type MemorySalesRepository struct {
	mu        sync.RWMutex
	orders    map[string]domainpharma.SalesOrder
	outbounds map[string]domainpharma.SalesOutbound
}

func NewMemorySalesRepository() *MemorySalesRepository {
	return &MemorySalesRepository{orders: map[string]domainpharma.SalesOrder{}, outbounds: map[string]domainpharma.SalesOutbound{}}
}
func (r *MemorySalesRepository) CreateOrder(_ context.Context, item *domainpharma.SalesOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return fmt.Errorf("sales order is required")
	}
	if _, ok := r.orders[item.ID.String()]; ok || salesNumberExistsMemory(r.orders, item.Number, "") {
		return fmt.Errorf("sales order already exists")
	}
	r.orders[item.ID.String()] = cloneValue(*item)
	return nil
}
func (r *MemorySalesRepository) UpsertOrder(_ context.Context, item *domainpharma.SalesOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return fmt.Errorf("sales order is required")
	}
	if salesNumberExistsMemory(r.orders, item.Number, item.ID.String()) {
		return fmt.Errorf("sales order number already exists")
	}
	r.orders[item.ID.String()] = cloneValue(*item)
	return nil
}
func (r *MemorySalesRepository) GetOrder(_ context.Context, id shared.ID) (*domainpharma.SalesOrder, error) {
	r.mu.RLock()
	item, ok := r.orders[id.String()]
	r.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	out := cloneValue(item)
	return &out, nil
}
func (r *MemorySalesRepository) ListOrders(_ context.Context, filter ListFilter) ([]domainpharma.SalesOrder, error) {
	r.mu.RLock()
	out := make([]domainpharma.SalesOrder, 0, len(r.orders))
	for _, item := range r.orders {
		if matchesOrderFilter(filter, item.Number, item.CustomerID, string(item.Status)) {
			out = append(out, cloneValue(item))
		}
	}
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return pageInventoryItems(out, filter), nil
}
func (r *MemorySalesRepository) CreateOutbound(_ context.Context, item *domainpharma.SalesOutbound) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return fmt.Errorf("sales outbound is required")
	}
	if _, ok := r.outbounds[item.ID.String()]; ok || outboundNumberExistsMemory(r.outbounds, item.Number, "") {
		return fmt.Errorf("sales outbound already exists")
	}
	r.outbounds[item.ID.String()] = cloneValue(*item)
	return nil
}
func (r *MemorySalesRepository) UpsertOutbound(_ context.Context, item *domainpharma.SalesOutbound) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return fmt.Errorf("sales outbound is required")
	}
	if outboundNumberExistsMemory(r.outbounds, item.Number, item.ID.String()) {
		return fmt.Errorf("sales outbound number already exists")
	}
	r.outbounds[item.ID.String()] = cloneValue(*item)
	return nil
}
func (r *MemorySalesRepository) GetOutbound(_ context.Context, id shared.ID) (*domainpharma.SalesOutbound, error) {
	r.mu.RLock()
	item, ok := r.outbounds[id.String()]
	r.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	out := cloneValue(item)
	return &out, nil
}
func (r *MemorySalesRepository) ListOutbounds(_ context.Context, filter ListFilter) ([]domainpharma.SalesOutbound, error) {
	r.mu.RLock()
	out := make([]domainpharma.SalesOutbound, 0, len(r.outbounds))
	for _, item := range r.outbounds {
		if matchesOrderFilter(filter, item.Number, item.CustomerID, string(item.Status)) {
			out = append(out, cloneValue(item))
		}
	}
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return pageInventoryItems(out, filter), nil
}

func cloneValue[T any](item T) T {
	out, err := cloneJSON(item)
	if err != nil {
		return item
	}
	return out
}
func matchesOrderFilter(filter ListFilter, number, party, status string) bool {
	if strings.TrimSpace(filter.Status) != "" && !strings.EqualFold(filter.Status, status) {
		return false
	}
	return matchesInventoryKeyword(filter.Keyword, number, party, status)
}
func purchaseRequestNumberExists(items map[string]domainpharma.PurchaseRequest, number, except string) bool {
	for id, item := range items {
		if id != except && strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true
		}
	}
	return false
}
func purchaseOrderNumberExists(items map[string]domainpharma.PurchaseOrder, number, requestID, except string) bool {
	for id, item := range items {
		if id != except && (strings.EqualFold(item.Number, strings.TrimSpace(number)) || item.PurchaseRequestID == strings.TrimSpace(requestID)) {
			return true
		}
	}
	return false
}
func salesNumberExistsMemory(items map[string]domainpharma.SalesOrder, number, except string) bool {
	for id, item := range items {
		if id != except && strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true
		}
	}
	return false
}
func outboundNumberExistsMemory(items map[string]domainpharma.SalesOutbound, number, except string) bool {
	for id, item := range items {
		if id != except && strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true
		}
	}
	return false
}

type MemoryStocktakeRepository struct {
	*memoryAggregate[domainpharma.StocktakeOrder]
}

func NewMemoryStocktakeRepository() *MemoryStocktakeRepository {
	return &MemoryStocktakeRepository{newMemoryAggregate(func(v domainpharma.StocktakeOrder) string { return v.Number }, func(v domainpharma.StocktakeOrder) string { return string(v.Status) }, func(v domainpharma.StocktakeOrder) string {
		return strings.Join([]string{v.ID.String(), v.Number, v.ProductID, v.WarehouseID, v.BatchID}, " ")
	})}
}
func (r *MemoryStocktakeRepository) Create(ctx context.Context, v *domainpharma.StocktakeOrder) error {
	if v == nil {
		return fmt.Errorf("stocktake is required")
	}
	return r.create(ctx, v.ID, *v)
}
func (r *MemoryStocktakeRepository) Upsert(ctx context.Context, v *domainpharma.StocktakeOrder) error {
	if v == nil {
		return fmt.Errorf("stocktake is required")
	}
	return r.upsert(ctx, v.ID, *v)
}
func (r *MemoryStocktakeRepository) Get(ctx context.Context, id shared.ID) (*domainpharma.StocktakeOrder, error) {
	return r.get(ctx, id)
}
func (r *MemoryStocktakeRepository) List(ctx context.Context, f ListFilter) ([]domainpharma.StocktakeOrder, error) {
	return r.list(ctx, f)
}

type MemoryTransferRepository struct {
	*memoryAggregate[domainpharma.TransferOrder]
}

func NewMemoryTransferRepository() *MemoryTransferRepository {
	return &MemoryTransferRepository{newMemoryAggregate(func(v domainpharma.TransferOrder) string { return v.Number }, func(v domainpharma.TransferOrder) string { return string(v.Status) }, func(v domainpharma.TransferOrder) string {
		return strings.Join([]string{v.ID.String(), v.Number, v.ProductID, v.FromWarehouseID, v.ToWarehouseID, v.BatchID}, " ")
	})}
}
func (r *MemoryTransferRepository) Create(ctx context.Context, v *domainpharma.TransferOrder) error {
	if v == nil {
		return fmt.Errorf("transfer is required")
	}
	return r.create(ctx, v.ID, *v)
}
func (r *MemoryTransferRepository) Upsert(ctx context.Context, v *domainpharma.TransferOrder) error {
	if v == nil {
		return fmt.Errorf("transfer is required")
	}
	return r.upsert(ctx, v.ID, *v)
}
func (r *MemoryTransferRepository) Get(ctx context.Context, id shared.ID) (*domainpharma.TransferOrder, error) {
	return r.get(ctx, id)
}
func (r *MemoryTransferRepository) List(ctx context.Context, f ListFilter) ([]domainpharma.TransferOrder, error) {
	return r.list(ctx, f)
}
