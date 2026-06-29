package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestSystemServiceUpsertAndList(t *testing.T) {
	svc := NewService(memory.NewSystemStore())

	first, err := svc.Upsert(context.Background(), UpsertInput{Key: "feature.alpha", Value: "on", Encrypted: false})
	if err != nil {
		t.Fatalf("Upsert create error: %v", err)
	}
	if first == nil || first.ID == "" {
		t.Fatalf("expected created setting")
	}

	second, err := svc.Upsert(context.Background(), UpsertInput{Key: "feature.alpha", Value: "off", Encrypted: true})
	if err != nil {
		t.Fatalf("Upsert update error: %v", err)
	}
	if second == nil || second.Value != "off" || !second.Encrypted {
		t.Fatalf("unexpected updated setting: %+v", second)
	}

	items, err := svc.List(context.Background(), ListInput{Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 setting, got %d", len(items))
	}
}

type failingSystemRepo struct {
	listItems []*domainsystem.Setting
	failOnID  shared.ID
}

func (f *failingSystemRepo) GetSettingByID(_ context.Context, _ shared.ID) (*domainsystem.Setting, error) {
	return nil, nil
}

func (f *failingSystemRepo) GetSettingByKey(_ context.Context, _ string) (*domainsystem.Setting, error) {
	return nil, nil
}

func (f *failingSystemRepo) ListSettings(_ context.Context, _ int, _ int) ([]*domainsystem.Setting, error) {
	return f.listItems, nil
}

func (f *failingSystemRepo) SaveSetting(_ context.Context, _ *domainsystem.Setting) error {
	return nil
}

func (f *failingSystemRepo) DeleteSetting(_ context.Context, id shared.ID) error {
	if id == f.failOnID {
		return fmt.Errorf("delete failed")
	}
	return nil
}

func (f *failingSystemRepo) GetDictionaryTypeByID(_ context.Context, _ shared.ID) (*domainsystem.DictionaryType, error) {
	return nil, nil
}

func (f *failingSystemRepo) GetDictionaryTypeByCode(_ context.Context, _ string) (*domainsystem.DictionaryType, error) {
	return nil, nil
}

func (f *failingSystemRepo) ListDictionaryTypes(_ context.Context, _ int, _ int) ([]domainsystem.DictionaryType, error) {
	return nil, nil
}

func (f *failingSystemRepo) SaveDictionaryType(_ context.Context, _ *domainsystem.DictionaryType) error {
	return nil
}

func (f *failingSystemRepo) DeleteDictionaryType(_ context.Context, _ shared.ID) error {
	return nil
}

func (f *failingSystemRepo) GetDictionaryItemByID(_ context.Context, _ shared.ID) (*domainsystem.DictionaryItem, error) {
	return nil, nil
}

func (f *failingSystemRepo) GetDictionaryItemByTypeAndValue(_ context.Context, _, _ string) (*domainsystem.DictionaryItem, error) {
	return nil, nil
}

func (f *failingSystemRepo) ListDictionaryItems(_ context.Context, _ string, _ int, _ int) ([]domainsystem.DictionaryItem, error) {
	return nil, nil
}

func (f *failingSystemRepo) SaveDictionaryItem(_ context.Context, _ *domainsystem.DictionaryItem) error {
	return nil
}

func (f *failingSystemRepo) DeleteDictionaryItem(_ context.Context, _ shared.ID) error {
	return nil
}

func TestSystemServiceValidationPaths(t *testing.T) {
	t.Run("nil repository", func(t *testing.T) {
		svc := NewService(nil)
		if _, err := svc.GetByKey(context.Background(), "x"); err == nil {
			t.Fatalf("expected repository not configured error")
		}
		if _, err := svc.List(context.Background(), ListInput{Offset: 0, Limit: 1}); err == nil {
			t.Fatalf("expected repository not configured error")
		}
		if _, err := svc.Upsert(context.Background(), UpsertInput{Key: "x", Value: "v"}); err == nil {
			t.Fatalf("expected repository not configured error")
		}
		if _, err := svc.Reset(context.Background()); err == nil {
			t.Fatalf("expected repository not configured error")
		}
	})

	t.Run("key and pagination validation", func(t *testing.T) {
		svc := NewService(memory.NewSystemStore())

		if _, err := svc.GetByKey(context.Background(), "   "); err == nil || !strings.Contains(strings.ToLower(err.Error()), "setting key") {
			t.Fatalf("expected key required error, got %v", err)
		}
		if _, err := svc.Upsert(context.Background(), UpsertInput{Key: "", Value: "v"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "setting key") {
			t.Fatalf("expected key required error, got %v", err)
		}
		if _, err := svc.List(context.Background(), ListInput{Offset: -1, Limit: 10}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "invalid pagination") {
			t.Fatalf("expected invalid pagination error, got %v", err)
		}
	})
}

func TestSystemServiceResetCountsAndStopsOnError(t *testing.T) {
	now := time.Now().UTC()
	s1, err := domainsystem.NewSetting(shared.ID("s1"), "k1", "v1", false, now)
	if err != nil {
		t.Fatalf("new setting s1 error: %v", err)
	}
	s2, err := domainsystem.NewSetting(shared.ID("s2"), "k2", "v2", false, now)
	if err != nil {
		t.Fatalf("new setting s2 error: %v", err)
	}

	repo := &failingSystemRepo{
		listItems: []*domainsystem.Setting{s1, nil, s2},
		failOnID:  shared.ID("s2"),
	}
	svc := NewService(repo)

	deleted, err := svc.Reset(context.Background())
	if err == nil {
		t.Fatalf("expected delete error")
	}
	if deleted != 1 {
		t.Fatalf("expected deleted count 1 before failure, got %d", deleted)
	}
}

func TestSystemServiceDefaultConfigSchema(t *testing.T) {
	svc := NewService(memory.NewSystemStore())

	schema, err := svc.GetConfigSchema(context.Background(), ConfigScopeSystem, "")
	if err != nil {
		t.Fatalf("GetConfigSchema error: %v", err)
	}
	if schema == nil {
		t.Fatalf("expected default system schema")
	}
	if schema.Scope != ConfigScopeSystem || schema.Owner != "system" {
		t.Fatalf("unexpected schema identity: %+v", schema)
	}
	if len(schema.Fields) < 4 {
		t.Fatalf("expected default fields, got %+v", schema.Fields)
	}
	if schema.Fields[0].Key != "audit.retention_days" || schema.Fields[0].Type != ConfigFieldNumber {
		t.Fatalf("unexpected first field: %+v", schema.Fields[0])
	}
	if schema.Fields[0].Min == nil || *schema.Fields[0].Min != 1 {
		t.Fatalf("expected min validation rule, got %+v", schema.Fields[0].Min)
	}

	values, err := svc.ValidateConfigValues(context.Background(), ConfigScopeSystem, "", map[string]any{
		"audit.retention_days": "90",
	})
	if err != nil {
		t.Fatalf("ValidateConfigValues error: %v", err)
	}
	if values["audit.retention_days"] != 90.0 {
		t.Fatalf("expected numeric retention value, got %#v", values["audit.retention_days"])
	}
	if values["plugin.auto_enable"] != false || values["plugin.dev_portal_enabled"] != true {
		t.Fatalf("expected boolean defaults, got %+v", values)
	}
}

func TestSystemServicePluginConfigSchemaRegistry(t *testing.T) {
	svc := NewService(memory.NewSystemStore())
	minLen := 3
	minRate := 0.0
	maxRate := 1.0

	schema, err := svc.RegisterConfigSchema(context.Background(), ConfigSchemaInput{
		Scope:       ConfigScopePlugin,
		Owner:       "reports",
		Title:       "Reports",
		Description: "Reports plugin configuration.",
		Fields: []ConfigField{
			{
				Key:      "reports.mode",
				Type:     ConfigFieldSelect,
				Required: true,
				Default:  "daily",
				Options: []ConfigOption{
					{Label: "Daily", Value: "daily"},
					{Label: "Weekly", Value: "weekly"},
				},
			},
			{Key: "reports.prefix", Type: ConfigFieldString, Default: "ops", MinLength: &minLen, Pattern: "^[a-z]+$"},
			{Key: "reports.rate", Type: ConfigFieldNumber, Default: "0.5", Min: &minRate, Max: &maxRate},
			{Key: "reports.enabled", Type: ConfigFieldBoolean},
		},
	})
	if err != nil {
		t.Fatalf("RegisterConfigSchema error: %v", err)
	}
	if schema.Owner != "reports" || len(schema.Fields) != 4 {
		t.Fatalf("unexpected registered schema: %+v", schema)
	}

	got, err := svc.GetConfigSchema(context.Background(), ConfigScopePlugin, "reports")
	if err != nil {
		t.Fatalf("GetConfigSchema plugin error: %v", err)
	}
	if got == nil || got.Fields[0].Key != "reports.mode" {
		t.Fatalf("unexpected plugin schema: %+v", got)
	}
	got.Fields[0].Key = "mutated"
	again, err := svc.GetConfigSchema(context.Background(), ConfigScopePlugin, "reports")
	if err != nil {
		t.Fatalf("GetConfigSchema again error: %v", err)
	}
	if again.Fields[0].Key != "reports.mode" {
		t.Fatalf("expected cloned schema, got %+v", again.Fields[0])
	}

	items, err := svc.ListConfigSchemas(context.Background(), ConfigSchemaListInput{Scope: ConfigScopePlugin})
	if err != nil {
		t.Fatalf("ListConfigSchemas error: %v", err)
	}
	if len(items) != 1 || items[0].Owner != "reports" {
		t.Fatalf("unexpected plugin schema list: %+v", items)
	}

	values, err := svc.ValidateConfigValues(context.Background(), ConfigScopePlugin, "reports", map[string]any{
		"reports.mode": "weekly",
		"reports.rate": json.Number("0.7"),
	})
	if err != nil {
		t.Fatalf("ValidateConfigValues plugin error: %v", err)
	}
	if values["reports.mode"] != "weekly" || values["reports.prefix"] != "ops" || values["reports.rate"] != 0.7 || values["reports.enabled"] != false {
		t.Fatalf("unexpected normalized values: %+v", values)
	}
}

func TestSystemServiceConfigSchemaValidationRejectsInvalidRules(t *testing.T) {
	svc := NewService(memory.NewSystemStore())
	min := 10.0
	max := 1.0

	cases := []struct {
		name  string
		input ConfigSchemaInput
	}{
		{
			name: "plugin owner required",
			input: ConfigSchemaInput{
				Scope:  ConfigScopePlugin,
				Fields: []ConfigField{{Key: "plugin.mode"}},
			},
		},
		{
			name: "invalid field key",
			input: ConfigSchemaInput{
				Scope:  ConfigScopeSystem,
				Fields: []ConfigField{{Key: "Bad Key"}},
			},
		},
		{
			name: "invalid type",
			input: ConfigSchemaInput{
				Scope:  ConfigScopeSystem,
				Fields: []ConfigField{{Key: "valid.key", Type: "date"}},
			},
		},
		{
			name: "invalid range",
			input: ConfigSchemaInput{
				Scope:  ConfigScopeSystem,
				Fields: []ConfigField{{Key: "valid.number", Type: ConfigFieldNumber, Min: &min, Max: &max}},
			},
		},
		{
			name: "select options required",
			input: ConfigSchemaInput{
				Scope:  ConfigScopeSystem,
				Fields: []ConfigField{{Key: "valid.select", Type: ConfigFieldSelect}},
			},
		},
		{
			name: "select default invalid",
			input: ConfigSchemaInput{
				Scope: ConfigScopeSystem,
				Fields: []ConfigField{{
					Key:     "valid.select",
					Type:    ConfigFieldSelect,
					Default: "missing",
					Options: []ConfigOption{
						{Value: "enabled"},
					},
				}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.RegisterConfigSchema(context.Background(), tt.input); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

func TestSystemServiceValidateConfigValuesRejectsInvalidValues(t *testing.T) {
	svc := NewService(memory.NewSystemStore())

	if _, err := svc.ValidateConfigValues(context.Background(), ConfigScopeSystem, "", map[string]any{
		"audit.retention_days": "0",
	}); err == nil || !strings.Contains(err.Error(), "audit.retention_days") {
		t.Fatalf("expected retention range error, got %v", err)
	}

	if _, err := svc.ValidateConfigValues(context.Background(), ConfigScopePlugin, "missing", map[string]any{}); err == nil {
		t.Fatalf("expected missing schema error")
	}
}
