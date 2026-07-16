package benchmark

import (
	"context"
	"fmt"
	"testing"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

const pharmaOABenchmarkRecords = 1000

func BenchmarkPharmaOAEmployeePagedQuery(b *testing.B) {
	ctx := context.Background()
	service := pharmaoasvc.NewEmployeeService(nil)
	for i := 0; i < pharmaOABenchmarkRecords; i++ {
		_, err := service.Create(ctx, pharmaoasvc.EmployeeWriteInput{
			Code:         fmt.Sprintf("EMP-BENCH-%04d", i),
			Name:         fmt.Sprintf("Employee %04d", i),
			DepartmentID: "sales",
			PositionID:   "representative",
			ActorID:      "benchmark",
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.List(ctx, pharmaoasvc.EmployeeListInput{Keyword: "employee", Status: "active", Offset: 400, Limit: 50}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPharmaOACustomerScopedQuery(b *testing.B) {
	ctx := context.Background()
	service := pharmaoasvc.NewCustomerService(nil)
	for i := 0; i < pharmaOABenchmarkRecords; i++ {
		ownerID := "sales-a"
		region := "East"
		if i%2 == 1 {
			ownerID = "sales-b"
			region = "West"
		}
		_, err := service.Create(ctx, pharmaoasvc.CustomerWriteInput{
			Code:           fmt.Sprintf("CUST-BENCH-%04d", i),
			Name:           fmt.Sprintf("Customer %04d", i),
			Region:         region,
			OrganizationID: "org-main",
			OwnerID:        ownerID,
			Contacts:       []domainpharma.CustomerContact{{Name: "Contact"}},
			ActorID:        ownerID,
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.List(ctx, pharmaoasvc.CustomerListInput{
			Keyword: "customer",
			Region:  "East",
			Offset:  200,
			Limit:   50,
			Scope:   pharmaoasvc.CustomerAccessScope{OwnerID: "sales-a"},
		}); err != nil {
			b.Fatal(err)
		}
	}
}
