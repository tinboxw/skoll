package pharmaoa

import (
	"context"
	"testing"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestQualityComplaintStartsWorkflowAndStoresConclusion(t *testing.T) {
	ctx := context.Background()
	service, audit := newQualityComplaintFixture(t, false)
	item, err := service.Create(ctx, QualityComplaintCreateInput{Number: "QC-001", Title: "Damaged package", Description: "Outer package was damaged on delivery", CustomerID: "pharma-customer-1", ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1", ReporterID: "reporter-1", HandlerID: "handler-1", AttachmentIDs: []string{"complaint-file-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != domainpharma.QualityComplaintPending || item.WorkflowInstanceID == "" || item.BatchNo != "BATCH-QC-001" || len(item.Attachments) != 1 {
		t.Fatalf("complaint relation incomplete: %+v", item)
	}
	if _, err = service.Create(ctx, QualityComplaintCreateInput{Number: "qc-001", Title: "Duplicate complaint", Description: "Duplicate number", CustomerID: "pharma-customer-1", ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1", ReporterID: "reporter-1", HandlerID: "handler-1", AttachmentIDs: []string{"complaint-file-1"}}); err == nil {
		t.Fatal("duplicate complaint number was accepted")
	}
	batches, err := service.ListBatches(ctx, "pharma-product-1")
	if err != nil || len(batches) != 1 || batches[0].ID.String() != item.BatchID {
		t.Fatalf("complaint batch selector incomplete: %+v %v", batches, err)
	}
	otherBatches, err := service.ListBatches(ctx, "pharma-product-2")
	if err != nil || len(otherBatches) != 0 {
		t.Fatalf("complaint batch selector ignored product: %+v %v", otherBatches, err)
	}
	if _, err = service.Resolve(ctx, item.ID.String(), QualityComplaintActionInput{ActorID: "other-user", Conclusion: "spoofed"}); err == nil {
		t.Fatal("non-assignee resolved complaint")
	}
	if _, err = service.Resolve(ctx, item.ID.String(), QualityComplaintActionInput{ActorID: "handler-1"}); err == nil {
		t.Fatal("empty conclusion was accepted")
	}
	resolved, err := service.Resolve(ctx, item.ID.String(), QualityComplaintActionInput{ActorID: "handler-1", Conclusion: "Replacement shipped and packaging CAPA opened"})
	if err != nil || resolved.Status != domainpharma.QualityComplaintResolved || resolved.ResolvedBy != "handler-1" || resolved.ResolvedAt == nil {
		t.Fatalf("resolve complaint: %+v %v", resolved, err)
	}
	again, err := service.Resolve(ctx, item.ID.String(), QualityComplaintActionInput{ActorID: "handler-1", Conclusion: "ignored"})
	if err != nil || again.Conclusion != resolved.Conclusion {
		t.Fatalf("resolve is not idempotent: %+v %v", again, err)
	}
	rejectable, err := service.Create(ctx, QualityComplaintCreateInput{Number: "QC-002", Title: "Unfounded complaint", Description: "Investigation found no product defect", CustomerID: "pharma-customer-1", ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1", ReporterID: "reporter-1", HandlerID: "handler-1", AttachmentIDs: []string{"complaint-file-1"}})
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := service.Reject(ctx, rejectable.ID.String(), QualityComplaintActionInput{ActorID: "handler-1", Conclusion: "Retained sample and logistics records show no quality deviation"})
	if err != nil || rejected.Status != domainpharma.QualityComplaintRejected || rejected.RejectedBy != "handler-1" || rejected.RejectedAt == nil {
		t.Fatalf("reject complaint: %+v %v", rejected, err)
	}
	records, _ := audit.ListByActor(ctx, "handler-1", 10)
	complaintActions := map[string]int{}
	for _, record := range records {
		if record.Action == "pharma_oa.quality_complaint.resolve" || record.Action == "pharma_oa.quality_complaint.reject" {
			complaintActions[record.Action]++
		}
	}
	if complaintActions["pharma_oa.quality_complaint.resolve"] != 1 || complaintActions["pharma_oa.quality_complaint.reject"] != 1 {
		t.Fatalf("resolution audit incomplete: actions=%+v records=%+v", complaintActions, records)
	}
}

func TestQualityComplaintRejectsInvalidBatchAndInaccessibleFile(t *testing.T) {
	ctx := context.Background()
	service, _ := newQualityComplaintFixture(t, false)
	_, err := service.Create(ctx, QualityComplaintCreateInput{Number: "QC-BATCH", Title: "Wrong batch", Description: "Mismatch", CustomerID: "pharma-customer-1", ProductID: "pharma-product-2", BatchID: "pharma-stock-batch-1", ReporterID: "reporter-1", HandlerID: "handler-1", AttachmentIDs: []string{"complaint-file-1"}})
	if err == nil {
		t.Fatal("mismatched product batch was accepted")
	}
	denied, _ := newQualityComplaintFixture(t, true)
	_, err = denied.Create(ctx, QualityComplaintCreateInput{Number: "QC-FILE", Title: "Denied file", Description: "File is private", CustomerID: "pharma-customer-1", ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1", ReporterID: "reporter-1", HandlerID: "handler-1", AttachmentIDs: []string{"complaint-file-1"}})
	if err == nil {
		t.Fatal("inaccessible complaint file was accepted")
	}
	_, err = service.Create(ctx, QualityComplaintCreateInput{Number: "QC-EMPTY-FILE", Title: "Empty file", Description: "File id missing", CustomerID: "pharma-customer-1", ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1", ReporterID: "reporter-1", HandlerID: "handler-1", AttachmentIDs: []string{""}})
	if err == nil {
		t.Fatal("empty complaint file id was accepted")
	}
}

func newQualityComplaintFixture(t *testing.T, denyFile bool) (QualityComplaintService, auditsvc.Service) {
	t.Helper()
	ctx := context.Background()
	customers := NewCustomerService(nil)
	if _, err := customers.Create(ctx, CustomerWriteInput{Code: "CUS-QC", Name: "Complaint Hospital", Region: "North", OrganizationID: "org-1", OwnerID: "owner-1", Rating: 5, Contacts: []domainpharma.CustomerContact{{Name: "Alice"}}, Scope: CustomerAccessScope{IncludeAll: true}}); err != nil {
		t.Fatal(err)
	}
	products := NewProductService(nil)
	if _, err := products.Create(ctx, ProductWriteInput{Code: "DRUG-QC", Name: "Complaint Drug", Spec: "10mg", DosageForm: "tablet", Manufacturer: "Skoll Pharma", ApprovalNumber: "APP-QC"}); err != nil {
		t.Fatal(err)
	}
	if _, err := products.Create(ctx, ProductWriteInput{Code: "DRUG-QC-ALT", Name: "Alternative Drug", Spec: "20mg", DosageForm: "tablet", Manufacturer: "Skoll Pharma", ApprovalNumber: "APP-QC-ALT"}); err != nil {
		t.Fatal(err)
	}
	inventory := NewInventoryService(nil)
	if _, err := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "IN-QC", ProductID: "pharma-product-1", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "BATCH-QC-001", ProductionDate: time.Now().UTC().AddDate(0, -1, 0), ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 10, ActorID: "warehouse-user"}); err != nil {
		t.Fatal(err)
	}
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	file := &domainfile.FileObject{ID: "complaint-file-1", Name: "complaint.jpg", MIME: "image/jpeg", Size: 2048, Status: domainfile.StatusAvailable}
	files := contractFileFixture{items: map[string]*domainfile.FileObject{"complaint-file-1": file}, deny: denyFile}
	return NewQualityComplaintService(customers, products, inventory, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), files, audit), audit
}
