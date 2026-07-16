package pharmaoa

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

const (
	DemoSeedStateReady    = "ready"
	DemoSeedStateApplying = "applying"
	DemoSeedStateApplied  = "applied"
	DemoSeedStateFailed   = "failed"
)

type DemoSeedService interface {
	Status(ctx context.Context, actorID string) (DemoSeedSnapshot, error)
	Apply(ctx context.Context, actorID string) (DemoSeedSnapshot, error)
}

type DemoSeedEntities struct {
	EmployeeID         string `json:"employeeId"`
	ProductID          string `json:"productId"`
	SupplierID         string `json:"supplierId"`
	CustomerID         string `json:"customerId"`
	WarehouseID        string `json:"warehouseId"`
	PurchaseRequestID  string `json:"purchaseRequestId"`
	WorkflowInstanceID string `json:"workflowInstanceId"`
	PurchaseOrderID    string `json:"purchaseOrderId"`
	PurchaseInboundID  string `json:"purchaseInboundId"`
	BatchID            string `json:"batchId"`
	SalesOrderID       string `json:"salesOrderId"`
	SalesOutboundID    string `json:"salesOutboundId"`
	CustomerFollowUpID string `json:"customerFollowUpId"`
}

type DemoSeedSnapshot struct {
	State     string           `json:"state"`
	Stage     string           `json:"stage"`
	Applied   bool             `json:"applied"`
	Reused    bool             `json:"reused"`
	ActorID   string           `json:"actorId,omitempty"`
	AppliedAt *time.Time       `json:"appliedAt,omitempty"`
	Error     string           `json:"error,omitempty"`
	Entities  DemoSeedEntities `json:"entities"`
	Counts    map[string]int   `json:"counts"`
}

type DemoSeedDependencies struct {
	Employees  EmployeeService
	Products   ProductService
	Suppliers  SupplierService
	Customers  CustomerService
	Warehouses WarehouseService
	Purchases  PurchaseService
	Inbounds   PurchaseInboundService
	Sales      SalesService
	Inventory  InventoryService
	FollowUps  CustomerFollowUpService
	Audit      auditsvc.Service
}

type demoSeedService struct {
	applyMu  sync.Mutex
	mu       sync.RWMutex
	deps     DemoSeedDependencies
	nowFn    func() time.Time
	snapshot DemoSeedSnapshot
}

func NewDemoSeedService(deps DemoSeedDependencies) DemoSeedService {
	return &demoSeedService{
		deps:     deps,
		nowFn:    func() time.Time { return time.Now().UTC() },
		snapshot: DemoSeedSnapshot{State: DemoSeedStateReady, Stage: "ready", Counts: map[string]int{}},
	}
}

func (s *demoSeedService) Status(ctx context.Context, actorID string) (DemoSeedSnapshot, error) {
	actorID, err := requireDemoSeedActor(actorID)
	if err != nil {
		return DemoSeedSnapshot{}, err
	}
	s.mu.RLock()
	out := cloneDemoSeedSnapshot(s.snapshot)
	s.mu.RUnlock()
	s.appendAudit(ctx, actorID, "pharma_oa.seed.read", map[string]any{"state": out.State, "stage": out.Stage})
	return out, nil
}

func (s *demoSeedService) Apply(ctx context.Context, actorID string) (DemoSeedSnapshot, error) {
	actorID, err := requireDemoSeedActor(actorID)
	if err != nil {
		return DemoSeedSnapshot{}, err
	}
	if err := s.validateDependencies(); err != nil {
		return DemoSeedSnapshot{}, err
	}
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	s.mu.RLock()
	current := cloneDemoSeedSnapshot(s.snapshot)
	s.mu.RUnlock()
	if current.State == DemoSeedStateApplied {
		current.Reused = true
		s.appendAudit(ctx, actorID, "pharma_oa.seed.apply", map[string]any{"state": current.State, "reused": true})
		return current, nil
	}
	if current.State == DemoSeedStateFailed {
		return current, fmt.Errorf("demo seed previously failed at %s: %s", current.Stage, current.Error)
	}

	s.setProgress(actorID, "employee")
	now := s.nowFn().UTC()
	employee, err := s.deps.Employees.Create(ctx, EmployeeWriteInput{
		Code: "DEMO-EMP-001", Name: "Demo Quality Pharmacist", DepartmentID: "demo-quality", PositionID: "demo-pharmacist",
		Phone: "13800000001", Email: "demo.pharmacist@skoll.local", ActorID: actorID,
		Certificates: []domainpharma.EmployeeCertificate{{ID: "demo-employee-license", Name: "Licensed Pharmacist Certificate", Number: "DEMO-LP-001", ExpiresAt: now.AddDate(0, 0, 20)}},
	})
	if err != nil {
		return s.fail(ctx, actorID, "employee", err)
	}

	s.setProgress(actorID, "product")
	product, err := s.deps.Products.Create(ctx, ProductWriteInput{
		Code: "DEMO-DRUG-001", Name: "Demo Cold-chain Medicine", Spec: "10mg*20", DosageForm: "tablet",
		Manufacturer: "Skoll Demo Pharma", ApprovalNumber: "DEMO-NMPA-001", ActorID: actorID,
		Temperature: domainpharma.ProductTemperature{Required: true, MinCelsius: 2, MaxCelsius: 8},
	})
	if err != nil {
		return s.fail(ctx, actorID, "product", err)
	}

	s.setProgress(actorID, "supplier")
	supplier, err := s.deps.Suppliers.Create(ctx, SupplierWriteInput{
		Code: "DEMO-SUP-001", Name: "Demo Qualified Supplier", Rating: 5, ActorID: actorID,
		Qualifications: []domainpharma.SupplierQualification{{ID: "demo-supplier-license", Name: "Drug Distribution License", Number: "DEMO-SUP-LIC-001", ExpiresAt: now.AddDate(1, 0, 0)}},
	})
	if err != nil {
		return s.fail(ctx, actorID, "supplier", err)
	}

	s.setProgress(actorID, "customer")
	customer, err := s.deps.Customers.Create(ctx, CustomerWriteInput{
		Code: "DEMO-CUST-001", Name: "Demo Central Hospital", Region: "East", OrganizationID: "demo-org", OwnerID: actorID, Rating: 5, ActorID: actorID,
		Scope:          CustomerAccessScope{IncludeAll: true},
		Qualifications: []domainpharma.CustomerQualification{{ID: "demo-customer-license", Name: "Medical Institution License", Number: "DEMO-CUST-LIC-001", ExpiresAt: now.AddDate(1, 0, 0)}},
	})
	if err != nil {
		return s.fail(ctx, actorID, "customer", err)
	}

	s.setProgress(actorID, "warehouse")
	warehouse, err := s.deps.Warehouses.Create(ctx, WarehouseWriteInput{
		Code: "DEMO-WH-001", Name: "Demo Cold-chain Warehouse", Region: "East", ActorID: actorID,
		Temperature: domainpharma.WarehouseTemperature{Controlled: true, MinCelsius: 2, MaxCelsius: 8},
		Areas:       []domainpharma.WarehouseArea{{ID: "demo-cold-area", Code: "COLD", Name: "Cold-chain Area", Temperature: domainpharma.WarehouseTemperature{Controlled: true, MinCelsius: 2, MaxCelsius: 8}, Locations: []domainpharma.WarehouseLocation{{ID: "demo-cold-location", Code: "COLD-01", Name: "Cold-chain Location", Temperature: domainpharma.WarehouseTemperature{Controlled: true, MinCelsius: 2, MaxCelsius: 8}}}}},
	})
	if err != nil {
		return s.fail(ctx, actorID, "warehouse", err)
	}

	s.setProgress(actorID, "purchase_workflow")
	request, err := s.deps.Purchases.CreateRequest(ctx, PurchaseRequestCreateInput{
		Number: "PR-DEMO-001", SupplierID: supplier.ID.String(), RequesterID: actorID, ApproverID: actorID,
		Reason: "Initialize the Pharma OA demonstration inventory", Lines: []domainpharma.PurchaseLine{{ProductID: product.ID.String(), Quantity: 100, UnitPrice: 1250}},
	})
	if err != nil {
		return s.fail(ctx, actorID, "purchase_workflow", err)
	}
	order, err := s.deps.Purchases.ApproveRequest(ctx, request.ID.String(), PurchaseApprovalInput{ActorID: actorID, Comment: "Approved by demo seed"})
	if err != nil {
		return s.fail(ctx, actorID, "purchase_approval", err)
	}

	s.setProgress(actorID, "purchase_inbound")
	inbound, err := s.deps.Inbounds.Create(ctx, PurchaseInboundCreateInput{
		Number: "IN-DEMO-001", PurchaseOrderID: order.ID.String(), WarehouseID: warehouse.ID.String(), AreaID: "demo-cold-area", LocationID: "demo-cold-location", ActorID: actorID,
		Lines: []domainpharma.PurchaseInboundLine{{ProductID: product.ID.String(), Quantity: 100, BatchNo: "DEMO-BATCH-001", ProductionDate: now.AddDate(0, -1, 0), ExpiresAt: now.AddDate(0, 6, 0)}},
	})
	if err != nil {
		return s.fail(ctx, actorID, "purchase_inbound", err)
	}

	s.setProgress(actorID, "sales_outbound")
	salesOrder, err := s.deps.Sales.CreateOrder(ctx, SalesOrderCreateInput{Number: "SO-DEMO-001", CustomerID: customer.ID.String(), ActorID: actorID, Lines: []domainpharma.SalesLine{{ProductID: product.ID.String(), Quantity: 12, UnitPrice: 1680}}})
	if err != nil {
		return s.fail(ctx, actorID, "sales_order", err)
	}
	outbound, err := s.deps.Sales.CreateOutbound(ctx, SalesOutboundCreateInput{
		Number: "OUT-DEMO-001", SalesOrderID: salesOrder.ID.String(), WarehouseID: warehouse.ID.String(), AreaID: "demo-cold-area", LocationID: "demo-cold-location", ActorID: actorID,
		Lines: []domainpharma.SalesOutboundLine{{ProductID: product.ID.String(), BatchID: inbound.Lines[0].BatchID, Quantity: 12}},
	})
	if err != nil {
		return s.fail(ctx, actorID, "sales_outbound", err)
	}

	s.setProgress(actorID, "customer_follow_up")
	followUp, err := s.deps.FollowUps.Create(ctx, CustomerFollowUpCreateInput{
		CustomerID: customer.ID.String(), ContactName: "Demo Pharmacy Director", Channel: domainpharma.CustomerFollowUpOnsite,
		ScheduledAt: now.AddDate(0, 0, 2), NextAction: "Review delivery and plan the next order", ActorID: actorID,
		Scope: CustomerFollowUpAccessScope{IncludeAll: true},
	})
	if err != nil {
		return s.fail(ctx, actorID, "customer_follow_up", err)
	}

	balances, err := s.deps.Inventory.ListBalances(ctx)
	if err != nil {
		return s.fail(ctx, actorID, "inventory_summary", err)
	}
	ledger, err := s.deps.Inventory.ListLedger(ctx)
	if err != nil {
		return s.fail(ctx, actorID, "inventory_summary", err)
	}
	reminders, err := s.deps.Employees.QualificationReminders(ctx, 30)
	if err != nil {
		return s.fail(ctx, actorID, "qualification_summary", err)
	}

	appliedAt := s.nowFn().UTC()
	result := DemoSeedSnapshot{
		State: DemoSeedStateApplied, Stage: "complete", Applied: true, ActorID: actorID, AppliedAt: &appliedAt,
		Entities: DemoSeedEntities{
			EmployeeID: employee.ID.String(), ProductID: product.ID.String(), SupplierID: supplier.ID.String(), CustomerID: customer.ID.String(), WarehouseID: warehouse.ID.String(),
			PurchaseRequestID: request.ID.String(), WorkflowInstanceID: request.WorkflowInstanceID, PurchaseOrderID: order.ID.String(), PurchaseInboundID: inbound.ID.String(), BatchID: inbound.Lines[0].BatchID,
			SalesOrderID: salesOrder.ID.String(), SalesOutboundID: outbound.ID.String(), CustomerFollowUpID: followUp.ID.String(),
		},
		Counts: map[string]int{"employees": 1, "products": 1, "suppliers": 1, "customers": 1, "warehouses": 1, "stockBalances": len(balances), "stockLedgerEntries": len(ledger), "workflows": 1, "qualificationReminders": len(reminders), "customerFollowUps": 1},
	}
	s.mu.Lock()
	s.snapshot = cloneDemoSeedSnapshot(result)
	s.mu.Unlock()
	s.appendAudit(ctx, actorID, "pharma_oa.seed.apply", map[string]any{"state": result.State, "reused": false, "counts": result.Counts})
	return cloneDemoSeedSnapshot(result), nil
}

func (s *demoSeedService) validateDependencies() error {
	if s == nil || s.deps.Employees == nil || s.deps.Products == nil || s.deps.Suppliers == nil || s.deps.Customers == nil || s.deps.Warehouses == nil || s.deps.Purchases == nil || s.deps.Inbounds == nil || s.deps.Sales == nil || s.deps.Inventory == nil || s.deps.FollowUps == nil {
		return fmt.Errorf("demo seed dependencies are required")
	}
	return nil
}

func (s *demoSeedService) setProgress(actorID, stage string) {
	s.mu.Lock()
	s.snapshot = DemoSeedSnapshot{State: DemoSeedStateApplying, Stage: stage, ActorID: actorID, Counts: map[string]int{}}
	s.mu.Unlock()
}

func (s *demoSeedService) fail(ctx context.Context, actorID, stage string, err error) (DemoSeedSnapshot, error) {
	result := DemoSeedSnapshot{State: DemoSeedStateFailed, Stage: stage, ActorID: actorID, Error: err.Error(), Counts: map[string]int{}}
	s.mu.Lock()
	s.snapshot = cloneDemoSeedSnapshot(result)
	s.mu.Unlock()
	s.appendAudit(ctx, actorID, "pharma_oa.seed.fail", map[string]any{"stage": stage, "error": err.Error()})
	return result, fmt.Errorf("demo seed failed at %s: %w", stage, err)
}

func (s *demoSeedService) appendAudit(ctx context.Context, actorID, action string, detail map[string]any) {
	if s != nil && s.deps.Audit != nil {
		_, _ = s.deps.Audit.Append(ctx, actorID, action, "pharma_oa_demo_seed", "pharma_oa", detail)
	}
}

func requireDemoSeedActor(actorID string) (string, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return "", fmt.Errorf("actorId is required")
	}
	return actorID, nil
}

func cloneDemoSeedSnapshot(in DemoSeedSnapshot) DemoSeedSnapshot {
	out := in
	if in.AppliedAt != nil {
		value := *in.AppliedAt
		out.AppliedAt = &value
	}
	out.Counts = make(map[string]int, len(in.Counts))
	for key, value := range in.Counts {
		out.Counts[key] = value
	}
	return out
}
