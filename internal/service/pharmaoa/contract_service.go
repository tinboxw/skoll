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
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
)

type ContractFileReader interface {
	Get(ctx context.Context, in filesvc.GetInput) (*domainfile.FileObject, filesvc.AccessDecision, error)
}

type ContractService interface {
	Create(ctx context.Context, in ContractCreateInput) (*domainpharma.Contract, error)
	List(ctx context.Context, in ContractListInput) ([]*domainpharma.Contract, error)
	Get(ctx context.Context, id string) (*domainpharma.Contract, error)
	Approve(ctx context.Context, id string, in ContractActionInput) (*domainpharma.Contract, error)
	Reject(ctx context.Context, id string, in ContractActionInput) (*domainpharma.Contract, error)
	ScanExpiry(ctx context.Context, in ContractExpiryScanInput) (*ContractExpiryScanResult, error)
}

type ContractCreateInput struct {
	Number        string
	Title         string
	PartyType     domainpharma.ContractPartyType
	PartyID       string
	OwnerID       string
	ApproverID    string
	Amount        float64
	Currency      string
	EffectiveAt   time.Time
	ExpiresAt     time.Time
	AttachmentIDs []string
}

type ContractListInput struct {
	Keyword   string
	PartyType domainpharma.ContractPartyType
	Status    domainpharma.ContractStatus
}

type ContractActionInput struct {
	ActorID string
	Comment string
}

type ContractExpiryScanInput struct {
	Days    int
	ActorID string
}

type ContractExpiryReminder struct {
	ContractID     string    `json:"contractId"`
	ContractNumber string    `json:"contractNumber"`
	PartyType      string    `json:"partyType"`
	PartyID        string    `json:"partyId"`
	PartyName      string    `json:"partyName"`
	ExpiresAt      time.Time `json:"expiresAt"`
	RecipientID    string    `json:"recipientId"`
	NotificationID string    `json:"notificationId"`
	TargetPath     string    `json:"targetPath"`
}

type ContractExpiryScanResult struct {
	MatchedCount int                      `json:"matchedCount"`
	CreatedCount int                      `json:"createdCount"`
	Reminders    []ContractExpiryReminder `json:"reminders"`
}

type contractService struct {
	mu            sync.RWMutex
	createMu      sync.Mutex
	actionMu      sync.Mutex
	scanMu        sync.Mutex
	items         map[string]*domainpharma.Contract
	reminders     map[string]ContractExpiryReminder
	suppliers     SupplierService
	customers     CustomerService
	workflow      workflowsvc.Service
	files         ContractFileReader
	notifications *notificationsvc.Service
	audit         auditsvc.Service
	nowFn         func() time.Time
	counter       int64
}

func NewContractService(suppliers SupplierService, customers CustomerService, workflow workflowsvc.Service, files ContractFileReader, notifications *notificationsvc.Service, audit auditsvc.Service) ContractService {
	return &contractService{items: map[string]*domainpharma.Contract{}, reminders: map[string]ContractExpiryReminder{}, suppliers: suppliers, customers: customers, workflow: workflow, files: files, notifications: notifications, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *contractService) Create(ctx context.Context, in ContractCreateInput) (*domainpharma.Contract, error) {
	if s == nil || s.suppliers == nil || s.customers == nil || s.workflow == nil || s.files == nil {
		return nil, fmt.Errorf("contract service dependencies are required")
	}
	s.createMu.Lock()
	defer s.createMu.Unlock()
	if s.contractNumberExists(in.Number) {
		return nil, fmt.Errorf("contract number already exists")
	}
	partyName, err := s.resolvePartyName(ctx, in.PartyType, in.PartyID)
	if err != nil {
		return nil, err
	}
	attachments, err := s.resolveAttachments(ctx, in.OwnerID, in.AttachmentIDs)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.counter++
	sequence := s.counter
	s.mu.Unlock()
	contractID := shared.ID("contract-" + strconv.FormatInt(sequence, 10))
	workflowID := shared.ID("contract-workflow-" + strconv.FormatInt(sequence, 10))
	definitionID := shared.ID("contract-definition-" + strconv.FormatInt(sequence, 10))
	item, err := domainpharma.NewContract(contractID, domainpharma.ContractInput{
		Number: in.Number, Title: in.Title, PartyType: in.PartyType, PartyID: in.PartyID, PartyName: partyName,
		OwnerID: in.OwnerID, ApproverID: in.ApproverID, Amount: in.Amount, Currency: in.Currency,
		EffectiveAt: in.EffectiveAt, ExpiresAt: in.ExpiresAt, Attachments: attachments, WorkflowInstanceID: workflowID.String(),
	}, s.nowFn())
	if err != nil {
		return nil, err
	}
	definition, err := s.workflow.CreateDefinition(ctx, workflowsvc.CreateDefinitionInput{ID: definitionID, Key: "pharma.contract." + strconv.FormatInt(sequence, 10), Name: "Contract Approval", Version: 1, Now: s.nowFn(), Nodes: []domainworkflow.Node{{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart}, {ID: "approval", Key: "approval", Name: "Contract Approval", Type: domainworkflow.NodeApproval, Assignees: []shared.ID{shared.ID(strings.TrimSpace(in.ApproverID))}}, {ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd}}, Transitions: []domainworkflow.Transition{{From: "start", To: "approval"}, {From: "approval", To: "end"}}})
	if err != nil {
		return nil, err
	}
	if _, err = s.workflow.PublishDefinition(ctx, definition.ID, s.nowFn()); err != nil {
		return nil, err
	}
	if _, err = s.workflow.Start(ctx, workflowsvc.StartInput{ID: workflowID, DefinitionID: definition.ID, BusinessType: "pharma_oa.contract", BusinessID: contractID.String(), Title: "Contract " + item.Number, Starter: domainworkflow.Actor{ID: shared.ID(item.OwnerID)}, Now: s.nowFn()}); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.items[item.ID.String()] = cloneContract(item)
	s.mu.Unlock()
	s.appendAudit(ctx, item.OwnerID, "pharma_oa.contract.create", item.ID.String(), map[string]any{"number": item.Number, "partyType": item.PartyType, "partyId": item.PartyID, "attachments": len(item.Attachments), "workflowInstanceId": item.WorkflowInstanceID})
	return cloneContract(item), nil
}

func (s *contractService) List(ctx context.Context, in ContractListInput) ([]*domainpharma.Contract, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	s.mu.RLock()
	out := make([]*domainpharma.Contract, 0, len(s.items))
	for _, item := range s.items {
		if in.PartyType != "" && item.PartyType != in.PartyType || in.Status != "" && item.Status != in.Status {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.Number+" "+item.Title+" "+item.PartyName), keyword) {
			continue
		}
		out = append(out, cloneContract(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.CreatedAt.After(out[j].Meta.CreatedAt) })
	return out, nil
}

func (s *contractService) Get(ctx context.Context, id string) (*domainpharma.Contract, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	item := cloneContract(s.items[strings.TrimSpace(id)])
	s.mu.RUnlock()
	if item == nil {
		return nil, fmt.Errorf("contract not found")
	}
	return item, nil
}

func (s *contractService) Approve(ctx context.Context, id string, in ContractActionInput) (*domainpharma.Contract, error) {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	item, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status == domainpharma.ContractActive || item.Status == domainpharma.ContractExpired {
		return item, nil
	}
	instance, err := s.workflow.GetInstance(ctx, shared.ID(item.WorkflowInstanceID))
	if err != nil {
		return nil, err
	}
	if len(instance.Tasks) == 0 {
		return nil, fmt.Errorf("contract approval task not found")
	}
	now := s.nowFn()
	approved, err := s.workflow.Approve(ctx, workflowsvc.TaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[len(instance.Tasks)-1].ID, Actor: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ActorID))}, Comment: in.Comment, Now: now})
	if err != nil {
		return nil, err
	}
	if approved.Status != domainworkflow.InstanceApproved {
		return nil, fmt.Errorf("contract workflow is not approved")
	}
	if err = item.Approve(in.ActorID, now); err != nil {
		return nil, err
	}
	item.MarkExpired(now)
	s.save(item)
	s.appendAudit(ctx, in.ActorID, "pharma_oa.contract.approve", item.ID.String(), map[string]any{"status": item.Status, "workflowInstanceId": item.WorkflowInstanceID, "comment": strings.TrimSpace(in.Comment)})
	return cloneContract(item), nil
}

func (s *contractService) Reject(ctx context.Context, id string, in ContractActionInput) (*domainpharma.Contract, error) {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	item, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Status == domainpharma.ContractRejected {
		return item, nil
	}
	instance, err := s.workflow.GetInstance(ctx, shared.ID(item.WorkflowInstanceID))
	if err != nil {
		return nil, err
	}
	if len(instance.Tasks) == 0 {
		return nil, fmt.Errorf("contract approval task not found")
	}
	now := s.nowFn()
	if _, err = s.workflow.Reject(ctx, workflowsvc.TaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[len(instance.Tasks)-1].ID, Actor: domainworkflow.Actor{ID: shared.ID(strings.TrimSpace(in.ActorID))}, Comment: in.Comment, Now: now}); err != nil {
		return nil, err
	}
	if err = item.Reject(in.ActorID, now); err != nil {
		return nil, err
	}
	s.save(item)
	s.appendAudit(ctx, in.ActorID, "pharma_oa.contract.reject", item.ID.String(), map[string]any{"workflowInstanceId": item.WorkflowInstanceID, "comment": strings.TrimSpace(in.Comment)})
	return cloneContract(item), nil
}

func (s *contractService) ScanExpiry(ctx context.Context, in ContractExpiryScanInput) (*ContractExpiryScanResult, error) {
	if s == nil || s.notifications == nil {
		return nil, fmt.Errorf("contract notification service is required")
	}
	if in.Days <= 0 {
		in.Days = 30
	}
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	now := s.nowFn()
	deadline := now.AddDate(0, 0, in.Days)
	s.mu.RLock()
	candidates := make([]*domainpharma.Contract, 0)
	for _, item := range s.items {
		if (item.Status == domainpharma.ContractActive || item.Status == domainpharma.ContractExpired) && !item.ExpiresAt.After(deadline) {
			candidates = append(candidates, cloneContract(item))
		}
	}
	s.mu.RUnlock()
	result := &ContractExpiryScanResult{MatchedCount: len(candidates), Reminders: make([]ContractExpiryReminder, 0, len(candidates))}
	for _, item := range candidates {
		s.mu.RLock()
		existing, exists := s.reminders[item.ID.String()]
		s.mu.RUnlock()
		if exists {
			result.Reminders = append(result.Reminders, existing)
			continue
		}
		targetPath := "/skoll/pharma-oa/contracts?contractId=" + item.ID.String()
		notification, err := s.notifications.Create(ctx, notificationsvc.CreateInput{ID: "contract-expiry-" + item.ID.String(), Category: notificationsvc.CategoryReminder, Title: "Contract expiry reminder", Body: fmt.Sprintf("Contract %s with %s expires on %s.", item.Number, item.PartyName, item.ExpiresAt.Format("2006-01-02")), ActorID: item.OwnerID, Target: notificationsvc.Target{Type: "pharma_contract", ID: item.ID.String(), Path: targetPath}, DueAt: item.ExpiresAt})
		if err != nil {
			return nil, err
		}
		reminder := ContractExpiryReminder{ContractID: item.ID.String(), ContractNumber: item.Number, PartyType: string(item.PartyType), PartyID: item.PartyID, PartyName: item.PartyName, ExpiresAt: item.ExpiresAt, RecipientID: item.OwnerID, NotificationID: notification.ID, TargetPath: targetPath}
		item.ReminderNotificationID = notification.ID
		item.MarkExpired(now)
		s.mu.Lock()
		s.items[item.ID.String()] = cloneContract(item)
		s.reminders[item.ID.String()] = reminder
		s.mu.Unlock()
		result.CreatedCount++
		result.Reminders = append(result.Reminders, reminder)
		s.appendAudit(ctx, normalizeContractActor(in.ActorID), "pharma_oa.contract.expiry_remind", item.ID.String(), map[string]any{"expiresAt": item.ExpiresAt, "recipientId": item.OwnerID, "notificationId": notification.ID})
	}
	sort.Slice(result.Reminders, func(i, j int) bool { return result.Reminders[i].ExpiresAt.Before(result.Reminders[j].ExpiresAt) })
	s.appendAudit(ctx, normalizeContractActor(in.ActorID), "pharma_oa.contract.expiry_scan", "contracts", map[string]any{"days": in.Days, "matched": result.MatchedCount, "created": result.CreatedCount})
	return result, nil
}

func (s *contractService) resolvePartyName(ctx context.Context, partyType domainpharma.ContractPartyType, partyID string) (string, error) {
	partyID = strings.TrimSpace(partyID)
	switch partyType {
	case domainpharma.ContractPartySupplier:
		for offset := 0; ; offset += 100 {
			items, err := s.suppliers.List(ctx, SupplierListInput{Offset: offset, Limit: 100})
			if err != nil {
				return "", err
			}
			for _, item := range items {
				if item.ID.String() == partyID {
					return item.Name, nil
				}
			}
			if len(items) < 100 {
				break
			}
		}
	case domainpharma.ContractPartyCustomer:
		for offset := 0; ; offset += 100 {
			items, err := s.customers.List(ctx, CustomerListInput{Offset: offset, Limit: 100, Scope: CustomerAccessScope{IncludeAll: true}})
			if err != nil {
				return "", err
			}
			for _, item := range items {
				if item.ID.String() == partyID {
					return item.Name, nil
				}
			}
			if len(items) < 100 {
				break
			}
		}
	default:
		return "", fmt.Errorf("contract party type is invalid")
	}
	return "", fmt.Errorf("contract party not found")
}

func (s *contractService) resolveAttachments(ctx context.Context, actorID string, ids []string) ([]domainpharma.ContractAttachment, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("contract requires at least one attachment")
	}
	out := make([]domainpharma.ContractAttachment, 0, len(ids))
	seen := map[string]struct{}{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			return nil, fmt.Errorf("contract attachment id is required")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		object, decision, err := s.files.Get(ctx, filesvc.GetInput{FileID: shared.ID(id), SubjectType: domainrbac.SubjectUser, SubjectID: shared.ID(strings.TrimSpace(actorID)), ActorName: strings.TrimSpace(actorID), Metadata: map[string]any{"module": "pharma_oa", "resource": "contract"}})
		if err != nil {
			return nil, fmt.Errorf("contract attachment %s is not accessible: %w", id, err)
		}
		if !decision.Allowed || object == nil || object.Status != domainfile.StatusAvailable {
			return nil, fmt.Errorf("contract attachment %s is not available", id)
		}
		seen[id] = struct{}{}
		out = append(out, domainpharma.ContractAttachment{FileID: object.ID.String(), FileName: object.Name, MIME: object.MIME, Size: object.Size})
	}
	return out, nil
}

func (s *contractService) contractNumberExists(number string) bool {
	number = strings.TrimSpace(number)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if strings.EqualFold(item.Number, number) {
			return true
		}
	}
	return false
}

func (s *contractService) save(item *domainpharma.Contract) {
	s.mu.Lock()
	s.items[item.ID.String()] = cloneContract(item)
	s.mu.Unlock()
}

func (s *contractService) appendAudit(ctx context.Context, actor, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	_, _ = s.audit.Append(ctx, normalizeContractActor(actor), action, "pharma_oa_contract", resourceID, detail)
}

func normalizeContractActor(actor string) string {
	if strings.TrimSpace(actor) == "" {
		return "system"
	}
	return strings.TrimSpace(actor)
}

func cloneContract(item *domainpharma.Contract) *domainpharma.Contract {
	if item == nil {
		return nil
	}
	out := *item
	out.Attachments = append([]domainpharma.ContractAttachment(nil), item.Attachments...)
	if item.ApprovedAt != nil {
		value := *item.ApprovedAt
		out.ApprovedAt = &value
	}
	if item.RejectedAt != nil {
		value := *item.RejectedAt
		out.RejectedAt = &value
	}
	return &out
}
