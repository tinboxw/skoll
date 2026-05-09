package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	rolesvc "github.com/tinboxw/skoll/internal/service/role"
	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/internal/store"
)

func TestRouterUserCreateAndGet(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}
	_ = auditsvc.NewService(bundle.Audit)
	userService := usersvc.NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	roleService := rolesvc.NewService(bundle.Roles)
	rbacService := rbacsvc.NewService(bundle.RBAC)

	router := NewRouter(Dependencies{UserService: userService, RoleService: roleService, RBACService: rbacService})

	createReq := map[string]any{
		"username":     "api_user",
		"displayName":  "API User",
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
}

type fakePluginManager struct {
	items     []plugin.Info
	snapshots map[string]plugin.RegistrySnapshot
	executor  func(pluginID, method, path string, w http.ResponseWriter, r *http.Request) bool
}

func (m *fakePluginManager) Install(path string) (plugin.Info, error) {
	return plugin.Info{}, errors.New("not implemented")
}

func (m *fakePluginManager) Enable(pluginID string) error {
	_ = pluginID
	return nil
}

func (m *fakePluginManager) Disable(pluginID string) error {
	_ = pluginID
	return nil
}

func (m *fakePluginManager) Uninstall(pluginID string) error {
	_ = pluginID
	return nil
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
