package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	httpHandler "github.com/tinboxw/skoll/internal/handler/http"
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

func TestPluginManagerWithExtensionsProxiesDeclaredExternalRoute(t *testing.T) {
	const routePath = "/v1/plugins/reports/api/items"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method=%s", r.Method)
		}
		if r.URL.Path != "/runtime"+routePath {
			t.Errorf("path=%s", r.URL.Path)
		}
		if r.URL.RawQuery != "page=2" {
			t.Errorf("query=%s", r.URL.RawQuery)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if string(body) != `{"name":"monthly"}` {
			t.Errorf("body=%s", body)
		}
		if r.Header.Get("X-Request-ID") != "request-42" {
			t.Errorf("request header not forwarded")
		}
		w.Header().Set("X-Plugin-Response", "reports")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"source":"plugin-backend"}`))
	}))
	defer backend.Close()

	manager := newExternalRouteTestManager(backend.URL+"/runtime", plugin.StateEnabled)
	router := httpHandler.NewRouter(httpHandler.Dependencies{PluginManager: manager})
	req := httptest.NewRequest(http.MethodPost, "/skoll"+routePath+"?page=2", bytes.NewBufferString(`{"name":"monthly"}`))
	req.Header.Set("X-Request-ID", "request-42")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("X-Plugin-Response") != "reports" {
		t.Fatalf("backend response header was not preserved")
	}
	if resp.Body.String() != `{"source":"plugin-backend"}` {
		t.Fatalf("body=%s", resp.Body.String())
	}
}

func TestPluginManagerWithExtensionsExternalRouteFailures(t *testing.T) {
	const routePath = "/v1/plugins/reports/api/items"

	tests := []struct {
		name       string
		serviceURL string
		state      plugin.State
		method     string
		handled    bool
		status     int
		code       string
	}{
		{name: "missing backend", state: plugin.StateEnabled, method: http.MethodPost, handled: true, status: http.StatusServiceUnavailable, code: "plugin_backend_not_configured"},
		{name: "disabled", serviceURL: "http://127.0.0.1:1", state: plugin.StateDisabled, method: http.MethodPost, handled: true, status: http.StatusServiceUnavailable, code: "plugin_not_enabled"},
		{name: "unreachable", serviceURL: "http://127.0.0.1:1", state: plugin.StateEnabled, method: http.MethodPost, handled: true, status: http.StatusBadGateway, code: "plugin_backend_unavailable"},
		{name: "undeclared method", serviceURL: "http://127.0.0.1:1", state: plugin.StateEnabled, method: http.MethodDelete, handled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := newExternalRouteTestManager(tt.serviceURL, tt.state)
			req := httptest.NewRequest(tt.method, routePath, nil)
			resp := httptest.NewRecorder()
			handled := manager.HandlePluginRoute("reports", tt.method, routePath, resp, req)
			if handled != tt.handled {
				t.Fatalf("handled=%v want=%v", handled, tt.handled)
			}
			if !tt.handled {
				return
			}
			if resp.Code != tt.status {
				t.Fatalf("status=%d want=%d body=%s", resp.Code, tt.status, resp.Body.String())
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Code != tt.code {
				t.Fatalf("code=%s want=%s", body.Code, tt.code)
			}
		})
	}
}

func newExternalRouteTestManager(serviceURL string, state plugin.State) *pluginManagerWithExtensions {
	const routePath = "/v1/plugins/reports/api/items"
	return &pluginManagerWithExtensions{
		Manager: &fakeManager{items: map[string]plugin.Info{
			"reports": {
				ID:               "reports",
				Name:             "Reports",
				Version:          "1.0.0",
				State:            state,
				ServiceBaseURL:   serviceURL,
				ServiceHealthURL: serviceURL + "/health",
				APIContract: &plugin.APIContract{Routes: []plugin.APIRoute{{
					Method: http.MethodPost,
					Path:   routePath,
				}}},
			},
		}},
		builtinInfos:  map[string]plugin.Info{},
		extensions:    map[string]plugin.RegistrySnapshot{},
		routeHandlers: map[string]http.HandlerFunc{},
		healthChecker: fixedHealthChecker{status: plugin.HealthStatusHealthy, code: "health_ok"},
		healthCache:   make(map[string]plugin.HealthReport),
		healthTTL:     5 * time.Second,
	}
}

type fixedHealthChecker struct {
	status plugin.HealthStatus
	code   string
}

func (c fixedHealthChecker) Check(_ context.Context, info plugin.Info) plugin.HealthReport {
	return plugin.HealthReport{
		PluginID:  info.ID,
		Status:    c.status,
		Code:      c.code,
		CheckedAt: time.Now().UTC(),
	}
}

func TestPluginManagerHealthBlocksTrafficAndDrivesReadiness(t *testing.T) {
	const routePath = "/v1/plugins/reports/api/items"
	var healthy atomic.Bool
	var businessCalls atomic.Int32
	var healthCalls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			healthCalls.Add(1)
			if healthy.Load() {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
		case routePath:
			businessCalls.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"executed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()

	manager := newExternalRouteTestManager(backend.URL, plugin.StateEnabled)
	manager.healthChecker = plugin.NewHTTPHealthChecker(time.Second)
	router := httpHandler.NewRouter(httpHandler.Dependencies{PluginManager: manager})

	assertRequestStatus := func(method, path string, want int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != want {
			t.Fatalf("path=%s status=%d want=%d body=%s", path, resp.Code, want, resp.Body.String())
		}
		return resp.Body.String()
	}

	unhealthyBody := assertRequestStatus(http.MethodPost, "/skoll"+routePath, http.StatusServiceUnavailable)
	if !strings.Contains(unhealthyBody, `"code":"plugin_unhealthy"`) || !strings.Contains(unhealthyBody, `"code":"health_http_status"`) {
		t.Fatalf("unexpected unhealthy route response: %s", unhealthyBody)
	}
	if businessCalls.Load() != 0 {
		t.Fatalf("unhealthy plugin received %d business calls", businessCalls.Load())
	}
	readyBody := assertRequestStatus(http.MethodGet, "/skoll/ready", http.StatusServiceUnavailable)
	if strings.Contains(readyBody, backend.URL) || !strings.Contains(readyBody, `"ready":false`) {
		t.Fatalf("readiness leaked endpoint or missed state: %s", readyBody)
	}
	assertRequestStatus(http.MethodGet, "/skoll/v1/plugins/reports/health", http.StatusServiceUnavailable)

	healthy.Store(true)
	assertRequestStatus(http.MethodGet, "/skoll/ready", http.StatusOK)
	assertRequestStatus(http.MethodPost, "/skoll"+routePath, http.StatusOK)
	if businessCalls.Load() != 1 {
		t.Fatalf("healthy plugin business calls=%d want=1", businessCalls.Load())
	}
	if healthCalls.Load() != 4 {
		t.Fatalf("health calls=%d want=4; business traffic should reuse the readiness cache", healthCalls.Load())
	}
}

func TestPluginManagerRouteLifecycleIsAtomicWithoutRouterRestart(t *testing.T) {
	const routePath = "/v1/plugins/reports/api/items"
	var businessCalls atomic.Int32
	var blockNext atomic.Bool
	requestEntered := make(chan struct{}, 1)
	releaseRequest := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != routePath {
			http.NotFound(w, r)
			return
		}
		businessCalls.Add(1)
		if blockNext.CompareAndSwap(true, false) {
			requestEntered <- struct{}{}
			<-releaseRequest
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"executed"}`))
	}))
	defer backend.Close()

	pluginDir := filepath.Join(t.TempDir(), "reports")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}
	manifest := strings.ReplaceAll(`id: reports
name: Reports
version: 1.0.0
service_base_url: SERVICE_URL
service_health_url: SERVICE_URL/health
api:
  routes:
    - method: POST
      path: /v1/plugins/reports/api/items
      permission: reports.items.write
      audit_action: reports.items.write
`, "SERVICE_URL", backend.URL)
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}

	manager := &pluginManagerWithExtensions{
		Manager:          plugin.NewRuntimeManager(plugin.NewFileLoader(), plugin.NewTopologicalResolver()),
		builtinInfos:     map[string]plugin.Info{},
		extensions:       map[string]plugin.RegistrySnapshot{},
		routeHandlers:    map[string]http.HandlerFunc{},
		routePermissions: mustEmptyRoutePermissionRegistry(),
		healthChecker:    fixedHealthChecker{status: plugin.HealthStatusHealthy, code: "health_ok"},
		healthCache:      make(map[string]plugin.HealthReport),
		healthTTL:        5 * time.Second,
	}
	if _, err := manager.Install(pluginDir); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	router := httpHandler.NewRouter(httpHandler.Dependencies{PluginManager: manager})

	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/skoll"+routePath, bytes.NewBufferString(`{"name":"monthly"}`))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}
	assertRuntimeState := func(enabled bool) {
		t.Helper()
		_, permissionOK := manager.ResolveRoutePermission(http.MethodPost, routePath)
		_, extensionOK := manager.GetExtensionSnapshot("reports")
		if permissionOK != enabled || extensionOK != enabled {
			t.Fatalf("runtime state permission=%v extension=%v want=%v", permissionOK, extensionOK, enabled)
		}
	}

	if resp := request(); resp.Code != http.StatusServiceUnavailable || !strings.Contains(resp.Body.String(), `"code":"plugin_not_enabled"`) {
		t.Fatalf("installed route status=%d body=%s", resp.Code, resp.Body.String())
	}
	assertRuntimeState(false)
	manager.healthCache["reports"] = plugin.HealthReport{
		PluginID:  "reports",
		Status:    plugin.HealthStatusUnhealthy,
		Code:      "stale_health",
		CheckedAt: time.Now().UTC(),
	}

	if err := manager.Enable("reports"); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}
	assertRuntimeState(true)
	if resp := request(); resp.Code != http.StatusOK {
		t.Fatalf("enabled route status=%d body=%s", resp.Code, resp.Body.String())
	}

	blockNext.Store(true)
	inFlightDone := make(chan *httptest.ResponseRecorder, 1)
	go func() { inFlightDone <- request() }()
	select {
	case <-requestEntered:
	case <-time.After(time.Second):
		t.Fatal("in-flight request did not reach plugin backend")
	}

	disableDone := make(chan error, 1)
	go func() { disableDone <- manager.Disable("reports") }()
	select {
	case err := <-disableDone:
		t.Fatalf("disable returned before in-flight request drained: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseRequest)
	if resp := <-inFlightDone; resp.Code != http.StatusOK {
		t.Fatalf("in-flight route status=%d body=%s", resp.Code, resp.Body.String())
	}
	if err := <-disableDone; err != nil {
		t.Fatalf("disable plugin: %v", err)
	}
	assertRuntimeState(false)
	if resp := request(); resp.Code != http.StatusServiceUnavailable || !strings.Contains(resp.Body.String(), `"code":"plugin_not_enabled"`) {
		t.Fatalf("disabled route status=%d body=%s", resp.Code, resp.Body.String())
	}
	if got := businessCalls.Load(); got != 2 {
		t.Fatalf("backend business calls=%d want=2", got)
	}

	if err := manager.Enable("reports"); err != nil {
		t.Fatalf("re-enable plugin: %v", err)
	}
	assertRuntimeState(true)
	if resp := request(); resp.Code != http.StatusOK {
		t.Fatalf("re-enabled route status=%d body=%s", resp.Code, resp.Body.String())
	}
}
