package pharmaoa

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMasterDataExchangeImportExportAllResources(t *testing.T) {
	service := newMasterDataExchangeFixture()
	ctx := context.Background()

	cases := []struct {
		resource string
		row      map[string]string
		wantCode string
	}{
		{
			resource: MasterDataResourceEmployees,
			row: map[string]string{
				"code":                 "EMP-IMP-001",
				"name":                 "Alice",
				"departmentId":         "quality",
				"positionId":           "qa",
				"phone":                "10086",
				"email":                "alice@example.test",
				"certificateName":      "GSP",
				"certificateNumber":    "GSP-001",
				"certificateExpiresAt": "2026-12-31",
			},
			wantCode: "EMP-IMP-001",
		},
		{
			resource: MasterDataResourceProducts,
			row: map[string]string{
				"code":                  "DRUG-IMP-001",
				"name":                  "Cold Chain Drug",
				"spec":                  "10ml",
				"dosageForm":            "Injection",
				"manufacturer":          "Acme Pharma",
				"approvalNumber":        "NMPA-001",
				"temperatureRequired":   "true",
				"temperatureMinCelsius": "2",
				"temperatureMaxCelsius": "8",
			},
			wantCode: "DRUG-IMP-001",
		},
		{
			resource: MasterDataResourceSuppliers,
			row: map[string]string{
				"code":                   "SUP-IMP-001",
				"name":                   "Supplier One",
				"rating":                 "5",
				"contactName":            "Bob",
				"qualificationName":      "Business License",
				"qualificationNumber":    "BL-001",
				"qualificationExpiresAt": "2026-12-31",
			},
			wantCode: "SUP-IMP-001",
		},
		{
			resource: MasterDataResourceCustomers,
			row: map[string]string{
				"code":                   "CUST-IMP-001",
				"name":                   "Customer One",
				"region":                 "East",
				"organizationId":         "org-a",
				"ownerId":                "sales-a",
				"rating":                 "4",
				"contactName":            "Cathy",
				"qualificationName":      "Medical License",
				"qualificationNumber":    "ML-001",
				"qualificationExpiresAt": "2026-12-31",
			},
			wantCode: "CUST-IMP-001",
		},
	}

	for _, tc := range cases {
		t.Run(tc.resource, func(t *testing.T) {
			template, err := service.Template(tc.resource)
			if err != nil {
				t.Fatalf("template: %v", err)
			}
			if template.Resource != tc.resource || len(template.Headers) == 0 || !strings.HasSuffix(template.Filename, "_template.xlsx") || template.FileBase64 == "" {
				t.Fatalf("unexpected template: %+v", template)
			}

			result, err := service.Import(ctx, MasterDataImportInput{Resource: tc.resource, Rows: []map[string]string{tc.row}, ActorID: "importer"})
			if err != nil {
				t.Fatalf("import: %v", err)
			}
			if result.Created != 1 || len(result.Failed) != 0 {
				t.Fatalf("unexpected import result: %+v", result)
			}

			job, err := service.Export(ctx, MasterDataExportInput{Resource: tc.resource, ActorID: "exporter"})
			if err != nil {
				t.Fatalf("export: %v", err)
			}
			if job.Status != "completed" || job.Resource != tc.resource || job.FileBase64 == "" || !exportRowsContain(job.Rows, tc.wantCode) {
				t.Fatalf("expected export to include %s, got %+v", tc.wantCode, job)
			}
		})
	}
}

func TestMasterDataExchangeImportReportsInvalidRows(t *testing.T) {
	service := newMasterDataExchangeFixture()
	result, err := service.Import(context.Background(), MasterDataImportInput{
		Resource: MasterDataResourceEmployees,
		Rows: []map[string]string{
			{"code": "", "name": "Missing Code"},
			{"code": "EMP-BAD-001", "name": "", "departmentId": "quality", "positionId": "qa"},
		},
		ActorID: "importer",
	})
	if err != nil {
		t.Fatalf("import invalid rows: %v", err)
	}
	if result.Created != 0 || len(result.Failed) != 2 {
		t.Fatalf("unexpected invalid import result: %+v", result)
	}
	if result.Failed[0].Row != 1 || result.Failed[0].Message != "code is required" {
		t.Fatalf("expected row 1 code error, got %+v", result.Failed[0])
	}
	if result.Failed[1].Row != 2 || !strings.Contains(result.Failed[1].Message, "name is required") {
		t.Fatalf("expected row 2 name error, got %+v", result.Failed[1])
	}
}

func newMasterDataExchangeFixture() MasterDataExchangeService {
	employees := NewEmployeeService(nil).(*employeeService)
	products := NewProductService(nil).(*productService)
	suppliers := NewSupplierService(nil).(*supplierService)
	customers := NewCustomerService(nil).(*customerService)
	now := func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }
	employees.nowFn = now
	products.nowFn = now
	suppliers.nowFn = now
	customers.nowFn = now
	service := NewMasterDataExchangeService(employees, products, suppliers, customers).(*masterDataExchangeService)
	service.nowFn = now
	return service
}

func exportRowsContain(rows [][]string, value string) bool {
	for _, row := range rows {
		for _, cell := range row {
			if cell == value {
				return true
			}
		}
	}
	return false
}
