package form

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

var formKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)

type FieldType string

const (
	FieldString      FieldType = "string"
	FieldTextarea    FieldType = "textarea"
	FieldNumber      FieldType = "number"
	FieldBoolean     FieldType = "boolean"
	FieldDate        FieldType = "date"
	FieldDateTime    FieldType = "datetime"
	FieldSelect      FieldType = "select"
	FieldMultiSelect FieldType = "multi_select"
	FieldDictionary  FieldType = "dictionary"
	FieldUser        FieldType = "user"
	FieldDepartment  FieldType = "department"
	FieldAttachment  FieldType = "attachment"
	FieldDetailTable FieldType = "detail_table"
)

type ValidationRuleType string

const (
	ValidationRequired  ValidationRuleType = "required"
	ValidationMin       ValidationRuleType = "min"
	ValidationMax       ValidationRuleType = "max"
	ValidationMinLength ValidationRuleType = "min_length"
	ValidationMaxLength ValidationRuleType = "max_length"
	ValidationPattern   ValidationRuleType = "pattern"
	ValidationEnum      ValidationRuleType = "enum"
	ValidationMinItems  ValidationRuleType = "min_items"
	ValidationMaxItems  ValidationRuleType = "max_items"
)

type Schema struct {
	ID           shared.ID
	Key          string
	Name         string
	Version      int
	BusinessType string
	Description  string
	Fields       []Field
	Meta         shared.AuditMeta
}

type Field struct {
	Key          string
	Label        string
	Type         FieldType
	Required     bool
	Placeholder  string
	Help         string
	Default      string
	Dictionary   string
	Options      []Option
	Validation   []ValidationRule
	Attachment   *AttachmentConfig
	DetailTable  *DetailTableConfig
	DisplayOrder int
}

type Option struct {
	Label string
	Value string
}

type ValidationRule struct {
	Type    ValidationRuleType
	Value   string
	Message string
}

type AttachmentConfig struct {
	MaxFiles  int
	MaxSizeMB int
	Accept    []string
	Required  bool
}

type DetailTableConfig struct {
	MinRows int
	MaxRows int
	Columns []Field
}

type SchemaInput struct {
	ID           shared.ID
	Key          string
	Name         string
	Version      int
	BusinessType string
	Description  string
	Fields       []Field
	Now          time.Time
}

func NewSchema(in SchemaInput) (*Schema, error) {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	schema := &Schema{
		ID:           in.ID,
		Key:          strings.TrimSpace(strings.ToLower(in.Key)),
		Name:         strings.TrimSpace(in.Name),
		Version:      in.Version,
		BusinessType: strings.TrimSpace(strings.ToLower(in.BusinessType)),
		Description:  strings.TrimSpace(in.Description),
		Fields:       normalizeFields(in.Fields),
	}
	if err := schema.Validate(); err != nil {
		return nil, err
	}
	schema.Meta.Touch(now)
	return schema, nil
}

func (s Schema) Validate() error {
	if s.ID.IsZero() {
		return fmt.Errorf("form schema id is required")
	}
	if !formKeyPattern.MatchString(strings.TrimSpace(s.Key)) {
		return fmt.Errorf("form schema key is invalid")
	}
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("form schema name is required")
	}
	if s.Version <= 0 {
		return fmt.Errorf("form schema version must be positive")
	}
	if !formKeyPattern.MatchString(strings.TrimSpace(s.BusinessType)) {
		return fmt.Errorf("form schema business type is invalid")
	}
	if len(s.Fields) == 0 {
		return fmt.Errorf("form schema fields are required")
	}
	seen := map[string]struct{}{}
	for _, field := range s.Fields {
		if err := validateField(field, false); err != nil {
			return err
		}
		if _, ok := seen[field.Key]; ok {
			return fmt.Errorf("form field key conflict: %s", field.Key)
		}
		seen[field.Key] = struct{}{}
	}
	return nil
}

func (s Schema) FieldByKey(key string) (Field, bool) {
	target := strings.TrimSpace(strings.ToLower(key))
	for _, field := range s.Fields {
		if field.Key == target {
			return field, true
		}
	}
	return Field{}, false
}

func normalizeFields(fields []Field) []Field {
	out := make([]Field, 0, len(fields))
	for _, field := range fields {
		field.Key = strings.TrimSpace(strings.ToLower(field.Key))
		field.Label = strings.TrimSpace(field.Label)
		field.Type = FieldType(strings.TrimSpace(strings.ToLower(string(field.Type))))
		field.Placeholder = strings.TrimSpace(field.Placeholder)
		field.Help = strings.TrimSpace(field.Help)
		field.Default = strings.TrimSpace(field.Default)
		field.Dictionary = strings.TrimSpace(strings.ToLower(field.Dictionary))
		field.Options = normalizeOptions(field.Options)
		field.Validation = normalizeValidation(field.Validation)
		if field.Attachment != nil {
			cfg := *field.Attachment
			cfg.Accept = normalizeStringList(cfg.Accept)
			field.Attachment = &cfg
		}
		if field.DetailTable != nil {
			cfg := *field.DetailTable
			cfg.Columns = normalizeFields(cfg.Columns)
			field.DetailTable = &cfg
		}
		out = append(out, field)
	}
	return out
}

func normalizeOptions(options []Option) []Option {
	out := make([]Option, 0, len(options))
	for _, option := range options {
		option.Label = strings.TrimSpace(option.Label)
		option.Value = strings.TrimSpace(option.Value)
		out = append(out, option)
	}
	return out
}

func normalizeValidation(rules []ValidationRule) []ValidationRule {
	out := make([]ValidationRule, 0, len(rules))
	for _, rule := range rules {
		rule.Type = ValidationRuleType(strings.TrimSpace(strings.ToLower(string(rule.Type))))
		rule.Value = strings.TrimSpace(rule.Value)
		rule.Message = strings.TrimSpace(rule.Message)
		out = append(out, rule)
	}
	return out
}

func normalizeStringList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func validateField(field Field, detailColumn bool) error {
	if !formKeyPattern.MatchString(strings.TrimSpace(field.Key)) {
		return fmt.Errorf("form field key is invalid")
	}
	if strings.TrimSpace(field.Label) == "" {
		return fmt.Errorf("form field label is required")
	}
	switch field.Type {
	case FieldString, FieldTextarea, FieldNumber, FieldBoolean, FieldDate, FieldDateTime, FieldUser, FieldDepartment:
	case FieldSelect, FieldMultiSelect:
		if len(field.Options) == 0 {
			return fmt.Errorf("form select field requires options: %s", field.Key)
		}
	case FieldDictionary:
		if !formKeyPattern.MatchString(strings.TrimSpace(field.Dictionary)) {
			return fmt.Errorf("form dictionary field requires dictionary code: %s", field.Key)
		}
	case FieldAttachment:
		if err := validateAttachment(field); err != nil {
			return err
		}
	case FieldDetailTable:
		if detailColumn {
			return fmt.Errorf("form detail table cannot be nested: %s", field.Key)
		}
		if err := validateDetailTable(field); err != nil {
			return err
		}
	default:
		return fmt.Errorf("form field type is invalid: %s", field.Key)
	}
	if err := validateOptions(field); err != nil {
		return err
	}
	for _, rule := range field.Validation {
		if err := validateRule(field, rule); err != nil {
			return err
		}
	}
	return nil
}

func validateOptions(field Field) error {
	seen := map[string]struct{}{}
	for _, option := range field.Options {
		if strings.TrimSpace(option.Value) == "" {
			return fmt.Errorf("form option value is required: %s", field.Key)
		}
		if _, ok := seen[option.Value]; ok {
			return fmt.Errorf("form option value conflict: %s", field.Key)
		}
		seen[option.Value] = struct{}{}
	}
	return nil
}

func validateRule(field Field, rule ValidationRule) error {
	switch rule.Type {
	case ValidationRequired, ValidationMin, ValidationMax, ValidationMinLength, ValidationMaxLength, ValidationPattern, ValidationEnum, ValidationMinItems, ValidationMaxItems:
	default:
		return fmt.Errorf("form validation rule is invalid: %s", field.Key)
	}
	if rule.Type == ValidationPattern {
		if _, err := regexp.Compile(rule.Value); err != nil {
			return fmt.Errorf("form validation pattern is invalid: %s", field.Key)
		}
	}
	if rule.Type == ValidationRequired && rule.Value == "" {
		return nil
	}
	if rule.Type != ValidationRequired && strings.TrimSpace(rule.Value) == "" {
		return fmt.Errorf("form validation rule value is required: %s", field.Key)
	}
	return nil
}

func validateAttachment(field Field) error {
	if field.Attachment == nil {
		return fmt.Errorf("form attachment field requires config: %s", field.Key)
	}
	if field.Attachment.MaxFiles <= 0 {
		return fmt.Errorf("form attachment max files must be positive: %s", field.Key)
	}
	if field.Attachment.MaxSizeMB <= 0 {
		return fmt.Errorf("form attachment max size must be positive: %s", field.Key)
	}
	return nil
}

func validateDetailTable(field Field) error {
	if field.DetailTable == nil {
		return fmt.Errorf("form detail table requires config: %s", field.Key)
	}
	if len(field.DetailTable.Columns) == 0 {
		return fmt.Errorf("form detail table requires columns: %s", field.Key)
	}
	if field.DetailTable.MinRows < 0 || field.DetailTable.MaxRows < 0 {
		return fmt.Errorf("form detail table row limits must be non-negative: %s", field.Key)
	}
	if field.DetailTable.MaxRows > 0 && field.DetailTable.MinRows > field.DetailTable.MaxRows {
		return fmt.Errorf("form detail table row limits are invalid: %s", field.Key)
	}
	seen := map[string]struct{}{}
	for _, column := range field.DetailTable.Columns {
		if err := validateField(column, true); err != nil {
			return err
		}
		if _, ok := seen[column.Key]; ok {
			return fmt.Errorf("form detail table column key conflict: %s", column.Key)
		}
		seen[column.Key] = struct{}{}
	}
	return nil
}
