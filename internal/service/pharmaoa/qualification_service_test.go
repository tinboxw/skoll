package pharmaoa

import (
	"context"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestQualificationLedgerAndExpiryScanAreUnifiedAndIdempotent(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)
	employees := NewEmployeeService(nil).(*employeeService)
	employees.nowFn = func() time.Time { return now }
	_, err := employees.Create(ctx, EmployeeWriteInput{Code: "EMP-Q-1", Name: "Alice", DepartmentID: "dept-1", PositionID: "position-1", Certificates: []domainpharma.EmployeeCertificate{{ID: "health", Name: "Health certificate", Number: "HC-1", ExpiresAt: now.AddDate(0, 0, 10)}}})
	if err != nil {
		t.Fatal(err)
	}
	suppliers := NewSupplierService(nil).(*supplierService)
	suppliers.nowFn = func() time.Time { return now }
	_, err = suppliers.Create(ctx, SupplierWriteInput{Code: "SUP-Q-1", Name: "Expired Supplier", Qualifications: []domainpharma.SupplierQualification{{ID: "license", Name: "Distribution license", Number: "SL-1", ExpiresAt: now.Add(-time.Hour)}}})
	if err != nil {
		t.Fatal(err)
	}
	customers := NewCustomerService(nil).(*customerService)
	customers.nowFn = func() time.Time { return now }
	_, err = customers.Create(ctx, CustomerWriteInput{Code: "CUS-Q-1", Name: "Qualified Hospital", Region: "East", OrganizationID: "org-1", OwnerID: "owner-1", Qualifications: []domainpharma.CustomerQualification{{ID: "license", Name: "Institution license", Number: "CL-1", ExpiresAt: now.AddDate(1, 0, 0)}}, Scope: CustomerAccessScope{IncludeAll: true}})
	if err != nil {
		t.Fatal(err)
	}

	notifications := notificationsvc.NewService(notificationsvc.NewMemoryRepository(), func() time.Time { return now }, nil)
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	service := NewQualificationService(employees, suppliers, customers, notifications, audit).(*qualificationService)
	service.nowFn = func() time.Time { return now }
	records, err := service.List(ctx, QualificationListInput{Days: 30})
	if err != nil || len(records) != 3 {
		t.Fatalf("qualification ledger: %+v %v", records, err)
	}
	statuses := map[QualificationSubjectType]QualificationStatus{}
	for _, record := range records {
		statuses[record.SubjectType] = record.Status
	}
	if statuses[QualificationSubjectEmployee] != QualificationStatusExpiring || statuses[QualificationSubjectSupplier] != QualificationStatusExpired || statuses[QualificationSubjectCustomer] != QualificationStatusValid {
		t.Fatalf("unexpected qualification statuses: %+v", statuses)
	}

	first, err := service.ScanExpiry(ctx, QualificationScanInput{Days: 30, ActorID: "scheduler"})
	if err != nil || first.MatchedCount != 2 || first.CreatedCount != 2 {
		t.Fatalf("first qualification scan: %+v %v", first, err)
	}
	second, err := service.ScanExpiry(ctx, QualificationScanInput{Days: 30, ActorID: "scheduler"})
	if err != nil || second.MatchedCount != 2 || second.CreatedCount != 0 {
		t.Fatalf("duplicate qualification scan: %+v %v", second, err)
	}
	employeeNotifications, _ := notifications.List(ctx, notificationsvc.Filter{ActorID: "pharma-employee-1", Category: notificationsvc.CategoryReminder})
	if len(employeeNotifications) != 1 || employeeNotifications[0].Target.Path != "/skoll/pharma-oa/employees?employeeId=pharma-employee-1" {
		t.Fatalf("unexpected employee reminder: %+v", employeeNotifications)
	}
	schedulerAudit, _ := audit.ListByActor(ctx, "scheduler", 10)
	if len(schedulerAudit) != 4 {
		t.Fatalf("qualification audit trail incomplete: %+v", schedulerAudit)
	}
}

func TestQualificationBlockingIsAuditedForPurchaseAndSales(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	suppliers := NewSupplierService(audit).(*supplierService)
	suppliers.nowFn = func() time.Time { return now }
	supplier, err := suppliers.Create(ctx, SupplierWriteInput{Code: "SUP-BLOCK", Name: "Blocked Supplier", Qualifications: []domainpharma.SupplierQualification{{ID: "license", Name: "Distribution license", Number: "SL-X", ExpiresAt: now.Add(-time.Hour)}}})
	if err != nil {
		t.Fatal(err)
	}
	purchase := NewPurchaseService(suppliers, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), audit)
	_, err = purchase.CreateRequest(ctx, PurchaseRequestCreateInput{Number: "PR-BLOCK", SupplierID: supplier.ID.String(), RequesterID: "buyer-1", ApproverID: "manager-1", Lines: []domainpharma.PurchaseLine{{ProductID: "product-1", Quantity: 1, UnitPrice: 1}}})
	if err == nil || !strings.Contains(err.Error(), "qualification expired") {
		t.Fatalf("expired supplier was not blocked: %v", err)
	}

	customers := NewCustomerService(audit).(*customerService)
	customers.nowFn = func() time.Time { return now }
	customer, err := customers.Create(ctx, CustomerWriteInput{Code: "CUS-BLOCK", Name: "Blocked Customer", Region: "East", OrganizationID: "org-1", OwnerID: "seller-1", Qualifications: []domainpharma.CustomerQualification{{ID: "license", Name: "Institution license", Number: "CL-X", ExpiresAt: now.Add(-time.Hour)}}, Scope: CustomerAccessScope{IncludeAll: true}})
	if err != nil {
		t.Fatal(err)
	}
	sales := NewSalesService(customers, nil, nil, audit)
	_, err = sales.CreateOrder(ctx, SalesOrderCreateInput{Number: "SO-BLOCK", CustomerID: customer.ID.String(), ActorID: "seller-1", Lines: []domainpharma.SalesLine{{ProductID: "product-1", Quantity: 1, UnitPrice: 1}}})
	if err == nil || !strings.Contains(err.Error(), "qualification expired") {
		t.Fatalf("expired customer was not blocked: %v", err)
	}

	buyerAudit, _ := audit.ListByActor(ctx, "buyer-1", 10)
	sellerAudit, _ := audit.ListByActor(ctx, "seller-1", 10)
	if len(buyerAudit) != 1 || buyerAudit[0].Action != "pharma_oa.qualification.block" {
		t.Fatalf("purchase block audit missing: %+v", buyerAudit)
	}
	qualifiedBlock := false
	for _, record := range sellerAudit {
		if record.Action == "pharma_oa.qualification.block" {
			qualifiedBlock = true
		}
	}
	if !qualifiedBlock {
		t.Fatalf("sales block audit missing: %+v", sellerAudit)
	}
}
