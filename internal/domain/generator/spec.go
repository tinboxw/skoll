package generator

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
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
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = in.CreatedAt
	}
	return in
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
	if in.Table.Name == "" || in.Table.DomainName == "" || in.Table.CollectionName == "" {
		return fmt.Errorf("generator table spec is incomplete")
	}
	if len(in.Fields) == 0 {
		return fmt.Errorf("generator fields are required")
	}
	for _, field := range in.Fields {
		if field.Name == "" || field.ColumnName == "" || field.Label == "" || field.Type == "" {
			return fmt.Errorf("generator field spec is incomplete")
		}
	}
	if in.Permissions.Resource == "" || in.Permissions.ReadKey == "" {
		return fmt.Errorf("generator permission spec is incomplete")
	}
	if in.Menu.Key == "" || in.Menu.Path == "" || in.Menu.Component == "" {
		return fmt.Errorf("generator menu spec is incomplete")
	}
	if in.Page.Title == "" || in.Page.RouteName == "" {
		return fmt.Errorf("generator page spec is incomplete")
	}
	if in.Audit.Resource == "" || len(in.Audit.Actions) == 0 {
		return fmt.Errorf("generator audit spec is incomplete")
	}
	if in.CreatedAt.IsZero() || in.UpdatedAt.IsZero() {
		return fmt.Errorf("generator spec timestamps are required")
	}
	if in.UpdatedAt.Before(in.CreatedAt) {
		return fmt.Errorf("generator spec updated time must not be before created time")
	}
	return nil
}
