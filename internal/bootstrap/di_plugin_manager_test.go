package bootstrap

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/pkg/logging"
)

func TestNewPluginManagerSkipsNonPluginDirectories(t *testing.T) {
	tmp := t.TempDir()
	pluginsDir := filepath.Join(tmp, "plugins")
	if err := os.MkdirAll(filepath.Join(pluginsDir, "logs"), 0o755); err != nil {
		t.Fatalf("create logs dir: %v", err)
	}

	pluginDir := filepath.Join(pluginsDir, "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}
	manifest := "id: demo\nname: Demo Plugin\nversion: 1.0.0\n"
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir to temp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	mgr, ok := newPluginManager(logging.Discard(), "test-secret", nil, nil, nil, nil, nil, nil, nil).(*pluginManagerWithExtensions)
	if !ok {
		t.Fatalf("expected pluginManagerWithExtensions")
	}

	item, err := mgr.Get("demo")
	if err != nil {
		t.Fatalf("expected demo plugin installed: %v", err)
	}
	if item.State != plugin.StateEnabled {
		t.Fatalf("expected demo plugin enabled, got %s", item.State)
	}
}

func TestPluginManagerRoutePermissionLifecycle(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "reports")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}
	manifest := `id: reports
name: Reports
version: 1.0.0
api:
  routes:
    - method: GET
      path: /v1/plugins/reports/api/items
      summary: List report items
      permission: reports.items.read
      audit_action: reports.items.read
`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}

	manager := &pluginManagerWithExtensions{
		Manager:          plugin.NewRuntimeManager(plugin.NewFileLoader(), plugin.NewTopologicalResolver()),
		builtinInfos:     map[string]plugin.Info{},
		extensions:       map[string]plugin.RegistrySnapshot{},
		routeHandlers:    map[string]http.HandlerFunc{},
		routePermissions: mustEmptyRoutePermissionRegistry(),
	}
	if _, err := manager.Install(pluginDir); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	assertRoutePermissionState(t, manager, false)

	if err := manager.Enable("reports"); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}
	assertRoutePermissionState(t, manager, true)

	if err := manager.Disable("reports"); err != nil {
		t.Fatalf("disable plugin: %v", err)
	}
	assertRoutePermissionState(t, manager, false)

	if err := manager.Enable("reports"); err != nil {
		t.Fatalf("re-enable plugin: %v", err)
	}
	assertRoutePermissionState(t, manager, true)

	if err := manager.Uninstall("reports"); err != nil {
		t.Fatalf("uninstall plugin: %v", err)
	}
	assertRoutePermissionState(t, manager, false)
}

func assertRoutePermissionState(t *testing.T, manager *pluginManagerWithExtensions, expected bool) {
	t.Helper()
	descriptor, ok := manager.ResolveRoutePermission(http.MethodGet, "/v1/plugins/reports/api/items")
	if ok != expected {
		t.Fatalf("route permission resolved=%v, want %v: %+v", ok, expected, descriptor)
	}
	if expected && (descriptor.Permission != "reports.items.read" || descriptor.AuditAction != "reports.items.read") {
		t.Fatalf("unexpected route permission descriptor: %+v", descriptor)
	}
	snapshot, snapshotOK := manager.GetExtensionSnapshot("reports")
	if snapshotOK != expected {
		t.Fatalf("extension snapshot resolved=%v, want %v: %+v", snapshotOK, expected, snapshot)
	}
	if expected && (len(snapshot.Routes) != 1 || snapshot.Routes[0].Path != "/v1/plugins/reports/api/items") {
		t.Fatalf("unexpected enabled extension snapshot: %+v", snapshot)
	}
}

type fakeManager struct {
	items map[string]plugin.Info
}

func (m *fakeManager) Install(path string) (plugin.Info, error) {
	_ = path
	return plugin.Info{}, nil
}

func (m *fakeManager) Enable(pluginID string) error {
	item, ok := m.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	item.State = plugin.StateEnabled
	m.items[pluginID] = item
	return nil
}

func (m *fakeManager) Disable(pluginID string) error {
	item, ok := m.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	item.State = plugin.StateDisabled
	m.items[pluginID] = item
	return nil
}

func (m *fakeManager) Uninstall(pluginID string) error {
	item, ok := m.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	item.State = plugin.StateUninstalled
	m.items[pluginID] = item
	return nil
}

func (m *fakeManager) List() []plugin.Info {
	items := make([]plugin.Info, 0, len(m.items))
	for _, item := range m.items {
		items = append(items, item)
	}
	return items
}

func (m *fakeManager) Get(pluginID string) (plugin.Info, error) {
	item, ok := m.items[pluginID]
	if !ok {
		return plugin.Info{}, plugin.ErrPluginNotFound
	}
	return item, nil
}

func TestRegisterBuiltinPluginExtensions(t *testing.T) {
	infos, snapshots, handlers := registerBuiltinPluginExtensions(logging.New("info"), "test-secret", nil)

	for _, id := range []string{"builtin-auth", "builtin-logger", "builtin-dashboard"} {
		info, ok := infos[id]
		if !ok {
			t.Fatalf("missing builtin info for %s", id)
		}
		if info.State != plugin.StateEnabled {
			t.Fatalf("expected enabled state for %s", id)
		}
		snapshot, ok := snapshots[id]
		if !ok {
			t.Fatalf("missing extension snapshot for %s", id)
		}
		if len(snapshot.Routes) == 0 {
			t.Fatalf("expected routes for %s", id)
		}
	}
	if len(handlers) == 0 {
		t.Fatalf("expected builtin route handlers")
	}
}

func TestPluginManagerWithExtensionsBuiltinFallback(t *testing.T) {
	infos, snapshots, handlers := registerBuiltinPluginExtensions(logging.New("info"), "test-secret", nil)
	mgr := &pluginManagerWithExtensions{
		Manager:       &fakeManager{items: map[string]plugin.Info{}},
		builtinInfos:  infos,
		extensions:    snapshots,
		routeHandlers: handlers,
	}

	items := mgr.List()
	if len(items) < 3 {
		t.Fatalf("expected builtin items in list, got %d", len(items))
	}

	item, err := mgr.Get("builtin-logger")
	if err != nil {
		t.Fatalf("expected builtin logger from fallback get: %v", err)
	}
	if item.ID != "builtin-logger" {
		t.Fatalf("unexpected item id: %s", item.ID)
	}

	if err := mgr.Disable("builtin-logger"); err == nil {
		t.Fatalf("expected disable builtin logger to be blocked")
	}
	item, _ = mgr.Get("builtin-logger")
	if item.State != plugin.StateEnabled {
		t.Fatalf("expected state unchanged for builtin logger, got %s", item.State)
	}

	if err := mgr.Enable("builtin-logger"); err != nil {
		t.Fatalf("enable builtin logger: %v", err)
	}
	item, _ = mgr.Get("builtin-logger")
	if item.State != plugin.StateEnabled {
		t.Fatalf("expected enabled state, got %s", item.State)
	}

	if err := mgr.Uninstall("builtin-logger"); err == nil {
		t.Fatalf("expected uninstall builtin logger to be blocked")
	}
	item, _ = mgr.Get("builtin-logger")
	if item.State != plugin.StateEnabled {
		t.Fatalf("expected state unchanged after uninstall attempt, got %s", item.State)
	}

	snapshot, ok := mgr.GetExtensionSnapshot("builtin-dashboard")
	if !ok {
		t.Fatalf("expected extension snapshot for builtin-dashboard")
	}
	if len(snapshot.Widgets) == 0 {
		t.Fatalf("expected dashboard widgets in snapshot")
	}
}

func TestPluginManagerWithExtensionsBuiltinRouteHandlers(t *testing.T) {
	infos, snapshots, handlers := registerBuiltinPluginExtensions(logging.New("info"), "test-secret", nil)
	mgr := &pluginManagerWithExtensions{
		Manager:       &fakeManager{items: map[string]plugin.Info{}},
		builtinInfos:  infos,
		extensions:    snapshots,
		routeHandlers: handlers,
	}

	t.Run("auth login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"account":"admin","password":"pass"}`))
		resp := httptest.NewRecorder()
		ok := mgr.HandlePluginRoute("builtin-auth", http.MethodPost, "/v1/auth/login", resp, req)
		if !ok {
			t.Fatalf("expected auth login handler")
		}
		if resp.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
		}

		var body struct {
			Data struct {
				Token string `json:"token"`
				User  struct {
					Role string `json:"role"`
				} `json:"user"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Data.Token == "" {
			t.Fatalf("expected token in response")
		}
		if body.Data.User.Role != "super_admin" {
			t.Fatalf("expected user role in response, got %q", body.Data.User.Role)
		}
	})

	t.Run("logger logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/logs?limit=3", nil)
		resp := httptest.NewRecorder()
		ok := mgr.HandlePluginRoute("builtin-logger", http.MethodGet, "/v1/logs", resp, req)
		if !ok {
			t.Fatalf("expected logger handler")
		}
		if resp.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
		}

		var body struct {
			Data struct {
				Total int `json:"total"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Data.Total != 3 {
			t.Fatalf("expected total=3 got %d", body.Data.Total)
		}
	})

	t.Run("dashboard widgets", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/widgets", nil)
		resp := httptest.NewRecorder()
		ok := mgr.HandlePluginRoute("builtin-dashboard", http.MethodGet, "/v1/dashboard/widgets", resp, req)
		if !ok {
			t.Fatalf("expected dashboard handler")
		}
		if resp.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
		}

		var body struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(body.Data) == 0 {
			t.Fatalf("expected widget payload")
		}
	})
}
