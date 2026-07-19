package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/plugin"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	menusvc "github.com/tinboxw/skoll/internal/service/menu"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	rolesvc "github.com/tinboxw/skoll/internal/service/role"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/internal/store"
)

type routerAllScopeResolver struct{}

func (routerAllScopeResolver) ResolveDataScope(_ context.Context, _ rbacsvc.ResolveDataScopeInput) (rbacsvc.DataScopeDecision, error) {
	return rbacsvc.DataScopeDecision{All: true}, nil
}

func TestRouterDocumentationRoutes(t *testing.T) {
	router := NewRouter(Dependencies{})

	openAPIReq := httptest.NewRequest(http.MethodGet, "/skoll/docs/openapi.yaml", nil)
	openAPIResp := httptest.NewRecorder()
	router.ServeHTTP(openAPIResp, openAPIReq)
	if openAPIResp.Code != http.StatusOK {
		t.Fatalf("openapi status=%d body=%s", openAPIResp.Code, openAPIResp.Body.String())
	}
	if !strings.Contains(openAPIResp.Body.String(), "openapi:") {
		t.Fatalf("unexpected openapi payload: %s", openAPIResp.Body.String())
	}
	if !strings.Contains(openAPIResp.Body.String(), "- url: /skoll") {
		t.Fatalf("expected openapi servers url to include /skoll, payload=%s", openAPIResp.Body.String())
	}

	swaggerReq := httptest.NewRequest(http.MethodGet, "/skoll/docs/swagger", nil)
	swaggerResp := httptest.NewRecorder()
	router.ServeHTTP(swaggerResp, swaggerReq)
	if swaggerResp.Code != http.StatusOK {
		t.Fatalf("swagger status=%d body=%s", swaggerResp.Code, swaggerResp.Body.String())
	}
	if !strings.Contains(swaggerResp.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("unexpected swagger html")
	}
	if !strings.Contains(swaggerResp.Body.String(), "url: '/skoll/docs/openapi.yaml'") {
		t.Fatalf("expected swagger html to reference prefixed openapi path, body=%s", swaggerResp.Body.String())
	}
}

func TestRouterDocumentationRoutesWithCustomPrefix(t *testing.T) {
	router := NewRouter(Dependencies{APIPrefix: "/gateway"})

	openAPIReq := httptest.NewRequest(http.MethodGet, "/gateway/docs/openapi.yaml", nil)
	openAPIResp := httptest.NewRecorder()
	router.ServeHTTP(openAPIResp, openAPIReq)
	if openAPIResp.Code != http.StatusOK {
		t.Fatalf("openapi status=%d body=%s", openAPIResp.Code, openAPIResp.Body.String())
	}
	if !strings.Contains(openAPIResp.Body.String(), "- url: /gateway") {
		t.Fatalf("expected openapi servers url to include /gateway, payload=%s", openAPIResp.Body.String())
	}

	swaggerReq := httptest.NewRequest(http.MethodGet, "/gateway/docs/swagger", nil)
	swaggerResp := httptest.NewRecorder()
	router.ServeHTTP(swaggerResp, swaggerReq)
	if swaggerResp.Code != http.StatusOK {
		t.Fatalf("swagger status=%d body=%s", swaggerResp.Code, swaggerResp.Body.String())
	}
	if !strings.Contains(swaggerResp.Body.String(), "url: '/gateway/docs/openapi.yaml'") {
		t.Fatalf("expected swagger html to reference custom prefix openapi path, body=%s", swaggerResp.Body.String())
	}
}

func TestOpenAPIContractFilesStayInSync(t *testing.T) {
	embedded := strings.TrimSpace(openAPIYAMLDocument)
	if embedded == "" {
		t.Fatal("embedded OpenAPI document is empty")
	}
	for _, marker := range []string{
		"openapi: 3.0.3",
		"info:",
		"servers:",
		"paths:",
		"components:",
		"schemas:",
		"MessageResponse:",
	} {
		if !strings.Contains(embedded, marker) {
			t.Fatalf("embedded OpenAPI document missing marker %q", marker)
		}
	}

	docsPath := filepath.Join("..", "..", "..", "docs", "api", "openapi.yaml")
	docsRaw, err := os.ReadFile(docsPath)
	if err != nil {
		t.Fatalf("read docs OpenAPI document: %v", err)
	}
	docs := strings.TrimSpace(string(docsRaw))
	if docs != embedded {
		t.Fatalf("docs/api/openapi.yaml is out of sync with internal/handler/http/openapi.yaml")
	}
}

func TestRouterUserCreateAndGet(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}
	_ = auditsvc.NewService(bundle.Audit)
	auditService := auditsvc.NewService(bundle.Audit)
	userService := usersvc.NewServiceWithDataScope(bundle.Users, bundle.Audit, bundle.UnitOfWork, routerAllScopeResolver{})
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
	req := httptest.NewRequest(http.MethodPost, "/skoll/v1/users", bytes.NewReader(raw))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", resp.Code, resp.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/users?offset=0&limit=10", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	roleEntity, err := roleService.Create(context.Background(), rolesvc.CreateRoleInput{
		Name:        "Operator",
		Key:         "operator",
		Description: "Operator role",
		Permissions: []string{"user.read"},
	})
	if err != nil {
		t.Fatalf("create role error: %v", err)
	}

	assignReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/users/new/roles", bytes.NewReader([]byte(`{"roleId":"`+roleEntity.ID.String()+`","scope":"self"}`)))
	assignResp := httptest.NewRecorder()
	router.ServeHTTP(assignResp, assignReq)
	if assignResp.Code != http.StatusOK {
		t.Fatalf("assign role status=%d body=%s", assignResp.Code, assignResp.Body.String())
	}

	roleUsersReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/roles/"+roleEntity.ID.String()+"/users", nil)
	roleUsersResp := httptest.NewRecorder()
	router.ServeHTTP(roleUsersResp, roleUsersReq)
	if roleUsersResp.Code != http.StatusOK {
		t.Fatalf("role users status=%d body=%s", roleUsersResp.Code, roleUsersResp.Body.String())
	}
	if !strings.Contains(roleUsersResp.Body.String(), "api_user") {
		t.Fatalf("expected assigned user in role users payload: %s", roleUsersResp.Body.String())
	}

	settingReq := httptest.NewRequest(http.MethodPut, "/skoll/v1/system/settings/demo.flag", bytes.NewReader([]byte(`{"value":"on","encrypted":false}`)))
	settingResp := httptest.NewRecorder()
	router.ServeHTTP(settingResp, settingReq)
	if settingResp.Code != http.StatusOK {
		t.Fatalf("upsert setting status=%d body=%s", settingResp.Code, settingResp.Body.String())
	}

	getSettingReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/system/settings/demo.flag", nil)
	getSettingResp := httptest.NewRecorder()
	router.ServeHTTP(getSettingResp, getSettingReq)
	if getSettingResp.Code != http.StatusOK {
		t.Fatalf("get setting status=%d body=%s", getSettingResp.Code, getSettingResp.Body.String())
	}

	missingSettingReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/system/settings/missing.key", nil)
	missingSettingResp := httptest.NewRecorder()
	router.ServeHTTP(missingSettingResp, missingSettingReq)
	if missingSettingResp.Code != http.StatusNotFound {
		t.Fatalf("missing setting status=%d body=%s", missingSettingResp.Code, missingSettingResp.Body.String())
	}

	invalidSettingReq := httptest.NewRequest(http.MethodPut, "/skoll/v1/system/settings/demo.flag", bytes.NewReader([]byte("{")))
	invalidSettingResp := httptest.NewRecorder()
	router.ServeHTTP(invalidSettingResp, invalidSettingReq)
	if invalidSettingResp.Code != http.StatusBadRequest {
		t.Fatalf("invalid upsert setting status=%d body=%s", invalidSettingResp.Code, invalidSettingResp.Body.String())
	}

	resetReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/system/settings/reset", nil)
	resetResp := httptest.NewRecorder()
	router.ServeHTTP(resetResp, resetReq)
	if resetResp.Code != http.StatusOK {
		t.Fatalf("reset setting status=%d body=%s", resetResp.Code, resetResp.Body.String())
	}

	listAfterResetReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/system/settings?offset=0&limit=10", nil)
	listAfterResetResp := httptest.NewRecorder()
	router.ServeHTTP(listAfterResetResp, listAfterResetReq)
	if listAfterResetResp.Code != http.StatusOK {
		t.Fatalf("list after reset status=%d body=%s", listAfterResetResp.Code, listAfterResetResp.Body.String())
	}
	if strings.Contains(listAfterResetResp.Body.String(), "demo.flag") {
		t.Fatalf("setting should be removed after reset, body=%s", listAfterResetResp.Body.String())
	}
}

func TestRouterRegistersPermissionAndMenuRoutes(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}
	permissionService := permissionsvc.NewService(bundle.Permissions)
	menuService := menusvc.NewService(bundle.Menus)
	if _, err := permissionService.RegisterResource(context.Background(), permissionsvc.RegisterResourceInput{
		Key:    "menu.read",
		Type:   "api",
		Module: "menu",
		Source: "system",
		Name:   "Read menu registry",
	}); err != nil {
		t.Fatalf("RegisterResource error: %v", err)
	}
	node, err := domainmenu.NewNode(domainmenu.NodeIdentity{
		Key:       "system.dashboard",
		ParentKey: "system",
		Source:    "system",
	}, domainmenu.NodeView{Path: "/dashboard", Name: "Dashboard"}, 10)
	if err != nil {
		t.Fatalf("NewNode error: %v", err)
	}
	if _, err := menuService.MergeNodes(context.Background(), menusvc.MergeNodesInput{Nodes: []domainmenu.MenuNode{node}}); err != nil {
		t.Fatalf("MergeNodes error: %v", err)
	}

	router := NewRouter(Dependencies{
		PermissionService: permissionService,
		MenuService:       menuService,
	})

	permissionReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/permissions?source=system", nil)
	permissionResp := httptest.NewRecorder()
	router.ServeHTTP(permissionResp, permissionReq)
	if permissionResp.Code != http.StatusOK {
		t.Fatalf("permission route status=%d body=%s", permissionResp.Code, permissionResp.Body.String())
	}
	if !strings.Contains(permissionResp.Body.String(), "menu.read") {
		t.Fatalf("expected permission catalog payload, got %s", permissionResp.Body.String())
	}

	menuReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/menus/tree?source=system&parentKey=system", nil)
	menuResp := httptest.NewRecorder()
	router.ServeHTTP(menuResp, menuReq)
	if menuResp.Code != http.StatusOK {
		t.Fatalf("menu route status=%d body=%s", menuResp.Code, menuResp.Body.String())
	}
	if !strings.Contains(menuResp.Body.String(), "system.dashboard") {
		t.Fatalf("expected menu tree payload, got %s", menuResp.Body.String())
	}
}

func TestRouterM1PermissionMenuIntegration(t *testing.T) {
	ctx := context.Background()
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}
	roleService := rolesvc.NewService(bundle.Roles)
	menuService := menusvc.NewService(bundle.Menus)
	permissionService := permissionsvc.NewService(bundle.Permissions)

	roleEntity, err := roleService.Create(ctx, rolesvc.CreateRoleInput{
		Name:        "Menu Operator",
		Key:         "menu_operator",
		Description: "menu operator",
		Permissions: []string{"menu.read"},
	})
	if err != nil {
		t.Fatalf("create role error: %v", err)
	}

	nodes := []domainmenu.MenuNode{
		mustRouterMenuNode(t, "system.dashboard", "", "/skoll/dashboard", 10, true, nil),
		mustRouterMenuNode(t, "system.menu", "", "/skoll/menu", 20, true, []string{"menu.read"}),
		mustRouterMenuNode(t, "system.admin", "", "/skoll/admin", 30, true, []string{"menu.manage"}),
		mustRouterMenuNode(t, "system.hidden", "", "/skoll/hidden", 40, false, nil),
	}
	if _, err := menuService.MergeNodes(ctx, menusvc.MergeNodesInput{Nodes: nodes}); err != nil {
		t.Fatalf("MergeNodes error: %v", err)
	}
	if _, err := permissionService.RegisterResource(ctx, permissionsvc.RegisterResourceInput{
		Key:    "menu.read",
		Type:   "api",
		Module: "menu",
		Source: "system",
		Name:   "Read menus",
	}); err != nil {
		t.Fatalf("RegisterResource error: %v", err)
	}

	router := NewRouter(Dependencies{
		RoleService:       roleService,
		MenuService:       menuService,
		PermissionService: permissionService,
	})

	grantReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/roles/"+roleEntity.ID.String()+"/grant", bytes.NewReader([]byte(`{"permission":"menu.manage"}`)))
	grantResp := httptest.NewRecorder()
	router.ServeHTTP(grantResp, grantReq)
	if grantResp.Code != http.StatusOK {
		t.Fatalf("grant status=%d body=%s", grantResp.Code, grantResp.Body.String())
	}
	if !strings.Contains(grantResp.Body.String(), "menu.manage") {
		t.Fatalf("expected granted permission in response: %s", grantResp.Body.String())
	}

	revokeReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/roles/"+roleEntity.ID.String()+"/revoke", bytes.NewReader([]byte(`{"permission":"menu.manage"}`)))
	revokeResp := httptest.NewRecorder()
	router.ServeHTTP(revokeResp, revokeReq)
	if revokeResp.Code != http.StatusOK {
		t.Fatalf("revoke status=%d body=%s", revokeResp.Code, revokeResp.Body.String())
	}
	if strings.Contains(revokeResp.Body.String(), "menu.manage") {
		t.Fatalf("expected revoked permission to be absent: %s", revokeResp.Body.String())
	}

	menuReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/menus/tree?visible=true&permissions=menu.read", nil)
	menuResp := httptest.NewRecorder()
	router.ServeHTTP(menuResp, menuReq)
	if menuResp.Code != http.StatusOK {
		t.Fatalf("menu tree status=%d body=%s", menuResp.Code, menuResp.Body.String())
	}
	menuBody := menuResp.Body.String()
	for _, want := range []string{"system.dashboard", "system.menu"} {
		if !strings.Contains(menuBody, want) {
			t.Fatalf("expected menu response to contain %q: %s", want, menuBody)
		}
	}
	for _, denied := range []string{"system.admin", "system.hidden"} {
		if strings.Contains(menuBody, denied) {
			t.Fatalf("expected menu response to filter %q: %s", denied, menuBody)
		}
	}

	badMenuReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/menus/tree?visible=maybe", nil)
	badMenuResp := httptest.NewRecorder()
	router.ServeHTTP(badMenuResp, badMenuReq)
	if badMenuResp.Code != http.StatusBadRequest {
		t.Fatalf("bad menu query status=%d body=%s", badMenuResp.Code, badMenuResp.Body.String())
	}

	badPermissionReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/permissions?enabled=maybe", nil)
	badPermissionResp := httptest.NewRecorder()
	router.ServeHTTP(badPermissionResp, badPermissionReq)
	if badPermissionResp.Code != http.StatusBadRequest {
		t.Fatalf("bad permission query status=%d body=%s", badPermissionResp.Code, badPermissionResp.Body.String())
	}
}

func mustRouterMenuNode(t *testing.T, key string, parent string, path string, sort int, visible bool, requiredPermissions []string) domainmenu.MenuNode {
	t.Helper()
	node, err := domainmenu.NewNode(domainmenu.NodeIdentity{
		Key:       key,
		ParentKey: parent,
		Source:    "system",
	}, domainmenu.NodeView{
		Name: key,
		Path: path,
	}, sort)
	if err != nil {
		t.Fatalf("NewNode error: %v", err)
	}
	node.Visible = visible
	node.RequiredPermissions = append([]string(nil), requiredPermissions...)
	return node
}

type fakePluginManager struct {
	items     []plugin.Info
	snapshots map[string]plugin.RegistrySnapshot
	executor  func(pluginID, method, path string, w http.ResponseWriter, r *http.Request) bool
}

func (m *fakePluginManager) SavePluginConfig(pluginID string, config map[string]any) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	for i := range m.items {
		if m.items[i].ID == pluginID {
			m.items[i].ConfigJSON = string(raw)
			return nil
		}
	}
	return plugin.ErrPluginNotFound
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
	req := httptest.NewRequest(http.MethodGet, "/skoll/v1/demo/ping", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/skoll/v1/demo/ping", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/skoll/v1/demo/ping", nil)
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

	pageReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/demo-frontend/page", nil)
	pageResp := httptest.NewRecorder()
	router.ServeHTTP(pageResp, pageReq)
	if pageResp.Code != http.StatusOK {
		t.Fatalf("page status=%d body=%s", pageResp.Code, pageResp.Body.String())
	}
	if !strings.Contains(pageResp.Body.String(), "/skoll/v1/plugins/demo-frontend/assets/") {
		t.Fatalf("expected injected base href in page body: %s", pageResp.Body.String())
	}

	assetReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/demo-frontend/assets/app.js", nil)
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
	logDir := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(logDir)
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

	router := NewRouter(Dependencies{PluginManager: manager, LogDir: logDir, LogPluginPerFile: true})

	enableReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/demo-frontend/enable", nil)
	enableResp := httptest.NewRecorder()
	router.ServeHTTP(enableResp, enableReq)
	if enableResp.Code != http.StatusOK {
		t.Fatalf("enable status=%d body=%s", enableResp.Code, enableResp.Body.String())
	}
	if manager.items[0].State != plugin.StateEnabled {
		t.Fatalf("expected enabled state, got %s", manager.items[0].State)
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/demo-frontend/disable", nil)
	disableResp := httptest.NewRecorder()
	router.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", disableResp.Code, disableResp.Body.String())
	}
	if manager.items[0].State != plugin.StateDisabled {
		t.Fatalf("expected disabled state, got %s", manager.items[0].State)
	}

	uninstallReq := httptest.NewRequest(http.MethodDelete, "/skoll/v1/plugins/demo-frontend", nil)
	uninstallResp := httptest.NewRecorder()
	router.ServeHTTP(uninstallResp, uninstallReq)
	if uninstallResp.Code != http.StatusOK {
		t.Fatalf("uninstall status=%d body=%s", uninstallResp.Code, uninstallResp.Body.String())
	}
	if manager.items[0].State != plugin.StateUninstalled {
		t.Fatalf("expected uninstalled state, got %s", manager.items[0].State)
	}

	logsReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/demo-frontend/logs", nil)
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

func TestRouterPluginConfigFlow(t *testing.T) {
	manager := &fakePluginManager{
		items: []plugin.Info{{
			ID:      "demo-frontend",
			Name:    "Demo Frontend",
			Version: "0.1.0",
			State:   plugin.StateEnabled,
			Source:  "plugins/demo-frontend",
		}},
	}
	router := NewRouter(Dependencies{PluginManager: manager})

	getEmptyReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/demo-frontend/config", nil)
	getEmptyResp := httptest.NewRecorder()
	router.ServeHTTP(getEmptyResp, getEmptyReq)
	if getEmptyResp.Code != http.StatusOK {
		t.Fatalf("get empty config status=%d body=%s", getEmptyResp.Code, getEmptyResp.Body.String())
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/demo-frontend", nil)
	detailResp := httptest.NewRecorder()
	router.ServeHTTP(detailResp, detailReq)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("get plugin detail status=%d body=%s", detailResp.Code, detailResp.Body.String())
	}
	if !strings.Contains(detailResp.Body.String(), "demo-frontend") {
		t.Fatalf("unexpected plugin detail payload: %s", detailResp.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/skoll/v1/plugins/demo-frontend/config", bytes.NewReader([]byte(`{"config":{"featureX":true,"threshold":3}}`)))
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update config status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/demo-frontend/config", nil)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get config status=%d body=%s", getResp.Code, getResp.Body.String())
	}
	if !strings.Contains(getResp.Body.String(), "featureX") || !strings.Contains(getResp.Body.String(), "threshold") {
		t.Fatalf("unexpected config payload: %s", getResp.Body.String())
	}

	notFoundReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/not-found/config", nil)
	notFoundResp := httptest.NewRecorder()
	router.ServeHTTP(notFoundResp, notFoundReq)
	if notFoundResp.Code != http.StatusNotFound {
		t.Fatalf("not found plugin config status=%d body=%s", notFoundResp.Code, notFoundResp.Body.String())
	}

	notFoundDetailReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/plugins/not-found", nil)
	notFoundDetailResp := httptest.NewRecorder()
	router.ServeHTTP(notFoundDetailResp, notFoundDetailReq)
	if notFoundDetailResp.Code != http.StatusNotFound {
		t.Fatalf("not found plugin detail status=%d body=%s", notFoundDetailResp.Code, notFoundDetailResp.Body.String())
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

	validateReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/validate", bytes.NewReader([]byte(`{"path":"`+filepath.ToSlash(manifestDir)+`"}`)))
	validateResp := httptest.NewRecorder()
	router.ServeHTTP(validateResp, validateReq)
	if validateResp.Code != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", validateResp.Code, validateResp.Body.String())
	}

	installReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/install", bytes.NewReader([]byte(`{"path":"`+filepath.ToSlash(manifestDir)+`"}`)))
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
	linkReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/link", bytes.NewReader([]byte(linkBody)))
	linkResp := httptest.NewRecorder()
	router.ServeHTTP(linkResp, linkReq)
	if linkResp.Code != http.StatusCreated {
		t.Fatalf("link status=%d body=%s", linkResp.Code, linkResp.Body.String())
	}

	embedBody := `{"pluginId":"ext-embed","name":"Ext Embed","url":"https://example.com/embed"}`
	embedReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/embed", bytes.NewReader([]byte(embedBody)))
	embedResp := httptest.NewRecorder()
	router.ServeHTTP(embedResp, embedReq)
	if embedResp.Code != http.StatusCreated {
		t.Fatalf("embed status=%d body=%s", embedResp.Code, embedResp.Body.String())
	}

	invalidLinkBody := `{"pluginId":"bad-link","name":"Bad Link"}`
	invalidLinkReq := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/link", bytes.NewReader([]byte(invalidLinkBody)))
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

func TestRouterAuditAPIs(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}
	auditService := auditsvc.NewService(bundle.Audit)
	auditEventService := auditsvc.NewEventService(bundle.AuditEvents)

	rec1, err := auditService.Append(context.Background(), "actor-a", "create", "user", "u1", map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("append rec1 error: %v", err)
	}
	_, err = auditService.Append(context.Background(), "actor-b", "update", "role", "r1", map[string]any{"k": "v2"})
	if err != nil {
		t.Fatalf("append rec2 error: %v", err)
	}
	event, err := domainaudit.NewEvent(domainaudit.EventInput{
		ID:     shared.ID("event-1"),
		Type:   domainaudit.EventTypeOperation,
		Action: domainaudit.AuditAction("user.account.create"),
		Actor: domainaudit.ActorRef{
			Type: "user",
			ID:   shared.ID("actor-a"),
		},
		Resource: domainaudit.ResourceRef{
			Type: "user",
			ID:   "u1",
		},
		Result:     domainaudit.EventResultSuccess,
		Risk:       domainaudit.EventRiskMedium,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("new audit event error: %v", err)
	}
	event.SourceData = map[string]any{"operation": "create"}
	if err := auditEventService.AppendEvent(context.Background(), event); err != nil {
		t.Fatalf("append audit event error: %v", err)
	}

	router := NewRouter(Dependencies{AuditService: auditService, AuditEventService: auditEventService})

	listReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/audit?limit=10", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	if !strings.Contains(listResp.Body.String(), "event-1") {
		t.Fatalf("expected audit event in list response, body=%s", listResp.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/audit/event-1", nil)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getResp.Code, getResp.Body.String())
	}
	if !strings.Contains(getResp.Body.String(), "event-1") {
		t.Fatalf("expected event detail, body=%s", getResp.Body.String())
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/audit/export?limit=10", nil)
	exportResp := httptest.NewRecorder()
	router.ServeHTTP(exportResp, exportReq)
	if exportResp.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", exportResp.Code, exportResp.Body.String())
	}
	exportBody := exportResp.Body.String()
	if !strings.Contains(exportBody, "eventId,sourceData") || !strings.Contains(exportBody, "event-1") || !strings.Contains(exportBody, `""operation"":""create""`) {
		t.Fatalf("unexpected export body: %s", exportBody)
	}

	from := rec1.OccurredAt.Add(-time.Second).Format(time.RFC3339)
	to := rec1.OccurredAt.Add(time.Second).Format(time.RFC3339)
	clearReq := httptest.NewRequest(http.MethodDelete, "/skoll/v1/audit?from="+from+"&to="+to, nil)
	clearResp := httptest.NewRecorder()
	router.ServeHTTP(clearResp, clearReq)
	if clearResp.Code != http.StatusOK {
		t.Fatalf("clear status=%d body=%s", clearResp.Code, clearResp.Body.String())
	}

	actorReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/audit/actors/actor-a?limit=10", nil)
	actorResp := httptest.NewRecorder()
	router.ServeHTTP(actorResp, actorReq)
	if actorResp.Code != http.StatusOK {
		t.Fatalf("actor list status=%d body=%s", actorResp.Code, actorResp.Body.String())
	}
	if strings.Contains(actorResp.Body.String(), rec1.ID.String()) {
		t.Fatalf("expected rec1 removed by clear range, body=%s", actorResp.Body.String())
	}

	legacyListReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/audit/logs?limit=10", nil)
	legacyListResp := httptest.NewRecorder()
	router.ServeHTTP(legacyListResp, legacyListReq)
	if legacyListResp.Code != http.StatusNotFound {
		t.Fatalf("legacy list route should be unavailable, got %d body=%s", legacyListResp.Code, legacyListResp.Body.String())
	}

	legacyExportReq := httptest.NewRequest(http.MethodGet, "/skoll/v1/audit/logs/export?limit=10", nil)
	legacyExportResp := httptest.NewRecorder()
	router.ServeHTTP(legacyExportResp, legacyExportReq)
	if legacyExportResp.Code != http.StatusNotFound {
		t.Fatalf("legacy export route should be unavailable, got %d body=%s", legacyExportResp.Code, legacyExportResp.Body.String())
	}
}
