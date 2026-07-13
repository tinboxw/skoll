package pharmaoa

import (
	"context"
	"sync"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
)

func TestPurchaseServiceApprovalCreatesOrderOnce(t *testing.T) {
	ctx := context.Background()
	suppliers := NewSupplierService(nil)
	supplier, err := suppliers.Create(ctx, SupplierWriteInput{
		Code: "SUP-P-001", Name: "Qualified Supplier", Rating: 5, ActorID: "admin",
		Qualifications: []domainpharma.SupplierQualification{{ID: "q-1", Name: "License", Number: "L-1", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0)}},
	})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	service := NewPurchaseService(suppliers, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), nil)
	request, err := service.CreateRequest(ctx, PurchaseRequestCreateInput{
		Number: "PR-001", SupplierID: supplier.ID.String(), RequesterID: "buyer-1", ApproverID: "manager-1", Reason: "replenishment",
		Lines: []domainpharma.PurchaseLine{{ProductID: "product-1", Quantity: 10, UnitPrice: 3.5}},
	})
	if err != nil {
		t.Fatalf("CreateRequest error: %v", err)
	}
	order, err := service.ApproveRequest(ctx, request.ID.String(), PurchaseApprovalInput{ActorID: "manager-1", Comment: "approved"})
	if err != nil {
		t.Fatalf("ApproveRequest error: %v", err)
	}
	second, err := service.ApproveRequest(ctx, request.ID.String(), PurchaseApprovalInput{ActorID: "manager-1"})
	if err != nil {
		t.Fatalf("idempotent ApproveRequest error: %v", err)
	}
	if order.ID != second.ID || order.PurchaseRequestID != request.ID.String() || order.TotalAmount != 35 {
		t.Fatalf("unexpected orders: %+v %+v", order, second)
	}
	orders, _ := service.ListOrders(ctx)
	if len(orders) != 1 {
		t.Fatalf("expected one order, got %+v", orders)
	}
}

func TestPurchaseServiceBlocksExpiredSupplier(t *testing.T) {
	ctx := context.Background()
	suppliers := NewSupplierService(nil)
	supplier, err := suppliers.Create(ctx, SupplierWriteInput{Code: "SUP-X", Name: "Expired Supplier", Qualifications: []domainpharma.SupplierQualification{{ID: "q-x", Name: "License", Number: "X", ExpiresAt: time.Now().UTC().Add(-time.Hour)}}})
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	service := NewPurchaseService(suppliers, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), nil)
	_, err = service.CreateRequest(ctx, PurchaseRequestCreateInput{Number: "PR-X", SupplierID: supplier.ID.String(), RequesterID: "buyer-1", ApproverID: "manager-1", Lines: []domainpharma.PurchaseLine{{ProductID: "product-1", Quantity: 1, UnitPrice: 1}}})
	if err == nil {
		t.Fatal("expected expired supplier to block purchase request")
	}
}

func TestPurchaseServiceConcurrentApprovalCreatesOneOrder(t *testing.T) {
	ctx := context.Background()
	suppliers := NewSupplierService(nil)
	supplier, err := suppliers.Create(ctx, SupplierWriteInput{Code: "SUP-C", Name: "Concurrent Supplier", Qualifications: []domainpharma.SupplierQualification{{ID: "q-c", Name: "License", Number: "C", ExpiresAt: time.Now().UTC().AddDate(1, 0, 0)}}})
	if err != nil {
		t.Fatal(err)
	}
	service := NewPurchaseService(suppliers, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), nil)
	request, err := service.CreateRequest(ctx, PurchaseRequestCreateInput{Number: "PR-C", SupplierID: supplier.ID.String(), RequesterID: "buyer-1", ApproverID: "manager-1", Lines: []domainpharma.PurchaseLine{{ProductID: "product-1", Quantity: 1, UnitPrice: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan string, 2)
	errors := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			order, approveErr := service.ApproveRequest(ctx, request.ID.String(), PurchaseApprovalInput{ActorID: "manager-1"})
			if approveErr != nil {
				errors <- approveErr
				return
			}
			results <- order.ID.String()
		}()
	}
	wg.Wait()
	close(results)
	close(errors)
	for approveErr := range errors {
		t.Fatalf("concurrent approval error: %v", approveErr)
	}
	var first string
	for id := range results {
		if first == "" {
			first = id
		} else if id != first {
			t.Fatalf("expected same order, got %s and %s", first, id)
		}
	}
	orders, _ := service.ListOrders(ctx)
	if len(orders) != 1 {
		t.Fatalf("expected one order, got %d", len(orders))
	}
}
