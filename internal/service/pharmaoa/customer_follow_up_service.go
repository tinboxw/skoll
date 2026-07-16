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

type CustomerFollowUpService interface {
	Create(context.Context, CustomerFollowUpCreateInput) (*domainpharma.CustomerFollowUp, error)
	List(context.Context, CustomerFollowUpListInput) ([]*domainpharma.CustomerFollowUp, error)
	Update(context.Context, string, CustomerFollowUpPlanInput) (*domainpharma.CustomerFollowUp, error)
	Complete(context.Context, string, CustomerFollowUpCompleteInput) (*domainpharma.CustomerFollowUp, error)
	Cancel(context.Context, string, CustomerFollowUpCancelInput) (*domainpharma.CustomerFollowUp, error)
}

type CustomerFollowUpAccessScope struct {
	OwnerID        string
	OrganizationID string
	IncludeAll     bool
}

type CustomerFollowUpCreateInput struct {
	CustomerID  string
	ContactName string
	Channel     domainpharma.CustomerFollowUpChannel
	ScheduledAt time.Time
	NextAction  string
	Attachments []domainpharma.CustomerFollowUpAttachment
	ActorID     string
	Scope       CustomerFollowUpAccessScope
}

type CustomerFollowUpPlanInput struct {
	ContactName string
	Channel     domainpharma.CustomerFollowUpChannel
	ScheduledAt time.Time
	NextAction  string
	Attachments []domainpharma.CustomerFollowUpAttachment
	ActorID     string
	Scope       CustomerFollowUpAccessScope
}

type CustomerFollowUpListInput struct {
	Keyword    string
	CustomerID string
	Status     string
	From       time.Time
	To         time.Time
	ActorID    string
	Scope      CustomerFollowUpAccessScope
}

type CustomerFollowUpCompleteInput struct {
	Summary     string
	NextAction  string
	Attachments []domainpharma.CustomerFollowUpAttachment
	ActorID     string
	Scope       CustomerFollowUpAccessScope
}

type CustomerFollowUpCancelInput struct {
	Reason  string
	ActorID string
	Scope   CustomerFollowUpAccessScope
}

type customerFollowUpService struct {
	mu        sync.RWMutex
	items     map[string]*domainpharma.CustomerFollowUp
	customers CustomerService
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter int64
}

func NewCustomerFollowUpService(customers CustomerService, audit auditsvc.Service) CustomerFollowUpService {
	return &customerFollowUpService{items: map[string]*domainpharma.CustomerFollowUp{}, customers: customers, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *customerFollowUpService) Create(ctx context.Context, in CustomerFollowUpCreateInput) (*domainpharma.CustomerFollowUp, error) {
	actorID, scope, err := normalizeFollowUpActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	customer, err := s.accessibleCustomer(ctx, in.CustomerID, scope)
	if err != nil {
		return nil, err
	}
	entity, err := domainpharma.NewCustomerFollowUp(s.nextID(), domainpharma.CustomerFollowUpInput{
		CustomerID: customer.ID.String(), CustomerCode: customer.Code, CustomerName: customer.Name, OrganizationID: customer.OrganizationID, OwnerID: actorID,
		ContactName: in.ContactName, Channel: in.Channel, ScheduledAt: in.ScheduledAt, NextAction: in.NextAction, Attachments: in.Attachments,
	}, s.nowFn())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.items[entity.ID.String()] = cloneCustomerFollowUp(entity)
	s.mu.Unlock()
	s.appendAudit(ctx, actorID, "pharma_oa.customer_follow_up.create", entity.ID.String(), map[string]any{"customerId": entity.CustomerID, "scheduledAt": entity.ScheduledAt})
	return cloneCustomerFollowUp(entity), nil
}

func (s *customerFollowUpService) List(ctx context.Context, in CustomerFollowUpListInput) ([]*domainpharma.CustomerFollowUp, error) {
	actorID, scope, err := normalizeFollowUpActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	status := strings.TrimSpace(in.Status)
	customerID := strings.TrimSpace(in.CustomerID)
	s.mu.RLock()
	out := make([]*domainpharma.CustomerFollowUp, 0, len(s.items))
	for _, item := range s.items {
		if !followUpInScope(item, scope) || (customerID != "" && item.CustomerID != customerID) || (status != "" && string(item.Status) != status) {
			continue
		}
		if !in.From.IsZero() && item.ScheduledAt.Before(in.From) || !in.To.IsZero() && item.ScheduledAt.After(in.To) {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.CustomerCode+" "+item.CustomerName+" "+item.ContactName+" "+item.NextAction), keyword) {
			continue
		}
		out = append(out, cloneCustomerFollowUp(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ScheduledAt.After(out[j].ScheduledAt) })
	s.appendAudit(ctx, actorID, "pharma_oa.customer_follow_up.read_list", "customer-follow-ups", map[string]any{"count": len(out), "customerId": customerID, "status": status})
	return out, nil
}

func (s *customerFollowUpService) Update(ctx context.Context, id string, in CustomerFollowUpPlanInput) (*domainpharma.CustomerFollowUp, error) {
	actorID, scope, err := normalizeFollowUpActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	return s.mutate(ctx, id, actorID, scope, "pharma_oa.customer_follow_up.update", func(item *domainpharma.CustomerFollowUp) error {
		return item.UpdatePlan(domainpharma.CustomerFollowUpInput{ContactName: in.ContactName, Channel: in.Channel, ScheduledAt: in.ScheduledAt, NextAction: in.NextAction, Attachments: in.Attachments}, s.nowFn())
	})
}

func (s *customerFollowUpService) Complete(ctx context.Context, id string, in CustomerFollowUpCompleteInput) (*domainpharma.CustomerFollowUp, error) {
	actorID, scope, err := normalizeFollowUpActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	return s.mutate(ctx, id, actorID, scope, "pharma_oa.customer_follow_up.complete", func(item *domainpharma.CustomerFollowUp) error {
		return item.Complete(in.Summary, in.NextAction, in.Attachments, s.nowFn())
	})
}

func (s *customerFollowUpService) Cancel(ctx context.Context, id string, in CustomerFollowUpCancelInput) (*domainpharma.CustomerFollowUp, error) {
	actorID, scope, err := normalizeFollowUpActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	return s.mutate(ctx, id, actorID, scope, "pharma_oa.customer_follow_up.cancel", func(item *domainpharma.CustomerFollowUp) error { return item.Cancel(in.Reason, s.nowFn()) })
}

func (s *customerFollowUpService) mutate(ctx context.Context, id, actorID string, scope CustomerFollowUpAccessScope, action string, apply func(*domainpharma.CustomerFollowUp) error) (*domainpharma.CustomerFollowUp, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("customer follow-up not found")
	}
	if !followUpInScope(current, scope) {
		s.mu.Unlock()
		return nil, fmt.Errorf("customer follow-up access denied")
	}
	next := cloneCustomerFollowUp(current)
	if err := apply(next); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneCustomerFollowUp(next)
	s.mu.Unlock()
	s.appendAudit(ctx, actorID, action, id, map[string]any{"customerId": next.CustomerID, "status": next.Status})
	return cloneCustomerFollowUp(next), nil
}

func (s *customerFollowUpService) accessibleCustomer(ctx context.Context, id string, scope CustomerFollowUpAccessScope) (*domainpharma.Customer, error) {
	if s.customers == nil {
		return nil, fmt.Errorf("customer service is required")
	}
	id = strings.TrimSpace(id)
	items, err := s.customers.List(ctx, CustomerListInput{Keyword: id, Limit: 100, Scope: CustomerAccessScope{OwnerID: scope.OwnerID, OrganizationID: scope.OrganizationID, IncludeAll: scope.IncludeAll}})
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID.String() == id {
			return item, nil
		}
	}
	return nil, fmt.Errorf("customer access denied")
}

func (s *customerFollowUpService) nextID() shared.ID {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idCounter++
	return shared.ID("customer-follow-up-" + strconv.FormatInt(s.idCounter, 10))
}

func (s *customerFollowUpService) appendAudit(ctx context.Context, actorID, action, id string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	_, _ = s.audit.Append(ctx, actorID, action, "pharma_oa_customer_follow_up", id, detail)
}

func normalizeFollowUpActorAndScope(actorID string, scope CustomerFollowUpAccessScope) (string, CustomerFollowUpAccessScope, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return "", scope, fmt.Errorf("actorId is required")
	}
	scope.OwnerID = strings.TrimSpace(scope.OwnerID)
	scope.OrganizationID = strings.TrimSpace(scope.OrganizationID)
	if !scope.IncludeAll && scope.OwnerID == "" && scope.OrganizationID == "" {
		scope.OwnerID = actorID
	}
	return actorID, scope, nil
}

func followUpInScope(item *domainpharma.CustomerFollowUp, scope CustomerFollowUpAccessScope) bool {
	return item != nil && (scope.IncludeAll || scope.OwnerID != "" && item.OwnerID == scope.OwnerID || scope.OrganizationID != "" && item.OrganizationID == scope.OrganizationID)
}

func cloneCustomerFollowUp(item *domainpharma.CustomerFollowUp) *domainpharma.CustomerFollowUp {
	if item == nil {
		return nil
	}
	out := *item
	out.Attachments = append([]domainpharma.CustomerFollowUpAttachment{}, item.Attachments...)
	if item.CompletedAt != nil {
		value := *item.CompletedAt
		out.CompletedAt = &value
	}
	return &out
}
