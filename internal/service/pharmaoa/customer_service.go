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

type CustomerService interface {
	Create(ctx context.Context, in CustomerWriteInput) (*domainpharma.Customer, error)
	List(ctx context.Context, in CustomerListInput) ([]*domainpharma.Customer, error)
	Update(ctx context.Context, id string, in CustomerWriteInput) (*domainpharma.Customer, error)
	Disable(ctx context.Context, id string, in CustomerDisableInput) (*domainpharma.Customer, error)
	QualificationReminders(ctx context.Context, in CustomerReminderInput) ([]CustomerQualificationReminder, error)
	ValidateSalesCustomer(ctx context.Context, id string, scope CustomerAccessScope) (CustomerSalesEligibility, error)
}

type CustomerAccessScope struct {
	OwnerID        string
	OrganizationID string
	IncludeAll     bool
}

type CustomerWriteInput struct {
	Code           string
	Name           string
	Region         string
	OrganizationID string
	OwnerID        string
	Rating         int
	Contacts       []domainpharma.CustomerContact
	Qualifications []domainpharma.CustomerQualification
	ActorID        string
	Scope          CustomerAccessScope
}

type CustomerListInput struct {
	Keyword string
	Status  string
	Region  string
	Offset  int
	Limit   int
	Scope   CustomerAccessScope
}

type CustomerDisableInput struct {
	Reason  string
	ActorID string
	Scope   CustomerAccessScope
}

type CustomerReminderInput struct {
	Days  int
	Scope CustomerAccessScope
}

type CustomerQualificationReminder struct {
	CustomerID    string    `json:"customerId"`
	CustomerCode  string    `json:"customerCode"`
	CustomerName  string    `json:"customerName"`
	Qualification string    `json:"qualification"`
	Number        string    `json:"number"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

type CustomerSalesEligibility struct {
	CustomerID string `json:"customerId"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason,omitempty"`
}

type customerService struct {
	mu        sync.RWMutex
	items     map[string]*domainpharma.Customer
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter int64
}

func NewCustomerService(audit auditsvc.Service) CustomerService {
	return &customerService{
		items: map[string]*domainpharma.Customer{},
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *customerService) Create(ctx context.Context, in CustomerWriteInput) (*domainpharma.Customer, error) {
	normalized := normalizeCustomerWriteInput(in)
	entity, err := domainpharma.NewCustomer(s.nextCustomerID(), toCustomerDomainInput(normalized), s.nowFn())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if existing := s.findCustomerByCodeLocked(entity.Code, ""); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("customer code already exists")
	}
	s.items[entity.ID.String()] = cloneCustomer(entity)
	s.mu.Unlock()
	s.appendCustomerAudit(ctx, normalized.ActorID, "pharma_oa.customer.create", entity.ID.String(), map[string]any{"code": entity.Code, "ownerId": entity.OwnerID, "organizationId": entity.OrganizationID})
	return cloneCustomer(entity), nil
}

func (s *customerService) List(_ context.Context, in CustomerListInput) ([]*domainpharma.Customer, error) {
	limit := normalizeLimit(in.Limit)
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	status := strings.ToLower(strings.TrimSpace(in.Status))
	region := strings.ToLower(strings.TrimSpace(in.Region))
	scope := normalizeCustomerAccessScope(in.Scope)
	s.mu.RLock()
	items := make([]*domainpharma.Customer, 0, len(s.items))
	for _, item := range s.items {
		if !customerInScope(item, scope) {
			continue
		}
		if status != "" && string(item.Status) != status {
			continue
		}
		if region != "" && strings.ToLower(item.Region) != region {
			continue
		}
		if keyword != "" && !customerMatches(item, keyword) {
			continue
		}
		items = append(items, cloneCustomer(item))
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool {
		return items[i].Code < items[j].Code
	})
	if offset >= len(items) {
		return []*domainpharma.Customer{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], nil
}

func (s *customerService) Update(ctx context.Context, id string, in CustomerWriteInput) (*domainpharma.Customer, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	normalized := normalizeCustomerWriteInput(in)
	scope := normalizeCustomerAccessScope(normalized.Scope)
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("customer not found")
	}
	if !customerInScope(current, scope) {
		s.mu.Unlock()
		return nil, fmt.Errorf("customer access denied")
	}
	if existing := s.findCustomerByCodeLocked(normalized.Code, id); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("customer code already exists")
	}
	next := cloneCustomer(current)
	if err := next.Update(toCustomerDomainInput(normalized), s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneCustomer(next)
	s.mu.Unlock()
	s.appendCustomerAudit(ctx, normalized.ActorID, "pharma_oa.customer.update", id, map[string]any{"code": next.Code, "ownerId": next.OwnerID, "organizationId": next.OrganizationID})
	return cloneCustomer(next), nil
}

func (s *customerService) Disable(ctx context.Context, id string, in CustomerDisableInput) (*domainpharma.Customer, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	scope := normalizeCustomerAccessScope(in.Scope)
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("customer not found")
	}
	if !customerInScope(current, scope) {
		s.mu.Unlock()
		return nil, fmt.Errorf("customer access denied")
	}
	next := cloneCustomer(current)
	if err := next.Disable(in.Reason, s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneCustomer(next)
	s.mu.Unlock()
	s.appendCustomerAudit(ctx, in.ActorID, "pharma_oa.customer.disable", id, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	return cloneCustomer(next), nil
}

func (s *customerService) QualificationReminders(_ context.Context, in CustomerReminderInput) ([]CustomerQualificationReminder, error) {
	days := in.Days
	if days <= 0 {
		days = 30
	}
	deadline := s.nowFn().AddDate(0, 0, days)
	scope := normalizeCustomerAccessScope(in.Scope)
	s.mu.RLock()
	out := make([]CustomerQualificationReminder, 0)
	for _, item := range s.items {
		if !customerInScope(item, scope) {
			continue
		}
		for _, qualification := range item.QualificationExpiringBefore(deadline) {
			out = append(out, CustomerQualificationReminder{
				CustomerID:    item.ID.String(),
				CustomerCode:  item.Code,
				CustomerName:  item.Name,
				Qualification: qualification.Name,
				Number:        qualification.Number,
				ExpiresAt:     qualification.ExpiresAt,
			})
		}
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExpiresAt.Before(out[j].ExpiresAt)
	})
	return out, nil
}

func (s *customerService) ValidateSalesCustomer(_ context.Context, id string, scope CustomerAccessScope) (CustomerSalesEligibility, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CustomerSalesEligibility{}, fmt.Errorf("id is required")
	}
	scope = normalizeCustomerAccessScope(scope)
	s.mu.RLock()
	item := cloneCustomer(s.items[id])
	s.mu.RUnlock()
	if item == nil {
		return CustomerSalesEligibility{}, fmt.Errorf("customer not found")
	}
	if !customerInScope(item, scope) {
		return CustomerSalesEligibility{}, fmt.Errorf("customer access denied")
	}
	result := CustomerSalesEligibility{CustomerID: item.ID.String(), Allowed: true}
	if err := item.CanUseForSales(s.nowFn()); err != nil {
		result.Allowed = false
		result.Reason = err.Error()
	}
	return result, nil
}

func (s *customerService) findCustomerByCodeLocked(code string, exceptID string) *domainpharma.Customer {
	code = strings.TrimSpace(code)
	for id, item := range s.items {
		if id == exceptID {
			continue
		}
		if strings.EqualFold(item.Code, code) {
			return item
		}
	}
	return nil
}

func (s *customerService) nextCustomerID() shared.ID {
	s.idCounter++
	return shared.ID("pharma-customer-" + strconv.FormatInt(s.idCounter, 10))
}

func (s *customerService) appendCustomerAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actor := strings.TrimSpace(actorID)
	if actor == "" {
		actor = "system"
	}
	_, _ = s.audit.Append(ctx, actor, action, "pharma_oa_customer", resourceID, detail)
}

func normalizeCustomerWriteInput(in CustomerWriteInput) CustomerWriteInput {
	in.ActorID = strings.TrimSpace(in.ActorID)
	in.OwnerID = strings.TrimSpace(in.OwnerID)
	in.OrganizationID = strings.TrimSpace(in.OrganizationID)
	if in.OwnerID == "" {
		in.OwnerID = in.ActorID
	}
	if in.OrganizationID == "" {
		in.OrganizationID = "default"
	}
	return in
}

func normalizeCustomerAccessScope(scope CustomerAccessScope) CustomerAccessScope {
	scope.OwnerID = strings.TrimSpace(scope.OwnerID)
	scope.OrganizationID = strings.TrimSpace(scope.OrganizationID)
	return scope
}

func toCustomerDomainInput(in CustomerWriteInput) domainpharma.CustomerInput {
	return domainpharma.CustomerInput{
		Code:           in.Code,
		Name:           in.Name,
		Region:         in.Region,
		OrganizationID: in.OrganizationID,
		OwnerID:        in.OwnerID,
		Rating:         in.Rating,
		Contacts:       in.Contacts,
		Qualifications: in.Qualifications,
	}
}

func customerInScope(item *domainpharma.Customer, scope CustomerAccessScope) bool {
	if item == nil {
		return false
	}
	if scope.IncludeAll {
		return true
	}
	if scope.OwnerID != "" && item.OwnerID == scope.OwnerID {
		return true
	}
	if scope.OrganizationID != "" && item.OrganizationID == scope.OrganizationID {
		return true
	}
	return false
}

func customerMatches(item *domainpharma.Customer, keyword string) bool {
	if item == nil {
		return false
	}
	return strings.Contains(strings.ToLower(item.ID.String()), keyword) ||
		strings.Contains(strings.ToLower(item.Code), keyword) ||
		strings.Contains(strings.ToLower(item.Name), keyword) ||
		strings.Contains(strings.ToLower(item.Region), keyword) ||
		strings.Contains(strings.ToLower(item.OwnerID), keyword) ||
		strings.Contains(strings.ToLower(item.OrganizationID), keyword)
}

func cloneCustomer(item *domainpharma.Customer) *domainpharma.Customer {
	if item == nil {
		return nil
	}
	out := *item
	out.Contacts = append([]domainpharma.CustomerContact(nil), item.Contacts...)
	out.Qualifications = make([]domainpharma.CustomerQualification, 0, len(item.Qualifications))
	for _, qualification := range item.Qualifications {
		next := qualification
		next.Attachments = append([]domainpharma.CustomerAttachment(nil), qualification.Attachments...)
		out.Qualifications = append(out.Qualifications, next)
	}
	return &out
}
