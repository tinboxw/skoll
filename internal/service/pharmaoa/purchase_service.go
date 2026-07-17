package pharmaoa

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
)

type PurchaseService interface {
	CreateRequest(ctx context.Context, in PurchaseRequestCreateInput) (*domainpharma.PurchaseRequest, error)
	ListRequests(ctx context.Context) ([]*domainpharma.PurchaseRequest, error)
	GetRequest(ctx context.Context, id string) (*domainpharma.PurchaseRequest, error)
	ApproveRequest(ctx context.Context, id string, in PurchaseApprovalInput) (*domainpharma.PurchaseOrder, error)
	RejectRequest(ctx context.Context, id string, in PurchaseApprovalInput) (*domainpharma.PurchaseRequest, error)
	ListOrders(ctx context.Context) ([]*domainpharma.PurchaseOrder, error)
	GetOrder(ctx context.Context, id string) (*domainpharma.PurchaseOrder, error)
}

type PurchaseRequestCreateInput struct {
	Number      string
	SupplierID  string
	RequesterID string
	ApproverID  string
	Reason      string
	Lines       []domainpharma.PurchaseLine
}

type PurchaseApprovalInput struct {
	ActorID string
	Comment string
}

type purchaseService struct {
	approvalMu sync.Mutex
	repo       pharmaoarepo.PurchaseRepository
	suppliers  SupplierService
	workflow   workflowsvc.Service
	audit      auditsvc.Service
	nowFn      func() time.Time
	counter    atomic.Int64
}

func NewPurchaseService(suppliers SupplierService, workflow workflowsvc.Service, audit auditsvc.Service, repositories ...pharmaoarepo.PurchaseRepository) PurchaseService {
	repo := pharmaoarepo.PurchaseRepository(pharmaoarepo.NewMemoryPurchaseRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &purchaseService{
		repo: repo, suppliers: suppliers, workflow: workflow, audit: audit, nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *purchaseService) CreateRequest(ctx context.Context, in PurchaseRequestCreateInput) (*domainpharma.PurchaseRequest, error) {
	if s == nil || s.suppliers == nil || s.workflow == nil {
		return nil, fmt.Errorf("purchase service dependencies are required")
	}
	if err := s.ensureSupplierEligible(ctx, in.SupplierID, in.RequesterID, "create_request"); err != nil {
		return nil, err
	}
	duplicateNumber, err := s.requestNumberExists(ctx, in.Number)
	if err != nil {
		return nil, err
	}
	if duplicateNumber {
		return nil, fmt.Errorf("purchase request number already exists")
	}
	now := s.nowFn()
	sequence, err := s.nextAvailablePurchaseSequence(ctx)
	if err != nil {
		return nil, err
	}
	requestID := shared.ID("purchase-request-" + strconv.FormatInt(sequence, 10))
	workflowID := shared.ID("purchase-workflow-" + strconv.FormatInt(sequence, 10))
	definitionID := shared.ID("purchase-definition-" + strconv.FormatInt(sequence, 10))
	definition, err := s.workflow.CreateDefinition(ctx, workflowsvc.CreateDefinitionInput{
		ID: definitionID, Key: "pharma.purchase." + strconv.FormatInt(sequence, 10), Name: "Purchase Request Approval", Version: 1, Now: now,
		Nodes: []domainworkflow.Node{
			{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
			{ID: "approval", Key: "approval", Name: "Purchase Approval", Type: domainworkflow.NodeApproval, Assignees: []shared.ID{shared.ID(strings.TrimSpace(in.ApproverID))}},
			{ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
		},
		Transitions: []domainworkflow.Transition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.workflow.PublishDefinition(ctx, definition.ID, now); err != nil {
		return nil, err
	}
	instance, err := s.workflow.Start(ctx, workflowsvc.StartInput{
		ID: workflowID, DefinitionID: definition.ID, BusinessType: "pharma_oa.purchase_request", BusinessID: requestID.String(),
		Title: "Purchase request " + strings.TrimSpace(in.Number), Starter: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.RequesterID))}, Now: now,
	})
	if err != nil {
		return nil, err
	}
	item, err := domainpharma.NewPurchaseRequest(requestID, domainpharma.PurchaseRequestInput{
		Number: in.Number, SupplierID: in.SupplierID, RequesterID: in.RequesterID, ApproverID: in.ApproverID, Reason: in.Reason, Lines: in.Lines,
	}, instance.ID.String(), now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateRequest(ctx, item); err != nil {
		return nil, err
	}
	s.appendPurchaseAudit(ctx, in.RequesterID, "pharma_oa.purchase.create", item.ID.String(), map[string]any{"number": item.Number, "workflowInstanceId": item.WorkflowInstanceID})
	return clonePurchaseRequest(item), nil
}

func (s *purchaseService) ListRequests(ctx context.Context) ([]*domainpharma.PurchaseRequest, error) {
	rows, err := s.repo.ListRequests(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return nil, err
	}
	items := make([]*domainpharma.PurchaseRequest, 0, len(rows))
	for index := range rows {
		items = append(items, clonePurchaseRequest(&rows[index]))
	}
	return items, nil
}

func (s *purchaseService) GetRequest(ctx context.Context, id string) (*domainpharma.PurchaseRequest, error) {
	item, err := s.repo.GetRequest(ctx, shared.ID(strings.TrimSpace(id)))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("purchase request not found")
	}
	return item, nil
}

func (s *purchaseService) ApproveRequest(ctx context.Context, id string, in PurchaseApprovalInput) (*domainpharma.PurchaseOrder, error) {
	s.approvalMu.Lock()
	defer s.approvalMu.Unlock()
	request, err := s.GetRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetOrder(ctx, shared.ID(request.PurchaseOrderID))
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	if err := s.ensureSupplierEligible(ctx, request.SupplierID, in.ActorID, "approve_request"); err != nil {
		return nil, err
	}
	instance, err := s.workflow.GetInstance(ctx, shared.ID(request.WorkflowInstanceID))
	if err != nil {
		return nil, err
	}
	if len(instance.Tasks) == 0 {
		return nil, fmt.Errorf("purchase approval task not found")
	}
	now := s.nowFn()
	approved, err := s.workflow.Approve(ctx, workflowsvc.TaskActionInput{
		InstanceID: instance.ID, TaskID: instance.Tasks[len(instance.Tasks)-1].ID,
		Actor: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ActorID))}, Comment: in.Comment, Now: now,
	})
	if err != nil {
		return nil, err
	}
	if approved.Status != domainworkflow.InstanceApproved {
		return nil, fmt.Errorf("purchase workflow is not approved")
	}
	orderID := shared.ID("purchase-order-" + strings.TrimPrefix(request.ID.String(), "purchase-request-"))
	if err := request.Approve(orderID.String(), now); err != nil {
		return nil, err
	}
	order, err := domainpharma.NewPurchaseOrder(orderID, "PO-"+strings.TrimPrefix(request.Number, "PR-"), *request, in.ActorID, now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ApproveRequest(ctx, request, order); err != nil {
		if current, getErr := s.repo.GetOrder(ctx, order.ID); getErr == nil && current != nil {
			order = current
		} else {
			return nil, err
		}
	}
	s.appendPurchaseAudit(ctx, in.ActorID, "pharma_oa.purchase.approve", request.ID.String(), map[string]any{"orderId": order.ID.String()})
	s.appendPurchaseAudit(ctx, in.ActorID, "pharma_oa.purchase.order.create", order.ID.String(), map[string]any{"requestId": request.ID.String()})
	return clonePurchaseOrder(order), nil
}

func (s *purchaseService) RejectRequest(ctx context.Context, id string, in PurchaseApprovalInput) (*domainpharma.PurchaseRequest, error) {
	request, err := s.GetRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	instance, err := s.workflow.GetInstance(ctx, shared.ID(request.WorkflowInstanceID))
	if err != nil {
		return nil, err
	}
	if len(instance.Tasks) == 0 {
		return nil, fmt.Errorf("purchase approval task not found")
	}
	now := s.nowFn()
	if _, err := s.workflow.Reject(ctx, workflowsvc.TaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[len(instance.Tasks)-1].ID, Actor: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ActorID))}, Comment: in.Comment, Now: now}); err != nil {
		return nil, err
	}
	if err := request.Reject(now); err != nil {
		return nil, err
	}
	if err := s.repo.UpsertRequest(ctx, request); err != nil {
		return nil, err
	}
	s.appendPurchaseAudit(ctx, in.ActorID, "pharma_oa.purchase.reject", request.ID.String(), map[string]any{"comment": strings.TrimSpace(in.Comment)})
	return clonePurchaseRequest(request), nil
}

func (s *purchaseService) ListOrders(ctx context.Context) ([]*domainpharma.PurchaseOrder, error) {
	rows, err := s.repo.ListOrders(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return nil, err
	}
	items := make([]*domainpharma.PurchaseOrder, 0, len(rows))
	for index := range rows {
		items = append(items, clonePurchaseOrder(&rows[index]))
	}
	return items, nil
}

func (s *purchaseService) GetOrder(ctx context.Context, id string) (*domainpharma.PurchaseOrder, error) {
	item, err := s.repo.GetOrder(ctx, shared.ID(strings.TrimSpace(id)))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("purchase order not found")
	}
	return item, nil
}

func (s *purchaseService) ensureSupplierEligible(ctx context.Context, supplierID, actorID, operation string) error {
	eligibility, err := s.suppliers.ValidatePurchaseSupplier(ctx, supplierID)
	if err != nil {
		s.appendPurchaseAudit(ctx, actorID, "pharma_oa.qualification.block", supplierID, map[string]any{"subjectType": "supplier", "operation": operation, "reason": err.Error()})
		return err
	}
	if !eligibility.Allowed {
		s.appendPurchaseAudit(ctx, actorID, "pharma_oa.qualification.block", supplierID, map[string]any{"subjectType": "supplier", "operation": operation, "reason": eligibility.Reason})
		return fmt.Errorf("supplier is not eligible for purchase: %s", eligibility.Reason)
	}
	return nil
}

func (s *purchaseService) requestNumberExists(ctx context.Context, number string) (bool, error) {
	items, err := s.repo.ListRequests(ctx, pharmaoarepo.ListFilter{Keyword: strings.TrimSpace(number)})
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

func (s *purchaseService) nextAvailablePurchaseSequence(ctx context.Context) (int64, error) {
	for {
		sequence := s.counter.Add(1)
		item, err := s.repo.GetRequest(ctx, shared.ID("purchase-request-"+strconv.FormatInt(sequence, 10)))
		if err != nil {
			return 0, err
		}
		if item == nil {
			return sequence, nil
		}
	}
}

func (s *purchaseService) appendPurchaseAudit(ctx context.Context, actor, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	if strings.TrimSpace(actor) == "" {
		actor = "system"
	}
	_, _ = s.audit.Append(ctx, actor, action, "pharma_oa_purchase", resourceID, detail)
}

func clonePurchaseRequest(item *domainpharma.PurchaseRequest) *domainpharma.PurchaseRequest {
	if item == nil {
		return nil
	}
	out := *item
	out.Lines = append([]domainpharma.PurchaseLine(nil), item.Lines...)
	return &out
}
func clonePurchaseOrder(item *domainpharma.PurchaseOrder) *domainpharma.PurchaseOrder {
	if item == nil {
		return nil
	}
	out := *item
	out.Lines = append([]domainpharma.PurchaseLine(nil), item.Lines...)
	return &out
}
