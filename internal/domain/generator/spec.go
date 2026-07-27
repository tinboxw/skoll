package generator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

var (
	generatorNamePattern     = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	generatorKeyPattern      = regexp.MustCompile(`^[a-z][a-z0-9_.\-]{1,127}$`)
	generatorPermKeyPattern  = regexp.MustCompile(`^[a-z][a-z0-9_:.\-]{1,127}$`)
	generatorPluginIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)
)

type FieldType string

const (
	FieldTypeString  FieldType = "string"
	FieldTypeText    FieldType = "text"
	FieldTypeInt     FieldType = "int"
	FieldTypeDecimal FieldType = "decimal"
	FieldTypeBool    FieldType = "bool"
	FieldTypeTime    FieldType = "time"
	FieldTypeJSON    FieldType = "json"
	FieldTypeID      FieldType = "id"
)

type ValidationRuleType string

const (
	ValidationRequired ValidationRuleType = "required"
	ValidationMin      ValidationRuleType = "min"
	ValidationMax      ValidationRuleType = "max"
	ValidationPattern  ValidationRuleType = "pattern"
	ValidationEnum     ValidationRuleType = "enum"
)

type GeneratorSpecInput struct {
	ID          shared.ID
	Module      ModuleSpec
	Table       TableSpec
	Fields      []FieldSpec
	Indexes     []IndexSpec
	Permissions PermissionSpec
	Menu        MenuSpec
	Page        PageSpec
	Audit       AuditSpec
	Plugin      PluginSpec
	Document    *DocumentSpec
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GeneratorSpec struct {
	ID          shared.ID
	Module      ModuleSpec
	Table       TableSpec
	Fields      []FieldSpec
	Indexes     []IndexSpec
	Permissions PermissionSpec
	Menu        MenuSpec
	Page        PageSpec
	Audit       AuditSpec
	Plugin      PluginSpec
	Document    *DocumentSpec `json:"document,omitempty"`
	Meta        shared.AuditMeta
}

type ModuleSpec struct {
	Name        string
	Package     string
	DisplayName string
	Description string
}

type TableSpec struct {
	Name           string
	DomainName     string
	CollectionName string
	Comment        string
}

type FieldSpec struct {
	Name        string
	ColumnName  string
	Label       string
	Type        FieldType
	GoType      string
	TypeScript  string
	PrimaryKey  bool
	Required    bool
	Nullable    bool
	Unique      bool
	Filterable  bool
	Sortable    bool
	ListVisible bool
	FormVisible bool
	Default     string
	Reference   *ReferenceSpec
	Validation  []ValidationRule
}

type ReferenceSpec struct {
	Module string
	Table  string
	Field  string
}

type ValidationRule struct {
	Type    ValidationRuleType
	Value   string
	Message string
}

type IndexSpec struct {
	Name   string
	Fields []string
	Unique bool
}

type PermissionSpec struct {
	Resource  string
	ReadKey   string
	CreateKey string
	UpdateKey string
	DeleteKey string
	ManageKey string
}

type MenuSpec struct {
	Key                 string
	ParentKey           string
	Path                string
	Component           string
	Icon                string
	Order               int
	RequiredPermissions []string
}

type PageSpec struct {
	Title     string
	RouteName string
	List      PageListSpec
	Form      PageFormSpec
}

type PageListSpec struct {
	Columns []string
	Filters []string
	Actions []string
}

type PageFormSpec struct {
	Fields []string
	Mode   string
}

type AuditSpec struct {
	Resource string
	Actions  []string
}

type PluginSpec struct {
	Enabled            bool
	ID                 string
	Name               string
	Version            string
	Description        string
	DataNamespace      string
	MigrationDirectory string
	UninstallPolicy    string
	RollbackPolicy     string
	FrontendEntry      string
	UIMode             string
	ServiceBaseURL     string
	ServiceHealthURL   string
	EventPublications  []PluginEventPublicationSpec
	EventSubscriptions []PluginEventSubscriptionSpec
}

type PluginEventPublicationSpec struct {
	Name          string
	SchemaVersion uint32
	PayloadType   string
	Scope         string
}

type PluginEventSubscriptionSpec struct {
	Publisher      string
	Name           string
	SchemaVersions []uint32
	Handler        string
	RetryPolicy    string
}

type DocumentSpec struct {
	Enabled      bool
	SchemaKey    string
	SchemaName   string
	DefinitionID string
	NumberPrefix string
	TitleField   string
}

func NewGeneratorSpec(in GeneratorSpecInput) (*GeneratorSpec, error) {
	in = normalizeGeneratorSpecInput(in)
	if err := validateGeneratorSpecInput(in); err != nil {
		return nil, err
	}
	return &GeneratorSpec{
		ID:          in.ID,
		Module:      in.Module,
		Table:       in.Table,
		Fields:      append([]FieldSpec(nil), in.Fields...),
		Indexes:     append([]IndexSpec(nil), in.Indexes...),
		Permissions: in.Permissions,
		Menu:        in.Menu,
		Page:        in.Page,
		Audit:       in.Audit,
		Plugin:      in.Plugin,
		Document:    cloneDocumentSpec(in.Document),
		Meta: shared.AuditMeta{
			CreatedAt: in.CreatedAt,
			UpdatedAt: in.UpdatedAt,
		},
	}, nil
}

func (s GeneratorSpec) FieldByName(name string) (FieldSpec, bool) {
	target := normalizeName(name)
	for _, field := range s.Fields {
		if normalizeName(field.Name) == target {
			return field, true
		}
	}
	return FieldSpec{}, false
}

func (s GeneratorSpec) FieldNames() []string {
	out := make([]string, 0, len(s.Fields))
	for _, field := range s.Fields {
		out = append(out, field.Name)
	}
	return out
}

func normalizeGeneratorSpecInput(in GeneratorSpecInput) GeneratorSpecInput {
	in.Module = normalizeModuleSpec(in.Module)
	in.Table = normalizeTableSpec(in.Table)
	in.Fields = normalizeFields(in.Fields)
	in.Indexes = normalizeIndexes(in.Indexes)
	in.Permissions = normalizePermissionSpec(in.Permissions)
	in.Menu = normalizeMenuSpec(in.Menu)
	in.Page = normalizePageSpec(in.Page)
	in.Audit = normalizeAuditSpec(in.Audit)
	in.Plugin = normalizePluginSpec(in.Plugin, in)
	in.Document = normalizeDocumentSpec(in.Document, in)
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = in.CreatedAt
	}
	return in
}

func normalizeDocumentSpec(in *DocumentSpec, spec GeneratorSpecInput) *DocumentSpec {
	if in == nil || !in.Enabled {
		return nil
	}
	out := *in
	out.SchemaKey = normalizeName(out.SchemaKey)
	if out.SchemaKey == "" {
		out.SchemaKey = spec.Module.Name
	}
	out.SchemaName = strings.TrimSpace(out.SchemaName)
	if out.SchemaName == "" {
		out.SchemaName = spec.Module.DisplayName
	}
	out.DefinitionID = strings.TrimSpace(out.DefinitionID)
	if out.DefinitionID == "" {
		out.DefinitionID = out.SchemaKey + "-approval"
	}
	out.NumberPrefix = strings.ToUpper(strings.TrimSpace(out.NumberPrefix))
	if out.NumberPrefix == "" {
		out.NumberPrefix = strings.ToUpper(strings.ReplaceAll(out.SchemaKey, "_", "-"))
	}
	out.TitleField = normalizeName(out.TitleField)
	return &out
}

func cloneDocumentSpec(in *DocumentSpec) *DocumentSpec {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func normalizeModuleSpec(in ModuleSpec) ModuleSpec {
	in.Name = normalizeName(in.Name)
	in.Package = normalizeName(in.Package)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Description = strings.TrimSpace(in.Description)
	return in
}

func normalizeTableSpec(in TableSpec) TableSpec {
	in.Name = normalizeName(in.Name)
	in.DomainName = strings.TrimSpace(in.DomainName)
	in.CollectionName = strings.TrimSpace(in.CollectionName)
	in.Comment = strings.TrimSpace(in.Comment)
	return in
}

func normalizeFields(items []FieldSpec) []FieldSpec {
	out := make([]FieldSpec, 0, len(items))
	for _, item := range items {
		item.Name = normalizeName(item.Name)
		item.ColumnName = normalizeName(item.ColumnName)
		item.Label = strings.TrimSpace(item.Label)
		item.GoType = strings.TrimSpace(item.GoType)
		item.TypeScript = strings.TrimSpace(item.TypeScript)
		item.Default = strings.TrimSpace(item.Default)
		if item.Type == "" {
			item.Type = FieldTypeString
		}
		if item.ColumnName == "" {
			item.ColumnName = item.Name
		}
		out = append(out, item)
	}
	return out
}

func normalizeIndexes(items []IndexSpec) []IndexSpec {
	out := make([]IndexSpec, 0, len(items))
	for _, item := range items {
		item.Name = normalizeName(item.Name)
		fields := make([]string, 0, len(item.Fields))
		for _, field := range item.Fields {
			if normalized := normalizeName(field); normalized != "" {
				fields = append(fields, normalized)
			}
		}
		item.Fields = fields
		out = append(out, item)
	}
	return out
}

func normalizePermissionSpec(in PermissionSpec) PermissionSpec {
	in.Resource = normalizePermissionKey(in.Resource)
	in.ReadKey = normalizePermissionKey(in.ReadKey)
	in.CreateKey = normalizePermissionKey(in.CreateKey)
	in.UpdateKey = normalizePermissionKey(in.UpdateKey)
	in.DeleteKey = normalizePermissionKey(in.DeleteKey)
	in.ManageKey = normalizePermissionKey(in.ManageKey)
	return in
}

func normalizeMenuSpec(in MenuSpec) MenuSpec {
	in.Key = normalizePermissionKey(in.Key)
	in.ParentKey = normalizePermissionKey(in.ParentKey)
	in.Path = strings.TrimSpace(in.Path)
	in.Component = strings.TrimSpace(in.Component)
	in.Icon = strings.TrimSpace(in.Icon)
	out := make([]string, 0, len(in.RequiredPermissions))
	for _, key := range in.RequiredPermissions {
		if normalized := normalizePermissionKey(key); normalized != "" {
			out = append(out, normalized)
		}
	}
	in.RequiredPermissions = out
	return in
}

func normalizePageSpec(in PageSpec) PageSpec {
	in.Title = strings.TrimSpace(in.Title)
	in.RouteName = normalizePermissionKey(in.RouteName)
	in.Form.Mode = normalizeName(in.Form.Mode)
	in.List.Columns = normalizeNameList(in.List.Columns)
	in.List.Filters = normalizeNameList(in.List.Filters)
	in.List.Actions = normalizeNameList(in.List.Actions)
	in.Form.Fields = normalizeNameList(in.Form.Fields)
	return in
}

func normalizeAuditSpec(in AuditSpec) AuditSpec {
	in.Resource = normalizePermissionKey(in.Resource)
	in.Actions = normalizeNameList(in.Actions)
	return in
}

func normalizePluginSpec(in PluginSpec, spec GeneratorSpecInput) PluginSpec {
	if !in.Enabled {
		return in
	}
	in.ID = strings.ToLower(strings.TrimSpace(in.ID))
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		in.Name = spec.Module.DisplayName
	}
	in.Version = strings.TrimSpace(in.Version)
	if in.Version == "" {
		in.Version = "1.0.0"
	}
	in.Description = strings.TrimSpace(in.Description)
	if in.Description == "" {
		in.Description = spec.Module.Description
	}
	in.DataNamespace = normalizeName(strings.ReplaceAll(in.DataNamespace, "-", "_"))
	if in.DataNamespace == "" {
		in.DataNamespace = normalizeName(strings.ReplaceAll(in.ID, "-", "_"))
	}
	in.MigrationDirectory = strings.Trim(strings.TrimSpace(in.MigrationDirectory), "/\\")
	if in.MigrationDirectory == "" {
		in.MigrationDirectory = "migrations"
	}
	in.UninstallPolicy = strings.ToLower(strings.TrimSpace(in.UninstallPolicy))
	if in.UninstallPolicy == "" {
		in.UninstallPolicy = "retain"
	}
	in.RollbackPolicy = strings.ToLower(strings.TrimSpace(in.RollbackPolicy))
	if in.RollbackPolicy == "" {
		in.RollbackPolicy = "automatic"
	}
	in.FrontendEntry = strings.TrimSpace(in.FrontendEntry)
	if in.FrontendEntry == "" && in.ID != "" {
		in.FrontendEntry = "/skoll/plugins/" + in.ID
	}
	in.UIMode = normalizeName(in.UIMode)
	if in.UIMode == "" {
		in.UIMode = "separated"
	}
	in.ServiceBaseURL = strings.TrimRight(strings.TrimSpace(in.ServiceBaseURL), "/")
	in.ServiceHealthURL = strings.TrimSpace(in.ServiceHealthURL)
	publications := make([]PluginEventPublicationSpec, 0, len(in.EventPublications))
	for _, item := range in.EventPublications {
		item.Name = strings.TrimSpace(strings.ToLower(item.Name))
		item.PayloadType = strings.TrimSpace(strings.ToLower(item.PayloadType))
		item.Scope = strings.TrimSpace(strings.ToLower(item.Scope))
		publications = append(publications, item)
	}
	in.EventPublications = publications
	out := make([]PluginEventSubscriptionSpec, 0, len(in.EventSubscriptions))
	for _, item := range in.EventSubscriptions {
		item.Publisher = strings.TrimSpace(strings.ToLower(item.Publisher))
		item.Name = strings.TrimSpace(strings.ToLower(item.Name))
		item.SchemaVersions = append([]uint32(nil), item.SchemaVersions...)
		item.Handler = strings.TrimSpace(item.Handler)
		item.RetryPolicy = strings.TrimSpace(strings.ToLower(item.RetryPolicy))
		if item.RetryPolicy == "" {
			item.RetryPolicy = "standard"
		}
		out = append(out, item)
	}
	in.EventSubscriptions = out
	return in
}

func normalizeNameList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if normalized := normalizeName(item); normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}

func normalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizePermissionKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validateGeneratorSpecInput(in GeneratorSpecInput) error {
	if in.ID.IsZero() {
		return fmt.Errorf("generator spec id is required")
	}
	if in.Module.Name == "" || in.Module.Package == "" || in.Module.DisplayName == "" {
		return fmt.Errorf("generator module spec is incomplete")
	}
	if err := validateGeneratorName(in.Module.Name, "module name"); err != nil {
		return err
	}
	if err := validateGeneratorName(in.Module.Package, "module package"); err != nil {
		return err
	}
	if in.Table.Name == "" || in.Table.DomainName == "" || in.Table.CollectionName == "" {
		return fmt.Errorf("generator table spec is incomplete")
	}
	if err := validateGeneratorName(in.Table.Name, "table name"); err != nil {
		return err
	}
	if len(in.Fields) == 0 {
		return fmt.Errorf("generator fields are required")
	}
	fieldNames := make(map[string]struct{}, len(in.Fields))
	columnNames := make(map[string]struct{}, len(in.Fields))
	primaryCount := 0
	for _, field := range in.Fields {
		if field.Name == "" || field.ColumnName == "" || field.Label == "" || field.Type == "" {
			return fmt.Errorf("generator field spec is incomplete")
		}
		if err := validateGeneratorName(field.Name, "field name"); err != nil {
			return err
		}
		if err := validateGeneratorName(field.ColumnName, "column name"); err != nil {
			return err
		}
		if err := validateFieldType(field.Type); err != nil {
			return err
		}
		if field.PrimaryKey {
			primaryCount++
			if field.Type != FieldTypeID || !field.Required {
				return fmt.Errorf("generator primary field must be a required id")
			}
		}
		if _, ok := fieldNames[field.Name]; ok {
			return fmt.Errorf("generator field name conflict: %s", field.Name)
		}
		if _, ok := columnNames[field.ColumnName]; ok {
			return fmt.Errorf("generator column name conflict: %s", field.ColumnName)
		}
		fieldNames[field.Name] = struct{}{}
		columnNames[field.ColumnName] = struct{}{}
		for _, rule := range field.Validation {
			if err := validateValidationRule(rule); err != nil {
				return err
			}
		}
	}
	if primaryCount != 1 {
		return fmt.Errorf("generator requires exactly one primary field")
	}
	for _, index := range in.Indexes {
		if err := validateIndexSpec(index, fieldNames); err != nil {
			return err
		}
	}
	if in.Permissions.Resource == "" || in.Permissions.ReadKey == "" || in.Permissions.CreateKey == "" || in.Permissions.UpdateKey == "" || in.Permissions.DeleteKey == "" || in.Permissions.ManageKey == "" {
		return fmt.Errorf("generator permission spec is incomplete")
	}
	if err := validatePermissionSpec(in.Permissions); err != nil {
		return err
	}
	if in.Menu.Key == "" || in.Menu.Path == "" || in.Menu.Component == "" {
		return fmt.Errorf("generator menu spec is incomplete")
	}
	if err := validateMenuSpec(in.Menu, in.Permissions); err != nil {
		return err
	}
	if in.Page.Title == "" || in.Page.RouteName == "" {
		return fmt.Errorf("generator page spec is incomplete")
	}
	if err := validatePageSpec(in.Page, fieldNames); err != nil {
		return err
	}
	if in.Audit.Resource == "" || len(in.Audit.Actions) == 0 {
		return fmt.Errorf("generator audit spec is incomplete")
	}
	if err := validateAuditSpec(in.Audit); err != nil {
		return err
	}
	if err := validatePluginSpec(in.Plugin); err != nil {
		return err
	}
	if err := validatePluginDataOwnership(in.Plugin, in.Table, in.Fields, in.Indexes, in.Permissions, in.Menu); err != nil {
		return err
	}
	if err := validateDocumentSpec(in.Document, in.Plugin, fieldNames); err != nil {
		return err
	}
	if in.CreatedAt.IsZero() || in.UpdatedAt.IsZero() {
		return fmt.Errorf("generator spec timestamps are required")
	}
	if in.UpdatedAt.Before(in.CreatedAt) {
		return fmt.Errorf("generator spec updated time must not be before created time")
	}
	return nil
}

func validateDocumentSpec(spec *DocumentSpec, plugin PluginSpec, fields map[string]struct{}) error {
	if spec == nil {
		return nil
	}
	if !plugin.Enabled {
		return fmt.Errorf("generator document target requires a plugin")
	}
	if !generatorNamePattern.MatchString(spec.SchemaKey) {
		return fmt.Errorf("generator document schema key must match %s", generatorNamePattern.String())
	}
	if spec.SchemaName == "" || !generatorKeyPattern.MatchString(spec.DefinitionID) {
		return fmt.Errorf("generator document schema name and definition id are required")
	}
	if spec.TitleField == "" {
		return fmt.Errorf("generator document title field is required")
	}
	if _, ok := fields[spec.TitleField]; !ok {
		return fmt.Errorf("generator document title field is unknown: %s", spec.TitleField)
	}
	if spec.NumberPrefix == "" || len(spec.NumberPrefix) > 24 || strings.ContainsAny(spec.NumberPrefix, " \t\r\n") {
		return fmt.Errorf("generator document number prefix is invalid")
	}
	return nil
}

func validateGeneratorName(value, label string) error {
	if !generatorNamePattern.MatchString(value) {
		return fmt.Errorf("generator %s must match %s", label, generatorNamePattern.String())
	}
	return nil
}

func validateFieldType(fieldType FieldType) error {
	switch fieldType {
	case FieldTypeString, FieldTypeText, FieldTypeInt, FieldTypeDecimal, FieldTypeBool, FieldTypeTime, FieldTypeJSON, FieldTypeID:
		return nil
	default:
		return fmt.Errorf("generator field type is invalid: %s", fieldType)
	}
}

func validateValidationRule(rule ValidationRule) error {
	switch rule.Type {
	case ValidationRequired:
		return nil
	case ValidationMin, ValidationMax, ValidationPattern, ValidationEnum:
		if strings.TrimSpace(rule.Value) == "" {
			return fmt.Errorf("generator validation value is required for %s", rule.Type)
		}
		return nil
	default:
		return fmt.Errorf("generator validation rule type is invalid: %s", rule.Type)
	}
}

func validateIndexSpec(index IndexSpec, fields map[string]struct{}) error {
	if index.Name == "" {
		return fmt.Errorf("generator index name is required")
	}
	if err := validateGeneratorName(index.Name, "index name"); err != nil {
		return err
	}
	if len(index.Fields) == 0 {
		return fmt.Errorf("generator index fields are required")
	}
	for _, field := range index.Fields {
		if _, ok := fields[field]; !ok {
			return fmt.Errorf("generator index %s references unknown field: %s", index.Name, field)
		}
	}
	return nil
}

func validatePermissionSpec(spec PermissionSpec) error {
	keys := []string{spec.Resource, spec.ReadKey, spec.CreateKey, spec.UpdateKey, spec.DeleteKey, spec.ManageKey}
	for _, key := range keys {
		if !generatorPermKeyPattern.MatchString(key) {
			return fmt.Errorf("generator permission key must match %s", generatorPermKeyPattern.String())
		}
	}
	return nil
}

func validateMenuSpec(spec MenuSpec, permissions PermissionSpec) error {
	if !generatorKeyPattern.MatchString(spec.Key) {
		return fmt.Errorf("generator menu key must match %s", generatorKeyPattern.String())
	}
	if spec.ParentKey != "" && !generatorKeyPattern.MatchString(spec.ParentKey) {
		return fmt.Errorf("generator menu parent key must match %s", generatorKeyPattern.String())
	}
	if !strings.HasPrefix(spec.Path, "/") || strings.ContainsAny(spec.Path, " \t\r\n") {
		return fmt.Errorf("generator menu path must start with / and must not contain whitespace")
	}
	if spec.Order < 0 {
		return fmt.Errorf("generator menu order must be non-negative")
	}
	permissionSet := map[string]struct{}{
		permissions.ReadKey:   {},
		permissions.CreateKey: {},
		permissions.UpdateKey: {},
		permissions.DeleteKey: {},
		permissions.ManageKey: {},
	}
	for _, key := range spec.RequiredPermissions {
		if _, ok := permissionSet[key]; !ok {
			return fmt.Errorf("generator menu references unknown permission: %s", key)
		}
	}
	return nil
}

func validatePageSpec(spec PageSpec, fields map[string]struct{}) error {
	if !generatorKeyPattern.MatchString(spec.RouteName) {
		return fmt.Errorf("generator page route name must match %s", generatorKeyPattern.String())
	}
	for _, field := range append(append([]string{}, spec.List.Columns...), append(spec.List.Filters, spec.Form.Fields...)...) {
		if _, ok := fields[field]; !ok {
			return fmt.Errorf("generator page references unknown field: %s", field)
		}
	}
	return nil
}

func validateAuditSpec(spec AuditSpec) error {
	if !generatorPermKeyPattern.MatchString(spec.Resource) {
		return fmt.Errorf("generator audit resource must match %s", generatorPermKeyPattern.String())
	}
	for _, action := range spec.Actions {
		if err := validateGeneratorName(action, "audit action"); err != nil {
			return err
		}
	}
	return nil
}

func validatePluginSpec(spec PluginSpec) error {
	if !spec.Enabled {
		return nil
	}
	if !generatorPluginIDPattern.MatchString(spec.ID) {
		return fmt.Errorf("generator plugin id must match %s", generatorPluginIDPattern.String())
	}
	if spec.Name == "" || spec.Version == "" || spec.DataNamespace == "" || spec.FrontendEntry == "" {
		return fmt.Errorf("generator plugin spec is incomplete")
	}
	if err := validateGeneratorName(spec.DataNamespace, "plugin data namespace"); err != nil {
		return err
	}
	if !strings.HasPrefix(spec.FrontendEntry, "/") {
		return fmt.Errorf("generator plugin frontend entry must start with /")
	}
	switch spec.UninstallPolicy {
	case "retain", "drop", "archive":
	default:
		return fmt.Errorf("generator plugin uninstall policy is invalid: %s", spec.UninstallPolicy)
	}
	switch spec.RollbackPolicy {
	case "manual", "automatic", "none":
	default:
		return fmt.Errorf("generator plugin rollback policy is invalid: %s", spec.RollbackPolicy)
	}
	switch spec.UIMode {
	case "backend_only", "frontend_only", "monolith", "separated":
	default:
		return fmt.Errorf("generator plugin ui mode is invalid: %s", spec.UIMode)
	}
	if (spec.ServiceBaseURL == "") != (spec.ServiceHealthURL == "") {
		return fmt.Errorf("generator plugin service base and health URLs must be declared together")
	}
	for _, serviceURL := range []struct {
		label string
		value string
	}{
		{label: "base", value: spec.ServiceBaseURL},
		{label: "health", value: spec.ServiceHealthURL},
	} {
		label, value := serviceURL.label, serviceURL.value
		if value == "" {
			continue
		}
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("generator plugin service %s URL is invalid", label)
		}
	}
	publications := map[string]struct{}{}
	for _, publication := range spec.EventPublications {
		if !generatorKeyPattern.MatchString(publication.Name) {
			return fmt.Errorf("generator plugin event publication name is invalid: %s", publication.Name)
		}
		if publication.SchemaVersion == 0 {
			return fmt.Errorf("generator plugin event publication schema version must be positive")
		}
		if !generatorKeyPattern.MatchString(publication.PayloadType) {
			return fmt.Errorf("generator plugin event publication payload type is invalid: %s", publication.PayloadType)
		}
		if publication.Scope != "tenant" && publication.Scope != "global" {
			return fmt.Errorf("generator plugin event publication scope is invalid: %s", publication.Scope)
		}
		key := fmt.Sprintf("%s::%d", publication.Name, publication.SchemaVersion)
		if _, ok := publications[key]; ok {
			return fmt.Errorf("generator plugin event publication conflict: %s", key)
		}
		publications[key] = struct{}{}
	}
	seen := map[string]struct{}{}
	for _, subscription := range spec.EventSubscriptions {
		if !generatorPluginIDPattern.MatchString(subscription.Publisher) {
			return fmt.Errorf("generator plugin event publisher is invalid: %s", subscription.Publisher)
		}
		if !generatorKeyPattern.MatchString(subscription.Name) {
			return fmt.Errorf("generator plugin event name is invalid: %s", subscription.Name)
		}
		if len(subscription.SchemaVersions) == 0 || len(subscription.SchemaVersions) > 16 {
			return fmt.Errorf("generator plugin event schema versions are required and bounded")
		}
		versions := make(map[uint32]struct{}, len(subscription.SchemaVersions))
		for _, version := range subscription.SchemaVersions {
			if version == 0 {
				return fmt.Errorf("generator plugin event schema version must be positive")
			}
			if _, ok := versions[version]; ok {
				return fmt.Errorf("generator plugin event schema version is duplicated: %d", version)
			}
			versions[version] = struct{}{}
		}
		if strings.TrimSpace(subscription.Handler) == "" {
			return fmt.Errorf("generator plugin event handler is required")
		}
		key := subscription.Publisher + "::" + subscription.Name + "::" + subscription.Handler
		if _, ok := seen[key]; ok {
			return fmt.Errorf("generator plugin event subscription conflict: %s", key)
		}
		seen[key] = struct{}{}
		switch subscription.RetryPolicy {
		case "none", "standard", "aggressive":
		default:
			return fmt.Errorf("generator plugin retry policy is invalid: %s", subscription.RetryPolicy)
		}
	}
	return nil
}

func validatePluginDataOwnership(plugin PluginSpec, table TableSpec, fields []FieldSpec, indexes []IndexSpec, permissions PermissionSpec, menu MenuSpec) error {
	if !plugin.Enabled {
		return nil
	}
	if len(table.Name) > 42 || !generatorNamePattern.MatchString(table.Name) ||
		strings.HasPrefix(table.Name, plugin.DataNamespace+"_") {
		return fmt.Errorf("generator plugin logical table is invalid: %s", table.Name)
	}
	reservedFields := map[string]struct{}{
		"tenant_id": {}, "organization_id": {}, "owner_id": {}, "version": {}, "created_at": {}, "updated_at": {},
	}
	for _, field := range fields {
		if field.Name != field.ColumnName {
			return fmt.Errorf("generator plugin field %s must use the same logical and storage name", field.Name)
		}
		if _, reserved := reservedFields[field.Name]; reserved {
			return fmt.Errorf("generator plugin field is reserved by the host: %s", field.Name)
		}
	}
	for _, index := range indexes {
		if !strings.HasPrefix(index.Name, "idx_"+plugin.DataNamespace+"_") {
			return fmt.Errorf("generator plugin index %s must use data namespace prefix idx_%s_", index.Name, plugin.DataNamespace)
		}
	}
	permissionPrefix := plugin.DataNamespace + "."
	for _, key := range []string{permissions.Resource, permissions.ReadKey, permissions.CreateKey, permissions.UpdateKey, permissions.DeleteKey, permissions.ManageKey} {
		if !strings.HasPrefix(key, permissionPrefix) {
			return fmt.Errorf("generator plugin permission %s must use data namespace prefix %s", key, permissionPrefix)
		}
	}
	if !strings.HasPrefix(menu.Key, permissionPrefix) {
		return fmt.Errorf("generator plugin menu %s must use data namespace prefix %s", menu.Key, permissionPrefix)
	}
	return nil
}
