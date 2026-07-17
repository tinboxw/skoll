package pharmaoa

import (
	"context"
	"testing"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
)

func TestMasterDataServicesDoNotOverwriteIDsAfterReconstruction(t *testing.T) {
	ctx := context.Background()

	t.Run("employee", func(t *testing.T) {
		repo := pharmaoarepo.NewMemoryEmployeeRepository()
		first, err := NewEmployeeService(nil, repo).Create(ctx, EmployeeWriteInput{Code: "E-001", Name: "Employee One", DepartmentID: "dept-1", PositionID: "position-1"})
		if err != nil {
			t.Fatalf("create first employee: %v", err)
		}
		secondService := NewEmployeeService(nil, repo)
		second, err := secondService.Create(ctx, EmployeeWriteInput{Code: "E-002", Name: "Employee Two", DepartmentID: "dept-1", PositionID: "position-1"})
		if err != nil {
			t.Fatalf("create after service reconstruction: %v", err)
		}
		assertReconstructedMasterData(t, first.ID.String(), second.ID.String(), func() (int, error) {
			items, listErr := secondService.List(ctx, EmployeeListInput{})
			return len(items), listErr
		})
	})

	t.Run("product", func(t *testing.T) {
		repo := pharmaoarepo.NewMemoryProductRepository()
		first, err := NewProductService(nil, repo).Create(ctx, productRepositoryFixture("P-001"))
		if err != nil {
			t.Fatalf("create first product: %v", err)
		}
		secondService := NewProductService(nil, repo)
		second, err := secondService.Create(ctx, productRepositoryFixture("P-002"))
		if err != nil {
			t.Fatalf("create after service reconstruction: %v", err)
		}
		assertReconstructedMasterData(t, first.ID.String(), second.ID.String(), func() (int, error) {
			items, listErr := secondService.List(ctx, ProductListInput{})
			return len(items), listErr
		})
	})

	t.Run("supplier", func(t *testing.T) {
		repo := pharmaoarepo.NewMemorySupplierRepository()
		first, err := NewSupplierService(nil, repo).Create(ctx, SupplierWriteInput{Code: "S-001", Name: "Supplier One"})
		if err != nil {
			t.Fatalf("create first supplier: %v", err)
		}
		secondService := NewSupplierService(nil, repo)
		second, err := secondService.Create(ctx, SupplierWriteInput{Code: "S-002", Name: "Supplier Two"})
		if err != nil {
			t.Fatalf("create after service reconstruction: %v", err)
		}
		assertReconstructedMasterData(t, first.ID.String(), second.ID.String(), func() (int, error) {
			items, listErr := secondService.List(ctx, SupplierListInput{})
			return len(items), listErr
		})
	})

	t.Run("customer", func(t *testing.T) {
		repo := pharmaoarepo.NewMemoryCustomerRepository()
		first, err := NewCustomerService(nil, repo).Create(ctx, CustomerWriteInput{Code: "C-001", Name: "Customer One", Region: "East", ActorID: "owner-1"})
		if err != nil {
			t.Fatalf("create first customer: %v", err)
		}
		secondService := NewCustomerService(nil, repo)
		second, err := secondService.Create(ctx, CustomerWriteInput{Code: "C-002", Name: "Customer Two", Region: "East", ActorID: "owner-1"})
		if err != nil {
			t.Fatalf("create after service reconstruction: %v", err)
		}
		assertReconstructedMasterData(t, first.ID.String(), second.ID.String(), func() (int, error) {
			items, listErr := secondService.List(ctx, CustomerListInput{Scope: CustomerAccessScope{IncludeAll: true}})
			return len(items), listErr
		})
	})

	t.Run("warehouse", func(t *testing.T) {
		repo := pharmaoarepo.NewMemoryWarehouseRepository()
		first, err := NewWarehouseService(nil, repo).Create(ctx, warehouseRepositoryFixture("W-001"))
		if err != nil {
			t.Fatalf("create first warehouse: %v", err)
		}
		secondService := NewWarehouseService(nil, repo)
		second, err := secondService.Create(ctx, warehouseRepositoryFixture("W-002"))
		if err != nil {
			t.Fatalf("create after service reconstruction: %v", err)
		}
		assertReconstructedMasterData(t, first.ID.String(), second.ID.String(), func() (int, error) {
			items, listErr := secondService.List(ctx, WarehouseListInput{})
			return len(items), listErr
		})
	})
}

func assertReconstructedMasterData(t *testing.T, firstID, secondID string, list func() (int, error)) {
	t.Helper()
	if firstID == secondID {
		t.Fatalf("service reconstruction reused id %s", firstID)
	}
	count, err := list()
	if err != nil || count != 2 {
		t.Fatalf("list after service reconstruction: count=%d err=%v", count, err)
	}
}

func productRepositoryFixture(code string) ProductWriteInput {
	return ProductWriteInput{
		Code: code, Name: "Product " + code, Spec: "10mg", DosageForm: "tablet", Manufacturer: "Skoll Pharma",
		ApprovalNumber: "APP-" + code,
	}
}

func warehouseRepositoryFixture(code string) WarehouseWriteInput {
	return WarehouseWriteInput{
		Code: code, Name: "Warehouse " + code, Region: "East",
		Areas: []domainpharma.WarehouseArea{{
			ID: "area-1", Code: "AREA-1", Name: "Area One", Status: domainpharma.WarehouseStatusEnabled,
			Locations: []domainpharma.WarehouseLocation{{ID: "location-1", Code: "LOCATION-1", Name: "Location One", Status: domainpharma.WarehouseStatusEnabled}},
		}},
	}
}
