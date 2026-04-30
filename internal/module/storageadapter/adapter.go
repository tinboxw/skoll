package storageadapter

import (
	"os"
	"path/filepath"
	"time"

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

// Adapter groups module repositories behind a single storage boundary.
type Adapter interface {
	Users() UserRepository
	Roles() RoleRepository
	Menus() MenuRepository
	Audit() AuditRepository
	Configs() ConfigRepository
	Dictionaries() DictionaryRepository
	Files() FileRepository
	Jobs() JobRepository
	Generators() GeneratorRepository
	Plugins() PluginRepository
	RBAC() RBACRepository
	APIs() APIRegistryRepository
	Releases() ReleaseRepository
}

type UserRepository interface {
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

type RoleRepository interface {
	Create(name string, permissions []string) role.Role
	Get(id int64) (role.Role, error)
	List() []role.Role
}

type MenuRepository interface {
	Create(title, path string, order int) menu.Item
	Get(id int64) (menu.Item, error)
	List() []menu.Item
}

type AuditRepository interface {
	Append(actor, action, target string) audit.Record
	Recent(limit int) []audit.Record
	Query(q audit.Query) audit.QueryResult
}

type ConfigRepository interface {
	Set(key, value, description string) config.Entry
	Get(key string) (config.Entry, error)
	List() []config.Entry
}

type DictionaryRepository interface {
	Create(itemType, label, value string, sortOrder int, enabled bool) dictionary.Item
	Get(id int64) (dictionary.Item, error)
	List() []dictionary.Item
	ListByType(itemType string) []dictionary.Item
}

type FileRepository interface {
	Upload(name string, content []byte) (fileservice.File, error)
	Get(id int64) (fileservice.File, error)
	List() []fileservice.File
	Download(id int64) (fileservice.File, []byte, error)
}

type JobRepository interface {
	Create(name, schedule string) jobscheduler.Job
	Get(id int64) (jobscheduler.Job, error)
	List() []jobscheduler.Job
	Run(jobID int64) (jobscheduler.Execution, error)
	History(jobID int64, limit int) []jobscheduler.Execution
	ClaimRun(jobID int64, executionKey, instanceID string, now time.Time) (jobscheduler.DispatchClaim, error)
	ClaimStatus(executionKey string) jobscheduler.DispatchClaim
}

type GeneratorRepository interface {
	Generate(module string) (modgenerator.Result, error)
	GenerateWithSchema(module string, schema *modgenerator.FormSchema, templateVersion string) (modgenerator.Result, error)
}

type PluginRepository interface {
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

type RBACRepository interface {
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

type APIRegistryRepository interface {
	RegisterMany(entries []string)
	Exists(entry string) bool
	List() []string
}

type ReleaseRepository interface {
	SubmitEvidence(input releasegov.EvidenceInput, now time.Time) (releasegov.Evidence, error)
	Scorecard(milestone string, allowedRegression float64) releasegov.Scorecard
}

type InMemoryAdapter struct {
	users    UserRepository
	roles    RoleRepository
	menus    MenuRepository
	audit    AuditRepository
	configs  ConfigRepository
	dicts    DictionaryRepository
	files    FileRepository
	jobs     JobRepository
	gen      GeneratorRepository
	plugins  PluginRepository
	rbac     RBACRepository
	apis     APIRegistryRepository
	releases ReleaseRepository
}

func NewInMemoryAdapter() *InMemoryAdapter {
	backend, err := fileservice.NewLocalBackend(filepath.Join(os.TempDir(), "skoll-uploads"))
	if err != nil {
		panic(err)
	}
	return &InMemoryAdapter{
		users:    user.NewService(),
		roles:    role.NewService(),
		menus:    menu.NewService(),
		audit:    audit.NewService(),
		configs:  config.NewService(),
		dicts:    dictionary.NewService(),
		files:    fileservice.NewService(backend),
		jobs:     jobscheduler.NewService(),
		gen:      modgenerator.NewService(),
		plugins:  pluginmgr.NewService(),
		rbac:     rbac.NewService(),
		apis:     apiregistry.NewService(),
		releases: releasegov.NewService(),
	}
}

func (a *InMemoryAdapter) Users() UserRepository {
	return a.users
}

func (a *InMemoryAdapter) Roles() RoleRepository {
	return a.roles
}

func (a *InMemoryAdapter) Menus() MenuRepository {
	return a.menus
}

func (a *InMemoryAdapter) Audit() AuditRepository {
	return a.audit
}

func (a *InMemoryAdapter) Configs() ConfigRepository {
	return a.configs
}

func (a *InMemoryAdapter) Dictionaries() DictionaryRepository {
	return a.dicts
}

func (a *InMemoryAdapter) Files() FileRepository {
	return a.files
}

func (a *InMemoryAdapter) Jobs() JobRepository {
	return a.jobs
}

func (a *InMemoryAdapter) Generators() GeneratorRepository {
	return a.gen
}

func (a *InMemoryAdapter) Plugins() PluginRepository {
	return a.plugins
}

func (a *InMemoryAdapter) RBAC() RBACRepository {
	return a.rbac
}

func (a *InMemoryAdapter) APIs() APIRegistryRepository {
	return a.apis
}

func (a *InMemoryAdapter) Releases() ReleaseRepository {
	return a.releases
}
