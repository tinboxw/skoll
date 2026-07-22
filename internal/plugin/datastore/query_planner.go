package datastore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type SQLDialect string

const (
	DialectSQLite      SQLDialect = "sqlite"
	DialectPostgreSQL  SQLDialect = "postgresql"
	DialectMySQL       SQLDialect = "mysql"
	MaxQueryParameters            = 900
)

type QueryPlan struct {
	Dialect      SQLDialect
	SQL          string
	Args         []any
	Fields       []string
	ScanFields   []string
	Sort         []pluginsdk.DataSort
	Limit        int
	FetchLimit   int
	PluginID     string
	LogicalTable string

	table ResolvedTable
}

type QueryPlanner struct {
	registry *SchemaRegistry
	scopes   pluginsdk.DataScopeService
	dialect  SQLDialect
}

type queryCursor struct {
	PluginID string                `json:"pluginId"`
	Table    string                `json:"table"`
	Sort     []pluginsdk.DataSort  `json:"sort"`
	Values   []pluginsdk.DataValue `json:"values"`
}

type sqlPlanBuilder struct {
	dialect SQLDialect
	args    []any
}

func ParseSQLDialect(value string) (SQLDialect, error) {
	switch SQLDialect(strings.ToLower(strings.TrimSpace(value))) {
	case DialectSQLite:
		return DialectSQLite, nil
	case DialectPostgreSQL:
		return DialectPostgreSQL, nil
	case DialectMySQL:
		return DialectMySQL, nil
	default:
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, "dialect", "database dialect is unsupported", false)
	}
}

func NewQueryPlanner(registry *SchemaRegistry, scopes pluginsdk.DataScopeService, dialect SQLDialect) (*QueryPlanner, error) {
	if registry == nil {
		return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "registry", "schema registry is unavailable", true)
	}
	if scopes == nil {
		return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "scope", "trusted data scope service is unavailable", true)
	}
	parsed, err := ParseSQLDialect(string(dialect))
	if err != nil {
		return nil, err
	}
	return &QueryPlanner{registry: registry, scopes: scopes, dialect: parsed}, nil
}

func (p *QueryPlanner) Plan(ctx context.Context, pluginID string, query pluginsdk.DataQuery) (QueryPlan, error) {
	if p == nil || p.registry == nil || p.scopes == nil {
		return QueryPlan{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "planner", "query planner is unavailable", true)
	}
	if ctx == nil {
		return QueryPlan{}, invalidSchema("context", "query context is required")
	}
	if err := query.Validate(); err != nil {
		return QueryPlan{}, err
	}
	pluginID = strings.TrimSpace(pluginID)
	if err := validateQueryPermission(pluginID, query.Scope.Permission); err != nil {
		return QueryPlan{}, err
	}
	table, err := p.registry.Resolve(pluginID, query.Table)
	if err != nil {
		return QueryPlan{}, err
	}
	if err := validateSelectedFields(table, query.Fields); err != nil {
		return QueryPlan{}, err
	}
	effectiveSort, err := buildEffectiveSort(table, query.Sort)
	if err != nil {
		return QueryPlan{}, err
	}

	trusted, err := p.scopes.Resolve(ctx, query.Scope.Permission)
	if err != nil {
		return QueryPlan{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope.permission", "trusted data scope could not be resolved", false)
	}
	trusted = trusted.Constrain(query.Scope.Filter)
	if err := validateTrustedScope(trusted); err != nil {
		return QueryPlan{}, err
	}

	builder := &sqlPlanBuilder{dialect: p.dialect}
	clauses, err := builder.scopeClauses(trusted)
	if err != nil {
		return QueryPlan{}, err
	}
	if query.Filter != nil {
		clause, compileErr := builder.filterClause(table, *query.Filter, "filter")
		if compileErr != nil {
			return QueryPlan{}, compileErr
		}
		clauses = append(clauses, clause)
	}
	if query.Page.Cursor != "" {
		cursor, decodeErr := decodeQueryCursor(query.Page.Cursor)
		if decodeErr != nil {
			return QueryPlan{}, decodeErr
		}
		clause, compileErr := builder.cursorClause(table, table.PluginID, effectiveSort, cursor)
		if compileErr != nil {
			return QueryPlan{}, compileErr
		}
		clauses = append(clauses, clause)
	}
	if len(builder.args)+1 > MaxQueryParameters {
		return QueryPlan{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "query", "query exceeds the bound parameter limit", false)
	}

	scanFields := appendUniqueFields(query.Fields, FieldVersion)
	for _, item := range effectiveSort {
		scanFields = appendUniqueFields(scanFields, item.Field)
	}
	quotedFields := make([]string, len(scanFields))
	for index, field := range scanFields {
		quotedFields[index] = quoteSQLIdentifier(p.dialect, field)
	}
	var statement strings.Builder
	statement.WriteString("SELECT ")
	statement.WriteString(strings.Join(quotedFields, ", "))
	statement.WriteString(" FROM ")
	statement.WriteString(quoteSQLIdentifier(p.dialect, table.PhysicalName))
	if len(clauses) > 0 {
		statement.WriteString(" WHERE ")
		statement.WriteString(strings.Join(clauses, " AND "))
	}
	statement.WriteString(" ORDER BY ")
	for index, item := range effectiveSort {
		if index > 0 {
			statement.WriteString(", ")
		}
		statement.WriteString(quoteSQLIdentifier(p.dialect, item.Field))
		statement.WriteByte(' ')
		statement.WriteString(strings.ToUpper(string(item.Direction)))
	}
	fetchLimit := query.Page.Limit + 1
	statement.WriteString(" LIMIT ")
	statement.WriteString(builder.bind(fetchLimit))

	return QueryPlan{
		Dialect: p.dialect, SQL: statement.String(), Args: append([]any(nil), builder.args...),
		Fields: append([]string(nil), query.Fields...), ScanFields: scanFields, Sort: effectiveSort,
		Limit: query.Page.Limit, FetchLimit: fetchLimit, PluginID: table.PluginID, LogicalTable: table.LogicalName,
		table: cloneResolvedTable(table),
	}, nil
}

func validateQueryPermission(pluginID string, permission pluginsdk.Permission) error {
	resource := strings.TrimSpace(permission.Resource)
	if resource != pluginID && !strings.HasPrefix(resource, pluginID+".") && !strings.HasPrefix(resource, pluginID+":") {
		return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope.permission.resource", "permission resource is not owned by the plugin", false)
	}
	return nil
}

func (p QueryPlan) EncodeCursor(values map[string]pluginsdk.DataValue) (string, error) {
	if p.PluginID == "" || p.LogicalTable == "" || len(p.Sort) == 0 || len(p.table.Fields) == 0 {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "cursor", "query plan cannot encode a cursor", false)
	}
	cursor := queryCursor{PluginID: p.PluginID, Table: p.LogicalTable, Sort: append([]pluginsdk.DataSort(nil), p.Sort...), Values: make([]pluginsdk.DataValue, len(p.Sort))}
	for index, item := range p.Sort {
		value, exists := values[item.Field]
		if !exists {
			return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, fmt.Sprintf("cursor.values.%s", item.Field), "cursor field value is missing", false)
		}
		if err := validateQueryValue(p.table.Fields[item.Field], value, fmt.Sprintf("cursor.values.%s", item.Field), false); err != nil {
			return "", err
		}
		cursor.Values[index] = value
	}
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "cursor", "cursor could not be encoded", false)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	if len(encoded) > pluginsdk.MaxDataCursorBytes {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "cursor", "cursor exceeds size limit", false)
	}
	return encoded, nil
}

func validateSelectedFields(table ResolvedTable, fields []string) error {
	for index, name := range fields {
		if _, exists := table.Fields[name]; !exists {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, fmt.Sprintf("fields[%d]", index), "field is not declared by the plugin schema", false)
		}
	}
	return nil
}

func buildEffectiveSort(table ResolvedTable, requested []pluginsdk.DataSort) ([]pluginsdk.DataSort, error) {
	effective := append([]pluginsdk.DataSort(nil), requested...)
	seen := make(map[string]struct{}, len(requested)+len(table.PrimaryKey))
	for index, item := range requested {
		field, exists := table.Fields[item.Field]
		if !exists {
			return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, fmt.Sprintf("sort[%d].field", index), "sort field is not declared by the plugin schema", false)
		}
		if !field.Sortable {
			return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, fmt.Sprintf("sort[%d].field", index), "field is not sortable", false)
		}
		if field.Nullable {
			return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, fmt.Sprintf("sort[%d].field", index), "nullable sort fields are not supported", false)
		}
		seen[item.Field] = struct{}{}
	}
	direction := pluginsdk.DataSortAscending
	if len(requested) > 0 {
		direction = requested[len(requested)-1].Direction
	}
	for _, field := range table.PrimaryKey {
		if _, exists := seen[field]; exists {
			continue
		}
		effective = append(effective, pluginsdk.DataSort{Field: field, Direction: direction})
		seen[field] = struct{}{}
	}
	return effective, nil
}

func validateTrustedScope(scope pluginsdk.ScopePredicate) error {
	if scope.Denied() {
		return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope.filter", "requested data scope is outside the trusted scope", false)
	}
	if strings.TrimSpace(scope.SubjectID()) == "" || (!scope.AllTenants() && len(scope.TenantIDs()) == 0) ||
		(!scope.AllOwners() && len(scope.OwnerIDs()) == 0) || (!scope.AllOrganizations() && len(scope.OrganizationIDs()) == 0) {
		return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope", "trusted data scope is incomplete", false)
	}
	return nil
}

func (b *sqlPlanBuilder) scopeClauses(scope pluginsdk.ScopePredicate) ([]string, error) {
	dimensions := []struct {
		field string
		ids   []string
		all   bool
	}{
		{field: FieldTenantID, ids: scope.TenantIDs(), all: scope.AllTenants()},
		{field: FieldOrganizationID, ids: scope.OrganizationIDs(), all: scope.AllOrganizations()},
		{field: FieldOwnerID, ids: scope.OwnerIDs(), all: scope.AllOwners()},
	}
	clauses := make([]string, 0, len(dimensions))
	for _, dimension := range dimensions {
		if dimension.all {
			continue
		}
		if len(dimension.ids) == 0 {
			return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope", "trusted data scope is incomplete", false)
		}
		ids := append([]string(nil), dimension.ids...)
		sort.Strings(ids)
		field := quoteSQLIdentifier(b.dialect, dimension.field)
		if len(ids) == 1 {
			clauses = append(clauses, field+" = "+b.bind(ids[0]))
			continue
		}
		placeholders := make([]string, len(ids))
		for index, id := range ids {
			placeholders[index] = b.bind(id)
		}
		clauses = append(clauses, field+" IN ("+strings.Join(placeholders, ", ")+")")
	}
	return clauses, nil
}

func (b *sqlPlanBuilder) filterClause(table ResolvedTable, filter pluginsdk.DataFilter, path string) (string, error) {
	if len(filter.All) > 0 || len(filter.Any) > 0 {
		children := filter.All
		operator := " AND "
		label := "all"
		if len(filter.Any) > 0 {
			children = filter.Any
			operator = " OR "
			label = "any"
		}
		clauses := make([]string, len(children))
		for index, child := range children {
			clause, err := b.filterClause(table, child, fmt.Sprintf("%s.%s[%d]", path, label, index))
			if err != nil {
				return "", err
			}
			clauses[index] = clause
		}
		return "(" + strings.Join(clauses, operator) + ")", nil
	}

	field, exists := table.Fields[filter.Field]
	if !exists || !field.Filterable {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, path+".field", "field is not declared as filterable", false)
	}
	identifier := quoteSQLIdentifier(b.dialect, filter.Field)
	switch filter.Operator {
	case pluginsdk.DataOperatorIsNull, pluginsdk.DataOperatorNotNull:
		if !field.Nullable {
			return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path+".field", "null filtering requires a nullable field", false)
		}
		if filter.Operator == pluginsdk.DataOperatorIsNull {
			return identifier + " IS NULL", nil
		}
		return identifier + " IS NOT NULL", nil
	case pluginsdk.DataOperatorIn, pluginsdk.DataOperatorNotIn:
		placeholders := make([]string, len(filter.Values))
		for index, value := range filter.Values {
			converted, err := convertQueryValue(field, value, fmt.Sprintf("%s.values[%d]", path, index), false)
			if err != nil {
				return "", err
			}
			placeholders[index] = b.bind(converted)
		}
		operator := " IN "
		if filter.Operator == pluginsdk.DataOperatorNotIn {
			operator = " NOT IN "
		}
		return identifier + operator + "(" + strings.Join(placeholders, ", ") + ")", nil
	default:
		converted, err := convertQueryValue(field, *filter.Value, path+".value", true)
		if err != nil {
			return "", err
		}
		if converted == nil {
			switch filter.Operator {
			case pluginsdk.DataOperatorEqual:
				return identifier + " IS NULL", nil
			case pluginsdk.DataOperatorNotEqual:
				return identifier + " IS NOT NULL", nil
			default:
				return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path+".operator", "operator does not support null", false)
			}
		}
		operator, err := queryOperator(field, filter.Operator, path)
		if err != nil {
			return "", err
		}
		if filter.Operator == pluginsdk.DataOperatorContains || filter.Operator == pluginsdk.DataOperatorPrefix {
			value := escapeLikePattern(converted.(string))
			if filter.Operator == pluginsdk.DataOperatorContains {
				value = "%" + value + "%"
			} else {
				value += "%"
			}
			return identifier + " LIKE " + b.bind(value) + " ESCAPE '!'", nil
		}
		return identifier + " " + operator + " " + b.bind(converted), nil
	}
}

func (b *sqlPlanBuilder) cursorClause(table ResolvedTable, pluginID string, effectiveSort []pluginsdk.DataSort, cursor queryCursor) (string, error) {
	if cursor.PluginID != pluginID || cursor.Table != table.LogicalName || !reflect.DeepEqual(cursor.Sort, effectiveSort) || len(cursor.Values) != len(effectiveSort) {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "page.cursor", "cursor does not match the query", false)
	}
	converted := make([]any, len(cursor.Values))
	for index, value := range cursor.Values {
		field := table.Fields[effectiveSort[index].Field]
		item, err := convertQueryValue(field, value, fmt.Sprintf("page.cursor.values[%d]", index), false)
		if err != nil {
			return "", err
		}
		converted[index] = item
	}
	branches := make([]string, len(effectiveSort))
	for index, item := range effectiveSort {
		parts := make([]string, 0, index+1)
		for previous := 0; previous < index; previous++ {
			parts = append(parts, quoteSQLIdentifier(b.dialect, effectiveSort[previous].Field)+" = "+b.bind(converted[previous]))
		}
		operator := ">"
		if item.Direction == pluginsdk.DataSortDescending {
			operator = "<"
		}
		parts = append(parts, quoteSQLIdentifier(b.dialect, item.Field)+" "+operator+" "+b.bind(converted[index]))
		branches[index] = "(" + strings.Join(parts, " AND ") + ")"
	}
	return "(" + strings.Join(branches, " OR ") + ")", nil
}

func queryOperator(field FieldSchema, operator pluginsdk.DataOperator, path string) (string, error) {
	switch operator {
	case pluginsdk.DataOperatorEqual:
		return "=", nil
	case pluginsdk.DataOperatorNotEqual:
		return "<>", nil
	case pluginsdk.DataOperatorLess, pluginsdk.DataOperatorLessOrEqual, pluginsdk.DataOperatorGreater, pluginsdk.DataOperatorGreaterOrEq:
		if field.Type == pluginsdk.DataValueBoolean {
			return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path+".operator", "boolean fields support equality operators only", false)
		}
		switch operator {
		case pluginsdk.DataOperatorLess:
			return "<", nil
		case pluginsdk.DataOperatorLessOrEqual:
			return "<=", nil
		case pluginsdk.DataOperatorGreater:
			return ">", nil
		default:
			return ">=", nil
		}
	case pluginsdk.DataOperatorContains, pluginsdk.DataOperatorPrefix:
		if field.Type != pluginsdk.DataValueString {
			return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path+".operator", "text operator requires a string field", false)
		}
		return "LIKE", nil
	default:
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path+".operator", "filter operator is unsupported", false)
	}
}

func validateQueryValue(field FieldSchema, value pluginsdk.DataValue, path string, allowNull bool) error {
	_, err := convertQueryValue(field, value, path, allowNull)
	return err
}

func convertQueryValue(field FieldSchema, value pluginsdk.DataValue, path string, allowNull bool) (any, error) {
	if err := value.Validate(); err != nil {
		return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, path, "query value is invalid", false)
	}
	if value.Type == pluginsdk.DataValueNull {
		if !allowNull || !field.Nullable {
			return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path, "null is not supported for this field", false)
		}
		return nil, nil
	}
	if value.Type != field.Type {
		return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, path+".type", "query value type does not match the schema", false)
	}
	switch value.Type {
	case pluginsdk.DataValueString, pluginsdk.DataValueDecimal, pluginsdk.DataValueJSON:
		return value.Value, nil
	case pluginsdk.DataValueInteger:
		return strconv.ParseInt(value.Value, 10, 64)
	case pluginsdk.DataValueBoolean:
		return strconv.ParseBool(value.Value)
	case pluginsdk.DataValueTimestamp:
		return time.Parse(time.RFC3339Nano, value.Value)
	case pluginsdk.DataValueBytes:
		return base64.StdEncoding.DecodeString(value.Value)
	default:
		return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path+".type", "query value type is unsupported", false)
	}
}

func decodeQueryCursor(encoded string) (queryCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return queryCursor{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "page.cursor", "cursor encoding is invalid", false)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var cursor queryCursor
	if err = decoder.Decode(&cursor); err != nil {
		return queryCursor{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "page.cursor", "cursor payload is invalid", false)
	}
	if err = decoder.Decode(&struct{}{}); err != io.EOF {
		return queryCursor{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "page.cursor", "cursor contains trailing data", false)
	}
	return cursor, nil
}

func (b *sqlPlanBuilder) bind(value any) string {
	b.args = append(b.args, value)
	if b.dialect == DialectPostgreSQL {
		return "$" + strconv.Itoa(len(b.args))
	}
	return "?"
}

func quoteSQLIdentifier(dialect SQLDialect, value string) string {
	if dialect == DialectMySQL {
		return "`" + value + "`"
	}
	return `"` + value + `"`
}

func appendUniqueFields(fields []string, candidates ...string) []string {
	out := append([]string(nil), fields...)
	seen := make(map[string]struct{}, len(out)+len(candidates))
	for _, field := range out {
		seen[field] = struct{}{}
	}
	for _, candidate := range candidates {
		if _, exists := seen[candidate]; exists {
			continue
		}
		out = append(out, candidate)
		seen[candidate] = struct{}{}
	}
	return out
}

func escapeLikePattern(value string) string {
	value = strings.ReplaceAll(value, "!", "!!")
	value = strings.ReplaceAll(value, "%", "!%")
	return strings.ReplaceAll(value, "_", "!_")
}
