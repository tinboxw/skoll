package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainorganization "github.com/tinboxw/skoll/internal/domain/organization"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	pharmahttp "github.com/tinboxw/skoll/internal/handler/http/v1/pharmaoa"
	rbachttp "github.com/tinboxw/skoll/internal/handler/http/v1/rbac"
	"github.com/tinboxw/skoll/internal/plugin/hostservice"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestPharmaCustomerOrganizationScopeMatrix(t *testing.T) {
	organizations := memory.NewOrganizationStore()
	seedScopeOrganizations(t, organizations)
	rbacService := rbacsvc.NewServiceWithOrganization(memory.NewRBACStore(), organizations)
	dataScopes, err := hostservice.NewDataScopeService(rbacService, organizations)
	if err != nil {
		t.Fatalf("build host data-scope service: %v", err)
	}
	auditService := auditsvc.NewService(clickhouse.NewAuditStore())
	customerService := pharmaoasvc.NewCustomerService(auditService)
	mux := http.NewServeMux()
	rbachttp.RegisterRBACRoutes(mux, rbacService, auditService)
	pharmahttp.RegisterCustomerRoutes(mux, customerService, dataScopes)

	customers := seedScopeCustomers(t, customerService)
	identities := []struct {
		name         string
		claims       security.JWTClaims
		scope        domainrbac.DataScope
		wantCodes    []string
		rejectCodes  []string
		withoutGrant bool
	}{
		{name: "self", claims: scopeClaims("scope-self", "org-sales", "employee"), scope: domainrbac.DataScopeSelf, wantCodes: []string{"SCOPE-SELF"}, rejectCodes: []string{"SCOPE-DEPT", "SCOPE-CHILD", "SCOPE-OTHER"}},
		{name: "department", claims: scopeClaims("scope-department", "org-sales", "employee"), scope: domainrbac.DataScopeDepartment, wantCodes: []string{"SCOPE-SELF", "SCOPE-DEPT"}, rejectCodes: []string{"SCOPE-CHILD", "SCOPE-OTHER"}},
		{name: "department_tree", claims: scopeClaims("scope-tree", "org-sales", "employee"), scope: domainrbac.DataScopeDepartmentTree, wantCodes: []string{"SCOPE-SELF", "SCOPE-DEPT", "SCOPE-CHILD"}, rejectCodes: []string{"SCOPE-OTHER"}},
		{name: "all", claims: scopeClaims("scope-all", "org-root", "employee"), scope: domainrbac.DataScopeAll, wantCodes: []string{"SCOPE-SELF", "SCOPE-DEPT", "SCOPE-CHILD", "SCOPE-OTHER"}},
		{name: "denied", claims: scopeClaims("scope-denied", "org-sales", "employee"), withoutGrant: true},
		{name: "super_admin", claims: scopeClaims("scope-super", "org-root", "super_admin"), scope: domainrbac.DataScopeAll, wantCodes: []string{"SCOPE-SELF", "SCOPE-DEPT", "SCOPE-CHILD", "SCOPE-OTHER"}},
	}

	for _, identity := range identities {
		identity := identity
		t.Run(identity.name, func(t *testing.T) {
			if !identity.withoutGrant && identity.name != "super_admin" {
				grantCustomerScope(t, rbacService, identity.claims.Subject, identity.scope)
			}

			scopeResponse := performScopeMatrixRequest(mux, identity.claims, http.MethodGet, "/v1/rbac/data-scope?resource=pharma_oa.customer&action=read", nil)
			listResponse := performScopeMatrixRequest(mux, identity.claims, http.MethodGet, "/v1/plugins/pharma_oa/api/customers?includeAll=true&organizationId=org-other&ownerId=scope-all", nil)
			if identity.withoutGrant {
				if scopeResponse.Code != http.StatusForbidden || listResponse.Code != http.StatusForbidden {
					t.Fatalf("denied identity statuses scope=%d list=%d", scopeResponse.Code, listResponse.Code)
				}
				return
			}
			if scopeResponse.Code != http.StatusOK || !strings.Contains(scopeResponse.Body.String(), `"scope":"`+string(identity.scope)+`"`) {
				t.Fatalf("scope response status=%d body=%s", scopeResponse.Code, scopeResponse.Body.String())
			}
			if listResponse.Code != http.StatusOK {
				t.Fatalf("list response status=%d body=%s", listResponse.Code, listResponse.Body.String())
			}
			for _, code := range identity.wantCodes {
				if !strings.Contains(listResponse.Body.String(), code) {
					t.Errorf("scope %s missing customer %s: %s", identity.name, code, listResponse.Body.String())
				}
			}
			for _, code := range identity.rejectCodes {
				if strings.Contains(listResponse.Body.String(), code) {
					t.Errorf("scope %s leaked customer %s: %s", identity.name, code, listResponse.Body.String())
				}
			}
		})
	}

	assertScopeWritesAndDeniedAudit(t, mux, auditService, customers)
}

func seedScopeOrganizations(t *testing.T, store *memory.OrganizationStore) {
	t.Helper()
	now := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	for _, input := range []domainorganization.DepartmentInput{
		{ID: "org-root", Code: "scope.root", Name: "Scope Root", CreatedAt: now},
		{ID: "org-sales", ParentID: "org-root", Code: "scope.sales", Name: "Sales", CreatedAt: now},
		{ID: "org-east", ParentID: "org-sales", Code: "scope.sales.east", Name: "Sales East", CreatedAt: now},
		{ID: "org-other", ParentID: "org-root", Code: "scope.other", Name: "Other", CreatedAt: now},
	} {
		item, err := domainorganization.NewDepartment(input)
		if err != nil {
			t.Fatalf("build organization %s: %v", input.ID, err)
		}
		if err := store.SaveDepartment(context.Background(), item); err != nil {
			t.Fatalf("save organization %s: %v", input.ID, err)
		}
	}
}

func seedScopeCustomers(t *testing.T, service pharmaoasvc.CustomerService) map[string]string {
	t.Helper()
	fixtures := []pharmaoasvc.CustomerWriteInput{
		{Code: "SCOPE-SELF", Name: "Self Customer", Region: "East", OrganizationID: "org-sales", OwnerID: "scope-self"},
		{Code: "SCOPE-DEPT", Name: "Department Customer", Region: "East", OrganizationID: "org-sales", OwnerID: "scope-department"},
		{Code: "SCOPE-CHILD", Name: "Child Customer", Region: "East", OrganizationID: "org-east", OwnerID: "scope-child"},
		{Code: "SCOPE-OTHER", Name: "Other Customer", Region: "West", OrganizationID: "org-other", OwnerID: "scope-all"},
	}
	ids := make(map[string]string, len(fixtures))
	for _, fixture := range fixtures {
		fixture.ActorID = "scope-fixture"
		fixture.Scope = pharmaoasvc.CustomerAccessScope{IncludeAll: true}
		item, err := service.Create(context.Background(), fixture)
		if err != nil {
			t.Fatalf("seed customer %s: %v", fixture.Code, err)
		}
		ids[fixture.Code] = item.ID.String()
	}
	return ids
}

func grantCustomerScope(t *testing.T, service rbacsvc.Service, userID string, scope domainrbac.DataScope) {
	t.Helper()
	roleID := "scope-role-" + userID
	if _, err := service.BindRole(context.Background(), rbacsvc.BindRoleInput{SubjectType: domainrbac.SubjectUser, SubjectID: userID, RoleID: roleID, Scope: scope}); err != nil {
		t.Fatalf("bind %s scope: %v", userID, err)
	}
	rules := make([]domainrbac.PolicyRule, 0, 6)
	for _, action := range []string{"read", "create", "update", "disable", "sales", "reminder"} {
		rules = append(rules, domainrbac.PolicyRule{Resource: "pharma_oa.customer", Action: action, Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeAll})
	}
	if err := service.SetRolePolicies(context.Background(), rbacsvc.SetRolePoliciesInput{RoleID: roleID, Rules: rules}); err != nil {
		t.Fatalf("set %s policies: %v", userID, err)
	}
}

func scopeClaims(subject, organizationID, role string) security.JWTClaims {
	return security.JWTClaims{Subject: subject, OrganizationID: organizationID, OrganizationPath: []string{"org-root", organizationID}, Role: role, Roles: []string{role}}
}

func assertScopeWritesAndDeniedAudit(t *testing.T, mux http.Handler, auditService auditsvc.Service, customers map[string]string) {
	t.Helper()
	selfClaims := scopeClaims("scope-self", "org-sales", "employee")
	allowed := performScopeMatrixRequest(mux, selfClaims, http.MethodPost, "/v1/plugins/pharma_oa/api/customers", map[string]any{
		"code": "SCOPE-SELF-CREATE", "name": "Self Created", "region": "East", "organizationId": "org-other", "ownerId": "scope-self",
	})
	if allowed.Code != http.StatusCreated {
		t.Fatalf("self create status=%d body=%s", allowed.Code, allowed.Body.String())
	}

	treeClaims := scopeClaims("scope-tree", "org-sales", "employee")
	deniedUpdate := performScopeMatrixRequest(mux, treeClaims, http.MethodPut, "/v1/plugins/pharma_oa/api/customers/"+customers["SCOPE-OTHER"], map[string]any{
		"code": "SCOPE-OTHER", "name": "Forbidden Update", "region": "West", "organizationId": "org-other", "ownerId": "scope-all",
	})
	if deniedUpdate.Code != http.StatusForbidden {
		t.Fatalf("cross-organization update status=%d body=%s", deniedUpdate.Code, deniedUpdate.Body.String())
	}
	deniedRead := performScopeMatrixRequest(mux, treeClaims, http.MethodGet, "/v1/plugins/pharma_oa/api/customers/"+customers["SCOPE-OTHER"]+"/sales-eligibility", nil)
	if deniedRead.Code != http.StatusForbidden {
		t.Fatalf("cross-organization sales read status=%d body=%s", deniedRead.Code, deniedRead.Body.String())
	}

	records, err := auditService.ListByActor(context.Background(), "scope-tree", 20)
	if err != nil {
		t.Fatalf("list denied audit: %v", err)
	}
	wantActions := map[string]bool{
		"pharma_oa.customer.update.denied": false,
		"pharma_oa.customer.sales.denied":  false,
	}
	for _, record := range records {
		if _, ok := wantActions[record.Action]; !ok {
			continue
		}
		if record.Detail["result"] != "denied" || record.Detail["reason"] != "data_scope" || record.Detail["organizationId"] != "org-other" {
			t.Errorf("incomplete denied audit: %+v", record)
		}
		wantActions[record.Action] = true
	}
	for action, found := range wantActions {
		if !found {
			t.Errorf("missing denied audit action %s in %+v", action, records)
		}
	}
}

func performScopeMatrixRequest(handler http.Handler, claims security.JWTClaims, method, path string, body any) *httptest.ResponseRecorder {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &claims))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
