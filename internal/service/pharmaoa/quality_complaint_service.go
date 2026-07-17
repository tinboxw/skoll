package pharmaoa

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
)

type QualityComplaintService interface {
	Create(ctx context.Context, in QualityComplaintCreateInput) (*domainpharma.QualityComplaint, error)
	List(ctx context.Context, in QualityComplaintListInput) ([]*domainpharma.QualityComplaint, error)
	ListBatches(ctx context.Context, productID string) ([]domainpharma.StockBatch, error)
	Get(ctx context.Context, id string) (*domainpharma.QualityComplaint, error)
	Resolve(ctx context.Context, id string, in QualityComplaintActionInput) (*domainpharma.QualityComplaint, error)
	Reject(ctx context.Context, id string, in QualityComplaintActionInput) (*domainpharma.QualityComplaint, error)
}

type QualityComplaintCreateInput struct {
	Number, Title, Description, CustomerID, ProductID, BatchID, ReporterID, HandlerID string
	AttachmentIDs                                                                     []string
}

type QualityComplaintListInput struct {
	Keyword string
	Status  domainpharma.QualityComplaintStatus
}

type QualityComplaintActionInput struct {
	ActorID    string
	Conclusion string
}

type qualityComplaintService struct {
	mu        sync.RWMutex
	createMu  sync.Mutex
	actionMu  sync.Mutex
	items     map[string]*domainpharma.QualityComplaint
	customers CustomerService
	products  ProductService
	inventory InventoryService
	workflow  workflowsvc.Service
	files     ContractFileReader
	audit     auditsvc.Service
	nowFn     func() time.Time
	counter   int64
	repo      pharmaoarepo.QualityComplaintRepository
}

func NewQualityComplaintService(customers CustomerService, products ProductService, inventory InventoryService, workflow workflowsvc.Service, files ContractFileReader, audit auditsvc.Service, repositories ...pharmaoarepo.QualityComplaintRepository) QualityComplaintService {
	repo := pharmaoarepo.QualityComplaintRepository(pharmaoarepo.NewMemoryQualityComplaintRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &qualityComplaintService{items: map[string]*domainpharma.QualityComplaint{}, customers: customers, products: products, inventory: inventory, workflow: workflow, files: files, audit: audit, repo: repo, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *qualityComplaintService) Create(ctx context.Context, in QualityComplaintCreateInput) (*domainpharma.QualityComplaint, error) {
	if s == nil || s.customers == nil || s.products == nil || s.inventory == nil || s.workflow == nil || s.files == nil {
		return nil, fmt.Errorf("quality complaint dependencies are required")
	}
	s.createMu.Lock()
	defer s.createMu.Unlock()
	if err := s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	if s.numberExists(in.Number) {
		return nil, fmt.Errorf("quality complaint number already exists")
	}
	customerName, err := s.customerName(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	productName, err := s.productName(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}
	batchNo, err := s.batchNo(ctx, in.BatchID, in.ProductID)
	if err != nil {
		return nil, err
	}
	attachments, err := s.attachments(ctx, in.ReporterID, in.AttachmentIDs)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.counter++
	sequence := s.counter
	s.mu.Unlock()
	id := shared.ID("quality-complaint-" + strconv.FormatInt(sequence, 10))
	workflowID := shared.ID("quality-complaint-workflow-" + strconv.FormatInt(sequence, 10))
	definitionID := shared.ID("quality-complaint-definition-" + strconv.FormatInt(sequence, 10))
	now := s.nowFn()
	item, err := domainpharma.NewQualityComplaint(id, domainpharma.QualityComplaintInput{Number: in.Number, Title: in.Title, Description: in.Description, CustomerID: in.CustomerID, CustomerName: customerName, ProductID: in.ProductID, ProductName: productName, BatchID: in.BatchID, BatchNo: batchNo, ReporterID: in.ReporterID, HandlerID: in.HandlerID, Attachments: attachments, WorkflowInstanceID: workflowID.String()}, now)
	if err != nil {
		return nil, err
	}
	definition, err := s.workflow.CreateDefinition(ctx, workflowsvc.CreateDefinitionInput{ID: definitionID, Key: "pharma.quality_complaint." + strconv.FormatInt(sequence, 10), Name: "Quality Complaint Handling", Version: 1, Now: now, Nodes: []domainworkflow.Node{{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart}, {ID: "handling", Key: "handling", Name: "Complaint Handling", Type: domainworkflow.NodeApproval, Assignees: []shared.ID{shared.ID(strings.TrimSpace(in.HandlerID))}}, {ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd}}, Transitions: []domainworkflow.Transition{{From: "start", To: "handling"}, {From: "handling", To: "end"}}})
	if err != nil {
		return nil, err
	}
	if _, err = s.workflow.PublishDefinition(ctx, definition.ID, now); err != nil {
		return nil, err
	}
	if _, err = s.workflow.Start(ctx, workflowsvc.StartInput{ID: workflowID, DefinitionID: definition.ID, BusinessType: "pharma_oa.quality_complaint", BusinessID: id.String(), Title: "Quality complaint " + item.Number, Starter: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ReporterID))}, Now: now}); err != nil {
		return nil, err
	}
	if err = s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	s.storeCached(item)
	s.appendAudit(ctx, in.ReporterID, "pharma_oa.quality_complaint.create", id.String(), map[string]any{"customerId": in.CustomerID, "productId": in.ProductID, "batchId": in.BatchID, "attachments": len(attachments), "workflowInstanceId": workflowID.String()})
	return cloneQualityComplaint(item), nil
}

func (s *qualityComplaintService) List(ctx context.Context, in QualityComplaintListInput) ([]*domainpharma.QualityComplaint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	s.mu.RLock()
	out := make([]*domainpharma.QualityComplaint, 0, len(s.items))
	for _, item := range s.items {
		if in.Status != "" && item.Status != in.Status {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.Number+" "+item.Title+" "+item.CustomerName+" "+item.ProductName+" "+item.BatchNo), keyword) {
			continue
		}
		out = append(out, cloneQualityComplaint(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.CreatedAt.After(out[j].Meta.CreatedAt) })
	return out, nil
}

func (s *qualityComplaintService) ListBatches(ctx context.Context, productID string) ([]domainpharma.StockBatch, error) {
	if s == nil || s.inventory == nil {
		return nil, fmt.Errorf("quality complaint inventory is required")
	}
	items, err := s.inventory.ListBatches(ctx)
	if err != nil {
		return nil, err
	}
	productID = strings.TrimSpace(productID)
	out := make([]domainpharma.StockBatch, 0, len(items))
	for _, item := range items {
		if productID == "" || item.ProductID == productID {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BatchNo != out[j].BatchNo {
			return out[i].BatchNo < out[j].BatchNo
		}
		return out[i].ID.String() < out[j].ID.String()
	})
	return out, nil
}

func (s *qualityComplaintService) Get(ctx context.Context, id string) (*domainpharma.QualityComplaint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	item := cloneQualityComplaint(s.items[strings.TrimSpace(id)])
	s.mu.RUnlock()
	if item == nil {
		return nil, fmt.Errorf("quality complaint not found")
	}
	return item, nil
}

func (s *qualityComplaintService) Resolve(ctx context.Context, id string, in QualityComplaintActionInput) (*domainpharma.QualityComplaint, error) {
	return s.finish(ctx, id, in, true)
}
func (s *qualityComplaintService) Reject(ctx context.Context, id string, in QualityComplaintActionInput) (*domainpharma.QualityComplaint, error) {
	return s.finish(ctx, id, in, false)
}

func (s *qualityComplaintService) finish(ctx context.Context, id string, in QualityComplaintActionInput, resolve bool) (*domainpharma.QualityComplaint, error) {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	item, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status != domainpharma.QualityComplaintPending {
		return item, nil
	}
	if strings.TrimSpace(in.Conclusion) == "" {
		return nil, fmt.Errorf("quality complaint conclusion is required")
	}
	instance, err := s.workflow.GetInstance(ctx, shared.ID(item.WorkflowInstanceID))
	if err != nil {
		return nil, err
	}
	if len(instance.Tasks) == 0 {
		return nil, fmt.Errorf("quality complaint workflow task not found")
	}
	now := s.nowFn()
	if resolve {
		approved, actionErr := s.workflow.Approve(ctx, workflowsvc.TaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[len(instance.Tasks)-1].ID, Actor: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ActorID))}, Comment: in.Conclusion, Now: now})
		if actionErr != nil {
			return nil, actionErr
		}
		if approved.Status != domainworkflow.InstanceApproved {
			return nil, fmt.Errorf("quality complaint workflow is not approved")
		}
		err = item.Resolve(in.ActorID, in.Conclusion, now)
	} else {
		_, err = s.workflow.Reject(ctx, workflowsvc.TaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[len(instance.Tasks)-1].ID, Actor: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ActorID))}, Comment: in.Conclusion, Now: now})
		if err == nil {
			err = item.Reject(in.ActorID, in.Conclusion, now)
		}
	}
	if err != nil {
		return nil, err
	}
	if err = s.save(ctx, item); err != nil {
		return nil, err
	}
	action := "pharma_oa.quality_complaint.reject"
	if resolve {
		action = "pharma_oa.quality_complaint.resolve"
	}
	s.appendAudit(ctx, in.ActorID, action, item.ID.String(), map[string]any{"conclusion": item.Conclusion, "workflowInstanceId": item.WorkflowInstanceID})
	return cloneQualityComplaint(item), nil
}

func (s *qualityComplaintService) customerName(ctx context.Context, id string) (string, error) {
	const pageSize = 200
	targetID := strings.TrimSpace(id)
	for offset := 0; ; offset += pageSize {
		items, err := s.customers.List(ctx, CustomerListInput{Offset: offset, Limit: pageSize, Scope: CustomerAccessScope{IncludeAll: true}})
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID.String() == targetID {
				return item.Name, nil
			}
		}
		if len(items) < pageSize {
			break
		}
	}
	return "", fmt.Errorf("quality complaint customer not found")
}
func (s *qualityComplaintService) productName(ctx context.Context, id string) (string, error) {
	const pageSize = 200
	targetID := strings.TrimSpace(id)
	for offset := 0; ; offset += pageSize {
		items, err := s.products.List(ctx, ProductListInput{Offset: offset, Limit: pageSize})
		if err != nil {
			return "", err
		}
		for _, item := range items {
			if item.ID.String() == targetID && item.Status == domainpharma.ProductStatusActive {
				return item.Name, nil
			}
		}
		if len(items) < pageSize {
			break
		}
	}
	return "", fmt.Errorf("quality complaint active product not found")
}
func (s *qualityComplaintService) batchNo(ctx context.Context, batchID, productID string) (string, error) {
	items, err := s.inventory.ListBatches(ctx)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if item.ID.String() == strings.TrimSpace(batchID) {
			if item.ProductID != strings.TrimSpace(productID) {
				return "", fmt.Errorf("quality complaint batch does not belong to product")
			}
			return item.BatchNo, nil
		}
	}
	return "", fmt.Errorf("quality complaint stock batch not found")
}
func (s *qualityComplaintService) attachments(ctx context.Context, actor string, ids []string) ([]domainpharma.ContractAttachment, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("quality complaint requires at least one attachment")
	}
	out := make([]domainpharma.ContractAttachment, 0, len(ids))
	seen := map[string]struct{}{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			return nil, fmt.Errorf("quality complaint attachment id is required")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		object, decision, err := s.files.Get(ctx, filesvc.GetInput{FileID: shared.ID(id), SubjectType: domainrbac.SubjectUser, SubjectID: shared.ID(strings.TrimSpace(actor)), ActorName: strings.TrimSpace(actor), Metadata: map[string]any{"module": "pharma_oa", "resource": "quality_complaint"}})
		if err != nil || !decision.Allowed || object == nil || object.Status != domainfile.StatusAvailable {
			return nil, fmt.Errorf("quality complaint attachment %s is not accessible", id)
		}
		seen[id] = struct{}{}
		out = append(out, domainpharma.ContractAttachment{FileID: object.ID.String(), FileName: object.Name, MIME: object.MIME, Size: object.Size})
	}
	return out, nil
}
func (s *qualityComplaintService) numberExists(number string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true
		}
	}
	return false
}
func (s *qualityComplaintService) save(ctx context.Context, item *domainpharma.QualityComplaint) error {
	if err := s.repo.Upsert(ctx, item); err != nil {
		return err
	}
	s.storeCached(item)
	return nil
}
func (s *qualityComplaintService) storeCached(item *domainpharma.QualityComplaint) {
	s.mu.Lock()
	s.items[item.ID.String()] = cloneQualityComplaint(item)
	s.mu.Unlock()
}
func (s *qualityComplaintService) syncFromRepository(ctx context.Context) error {
	items, err := s.repo.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return err
	}
	cached := make(map[string]*domainpharma.QualityComplaint, len(items))
	var counter int64
	for index := range items {
		item := cloneQualityComplaint(&items[index])
		cached[item.ID.String()] = item
		if sequence := sequenceFromID(item.ID.String(), "quality-complaint-"); sequence > counter {
			counter = sequence
		}
	}
	s.mu.Lock()
	s.items, s.counter = cached, counter
	s.mu.Unlock()
	return nil
}
func (s *qualityComplaintService) appendAudit(ctx context.Context, actor, action, id string, detail map[string]any) {
	if s.audit != nil {
		_, _ = s.audit.Append(ctx, normalizeContractActor(actor), action, "pharma_oa_quality_complaint", id, detail)
	}
}
func cloneQualityComplaint(item *domainpharma.QualityComplaint) *domainpharma.QualityComplaint {
	if item == nil {
		return nil
	}
	out := *item
	out.Attachments = append([]domainpharma.ContractAttachment(nil), item.Attachments...)
	if item.ResolvedAt != nil {
		v := *item.ResolvedAt
		out.ResolvedAt = &v
	}
	if item.RejectedAt != nil {
		v := *item.RejectedAt
		out.RejectedAt = &v
	}
	return &out
}
