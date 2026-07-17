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

type WarehouseService interface {
	Create(ctx context.Context, in WarehouseWriteInput) (*domainpharma.Warehouse, error)
	List(ctx context.Context, in WarehouseListInput) ([]*domainpharma.Warehouse, error)
	Update(ctx context.Context, id string, in WarehouseWriteInput) (*domainpharma.Warehouse, error)
	Disable(ctx context.Context, id string, in WarehouseDisableInput) (*domainpharma.Warehouse, error)
	ValidateMovementLocation(ctx context.Context, in WarehouseMovementLocationInput) (WarehouseMovementLocationEligibility, error)
}

type WarehouseWriteInput struct {
	Code        string
	Name        string
	Region      string
	Temperature domainpharma.WarehouseTemperature
	Areas       []domainpharma.WarehouseArea
	ActorID     string
}

type WarehouseListInput struct {
	Keyword string
	Status  string
	Region  string
	Offset  int
	Limit   int
}

type WarehouseDisableInput struct {
	Reason  string
	ActorID string
}

type WarehouseMovementLocationInput struct {
	WarehouseID string
	AreaID      string
	LocationID  string
}

type WarehouseMovementLocationEligibility struct {
	WarehouseID string `json:"warehouseId"`
	AreaID      string `json:"areaId"`
	LocationID  string `json:"locationId"`
	Allowed     bool   `json:"allowed"`
	Reason      string `json:"reason,omitempty"`
}

type warehouseService struct {
	repo      pharmaoarepo.WarehouseRepository
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter atomic.Int64
}

func NewWarehouseService(audit auditsvc.Service, repositories ...pharmaoarepo.WarehouseRepository) WarehouseService {
	repo := pharmaoarepo.WarehouseRepository(pharmaoarepo.NewMemoryWarehouseRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &warehouseService{
		repo:  repo,
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *warehouseService) Create(ctx context.Context, in WarehouseWriteInput) (*domainpharma.Warehouse, error) {
	id, err := s.nextAvailableWarehouseID(ctx)
	if err != nil {
		return nil, err
	}
	entity, err := domainpharma.NewWarehouse(id, toWarehouseDomainInput(in), s.nowFn())
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByCode(ctx, entity.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("warehouse code already exists")
	}
	if err := s.repo.Create(ctx, entity); err != nil {
		return nil, err
	}
	s.appendWarehouseAudit(ctx, in.ActorID, "pharma_oa.warehouse.create", entity.ID.String(), map[string]any{"code": entity.Code, "region": entity.Region})
	return cloneWarehouse(entity), nil
}

func (s *warehouseService) List(ctx context.Context, in WarehouseListInput) ([]*domainpharma.Warehouse, error) {
	rows, err := s.repo.List(ctx, pharmaoarepo.ListFilter{Keyword: in.Keyword, Status: in.Status, Region: in.Region, Offset: in.Offset, Limit: normalizeLimit(in.Limit)})
	if err != nil {
		return nil, err
	}
	items := make([]*domainpharma.Warehouse, 0, len(rows))
	for index := range rows {
		items = append(items, cloneWarehouse(&rows[index]))
	}
	return items, nil
}

func (s *warehouseService) Update(ctx context.Context, id string, in WarehouseWriteInput) (*domainpharma.Warehouse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("warehouse not found")
	}
	existing, err := s.repo.GetByCode(ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID.String() != id {
		return nil, fmt.Errorf("warehouse code already exists")
	}
	next := cloneWarehouse(current)
	if err := next.Update(toWarehouseDomainInput(in), s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.appendWarehouseAudit(ctx, in.ActorID, "pharma_oa.warehouse.update", id, map[string]any{"code": next.Code, "region": next.Region})
	return cloneWarehouse(next), nil
}

func (s *warehouseService) Disable(ctx context.Context, id string, in WarehouseDisableInput) (*domainpharma.Warehouse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("warehouse not found")
	}
	next := cloneWarehouse(current)
	if err := next.Disable(in.Reason, s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.appendWarehouseAudit(ctx, in.ActorID, "pharma_oa.warehouse.disable", id, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	return cloneWarehouse(next), nil
}

func (s *warehouseService) ValidateMovementLocation(ctx context.Context, in WarehouseMovementLocationInput) (WarehouseMovementLocationEligibility, error) {
	warehouseID := strings.TrimSpace(in.WarehouseID)
	if warehouseID == "" {
		return WarehouseMovementLocationEligibility{}, fmt.Errorf("warehouseId is required")
	}
	item, err := s.repo.Get(ctx, shared.ID(warehouseID))
	if err != nil {
		return WarehouseMovementLocationEligibility{}, err
	}
	if item == nil {
		return WarehouseMovementLocationEligibility{}, fmt.Errorf("warehouse not found")
	}
	result := WarehouseMovementLocationEligibility{
		WarehouseID: item.ID.String(),
		AreaID:      strings.TrimSpace(in.AreaID),
		LocationID:  strings.TrimSpace(in.LocationID),
		Allowed:     true,
	}
	if err := item.ValidateMovementLocation(in.AreaID, in.LocationID); err != nil {
		result.Allowed = false
		result.Reason = err.Error()
	}
	return result, nil
}

func (s *warehouseService) nextWarehouseID() shared.ID {
	return shared.ID("pharma-warehouse-" + strconv.FormatInt(s.idCounter.Add(1), 10))
}

func (s *warehouseService) nextAvailableWarehouseID(ctx context.Context) (shared.ID, error) {
	for {
		id := s.nextWarehouseID()
		item, err := s.repo.Get(ctx, id)
		if err != nil {
			return "", err
		}
		if item == nil {
			return id, nil
		}
	}
}

func (s *warehouseService) appendWarehouseAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actor := strings.TrimSpace(actorID)
	if actor == "" {
		actor = "system"
	}
	_, _ = s.audit.Append(ctx, actor, action, "pharma_oa_warehouse", resourceID, detail)
}

func toWarehouseDomainInput(in WarehouseWriteInput) domainpharma.WarehouseInput {
	return domainpharma.WarehouseInput{
		Code:        in.Code,
		Name:        in.Name,
		Region:      in.Region,
		Temperature: in.Temperature,
		Areas:       in.Areas,
	}
}

func warehouseMatches(item *domainpharma.Warehouse, keyword string) bool {
	if item == nil {
		return false
	}
	if strings.Contains(strings.ToLower(item.ID.String()), keyword) ||
		strings.Contains(strings.ToLower(item.Code), keyword) ||
		strings.Contains(strings.ToLower(item.Name), keyword) ||
		strings.Contains(strings.ToLower(item.Region), keyword) {
		return true
	}
	for _, area := range item.Areas {
		if strings.Contains(strings.ToLower(area.Code), keyword) ||
			strings.Contains(strings.ToLower(area.Name), keyword) {
			return true
		}
		for _, location := range area.Locations {
			if strings.Contains(strings.ToLower(location.Code), keyword) ||
				strings.Contains(strings.ToLower(location.Name), keyword) {
				return true
			}
		}
	}
	return false
}

func cloneWarehouse(item *domainpharma.Warehouse) *domainpharma.Warehouse {
	if item == nil {
		return nil
	}
	out := *item
	out.Areas = make([]domainpharma.WarehouseArea, 0, len(item.Areas))
	for _, area := range item.Areas {
		next := area
		next.Locations = append([]domainpharma.WarehouseLocation(nil), area.Locations...)
		out.Areas = append(out.Areas, next)
	}
	return &out
}
