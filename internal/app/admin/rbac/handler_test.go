package rbac

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminapiregistry "github.com/tinboxw/skoll/internal/app/admin/apiregistry"
	adminmenu "github.com/tinboxw/skoll/internal/app/admin/menu"
	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/menu"
	rbacmod "github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/role"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func newRBACMux() (*http.ServeMux, *role.Service, *menu.Service, *rbacmod.Service, *apiregistry.Service, *audit.Service) {
	mux := http.NewServeMux()
	roleSvc := role.NewService()
	menuSvc := menu.NewService()
	rbacSvc := rbacmod.NewService()
	apiSvc := apiregistry.NewService()
	auditSvc := audit.NewService()
	NewHandler(roleSvc, menuSvc, rbacSvc, apiSvc, auditSvc).Register(mux, nil, apiSvc)
	// Also register menu handler so menu CRUD routes are available.
	adminmenu.NewHandler(menuSvc).Register(mux, nil, apiSvc)
	// Register apiregistry handler so /admin/v1/apis is available.
	adminapiregistry.NewHandler(apiSvc).Register(mux, nil)
	// Pre-register test user APIs that would normally come from the users handler.
	apiSvc.RegisterMany([]string{"GET:/admin/v1/users", "POST:/admin/v1/users"})
	// Also register apiregistry handler so /admin/v1/apis is available.
	return mux, roleSvc, menuSvc, rbacSvc, apiSvc, auditSvc
}

func TestAdminRoleAndMenuRoutes(t *testing.T) {
	mux, _, _, _, _, _ := newRBACMux()

	roleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read","menu.read"]}`))
	roleReq.Header.Set("Content-Type", "application/json")
	roleRR := httptest.NewRecorder()
	mux.ServeHTTP(roleRR, roleReq)
	if roleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", roleRR.Code)
	}

	menuReq := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"Dashboard","path":"/dashboard","order":1}`))
	menuReq.Header.Set("Content-Type", "application/json")
	menuRR := httptest.NewRecorder()
	mux.ServeHTTP(menuRR, menuReq)
	if menuRR.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", menuRR.Code)
	}

	roleGetReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1", nil)
	roleGetRR := httptest.NewRecorder()
	mux.ServeHTTP(roleGetRR, roleGetReq)
	if roleGetRR.Code != http.StatusOK {
		t.Fatalf("expected role get status 200, got %d", roleGetRR.Code)
	}

	menuListReq := httptest.NewRequest(http.MethodGet, "/admin/v1/menus", nil)
	menuListRR := httptest.NewRecorder()
	mux.ServeHTTP(menuListRR, menuListReq)
	if menuListRR.Code != http.StatusOK {
		t.Fatalf("expected menu list status 200, got %d", menuListRR.Code)
	}
}

func TestRoleBindingRoutes_MenuAndAPI(t *testing.T) {
	mux, _, _, _, _, _ := newRBACMux()

	createRoleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read"]}`))
	createRoleReq.Header.Set("Content-Type", "application/json")
	createRoleRR := httptest.NewRecorder()
	mux.ServeHTTP(createRoleRR, createRoleReq)
	if createRoleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", createRoleRR.Code)
	}

	createMenuReq1 := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"Dashboard","path":"/dashboard","order":1}`))
	createMenuReq1.Header.Set("Content-Type", "application/json")
	createMenuRR1 := httptest.NewRecorder()
	mux.ServeHTTP(createMenuRR1, createMenuReq1)
	if createMenuRR1.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", createMenuRR1.Code)
	}

	createMenuReq2 := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"System","path":"/system","order":2}`))
	createMenuReq2.Header.Set("Content-Type", "application/json")
	createMenuRR2 := httptest.NewRecorder()
	mux.ServeHTTP(createMenuRR2, createMenuReq2)
	if createMenuRR2.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", createMenuRR2.Code)
	}

	setMenusReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/menus", strings.NewReader(`{"menu_ids":[2,1,1]}`))
	setMenusReq.Header.Set("Content-Type", "application/json")
	setMenusRR := httptest.NewRecorder()
	mux.ServeHTTP(setMenusRR, setMenusReq)
	if setMenusRR.Code != http.StatusOK {
		t.Fatalf("expected set role menus status 200, got %d", setMenusRR.Code)
	}

	getMenusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/menus", nil)
	getMenusRR := httptest.NewRecorder()
	mux.ServeHTTP(getMenusRR, getMenusReq)
	if getMenusRR.Code != http.StatusOK {
		t.Fatalf("expected get role menus status 200, got %d", getMenusRR.Code)
	}
	if !strings.Contains(getMenusRR.Body.String(), `"menu_ids":[1,2]`) {
		t.Fatalf("expected sorted deduplicated menu ids, got %s", getMenusRR.Body.String())
	}

	setAPIsReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/apis", strings.NewReader(`{"apis":["POST:/admin/v1/users","GET:/admin/v1/users","GET:/admin/v1/users"]}`))
	setAPIsReq.Header.Set("Content-Type", "application/json")
	setAPIsRR := httptest.NewRecorder()
	mux.ServeHTTP(setAPIsRR, setAPIsReq)
	if setAPIsRR.Code != http.StatusOK {
		t.Fatalf("expected set role apis status 200, got %d", setAPIsRR.Code)
	}

	getAPIsReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/apis", nil)
	getAPIsRR := httptest.NewRecorder()
	mux.ServeHTTP(getAPIsRR, getAPIsReq)
	if getAPIsRR.Code != http.StatusOK {
		t.Fatalf("expected get role apis status 200, got %d", getAPIsRR.Code)
	}
	if !strings.Contains(getAPIsRR.Body.String(), `"apis":["GET:/admin/v1/users","POST:/admin/v1/users"]`) {
		t.Fatalf("expected sorted deduplicated api bindings, got %s", getAPIsRR.Body.String())
	}

	setPoliciesReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/policies", strings.NewReader(`{"rules":[{"api":"GET:/admin/v1/users","effect":"allow","require_verified":true,"require_claims_version":"v2"},{"api":"POST:/admin/v1/users","effect":"deny"}]}`))
	setPoliciesReq.Header.Set("Content-Type", "application/json")
	setPoliciesRR := httptest.NewRecorder()
	mux.ServeHTTP(setPoliciesRR, setPoliciesReq)
	if setPoliciesRR.Code != http.StatusOK {
		t.Fatalf("expected set role policies status 200, got %d", setPoliciesRR.Code)
	}

	getPoliciesReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/policies", nil)
	getPoliciesRR := httptest.NewRecorder()
	mux.ServeHTTP(getPoliciesRR, getPoliciesReq)
	if getPoliciesRR.Code != http.StatusOK {
		t.Fatalf("expected get role policies status 200, got %d", getPoliciesRR.Code)
	}
	if !strings.Contains(getPoliciesRR.Body.String(), `"effect":"allow"`) {
		t.Fatalf("expected role policy bindings in response, got %s", getPoliciesRR.Body.String())
	}

	setDataScopeReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/data-scope", strings.NewReader(`{"tenant_ids":["tenant-b","tenant-a","tenant-a"],"require_owner_match":true,"cross_tenant_admin_allow":["ops@example.com","ops@example.com"]}`))
	setDataScopeReq.Header.Set("Content-Type", "application/json")
	setDataScopeRR := httptest.NewRecorder()
	mux.ServeHTTP(setDataScopeRR, setDataScopeReq)
	if setDataScopeRR.Code != http.StatusOK {
		t.Fatalf("expected set role data scope status 200, got %d", setDataScopeRR.Code)
	}

	getDataScopeReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/data-scope", nil)
	getDataScopeRR := httptest.NewRecorder()
	mux.ServeHTTP(getDataScopeRR, getDataScopeReq)
	if getDataScopeRR.Code != http.StatusOK {
		t.Fatalf("expected get role data scope status 200, got %d", getDataScopeRR.Code)
	}
	if !strings.Contains(getDataScopeRR.Body.String(), `"tenant_ids":["tenant-a","tenant-b"]`) {
		t.Fatalf("expected normalized tenant scope in response, got %s", getDataScopeRR.Body.String())
	}
	if !strings.Contains(getDataScopeRR.Body.String(), `"cross_tenant_admin_allow":["ops@example.com"]`) {
		t.Fatalf("expected normalized cross tenant whitelist in response, got %s", getDataScopeRR.Body.String())
	}

	diffReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/permissions/diff", strings.NewReader(`{"menu_ids":[1,2],"apis":["GET:/admin/v1/users","POST:/admin/v1/users"],"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}],"data_scope":{"tenant_ids":["tenant-a","tenant-b","tenant-c"],"require_owner_match":false,"cross_tenant_admin_allow":["ops@example.com","root@example.com"]},"permission_contract":{"version":"v2","items":[{"menu_id":1,"route":"/dashboard","buttons":["view"]}]}}`))
	diffReq.Header.Set("Content-Type", "application/json")
	diffRR := httptest.NewRecorder()
	mux.ServeHTTP(diffRR, diffReq)
	if diffRR.Code != http.StatusOK {
		t.Fatalf("expected permissions diff status 200, got %d body=%s", diffRR.Code, diffRR.Body.String())
	}
	if !strings.Contains(diffRR.Body.String(), `"data_scope_changed":true`) {
		t.Fatalf("expected data scope drift in diff response, got %s", diffRR.Body.String())
	}

	checkReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/permissions/check", strings.NewReader(`{"menu_ids":[1,2],"apis":["GET:/admin/v1/users","POST:/admin/v1/users"],"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}],"data_scope":{"tenant_ids":["tenant-a","tenant-b","tenant-c"],"require_owner_match":false,"cross_tenant_admin_allow":["ops@example.com","root@example.com"]},"permission_contract":{"version":"v2","items":[{"menu_id":1,"route":"/dashboard","buttons":["view"]}]}}`))
	checkReq.Header.Set("Content-Type", "application/json")
	checkRR := httptest.NewRecorder()
	mux.ServeHTTP(checkRR, checkReq)
	if checkRR.Code != http.StatusOK {
		t.Fatalf("expected permissions check status 200, got %d body=%s", checkRR.Code, checkRR.Body.String())
	}
	if !strings.Contains(checkRR.Body.String(), `"blocking":true`) {
		t.Fatalf("expected blocking permission check, got %s", checkRR.Body.String())
	}

	setContractReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/permission-contract", strings.NewReader(`{"version":"v2","items":[{"menu_id":1,"route":"/dashboard","buttons":["view"]},{"menu_id":2,"route":"/system","buttons":["create","delete","create"]}]}`))
	setContractReq.Header.Set("Content-Type", "application/json")
	setContractRR := httptest.NewRecorder()
	mux.ServeHTTP(setContractRR, setContractReq)
	if setContractRR.Code != http.StatusOK {
		t.Fatalf("expected set permission contract status 200, got %d", setContractRR.Code)
	}
	if !strings.Contains(setContractRR.Body.String(), `"version":"v2"`) {
		t.Fatalf("expected permission contract version in response, got %s", setContractRR.Body.String())
	}

	getContractReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/permission-contract", nil)
	getContractRR := httptest.NewRecorder()
	mux.ServeHTTP(getContractRR, getContractReq)
	if getContractRR.Code != http.StatusOK {
		t.Fatalf("expected get permission contract status 200, got %d", getContractRR.Code)
	}
	if !strings.Contains(getContractRR.Body.String(), `"buttons":["create","delete"]`) {
		t.Fatalf("expected normalized button list in contract, got %s", getContractRR.Body.String())
	}

	checkContractReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/permission-contract/consistency-check", nil)
	checkContractRR := httptest.NewRecorder()
	mux.ServeHTTP(checkContractRR, checkContractReq)
	if checkContractRR.Code != http.StatusOK {
		t.Fatalf("expected permission contract consistency status 200, got %d", checkContractRR.Code)
	}
	if !strings.Contains(checkContractRR.Body.String(), `"passed":true`) {
		t.Fatalf("expected consistency check passed, got %s", checkContractRR.Body.String())
	}

	listRegistryReq := httptest.NewRequest(http.MethodGet, "/admin/v1/apis", nil)
	listRegistryRR := httptest.NewRecorder()
	mux.ServeHTTP(listRegistryRR, listRegistryReq)
	if listRegistryRR.Code != http.StatusOK {
		t.Fatalf("expected list api registry status 200, got %d", listRegistryRR.Code)
	}
	if !strings.Contains(listRegistryRR.Body.String(), `"GET:/admin/v1/roles/{id}/apis"`) {
		t.Fatalf("expected registered apis in response, got %s", listRegistryRR.Body.String())
	}

	invalidAPIReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/apis", strings.NewReader(`{"apis":["DELETE:/admin/v1/users"]}`))
	invalidAPIReq.Header.Set("Content-Type", "application/json")
	invalidAPIRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidAPIRR, invalidAPIReq)
	if invalidAPIRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid api binding status 400, got %d", invalidAPIRR.Code)
	}
}

func TestRolePolicySnapshotAndRollbackRoutes(t *testing.T) {
	mux, _, _, _, _, _ := newRBACMux()

	createRoleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read"]}`))
	createRoleReq.Header.Set("Content-Type", "application/json")
	createRoleRR := httptest.NewRecorder()
	mux.ServeHTTP(createRoleRR, createRoleReq)
	if createRoleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", createRoleRR.Code)
	}

	setV1Req := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/policies", strings.NewReader(`{"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}]}`))
	setV1Req.Header.Set("Content-Type", "application/json")
	setV1RR := httptest.NewRecorder()
	mux.ServeHTTP(setV1RR, setV1Req)
	if setV1RR.Code != http.StatusOK {
		t.Fatalf("expected set policies v1 status 200, got %d body=%s", setV1RR.Code, setV1RR.Body.String())
	}

	snapV1Req := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/snapshots", nil)
	snapV1RR := httptest.NewRecorder()
	mux.ServeHTTP(snapV1RR, snapV1Req)
	if snapV1RR.Code != http.StatusCreated || !strings.Contains(snapV1RR.Body.String(), `"version":"v1"`) {
		t.Fatalf("expected snapshot v1 created, got code=%d body=%s", snapV1RR.Code, snapV1RR.Body.String())
	}

	setV2Req := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/policies", strings.NewReader(`{"rules":[{"api":"POST:/admin/v1/users","effect":"deny"}]}`))
	setV2Req.Header.Set("Content-Type", "application/json")
	setV2RR := httptest.NewRecorder()
	mux.ServeHTTP(setV2RR, setV2Req)
	if setV2RR.Code != http.StatusOK {
		t.Fatalf("expected set policies v2 status 200, got %d body=%s", setV2RR.Code, setV2RR.Body.String())
	}

	snapV2Req := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/snapshots", nil)
	snapV2RR := httptest.NewRecorder()
	mux.ServeHTTP(snapV2RR, snapV2Req)
	if snapV2RR.Code != http.StatusCreated || !strings.Contains(snapV2RR.Body.String(), `"version":"v2"`) {
		t.Fatalf("expected snapshot v2 created, got code=%d body=%s", snapV2RR.Code, snapV2RR.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/policies/snapshots", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected list snapshots status 200, got %d", listRR.Code)
	}
	if !strings.Contains(listRR.Body.String(), `"version":"v1"`) || !strings.Contains(listRR.Body.String(), `"version":"v2"`) {
		t.Fatalf("expected snapshot versions in list, got %s", listRR.Body.String())
	}

	rollbackReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/rollback", strings.NewReader(`{"snapshot_version":"v1","approver":"security.lead"}`))
	rollbackReq.Header.Set("Content-Type", "application/json")
	rollbackRR := httptest.NewRecorder()
	mux.ServeHTTP(rollbackRR, rollbackReq)
	if rollbackRR.Code != http.StatusOK {
		t.Fatalf("expected rollback status 200, got %d body=%s", rollbackRR.Code, rollbackRR.Body.String())
	}
	if !strings.Contains(rollbackRR.Body.String(), `"api":"GET:/admin/v1/users"`) {
		t.Fatalf("expected rollback to restore v1 policy, got %s", rollbackRR.Body.String())
	}

	missingApproverReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/rollback", strings.NewReader(`{"snapshot_version":"v1"}`))
	missingApproverReq.Header.Set("Content-Type", "application/json")
	missingApproverRR := httptest.NewRecorder()
	mux.ServeHTTP(missingApproverRR, missingApproverReq)
	if missingApproverRR.Code != http.StatusBadRequest {
		t.Fatalf("expected rollback approver required status 400, got %d", missingApproverRR.Code)
	}
}

func TestRolePolicyPersistenceExportImportRoutes(t *testing.T) {
	mux, _, _, _, _, _ := newRBACMux()

	createRoleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read"]}`))
	createRoleReq.Header.Set("Content-Type", "application/json")
	createRoleRR := httptest.NewRecorder()
	mux.ServeHTTP(createRoleRR, createRoleReq)
	if createRoleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", createRoleRR.Code)
	}

	createMenuReq := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"Dashboard","path":"/dashboard","order":1}`))
	createMenuReq.Header.Set("Content-Type", "application/json")
	createMenuRR := httptest.NewRecorder()
	mux.ServeHTTP(createMenuRR, createMenuReq)
	if createMenuRR.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", createMenuRR.Code)
	}

	setPoliciesReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/policies", strings.NewReader(`{"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}]}`))
	setPoliciesReq.Header.Set("Content-Type", "application/json")
	setPoliciesRR := httptest.NewRecorder()
	mux.ServeHTTP(setPoliciesRR, setPoliciesReq)
	if setPoliciesRR.Code != http.StatusOK {
		t.Fatalf("expected set policies status 200, got %d", setPoliciesRR.Code)
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/policies/persistence/export", nil)
	exportRR := httptest.NewRecorder()
	mux.ServeHTTP(exportRR, exportReq)
	if exportRR.Code != http.StatusOK {
		t.Fatalf("expected export status 200, got %d body=%s", exportRR.Code, exportRR.Body.String())
	}
	if !strings.Contains(exportRR.Body.String(), `"role_id":1`) {
		t.Fatalf("expected role id in export response, got %s", exportRR.Body.String())
	}

	importReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/persistence/import", strings.NewReader(`{"operator":"security.lead","bundle":{"menu_ids":[1],"apis":["GET:/admin/v1/users"],"rules":[{"api":"GET:/admin/v1/users","effect":"allow"}],"data_scope":{"tenant_ids":["tenant-a"],"require_owner_match":true,"cross_tenant_admin_allow":["ops@example.com"]},"permission_contract":{"version":"v2","items":[{"menu_id":1,"route":"/dashboard","buttons":["view"]}]}}}`))
	importReq.Header.Set("Content-Type", "application/json")
	importRR := httptest.NewRecorder()
	mux.ServeHTTP(importRR, importReq)
	if importRR.Code != http.StatusOK {
		t.Fatalf("expected import status 200, got %d body=%s", importRR.Code, importRR.Body.String())
	}
	if !strings.Contains(importRR.Body.String(), `"cross_tenant_admin_allow":["ops@example.com"]`) {
		t.Fatalf("expected imported cross-tenant allow list, got %s", importRR.Body.String())
	}

	invalidImportReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles/1/policies/persistence/import", strings.NewReader(`{"bundle":{"apis":["GET:/admin/v1/users"]}}`))
	invalidImportReq.Header.Set("Content-Type", "application/json")
	invalidImportRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidImportRR, invalidImportReq)
	if invalidImportRR.Code != http.StatusBadRequest {
		t.Fatalf("expected import operator-required status 400, got %d", invalidImportRR.Code)
	}
}
