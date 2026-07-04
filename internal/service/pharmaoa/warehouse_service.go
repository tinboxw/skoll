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
	mu        sync.RWMutex
	items     map[string]*domainpharma.Warehouse
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter int64
}

func NewWarehouseService(audit auditsvc.Service) WarehouseService {
	return &warehouseService{
		items: map[string]*domainpharma.Warehouse{},
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *warehouseService) Create(ctx context.Context, in WarehouseWriteInput) (*domainpharma.Warehouse, error) {
	entity, err := domainpharma.NewWarehouse(s.nextWarehouseID(), toWarehouseDomainInput(in), s.nowFn())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if existing := s.findWarehouseByCodeLocked(entity.Code, ""); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("warehouse code already exists")
	}
	s.items[entity.ID.String()] = cloneWarehouse(entity)
	s.mu.Unlock()
	s.appendWarehouseAudit(ctx, in.ActorID, "pharma_oa.warehouse.create", entity.ID.String(), map[string]any{"code": entity.Code, "region": entity.Region})
	return cloneWarehouse(entity), nil
}

func (s *warehouseService) List(_ context.Context, in WarehouseListInput) ([]*domainpharma.Warehouse, error) {
	limit := normalizeLimit(in.Limit)
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	status := strings.ToLower(strings.TrimSpace(in.Status))
	region := strings.ToLower(strings.TrimSpace(in.Region))
	s.mu.RLock()
	items := make([]*domainpharma.Warehouse, 0, len(s.items))
	for _, item := range s.items {
		if status != "" && string(item.Status) != status {
			continue
		}
		if region != "" && strings.ToLower(item.Region) != region {
			continue
		}
		if keyword != "" && !warehouseMatches(item, keyword) {
			continue
		}
		items = append(items, cloneWarehouse(item))
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool {
		return items[i].Code < items[j].Code
	})
	if offset >= len(items) {
		return []*domainpharma.Warehouse{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], nil
}

func (s *warehouseService) Update(ctx context.Context, id string, in WarehouseWriteInput) (*domainpharma.Warehouse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("warehouse not found")
	}
	if existing := s.findWarehouseByCodeLocked(in.Code, id); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("warehouse code already exists")
	}
	next := cloneWarehouse(current)
	if err := next.Update(toWarehouseDomainInput(in), s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneWarehouse(next)
	s.mu.Unlock()
	s.appendWarehouseAudit(ctx, in.ActorID, "pharma_oa.warehouse.update", id, map[string]any{"code": next.Code, "region": next.Region})
	return cloneWarehouse(next), nil
}

func (s *warehouseService) Disable(ctx context.Context, id string, in WarehouseDisableInput) (*domainpharma.Warehouse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("warehouse not found")
	}
	next := cloneWarehouse(current)
	if err := next.Disable(in.Reason, s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneWarehouse(next)
	s.mu.Unlock()
	s.appendWarehouseAudit(ctx, in.ActorID, "pharma_oa.warehouse.disable", id, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	return cloneWarehouse(next), nil
}

func (s *warehouseService) ValidateMovementLocation(_ context.Context, in WarehouseMovementLocationInput) (WarehouseMovementLocationEligibility, error) {
	warehouseID := strings.TrimSpace(in.WarehouseID)
	if warehouseID == "" {
		return WarehouseMovementLocationEligibility{}, fmt.Errorf("warehouseId is required")
	}
	s.mu.RLock()
	item := cloneWarehouse(s.items[warehouseID])
	s.mu.RUnlock()
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

func (s *warehouseService) findWarehouseByCodeLocked(code string, exceptID string) *domainpharma.Warehouse {
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

func (s *warehouseService) nextWarehouseID() shared.ID {
	s.idCounter++
	return shared.ID("pharma-warehouse-" + strconv.FormatInt(s.idCounter, 10))
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
