package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
)

type controlHealthManager struct {
	*fakePluginManager
	report pluginruntime.HealthReport
	err    error
}

func (m *controlHealthManager) CheckPluginHealth(_ context.Context, _ string) (pluginruntime.HealthReport, error) {
	return m.report, m.err
}

func (m *controlHealthManager) PluginReadiness(context.Context) pluginruntime.ReadinessReport {
	return pluginruntime.ReadinessReport{Ready: m.err == nil, Plugins: []pluginruntime.HealthReport{m.report}}
}

func TestPluginControlSnapshotPublishesRuntimeAndCapabilities(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	manager := &controlHealthManager{
		fakePluginManager: &fakePluginManager{
			items: map[string]pluginruntime.Info{
				"reports": {
					ID: "reports", Name: "Reports", Version: "2.1.0", APIVersion: "v1",
					MigrationVersion: "2.0.0", State: pluginruntime.StateEnabled, InstalledAt: now,
					ServiceBaseURL: "http://127.0.0.1:19090", ServiceHealthURL: "http://127.0.0.1:19090/health",
					Permissions: []string{"reports.read", "reports.export"},
					PermissionResources: []pluginruntime.PermissionDeclaration{
						{Key: "reports.read", Type: "api", Module: "reports", Name: "Read reports", Risk: "low"},
					},
					Dependencies: []pluginruntime.Dependency{{ID: "auth", Version: ">=1.0.0"}},
					APIContract: &pluginruntime.APIContract{Routes: []pluginruntime.APIRoute{
						{Method: http.MethodGet, Path: "/v1/plugins/reports/api/items", Permission: "reports.read", AuditAction: "reports.items.read"},
					}},
				},
			},
			snapshots: map[string]pluginruntime.RegistrySnapshot{
				"reports": {
					Routes: []pluginruntime.RouteExtension{{Method: http.MethodPost, Path: "/v1/plugins/reports/api/export", Permission: "reports.export", Source: "plugin.reports"}},
					Events: []string{"report.completed"}, Menus: []pluginruntime.MenuExtension{{Name: "Reports", Path: "/skoll/plugins/reports"}},
				},
			},
		},
		report: pluginruntime.HealthReport{
			PluginID: "reports", Status: pluginruntime.HealthStatusHealthy, Code: "health_ok",
			CheckedAt: now, LatencyMillis: 8, HTTPStatus: http.StatusOK,
		},
	}

	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, manager)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/plugins/reports/control", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Code string                `json:"code"`
		Data pluginControlSnapshot `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != "ok" || response.Data.Plugin.ID != "reports" {
		t.Fatalf("unexpected identity: %+v", response)
	}
	if response.Data.Runtime.State != string(pluginruntime.StateEnabled) || response.Data.Runtime.Health == nil || !response.Data.Runtime.Health.Ready() {
		t.Fatalf("unexpected runtime: %+v", response.Data.Runtime)
	}
	if !response.Data.StaleAfter.After(response.Data.CapturedAt) {
		t.Fatalf("snapshot must declare freshness: captured=%s stale=%s", response.Data.CapturedAt, response.Data.StaleAfter)
	}
	capabilities := response.Data.Capabilities
	if len(capabilities.HostServices) != 11 || len(capabilities.Permissions) != 2 || len(capabilities.Routes) != 2 || len(capabilities.Dependencies) != 1 {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	if capabilities.Extensions.Routes != 1 || capabilities.Extensions.Events != 1 || capabilities.Extensions.Menus != 1 {
		t.Fatalf("unexpected extensions: %+v", capabilities.Extensions)
	}
}

func TestPluginControlSnapshotKeepsUnavailableHealthInspectable(t *testing.T) {
	manager := &controlHealthManager{
		fakePluginManager: &fakePluginManager{items: map[string]pluginruntime.Info{
			"reports": {ID: "reports", Name: "Reports", Version: "1.0.0", State: pluginruntime.StateEnabled},
		}},
		err: errors.New("health provider failed"),
	}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, manager)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/plugins/reports/control", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data pluginControlSnapshot `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.Runtime.Health == nil || response.Data.Runtime.Health.Code != "health_unavailable" {
		t.Fatalf("unavailable health must remain inspectable: %+v", response.Data.Runtime.Health)
	}
}
