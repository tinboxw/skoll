package pharmaoa

import (
	"fmt"
	"strings"
)

type Dialect string

const (
	DialectMySQL      Dialect = "mysql"
	DialectPostgreSQL Dialect = "postgres"
	DialectSQLite     Dialect = "sqlite"

	UninstallPolicyRetain = "retain"
)

type IndexSpec struct {
	Name    string
	Columns []string
	Unique  bool
}

type TableSpec struct {
	Name             string
	Owner            string
	Columns          []string
	PrimaryKey       []string
	Indexes          []IndexSpec
	AuditColumns     []string
	Immutable        bool
	TransactionGroup string
}

type MigrationStep struct {
	Order       int
	Table       string
	Action      string
	Destructive bool
}

type MigrationPlan struct {
	Dialect         Dialect
	UninstallPolicy string
	Up              []MigrationStep
	Down            []MigrationStep
}

func SchemaBaseline() []TableSpec {
	return cloneTableSpecs(schemaBaseline)
}

func BuildMigrationPlan(dialect Dialect) (MigrationPlan, error) {
	switch dialect {
	case DialectMySQL, DialectPostgreSQL, DialectSQLite:
	default:
		return MigrationPlan{}, fmt.Errorf("unsupported pharma OA schema dialect: %s", dialect)
	}
	if err := ValidateSchemaBaseline(); err != nil {
		return MigrationPlan{}, err
	}
	tables := SchemaBaseline()
	plan := MigrationPlan{Dialect: dialect, UninstallPolicy: UninstallPolicyRetain}
	for index, table := range tables {
		plan.Up = append(plan.Up, MigrationStep{Order: index + 1, Table: table.Name, Action: "create"})
	}
	for index := len(tables) - 1; index >= 0; index-- {
		plan.Down = append(plan.Down, MigrationStep{Order: len(tables) - index, Table: tables[index].Name, Action: "drop", Destructive: true})
	}
	return plan, nil
}

func ValidateSchemaBaseline() error {
	seenTables := map[string]struct{}{}
	seenIndexes := map[string]struct{}{}
	for _, table := range schemaBaseline {
		if !strings.HasPrefix(table.Name, "pharma_oa_") {
			return fmt.Errorf("pharma OA table is outside namespace: %s", table.Name)
		}
		if _, exists := seenTables[table.Name]; exists {
			return fmt.Errorf("duplicate pharma OA table: %s", table.Name)
		}
		seenTables[table.Name] = struct{}{}
		columns := stringSet(table.Columns)
		if strings.TrimSpace(table.Owner) == "" || len(table.PrimaryKey) == 0 {
			return fmt.Errorf("pharma OA table ownership or primary key missing: %s", table.Name)
		}
		for _, column := range append(append([]string{}, table.PrimaryKey...), table.AuditColumns...) {
			if _, exists := columns[column]; !exists {
				return fmt.Errorf("pharma OA table %s references unknown column %s", table.Name, column)
			}
		}
		if len(table.AuditColumns) == 0 {
			return fmt.Errorf("pharma OA table audit columns missing: %s", table.Name)
		}
		for _, index := range table.Indexes {
			if strings.TrimSpace(index.Name) == "" || len(index.Columns) == 0 {
				return fmt.Errorf("pharma OA table index incomplete: %s", table.Name)
			}
			if _, exists := seenIndexes[index.Name]; exists {
				return fmt.Errorf("duplicate pharma OA index: %s", index.Name)
			}
			seenIndexes[index.Name] = struct{}{}
			for _, column := range index.Columns {
				if _, exists := columns[column]; !exists {
					return fmt.Errorf("pharma OA index %s references unknown column %s", index.Name, column)
				}
			}
		}
	}
	return nil
}

func table(name, owner string, businessColumns []string, indexes []IndexSpec, group string) TableSpec {
	columns := append([]string{"id"}, businessColumns...)
	columns = append(columns, "created_at", "updated_at", "created_by", "updated_by")
	return TableSpec{
		Name: name, Owner: owner, Columns: columns, PrimaryKey: []string{"id"}, Indexes: indexes,
		AuditColumns: []string{"created_at", "updated_at", "created_by", "updated_by"}, TransactionGroup: group,
	}
}

func immutableTable(name, owner string, businessColumns []string, indexes []IndexSpec, group string) TableSpec {
	columns := append([]string{"id"}, businessColumns...)
	columns = append(columns, "created_at", "created_by")
	return TableSpec{
		Name: name, Owner: owner, Columns: columns, PrimaryKey: []string{"id"}, Indexes: indexes,
		AuditColumns: []string{"created_at", "created_by"}, Immutable: true, TransactionGroup: group,
	}
}

func index(name string, unique bool, columns ...string) IndexSpec {
	return IndexSpec{Name: name, Columns: columns, Unique: unique}
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func cloneTableSpecs(in []TableSpec) []TableSpec {
	out := make([]TableSpec, len(in))
	for i, item := range in {
		out[i] = item
		out[i].Columns = append([]string(nil), item.Columns...)
		out[i].PrimaryKey = append([]string(nil), item.PrimaryKey...)
		out[i].AuditColumns = append([]string(nil), item.AuditColumns...)
		out[i].Indexes = make([]IndexSpec, len(item.Indexes))
		for j, idx := range item.Indexes {
			out[i].Indexes[j] = idx
			out[i].Indexes[j].Columns = append([]string(nil), idx.Columns...)
		}
	}
	return out
}

var schemaBaseline = []TableSpec{
	table("pharma_oa_employees", "master.employee", []string{"code", "name", "department_id", "position_id", "phone", "email", "status", "leave_reason", "certificates_json"}, []IndexSpec{index("uk_pharma_employees_code", true, "code"), index("idx_pharma_employees_org_status", false, "department_id", "status"), index("idx_pharma_employees_name", false, "name")}, "master"),
	table("pharma_oa_products", "master.product", []string{"code", "name", "specification", "dosage_form", "manufacturer", "approval_number", "status", "disable_reason", "temperature_json"}, []IndexSpec{index("uk_pharma_products_code", true, "code"), index("uk_pharma_products_approval", true, "approval_number"), index("idx_pharma_products_status_name", false, "status", "name")}, "master"),
	table("pharma_oa_suppliers", "master.supplier", []string{"code", "name", "rating", "status", "disable_reason", "contacts_json", "qualifications_json"}, []IndexSpec{index("uk_pharma_suppliers_code", true, "code"), index("idx_pharma_suppliers_status_name", false, "status", "name")}, "master"),
	table("pharma_oa_customers", "master.customer", []string{"code", "name", "region", "status", "disable_reason", "rating", "owner_id", "organization_id", "contacts_json", "qualifications_json"}, []IndexSpec{index("uk_pharma_customers_code", true, "code"), index("idx_pharma_customers_scope_status", false, "organization_id", "owner_id", "status"), index("idx_pharma_customers_name", false, "name")}, "master"),
	table("pharma_oa_warehouses", "master.warehouse", []string{"code", "name", "region", "status", "disable_reason", "temperature_json", "areas_json"}, []IndexSpec{index("uk_pharma_warehouses_code", true, "code"), index("idx_pharma_warehouses_status_name", false, "status", "name")}, "master"),
	table("pharma_oa_stock_batches", "inventory.batch", []string{"product_id", "batch_no", "production_date", "expires_at"}, []IndexSpec{index("uk_pharma_batches_product_no", true, "product_id", "batch_no"), index("idx_pharma_batches_expiry", false, "expires_at", "product_id")}, "inventory"),
	table("pharma_oa_stock_balances", "inventory.balance", []string{"product_id", "warehouse_id", "area_id", "location_id", "batch_id", "quantity", "locked_quantity", "version"}, []IndexSpec{index("uk_pharma_balance_position", true, "product_id", "warehouse_id", "area_id", "location_id", "batch_id"), index("idx_pharma_balance_warehouse_product", false, "warehouse_id", "product_id"), index("idx_pharma_balance_batch", false, "batch_id")}, "inventory"),
	immutableTable("pharma_oa_stock_ledger", "inventory.ledger", []string{"operation", "reference_id", "product_id", "warehouse_id", "area_id", "location_id", "batch_id", "quantity_delta", "balance_after", "occurred_at", "idempotency_key"}, []IndexSpec{index("uk_pharma_ledger_idempotency", true, "idempotency_key"), index("idx_pharma_ledger_reference_time", false, "reference_id", "occurred_at"), index("idx_pharma_ledger_position_time", false, "product_id", "warehouse_id", "batch_id", "occurred_at")}, "inventory"),
	table("pharma_oa_stock_locks", "inventory.lock", []string{"balance_id", "quantity", "reason", "status", "released_at"}, []IndexSpec{index("idx_pharma_locks_balance_status", false, "balance_id", "status")}, "inventory"),
	table("pharma_oa_purchase_requests", "purchase.request", []string{"number", "supplier_id", "requester_id", "approver_id", "reason", "lines_json", "total_amount", "status", "workflow_instance_id", "purchase_order_id"}, []IndexSpec{index("uk_pharma_purchase_requests_number", true, "number"), index("idx_pharma_purchase_requests_status_approver", false, "status", "approver_id"), index("idx_pharma_purchase_requests_supplier", false, "supplier_id")}, "purchase"),
	table("pharma_oa_purchase_orders", "purchase.order", []string{"number", "purchase_request_id", "supplier_id", "lines_json", "total_amount", "status", "approved_by", "approved_at"}, []IndexSpec{index("uk_pharma_purchase_orders_number", true, "number"), index("uk_pharma_purchase_orders_request", true, "purchase_request_id"), index("idx_pharma_purchase_orders_supplier_status", false, "supplier_id", "status")}, "purchase"),
	table("pharma_oa_purchase_inbounds", "purchase.inbound", []string{"number", "purchase_order_id", "warehouse_id", "area_id", "location_id", "lines_json", "status", "received_by", "received_at", "attachments_json", "idempotency_key"}, []IndexSpec{index("uk_pharma_inbounds_number", true, "number"), index("uk_pharma_inbounds_idempotency", true, "idempotency_key"), index("idx_pharma_inbounds_order", false, "purchase_order_id")}, "inventory"),
	table("pharma_oa_sales_orders", "sales.order", []string{"number", "customer_id", "lines_json", "total_amount", "status", "organization_id", "actor_id", "business_created_at"}, []IndexSpec{index("uk_pharma_sales_orders_number", true, "number"), index("idx_pharma_sales_orders_customer_status", false, "customer_id", "status"), index("idx_pharma_sales_orders_scope", false, "organization_id", "created_at")}, "sales"),
	table("pharma_oa_sales_outbounds", "sales.outbound", []string{"number", "sales_order_id", "customer_id", "warehouse_id", "area_id", "location_id", "lines_json", "status", "shipped_by", "shipped_at", "idempotency_key"}, []IndexSpec{index("uk_pharma_outbounds_number", true, "number"), index("uk_pharma_outbounds_idempotency", true, "idempotency_key"), index("idx_pharma_outbounds_order", false, "sales_order_id")}, "inventory"),
	table("pharma_oa_stocktakes", "inventory.stocktake", []string{"number", "product_id", "warehouse_id", "area_id", "location_id", "batch_id", "system_quantity", "actual_quantity", "difference", "reason", "approver_id", "workflow_instance_id", "ledger_id", "status", "actor_id", "approved_by", "business_created_at", "completed_at"}, []IndexSpec{index("uk_pharma_stocktakes_number", true, "number"), index("idx_pharma_stocktakes_warehouse_status", false, "warehouse_id", "status")}, "inventory"),
	table("pharma_oa_transfers", "inventory.transfer", []string{"number", "product_id", "batch_id", "quantity", "source_warehouse_id", "source_area_id", "source_location_id", "target_warehouse_id", "target_area_id", "target_location_id", "out_ledger_id", "in_ledger_id", "status", "transferred_by", "transferred_at", "idempotency_key"}, []IndexSpec{index("uk_pharma_transfers_number", true, "number"), index("uk_pharma_transfers_idempotency", true, "idempotency_key"), index("idx_pharma_transfers_status", false, "status")}, "inventory"),
	table("pharma_oa_announcements", "collaboration.announcement", []string{"title", "content", "status", "published_at", "audience_json", "documents_json"}, []IndexSpec{index("idx_pharma_announcements_status_published", false, "status", "published_at")}, "collaboration"),
	table("pharma_oa_contracts", "compliance.contract", []string{"number", "title", "party_type", "party_id", "status", "workflow_instance_id", "owner_id", "approver_id", "amount", "currency", "effective_at", "expires_at", "payload_json"}, []IndexSpec{index("uk_pharma_contracts_number", true, "number"), index("idx_pharma_contracts_party_status", false, "party_type", "party_id", "status"), index("idx_pharma_contracts_expiry", false, "expires_at", "status")}, "compliance"),
	table("pharma_oa_qualifications", "compliance.qualification", []string{"subject_type", "subject_id", "qualification_type", "number", "status", "expires_at", "attachment_ids_json"}, []IndexSpec{index("uk_pharma_qualifications_subject_number", true, "subject_type", "subject_id", "number"), index("idx_pharma_qualifications_expiry", false, "status", "expires_at")}, "compliance"),
	table("pharma_oa_quality_complaints", "compliance.complaint", []string{"number", "title", "customer_id", "product_id", "batch_id", "status", "workflow_instance_id", "reporter_id", "handler_id", "payload_json"}, []IndexSpec{index("uk_pharma_complaints_number", true, "number"), index("idx_pharma_complaints_status_product", false, "status", "product_id"), index("idx_pharma_complaints_batch", false, "batch_id")}, "compliance"),
	table("pharma_oa_drug_recalls", "compliance.recall", []string{"number", "title", "product_id", "batch_id", "source_complaint_id", "status", "initiated_by", "payload_json"}, []IndexSpec{index("uk_pharma_recalls_number", true, "number"), index("idx_pharma_recalls_batch_status", false, "batch_id", "status")}, "compliance"),
	immutableTable("pharma_oa_cold_chain_records", "compliance.cold_chain", []string{"product_id", "batch_id", "warehouse_id", "temperature", "humidity", "recorded_at", "source"}, []IndexSpec{index("idx_pharma_cold_chain_batch_time", false, "batch_id", "recorded_at"), index("idx_pharma_cold_chain_warehouse_time", false, "warehouse_id", "recorded_at")}, "compliance"),
	table("pharma_oa_customer_follow_ups", "crm.follow_up", []string{"customer_id", "customer_code", "customer_name", "owner_id", "organization_id", "status", "scheduled_at", "payload_json"}, []IndexSpec{index("idx_pharma_followups_scope_status", false, "organization_id", "owner_id", "status"), index("idx_pharma_followups_customer_time", false, "customer_id", "scheduled_at")}, "crm"),
	table("pharma_oa_sales_opportunities", "crm.opportunity", []string{"title", "customer_id", "customer_code", "customer_name", "owner_id", "organization_id", "stage", "expected_amount_cents", "estimated_close_date", "payload_json"}, []IndexSpec{index("idx_pharma_opportunities_scope_stage", false, "organization_id", "owner_id", "stage"), index("idx_pharma_opportunities_customer", false, "customer_id")}, "crm"),
	table("pharma_oa_payment_plans", "finance.payment", []string{"sales_order_id", "sales_order_number", "customer_id", "owner_id", "order_total_cents", "amount_cents", "paid_amount_cents", "status", "due_at", "payload_json"}, []IndexSpec{index("uk_pharma_payment_plan_order", true, "sales_order_id"), index("idx_pharma_payment_plan_due_status", false, "status", "due_at")}, "finance"),
	table("pharma_oa_invoice_records", "finance.invoice", []string{"number", "sales_order_id", "sales_order_number", "customer_id", "owner_id", "order_total_cents", "amount_cents", "status", "issued_at", "voided_at", "payload_json"}, []IndexSpec{index("uk_pharma_invoices_number", true, "number"), index("idx_pharma_invoices_order_status", false, "sales_order_id", "status")}, "finance"),
	table("pharma_oa_payment_reminder_jobs", "jobs.payment_reminder", []string{"status", "recipient_id", "owner_id", "initiated_by", "last_run_by", "retry_count", "error", "started_at", "completed_at", "payload_json"}, []IndexSpec{index("idx_pharma_payment_reminder_jobs_status_created", false, "status", "created_at")}, "jobs"),
	table("pharma_oa_inventory_alerts", "jobs.inventory_alert", []string{"type", "status", "balance_id", "product_id", "warehouse_id", "batch_id", "recipient_id", "notification_id", "last_seen_at", "resolved_at", "payload_json"}, []IndexSpec{index("uk_pharma_inventory_alert_position", true, "type", "balance_id"), index("idx_pharma_inventory_alerts_status_seen", false, "status", "last_seen_at")}, "jobs"),
	table("pharma_oa_inventory_alert_jobs", "jobs.inventory_alert", []string{"status", "idempotency_key", "retry_count", "error", "started_at", "completed_at", "payload_json"}, []IndexSpec{index("uk_pharma_alert_jobs_idempotency", true, "idempotency_key"), index("idx_pharma_alert_jobs_status_created", false, "status", "created_at")}, "jobs"),
	table("pharma_oa_report_export_jobs", "jobs.report_export", []string{"report_type", "status", "owner_id", "idempotency_key", "retry_count", "file_id", "error", "started_at", "completed_at", "payload_json"}, []IndexSpec{index("uk_pharma_export_jobs_idempotency", true, "idempotency_key"), index("idx_pharma_export_jobs_status_created", false, "status", "created_at")}, "jobs"),
}
