package hostservice

import (
	"context"
	"testing"
	"time"

	domainorg "github.com/tinboxw/skoll/internal/domain/organization"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

type staticTrustedScopeResolver struct {
	decision rbacsvc.DataScopeDecision
	input    rbacsvc.ResolveDataScopeInput
}

func (r *staticTrustedScopeResolver) ResolveDataScope(_ context.Context, in rbacsvc.ResolveDataScopeInput) (rbacsvc.DataScopeDecision, error) {
	r.input = in
	return r.decision, nil
}

func TestDataScopeServiceUsesTrustedIdentityAndTenantTree(t *testing.T) {
	organizations := memory.NewOrganizationStore()
	saveDepartment(t, organizations, "tenant-a", "", "tenant.a")
	saveDepartment(t, organizations, "org-a", "tenant-a", "tenant.a.sales")
	saveDepartment(t, organizations, "org-a-child", "org-a", "tenant.a.sales.child")
	saveDepartment(t, organizations, "tenant-b", "", "tenant.b")
	resolver := &staticTrustedScopeResolver{decision: rbacsvc.DataScopeDecision{Scope: domainrbac.DataScopeAll, All: true}}
	service, err := NewDataScopeService(resolver, organizations)
	if err != nil {
		t.Fatalf("NewDataScopeService error: %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{
		Subject: "employee-1", OrganizationID: "org-a", OrganizationPath: []string{"tenant-a", "org-a"}, Role: "manager", Roles: []string{"manager"},
	})
	predicate, err := service.Resolve(ctx, pluginsdk.Permission{Resource: "pharma_oa.customer", Action: "read"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if resolver.input.Resource != "pharma_oa.customer" || resolver.input.Action != "read" {
		t.Fatalf("unexpected RBAC input: %+v", resolver.input)
	}
	if predicate.AllTenants() || predicate.AllOrganizations() || !predicate.AllOwners() {
		t.Fatalf("non-super all scope escaped tenant boundary: %+v", predicate)
	}
	if !predicate.Allows(pluginsdk.ScopedRecord{TenantID: "tenant-a", OwnerID: "any", OrganizationID: "org-a-child"}) {
		t.Fatal("tenant descendant should be authorized")
	}
	forged := predicate.Constrain(pluginsdk.ScopeFilter{TenantIDs: []string{"tenant-b"}, OrganizationIDs: []string{"tenant-b"}})
	if !forged.Denied() || forged.Allows(pluginsdk.ScopedRecord{TenantID: "tenant-b", OwnerID: "employee-1", OrganizationID: "tenant-b"}) {
		t.Fatal("request values broadened trusted tenant scope")
	}
}

func TestDataScopeServicePinsSelfScopeToAuthenticatedSubject(t *testing.T) {
	resolver := &staticTrustedScopeResolver{decision: rbacsvc.DataScopeDecision{Scope: domainrbac.DataScopeSelf, UserIDs: []string{"request-user"}}}
	service, err := NewDataScopeService(resolver, memory.NewOrganizationStore())
	if err != nil {
		t.Fatalf("NewDataScopeService error: %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{
		Subject: "trusted-user", OrganizationID: "tenant-a", Role: "sales", Roles: []string{"sales"},
	})
	predicate, err := service.Resolve(ctx, pluginsdk.Permission{Resource: "customer", Action: "update"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if owners := predicate.OwnerIDs(); len(owners) != 1 || owners[0] != "trusted-user" {
		t.Fatalf("self scope trusted request-provided user IDs: %+v", owners)
	}
	if !predicate.Constrain(pluginsdk.ScopeFilter{OwnerIDs: []string{"attacker"}}).Denied() {
		t.Fatal("forged owner scope was not denied")
	}
}

func TestDataScopeServiceRejectsMissingTrustedIdentityOrTenant(t *testing.T) {
	resolver := &staticTrustedScopeResolver{decision: rbacsvc.DataScopeDecision{Scope: domainrbac.DataScopeSelf}}
	service, err := NewDataScopeService(resolver, memory.NewOrganizationStore())
	if err != nil {
		t.Fatalf("NewDataScopeService error: %v", err)
	}

	if _, err := service.Resolve(context.Background(), pluginsdk.Permission{Resource: "customer", Action: "read"}); err == nil {
		t.Fatal("missing trusted identity must be rejected")
	}
	identityOnly := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "user-without-tenant"})
	if _, err := service.Resolve(identityOnly, pluginsdk.Permission{Resource: "customer", Action: "read"}); err == nil {
		t.Fatal("non-super identity without trusted tenant must be rejected")
	}
}

func TestDataScopeServiceAllowsGlobalScopeOnlyForSuperAdmin(t *testing.T) {
	resolver := &staticTrustedScopeResolver{decision: rbacsvc.DataScopeDecision{Scope: domainrbac.DataScopeSelf}}
	service, err := NewDataScopeService(resolver, memory.NewOrganizationStore())
	if err != nil {
		t.Fatalf("NewDataScopeService error: %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{
		Subject: "root", Role: "super_admin", Roles: []string{"super_admin"},
	})
	predicate, err := service.Resolve(ctx, pluginsdk.Permission{Resource: "customer", Action: "read"})
	if err != nil {
		t.Fatalf("Resolve super admin scope: %v", err)
	}
	if !predicate.AllTenants() || !predicate.AllOwners() || !predicate.AllOrganizations() {
		t.Fatalf("super admin scope is not global: %+v", predicate)
	}
}

func saveDepartment(t *testing.T, store *memory.OrganizationStore, id, parentID, code string) {
	t.Helper()
	now := time.Date(2026, 7, 22, 23, 0, 0, 0, time.UTC)
	item, err := domainorg.NewDepartment(domainorg.DepartmentInput{
		ID: shared.ID(id), ParentID: shared.ID(parentID), Code: code, Name: id, Status: domainorg.StatusEnabled, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("NewDepartment error: %v", err)
	}
	if err := store.SaveDepartment(context.Background(), item); err != nil {
		t.Fatalf("SaveDepartment error: %v", err)
	}
}
