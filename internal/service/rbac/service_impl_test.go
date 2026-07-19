package rbac

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domainorganization "github.com/tinboxw/skoll/internal/domain/organization"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/pkg/security"
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

func (f *fakeRBACRepo) ReplacePolicyRules(_ context.Context, roleID shared.ID, rules []domainrbac.PolicyRule) error {
	f.replacePolicyRulesCall = true
	if f.policyRulesByRole == nil {
		f.policyRulesByRole = map[string][]domainrbac.PolicyRule{}
	}
	f.policyRulesByRole[roleID.String()] = append([]domainrbac.PolicyRule(nil), rules...)
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

	decision, err := svc.ResolvePermission(context.Background(), CheckPermissionInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-1",
		Resource:    "user:profile",
		Action:      "read",
	})
	if err != nil {
		t.Fatalf("ResolvePermission error: %v", err)
	}
	if !decision.Allowed || decision.Scope != domainrbac.DataScopeAll {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestRBACServiceResolvePermissionUsesMostRestrictiveScope(t *testing.T) {
	repo := memory.NewRBACStore()
	svc := NewService(repo)

	_, err := svc.BindRole(context.Background(), BindRoleInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-scope",
		RoleID:      "r-scope",
		Scope:       domainrbac.DataScopeDeptTree,
	})
	if err != nil {
		t.Fatalf("BindRole error: %v", err)
	}

	err = svc.SetRolePolicies(context.Background(), SetRolePoliciesInput{
		RoleID: "r-scope",
		Rules: []domainrbac.PolicyRule{
			{Resource: "user:*", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeDept},
			{Resource: "user:*", Action: "update", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeAll},
		},
	})
	if err != nil {
		t.Fatalf("SetRolePolicies error: %v", err)
	}

	readDecision, err := svc.ResolvePermission(context.Background(), CheckPermissionInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-scope",
		Resource:    "user:profile",
		Action:      "read",
	})
	if err != nil {
		t.Fatalf("ResolvePermission read error: %v", err)
	}
	if !readDecision.Allowed || readDecision.Scope != domainrbac.DataScopeDepartment {
		t.Fatalf("expected dept scope, got %+v", readDecision)
	}

	updateDecision, err := svc.ResolvePermission(context.Background(), CheckPermissionInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   "u-scope",
		Resource:    "user:profile",
		Action:      "update",
	})
	if err != nil {
		t.Fatalf("ResolvePermission update error: %v", err)
	}
	if !updateDecision.Allowed || updateDecision.Scope != domainrbac.DataScopeDepartmentTree {
		t.Fatalf("expected dept_tree scope, got %+v", updateDecision)
	}
}

func TestRBACServiceSetRolePoliciesNormalizesDataScope(t *testing.T) {
	repo := &fakeRBACRepo{}
	svc := NewService(repo)

	err := svc.SetRolePolicies(context.Background(), SetRolePoliciesInput{
		RoleID: "r-1",
		Rules:  []domainrbac.PolicyRule{{Resource: " user:* ", Action: " read ", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScope("dept_tree")}},
	})
	if err != nil {
		t.Fatalf("SetRolePolicies error: %v", err)
	}
	rules := repo.policyRulesByRole["r-1"]
	if len(rules) != 1 || rules[0].Scope != domainrbac.DataScopeDepartmentTree || rules[0].Resource != "user:*" || rules[0].Action != "read" {
		t.Fatalf("expected normalized rules, got %+v", rules)
	}
}

func TestRBACServiceResolveDataScope(t *testing.T) {
	rbacRepo := memory.NewRBACStore()
	organizationRepo := memory.NewOrganizationStore()
	now := nowForTest()
	for _, input := range []domainorganization.DepartmentInput{
		{ID: "org-root", Code: "root", Name: "Root", CreatedAt: now},
		{ID: "org-sales", ParentID: "org-root", Code: "sales", Name: "Sales", CreatedAt: now},
		{ID: "org-east", ParentID: "org-sales", Code: "sales.east", Name: "East", CreatedAt: now},
		{ID: "org-other", ParentID: "org-root", Code: "other", Name: "Other", CreatedAt: now},
	} {
		item, err := domainorganization.NewDepartment(input)
		if err != nil {
			t.Fatalf("NewDepartment error: %v", err)
		}
		if err := organizationRepo.SaveDepartment(context.Background(), item); err != nil {
			t.Fatalf("SaveDepartment error: %v", err)
		}
	}
	svc := NewServiceWithOrganization(rbacRepo, organizationRepo)

	for _, test := range []struct {
		name              string
		userID            string
		scope             domainrbac.DataScope
		organizationID    string
		wantAll           bool
		wantUsers         string
		wantOrganizations string
	}{
		{name: "self", userID: "user-self", scope: domainrbac.DataScopeSelf, organizationID: "org-sales", wantUsers: "user-self"},
		{name: "organization", userID: "user-org", scope: domainrbac.DataScopeDepartment, organizationID: "org-sales", wantOrganizations: "org-sales"},
		{name: "organization tree", userID: "user-tree", scope: domainrbac.DataScopeDepartmentTree, organizationID: "org-sales", wantOrganizations: "org-sales,org-east"},
		{name: "all", userID: "user-all", scope: domainrbac.DataScopeAll, organizationID: "org-sales", wantAll: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			roleID := "role-" + test.userID
			if _, err := svc.BindRole(context.Background(), BindRoleInput{SubjectType: domainrbac.SubjectUser, SubjectID: test.userID, RoleID: roleID, Scope: test.scope}); err != nil {
				t.Fatalf("BindRole error: %v", err)
			}
			if err := svc.SetRolePolicies(context.Background(), SetRolePoliciesInput{RoleID: roleID, Rules: []domainrbac.PolicyRule{{Resource: "pharma_oa.customer", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeAll}}}); err != nil {
				t.Fatalf("SetRolePolicies error: %v", err)
			}
			ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: test.userID, OrganizationID: test.organizationID, OrganizationPath: []string{"org-root", test.organizationID}, Role: "employee", Roles: []string{"employee"}})
			got, err := svc.ResolveDataScope(ctx, ResolveDataScopeInput{Resource: "pharma_oa.customer", Action: "read"})
			if err != nil {
				t.Fatalf("ResolveDataScope error: %v", err)
			}
			if got.All != test.wantAll || strings.Join(got.UserIDs, ",") != test.wantUsers || strings.Join(got.DepartmentIDs, ",") != test.wantOrganizations {
				t.Fatalf("unexpected decision: %+v", got)
			}
		})
	}
}

func TestRBACServiceResolveDataScopeRejectsMissingContext(t *testing.T) {
	svc := NewService(memory.NewRBACStore())
	if _, err := svc.ResolveDataScope(context.Background(), ResolveDataScopeInput{Resource: "user", Action: "read"}); !errors.Is(err, ErrDataScopeDenied) {
		t.Fatalf("expected trusted identity denial, got %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "root", Role: "super_admin", Roles: []string{"super_admin"}})
	decision, err := svc.ResolveDataScope(ctx, ResolveDataScopeInput{})
	if err != nil || !decision.All {
		t.Fatalf("expected explicit super-admin all scope, decision=%+v err=%v", decision, err)
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
