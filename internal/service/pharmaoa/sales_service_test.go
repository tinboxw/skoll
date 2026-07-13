package pharmaoa

import (
	"context"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
)

func TestSalesOutboundChecksCustomerQualificationAndWritesLedger(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	customers := NewCustomerService(nil).(*customerService)
	customers.nowFn = func() time.Time { return now }
	customer, err := customers.Create(ctx, CustomerWriteInput{
		Code: "CUST-SALES-1", Name: "Qualified Hospital", Region: "East", OrganizationID: "org-1", OwnerID: "sales-1", ActorID: "sales-1",
		Qualifications: []domainpharma.CustomerQualification{{Name: "Medical Institution License", Number: "LIC-1", ExpiresAt: now.AddDate(0, 1, 0)}},
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	warehouses := NewWarehouseService(nil)
	warehouse, err := warehouses.Create(ctx, WarehouseWriteInput{Code: "WH-SALES", Name: "Sales Warehouse", Region: "East", ActorID: "warehouse-admin", Areas: []domainpharma.WarehouseArea{{ID: "area-1", Code: "A1", Name: "Area 1", Locations: []domainpharma.WarehouseLocation{{ID: "location-1", Code: "L1", Name: "Location 1"}}}}})
	if err != nil {
		t.Fatalf("create warehouse: %v", err)
	}
	inventory := NewInventoryService(nil)
	inbound, err := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "seed-inbound", ProductID: "product-1", WarehouseID: warehouse.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchNo: "B-001", ProductionDate: now.AddDate(0, -1, 0), ExpiresAt: now.AddDate(1, 0, 0), Quantity: 10, ActorID: "inventory-admin"})
	if err != nil {
		t.Fatalf("seed stock: %v", err)
	}
	service := NewSalesService(customers, warehouses, inventory, nil)
	order, err := service.CreateOrder(ctx, SalesOrderCreateInput{Number: "SO-001", CustomerID: customer.ID.String(), ActorID: "sales-1", Lines: []domainpharma.SalesLine{{ProductID: "product-1", Quantity: 4, UnitPrice: 12.5}}})
	if err != nil {
		t.Fatalf("create sales order: %v", err)
	}
	if order.TotalAmount != 50 {
		t.Fatalf("unexpected order total: %+v", order)
	}

	customers.nowFn = func() time.Time { return now.AddDate(0, 2, 0) }
	_, err = service.CreateOutbound(ctx, SalesOutboundCreateInput{Number: "OUT-001", SalesOrderID: order.ID.String(), WarehouseID: warehouse.ID.String(), AreaID: "area-1", LocationID: "location-1", ActorID: "sales-1", Lines: []domainpharma.SalesOutboundLine{{ProductID: "product-1", BatchID: inbound.Balance.BatchID, Quantity: 4}}})
	if err == nil || !strings.Contains(err.Error(), "qualification expired") {
		t.Fatalf("expected expired qualification denial, got %v", err)
	}
	balances, _ := inventory.ListBalances(ctx)
	if balances[0].Quantity != 10 {
		t.Fatalf("denied outbound changed inventory: %+v", balances[0])
	}

	customers.nowFn = func() time.Time { return now }
	outbound, err := service.CreateOutbound(ctx, SalesOutboundCreateInput{Number: "OUT-001", SalesOrderID: order.ID.String(), WarehouseID: warehouse.ID.String(), AreaID: "area-1", LocationID: "location-1", ActorID: "sales-1", Lines: []domainpharma.SalesOutboundLine{{ProductID: "product-1", BatchID: inbound.Balance.BatchID, Quantity: 4}}})
	if err != nil {
		t.Fatalf("create sales outbound: %v", err)
	}
	if outbound.Lines[0].LedgerID == "" {
		t.Fatalf("expected outbound ledger reference: %+v", outbound)
	}
	balances, _ = inventory.ListBalances(ctx)
	if balances[0].Quantity != 6 {
		t.Fatalf("expected stock quantity 6, got %+v", balances[0])
	}
	ledger, _ := inventory.ListLedger(ctx)
	if len(ledger) != 2 || ledger[1].QuantityDelta != -4 || ledger[1].BalanceAfter != 6 {
		t.Fatalf("unexpected inventory ledger: %+v", ledger)
	}
}

func TestSalesOutboundRejectsInsufficientStockBeforeMutation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	customers := NewCustomerService(nil).(*customerService)
	customers.nowFn = func() time.Time { return now }
	customer, _ := customers.Create(ctx, CustomerWriteInput{Code: "CUST-2", Name: "Hospital 2", Region: "East", OrganizationID: "org-1", OwnerID: "sales-1", ActorID: "sales-1"})
	warehouses := NewWarehouseService(nil)
	warehouse, _ := warehouses.Create(ctx, WarehouseWriteInput{Code: "WH-2", Name: "Warehouse 2", Region: "East", Areas: []domainpharma.WarehouseArea{{ID: "area-1", Code: "A1", Name: "Area 1", Locations: []domainpharma.WarehouseLocation{{ID: "location-1", Code: "L1", Name: "Location 1"}}}}})
	inventory := NewInventoryService(nil)
	inbound, _ := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "seed", ProductID: "p1", WarehouseID: warehouse.ID.String(), AreaID: "area-1", LocationID: "location-1", BatchNo: "B1", ExpiresAt: now.AddDate(1, 0, 0), Quantity: 2})
	service := NewSalesService(customers, warehouses, inventory, nil)
	order, _ := service.CreateOrder(ctx, SalesOrderCreateInput{Number: "SO-2", CustomerID: customer.ID.String(), ActorID: "sales-1", Lines: []domainpharma.SalesLine{{ProductID: "p1", Quantity: 3, UnitPrice: 1}}})
	_, err := service.CreateOutbound(ctx, SalesOutboundCreateInput{Number: "OUT-2", SalesOrderID: order.ID.String(), WarehouseID: warehouse.ID.String(), AreaID: "area-1", LocationID: "location-1", ActorID: "sales-1", Lines: []domainpharma.SalesOutboundLine{{ProductID: "p1", BatchID: inbound.Balance.BatchID, Quantity: 3}}})
	if err == nil || !strings.Contains(err.Error(), "insufficient available stock") {
		t.Fatalf("expected insufficient stock, got %v", err)
	}
	ledger, _ := inventory.ListLedger(ctx)
	if len(ledger) != 1 {
		t.Fatalf("failed outbound must not append ledger: %+v", ledger)
	}
}
