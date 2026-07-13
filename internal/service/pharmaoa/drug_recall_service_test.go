package pharmaoa

import (
	"context"
	"sync"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestDrugRecallLocatesOutboundCustomersAndTracksCompletion(t *testing.T) {
	ctx := context.Background()
	service, audit := newDrugRecallFixture(t)
	scope, err := service.PreviewScope(ctx, "pharma-stock-batch-1")
	if err != nil || len(scope) != 2 || scope[0].Quantity+scope[1].Quantity != 8 {
		t.Fatalf("preview recall scope: %+v %v", scope, err)
	}
	item, err := service.Create(ctx, DrugRecallCreateInput{Number: "DR-001", Title: "Recall BATCH-R-001", Reason: "Confirmed quality deviation", BatchID: "pharma-stock-batch-1", SourceComplaintID: "complaint-1", ActorID: "quality-1"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != domainpharma.DrugRecallActive || len(item.Tasks) != 2 || item.ProductID != "pharma-product-1" || item.BatchNo != "BATCH-R-001" || item.SourceComplaintID != "complaint-1" {
		t.Fatalf("recall relation incomplete: %+v", item)
	}
	if _, err = service.Create(ctx, DrugRecallCreateInput{Number: "dr-001", Title: "Duplicate", Reason: "Duplicate", BatchID: "pharma-stock-batch-1", ActorID: "quality-1"}); err == nil {
		t.Fatal("duplicate recall number was accepted")
	}
	first, err := service.CompleteTask(ctx, item.ID.String(), item.Tasks[0].ID, DrugRecallTaskCompleteInput{ActorID: "handler-1", Note: "Customer notified and stock quarantined"})
	if err != nil || first.Status != domainpharma.DrugRecallActive {
		t.Fatalf("complete first recall task: %+v %v", first, err)
	}
	completed, err := service.CompleteTask(ctx, item.ID.String(), item.Tasks[1].ID, DrugRecallTaskCompleteInput{ActorID: "handler-2", Note: "Affected units returned"})
	if err != nil || completed.Status != domainpharma.DrugRecallCompleted || completed.CompletedAt == nil {
		t.Fatalf("complete recall: %+v %v", completed, err)
	}
	again, err := service.CompleteTask(ctx, item.ID.String(), item.Tasks[1].ID, DrugRecallTaskCompleteInput{ActorID: "handler-2", Note: "ignored"})
	if err != nil || again.Tasks[1].CompletionNote != completed.Tasks[1].CompletionNote {
		t.Fatalf("task completion is not idempotent: %+v %v", again, err)
	}
	records, _ := audit.ListByActor(ctx, "handler-2", 10)
	actions := map[string]int{}
	for _, record := range records {
		actions[record.Action]++
	}
	if actions["pharma_oa.drug_recall.task_complete"] != 1 || actions["pharma_oa.drug_recall.complete"] != 1 {
		t.Fatalf("recall completion audit incomplete: %+v", actions)
	}
}

func TestDrugRecallRejectsMissingScopeAndMismatchedComplaint(t *testing.T) {
	ctx := context.Background()
	service, _ := newDrugRecallFixture(t)
	if _, err := service.Create(ctx, DrugRecallCreateInput{Number: "DR-NONE", Title: "No outbound", Reason: "Test", BatchID: "pharma-stock-batch-2", ActorID: "quality-1"}); err == nil {
		t.Fatal("batch without outbound customers was accepted")
	}
	if _, err := service.Create(ctx, DrugRecallCreateInput{Number: "DR-MISMATCH", Title: "Wrong complaint", Reason: "Test", BatchID: "pharma-stock-batch-1", SourceComplaintID: "complaint-mismatch", ActorID: "quality-1"}); err == nil {
		t.Fatal("mismatched source complaint was accepted")
	}
	if _, err := service.CompleteTask(ctx, "missing", "task", DrugRecallTaskCompleteInput{ActorID: "handler", Note: "done"}); err == nil {
		t.Fatal("missing recall task was completed")
	}
}

func TestDrugRecallRejectsConcurrentDuplicateNumber(t *testing.T) {
	ctx := context.Background()
	service, _ := newDrugRecallFixture(t)
	const workers = 8
	results := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := service.Create(ctx, DrugRecallCreateInput{Number: "DR-CONCURRENT", Title: "Concurrent recall", Reason: "Concurrency check", BatchID: "pharma-stock-batch-1", ActorID: "quality-1"})
			results <- err
		}()
	}
	group.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("expected one concurrent recall creation, got %d", succeeded)
	}
}

type drugRecallSalesFixture struct{ outbounds []*domainpharma.SalesOutbound }

func (f drugRecallSalesFixture) ListOutbounds(context.Context) ([]*domainpharma.SalesOutbound, error) {
	return f.outbounds, nil
}

type drugRecallComplaintFixture struct {
	items map[string]*domainpharma.QualityComplaint
}

func (f drugRecallComplaintFixture) Get(_ context.Context, id string) (*domainpharma.QualityComplaint, error) {
	item := f.items[id]
	if item == nil {
		return nil, context.Canceled
	}
	return item, nil
}

func newDrugRecallFixture(t *testing.T) (DrugRecallService, auditsvc.Service) {
	t.Helper()
	ctx := context.Background()
	products := NewProductService(nil)
	if _, err := products.Create(ctx, ProductWriteInput{Code: "DRUG-R", Name: "Recall Drug", Spec: "10mg", DosageForm: "tablet", Manufacturer: "Skoll Pharma", ApprovalNumber: "APP-R"}); err != nil {
		t.Fatal(err)
	}
	if _, err := products.Create(ctx, ProductWriteInput{Code: "DRUG-R2", Name: "Other Drug", Spec: "20mg", DosageForm: "tablet", Manufacturer: "Skoll Pharma", ApprovalNumber: "APP-R2"}); err != nil {
		t.Fatal(err)
	}
	customers := NewCustomerService(nil)
	for index, name := range []string{"Hospital A", "Hospital B"} {
		if _, err := customers.Create(ctx, CustomerWriteInput{Code: "CUS-R-" + string(rune('1'+index)), Name: name, Region: "North", OrganizationID: "org-1", OwnerID: "owner-1", Rating: 5, Contacts: []domainpharma.CustomerContact{{Name: "Contact"}}, Scope: CustomerAccessScope{IncludeAll: true}}); err != nil {
			t.Fatal(err)
		}
	}
	inventory := NewInventoryService(nil)
	if _, err := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "IN-R1", ProductID: "pharma-product-1", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "BATCH-R-001", ProductionDate: time.Now().UTC().AddDate(0, -1, 0), ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 20, ActorID: "warehouse-user"}); err != nil {
		t.Fatal(err)
	}
	if _, err := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "IN-R2", ProductID: "pharma-product-2", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", BatchNo: "BATCH-R-002", ProductionDate: time.Now().UTC().AddDate(0, -1, 0), ExpiresAt: time.Now().UTC().AddDate(1, 0, 0), Quantity: 20, ActorID: "warehouse-user"}); err != nil {
		t.Fatal(err)
	}
	outbounds := []*domainpharma.SalesOutbound{
		{ID: "outbound-1", CustomerID: "pharma-customer-1", Lines: []domainpharma.SalesOutboundLine{{ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1", Quantity: 3}}},
		{ID: "outbound-2", CustomerID: "pharma-customer-1", Lines: []domainpharma.SalesOutboundLine{{ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1", Quantity: 1}}},
		{ID: "outbound-3", CustomerID: "pharma-customer-2", Lines: []domainpharma.SalesOutboundLine{{ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1", Quantity: 4}}},
	}
	complaints := drugRecallComplaintFixture{items: map[string]*domainpharma.QualityComplaint{
		"complaint-1":        {ID: "complaint-1", ProductID: "pharma-product-1", BatchID: "pharma-stock-batch-1"},
		"complaint-mismatch": {ID: "complaint-mismatch", ProductID: "pharma-product-2", BatchID: "pharma-stock-batch-2"},
	}}
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	return NewDrugRecallService(drugRecallSalesFixture{outbounds: outbounds}, inventory, products, customers, complaints, audit), audit
}
