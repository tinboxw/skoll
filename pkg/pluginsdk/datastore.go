package pluginsdk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	MaxDataQueryFields     = 64
	MaxDataQuerySorts      = 8
	MaxDataQueryLimit      = 200
	MaxDataFilterDepth     = 8
	MaxDataFilterNodes     = 64
	MaxDataFilterValues    = 100
	MaxDataMutationValues  = 128
	MaxDataScopeIDs        = 64
	MaxDataValueBytes      = 1 << 20
	MaxDataCursorBytes     = 512
	MaxDataIdempotencySize = 128
)

var (
	dataIdentifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	dataPermissionPattern = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{0,127}$`)
	dataDecimalPattern    = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
	dataKeyPattern        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

var reservedDataScopeFields = map[string]struct{}{
	"tenant_id": {}, "organization_id": {}, "owner_id": {},
}

type DataStoreService interface {
	// Query and Mutate honor the transaction carried by a TransactionService callback context.
	Query(ctx context.Context, query DataQuery) (DataPage, error)
	Mutate(ctx context.Context, mutation DataMutation) (DataMutationResult, error)
}

type DataValueType string

const (
	DataValueNull      DataValueType = "null"
	DataValueString    DataValueType = "string"
	DataValueInteger   DataValueType = "integer"
	DataValueDecimal   DataValueType = "decimal"
	DataValueBoolean   DataValueType = "boolean"
	DataValueTimestamp DataValueType = "timestamp"
	DataValueBytes     DataValueType = "bytes"
	DataValueJSON      DataValueType = "json"
)

// DataValue keeps wire values explicit and lossless across JSON, Go, and SQL dialects.
type DataValue struct {
	Type  DataValueType `json:"type"`
	Value string        `json:"value,omitempty"`
}

func (v DataValue) Validate() error {
	switch v.Type {
	case DataValueNull:
		if v.Value != "" {
			return invalidDataContract("value", "null value must be empty")
		}
	case DataValueString:
		if len(v.Value) > MaxDataValueBytes {
			return invalidDataContract("value", "string value exceeds size limit")
		}
	case DataValueInteger:
		parsed, err := strconv.ParseInt(v.Value, 10, 64)
		if err != nil || strconv.FormatInt(parsed, 10) != v.Value {
			return invalidDataContract("value", "integer value must be canonical base-10 int64")
		}
	case DataValueDecimal:
		if len(v.Value) == 0 || len(v.Value) > 128 || !dataDecimalPattern.MatchString(v.Value) {
			return invalidDataContract("value", "decimal value must be a plain base-10 number")
		}
	case DataValueBoolean:
		if v.Value != "true" && v.Value != "false" {
			return invalidDataContract("value", "boolean value must be true or false")
		}
	case DataValueTimestamp:
		if _, err := time.Parse(time.RFC3339Nano, v.Value); err != nil {
			return invalidDataContract("value", "timestamp value must use RFC3339Nano")
		}
	case DataValueBytes:
		raw, err := base64.StdEncoding.DecodeString(v.Value)
		if err != nil || len(raw) > MaxDataValueBytes {
			return invalidDataContract("value", "bytes value must be bounded standard base64")
		}
	case DataValueJSON:
		if len(v.Value) == 0 || len(v.Value) > MaxDataValueBytes || !json.Valid([]byte(v.Value)) {
			return invalidDataContract("value", "json value must contain bounded valid JSON")
		}
	default:
		return invalidDataContract("type", "data value type is unsupported")
	}
	return nil
}

type DataScopeIntent struct {
	Permission Permission  `json:"permission"`
	Filter     ScopeFilter `json:"filter,omitempty"`
}

func (s DataScopeIntent) Validate() error {
	if !dataPermissionPattern.MatchString(strings.TrimSpace(s.Permission.Resource)) {
		return invalidDataContract("scope.permission.resource", "permission resource is invalid")
	}
	if !dataPermissionPattern.MatchString(strings.TrimSpace(s.Permission.Action)) {
		return invalidDataContract("scope.permission.action", "permission action is invalid")
	}
	dimensions := []struct {
		path   string
		values []string
	}{
		{path: "scope.filter.tenantIds", values: s.Filter.TenantIDs},
		{path: "scope.filter.ownerIds", values: s.Filter.OwnerIDs},
		{path: "scope.filter.organizationIds", values: s.Filter.OrganizationIDs},
	}
	for _, dimension := range dimensions {
		if err := validateDataScopeIDs(dimension.path, dimension.values); err != nil {
			return err
		}
	}
	return nil
}

type DataOperator string

const (
	DataOperatorEqual       DataOperator = "eq"
	DataOperatorNotEqual    DataOperator = "ne"
	DataOperatorLess        DataOperator = "lt"
	DataOperatorLessOrEqual DataOperator = "lte"
	DataOperatorGreater     DataOperator = "gt"
	DataOperatorGreaterOrEq DataOperator = "gte"
	DataOperatorIn          DataOperator = "in"
	DataOperatorNotIn       DataOperator = "not_in"
	DataOperatorContains    DataOperator = "contains"
	DataOperatorPrefix      DataOperator = "prefix"
	DataOperatorIsNull      DataOperator = "is_null"
	DataOperatorNotNull     DataOperator = "not_null"
)

type DataFilter struct {
	Field    string       `json:"field,omitempty"`
	Operator DataOperator `json:"operator,omitempty"`
	Value    *DataValue   `json:"value,omitempty"`
	Values   []DataValue  `json:"values,omitempty"`
	All      []DataFilter `json:"all,omitempty"`
	Any      []DataFilter `json:"any,omitempty"`
}

func (f DataFilter) Validate() error {
	nodes := 0
	return f.validate("filter", 1, &nodes)
}

func (f DataFilter) validate(path string, depth int, nodes *int) error {
	*nodes = *nodes + 1
	if depth > MaxDataFilterDepth {
		return invalidDataContract(path, "filter exceeds depth limit")
	}
	if *nodes > MaxDataFilterNodes {
		return invalidDataContract(path, "filter exceeds node limit")
	}
	isLeaf := f.Field != "" || f.Operator != "" || f.Value != nil || len(f.Values) > 0
	isAll := len(f.All) > 0
	isAny := len(f.Any) > 0
	if countDataBooleans(isLeaf, isAll, isAny) != 1 {
		return invalidDataContract(path, "filter must contain exactly one leaf, all group, or any group")
	}
	if isAll || isAny {
		children := f.All
		label := "all"
		if isAny {
			children = f.Any
			label = "any"
		}
		for index, child := range children {
			if err := child.validate(fmt.Sprintf("%s.%s[%d]", path, label, index), depth+1, nodes); err != nil {
				return err
			}
		}
		return nil
	}
	if err := validateDataIdentifier(path+".field", f.Field); err != nil {
		return err
	}
	if _, reserved := reservedDataScopeFields[f.Field]; reserved {
		return invalidDataContract(path+".field", "trusted scope field must use scope.filter")
	}
	switch f.Operator {
	case DataOperatorIn, DataOperatorNotIn:
		if f.Value != nil || len(f.Values) == 0 || len(f.Values) > MaxDataFilterValues {
			return invalidDataContract(path, "set operator requires a bounded values list")
		}
		for index, value := range f.Values {
			if err := validateDataValueAt(fmt.Sprintf("%s.values[%d]", path, index), value); err != nil {
				return err
			}
		}
	case DataOperatorIsNull, DataOperatorNotNull:
		if f.Value != nil || len(f.Values) > 0 {
			return invalidDataContract(path, "null operator does not accept values")
		}
	case DataOperatorEqual, DataOperatorNotEqual, DataOperatorLess, DataOperatorLessOrEqual,
		DataOperatorGreater, DataOperatorGreaterOrEq, DataOperatorContains, DataOperatorPrefix:
		if f.Value == nil || len(f.Values) > 0 {
			return invalidDataContract(path, "operator requires exactly one value")
		}
		if err := validateDataValueAt(path+".value", *f.Value); err != nil {
			return err
		}
		if (f.Operator == DataOperatorContains || f.Operator == DataOperatorPrefix) && f.Value.Type != DataValueString {
			return invalidDataContract(path+".value", "text operator requires a string value")
		}
	default:
		return invalidDataContract(path+".operator", "filter operator is unsupported")
	}
	return nil
}

type DataSortDirection string

const (
	DataSortAscending  DataSortDirection = "asc"
	DataSortDescending DataSortDirection = "desc"
)

type DataSort struct {
	Field     string            `json:"field"`
	Direction DataSortDirection `json:"direction"`
}

func (s DataSort) Validate() error {
	return s.validate("sort")
}

func (s DataSort) validate(path string) error {
	if err := validateDataIdentifier(path+".field", s.Field); err != nil {
		return err
	}
	if s.Direction != DataSortAscending && s.Direction != DataSortDescending {
		return invalidDataContract(path+".direction", "sort direction is unsupported")
	}
	return nil
}

type DataPageRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit"`
}

func (p DataPageRequest) Validate() error {
	if len(p.Cursor) > MaxDataCursorBytes {
		return invalidDataContract("page.cursor", "cursor exceeds size limit")
	}
	if p.Limit < 1 || p.Limit > MaxDataQueryLimit {
		return invalidDataContract("page.limit", "page limit is outside the supported range")
	}
	return nil
}

type DataQuery struct {
	Table  string          `json:"table"`
	Fields []string        `json:"fields"`
	Scope  DataScopeIntent `json:"scope"`
	Filter *DataFilter     `json:"filter,omitempty"`
	Sort   []DataSort      `json:"sort"`
	Page   DataPageRequest `json:"page"`
}

func (q DataQuery) Validate() error {
	if err := validateDataIdentifier("table", q.Table); err != nil {
		return err
	}
	if err := validateDataIdentifiers("fields", q.Fields, 1, MaxDataQueryFields); err != nil {
		return err
	}
	if err := q.Scope.Validate(); err != nil {
		return err
	}
	if q.Filter != nil {
		if err := q.Filter.Validate(); err != nil {
			return err
		}
	}
	if len(q.Sort) == 0 || len(q.Sort) > MaxDataQuerySorts {
		return invalidDataContract("sort", "query requires a bounded stable sort")
	}
	seenSorts := make(map[string]struct{}, len(q.Sort))
	for index, item := range q.Sort {
		path := fmt.Sprintf("sort[%d]", index)
		if err := item.validate(path); err != nil {
			return err
		}
		if _, exists := seenSorts[item.Field]; exists {
			return invalidDataContract(fmt.Sprintf("sort[%d].field", index), "sort field is duplicated")
		}
		seenSorts[item.Field] = struct{}{}
	}
	return q.Page.Validate()
}

type DataRecord struct {
	Values  map[string]DataValue `json:"values"`
	Version int64                `json:"version"`
}

func (r DataRecord) Validate() error {
	return r.validate("record")
}

func (r DataRecord) validate(path string) error {
	if err := validateDataValueMap(path+".values", r.Values, 1, MaxDataQueryFields, false); err != nil {
		return err
	}
	if r.Version < 1 {
		return invalidDataContract(path+".version", "record version must be positive")
	}
	return nil
}

type DataPage struct {
	Records    []DataRecord `json:"records"`
	NextCursor string       `json:"nextCursor,omitempty"`
	HasMore    bool         `json:"hasMore"`
}

func (p DataPage) Validate() error {
	if len(p.Records) > MaxDataQueryLimit {
		return invalidDataContract("records", "data page exceeds record limit")
	}
	if len(p.NextCursor) > MaxDataCursorBytes {
		return invalidDataContract("nextCursor", "next cursor exceeds size limit")
	}
	if p.HasMore != (p.NextCursor != "") {
		return invalidDataContract("nextCursor", "next cursor presence must match hasMore")
	}
	for index, record := range p.Records {
		if err := record.validate(fmt.Sprintf("records[%d]", index)); err != nil {
			return err
		}
	}
	return nil
}

type DataMutationOperation string

const (
	DataMutationInsert DataMutationOperation = "insert"
	DataMutationUpdate DataMutationOperation = "update"
	DataMutationUpsert DataMutationOperation = "upsert"
	DataMutationDelete DataMutationOperation = "delete"
)

type DataMutation struct {
	Table           string                `json:"table"`
	Operation       DataMutationOperation `json:"operation"`
	Scope           DataScopeIntent       `json:"scope"`
	Key             map[string]DataValue  `json:"key"`
	Values          map[string]DataValue  `json:"values,omitempty"`
	Returning       []string              `json:"returning,omitempty"`
	IdempotencyKey  string                `json:"idempotencyKey"`
	ExpectedVersion *int64                `json:"expectedVersion,omitempty"`
}

func (m DataMutation) Validate() error {
	if err := validateDataIdentifier("table", m.Table); err != nil {
		return err
	}
	if err := m.Scope.Validate(); err != nil {
		return err
	}
	if !dataKeyPattern.MatchString(m.IdempotencyKey) || len(m.IdempotencyKey) > MaxDataIdempotencySize {
		return invalidDataContract("idempotencyKey", "idempotency key is invalid")
	}
	if err := validateDataValueMap("key", m.Key, 1, 4, true); err != nil {
		return err
	}
	if err := validateDataIdentifiers("returning", m.Returning, 0, MaxDataQueryFields); err != nil {
		return err
	}
	switch m.Operation {
	case DataMutationInsert, DataMutationUpdate, DataMutationUpsert:
		if err := validateDataValueMap("values", m.Values, 1, MaxDataMutationValues, true); err != nil {
			return err
		}
	case DataMutationDelete:
		if len(m.Values) > 0 {
			return invalidDataContract("values", "delete mutation does not accept values")
		}
	default:
		return invalidDataContract("operation", "mutation operation is unsupported")
	}
	for _, field := range sortedDataKeys(m.Key) {
		if _, exists := m.Values[field]; exists {
			return invalidDataContract("values."+field, "key field cannot also be mutated")
		}
	}
	if m.ExpectedVersion != nil {
		if *m.ExpectedVersion < 1 {
			return invalidDataContract("expectedVersion", "expected version must be positive")
		}
		if m.Operation == DataMutationInsert {
			return invalidDataContract("expectedVersion", "insert mutation cannot declare an expected version")
		}
	}
	return nil
}

type DataMutationResult struct {
	RowsAffected int64       `json:"rowsAffected"`
	Record       *DataRecord `json:"record,omitempty"`
}

func (r DataMutationResult) Validate() error {
	if r.RowsAffected < 0 || r.RowsAffected > 1 {
		return invalidDataContract("rowsAffected", "single-record mutation affected an invalid row count")
	}
	if r.Record != nil {
		if r.RowsAffected != 1 {
			return invalidDataContract("record", "returned record requires one affected row")
		}
		if err := r.Record.validate("record"); err != nil {
			return err
		}
	}
	return nil
}

type DataStoreErrorCode string

const (
	DataStoreErrorInvalidRequest DataStoreErrorCode = "invalid_request"
	DataStoreErrorForbidden      DataStoreErrorCode = "forbidden"
	DataStoreErrorNotFound       DataStoreErrorCode = "not_found"
	DataStoreErrorConflict       DataStoreErrorCode = "conflict"
	DataStoreErrorLimitExceeded  DataStoreErrorCode = "limit_exceeded"
	DataStoreErrorUnsupported    DataStoreErrorCode = "unsupported"
	DataStoreErrorUnavailable    DataStoreErrorCode = "unavailable"
)

type DataStoreError struct {
	Code      DataStoreErrorCode `json:"code"`
	Field     string             `json:"field,omitempty"`
	Message   string             `json:"message"`
	Retryable bool               `json:"retryable"`
}

func (e *DataStoreError) Error() string {
	if e == nil {
		return "plugin datastore operation failed"
	}
	if e.Field == "" {
		return fmt.Sprintf("plugin datastore operation failed: %s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("plugin datastore operation failed: %s: %s: %s", e.Code, e.Field, e.Message)
}

func NewDataStoreError(code DataStoreErrorCode, field, message string, retryable bool) *DataStoreError {
	return &DataStoreError{Code: code, Field: strings.TrimSpace(field), Message: strings.TrimSpace(message), Retryable: retryable}
}

func invalidDataContract(field, message string) error {
	return NewDataStoreError(DataStoreErrorInvalidRequest, field, message, false)
}

func validateDataIdentifier(path, value string) error {
	if !dataIdentifierPattern.MatchString(value) {
		return invalidDataContract(path, "identifier is invalid")
	}
	return nil
}

func validateDataIdentifiers(path string, values []string, minimum, maximum int) error {
	if len(values) < minimum || len(values) > maximum {
		return invalidDataContract(path, "identifier list is outside the supported range")
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if err := validateDataIdentifier(fmt.Sprintf("%s[%d]", path, index), value); err != nil {
			return err
		}
		if _, exists := seen[value]; exists {
			return invalidDataContract(fmt.Sprintf("%s[%d]", path, index), "identifier is duplicated")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateDataScopeIDs(path string, values []string) error {
	if len(values) > MaxDataScopeIDs {
		return invalidDataContract(path, "scope identifier list exceeds limit")
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 128 {
			return invalidDataContract(fmt.Sprintf("%s[%d]", path, index), "scope identifier is invalid")
		}
		if _, exists := seen[value]; exists {
			return invalidDataContract(fmt.Sprintf("%s[%d]", path, index), "scope identifier is duplicated")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateDataValueAt(path string, value DataValue) error {
	if err := value.Validate(); err != nil {
		return withDataErrorPath(path, err)
	}
	return nil
}

func validateDataValueMap(path string, values map[string]DataValue, minimum, maximum int, rejectReserved bool) error {
	if len(values) < minimum || len(values) > maximum {
		return invalidDataContract(path, "value map is outside the supported range")
	}
	for _, key := range sortedDataKeys(values) {
		if err := validateDataIdentifier(path+"."+key, key); err != nil {
			return err
		}
		if rejectReserved {
			if _, reserved := reservedDataScopeFields[key]; reserved {
				return invalidDataContract(path+"."+key, "trusted scope field cannot be supplied by a plugin")
			}
		}
		if err := validateDataValueAt(path+"."+key, values[key]); err != nil {
			return err
		}
	}
	return nil
}

func sortedDataKeys(values map[string]DataValue) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func withDataErrorPath(prefix string, err error) error {
	dataErr, ok := err.(*DataStoreError)
	if !ok {
		return err
	}
	field := prefix
	if dataErr.Field != "" {
		field += "." + dataErr.Field
	}
	return NewDataStoreError(dataErr.Code, field, dataErr.Message, dataErr.Retryable)
}

func countDataBooleans(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}
