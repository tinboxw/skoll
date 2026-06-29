package organization

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type Status string

const (
	StatusEnabled  Status = "enabled"
	StatusDisabled Status = "disabled"
)

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)

type DepartmentInput struct {
	ID           shared.ID
	ParentID     shared.ID
	Code         string
	Name         string
	LeaderUserID shared.ID
	Status       Status
	Sort         int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PositionInput struct {
	ID          shared.ID
	Code        string
	Name        string
	Description string
	Status      Status
	Sort        int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type UserAssignmentInput struct {
	UserID       shared.ID
	DepartmentID shared.ID
	PositionID   shared.ID
	Primary      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Department struct {
	ID           shared.ID
	ParentID     shared.ID
	Code         string
	Name         string
	LeaderUserID shared.ID
	Status       Status
	Sort         int
	Meta         shared.AuditMeta
}

type Position struct {
	ID          shared.ID
	Code        string
	Name        string
	Description string
	Status      Status
	Sort        int
	Meta        shared.AuditMeta
}

type UserAssignment struct {
	UserID       shared.ID
	DepartmentID shared.ID
	PositionID   shared.ID
	Primary      bool
	Meta         shared.AuditMeta
}

type DepartmentTreeNode struct {
	Department Department
	Children   []DepartmentTreeNode
}

func NewDepartment(in DepartmentInput) (*Department, error) {
	in = normalizeDepartmentInput(in)
	if err := validateDepartmentInput(in); err != nil {
		return nil, err
	}
	item := &Department{
		ID:           in.ID,
		ParentID:     in.ParentID,
		Code:         in.Code,
		Name:         in.Name,
		LeaderUserID: in.LeaderUserID,
		Status:       in.Status,
		Sort:         in.Sort,
	}
	item.Meta.CreatedAt = in.CreatedAt
	item.Meta.UpdatedAt = in.UpdatedAt
	return item, nil
}

func NewPosition(in PositionInput) (*Position, error) {
	in = normalizePositionInput(in)
	if err := validatePositionInput(in); err != nil {
		return nil, err
	}
	item := &Position{
		ID:          in.ID,
		Code:        in.Code,
		Name:        in.Name,
		Description: in.Description,
		Status:      in.Status,
		Sort:        in.Sort,
	}
	item.Meta.CreatedAt = in.CreatedAt
	item.Meta.UpdatedAt = in.UpdatedAt
	return item, nil
}

func NewUserAssignment(in UserAssignmentInput) (*UserAssignment, error) {
	in = normalizeUserAssignmentInput(in)
	if err := validateUserAssignmentInput(in); err != nil {
		return nil, err
	}
	item := &UserAssignment{
		UserID:       in.UserID,
		DepartmentID: in.DepartmentID,
		PositionID:   in.PositionID,
		Primary:      in.Primary,
	}
	item.Meta.CreatedAt = in.CreatedAt
	item.Meta.UpdatedAt = in.UpdatedAt
	return item, nil
}

func (d *Department) Rename(name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if err := validateName(name, "department name"); err != nil {
		return err
	}
	d.Name = name
	d.Meta.Touch(now)
	return nil
}

func (d *Department) Move(parentID shared.ID, now time.Time) error {
	if parentID == d.ID {
		return fmt.Errorf("department parent cannot be itself")
	}
	d.ParentID = parentID
	d.Meta.Touch(now)
	return nil
}

func (d *Department) Enable(now time.Time) {
	d.Status = StatusEnabled
	d.Meta.Touch(now)
}

func (d *Department) Disable(now time.Time) {
	d.Status = StatusDisabled
	d.Meta.Touch(now)
}

func (d *Department) Reorder(sortValue int, now time.Time) error {
	if sortValue < 0 {
		return fmt.Errorf("department sort must be non-negative")
	}
	d.Sort = sortValue
	d.Meta.Touch(now)
	return nil
}

func (d Department) IsRoot() bool {
	return d.ParentID.IsZero()
}

func (p *Position) Rename(name, description string, now time.Time) error {
	name = strings.TrimSpace(name)
	if err := validateName(name, "position name"); err != nil {
		return err
	}
	p.Name = name
	p.Description = strings.TrimSpace(description)
	p.Meta.Touch(now)
	return nil
}

func (p *Position) Enable(now time.Time) {
	p.Status = StatusEnabled
	p.Meta.Touch(now)
}

func (p *Position) Disable(now time.Time) {
	p.Status = StatusDisabled
	p.Meta.Touch(now)
}

func (p *Position) Reorder(sortValue int, now time.Time) error {
	if sortValue < 0 {
		return fmt.Errorf("position sort must be non-negative")
	}
	p.Sort = sortValue
	p.Meta.Touch(now)
	return nil
}

func SortDepartments(items []Department) []Department {
	out := append([]Department(nil), items...)
	sort.SliceStable(out, func(left, right int) bool {
		if out[left].Sort != out[right].Sort {
			return out[left].Sort < out[right].Sort
		}
		return out[left].Code < out[right].Code
	})
	return out
}

func SortPositions(items []Position) []Position {
	out := append([]Position(nil), items...)
	sort.SliceStable(out, func(left, right int) bool {
		if out[left].Sort != out[right].Sort {
			return out[left].Sort < out[right].Sort
		}
		return out[left].Code < out[right].Code
	})
	return out
}

func BuildDepartmentTree(items []Department) ([]DepartmentTreeNode, error) {
	if err := ValidateDepartmentTree(items); err != nil {
		return nil, err
	}
	byParent := make(map[shared.ID][]Department, len(items))
	for _, item := range items {
		byParent[item.ParentID] = append(byParent[item.ParentID], item)
	}
	var build func(parentID shared.ID) []DepartmentTreeNode
	build = func(parentID shared.ID) []DepartmentTreeNode {
		children := SortDepartments(byParent[parentID])
		nodes := make([]DepartmentTreeNode, 0, len(children))
		for _, child := range children {
			nodes = append(nodes, DepartmentTreeNode{
				Department: child,
				Children:   build(child.ID),
			})
		}
		return nodes
	}
	return build(""), nil
}

func ValidateDepartmentTree(items []Department) error {
	byID := make(map[shared.ID]Department, len(items))
	seenCodes := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.ID.IsZero() {
			return fmt.Errorf("department id is required")
		}
		if _, exists := byID[item.ID]; exists {
			return fmt.Errorf("department id is duplicated: %s", item.ID)
		}
		byID[item.ID] = item
		if _, exists := seenCodes[item.Code]; exists {
			return fmt.Errorf("department code is duplicated: %s", item.Code)
		}
		seenCodes[item.Code] = struct{}{}
		if item.ParentID == item.ID {
			return fmt.Errorf("department parent cannot be itself: %s", item.ID)
		}
	}
	for _, item := range items {
		if item.ParentID.IsZero() {
			continue
		}
		if _, exists := byID[item.ParentID]; !exists {
			return fmt.Errorf("department parent does not exist: %s", item.ParentID)
		}
	}
	visiting := map[shared.ID]bool{}
	visited := map[shared.ID]bool{}
	var visit func(shared.ID) error
	visit = func(id shared.ID) error {
		if visited[id] {
			return nil
		}
		if visiting[id] {
			return fmt.Errorf("department tree contains cycle: %s", id)
		}
		visiting[id] = true
		item := byID[id]
		if !item.ParentID.IsZero() {
			if err := visit(item.ParentID); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func normalizeDepartmentInput(in DepartmentInput) DepartmentInput {
	in.Code = normalizeCode(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	if in.Status == "" {
		in.Status = StatusEnabled
	}
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = in.CreatedAt
	}
	return in
}

func normalizePositionInput(in PositionInput) PositionInput {
	in.Code = normalizeCode(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Status == "" {
		in.Status = StatusEnabled
	}
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = in.CreatedAt
	}
	return in
}

func normalizeUserAssignmentInput(in UserAssignmentInput) UserAssignmentInput {
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = in.CreatedAt
	}
	return in
}

func normalizeCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func validateDepartmentInput(in DepartmentInput) error {
	if in.ID.IsZero() {
		return fmt.Errorf("department id is required")
	}
	if in.ParentID == in.ID {
		return fmt.Errorf("department parent cannot be itself")
	}
	if err := validateCode(in.Code, "department code"); err != nil {
		return err
	}
	if err := validateName(in.Name, "department name"); err != nil {
		return err
	}
	if err := validateStatus(in.Status); err != nil {
		return err
	}
	if in.Sort < 0 {
		return fmt.Errorf("department sort must be non-negative")
	}
	return validateTimestamps(in.CreatedAt, in.UpdatedAt, "department")
}

func validatePositionInput(in PositionInput) error {
	if in.ID.IsZero() {
		return fmt.Errorf("position id is required")
	}
	if err := validateCode(in.Code, "position code"); err != nil {
		return err
	}
	if err := validateName(in.Name, "position name"); err != nil {
		return err
	}
	if err := validateStatus(in.Status); err != nil {
		return err
	}
	if in.Sort < 0 {
		return fmt.Errorf("position sort must be non-negative")
	}
	return validateTimestamps(in.CreatedAt, in.UpdatedAt, "position")
}

func validateUserAssignmentInput(in UserAssignmentInput) error {
	if in.UserID.IsZero() {
		return fmt.Errorf("user assignment user id is required")
	}
	if in.DepartmentID.IsZero() {
		return fmt.Errorf("user assignment department id is required")
	}
	return validateTimestamps(in.CreatedAt, in.UpdatedAt, "user assignment")
}

func validateCode(code, label string) error {
	if !codePattern.MatchString(code) {
		return fmt.Errorf("%s must match %s", label, codePattern.String())
	}
	return nil
}

func validateName(value, label string) error {
	if value == "" {
		return fmt.Errorf("%s is required", label)
	}
	if len([]rune(value)) > 255 {
		return fmt.Errorf("%s is too long", label)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s must not contain line breaks", label)
	}
	return nil
}

func validateStatus(status Status) error {
	switch status {
	case StatusEnabled, StatusDisabled:
		return nil
	default:
		return fmt.Errorf("organization status is invalid")
	}
}

func validateTimestamps(createdAt, updatedAt time.Time, label string) error {
	if createdAt.IsZero() {
		return fmt.Errorf("%s created time is required", label)
	}
	if updatedAt.IsZero() {
		return fmt.Errorf("%s updated time is required", label)
	}
	if updatedAt.Before(createdAt) {
		return fmt.Errorf("%s updated time must not be before created time", label)
	}
	return nil
}
