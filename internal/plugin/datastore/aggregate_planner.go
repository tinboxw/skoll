package datastore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type AggregatePlan struct {
	Dialect      SQLDialect
	SQL          string
	Args         []any
	Metrics      []pluginsdk.DataAggregateMetric
	GroupBy      []string
	Limit        int
	FetchLimit   int
	PluginID     string
	LogicalTable string

	table ResolvedTable
}

type AggregatePlanner struct {
	registry *SchemaRegistry
	scopes   pluginsdk.DataScopeService
	dialect  SQLDialect
}

type aggregateCursor struct {
	PluginID string                `json:"pluginId"`
	Table    string                `json:"table"`
	GroupBy  []string              `json:"groupBy"`
	Values   []pluginsdk.DataValue `json:"values"`
}

func NewAggregatePlanner(registry *SchemaRegistry, scopes pluginsdk.DataScopeService, dialect SQLDialect) (*AggregatePlanner, error) {
	if registry == nil || scopes == nil {
		return nil, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "planner", "aggregate planner services are incomplete", true)
	}
	parsed, err := ParseSQLDialect(string(dialect))
	if err != nil {
		return nil, err
	}
	return &AggregatePlanner{registry: registry, scopes: scopes, dialect: parsed}, nil
}

func (p *AggregatePlanner) Plan(ctx context.Context, pluginID string, query pluginsdk.DataAggregateQuery) (AggregatePlan, error) {
	if p == nil || p.registry == nil || p.scopes == nil {
		return AggregatePlan{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "planner", "aggregate planner is unavailable", true)
	}
	if ctx == nil {
		return AggregatePlan{}, invalidSchema("context", "aggregate context is required")
	}
	if err := query.Validate(); err != nil {
		return AggregatePlan{}, err
	}
	pluginID = strings.TrimSpace(pluginID)
	if err := validateQueryPermission(pluginID, query.Scope.Permission); err != nil {
		return AggregatePlan{}, err
	}
	table, err := p.registry.Resolve(pluginID, query.Table)
	if err != nil {
		return AggregatePlan{}, err
	}
	if err = validateAggregateSchema(p.dialect, table, query); err != nil {
		return AggregatePlan{}, err
	}
	trusted, err := p.scopes.Resolve(ctx, query.Scope.Permission)
	if err != nil {
		return AggregatePlan{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope.permission", "trusted data scope could not be resolved", false)
	}
	trusted = trusted.Constrain(query.Scope.Filter)
	if err = validateTrustedScope(trusted); err != nil {
		return AggregatePlan{}, err
	}

	builder := &sqlPlanBuilder{dialect: p.dialect}
	clauses, err := builder.scopeClauses(trusted)
	if err != nil {
		return AggregatePlan{}, err
	}
	if query.Filter != nil {
		clause, compileErr := builder.filterClause(table, *query.Filter, "filter")
		if compileErr != nil {
			return AggregatePlan{}, compileErr
		}
		clauses = append(clauses, clause)
	}
	if query.Page.Cursor != "" {
		cursor, decodeErr := decodeAggregateCursor(query.Page.Cursor)
		if decodeErr != nil {
			return AggregatePlan{}, decodeErr
		}
		clause, compileErr := builder.aggregateCursorClause(table, query.GroupBy, cursor)
		if compileErr != nil {
			return AggregatePlan{}, compileErr
		}
		clauses = append(clauses, clause)
	}

	selections := make([]string, 0, len(query.GroupBy)+len(query.Metrics))
	groups := make([]string, len(query.GroupBy))
	for index, field := range query.GroupBy {
		groups[index] = quoteSQLIdentifier(p.dialect, field)
		selections = append(selections, groups[index])
	}
	for index, metric := range query.Metrics {
		expression := "COUNT(*)"
		if metric.Operation != pluginsdk.DataAggregateCount {
			expression = strings.ToUpper(string(metric.Operation)) + "(" + quoteSQLIdentifier(p.dialect, metric.Field) + ")"
		}
		selections = append(selections, expression+" AS "+quoteSQLIdentifier(p.dialect, fmt.Sprintf("__metric_%d", index)))
	}

	fetchLimit := 1
	if len(query.GroupBy) > 0 {
		fetchLimit = query.Page.Limit + 1
	}
	if len(builder.args)+1 > MaxQueryParameters {
		return AggregatePlan{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "query", "aggregate query exceeds the bound parameter limit", false)
	}
	var statement strings.Builder
	statement.WriteString("SELECT ")
	statement.WriteString(strings.Join(selections, ", "))
	statement.WriteString(" FROM ")
	statement.WriteString(quoteSQLIdentifier(p.dialect, table.PhysicalName))
	if len(clauses) > 0 {
		statement.WriteString(" WHERE ")
		statement.WriteString(strings.Join(clauses, " AND "))
	}
	if len(groups) > 0 {
		statement.WriteString(" GROUP BY ")
		statement.WriteString(strings.Join(groups, ", "))
		statement.WriteString(" ORDER BY ")
		for index, group := range groups {
			if index > 0 {
				statement.WriteString(", ")
			}
			statement.WriteString(group)
			statement.WriteString(" ASC")
		}
	}
	statement.WriteString(" LIMIT ")
	statement.WriteString(builder.bind(fetchLimit))

	return AggregatePlan{
		Dialect: p.dialect, SQL: statement.String(), Args: append([]any(nil), builder.args...),
		Metrics: append([]pluginsdk.DataAggregateMetric(nil), query.Metrics...), GroupBy: append([]string(nil), query.GroupBy...),
		Limit: query.Page.Limit, FetchLimit: fetchLimit, PluginID: table.PluginID, LogicalTable: table.LogicalName,
		table: cloneResolvedTable(table),
	}, nil
}

func validateAggregateSchema(dialect SQLDialect, table ResolvedTable, query pluginsdk.DataAggregateQuery) error {
	for index, metric := range query.Metrics {
		if metric.Operation == pluginsdk.DataAggregateCount {
			continue
		}
		field, exists := table.Fields[metric.Field]
		path := fmt.Sprintf("metrics[%d].field", index)
		if !exists || field.HostManaged || !field.Aggregatable {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, path, "field is not declared as aggregatable", false)
		}
		if field.Type != pluginsdk.DataValueInteger && field.Type != pluginsdk.DataValueDecimal {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, path, "aggregate field is not numeric", false)
		}
		if dialect == DialectSQLite && field.Type == pluginsdk.DataValueDecimal {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path, "sqlite cannot execute exact decimal aggregates", false)
		}
	}
	for index, name := range query.GroupBy {
		field, exists := table.Fields[name]
		path := fmt.Sprintf("groupBy[%d]", index)
		if !exists || field.HostManaged || !field.Groupable {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, path, "field is not declared as groupable", false)
		}
		if field.Nullable {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path, "nullable group fields are not supported", false)
		}
		if dialect == DialectSQLite && field.Type == pluginsdk.DataValueDecimal {
			return pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, path, "sqlite cannot group exact decimal values", false)
		}
	}
	return nil
}

func (p AggregatePlan) EncodeCursor(group map[string]pluginsdk.DataValue) (string, error) {
	if p.PluginID == "" || p.LogicalTable == "" || len(p.GroupBy) == 0 || len(p.table.Fields) == 0 {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "cursor", "aggregate plan cannot encode a cursor", false)
	}
	cursor := aggregateCursor{
		PluginID: p.PluginID, Table: p.LogicalTable, GroupBy: append([]string(nil), p.GroupBy...),
		Values: make([]pluginsdk.DataValue, len(p.GroupBy)),
	}
	for index, field := range p.GroupBy {
		value, exists := group[field]
		if !exists {
			return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "cursor.values."+field, "aggregate cursor value is missing", false)
		}
		if err := validateQueryValue(p.table.Fields[field], value, "cursor.values."+field, false); err != nil {
			return "", err
		}
		cursor.Values[index] = value
	}
	raw, err := json.Marshal(cursor)
	if err != nil {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "cursor", "aggregate cursor could not be encoded", false)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	if len(encoded) > pluginsdk.MaxDataCursorBytes {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorLimitExceeded, "cursor", "aggregate cursor exceeds size limit", false)
	}
	return encoded, nil
}

func (b *sqlPlanBuilder) aggregateCursorClause(table ResolvedTable, groupBy []string, cursor aggregateCursor) (string, error) {
	if cursor.PluginID != table.PluginID || cursor.Table != table.LogicalName || !reflect.DeepEqual(cursor.GroupBy, groupBy) || len(cursor.Values) != len(groupBy) {
		return "", pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "page.cursor", "cursor does not match the aggregate query", false)
	}
	converted := make([]any, len(cursor.Values))
	for index, value := range cursor.Values {
		item, err := convertQueryValue(table.Fields[groupBy[index]], value, fmt.Sprintf("page.cursor.values[%d]", index), false)
		if err != nil {
			return "", err
		}
		converted[index] = item
	}
	branches := make([]string, len(groupBy))
	for index, field := range groupBy {
		parts := make([]string, 0, index+1)
		for previous := 0; previous < index; previous++ {
			parts = append(parts, quoteSQLIdentifier(b.dialect, groupBy[previous])+" = "+b.bind(converted[previous]))
		}
		parts = append(parts, quoteSQLIdentifier(b.dialect, field)+" > "+b.bind(converted[index]))
		branches[index] = "(" + strings.Join(parts, " AND ") + ")"
	}
	return "(" + strings.Join(branches, " OR ") + ")", nil
}

func decodeAggregateCursor(encoded string) (aggregateCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return aggregateCursor{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "page.cursor", "cursor encoding is invalid", false)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var cursor aggregateCursor
	if err = decoder.Decode(&cursor); err != nil {
		return aggregateCursor{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "page.cursor", "cursor payload is invalid", false)
	}
	if err = decoder.Decode(&struct{}{}); err != io.EOF {
		return aggregateCursor{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "page.cursor", "cursor contains trailing data", false)
	}
	return cursor, nil
}
