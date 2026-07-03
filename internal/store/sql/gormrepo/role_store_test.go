package gormrepo

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestRoleStore_GetByID(t *testing.T) {
	db := TestDB(t)
	store := NewRoleStore(db, nil)
	ctx := context.Background()

	testRole := &role.Role{
		ID:   shared.ID("1"),
		Key:  "admin",
		Name: "Administrator",
	}
	testRole.Meta.Touch(time.Now())

	err := store.Save(ctx, testRole)
	if err != nil {
		t.Fatalf("failed to save role: %v", err)
	}

	t.Run("existing role", func(t *testing.T) {
		result, err := store.GetByID(ctx, shared.ID("1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected role, got nil")
		}
		if result.Key != "admin" {
			t.Errorf("expected key 'admin', got '%s'", result.Key)
		}
	})

	t.Run("non-existing role", func(t *testing.T) {
		result, err := store.GetByID(ctx, shared.ID("999"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for non-existing role, got %+v", result)
		}
	})
}

func TestRoleStore_GetByKey(t *testing.T) {
	db := TestDB(t)
	store := NewRoleStore(db, nil)
	ctx := context.Background()

	roles := []*role.Role{
		{ID: shared.ID("1"), Key: "admin", Name: "Admin"},
		{ID: shared.ID("2"), Key: "editor", Name: "Editor"},
	}
	for _, r := range roles {
		r.Meta.Touch(time.Now())
		if err := store.Save(ctx, r); err != nil {
			t.Fatalf("failed to save role: %v", err)
		}
	}

	t.Run("existing key", func(t *testing.T) {
		result, err := store.GetByKey(ctx, "admin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected role, got nil")
		}
	})

	t.Run("non-existing key", func(t *testing.T) {
		result, err := store.GetByKey(ctx, "superadmin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})
}

func TestRoleStore_List(t *testing.T) {
	db := TestDB(t)
	store := NewRoleStore(db, nil)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		r := &role.Role{
			ID:   shared.ID(string(rune('0' + i))),
			Key:  string(rune('a'+i-1)) + "_role",
			Name: "Role " + string(rune('0'+i)),
		}
		r.Meta.Touch(time.Now())
		if err := store.Save(ctx, r); err != nil {
			t.Fatalf("failed to save role %d: %v", i, err)
		}
	}

	t.Run("list all", func(t *testing.T) {
		result, err := store.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 roles, got %d", len(result))
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		result, err := store.List(ctx, 1, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 roles, got %d", len(result))
		}
	})
}

func TestRoleStore_SaveAndUpdate(t *testing.T) {
	db := TestDB(t)
	store := NewRoleStore(db, nil)
	ctx := context.Background()

	original := &role.Role{
		ID:   shared.ID("1"),
		Key:  "viewer",
		Name: "Viewer Role",
	}
	original.Meta.Touch(time.Now())

	err := store.Save(ctx, original)
	if err != nil {
		t.Fatalf("failed to save role: %v", err)
	}

	updated := &role.Role{
		ID:   shared.ID("1"),
		Key:  "viewer",
		Name: "Updated Viewer",
	}
	updated.Meta.Touch(time.Now())

	err = store.Save(ctx, updated)
	if err != nil {
		t.Fatalf("failed to update role: %v", err)
	}

	result, err := store.GetByID(ctx, shared.ID("1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Viewer" {
		t.Errorf("expected name 'Updated Viewer', got '%s'", result.Name)
	}
}

func TestRoleStore_Delete(t *testing.T) {
	db := TestDB(t)
	store := NewRoleStore(db, nil)
	ctx := context.Background()

	r := &role.Role{
		ID:   shared.ID("1"),
		Key:  "deletable",
		Name: "To Delete",
	}
	r.Meta.Touch(time.Now())

	if err := store.Save(ctx, r); err != nil {
		t.Fatalf("failed to save role: %v", err)
	}

	exists, _ := store.GetByID(ctx, shared.ID("1"))
	if exists == nil {
		t.Fatal("role should exist before delete")
	}

	err := store.Delete(ctx, shared.ID("1"))
	if err != nil {
		t.Fatalf("failed to delete role: %v", err)
	}

	deleted, _ := store.GetByID(ctx, shared.ID("1"))
	if deleted != nil {
		t.Errorf("expected nil after delete, got %+v", deleted)
	}
}
