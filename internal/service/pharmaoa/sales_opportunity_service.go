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

type SalesOpportunityService interface {
	Create(context.Context, SalesOpportunityCreateInput) (*domainpharma.SalesOpportunity, error)
	List(context.Context, SalesOpportunityListInput) ([]*domainpharma.SalesOpportunity, error)
	Update(context.Context, string, SalesOpportunityUpdateInput) (*domainpharma.SalesOpportunity, error)
	Advance(context.Context, string, SalesOpportunityAdvanceInput) (*domainpharma.SalesOpportunity, error)
	Statistics(context.Context, SalesOpportunityStatisticsInput) (SalesOpportunityStatistics, error)
}

type SalesOpportunityAccessScope struct {
	OwnerID        string
	OrganizationID string
	IncludeAll     bool
}

type SalesOpportunityCreateInput struct {
	Title               string
	CustomerID          string
	ProductIDs          []string
	ExpectedAmountCents int64
	EstimatedCloseDate  time.Time
	ActorID             string
	Scope               SalesOpportunityAccessScope
}

type SalesOpportunityUpdateInput struct {
	Title               string
	ProductIDs          []string
	ExpectedAmountCents int64
	EstimatedCloseDate  time.Time
	ActorID             string
	Scope               SalesOpportunityAccessScope
}

type SalesOpportunityListInput struct {
	Keyword    string
	CustomerID string
	Stage      string
	ActorID    string
	Scope      SalesOpportunityAccessScope
}

type SalesOpportunityAdvanceInput struct {
	Stage   domainpharma.SalesOpportunityStage
	Note    string
	ActorID string
	Scope   SalesOpportunityAccessScope
}

type SalesOpportunityStatisticsInput struct {
	ActorID string
	Scope   SalesOpportunityAccessScope
}

type SalesOpportunityStageStatistic struct {
	Stage               domainpharma.SalesOpportunityStage `json:"stage"`
	Count               int                                `json:"count"`
	ExpectedAmountCents int64                              `json:"expectedAmountCents"`
}

type SalesOpportunityStatistics struct {
	TotalCount          int                              `json:"totalCount"`
	OpenCount           int                              `json:"openCount"`
	WonCount            int                              `json:"wonCount"`
	LostCount           int                              `json:"lostCount"`
	ExpectedAmountCents int64                            `json:"expectedAmountCents"`
	Stages              []SalesOpportunityStageStatistic `json:"stages"`
}

type salesOpportunityService struct {
	mu        sync.RWMutex
	items     map[string]*domainpharma.SalesOpportunity
	customers CustomerService
	products  ProductService
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter int64
}

func NewSalesOpportunityService(customers CustomerService, products ProductService, audit auditsvc.Service) SalesOpportunityService {
	return &salesOpportunityService{items: map[string]*domainpharma.SalesOpportunity{}, customers: customers, products: products, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *salesOpportunityService) Create(ctx context.Context, in SalesOpportunityCreateInput) (*domainpharma.SalesOpportunity, error) {
	actorID, scope, err := normalizeSalesOpportunityActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	customer, err := s.accessibleSalesOpportunityCustomer(ctx, in.CustomerID, scope)
	if err != nil {
		return nil, err
	}
	products, err := s.activeSalesOpportunityProducts(ctx, in.ProductIDs)
	if err != nil {
		return nil, err
	}
	item, err := domainpharma.NewSalesOpportunity(s.nextID(), domainpharma.SalesOpportunityInput{
		Title: in.Title, CustomerID: customer.ID.String(), CustomerCode: customer.Code, CustomerName: customer.Name,
		OrganizationID: customer.OrganizationID, OwnerID: actorID, Products: products,
		ExpectedAmountCents: in.ExpectedAmountCents, EstimatedCloseDate: in.EstimatedCloseDate,
	}, actorID, s.nowFn())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.items[item.ID.String()] = cloneSalesOpportunity(item)
	s.mu.Unlock()
	s.appendAudit(ctx, actorID, "pharma_oa.sales_opportunity.create", item.ID.String(), map[string]any{"customerId": item.CustomerID, "expectedAmountCents": item.ExpectedAmountCents})
	return cloneSalesOpportunity(item), nil
}

func (s *salesOpportunityService) List(ctx context.Context, in SalesOpportunityListInput) ([]*domainpharma.SalesOpportunity, error) {
	actorID, scope, err := normalizeSalesOpportunityActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	customerID, stage := strings.TrimSpace(in.CustomerID), strings.TrimSpace(in.Stage)
	s.mu.RLock()
	out := make([]*domainpharma.SalesOpportunity, 0, len(s.items))
	for _, item := range s.items {
		if !salesOpportunityInScope(item, scope) || customerID != "" && item.CustomerID != customerID || stage != "" && string(item.Stage) != stage {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.Title+" "+item.CustomerCode+" "+item.CustomerName+" "+salesOpportunityProductText(item.Products)), keyword) {
			continue
		}
		out = append(out, cloneSalesOpportunity(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.UpdatedAt.After(out[j].Meta.UpdatedAt) })
	s.appendAudit(ctx, actorID, "pharma_oa.sales_opportunity.read_list", "sales-opportunities", map[string]any{"count": len(out), "stage": stage})
	return out, nil
}

func (s *salesOpportunityService) Update(ctx context.Context, id string, in SalesOpportunityUpdateInput) (*domainpharma.SalesOpportunity, error) {
	actorID, scope, err := normalizeSalesOpportunityActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	products, err := s.activeSalesOpportunityProducts(ctx, in.ProductIDs)
	if err != nil {
		return nil, err
	}
	return s.mutate(ctx, id, actorID, scope, "pharma_oa.sales_opportunity.update", func(item *domainpharma.SalesOpportunity) error {
		return item.Update(domainpharma.SalesOpportunityInput{Title: in.Title, CustomerID: item.CustomerID, CustomerCode: item.CustomerCode, CustomerName: item.CustomerName, OrganizationID: item.OrganizationID, OwnerID: item.OwnerID, Products: products, ExpectedAmountCents: in.ExpectedAmountCents, EstimatedCloseDate: in.EstimatedCloseDate}, s.nowFn())
	})
}

func (s *salesOpportunityService) Advance(ctx context.Context, id string, in SalesOpportunityAdvanceInput) (*domainpharma.SalesOpportunity, error) {
	actorID, scope, err := normalizeSalesOpportunityActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	return s.mutate(ctx, id, actorID, scope, "pharma_oa.sales_opportunity.advance", func(item *domainpharma.SalesOpportunity) error {
		return item.Advance(in.Stage, actorID, in.Note, s.nowFn())
	})
}

func (s *salesOpportunityService) Statistics(ctx context.Context, in SalesOpportunityStatisticsInput) (SalesOpportunityStatistics, error) {
	actorID, scope, err := normalizeSalesOpportunityActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return SalesOpportunityStatistics{}, err
	}
	stages := []domainpharma.SalesOpportunityStage{domainpharma.SalesOpportunityLead, domainpharma.SalesOpportunityQualified, domainpharma.SalesOpportunityProposal, domainpharma.SalesOpportunityNegotiation, domainpharma.SalesOpportunityWon, domainpharma.SalesOpportunityLost}
	index := make(map[domainpharma.SalesOpportunityStage]int, len(stages))
	stats := SalesOpportunityStatistics{Stages: make([]SalesOpportunityStageStatistic, len(stages))}
	for i, stage := range stages {
		index[stage] = i
		stats.Stages[i].Stage = stage
	}
	s.mu.RLock()
	for _, item := range s.items {
		if !salesOpportunityInScope(item, scope) {
			continue
		}
		stats.TotalCount++
		stats.ExpectedAmountCents += item.ExpectedAmountCents
		bucket := &stats.Stages[index[item.Stage]]
		bucket.Count++
		bucket.ExpectedAmountCents += item.ExpectedAmountCents
		switch item.Stage {
		case domainpharma.SalesOpportunityWon:
			stats.WonCount++
		case domainpharma.SalesOpportunityLost:
			stats.LostCount++
		default:
			stats.OpenCount++
		}
	}
	s.mu.RUnlock()
	s.appendAudit(ctx, actorID, "pharma_oa.sales_opportunity.read_statistics", "sales-opportunity-statistics", map[string]any{"count": stats.TotalCount, "expectedAmountCents": stats.ExpectedAmountCents})
	return stats, nil
}

func (s *salesOpportunityService) mutate(ctx context.Context, id, actorID string, scope SalesOpportunityAccessScope, action string, apply func(*domainpharma.SalesOpportunity) error) (*domainpharma.SalesOpportunity, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("sales opportunity not found")
	}
	if !salesOpportunityInScope(current, scope) {
		s.mu.Unlock()
		return nil, fmt.Errorf("sales opportunity access denied")
	}
	next := cloneSalesOpportunity(current)
	if err := apply(next); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneSalesOpportunity(next)
	s.mu.Unlock()
	s.appendAudit(ctx, actorID, action, id, map[string]any{"customerId": next.CustomerID, "stage": next.Stage, "expectedAmountCents": next.ExpectedAmountCents})
	return cloneSalesOpportunity(next), nil
}

func (s *salesOpportunityService) accessibleSalesOpportunityCustomer(ctx context.Context, id string, scope SalesOpportunityAccessScope) (*domainpharma.Customer, error) {
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
			eligibility, validateErr := s.customers.ValidateSalesCustomer(ctx, id, CustomerAccessScope{OwnerID: scope.OwnerID, OrganizationID: scope.OrganizationID, IncludeAll: scope.IncludeAll})
			if validateErr != nil {
				return nil, validateErr
			}
			if !eligibility.Allowed {
				return nil, fmt.Errorf("customer is not eligible for sales: %s", eligibility.Reason)
			}
			return item, nil
		}
	}
	return nil, fmt.Errorf("customer access denied")
}

func (s *salesOpportunityService) activeSalesOpportunityProducts(ctx context.Context, ids []string) ([]domainpharma.SalesOpportunityProduct, error) {
	if s.products == nil {
		return nil, fmt.Errorf("product service is required")
	}
	out := make([]domainpharma.SalesOpportunityProduct, 0, len(ids))
	seen := map[string]struct{}{}
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return nil, fmt.Errorf("productId is required")
		}
		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("duplicate product: %s", id)
		}
		seen[id] = struct{}{}
		items, err := s.products.List(ctx, ProductListInput{Keyword: id, Status: string(domainpharma.ProductStatusActive), Limit: 100})
		if err != nil {
			return nil, err
		}
		found := false
		for _, item := range items {
			if item.ID.String() == id {
				out = append(out, domainpharma.SalesOpportunityProduct{ProductID: id, ProductCode: item.Code, ProductName: item.Name})
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("active product not found: %s", id)
		}
	}
	return out, nil
}

func (s *salesOpportunityService) nextID() shared.ID {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idCounter++
	return shared.ID("sales-opportunity-" + strconv.FormatInt(s.idCounter, 10))
}

func (s *salesOpportunityService) appendAudit(ctx context.Context, actorID, action, id string, detail map[string]any) {
	if s.audit != nil {
		_, _ = s.audit.Append(ctx, actorID, action, "pharma_oa_sales_opportunity", id, detail)
	}
}

func normalizeSalesOpportunityActorAndScope(actorID string, scope SalesOpportunityAccessScope) (string, SalesOpportunityAccessScope, error) {
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

func salesOpportunityInScope(item *domainpharma.SalesOpportunity, scope SalesOpportunityAccessScope) bool {
	return item != nil && (scope.IncludeAll || scope.OwnerID != "" && item.OwnerID == scope.OwnerID || scope.OrganizationID != "" && item.OrganizationID == scope.OrganizationID)
}

func salesOpportunityProductText(items []domainpharma.SalesOpportunityProduct) string {
	parts := make([]string, 0, len(items)*2)
	for _, item := range items {
		parts = append(parts, item.ProductCode, item.ProductName)
	}
	return strings.Join(parts, " ")
}

func cloneSalesOpportunity(item *domainpharma.SalesOpportunity) *domainpharma.SalesOpportunity {
	if item == nil {
		return nil
	}
	out := *item
	out.Products = append([]domainpharma.SalesOpportunityProduct{}, item.Products...)
	out.StageHistory = append([]domainpharma.SalesOpportunityStageChange{}, item.StageHistory...)
	return &out
}
