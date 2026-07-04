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

type SupplierService interface {
	Create(ctx context.Context, in SupplierWriteInput) (*domainpharma.Supplier, error)
	List(ctx context.Context, in SupplierListInput) ([]*domainpharma.Supplier, error)
	Update(ctx context.Context, id string, in SupplierWriteInput) (*domainpharma.Supplier, error)
	Disable(ctx context.Context, id string, in SupplierDisableInput) (*domainpharma.Supplier, error)
	QualificationReminders(ctx context.Context, days int) ([]SupplierQualificationReminder, error)
	ValidatePurchaseSupplier(ctx context.Context, id string) (SupplierPurchaseEligibility, error)
}

type SupplierWriteInput struct {
	Code           string
	Name           string
	Rating         int
	Contacts       []domainpharma.SupplierContact
	Qualifications []domainpharma.SupplierQualification
	ActorID        string
}

type SupplierListInput struct {
	Keyword string
	Status  string
	Offset  int
	Limit   int
}

type SupplierDisableInput struct {
	Reason  string
	ActorID string
}

type SupplierQualificationReminder struct {
	SupplierID    string    `json:"supplierId"`
	SupplierCode  string    `json:"supplierCode"`
	SupplierName  string    `json:"supplierName"`
	Qualification string    `json:"qualification"`
	Number        string    `json:"number"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

type SupplierPurchaseEligibility struct {
	SupplierID string `json:"supplierId"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason"`
}

type supplierService struct {
	mu        sync.RWMutex
	items     map[string]*domainpharma.Supplier
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter int64
}

func NewSupplierService(audit auditsvc.Service) SupplierService {
	return &supplierService{
		items: map[string]*domainpharma.Supplier{},
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *supplierService) Create(ctx context.Context, in SupplierWriteInput) (*domainpharma.Supplier, error) {
	entity, err := domainpharma.NewSupplier(s.nextSupplierID(), toSupplierDomainInput(in), s.nowFn())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if existing := s.findSupplierByCodeLocked(entity.Code, ""); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("supplier code already exists")
	}
	s.items[entity.ID.String()] = cloneSupplier(entity)
	s.mu.Unlock()
	s.appendSupplierAudit(ctx, in.ActorID, "pharma_oa.supplier.create", entity.ID.String(), map[string]any{"code": entity.Code})
	return cloneSupplier(entity), nil
}

func (s *supplierService) List(_ context.Context, in SupplierListInput) ([]*domainpharma.Supplier, error) {
	limit := normalizeLimit(in.Limit)
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	status := strings.ToLower(strings.TrimSpace(in.Status))
	s.mu.RLock()
	items := make([]*domainpharma.Supplier, 0, len(s.items))
	for _, item := range s.items {
		if status != "" && string(item.Status) != status {
			continue
		}
		if keyword != "" && !supplierMatches(item, keyword) {
			continue
		}
		items = append(items, cloneSupplier(item))
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool {
		return items[i].Code < items[j].Code
	})
	if offset >= len(items) {
		return []*domainpharma.Supplier{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], nil
}

func (s *supplierService) Update(ctx context.Context, id string, in SupplierWriteInput) (*domainpharma.Supplier, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("supplier not found")
	}
	if existing := s.findSupplierByCodeLocked(in.Code, id); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("supplier code already exists")
	}
	next := cloneSupplier(current)
	if err := next.Update(toSupplierDomainInput(in), s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneSupplier(next)
	s.mu.Unlock()
	s.appendSupplierAudit(ctx, in.ActorID, "pharma_oa.supplier.update", id, map[string]any{"code": next.Code})
	return cloneSupplier(next), nil
}

func (s *supplierService) Disable(ctx context.Context, id string, in SupplierDisableInput) (*domainpharma.Supplier, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("supplier not found")
	}
	next := cloneSupplier(current)
	if err := next.Disable(in.Reason, s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneSupplier(next)
	s.mu.Unlock()
	s.appendSupplierAudit(ctx, in.ActorID, "pharma_oa.supplier.disable", id, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	return cloneSupplier(next), nil
}

func (s *supplierService) QualificationReminders(_ context.Context, days int) ([]SupplierQualificationReminder, error) {
	if days <= 0 {
		days = 30
	}
	deadline := s.nowFn().AddDate(0, 0, days)
	s.mu.RLock()
	out := make([]SupplierQualificationReminder, 0)
	for _, item := range s.items {
		for _, qualification := range item.QualificationExpiringBefore(deadline) {
			out = append(out, SupplierQualificationReminder{
				SupplierID:    item.ID.String(),
				SupplierCode:  item.Code,
				SupplierName:  item.Name,
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

func (s *supplierService) ValidatePurchaseSupplier(_ context.Context, id string) (SupplierPurchaseEligibility, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return SupplierPurchaseEligibility{}, fmt.Errorf("id is required")
	}
	s.mu.RLock()
	item := cloneSupplier(s.items[id])
	s.mu.RUnlock()
	if item == nil {
		return SupplierPurchaseEligibility{}, fmt.Errorf("supplier not found")
	}
	err := item.CanUseForPurchase(s.nowFn())
	if err != nil {
		return SupplierPurchaseEligibility{SupplierID: id, Allowed: false, Reason: err.Error()}, nil
	}
	return SupplierPurchaseEligibility{SupplierID: id, Allowed: true}, nil
}

func (s *supplierService) findSupplierByCodeLocked(code string, exceptID string) *domainpharma.Supplier {
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

func (s *supplierService) nextSupplierID() shared.ID {
	s.idCounter++
	return shared.ID("pharma-supplier-" + strconv.FormatInt(s.idCounter, 10))
}

func (s *supplierService) appendSupplierAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actor := strings.TrimSpace(actorID)
	if actor == "" {
		actor = "system"
	}
	_, _ = s.audit.Append(ctx, actor, action, "pharma_oa_supplier", resourceID, detail)
}

func toSupplierDomainInput(in SupplierWriteInput) domainpharma.SupplierInput {
	return domainpharma.SupplierInput{
		Code:           in.Code,
		Name:           in.Name,
		Rating:         in.Rating,
		Contacts:       in.Contacts,
		Qualifications: in.Qualifications,
	}
}

func supplierMatches(item *domainpharma.Supplier, keyword string) bool {
	if item == nil {
		return false
	}
	return strings.Contains(strings.ToLower(item.ID.String()), keyword) ||
		strings.Contains(strings.ToLower(item.Code), keyword) ||
		strings.Contains(strings.ToLower(item.Name), keyword)
}

func cloneSupplier(item *domainpharma.Supplier) *domainpharma.Supplier {
	if item == nil {
		return nil
	}
	out := *item
	out.Contacts = append([]domainpharma.SupplierContact(nil), item.Contacts...)
	out.Qualifications = make([]domainpharma.SupplierQualification, 0, len(item.Qualifications))
	for _, qualification := range item.Qualifications {
		next := qualification
		next.Attachments = append([]domainpharma.SupplierAttachment(nil), qualification.Attachments...)
		out.Qualifications = append(out.Qualifications, next)
	}
	return &out
}
