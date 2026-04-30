package plugins

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/pluginmgr"
)

type APIRegistry interface {
	RegisterMany(entries []string)
}

type PluginService interface {
	Install(name, version string, hooks []string) pluginmgr.Manifest
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

type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type Handler struct {
	plugins PluginService
	audit   AuditService
}

func NewHandler(plugins PluginService, auditSvc AuditService) *Handler {
	return &Handler{plugins: plugins, audit: auditSvc}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h == nil || h.plugins == nil {
		return
	}

	handle := func(pattern string, next http.HandlerFunc) {
		if apis != nil {
			apis.RegisterMany([]string{pattern})
		}
		hd := http.Handler(next)
		if wrapper != nil {
			hd = wrapper(hd)
		}
		mux.Handle(pattern, hd)
	}

	handle("POST /admin/v1/plugins/manifests", h.installPlugin)
	handle("POST /admin/v1/plugins/packages/install", h.installPluginPackage)
	handle("GET /admin/v1/plugins", h.listPlugins)
	handle("GET /admin/v1/plugins/{name}", h.getPlugin)
	handle("POST /admin/v1/plugins/{name}/enable", h.enablePlugin)
	handle("POST /admin/v1/plugins/{name}/disable", h.disablePlugin)
	handle("POST /admin/v1/plugins/{name}/version-check", h.checkPluginVersion)
	handle("POST /admin/v1/plugins/{name}/upgrade", h.upgradePlugin)
	handle("POST /admin/v1/plugins/{name}/remove", h.removePlugin)
	handle("POST /admin/v1/plugins/compatibility-check", h.checkPluginCompatibility)
	handle("POST /admin/v1/plugins/hooks/register", h.registerPluginHook)
	handle("GET /admin/v1/plugins/hooks", h.listPluginHooks)
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/enable", h.enablePluginHook)
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/disable", h.disablePluginHook)
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/order", h.setPluginHookOrder)
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/runtime", h.setPluginHookRuntime)
	handle("POST /admin/v1/plugins/hooks/{namespace}/{name}/execute-diagnostic", h.executePluginHookDiagnostic)
	handle("GET /admin/v1/plugins/hooks/dead-letters", h.listPluginHookDeadLetters)
	handle("PUT /admin/v1/plugins/marketplace/trust-roots", h.setMarketplaceTrustRoots)
	handle("GET /admin/v1/plugins/marketplace/trust-roots", h.listMarketplaceTrustRoots)
	handle("POST /admin/v1/plugins/marketplace/index/ingest", h.ingestMarketplaceIndex)
	handle("GET /admin/v1/plugins/marketplace/index/sources", h.listMarketplaceIndexSources)
	handle("POST /admin/v1/plugins/dependency-solver/resolve", h.resolvePluginDependencies)
	handle("POST /admin/v1/plugins/{name}/upgrade/transaction", h.upgradePluginTransactional)
	handle("GET /admin/v1/plugins/upgrade/provenance", h.listPluginUpgradeProvenance)
}

func (h *Handler) installPlugin(w http.ResponseWriter, r *http.Request) {
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
	respondJSON(w, http.StatusCreated, h.plugins.Install(req.Name, req.Version, req.Hooks))
}

func (h *Handler) installPluginPackage(w http.ResponseWriter, r *http.Request) {
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
	item, err := h.plugins.InstallPackageVerified(req.Name, req.Version, req.PackageURL, req.PackageHash, req.Signature, req.Dependencies, req.Hooks)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *Handler) listPlugins(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.plugins.List())
}

func (h *Handler) getPlugin(w http.ResponseWriter, r *http.Request) {
	name, err := parsePathString(r, "name")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	item, err := h.plugins.Get(name)
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

func (h *Handler) enablePlugin(w http.ResponseWriter, r *http.Request) {
	name, err := parsePathString(r, "name")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	item, err := h.plugins.Enable(name)
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

func (h *Handler) disablePlugin(w http.ResponseWriter, r *http.Request) {
	name, err := parsePathString(r, "name")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	item, err := h.plugins.Disable(name)
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

func (h *Handler) checkPluginVersion(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.plugins.CheckVersion(name, strings.TrimSpace(req.LatestVersion))
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

func (h *Handler) upgradePlugin(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.plugins.UpgradePackage(name, req.TargetVersion, req.PackageURL, req.PackageHash, req.Signature, req.Dependencies, req.Hooks)
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

func (h *Handler) removePlugin(w http.ResponseWriter, r *http.Request) {
	name, err := parsePathString(r, "name")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result := h.plugins.Remove(name)
	if !result.Succeeded {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": result.Message})
		return
	}
	h.appendAudit("plugin-governance", "plugin_remove", name)
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) checkPluginCompatibility(w http.ResponseWriter, r *http.Request) {
	var req pluginCompatibilityCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	result := h.plugins.CheckCompatibility(strings.TrimSpace(req.Name), strings.TrimSpace(req.Version), req.Dependencies)
	h.appendAudit("plugin-governance", "plugin_compatibility_check", result.Name)
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) registerPluginHook(w http.ResponseWriter, r *http.Request) {
	var req registerPluginHookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	deadLetter := true
	if req.DeadLetter != nil {
		deadLetter = *req.DeadLetter
	}
	item, err := h.plugins.RegisterHook(req.Name, req.Namespace, req.Version, req.Order, req.TimeoutMillis, req.RetryLimit, deadLetter)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("plugin-governance", "hook_register", item.Namespace+":"+item.Name)
	respondJSON(w, http.StatusCreated, item)
}

func (h *Handler) listPluginHooks(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"items": h.plugins.ListHooks()})
}

func (h *Handler) enablePluginHook(w http.ResponseWriter, r *http.Request) {
	h.setPluginHookEnabled(w, r, true)
}

func (h *Handler) disablePluginHook(w http.ResponseWriter, r *http.Request) {
	h.setPluginHookEnabled(w, r, false)
}

func (h *Handler) setPluginHookEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	namespace, err := parsePathString(r, "namespace")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	name, err := parsePathString(r, "name")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	item, err := h.plugins.SetHookEnabled(name, namespace, enabled)
	if err != nil {
		if err == pluginmgr.ErrHookNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	action := "hook_disable"
	if enabled {
		action = "hook_enable"
	}
	h.appendAudit("plugin-governance", action, item.Namespace+":"+item.Name)
	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) setPluginHookOrder(w http.ResponseWriter, r *http.Request) {
	namespace, err := parsePathString(r, "namespace")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	name, err := parsePathString(r, "name")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req setPluginHookOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	item, err := h.plugins.SetHookOrder(name, namespace, req.Order)
	if err != nil {
		if err == pluginmgr.ErrHookNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("plugin-governance", "hook_order", item.Namespace+":"+item.Name)
	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) setPluginHookRuntime(w http.ResponseWriter, r *http.Request) {
	namespace, err := parsePathString(r, "namespace")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	name, err := parsePathString(r, "name")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req setPluginHookRuntimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	deadLetter := true
	if req.DeadLetter != nil {
		deadLetter = *req.DeadLetter
	}
	item, err := h.plugins.SetHookRuntimePolicy(name, namespace, req.TimeoutMillis, req.RetryLimit, deadLetter)
	if err != nil {
		if err == pluginmgr.ErrHookNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("plugin-governance", "hook_runtime", item.Namespace+":"+item.Name)
	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) executePluginHookDiagnostic(w http.ResponseWriter, r *http.Request) {
	namespace, err := parsePathString(r, "namespace")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	name, err := parsePathString(r, "name")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req executePluginHookDiagnosticRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	result, err := h.plugins.ExecuteHookDiagnostic(name, namespace, req.FailTimes)
	if err != nil {
		if err == pluginmgr.ErrHookNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("plugin-governance", "hook_execute_diagnostic", namespace+":"+name)
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) listPluginHookDeadLetters(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"items": h.plugins.ListHookDeadLetters()})
}

func (h *Handler) setMarketplaceTrustRoots(w http.ResponseWriter, r *http.Request) {
	var req setMarketplaceTrustRootsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	roots := h.plugins.SetMarketplaceTrustRoots(req.Roots)
	h.appendAudit("plugin-governance", "marketplace_trust_roots_set", fmt.Sprintf("count:%d", len(roots)))
	respondJSON(w, http.StatusOK, map[string]any{"items": roots})
}

func (h *Handler) listMarketplaceTrustRoots(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"items": h.plugins.ListMarketplaceTrustRoots()})
}

func (h *Handler) ingestMarketplaceIndex(w http.ResponseWriter, r *http.Request) {
	var req ingestMarketplaceIndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.ExpiresAtUnix <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "expires_at_unix_sec must be positive"})
		return
	}
	result, err := h.plugins.IngestMarketplaceIndex(
		strings.TrimSpace(req.Source), strings.TrimSpace(req.SignedBy), strings.TrimSpace(req.Signature),
		time.Unix(req.ExpiresAtUnix, 0).UTC(), req.Packages, time.Now().UTC(),
	)
	if err != nil {
		if err == pluginmgr.ErrMarketplaceTrustRootNotFound {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("plugin-governance", "marketplace_index_ingest", result.Source)
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) listMarketplaceIndexSources(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"items": h.plugins.ListMarketplaceIndexSources()})
}

func (h *Handler) resolvePluginDependencies(w http.ResponseWriter, r *http.Request) {
	var req resolvePluginDependenciesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	result := h.plugins.SolveDependencies(req.Items)
	h.appendAudit("plugin-governance", "dependency_solver_resolve", fmt.Sprintf("items:%d conflicts:%d", len(req.Items), len(result.Conflicts)))
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) upgradePluginTransactional(w http.ResponseWriter, r *http.Request) {
	name, err := parsePathString(r, "name")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req upgradePluginTransactionalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	result, err := h.plugins.UpgradePackageTransactional(
		strings.TrimSpace(req.TransactionID), strings.TrimSpace(name), strings.TrimSpace(req.TargetVersion),
		strings.TrimSpace(req.PackageURL), strings.TrimSpace(req.PackageHash), strings.TrimSpace(req.Signature),
		req.Dependencies, req.Hooks, time.Now().UTC(),
	)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("plugin-governance", "upgrade_transaction", result.TransactionID)
	respondJSON(w, http.StatusOK, result)
}

func (h *Handler) listPluginUpgradeProvenance(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": h.plugins.ListUpgradeProvenance(limit)})
}

func (h *Handler) appendAudit(actor, action, target string) {
	if h.audit == nil {
		return
	}
	h.audit.Append(actor, action, target)
}

func parsePathString(r *http.Request, key string) (string, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return raw, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// --- Request/Response DTOs ---

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

type registerPluginHookRequest struct {
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	Version       string `json:"version"`
	Order         int    `json:"order"`
	TimeoutMillis int    `json:"timeout_millis"`
	RetryLimit    int    `json:"retry_limit"`
	DeadLetter    *bool  `json:"dead_letter"`
}

type setPluginHookOrderRequest struct {
	Order int `json:"order"`
}

type setPluginHookRuntimeRequest struct {
	TimeoutMillis int   `json:"timeout_millis"`
	RetryLimit    int   `json:"retry_limit"`
	DeadLetter    *bool `json:"dead_letter"`
}

type executePluginHookDiagnosticRequest struct {
	FailTimes int `json:"fail_times"`
}

type setMarketplaceTrustRootsRequest struct {
	Roots []string `json:"roots"`
}

type ingestMarketplaceIndexRequest struct {
	Source        string                              `json:"source"`
	SignedBy      string                              `json:"signed_by"`
	Signature     string                              `json:"signature"`
	ExpiresAtUnix int64                               `json:"expires_at_unix_sec"`
	Packages      []pluginmgr.MarketplaceIndexPackage `json:"packages"`
}

type resolvePluginDependenciesRequest struct {
	Items []pluginmgr.DependencySolveItem `json:"items"`
}

type upgradePluginTransactionalRequest struct {
	TransactionID string                 `json:"transaction_id"`
	TargetVersion string                 `json:"target_version"`
	PackageURL    string                 `json:"package_url"`
	PackageHash   string                 `json:"package_hash"`
	Signature     string                 `json:"signature"`
	Dependencies  []pluginmgr.Dependency `json:"dependencies"`
	Hooks         []string               `json:"hooks"`
}

type pluginVersionCheckRequest struct {
	LatestVersion string `json:"latest_version"`
}

type pluginCompatibilityCheckRequest struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Dependencies []pluginmgr.Dependency `json:"dependencies"`
}
