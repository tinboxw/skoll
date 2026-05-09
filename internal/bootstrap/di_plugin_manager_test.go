package bootstrap

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/pkg/logging"
)

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
	infos, snapshots, handlers := registerBuiltinPluginExtensions(logging.New("info"), "test-secret")

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
	infos, snapshots, handlers := registerBuiltinPluginExtensions(logging.New("info"), "test-secret")
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

	if err := mgr.Disable("builtin-logger"); err != nil {
		t.Fatalf("disable builtin logger: %v", err)
	}
	item, _ = mgr.Get("builtin-logger")
	if item.State != plugin.StateDisabled {
		t.Fatalf("expected disabled state, got %s", item.State)
	}

	if err := mgr.Enable("builtin-logger"); err != nil {
		t.Fatalf("enable builtin logger: %v", err)
	}
	item, _ = mgr.Get("builtin-logger")
	if item.State != plugin.StateEnabled {
		t.Fatalf("expected enabled state, got %s", item.State)
	}

	if err := mgr.Uninstall("builtin-logger"); err != nil {
		t.Fatalf("uninstall builtin logger: %v", err)
	}
	item, _ = mgr.Get("builtin-logger")
	if item.State != plugin.StateUninstalled {
		t.Fatalf("expected uninstalled state, got %s", item.State)
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
	infos, snapshots, handlers := registerBuiltinPluginExtensions(logging.New("info"), "test-secret")
	mgr := &pluginManagerWithExtensions{
		Manager:       &fakeManager{items: map[string]plugin.Info{}},
		builtinInfos:  infos,
		extensions:    snapshots,
		routeHandlers: handlers,
	}

	t.Run("auth login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"pass"}`))
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
