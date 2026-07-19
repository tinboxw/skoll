package bootstrap

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	domainorganization "github.com/tinboxw/skoll/internal/domain/organization"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	domainrole "github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	organizationrepo "github.com/tinboxw/skoll/internal/repository/organization"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	rolerepo "github.com/tinboxw/skoll/internal/repository/role"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const scopeMatrixFixturePassword = "Scope@123456"

type scopeMatrixFixture struct {
	key            string
	scope          domainrbac.DataScope
	organizationID string
	grantCustomer  bool
}

var scopeMatrixFixtures = []scopeMatrixFixture{
	{key: "scope_self", scope: domainrbac.DataScopeSelf, organizationID: "scope-org-sales", grantCustomer: true},
	{key: "scope_department", scope: domainrbac.DataScopeDepartment, organizationID: "scope-org-sales", grantCustomer: true},
	{key: "scope_tree", scope: domainrbac.DataScopeDepartmentTree, organizationID: "scope-org-sales", grantCustomer: true},
	{key: "scope_all", scope: domainrbac.DataScopeAll, organizationID: "scope-org-root", grantCustomer: true},
	{key: "scope_denied", scope: domainrbac.DataScopeSelf, organizationID: "scope-org-sales", grantCustomer: false},
}

func scopeMatrixFixturesEnabled() bool {
	enabled, _ := strconv.ParseBool(strings.TrimSpace(os.Getenv("SKOLL_SCOPE_MATRIX_FIXTURES")))
	return enabled
}

func ensureScopeMatrixFixtures(
	ctx context.Context,
	users userrepo.UserRepository,
	roles rolerepo.RoleRepository,
	rbac rbacrepo.RBACRepository,
	organizations organizationrepo.OrganizationRepository,
	customers pharmaoasvc.CustomerService,
) error {
	if users == nil || roles == nil || rbac == nil || organizations == nil || customers == nil {
		return fmt.Errorf("scope matrix fixture dependencies are required")
	}
	now := time.Now().UTC()
	if err := ensureScopeMatrixOrganizations(ctx, organizations, now); err != nil {
		return err
	}
	for _, fixture := range scopeMatrixFixtures {
		if err := ensureScopeMatrixIdentity(ctx, users, roles, rbac, fixture, now); err != nil {
			return err
		}
	}
	return ensureScopeMatrixCustomers(ctx, users, customers)
}

func ensureScopeMatrixOrganizations(ctx context.Context, repo organizationrepo.OrganizationRepository, now time.Time) error {
	for _, input := range []domainorganization.DepartmentInput{
		{ID: "scope-org-root", Code: "scope.root", Name: "Scope Root", CreatedAt: now},
		{ID: "scope-org-sales", ParentID: "scope-org-root", Code: "scope.sales", Name: "Scope Sales", CreatedAt: now},
		{ID: "scope-org-east", ParentID: "scope-org-sales", Code: "scope.sales.east", Name: "Scope Sales East", CreatedAt: now},
		{ID: "scope-org-other", ParentID: "scope-org-root", Code: "scope.other", Name: "Scope Other", CreatedAt: now},
	} {
		current, err := repo.GetDepartmentByID(ctx, input.ID)
		if err != nil {
			return fmt.Errorf("load scope organization %s: %w", input.ID, err)
		}
		if current != nil {
			continue
		}
		item, err := domainorganization.NewDepartment(input)
		if err != nil {
			return fmt.Errorf("build scope organization %s: %w", input.ID, err)
		}
		if err := repo.SaveDepartment(ctx, item); err != nil {
			return fmt.Errorf("save scope organization %s: %w", input.ID, err)
		}
	}
	return nil
}

func ensureScopeMatrixIdentity(ctx context.Context, users userrepo.UserRepository, roles rolerepo.RoleRepository, rbac rbacrepo.RBACRepository, fixture scopeMatrixFixture, now time.Time) error {
	roleID := shared.ID("scope-role-" + fixture.key)
	userID := shared.ID("scope-user-" + fixture.key)
	permissions := []string{"user.read"}
	rules := []domainrbac.PolicyRule{{Resource: "user", Action: "read", Effect: domainrbac.EffectAllow, Scope: fixture.scope}}
	if fixture.grantCustomer {
		for _, action := range []string{"read", "create", "update", "disable", "sales", "reminder"} {
			permissions = append(permissions, "pharma_oa.customer."+action)
			rules = append(rules, domainrbac.PolicyRule{Resource: "pharma_oa.customer", Action: action, Effect: domainrbac.EffectAllow, Scope: fixture.scope})
		}
	}

	role, err := roles.GetByKey(ctx, fixture.key)
	if err != nil {
		return fmt.Errorf("load scope role %s: %w", fixture.key, err)
	}
	if role == nil {
		role, err = domainrole.New(roleID, scopeFixtureDisplayName(fixture.key), fixture.key, "H3 organization scope matrix fixture", permissions, false, now)
		if err != nil {
			return fmt.Errorf("build scope role %s: %w", fixture.key, err)
		}
		if err := roles.Save(ctx, role); err != nil {
			return fmt.Errorf("save scope role %s: %w", fixture.key, err)
		}
	}
	if err := rbac.ReplacePolicyRules(ctx, role.ID, rules); err != nil {
		return fmt.Errorf("save scope policies %s: %w", fixture.key, err)
	}

	user, err := users.GetByAccount(ctx, fixture.key)
	if err != nil {
		return fmt.Errorf("load scope user %s: %w", fixture.key, err)
	}
	if user == nil {
		user, err = domainuser.New(userID, fixture.key, scopeFixtureDisplayName(fixture.key), fixture.key+"@skoll.local", now)
		if err != nil {
			return fmt.Errorf("build scope user %s: %w", fixture.key, err)
		}
	}
	hash, err := domainuser.HashPassword(scopeMatrixFixturePassword)
	if err != nil {
		return fmt.Errorf("hash scope password %s: %w", fixture.key, err)
	}
	if err := user.SetPasswordHash(hash.String()); err != nil {
		return fmt.Errorf("set scope password %s: %w", fixture.key, err)
	}
	user.SetOrganization(fixture.organizationID, "", now)
	user.Activate(now)
	if err := users.Save(ctx, user); err != nil {
		return fmt.Errorf("save scope user %s: %w", fixture.key, err)
	}

	bindings, err := rbac.ListBindingsBySubject(ctx, domainrbac.SubjectUser, user.ID)
	if err != nil {
		return fmt.Errorf("load scope binding %s: %w", fixture.key, err)
	}
	for _, binding := range bindings {
		if binding.RoleID == role.ID {
			return nil
		}
	}
	binding, err := domainrbac.NewBinding(shared.ID("scope-binding-"+fixture.key), domainrbac.SubjectUser, user.ID, role.ID, domainrbac.DataScopeAll, now)
	if err != nil {
		return fmt.Errorf("build scope binding %s: %w", fixture.key, err)
	}
	if err := rbac.CreateBinding(ctx, binding); err != nil {
		return fmt.Errorf("save scope binding %s: %w", fixture.key, err)
	}
	return nil
}

func ensureScopeMatrixCustomers(ctx context.Context, users userrepo.UserRepository, service pharmaoasvc.CustomerService) error {
	items, err := service.List(ctx, pharmaoasvc.CustomerListInput{Scope: pharmaoasvc.CustomerAccessScope{IncludeAll: true}})
	if err != nil {
		return fmt.Errorf("list scope customers: %w", err)
	}
	existing := make(map[string]struct{}, len(items))
	for _, item := range items {
		existing[item.Code] = struct{}{}
	}
	ownerIDs := make(map[string]string, 4)
	for _, account := range []string{"scope_self", "scope_department", "scope_tree", "scope_all"} {
		user, loadErr := users.GetByAccount(ctx, account)
		if loadErr != nil {
			return fmt.Errorf("load scope customer owner %s: %w", account, loadErr)
		}
		if user == nil || user.ID.IsZero() {
			return fmt.Errorf("scope customer owner %s is missing", account)
		}
		ownerIDs[account] = user.ID.String()
	}
	for _, input := range []pharmaoasvc.CustomerWriteInput{
		{Code: "SCOPE-SELF", Name: "Self Customer", Region: "East", OrganizationID: "scope-org-sales", OwnerID: ownerIDs["scope_self"]},
		{Code: "SCOPE-DEPT", Name: "Department Customer", Region: "East", OrganizationID: "scope-org-sales", OwnerID: ownerIDs["scope_department"]},
		{Code: "SCOPE-CHILD", Name: "Child Customer", Region: "East", OrganizationID: "scope-org-east", OwnerID: ownerIDs["scope_tree"]},
		{Code: "SCOPE-OTHER", Name: "Other Customer", Region: "West", OrganizationID: "scope-org-other", OwnerID: ownerIDs["scope_all"]},
	} {
		if _, ok := existing[input.Code]; ok {
			continue
		}
		input.ActorID = "scope-fixture"
		input.Scope = pharmaoasvc.CustomerAccessScope{IncludeAll: true}
		if _, err := service.Create(ctx, input); err != nil {
			return fmt.Errorf("create scope customer %s: %w", input.Code, err)
		}
	}
	return nil
}

func scopeFixtureDisplayName(key string) string {
	return strings.Title(strings.ReplaceAll(key, "_", " "))
}
