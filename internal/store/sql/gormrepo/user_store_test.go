package gormrepo

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
)

func TestUserStore_GetByID(t *testing.T) {
	db := TestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	// Create a test user
	testUser := &user.User{
		ID:      shared.ID("1"),
		Account: "testuser",
		Name:    "Test User",
		Email:   user.Email("test@example.com"),
		Status:  user.StatusActive,
	}
	testUser.Meta.Touch(time.Now())

	err := store.Save(ctx, testUser)
	if err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	t.Run("existing user", func(t *testing.T) {
		result, err := store.GetByID(ctx, shared.ID("1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected user, got nil")
		}
		if result.Account != "testuser" {
			t.Errorf("expected account 'testuser', got '%s'", result.Account)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		result, err := store.GetByID(ctx, shared.ID("999"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for non-existing user, got %+v", result)
		}
	})
}

func TestUserStore_GetByAccount(t *testing.T) {
	db := TestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	// Create test users
	users := []*user.User{
		{ID: shared.ID("1"), Account: "admin", Name: "Admin User", Email: user.Email("admin@test.com")},
		{ID: shared.ID("2"), Account: "editor", Name: "Editor User", Email: user.Email("editor@test.com")},
	}
	for _, u := range users {
		u.Meta.Touch(time.Now())
		if err := store.Save(ctx, u); err != nil {
			t.Fatalf("failed to save user: %v", err)
		}
	}

	t.Run("existing account", func(t *testing.T) {
		result, err := store.GetByAccount(ctx, "admin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected user, got nil")
		}
		if result.Name != "Admin User" {
			t.Errorf("expected name 'Admin User', got '%s'", result.Name)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		result, err := store.GetByAccount(ctx, "EDITOR")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected user (case insensitive), got nil")
		}
	})

	t.Run("non-existing account", func(t *testing.T) {
		result, err := store.GetByAccount(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})
}

func TestUserStore_List(t *testing.T) {
	db := TestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	// Create multiple users
	for i := 1; i <= 5; i++ {
		u := &user.User{
			ID:      shared.ID(string(rune('0' + i))),
			Account: string(rune('a'+i-1)) + "user",
			Name:    "User " + string(rune('0'+i)),
			Email:   user.Email(string(rune('a'+i-1)) + "@test.com"),
		}
		u.Meta.Touch(time.Now())
		if err := store.Save(ctx, u); err != nil {
			t.Fatalf("failed to save user %d: %v", i, err)
		}
	}

	t.Run("list all with pagination", func(t *testing.T) {
		result, err := store.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 5 {
			t.Errorf("expected 5 users, got %d", len(result))
		}
	})

	t.Run("list with offset and limit", func(t *testing.T) {
		result, err := store.List(ctx, 2, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 users, got %d", len(result))
		}
	})
}

func TestUserStore_ListFiltered(t *testing.T) {
	db := TestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	users := []*user.User{
		{ID: shared.ID("1"), Account: "alice", Name: "Alice", Email: user.Email("alice@test.com"), DepartmentID: "dept-a"},
		{ID: shared.ID("2"), Account: "bob", Name: "Bob", Email: user.Email("bob@test.com"), DepartmentID: "dept-b"},
		{ID: shared.ID("3"), Account: "carol", Name: "Carol", Email: user.Email("carol@test.com"), DepartmentID: "dept-c"},
		{ID: shared.ID("4"), Account: "dave", Name: "Dave", Email: user.Email("dave@test.com"), DepartmentID: "dept-b"},
	}
	for _, u := range users {
		u.Meta.Touch(time.Now())
		if err := store.Save(ctx, u); err != nil {
			t.Fatalf("failed to save user: %v", err)
		}
	}

	result, err := store.ListFiltered(ctx, userrepo.ListFilter{
		UserIDs:       []shared.ID{"1"},
		DepartmentIDs: []string{"dept-b"},
	}, 0, 10)
	if err != nil {
		t.Fatalf("ListFiltered error: %v", err)
	}
	if accounts := gormUserAccounts(result); accounts != "alice,bob,dave" {
		t.Fatalf("unexpected filtered accounts: %s", accounts)
	}

	paged, err := store.ListFiltered(ctx, userrepo.ListFilter{DepartmentIDs: []string{"dept-b"}}, 1, 1)
	if err != nil {
		t.Fatalf("ListFiltered paged error: %v", err)
	}
	if accounts := gormUserAccounts(paged); accounts != "dave" {
		t.Fatalf("unexpected paged accounts: %s", accounts)
	}
}

func TestUserStore_SaveAndUpdate(t *testing.T) {
	db := TestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	// Create
	original := &user.User{
		ID:      shared.ID("1"),
		Account: "original",
		Name:    "Original Name",
	}
	original.Meta.Touch(time.Now())

	err := store.Save(ctx, original)
	if err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	// Update
	updated := &user.User{
		ID:      shared.ID("1"),
		Account: "updated",
		Name:    "Updated Name",
	}
	updated.Meta.Touch(time.Now())

	err = store.Save(ctx, updated)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	// Verify update
	result, err := store.GetByID(ctx, shared.ID("1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Account != "updated" {
		t.Errorf("expected account 'updated', got '%s'", result.Account)
	}
	if result.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", result.Name)
	}
}

func gormUserAccounts(items []*user.User) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ","
		}
		out += item.Account
	}
	return out
}

func TestUserStore_Delete(t *testing.T) {
	db := TestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	// Create user
	u := &user.User{
		ID:      shared.ID("1"),
		Account: "deletable",
		Name:    "To Delete",
	}
	u.Meta.Touch(time.Now())
	if err := store.Save(ctx, u); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	// Verify exists
	exists, _ := store.GetByID(ctx, shared.ID("1"))
	if exists == nil {
		t.Fatal("user should exist before delete")
	}

	// Delete
	err := store.Delete(ctx, shared.ID("1"))
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Verify deleted
	deleted, _ := store.GetByID(ctx, shared.ID("1"))
	if deleted != nil {
		t.Errorf("expected nil after delete, got %+v", deleted)
	}
}
