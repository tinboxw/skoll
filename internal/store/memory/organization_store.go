package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	domainorg "github.com/tinboxw/skoll/internal/domain/organization"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type OrganizationStore struct {
	mu              sync.RWMutex
	departments     map[shared.ID]domainorg.Department
	departmentCodes map[string]shared.ID
	positions       map[shared.ID]domainorg.Position
	positionCodes   map[string]shared.ID
	assignments     map[shared.ID]domainorg.UserAssignment
}

func NewOrganizationStore() *OrganizationStore {
	return &OrganizationStore{
		departments:     map[shared.ID]domainorg.Department{},
		departmentCodes: map[string]shared.ID{},
		positions:       map[shared.ID]domainorg.Position{},
		positionCodes:   map[string]shared.ID{},
		assignments:     map[shared.ID]domainorg.UserAssignment{},
	}
}

func (s *OrganizationStore) GetDepartmentByID(_ context.Context, id shared.ID) (*domainorg.Department, error) {
	if id.IsZero() {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.departments[id]
	if !ok {
		return nil, nil
	}
	return cloneDepartment(item), nil
}

func (s *OrganizationStore) GetDepartmentByCode(_ context.Context, code string) (*domainorg.Department, error) {
	code = normalizeOrgCode(code)
	if code == "" {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.departmentCodes[code]
	if !ok {
		return nil, nil
	}
	item := s.departments[id]
	return cloneDepartment(item), nil
}

func (s *OrganizationStore) ListDepartments(_ context.Context, offset, limit int) ([]domainorg.Department, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domainorg.Department, 0, len(s.departments))
	for _, item := range s.departments {
		items = append(items, item)
	}
	return paginateDepartments(domainorg.SortDepartments(items), offset, limit), nil
}

func (s *OrganizationStore) SaveDepartment(_ context.Context, department *domainorg.Department) error {
	if department == nil {
		return fmt.Errorf("department is required")
	}
	normalized, err := domainorg.NewDepartment(domainorg.DepartmentInput{
		ID:           department.ID,
		ParentID:     department.ParentID,
		Code:         department.Code,
		Name:         department.Name,
		LeaderUserID: department.LeaderUserID,
		Status:       department.Status,
		Sort:         department.Sort,
		CreatedAt:    department.Meta.CreatedAt,
		UpdatedAt:    department.Meta.UpdatedAt,
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if existingID, ok := s.departmentCodes[normalized.Code]; ok && existingID != normalized.ID {
		return fmt.Errorf("department code already exists")
	}
	if !normalized.ParentID.IsZero() {
		if _, ok := s.departments[normalized.ParentID]; !ok {
			return fmt.Errorf("department parent does not exist")
		}
	}
	next := make([]domainorg.Department, 0, len(s.departments)+1)
	for id, item := range s.departments {
		if id != normalized.ID {
			next = append(next, item)
		}
	}
	next = append(next, *normalized)
	if err := domainorg.ValidateDepartmentTree(next); err != nil {
		return err
	}
	if old, ok := s.departments[normalized.ID]; ok && old.Code != normalized.Code {
		delete(s.departmentCodes, old.Code)
	}
	s.departments[normalized.ID] = *normalized
	s.departmentCodes[normalized.Code] = normalized.ID
	return nil
}

func (s *OrganizationStore) DeleteDepartment(_ context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := s.departmentSubtreeIDsLocked(id)
	for departmentID := range ids {
		if item, ok := s.departments[departmentID]; ok {
			delete(s.departmentCodes, item.Code)
			delete(s.departments, departmentID)
		}
	}
	for userID, assignment := range s.assignments {
		if _, ok := ids[assignment.DepartmentID]; ok {
			delete(s.assignments, userID)
		}
	}
	return nil
}

func (s *OrganizationStore) GetPositionByID(_ context.Context, id shared.ID) (*domainorg.Position, error) {
	if id.IsZero() {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.positions[id]
	if !ok {
		return nil, nil
	}
	return clonePosition(item), nil
}

func (s *OrganizationStore) GetPositionByCode(_ context.Context, code string) (*domainorg.Position, error) {
	code = normalizeOrgCode(code)
	if code == "" {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.positionCodes[code]
	if !ok {
		return nil, nil
	}
	item := s.positions[id]
	return clonePosition(item), nil
}

func (s *OrganizationStore) ListPositions(_ context.Context, offset, limit int) ([]domainorg.Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domainorg.Position, 0, len(s.positions))
	for _, item := range s.positions {
		items = append(items, item)
	}
	return paginatePositions(domainorg.SortPositions(items), offset, limit), nil
}

func (s *OrganizationStore) SavePosition(_ context.Context, position *domainorg.Position) error {
	if position == nil {
		return fmt.Errorf("position is required")
	}
	normalized, err := domainorg.NewPosition(domainorg.PositionInput{
		ID:          position.ID,
		Code:        position.Code,
		Name:        position.Name,
		Description: position.Description,
		Status:      position.Status,
		Sort:        position.Sort,
		CreatedAt:   position.Meta.CreatedAt,
		UpdatedAt:   position.Meta.UpdatedAt,
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existingID, ok := s.positionCodes[normalized.Code]; ok && existingID != normalized.ID {
		return fmt.Errorf("position code already exists")
	}
	if old, ok := s.positions[normalized.ID]; ok && old.Code != normalized.Code {
		delete(s.positionCodes, old.Code)
	}
	s.positions[normalized.ID] = *normalized
	s.positionCodes[normalized.Code] = normalized.ID
	return nil
}

func (s *OrganizationStore) DeletePosition(_ context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.positions[id]; ok {
		delete(s.positionCodes, item.Code)
		delete(s.positions, id)
	}
	for userID, assignment := range s.assignments {
		if assignment.PositionID == id {
			assignment.PositionID = ""
			s.assignments[userID] = assignment
		}
	}
	return nil
}

func (s *OrganizationStore) GetUserAssignment(_ context.Context, userID shared.ID) (*domainorg.UserAssignment, error) {
	if userID.IsZero() {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.assignments[userID]
	if !ok {
		return nil, nil
	}
	return cloneUserAssignment(item), nil
}

func (s *OrganizationStore) ListUserAssignmentsByDepartment(_ context.Context, departmentIDs []shared.ID, offset, limit int) ([]domainorg.UserAssignment, error) {
	departmentSet := make(map[shared.ID]struct{}, len(departmentIDs))
	for _, id := range departmentIDs {
		if !id.IsZero() {
			departmentSet[id] = struct{}{}
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domainorg.UserAssignment, 0, len(s.assignments))
	for _, item := range s.assignments {
		if len(departmentSet) > 0 {
			if _, ok := departmentSet[item.DepartmentID]; !ok {
				continue
			}
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].UserID < items[j].UserID })
	return paginateAssignments(items, offset, limit), nil
}

func (s *OrganizationStore) SaveUserAssignment(_ context.Context, assignment *domainorg.UserAssignment) error {
	if assignment == nil {
		return fmt.Errorf("user assignment is required")
	}
	normalized, err := domainorg.NewUserAssignment(domainorg.UserAssignmentInput{
		UserID:       assignment.UserID,
		DepartmentID: assignment.DepartmentID,
		PositionID:   assignment.PositionID,
		Primary:      assignment.Primary,
		CreatedAt:    assignment.Meta.CreatedAt,
		UpdatedAt:    assignment.Meta.UpdatedAt,
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.departments[normalized.DepartmentID]; !ok {
		return fmt.Errorf("department does not exist")
	}
	if !normalized.PositionID.IsZero() {
		if _, ok := s.positions[normalized.PositionID]; !ok {
			return fmt.Errorf("position does not exist")
		}
	}
	s.assignments[normalized.UserID] = *normalized
	return nil
}

func (s *OrganizationStore) DeleteUserAssignment(_ context.Context, userID shared.ID) error {
	if userID.IsZero() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.assignments, userID)
	return nil
}

func (s *OrganizationStore) departmentSubtreeIDsLocked(rootID shared.ID) map[shared.ID]struct{} {
	out := map[shared.ID]struct{}{}
	var visit func(shared.ID)
	visit = func(id shared.ID) {
		if _, ok := out[id]; ok {
			return
		}
		out[id] = struct{}{}
		for _, item := range s.departments {
			if item.ParentID == id {
				visit(item.ID)
			}
		}
	}
	visit(rootID)
	return out
}

func normalizeOrgCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cloneDepartment(item domainorg.Department) *domainorg.Department {
	clone := item
	return &clone
}

func clonePosition(item domainorg.Position) *domainorg.Position {
	clone := item
	return &clone
}

func cloneUserAssignment(item domainorg.UserAssignment) *domainorg.UserAssignment {
	clone := item
	return &clone
}

func paginateDepartments(items []domainorg.Department, offset, limit int) []domainorg.Department {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(items)
	}
	if offset > len(items) {
		return []domainorg.Department{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]domainorg.Department(nil), items[offset:end]...)
}

func paginatePositions(items []domainorg.Position, offset, limit int) []domainorg.Position {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(items)
	}
	if offset > len(items) {
		return []domainorg.Position{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]domainorg.Position(nil), items[offset:end]...)
}

func paginateAssignments(items []domainorg.UserAssignment, offset, limit int) []domainorg.UserAssignment {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(items)
	}
	if offset > len(items) {
		return []domainorg.UserAssignment{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]domainorg.UserAssignment(nil), items[offset:end]...)
}
