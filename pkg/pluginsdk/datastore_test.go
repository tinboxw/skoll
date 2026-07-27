package pluginsdk

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDataValueValidation(t *testing.T) {
	valid := []DataValue{
		{Type: DataValueNull},
		{Type: DataValueString, Value: "medical product"},
		{Type: DataValueInteger, Value: "-42"},
		{Type: DataValueDecimal, Value: "12345.6700"},
		{Type: DataValueBoolean, Value: "false"},
		{Type: DataValueTimestamp, Value: "2026-07-22T10:20:30.123Z"},
		{Type: DataValueBytes, Value: base64.StdEncoding.EncodeToString([]byte("proof"))},
		{Type: DataValueJSON, Value: `{"batch":"A-01","valid":true}`},
	}
	for _, value := range valid {
		if err := value.Validate(); err != nil {
			t.Fatalf("valid value %+v: %v", value, err)
		}
	}

	invalid := []DataValue{
		{Type: DataValueNull, Value: "null"},
		{Type: DataValueInteger, Value: "01"},
		{Type: DataValueDecimal, Value: "1e3"},
		{Type: DataValueBoolean, Value: "TRUE"},
		{Type: DataValueTimestamp, Value: "2026-07-22"},
		{Type: DataValueBytes, Value: "%%%"},
		{Type: DataValueJSON, Value: "{"},
		{Type: "float", Value: "1.2"},
	}
	for _, value := range invalid {
		if err := value.Validate(); !isDataContractError(err, "") {
			t.Fatalf("invalid value %+v returned %v", value, err)
		}
	}
}

func TestDataQueryValidationAcceptsStructuredScopedQuery(t *testing.T) {
	query := validDataQuery()
	if err := query.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDataQueryValidationRejectsAmbiguousOrUnboundedInput(t *testing.T) {
	tests := []struct {
		name  string
		alter func(*DataQuery)
		field string
	}{
		{name: "table", alter: func(q *DataQuery) { q.Table = "Core.Users" }, field: "table"},
		{name: "explicit fields", alter: func(q *DataQuery) { q.Fields = nil }, field: "fields"},
		{name: "duplicate fields", alter: func(q *DataQuery) { q.Fields = []string{"id", "id"} }, field: "fields[1]"},
		{name: "permission", alter: func(q *DataQuery) { q.Scope.Permission.Resource = "" }, field: "scope.permission.resource"},
		{name: "duplicate scope", alter: func(q *DataQuery) { q.Scope.Filter.TenantIDs = []string{"tenant-1", "tenant-1"} }, field: "scope.filter.tenantIds[1]"},
		{name: "mixed filter", alter: func(q *DataQuery) { q.Filter.All = []DataFilter{{Field: "id", Operator: DataOperatorIsNull}} }, field: "filter"},
		{name: "text operator type", alter: func(q *DataQuery) { q.Filter.Value = dataValue(DataValueInteger, "1") }, field: "filter.value"},
		{name: "duplicate sort", alter: func(q *DataQuery) { q.Sort = append(q.Sort, q.Sort[0]) }, field: "sort[1].field"},
		{name: "missing stable sort", alter: func(q *DataQuery) { q.Sort = nil }, field: "sort"},
		{name: "limit", alter: func(q *DataQuery) { q.Page.Limit = MaxDataQueryLimit + 1 }, field: "page.limit"},
		{name: "cursor", alter: func(q *DataQuery) { q.Page.Cursor = strings.Repeat("x", MaxDataCursorBytes+1) }, field: "page.cursor"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := validDataQuery()
			test.alter(&query)
			if err := query.Validate(); !isDataContractError(err, test.field) {
				t.Fatalf("error=%v want field=%s", err, test.field)
			}
		})
	}
}

func TestDataFilterValidationEnforcesShapeAndComplexity(t *testing.T) {
	value := dataValue(DataValueString, "active")
	valid := DataFilter{All: []DataFilter{
		{Field: "status", Operator: DataOperatorEqual, Value: value},
		{Any: []DataFilter{
			{Field: "batch_no", Operator: DataOperatorPrefix, Value: dataValue(DataValueString, "B-2026")},
			{Field: "released_at", Operator: DataOperatorIsNull},
		}},
	}}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}

	tooDeep := DataFilter{Field: "id", Operator: DataOperatorIsNull}
	for range MaxDataFilterDepth {
		tooDeep = DataFilter{All: []DataFilter{tooDeep}}
	}
	if err := tooDeep.Validate(); !isDataContractError(err, "filter.all[0]") {
		t.Fatalf("depth error=%v", err)
	}

	tooMany := DataFilter{Any: make([]DataFilter, MaxDataFilterNodes+1)}
	for index := range tooMany.Any {
		tooMany.Any[index] = DataFilter{Field: "id", Operator: DataOperatorIsNull}
	}
	if err := tooMany.Validate(); !isDataContractError(err, "filter.any") {
		t.Fatalf("node error=%v", err)
	}
}

func TestDataMutationValidationAcceptsCurrentOperations(t *testing.T) {
	for _, operation := range []DataMutationOperation{DataMutationInsert, DataMutationUpdate, DataMutationUpsert, DataMutationDelete, DataMutationAdjust} {
		mutation := validDataMutation(operation)
		if err := mutation.Validate(); err != nil {
			t.Fatalf("operation=%s: %v", operation, err)
		}
	}
}

func TestDataAdjustmentValidationRejectsUnsafeArithmetic(t *testing.T) {
	tests := []struct {
		name  string
		alter func(*DataMutation)
		field string
	}{
		{name: "missing adjustment", alter: func(m *DataMutation) { m.Adjustment = nil }, field: "adjustment"},
		{name: "replacement values", alter: func(m *DataMutation) {
			m.Values = map[string]DataValue{"amount": {Type: DataValueDecimal, Value: "12.30"}}
		}, field: "values"},
		{name: "scope field", alter: func(m *DataMutation) { m.Adjustment.Field = "tenant_id" }, field: "adjustment.field"},
		{name: "raw expression", alter: func(m *DataMutation) {
			m.Adjustment.Delta = DataValue{Type: DataValueDecimal, Value: "amount + 1"}
		}, field: "adjustment.delta"},
		{name: "float type", alter: func(m *DataMutation) {
			m.Adjustment.Delta = DataValue{Type: "float", Value: "1.25"}
		}, field: "adjustment.delta.type"},
		{name: "zero delta", alter: func(m *DataMutation) {
			m.Adjustment.Delta = DataValue{Type: DataValueDecimal, Value: "0.00"}
		}, field: "adjustment.delta"},
		{name: "mixed bound type", alter: func(m *DataMutation) {
			m.Adjustment.Minimum = dataValue(DataValueInteger, "0")
		}, field: "adjustment.minimum.type"},
		{name: "reversed bounds", alter: func(m *DataMutation) {
			m.Adjustment.Minimum = dataValue(DataValueDecimal, "10.00")
			m.Adjustment.Maximum = dataValue(DataValueDecimal, "9.99")
		}, field: "adjustment.minimum"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutation := validDataMutation(DataMutationAdjust)
			test.alter(&mutation)
			if err := mutation.Validate(); !isDataContractError(err, test.field) {
				t.Fatalf("error=%v want field=%s", err, test.field)
			}
		})
	}
}

func TestDataMutationValidationRejectsScopeInjectionAndInvalidSemantics(t *testing.T) {
	tests := []struct {
		name  string
		alter func(*DataMutation)
		field string
	}{
		{name: "missing key", alter: func(m *DataMutation) { m.Key = nil }, field: "key"},
		{name: "key scope injection", alter: func(m *DataMutation) {
			m.Key = map[string]DataValue{"tenant_id": {Type: DataValueString, Value: "other"}}
		}, field: "key.tenant_id"},
		{name: "scope injection", alter: func(m *DataMutation) { m.Values["tenant_id"] = DataValue{Type: DataValueString, Value: "other"} }, field: "values.tenant_id"},
		{name: "key mutation", alter: func(m *DataMutation) { m.Values["id"] = DataValue{Type: DataValueString, Value: "new"} }, field: "values.id"},
		{name: "invalid idempotency", alter: func(m *DataMutation) { m.IdempotencyKey = "contains space" }, field: "idempotencyKey"},
		{name: "insert version", alter: func(m *DataMutation) { version := int64(1); m.ExpectedVersion = &version }, field: "expectedVersion"},
		{name: "delete values", alter: func(m *DataMutation) { m.Operation = DataMutationDelete }, field: "values"},
		{name: "unknown operation", alter: func(m *DataMutation) { m.Operation = "merge" }, field: "operation"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutation := validDataMutation(DataMutationInsert)
			test.alter(&mutation)
			if err := mutation.Validate(); !isDataContractError(err, test.field) {
				t.Fatalf("error=%v want field=%s", err, test.field)
			}
		})
	}
}

func TestDataResponseValidation(t *testing.T) {
	record := DataRecord{Values: map[string]DataValue{"id": {Type: DataValueString, Value: "product-1"}}, Version: 1}
	page := DataPage{Records: []DataRecord{record}, NextCursor: "cursor-2", HasMore: true}
	if err := page.Validate(); err != nil {
		t.Fatal(err)
	}
	result := DataMutationResult{RowsAffected: 1, Record: &record}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}

	page.HasMore = false
	if err := page.Validate(); !isDataContractError(err, "nextCursor") {
		t.Fatalf("page error=%v", err)
	}
	result.RowsAffected = 0
	if err := result.Validate(); !isDataContractError(err, "record") {
		t.Fatalf("mutation result error=%v", err)
	}
}

func TestDataContractWireUsesCamelCase(t *testing.T) {
	raw, err := json.Marshal(validDataQuery())
	if err != nil {
		t.Fatal(err)
	}
	wire := string(raw)
	for _, expected := range []string{`"permission":{"resource":`, `"filter":{"tenantIds":`, `"organizationIds":`} {
		if !strings.Contains(wire, expected) {
			t.Fatalf("wire=%s missing=%s", wire, expected)
		}
	}
	adjustmentRaw, err := json.Marshal(validDataMutation(DataMutationAdjust))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"operation":"adjust"`, `"adjustment":{"field":"amount"`, `"delta":{"type":"decimal","value":"1.25"}`, `"minimum":{"type":"decimal","value":"0.00"}`} {
		if !strings.Contains(string(adjustmentRaw), expected) {
			t.Fatalf("adjustment wire=%s missing=%s", adjustmentRaw, expected)
		}
	}
	for _, forbidden := range []string{`"Resource"`, `"TenantIDs"`, `"OrganizationIDs"`} {
		if strings.Contains(wire, forbidden) {
			t.Fatalf("wire=%s contains=%s", wire, forbidden)
		}
	}
}

func TestDataStoreErrorIsStableAndInspectable(t *testing.T) {
	err := NewDataStoreError(DataStoreErrorConflict, "expectedVersion", "record version changed", true)
	var target *DataStoreError
	if !errors.As(err, &target) || target.Code != DataStoreErrorConflict || target.Field != "expectedVersion" || !target.Retryable {
		t.Fatalf("unexpected error: %+v", target)
	}
	if got := err.Error(); got != "plugin datastore operation failed: conflict: expectedVersion: record version changed" {
		t.Fatalf("error text=%q", got)
	}
}

func validDataQuery() DataQuery {
	return DataQuery{
		Table:  "products",
		Fields: []string{"id", "code", "name", "status", "updated_at"},
		Scope: DataScopeIntent{
			Permission: Permission{Resource: "medical_oa.product", Action: "read"},
			Filter:     ScopeFilter{TenantIDs: []string{"tenant-1"}, OrganizationIDs: []string{"org-1"}},
		},
		Filter: &DataFilter{Field: "name", Operator: DataOperatorContains, Value: dataValue(DataValueString, "tablet")},
		Sort:   []DataSort{{Field: "updated_at", Direction: DataSortDescending}},
		Page:   DataPageRequest{Limit: 50},
	}
}

func validDataMutation(operation DataMutationOperation) DataMutation {
	mutation := DataMutation{
		Table:          "products",
		Operation:      operation,
		Scope:          DataScopeIntent{Permission: Permission{Resource: "medical_oa.product", Action: "write"}},
		Key:            map[string]DataValue{"id": {Type: DataValueString, Value: "product-1"}},
		Values:         map[string]DataValue{"name": {Type: DataValueString, Value: "Tablet"}},
		Returning:      []string{"id", "name", "updated_at"},
		IdempotencyKey: "product-1.create",
	}
	if operation == DataMutationDelete {
		mutation.Values = nil
	}
	if operation == DataMutationAdjust {
		mutation.Values = nil
		mutation.Adjustment = &DataAdjustment{
			Field: "amount", Delta: DataValue{Type: DataValueDecimal, Value: "1.25"},
			Minimum: dataValue(DataValueDecimal, "0.00"),
			Maximum: dataValue(DataValueDecimal, "999999.99"),
		}
	}
	return mutation
}

func dataValue(kind DataValueType, value string) *DataValue {
	return &DataValue{Type: kind, Value: value}
}

func isDataContractError(err error, field string) bool {
	var target *DataStoreError
	if !errors.As(err, &target) || target.Code != DataStoreErrorInvalidRequest {
		return false
	}
	return field == "" || strings.HasPrefix(target.Field, field)
}
