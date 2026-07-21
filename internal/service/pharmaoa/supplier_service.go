package pharmaoa

import (
	"context"
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

type SupplierService interface {
	Create(ctx context.Context, in SupplierWriteInput) (*domainpharma.Supplier, error)
	List(ctx context.Context, in SupplierListInput) ([]*domainpharma.Supplier, error)
	ListPage(ctx context.Context, in SupplierListInput) (ListPage[*domainpharma.Supplier], error)
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
	repo      pharmaoarepo.SupplierRepository
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter atomic.Int64
}

func NewSupplierService(audit auditsvc.Service, repositories ...pharmaoarepo.SupplierRepository) SupplierService {
	repo := pharmaoarepo.SupplierRepository(pharmaoarepo.NewMemorySupplierRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &supplierService{
		repo:  repo,
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *supplierService) Create(ctx context.Context, in SupplierWriteInput) (*domainpharma.Supplier, error) {
	id, err := s.nextAvailableSupplierID(ctx)
	if err != nil {
		return nil, err
	}
	entity, err := domainpharma.NewSupplier(id, toSupplierDomainInput(in), s.nowFn())
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByCode(ctx, entity.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("supplier code already exists")
	}
	if err := s.repo.Create(ctx, entity); err != nil {
		return nil, err
	}
	s.appendSupplierAudit(ctx, in.ActorID, "pharma_oa.supplier.create", entity.ID.String(), map[string]any{"code": entity.Code})
	return cloneSupplier(entity), nil
}

func (s *supplierService) List(ctx context.Context, in SupplierListInput) ([]*domainpharma.Supplier, error) {
	page, err := s.ListPage(ctx, in)
	return page.Items, err
}

func (s *supplierService) ListPage(ctx context.Context, in SupplierListInput) (ListPage[*domainpharma.Supplier], error) {
	page, err := s.repo.ListPage(ctx, pharmaoarepo.ListFilter{Keyword: in.Keyword, Status: in.Status, Offset: in.Offset, Limit: normalizeLimit(in.Limit)})
	if err != nil {
		return ListPage[*domainpharma.Supplier]{}, err
	}
	items := make([]*domainpharma.Supplier, 0, len(page.Items))
	for index := range page.Items {
		items = append(items, cloneSupplier(&page.Items[index]))
	}
	return ListPage[*domainpharma.Supplier]{Items: items, Total: page.Total}, nil
}

func (s *supplierService) Update(ctx context.Context, id string, in SupplierWriteInput) (*domainpharma.Supplier, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("supplier not found")
	}
	existing, err := s.repo.GetByCode(ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID.String() != id {
		return nil, fmt.Errorf("supplier code already exists")
	}
	next := cloneSupplier(current)
	if err := next.Update(toSupplierDomainInput(in), s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.appendSupplierAudit(ctx, in.ActorID, "pharma_oa.supplier.update", id, map[string]any{"code": next.Code})
	return cloneSupplier(next), nil
}

func (s *supplierService) Disable(ctx context.Context, id string, in SupplierDisableInput) (*domainpharma.Supplier, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("supplier not found")
	}
	next := cloneSupplier(current)
	if err := next.Disable(in.Reason, s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.appendSupplierAudit(ctx, in.ActorID, "pharma_oa.supplier.disable", id, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	return cloneSupplier(next), nil
}

func (s *supplierService) QualificationReminders(ctx context.Context, days int) ([]SupplierQualificationReminder, error) {
	if days <= 0 {
		days = 30
	}
	deadline := s.nowFn().AddDate(0, 0, days)
	items, err := s.repo.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]SupplierQualificationReminder, 0)
	for index := range items {
		item := &items[index]
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
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExpiresAt.Before(out[j].ExpiresAt)
	})
	return out, nil
}

func (s *supplierService) ValidatePurchaseSupplier(ctx context.Context, id string) (SupplierPurchaseEligibility, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return SupplierPurchaseEligibility{}, fmt.Errorf("id is required")
	}
	item, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return SupplierPurchaseEligibility{}, err
	}
	if item == nil {
		return SupplierPurchaseEligibility{}, fmt.Errorf("supplier not found")
	}
	err = item.CanUseForPurchase(s.nowFn())
	if err != nil {
		return SupplierPurchaseEligibility{SupplierID: id, Allowed: false, Reason: err.Error()}, nil
	}
	return SupplierPurchaseEligibility{SupplierID: id, Allowed: true}, nil
}

func (s *supplierService) nextSupplierID() shared.ID {
	return shared.ID("pharma-supplier-" + strconv.FormatInt(s.idCounter.Add(1), 10))
}

func (s *supplierService) nextAvailableSupplierID(ctx context.Context) (shared.ID, error) {
	for {
		id := s.nextSupplierID()
		item, err := s.repo.Get(ctx, id)
		if err != nil {
			return "", err
		}
		if item == nil {
			return id, nil
		}
	}
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
