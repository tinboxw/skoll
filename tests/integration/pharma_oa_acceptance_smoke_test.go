package integration_test

import (
	"context"
	"testing"

	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestPharmaOAEndToEndAcceptanceSmoke(t *testing.T) {
	ctx := context.Background()
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	employees := pharmaoasvc.NewEmployeeService(audit)
	products := pharmaoasvc.NewProductService(audit)
	suppliers := pharmaoasvc.NewSupplierService(audit)
	customers := pharmaoasvc.NewCustomerService(audit)
	warehouses := pharmaoasvc.NewWarehouseService(audit)
	inventory := pharmaoasvc.NewInventoryService(audit)
	workflow := workflowsvc.NewService(workflowsvc.NewMemoryRepository())
	purchases := pharmaoasvc.NewPurchaseService(suppliers, workflow, audit)
	inbounds := pharmaoasvc.NewPurchaseInboundService(purchases, warehouses, inventory, audit)
	sales := pharmaoasvc.NewSalesService(customers, warehouses, inventory, audit)
	followUps := pharmaoasvc.NewCustomerFollowUpService(customers, audit)
	seed := pharmaoasvc.NewDemoSeedService(pharmaoasvc.DemoSeedDependencies{
		Employees: employees, Products: products, Suppliers: suppliers, Customers: customers, Warehouses: warehouses,
		Purchases: purchases, Inbounds: inbounds, Sales: sales, Inventory: inventory, FollowUps: followUps, Audit: audit,
	})

	result, err := seed.Apply(ctx, "acceptance-admin")
	if err != nil || !result.Applied {
		t.Fatalf("apply demo scenario: %+v err=%v", result, err)
	}

	employeeItems, err := employees.List(ctx, pharmaoasvc.EmployeeListInput{})
	if err != nil || len(employeeItems) != 1 || employeeItems[0].ID.String() != result.Entities.EmployeeID {
		t.Fatalf("employee onboarding failed: %+v err=%v", employeeItems, err)
	}
	requestItems, err := purchases.ListRequests(ctx)
	if err != nil || len(requestItems) != 1 || requestItems[0].ID.String() != result.Entities.PurchaseRequestID {
		t.Fatalf("purchase request failed: %+v err=%v", requestItems, err)
	}
	inboundItems, err := inbounds.List(ctx)
	if err != nil || len(inboundItems) != 1 || inboundItems[0].ID.String() != result.Entities.PurchaseInboundID {
		t.Fatalf("purchase inbound failed: %+v err=%v", inboundItems, err)
	}
	outboundItems, err := sales.ListOutbounds(ctx)
	if err != nil || len(outboundItems) != 1 || outboundItems[0].ID.String() != result.Entities.SalesOutboundID {
		t.Fatalf("sales outbound failed: %+v err=%v", outboundItems, err)
	}

	instance, err := workflow.GetInstance(ctx, shared.ID(result.Entities.WorkflowInstanceID))
	if err != nil || string(instance.Status) != "approved" {
		t.Fatalf("purchase workflow was not approved: %+v err=%v", instance, err)
	}
	reminders, err := employees.QualificationReminders(ctx, 30)
	if err != nil || len(reminders) != 1 || reminders[0].EmployeeID != result.Entities.EmployeeID {
		t.Fatalf("qualification alert failed: %+v err=%v", reminders, err)
	}
	followUpItems, err := followUps.List(ctx, pharmaoasvc.CustomerFollowUpListInput{ActorID: "acceptance-admin"})
	if err != nil || len(followUpItems) != 1 || followUpItems[0].ID.String() != result.Entities.CustomerFollowUpID {
		t.Fatalf("customer follow-up failed: %+v err=%v", followUpItems, err)
	}
	balances, _ := inventory.ListBalances(ctx)
	ledger, _ := inventory.ListLedger(ctx)
	if len(balances) != 1 || balances[0].Quantity != 88 || len(ledger) != 2 {
		t.Fatalf("inventory evidence mismatch: balances=%+v ledger=%+v", balances, ledger)
	}
	reused, err := seed.Apply(ctx, "acceptance-admin")
	if err != nil || !reused.Reused {
		t.Fatalf("repeat acceptance seed must be idempotent: %+v err=%v", reused, err)
	}
}
