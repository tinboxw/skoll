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
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
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
	mu         sync.RWMutex
	approvalMu sync.Mutex
	requests   map[string]*domainpharma.PurchaseRequest
	orders     map[string]*domainpharma.PurchaseOrder
	suppliers  SupplierService
	workflow   workflowsvc.Service
	audit      auditsvc.Service
	nowFn      func() time.Time
	counter    int64
}

func NewPurchaseService(suppliers SupplierService, workflow workflowsvc.Service, audit auditsvc.Service) PurchaseService {
	return &purchaseService{
		requests: map[string]*domainpharma.PurchaseRequest{}, orders: map[string]*domainpharma.PurchaseOrder{},
		suppliers: suppliers, workflow: workflow, audit: audit, nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *purchaseService) CreateRequest(ctx context.Context, in PurchaseRequestCreateInput) (*domainpharma.PurchaseRequest, error) {
	if s == nil || s.suppliers == nil || s.workflow == nil {
		return nil, fmt.Errorf("purchase service dependencies are required")
	}
	if err := s.ensureSupplierEligible(ctx, in.SupplierID); err != nil {
		return nil, err
	}
	s.mu.RLock()
	duplicateNumber := s.requestNumberExistsLocked(in.Number)
	s.mu.RUnlock()
	if duplicateNumber {
		return nil, fmt.Errorf("purchase request number already exists")
	}
	now := s.nowFn()
	s.mu.Lock()
	s.counter++
	sequence := s.counter
	s.mu.Unlock()
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
	s.mu.Lock()
	if s.requestNumberExistsLocked(item.Number) {
		s.mu.Unlock()
		return nil, fmt.Errorf("purchase request number already exists")
	}
	s.requests[item.ID.String()] = clonePurchaseRequest(item)
	s.mu.Unlock()
	s.appendPurchaseAudit(ctx, in.RequesterID, "pharma_oa.purchase.create", item.ID.String(), map[string]any{"number": item.Number, "workflowInstanceId": item.WorkflowInstanceID})
	return clonePurchaseRequest(item), nil
}

func (s *purchaseService) ListRequests(context.Context) ([]*domainpharma.PurchaseRequest, error) {
	s.mu.RLock()
	items := make([]*domainpharma.PurchaseRequest, 0, len(s.requests))
	for _, item := range s.requests {
		items = append(items, clonePurchaseRequest(item))
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Number < items[j].Number })
	return items, nil
}

func (s *purchaseService) GetRequest(_ context.Context, id string) (*domainpharma.PurchaseRequest, error) {
	s.mu.RLock()
	item := clonePurchaseRequest(s.requests[strings.TrimSpace(id)])
	s.mu.RUnlock()
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
	s.mu.RLock()
	existing := clonePurchaseOrder(s.orders[request.PurchaseOrderID])
	s.mu.RUnlock()
	if existing != nil {
		return existing, nil
	}
	if err := s.ensureSupplierEligible(ctx, request.SupplierID); err != nil {
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
	s.mu.Lock()
	if current := s.orders[order.ID.String()]; current != nil {
		order = clonePurchaseOrder(current)
	} else {
		s.requests[request.ID.String()] = clonePurchaseRequest(request)
		s.orders[order.ID.String()] = clonePurchaseOrder(order)
	}
	s.mu.Unlock()
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
	s.mu.Lock()
	s.requests[request.ID.String()] = clonePurchaseRequest(request)
	s.mu.Unlock()
	s.appendPurchaseAudit(ctx, in.ActorID, "pharma_oa.purchase.reject", request.ID.String(), map[string]any{"comment": strings.TrimSpace(in.Comment)})
	return clonePurchaseRequest(request), nil
}

func (s *purchaseService) ListOrders(context.Context) ([]*domainpharma.PurchaseOrder, error) {
	s.mu.RLock()
	items := make([]*domainpharma.PurchaseOrder, 0, len(s.orders))
	for _, item := range s.orders {
		items = append(items, clonePurchaseOrder(item))
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Number < items[j].Number })
	return items, nil
}

func (s *purchaseService) GetOrder(_ context.Context, id string) (*domainpharma.PurchaseOrder, error) {
	s.mu.RLock()
	item := clonePurchaseOrder(s.orders[strings.TrimSpace(id)])
	s.mu.RUnlock()
	if item == nil {
		return nil, fmt.Errorf("purchase order not found")
	}
	return item, nil
}

func (s *purchaseService) ensureSupplierEligible(ctx context.Context, supplierID string) error {
	eligibility, err := s.suppliers.ValidatePurchaseSupplier(ctx, supplierID)
	if err != nil {
		return err
	}
	if !eligibility.Allowed {
		return fmt.Errorf("supplier is not eligible for purchase: %s", eligibility.Reason)
	}
	return nil
}

func (s *purchaseService) requestNumberExistsLocked(number string) bool {
	for _, item := range s.requests {
		if strings.EqualFold(item.Number, strings.TrimSpace(number)) {
			return true
		}
	}
	return false
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
