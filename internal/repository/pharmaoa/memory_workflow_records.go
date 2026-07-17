package pharmaoa

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type memoryAggregateRepository[T any] struct {
	mu    sync.RWMutex
	items map[string]T
	id    func(*T) string
	match func(T, ListFilter) bool
}

func newMemoryAggregateRepository[T any](id func(*T) string, match func(T, ListFilter) bool) *memoryAggregateRepository[T] {
	return &memoryAggregateRepository[T]{items: map[string]T{}, id: id, match: match}
}

func (r *memoryAggregateRepository[T]) Create(ctx context.Context, item *T) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cloned, err := cloneAggregate(item)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(r.id(cloned))
	if id == "" {
		return fmt.Errorf("aggregate id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[id]; exists {
		return fmt.Errorf("aggregate already exists: %s", id)
	}
	r.items[id] = *cloned
	return nil
}

func (r *memoryAggregateRepository[T]) Upsert(ctx context.Context, item *T) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cloned, err := cloneAggregate(item)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(r.id(cloned))
	if id == "" {
		return fmt.Errorf("aggregate id is required")
	}
	r.mu.Lock()
	r.items[id] = *cloned
	r.mu.Unlock()
	return nil
}

func (r *memoryAggregateRepository[T]) Get(ctx context.Context, id shared.ID) (*T, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	item, exists := r.items[id.String()]
	r.mu.RUnlock()
	if !exists {
		return nil, nil
	}
	return cloneAggregate(&item)
}

func (r *memoryAggregateRepository[T]) List(ctx context.Context, filter ListFilter) ([]T, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	items := make([]T, 0, len(r.items))
	for _, item := range r.items {
		if r.match != nil && !r.match(item, filter) {
			continue
		}
		cloned, err := cloneAggregate(&item)
		if err != nil {
			r.mu.RUnlock()
			return nil, err
		}
		items = append(items, *cloned)
	}
	r.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return r.id(&items[i]) < r.id(&items[j]) })
	return pageAggregateItems(items, filter.Offset, filter.Limit), nil
}

func cloneAggregate[T any](item *T) (*T, error) {
	if item == nil {
		return nil, fmt.Errorf("aggregate is required")
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	var out T
	if err = json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func pageAggregateItems[T any](items []T, offset, limit int) []T {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []T{}
	}
	items = items[offset:]
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}

func containsKeyword(filter ListFilter, values ...string) bool {
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	if keyword == "" {
		return true
	}
	return strings.Contains(strings.ToLower(strings.Join(values, " ")), keyword)
}

type MemoryContractRepository struct {
	*memoryAggregateRepository[domainpharma.Contract]
}
type MemoryQualityComplaintRepository struct {
	*memoryAggregateRepository[domainpharma.QualityComplaint]
}
type MemoryDrugRecallRepository struct {
	*memoryAggregateRepository[domainpharma.DrugRecall]
}
type MemoryCustomerFollowUpRepository struct {
	*memoryAggregateRepository[domainpharma.CustomerFollowUp]
}
type MemorySalesOpportunityRepository struct {
	*memoryAggregateRepository[domainpharma.SalesOpportunity]
}
type MemoryPaymentPlanRepository struct {
	*memoryAggregateRepository[domainpharma.PaymentPlan]
}
type MemoryInvoiceRecordRepository struct {
	*memoryAggregateRepository[domainpharma.InvoiceRecord]
}
type MemoryPaymentReminderJobRepository struct {
	*memoryAggregateRepository[domainpharma.PaymentReminderJob]
}

func NewMemoryContractRepository() *MemoryContractRepository {
	return &MemoryContractRepository{newMemoryAggregateRepository(func(item *domainpharma.Contract) string { return item.ID.String() }, func(item domainpharma.Contract, filter ListFilter) bool {
		return (filter.Status == "" || string(item.Status) == filter.Status) && containsKeyword(filter, item.Number, item.Title, item.PartyName)
	})}
}

func NewMemoryQualityComplaintRepository() *MemoryQualityComplaintRepository {
	return &MemoryQualityComplaintRepository{newMemoryAggregateRepository(func(item *domainpharma.QualityComplaint) string { return item.ID.String() }, func(item domainpharma.QualityComplaint, filter ListFilter) bool {
		return (filter.Status == "" || string(item.Status) == filter.Status) && containsKeyword(filter, item.Number, item.Title, item.CustomerName, item.ProductName, item.BatchNo)
	})}
}

func NewMemoryDrugRecallRepository() *MemoryDrugRecallRepository {
	return &MemoryDrugRecallRepository{newMemoryAggregateRepository(func(item *domainpharma.DrugRecall) string { return item.ID.String() }, func(item domainpharma.DrugRecall, filter ListFilter) bool {
		return (filter.Status == "" || string(item.Status) == filter.Status) && containsKeyword(filter, item.Number, item.Title, item.ProductName, item.BatchNo)
	})}
}

func NewMemoryCustomerFollowUpRepository() *MemoryCustomerFollowUpRepository {
	return &MemoryCustomerFollowUpRepository{newMemoryAggregateRepository(func(item *domainpharma.CustomerFollowUp) string { return item.ID.String() }, func(item domainpharma.CustomerFollowUp, filter ListFilter) bool {
		scope := filter.ScopeAny || filter.OrganizationID.IsZero() || item.OrganizationID == filter.OrganizationID.String() || !filter.OwnerID.IsZero() && item.OwnerID == filter.OwnerID.String()
		return scope && (filter.Status == "" || string(item.Status) == filter.Status) && containsKeyword(filter, item.CustomerCode, item.CustomerName, item.ContactName, item.Summary, item.NextAction)
	})}
}

func NewMemorySalesOpportunityRepository() *MemorySalesOpportunityRepository {
	return &MemorySalesOpportunityRepository{newMemoryAggregateRepository(func(item *domainpharma.SalesOpportunity) string { return item.ID.String() }, func(item domainpharma.SalesOpportunity, filter ListFilter) bool {
		scope := filter.ScopeAny || filter.OrganizationID.IsZero() || item.OrganizationID == filter.OrganizationID.String() || !filter.OwnerID.IsZero() && item.OwnerID == filter.OwnerID.String()
		return scope && (filter.Status == "" || string(item.Stage) == filter.Status) && containsKeyword(filter, item.Title, item.CustomerCode, item.CustomerName)
	})}
}

func NewMemoryPaymentPlanRepository() *MemoryPaymentPlanRepository {
	return &MemoryPaymentPlanRepository{newMemoryAggregateRepository(func(item *domainpharma.PaymentPlan) string { return item.ID.String() }, func(item domainpharma.PaymentPlan, filter ListFilter) bool {
		return (filter.Status == "" || string(item.Status) == filter.Status) && (filter.OwnerID.IsZero() || item.OwnerID == filter.OwnerID.String()) && containsKeyword(filter, item.SalesOrderNumber, item.CustomerID, item.Note)
	})}
}

func NewMemoryInvoiceRecordRepository() *MemoryInvoiceRecordRepository {
	return &MemoryInvoiceRecordRepository{newMemoryAggregateRepository(func(item *domainpharma.InvoiceRecord) string { return item.ID.String() }, func(item domainpharma.InvoiceRecord, filter ListFilter) bool {
		return (filter.Status == "" || string(item.Status) == filter.Status) && (filter.OwnerID.IsZero() || item.OwnerID == filter.OwnerID.String()) && containsKeyword(filter, item.Number, item.SalesOrderNumber, item.CustomerID, item.Note)
	})}
}

func NewMemoryPaymentReminderJobRepository() *MemoryPaymentReminderJobRepository {
	return &MemoryPaymentReminderJobRepository{newMemoryAggregateRepository(func(item *domainpharma.PaymentReminderJob) string { return item.ID.String() }, func(item domainpharma.PaymentReminderJob, filter ListFilter) bool {
		return (filter.Status == "" || string(item.Status) == filter.Status) && (filter.OwnerID.IsZero() || item.OwnerID == filter.OwnerID.String())
	})}
}

type MemoryInventoryAlertRepository struct {
	jobs   *memoryAggregateRepository[domainpharma.InventoryAlertJob]
	alerts *memoryAggregateRepository[domainpharma.InventoryAlert]
}

func NewMemoryInventoryAlertRepository() *MemoryInventoryAlertRepository {
	return &MemoryInventoryAlertRepository{
		jobs: newMemoryAggregateRepository(func(item *domainpharma.InventoryAlertJob) string { return item.ID.String() }, func(item domainpharma.InventoryAlertJob, filter ListFilter) bool {
			return filter.Status == "" || string(item.Status) == filter.Status
		}),
		alerts: newMemoryAggregateRepository(func(item *domainpharma.InventoryAlert) string { return item.ID.String() }, func(item domainpharma.InventoryAlert, filter ListFilter) bool {
			return filter.Status == "" || string(item.Status) == filter.Status
		}),
	}
}

func (r *MemoryInventoryAlertRepository) CreateJob(ctx context.Context, item *domainpharma.InventoryAlertJob) error {
	return r.jobs.Create(ctx, item)
}
func (r *MemoryInventoryAlertRepository) UpsertJob(ctx context.Context, item *domainpharma.InventoryAlertJob) error {
	return r.jobs.Upsert(ctx, item)
}
func (r *MemoryInventoryAlertRepository) GetJob(ctx context.Context, id shared.ID) (*domainpharma.InventoryAlertJob, error) {
	return r.jobs.Get(ctx, id)
}
func (r *MemoryInventoryAlertRepository) ListJobs(ctx context.Context, filter ListFilter) ([]domainpharma.InventoryAlertJob, error) {
	return r.jobs.List(ctx, filter)
}
func (r *MemoryInventoryAlertRepository) UpsertAlert(ctx context.Context, item *domainpharma.InventoryAlert) error {
	return r.alerts.Upsert(ctx, item)
}
func (r *MemoryInventoryAlertRepository) GetAlert(ctx context.Context, id shared.ID) (*domainpharma.InventoryAlert, error) {
	return r.alerts.Get(ctx, id)
}
func (r *MemoryInventoryAlertRepository) ListAlerts(ctx context.Context, filter ListFilter) ([]domainpharma.InventoryAlert, error) {
	return r.alerts.List(ctx, filter)
}

type MemoryReportExportJobRepository struct {
	*memoryAggregateRepository[domainpharma.ReportExportJob]
}

func NewMemoryReportExportJobRepository() *MemoryReportExportJobRepository {
	return &MemoryReportExportJobRepository{newMemoryAggregateRepository(func(item *domainpharma.ReportExportJob) string { return item.ID }, func(item domainpharma.ReportExportJob, filter ListFilter) bool {
		return (filter.Status == "" || string(item.Status) == filter.Status) && (filter.OwnerID.IsZero() || item.OwnerID == filter.OwnerID.String())
	})}
}

func (r *MemoryReportExportJobRepository) Get(ctx context.Context, id string) (*domainpharma.ReportExportJob, error) {
	return r.memoryAggregateRepository.Get(ctx, shared.ID(strings.TrimSpace(id)))
}

func (r *MemoryReportExportJobRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domainpharma.ReportExportJob, error) {
	items, err := r.List(ctx, ListFilter{})
	if err != nil {
		return nil, err
	}
	for index := range items {
		if items[index].IdempotencyKey == strings.TrimSpace(key) {
			return cloneAggregate(&items[index])
		}
	}
	return nil, nil
}
