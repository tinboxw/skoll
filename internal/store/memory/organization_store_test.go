package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	domainorg "github.com/tinboxw/skoll/internal/domain/organization"
	"github.com/tinboxw/skoll/internal/domain/shared"
	organizationrepo "github.com/tinboxw/skoll/internal/repository/organization"
)

func TestOrganizationStoreImplementsRepository(t *testing.T) {
	var _ organizationrepo.OrganizationRepository = (*OrganizationStore)(nil)
}

func TestOrganizationStoreDepartmentPositionAndAssignmentContract(t *testing.T) {
	ctx := context.Background()
	store := NewOrganizationStore()

	root := mustMemoryDepartment(t, "dept-root", "", "root", 0)
	sales := mustMemoryDepartment(t, "dept-sales", "dept-root", "sales", 20)
	ops := mustMemoryDepartment(t, "dept-ops", "dept-root", "ops", 10)
	if err := store.SaveDepartment(ctx, root); err != nil {
		t.Fatalf("SaveDepartment(root) error = %v", err)
	}
	if err := store.SaveDepartment(ctx, sales); err != nil {
		t.Fatalf("SaveDepartment(sales) error = %v", err)
	}
	if err := store.SaveDepartment(ctx, ops); err != nil {
		t.Fatalf("SaveDepartment(ops) error = %v", err)
	}
	gotDept, err := store.GetDepartmentByCode(ctx, "SALES")
	if err != nil {
		t.Fatalf("GetDepartmentByCode() error = %v", err)
	}
	if gotDept == nil || gotDept.ID != "dept-sales" {
		t.Fatalf("gotDept = %#v", gotDept)
	}
	departments, err := store.ListDepartments(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListDepartments() error = %v", err)
	}
	if len(departments) != 3 || departments[0].Code != "root" || departments[1].Code != "ops" {
		t.Fatalf("departments = %#v", departments)
	}

	manager := mustMemoryPosition(t, "pos-manager", "manager", 20)
	associate := mustMemoryPosition(t, "pos-associate", "associate", 10)
	if err := store.SavePosition(ctx, manager); err != nil {
		t.Fatalf("SavePosition(manager) error = %v", err)
	}
	if err := store.SavePosition(ctx, associate); err != nil {
		t.Fatalf("SavePosition(associate) error = %v", err)
	}
	positions, err := store.ListPositions(ctx, 0, 1)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(positions) != 1 || positions[0].Code != "associate" {
		t.Fatalf("positions = %#v", positions)
	}

	assignment := mustMemoryAssignment(t, "user-1", "dept-sales", "pos-manager")
	if err := store.SaveUserAssignment(ctx, assignment); err != nil {
		t.Fatalf("SaveUserAssignment() error = %v", err)
	}
	gotAssignment, err := store.GetUserAssignment(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserAssignment() error = %v", err)
	}
	if gotAssignment == nil || gotAssignment.DepartmentID != "dept-sales" || gotAssignment.PositionID != "pos-manager" {
		t.Fatalf("gotAssignment = %#v", gotAssignment)
	}
	assignments, err := store.ListUserAssignmentsByDepartment(ctx, []shared.ID{"dept-sales"}, 0, 0)
	if err != nil {
		t.Fatalf("ListUserAssignmentsByDepartment() error = %v", err)
	}
	if len(assignments) != 1 || assignments[0].UserID != "user-1" {
		t.Fatalf("assignments = %#v", assignments)
	}

	if err := store.DeletePosition(ctx, "pos-manager"); err != nil {
		t.Fatalf("DeletePosition() error = %v", err)
	}
	gotAssignment, err = store.GetUserAssignment(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserAssignment(after position delete) error = %v", err)
	}
	if gotAssignment == nil || gotAssignment.PositionID != "" {
		t.Fatalf("expected assignment position cleared, got %#v", gotAssignment)
	}

	if err := store.DeleteDepartment(ctx, "dept-root"); err != nil {
		t.Fatalf("DeleteDepartment() error = %v", err)
	}
	departments, err = store.ListDepartments(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListDepartments(after delete) error = %v", err)
	}
	if len(departments) != 0 {
		t.Fatalf("expected subtree deleted, got %#v", departments)
	}
	gotAssignment, err = store.GetUserAssignment(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserAssignment(after department delete) error = %v", err)
	}
	if gotAssignment != nil {
		t.Fatalf("expected assignment deleted, got %#v", gotAssignment)
	}
}

func TestOrganizationStoreRejectsInvalidReferences(t *testing.T) {
	ctx := context.Background()
	store := NewOrganizationStore()

	if err := store.SaveDepartment(ctx, mustMemoryDepartment(t, "dept-child", "missing", "child", 10)); err == nil || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("expected missing parent error, got %v", err)
	}
	if err := store.SaveDepartment(ctx, mustMemoryDepartment(t, "dept-root", "", "root", 10)); err != nil {
		t.Fatalf("SaveDepartment(root) error = %v", err)
	}
	if err := store.SaveDepartment(ctx, mustMemoryDepartment(t, "dept-root-2", "", "ROOT", 20)); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate code error, got %v", err)
	}
	if err := store.SaveUserAssignment(ctx, mustMemoryAssignment(t, "user-1", "missing", "")); err == nil || !strings.Contains(err.Error(), "department") {
		t.Fatalf("expected missing department error, got %v", err)
	}
	if err := store.SaveUserAssignment(ctx, mustMemoryAssignment(t, "user-1", "dept-root", "missing")); err == nil || !strings.Contains(err.Error(), "position") {
		t.Fatalf("expected missing position error, got %v", err)
	}
}

func mustMemoryDepartment(t *testing.T, id, parentID, code string, sortValue int) *domainorg.Department {
	t.Helper()
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	item, err := domainorg.NewDepartment(domainorg.DepartmentInput{
		ID:        shared.ID(id),
		ParentID:  shared.ID(parentID),
		Code:      code,
		Name:      code,
		Sort:      sortValue,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("NewDepartment() error = %v", err)
	}
	return item
}

func mustMemoryPosition(t *testing.T, id, code string, sortValue int) *domainorg.Position {
	t.Helper()
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	item, err := domainorg.NewPosition(domainorg.PositionInput{
		ID:        shared.ID(id),
		Code:      code,
		Name:      code,
		Sort:      sortValue,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("NewPosition() error = %v", err)
	}
	return item
}

func mustMemoryAssignment(t *testing.T, userID, departmentID, positionID string) *domainorg.UserAssignment {
	t.Helper()
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	item, err := domainorg.NewUserAssignment(domainorg.UserAssignmentInput{
		UserID:       shared.ID(userID),
		DepartmentID: shared.ID(departmentID),
		PositionID:   shared.ID(positionID),
		Primary:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("NewUserAssignment() error = %v", err)
	}
	return item
}
