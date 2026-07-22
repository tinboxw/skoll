package datastore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type recordingScopeService struct {
	predicate  pluginsdk.ScopePredicate
	err        error
	permission pluginsdk.Permission
}

func (s *recordingScopeService) Resolve(_ context.Context, permission pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	s.permission = permission
	return s.predicate, s.err
}

func TestQueryPlannerBuildsBoundScopedSQLForEveryDialect(t *testing.T) {
	trusted := mustScope(t, pluginsdk.TrustedScope{
		SubjectID: "employee-1", TenantIDs: []string{"tenant-b", "tenant-a"}, AllOwners: true,
		OrganizationIDs: []string{"org-b", "org-a"},
	})
	query := pluginsdk.DataQuery{
		Table:  "products",
		Fields: []string{"id", "name", "status"},
		Scope: pluginsdk.DataScopeIntent{
			Permission: pluginsdk.Permission{Resource: "medical_oa.product", Action: "read"},
			Filter: pluginsdk.ScopeFilter{
				TenantIDs: []string{"tenant-a", "tenant-outside"}, OrganizationIDs: []string{"org-a"},
			},
		},
		Filter: &pluginsdk.DataFilter{All: []pluginsdk.DataFilter{
			{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: dataValue(pluginsdk.DataValueString, "active")},
			{Any: []pluginsdk.DataFilter{
				{Field: "name", Operator: pluginsdk.DataOperatorContains, Value: dataValue(pluginsdk.DataValueString, "A_%!")},
				{Field: "id", Operator: pluginsdk.DataOperatorPrefix, Value: dataValue(pluginsdk.DataValueString, "prod")},
			}},
		}},
		Sort: []pluginsdk.DataSort{{Field: "name", Direction: pluginsdk.DataSortDescending}},
		Page: pluginsdk.DataPageRequest{Limit: 20},
	}

	tests := []struct {
		name         string
		dialect      SQLDialect
		quote        string
		placeholders []string
	}{
		{name: "sqlite", dialect: DialectSQLite, quote: `"`, placeholders: []string{"?", "?", "?", "?", "?", "?"}},
		{name: "postgresql", dialect: DialectPostgreSQL, quote: `"`, placeholders: []string{"$1", "$2", "$3", "$4", "$5", "$6"}},
		{name: "mysql", dialect: DialectMySQL, quote: "`", placeholders: []string{"?", "?", "?", "?", "?", "?"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry, registered := registeredProducts(t)
			scopes := &recordingScopeService{predicate: trusted}
			planner, err := NewQueryPlanner(registry, scopes, test.dialect)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := planner.Plan(context.Background(), "medical_oa", query)
			if err != nil {
				t.Fatal(err)
			}

			q := func(value string) string { return test.quote + value + test.quote }
			p := test.placeholders
			expected := "SELECT " + strings.Join([]string{q("id"), q("name"), q("status"), q("version")}, ", ") +
				" FROM " + q(registered.Tables[0].PhysicalName) +
				" WHERE " + q("tenant_id") + " = " + p[0] + " AND " + q("organization_id") + " = " + p[1] +
				" AND (" + q("status") + " = " + p[2] + " AND (" + q("name") + " LIKE " + p[3] + " ESCAPE '!' OR " + q("id") + " LIKE " + p[4] + " ESCAPE '!'))" +
				" ORDER BY " + q("name") + " DESC, " + q("id") + " DESC LIMIT " + p[5]
			if plan.SQL != expected {
				t.Fatalf("SQL:\n%s\nwant:\n%s", plan.SQL, expected)
			}
			wantArgs := []any{"tenant-a", "org-a", "active", "%A!_!%!!%", "prod%", 21}
			if !reflect.DeepEqual(plan.Args, wantArgs) {
				t.Fatalf("args=%#v want=%#v", plan.Args, wantArgs)
			}
			if strings.Contains(plan.SQL, "tenant-a") || strings.Contains(plan.SQL, "active") || strings.Contains(plan.SQL, "A_%!") {
				t.Fatalf("untrusted value was interpolated into SQL: %s", plan.SQL)
			}
			if !reflect.DeepEqual(plan.Sort, []pluginsdk.DataSort{{Field: "name", Direction: pluginsdk.DataSortDescending}, {Field: "id", Direction: pluginsdk.DataSortDescending}}) {
				t.Fatalf("sort=%+v", plan.Sort)
			}
			if !reflect.DeepEqual(plan.ScanFields, []string{"id", "name", "status", "version"}) || plan.Limit != 20 || plan.FetchLimit != 21 {
				t.Fatalf("plan metadata=%+v", plan)
			}
			if scopes.permission != query.Scope.Permission {
				t.Fatalf("resolved permission=%+v", scopes.permission)
			}
		})
	}
}

func TestQueryPlannerRejectsScopeBroadeningAndIncompleteTrustedScope(t *testing.T) {
	registry, _ := registeredProducts(t)
	query := basicQuery()
	query.Scope.Filter.TenantIDs = []string{"tenant-outside"}

	tests := []struct {
		name      string
		predicate pluginsdk.ScopePredicate
		err       error
		field     string
	}{
		{name: "outside trusted scope", predicate: mustScope(t, pluginsdk.TrustedScope{SubjectID: "employee-1", TenantIDs: []string{"tenant-a"}, AllOwners: true, AllOrganizations: true}), field: "scope.filter"},
		{name: "denied predicate", predicate: pluginsdk.NewDeniedScopePredicate("employee-1"), field: "scope.filter"},
		{name: "zero predicate", predicate: pluginsdk.ScopePredicate{}, field: "scope"},
		{name: "resolver failure", err: errors.New("permission denied"), field: "scope.permission"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			planner, err := NewQueryPlanner(registry, &recordingScopeService{predicate: test.predicate, err: test.err}, DialectSQLite)
			if err != nil {
				t.Fatal(err)
			}
			_, err = planner.Plan(context.Background(), "medical_oa", query)
			assertStoreError(t, err, pluginsdk.DataStoreErrorForbidden, test.field)
		})
	}
}

func TestQueryPlannerRejectsForeignPermissionAndExcessiveParameters(t *testing.T) {
	registry, _ := registeredProducts(t)
	query := basicQuery()
	query.Scope.Permission.Resource = "other_plugin.product"
	planner, err := NewQueryPlanner(registry, &recordingScopeService{predicate: allScope(t)}, DialectSQLite)
	if err != nil {
		t.Fatal(err)
	}
	_, err = planner.Plan(context.Background(), "medical_oa", query)
	assertStoreError(t, err, pluginsdk.DataStoreErrorForbidden, "scope.permission.resource")

	ids := make([]string, MaxQueryParameters)
	for index := range ids {
		ids[index] = fmt.Sprintf("tenant-%04d", index)
	}
	query = basicQuery()
	planner, err = NewQueryPlanner(registry, &recordingScopeService{predicate: mustScope(t, pluginsdk.TrustedScope{
		SubjectID: "employee-1", TenantIDs: ids, AllOwners: true, AllOrganizations: true,
	})}, DialectSQLite)
	if err != nil {
		t.Fatal(err)
	}
	_, err = planner.Plan(context.Background(), "medical_oa", query)
	assertStoreError(t, err, pluginsdk.DataStoreErrorLimitExceeded, "query")
}

func TestQueryPlannerRejectsUndeclaredAndUnsupportedFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PluginSchema, *pluginsdk.DataQuery)
		code   pluginsdk.DataStoreErrorCode
		field  string
	}{
		{name: "undeclared selection", mutate: func(_ *PluginSchema, query *pluginsdk.DataQuery) { query.Fields = []string{"missing"} }, code: pluginsdk.DataStoreErrorInvalidRequest, field: "fields[0]"},
		{name: "undeclared filter", mutate: func(_ *PluginSchema, query *pluginsdk.DataQuery) {
			query.Filter = &pluginsdk.DataFilter{Field: "missing", Operator: pluginsdk.DataOperatorEqual, Value: dataValue(pluginsdk.DataValueString, "x")}
		}, code: pluginsdk.DataStoreErrorInvalidRequest, field: "filter.field"},
		{name: "non filterable", mutate: func(schema *PluginSchema, query *pluginsdk.DataQuery) {
			schema.Tables[0].Fields[2].Filterable = false
			query.Filter = &pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: dataValue(pluginsdk.DataValueString, "x")}
		}, code: pluginsdk.DataStoreErrorInvalidRequest, field: "filter.field"},
		{name: "wrong value type", mutate: func(_ *PluginSchema, query *pluginsdk.DataQuery) {
			query.Filter = &pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: dataValue(pluginsdk.DataValueInteger, "1")}
		}, code: pluginsdk.DataStoreErrorInvalidRequest, field: "filter.value.type"},
		{name: "non sortable", mutate: func(schema *PluginSchema, query *pluginsdk.DataQuery) {
			schema.Tables[0].Fields[2].Sortable = false
			query.Sort = []pluginsdk.DataSort{{Field: "status", Direction: pluginsdk.DataSortAscending}}
		}, code: pluginsdk.DataStoreErrorInvalidRequest, field: "sort[0].field"},
		{name: "nullable sort", mutate: func(schema *PluginSchema, query *pluginsdk.DataQuery) {
			schema.Tables[0].Fields = append(schema.Tables[0].Fields, FieldSchema{Name: "notes", Type: pluginsdk.DataValueString, Nullable: true, Mutable: true, Sortable: true})
			query.Sort = []pluginsdk.DataSort{{Field: "notes", Direction: pluginsdk.DataSortAscending}}
		}, code: pluginsdk.DataStoreErrorUnsupported, field: "sort[0].field"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schema := validPluginSchema("medical_oa")
			query := basicQuery()
			test.mutate(&schema, &query)
			registry := NewSchemaRegistry()
			if _, err := registry.Register(schema); err != nil {
				t.Fatal(err)
			}
			planner, err := NewQueryPlanner(registry, &recordingScopeService{predicate: allScope(t)}, DialectSQLite)
			if err != nil {
				t.Fatal(err)
			}
			_, err = planner.Plan(context.Background(), "medical_oa", query)
			assertStoreError(t, err, test.code, test.field)
		})
	}
}

func TestQueryPlannerCursorUsesStableKeysetPagination(t *testing.T) {
	registry, registered := registeredProducts(t)
	planner, err := NewQueryPlanner(registry, &recordingScopeService{predicate: allScope(t)}, DialectPostgreSQL)
	if err != nil {
		t.Fatal(err)
	}
	query := basicQuery()
	query.Fields = []string{"name"}
	query.Sort = []pluginsdk.DataSort{{Field: FieldUpdatedAt, Direction: pluginsdk.DataSortAscending}}
	query.Page.Limit = 10
	first, err := planner.Plan(context.Background(), "medical_oa", query)
	if err != nil {
		t.Fatal(err)
	}
	timestamp := "2026-07-22T10:11:12.123Z"
	cursor, err := first.EncodeCursor(map[string]pluginsdk.DataValue{
		FieldUpdatedAt: {Type: pluginsdk.DataValueTimestamp, Value: timestamp},
		"id":           {Type: pluginsdk.DataValueString, Value: "product-100"},
	})
	if err != nil {
		t.Fatal(err)
	}
	query.Page.Cursor = cursor
	second, err := planner.Plan(context.Background(), "medical_oa", query)
	if err != nil {
		t.Fatal(err)
	}
	expectedSQL := fmt.Sprintf(`SELECT "name", "version", "updated_at", "id" FROM "%s" WHERE (("updated_at" > $1) OR ("updated_at" = $2 AND "id" > $3)) ORDER BY "updated_at" ASC, "id" ASC LIMIT $4`, registered.Tables[0].PhysicalName)
	if second.SQL != expectedSQL {
		t.Fatalf("SQL:\n%s\nwant:\n%s", second.SQL, expectedSQL)
	}
	wantTime, _ := time.Parse(time.RFC3339Nano, timestamp)
	if !reflect.DeepEqual(second.Args, []any{wantTime, wantTime, "product-100", 11}) {
		t.Fatalf("cursor args=%#v", second.Args)
	}
	if !reflect.DeepEqual(second.ScanFields, []string{"name", "version", "updated_at", "id"}) {
		t.Fatalf("scan fields=%v", second.ScanFields)
	}

	query.Sort = []pluginsdk.DataSort{{Field: "name", Direction: pluginsdk.DataSortAscending}}
	_, err = planner.Plan(context.Background(), "medical_oa", query)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "page.cursor")
}

func TestQueryPlannerRejectsMalformedCursorAndUnsupportedDialect(t *testing.T) {
	registry, _ := registeredProducts(t)
	if _, err := NewQueryPlanner(registry, &recordingScopeService{predicate: allScope(t)}, SQLDialect("oracle")); err == nil {
		t.Fatal("unsupported dialect succeeded")
	} else {
		assertStoreError(t, err, pluginsdk.DataStoreErrorUnsupported, "dialect")
	}
	planner, err := NewQueryPlanner(registry, &recordingScopeService{predicate: allScope(t)}, DialectSQLite)
	if err != nil {
		t.Fatal(err)
	}
	query := basicQuery()
	query.Page.Cursor = "not-base64"
	_, err = planner.Plan(context.Background(), "medical_oa", query)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "page.cursor")
}

func registeredProducts(t *testing.T) (*SchemaRegistry, RegisteredSchema) {
	t.Helper()
	registry := NewSchemaRegistry()
	registered, err := registry.Register(validPluginSchema("medical_oa"))
	if err != nil {
		t.Fatal(err)
	}
	return registry, registered
}

func basicQuery() pluginsdk.DataQuery {
	return pluginsdk.DataQuery{
		Table: "products", Fields: []string{"id", "name"},
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "medical_oa.product", Action: "read"}},
		Sort:  []pluginsdk.DataSort{{Field: "id", Direction: pluginsdk.DataSortAscending}},
		Page:  pluginsdk.DataPageRequest{Limit: 50},
	}
}

func mustScope(t *testing.T, scope pluginsdk.TrustedScope) pluginsdk.ScopePredicate {
	t.Helper()
	predicate, err := pluginsdk.NewScopePredicate(scope)
	if err != nil {
		t.Fatal(err)
	}
	return predicate
}

func allScope(t *testing.T) pluginsdk.ScopePredicate {
	t.Helper()
	return mustScope(t, pluginsdk.TrustedScope{SubjectID: "super-admin", AllTenants: true, AllOwners: true, AllOrganizations: true})
}

func dataValue(kind pluginsdk.DataValueType, value string) *pluginsdk.DataValue {
	return &pluginsdk.DataValue{Type: kind, Value: value}
}
