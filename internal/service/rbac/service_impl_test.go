package rbac

import (
	"context"
	"testing"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestRBACServiceCheckPermission(t *testing.T) {
	repo := memory.NewRBACStore()
	svc := NewService(repo)

	_, err := svc.BindRole(context.Background(), BindRoleInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-1",
		RoleID:      "r-1",
		Scope:       domainrbac.DataScopeAll,
	})
	if err != nil {
		t.Fatalf("BindRole error: %v", err)
	}

	err = svc.SetRolePolicies(context.Background(), SetRolePoliciesInput{
		RoleID: "r-1",
		Rules: []domainrbac.PolicyRule{
			{Resource: "user:*", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeAll},
		},
	})
	if err != nil {
		t.Fatalf("SetRolePolicies error: %v", err)
	}

	ok, err := svc.CheckPermission(context.Background(), CheckPermissionInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-1",
		Resource:    "user:profile",
		Action:      "read",
	})
	if err != nil {
		t.Fatalf("CheckPermission error: %v", err)
	}
	if !ok {
		t.Fatalf("expected permission allowed")
	}
}

func TestRBACServiceListBindingsByUser(t *testing.T) {
	repo := memory.NewRBACStore()
	svc := NewService(repo)

	_, err := svc.BindRole(context.Background(), BindRoleInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-2",
		RoleID:      "r-2",
		Scope:       domainrbac.DataScopeSelf,
	})
	if err != nil {
		t.Fatalf("BindRole error: %v", err)
	}

	bindings, err := svc.ListBindingsByUser(context.Background(), "u-2")
	if err != nil {
		t.Fatalf("ListBindingsByUser error: %v", err)
	}
	if len(bindings) != 1 {
		t.Fatalf("expected one binding, got %d", len(bindings))
	}
}
