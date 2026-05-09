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
