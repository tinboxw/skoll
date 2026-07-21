package hostservice

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	"github.com/tinboxw/skoll/internal/store/memory"
)

type hostConfigStore struct {
	info  pluginruntime.Info
	saves int
}

func TestConfigServiceRejectsUnavailableManifestSchema(t *testing.T) {
	store := &hostConfigStore{info: pluginruntime.Info{ID: "pharma_oa", Source: filepath.Join(t.TempDir(), "missing-plugin")}}
	auditPort, err := NewAuditService("pharma_oa", auditsvc.NewService(clickhouse.NewAuditStore()))
	if err != nil {
		t.Fatalf("NewAuditService error: %v", err)
	}
	service, err := NewConfigService("pharma_oa", store, systemsvc.NewService(memory.NewSystemStore()), auditPort)
	if err != nil {
		t.Fatalf("NewConfigService error: %v", err)
	}
	if _, err := service.Replace(context.Background(), map[string]any{"mode": "safe"}); err == nil {
		t.Fatal("config update proceeded without the declared manifest schema")
	}
	if store.saves != 0 {
		t.Fatalf("config was saved after schema load failure: saves=%d", store.saves)
	}
}

func (s *hostConfigStore) Get(pluginID string) (pluginruntime.Info, error) {
	if pluginID != s.info.ID {
		return pluginruntime.Info{}, pluginruntime.ErrPluginNotFound
	}
	return s.info, nil
}

func (s *hostConfigStore) SavePluginConfig(_ string, config map[string]any) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	s.info.ConfigJSON = string(raw)
	s.saves++
	return nil
}

func TestConfigServiceValidatesSchemaAndRejectsSensitiveValues(t *testing.T) {
	minLength := 3
	store := &hostConfigStore{info: pluginruntime.Info{
		ID: "pharma_oa", ConfigJSON: `{"mode":"safe","api_token":"legacy-secret"}`,
		ConfigSchema: &pluginruntime.ConfigSchema{Fields: []pluginruntime.ConfigField{
			{Key: "mode", Type: "select", Required: true, Options: []pluginruntime.ConfigOption{{Value: "safe"}, {Value: "fast"}}},
			{Key: "endpoint", Type: "string", MinLength: &minLength},
		}},
	}}
	auditPort, err := NewAuditService("pharma_oa", auditsvc.NewService(clickhouse.NewAuditStore()))
	if err != nil {
		t.Fatalf("NewAuditService error: %v", err)
	}
	service, err := NewConfigService("pharma_oa", store, systemsvc.NewService(memory.NewSystemStore()), auditPort)
	if err != nil {
		t.Fatalf("NewConfigService error: %v", err)
	}
	current, err := service.Get(context.Background())
	if err != nil || current["api_token"] != "[REDACTED]" {
		t.Fatalf("Get config=%+v err=%v", current, err)
	}
	if _, err := service.Replace(context.Background(), map[string]any{"mode": "unsafe", "endpoint": "abc"}); err == nil {
		t.Fatal("invalid schema option was accepted")
	}
	if _, err := service.Replace(context.Background(), map[string]any{"mode": "safe", "password": "secret"}); err == nil {
		t.Fatal("sensitive config key was accepted")
	}
	updated, err := service.Replace(context.Background(), map[string]any{"mode": "fast", "endpoint": "https://example.test"})
	if err != nil || updated["mode"] != "fast" || store.saves != 1 {
		t.Fatalf("Replace config=%+v saves=%d err=%v", updated, store.saves, err)
	}
}
