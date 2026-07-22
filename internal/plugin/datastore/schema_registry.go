package datastore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	MaxTablesPerPlugin = 128
	MaxFieldsPerTable  = 128
	MaxIndexesPerTable = 32
	MaxIndexFields     = 8
	MaxLogicalTableLen = 42
)

const (
	FieldTenantID       = "tenant_id"
	FieldOrganizationID = "organization_id"
	FieldOwnerID        = "owner_id"
	FieldVersion        = "version"
	FieldCreatedAt      = "created_at"
	FieldUpdatedAt      = "updated_at"
)

var (
	pluginIDPattern         = regexp.MustCompile(`^[a-z][a-z0-9_-]{2,63}$`)
	schemaIdentifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	reservedSchemaNames     = map[string]struct{}{
		"core": {}, "information_schema": {}, "internal": {}, "mysql": {}, "postgres": {},
		"skoll": {}, "sqlite": {}, "system": {},
	}
	reservedSchemaPrefixes = []string{"core_", "information_schema_", "internal_", "mysql_", "pg_", "sk_", "skoll_", "skp_", "sqlite_", "system_"}
	hostFields             = []FieldSchema{
		{Name: FieldTenantID, Type: pluginsdk.DataValueString, Filterable: false, Sortable: false, HostManaged: true},
		{Name: FieldOrganizationID, Type: pluginsdk.DataValueString, Filterable: false, Sortable: false, HostManaged: true},
		{Name: FieldOwnerID, Type: pluginsdk.DataValueString, Filterable: false, Sortable: false, HostManaged: true},
		{Name: FieldVersion, Type: pluginsdk.DataValueInteger, Filterable: true, Sortable: true, HostManaged: true},
		{Name: FieldCreatedAt, Type: pluginsdk.DataValueTimestamp, Filterable: true, Sortable: true, HostManaged: true},
		{Name: FieldUpdatedAt, Type: pluginsdk.DataValueTimestamp, Filterable: true, Sortable: true, HostManaged: true},
	}
)

type FieldSchema struct {
	Name        string
	Type        pluginsdk.DataValueType
	Nullable    bool
	Mutable     bool
	Filterable  bool
	Sortable    bool
	HostManaged bool
}

type IndexSchema struct {
	Fields []string
	Unique bool
}

type TableSchema struct {
	Name       string
	Fields     []FieldSchema
	PrimaryKey []string
	Indexes    []IndexSchema
}

type PluginSchema struct {
	PluginID string
	Tables   []TableSchema
}

type ResolvedTable struct {
	PluginID     string
	Namespace    string
	LogicalName  string
	PhysicalName string
	Fields       map[string]FieldSchema
	PrimaryKey   []string
	Indexes      []IndexSchema
}

type RegisteredSchema struct {
	PluginID  string
	Namespace string
	Tables    []ResolvedTable
}

type SchemaRegistry struct {
	mu         sync.RWMutex
	plugins    map[string]RegisteredSchema
	namespaces map[string]string
}

func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{plugins: make(map[string]RegisteredSchema), namespaces: make(map[string]string)}
}

func (r *SchemaRegistry) Register(schema PluginSchema) (RegisteredSchema, error) {
	if r == nil {
		return RegisteredSchema{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "registry", "schema registry is unavailable", true)
	}
	registered, err := buildRegisteredSchema(schema)
	if err != nil {
		return RegisteredSchema{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.plugins[registered.PluginID]; exists {
		return RegisteredSchema{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "pluginId", "plugin schema is already registered", false)
	}
	if owner, exists := r.namespaces[registered.Namespace]; exists && owner != registered.PluginID {
		return RegisteredSchema{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "namespace", "plugin namespace collision", false)
	}
	r.plugins[registered.PluginID] = cloneRegisteredSchema(registered)
	r.namespaces[registered.Namespace] = registered.PluginID
	return cloneRegisteredSchema(registered), nil
}

func (r *SchemaRegistry) Resolve(pluginID, table string) (ResolvedTable, error) {
	if r == nil {
		return ResolvedTable{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "registry", "schema registry is unavailable", true)
	}
	pluginID = strings.TrimSpace(pluginID)
	table = strings.TrimSpace(table)
	if !pluginIDPattern.MatchString(pluginID) {
		return ResolvedTable{}, invalidSchema("pluginId", "plugin id is invalid")
	}
	if err := validateLogicalName("table", table, MaxLogicalTableLen); err != nil {
		return ResolvedTable{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	registered, exists := r.plugins[pluginID]
	if !exists {
		return ResolvedTable{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "pluginId", "plugin schema is not registered", false)
	}
	for _, candidate := range registered.Tables {
		if candidate.LogicalName == table {
			return cloneResolvedTable(candidate), nil
		}
	}
	return ResolvedTable{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "table", "table is not declared by the plugin", false)
}

func (r *SchemaRegistry) ResolveNamespace(pluginID, namespace string) (RegisteredSchema, error) {
	if r == nil {
		return RegisteredSchema{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "registry", "schema registry is unavailable", true)
	}
	pluginID = strings.TrimSpace(pluginID)
	namespace = strings.TrimSpace(namespace)
	if !pluginIDPattern.MatchString(pluginID) {
		return RegisteredSchema{}, invalidSchema("pluginId", "plugin id is invalid")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	owner, exists := r.namespaces[namespace]
	if !exists {
		return RegisteredSchema{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "namespace", "plugin namespace is not registered", false)
	}
	if owner != pluginID {
		return RegisteredSchema{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "namespace", "plugin does not own the requested namespace", false)
	}
	return cloneRegisteredSchema(r.plugins[owner]), nil
}

func (r *SchemaRegistry) Snapshot(pluginID string) (RegisteredSchema, bool) {
	if r == nil {
		return RegisteredSchema{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	registered, exists := r.plugins[strings.TrimSpace(pluginID)]
	return cloneRegisteredSchema(registered), exists
}

func (r *SchemaRegistry) Unregister(pluginID string) bool {
	if r == nil {
		return false
	}
	pluginID = strings.TrimSpace(pluginID)
	r.mu.Lock()
	defer r.mu.Unlock()
	registered, exists := r.plugins[pluginID]
	if !exists {
		return false
	}
	delete(r.plugins, pluginID)
	delete(r.namespaces, registered.Namespace)
	return true
}

func PluginNamespace(pluginID string) (string, error) {
	pluginID = strings.TrimSpace(pluginID)
	if !pluginIDPattern.MatchString(pluginID) {
		return "", invalidSchema("pluginId", "plugin id is invalid")
	}
	if isReservedSchemaName(pluginID) {
		return "", invalidSchema("pluginId", "plugin id uses a reserved database identifier")
	}
	sum := sha256.Sum256([]byte(pluginID))
	return "skp_" + hex.EncodeToString(sum[:8]), nil
}

func buildRegisteredSchema(schema PluginSchema) (RegisteredSchema, error) {
	pluginID := strings.TrimSpace(schema.PluginID)
	namespace, err := PluginNamespace(pluginID)
	if err != nil {
		return RegisteredSchema{}, err
	}
	if len(schema.Tables) == 0 || len(schema.Tables) > MaxTablesPerPlugin {
		return RegisteredSchema{}, invalidSchema("tables", "table count is outside the supported range")
	}
	seenTables := make(map[string]struct{}, len(schema.Tables))
	registered := RegisteredSchema{PluginID: pluginID, Namespace: namespace, Tables: make([]ResolvedTable, 0, len(schema.Tables))}
	for index, table := range schema.Tables {
		resolved, buildErr := buildResolvedTable(pluginID, namespace, table, fmt.Sprintf("tables[%d]", index))
		if buildErr != nil {
			return RegisteredSchema{}, buildErr
		}
		if _, exists := seenTables[resolved.LogicalName]; exists {
			return RegisteredSchema{}, invalidSchema(fmt.Sprintf("tables[%d].name", index), "table is duplicated")
		}
		seenTables[resolved.LogicalName] = struct{}{}
		registered.Tables = append(registered.Tables, resolved)
	}
	sort.Slice(registered.Tables, func(i, j int) bool { return registered.Tables[i].LogicalName < registered.Tables[j].LogicalName })
	return registered, nil
}

func buildResolvedTable(pluginID, namespace string, table TableSchema, path string) (ResolvedTable, error) {
	name := strings.TrimSpace(table.Name)
	if err := validateLogicalName(path+".name", name, MaxLogicalTableLen); err != nil {
		return ResolvedTable{}, err
	}
	if len(table.Fields) == 0 || len(table.Fields) > MaxFieldsPerTable {
		return ResolvedTable{}, invalidSchema(path+".fields", "field count is outside the supported range")
	}
	if len(table.PrimaryKey) == 0 || len(table.PrimaryKey) > 4 {
		return ResolvedTable{}, invalidSchema(path+".primaryKey", "primary key width is outside the supported range")
	}
	if len(table.Indexes) > MaxIndexesPerTable {
		return ResolvedTable{}, invalidSchema(path+".indexes", "index count exceeds limit")
	}

	fields := make(map[string]FieldSchema, len(table.Fields)+len(hostFields))
	for _, field := range hostFields {
		fields[field.Name] = field
	}
	for index, field := range table.Fields {
		fieldPath := fmt.Sprintf("%s.fields[%d]", path, index)
		field.Name = strings.TrimSpace(field.Name)
		if err := validateLogicalName(fieldPath+".name", field.Name, 63); err != nil {
			return ResolvedTable{}, err
		}
		if existing, exists := fields[field.Name]; exists {
			if existing.HostManaged {
				return ResolvedTable{}, invalidSchema(fieldPath+".name", "field is reserved by the host")
			}
			return ResolvedTable{}, invalidSchema(fieldPath+".name", "field is duplicated")
		}
		if !validStorageType(field.Type) {
			return ResolvedTable{}, invalidSchema(fieldPath+".type", "field storage type is unsupported")
		}
		if (field.Type == pluginsdk.DataValueBytes || field.Type == pluginsdk.DataValueJSON) && (field.Filterable || field.Sortable) {
			return ResolvedTable{}, invalidSchema(fieldPath, "bytes and json fields cannot be filterable or sortable")
		}
		if field.HostManaged {
			return ResolvedTable{}, invalidSchema(fieldPath+".hostManaged", "plugin fields cannot be host managed")
		}
		fields[field.Name] = field
	}

	primaryKey := append([]string(nil), table.PrimaryKey...)
	seenKeys := make(map[string]struct{}, len(primaryKey))
	for index, fieldName := range primaryKey {
		fieldName = strings.TrimSpace(fieldName)
		primaryKey[index] = fieldName
		field, exists := fields[fieldName]
		if !exists || field.HostManaged {
			return ResolvedTable{}, invalidSchema(fmt.Sprintf("%s.primaryKey[%d]", path, index), "primary key field must be plugin declared")
		}
		if field.Nullable || field.Mutable {
			return ResolvedTable{}, invalidSchema(fmt.Sprintf("%s.primaryKey[%d]", path, index), "primary key field must be immutable and non-null")
		}
		if _, duplicate := seenKeys[fieldName]; duplicate {
			return ResolvedTable{}, invalidSchema(fmt.Sprintf("%s.primaryKey[%d]", path, index), "primary key field is duplicated")
		}
		seenKeys[fieldName] = struct{}{}
	}

	indexes := make([]IndexSchema, 0, len(table.Indexes))
	seenIndexes := make(map[string]struct{}, len(table.Indexes))
	for index, item := range table.Indexes {
		indexPath := fmt.Sprintf("%s.indexes[%d]", path, index)
		if len(item.Fields) == 0 || len(item.Fields) > MaxIndexFields {
			return ResolvedTable{}, invalidSchema(indexPath+".fields", "index width is outside the supported range")
		}
		fieldsCopy := append([]string(nil), item.Fields...)
		seenIndexFields := make(map[string]struct{}, len(fieldsCopy))
		for fieldIndex, fieldName := range fieldsCopy {
			fieldName = strings.TrimSpace(fieldName)
			fieldsCopy[fieldIndex] = fieldName
			field, exists := fields[fieldName]
			if !exists || (!field.Filterable && !field.Sortable) {
				return ResolvedTable{}, invalidSchema(fmt.Sprintf("%s.fields[%d]", indexPath, fieldIndex), "index field must be declared and queryable")
			}
			if _, duplicate := seenIndexFields[fieldName]; duplicate {
				return ResolvedTable{}, invalidSchema(fmt.Sprintf("%s.fields[%d]", indexPath, fieldIndex), "index field is duplicated")
			}
			seenIndexFields[fieldName] = struct{}{}
		}
		fingerprint := strings.Join(fieldsCopy, ",")
		if _, duplicate := seenIndexes[fingerprint]; duplicate {
			return ResolvedTable{}, invalidSchema(indexPath, "index is duplicated")
		}
		seenIndexes[fingerprint] = struct{}{}
		indexes = append(indexes, IndexSchema{Fields: fieldsCopy, Unique: item.Unique})
	}

	physicalName := namespace + "_" + name
	if len(physicalName) > 63 {
		return ResolvedTable{}, invalidSchema(path+".name", "physical table name exceeds database limit")
	}
	return ResolvedTable{
		PluginID: pluginID, Namespace: namespace, LogicalName: name, PhysicalName: physicalName,
		Fields: fields, PrimaryKey: primaryKey, Indexes: indexes,
	}, nil
}

func validateLogicalName(path, value string, maximum int) error {
	if len(value) == 0 || len(value) > maximum || !schemaIdentifierPattern.MatchString(value) || isReservedSchemaName(value) {
		return invalidSchema(path, "database identifier is invalid or reserved")
	}
	return nil
}

func isReservedSchemaName(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, reserved := reservedSchemaNames[value]; reserved {
		return true
	}
	for _, prefix := range reservedSchemaPrefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func validStorageType(value pluginsdk.DataValueType) bool {
	switch value {
	case pluginsdk.DataValueString, pluginsdk.DataValueInteger, pluginsdk.DataValueDecimal,
		pluginsdk.DataValueBoolean, pluginsdk.DataValueTimestamp, pluginsdk.DataValueBytes, pluginsdk.DataValueJSON:
		return true
	default:
		return false
	}
}

func invalidSchema(field, message string) error {
	return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, field, message, false)
}

func cloneRegisteredSchema(schema RegisteredSchema) RegisteredSchema {
	cloned := RegisteredSchema{PluginID: schema.PluginID, Namespace: schema.Namespace, Tables: make([]ResolvedTable, len(schema.Tables))}
	for index, table := range schema.Tables {
		cloned.Tables[index] = cloneResolvedTable(table)
	}
	return cloned
}

func cloneResolvedTable(table ResolvedTable) ResolvedTable {
	cloned := table
	cloned.Fields = make(map[string]FieldSchema, len(table.Fields))
	for name, field := range table.Fields {
		cloned.Fields[name] = field
	}
	cloned.PrimaryKey = append([]string(nil), table.PrimaryKey...)
	cloned.Indexes = make([]IndexSchema, len(table.Indexes))
	for index, item := range table.Indexes {
		cloned.Indexes[index] = IndexSchema{Fields: append([]string(nil), item.Fields...), Unique: item.Unique}
	}
	return cloned
}
