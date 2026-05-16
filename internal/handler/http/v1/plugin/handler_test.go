package plugin

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
)

type fakePluginManager struct {
	items     map[string]plugin.Info
	snapshots map[string]plugin.RegistrySnapshot
}

func (f *fakePluginManager) Install(path string) (plugin.Info, error) {
	_ = path
	return plugin.Info{}, nil
}

func (f *fakePluginManager) List() []plugin.Info {
	items := make([]plugin.Info, 0, len(f.items))
	for _, item := range f.items {
		items = append(items, item)
	}
	return items
}

func (f *fakePluginManager) Enable(pluginID string) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	item.State = plugin.StateEnabled
	f.items[pluginID] = item
	return nil
}

func (f *fakePluginManager) Disable(pluginID string) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	if item.SystemBuiltin || strings.EqualFold(item.Source, "builtin") {
		return plugin.ErrPluginSystemProtected
	}
	item.State = plugin.StateDisabled
	f.items[pluginID] = item
	return nil
}

func (f *fakePluginManager) Uninstall(pluginID string) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	if item.SystemBuiltin || strings.EqualFold(item.Source, "builtin") {
		return plugin.ErrPluginSystemProtected
	}
	item.State = plugin.StateUninstalled
	f.items[pluginID] = item
	return nil
}

func (f *fakePluginManager) Get(pluginID string) (plugin.Info, error) {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.Info{}, plugin.ErrPluginNotFound
	}
	return item, nil
}

func (f *fakePluginManager) GetExtensionSnapshot(pluginID string) (plugin.RegistrySnapshot, bool) {
	s, ok := f.snapshots[pluginID]
	return s, ok
}

func TestPluginHandlerListFromProvider(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{
		"demo": {ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateEnabled},
		"tool": {ID: "tool", Name: "Tool", Version: "0.2.0", State: plugin.StateInstalled},
	}})

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Code string         `json:"code"`
		Data []pluginRecord `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != "ok" {
		t.Fatalf("unexpected code: %s", body.Code)
	}
	if len(body.Data) != 2 {
		t.Fatalf("expected 2 records, got %d", len(body.Data))
	}
	if body.Data[0].UIMode == "" || body.Data[1].UIMode == "" {
		t.Fatalf("expected ui mode in list payload: %+v", body.Data)
	}
}

func TestPluginHandlerEnabledFilterFallback(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{
		"demo": {ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateInstalled},
	}})

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins?enabled=true", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data []pluginRecord `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].ID != "builtin-auth" {
		t.Fatalf("expected fallback builtin-auth, got %+v", body.Data)
	}
}

func TestPluginHandlerStateOperations(t *testing.T) {
	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo": {ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateInstalled},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	enableReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/demo/enable", nil)
	enableResp := httptest.NewRecorder()
	mux.ServeHTTP(enableResp, enableReq)
	if enableResp.Code != http.StatusOK {
		t.Fatalf("enable status=%d body=%s", enableResp.Code, enableResp.Body.String())
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/demo/disable", nil)
	disableResp := httptest.NewRecorder()
	mux.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", disableResp.Code, disableResp.Body.String())
	}

	uninstallReq := httptest.NewRequest(http.MethodDelete, "/v1/plugins/demo", nil)
	uninstallResp := httptest.NewRecorder()
	mux.ServeHTTP(uninstallResp, uninstallReq)
	if uninstallResp.Code != http.StatusOK {
		t.Fatalf("uninstall status=%d body=%s", uninstallResp.Code, uninstallResp.Body.String())
	}
}

func TestPluginHandlerProtectedBuiltinActions(t *testing.T) {
	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"builtin-auth": {ID: "builtin-auth", Name: "Builtin Auth", Version: "1.0.0", State: plugin.StateEnabled, Source: "builtin", SystemBuiltin: true},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	disableReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/builtin-auth/disable", nil)
	disableResp := httptest.NewRecorder()
	mux.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusForbidden {
		t.Fatalf("disable builtin status=%d body=%s", disableResp.Code, disableResp.Body.String())
	}

	uninstallReq := httptest.NewRequest(http.MethodDelete, "/v1/plugins/builtin-auth", nil)
	uninstallResp := httptest.NewRecorder()
	mux.ServeHTTP(uninstallResp, uninstallReq)
	if uninstallResp.Code != http.StatusForbidden {
		t.Fatalf("uninstall builtin status=%d body=%s", uninstallResp.Code, uninstallResp.Body.String())
	}
}

func TestPluginHandlerInstallAndValidate(t *testing.T) {
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

	mgr := &fakePluginManager{items: map[string]plugin.Info{}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	validatePayload := []byte(`{"path":"` + filepath.ToSlash(manifestDir) + `"}`)
	validateReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/validate", bytes.NewReader(validatePayload))
	validateResp := httptest.NewRecorder()
	mux.ServeHTTP(validateResp, validateReq)
	if validateResp.Code != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", validateResp.Code, validateResp.Body.String())
	}

	installReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/install", bytes.NewReader(validatePayload))
	installResp := httptest.NewRecorder()
	mux.ServeHTTP(installResp, installReq)
	if installResp.Code != http.StatusCreated {
		t.Fatalf("install status=%d body=%s", installResp.Code, installResp.Body.String())
	}
}

func TestPluginHandlerInstallValidateBadRequest(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}})

	for _, path := range []string{"/v1/plugins/install", "/v1/plugins/validate"} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(`{"path":""}`)))
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%s expected 400 got %d body=%s", path, resp.Code, resp.Body.String())
		}
	}
}

func TestPluginHandlerDebugAndLogs(t *testing.T) {
	t.Cleanup(func() {
		_ = os.RemoveAll("log")
	})

	if err := os.MkdirAll("log", 0o755); err != nil {
		t.Fatalf("mkdir log dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("log", "demo.log"), []byte("line-1\nline-2\n"), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo": {
			ID:          "demo",
			Name:        "Demo",
			Version:     "0.1.0",
			Description: "demo plugin",
			State:       plugin.StateEnabled,
			Permissions: []string{"menu.read"},
			Dependencies: []plugin.Dependency{
				{ID: "core", Version: "1.0.0"},
			},
			Source: "plugins/demo",
		},
	}, snapshots: map[string]plugin.RegistrySnapshot{
		"demo": {
			Routes: []plugin.RouteExtension{{Method: "GET", Path: "/v1/demo"}},
		},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr, WithPluginLogTarget("info", "log", "", true))

	debugReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo/debug", nil)
	debugResp := httptest.NewRecorder()
	mux.ServeHTTP(debugResp, debugReq)
	if debugResp.Code != http.StatusOK {
		t.Fatalf("debug status=%d body=%s", debugResp.Code, debugResp.Body.String())
	}

	var debugBody struct {
		Data pluginDebugRecord `json:"data"`
	}
	if err := json.Unmarshal(debugResp.Body.Bytes(), &debugBody); err != nil {
		t.Fatalf("decode debug error: %v", err)
	}
	if debugBody.Data.ID != "demo" || debugBody.Data.State != string(plugin.StateEnabled) {
		t.Fatalf("unexpected debug data: %+v", debugBody.Data)
	}
	if debugBody.Data.Extensions == nil {
		t.Fatalf("expected extension snapshot in debug payload")
	}

	logsReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo/logs", nil)
	logsResp := httptest.NewRecorder()
	mux.ServeHTTP(logsResp, logsReq)
	if logsResp.Code != http.StatusOK {
		t.Fatalf("logs status=%d body=%s", logsResp.Code, logsResp.Body.String())
	}

	var logsBody struct {
		Data pluginLogsRecord `json:"data"`
	}
	if err := json.Unmarshal(logsResp.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("decode logs error: %v", err)
	}
	if logsBody.Data.Content != "line-1\nline-2" {
		t.Fatalf("unexpected logs content: %q", logsBody.Data.Content)
	}
}

func TestPluginHandlerLogsFromUnifiedFile(t *testing.T) {
	tmp := t.TempDir()
	logFile := filepath.Join(tmp, "skoll.log")
	content := strings.Join([]string{
		"2026-01-01T00:00:00Z action=install result=ok plugin_id=demo message=installed",
		"2026-01-01T00:00:01Z action=enable result=ok plugin_id=other message=enabled",
		"2026-01-01T00:00:02Z action=disable result=ok plugin_id=demo message=disabled",
	}, "\n")
	if err := os.WriteFile(logFile, []byte(content+"\n"), 0o644); err != nil {
		t.Fatalf("write unified log file: %v", err)
	}

	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo": {ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateEnabled},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr, WithPluginLogTarget("info", filepath.Dir(logFile), filepath.Base(logFile), false))

	logsReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo/logs", nil)
	logsResp := httptest.NewRecorder()
	mux.ServeHTTP(logsResp, logsReq)
	if logsResp.Code != http.StatusOK {
		t.Fatalf("logs status=%d body=%s", logsResp.Code, logsResp.Body.String())
	}

	var logsBody struct {
		Data pluginLogsRecord `json:"data"`
	}
	if err := json.Unmarshal(logsResp.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("decode logs error: %v", err)
	}
	if !strings.Contains(logsBody.Data.Content, "plugin_id=demo") {
		t.Fatalf("expected filtered demo logs, got: %q", logsBody.Data.Content)
	}
	if strings.Contains(logsBody.Data.Content, "plugin_id=other") {
		t.Fatalf("unexpected logs from other plugin: %q", logsBody.Data.Content)
	}
}

func TestPluginHandlerNotFound(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/v1/plugins/missing/enable"},
		{method: http.MethodPost, path: "/v1/plugins/missing/disable"},
		{method: http.MethodDelete, path: "/v1/plugins/missing"},
		{method: http.MethodGet, path: "/v1/plugins/missing/debug"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s %s status=%d body=%s", tc.method, tc.path, resp.Code, resp.Body.String())
		}
	}

	logsReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/missing/logs", nil)
	logsResp := httptest.NewRecorder()
	mux.ServeHTTP(logsResp, logsReq)
	if logsResp.Code != http.StatusNotFound {
		t.Fatalf("logs not found status=%d body=%s", logsResp.Code, logsResp.Body.String())
	}
}

func TestPluginHandlerPageAndAssets(t *testing.T) {
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

	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo-frontend": {
			ID:            "demo-frontend",
			Name:          "Demo Frontend",
			Version:       "0.1.0",
			State:         plugin.StateEnabled,
			Source:        pluginDir,
			UIMode:        plugin.UIModeFrontendOnly,
			FrontendEntry: "/plugins/demo-frontend",
		},
	}}

	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	pageReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo-frontend/page", nil)
	pageResp := httptest.NewRecorder()
	mux.ServeHTTP(pageResp, pageReq)
	if pageResp.Code != http.StatusOK {
		t.Fatalf("page status=%d body=%s", pageResp.Code, pageResp.Body.String())
	}
	if !strings.Contains(pageResp.Body.String(), "/v1/plugins/demo-frontend/assets/") {
		t.Fatalf("expected injected base href in page body: %s", pageResp.Body.String())
	}

	assetReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo-frontend/assets/app.js", nil)
	assetResp := httptest.NewRecorder()
	mux.ServeHTTP(assetResp, assetReq)
	if assetResp.Code != http.StatusOK {
		t.Fatalf("asset status=%d body=%s", assetResp.Code, assetResp.Body.String())
	}
	if !strings.Contains(assetResp.Body.String(), "console.log('ok')") {
		t.Fatalf("unexpected asset body: %s", assetResp.Body.String())
	}
}

func TestPluginHandlerPageBackendOnlyNotFound(t *testing.T) {
	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo-backend": {ID: "demo-backend", Name: "Demo Backend", Version: "0.1.0", State: plugin.StateEnabled, Source: "plugins/demo-backend", UIMode: plugin.UIModeBackendOnly},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo-backend/page", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for backend-only page, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestFakePluginManagerImplementsManager(t *testing.T) {
	var _ PluginManager = (*fakePluginManager)(nil)
	var _ PluginExtensionSnapshotProvider = (*fakePluginManager)(nil)
	t.Log("compile assertions passed")
}
