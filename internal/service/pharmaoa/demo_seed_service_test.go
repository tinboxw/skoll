package pharmaoa

import (
	"context"
	"testing"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestDemoSeedApplyBuildsCompleteIdempotentScenario(t *testing.T) {
	ctx := context.Background()
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	employees := NewEmployeeService(audit)
	products := NewProductService(audit)
	suppliers := NewSupplierService(audit)
	customers := NewCustomerService(audit)
	warehouses := NewWarehouseService(audit)
	inventory := NewInventoryService(audit)
	workflow := workflowsvc.NewService(workflowsvc.NewMemoryRepository())
	purchases := NewPurchaseService(suppliers, workflow, audit)
	inbounds := NewPurchaseInboundService(purchases, warehouses, inventory, audit)
	sales := NewSalesService(customers, warehouses, inventory, audit)
	followUps := NewCustomerFollowUpService(customers, audit)
	service := NewDemoSeedService(DemoSeedDependencies{
		Employees: employees, Products: products, Suppliers: suppliers, Customers: customers, Warehouses: warehouses,
		Purchases: purchases, Inbounds: inbounds, Sales: sales, Inventory: inventory, FollowUps: followUps, Audit: audit,
	})

	ready, err := service.Status(ctx, "admin-1")
	if err != nil || ready.State != DemoSeedStateReady || ready.Applied {
		t.Fatalf("unexpected initial status: %+v err=%v", ready, err)
	}
	result, err := service.Apply(ctx, "admin-1")
	if err != nil {
		t.Fatalf("apply demo seed: %v", err)
	}
	if !result.Applied || result.State != DemoSeedStateApplied || result.Stage != "complete" || result.Reused {
		t.Fatalf("unexpected seed result: %+v", result)
	}
	if result.Entities.EmployeeID == "" || result.Entities.ProductID == "" || result.Entities.SupplierID == "" || result.Entities.CustomerID == "" || result.Entities.WarehouseID == "" || result.Entities.WorkflowInstanceID == "" || result.Entities.BatchID == "" || result.Entities.SalesOutboundID == "" || result.Entities.CustomerFollowUpID == "" {
		t.Fatalf("seed entity references are incomplete: %+v", result.Entities)
	}
	if result.Counts["stockBalances"] != 1 || result.Counts["stockLedgerEntries"] != 2 || result.Counts["qualificationReminders"] != 1 || result.Counts["customerFollowUps"] != 1 {
		t.Fatalf("unexpected seed counts: %+v", result.Counts)
	}

	balances, _ := inventory.ListBalances(ctx)
	if len(balances) != 1 || balances[0].Quantity != 88 || balances[0].BatchID != result.Entities.BatchID {
		t.Fatalf("unexpected seeded stock: %+v", balances)
	}
	requests, _ := purchases.ListRequests(ctx)
	orders, _ := purchases.ListOrders(ctx)
	inboundItems, _ := inbounds.List(ctx)
	salesOrders, _ := sales.ListOrders(ctx)
	outbounds, _ := sales.ListOutbounds(ctx)
	if len(requests) != 1 || len(orders) != 1 || len(inboundItems) != 1 || len(salesOrders) != 1 || len(outbounds) != 1 {
		t.Fatalf("seeded business chain is incomplete")
	}

	reused, err := service.Apply(ctx, "admin-2")
	if err != nil || !reused.Reused || reused.Entities != result.Entities {
		t.Fatalf("second apply should reuse the completed seed: %+v err=%v", reused, err)
	}
	ledger, _ := inventory.ListLedger(ctx)
	if len(ledger) != 2 {
		t.Fatalf("idempotent apply duplicated inventory ledger: %+v", ledger)
	}
	records, err := audit.ListByActor(ctx, "admin-1", 500)
	if err != nil || !hasDemoSeedAudit(records, "pharma_oa.seed.read") || !hasDemoSeedAudit(records, "pharma_oa.seed.apply") {
		t.Fatalf("seed audit chain missing: %+v err=%v", records, err)
	}
}

func TestDemoSeedApplyRequiresDependenciesAndActor(t *testing.T) {
	service := NewDemoSeedService(DemoSeedDependencies{})
	if _, err := service.Apply(context.Background(), ""); err == nil {
		t.Fatal("expected actor validation error")
	}
	if _, err := service.Apply(context.Background(), "admin"); err == nil {
		t.Fatal("expected dependency validation error")
	}
}

func hasDemoSeedAudit(records []*domainaudit.Record, action string) bool {
	for _, record := range records {
		if record != nil && record.Action == action {
			return true
		}
	}
	return false
}
