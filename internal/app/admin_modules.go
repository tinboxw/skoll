package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/internal/integration/adminauth"
	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/config"
	"github.com/tinboxw/skoll/internal/module/dictionary"
	"github.com/tinboxw/skoll/internal/module/fileservice"
	"github.com/tinboxw/skoll/internal/module/jobscheduler"
	"github.com/tinboxw/skoll/internal/module/menu"
	"github.com/tinboxw/skoll/internal/module/modgenerator"
	"github.com/tinboxw/skoll/internal/module/pluginmgr"
	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/releasegov"
	"github.com/tinboxw/skoll/internal/module/role"
	"github.com/tinboxw/skoll/internal/module/user"
)

var adminModuleStartTime = time.Now().UTC()

const dashboardUIBootstrapContractVersion = "v1"

const dashboardJWTRefreshLeadWindow = 5 * time.Minute

type AdminModuleServices struct {
	Users        UserService
	Roles        RoleService
	Menus        MenuService
	Audit        AuditService
	Configs      ConfigService
	Dictionaries DictionaryService
	Files        FileService
	Jobs         JobService
	Generator    GeneratorService
	Plugins      PluginService
	RBAC         RBACService
	APIs         APIRegistryService
	Releases     ReleaseService
}

type UserService interface {
	Create(name, email string) user.User
	Get(id int64) (user.User, error)
	List() []user.User
	RotatePassword(userID int64, minInterval time.Duration, now time.Time) (user.SecurityState, error)
	RegisterLoginFailure(userID int64, lockThreshold int, lockDuration time.Duration, now time.Time) (user.SecurityState, error)
	ResetUserLock(userID int64, now time.Time) (user.SecurityState, error)
	SetMFA(userID int64, enabled bool, provider string, now time.Time) (user.SecurityState, error)
	RevokeSession(sessionID, reason string, now time.Time) user.SessionStatus
	SessionStatus(sessionID string) user.SessionStatus
	ReportSessionAnomaly(sessionID, category, detail string, now time.Time) user.SessionAnomaly
	HeartbeatSessionConsistency(sessionID, instanceID string, version int64, now time.Time) user.SessionConsistency
	SessionConsistencyStatus(sessionID string) user.SessionConsistency
	CreateAuthSession(userID, roleID int64, claimsVersion string, now time.Time) (user.AuthTokenPair, error)
	RefreshAuthSession(refreshToken string, now time.Time) (user.AuthTokenPair, error)
	RevokeAuthSession(sessionID, reason string, now time.Time) (user.AuthSession, error)
	GetAuthSession(sessionID string) (user.AuthSession, error)
}

type RoleService interface {
	Create(name string, permissions []string) role.Role
	Get(id int64) (role.Role, error)
	List() []role.Role
}

type MenuService interface {
	Create(title, path string, order int) menu.Item
	Get(id int64) (menu.Item, error)
	List() []menu.Item
}

type AuditService interface {
	Append(actor, action, target string) audit.Record
	Recent(limit int) []audit.Record
	Query(q audit.Query) audit.QueryResult
}

type ConfigService interface {
	Set(key, value, description string) config.Entry
	Get(key string) (config.Entry, error)
	List() []config.Entry
}

type DictionaryService interface {
	Create(itemType, label, value string, sortOrder int, enabled bool) dictionary.Item
	Get(id int64) (dictionary.Item, error)
	List() []dictionary.Item
	ListByType(itemType string) []dictionary.Item
}

type FileService interface {
	Upload(name string, content []byte) (fileservice.File, error)
	Get(id int64) (fileservice.File, error)
	List() []fileservice.File
	Download(id int64) (fileservice.File, []byte, error)
}

type JobService interface {
	Create(name, schedule string) jobscheduler.Job
	Get(id int64) (jobscheduler.Job, error)
	List() []jobscheduler.Job
	Run(jobID int64) (jobscheduler.Execution, error)
	History(jobID int64, limit int) []jobscheduler.Execution
	ClaimRun(jobID int64, executionKey, instanceID string, now time.Time) (jobscheduler.DispatchClaim, error)
	ClaimStatus(executionKey string) jobscheduler.DispatchClaim
}

type GeneratorService interface {
	Generate(module string) (modgenerator.Result, error)
	GenerateWithSchema(module string, schema *modgenerator.FormSchema, templateVersion string) (modgenerator.Result, error)
}

type PluginService interface {
	Install(name, version string, hooks []string) pluginmgr.Manifest
	InstallPackage(name, version, packageURL, packageHash string, hooks []string) (pluginmgr.Manifest, error)
	InstallPackageVerified(name, version, packageURL, packageHash, signature string, dependencies []pluginmgr.Dependency, hooks []string) (pluginmgr.Manifest, error)
	Get(name string) (pluginmgr.Manifest, error)
	List() []pluginmgr.Manifest
	Enable(name string) (pluginmgr.Manifest, error)
	Disable(name string) (pluginmgr.Manifest, error)
	CheckVersion(name, latestVersion string) (pluginmgr.VersionCheckResult, error)
	UpgradePackage(name, targetVersion, packageURL, packageHash, signature string, dependencies []pluginmgr.Dependency, hooks []string) (pluginmgr.UpgradeResult, error)
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
	RegisterMany(entries []string)
	Exists(entry string) bool
	List() []string
}

type ReleaseService interface {
	SubmitEvidence(input releasegov.EvidenceInput, now time.Time) (releasegov.Evidence, error)
	Scorecard(milestone string, allowedRegression float64) releasegov.Scorecard
}

func MountAdminModuleRoutes(mux *http.ServeMux, services AdminModuleServices, wrapper func(http.Handler) http.Handler) {
	if mux == nil {
		return
	}
	if services.Users == nil || services.Roles == nil || services.Menus == nil || services.Audit == nil || services.Configs == nil || services.Dictionaries == nil || services.Files == nil || services.Jobs == nil || services.Generator == nil || services.Plugins == nil || services.RBAC == nil || services.APIs == nil {
		return
	}

	handle := func(pattern string, next http.HandlerFunc) {
		services.APIs.RegisterMany([]string{pattern})
		h := http.Handler(next)
		if wrapper != nil {
			h = wrapper(h)
		}
		mux.Handle(pattern, h)
	}
	handlePublic := func(pattern string, next http.HandlerFunc) {
		services.APIs.RegisterMany([]string{pattern})
		mux.Handle(pattern, next)
	}

	handlePublic("POST /admin/v1/auth/login", loginAdminAuthHandler(services.Users, services.Audit))
	handlePublic("POST /admin/v1/auth/refresh", refreshAdminAuthHandler(services.Users, services.Audit))
	handlePublic("POST /admin/v1/auth/logout", logoutAdminAuthHandler(services.Users, services.Audit))
	handlePublic("GET /admin/v1/auth/sessions/{session_id}", getAdminAuthSessionHandler(services.Users))

	handle("POST /admin/v1/users", createUserHandler(services.Users))
	handle("GET /admin/v1/users", listUsersHandler(services.Users))
	handle("GET /admin/v1/users/{id}", getUserHandler(services.Users))
	handle("POST /admin/v1/users/{id}/password/rotate", rotateUserPasswordHandler(services.Users, services.Audit))
	handle("POST /admin/v1/users/{id}/login-failures", registerUserLoginFailureHandler(services.Users, services.Audit))
	handle("POST /admin/v1/users/{id}/lock/reset", resetUserLockHandler(services.Users, services.Audit))
	handle("POST /admin/v1/users/{id}/mfa", setUserMFAHandler(services.Users, services.Audit))
	handle("POST /admin/v1/sessions/revoke", revokeSessionHandler(services.Users, services.Audit))
	handle("GET /admin/v1/sessions/{session_id}/status", getSessionStatusHandler(services.Users))
	handle("POST /admin/v1/sessions/anomalies", reportSessionAnomalyHandler(services.Users, services.Audit))
	handle("POST /admin/v1/sessions/consistency/heartbeat", heartbeatSessionConsistencyHandler(services.Users, services.Audit))
	handle("GET /admin/v1/sessions/{session_id}/consistency", getSessionConsistencyHandler(services.Users))

	handle("POST /admin/v1/roles", createRoleHandler(services.Roles))
	handle("GET /admin/v1/roles", listRolesHandler(services.Roles))
	handle("GET /admin/v1/roles/{id}", getRoleHandler(services.Roles))
	handle("PUT /admin/v1/roles/{id}/menus", setRoleMenusHandler(services.Roles, services.Menus, services.RBAC))
	handle("GET /admin/v1/roles/{id}/menus", getRoleMenusHandler(services.Roles, services.RBAC))
	handle("PUT /admin/v1/roles/{id}/apis", setRoleAPIsHandler(services.Roles, services.RBAC, services.APIs))
	handle("GET /admin/v1/roles/{id}/apis", getRoleAPIsHandler(services.Roles, services.RBAC))
	handle("PUT /admin/v1/roles/{id}/policies", setRolePoliciesHandler(services.Roles, services.RBAC, services.APIs))
	handle("GET /admin/v1/roles/{id}/policies", getRolePoliciesHandler(services.Roles, services.RBAC))
	handle("POST /admin/v1/roles/{id}/policies/snapshots", createRolePolicySnapshotHandler(services.Roles, services.RBAC, services.Audit))
	handle("GET /admin/v1/roles/{id}/policies/snapshots", listRolePolicySnapshotsHandler(services.Roles, services.RBAC))
	handle("POST /admin/v1/roles/{id}/permissions/diff", diffRolePermissionsHandler(services.Roles, services.RBAC))
	handle("POST /admin/v1/roles/{id}/permissions/check", checkRolePermissionsHandler(services.Roles, services.RBAC))
	handle("POST /admin/v1/roles/{id}/policies/rollback", rollbackRolePoliciesHandler(services.Roles, services.RBAC, services.Audit))
	handle("PUT /admin/v1/roles/{id}/data-scope", setRoleDataScopeHandler(services.Roles, services.RBAC))
	handle("GET /admin/v1/roles/{id}/data-scope", getRoleDataScopeHandler(services.Roles, services.RBAC))
	handle("PUT /admin/v1/roles/{id}/permission-contract", setRolePermissionContractHandler(services.Roles, services.RBAC, services.Menus))
	handle("GET /admin/v1/roles/{id}/permission-contract", getRolePermissionContractHandler(services.Roles, services.RBAC))
	handle("POST /admin/v1/roles/{id}/permission-contract/consistency-check", checkRolePermissionContractConsistencyHandler(services.Roles, services.RBAC))

	handle("POST /admin/v1/menus", createMenuHandler(services.Menus))
	handle("GET /admin/v1/menus", listMenusHandler(services.Menus))
	handle("GET /admin/v1/menus/{id}", getMenuHandler(services.Menus))

	handle("POST /admin/v1/audit-logs", appendAuditLogHandler(services.Audit))
	handle("GET /admin/v1/audit-logs", recentAuditLogsHandler(services.Audit))
	handle("POST /admin/v1/configs", upsertConfigHandler(services.Configs))
	handle("GET /admin/v1/configs", listConfigsHandler(services.Configs))
	handle("GET /admin/v1/configs/{key}", getConfigHandler(services.Configs))
	handle("POST /admin/v1/dictionaries", createDictionaryHandler(services.Dictionaries))
	handle("GET /admin/v1/dictionaries", listDictionariesHandler(services.Dictionaries))
	handle("GET /admin/v1/dictionaries/{id}", getDictionaryHandler(services.Dictionaries))
	handle("POST /admin/v1/files", uploadFileHandler(services.Files))
	handle("GET /admin/v1/files", listFilesHandler(services.Files))
	handle("GET /admin/v1/files/{id}", getFileHandler(services.Files))
	handle("GET /admin/v1/files/{id}/download", downloadFileHandler(services.Files))
	handle("POST /admin/v1/jobs", createJobHandler(services.Jobs))
	handle("GET /admin/v1/jobs", listJobsHandler(services.Jobs))
	handle("POST /admin/v1/jobs/{id}/run", runJobHandler(services.Jobs))
	handle("GET /admin/v1/jobs/{id}/history", listJobHistoryHandler(services.Jobs))
	handle("POST /admin/v1/jobs/{id}/dispatch-claim", claimJobDispatchHandler(services.Jobs, services.Audit))
	handle("GET /admin/v1/job-dispatch-claims/{execution_key}", getJobDispatchClaimHandler(services.Jobs))
	handle("POST /admin/v1/generator/modules", generateModuleHandler(services.Generator))
	handle("POST /admin/v1/plugins/manifests", installPluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/packages/install", installPluginPackageHandler(services.Plugins))
	handle("GET /admin/v1/plugins", listPluginsHandler(services.Plugins))
	handle("GET /admin/v1/plugins/{name}", getPluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/enable", enablePluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/disable", disablePluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/version-check", checkPluginVersionHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/upgrade", upgradePluginHandler(services.Plugins))
	handle("POST /admin/v1/db/migrations/plan", planDatabaseMigrationHandler(services.Audit))
	handle("POST /admin/v1/db/backup", backupDatabaseHandler(services.Audit))
	handle("POST /admin/v1/db/restore", restoreDatabaseHandler(services.Audit))
	handle("POST /admin/v1/db/sql/execute", executeControlledSQLHandler(services.Audit))
	handle("GET /admin/v1/system/status", systemStatusHandler(services))
	handle("GET /admin/v1/system/runtime-metrics", runtimeMetricsHandler())
	handle("GET /admin/v1/system/node-health", nodeHealthHandler(services))
	handle("GET /admin/v1/system/dashboard", dashboardAggregateHandler(services))
	handle("GET /admin/v1/apis", listRegisteredAPIsHandler(services.APIs))
	if services.Releases != nil {
		handle("POST /admin/v1/release-governance/evidence", submitReleaseEvidenceHandler(services.Releases, services.Audit))
		handle("GET /admin/v1/release-governance/scorecard/{milestone}", getReleaseScorecardHandler(services.Releases))
	}
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type rotateUserPasswordRequest struct {
	MinIntervalMinutes int `json:"min_interval_minutes"`
}

type registerUserLoginFailureRequest struct {
	LockThreshold      int `json:"lock_threshold"`
	LockDurationMinute int `json:"lock_duration_minutes"`
}

type setUserMFARequest struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
}

type revokeSessionRequest struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}

type reportSessionAnomalyRequest struct {
	SessionID string `json:"session_id"`
	Category  string `json:"category"`
	Detail    string `json:"detail"`
}

type heartbeatSessionConsistencyRequest struct {
	SessionID  string `json:"session_id"`
	InstanceID string `json:"instance_id"`
	Version    int64  `json:"version"`
}

type adminAuthLoginRequest struct {
	UserID        int64  `json:"user_id"`
	RoleID        int64  `json:"role_id"`
	ClaimsVersion string `json:"claims_version"`
}

type adminAuthRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type adminAuthLogoutRequest struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}

type createRoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type createMenuRequest struct {
	Title string `json:"title"`
	Path  string `json:"path"`
	Order int    `json:"order"`
}

type appendAuditLogRequest struct {
	Actor  string `json:"actor"`
	Action string `json:"action"`
	Target string `json:"target"`
}

type upsertConfigRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type createDictionaryRequest struct {
	Type    string `json:"type"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	Sort    int    `json:"sort"`
	Enabled *bool  `json:"enabled"`
}

type createJobRequest struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
}

type claimJobDispatchRequest struct {
	ExecutionKey string `json:"execution_key"`
	InstanceID   string `json:"instance_id"`
}

type generateModuleRequest struct {
	Module          string                   `json:"module"`
	TemplateVersion string                   `json:"template_version"`
	FormSchema      *modgenerator.FormSchema `json:"form_schema"`
}

type installPluginRequest struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Hooks   []string `json:"hooks"`
}

type installPluginPackageRequest struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	PackageURL   string                 `json:"package_url"`
	PackageHash  string                 `json:"package_hash"`
	Signature    string                 `json:"signature"`
	Dependencies []pluginmgr.Dependency `json:"dependencies"`
	Hooks        []string               `json:"hooks"`
}

type upgradePluginRequest struct {
	TargetVersion string                 `json:"target_version"`
	PackageURL    string                 `json:"package_url"`
	PackageHash   string                 `json:"package_hash"`
	Signature     string                 `json:"signature"`
	Dependencies  []pluginmgr.Dependency `json:"dependencies"`
	Hooks         []string               `json:"hooks"`
}

type migrationPlanRequest struct {
	FromVersion string   `json:"from_version"`
	ToVersion   string   `json:"to_version"`
	Steps       []string `json:"steps"`
}

type migrationPlanResponse struct {
	PlanID       string   `json:"plan_id"`
	FromVersion  string   `json:"from_version"`
	ToVersion    string   `json:"to_version"`
	Steps        []string `json:"steps"`
	CreatedAtSec int64    `json:"created_at_unix_sec"`
}

type backupRequest struct {
	BackupID string `json:"backup_id"`
	Reason   string `json:"reason"`
}

type backupResponse struct {
	BackupID     string `json:"backup_id"`
	Status       string `json:"status"`
	CreatedAtSec int64  `json:"created_at_unix_sec"`
}

type restoreRequest struct {
	BackupID     string `json:"backup_id"`
	ConfirmToken string `json:"confirm_token"`
}

type controlledSQLRequest struct {
	SQL            string `json:"sql"`
	AllowDangerous bool   `json:"allow_dangerous"`
	ConfirmToken   string `json:"confirm_token"`
}

type controlledSQLResponse struct {
	Allowed      bool   `json:"allowed"`
	ExecutionID  string `json:"execution_id,omitempty"`
	SafetyResult string `json:"safety_result"`
}

type submitReleaseEvidenceRequest struct {
	Milestone           string  `json:"milestone"`
	GoTestPassed        bool    `json:"go_test_passed"`
	GoRacePassed        bool    `json:"go_race_passed"`
	ReadmeSynced        bool    `json:"readme_synced"`
	BenchmarkNsPerOp    float64 `json:"benchmark_ns_per_op"`
	BaselineNsPerOp     float64 `json:"baseline_ns_per_op"`
	BenchmarkCommand    string  `json:"benchmark_command"`
	EvidenceDescription string  `json:"evidence_description"`
}

type dbOpsState struct {
	mu      sync.Mutex
	nextID  int64
	backups map[string]time.Time
}

var adminDBOpsState = dbOpsState{nextID: 1, backups: make(map[string]time.Time)}

const dbDangerousConfirmToken = "I_UNDERSTAND"

type pluginVersionCheckRequest struct {
	LatestVersion string `json:"latest_version"`
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

type listAPIsResponse struct {
	Items []string `json:"items"`
}

type systemStatusResponse struct {
	Version       string `json:"version"`
	UserCount     int    `json:"user_count"`
	RoleCount     int    `json:"role_count"`
	MenuCount     int    `json:"menu_count"`
	ConfigCount   int    `json:"config_count"`
	DictCount     int    `json:"dict_count"`
	FileCount     int    `json:"file_count"`
	JobCount      int    `json:"job_count"`
	PluginCount   int    `json:"plugin_count"`
	APIEntryCount int    `json:"api_entry_count"`
}

type runtimeMetricsResponse struct {
	UptimeSeconds   int64  `json:"uptime_seconds"`
	Goroutines      int    `json:"goroutines"`
	MemoryAlloc     uint64 `json:"memory_alloc_bytes"`
	MemorySys       uint64 `json:"memory_sys_bytes"`
	HeapAlloc       uint64 `json:"heap_alloc_bytes"`
	HeapSys         uint64 `json:"heap_sys_bytes"`
	TotalAlloc      uint64 `json:"total_alloc_bytes"`
	GCCount         uint32 `json:"gc_count"`
	LastGCPauseNs   uint64 `json:"last_gc_pause_ns"`
	SnapshotUnixSec int64  `json:"snapshot_unix_sec"`
}

type healthCheckItem struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
	CheckedAt int64  `json:"checked_at_unix_sec"`
}

type nodeHealthResponse struct {
	NodeStatus   string            `json:"node_status"`
	CheckedAt    int64             `json:"checked_at_unix_sec"`
	Dependencies []healthCheckItem `json:"dependencies"`
}

type dashboardAggregateResponse struct {
	Contract            dashboardContractDescriptor  `json:"contract"`
	GeneratedAtUnixSec  int64                        `json:"generated_at_unix_sec"`
	AuthSession         dashboardAuthSessionContext  `json:"auth_session"`
	AuthObservability   dashboardAuthObservability   `json:"auth_observability"`
	AuthActionability   dashboardAuthActionability   `json:"auth_actionability"`
	JWTSessionBootstrap dashboardJWTSessionBootstrap `json:"jwt_session_bootstrap"`
	Status              systemStatusResponse         `json:"status"`
	RuntimeMetrics      runtimeMetricsResponse       `json:"runtime_metrics"`
	NodeHealth          nodeHealthResponse           `json:"node_health"`
}

type dashboardContractDescriptor struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Stability        string   `json:"stability"`
	RequiredSections []string `json:"required_sections"`
}

type dashboardAuthSessionContext struct {
	Authenticated          bool   `json:"authenticated"`
	AuthModeHint           string `json:"auth_mode_hint"`
	RoleID                 string `json:"role_id,omitempty"`
	HasRoleBinding         bool   `json:"has_role_binding"`
	TokenHeaderPresent     bool   `json:"token_header_present"`
	SignatureHeaderPresent bool   `json:"signature_header_present"`
}

type dashboardAuthModeCounters struct {
	Success uint64            `json:"success"`
	Failure uint64            `json:"failure"`
	Reasons map[string]uint64 `json:"reasons"`
}

type dashboardAuthObservability struct {
	StaticToken  dashboardAuthModeCounters `json:"static_token"`
	HMACSHA256   dashboardAuthModeCounters `json:"hmac_sha256"`
	TotalFailure uint64                    `json:"total_failure"`
}

type dashboardAuthActionability struct {
	Severity            string   `json:"severity"`
	RecommendedAuthMode string   `json:"recommended_auth_mode"`
	FailureRate         float64  `json:"failure_rate"`
	TopFailureReasons   []string `json:"top_failure_reasons"`
	NextActions         []string `json:"next_actions"`
	Docs                []string `json:"docs"`
}

type dashboardJWTSessionBootstrap struct {
	TokenPresent          bool                              `json:"token_present"`
	TokenFormat           string                            `json:"token_format"`
	ClaimsTrusted         bool                              `json:"claims_trusted"`
	MiddlewareBridge      dashboardJWTMiddlewareBridge      `json:"middleware_bridge"`
	ProvenanceAuditExport dashboardJWTProvenanceAuditExport `json:"provenance_audit_export"`
	SessionState          string                            `json:"session_state"`
	VerificationState     string                            `json:"verification_state"`
	VerificationHint      string                            `json:"verification_hint,omitempty"`
	TrustLevel            string                            `json:"trust_level"`
	TrustMessage          string                            `json:"trust_message"`
	Subject               string                            `json:"subject,omitempty"`
	Issuer                string                            `json:"issuer,omitempty"`
	Audience              []string                          `json:"audience,omitempty"`
	IssuedAtUnixSec       int64                             `json:"issued_at_unix_sec,omitempty"`
	ExpiresAtUnixSec      int64                             `json:"expires_at_unix_sec,omitempty"`
	ExpiresInSec          int64                             `json:"expires_in_sec,omitempty"`
	RefreshAfterUnixSec   int64                             `json:"refresh_after_unix_sec,omitempty"`
	RefreshRecommended    bool                              `json:"refresh_recommended"`
	RefreshReason         string                            `json:"refresh_reason,omitempty"`
	Expired               bool                              `json:"expired"`
	ParseError            string                            `json:"parse_error,omitempty"`
}

type dashboardJWTMiddlewareBridge struct {
	Present             bool     `json:"present"`
	Verified            bool     `json:"verified"`
	Source              string   `json:"source"`
	SourceProvenance    []string `json:"source_provenance"`
	Subject             string   `json:"subject,omitempty"`
	SubjectSource       string   `json:"subject_source,omitempty"`
	RoleID              string   `json:"role_id,omitempty"`
	RoleSource          string   `json:"role_source,omitempty"`
	ClaimsVersion       string   `json:"claims_version,omitempty"`
	ClaimsVersionSource string   `json:"claims_version_source,omitempty"`
	VerifiedSource      string   `json:"verified_source,omitempty"`
}

type dashboardJWTProvenanceAuditExport struct {
	Enabled             bool                                     `json:"enabled"`
	SourceProvenance    []string                                 `json:"source_provenance"`
	SourcePath          string                                   `json:"source_path,omitempty"`
	OperationalMetrics  dashboardJWTProvenanceOperationalMetrics `json:"operational_metrics"`
	SLODashboard        dashboardJWTProvenanceSLODashboard       `json:"slo_dashboard"`
	ErrorBudgetPolicy   dashboardJWTProvenanceErrorBudgetPolicy  `json:"error_budget_policy"`
	AlertingHints       []string                                 `json:"alerting_hints,omitempty"`
	Source              string                                   `json:"source,omitempty"`
	Verified            bool                                     `json:"verified"`
	ClaimsTrusted       bool                                     `json:"claims_trusted"`
	VerificationState   string                                   `json:"verification_state"`
	Subject             string                                   `json:"subject,omitempty"`
	RoleID              string                                   `json:"role_id,omitempty"`
	ClaimsVersion       string                                   `json:"claims_version,omitempty"`
	RoleSource          string                                   `json:"role_source,omitempty"`
	SubjectSource       string                                   `json:"subject_source,omitempty"`
	ClaimsVersionSource string                                   `json:"claims_version_source,omitempty"`
	VerifiedSource      string                                   `json:"verified_source,omitempty"`
}

func createUserHandler(svc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Email = strings.TrimSpace(req.Email)
		if req.Name == "" || req.Email == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name and email are required"})
			return
		}

		respondJSON(w, http.StatusCreated, svc.Create(req.Name, req.Email))
	}
}

func listUsersHandler(svc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func getUserHandler(svc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		out, err := svc.Get(id)
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, out)
	}
}

func rotateUserPasswordHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		var req rotateUserPasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if req.MinIntervalMinutes < 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "min_interval_minutes must be >= 0"})
			return
		}
		state, err := userSvc.RotatePassword(userID, time.Duration(req.MinIntervalMinutes)*time.Minute, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("security", "password_rotate", fmt.Sprintf("user:%d", userID))
		respondJSON(w, http.StatusOK, state)
	}
}

func registerUserLoginFailureHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		var req registerUserLoginFailureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if req.LockThreshold <= 0 {
			req.LockThreshold = 5
		}
		if req.LockDurationMinute <= 0 {
			req.LockDurationMinute = 30
		}
		state, err := userSvc.RegisterLoginFailure(userID, req.LockThreshold, time.Duration(req.LockDurationMinute)*time.Minute, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("security", "login_failure", fmt.Sprintf("user:%d", userID))
		respondJSON(w, http.StatusOK, state)
	}
}

func resetUserLockHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		state, err := userSvc.ResetUserLock(userID, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("security", "lock_reset", fmt.Sprintf("user:%d", userID))
		respondJSON(w, http.StatusOK, state)
	}
}

func setUserMFAHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		var req setUserMFARequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		provider := strings.TrimSpace(req.Provider)
		if req.Enabled && provider == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "provider is required when MFA is enabled"})
			return
		}
		state, err := userSvc.SetMFA(userID, req.Enabled, provider, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("security", "mfa_update", fmt.Sprintf("user:%d", userID))
		respondJSON(w, http.StatusOK, state)
	}
}

func loginAdminAuthHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req adminAuthLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if req.UserID <= 0 || req.RoleID <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id and role_id must be > 0"})
			return
		}

		pair, err := userSvc.CreateAuthSession(req.UserID, req.RoleID, req.ClaimsVersion, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		auditSvc.Append("auth", "login", fmt.Sprintf("user:%d session:%s", req.UserID, pair.SessionID))
		respondJSON(w, http.StatusOK, pair)
	}
}

func refreshAdminAuthHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req adminAuthRefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		pair, err := userSvc.RefreshAuthSession(strings.TrimSpace(req.RefreshToken), time.Now().UTC())
		if err != nil {
			switch err {
			case user.ErrAuthInvalidRefreshToken, user.ErrAuthRefreshTokenExpired, user.ErrAuthSessionRevoked:
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_refresh_token"})
				return
			case user.ErrAuthSessionNotFound:
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			default:
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
				return
			}
		}

		auditSvc.Append("auth", "refresh", pair.SessionID)
		respondJSON(w, http.StatusOK, pair)
	}
}

func logoutAdminAuthHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req adminAuthLogoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.SessionID = strings.TrimSpace(req.SessionID)
		req.Reason = strings.TrimSpace(req.Reason)
		if req.SessionID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id is required"})
			return
		}
		if req.Reason == "" {
			req.Reason = "logout"
		}

		session, err := userSvc.RevokeAuthSession(req.SessionID, req.Reason, time.Now().UTC())
		if err != nil {
			if err == user.ErrAuthSessionNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		_ = userSvc.RevokeSession(req.SessionID, req.Reason, time.Now().UTC())
		auditSvc.Append("auth", "logout", req.SessionID)
		respondJSON(w, http.StatusOK, session)
	}
}

func getAdminAuthSessionHandler(userSvc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := parsePathString(r, "session_id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		session, err := userSvc.GetAuthSession(sessionID)
		if err != nil {
			if err == user.ErrAuthSessionNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, session)
	}
}

func revokeSessionHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req revokeSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.SessionID = strings.TrimSpace(req.SessionID)
		req.Reason = strings.TrimSpace(req.Reason)
		if req.SessionID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id is required"})
			return
		}
		if req.Reason == "" {
			req.Reason = "manual revoke"
		}
		status := userSvc.RevokeSession(req.SessionID, req.Reason, time.Now().UTC())
		auditSvc.Append("security", "session_revoke", req.SessionID)
		respondJSON(w, http.StatusOK, status)
	}
}

func getSessionStatusHandler(userSvc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := parsePathString(r, "session_id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, userSvc.SessionStatus(sessionID))
	}
}

func reportSessionAnomalyHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req reportSessionAnomalyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.SessionID = strings.TrimSpace(req.SessionID)
		req.Category = strings.TrimSpace(req.Category)
		req.Detail = strings.TrimSpace(req.Detail)
		if req.SessionID == "" || req.Category == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id and category are required"})
			return
		}
		out := userSvc.ReportSessionAnomaly(req.SessionID, req.Category, req.Detail, time.Now().UTC())
		auditSvc.Append("security", "session_anomaly", req.SessionID+":"+req.Category)
		respondJSON(w, http.StatusCreated, out)
	}
}

func heartbeatSessionConsistencyHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req heartbeatSessionConsistencyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.SessionID = strings.TrimSpace(req.SessionID)
		req.InstanceID = strings.TrimSpace(req.InstanceID)
		if req.SessionID == "" || req.InstanceID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id and instance_id are required"})
			return
		}
		if req.Version <= 0 {
			req.Version = 1
		}

		out := userSvc.HeartbeatSessionConsistency(req.SessionID, req.InstanceID, req.Version, time.Now().UTC())
		action := "session_consistency_heartbeat"
		if !out.Consistent {
			action = "session_consistency_conflict"
		}
		auditSvc.Append("consistency", action, req.SessionID)
		respondJSON(w, http.StatusOK, out)
	}
}

func getSessionConsistencyHandler(userSvc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := parsePathString(r, "session_id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, userSvc.SessionConsistencyStatus(sessionID))
	}
}

func createRoleHandler(svc RoleService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}

		respondJSON(w, http.StatusCreated, svc.Create(req.Name, req.Permissions))
	}
}

func listRolesHandler(svc RoleService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func getRoleHandler(svc RoleService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		out, err := svc.Get(id)
		if err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, out)
	}
}

func setRoleMenusHandler(roleSvc RoleService, menuSvc MenuService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req setRoleMenusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		for _, menuID := range req.MenuIDs {
			if menuID <= 0 {
				continue
			}
			if _, err := menuSvc.Get(menuID); err != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("menu id %d not found", menuID)})
				return
			}
		}

		respondJSON(w, http.StatusOK, roleMenusResponse{RoleID: roleID, MenuIDs: rbacSvc.SetRoleMenus(roleID, req.MenuIDs)})
	}
}

func getRoleMenusHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, roleMenusResponse{RoleID: roleID, MenuIDs: rbacSvc.GetRoleMenus(roleID)})
	}
}

func setRoleAPIsHandler(roleSvc RoleService, rbacSvc RBACService, apiSvc APIRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req setRoleAPIsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		normalized := make([]string, 0, len(req.APIs))
		for _, item := range req.APIs {
			n := apiregistry.Normalize(item)
			if n == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid api format: %q", item)})
				return
			}
			if !apiSvc.Exists(n) {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("api not registered: %s", n)})
				return
			}
			normalized = append(normalized, n)
		}

		respondJSON(w, http.StatusOK, roleAPIsResponse{RoleID: roleID, APIs: rbacSvc.SetRoleAPIs(roleID, normalized)})
	}
}

func getRoleAPIsHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, roleAPIsResponse{RoleID: roleID, APIs: rbacSvc.GetRoleAPIs(roleID)})
	}
}

func setRolePoliciesHandler(roleSvc RoleService, rbacSvc RBACService, apiSvc APIRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req setRolePoliciesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		rules := make([]rbac.PolicyRule, 0, len(req.Rules))
		for _, item := range req.Rules {
			n := apiregistry.Normalize(item.API)
			if n == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid api format: %q", item.API)})
				return
			}
			if !apiSvc.Exists(n) {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("api not registered: %s", n)})
				return
			}
			effect := strings.ToLower(strings.TrimSpace(item.Effect))
			if effect != "allow" && effect != "deny" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid policy effect: %q", item.Effect)})
				return
			}
			rules = append(rules, rbac.PolicyRule{
				API:                  n,
				Effect:               effect,
				RequireVerified:      item.RequireVerified,
				RequireClaimsVersion: strings.ToLower(strings.TrimSpace(item.RequireClaimsVersion)),
			})
		}

		out := rbacSvc.SetRolePolicies(roleID, rules)
		respondJSON(w, http.StatusOK, rolePoliciesResponse{RoleID: roleID, Rules: toRolePolicyRuleItems(out)})
	}
}

func getRolePoliciesHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		out := rbacSvc.GetRolePolicies(roleID)
		respondJSON(w, http.StatusOK, rolePoliciesResponse{RoleID: roleID, Rules: toRolePolicyRuleItems(out)})
	}
}

func createRolePolicySnapshotHandler(roleSvc RoleService, rbacSvc RBACService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		snapshot := rbacSvc.CreateRolePolicySnapshot(roleID)
		auditSvc.Append("rbac", "policy_snapshot_create", fmt.Sprintf("role:%d@%s", roleID, snapshot.Version))
		respondJSON(w, http.StatusCreated, rolePolicySnapshotResponse{RoleID: snapshot.RoleID, Version: snapshot.Version, Rules: toRolePolicyRuleItems(snapshot.Rules)})
	}
}

func listRolePolicySnapshotsHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		raw := rbacSvc.ListRolePolicySnapshots(roleID)
		snapshots := make([]rolePolicySnapshotResponse, 0, len(raw))
		for _, item := range raw {
			snapshots = append(snapshots, rolePolicySnapshotResponse{RoleID: item.RoleID, Version: item.Version, Rules: toRolePolicyRuleItems(item.Rules)})
		}
		respondJSON(w, http.StatusOK, rolePolicySnapshotListResponse{RoleID: roleID, Snapshots: snapshots})
	}
}

func rollbackRolePoliciesHandler(roleSvc RoleService, rbacSvc RBACService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req rollbackRolePoliciesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		version := strings.ToLower(strings.TrimSpace(req.SnapshotVersion))
		approver := strings.TrimSpace(req.Approver)
		if version == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "snapshot_version is required"})
			return
		}
		if approver == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "approver is required"})
			return
		}

		rules, err := rbacSvc.RollbackRolePolicies(roleID, version)
		if err != nil {
			if err == rbac.ErrPolicySnapshotNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		auditSvc.Append("rbac", "policy_snapshot_rollback", fmt.Sprintf("role:%d@%s approver:%s", roleID, version, approver))
		respondJSON(w, http.StatusOK, rolePoliciesResponse{RoleID: roleID, Rules: toRolePolicyRuleItems(rules)})
	}
}

func diffRolePermissionsHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req rolePermissionDiffRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		current := rbacSvc.PermissionBundle(roleID)
		target := rbac.PermissionBundle{
			MenuIDs:  req.MenuIDs,
			APIs:     req.APIs,
			Policies: toRolePolicyRules(req.Rules),
			DataScope: rbac.DataScope{
				TenantIDs:             req.DataScope.TenantIDs,
				RequireOwnerMatch:     req.DataScope.RequireOwnerMatch,
				CrossTenantAdminAllow: req.DataScope.CrossTenantAdminAllow,
			},
			RoutePermission: rbac.RoutePermissionContract{Version: req.PermissionContract.Version, Items: toRoutePermissionItems(req.PermissionContract.Items)},
		}
		diff := rbac.BuildPermissionDiff(current, target)
		respondJSON(w, http.StatusOK, toRolePermissionDiffResponse(roleID, diff))
	}
}

func checkRolePermissionsHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req rolePermissionDiffRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		current := rbacSvc.PermissionBundle(roleID)
		target := rbac.PermissionBundle{
			MenuIDs:  req.MenuIDs,
			APIs:     req.APIs,
			Policies: toRolePolicyRules(req.Rules),
			DataScope: rbac.DataScope{
				TenantIDs:             req.DataScope.TenantIDs,
				RequireOwnerMatch:     req.DataScope.RequireOwnerMatch,
				CrossTenantAdminAllow: req.DataScope.CrossTenantAdminAllow,
			},
			RoutePermission: rbac.RoutePermissionContract{Version: req.PermissionContract.Version, Items: toRoutePermissionItems(req.PermissionContract.Items)},
		}
		diff := rbac.BuildPermissionDiff(current, target)
		check := rbac.EvaluatePermissionCheck(diff)
		respondJSON(w, http.StatusOK, rolePermissionCheckResponse{
			RoleID:   roleID,
			Pass:     check.Pass,
			Blocking: check.Blocking,
			Reasons:  check.Reasons,
			Diff:     toRolePermissionDiffResponse(roleID, check.Diff),
		})
	}
}

func setRoleDataScopeHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req setRoleDataScopeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		tenantIDs := make([]string, 0, len(req.TenantIDs))
		for _, tenantID := range req.TenantIDs {
			tenantID = strings.TrimSpace(tenantID)
			if tenantID == "" {
				continue
			}
			tenantIDs = append(tenantIDs, tenantID)
		}

		crossTenantAllow := make([]string, 0, len(req.CrossTenantAdminAllow))
		for _, subject := range req.CrossTenantAdminAllow {
			subject = strings.TrimSpace(subject)
			if subject == "" {
				continue
			}
			crossTenantAllow = append(crossTenantAllow, subject)
		}

		out := rbacSvc.SetRoleDataScope(roleID, rbac.DataScope{TenantIDs: tenantIDs, RequireOwnerMatch: req.RequireOwnerMatch, CrossTenantAdminAllow: crossTenantAllow})
		respondJSON(w, http.StatusOK, roleDataScopeResponse{RoleID: roleID, TenantIDs: out.TenantIDs, RequireOwnerMatch: out.RequireOwnerMatch, CrossTenantAdminAllow: out.CrossTenantAdminAllow})
	}
}

func getRoleDataScopeHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		out := rbacSvc.GetRoleDataScope(roleID)
		respondJSON(w, http.StatusOK, roleDataScopeResponse{RoleID: roleID, TenantIDs: out.TenantIDs, RequireOwnerMatch: out.RequireOwnerMatch, CrossTenantAdminAllow: out.CrossTenantAdminAllow})
	}
}

func setRolePermissionContractHandler(roleSvc RoleService, rbacSvc RBACService, menuSvc MenuService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		var req setRolePermissionContractRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		items := make([]rbac.RoutePermissionItem, 0, len(req.Items))
		for _, item := range req.Items {
			if item.MenuID > 0 {
				if _, err := menuSvc.Get(item.MenuID); err != nil {
					respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("menu id %d not found", item.MenuID)})
					return
				}
			}
			buttons := make([]string, 0, len(item.Buttons))
			for _, button := range item.Buttons {
				button = strings.TrimSpace(button)
				if button == "" {
					continue
				}
				buttons = append(buttons, button)
			}
			items = append(items, rbac.RoutePermissionItem{MenuID: item.MenuID, Route: strings.TrimSpace(item.Route), Buttons: buttons})
		}

		out := rbacSvc.SetRoleRoutePermissions(roleID, req.Version, items)
		respondJSON(w, http.StatusOK, rolePermissionContractResponse{RoleID: roleID, Version: out.Version, Items: toRolePermissionContractItems(out.Items)})
	}
}

func getRolePermissionContractHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		out := rbacSvc.GetRoleRoutePermissions(roleID)
		respondJSON(w, http.StatusOK, rolePermissionContractResponse{RoleID: roleID, Version: out.Version, Items: toRolePermissionContractItems(out.Items)})
	}
}

func checkRolePermissionContractConsistencyHandler(roleSvc RoleService, rbacSvc RBACService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if _, err := roleSvc.Get(roleID); err != nil {
			if err == role.ErrRoleNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		out := rbacSvc.CheckRoleRoutePermissionConsistency(roleID)
		respondJSON(w, http.StatusOK, rolePermissionContractConsistencyResponse{RoleID: roleID, Passed: out.Passed, Problems: out.Problems})
	}
}

func toRolePermissionContractItems(items []rbac.RoutePermissionItem) []rolePermissionContractItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]rolePermissionContractItem, len(items))
	for i, item := range items {
		out[i] = rolePermissionContractItem{MenuID: item.MenuID, Route: item.Route, Buttons: append([]string(nil), item.Buttons...)}
	}
	return out
}

func toRolePolicyRuleItems(rules []rbac.PolicyRule) []rolePolicyRuleItem {
	if len(rules) == 0 {
		return nil
	}
	out := make([]rolePolicyRuleItem, len(rules))
	for i, rule := range rules {
		out[i] = rolePolicyRuleItem{
			API:                  rule.API,
			Effect:               rule.Effect,
			RequireVerified:      rule.RequireVerified,
			RequireClaimsVersion: rule.RequireClaimsVersion,
		}
	}
	return out
}

func toRolePolicyRules(items []rolePolicyRuleItem) []rbac.PolicyRule {
	if len(items) == 0 {
		return nil
	}
	out := make([]rbac.PolicyRule, len(items))
	for i, item := range items {
		out[i] = rbac.PolicyRule{
			API:                  strings.TrimSpace(item.API),
			Effect:               strings.TrimSpace(item.Effect),
			RequireVerified:      item.RequireVerified,
			RequireClaimsVersion: strings.TrimSpace(item.RequireClaimsVersion),
		}
	}
	return out
}

func toRoutePermissionItems(items []rolePermissionContractItem) []rbac.RoutePermissionItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]rbac.RoutePermissionItem, len(items))
	for i, item := range items {
		out[i] = rbac.RoutePermissionItem{MenuID: item.MenuID, Route: strings.TrimSpace(item.Route), Buttons: append([]string(nil), item.Buttons...)}
	}
	return out
}

func toRolePermissionDiffResponse(roleID int64, diff rbac.PermissionDiff) rolePermissionDiffResponse {
	return rolePermissionDiffResponse{
		RoleID:           roleID,
		AddedMenus:       append([]int64(nil), diff.AddedMenus...),
		RemovedMenus:     append([]int64(nil), diff.RemovedMenus...),
		AddedAPIs:        append([]string(nil), diff.AddedAPIs...),
		RemovedAPIs:      append([]string(nil), diff.RemovedAPIs...),
		AddedRules:       toRolePolicyRuleItems(diff.AddedPolicies),
		RemovedRules:     toRolePolicyRuleItems(diff.RemovedPolicies),
		DataScopeChanged: diff.DataScopeChanged,
		RouteChanged:     diff.RouteChanged,
	}
}

func createMenuHandler(svc MenuService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createMenuRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		req.Path = strings.TrimSpace(req.Path)
		if req.Title == "" || req.Path == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "title and path are required"})
			return
		}

		respondJSON(w, http.StatusCreated, svc.Create(req.Title, req.Path, req.Order))
	}
}

func listMenusHandler(svc MenuService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func getMenuHandler(svc MenuService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		out, err := svc.Get(id)
		if err != nil {
			if err == menu.ErrMenuNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, out)
	}
}

func appendAuditLogHandler(svc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req appendAuditLogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Actor = strings.TrimSpace(req.Actor)
		req.Action = strings.TrimSpace(req.Action)
		req.Target = strings.TrimSpace(req.Target)
		if req.Actor == "" || req.Action == "" || req.Target == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "actor, action and target are required"})
			return
		}

		respondJSON(w, http.StatusCreated, svc.Append(req.Actor, req.Action, req.Target))
	}
}

func recentAuditLogsHandler(svc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := parseAuditQuery(r)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, svc.Query(query))
	}
}

func upsertConfigHandler(svc ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req upsertConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Key = strings.TrimSpace(req.Key)
		req.Value = strings.TrimSpace(req.Value)
		req.Description = strings.TrimSpace(req.Description)
		if req.Key == "" || req.Value == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "key and value are required"})
			return
		}

		respondJSON(w, http.StatusCreated, svc.Set(req.Key, req.Value, req.Description))
	}
}

func listConfigsHandler(svc ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func getConfigHandler(svc ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, err := parsePathString(r, "key")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		item, err := svc.Get(key)
		if err != nil {
			if err == config.ErrConfigNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, item)
	}
}

func createDictionaryHandler(svc DictionaryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createDictionaryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Type = strings.TrimSpace(req.Type)
		req.Label = strings.TrimSpace(req.Label)
		req.Value = strings.TrimSpace(req.Value)
		if req.Type == "" || req.Label == "" || req.Value == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "type, label and value are required"})
			return
		}

		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		respondJSON(w, http.StatusCreated, svc.Create(req.Type, req.Label, req.Value, req.Sort, enabled))
	}
}

func listDictionariesHandler(svc DictionaryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemType := strings.TrimSpace(r.URL.Query().Get("type"))
		if itemType == "" {
			respondJSON(w, http.StatusOK, svc.List())
			return
		}
		respondJSON(w, http.StatusOK, svc.ListByType(itemType))
	}
}

func getDictionaryHandler(svc DictionaryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		item, err := svc.Get(id)
		if err != nil {
			if err == dictionary.ErrItemNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		respondJSON(w, http.StatusOK, item)
	}
}

func uploadFileHandler(svc FileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(16 << 20); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read file content"})
			return
		}

		uploaded, err := svc.Upload(header.Filename, content)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusCreated, uploaded)
	}
}

func listFilesHandler(svc FileService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func getFileHandler(svc FileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		item, err := svc.Get(id)
		if err != nil {
			if err == fileservice.ErrFileNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		respondJSON(w, http.StatusOK, item)
	}
}

func downloadFileHandler(svc FileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		item, content, err := svc.Download(id)
		if err != nil {
			if err == fileservice.ErrFileNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(item.Name)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}
}

func createJobHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createJobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Schedule = strings.TrimSpace(req.Schedule)
		if req.Name == "" || req.Schedule == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name and schedule are required"})
			return
		}
		respondJSON(w, http.StatusCreated, svc.Create(req.Name, req.Schedule))
	}
}

func listJobsHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func runJobHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		exec, err := svc.Run(id)
		if err != nil {
			if err == jobscheduler.ErrJobNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		respondJSON(w, http.StatusOK, exec)
	}
}

func listJobHistoryHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, err := svc.Get(id); err != nil {
			if err == jobscheduler.ErrJobNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		limit := 20
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
				return
			}
			if parsed > 200 {
				parsed = 200
			}
			limit = parsed
		}
		respondJSON(w, http.StatusOK, svc.History(id, limit))
	}
}

func claimJobDispatchHandler(svc JobService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var req claimJobDispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.ExecutionKey = strings.TrimSpace(req.ExecutionKey)
		req.InstanceID = strings.TrimSpace(req.InstanceID)
		if req.ExecutionKey == "" || req.InstanceID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_key and instance_id are required"})
			return
		}

		out, err := svc.ClaimRun(jobID, req.ExecutionKey, req.InstanceID, time.Now().UTC())
		if err != nil {
			if err == jobscheduler.ErrJobNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		action := "job_dispatch_claim"
		if out.DuplicateBlocked {
			action = "job_dispatch_duplicate_blocked"
		}
		auditSvc.Append("consistency", action, req.ExecutionKey)
		respondJSON(w, http.StatusOK, out)
	}
}

func getJobDispatchClaimHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		executionKey, err := parsePathString(r, "execution_key")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, svc.ClaimStatus(executionKey))
	}
}

func generateModuleHandler(svc GeneratorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req generateModuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		result, err := svc.GenerateWithSchema(strings.TrimSpace(req.Module), req.FormSchema, strings.TrimSpace(req.TemplateVersion))
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, result)
	}
}

func installPluginHandler(svc PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req installPluginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Version = strings.TrimSpace(req.Version)
		if req.Name == "" || req.Version == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name and version are required"})
			return
		}
		respondJSON(w, http.StatusCreated, svc.Install(req.Name, req.Version, req.Hooks))
	}
}

func installPluginPackageHandler(svc PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req installPluginPackageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Version = strings.TrimSpace(req.Version)
		req.PackageURL = strings.TrimSpace(req.PackageURL)
		req.PackageHash = strings.TrimSpace(req.PackageHash)
		req.Signature = strings.TrimSpace(req.Signature)
		if req.Name == "" || req.Version == "" || req.PackageURL == "" || req.PackageHash == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name, version, package_url and package_hash are required"})
			return
		}
		item, err := svc.InstallPackageVerified(req.Name, req.Version, req.PackageURL, req.PackageHash, req.Signature, req.Dependencies, req.Hooks)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusCreated, item)
	}
}

func upgradePluginHandler(svc PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, err := parsePathString(r, "name")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		var req upgradePluginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.TargetVersion = strings.TrimSpace(req.TargetVersion)
		req.PackageURL = strings.TrimSpace(req.PackageURL)
		req.PackageHash = strings.TrimSpace(req.PackageHash)
		req.Signature = strings.TrimSpace(req.Signature)
		if req.TargetVersion == "" || req.PackageURL == "" || req.PackageHash == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "target_version, package_url and package_hash are required"})
			return
		}
		result, err := svc.UpgradePackage(name, req.TargetVersion, req.PackageURL, req.PackageHash, req.Signature, req.Dependencies, req.Hooks)
		if err != nil {
			if err == pluginmgr.ErrPluginNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, result)
	}
}

func planDatabaseMigrationHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req migrationPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.FromVersion = strings.TrimSpace(req.FromVersion)
		req.ToVersion = strings.TrimSpace(req.ToVersion)
		if req.FromVersion == "" || req.ToVersion == "" || len(req.Steps) == 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "from_version, to_version and steps are required"})
			return
		}

		adminDBOpsState.mu.Lock()
		planID := fmt.Sprintf("mig-%06d", adminDBOpsState.nextID)
		adminDBOpsState.nextID++
		adminDBOpsState.mu.Unlock()

		auditSvc.Append("dbops", "migration_plan", planID)
		respondJSON(w, http.StatusOK, migrationPlanResponse{PlanID: planID, FromVersion: req.FromVersion, ToVersion: req.ToVersion, Steps: req.Steps, CreatedAtSec: time.Now().UTC().Unix()})
	}
}

func backupDatabaseHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req backupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.BackupID = strings.TrimSpace(req.BackupID)
		if req.BackupID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "backup_id is required"})
			return
		}
		now := time.Now().UTC()
		adminDBOpsState.mu.Lock()
		adminDBOpsState.backups[req.BackupID] = now
		adminDBOpsState.mu.Unlock()
		auditSvc.Append("dbops", "backup", req.BackupID)
		respondJSON(w, http.StatusCreated, backupResponse{BackupID: req.BackupID, Status: "ready", CreatedAtSec: now.Unix()})
	}
}

func restoreDatabaseHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req restoreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.BackupID = strings.TrimSpace(req.BackupID)
		req.ConfirmToken = strings.TrimSpace(req.ConfirmToken)
		if req.BackupID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "backup_id is required"})
			return
		}
		if req.ConfirmToken != dbDangerousConfirmToken {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "confirm_token required for restore"})
			return
		}
		adminDBOpsState.mu.Lock()
		_, ok := adminDBOpsState.backups[req.BackupID]
		adminDBOpsState.mu.Unlock()
		if !ok {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "backup_id not found"})
			return
		}
		auditSvc.Append("dbops", "restore", req.BackupID)
		respondJSON(w, http.StatusOK, map[string]any{"backup_id": req.BackupID, "status": "restored", "restored_at_unix_sec": time.Now().UTC().Unix()})
	}
}

func executeControlledSQLHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req controlledSQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		sqlText := strings.TrimSpace(req.SQL)
		if sqlText == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "sql is required"})
			return
		}
		upper := strings.ToUpper(sqlText)
		dangerous := strings.Contains(upper, "DROP ") || strings.Contains(upper, "TRUNCATE ") || strings.Contains(upper, "DELETE ")
		if dangerous {
			if !req.AllowDangerous || strings.TrimSpace(req.ConfirmToken) != dbDangerousConfirmToken {
				respondJSON(w, http.StatusForbidden, controlledSQLResponse{Allowed: false, SafetyResult: "dangerous_sql_blocked"})
				return
			}
		}

		adminDBOpsState.mu.Lock()
		execID := fmt.Sprintf("sql-%06d", adminDBOpsState.nextID)
		adminDBOpsState.nextID++
		adminDBOpsState.mu.Unlock()

		auditSvc.Append("dbops", "sql_execute", execID)
		respondJSON(w, http.StatusOK, controlledSQLResponse{Allowed: true, ExecutionID: execID, SafetyResult: "approved"})
	}
}

func submitReleaseEvidenceHandler(svc ReleaseService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req submitReleaseEvidenceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		out, err := svc.SubmitEvidence(releasegov.EvidenceInput{
			Milestone:           req.Milestone,
			GoTestPassed:        req.GoTestPassed,
			GoRacePassed:        req.GoRacePassed,
			ReadmeSynced:        req.ReadmeSynced,
			BenchmarkNsPerOp:    req.BenchmarkNsPerOp,
			BaselineNsPerOp:     req.BaselineNsPerOp,
			BenchmarkCommand:    req.BenchmarkCommand,
			EvidenceDescription: req.EvidenceDescription,
		}, time.Now().UTC())
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("release-governance", "evidence_submit", out.Milestone)
		respondJSON(w, http.StatusCreated, out)
	}
}

func getReleaseScorecardHandler(svc ReleaseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		milestone, err := parsePathString(r, "milestone")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		allowedRegression := 0.10
		if raw := strings.TrimSpace(r.URL.Query().Get("allowed_regression")); raw != "" {
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil || parsed <= 0 {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "allowed_regression must be a positive float"})
				return
			}
			allowedRegression = parsed
		}
		respondJSON(w, http.StatusOK, svc.Scorecard(milestone, allowedRegression))
	}
}

func listPluginsHandler(svc PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func getPluginHandler(svc PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, err := parsePathString(r, "name")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		item, err := svc.Get(name)
		if err != nil {
			if err == pluginmgr.ErrPluginNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		respondJSON(w, http.StatusOK, item)
	}
}

func enablePluginHandler(svc PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, err := parsePathString(r, "name")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		item, err := svc.Enable(name)
		if err != nil {
			if err == pluginmgr.ErrPluginNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		respondJSON(w, http.StatusOK, item)
	}
}

func disablePluginHandler(svc PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, err := parsePathString(r, "name")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		item, err := svc.Disable(name)
		if err != nil {
			if err == pluginmgr.ErrPluginNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		respondJSON(w, http.StatusOK, item)
	}
}

func checkPluginVersionHandler(svc PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, err := parsePathString(r, "name")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		var req pluginVersionCheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		result, err := svc.CheckVersion(name, strings.TrimSpace(req.LatestVersion))
		if err != nil {
			if err == pluginmgr.ErrPluginNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, result)
	}
}

func listRegisteredAPIsHandler(svc APIRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, listAPIsResponse{Items: svc.List()})
	}
}

func systemStatusHandler(services AdminModuleServices) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, collectSystemStatus(services))
	}
}

func runtimeMetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, collectRuntimeMetrics())
	}
}

func nodeHealthHandler(services AdminModuleServices) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, collectNodeHealth(services))
	}
}

func dashboardAggregateHandler(services AdminModuleServices) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authSession := collectDashboardAuthSessionContext(r)
		authObservability := collectDashboardAuthObservability()
		jwtSession := collectDashboardJWTSessionBootstrap(r, time.Now().UTC())
		respondJSON(w, http.StatusOK, dashboardAggregateResponse{
			Contract:            collectDashboardContractDescriptor(),
			GeneratedAtUnixSec:  time.Now().UTC().Unix(),
			AuthSession:         authSession,
			AuthObservability:   authObservability,
			AuthActionability:   collectDashboardAuthActionability(authSession, authObservability),
			JWTSessionBootstrap: jwtSession,
			Status:              collectSystemStatus(services),
			RuntimeMetrics:      collectRuntimeMetrics(),
			NodeHealth:          collectNodeHealth(services),
		})
	}
}

func collectDashboardContractDescriptor() dashboardContractDescriptor {
	return dashboardContractDescriptor{
		Name:      "dashboard-ui-bootstrap",
		Version:   dashboardUIBootstrapContractVersion,
		Stability: "stable",
		RequiredSections: []string{
			"auth_session",
			"auth_observability",
			"auth_actionability",
			"jwt_session_bootstrap",
			"status",
			"runtime_metrics",
			"node_health",
		},
	}
}

func collectDashboardJWTSessionBootstrap(r *http.Request, now time.Time) dashboardJWTSessionBootstrap {
	middlewareBridge := collectDashboardJWTMiddlewareBridge(r)
	claimsTrusted := middlewareBridge.Verified
	buildAuditExport := func(verificationState string) dashboardJWTProvenanceAuditExport {
		return collectDashboardJWTProvenanceAuditExport(middlewareBridge, verificationState, claimsTrusted)
	}

	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("none", "none", claimsTrusted, "")
		return dashboardJWTSessionBootstrap{
			TokenPresent:          false,
			TokenFormat:           "none",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "none",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			Expired:               false,
			RefreshRecommended:    false,
		}
	}

	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("unsupported", "invalid", claimsTrusted, "authorization header must use bearer scheme")
		return dashboardJWTSessionBootstrap{
			TokenPresent:          true,
			TokenFormat:           "unsupported",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "invalid",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			RefreshRecommended:    true,
			RefreshReason:         "authorization_header_invalid",
			ParseError:            "authorization header must use bearer scheme",
		}
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("unsupported", "invalid", claimsTrusted, "bearer token is empty")
		return dashboardJWTSessionBootstrap{
			TokenPresent:          true,
			TokenFormat:           "unsupported",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "invalid",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			RefreshRecommended:    true,
			RefreshReason:         "authorization_header_invalid",
			ParseError:            "bearer token is empty",
		}
	}

	if strings.Count(token, ".") != 2 {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("bearer-non-jwt", "invalid", claimsTrusted, "token does not match jwt compact format")
		return dashboardJWTSessionBootstrap{
			TokenPresent:          true,
			TokenFormat:           "bearer-non-jwt",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "invalid",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			RefreshRecommended:    true,
			RefreshReason:         "token_not_jwt",
			ParseError:            "token does not match jwt compact format",
		}
	}

	claims, err := decodeJWTClaims(token)
	if err != nil {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("bearer-jwt", "invalid", claimsTrusted, err.Error())
		return dashboardJWTSessionBootstrap{
			TokenPresent:          true,
			TokenFormat:           "bearer-jwt",
			ClaimsTrusted:         claimsTrusted,
			MiddlewareBridge:      middlewareBridge,
			ProvenanceAuditExport: buildAuditExport(verificationState),
			SessionState:          "invalid",
			VerificationState:     verificationState,
			VerificationHint:      verificationHint,
			TrustLevel:            trustLevel,
			TrustMessage:          trustMessage,
			RefreshRecommended:    true,
			RefreshReason:         "jwt_parse_error",
			ParseError:            err.Error(),
		}
	}

	expiresAt := claimInt64(claims, "exp")
	expiresInSec := int64(0)
	refreshAfterUnixSec := int64(0)
	refreshRecommended := false
	refreshReason := ""
	sessionState := "active"

	if expiresAt <= 0 {
		sessionState = "no-exp"
		refreshRecommended = true
		refreshReason = "exp_claim_missing"
	} else {
		expiresInSec = expiresAt - now.Unix()
		if expiresInSec <= 0 {
			sessionState = "expired"
			refreshRecommended = true
			refreshReason = "token_expired"
		} else if expiresInSec <= int64(dashboardJWTRefreshLeadWindow/time.Second) {
			sessionState = "expiring"
			refreshRecommended = true
			refreshReason = "token_expiring_soon"
			refreshAfterUnixSec = now.Unix()
		} else {
			refreshAfterUnixSec = expiresAt - int64(dashboardJWTRefreshLeadWindow/time.Second)
		}
	}

	verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("bearer-jwt", sessionState, claimsTrusted, "")

	return dashboardJWTSessionBootstrap{
		TokenPresent:          true,
		TokenFormat:           "bearer-jwt",
		ClaimsTrusted:         claimsTrusted,
		MiddlewareBridge:      middlewareBridge,
		ProvenanceAuditExport: buildAuditExport(verificationState),
		SessionState:          sessionState,
		VerificationState:     verificationState,
		VerificationHint:      verificationHint,
		TrustLevel:            trustLevel,
		TrustMessage:          trustMessage,
		Subject:               claimString(claims, "sub"),
		Issuer:                claimString(claims, "iss"),
		Audience:              claimAudience(claims, "aud"),
		IssuedAtUnixSec:       claimInt64(claims, "iat"),
		ExpiresAtUnixSec:      expiresAt,
		ExpiresInSec:          expiresInSec,
		RefreshAfterUnixSec:   refreshAfterUnixSec,
		RefreshRecommended:    refreshRecommended,
		RefreshReason:         refreshReason,
		Expired:               expiresAt > 0 && now.Unix() >= expiresAt,
	}
}

func collectDashboardJWTProvenanceAuditExport(bridge dashboardJWTMiddlewareBridge, verificationState string, claimsTrusted bool) dashboardJWTProvenanceAuditExport {
	sourcePath := ""
	if len(bridge.SourceProvenance) > 0 {
		sourcePath = strings.Join(bridge.SourceProvenance, ">")
	}
	enabled := bridge.Present || len(bridge.SourceProvenance) > 0

	export := dashboardJWTProvenanceAuditExport{
		Enabled:             enabled,
		SourceProvenance:    append([]string(nil), bridge.SourceProvenance...),
		SourcePath:          sourcePath,
		Source:              bridge.Source,
		Verified:            bridge.Verified,
		ClaimsTrusted:       claimsTrusted,
		VerificationState:   verificationState,
		Subject:             bridge.Subject,
		RoleID:              bridge.RoleID,
		ClaimsVersion:       bridge.ClaimsVersion,
		RoleSource:          bridge.RoleSource,
		SubjectSource:       bridge.SubjectSource,
		ClaimsVersionSource: bridge.ClaimsVersionSource,
		VerifiedSource:      bridge.VerifiedSource,
	}

	hints := deriveDashboardJWTProvenanceAlertingHints(export)
	observeDashboardJWTProvenanceAuditExport(export, hints)
	opsMetrics := snapshotDashboardJWTProvenanceOperationalMetrics()
	export.OperationalMetrics = opsMetrics
	export.SLODashboard = buildDashboardJWTProvenanceSLODashboard(opsMetrics)
	export.ErrorBudgetPolicy = buildDashboardJWTProvenanceErrorBudgetPolicy(export.SLODashboard)
	export.AlertingHints = hints
	return export
}

func deriveJWTVerificationHints(tokenFormat, sessionState string, claimsTrusted bool, parseError string) (string, string, string, string) {
	if claimsTrusted {
		return "verified", "Claims are verified by trusted middleware.", "trusted", "JWT claims verified; safe for authorization-adjacent UX hints."
	}

	if parseError != "" || sessionState == "invalid" || tokenFormat == "unsupported" || tokenFormat == "bearer-non-jwt" {
		return "invalid", "Token cannot be verified from dashboard bootstrap context.", "untrusted", "Treat token payload as untrusted input and request re-authentication."
	}

	if tokenFormat == "none" || sessionState == "none" {
		return "not_present", "No bearer token provided for JWT bootstrap context.", "none", "No JWT trust context available."
	}

	return "unverified", "JWT payload is parsed without signature validation in bootstrap mode.", "low", "Display identity hints only; defer privileged actions until backend-verified session checks succeed."
}

func collectDashboardJWTMiddlewareBridge(r *http.Request) dashboardJWTMiddlewareBridge {
	claims := resolveAdminVerifiedClaims(r)

	return dashboardJWTMiddlewareBridge{
		Present:             claims.Present,
		Verified:            claims.Verified,
		Source:              claims.Source,
		SourceProvenance:    claims.SourceProvenance,
		Subject:             claims.Subject,
		SubjectSource:       claims.SubjectSource,
		RoleID:              claims.RoleID,
		RoleSource:          claims.RoleSource,
		ClaimsVersion:       claims.ClaimsVersion,
		ClaimsVersionSource: claims.ClaimsVersionSource,
		VerifiedSource:      claims.VerifiedSource,
	}
}

func decodeJWTClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt segments")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid jwt payload encoding")
	}
	claims := make(map[string]any)
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid jwt payload json")
	}
	return claims, nil
}

func claimString(claims map[string]any, key string) string {
	if claims == nil {
		return ""
	}
	if value, ok := claims[key].(string); ok {
		return value
	}
	return ""
}

func claimInt64(claims map[string]any, key string) int64 {
	if claims == nil {
		return 0
	}
	switch value := claims[key].(type) {
	case float64:
		return int64(value)
	case json.Number:
		n, err := value.Int64()
		if err == nil {
			return n
		}
	}
	return 0
}

func claimAudience(claims map[string]any, key string) []string {
	if claims == nil {
		return nil
	}
	value, ok := claims[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return nil
}

func collectDashboardAuthSessionContext(r *http.Request) dashboardAuthSessionContext {
	tokenPresent := strings.TrimSpace(r.Header.Get(adminauth.HeaderToken)) != ""
	signaturePresent := strings.TrimSpace(r.Header.Get(adminauth.HeaderSignature)) != ""
	mode := "none"
	authenticated := false

	if signaturePresent {
		mode = "hmac-sha256"
		authenticated = true
	} else if tokenPresent {
		mode = "static-token"
		authenticated = true
	}

	roleID := strings.TrimSpace(r.Header.Get(HeaderAdminRoleID))

	return dashboardAuthSessionContext{
		Authenticated:          authenticated,
		AuthModeHint:           mode,
		RoleID:                 roleID,
		HasRoleBinding:         roleID != "",
		TokenHeaderPresent:     tokenPresent,
		SignatureHeaderPresent: signaturePresent,
	}
}

func collectDashboardAuthObservability() dashboardAuthObservability {
	snapshot := adminauth.Snapshot()
	return dashboardAuthObservability{
		StaticToken: dashboardAuthModeCounters{
			Success: snapshot.StaticToken.Success,
			Failure: snapshot.StaticToken.Failure,
			Reasons: snapshot.StaticToken.Reasons,
		},
		HMACSHA256: dashboardAuthModeCounters{
			Success: snapshot.HMACSHA256.Success,
			Failure: snapshot.HMACSHA256.Failure,
			Reasons: snapshot.HMACSHA256.Reasons,
		},
		TotalFailure: snapshot.StaticToken.Failure + snapshot.HMACSHA256.Failure,
	}
}

func collectDashboardAuthActionability(authSession dashboardAuthSessionContext, authObs dashboardAuthObservability) dashboardAuthActionability {
	totalSuccess := authObs.StaticToken.Success + authObs.HMACSHA256.Success
	totalAttempts := totalSuccess + authObs.TotalFailure
	failureRate := 0.0
	if totalAttempts > 0 {
		failureRate = float64(authObs.TotalFailure) / float64(totalAttempts)
		failureRate = math.Round(failureRate*10000) / 10000
	}

	severity := "info"
	switch {
	case authObs.TotalFailure >= 20 || failureRate >= 0.30:
		severity = "critical"
	case authObs.TotalFailure > 0 || failureRate >= 0.10:
		severity = "warning"
	}

	nextActions := make([]string, 0, 4)
	switch authSession.AuthModeHint {
	case "none":
		nextActions = append(nextActions,
			"Enable admin auth for dashboard entry traffic before external exposure.",
			"Use hmac-sha256 as the preferred auth mode for production environments.",
		)
		if severity == "info" {
			severity = "warning"
		}
	case "static-token":
		nextActions = append(nextActions,
			"Plan migration from static-token to hmac-sha256 for stronger replay-resistant protection.",
			"Rotate static token on a fixed cadence and after any suspicious access pattern.",
		)
		if severity == "info" {
			severity = "warning"
		}
	case "hmac-sha256":
		nextActions = append(nextActions,
			"Keep SKOLL_ADMIN_AUTH_HMAC_SECRET rotation on the defined runbook cadence.",
		)
	}

	if authSession.Authenticated && !authSession.HasRoleBinding {
		nextActions = append(nextActions,
			"Attach role binding context for authenticated dashboard requests to unlock RBAC-aware UX decisions.",
		)
		if severity == "info" {
			severity = "warning"
		}
	}

	if authObs.TotalFailure > 0 {
		nextActions = append(nextActions,
			"Review top auth failure reasons and tune alerts for replay_nonce/missing_headers spikes.",
		)
	}

	if len(nextActions) == 0 {
		nextActions = append(nextActions, "No immediate auth/session action required.")
	}

	return dashboardAuthActionability{
		Severity:            severity,
		RecommendedAuthMode: "hmac-sha256",
		FailureRate:         failureRate,
		TopFailureReasons:   topDashboardFailureReasons(authObs),
		NextActions:         nextActions,
		Docs: []string{
			"docs/community/DASHBOARD_AUTH_SESSION_POLICY.md",
			"docs/planning/ADMIN_AUTH_SECURITY_RUNBOOK.md",
		},
	}
}

func topDashboardFailureReasons(authObs dashboardAuthObservability) []string {
	reasons := make([]string, 0, len(authObs.StaticToken.Reasons)+len(authObs.HMACSHA256.Reasons))
	for reason, count := range authObs.StaticToken.Reasons {
		if count > 0 {
			reasons = append(reasons, fmt.Sprintf("static-token:%s=%d", reason, count))
		}
	}
	for reason, count := range authObs.HMACSHA256.Reasons {
		if count > 0 {
			reasons = append(reasons, fmt.Sprintf("hmac-sha256:%s=%d", reason, count))
		}
	}
	if len(reasons) == 0 {
		return []string{}
	}
	sort.Strings(reasons)
	if len(reasons) > 5 {
		return reasons[:5]
	}
	return reasons
}

func collectSystemStatus(services AdminModuleServices) systemStatusResponse {
	return systemStatusResponse{
		Version:       "v1",
		UserCount:     len(services.Users.List()),
		RoleCount:     len(services.Roles.List()),
		MenuCount:     len(services.Menus.List()),
		ConfigCount:   len(services.Configs.List()),
		DictCount:     len(services.Dictionaries.List()),
		FileCount:     len(services.Files.List()),
		JobCount:      len(services.Jobs.List()),
		PluginCount:   len(services.Plugins.List()),
		APIEntryCount: len(services.APIs.List()),
	}
}

func collectRuntimeMetrics() runtimeMetricsResponse {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	lastPause := uint64(0)
	if mem.NumGC > 0 {
		lastPause = mem.PauseNs[(mem.NumGC-1)%uint32(len(mem.PauseNs))]
	}
	now := time.Now().UTC()
	uptime := now.Sub(adminModuleStartTime)
	if uptime < 0 {
		uptime = 0
	}
	return runtimeMetricsResponse{
		UptimeSeconds:   int64(uptime / time.Second),
		Goroutines:      runtime.NumGoroutine(),
		MemoryAlloc:     mem.Alloc,
		MemorySys:       mem.Sys,
		HeapAlloc:       mem.HeapAlloc,
		HeapSys:         mem.HeapSys,
		TotalAlloc:      mem.TotalAlloc,
		GCCount:         mem.NumGC,
		LastGCPauseNs:   lastPause,
		SnapshotUnixSec: now.Unix(),
	}
}

func collectNodeHealth(services AdminModuleServices) nodeHealthResponse {
	now := time.Now().UTC().Unix()
	deps := []healthCheckItem{
		{Name: "users", Status: healthStatus(services.Users != nil), Detail: "admin user service", CheckedAt: now},
		{Name: "roles", Status: healthStatus(services.Roles != nil), Detail: "admin role service", CheckedAt: now},
		{Name: "menus", Status: healthStatus(services.Menus != nil), Detail: "admin menu service", CheckedAt: now},
		{Name: "audit", Status: healthStatus(services.Audit != nil), Detail: "admin audit service", CheckedAt: now},
		{Name: "configs", Status: healthStatus(services.Configs != nil), Detail: "admin config service", CheckedAt: now},
		{Name: "dictionaries", Status: healthStatus(services.Dictionaries != nil), Detail: "admin dictionary service", CheckedAt: now},
		{Name: "files", Status: healthStatus(services.Files != nil), Detail: "admin file service", CheckedAt: now},
		{Name: "jobs", Status: healthStatus(services.Jobs != nil), Detail: "admin job service", CheckedAt: now},
		{Name: "generator", Status: healthStatus(services.Generator != nil), Detail: "admin generator service", CheckedAt: now},
		{Name: "plugins", Status: healthStatus(services.Plugins != nil), Detail: "admin plugin service", CheckedAt: now},
		{Name: "rbac", Status: healthStatus(services.RBAC != nil), Detail: "admin rbac service", CheckedAt: now},
		{Name: "api_registry", Status: healthStatus(services.APIs != nil), Detail: "admin api registry service", CheckedAt: now},
		{Name: "release_governance", Status: healthStatus(services.Releases != nil), Detail: "admin release governance service", CheckedAt: now},
	}
	nodeStatus := "up"
	for _, item := range deps {
		if item.Status != "up" {
			nodeStatus = "degraded"
			break
		}
	}
	return nodeHealthResponse{NodeStatus: nodeStatus, CheckedAt: now, Dependencies: deps}
}

func healthStatus(ok bool) string {
	if ok {
		return "up"
	}
	return "down"
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return id, nil
}

func parsePathString(r *http.Request, key string) (string, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return raw, nil
}

func parseAuditQuery(r *http.Request) (audit.Query, error) {
	q := audit.Query{
		Page:   audit.DefaultPage,
		Size:   audit.DefaultSize,
		Actor:  strings.TrimSpace(r.URL.Query().Get("actor")),
		Action: strings.TrimSpace(r.URL.Query().Get("action")),
		Target: strings.TrimSpace(r.URL.Query().Get("target")),
		Q:      strings.TrimSpace(r.URL.Query().Get("q")),
	}

	if rawPage := strings.TrimSpace(r.URL.Query().Get("page")); rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed <= 0 {
			return audit.Query{}, fmt.Errorf("page must be a positive integer")
		}
		q.Page = parsed
	}

	if rawSize := strings.TrimSpace(r.URL.Query().Get("size")); rawSize != "" {
		parsed, err := strconv.Atoi(rawSize)
		if err != nil || parsed <= 0 {
			return audit.Query{}, fmt.Errorf("size must be a positive integer")
		}
		q.Size = parsed
	}

	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			return audit.Query{}, fmt.Errorf("limit must be a positive integer")
		}
		q.Page = 1
		q.Size = parsed
	}

	return q, nil
}
