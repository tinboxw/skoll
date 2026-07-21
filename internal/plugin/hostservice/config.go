package hostservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type PluginConfigStore interface {
	Get(pluginID string) (pluginruntime.Info, error)
	SavePluginConfig(pluginID string, config map[string]any) error
}

type configValidator interface {
	RegisterConfigSchema(ctx context.Context, in systemsvc.ConfigSchemaInput) (*systemsvc.ConfigSchema, error)
	ValidateConfigValues(ctx context.Context, scope systemsvc.ConfigScope, owner string, values map[string]any) (map[string]any, error)
}

type configService struct {
	pluginID string
	store    PluginConfigStore
	validate configValidator
	audit    pluginsdk.AuditService
}

func NewConfigService(pluginID string, store PluginConfigStore, validate configValidator, audit pluginsdk.AuditService) (pluginsdk.ConfigService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" {
		return nil, fmt.Errorf("plugin host identity is required")
	}
	if store == nil || validate == nil || audit == nil {
		return nil, fmt.Errorf("plugin host config dependencies are required")
	}
	return &configService{pluginID: pluginID, store: store, validate: validate, audit: audit}, nil
}

func (s *configService) Get(_ context.Context) (map[string]any, error) {
	info, err := s.store.Get(s.pluginID)
	if err != nil {
		return nil, err
	}
	values := map[string]any{}
	if raw := strings.TrimSpace(info.ConfigJSON); raw != "" {
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, fmt.Errorf("decode plugin config: %w", err)
		}
	}
	return redactMap(values), nil
}

func (s *configService) Replace(ctx context.Context, values map[string]any) (map[string]any, error) {
	if values == nil {
		values = map[string]any{}
	}
	if containsSensitiveKey(values) {
		return nil, fmt.Errorf("sensitive plugin values must use the secret service")
	}
	if err := validateMetadataSize(values); err != nil {
		return nil, err
	}
	info, err := s.store.Get(s.pluginID)
	if err != nil {
		return nil, err
	}
	normalized := redactMap(values)
	schema, err := currentPluginConfigSchema(info)
	if err != nil {
		return nil, err
	}
	if schema != nil {
		if _, err := s.validate.RegisterConfigSchema(ctx, pluginConfigSchemaInput(s.pluginID, *schema)); err != nil {
			return nil, err
		}
		normalized, err = s.validate.ValidateConfigValues(ctx, systemsvc.ConfigScopePlugin, s.pluginID, normalized)
		if err != nil {
			return nil, err
		}
	}
	if err := s.store.SavePluginConfig(s.pluginID, normalized); err != nil {
		return nil, err
	}
	if _, err := s.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: "config.replace", Resource: "config", ResourceID: s.pluginID,
		Risk: pluginsdk.AuditRiskMedium, Detail: map[string]any{"configKeys": len(normalized)},
	}); err != nil {
		return nil, err
	}
	return redactMap(normalized), nil
}

func currentPluginConfigSchema(info pluginruntime.Info) (*pluginruntime.ConfigSchema, error) {
	if info.ConfigSchema != nil {
		return info.ConfigSchema, nil
	}
	if strings.TrimSpace(info.Source) == "" {
		return nil, nil
	}
	loaded, err := pluginruntime.NewFileLoader().Load(info.Source)
	if err != nil {
		return nil, fmt.Errorf("load plugin config schema: %w", err)
	}
	return loaded.ConfigSchema, nil
}

func pluginConfigSchemaInput(pluginID string, schema pluginruntime.ConfigSchema) systemsvc.ConfigSchemaInput {
	fields := make([]systemsvc.ConfigField, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		options := make([]systemsvc.ConfigOption, 0, len(field.Options))
		for _, option := range field.Options {
			options = append(options, systemsvc.ConfigOption{
				Label: option.Label, LabelZhCN: option.LabelZhCN, LabelEnUS: option.LabelEnUS, Value: option.Value,
			})
		}
		fields = append(fields, systemsvc.ConfigField{
			Key: field.Key, Label: field.Label, LabelZhCN: field.LabelZhCN, LabelEnUS: field.LabelEnUS,
			Type: systemsvc.ConfigFieldType(field.Type), Required: field.Required, Default: field.Default,
			Placeholder: field.Placeholder, Help: field.Help, Min: field.Min, Max: field.Max,
			MinLength: field.MinLength, MaxLength: field.MaxLength, Pattern: field.Pattern, Options: options,
		})
	}
	return systemsvc.ConfigSchemaInput{
		Scope: systemsvc.ConfigScopePlugin, Owner: pluginID, Title: schema.Title, TitleZhCN: schema.TitleZhCN,
		TitleEnUS: schema.TitleEnUS, Description: schema.Description, Fields: fields,
	}
}

var _ pluginsdk.ConfigService = (*configService)(nil)
