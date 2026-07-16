package pharmaoa

import (
	"context"
	"fmt"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
)

const pharmaOAPerformanceSampleSize = 600

func TestPharmaOALargeListPaginationAndCommonQueries(t *testing.T) {
	ctx := context.Background()
	employees := NewEmployeeService(nil)
	customers := NewCustomerService(nil)

	for i := 0; i < pharmaOAPerformanceSampleSize; i++ {
		code := fmt.Sprintf("EMP-PERF-%04d", i)
		if _, err := employees.Create(ctx, EmployeeWriteInput{
			Code:         code,
			Name:         fmt.Sprintf("Employee %04d", i),
			DepartmentID: "sales",
			PositionID:   "representative",
			ActorID:      "seed",
		}); err != nil {
			t.Fatalf("create employee %s: %v", code, err)
		}

		ownerID := "sales-a"
		region := "East"
		if i%2 == 1 {
			ownerID = "sales-b"
			region = "West"
		}
		customerCode := fmt.Sprintf("CUST-PERF-%04d", i)
		if _, err := customers.Create(ctx, CustomerWriteInput{
			Code:           customerCode,
			Name:           fmt.Sprintf("Customer %04d", i),
			Region:         region,
			OrganizationID: "org-main",
			OwnerID:        ownerID,
			Contacts:       []domainpharma.CustomerContact{{Name: "Contact"}},
			ActorID:        ownerID,
		}); err != nil {
			t.Fatalf("create customer %s: %v", customerCode, err)
		}
	}

	started := time.Now()
	defaultPage, err := employees.List(ctx, EmployeeListInput{})
	if err != nil {
		t.Fatalf("list default employee page: %v", err)
	}
	if len(defaultPage) != 50 {
		t.Fatalf("default employee page size=%d, want 50", len(defaultPage))
	}

	cappedPage, err := employees.List(ctx, EmployeeListInput{Offset: 200, Limit: 500})
	if err != nil {
		t.Fatalf("list capped employee page: %v", err)
	}
	if len(cappedPage) != 200 || cappedPage[0].Code != "EMP-PERF-0200" {
		t.Fatalf("unexpected capped employee page: len=%d first=%s", len(cappedPage), cappedPage[0].Code)
	}

	keywordPage, err := employees.List(ctx, EmployeeListInput{Keyword: "0599", Status: "active", Limit: 20})
	if err != nil {
		t.Fatalf("query employees: %v", err)
	}
	if len(keywordPage) != 1 || keywordPage[0].Code != "EMP-PERF-0599" {
		t.Fatalf("unexpected employee query result: %+v", keywordPage)
	}

	ownedPage, err := customers.List(ctx, CustomerListInput{
		Region: "East",
		Limit:  200,
		Scope:  CustomerAccessScope{OwnerID: "sales-a"},
	})
	if err != nil {
		t.Fatalf("query scoped customers: %v", err)
	}
	if len(ownedPage) != 200 {
		t.Fatalf("scoped customer page size=%d, want 200", len(ownedPage))
	}
	for _, item := range ownedPage {
		if item.OwnerID != "sales-a" || item.Region != "East" {
			t.Fatalf("customer escaped data scope: %+v", item)
		}
	}

	elapsed := time.Since(started)
	t.Logf("sample=%d default=%d capped=%d scoped=%d query_duration=%s", pharmaOAPerformanceSampleSize, len(defaultPage), len(cappedPage), len(ownedPage), elapsed)
	if elapsed > 2*time.Second {
		t.Fatalf("representative list queries took %s, limit is 2s", elapsed)
	}
}
