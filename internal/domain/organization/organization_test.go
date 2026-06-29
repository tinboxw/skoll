package organization

import (
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewDepartment(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)

	got, err := NewDepartment(DepartmentInput{
		ID:           "dept-sales",
		ParentID:     "dept-root",
		Code:         " Sales.Team ",
		Name:         " Sales Team ",
		LeaderUserID: "user-lead",
		Sort:         20,
		CreatedAt:    now,
	})
	if err != nil {
		t.Fatalf("NewDepartment() error = %v", err)
	}
	if got.Code != "sales.team" || got.Name != "Sales Team" || got.Status != StatusEnabled || got.ParentID != "dept-root" {
		t.Fatalf("department = %+v", got)
	}
	if !got.Meta.CreatedAt.Equal(now) || !got.Meta.UpdatedAt.Equal(now) {
		t.Fatalf("meta = %+v", got.Meta)
	}
	if got.IsRoot() {
		t.Fatalf("department should not be root")
	}
}

func TestNewDepartmentRejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	valid := DepartmentInput{
		ID:        "dept-sales",
		Code:      "sales.team",
		Name:      "Sales Team",
		CreatedAt: now,
	}
	cases := []struct {
		name   string
		mutate func(*DepartmentInput)
	}{
		{name: "missing id", mutate: func(in *DepartmentInput) { in.ID = "" }},
		{name: "self parent", mutate: func(in *DepartmentInput) { in.ParentID = in.ID }},
		{name: "bad code", mutate: func(in *DepartmentInput) { in.Code = "bad code" }},
		{name: "missing name", mutate: func(in *DepartmentInput) { in.Name = " " }},
		{name: "bad status", mutate: func(in *DepartmentInput) { in.Status = Status("archived") }},
		{name: "bad sort", mutate: func(in *DepartmentInput) { in.Sort = -1 }},
		{name: "bad timestamp", mutate: func(in *DepartmentInput) { in.UpdatedAt = now.Add(-time.Minute) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := valid
			tc.mutate(&in)
			if _, err := NewDepartment(in); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBuildDepartmentTreeSortsAndValidates(t *testing.T) {
	root := mustDepartment(t, "dept-root", "", "root", 10)
	sales := mustDepartment(t, "dept-sales", "dept-root", "sales", 20)
	ops := mustDepartment(t, "dept-ops", "dept-root", "ops", 10)
	east := mustDepartment(t, "dept-east", "dept-sales", "sales.east", 10)

	tree, err := BuildDepartmentTree([]Department{sales, east, root, ops})
	if err != nil {
		t.Fatalf("BuildDepartmentTree() error = %v", err)
	}
	if len(tree) != 1 || tree[0].Department.ID != "dept-root" {
		t.Fatalf("unexpected roots: %+v", tree)
	}
	children := tree[0].Children
	if len(children) != 2 || children[0].Department.ID != "dept-ops" || children[1].Department.ID != "dept-sales" {
		t.Fatalf("unexpected sorted children: %+v", children)
	}
	if len(children[1].Children) != 1 || children[1].Children[0].Department.ID != "dept-east" {
		t.Fatalf("unexpected nested children: %+v", children[1].Children)
	}

	_, err = BuildDepartmentTree([]Department{root, mustDepartment(t, "dept-orphan", "missing", "orphan", 10)})
	if err == nil || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("expected parent validation error, got %v", err)
	}

	cycleA := mustDepartment(t, "dept-a", "dept-b", "cycle.a", 10)
	cycleB := mustDepartment(t, "dept-b", "dept-a", "cycle.b", 20)
	_, err = BuildDepartmentTree([]Department{cycleA, cycleB})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle validation error, got %v", err)
	}
}

func TestDepartmentStatusAndOrdering(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	item := mustDepartment(t, "dept-sales", "", "sales", 20)
	item.Disable(now.Add(time.Minute))
	if item.Status != StatusDisabled {
		t.Fatalf("expected disabled department, got %+v", item)
	}
	item.Enable(now.Add(2 * time.Minute))
	if item.Status != StatusEnabled {
		t.Fatalf("expected enabled department, got %+v", item)
	}
	if err := item.Move("dept-root", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	if item.ParentID != "dept-root" {
		t.Fatalf("expected moved department, got %+v", item)
	}
	if err := item.Reorder(5, now.Add(4*time.Minute)); err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}
	if item.Sort != 5 {
		t.Fatalf("sort = %d", item.Sort)
	}
	if err := item.Reorder(-1, now); err == nil {
		t.Fatal("expected reorder validation error")
	}
	sorted := SortDepartments([]Department{
		mustDepartment(t, "dept-z", "", "zeta", 20),
		mustDepartment(t, "dept-a", "", "alpha", 20),
		mustDepartment(t, "dept-b", "", "beta", 10),
	})
	if sorted[0].Code != "beta" || sorted[1].Code != "alpha" || sorted[2].Code != "zeta" {
		t.Fatalf("sorted departments = %+v", sorted)
	}
}

func TestNewPositionAndUserAssignment(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)

	pos, err := NewPosition(PositionInput{
		ID:          "pos-manager",
		Code:        " Manager ",
		Name:        " Manager ",
		Description: " Sales manager ",
		Sort:        10,
		CreatedAt:   now,
	})
	if err != nil {
		t.Fatalf("NewPosition() error = %v", err)
	}
	if pos.Code != "manager" || pos.Name != "Manager" || pos.Description != "Sales manager" || pos.Status != StatusEnabled {
		t.Fatalf("position = %+v", pos)
	}
	pos.Disable(now.Add(time.Minute))
	if pos.Status != StatusDisabled {
		t.Fatalf("expected disabled position, got %+v", pos)
	}
	if err := pos.Rename("Senior Manager", "Lead sales team", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := pos.Reorder(30, now.Add(3*time.Minute)); err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}
	if pos.Name != "Senior Manager" || pos.Sort != 30 {
		t.Fatalf("position after update = %+v", pos)
	}

	assignment, err := NewUserAssignment(UserAssignmentInput{
		UserID:       "user-1",
		DepartmentID: "dept-sales",
		PositionID:   "pos-manager",
		Primary:      true,
		CreatedAt:    now,
	})
	if err != nil {
		t.Fatalf("NewUserAssignment() error = %v", err)
	}
	if assignment.UserID != "user-1" || assignment.DepartmentID != "dept-sales" || assignment.PositionID != "pos-manager" || !assignment.Primary {
		t.Fatalf("assignment = %+v", assignment)
	}
}

func TestPositionAndAssignmentRejectInvalidInput(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	if _, err := NewPosition(PositionInput{ID: "pos-1", Code: "bad code", Name: "Manager", CreatedAt: now}); err == nil {
		t.Fatal("expected position code validation error")
	}
	if _, err := NewPosition(PositionInput{ID: "pos-1", Code: "manager", Name: "", CreatedAt: now}); err == nil {
		t.Fatal("expected position name validation error")
	}
	if _, err := NewPosition(PositionInput{ID: "pos-1", Code: "manager", Name: "Manager", Status: Status("archived"), CreatedAt: now}); err == nil {
		t.Fatal("expected position status validation error")
	}
	if _, err := NewUserAssignment(UserAssignmentInput{DepartmentID: "dept-sales", CreatedAt: now}); err == nil {
		t.Fatal("expected user id validation error")
	}
	if _, err := NewUserAssignment(UserAssignmentInput{UserID: "user-1", CreatedAt: now}); err == nil {
		t.Fatal("expected department id validation error")
	}
}

func mustDepartment(t *testing.T, id, parentID, code string, sortValue int) Department {
	t.Helper()
	item, err := NewDepartment(DepartmentInput{
		ID:        shared.ID(id),
		ParentID:  shared.ID(parentID),
		Code:      code,
		Name:      code,
		Sort:      sortValue,
		CreatedAt: time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewDepartment() error = %v", err)
	}
	return *item
}
