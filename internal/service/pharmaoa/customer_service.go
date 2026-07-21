package pharmaoa

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type CustomerService interface {
	Create(ctx context.Context, in CustomerWriteInput) (*domainpharma.Customer, error)
	List(ctx context.Context, in CustomerListInput) ([]*domainpharma.Customer, error)
	ListPage(ctx context.Context, in CustomerListInput) (ListPage[*domainpharma.Customer], error)
	Update(ctx context.Context, id string, in CustomerWriteInput) (*domainpharma.Customer, error)
	Disable(ctx context.Context, id string, in CustomerDisableInput) (*domainpharma.Customer, error)
	QualificationReminders(ctx context.Context, in CustomerReminderInput) ([]CustomerQualificationReminder, error)
	ValidateSalesCustomer(ctx context.Context, id string, scope CustomerAccessScope) (CustomerSalesEligibility, error)
}

var ErrCustomerAccessDenied = errors.New("customer access denied")

type CustomerAccessScope struct {
	ActorID         string
	OwnerID         string
	OrganizationID  string
	OrganizationIDs []string
	IncludeAll      bool
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
	repo      pharmaoarepo.CustomerRepository
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter atomic.Int64
}

func NewCustomerService(audit auditsvc.Service, repositories ...pharmaoarepo.CustomerRepository) CustomerService {
	repo := pharmaoarepo.CustomerRepository(pharmaoarepo.NewMemoryCustomerRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &customerService{
		repo:  repo,
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *customerService) Create(ctx context.Context, in CustomerWriteInput) (*domainpharma.Customer, error) {
	normalized := normalizeCustomerWriteInput(in)
	scope := normalizeCustomerAccessScope(normalized.Scope)
	if !scope.IncludeAll && scope.OwnerID == "" && len(scope.OrganizationIDs) == 0 {
		scope.OwnerID = normalized.ActorID
		if scope.OwnerID == "" {
			scope.OwnerID = normalized.OwnerID
		}
	}
	if !customerValuesInScope(normalized.OwnerID, normalized.OrganizationID, scope) {
		return nil, s.customerAccessDenied(ctx, normalized.ActorID, "create", "", normalized.OwnerID, normalized.OrganizationID)
	}
	id, err := s.nextAvailableCustomerID(ctx)
	if err != nil {
		return nil, err
	}
	entity, err := domainpharma.NewCustomer(id, toCustomerDomainInput(normalized), s.nowFn())
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByCode(ctx, entity.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("customer code already exists")
	}
	if err := s.repo.Create(ctx, entity); err != nil {
		return nil, err
	}
	s.appendCustomerAudit(ctx, normalized.ActorID, "pharma_oa.customer.create", entity.ID.String(), map[string]any{"code": entity.Code, "ownerId": entity.OwnerID, "organizationId": entity.OrganizationID})
	return cloneCustomer(entity), nil
}

func (s *customerService) List(ctx context.Context, in CustomerListInput) ([]*domainpharma.Customer, error) {
	page, err := s.ListPage(ctx, in)
	return page.Items, err
}

func (s *customerService) ListPage(ctx context.Context, in CustomerListInput) (ListPage[*domainpharma.Customer], error) {
	scope := normalizeCustomerAccessScope(in.Scope)
	if !scope.IncludeAll && scope.OwnerID == "" && len(scope.OrganizationIDs) == 0 {
		return ListPage[*domainpharma.Customer]{Items: []*domainpharma.Customer{}}, nil
	}
	filter := pharmaoarepo.ListFilter{Keyword: in.Keyword, Status: in.Status, Region: in.Region, Offset: in.Offset, Limit: normalizeLimit(in.Limit)}
	if !scope.IncludeAll {
		filter.OrganizationIDs = sharedIDs(scope.OrganizationIDs)
		filter.OwnerID = shared.ID(scope.OwnerID)
		filter.ScopeAny = true
	}
	page, err := s.repo.ListPage(ctx, filter)
	if err != nil {
		return ListPage[*domainpharma.Customer]{}, err
	}
	items := make([]*domainpharma.Customer, 0, len(page.Items))
	for index := range page.Items {
		items = append(items, cloneCustomer(&page.Items[index]))
	}
	return ListPage[*domainpharma.Customer]{Items: items, Total: page.Total}, nil
}

func (s *customerService) Update(ctx context.Context, id string, in CustomerWriteInput) (*domainpharma.Customer, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	normalized := normalizeCustomerWriteInput(in)
	scope := normalizeCustomerAccessScope(normalized.Scope)
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("customer not found")
	}
	if !customerInScope(current, scope) {
		return nil, s.customerAccessDenied(ctx, normalized.ActorID, "update", id, current.OwnerID, current.OrganizationID)
	}
	if !customerValuesInScope(normalized.OwnerID, normalized.OrganizationID, scope) {
		return nil, s.customerAccessDenied(ctx, normalized.ActorID, "update", id, normalized.OwnerID, normalized.OrganizationID)
	}
	existing, err := s.repo.GetByCode(ctx, normalized.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID.String() != id {
		return nil, fmt.Errorf("customer code already exists")
	}
	next := cloneCustomer(current)
	if err := next.Update(toCustomerDomainInput(normalized), s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.appendCustomerAudit(ctx, normalized.ActorID, "pharma_oa.customer.update", id, map[string]any{"code": next.Code, "ownerId": next.OwnerID, "organizationId": next.OrganizationID})
	return cloneCustomer(next), nil
}

func (s *customerService) Disable(ctx context.Context, id string, in CustomerDisableInput) (*domainpharma.Customer, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	scope := normalizeCustomerAccessScope(in.Scope)
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("customer not found")
	}
	if !customerInScope(current, scope) {
		return nil, s.customerAccessDenied(ctx, in.ActorID, "disable", id, current.OwnerID, current.OrganizationID)
	}
	next := cloneCustomer(current)
	if err := next.Disable(in.Reason, s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.appendCustomerAudit(ctx, in.ActorID, "pharma_oa.customer.disable", id, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	return cloneCustomer(next), nil
}

func (s *customerService) QualificationReminders(ctx context.Context, in CustomerReminderInput) ([]CustomerQualificationReminder, error) {
	days := in.Days
	if days <= 0 {
		days = 30
	}
	deadline := s.nowFn().AddDate(0, 0, days)
	scope := normalizeCustomerAccessScope(in.Scope)
	if !scope.IncludeAll && scope.OwnerID == "" && len(scope.OrganizationIDs) == 0 {
		return []CustomerQualificationReminder{}, nil
	}
	filter := pharmaoarepo.ListFilter{}
	if !scope.IncludeAll {
		filter.OrganizationIDs = sharedIDs(scope.OrganizationIDs)
		filter.OwnerID = shared.ID(scope.OwnerID)
		filter.ScopeAny = true
	}
	items, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]CustomerQualificationReminder, 0)
	for index := range items {
		item := &items[index]
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
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExpiresAt.Before(out[j].ExpiresAt)
	})
	return out, nil
}

func (s *customerService) ValidateSalesCustomer(ctx context.Context, id string, scope CustomerAccessScope) (CustomerSalesEligibility, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CustomerSalesEligibility{}, fmt.Errorf("id is required")
	}
	scope = normalizeCustomerAccessScope(scope)
	item, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return CustomerSalesEligibility{}, err
	}
	if item == nil {
		return CustomerSalesEligibility{}, fmt.Errorf("customer not found")
	}
	if !customerInScope(item, scope) {
		return CustomerSalesEligibility{}, s.customerAccessDenied(ctx, scope.ActorID, "sales", id, item.OwnerID, item.OrganizationID)
	}
	result := CustomerSalesEligibility{CustomerID: item.ID.String(), Allowed: true}
	if err := item.CanUseForSales(s.nowFn()); err != nil {
		result.Allowed = false
		result.Reason = err.Error()
	}
	return result, nil
}

func (s *customerService) nextCustomerID() shared.ID {
	return shared.ID("pharma-customer-" + strconv.FormatInt(s.idCounter.Add(1), 10))
}

func (s *customerService) nextAvailableCustomerID(ctx context.Context) (shared.ID, error) {
	for {
		id := s.nextCustomerID()
		item, err := s.repo.Get(ctx, id)
		if err != nil {
			return "", err
		}
		if item == nil {
			return id, nil
		}
	}
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

func (s *customerService) customerAccessDenied(ctx context.Context, actorID, operation, resourceID, ownerID, organizationID string) error {
	s.appendCustomerAudit(ctx, actorID, "pharma_oa.customer."+operation+".denied", resourceID, map[string]any{
		"result":         "denied",
		"reason":         "data_scope",
		"ownerId":        strings.TrimSpace(ownerID),
		"organizationId": strings.TrimSpace(organizationID),
	})
	return ErrCustomerAccessDenied
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
	scope.ActorID = strings.TrimSpace(scope.ActorID)
	scope.OwnerID = strings.TrimSpace(scope.OwnerID)
	scope.OrganizationID = strings.TrimSpace(scope.OrganizationID)
	scope.OrganizationIDs = compactCustomerOrganizationIDs(append(scope.OrganizationIDs, scope.OrganizationID))
	if len(scope.OrganizationIDs) > 0 {
		scope.OrganizationID = scope.OrganizationIDs[0]
	} else {
		scope.OrganizationID = ""
	}
	return scope
}

func compactCustomerOrganizationIDs(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func sharedIDs(items []string) []shared.ID {
	out := make([]shared.ID, 0, len(items))
	for _, item := range items {
		out = append(out, shared.ID(item))
	}
	return out
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
	return customerValuesInScope(item.OwnerID, item.OrganizationID, scope)
}

func customerValuesInScope(ownerID, organizationID string, scope CustomerAccessScope) bool {
	if scope.IncludeAll {
		return true
	}
	if scope.OwnerID != "" && strings.TrimSpace(ownerID) == scope.OwnerID {
		return true
	}
	for _, allowedOrganizationID := range scope.OrganizationIDs {
		if strings.TrimSpace(organizationID) == allowedOrganizationID {
			return true
		}
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
	out.Contacts = append(make([]domainpharma.CustomerContact, 0, len(item.Contacts)), item.Contacts...)
	out.Qualifications = make([]domainpharma.CustomerQualification, 0, len(item.Qualifications))
	for _, qualification := range item.Qualifications {
		next := qualification
		next.Attachments = append(make([]domainpharma.CustomerAttachment, 0, len(qualification.Attachments)), qualification.Attachments...)
		out.Qualifications = append(out.Qualifications, next)
	}
	return &out
}
