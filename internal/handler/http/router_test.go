package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	rolesvc "github.com/tinboxw/skoll/internal/service/role"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/internal/store"
)

func TestRouterUserCreateAndGet(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}
	_ = auditsvc.NewService(bundle.Audit)
	auditService := auditsvc.NewService(bundle.Audit)
	userService := usersvc.NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	roleService := rolesvc.NewService(bundle.Roles)
	rbacService := rbacsvc.NewService(bundle.RBAC)
	systemService := systemsvc.NewService(bundle.System)

	router := NewRouter(Dependencies{UserService: userService, RoleService: roleService, RBACService: rbacService, AuditService: auditService, SystemService: systemService})

	createReq := map[string]any{
		"account":      "api_user",
		"name":         "API User",
		"email":        "api@example.com",
		"passwordHash": "1234567890abcdef",
		"actorID":      "admin-1",
	}
	raw, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewReader(raw))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", resp.Code, resp.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/users?offset=0&limit=10", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	settingReq := httptest.NewRequest(http.MethodPut, "/v1/system/settings/demo.flag", bytes.NewReader([]byte(`{"value":"on","encrypted":false}`)))
	settingResp := httptest.NewRecorder()
	router.ServeHTTP(settingResp, settingReq)
	if settingResp.Code != http.StatusOK {
		t.Fatalf("upsert setting status=%d body=%s", settingResp.Code, settingResp.Body.String())
	}

	getSettingReq := httptest.NewRequest(http.MethodGet, "/v1/system/settings/demo.flag", nil)
	getSettingResp := httptest.NewRecorder()
	router.ServeHTTP(getSettingResp, getSettingReq)
	if getSettingResp.Code != http.StatusOK {
		t.Fatalf("get setting status=%d body=%s", getSettingResp.Code, getSettingResp.Body.String())
	}
}

type fakePluginManager struct {
	items     []plugin.Info
	snapshots map[string]plugin.RegistrySnapshot
	executor  func(pluginID, method, path string, w http.ResponseWriter, r *http.Request) bool
}

func (m *fakePluginManager) Install(path string) (plugin.Info, error) {
	loader := plugin.NewFileLoader()
	info, err := loader.Load(path)
	if err != nil {
		return plugin.Info{}, err
	}
	for _, item := range m.items {
		if item.ID == info.ID && item.State != plugin.StateUninstalled {
			return plugin.Info{}, plugin.ErrPluginAlreadyExists
		}
	}
	info.State = plugin.StateInstalled
	info.Source = path
	m.items = append(m.items, info)
	return info, nil
}

func (m *fakePluginManager) Enable(pluginID string) error {
	for i := range m.items {
		if m.items[i].ID == pluginID {
			m.items[i].State = plugin.StateEnabled
			return nil
		}
	}
	return plugin.ErrPluginNotFound
}

func (m *fakePluginManager) Disable(pluginID string) error {
	for i := range m.items {
		if m.items[i].ID == pluginID {
			if m.items[i].SystemBuiltin || strings.EqualFold(m.items[i].Source, "builtin") {
				return plugin.ErrPluginSystemProtected
			}
			m.items[i].State = plugin.StateDisabled
			return nil
		}
	}
	return plugin.ErrPluginNotFound
}

func (m *fakePluginManager) Uninstall(pluginID string) error {
	for i := range m.items {
		if m.items[i].ID == pluginID {
			if m.items[i].SystemBuiltin || strings.EqualFold(m.items[i].Source, "builtin") {
				return plugin.ErrPluginSystemProtected
			}
			m.items[i].State = plugin.StateUninstalled
			return nil
		}
	}
	return plugin.ErrPluginNotFound
}

func (m *fakePluginManager) List() []plugin.Info {
	return append([]plugin.Info(nil), m.items...)
}

func (m *fakePluginManager) Get(pluginID string) (plugin.Info, error) {
	for _, item := range m.items {
		if item.ID == pluginID {
			return item, nil
		}
	}
	return plugin.Info{}, plugin.ErrPluginNotFound
}

func (m *fakePluginManager) GetExtensionSnapshot(pluginID string) (plugin.RegistrySnapshot, bool) {
	snapshot, ok := m.snapshots[pluginID]
	return snapshot, ok
}

func (m *fakePluginManager) HandlePluginRoute(pluginID, method, path string, w http.ResponseWriter, r *http.Request) bool {
	if m.executor == nil {
		return false
	}
	return m.executor(pluginID, method, path, w, r)
}

func (m *fakePluginManager) RegisterExternalPlugin(info plugin.Info) error {
	for _, item := range m.items {
		if item.ID == info.ID && item.State != plugin.StateUninstalled {
			return plugin.ErrPluginAlreadyExists
		}
	}
	m.items = append(m.items, info)
	return nil
}

func TestRouterMountsEnabledPluginExtensionRoutes(t *testing.T) {
	manager := &fakePluginManager{
		items: []plugin.Info{
			{ID: "demo", State: plugin.StateEnabled},
		},
		snapshots: map[string]plugin.RegistrySnapshot{
			"demo": {
				Routes: []plugin.RouteExtension{{Method: http.MethodGet, Path: "/v1/demo/ping"}},
			},
		},
	}

	router := NewRouter(Dependencies{PluginManager: manager})
	req := httptest.NewRequest(http.MethodGet, "/v1/demo/ping", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("plugin route status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Code string            `json:"code"`
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body error: %v", err)
	}
	if body.Data["plugin"] != "demo" || body.Data["route"] != "/v1/demo/ping" {
		t.Fatalf("unexpected body data: %+v", body.Data)
	}
}

func TestRouterSkipsDisabledPluginExtensionRoutes(t *testing.T) {
	manager := &fakePluginManager{
		items: []plugin.Info{{ID: "demo", State: plugin.StateDisabled}},
		snapshots: map[string]plugin.RegistrySnapshot{
			"demo": {
				Routes: []plugin.RouteExtension{{Method: http.MethodGet, Path: "/v1/demo/ping"}},
			},
		},
	}

	router := NewRouter(Dependencies{PluginManager: manager})
	req := httptest.NewRequest(http.MethodGet, "/v1/demo/ping", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("disabled plugin route should not mount, got %d", resp.Code)
	}
}

func TestRouterDelegatesToPluginRouteExecutor(t *testing.T) {
	manager := &fakePluginManager{
		items: []plugin.Info{{ID: "demo", State: plugin.StateEnabled}},
		snapshots: map[string]plugin.RegistrySnapshot{
			"demo": {Routes: []plugin.RouteExtension{{Method: http.MethodGet, Path: "/v1/demo/ping"}}},
		},
		executor: func(pluginID, method, path string, w http.ResponseWriter, _ *http.Request) bool {
			if pluginID != "demo" || method != http.MethodGet || path != "/v1/demo/ping" {
				return false
			}
			WriteJSON(w, http.StatusAccepted, map[string]string{"status": "executed", "plugin": pluginID})
			return true
		},
	}

	router := NewRouter(Dependencies{PluginManager: manager})
	req := httptest.NewRequest(http.MethodGet, "/v1/demo/ping", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("plugin executor status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body error: %v", err)
	}
	if body.Data["status"] != "executed" {
		t.Fatalf("unexpected executor response: %+v", body.Data)
	}
}

func TestRouterPluginPageAndAssetsFlow(t *testing.T) {
	tmp := t.TempDir()
	pluginDir := filepath.Join(tmp, "demo-frontend")
	if err := os.MkdirAll(filepath.Join(pluginDir, "static"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "static", "index.html"), []byte("<html><head></head><body><script src=\"./app.js\"></script></body></html>"), 0o644); err != nil {
		t.Fatalf("write index failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "static", "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatalf("write app.js failed: %v", err)
	}

	manager := &fakePluginManager{
		items: []plugin.Info{{
			ID:            "demo-frontend",
			Name:          "Demo Frontend",
			Version:       "0.1.0",
			State:         plugin.StateEnabled,
			Source:        pluginDir,
			UIMode:        plugin.UIModeFrontendOnly,
			FrontendEntry: "/plugins/demo-frontend",
		}},
	}

	router := NewRouter(Dependencies{PluginManager: manager})

	pageReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo-frontend/page", nil)
	pageResp := httptest.NewRecorder()
	router.ServeHTTP(pageResp, pageReq)
	if pageResp.Code != http.StatusOK {
		t.Fatalf("page status=%d body=%s", pageResp.Code, pageResp.Body.String())
	}
	if !strings.Contains(pageResp.Body.String(), "/v1/plugins/demo-frontend/assets/") {
		t.Fatalf("expected injected base href in page body: %s", pageResp.Body.String())
	}

	assetReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo-frontend/assets/app.js", nil)
	assetResp := httptest.NewRecorder()
	router.ServeHTTP(assetResp, assetReq)
	if assetResp.Code != http.StatusOK {
		t.Fatalf("asset status=%d body=%s", assetResp.Code, assetResp.Body.String())
	}
	if !strings.Contains(assetResp.Body.String(), "console.log('ok')") {
		t.Fatalf("unexpected asset body: %s", assetResp.Body.String())
	}
}

func TestRouterPluginLifecycleAndLogsFlow(t *testing.T) {
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join("plugins", "logs"))
	})

	manager := &fakePluginManager{
		items: []plugin.Info{{
			ID:      "demo-frontend",
			Name:    "Demo Frontend",
			Version: "0.1.0",
			State:   plugin.StateInstalled,
			Source:  "plugins/demo-frontend",
		}},
	}

	router := NewRouter(Dependencies{PluginManager: manager})

	enableReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/demo-frontend/enable", nil)
	enableResp := httptest.NewRecorder()
	router.ServeHTTP(enableResp, enableReq)
	if enableResp.Code != http.StatusOK {
		t.Fatalf("enable status=%d body=%s", enableResp.Code, enableResp.Body.String())
	}
	if manager.items[0].State != plugin.StateEnabled {
		t.Fatalf("expected enabled state, got %s", manager.items[0].State)
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/demo-frontend/disable", nil)
	disableResp := httptest.NewRecorder()
	router.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", disableResp.Code, disableResp.Body.String())
	}
	if manager.items[0].State != plugin.StateDisabled {
		t.Fatalf("expected disabled state, got %s", manager.items[0].State)
	}

	uninstallReq := httptest.NewRequest(http.MethodDelete, "/v1/plugins/demo-frontend", nil)
	uninstallResp := httptest.NewRecorder()
	router.ServeHTTP(uninstallResp, uninstallReq)
	if uninstallResp.Code != http.StatusOK {
		t.Fatalf("uninstall status=%d body=%s", uninstallResp.Code, uninstallResp.Body.String())
	}
	if manager.items[0].State != plugin.StateUninstalled {
		t.Fatalf("expected uninstalled state, got %s", manager.items[0].State)
	}

	logsReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo-frontend/logs", nil)
	logsResp := httptest.NewRecorder()
	router.ServeHTTP(logsResp, logsReq)
	if logsResp.Code != http.StatusOK {
		t.Fatalf("logs status=%d body=%s", logsResp.Code, logsResp.Body.String())
	}

	var logsBody struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(logsResp.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("decode logs error: %v", err)
	}
	for _, marker := range []string{"action=enable", "action=disable", "action=uninstall"} {
		if !strings.Contains(logsBody.Data.Content, marker) {
			t.Fatalf("expected logs to contain %q, got %q", marker, logsBody.Data.Content)
		}
	}
}

func TestRouterPluginValidateInstallAndExternalFlow(t *testing.T) {
	tmp := t.TempDir()
	manifestDir := filepath.Join(tmp, "sample")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	manifest := strings.Join([]string{
		"id: sample",
		"name: Sample Plugin",
		"version: 0.1.0",
		"permissions:",
		"  - menu.read",
	}, "\n")
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	manager := &fakePluginManager{items: []plugin.Info{}}
	router := NewRouter(Dependencies{PluginManager: manager})

	validateReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/validate", bytes.NewReader([]byte(`{"path":"`+filepath.ToSlash(manifestDir)+`"}`)))
	validateResp := httptest.NewRecorder()
	router.ServeHTTP(validateResp, validateReq)
	if validateResp.Code != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", validateResp.Code, validateResp.Body.String())
	}

	installReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/install", bytes.NewReader([]byte(`{"path":"`+filepath.ToSlash(manifestDir)+`"}`)))
	installResp := httptest.NewRecorder()
	router.ServeHTTP(installResp, installReq)
	if installResp.Code != http.StatusCreated {
		t.Fatalf("install status=%d body=%s", installResp.Code, installResp.Body.String())
	}
	installed, err := manager.Get("sample")
	if err != nil {
		t.Fatalf("expected installed plugin, err=%v", err)
	}
	if installed.State != plugin.StateInstalled {
		t.Fatalf("expected installed state, got %s", installed.State)
	}

	linkBody := `{"pluginId":"ext-link","name":"Ext Link","url":"https://example.com/link"}`
	linkReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/link", bytes.NewReader([]byte(linkBody)))
	linkResp := httptest.NewRecorder()
	router.ServeHTTP(linkResp, linkReq)
	if linkResp.Code != http.StatusCreated {
		t.Fatalf("link status=%d body=%s", linkResp.Code, linkResp.Body.String())
	}

	embedBody := `{"pluginId":"ext-embed","name":"Ext Embed","url":"https://example.com/embed"}`
	embedReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/embed", bytes.NewReader([]byte(embedBody)))
	embedResp := httptest.NewRecorder()
	router.ServeHTTP(embedResp, embedReq)
	if embedResp.Code != http.StatusCreated {
		t.Fatalf("embed status=%d body=%s", embedResp.Code, embedResp.Body.String())
	}

	invalidLinkBody := `{"pluginId":"bad-link","name":"Bad Link"}`
	invalidLinkReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/link", bytes.NewReader([]byte(invalidLinkBody)))
	invalidLinkResp := httptest.NewRecorder()
	router.ServeHTTP(invalidLinkResp, invalidLinkReq)
	if invalidLinkResp.Code != http.StatusBadRequest {
		t.Fatalf("invalid link status=%d body=%s", invalidLinkResp.Code, invalidLinkResp.Body.String())
	}

	for _, pluginID := range []string{"ext-link", "ext-embed"} {
		ext, getErr := manager.Get(pluginID)
		if getErr != nil {
			t.Fatalf("expected external plugin %s, err=%v", pluginID, getErr)
		}
		if ext.State != plugin.StateEnabled {
			t.Fatalf("expected external plugin %s enabled, got %s", pluginID, ext.State)
		}
	}
}
