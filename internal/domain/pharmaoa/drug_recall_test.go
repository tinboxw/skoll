package pharmaoa

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestDrugRecallCompletesTrackedCustomerTasks(t *testing.T) {
	now := time.Date(2026, 7, 13, 9, 0, 0, 0, time.UTC)
	item, err := NewDrugRecall("drug-recall-1", DrugRecallInput{Number: "DR-001", Title: "Batch recall", Reason: "Quality risk", ProductID: "product-1", ProductName: "Drug A", BatchID: "batch-1", BatchNo: "B001", InitiatedBy: "quality-1", Scopes: []DrugRecallScope{{CustomerID: "customer-1", CustomerName: "Hospital A", OutboundIDs: []string{"outbound-1"}, Quantity: 3}, {CustomerID: "customer-2", CustomerName: "Hospital B", OutboundIDs: []string{"outbound-2"}, Quantity: 2}}}, now)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := item.CompleteTask(item.Tasks[0].ID, "handler-1", "Customer quarantined stock", now.Add(time.Hour))
	if err != nil || !changed || item.Status != DrugRecallActive {
		t.Fatalf("complete first task: %+v %v", item, err)
	}
	changed, err = item.CompleteTask(item.Tasks[1].ID, "handler-2", "All units returned", now.Add(2*time.Hour))
	if err != nil || !changed || item.Status != DrugRecallCompleted || item.CompletedAt == nil || item.CompletedBy != "handler-2" {
		t.Fatalf("complete recall: %+v %v", item, err)
	}
	changed, err = item.CompleteTask(item.Tasks[1].ID, "handler-2", "ignored", now.Add(3*time.Hour))
	if err != nil || changed || item.Tasks[1].CompletionNote != "All units returned" {
		t.Fatalf("completion is not idempotent: %+v %v", item, err)
	}
	if item.Meta.UpdatedAt == (shared.AuditMeta{}).UpdatedAt {
		t.Fatal("recall audit metadata was not touched")
	}
}
