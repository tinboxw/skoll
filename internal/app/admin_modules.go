package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	RenewClaimLease(executionKey, instanceID string, leaseTTLSeconds int64, now time.Time) (jobscheduler.DispatchClaim, error)
	ClaimStatus(executionKey string) jobscheduler.DispatchClaim
	SetRetryPolicy(jobID int64, policy jobscheduler.RetryPolicy) (jobscheduler.RetryPolicy, error)
	GetRetryPolicy(jobID int64) (jobscheduler.RetryPolicy, error)
	ScheduleRetry(jobID int64, executionKey string, attempt int, now time.Time) (jobscheduler.RetrySchedule, error)
	MarkDeadLetter(jobID int64, executionKey, reason string, retryCount int, now time.Time) (jobscheduler.DeadLetter, error)
	ListDeadLetters(limit int) []jobscheduler.DeadLetter
	ReplayDeadLetter(executionKey, operator string, now time.Time) (jobscheduler.DeadLetter, error)
	ReliabilitySnapshot(now time.Time) jobscheduler.ReliabilityMetrics
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
	Remove(name string) pluginmgr.LifecycleResult
	CheckCompatibility(name, version string, dependencies []pluginmgr.Dependency) pluginmgr.CompatibilityResult
	RegisterHook(name, namespace, version string, order, timeoutMillis, retryLimit int, deadLetter bool) (pluginmgr.HookRegistration, error)
	ListHooks() []pluginmgr.HookRegistration
	SetHookEnabled(name, namespace string, enabled bool) (pluginmgr.HookRegistration, error)
	SetHookOrder(name, namespace string, order int) (pluginmgr.HookRegistration, error)
	SetHookRuntimePolicy(name, namespace string, timeoutMillis, retryLimit int, deadLetter bool) (pluginmgr.HookRegistration, error)
	ExecuteHookDiagnostic(name, namespace string, failTimes int) (pluginmgr.HookExecutionResult, error)
	ListHookDeadLetters() []pluginmgr.HookDeadLetterRecord
	SetMarketplaceTrustRoots(roots []string) []string
	ListMarketplaceTrustRoots() []string
	IngestMarketplaceIndex(source, signedBy, signature string, expiresAt time.Time, packages []pluginmgr.MarketplaceIndexPackage, now time.Time) (pluginmgr.MarketplaceIndexIngestResult, error)
	ListMarketplaceIndexSources() []pluginmgr.MarketplaceIndexSource
	SolveDependencies(items []pluginmgr.DependencySolveItem) pluginmgr.DependencySolveResult
	UpgradePackageTransactional(transactionID, name, targetVersion, packageURL, packageHash, signature string, dependencies []pluginmgr.Dependency, hooks []string, now time.Time) (pluginmgr.UpgradeTransactionResult, error)
	ListUpgradeProvenance(limit int) []pluginmgr.UpgradeProvenanceRecord
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
	handle("POST /admin/v1/users/bulk", createUsersBulkHandler(services.Users, services.Audit))
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
	handle("GET /admin/v1/roles/{id}/policies/persistence/export", exportRolePolicyPersistenceHandler(services.Roles, services.RBAC))
	handle("POST /admin/v1/roles/{id}/policies/persistence/import", importRolePolicyPersistenceHandler(services.Roles, services.Menus, services.RBAC, services.APIs, services.Audit))
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
	handle("GET /admin/v1/audit-logs/profile", auditQueryProfileHandler())
	handle("GET /admin/v1/admin-ops/control-profile", adminOpsControlProfileHandler())
	handle("POST /admin/v1/configs", upsertConfigHandler(services.Configs))
	handle("POST /admin/v1/configs/bulk", upsertConfigsBulkHandler(services.Configs, services.Audit))
	handle("GET /admin/v1/configs", listConfigsHandler(services.Configs))
	handle("GET /admin/v1/configs/query", listConfigsQueryHandler(services.Configs))
	handle("GET /admin/v1/configs/{key}", getConfigHandler(services.Configs))
	handle("POST /admin/v1/dictionaries", createDictionaryHandler(services.Dictionaries))
	handle("POST /admin/v1/dictionaries/bulk", createDictionariesBulkHandler(services.Dictionaries, services.Audit))
	handle("GET /admin/v1/dictionaries", listDictionariesHandler(services.Dictionaries))
	handle("GET /admin/v1/dictionaries/query", listDictionariesQueryHandler(services.Dictionaries))
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
	handle("POST /admin/v1/job-dispatch-claims/{execution_key}/renew", renewJobDispatchClaimHandler(services.Jobs, services.Audit))
	handle("GET /admin/v1/job-dispatch-claims/{execution_key}", getJobDispatchClaimHandler(services.Jobs))
	handle("PUT /admin/v1/jobs/{id}/retry-policy", setJobRetryPolicyHandler(services.Jobs, services.Audit))
	handle("GET /admin/v1/jobs/{id}/retry-policy", getJobRetryPolicyHandler(services.Jobs))
	handle("POST /admin/v1/jobs/{id}/retries/schedule", scheduleJobRetryHandler(services.Jobs, services.Audit))
	handle("POST /admin/v1/jobs/{id}/dead-letters", markJobDeadLetterHandler(services.Jobs, services.Audit))
	handle("GET /admin/v1/jobs/dead-letters", listJobDeadLettersHandler(services.Jobs))
	handle("POST /admin/v1/jobs/dead-letters/{execution_key}/replay", replayJobDeadLetterHandler(services.Jobs, services.Audit))
	handle("GET /admin/v1/jobs/reliability/metrics", jobReliabilityMetricsHandler(services.Jobs))
	handle("POST /admin/v1/generator/modules", generateModuleHandler(services.Generator))
	handle("POST /admin/v1/plugins/manifests", installPluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/packages/install", installPluginPackageHandler(services.Plugins))
	handle("GET /admin/v1/plugins", listPluginsHandler(services.Plugins))
	handle("GET /admin/v1/plugins/{name}", getPluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/enable", enablePluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/disable", disablePluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/version-check", checkPluginVersionHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/upgrade", upgradePluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/remove", removePluginHandler(services.Plugins, services.Audit))
	handle("POST /admin/v1/plugins/compatibility-check", checkPluginCompatibilityHandler(services.Plugins, services.Audit))
	handle("POST /admin/v1/plugins/hooks/register", registerPluginHookHandler(services.Plugins, services.Audit))
	handle("GET /admin/v1/plugins/hooks", listPluginHooksHandler(services.Plugins))
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/enable", setPluginHookEnabledHandler(services.Plugins, services.Audit, true))
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/disable", setPluginHookEnabledHandler(services.Plugins, services.Audit, false))
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/order", setPluginHookOrderHandler(services.Plugins, services.Audit))
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/runtime", setPluginHookRuntimeHandler(services.Plugins, services.Audit))
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/execute-diagnostic", executePluginHookDiagnosticHandler(services.Plugins, services.Audit))
	handle("GET /admin/v1/plugins/hooks/dead-letters", listPluginHookDeadLettersHandler(services.Plugins))
	handle("PUT /admin/v1/plugins/marketplace/trust-roots", setMarketplaceTrustRootsHandler(services.Plugins, services.Audit))
	handle("GET /admin/v1/plugins/marketplace/trust-roots", listMarketplaceTrustRootsHandler(services.Plugins))
	handle("POST /admin/v1/plugins/marketplace/index/ingest", ingestMarketplaceIndexHandler(services.Plugins, services.Audit))
	handle("GET /admin/v1/plugins/marketplace/index/sources", listMarketplaceIndexSourcesHandler(services.Plugins))
	handle("POST /admin/v1/plugins/dependency-solver/resolve", resolvePluginDependenciesHandler(services.Plugins, services.Audit))
	handle("POST /admin/v1/plugins/{name}/upgrade/transaction", upgradePluginTransactionalHandler(services.Plugins, services.Audit))
	handle("GET /admin/v1/plugins/upgrade/provenance", listPluginUpgradeProvenanceHandler(services.Plugins))
	handle("POST /admin/v1/db/migrations/plan", planDatabaseMigrationHandler(services.Audit))
	handle("POST /admin/v1/db/migrations/drift-detect", detectDatabaseMigrationDriftHandler(services.Audit))
	handle("POST /admin/v1/db/backup", backupDatabaseHandler(services.Audit))
	handle("GET /admin/v1/db/backups/catalog", listBackupCatalogHandler())
	handle("POST /admin/v1/db/restore", restoreDatabaseHandler(services.Audit))
	handle("POST /admin/v1/db/restore/drills", executeRestoreDrillHandler(services.Audit))
	handle("GET /admin/v1/db/restore/drills", listRestoreDrillEvidenceHandler())
	handle("POST /admin/v1/db/sql/execute", executeControlledSQLHandler(services.Audit))
	handle("GET /admin/v1/system/status", systemStatusHandler(services))
	handle("GET /admin/v1/system/runtime-metrics", runtimeMetricsHandler())
	handle("GET /admin/v1/system/node-health", nodeHealthHandler(services))
	handle("GET /admin/v1/system/dashboard", dashboardAggregateHandler(services))
	handle("PUT /admin/v1/system/hardening/endpoint-guardrails", setEndpointGuardrailsHandler(services.Audit))
	handle("GET /admin/v1/system/hardening/endpoint-guardrails", listEndpointGuardrailsHandler())
	handle("PUT /admin/v1/system/hardening/alert-profiles", setAlertProfilesHandler(services.Audit))
	handle("GET /admin/v1/system/hardening/alert-profiles", listAlertProfilesHandler())
	handle("PUT /admin/v1/system/hardening/incident-runbooks", setIncidentRunbooksHandler(services.Audit))
	handle("GET /admin/v1/system/hardening/incident-runbooks", listIncidentRunbooksHandler())
	handle("POST /admin/v1/system/hardening/fault-drills", createFaultDrillHandler(services.Audit))
	handle("GET /admin/v1/system/hardening/fault-drills", listFaultDrillsHandler())
	handle("GET /admin/v1/apis", listRegisteredAPIsHandler(services.APIs))
	if services.Releases != nil {
		handle("POST /admin/v1/release-governance/evidence", submitReleaseEvidenceHandler(services.Releases, services.Audit))
		handle("GET /admin/v1/release-governance/scorecard/{milestone}", getReleaseScorecardHandler(services.Releases))
		handle("PUT /admin/v1/release-governance/blocking-policy", setReleaseBlockingPolicyHandler(services.Audit))
		handle("GET /admin/v1/release-governance/blocking-policy", getReleaseBlockingPolicyHandler())
		handle("GET /admin/v1/release-governance/block-decision/{milestone}", getReleaseBlockDecisionHandler(services.Releases))
		handle("PUT /admin/v1/release-governance/parity-closure/checkpoints", setParityClosureCheckpointsHandler(services.Audit))
		handle("GET /admin/v1/release-governance/parity-closure/checkpoints", listParityClosureCheckpointsHandler())
		handle("GET /admin/v1/release-governance/parity-closure/report", getParityClosureReportHandler())
	}
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type createUsersBulkRequest struct {
	Items []createUserRequest `json:"items"`
}

type createUsersBulkResponse struct {
	Atomic bool        `json:"atomic"`
	Count  int         `json:"count"`
	Items  []user.User `json:"items"`
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

type auditQueryProfileResponse struct {
	DefaultPage      int      `json:"default_page"`
	DefaultSize      int      `json:"default_size"`
	MaxSize          int      `json:"max_size"`
	TargetP95Millis  int      `json:"target_p95_millis"`
	SupportedFilters []string `json:"supported_filters"`
}

type adminOpsControlProfileResponse struct {
	AuditRetentionDays int      `json:"audit_retention_days"`
	AuditArchiveDays   int      `json:"audit_archive_days"`
	ArchiveBatchSize   int      `json:"archive_batch_size"`
	RateGuardHints     []string `json:"rate_guard_hints"`
}

type upsertConfigRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type upsertConfigsBulkRequest struct {
	Items []upsertConfigRequest `json:"items"`
}

type upsertConfigsBulkResponse struct {
	Atomic bool           `json:"atomic"`
	Count  int            `json:"count"`
	Items  []config.Entry `json:"items"`
}

type configQueryResponse struct {
	Items   []config.Entry `json:"items"`
	Page    int            `json:"page"`
	Size    int            `json:"size"`
	Total   int            `json:"total"`
	HasNext bool           `json:"has_next"`
}

type createDictionaryRequest struct {
	Type    string `json:"type"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	Sort    int    `json:"sort"`
	Enabled *bool  `json:"enabled"`
}

type createDictionariesBulkRequest struct {
	Items []createDictionaryRequest `json:"items"`
}

type createDictionariesBulkResponse struct {
	Atomic bool              `json:"atomic"`
	Count  int               `json:"count"`
	Items  []dictionary.Item `json:"items"`
}

type dictionaryQueryResponse struct {
	Items   []dictionary.Item `json:"items"`
	Page    int               `json:"page"`
	Size    int               `json:"size"`
	Total   int               `json:"total"`
	HasNext bool              `json:"has_next"`
}

type createJobRequest struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
}

type claimJobDispatchRequest struct {
	ExecutionKey string `json:"execution_key"`
	InstanceID   string `json:"instance_id"`
}

type renewJobDispatchClaimRequest struct {
	InstanceID     string `json:"instance_id"`
	LeaseTTLSecond int64  `json:"lease_ttl_sec"`
}

type setJobRetryPolicyRequest struct {
	MaxRetries        int `json:"max_retries"`
	BackoffBaseMillis int `json:"backoff_base_millis"`
	BackoffMaxMillis  int `json:"backoff_max_millis"`
	JitterPercent     int `json:"jitter_percent"`
}

type scheduleJobRetryRequest struct {
	ExecutionKey string `json:"execution_key"`
	Attempt      int    `json:"attempt"`
}

type markJobDeadLetterRequest struct {
	ExecutionKey string `json:"execution_key"`
	Reason       string `json:"reason"`
	RetryCount   int    `json:"retry_count"`
}

type replayJobDeadLetterRequest struct {
	Operator string `json:"operator"`
}

type generateModuleRequest struct {
	Module          string                   `json:"module"`
	TemplateVersion string                   `json:"template_version"`
	FormSchema      *modgenerator.FormSchema `json:"form_schema"`
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
	Contract             dashboardContractDescriptor     `json:"contract"`
	GeneratedAtUnixSec   int64                           `json:"generated_at_unix_sec"`
	AuthSession          dashboardAuthSessionContext     `json:"auth_session"`
	AuthObservability    dashboardAuthObservability      `json:"auth_observability"`
	AuthActionability    dashboardAuthActionability      `json:"auth_actionability"`
	JWTSessionBootstrap  dashboardJWTSessionBootstrap    `json:"jwt_session_bootstrap"`
	Status               systemStatusResponse            `json:"status"`
	RuntimeMetrics       runtimeMetricsResponse          `json:"runtime_metrics"`
	NodeHealth           nodeHealthResponse              `json:"node_health"`
	SchedulerReliability jobscheduler.ReliabilityMetrics `json:"scheduler_reliability"`
	HardeningPosture     dashboardHardeningPosture       `json:"hardening_posture"`
}

type dashboardHardeningPosture struct {
	EndpointGuardrailsEnabled int   `json:"endpoint_guardrails_enabled"`
	AlertProfilesEnabled      int   `json:"alert_profiles_enabled"`
	IncidentRunbookProfiles   int   `json:"incident_runbook_profiles"`
	FaultDrillsRecorded       int   `json:"fault_drills_recorded"`
	LatestDrillRecordedAtSec  int64 `json:"latest_drill_recorded_at_unix_sec"`
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

func auditQueryProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, auditQueryProfileResponse{
			DefaultPage:     audit.DefaultPage,
			DefaultSize:     audit.DefaultSize,
			MaxSize:         auditQueryMaxSize,
			TargetP95Millis: 100,
			SupportedFilters: []string{
				"actor",
				"action",
				"target",
				"q",
				"page",
				"size",
				"limit",
			},
		})
	}
}

func adminOpsControlProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, adminOpsControlProfileResponse{
			AuditRetentionDays: 180,
			AuditArchiveDays:   30,
			ArchiveBatchSize:   1000,
			RateGuardHints: []string{
				"actor:security action:policy_snapshot_rollback burst<=2/min",
				"actor:admin action:users_bulk_create burst<=10/min",
				"actor:admin action:configs_bulk_upsert burst<=15/min",
			},
		})
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

func listRegisteredAPIsHandler(svc APIRegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, listAPIsResponse{Items: svc.List()})
	}
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

func uniqueNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, item := range values {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
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
		if parsed > auditQueryMaxSize {
			return audit.Query{}, fmt.Errorf("size must be <= %d", auditQueryMaxSize)
		}
		q.Size = parsed
	}

	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			return audit.Query{}, fmt.Errorf("limit must be a positive integer")
		}
		if parsed > auditQueryMaxSize {
			return audit.Query{}, fmt.Errorf("limit must be <= %d", auditQueryMaxSize)
		}
		q.Page = 1
		q.Size = parsed
	}

	return q, nil
}

const auditQueryMaxSize = 200

func parsePageSizeQuery(r *http.Request, defaultPage, defaultSize, maxSize int) (int, int, error) {
	page := defaultPage
	size := defaultSize
	if rawPage := strings.TrimSpace(r.URL.Query().Get("page")); rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed <= 0 {
			return 0, 0, fmt.Errorf("page must be a positive integer")
		}
		page = parsed
	}
	if rawSize := strings.TrimSpace(r.URL.Query().Get("size")); rawSize != "" {
		parsed, err := strconv.Atoi(rawSize)
		if err != nil || parsed <= 0 {
			return 0, 0, fmt.Errorf("size must be a positive integer")
		}
		if parsed > maxSize {
			return 0, 0, fmt.Errorf("size must be <= %d", maxSize)
		}
		size = parsed
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			return 0, 0, fmt.Errorf("limit must be a positive integer")
		}
		if parsed > maxSize {
			return 0, 0, fmt.Errorf("limit must be <= %d", maxSize)
		}
		page = 1
		size = parsed
	}
	return page, size, nil
}

func paginateSlice[T any](items []T, page, size int) ([]T, int) {
	total := len(items)
	start := (page - 1) * size
	if start >= total {
		return []T{}, total
	}
	end := start + size
	if end > total {
		end = total
	}
	return items[start:end], total
}
