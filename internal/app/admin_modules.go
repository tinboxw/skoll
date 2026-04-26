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
}

type UserService interface {
	Create(name, email string) user.User
	Get(id int64) (user.User, error)
	List() []user.User
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
}

type GeneratorService interface {
	Generate(module string) (modgenerator.Result, error)
}

type PluginService interface {
	Install(name, version string, hooks []string) pluginmgr.Manifest
	InstallPackage(name, version, packageURL, packageHash string, hooks []string) (pluginmgr.Manifest, error)
	Get(name string) (pluginmgr.Manifest, error)
	List() []pluginmgr.Manifest
	Enable(name string) (pluginmgr.Manifest, error)
	Disable(name string) (pluginmgr.Manifest, error)
	CheckVersion(name, latestVersion string) (pluginmgr.VersionCheckResult, error)
}

type RBACService interface {
	SetRoleMenus(roleID int64, menuIDs []int64) []int64
	GetRoleMenus(roleID int64) []int64
	SetRoleAPIs(roleID int64, apis []string) []string
	GetRoleAPIs(roleID int64) []string
}

type APIRegistryService interface {
	RegisterMany(entries []string)
	Exists(entry string) bool
	List() []string
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

	handle("POST /admin/v1/users", createUserHandler(services.Users))
	handle("GET /admin/v1/users", listUsersHandler(services.Users))
	handle("GET /admin/v1/users/{id}", getUserHandler(services.Users))

	handle("POST /admin/v1/roles", createRoleHandler(services.Roles))
	handle("GET /admin/v1/roles", listRolesHandler(services.Roles))
	handle("GET /admin/v1/roles/{id}", getRoleHandler(services.Roles))
	handle("PUT /admin/v1/roles/{id}/menus", setRoleMenusHandler(services.Roles, services.Menus, services.RBAC))
	handle("GET /admin/v1/roles/{id}/menus", getRoleMenusHandler(services.Roles, services.RBAC))
	handle("PUT /admin/v1/roles/{id}/apis", setRoleAPIsHandler(services.Roles, services.RBAC, services.APIs))
	handle("GET /admin/v1/roles/{id}/apis", getRoleAPIsHandler(services.Roles, services.RBAC))

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
	handle("POST /admin/v1/generator/modules", generateModuleHandler(services.Generator))
	handle("POST /admin/v1/plugins/manifests", installPluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/packages/install", installPluginPackageHandler(services.Plugins))
	handle("GET /admin/v1/plugins", listPluginsHandler(services.Plugins))
	handle("GET /admin/v1/plugins/{name}", getPluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/enable", enablePluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/disable", disablePluginHandler(services.Plugins))
	handle("POST /admin/v1/plugins/{name}/version-check", checkPluginVersionHandler(services.Plugins))
	handle("GET /admin/v1/system/status", systemStatusHandler(services))
	handle("GET /admin/v1/system/runtime-metrics", runtimeMetricsHandler())
	handle("GET /admin/v1/system/node-health", nodeHealthHandler(services))
	handle("GET /admin/v1/system/dashboard", dashboardAggregateHandler(services))
	handle("GET /admin/v1/apis", listRegisteredAPIsHandler(services.APIs))
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
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

type generateModuleRequest struct {
	Module string `json:"module"`
}

type installPluginRequest struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Hooks   []string `json:"hooks"`
}

type installPluginPackageRequest struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	PackageURL  string   `json:"package_url"`
	PackageHash string   `json:"package_hash"`
	Hooks       []string `json:"hooks"`
}

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
	TokenPresent        bool                         `json:"token_present"`
	TokenFormat         string                       `json:"token_format"`
	ClaimsTrusted       bool                         `json:"claims_trusted"`
	MiddlewareBridge    dashboardJWTMiddlewareBridge `json:"middleware_bridge"`
	SessionState        string                       `json:"session_state"`
	VerificationState   string                       `json:"verification_state"`
	VerificationHint    string                       `json:"verification_hint,omitempty"`
	TrustLevel          string                       `json:"trust_level"`
	TrustMessage        string                       `json:"trust_message"`
	Subject             string                       `json:"subject,omitempty"`
	Issuer              string                       `json:"issuer,omitempty"`
	Audience            []string                     `json:"audience,omitempty"`
	IssuedAtUnixSec     int64                        `json:"issued_at_unix_sec,omitempty"`
	ExpiresAtUnixSec    int64                        `json:"expires_at_unix_sec,omitempty"`
	ExpiresInSec        int64                        `json:"expires_in_sec,omitempty"`
	RefreshAfterUnixSec int64                        `json:"refresh_after_unix_sec,omitempty"`
	RefreshRecommended  bool                         `json:"refresh_recommended"`
	RefreshReason       string                       `json:"refresh_reason,omitempty"`
	Expired             bool                         `json:"expired"`
	ParseError          string                       `json:"parse_error,omitempty"`
}

type dashboardJWTMiddlewareBridge struct {
	Present       bool   `json:"present"`
	Verified      bool   `json:"verified"`
	Source        string `json:"source"`
	Subject       string `json:"subject,omitempty"`
	RoleID        string `json:"role_id,omitempty"`
	ClaimsVersion string `json:"claims_version,omitempty"`
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

func generateModuleHandler(svc GeneratorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req generateModuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		result, err := svc.Generate(strings.TrimSpace(req.Module))
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
		if req.Name == "" || req.Version == "" || req.PackageURL == "" || req.PackageHash == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name, version, package_url and package_hash are required"})
			return
		}
		item, err := svc.InstallPackage(req.Name, req.Version, req.PackageURL, req.PackageHash, req.Hooks)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusCreated, item)
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

	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("none", "none", claimsTrusted, "")
		return dashboardJWTSessionBootstrap{
			TokenPresent:       false,
			TokenFormat:        "none",
			ClaimsTrusted:      claimsTrusted,
			MiddlewareBridge:   middlewareBridge,
			SessionState:       "none",
			VerificationState:  verificationState,
			VerificationHint:   verificationHint,
			TrustLevel:         trustLevel,
			TrustMessage:       trustMessage,
			Expired:            false,
			RefreshRecommended: false,
		}
	}

	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("unsupported", "invalid", claimsTrusted, "authorization header must use bearer scheme")
		return dashboardJWTSessionBootstrap{
			TokenPresent:       true,
			TokenFormat:        "unsupported",
			ClaimsTrusted:      claimsTrusted,
			MiddlewareBridge:   middlewareBridge,
			SessionState:       "invalid",
			VerificationState:  verificationState,
			VerificationHint:   verificationHint,
			TrustLevel:         trustLevel,
			TrustMessage:       trustMessage,
			RefreshRecommended: true,
			RefreshReason:      "authorization_header_invalid",
			ParseError:         "authorization header must use bearer scheme",
		}
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("unsupported", "invalid", claimsTrusted, "bearer token is empty")
		return dashboardJWTSessionBootstrap{
			TokenPresent:       true,
			TokenFormat:        "unsupported",
			ClaimsTrusted:      claimsTrusted,
			MiddlewareBridge:   middlewareBridge,
			SessionState:       "invalid",
			VerificationState:  verificationState,
			VerificationHint:   verificationHint,
			TrustLevel:         trustLevel,
			TrustMessage:       trustMessage,
			RefreshRecommended: true,
			RefreshReason:      "authorization_header_invalid",
			ParseError:         "bearer token is empty",
		}
	}

	if strings.Count(token, ".") != 2 {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("bearer-non-jwt", "invalid", claimsTrusted, "token does not match jwt compact format")
		return dashboardJWTSessionBootstrap{
			TokenPresent:       true,
			TokenFormat:        "bearer-non-jwt",
			ClaimsTrusted:      claimsTrusted,
			MiddlewareBridge:   middlewareBridge,
			SessionState:       "invalid",
			VerificationState:  verificationState,
			VerificationHint:   verificationHint,
			TrustLevel:         trustLevel,
			TrustMessage:       trustMessage,
			RefreshRecommended: true,
			RefreshReason:      "token_not_jwt",
			ParseError:         "token does not match jwt compact format",
		}
	}

	claims, err := decodeJWTClaims(token)
	if err != nil {
		verificationState, verificationHint, trustLevel, trustMessage := deriveJWTVerificationHints("bearer-jwt", "invalid", claimsTrusted, err.Error())
		return dashboardJWTSessionBootstrap{
			TokenPresent:       true,
			TokenFormat:        "bearer-jwt",
			ClaimsTrusted:      claimsTrusted,
			MiddlewareBridge:   middlewareBridge,
			SessionState:       "invalid",
			VerificationState:  verificationState,
			VerificationHint:   verificationHint,
			TrustLevel:         trustLevel,
			TrustMessage:       trustMessage,
			RefreshRecommended: true,
			RefreshReason:      "jwt_parse_error",
			ParseError:         err.Error(),
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
		TokenPresent:        true,
		TokenFormat:         "bearer-jwt",
		ClaimsTrusted:       claimsTrusted,
		MiddlewareBridge:    middlewareBridge,
		SessionState:        sessionState,
		VerificationState:   verificationState,
		VerificationHint:    verificationHint,
		TrustLevel:          trustLevel,
		TrustMessage:        trustMessage,
		Subject:             claimString(claims, "sub"),
		Issuer:              claimString(claims, "iss"),
		Audience:            claimAudience(claims, "aud"),
		IssuedAtUnixSec:     claimInt64(claims, "iat"),
		ExpiresAtUnixSec:    expiresAt,
		ExpiresInSec:        expiresInSec,
		RefreshAfterUnixSec: refreshAfterUnixSec,
		RefreshRecommended:  refreshRecommended,
		RefreshReason:       refreshReason,
		Expired:             expiresAt > 0 && now.Unix() >= expiresAt,
	}
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
		Present:       claims.Present,
		Verified:      claims.Verified,
		Source:        claims.Source,
		Subject:       claims.Subject,
		RoleID:        claims.RoleID,
		ClaimsVersion: claims.ClaimsVersion,
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
