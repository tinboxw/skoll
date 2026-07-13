package pharmaoa

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestPurchaseRequestAndOrderLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)
	request, err := NewPurchaseRequest("pr-1", PurchaseRequestInput{
		Number: "PR-001", SupplierID: "supplier-1", RequesterID: "buyer-1", ApproverID: "manager-1",
		Lines: []PurchaseLine{{ProductID: "product-1", Quantity: 2, UnitPrice: 12.5}},
	}, "workflow-pr-1", now)
	if err != nil {
		t.Fatalf("NewPurchaseRequest error: %v", err)
	}
	if request.TotalAmount != 25 || request.Status != PurchaseRequestPending {
		t.Fatalf("unexpected request: %+v", request)
	}
	if err := request.Approve("po-1", now.Add(time.Minute)); err != nil {
		t.Fatalf("Approve error: %v", err)
	}
	order, err := NewPurchaseOrder(shared.ID("po-1"), "PO-001", *request, "manager-1", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewPurchaseOrder error: %v", err)
	}
	if order.PurchaseRequestID != "pr-1" || order.TotalAmount != 25 || order.Status != PurchaseOrderOpen {
		t.Fatalf("unexpected order: %+v", order)
	}
}

func TestPurchaseRequestRejectsInvalidLine(t *testing.T) {
	_, err := NewPurchaseRequest("pr-1", PurchaseRequestInput{
		Number: "PR-001", SupplierID: "supplier-1", RequesterID: "buyer-1", ApproverID: "manager-1",
		Lines: []PurchaseLine{{ProductID: "product-1", Quantity: 0, UnitPrice: 12.5}},
	}, "workflow-pr-1", time.Now())
	if err == nil {
		t.Fatal("expected invalid line error")
	}
}
