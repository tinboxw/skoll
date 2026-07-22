package pluginsdk

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxDocumentHeaderFields     = 128
	MaxDocumentLineSchemas      = 16
	MaxDocumentLineFields       = 64
	MaxDocumentLinesPerSchema   = 1000
	MaxDocumentValidationRules  = 16
	MaxDocumentStates           = 32
	MaxDocumentActions          = 64
	MaxDocumentValueBytes       = 1 << 20
	MaxDocumentTags             = 32
	MaxDocumentStringCharacters = 65535
)

var (
	documentKeyPattern      = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	documentIDPattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	documentDecimalPattern  = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
	documentCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
	documentUnitPattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._/%-]{0,31}$`)
)

type DocumentFieldType string

const (
	DocumentFieldString    DocumentFieldType = "string"
	DocumentFieldText      DocumentFieldType = "text"
	DocumentFieldInteger   DocumentFieldType = "integer"
	DocumentFieldDecimal   DocumentFieldType = "decimal"
	DocumentFieldMoney     DocumentFieldType = "money"
	DocumentFieldQuantity  DocumentFieldType = "quantity"
	DocumentFieldBoolean   DocumentFieldType = "boolean"
	DocumentFieldDate      DocumentFieldType = "date"
	DocumentFieldDateTime  DocumentFieldType = "datetime"
	DocumentFieldReference DocumentFieldType = "reference"
	DocumentFieldJSON      DocumentFieldType = "json"
)

type DocumentValidationRuleType string

const (
	DocumentValidationMin       DocumentValidationRuleType = "min"
	DocumentValidationMax       DocumentValidationRuleType = "max"
	DocumentValidationMinLength DocumentValidationRuleType = "min_length"
	DocumentValidationMaxLength DocumentValidationRuleType = "max_length"
	DocumentValidationPattern   DocumentValidationRuleType = "pattern"
)

type DocumentSchema struct {
	Key          string                 `json:"key"`
	Name         string                 `json:"name"`
	Version      int                    `json:"version"`
	InitialState string                 `json:"initialState"`
	Header       []DocumentFieldSchema  `json:"header"`
	Lines        []DocumentLineSchema   `json:"lines"`
	States       []DocumentStateSchema  `json:"states"`
	Actions      []DocumentActionSchema `json:"actions"`
}

type DocumentFieldSchema struct {
	Key           string                   `json:"key"`
	Label         string                   `json:"label"`
	Type          DocumentFieldType        `json:"type"`
	Required      bool                     `json:"required"`
	ReferenceType string                   `json:"referenceType,omitempty"`
	Rules         []DocumentValidationRule `json:"rules,omitempty"`
}

type DocumentValidationRule struct {
	Type    DocumentValidationRuleType `json:"type"`
	Value   string                     `json:"value"`
	Message string                     `json:"message,omitempty"`
}

type DocumentLineSchema struct {
	Key      string                `json:"key"`
	Name     string                `json:"name"`
	MinItems int                   `json:"minItems"`
	MaxItems int                   `json:"maxItems"`
	Fields   []DocumentFieldSchema `json:"fields"`
}

type DocumentStateSchema struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Terminal bool   `json:"terminal"`
}

type DocumentActionSchema struct {
	Key             string   `json:"key"`
	Name            string   `json:"name"`
	From            []string `json:"from"`
	To              string   `json:"to"`
	RequiresComment bool     `json:"requiresComment"`
}

type DocumentReference struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

// DocumentValue keeps business values explicit and lossless on the plugin wire.
type DocumentValue struct {
	Type      DocumentFieldType  `json:"type"`
	Value     string             `json:"value,omitempty"`
	Currency  string             `json:"currency,omitempty"`
	Unit      string             `json:"unit,omitempty"`
	Reference *DocumentReference `json:"reference,omitempty"`
}

type DocumentLine struct {
	ID     string                   `json:"id"`
	Values map[string]DocumentValue `json:"values"`
}

type DocumentMetadata struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedBy string    `json:"updatedBy"`
	Tags      []string  `json:"tags,omitempty"`
}

type DocumentRecord struct {
	ID            string                    `json:"id"`
	Type          string                    `json:"type"`
	SchemaVersion int                       `json:"schemaVersion"`
	Number        string                    `json:"number"`
	Title         string                    `json:"title"`
	State         string                    `json:"state"`
	Version       int64                     `json:"version"`
	Header        map[string]DocumentValue  `json:"header"`
	Lines         map[string][]DocumentLine `json:"lines"`
	Metadata      DocumentMetadata          `json:"metadata"`
}

type DocumentActionInput struct {
	DocumentID      string `json:"documentId"`
	Action          string `json:"action"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Comment         string `json:"comment,omitempty"`
}

type DocumentContractError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *DocumentContractError) Error() string {
	if e == nil {
		return "document contract is invalid"
	}
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

func (s DocumentSchema) Validate() error {
	if err := validateDocumentKey("key", s.Key); err != nil {
		return err
	}
	if err := validateDocumentLabel("name", s.Name); err != nil {
		return err
	}
	if s.Version <= 0 {
		return invalidDocument("version", "schema version must be positive")
	}
	if len(s.Header) == 0 || len(s.Header) > MaxDocumentHeaderFields {
		return invalidDocument("header", "header fields are outside the supported range")
	}
	if len(s.Lines) > MaxDocumentLineSchemas {
		return invalidDocument("lines", "line schemas exceed the supported limit")
	}
	if len(s.States) == 0 || len(s.States) > MaxDocumentStates {
		return invalidDocument("states", "states are outside the supported range")
	}
	if len(s.Actions) > MaxDocumentActions {
		return invalidDocument("actions", "actions exceed the supported limit")
	}
	if err := validateDocumentFields("header", s.Header, MaxDocumentHeaderFields); err != nil {
		return err
	}
	if err := validateDocumentLines(s.Lines); err != nil {
		return err
	}
	states, err := validateDocumentStates(s.States, s.InitialState)
	if err != nil {
		return err
	}
	return validateDocumentActions(s.Actions, states)
}

func (s DocumentSchema) ValidateRecord(record DocumentRecord) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := validateDocumentID("id", record.ID); err != nil {
		return err
	}
	if record.Type != s.Key {
		return invalidDocument("type", "document type does not match schema")
	}
	if record.SchemaVersion != s.Version {
		return invalidDocument("schemaVersion", "document schema version does not match schema")
	}
	if err := validateDocumentBoundedText("number", record.Number, 128, true); err != nil {
		return err
	}
	if err := validateDocumentBoundedText("title", record.Title, 256, true); err != nil {
		return err
	}
	if _, ok := documentStateByKey(s.States, record.State); !ok {
		return invalidDocument("state", "document state is not declared")
	}
	if record.Version <= 0 {
		return invalidDocument("version", "document version must be positive")
	}
	if err := validateDocumentValues("header", s.Header, record.Header); err != nil {
		return err
	}
	if err := validateDocumentLineValues(s.Lines, record.Lines); err != nil {
		return err
	}
	return record.Metadata.Validate()
}

func (s DocumentSchema) NextState(currentState, action, comment string) (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}
	if _, ok := documentStateByKey(s.States, currentState); !ok {
		return "", invalidDocument("state", "document state is not declared")
	}
	for index, candidate := range s.Actions {
		if candidate.Key != action {
			continue
		}
		for _, from := range candidate.From {
			if from == currentState {
				if candidate.RequiresComment && strings.TrimSpace(comment) == "" {
					return "", invalidDocument(fmt.Sprintf("actions[%d].comment", index), "action comment is required")
				}
				return candidate.To, nil
			}
		}
		return "", invalidDocument("action", "action is not available from the current state")
	}
	return "", invalidDocument("action", "document action is not declared")
}

func (v DocumentValue) Validate() error {
	return validateDocumentValue("value", DocumentFieldSchema{Type: v.Type}, v)
}

func (m DocumentMetadata) Validate() error {
	if m.CreatedAt.IsZero() || m.UpdatedAt.IsZero() {
		return invalidDocument("metadata", "creation and update timestamps are required")
	}
	if _, offset := m.CreatedAt.Zone(); offset != 0 {
		return invalidDocument("metadata.createdAt", "timestamp must use UTC")
	}
	if _, offset := m.UpdatedAt.Zone(); offset != 0 {
		return invalidDocument("metadata.updatedAt", "timestamp must use UTC")
	}
	if m.UpdatedAt.Before(m.CreatedAt) {
		return invalidDocument("metadata.updatedAt", "update timestamp cannot precede creation")
	}
	if err := validateDocumentID("metadata.createdBy", m.CreatedBy); err != nil {
		return err
	}
	if err := validateDocumentID("metadata.updatedBy", m.UpdatedBy); err != nil {
		return err
	}
	if len(m.Tags) > MaxDocumentTags {
		return invalidDocument("metadata.tags", "tag count exceeds the supported limit")
	}
	seen := make(map[string]struct{}, len(m.Tags))
	for index, tag := range m.Tags {
		path := fmt.Sprintf("metadata.tags[%d]", index)
		if err := validateDocumentBoundedText(path, tag, 64, true); err != nil {
			return err
		}
		if _, exists := seen[tag]; exists {
			return invalidDocument(path, "tag is duplicated")
		}
		seen[tag] = struct{}{}
	}
	return nil
}

func (in DocumentActionInput) Validate() error {
	if err := validateDocumentID("documentId", in.DocumentID); err != nil {
		return err
	}
	if err := validateDocumentKey("action", in.Action); err != nil {
		return err
	}
	if in.ExpectedVersion <= 0 {
		return invalidDocument("expectedVersion", "expected version must be positive")
	}
	return validateDocumentBoundedText("comment", in.Comment, 2000, false)
}

func validateDocumentFields(path string, fields []DocumentFieldSchema, maximum int) error {
	if len(fields) == 0 || len(fields) > maximum {
		return invalidDocument(path, "fields are outside the supported range")
	}
	seen := make(map[string]struct{}, len(fields))
	for index, field := range fields {
		fieldPath := fmt.Sprintf("%s[%d]", path, index)
		if err := validateDocumentKey(fieldPath+".key", field.Key); err != nil {
			return err
		}
		if _, exists := seen[field.Key]; exists {
			return invalidDocument(fieldPath+".key", "field key is duplicated")
		}
		seen[field.Key] = struct{}{}
		if err := validateDocumentLabel(fieldPath+".label", field.Label); err != nil {
			return err
		}
		if !supportedDocumentFieldType(field.Type) {
			return invalidDocument(fieldPath+".type", "field type is unsupported")
		}
		if field.Type == DocumentFieldReference {
			if err := validateDocumentKey(fieldPath+".referenceType", field.ReferenceType); err != nil {
				return err
			}
		} else if field.ReferenceType != "" {
			return invalidDocument(fieldPath+".referenceType", "reference type is only valid for reference fields")
		}
		if err := validateDocumentRules(fieldPath, field); err != nil {
			return err
		}
	}
	return nil
}

func validateDocumentRules(path string, field DocumentFieldSchema) error {
	if len(field.Rules) > MaxDocumentValidationRules {
		return invalidDocument(path+".rules", "validation rules exceed the supported limit")
	}
	seen := make(map[DocumentValidationRuleType]struct{}, len(field.Rules))
	var minimum, maximum *big.Rat
	var minimumLength, maximumLength *int64
	for index, rule := range field.Rules {
		rulePath := fmt.Sprintf("%s.rules[%d]", path, index)
		if _, exists := seen[rule.Type]; exists {
			return invalidDocument(rulePath+".type", "validation rule is duplicated")
		}
		seen[rule.Type] = struct{}{}
		if err := validateDocumentBoundedText(rulePath+".message", rule.Message, 256, false); err != nil {
			return err
		}
		switch rule.Type {
		case DocumentValidationMin, DocumentValidationMax:
			if !numericDocumentField(field.Type) {
				return invalidDocument(rulePath+".type", "numeric rule requires a numeric field")
			}
			value, ok := canonicalDocumentNumber(rule.Value)
			if !ok {
				return invalidDocument(rulePath+".value", "numeric rule requires a canonical decimal")
			}
			if rule.Type == DocumentValidationMin {
				minimum = value
			} else {
				maximum = value
			}
		case DocumentValidationMinLength, DocumentValidationMaxLength:
			if field.Type != DocumentFieldString && field.Type != DocumentFieldText {
				return invalidDocument(rulePath+".type", "length rule requires a string or text field")
			}
			value, err := strconv.ParseInt(rule.Value, 10, 32)
			if err != nil || value < 0 || strconv.FormatInt(value, 10) != rule.Value {
				return invalidDocument(rulePath+".value", "length rule requires a canonical non-negative integer")
			}
			if rule.Type == DocumentValidationMinLength {
				minimumLength = &value
			} else {
				maximumLength = &value
			}
		case DocumentValidationPattern:
			if field.Type != DocumentFieldString && field.Type != DocumentFieldText {
				return invalidDocument(rulePath+".type", "pattern rule requires a string or text field")
			}
			if len(rule.Value) == 0 || len(rule.Value) > 512 {
				return invalidDocument(rulePath+".value", "pattern is outside the supported range")
			}
			if _, err := regexp.Compile(rule.Value); err != nil {
				return invalidDocument(rulePath+".value", "pattern is invalid")
			}
		default:
			return invalidDocument(rulePath+".type", "validation rule is unsupported")
		}
	}
	if minimum != nil && maximum != nil && minimum.Cmp(maximum) > 0 {
		return invalidDocument(path+".rules", "minimum cannot exceed maximum")
	}
	if minimumLength != nil && maximumLength != nil && *minimumLength > *maximumLength {
		return invalidDocument(path+".rules", "minimum length cannot exceed maximum length")
	}
	return nil
}

func validateDocumentLines(lines []DocumentLineSchema) error {
	seen := make(map[string]struct{}, len(lines))
	for index, line := range lines {
		path := fmt.Sprintf("lines[%d]", index)
		if err := validateDocumentKey(path+".key", line.Key); err != nil {
			return err
		}
		if _, exists := seen[line.Key]; exists {
			return invalidDocument(path+".key", "line schema key is duplicated")
		}
		seen[line.Key] = struct{}{}
		if err := validateDocumentLabel(path+".name", line.Name); err != nil {
			return err
		}
		if line.MinItems < 0 || line.MaxItems < 1 || line.MinItems > line.MaxItems || line.MaxItems > MaxDocumentLinesPerSchema {
			return invalidDocument(path, "line item limits are invalid")
		}
		if err := validateDocumentFields(path+".fields", line.Fields, MaxDocumentLineFields); err != nil {
			return err
		}
	}
	return nil
}

func validateDocumentStates(states []DocumentStateSchema, initial string) (map[string]DocumentStateSchema, error) {
	byKey := make(map[string]DocumentStateSchema, len(states))
	for index, state := range states {
		path := fmt.Sprintf("states[%d]", index)
		if err := validateDocumentKey(path+".key", state.Key); err != nil {
			return nil, err
		}
		if _, exists := byKey[state.Key]; exists {
			return nil, invalidDocument(path+".key", "state key is duplicated")
		}
		if err := validateDocumentLabel(path+".name", state.Name); err != nil {
			return nil, err
		}
		byKey[state.Key] = state
	}
	initialState, exists := byKey[initial]
	if !exists {
		return nil, invalidDocument("initialState", "initial state is not declared")
	}
	if initialState.Terminal {
		return nil, invalidDocument("initialState", "initial state cannot be terminal")
	}
	return byKey, nil
}

func validateDocumentActions(actions []DocumentActionSchema, states map[string]DocumentStateSchema) error {
	seenActions := make(map[string]struct{}, len(actions))
	sources := make(map[string]struct{})
	for index, action := range actions {
		path := fmt.Sprintf("actions[%d]", index)
		if err := validateDocumentKey(path+".key", action.Key); err != nil {
			return err
		}
		if _, exists := seenActions[action.Key]; exists {
			return invalidDocument(path+".key", "action key is duplicated")
		}
		seenActions[action.Key] = struct{}{}
		if err := validateDocumentLabel(path+".name", action.Name); err != nil {
			return err
		}
		if len(action.From) == 0 || len(action.From) > len(states) {
			return invalidDocument(path+".from", "action sources are outside the supported range")
		}
		if _, exists := states[action.To]; !exists {
			return invalidDocument(path+".to", "action target state is not declared")
		}
		seenFrom := make(map[string]struct{}, len(action.From))
		for sourceIndex, source := range action.From {
			sourcePath := fmt.Sprintf("%s.from[%d]", path, sourceIndex)
			state, exists := states[source]
			if !exists {
				return invalidDocument(sourcePath, "action source state is not declared")
			}
			if state.Terminal {
				return invalidDocument(sourcePath, "terminal state cannot have outgoing actions")
			}
			if source == action.To {
				return invalidDocument(sourcePath, "action must change document state")
			}
			if _, exists := seenFrom[source]; exists {
				return invalidDocument(sourcePath, "action source state is duplicated")
			}
			seenFrom[source] = struct{}{}
			sources[source] = struct{}{}
		}
	}
	for _, state := range sortedDocumentStates(states) {
		if state.Terminal {
			continue
		}
		if _, exists := sources[state.Key]; !exists {
			return invalidDocument("states."+state.Key, "non-terminal state requires an outgoing action")
		}
	}
	return nil
}

func validateDocumentValues(path string, fields []DocumentFieldSchema, values map[string]DocumentValue) error {
	declared := make(map[string]DocumentFieldSchema, len(fields))
	for _, field := range fields {
		declared[field.Key] = field
	}
	for _, key := range sortedDocumentValueKeys(values) {
		if _, exists := declared[key]; !exists {
			return invalidDocument(path+"."+key, "field is not declared")
		}
	}
	for _, field := range fields {
		value, exists := values[field.Key]
		if !exists {
			if field.Required {
				return invalidDocument(path+"."+field.Key, "required field is missing")
			}
			continue
		}
		if err := validateDocumentValue(path+"."+field.Key, field, value); err != nil {
			return err
		}
	}
	return nil
}

func validateDocumentLineValues(schemas []DocumentLineSchema, groups map[string][]DocumentLine) error {
	declared := make(map[string]DocumentLineSchema, len(schemas))
	for _, schema := range schemas {
		declared[schema.Key] = schema
	}
	groupKeys := make([]string, 0, len(groups))
	for key := range groups {
		groupKeys = append(groupKeys, key)
	}
	sort.Strings(groupKeys)
	for _, key := range groupKeys {
		if _, exists := declared[key]; !exists {
			return invalidDocument("lines."+key, "line schema is not declared")
		}
	}
	for _, schema := range schemas {
		items := groups[schema.Key]
		if len(items) < schema.MinItems || len(items) > schema.MaxItems {
			return invalidDocument("lines."+schema.Key, "line count is outside the declared range")
		}
		seenIDs := make(map[string]struct{}, len(items))
		for index, item := range items {
			path := fmt.Sprintf("lines.%s[%d]", schema.Key, index)
			if err := validateDocumentID(path+".id", item.ID); err != nil {
				return err
			}
			if _, exists := seenIDs[item.ID]; exists {
				return invalidDocument(path+".id", "line id is duplicated")
			}
			seenIDs[item.ID] = struct{}{}
			if err := validateDocumentValues(path+".values", schema.Fields, item.Values); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDocumentValue(path string, field DocumentFieldSchema, value DocumentValue) error {
	if value.Type != field.Type {
		return invalidDocument(path+".type", "value type does not match field type")
	}
	if !supportedDocumentFieldType(value.Type) {
		return invalidDocument(path+".type", "value type is unsupported")
	}
	if len(value.Value) > MaxDocumentValueBytes {
		return invalidDocument(path+".value", "value exceeds the supported size")
	}
	switch value.Type {
	case DocumentFieldString, DocumentFieldText:
		if value.Currency != "" || value.Unit != "" || value.Reference != nil {
			return invalidDocument(path, "string value contains unrelated attributes")
		}
		if !utf8.ValidString(value.Value) || utf8.RuneCountInString(value.Value) > MaxDocumentStringCharacters {
			return invalidDocument(path+".value", "string value is invalid or too long")
		}
	case DocumentFieldInteger:
		parsed, err := strconv.ParseInt(value.Value, 10, 64)
		if err != nil || strconv.FormatInt(parsed, 10) != value.Value {
			return invalidDocument(path+".value", "integer must be canonical base-10 int64")
		}
		if value.Currency != "" || value.Unit != "" || value.Reference != nil {
			return invalidDocument(path, "integer value contains unrelated attributes")
		}
	case DocumentFieldDecimal:
		if _, ok := canonicalDocumentNumber(value.Value); !ok {
			return invalidDocument(path+".value", "decimal must be a canonical plain base-10 number")
		}
		if value.Currency != "" || value.Unit != "" || value.Reference != nil {
			return invalidDocument(path, "decimal value contains unrelated attributes")
		}
	case DocumentFieldMoney:
		if _, ok := canonicalDocumentNumber(value.Value); !ok {
			return invalidDocument(path+".value", "money amount must be a canonical plain base-10 number")
		}
		if !documentCurrencyPattern.MatchString(value.Currency) {
			return invalidDocument(path+".currency", "money currency must be a three-letter uppercase code")
		}
		if value.Unit != "" || value.Reference != nil {
			return invalidDocument(path, "money value contains unrelated attributes")
		}
	case DocumentFieldQuantity:
		if _, ok := canonicalDocumentNumber(value.Value); !ok {
			return invalidDocument(path+".value", "quantity must be a canonical plain base-10 number")
		}
		if !documentUnitPattern.MatchString(value.Unit) {
			return invalidDocument(path+".unit", "quantity unit is invalid")
		}
		if value.Currency != "" || value.Reference != nil {
			return invalidDocument(path, "quantity value contains unrelated attributes")
		}
	case DocumentFieldBoolean:
		if value.Value != "true" && value.Value != "false" {
			return invalidDocument(path+".value", "boolean must be true or false")
		}
		if value.Currency != "" || value.Unit != "" || value.Reference != nil {
			return invalidDocument(path, "boolean value contains unrelated attributes")
		}
	case DocumentFieldDate:
		parsed, err := time.Parse("2006-01-02", value.Value)
		if err != nil || parsed.Format("2006-01-02") != value.Value {
			return invalidDocument(path+".value", "date must use YYYY-MM-DD")
		}
		if value.Currency != "" || value.Unit != "" || value.Reference != nil {
			return invalidDocument(path, "date value contains unrelated attributes")
		}
	case DocumentFieldDateTime:
		parsed, err := time.Parse(time.RFC3339Nano, value.Value)
		if err != nil || parsed.UTC().Format(time.RFC3339Nano) != value.Value {
			return invalidDocument(path+".value", "datetime must be canonical UTC RFC3339Nano")
		}
		if value.Currency != "" || value.Unit != "" || value.Reference != nil {
			return invalidDocument(path, "datetime value contains unrelated attributes")
		}
	case DocumentFieldReference:
		if value.Value != "" || value.Currency != "" || value.Unit != "" || value.Reference == nil {
			return invalidDocument(path, "reference value must contain only a reference")
		}
		if field.ReferenceType != "" && value.Reference.Type != field.ReferenceType {
			return invalidDocument(path+".reference.type", "reference type does not match field declaration")
		}
		if err := validateDocumentKey(path+".reference.type", value.Reference.Type); err != nil {
			return err
		}
		if err := validateDocumentID(path+".reference.id", value.Reference.ID); err != nil {
			return err
		}
		if err := validateDocumentBoundedText(path+".reference.label", value.Reference.Label, 256, false); err != nil {
			return err
		}
	case DocumentFieldJSON:
		if value.Currency != "" || value.Unit != "" || value.Reference != nil {
			return invalidDocument(path, "json value contains unrelated attributes")
		}
		if value.Value == "" || !json.Valid([]byte(value.Value)) {
			return invalidDocument(path+".value", "json value must contain valid JSON")
		}
	}
	return applyDocumentRules(path, field, value)
}

func applyDocumentRules(path string, field DocumentFieldSchema, value DocumentValue) error {
	for _, rule := range field.Rules {
		message := strings.TrimSpace(rule.Message)
		if message == "" {
			message = "value violates " + string(rule.Type)
		}
		switch rule.Type {
		case DocumentValidationMin, DocumentValidationMax:
			actual, _ := canonicalDocumentNumber(value.Value)
			expected, _ := canonicalDocumentNumber(rule.Value)
			comparison := actual.Cmp(expected)
			if (rule.Type == DocumentValidationMin && comparison < 0) || (rule.Type == DocumentValidationMax && comparison > 0) {
				return invalidDocument(path, message)
			}
		case DocumentValidationMinLength, DocumentValidationMaxLength:
			expected, _ := strconv.ParseInt(rule.Value, 10, 32)
			length := int64(utf8.RuneCountInString(value.Value))
			if (rule.Type == DocumentValidationMinLength && length < expected) || (rule.Type == DocumentValidationMaxLength && length > expected) {
				return invalidDocument(path, message)
			}
		case DocumentValidationPattern:
			matched, _ := regexp.MatchString(rule.Value, value.Value)
			if !matched {
				return invalidDocument(path, message)
			}
		}
	}
	return nil
}

func supportedDocumentFieldType(fieldType DocumentFieldType) bool {
	switch fieldType {
	case DocumentFieldString, DocumentFieldText, DocumentFieldInteger, DocumentFieldDecimal, DocumentFieldMoney,
		DocumentFieldQuantity, DocumentFieldBoolean, DocumentFieldDate, DocumentFieldDateTime, DocumentFieldReference, DocumentFieldJSON:
		return true
	default:
		return false
	}
}

func numericDocumentField(fieldType DocumentFieldType) bool {
	return fieldType == DocumentFieldInteger || fieldType == DocumentFieldDecimal || fieldType == DocumentFieldMoney || fieldType == DocumentFieldQuantity
}

func canonicalDocumentNumber(value string) (*big.Rat, bool) {
	if len(value) == 0 || len(value) > 128 || !documentDecimalPattern.MatchString(value) {
		return nil, false
	}
	number, ok := new(big.Rat).SetString(value)
	return number, ok
}

func documentStateByKey(states []DocumentStateSchema, key string) (DocumentStateSchema, bool) {
	for _, state := range states {
		if state.Key == key {
			return state, true
		}
	}
	return DocumentStateSchema{}, false
}

func sortedDocumentStates(states map[string]DocumentStateSchema) []DocumentStateSchema {
	keys := make([]string, 0, len(states))
	for key := range states {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]DocumentStateSchema, 0, len(keys))
	for _, key := range keys {
		out = append(out, states[key])
	}
	return out
}

func sortedDocumentValueKeys(values map[string]DocumentValue) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validateDocumentKey(path, value string) error {
	if !documentKeyPattern.MatchString(value) {
		return invalidDocument(path, "value must be a lowercase document key")
	}
	return nil
}

func validateDocumentID(path, value string) error {
	if !documentIDPattern.MatchString(value) {
		return invalidDocument(path, "value must be a bounded identifier")
	}
	return nil
}

func validateDocumentLabel(path, value string) error {
	return validateDocumentBoundedText(path, value, 256, true)
}

func validateDocumentBoundedText(path, value string, maximum int, required bool) error {
	if !utf8.ValidString(value) {
		return invalidDocument(path, "value must contain valid UTF-8")
	}
	trimmed := strings.TrimSpace(value)
	if required && trimmed == "" {
		return invalidDocument(path, "value is required")
	}
	if value != "" && trimmed != value {
		return invalidDocument(path, "value must not contain surrounding whitespace")
	}
	if utf8.RuneCountInString(value) > maximum {
		return invalidDocument(path, "value exceeds the supported length")
	}
	return nil
}

func invalidDocument(field, message string) error {
	return &DocumentContractError{Field: field, Message: message}
}
