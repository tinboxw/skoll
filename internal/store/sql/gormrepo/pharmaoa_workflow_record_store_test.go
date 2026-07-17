package gormrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	mysqldriver "gorm.io/driver/mysql"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPharmaWorkflowRecordRepositoriesPersistAcrossSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow-records.db")
	db := openWorkflowRecordSQLite(t, path)
	seedWorkflowRecordContract(t, db, "sqlite")
	closeTestDB(t, db)

	db = openWorkflowRecordSQLite(t, path)
	defer closeTestDB(t, db)
	assertWorkflowRecordContract(t, db, "sqlite")
}

func TestPharmaWorkflowRecordRepositoriesMySQLContract(t *testing.T) {
	dsn := os.Getenv("SKOLL_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("SKOLL_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysqldriver.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	runExternalWorkflowRecordContract(t, db, "mysql")
}

func TestPharmaWorkflowRecordRepositoriesPostgreSQLContract(t *testing.T) {
	dsn := os.Getenv("SKOLL_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("SKOLL_TEST_POSTGRES_DSN is not set")
	}
	db, err := gorm.Open(postgresdriver.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	runExternalWorkflowRecordContract(t, db, "postgres")
}

func TestPharmaWorkflowRecordModelsMatchSchemaAndMigrations(t *testing.T) {
	db := TestDB(t)
	implemented := map[string]bool{
		"pharma_oa_contracts": true, "pharma_oa_quality_complaints": true,
		"pharma_oa_drug_recalls": true, "pharma_oa_customer_follow_ups": true,
		"pharma_oa_sales_opportunities": true, "pharma_oa_payment_plans": true,
		"pharma_oa_invoice_records": true, "pharma_oa_payment_reminder_jobs": true,
		"pharma_oa_inventory_alerts": true, "pharma_oa_inventory_alert_jobs": true,
		"pharma_oa_report_export_jobs": true,
	}
	for _, table := range pharmaoarepo.SchemaBaseline() {
		if !implemented[table.Name] {
			continue
		}
		for _, column := range table.Columns {
			if !db.Migrator().HasColumn(table.Name, column) {
				t.Fatalf("GORM model %s missing schema column %s", table.Name, column)
			}
		}
		for _, index := range table.Indexes {
			if !db.Migrator().HasIndex(table.Name, index.Name) {
				t.Fatalf("GORM model %s missing schema index %s", table.Name, index.Name)
			}
		}
	}
	for _, path := range []string{
		filepath.Join("..", "..", "..", "..", "migrations", "mysql", "20260718_000021_create_pharma_oa_workflow_records.sql"),
		filepath.Join("..", "..", "..", "..", "migrations", "postgres", "20260718_000021_create_pharma_oa_workflow_records.sql"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ToLower(string(raw))
		if strings.Contains(text, "drop table") {
			t.Fatalf("migration must retain data: %s", path)
		}
		for table := range implemented {
			if !strings.Contains(text, table) {
				t.Fatalf("migration %s missing %s", path, table)
			}
		}
	}
}

func workflowRecordModels() []any {
	return []any{
		&PharmaContractModel{}, &PharmaQualityComplaintModel{}, &PharmaDrugRecallModel{},
		&PharmaCustomerFollowUpModel{}, &PharmaSalesOpportunityModel{}, &PharmaPaymentPlanModel{},
		&PharmaInvoiceRecordModel{}, &PharmaPaymentReminderJobModel{}, &PharmaInventoryAlertModel{},
		&PharmaInventoryAlertJobModel{}, &PharmaReportExportJobModel{},
	}
}

func openWorkflowRecordSQLite(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(workflowRecordModels()...); err != nil {
		t.Fatal(err)
	}
	return db
}

func runExternalWorkflowRecordContract(t *testing.T, db *gorm.DB, dialect string) {
	t.Helper()
	if err := db.AutoMigrate(workflowRecordModels()...); err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%s-%d", dialect, time.Now().UnixNano())
	seedWorkflowRecordContract(t, db, suffix)
	assertWorkflowRecordContract(t, db, suffix)
	t.Cleanup(func() {
		for _, model := range workflowRecordModels() {
			_ = db.Where("id LIKE ?", "%"+suffix+"%").Delete(model).Error
		}
	})
}

func seedWorkflowRecordContract(t *testing.T, db *gorm.DB, suffix string) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	meta := shared.AuditMeta{CreatedAt: now, UpdatedAt: now}
	attachment := domainpharma.ContractAttachment{FileID: "file-" + suffix, FileName: "质量附件.pdf", MIME: "application/pdf", Size: 128}

	contract := &domainpharma.Contract{ID: shared.ID("contract-" + suffix), Number: "HT-" + suffix, Title: "年度供货合同", PartyType: domainpharma.ContractPartySupplier, PartyID: "supplier-1", PartyName: "华东供应商", OwnerID: "owner-1", ApproverID: "approver-1", Amount: 1234.50, Currency: "CNY", EffectiveAt: now, ExpiresAt: now.AddDate(1, 0, 0), Attachments: []domainpharma.ContractAttachment{attachment}, WorkflowInstanceID: "workflow-contract-" + suffix, Status: domainpharma.ContractActive, ApprovedBy: "approver-1", ApprovedAt: timePointer(now.Add(time.Hour)), ReminderNotificationID: "notice-contract-" + suffix, Meta: meta}
	if err := NewPharmaContractStore(db).Create(ctx, contract); err != nil {
		t.Fatal(err)
	}
	complaint := &domainpharma.QualityComplaint{ID: shared.ID("complaint-" + suffix), Number: "TS-" + suffix, Title: "包装破损", Description: "到货包装破损", CustomerID: "customer-1", CustomerName: "第一医院", ProductID: "product-1", ProductName: "测试药品", BatchID: "batch-1", BatchNo: "B202607", ReporterID: "reporter-1", HandlerID: "handler-1", Attachments: []domainpharma.ContractAttachment{attachment}, WorkflowInstanceID: "workflow-complaint-" + suffix, Status: domainpharma.QualityComplaintResolved, Conclusion: "完成换货", ResolvedBy: "handler-1", ResolvedAt: timePointer(now.Add(2 * time.Hour)), Meta: meta}
	if err := NewPharmaQualityComplaintStore(db).Create(ctx, complaint); err != nil {
		t.Fatal(err)
	}
	recall := &domainpharma.DrugRecall{ID: shared.ID("recall-" + suffix), Number: "ZH-" + suffix, Title: "批次召回", Reason: "质量复核", ProductID: "product-1", ProductName: "测试药品", BatchID: "batch-1", BatchNo: "B202607", SourceComplaintID: complaint.ID.String(), Tasks: []domainpharma.DrugRecallTask{{ID: "task-1", CustomerID: "customer-1", CustomerName: "第一医院", OutboundIDs: []string{"outbound-1"}, Quantity: 3, Status: domainpharma.DrugRecallTaskCompleted, CompletionNote: "已回收", CompletedBy: "operator-1", CompletedAt: timePointer(now.Add(3 * time.Hour))}}, Status: domainpharma.DrugRecallCompleted, InitiatedBy: "quality-1", InitiatedAt: now, CompletedBy: "operator-1", CompletedAt: timePointer(now.Add(3 * time.Hour)), Meta: meta}
	if err := NewPharmaDrugRecallStore(db).Create(ctx, recall); err != nil {
		t.Fatal(err)
	}
	followUp := &domainpharma.CustomerFollowUp{ID: shared.ID("followup-" + suffix), CustomerID: "customer-1", CustomerCode: "KH001", CustomerName: "第一医院", OrganizationID: "org-1", OwnerID: "owner-1", ContactName: "张主任", Channel: domainpharma.CustomerFollowUpOnsite, Status: domainpharma.CustomerFollowUpCompleted, ScheduledAt: now, CompletedAt: timePointer(now.Add(time.Hour)), Summary: "完成产品回访", NextAction: "下月复访", Attachments: []domainpharma.CustomerFollowUpAttachment{{FileID: "follow-file", FileName: "回访记录.pdf", Size: 64}}, CreatedAt: now, UpdatedAt: now.Add(time.Hour)}
	if err := NewPharmaCustomerFollowUpStore(db).Create(ctx, followUp); err != nil {
		t.Fatal(err)
	}
	opportunity := &domainpharma.SalesOpportunity{ID: shared.ID("opportunity-" + suffix), Title: "医院配送合作", CustomerID: "customer-1", CustomerCode: "KH001", CustomerName: "第一医院", OrganizationID: "org-1", OwnerID: "owner-1", Products: []domainpharma.SalesOpportunityProduct{{ProductID: "product-1", ProductCode: "YP001", ProductName: "测试药品"}}, ExpectedAmountCents: 880000, EstimatedCloseDate: now.AddDate(0, 1, 0), Stage: domainpharma.SalesOpportunityQualified, StageHistory: []domainpharma.SalesOpportunityStageChange{{From: domainpharma.SalesOpportunityLead, To: domainpharma.SalesOpportunityQualified, ChangedBy: "owner-1", ChangedAt: now, Note: "资质已核验"}}, Meta: meta}
	if err := NewPharmaSalesOpportunityStore(db).Create(ctx, opportunity); err != nil {
		t.Fatal(err)
	}
	plan := &domainpharma.PaymentPlan{ID: shared.ID("payment-" + suffix), SalesOrderID: "sales-order-" + suffix, SalesOrderNumber: "SO-" + suffix, CustomerID: "customer-1", OwnerID: "owner-1", OrderTotalCents: 100000, AmountCents: 100000, PaidAmountCents: 40000, DueAt: now.AddDate(0, 0, -1), Status: domainpharma.PaymentPlanOverdue, Note: "分期回款", Attachments: []domainpharma.FinancialAttachment{{FileID: "payment-file", FileName: "付款计划.pdf", Size: 32}}, Receipts: []domainpharma.PaymentReceipt{{ID: shared.ID("receipt-" + suffix), AmountCents: 40000, PaidAt: now, Reference: "流水号-001", RecordedBy: "finance-1", RecordedAt: now}}, NotificationID: "notice-payment-" + suffix, ReminderRecipientID: "finance-manager", RemindedAt: timePointer(now.Add(time.Hour)), CreatedAt: now, UpdatedAt: now.Add(time.Hour)}
	if err := NewPharmaPaymentPlanStore(db).Create(ctx, plan); err != nil {
		t.Fatal(err)
	}
	invoice := &domainpharma.InvoiceRecord{ID: shared.ID("invoice-" + suffix), Number: "FP-" + suffix, SalesOrderID: plan.SalesOrderID, SalesOrderNumber: plan.SalesOrderNumber, CustomerID: plan.CustomerID, OwnerID: plan.OwnerID, OrderTotalCents: plan.OrderTotalCents, AmountCents: 40000, IssuedAt: now, Status: domainpharma.InvoiceRecordVoided, Note: "首期开票", Attachments: []domainpharma.FinancialAttachment{{FileID: "invoice-file", FileName: "发票.pdf", Size: 48}}, CreatedBy: "finance-1", CreatedAt: now, VoidedBy: "finance-manager", VoidReason: "信息有误", VoidedAt: timePointer(now.Add(time.Hour))}
	if err := NewPharmaInvoiceRecordStore(db).Create(ctx, invoice); err != nil {
		t.Fatal(err)
	}
	reminder := &domainpharma.PaymentReminderJob{ID: shared.ID("reminder-job-" + suffix), Status: domainpharma.PaymentReminderJobFailed, RecipientID: "finance-manager", OwnerID: "owner-1", InitiatedBy: "finance-1", LastRunBy: "finance-1", RetryCount: 1, MatchedCount: 1, Error: "通知服务暂不可用", Logs: []domainpharma.PaymentReminderJobLog{{Level: domainpharma.PaymentReminderJobLogError, Message: "通知服务暂不可用", CreatedAt: now}}, CreatedAt: now, StartedAt: timePointer(now), CompletedAt: timePointer(now.Add(time.Minute))}
	if err := NewPharmaPaymentReminderJobStore(db).Create(ctx, reminder); err != nil {
		t.Fatal(err)
	}
	if err := reminder.Start("finance-manager", true, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	reminder.Succeed(1, 0, now.Add(3*time.Minute))
	if err := NewPharmaPaymentReminderJobStore(db).Upsert(ctx, reminder); err != nil {
		t.Fatal(err)
	}

	alertRepo := NewPharmaInventoryAlertStore(db)
	alertJob := &domainpharma.InventoryAlertJob{ID: shared.ID("alert-job-" + suffix), Status: domainpharma.InventoryAlertJobSucceeded, IdempotencyKey: "inventory-alert:alert-job-" + suffix, Policy: domainpharma.InventoryAlertPolicy{NearExpiryDays: 30, LowStockThreshold: 5, OverStockThreshold: 100, RecipientID: "warehouse-manager"}, RetryCount: 1, MatchedCount: 1, CreatedCount: 1, Logs: []domainpharma.InventoryAlertJobLog{{Level: "info", Message: "重试成功", CreatedAt: now}}, CreatedAt: now, StartedAt: timePointer(now), CompletedAt: timePointer(now.Add(time.Minute))}
	if err := alertRepo.CreateJob(ctx, alertJob); err != nil {
		t.Fatal(err)
	}
	alert := &domainpharma.InventoryAlert{ID: shared.ID("alert-" + suffix), Type: domainpharma.InventoryAlertNearExpiry, Status: domainpharma.InventoryAlertActive, BalanceID: "balance-1", ProductID: "product-1", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchID: "batch-1", Quantity: 4, Threshold: 5, ExpiresAt: now.AddDate(0, 0, 20), RecipientID: "warehouse-manager", NotificationID: "notice-alert-" + suffix, TargetPath: "/skoll/pharma/inventory", FirstSeenAt: now, LastSeenAt: now.Add(time.Hour)}
	if err := alertRepo.UpsertAlert(ctx, alert); err != nil {
		t.Fatal(err)
	}
	report := &domainpharma.ReportExportJob{ID: "report-job-" + suffix, ReportType: domainpharma.ReportExportBusinessMetrics, Status: domainpharma.ReportExportSucceeded, Query: domainpharma.ReportExportQuery{From: now.AddDate(0, -1, 0), To: now, Bucket: "day", QualificationDays: 30}, OwnerID: "analyst-1", IdempotencyKey: "report-export:" + suffix, FileID: "file-report-" + suffix, Filename: "业务报表.csv", ContentType: "text/csv", Size: 256, RowCount: 8, RetryCount: 1, Logs: []domainpharma.ReportExportJobLog{{Level: "info", Message: "导出完成", CreatedAt: now}}, CreatedAt: now, StartedAt: timePointer(now), CompletedAt: timePointer(now.Add(time.Minute))}
	if err := NewPharmaReportExportJobStore(db).Create(ctx, report); err != nil {
		t.Fatal(err)
	}
}

func assertWorkflowRecordContract(t *testing.T, db *gorm.DB, suffix string) {
	t.Helper()
	ctx := context.Background()
	contract, err := NewPharmaContractStore(db).Get(ctx, shared.ID("contract-"+suffix))
	if err != nil || contract == nil || contract.WorkflowInstanceID != "workflow-contract-"+suffix || len(contract.Attachments) != 1 || contract.ReminderNotificationID == "" {
		t.Fatalf("contract restart: %+v %v", contract, err)
	}
	complaint, err := NewPharmaQualityComplaintStore(db).Get(ctx, shared.ID("complaint-"+suffix))
	if err != nil || complaint == nil || complaint.ResolvedBy != "handler-1" || len(complaint.Attachments) != 1 {
		t.Fatalf("complaint restart: %+v %v", complaint, err)
	}
	recall, err := NewPharmaDrugRecallStore(db).Get(ctx, shared.ID("recall-"+suffix))
	if err != nil || recall == nil || len(recall.Tasks) != 1 || recall.Tasks[0].CompletionNote != "已回收" {
		t.Fatalf("recall restart: %+v %v", recall, err)
	}
	followUp, err := NewPharmaCustomerFollowUpStore(db).Get(ctx, shared.ID("followup-"+suffix))
	if err != nil || followUp == nil || len(followUp.Attachments) != 1 || followUp.Summary == "" {
		t.Fatalf("follow-up restart: %+v %v", followUp, err)
	}
	opportunity, err := NewPharmaSalesOpportunityStore(db).Get(ctx, shared.ID("opportunity-"+suffix))
	if err != nil || opportunity == nil || len(opportunity.Products) != 1 || len(opportunity.StageHistory) != 1 {
		t.Fatalf("opportunity restart: %+v %v", opportunity, err)
	}
	plan, err := NewPharmaPaymentPlanStore(db).Get(ctx, shared.ID("payment-"+suffix))
	if err != nil || plan == nil || len(plan.Receipts) != 1 || plan.NotificationID == "" {
		t.Fatalf("payment plan restart: %+v %v", plan, err)
	}
	invoice, err := NewPharmaInvoiceRecordStore(db).Get(ctx, shared.ID("invoice-"+suffix))
	if err != nil || invoice == nil || invoice.VoidReason == "" || len(invoice.Attachments) != 1 {
		t.Fatalf("invoice restart: %+v %v", invoice, err)
	}
	reminder, err := NewPharmaPaymentReminderJobStore(db).Get(ctx, shared.ID("reminder-job-"+suffix))
	if err != nil || reminder == nil || reminder.Status != domainpharma.PaymentReminderJobSucceeded || reminder.RetryCount != 2 || len(reminder.Logs) < 3 {
		t.Fatalf("payment reminder restart/retry: %+v %v", reminder, err)
	}
	alertRepo := NewPharmaInventoryAlertStore(db)
	alertJob, err := alertRepo.GetJob(ctx, shared.ID("alert-job-"+suffix))
	if err != nil || alertJob == nil || alertJob.IdempotencyKey == "" || alertJob.RetryCount != 1 || len(alertJob.Logs) != 1 {
		t.Fatalf("inventory alert job restart: %+v %v", alertJob, err)
	}
	alert, err := alertRepo.GetAlert(ctx, shared.ID("alert-"+suffix))
	if err != nil || alert == nil || alert.NotificationID == "" || alert.TargetPath == "" {
		t.Fatalf("inventory alert restart: %+v %v", alert, err)
	}
	reportRepo := NewPharmaReportExportJobStore(db)
	report, err := reportRepo.GetByIdempotencyKey(ctx, "report-export:"+suffix)
	if err != nil || report == nil || report.FileID == "" || report.RowCount != 8 || len(report.Logs) != 1 {
		t.Fatalf("report export restart/idempotency: %+v %v", report, err)
	}
	jobs, err := reportRepo.List(ctx, pharmaoarepo.ListFilter{OwnerID: shared.ID("analyst-1")})
	if err != nil || len(jobs) != 1 {
		t.Fatalf("report export list: %+v %v", jobs, err)
	}
}

func timePointer(value time.Time) *time.Time { return &value }
