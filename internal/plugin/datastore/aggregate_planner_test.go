package datastore

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAggregatePlannerBuildsScopedSQLForEveryDialect(t *testing.T) {
	registry, table := registeredAggregateTable(t)
	scope := mustScope(t, pluginsdk.TrustedScope{
		SubjectID: "employee-1", TenantIDs: []string{"tenant-a"}, OrganizationIDs: []string{"org-a"}, OwnerIDs: []string{"employee-1"},
	})
	query := aggregateQuery(1)
	query.Filter = &pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: aggregateValue(pluginsdk.DataValueString, "active")}
	for _, test := range []struct {
		dialect    SQLDialect
		quote      string
		limitToken string
	}{
		{dialect: DialectSQLite, quote: `"`, limitToken: "?"},
		{dialect: DialectPostgreSQL, quote: `"`, limitToken: "$5"},
		{dialect: DialectMySQL, quote: "`", limitToken: "?"},
	} {
		t.Run(string(test.dialect), func(t *testing.T) {
			planner, err := NewAggregatePlanner(registry, fixedMutationScopeService{predicate: scope}, test.dialect)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := planner.Plan(context.Background(), "medical_oa", query)
			if err != nil {
				t.Fatal(err)
			}
			quoted := func(value string) string { return test.quote + value + test.quote }
			for _, fragment := range []string{
				"COUNT(*) AS " + quoted("__metric_0"),
				"SUM(" + quoted("quantity") + ") AS " + quoted("__metric_1"),
				"MIN(" + quoted("quantity") + ") AS " + quoted("__metric_2"),
				" FROM " + quoted(table.PhysicalName),
				quoted("tenant_id") + " = ",
				" GROUP BY " + quoted("status"),
				" ORDER BY " + quoted("status") + " ASC",
				" LIMIT " + test.limitToken,
			} {
				if !strings.Contains(plan.SQL, fragment) {
					t.Fatalf("sql=%s missing=%s", plan.SQL, fragment)
				}
			}
			if !reflect.DeepEqual(plan.Args[:4], []any{"tenant-a", "org-a", "employee-1", "active"}) || plan.Args[4] != 2 {
				t.Fatalf("args=%v", plan.Args)
			}
		})
	}
}

func TestAggregatePlannerRejectsUnauthorizedNullableAndInexactFields(t *testing.T) {
	registry, _ := registeredAggregateTable(t)
	scope := mustScope(t, pluginsdk.TrustedScope{
		SubjectID: "employee-1", TenantIDs: []string{"tenant-a"}, OrganizationIDs: []string{"org-a"}, OwnerIDs: []string{"employee-1"},
	})
	planner, err := NewAggregatePlanner(registry, fixedMutationScopeService{predicate: scope}, DialectSQLite)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		alter func(*pluginsdk.DataAggregateQuery)
		field string
		code  pluginsdk.DataStoreErrorCode
	}{
		{name: "not aggregatable", alter: func(q *pluginsdk.DataAggregateQuery) { q.Metrics[1].Field = "unlisted_quantity" }, field: "metrics[1].field", code: pluginsdk.DataStoreErrorInvalidRequest},
		{name: "not groupable", alter: func(q *pluginsdk.DataAggregateQuery) { q.GroupBy = []string{"unlisted_status"} }, field: "groupBy[0]", code: pluginsdk.DataStoreErrorInvalidRequest},
		{name: "nullable group", alter: func(q *pluginsdk.DataAggregateQuery) { q.GroupBy = []string{"nullable_status"} }, field: "groupBy[0]", code: pluginsdk.DataStoreErrorUnsupported},
		{name: "sqlite decimal", alter: func(q *pluginsdk.DataAggregateQuery) { q.Metrics[1].Field = "amount" }, field: "metrics[1].field", code: pluginsdk.DataStoreErrorUnsupported},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := aggregateQuery(10)
			test.alter(&query)
			_, err := planner.Plan(context.Background(), "medical_oa", query)
			assertStoreError(t, err, test.code, test.field)
		})
	}
}

func TestAggregateServiceReconcilesGroupsCursorsEmptySetsScopesAndTransactions(t *testing.T) {
	service, db, table := newAggregateFixture(t)
	ctx := context.Background()
	first, err := service.Aggregate(ctx, aggregateQuery(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Rows) != 1 || !first.HasMore || first.NextCursor == "" {
		t.Fatalf("first=%+v", first)
	}
	assertAggregateRow(t, first.Rows[0], "active", []string{"2", "15", "5"})

	secondQuery := aggregateQuery(1)
	secondQuery.Page.Cursor = first.NextCursor
	second, err := service.Aggregate(ctx, secondQuery)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Rows) != 1 || second.HasMore {
		t.Fatalf("second=%+v", second)
	}
	assertAggregateRow(t, second.Rows[0], "inactive", []string{"1", "2", "2"})

	ungrouped := aggregateQuery(1)
	ungrouped.GroupBy = nil
	ungrouped.Page = pluginsdk.DataPageRequest{Limit: 1}
	total, err := service.Aggregate(ctx, ungrouped)
	if err != nil {
		t.Fatal(err)
	}
	if len(total.Rows) != 1 {
		t.Fatalf("total=%+v", total)
	}
	for index, want := range []string{"3", "17", "2"} {
		if total.Rows[0].Values[index].Value != want {
			t.Fatalf("total values=%+v", total.Rows[0].Values)
		}
	}

	empty := ungrouped
	empty.Filter = &pluginsdk.DataFilter{Field: "status", Operator: pluginsdk.DataOperatorEqual, Value: aggregateValue(pluginsdk.DataValueString, "missing")}
	emptyPage, err := service.Aggregate(ctx, empty)
	if err != nil {
		t.Fatal(err)
	}
	if emptyPage.Rows[0].Values[0].Value != "0" || emptyPage.Rows[0].Values[1].Type != pluginsdk.DataValueNull || emptyPage.Rows[0].Values[2].Type != pluginsdk.DataValueNull {
		t.Fatalf("empty=%+v", emptyPage)
	}

	err = service.mutator.uow.Do(ctx, func(tx repository.Tx) error {
		transactionDB := storesql.ResolveDB(tx.Context(), db)
		if execErr := insertAggregateRow(transactionDB, table.PhysicalName, "inside-tx", "active", 8, "tenant-a", "org-a", "employee-1"); execErr != nil {
			return execErr
		}
		page, aggregateErr := service.Aggregate(tx.Context(), ungrouped)
		if aggregateErr != nil {
			return aggregateErr
		}
		if page.Rows[0].Values[0].Value != "4" || page.Rows[0].Values[1].Value != "25" {
			return fmt.Errorf("transaction aggregate=%+v", page)
		}
		return fmt.Errorf("rollback aggregate fixture")
	})
	if err == nil {
		t.Fatal("outer transaction unexpectedly committed")
	}
	after, err := service.Aggregate(ctx, ungrouped)
	if err != nil || after.Rows[0].Values[0].Value != "3" {
		t.Fatalf("after rollback=%+v err=%v", after, err)
	}
}

func TestAggregateServiceBoundsLargeGroupPagesAndReconcilesTotals(t *testing.T) {
	service, db, table := newAggregateFixture(t)
	for index := 0; index < 150; index++ {
		id := fmt.Sprintf("bulk-%03d", index)
		status := fmt.Sprintf("group-%03d", index)
		if err := insertAggregateRow(db, table.PhysicalName, id, status, 1, "tenant-a", "org-a", "employee-1"); err != nil {
			t.Fatal(err)
		}
	}

	query := aggregateQuery(pluginsdk.MaxDataAggregateGroups)
	seen := make(map[string]struct{}, 152)
	totalRows := 0
	totalCount := int64(0)
	for {
		page, err := service.Aggregate(context.Background(), query)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Rows) > pluginsdk.MaxDataAggregateGroups {
			t.Fatalf("page rows=%d", len(page.Rows))
		}
		for _, row := range page.Rows {
			group := row.Group["status"].Value
			if _, duplicate := seen[group]; duplicate {
				t.Fatalf("duplicate group %q", group)
			}
			seen[group] = struct{}{}
			count, parseErr := strconv.ParseInt(row.Values[0].Value, 10, 64)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			totalCount += count
			totalRows++
		}
		if !page.HasMore {
			break
		}
		query.Page.Cursor = page.NextCursor
	}
	if totalRows != 152 || totalCount != 153 {
		t.Fatalf("groups=%d count=%d", totalRows, totalCount)
	}
}

func registeredAggregateTable(t *testing.T) (*SchemaRegistry, ResolvedTable) {
	t.Helper()
	registry := NewSchemaRegistry()
	schema := validPluginSchema("medical_oa")
	schema.Tables[0].Fields = append(schema.Tables[0].Fields,
		FieldSchema{Name: "quantity", Type: pluginsdk.DataValueInteger, Filterable: true, Aggregatable: true},
		FieldSchema{Name: "unlisted_quantity", Type: pluginsdk.DataValueInteger, Filterable: true},
		FieldSchema{Name: "unlisted_status", Type: pluginsdk.DataValueString},
		FieldSchema{Name: "nullable_status", Type: pluginsdk.DataValueString, Nullable: true, Groupable: true},
		FieldSchema{Name: "amount", Type: pluginsdk.DataValueDecimal, Aggregatable: true},
	)
	for index := range schema.Tables[0].Fields {
		if schema.Tables[0].Fields[index].Name == "status" {
			schema.Tables[0].Fields[index].Groupable = true
		}
	}
	if _, err := registry.Register(schema); err != nil {
		t.Fatal(err)
	}
	table, err := registry.Resolve("medical_oa", "products")
	if err != nil {
		t.Fatal(err)
	}
	return registry, table
}

func newAggregateFixture(t *testing.T) (*Service, *gorm.DB, ResolvedTable) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "aggregate.db") + "?_busy_timeout=5000&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		t.Skipf("sqlite test requires cgo: %v", err)
	}
	if err = db.AutoMigrate(&gormrepo.PluginDataMutationModel{}, &mutationAuditRow{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	registry, table := registeredAggregateTable(t)
	ddl := fmt.Sprintf(`CREATE TABLE "%s" (
"id" TEXT PRIMARY KEY, "name" TEXT NOT NULL, "status" TEXT NOT NULL, "quantity" INTEGER NOT NULL,
"unlisted_quantity" INTEGER NOT NULL DEFAULT 0, "unlisted_status" TEXT NOT NULL DEFAULT '', "nullable_status" TEXT, "amount" NUMERIC NOT NULL DEFAULT 0,
"tenant_id" TEXT NOT NULL, "organization_id" TEXT NOT NULL, "owner_id" TEXT NOT NULL,
"version" INTEGER NOT NULL, "created_at" DATETIME NOT NULL, "updated_at" DATETIME NOT NULL
)`, table.PhysicalName)
	if err = db.Exec(ddl).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		id, status, tenant, organization, owner string
		quantity                                int64
	}{
		{id: "a-1", status: "active", quantity: 10, tenant: "tenant-a", organization: "org-a", owner: "employee-1"},
		{id: "a-2", status: "active", quantity: 5, tenant: "tenant-a", organization: "org-a", owner: "employee-1"},
		{id: "i-1", status: "inactive", quantity: 2, tenant: "tenant-a", organization: "org-a", owner: "employee-1"},
		{id: "foreign", status: "active", quantity: 100, tenant: "tenant-b", organization: "org-a", owner: "employee-1"},
	} {
		if err = insertAggregateRow(db, table.PhysicalName, row.id, row.status, row.quantity, row.tenant, row.organization, row.owner); err != nil {
			t.Fatal(err)
		}
	}
	scope := mustScope(t, pluginsdk.TrustedScope{
		SubjectID: "employee-1", TenantIDs: []string{"tenant-a"}, OrganizationIDs: []string{"org-a"}, OwnerIDs: []string{"employee-1"},
	})
	audit := &databaseMutationAudit{db: db}
	service, err := NewService(db, storesql.NewUnitOfWorkWithDB(db), registry, fixedMutationScopeService{predicate: scope}, audit, DialectSQLite, "medical_oa")
	if err != nil {
		t.Fatal(err)
	}
	return service, db, table
}

func insertAggregateRow(db *gorm.DB, table, id, status string, quantity int64, tenant, organization, owner string) error {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	return db.Table(table).Create(map[string]any{
		"id": id, "name": id, "status": status, "quantity": quantity,
		"tenant_id": tenant, "organization_id": organization, "owner_id": owner,
		"version": int64(1), "created_at": now, "updated_at": now,
	}).Error
}

func aggregateQuery(limit int) pluginsdk.DataAggregateQuery {
	return pluginsdk.DataAggregateQuery{
		Table: "products",
		Scope: pluginsdk.DataScopeIntent{Permission: pluginsdk.Permission{Resource: "medical_oa.products", Action: "read"}},
		Metrics: []pluginsdk.DataAggregateMetric{
			{Operation: pluginsdk.DataAggregateCount},
			{Operation: pluginsdk.DataAggregateSum, Field: "quantity"},
			{Operation: pluginsdk.DataAggregateMin, Field: "quantity"},
		},
		GroupBy: []string{"status"},
		Page:    pluginsdk.DataPageRequest{Limit: limit},
	}
}

func aggregateValue(kind pluginsdk.DataValueType, value string) *pluginsdk.DataValue {
	return &pluginsdk.DataValue{Type: kind, Value: value}
}

func assertAggregateRow(t *testing.T, row pluginsdk.DataAggregateRow, group string, values []string) {
	t.Helper()
	if row.Group["status"].Value != group || len(row.Values) != len(values) {
		t.Fatalf("row=%+v", row)
	}
	for index, want := range values {
		if row.Values[index].Value != want {
			t.Fatalf("row values=%+v want=%v", row.Values, values)
		}
	}
}
