package rbac

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/store/memory"
)

type fakeRBACRepo struct {
	createBindingErr       error
	listBindingsErr        error
	listPolicyRulesErr     error
	replacePolicyRulesErr  error
	bindings               []*domainrbac.Binding
	policyRulesByRole      map[string][]domainrbac.PolicyRule
	replacePolicyRulesCall bool
}

func (f *fakeRBACRepo) CreateBinding(_ context.Context, _ *domainrbac.Binding) error {
	return f.createBindingErr
}

func (f *fakeRBACRepo) DeleteBinding(_ context.Context, _ shared.ID) error {
	return nil
}

func (f *fakeRBACRepo) ListBindingsBySubject(_ context.Context, _ domainrbac.SubjectType, _ shared.ID) ([]*domainrbac.Binding, error) {
	if f.listBindingsErr != nil {
		return nil, f.listBindingsErr
	}
	return f.bindings, nil
}

func (f *fakeRBACRepo) ListPolicyRulesByRoleID(_ context.Context, roleID shared.ID) ([]domainrbac.PolicyRule, error) {
	if f.listPolicyRulesErr != nil {
		return nil, f.listPolicyRulesErr
	}
	if f.policyRulesByRole == nil {
		return nil, nil
	}
	return f.policyRulesByRole[roleID.String()], nil
}

func (f *fakeRBACRepo) ReplacePolicyRules(_ context.Context, _ shared.ID, _ []domainrbac.PolicyRule) error {
	f.replacePolicyRulesCall = true
	return f.replacePolicyRulesErr
}

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

func TestRBACServiceCheckPermissionNoBindings(t *testing.T) {
	repo := memory.NewRBACStore()
	svc := NewService(repo)

	ok, err := svc.CheckPermission(context.Background(), CheckPermissionInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-no-bind",
		Resource:    "user:profile",
		Action:      "read",
	})
	if err != nil {
		t.Fatalf("CheckPermission error: %v", err)
	}
	if ok {
		t.Fatalf("expected permission denied without bindings")
	}
}

func TestRBACServiceCheckPermissionDenyOverridesAllow(t *testing.T) {
	repo := memory.NewRBACStore()
	svc := NewService(repo)

	_, err := svc.BindRole(context.Background(), BindRoleInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-3",
		RoleID:      "r-3",
		Scope:       domainrbac.DataScopeAll,
	})
	if err != nil {
		t.Fatalf("BindRole error: %v", err)
	}

	err = svc.SetRolePolicies(context.Background(), SetRolePoliciesInput{
		RoleID: "r-3",
		Rules: []domainrbac.PolicyRule{
			{Resource: "user:*", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeAll},
			{Resource: "user:*", Action: "read", Effect: domainrbac.EffectDeny, Scope: domainrbac.DataScopeAll},
		},
	})
	if err != nil {
		t.Fatalf("SetRolePolicies error: %v", err)
	}

	ok, err := svc.CheckPermission(context.Background(), CheckPermissionInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-3",
		Resource:    "user:profile",
		Action:      "read",
	})
	if err != nil {
		t.Fatalf("CheckPermission error: %v", err)
	}
	if ok {
		t.Fatalf("expected deny to override allow")
	}
}

func TestRBACServiceBindRoleInvalidInput(t *testing.T) {
	svc := NewService(memory.NewRBACStore())

	_, err := svc.BindRole(context.Background(), BindRoleInput{SubjectType: domainrbac.SubjectUser, SubjectID: "", RoleID: "r-1", Scope: domainrbac.DataScopeAll})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestRBACServiceListBindingsByUserBlankID(t *testing.T) {
	svc := NewService(memory.NewRBACStore())

	items, err := svc.ListBindingsByUser(context.Background(), "   ")
	if err != nil {
		t.Fatalf("ListBindingsByUser error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list for blank user id, got %d", len(items))
	}
}

func TestRBACServiceSetRolePoliciesInvalidRule(t *testing.T) {
	repo := &fakeRBACRepo{}
	svc := NewService(repo)

	err := svc.SetRolePolicies(context.Background(), SetRolePoliciesInput{RoleID: "r-1", Rules: []domainrbac.PolicyRule{{Resource: "", Action: "read", Effect: domainrbac.EffectAllow}}})
	if err == nil {
		t.Fatalf("expected rule validation error")
	}
	if repo.replacePolicyRulesCall {
		t.Fatalf("replace should not be called for invalid rules")
	}
}

func TestRBACServiceRepoErrors(t *testing.T) {
	t.Run("bind role create binding error", func(t *testing.T) {
		repo := &fakeRBACRepo{createBindingErr: errors.New("create binding failed")}
		svc := NewService(repo)

		_, err := svc.BindRole(context.Background(), BindRoleInput{SubjectType: domainrbac.SubjectUser, SubjectID: "u-1", RoleID: "r-1", Scope: domainrbac.DataScopeAll})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "create binding failed") {
			t.Fatalf("expected create binding error, got %v", err)
		}
	})

	t.Run("check permission list bindings error", func(t *testing.T) {
		repo := &fakeRBACRepo{listBindingsErr: errors.New("list bindings failed")}
		svc := NewService(repo)

		_, err := svc.CheckPermission(context.Background(), CheckPermissionInput{SubjectType: domainrbac.SubjectUser, SubjectID: "u-1", Resource: "user:profile", Action: "read"})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "list bindings failed") {
			t.Fatalf("expected list bindings error, got %v", err)
		}
	})

	t.Run("check permission list policy rules error", func(t *testing.T) {
		binding, err := domainrbac.NewBinding(shared.ID("b-1"), domainrbac.SubjectUser, shared.ID("u-1"), shared.ID("r-1"), domainrbac.DataScopeAll, nowForTest())
		if err != nil {
			t.Fatalf("NewBinding error: %v", err)
		}
		repo := &fakeRBACRepo{bindings: []*domainrbac.Binding{binding}, listPolicyRulesErr: errors.New("list rules failed")}
		svc := NewService(repo)

		_, err = svc.CheckPermission(context.Background(), CheckPermissionInput{SubjectType: domainrbac.SubjectUser, SubjectID: "u-1", Resource: "user:profile", Action: "read"})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "list rules failed") {
			t.Fatalf("expected list rules error, got %v", err)
		}
	})

	t.Run("set role policies replace error", func(t *testing.T) {
		repo := &fakeRBACRepo{replacePolicyRulesErr: errors.New("replace failed")}
		svc := NewService(repo)

		err := svc.SetRolePolicies(context.Background(), SetRolePoliciesInput{RoleID: "r-1", Rules: []domainrbac.PolicyRule{{Resource: "user:*", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeAll}}})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "replace failed") {
			t.Fatalf("expected replace error, got %v", err)
		}
	})
}

func nowForTest() time.Time {
	return time.Unix(1700000000, 0).UTC()
}
