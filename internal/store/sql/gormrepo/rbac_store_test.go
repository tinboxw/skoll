package gormrepo

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestRBACStore_CreateBinding(t *testing.T) {
	db := TestDB(t)
	store := NewRBACStore(db)
	ctx := context.Background()

	binding := &rbac.Binding{
		ID:          shared.ID("1"),
		SubjectType: rbac.SubjectUser,
		SubjectID:   shared.ID("1"),
		RoleID:      shared.ID("1"),
	}
	binding.Meta.Touch(time.Now())

	err := store.CreateBinding(ctx, binding)
	if err != nil {
		t.Fatalf("failed to create binding: %v", err)
	}

	t.Run("retrieve binding", func(t *testing.T) {
		bindings, err := store.ListBindingsBySubject(ctx, rbac.SubjectUser, shared.ID("1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(bindings) != 1 {
			t.Errorf("expected 1 binding, got %d", len(bindings))
		}
	})
}

func TestRBACStore_DeleteBinding(t *testing.T) {
	db := TestDB(t)
	store := NewRBACStore(db)
	ctx := context.Background()

	binding := &rbac.Binding{
		ID:          shared.ID("2"),
		SubjectType: rbac.SubjectUser,
		SubjectID:   shared.ID("2"),
		RoleID:      shared.ID("2"),
	}
	binding.Meta.Touch(time.Now())

	err := store.CreateBinding(ctx, binding)
	if err != nil {
		t.Fatalf("failed to create binding: %v", err)
	}

	err = store.DeleteBinding(ctx, shared.ID("2"))
	if err != nil {
		t.Fatalf("failed to delete binding: %v", err)
	}

	bindings, _ := store.ListBindingsBySubject(ctx, rbac.SubjectUser, shared.ID("2"))
	if len(bindings) != 0 {
		t.Errorf("expected 0 bindings after delete, got %d", len(bindings))
	}
}

func TestRBACStore_PolicyRules(t *testing.T) {
	db := TestDB(t)
	store := NewRBACStore(db)
	ctx := context.Background()

	rules := []rbac.PolicyRule{
		{Resource: "users:*", Action: "read"},
		{Resource: "posts:*", Action: "write"},
	}

	err := store.ReplacePolicyRules(ctx, shared.ID("1"), rules)
	if err != nil {
		t.Fatalf("failed to replace policy rules: %v", err)
	}

	t.Run("list policy rules by role", func(t *testing.T) {
		result, err := store.ListPolicyRulesByRoleID(ctx, shared.ID("1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 rules, got %d", len(result))
		}
	})

	t.Run("replace with new rules", func(t *testing.T) {
		newRules := []rbac.PolicyRule{
			{Resource: "all:*", Action: "*"},
		}

		err = store.ReplacePolicyRules(ctx, shared.ID("1"), newRules)
		if err != nil {
			t.Fatalf("failed to replace policy rules: %v", err)
		}

		result, _ := store.ListPolicyRulesByRoleID(ctx, shared.ID("1"))
		if len(result) != 1 {
			t.Errorf("expected 1 rule after replace, got %d", len(result))
		}
	})
}
