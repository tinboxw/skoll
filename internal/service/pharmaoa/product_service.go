package pharmaoa

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
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
	repo      pharmaoarepo.ProductRepository
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter atomic.Int64
}

func NewProductService(audit auditsvc.Service, repositories ...pharmaoarepo.ProductRepository) ProductService {
	repo := pharmaoarepo.ProductRepository(pharmaoarepo.NewMemoryProductRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &productService{
		repo:  repo,
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *productService) Create(ctx context.Context, in ProductWriteInput) (*domainpharma.Product, error) {
	id, err := s.nextAvailableProductID(ctx)
	if err != nil {
		return nil, err
	}
	entity, err := domainpharma.NewProduct(id, toProductDomainInput(in), s.nowFn())
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByCode(ctx, entity.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("product code already exists")
	}
	existing, err = s.repo.GetByApprovalNumber(ctx, entity.ApprovalNumber)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("product approval number already exists")
	}
	if err := s.repo.Create(ctx, entity); err != nil {
		return nil, err
	}
	s.appendProductAudit(ctx, in.ActorID, "pharma_oa.product.create", entity.ID.String(), map[string]any{"code": entity.Code})
	return cloneProduct(entity), nil
}

func (s *productService) List(ctx context.Context, in ProductListInput) ([]*domainpharma.Product, error) {
	rows, err := s.repo.List(ctx, pharmaoarepo.ListFilter{Keyword: in.Keyword, Status: in.Status, Offset: in.Offset, Limit: normalizeLimit(in.Limit)})
	if err != nil {
		return nil, err
	}
	items := make([]*domainpharma.Product, 0, len(rows))
	for index := range rows {
		items = append(items, cloneProduct(&rows[index]))
	}
	return items, nil
}

func (s *productService) Update(ctx context.Context, id string, in ProductWriteInput) (*domainpharma.Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("product not found")
	}
	existing, err := s.repo.GetByCode(ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID.String() != id {
		return nil, fmt.Errorf("product code already exists")
	}
	existing, err = s.repo.GetByApprovalNumber(ctx, in.ApprovalNumber)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID.String() != id {
		return nil, fmt.Errorf("product approval number already exists")
	}
	next := cloneProduct(current)
	if err := next.Update(toProductDomainInput(in), s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.appendProductAudit(ctx, in.ActorID, "pharma_oa.product.update", id, map[string]any{"code": next.Code})
	return cloneProduct(next), nil
}

func (s *productService) Disable(ctx context.Context, id string, in ProductDisableInput) (*domainpharma.Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("product not found")
	}
	next := cloneProduct(current)
	if err := next.Disable(in.Reason, s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
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

func (s *productService) nextProductID() shared.ID {
	return shared.ID("pharma-product-" + strconv.FormatInt(s.idCounter.Add(1), 10))
}

func (s *productService) nextAvailableProductID(ctx context.Context) (shared.ID, error) {
	for {
		id := s.nextProductID()
		item, err := s.repo.Get(ctx, id)
		if err != nil {
			return "", err
		}
		if item == nil {
			return id, nil
		}
	}
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
