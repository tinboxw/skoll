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
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type DrugRecallService interface {
	Create(ctx context.Context, in DrugRecallCreateInput) (*domainpharma.DrugRecall, error)
	List(ctx context.Context, in DrugRecallListInput) ([]*domainpharma.DrugRecall, error)
	Get(ctx context.Context, id string) (*domainpharma.DrugRecall, error)
	ListBatches(ctx context.Context) ([]domainpharma.StockBatch, error)
	PreviewScope(ctx context.Context, batchID string) ([]domainpharma.DrugRecallScope, error)
	CompleteTask(ctx context.Context, recallID, taskID string, in DrugRecallTaskCompleteInput) (*domainpharma.DrugRecall, error)
}

type DrugRecallCreateInput struct {
	Number            string
	Title             string
	Reason            string
	BatchID           string
	SourceComplaintID string
	ActorID           string
}

type DrugRecallListInput struct {
	Keyword string
	Status  domainpharma.DrugRecallStatus
}

type DrugRecallTaskCompleteInput struct {
	ActorID string
	Note    string
}

type recallSalesReader interface {
	ListOutbounds(context.Context) ([]*domainpharma.SalesOutbound, error)
}

type recallInventoryReader interface {
	ListBatches(context.Context) ([]domainpharma.StockBatch, error)
}

type recallComplaintReader interface {
	Get(context.Context, string) (*domainpharma.QualityComplaint, error)
}

type drugRecallService struct {
	mu         sync.RWMutex
	createMu   sync.Mutex
	actionMu   sync.Mutex
	items      map[string]*domainpharma.DrugRecall
	sales      recallSalesReader
	inventory  recallInventoryReader
	products   ProductService
	customers  CustomerService
	complaints recallComplaintReader
	audit      auditsvc.Service
	nowFn      func() time.Time
	counter    int64
	repo       pharmaoarepo.DrugRecallRepository
}

func NewDrugRecallService(sales recallSalesReader, inventory recallInventoryReader, products ProductService, customers CustomerService, complaints recallComplaintReader, audit auditsvc.Service, repositories ...pharmaoarepo.DrugRecallRepository) DrugRecallService {
	repo := pharmaoarepo.DrugRecallRepository(pharmaoarepo.NewMemoryDrugRecallRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &drugRecallService{items: map[string]*domainpharma.DrugRecall{}, sales: sales, inventory: inventory, products: products, customers: customers, complaints: complaints, audit: audit, repo: repo, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *drugRecallService) Create(ctx context.Context, in DrugRecallCreateInput) (*domainpharma.DrugRecall, error) {
	if s == nil || s.sales == nil || s.inventory == nil || s.products == nil || s.customers == nil {
		return nil, fmt.Errorf("drug recall dependencies are required")
	}
	batch, err := s.batch(ctx, in.BatchID)
	if err != nil {
		return nil, err
	}
	productName, err := s.productName(ctx, batch.ProductID)
	if err != nil {
		return nil, err
	}
	if err = s.validateComplaint(ctx, in.SourceComplaintID, batch); err != nil {
		return nil, err
	}
	scopes, err := s.previewScope(ctx, batch)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		return nil, fmt.Errorf("drug recall batch has no affected outbound customers")
	}
	s.createMu.Lock()
	defer s.createMu.Unlock()
	if err = s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.numberExistsLocked(in.Number) {
		s.mu.Unlock()
		return nil, fmt.Errorf("drug recall number already exists")
	}
	s.counter++
	id := shared.ID("drug-recall-" + strconv.FormatInt(s.counter, 10))
	s.mu.Unlock()
	item, err := domainpharma.NewDrugRecall(id, domainpharma.DrugRecallInput{Number: in.Number, Title: in.Title, Reason: in.Reason, ProductID: batch.ProductID, ProductName: productName, BatchID: batch.ID.String(), BatchNo: batch.BatchNo, SourceComplaintID: in.SourceComplaintID, InitiatedBy: in.ActorID, Scopes: scopes}, s.nowFn())
	if err != nil {
		return nil, err
	}
	if err = s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	s.storeCached(item)
	s.appendAudit(ctx, in.ActorID, "pharma_oa.drug_recall.create", item.ID.String(), map[string]any{"batchId": item.BatchID, "batchNo": item.BatchNo, "productId": item.ProductID, "sourceComplaintId": item.SourceComplaintID, "affectedCustomers": len(item.Tasks)})
	return cloneDrugRecall(item), nil
}

func (s *drugRecallService) List(ctx context.Context, in DrugRecallListInput) ([]*domainpharma.DrugRecall, error) {
	if err := s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	if in.Status != "" && in.Status != domainpharma.DrugRecallActive && in.Status != domainpharma.DrugRecallCompleted {
		return nil, fmt.Errorf("invalid drug recall status: %s", in.Status)
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	s.mu.RLock()
	out := make([]*domainpharma.DrugRecall, 0, len(s.items))
	for _, item := range s.items {
		if in.Status != "" && item.Status != in.Status {
			continue
		}
		if keyword != "" && !drugRecallMatches(item, keyword) {
			continue
		}
		out = append(out, cloneDrugRecall(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].InitiatedAt.After(out[j].InitiatedAt) })
	return out, nil
}

func (s *drugRecallService) Get(ctx context.Context, id string) (*domainpharma.DrugRecall, error) {
	if err := s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	item := cloneDrugRecall(s.items[strings.TrimSpace(id)])
	s.mu.RUnlock()
	if item == nil {
		return nil, fmt.Errorf("drug recall not found")
	}
	return item, nil
}

func (s *drugRecallService) ListBatches(ctx context.Context) ([]domainpharma.StockBatch, error) {
	if s == nil || s.inventory == nil {
		return nil, fmt.Errorf("inventory service is required")
	}
	items, err := s.inventory.ListBatches(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].BatchNo < items[j].BatchNo })
	return items, nil
}

func (s *drugRecallService) PreviewScope(ctx context.Context, batchID string) ([]domainpharma.DrugRecallScope, error) {
	batch, err := s.batch(ctx, batchID)
	if err != nil {
		return nil, err
	}
	return s.previewScope(ctx, batch)
}

func (s *drugRecallService) CompleteTask(ctx context.Context, recallID, taskID string, in DrugRecallTaskCompleteInput) (*domainpharma.DrugRecall, error) {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	item, err := s.Get(ctx, recallID)
	if err != nil {
		return nil, err
	}
	wasCompleted := item.Status == domainpharma.DrugRecallCompleted
	changed, err := item.CompleteTask(taskID, in.ActorID, in.Note, s.nowFn())
	if err != nil {
		return nil, err
	}
	out := cloneDrugRecall(item)
	if changed {
		if err = s.repo.Upsert(ctx, out); err != nil {
			return nil, err
		}
		s.storeCached(out)
	}
	if changed {
		s.appendAudit(ctx, in.ActorID, "pharma_oa.drug_recall.task_complete", out.ID.String(), map[string]any{"taskId": strings.TrimSpace(taskID), "note": strings.TrimSpace(in.Note), "status": out.Status})
		if !wasCompleted && out.Status == domainpharma.DrugRecallCompleted {
			s.appendAudit(ctx, in.ActorID, "pharma_oa.drug_recall.complete", out.ID.String(), map[string]any{"tasks": len(out.Tasks), "batchId": out.BatchID})
		}
	}
	return out, nil
}

func (s *drugRecallService) previewScope(ctx context.Context, batch domainpharma.StockBatch) ([]domainpharma.DrugRecallScope, error) {
	outbounds, err := s.sales.ListOutbounds(ctx)
	if err != nil {
		return nil, err
	}
	names, err := s.customerNames(ctx)
	if err != nil {
		return nil, err
	}
	grouped := map[string]*domainpharma.DrugRecallScope{}
	for _, outbound := range outbounds {
		if outbound == nil {
			continue
		}
		quantity := 0
		for _, line := range outbound.Lines {
			if strings.TrimSpace(line.BatchID) == batch.ID.String() && strings.TrimSpace(line.ProductID) == batch.ProductID {
				quantity += line.Quantity
			}
		}
		if quantity <= 0 {
			continue
		}
		customerID := strings.TrimSpace(outbound.CustomerID)
		customerName := strings.TrimSpace(names[customerID])
		if customerID == "" || customerName == "" {
			return nil, fmt.Errorf("drug recall outbound customer not found")
		}
		scope := grouped[customerID]
		if scope == nil {
			scope = &domainpharma.DrugRecallScope{CustomerID: customerID, CustomerName: customerName}
			grouped[customerID] = scope
		}
		scope.Quantity += quantity
		scope.OutboundIDs = append(scope.OutboundIDs, outbound.ID.String())
	}
	out := make([]domainpharma.DrugRecallScope, 0, len(grouped))
	for _, scope := range grouped {
		sort.Strings(scope.OutboundIDs)
		out = append(out, *scope)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CustomerName == out[j].CustomerName {
			return out[i].CustomerID < out[j].CustomerID
		}
		return out[i].CustomerName < out[j].CustomerName
	})
	return out, nil
}

func (s *drugRecallService) batch(ctx context.Context, id string) (domainpharma.StockBatch, error) {
	items, err := s.inventory.ListBatches(ctx)
	if err != nil {
		return domainpharma.StockBatch{}, err
	}
	for _, item := range items {
		if item.ID.String() == strings.TrimSpace(id) {
			return item, nil
		}
	}
	return domainpharma.StockBatch{}, fmt.Errorf("drug recall stock batch not found")
}

func (s *drugRecallService) productName(ctx context.Context, id string) (string, error) {
	const pageSize = 200
	for offset := 0; ; offset += pageSize {
		items, err := s.products.List(ctx, ProductListInput{Offset: offset, Limit: pageSize})
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID.String() == strings.TrimSpace(id) {
				return item.Name, nil
			}
		}
		if len(items) < pageSize {
			break
		}
	}
	return "", fmt.Errorf("drug recall product not found")
}

func (s *drugRecallService) customerNames(ctx context.Context) (map[string]string, error) {
	const pageSize = 200
	names := map[string]string{}
	for offset := 0; ; offset += pageSize {
		items, err := s.customers.List(ctx, CustomerListInput{Offset: offset, Limit: pageSize, Scope: CustomerAccessScope{IncludeAll: true}})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			names[item.ID.String()] = item.Name
		}
		if len(items) < pageSize {
			break
		}
	}
	return names, nil
}

func (s *drugRecallService) validateComplaint(ctx context.Context, id string, batch domainpharma.StockBatch) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if s.complaints == nil {
		return fmt.Errorf("quality complaint service is required")
	}
	item, err := s.complaints.Get(ctx, id)
	if err != nil {
		return err
	}
	if item.BatchID != batch.ID.String() || item.ProductID != batch.ProductID {
		return fmt.Errorf("drug recall complaint does not match the affected batch")
	}
	return nil
}

func (s *drugRecallService) numberExistsLocked(number string) bool {
	for _, item := range s.items {
		if strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true
		}
	}
	return false
}

func (s *drugRecallService) storeCached(item *domainpharma.DrugRecall) {
	s.mu.Lock()
	s.items[item.ID.String()] = cloneDrugRecall(item)
	s.mu.Unlock()
}

func (s *drugRecallService) syncFromRepository(ctx context.Context) error {
	items, err := s.repo.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return err
	}
	cached := make(map[string]*domainpharma.DrugRecall, len(items))
	var counter int64
	for index := range items {
		item := cloneDrugRecall(&items[index])
		cached[item.ID.String()] = item
		if sequence := sequenceFromID(item.ID.String(), "drug-recall-"); sequence > counter {
			counter = sequence
		}
	}
	s.mu.Lock()
	s.items, s.counter = cached, counter
	s.mu.Unlock()
	return nil
}

func (s *drugRecallService) appendAudit(ctx context.Context, actor, action, id string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}
	_, _ = s.audit.Append(ctx, actor, action, "pharma_oa_drug_recall", id, detail)
}

func drugRecallMatches(item *domainpharma.DrugRecall, keyword string) bool {
	if item == nil {
		return false
	}
	values := []string{item.ID.String(), item.Number, item.Title, item.Reason, item.ProductID, item.ProductName, item.BatchID, item.BatchNo, item.SourceComplaintID}
	for _, task := range item.Tasks {
		values = append(values, task.CustomerID, task.CustomerName)
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

func cloneDrugRecall(item *domainpharma.DrugRecall) *domainpharma.DrugRecall {
	if item == nil {
		return nil
	}
	out := *item
	out.Tasks = make([]domainpharma.DrugRecallTask, len(item.Tasks))
	for index, task := range item.Tasks {
		out.Tasks[index] = task
		out.Tasks[index].OutboundIDs = append([]string(nil), task.OutboundIDs...)
		if task.CompletedAt != nil {
			value := *task.CompletedAt
			out.Tasks[index].CompletedAt = &value
		}
	}
	if item.CompletedAt != nil {
		value := *item.CompletedAt
		out.CompletedAt = &value
	}
	return &out
}
