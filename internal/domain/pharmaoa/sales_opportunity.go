package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type SalesOpportunityStage string

const (
	SalesOpportunityLead        SalesOpportunityStage = "lead"
	SalesOpportunityQualified   SalesOpportunityStage = "qualified"
	SalesOpportunityProposal    SalesOpportunityStage = "proposal"
	SalesOpportunityNegotiation SalesOpportunityStage = "negotiation"
	SalesOpportunityWon         SalesOpportunityStage = "won"
	SalesOpportunityLost        SalesOpportunityStage = "lost"
)

type SalesOpportunityProduct struct {
	ProductID   string `json:"productId"`
	ProductCode string `json:"productCode"`
	ProductName string `json:"productName"`
}

type SalesOpportunityStageChange struct {
	From      SalesOpportunityStage `json:"from,omitempty"`
	To        SalesOpportunityStage `json:"to"`
	ChangedBy string                `json:"changedBy"`
	ChangedAt time.Time             `json:"changedAt"`
	Note      string                `json:"note,omitempty"`
}

type SalesOpportunity struct {
	ID                  shared.ID                     `json:"id"`
	Title               string                        `json:"title"`
	CustomerID          string                        `json:"customerId"`
	CustomerCode        string                        `json:"customerCode"`
	CustomerName        string                        `json:"customerName"`
	OrganizationID      string                        `json:"organizationId"`
	OwnerID             string                        `json:"ownerId"`
	Products            []SalesOpportunityProduct     `json:"products"`
	ExpectedAmountCents int64                         `json:"expectedAmountCents"`
	EstimatedCloseDate  time.Time                     `json:"estimatedCloseDate"`
	Stage               SalesOpportunityStage         `json:"stage"`
	LostReason          string                        `json:"lostReason,omitempty"`
	StageHistory        []SalesOpportunityStageChange `json:"stageHistory"`
	Meta                shared.AuditMeta              `json:"meta"`
}

type SalesOpportunityInput struct {
	Title               string
	CustomerID          string
	CustomerCode        string
	CustomerName        string
	OrganizationID      string
	OwnerID             string
	Products            []SalesOpportunityProduct
	ExpectedAmountCents int64
	EstimatedCloseDate  time.Time
}

func NewSalesOpportunity(id shared.ID, in SalesOpportunityInput, actorID string, now time.Time) (*SalesOpportunity, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	if err := validateSalesOpportunityInput(in); err != nil {
		return nil, err
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("actorId is required")
	}
	now = normalizeSalesTime(now)
	item := &SalesOpportunity{
		ID:                  id,
		Title:               strings.TrimSpace(in.Title),
		CustomerID:          strings.TrimSpace(in.CustomerID),
		CustomerCode:        strings.TrimSpace(in.CustomerCode),
		CustomerName:        strings.TrimSpace(in.CustomerName),
		OrganizationID:      strings.TrimSpace(in.OrganizationID),
		OwnerID:             strings.TrimSpace(in.OwnerID),
		Products:            normalizeSalesOpportunityProducts(in.Products),
		ExpectedAmountCents: in.ExpectedAmountCents,
		EstimatedCloseDate:  in.EstimatedCloseDate.UTC(),
		Stage:               SalesOpportunityLead,
		StageHistory:        []SalesOpportunityStageChange{{To: SalesOpportunityLead, ChangedBy: actorID, ChangedAt: now}},
	}
	item.Meta.Touch(now)
	return item, nil
}

func (s *SalesOpportunity) Update(in SalesOpportunityInput, now time.Time) error {
	if s == nil {
		return fmt.Errorf("sales opportunity is required")
	}
	if s.IsTerminal() {
		return fmt.Errorf("terminal sales opportunity cannot be updated")
	}
	if err := validateSalesOpportunityInput(in); err != nil {
		return err
	}
	s.Title = strings.TrimSpace(in.Title)
	s.Products = normalizeSalesOpportunityProducts(in.Products)
	s.ExpectedAmountCents = in.ExpectedAmountCents
	s.EstimatedCloseDate = in.EstimatedCloseDate.UTC()
	s.Meta.Touch(normalizeSalesTime(now))
	return nil
}

func (s *SalesOpportunity) Advance(target SalesOpportunityStage, actorID, note string, now time.Time) error {
	if s == nil {
		return fmt.Errorf("sales opportunity is required")
	}
	if s.IsTerminal() {
		return fmt.Errorf("terminal sales opportunity cannot advance")
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return fmt.Errorf("actorId is required")
	}
	note = strings.TrimSpace(note)
	if target == SalesOpportunityLost {
		if note == "" {
			return fmt.Errorf("lost reason is required")
		}
		s.LostReason = note
	} else if target != nextSalesOpportunityStage(s.Stage) {
		return fmt.Errorf("sales opportunity cannot advance from %s to %s", s.Stage, target)
	}
	now = normalizeSalesTime(now)
	previous := s.Stage
	s.Stage = target
	s.StageHistory = append(s.StageHistory, SalesOpportunityStageChange{From: previous, To: target, ChangedBy: actorID, ChangedAt: now, Note: note})
	s.Meta.Touch(now)
	return nil
}

func (s *SalesOpportunity) IsTerminal() bool {
	return s != nil && (s.Stage == SalesOpportunityWon || s.Stage == SalesOpportunityLost)
}

func nextSalesOpportunityStage(stage SalesOpportunityStage) SalesOpportunityStage {
	switch stage {
	case SalesOpportunityLead:
		return SalesOpportunityQualified
	case SalesOpportunityQualified:
		return SalesOpportunityProposal
	case SalesOpportunityProposal:
		return SalesOpportunityNegotiation
	case SalesOpportunityNegotiation:
		return SalesOpportunityWon
	default:
		return ""
	}
}

func validateSalesOpportunityInput(in SalesOpportunityInput) error {
	for field, value := range map[string]string{
		"title": in.Title, "customerId": in.CustomerID, "customerCode": in.CustomerCode,
		"customerName": in.CustomerName, "organizationId": in.OrganizationID, "ownerId": in.OwnerID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if in.ExpectedAmountCents <= 0 {
		return fmt.Errorf("expectedAmountCents must be positive")
	}
	if in.EstimatedCloseDate.IsZero() {
		return fmt.Errorf("estimatedCloseDate is required")
	}
	products := normalizeSalesOpportunityProducts(in.Products)
	if len(products) == 0 {
		return fmt.Errorf("at least one product is required")
	}
	seen := map[string]struct{}{}
	for _, product := range products {
		if product.ProductID == "" || product.ProductCode == "" || product.ProductName == "" {
			return fmt.Errorf("sales opportunity product is incomplete")
		}
		if _, exists := seen[product.ProductID]; exists {
			return fmt.Errorf("duplicate product: %s", product.ProductID)
		}
		seen[product.ProductID] = struct{}{}
	}
	return nil
}

func normalizeSalesOpportunityProducts(items []SalesOpportunityProduct) []SalesOpportunityProduct {
	out := make([]SalesOpportunityProduct, 0, len(items))
	for _, item := range items {
		item.ProductID = strings.TrimSpace(item.ProductID)
		item.ProductCode = strings.TrimSpace(item.ProductCode)
		item.ProductName = strings.TrimSpace(item.ProductName)
		out = append(out, item)
	}
	return out
}
