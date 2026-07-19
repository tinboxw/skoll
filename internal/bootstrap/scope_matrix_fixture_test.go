package bootstrap

import (
	"context"
	"errors"
	"strconv"
	"testing"

	domainrole "github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/pkg/security"
)

type remappingScopeUserRepository struct {
	userrepo.UserRepository
	next int
}

func (r *remappingScopeUserRepository) Save(ctx context.Context, entity *domainuser.User) error {
	if _, err := strconv.Atoi(entity.ID.String()); err != nil {
		r.next++
		entity.ID = shared.ID(strconv.Itoa(r.next))
	}
	return r.UserRepository.Save(ctx, entity)
}

func TestScopeMatrixFixturesAreExplicitAndIdempotent(t *testing.T) {
	t.Setenv("SKOLL_SCOPE_MATRIX_FIXTURES", "true")
	if !scopeMatrixFixturesEnabled() {
		t.Fatal("scope matrix fixtures should be enabled explicitly")
	}
	ctx := context.Background()
	users := memory.NewUserStore()
	roles := memory.NewRoleStore()
	rbac := memory.NewRBACStore()
	organizations := memory.NewOrganizationStore()
	customers := pharmaoasvc.NewCustomerService(nil)
	for attempt := 0; attempt < 2; attempt++ {
		if err := ensureScopeMatrixFixtures(ctx, users, roles, rbac, organizations, customers); err != nil {
			t.Fatalf("ensure scope fixtures attempt %d: %v", attempt+1, err)
		}
	}

	rbacService := rbacsvc.NewServiceWithOrganization(rbac, organizations)
	for _, fixture := range scopeMatrixFixtures {
		user, err := users.GetByAccount(ctx, fixture.key)
		if err != nil || user == nil || user.DepartmentID != fixture.organizationID {
			t.Fatalf("scope user %s: %+v err=%v", fixture.key, user, err)
		}
		role, err := roles.GetByKey(ctx, fixture.key)
		if err != nil || role == nil {
			t.Fatalf("scope role %s: %+v err=%v", fixture.key, role, err)
		}
		assertScopeFixtureBinding(t, rbacService, user.ID.String(), role)

		claims := security.JWTClaims{Subject: user.ID.String(), OrganizationID: user.DepartmentID, OrganizationPath: []string{"scope-org-root", user.DepartmentID}, Role: fixture.key, Roles: []string{fixture.key}}
		requestContext := security.WithJWTClaimsContext(ctx, &claims)
		decision, resolveErr := rbacService.ResolveDataScope(requestContext, rbacsvc.ResolveDataScopeInput{Resource: "pharma_oa.customer", Action: "read"})
		if !fixture.grantCustomer {
			if !errors.Is(resolveErr, rbacsvc.ErrDataScopeDenied) {
				t.Fatalf("denied fixture resolved decision=%+v err=%v", decision, resolveErr)
			}
			continue
		}
		if resolveErr != nil || decision.Scope != fixture.scope {
			t.Fatalf("scope fixture %s decision=%+v err=%v", fixture.key, decision, resolveErr)
		}
	}

	items, err := customers.List(ctx, pharmaoasvc.CustomerListInput{Scope: pharmaoasvc.CustomerAccessScope{IncludeAll: true}})
	if err != nil || len(items) != 4 {
		t.Fatalf("scope fixture customers=%d err=%v", len(items), err)
	}
}

func assertScopeFixtureBinding(t *testing.T, service rbacsvc.Service, userID string, role *domainrole.Role) {
	t.Helper()
	bindings, err := service.ListBindingsByUser(context.Background(), userID)
	if err != nil || len(bindings) != 1 || bindings[0].RoleID != role.ID {
		t.Fatalf("scope fixture binding user=%s bindings=%+v err=%v", userID, bindings, err)
	}
}

func TestScopeMatrixFixturesDisabledByDefault(t *testing.T) {
	t.Setenv("SKOLL_SCOPE_MATRIX_FIXTURES", "")
	if scopeMatrixFixturesEnabled() {
		t.Fatal("scope matrix fixtures must be disabled by default")
	}
}

func TestScopeMatrixCustomerOwnersUsePersistedUserIDs(t *testing.T) {
	users := &remappingScopeUserRepository{UserRepository: memory.NewUserStore(), next: 100}
	customers := pharmaoasvc.NewCustomerService(nil)
	if err := ensureScopeMatrixFixtures(context.Background(), users, memory.NewRoleStore(), memory.NewRBACStore(), memory.NewOrganizationStore(), customers); err != nil {
		t.Fatalf("ensure scope fixtures: %v", err)
	}
	self, err := users.GetByAccount(context.Background(), "scope_self")
	if err != nil || self == nil {
		t.Fatalf("load remapped self user: %+v err=%v", self, err)
	}
	items, err := customers.List(context.Background(), pharmaoasvc.CustomerListInput{Scope: pharmaoasvc.CustomerAccessScope{OwnerID: self.ID.String()}})
	if err != nil || len(items) != 1 || items[0].Code != "SCOPE-SELF" {
		t.Fatalf("self customer with persisted owner id: %+v err=%v", items, err)
	}
}
