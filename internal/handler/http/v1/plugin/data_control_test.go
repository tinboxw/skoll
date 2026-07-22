package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
)

type dataControlManager struct {
	*fakePluginManager
	snapshot      pluginruntime.DataControlSnapshot
	rollbackLimit int
}

func (m *dataControlManager) PluginDataControl(context.Context, string) (pluginruntime.DataControlSnapshot, error) {
	return m.snapshot, nil
}

func (m *dataControlManager) RollbackPluginData(_ string, limit int) error {
	m.rollbackLimit = limit
	m.snapshot.Migration.CurrentVersion--
	m.snapshot.Migration.Applied = m.snapshot.Migration.Applied[:len(m.snapshot.Migration.Applied)-limit]
	m.snapshot.Actions.RollbackMaxSteps = len(m.snapshot.Migration.Applied)
	return nil
}

func newDataControlManager() *dataControlManager {
	return &dataControlManager{
		fakePluginManager: &fakePluginManager{items: map[string]pluginruntime.Info{
			"reports": {ID: "reports", Name: "Reports", State: pluginruntime.StateDisabled},
		}},
		snapshot: pluginruntime.DataControlSnapshot{
			PluginID: "reports", CapturedAt: time.Now().UTC(), State: string(pluginruntime.StateDisabled),
			Schema: pluginruntime.DataControlSchema{Available: true, Namespace: "reports", Tables: []pluginruntime.DataControlTable{
				{LogicalName: "entries", PhysicalName: "plugin_reports_entries", Fields: []string{"id"}, Exists: true},
			}},
			Migration: pluginruntime.DataControlMigration{CurrentVersion: 2, Applied: []pluginruntime.DataControlMigrationStep{
				{Version: 1, Name: "create_entries"}, {Version: 2, Name: "add_status"},
			}, Pending: []pluginruntime.DataControlMigrationStep{}},
			Policy:  pluginruntime.DataControlPolicy{Uninstall: "retain", Rollback: "automatic", Effect: "retain_data"},
			Actions: pluginruntime.DataControlActions{CanRollback: true, RollbackMaxSteps: 2},
		},
	}
}

func TestPluginDataControlPublishesAuthoritativeSnapshot(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, newDataControlManager())
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/plugins/reports/data-control", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data pluginruntime.DataControlSnapshot `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.PluginID != "reports" || len(response.Data.Schema.Tables) != 1 || response.Data.Migration.CurrentVersion != 2 {
		t.Fatalf("unexpected snapshot: %+v", response.Data)
	}
}

func TestPluginMigrationRollbackRequiresSuperAdmin(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, newDataControlManager())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/plugins/reports/migrations/rollback", bytes.NewBufferString(`{"limit":1,"confirmPluginId":"reports"}`))
	mux.ServeHTTP(recorder, withRole(request, "admin"))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestPluginMigrationRollbackRequiresExactConfirmation(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, newDataControlManager())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/plugins/reports/migrations/rollback", bytes.NewBufferString(`{"limit":1,"confirmPluginId":"other"}`))
	mux.ServeHTTP(recorder, withRole(request, "super_admin"))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestPluginMigrationRollbackReturnsOperationAndFreshSnapshot(t *testing.T) {
	manager := newDataControlManager()
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, manager)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/plugins/reports/migrations/rollback", bytes.NewBufferString(`{"limit":1,"confirmPluginId":"reports"}`))
	mux.ServeHTTP(recorder, withRole(request, "super_admin"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data pluginMigrationRollbackResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if manager.rollbackLimit != 1 || response.Data.OperationID == "" || response.Data.RolledBackSteps != 1 || response.Data.Snapshot.Migration.CurrentVersion != 1 {
		t.Fatalf("unexpected rollback result: %+v limit=%d", response.Data, manager.rollbackLimit)
	}
}
