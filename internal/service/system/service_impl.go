package system

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/internal/cache"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	systemrepo "github.com/tinboxw/skoll/internal/repository/system"
)

const defaultDictionaryCacheTTL = 5 * time.Minute

var configKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)

type serviceImpl struct {
	repo             systemrepo.SystemRepository
	nowFn            func() time.Time
	idFn             func(prefix string) shared.ID
	dictionaryCache  cache.BytesCache
	dictionaryTTL    time.Duration
	cacheMu          sync.Mutex
	dictListKeys     map[string]struct{}
	dictItemKeys     map[string]struct{}
	dictItemListKeys map[string]struct{}
	configMu         sync.RWMutex
	configSchemas    map[string]ConfigSchema
}

func NewService(repo systemrepo.SystemRepository) Service {
	return newServiceImpl(repo)
}

func newServiceImpl(repo systemrepo.SystemRepository) *serviceImpl {
	return &serviceImpl{
		repo:             repo,
		dictionaryTTL:    defaultDictionaryCacheTTL,
		dictListKeys:     map[string]struct{}{},
		dictItemKeys:     map[string]struct{}{},
		dictItemListKeys: map[string]struct{}{},
		configSchemas:    defaultConfigSchemas(),
		nowFn:            func() time.Time { return time.Now().UTC() },
		idFn: func(prefix string) shared.ID {
			return shared.ID("new")
		},
	}
}

func (s *serviceImpl) Upsert(ctx context.Context, in UpsertInput) (*domainsystem.Setting, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("system repository is not configured")
	}
	key := strings.TrimSpace(in.Key)
	if key == "" {
		return nil, fmt.Errorf("setting key is required")
	}

	now := s.nowFn()
	existing, err := s.repo.GetSettingByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		existing.UpdateValue(in.Value, now)
		existing.Encrypted = in.Encrypted
		if err := s.repo.SaveSetting(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	created, err := domainsystem.NewSetting(s.idFn("setting"), key, in.Value, in.Encrypted, now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SaveSetting(ctx, created); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *serviceImpl) GetByKey(ctx context.Context, key string) (*domainsystem.Setting, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("system repository is not configured")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("setting key is required")
	}
	return s.repo.GetSettingByKey(ctx, key)
}

func (s *serviceImpl) List(ctx context.Context, in ListInput) ([]*domainsystem.Setting, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("system repository is not configured")
	}
	if in.Offset < 0 || in.Limit < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	return s.repo.ListSettings(ctx, in.Offset, in.Limit)
}

func (s *serviceImpl) Reset(ctx context.Context) (int, error) {
	if s == nil || s.repo == nil {
		return 0, fmt.Errorf("system repository is not configured")
	}
	items, err := s.repo.ListSettings(ctx, 0, 0)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, item := range items {
		if item == nil || item.ID.IsZero() {
			continue
		}
		if err := s.repo.DeleteSetting(ctx, item.ID); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func (s *serviceImpl) SaveDictionaryType(ctx context.Context, in DictionaryTypeInput) (*domainsystem.DictionaryType, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	now := s.nowFn()
	id := shared.ID(strings.TrimSpace(in.ID))
	if id.IsZero() {
		id = s.idFn("dict_type")
	}
	item, err := domainsystem.NewDictionaryType(domainsystem.DictionaryTypeInput{
		ID:          id,
		Code:        in.Code,
		Name:        in.Name,
		Description: in.Description,
		Status:      domainsystem.DictionaryStatus(strings.TrimSpace(in.Status)),
		Sort:        in.Sort,
		Builtin:     in.Builtin,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, err
	}
	var oldTypeCode string
	if existing, err := s.repo.GetDictionaryTypeByID(ctx, item.ID); err != nil {
		return nil, err
	} else if existing != nil && existing.Meta.CreatedAt.IsZero() == false {
		item.Meta.CreatedAt = existing.Meta.CreatedAt
		oldTypeCode = existing.Code
	}
	if err := s.repo.SaveDictionaryType(ctx, item); err != nil {
		return nil, err
	}
	if oldTypeCode != "" && oldTypeCode != item.Code {
		s.invalidateDictionaryType(oldTypeCode)
		s.invalidateDictionaryItemsForType(oldTypeCode)
	}
	s.invalidateDictionaryType(item.Code)
	return item, nil
}

func (s *serviceImpl) GetDictionaryTypeByCode(ctx context.Context, code string) (*domainsystem.DictionaryType, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	code = normalizeDictionaryCode(code)
	if code == "" {
		return nil, fmt.Errorf("dictionary type code is required")
	}
	key := dictionaryTypeCacheKey(code)
	var cached domainsystem.DictionaryType
	if s.cacheGet(key, &cached) {
		return &cached, nil
	}
	item, err := s.repo.GetDictionaryTypeByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if item != nil {
		s.cacheSet(key, item)
	}
	return item, nil
}

func (s *serviceImpl) ListDictionaryTypes(ctx context.Context, in DictionaryTypeListInput) ([]domainsystem.DictionaryType, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	if in.Offset < 0 || in.Limit < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	key := dictionaryTypeListCacheKey(in.Offset, in.Limit)
	var cached []domainsystem.DictionaryType
	if s.cacheGet(key, &cached) {
		return cached, nil
	}
	items, err := s.repo.ListDictionaryTypes(ctx, in.Offset, in.Limit)
	if err != nil {
		return nil, err
	}
	s.cacheSet(key, items)
	s.trackDictListKey(key)
	return items, nil
}

func (s *serviceImpl) DeleteDictionaryType(ctx context.Context, id string) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	dictID := shared.ID(strings.TrimSpace(id))
	if dictID.IsZero() {
		return fmt.Errorf("dictionary type id is required")
	}
	existing, err := s.repo.GetDictionaryTypeByID(ctx, dictID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteDictionaryType(ctx, dictID); err != nil {
		return err
	}
	if existing != nil {
		s.invalidateDictionaryType(existing.Code)
	} else {
		s.invalidateDictionaryLists()
	}
	s.invalidateDictionaryItemsForType("")
	return nil
}

func (s *serviceImpl) SaveDictionaryItem(ctx context.Context, in DictionaryItemInput) (*domainsystem.DictionaryItem, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	now := s.nowFn()
	id := shared.ID(strings.TrimSpace(in.ID))
	if id.IsZero() {
		id = s.idFn("dict_item")
	}
	item, err := domainsystem.NewDictionaryItem(domainsystem.DictionaryItemInput{
		ID:        id,
		TypeCode:  in.TypeCode,
		Label:     in.Label,
		Value:     in.Value,
		Status:    domainsystem.DictionaryStatus(strings.TrimSpace(in.Status)),
		Sort:      in.Sort,
		Builtin:   in.Builtin,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	var oldItemTypeCode string
	if existing, err := s.repo.GetDictionaryItemByID(ctx, item.ID); err != nil {
		return nil, err
	} else if existing != nil {
		item.Meta.CreatedAt = existing.Meta.CreatedAt
		oldItemTypeCode = existing.TypeCode
		s.cacheDelete(dictionaryItemCacheKey(existing.TypeCode, existing.Value))
	}
	if err := s.repo.SaveDictionaryItem(ctx, item); err != nil {
		return nil, err
	}
	if oldItemTypeCode != "" && oldItemTypeCode != item.TypeCode {
		s.invalidateDictionaryItemsForType(oldItemTypeCode)
	}
	s.invalidateDictionaryItemsForType(item.TypeCode)
	return item, nil
}

func (s *serviceImpl) GetDictionaryItemByTypeAndValue(ctx context.Context, typeCode, value string) (*domainsystem.DictionaryItem, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	typeCode = normalizeDictionaryCode(typeCode)
	value = strings.TrimSpace(value)
	if typeCode == "" || value == "" {
		return nil, fmt.Errorf("dictionary item type code and value are required")
	}
	key := dictionaryItemCacheKey(typeCode, value)
	var cached domainsystem.DictionaryItem
	if s.cacheGet(key, &cached) {
		return &cached, nil
	}
	item, err := s.repo.GetDictionaryItemByTypeAndValue(ctx, typeCode, value)
	if err != nil {
		return nil, err
	}
	if item != nil {
		s.cacheSet(key, item)
		s.trackDictItemKey(key)
	}
	return item, nil
}

func (s *serviceImpl) ListDictionaryItems(ctx context.Context, in DictionaryItemListInput) ([]domainsystem.DictionaryItem, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	if in.Offset < 0 || in.Limit < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	typeCode := normalizeDictionaryCode(in.TypeCode)
	if typeCode == "" {
		return nil, fmt.Errorf("dictionary type code is required")
	}
	key := dictionaryItemListCacheKey(typeCode, in.Offset, in.Limit)
	var cached []domainsystem.DictionaryItem
	if s.cacheGet(key, &cached) {
		return cached, nil
	}
	items, err := s.repo.ListDictionaryItems(ctx, typeCode, in.Offset, in.Limit)
	if err != nil {
		return nil, err
	}
	s.cacheSet(key, items)
	s.trackDictItemListKey(key)
	return items, nil
}

func (s *serviceImpl) DeleteDictionaryItem(ctx context.Context, id string) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	itemID := shared.ID(strings.TrimSpace(id))
	if itemID.IsZero() {
		return fmt.Errorf("dictionary item id is required")
	}
	existing, err := s.repo.GetDictionaryItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteDictionaryItem(ctx, itemID); err != nil {
		return err
	}
	if existing != nil {
		s.invalidateDictionaryItemsForType(existing.TypeCode)
	} else {
		s.invalidateDictionaryItemsForType("")
	}
	return nil
}

func (s *serviceImpl) RegisterConfigSchema(_ context.Context, in ConfigSchemaInput) (*ConfigSchema, error) {
	if s == nil {
		return nil, fmt.Errorf("system service is not configured")
	}
	schema, err := normalizeConfigSchema(in)
	if err != nil {
		return nil, err
	}
	s.configMu.Lock()
	if s.configSchemas == nil {
		s.configSchemas = map[string]ConfigSchema{}
	}
	s.configSchemas[configSchemaRegistryKey(schema.Scope, schema.Owner)] = schema
	s.configMu.Unlock()
	out := cloneConfigSchema(schema)
	return &out, nil
}

func (s *serviceImpl) GetConfigSchema(_ context.Context, scope ConfigScope, owner string) (*ConfigSchema, error) {
	if s == nil {
		return nil, fmt.Errorf("system service is not configured")
	}
	normalizedScope, normalizedOwner, err := normalizeConfigScopeOwner(scope, owner)
	if err != nil {
		return nil, err
	}
	s.configMu.RLock()
	schema, ok := s.configSchemas[configSchemaRegistryKey(normalizedScope, normalizedOwner)]
	s.configMu.RUnlock()
	if !ok {
		return nil, nil
	}
	out := cloneConfigSchema(schema)
	return &out, nil
}

func (s *serviceImpl) ListConfigSchemas(_ context.Context, in ConfigSchemaListInput) ([]ConfigSchema, error) {
	if s == nil {
		return nil, fmt.Errorf("system service is not configured")
	}
	scope := ConfigScope(strings.ToLower(strings.TrimSpace(string(in.Scope))))
	if scope != "" && scope != ConfigScopeSystem && scope != ConfigScopePlugin {
		return nil, fmt.Errorf("config schema scope is invalid")
	}
	s.configMu.RLock()
	items := make([]ConfigSchema, 0, len(s.configSchemas))
	for _, schema := range s.configSchemas {
		if scope != "" && schema.Scope != scope {
			continue
		}
		items = append(items, cloneConfigSchema(schema))
	}
	s.configMu.RUnlock()
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Scope == items[j].Scope {
			return items[i].Owner < items[j].Owner
		}
		return items[i].Scope < items[j].Scope
	})
	return items, nil
}

func (s *serviceImpl) ValidateConfigValues(ctx context.Context, scope ConfigScope, owner string, values map[string]any) (map[string]any, error) {
	schema, err := s.GetConfigSchema(ctx, scope, owner)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, fmt.Errorf("config schema is not registered")
	}
	out := make(map[string]any, len(schema.Fields))
	for _, field := range schema.Fields {
		raw, exists := values[field.Key]
		if !exists || raw == nil {
			raw = defaultConfigValue(field)
		}
		normalized, err := normalizeConfigValue(field, raw)
		if err != nil {
			return nil, err
		}
		out[field.Key] = normalized
	}
	return out, nil
}

func defaultConfigValue(field ConfigField) any {
	if field.Default != "" {
		return field.Default
	}
	switch field.Type {
	case ConfigFieldBoolean:
		return false
	case ConfigFieldNumber:
		return 0
	default:
		return ""
	}
}

func DefaultSystemConfigSchema() ConfigSchema {
	auditMin := 1.0
	auditMax := 3650.0
	return ConfigSchema{
		Scope:       ConfigScopeSystem,
		Owner:       "system",
		Title:       "Common Settings",
		TitleZhCN:   "Common Settings",
		TitleEnUS:   "Common Settings",
		Description: "System settings rendered by the shared SchemaForm.",
		Fields: []ConfigField{
			{
				Key:       "audit.retention_days",
				LabelZhCN: "Audit retention days",
				LabelEnUS: "Audit retention days",
				Type:      ConfigFieldNumber,
				Default:   "30",
				Min:       &auditMin,
				Max:       &auditMax,
				Help:      "Controls audit cleanup retention in days.",
			},
			{
				Key:       "plugin.auto_enable",
				LabelZhCN: "Auto-enable installed plugins",
				LabelEnUS: "Auto-enable installed plugins",
				Type:      ConfigFieldBoolean,
				Default:   "false",
			},
			{
				Key:       "plugin.dev_portal_enabled",
				LabelZhCN: "Developer portal enabled",
				LabelEnUS: "Developer portal enabled",
				Type:      ConfigFieldBoolean,
				Default:   "true",
			},
			{
				Key:         "skoll.menu.tree",
				LabelZhCN:   "Menu tree JSON",
				LabelEnUS:   "Menu tree JSON",
				Type:        ConfigFieldTextarea,
				Default:     "",
				Placeholder: "[]",
			},
		},
	}
}

func defaultConfigSchemas() map[string]ConfigSchema {
	schema := DefaultSystemConfigSchema()
	return map[string]ConfigSchema{
		configSchemaRegistryKey(schema.Scope, schema.Owner): schema,
	}
}

func normalizeConfigSchema(in ConfigSchemaInput) (ConfigSchema, error) {
	scope, owner, err := normalizeConfigScopeOwner(in.Scope, in.Owner)
	if err != nil {
		return ConfigSchema{}, err
	}
	schema := ConfigSchema{
		Scope:       scope,
		Owner:       owner,
		Title:       strings.TrimSpace(in.Title),
		TitleZhCN:   strings.TrimSpace(in.TitleZhCN),
		TitleEnUS:   strings.TrimSpace(in.TitleEnUS),
		Description: strings.TrimSpace(in.Description),
		Fields:      make([]ConfigField, 0, len(in.Fields)),
	}
	if len(in.Fields) == 0 {
		return ConfigSchema{}, fmt.Errorf("config schema fields are required")
	}
	seen := map[string]struct{}{}
	for _, field := range in.Fields {
		normalized, err := normalizeConfigField(field)
		if err != nil {
			return ConfigSchema{}, err
		}
		if _, exists := seen[normalized.Key]; exists {
			return ConfigSchema{}, fmt.Errorf("config field key is duplicated: %s", normalized.Key)
		}
		seen[normalized.Key] = struct{}{}
		schema.Fields = append(schema.Fields, normalized)
	}
	return schema, nil
}

func normalizeConfigScopeOwner(scope ConfigScope, owner string) (ConfigScope, string, error) {
	normalizedScope := ConfigScope(strings.ToLower(strings.TrimSpace(string(scope))))
	if normalizedScope == "" {
		normalizedScope = ConfigScopeSystem
	}
	switch normalizedScope {
	case ConfigScopeSystem:
		return ConfigScopeSystem, "system", nil
	case ConfigScopePlugin:
		normalizedOwner := strings.TrimSpace(owner)
		if normalizedOwner == "" {
			return "", "", fmt.Errorf("plugin config schema owner is required")
		}
		return ConfigScopePlugin, normalizedOwner, nil
	default:
		return "", "", fmt.Errorf("config schema scope is invalid")
	}
}

func normalizeConfigField(field ConfigField) (ConfigField, error) {
	out := ConfigField{
		Key:         strings.TrimSpace(field.Key),
		Label:       strings.TrimSpace(field.Label),
		LabelZhCN:   strings.TrimSpace(field.LabelZhCN),
		LabelEnUS:   strings.TrimSpace(field.LabelEnUS),
		Type:        ConfigFieldType(strings.ToLower(strings.TrimSpace(string(field.Type)))),
		Required:    field.Required,
		Default:     field.Default,
		Placeholder: strings.TrimSpace(field.Placeholder),
		Help:        strings.TrimSpace(field.Help),
		Min:         cloneFloat64Ptr(field.Min),
		Max:         cloneFloat64Ptr(field.Max),
		MinLength:   cloneIntPtr(field.MinLength),
		MaxLength:   cloneIntPtr(field.MaxLength),
		Pattern:     strings.TrimSpace(field.Pattern),
		Options:     make([]ConfigOption, 0, len(field.Options)),
	}
	if !configKeyPattern.MatchString(out.Key) {
		return ConfigField{}, fmt.Errorf("config field key is invalid: %s", out.Key)
	}
	if out.Type == "" {
		out.Type = ConfigFieldString
	}
	switch out.Type {
	case ConfigFieldString, ConfigFieldTextarea, ConfigFieldNumber, ConfigFieldBoolean, ConfigFieldSelect:
	default:
		return ConfigField{}, fmt.Errorf("config field type is invalid: %s", out.Type)
	}
	if out.Min != nil && out.Max != nil && *out.Min > *out.Max {
		return ConfigField{}, fmt.Errorf("config field min cannot be greater than max: %s", out.Key)
	}
	if out.MinLength != nil && *out.MinLength < 0 {
		return ConfigField{}, fmt.Errorf("config field minLength cannot be negative: %s", out.Key)
	}
	if out.MaxLength != nil && *out.MaxLength < 0 {
		return ConfigField{}, fmt.Errorf("config field maxLength cannot be negative: %s", out.Key)
	}
	if out.MinLength != nil && out.MaxLength != nil && *out.MinLength > *out.MaxLength {
		return ConfigField{}, fmt.Errorf("config field minLength cannot be greater than maxLength: %s", out.Key)
	}
	if out.Pattern != "" {
		if _, err := regexp.Compile(out.Pattern); err != nil {
			return ConfigField{}, fmt.Errorf("config field pattern is invalid: %s", out.Key)
		}
	}
	seenOptions := map[string]struct{}{}
	for _, option := range field.Options {
		value := strings.TrimSpace(option.Value)
		if value == "" {
			return ConfigField{}, fmt.Errorf("config field option value is required: %s", out.Key)
		}
		if _, exists := seenOptions[value]; exists {
			return ConfigField{}, fmt.Errorf("config field option value is duplicated: %s", out.Key)
		}
		seenOptions[value] = struct{}{}
		out.Options = append(out.Options, ConfigOption{
			Label:     strings.TrimSpace(option.Label),
			LabelZhCN: strings.TrimSpace(option.LabelZhCN),
			LabelEnUS: strings.TrimSpace(option.LabelEnUS),
			Value:     value,
		})
	}
	if out.Type == ConfigFieldSelect && len(out.Options) == 0 {
		return ConfigField{}, fmt.Errorf("config field select options are required: %s", out.Key)
	}
	if out.Default != "" {
		if _, err := normalizeConfigValue(out, out.Default); err != nil {
			return ConfigField{}, err
		}
	}
	return out, nil
}

func normalizeConfigValue(field ConfigField, raw any) (any, error) {
	switch field.Type {
	case ConfigFieldNumber:
		value, err := parseConfigNumber(raw)
		if err != nil {
			return nil, fmt.Errorf("config field %s must be a number", field.Key)
		}
		if field.Min != nil && value < *field.Min {
			return nil, fmt.Errorf("config field %s cannot be less than %v", field.Key, *field.Min)
		}
		if field.Max != nil && value > *field.Max {
			return nil, fmt.Errorf("config field %s cannot be greater than %v", field.Key, *field.Max)
		}
		return value, nil
	case ConfigFieldBoolean:
		value, err := parseConfigBool(raw)
		if err != nil {
			return nil, fmt.Errorf("config field %s must be a boolean", field.Key)
		}
		return value, nil
	default:
		value := strings.TrimSpace(fmt.Sprint(raw))
		if field.Required && value == "" {
			return nil, fmt.Errorf("config field %s is required", field.Key)
		}
		if field.MinLength != nil && len(value) < *field.MinLength {
			return nil, fmt.Errorf("config field %s length cannot be less than %d", field.Key, *field.MinLength)
		}
		if field.MaxLength != nil && len(value) > *field.MaxLength {
			return nil, fmt.Errorf("config field %s length cannot be greater than %d", field.Key, *field.MaxLength)
		}
		if field.Pattern != "" {
			matched, err := regexp.MatchString(field.Pattern, value)
			if err != nil || !matched {
				return nil, fmt.Errorf("config field %s format is invalid", field.Key)
			}
		}
		if field.Type == ConfigFieldSelect {
			for _, option := range field.Options {
				if option.Value == value {
					return value, nil
				}
			}
			return nil, fmt.Errorf("config field %s option is invalid", field.Key)
		}
		return value, nil
	}
}

func parseConfigNumber(raw any) (float64, error) {
	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case json.Number:
		return v.Float64()
	case string:
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	default:
		return strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(raw)), 64)
	}
}

func parseConfigBool(raw any) (bool, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(v))
	default:
		return strconv.ParseBool(strings.TrimSpace(fmt.Sprint(raw)))
	}
}

func configSchemaRegistryKey(scope ConfigScope, owner string) string {
	return strings.ToLower(strings.TrimSpace(string(scope))) + ":" + strings.TrimSpace(owner)
}

func cloneConfigSchema(schema ConfigSchema) ConfigSchema {
	out := schema
	out.Fields = make([]ConfigField, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		out.Fields = append(out.Fields, cloneConfigField(field))
	}
	return out
}

func cloneConfigField(field ConfigField) ConfigField {
	out := field
	out.Min = cloneFloat64Ptr(field.Min)
	out.Max = cloneFloat64Ptr(field.Max)
	out.MinLength = cloneIntPtr(field.MinLength)
	out.MaxLength = cloneIntPtr(field.MaxLength)
	out.Options = append([]ConfigOption(nil), field.Options...)
	return out
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}

func (s *serviceImpl) requireRepo() error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("system repository is not configured")
	}
	return nil
}

func (s *serviceImpl) cacheGet(key string, out any) bool {
	if s == nil || s.dictionaryCache == nil {
		return false
	}
	data, ok := s.dictionaryCache.Get(key)
	if !ok {
		return false
	}
	return json.Unmarshal(data, out) == nil
}

func (s *serviceImpl) cacheSet(key string, value any) {
	if s == nil || s.dictionaryCache == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	s.dictionaryCache.Set(key, data, s.dictionaryTTL)
}

func (s *serviceImpl) cacheDelete(key string) {
	if s == nil || s.dictionaryCache == nil || key == "" {
		return
	}
	s.dictionaryCache.Delete(key)
}

func (s *serviceImpl) trackDictListKey(key string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.dictListKeys[key] = struct{}{}
}

func (s *serviceImpl) trackDictItemKey(key string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.dictItemKeys[key] = struct{}{}
}

func (s *serviceImpl) trackDictItemListKey(key string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.dictItemListKeys[key] = struct{}{}
}

func (s *serviceImpl) invalidateDictionaryType(code string) {
	s.cacheDelete(dictionaryTypeCacheKey(code))
	s.invalidateDictionaryLists()
}

func (s *serviceImpl) invalidateDictionaryLists() {
	s.cacheMu.Lock()
	keys := make([]string, 0, len(s.dictListKeys))
	for key := range s.dictListKeys {
		keys = append(keys, key)
		delete(s.dictListKeys, key)
	}
	s.cacheMu.Unlock()
	for _, key := range keys {
		s.cacheDelete(key)
	}
}

func (s *serviceImpl) invalidateDictionaryItemsForType(typeCode string) {
	typeCode = normalizeDictionaryCode(typeCode)
	s.cacheMu.Lock()
	itemKeys := make([]string, 0, len(s.dictItemKeys))
	for key := range s.dictItemKeys {
		if typeCode == "" || strings.Contains(key, ":"+typeCode+":") {
			itemKeys = append(itemKeys, key)
			delete(s.dictItemKeys, key)
		}
	}
	listKeys := make([]string, 0, len(s.dictItemListKeys))
	for key := range s.dictItemListKeys {
		if typeCode == "" || strings.Contains(key, ":"+typeCode+":") {
			listKeys = append(listKeys, key)
			delete(s.dictItemListKeys, key)
		}
	}
	s.cacheMu.Unlock()
	for _, key := range itemKeys {
		s.cacheDelete(key)
	}
	for _, key := range listKeys {
		s.cacheDelete(key)
	}
}

func normalizeDictionaryCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func dictionaryTypeCacheKey(code string) string {
	return "dictionary:type:code:" + normalizeDictionaryCode(code)
}

func dictionaryTypeListCacheKey(offset, limit int) string {
	return fmt.Sprintf("dictionary:types:list:%d:%d", offset, limit)
}

func dictionaryItemCacheKey(typeCode, value string) string {
	return "dictionary:item:" + normalizeDictionaryCode(typeCode) + ":" + strings.TrimSpace(value)
}

func dictionaryItemListCacheKey(typeCode string, offset, limit int) string {
	return fmt.Sprintf("dictionary:items:list:%s:%d:%d", normalizeDictionaryCode(typeCode), offset, limit)
}
