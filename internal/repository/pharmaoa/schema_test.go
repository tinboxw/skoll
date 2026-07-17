package pharmaoa

import (
	"slices"
	"testing"
)

func TestSchemaBaselineValidatesOwnershipConstraintsAndIndexes(t *testing.T) {
	if err := ValidateSchemaBaseline(); err != nil {
		t.Fatalf("validate schema baseline: %v", err)
	}
	tables := SchemaBaseline()
	if len(tables) < 25 {
		t.Fatalf("schema baseline has %d tables, want at least 25", len(tables))
	}
	byName := make(map[string]TableSpec, len(tables))
	for _, table := range tables {
		byName[table.Name] = table
	}
	for _, name := range []string{
		"pharma_oa_employees", "pharma_oa_products", "pharma_oa_suppliers", "pharma_oa_customers", "pharma_oa_warehouses",
		"pharma_oa_stock_balances", "pharma_oa_stock_ledger", "pharma_oa_purchase_requests", "pharma_oa_sales_outbounds",
		"pharma_oa_contracts", "pharma_oa_quality_complaints", "pharma_oa_drug_recalls", "pharma_oa_customer_follow_ups",
	} {
		if _, exists := byName[name]; !exists {
			t.Fatalf("required schema table missing: %s", name)
		}
	}
	if !byName["pharma_oa_stock_ledger"].Immutable {
		t.Fatal("stock ledger must be immutable")
	}
	for _, name := range []string{"pharma_oa_stock_ledger", "pharma_oa_cold_chain_records"} {
		table := byName[name]
		if !slices.Equal(table.AuditColumns, []string{"created_at", "created_by"}) {
			t.Fatalf("immutable table %s audit columns = %v", name, table.AuditColumns)
		}
		if slices.Contains(table.Columns, "updated_at") || slices.Contains(table.Columns, "updated_by") {
			t.Fatalf("immutable table %s exposes update audit columns", name)
		}
	}
	if byName["pharma_oa_stock_balances"].TransactionGroup != "inventory" || byName["pharma_oa_sales_outbounds"].TransactionGroup != "inventory" {
		t.Fatal("inventory balance and outbound must share the inventory transaction group")
	}
	assertUniqueIndex(t, byName["pharma_oa_products"], "approval_number")
	assertUniqueIndex(t, byName["pharma_oa_stock_balances"], "product_id", "warehouse_id", "area_id", "location_id", "batch_id")
	assertUniqueIndex(t, byName["pharma_oa_stock_ledger"], "idempotency_key")
	assertUniqueIndex(t, byName["pharma_oa_purchase_inbounds"], "idempotency_key")
	assertUniqueIndex(t, byName["pharma_oa_sales_outbounds"], "idempotency_key")
}

func TestMigrationPlanSupportsAllDialectsAndReversesOrder(t *testing.T) {
	for _, dialect := range []Dialect{DialectMySQL, DialectPostgreSQL, DialectSQLite} {
		plan, err := BuildMigrationPlan(dialect)
		if err != nil {
			t.Fatalf("build %s migration plan: %v", dialect, err)
		}
		if plan.UninstallPolicy != UninstallPolicyRetain {
			t.Fatalf("%s uninstall policy = %s", dialect, plan.UninstallPolicy)
		}
		if len(plan.Up) == 0 || len(plan.Up) != len(plan.Down) {
			t.Fatalf("%s migration plan lengths: up=%d down=%d", dialect, len(plan.Up), len(plan.Down))
		}
		positions := map[string]int{}
		for index, step := range plan.Up {
			positions[step.Table] = index
		}
		if !(positions["pharma_oa_stock_batches"] < positions["pharma_oa_stock_balances"] && positions["pharma_oa_stock_balances"] < positions["pharma_oa_stock_ledger"]) {
			t.Fatalf("%s inventory migration dependency order is invalid", dialect)
		}
		for index, step := range plan.Down {
			want := plan.Up[len(plan.Up)-1-index].Table
			if step.Table != want || step.Action != "drop" || !step.Destructive {
				t.Fatalf("%s rollback step %d = %+v, want destructive drop of %s", dialect, index, step, want)
			}
		}
	}
}

func TestMigrationPlanRejectsUnsupportedDialect(t *testing.T) {
	if _, err := BuildMigrationPlan(Dialect("oracle")); err == nil {
		t.Fatal("unsupported schema dialect must fail")
	}
}

func TestSchemaBaselineReturnsDefensiveCopy(t *testing.T) {
	first := SchemaBaseline()
	first[0].Columns[0] = "changed"
	first[0].Indexes[0].Columns[0] = "changed"
	second := SchemaBaseline()
	if second[0].Columns[0] == "changed" || second[0].Indexes[0].Columns[0] == "changed" {
		t.Fatal("schema baseline was mutated through returned slice")
	}
}

func assertUniqueIndex(t *testing.T, table TableSpec, columns ...string) {
	t.Helper()
	for _, idx := range table.Indexes {
		if idx.Unique && slices.Equal(idx.Columns, columns) {
			return
		}
	}
	t.Fatalf("table %s has no unique index on %v", table.Name, columns)
}
