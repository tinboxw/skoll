package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

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
		respondJSON(w, http.StatusOK, systemStatusResponse{
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
		})
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
