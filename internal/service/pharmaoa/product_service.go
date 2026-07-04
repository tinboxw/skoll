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

type ProductService interface {
	Create(ctx context.Context, in ProductWriteInput) (*domainpharma.Product, error)
	List(ctx context.Context, in ProductListInput) ([]*domainpharma.Product, error)
	Update(ctx context.Context, id string, in ProductWriteInput) (*domainpharma.Product, error)
	Disable(ctx context.Context, id string, in ProductDisableInput) (*domainpharma.Product, error)
	Import(ctx context.Context, rows []ProductWriteInput) (ProductImportResult, error)
}

type ProductWriteInput struct {
	Code           string
	Name           string
	Spec           string
	DosageForm     string
	Manufacturer   string
	ApprovalNumber string
	Temperature    domainpharma.ProductTemperature
	ActorID        string
}

type ProductListInput struct {
	Keyword string
	Status  string
	Offset  int
	Limit   int
}

type ProductDisableInput struct {
	Reason  string
	ActorID string
}

type ProductImportResult struct {
	Created int                 `json:"created"`
	Failed  []ProductImportFail `json:"failed"`
}

type ProductImportFail struct {
	Row     int    `json:"row"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type productService struct {
	mu        sync.RWMutex
	items     map[string]*domainpharma.Product
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter int64
}

func NewProductService(audit auditsvc.Service) ProductService {
	return &productService{
		items: map[string]*domainpharma.Product{},
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *productService) Create(ctx context.Context, in ProductWriteInput) (*domainpharma.Product, error) {
	entity, err := domainpharma.NewProduct(s.nextProductID(), toProductDomainInput(in), s.nowFn())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if existing := s.findProductByCodeLocked(entity.Code, ""); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("product code already exists")
	}
	s.items[entity.ID.String()] = cloneProduct(entity)
	s.mu.Unlock()
	s.appendProductAudit(ctx, in.ActorID, "pharma_oa.product.create", entity.ID.String(), map[string]any{"code": entity.Code})
	return cloneProduct(entity), nil
}

func (s *productService) List(_ context.Context, in ProductListInput) ([]*domainpharma.Product, error) {
	limit := normalizeLimit(in.Limit)
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	status := strings.ToLower(strings.TrimSpace(in.Status))
	s.mu.RLock()
	items := make([]*domainpharma.Product, 0, len(s.items))
	for _, item := range s.items {
		if status != "" && string(item.Status) != status {
			continue
		}
		if keyword != "" && !productMatches(item, keyword) {
			continue
		}
		items = append(items, cloneProduct(item))
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool {
		return items[i].Code < items[j].Code
	})
	if offset >= len(items) {
		return []*domainpharma.Product{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], nil
}

func (s *productService) Update(ctx context.Context, id string, in ProductWriteInput) (*domainpharma.Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("product not found")
	}
	if existing := s.findProductByCodeLocked(in.Code, id); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("product code already exists")
	}
	next := cloneProduct(current)
	if err := next.Update(toProductDomainInput(in), s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneProduct(next)
	s.mu.Unlock()
	s.appendProductAudit(ctx, in.ActorID, "pharma_oa.product.update", id, map[string]any{"code": next.Code})
	return cloneProduct(next), nil
}

func (s *productService) Disable(ctx context.Context, id string, in ProductDisableInput) (*domainpharma.Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("product not found")
	}
	next := cloneProduct(current)
	if err := next.Disable(in.Reason, s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneProduct(next)
	s.mu.Unlock()
	s.appendProductAudit(ctx, in.ActorID, "pharma_oa.product.disable", id, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	return cloneProduct(next), nil
}

func (s *productService) Import(ctx context.Context, rows []ProductWriteInput) (ProductImportResult, error) {
	result := ProductImportResult{Failed: []ProductImportFail{}}
	seen := map[string]struct{}{}
	for idx, row := range rows {
		rowNumber := idx + 1
		code := strings.TrimSpace(row.Code)
		if code == "" {
			result.Failed = append(result.Failed, ProductImportFail{Row: rowNumber, Code: code, Message: "code is required"})
			continue
		}
		key := strings.ToLower(code)
		if _, ok := seen[key]; ok {
			result.Failed = append(result.Failed, ProductImportFail{Row: rowNumber, Code: code, Message: "duplicate code in import"})
			continue
		}
		seen[key] = struct{}{}
		if _, err := s.Create(ctx, row); err != nil {
			result.Failed = append(result.Failed, ProductImportFail{Row: rowNumber, Code: code, Message: err.Error()})
			continue
		}
		result.Created++
	}
	s.appendProductAudit(ctx, importActor(rows), "pharma_oa.product.import", "batch", map[string]any{"created": result.Created, "failed": len(result.Failed)})
	return result, nil
}

func (s *productService) findProductByCodeLocked(code string, exceptID string) *domainpharma.Product {
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

func (s *productService) nextProductID() shared.ID {
	s.idCounter++
	return shared.ID("pharma-product-" + strconv.FormatInt(s.idCounter, 10))
}

func (s *productService) appendProductAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actor := strings.TrimSpace(actorID)
	if actor == "" {
		actor = "system"
	}
	_, _ = s.audit.Append(ctx, actor, action, "pharma_oa_product", resourceID, detail)
}

func toProductDomainInput(in ProductWriteInput) domainpharma.ProductInput {
	return domainpharma.ProductInput{
		Code:           in.Code,
		Name:           in.Name,
		Spec:           in.Spec,
		DosageForm:     in.DosageForm,
		Manufacturer:   in.Manufacturer,
		ApprovalNumber: in.ApprovalNumber,
		Temperature:    in.Temperature,
	}
}

func productMatches(item *domainpharma.Product, keyword string) bool {
	if item == nil {
		return false
	}
	return strings.Contains(strings.ToLower(item.ID.String()), keyword) ||
		strings.Contains(strings.ToLower(item.Code), keyword) ||
		strings.Contains(strings.ToLower(item.Name), keyword) ||
		strings.Contains(strings.ToLower(item.Spec), keyword) ||
		strings.Contains(strings.ToLower(item.DosageForm), keyword) ||
		strings.Contains(strings.ToLower(item.Manufacturer), keyword) ||
		strings.Contains(strings.ToLower(item.ApprovalNumber), keyword)
}

func cloneProduct(item *domainpharma.Product) *domainpharma.Product {
	if item == nil {
		return nil
	}
	out := *item
	return &out
}

func importActor(rows []ProductWriteInput) string {
	for _, row := range rows {
		if actor := strings.TrimSpace(row.ActorID); actor != "" {
			return actor
		}
	}
	return "system"
}
