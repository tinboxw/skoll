package rbac

import (
	"net/http"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/menu"
	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/role"
)

type RoleService interface {
	Create(name string, permissions []string) role.Role
	Get(id int64) (role.Role, error)
	List() []role.Role
}

type MenuService interface {
	Get(id int64) (menu.Item, error)
}

type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type RBACService interface {
	SetRoleMenus(roleID int64, menuIDs []int64) []int64
	GetRoleMenus(roleID int64) []int64
	SetRoleAPIs(roleID int64, apis []string) []string
	GetRoleAPIs(roleID int64) []string
	SetRolePolicies(roleID int64, rules []rbac.PolicyRule) []rbac.PolicyRule
	GetRolePolicies(roleID int64) []rbac.PolicyRule
	CreateRolePolicySnapshot(roleID int64) rbac.PolicySnapshot
	ListRolePolicySnapshots(roleID int64) []rbac.PolicySnapshot
	RollbackRolePolicies(roleID int64, version string) ([]rbac.PolicyRule, error)
	PermissionBundle(roleID int64) rbac.PermissionBundle
	SetRoleDataScope(roleID int64, scope rbac.DataScope) rbac.DataScope
	GetRoleDataScope(roleID int64) rbac.DataScope
	SetRoleRoutePermissions(roleID int64, version string, items []rbac.RoutePermissionItem) rbac.RoutePermissionContract
	GetRoleRoutePermissions(roleID int64) rbac.RoutePermissionContract
	CheckRoleRoutePermissionConsistency(roleID int64) rbac.RoutePermissionConsistency
}

type APIRegistryService interface {
	Exists(entry string) bool
}

type APIRegistry interface {
	RegisterMany(entries []string)
}

type Handler struct {
	roles RoleService
	menus MenuService
	rbac  RBACService
	apis  APIRegistryService
	audit AuditService
}

func NewHandler(roles RoleService, menus MenuService, rbacSvc RBACService, apiSvc APIRegistryService, auditSvc AuditService) *Handler {
	return &Handler{roles: roles, menus: menus, rbac: rbacSvc, apis: apiSvc, audit: auditSvc}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, registry APIRegistry) {
	if mux == nil || h == nil || h.roles == nil || h.menus == nil || h.rbac == nil || h.apis == nil {
		return
	}
	handle := func(pattern string, next http.HandlerFunc) {
		if registry != nil {
			registry.RegisterMany([]string{pattern})
		}
		hd := http.Handler(next)
		if wrapper != nil {
			hd = wrapper(hd)
		}
		mux.Handle(pattern, hd)
	}

	handle("POST /admin/v1/roles", h.createRoleHandler())
	handle("GET /admin/v1/roles", h.listRolesHandler())
	handle("GET /admin/v1/roles/{id}", h.getRoleHandler())
	handle("PUT /admin/v1/roles/{id}/menus", h.setRoleMenusHandler())
	handle("GET /admin/v1/roles/{id}/menus", h.getRoleMenusHandler())
	handle("PUT /admin/v1/roles/{id}/apis", h.setRoleAPIsHandler())
	handle("GET /admin/v1/roles/{id}/apis", h.getRoleAPIsHandler())
	handle("PUT /admin/v1/roles/{id}/policies", h.setRolePoliciesHandler())
	handle("GET /admin/v1/roles/{id}/policies", h.getRolePoliciesHandler())
	handle("POST /admin/v1/roles/{id}/policies/snapshots", h.createRolePolicySnapshotHandler())
	handle("GET /admin/v1/roles/{id}/policies/snapshots", h.listRolePolicySnapshotsHandler())
	handle("POST /admin/v1/roles/{id}/permissions/diff", h.diffRolePermissionsHandler())
	handle("POST /admin/v1/roles/{id}/permissions/check", h.checkRolePermissionsHandler())
	handle("GET /admin/v1/roles/{id}/policies/persistence/export", h.exportRolePolicyPersistenceHandler())
	handle("POST /admin/v1/roles/{id}/policies/persistence/import", h.importRolePolicyPersistenceHandler())
	handle("POST /admin/v1/roles/{id}/policies/rollback", h.rollbackRolePoliciesHandler())
	handle("PUT /admin/v1/roles/{id}/data-scope", h.setRoleDataScopeHandler())
	handle("GET /admin/v1/roles/{id}/data-scope", h.getRoleDataScopeHandler())
	handle("PUT /admin/v1/roles/{id}/permission-contract", h.setRolePermissionContractHandler())
	handle("GET /admin/v1/roles/{id}/permission-contract", h.getRolePermissionContractHandler())
	handle("POST /admin/v1/roles/{id}/permission-contract/consistency-check", h.checkRolePermissionContractConsistencyHandler())
}

type createRoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type setRoleMenusRequest struct {
	MenuIDs []int64 `json:"menu_ids"`
}

type roleMenusResponse struct {
	RoleID  int64   `json:"role_id"`
	MenuIDs []int64 `json:"menu_ids"`
}

type setRoleAPIsRequest struct {
	APIs []string `json:"apis"`
}

type roleAPIsResponse struct {
	RoleID int64    `json:"role_id"`
	APIs   []string `json:"apis"`
}

type rolePolicyRuleItem struct {
	API                  string `json:"api"`
	Effect               string `json:"effect"`
	RequireVerified      bool   `json:"require_verified"`
	RequireClaimsVersion string `json:"require_claims_version,omitempty"`
}

type setRolePoliciesRequest struct {
	Rules []rolePolicyRuleItem `json:"rules"`
}

type rolePoliciesResponse struct {
	RoleID int64                `json:"role_id"`
	Rules  []rolePolicyRuleItem `json:"rules"`
}

type rolePolicySnapshotResponse struct {
	RoleID  int64                `json:"role_id"`
	Version string               `json:"version"`
	Rules   []rolePolicyRuleItem `json:"rules"`
}

type rolePolicySnapshotListResponse struct {
	RoleID    int64                        `json:"role_id"`
	Snapshots []rolePolicySnapshotResponse `json:"snapshots"`
}

type rollbackRolePoliciesRequest struct {
	SnapshotVersion string `json:"snapshot_version"`
	Approver        string `json:"approver"`
}

type setRoleDataScopeRequest struct {
	TenantIDs             []string `json:"tenant_ids"`
	RequireOwnerMatch     bool     `json:"require_owner_match"`
	CrossTenantAdminAllow []string `json:"cross_tenant_admin_allow"`
}

type roleDataScopeResponse struct {
	RoleID                int64    `json:"role_id"`
	TenantIDs             []string `json:"tenant_ids"`
	RequireOwnerMatch     bool     `json:"require_owner_match"`
	CrossTenantAdminAllow []string `json:"cross_tenant_admin_allow"`
}

type rolePermissionContractItem struct {
	MenuID  int64    `json:"menu_id"`
	Route   string   `json:"route"`
	Buttons []string `json:"buttons"`
}

type setRolePermissionContractRequest struct {
	Version string                       `json:"version"`
	Items   []rolePermissionContractItem `json:"items"`
}

type rolePermissionContractResponse struct {
	RoleID  int64                        `json:"role_id"`
	Version string                       `json:"version"`
	Items   []rolePermissionContractItem `json:"items"`
}

type rolePermissionContractConsistencyResponse struct {
	RoleID   int64    `json:"role_id"`
	Passed   bool     `json:"passed"`
	Problems []string `json:"problems,omitempty"`
}

type rolePermissionDiffRequest struct {
	MenuIDs            []int64                          `json:"menu_ids"`
	APIs               []string                         `json:"apis"`
	Rules              []rolePolicyRuleItem             `json:"rules"`
	DataScope          setRoleDataScopeRequest          `json:"data_scope"`
	PermissionContract setRolePermissionContractRequest `json:"permission_contract"`
}

type rolePermissionDiffResponse struct {
	RoleID           int64                `json:"role_id"`
	AddedMenus       []int64              `json:"added_menus"`
	RemovedMenus     []int64              `json:"removed_menus"`
	AddedAPIs        []string             `json:"added_apis"`
	RemovedAPIs      []string             `json:"removed_apis"`
	AddedRules       []rolePolicyRuleItem `json:"added_rules"`
	RemovedRules     []rolePolicyRuleItem `json:"removed_rules"`
	DataScopeChanged bool                 `json:"data_scope_changed"`
	RouteChanged     bool                 `json:"route_changed"`
}

type rolePermissionCheckResponse struct {
	RoleID   int64                      `json:"role_id"`
	Pass     bool                       `json:"pass"`
	Blocking bool                       `json:"blocking"`
	Reasons  []string                   `json:"reasons"`
	Diff     rolePermissionDiffResponse `json:"diff"`
}

type rolePolicyPersistenceBundle struct {
	MenuIDs            []int64                          `json:"menu_ids"`
	APIs               []string                         `json:"apis"`
	Rules              []rolePolicyRuleItem             `json:"rules"`
	DataScope          setRoleDataScopeRequest          `json:"data_scope"`
	PermissionContract setRolePermissionContractRequest `json:"permission_contract"`
	Snapshots          []rolePolicySnapshotResponse     `json:"snapshots,omitempty"`
}

type rolePolicyPersistenceResponse struct {
	RoleID         int64                       `json:"role_id"`
	ExportedAtUnix int64                       `json:"exported_at_unix_sec"`
	Bundle         rolePolicyPersistenceBundle `json:"bundle"`
}

type importRolePolicyPersistenceRequest struct {
	Operator string                      `json:"operator"`
	Bundle   rolePolicyPersistenceBundle `json:"bundle"`
}
