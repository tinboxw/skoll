package gormrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	mysqldriver "gorm.io/driver/mysql"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPharmaMasterRepositoriesPersistAcrossSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pharma-master.db")
	db := openPharmaSQLite(t, path)
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	fixtures := seedPharmaMasterContract(t, db, suffix)
	closeTestDB(t, db)

	db = openPharmaSQLite(t, path)
	defer closeTestDB(t, db)
	assertPharmaMasterContract(t, db, fixtures)
}

func TestPharmaMasterRepositoriesPagingFiltersAndUniqueness(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()
	employees := NewPharmaEmployeeStore(db)
	now := time.Now().UTC()
	firstEmployee := &domainpharma.Employee{ID: "employee-stable", Code: "E-STABLE", Name: "Original", Status: domainpharma.EmployeeStatusActive, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}}
	if err := employees.Create(ctx, firstEmployee); err != nil {
		t.Fatalf("create employee: %v", err)
	}
	overwrite := *firstEmployee
	overwrite.Code = "E-OVERWRITE"
	overwrite.Name = "Overwritten"
	if err := employees.Create(ctx, &overwrite); err == nil {
		t.Fatal("duplicate employee id must fail instead of overwriting")
	}
	storedEmployee, err := employees.Get(ctx, firstEmployee.ID)
	if err != nil || storedEmployee == nil || storedEmployee.Name != "Original" {
		t.Fatalf("employee changed after duplicate create: %+v, err=%v", storedEmployee, err)
	}

	products := NewPharmaProductStore(db)
	for index, code := range []string{"P-A", "P-B", "P-C"} {
		item := &domainpharma.Product{
			ID: shared.ID(fmt.Sprintf("product-%d", index+1)), Code: code, Name: "Product " + code, Spec: "10mg",
			DosageForm: "tablet", Manufacturer: "Skoll Pharma", ApprovalNumber: "APP-" + code,
			Status: domainpharma.ProductStatusActive, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
		}
		if err := products.Upsert(ctx, item); err != nil {
			t.Fatalf("upsert product %s: %v", code, err)
		}
	}
	page, err := products.List(ctx, pharmaoarepo.ListFilter{Keyword: "product", Status: "active", Offset: 1, Limit: 1})
	if err != nil || len(page) != 1 || page[0].Code != "P-B" {
		t.Fatalf("paged product list = %+v, err=%v", page, err)
	}
	pageWithTotal, err := products.ListPage(ctx, pharmaoarepo.ListFilter{Keyword: "product", Status: "active", Offset: 1, Limit: 1})
	if err != nil || pageWithTotal.Total != 3 || len(pageWithTotal.Items) != 1 || pageWithTotal.Items[0].Code != "P-B" {
		t.Fatalf("product page with total = %+v, err=%v", pageWithTotal, err)
	}
	if err := products.Upsert(ctx, &domainpharma.Product{
		ID: "product-duplicate-code", Code: "P-A", Name: "Duplicate", Spec: "1mg", DosageForm: "tablet",
		Manufacturer: "Skoll", ApprovalNumber: "APP-OTHER", Status: domainpharma.ProductStatusActive,
		Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
	}); err == nil {
		t.Fatal("duplicate product code must fail")
	}
	if err := products.Upsert(ctx, &domainpharma.Product{
		ID: "product-duplicate-approval", Code: "P-D", Name: "Duplicate approval", Spec: "1mg", DosageForm: "tablet",
		Manufacturer: "Skoll", ApprovalNumber: "APP-P-A", Status: domainpharma.ProductStatusActive,
		Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
	}); err == nil {
		t.Fatal("duplicate approval number must fail")
	}

	customers := NewPharmaCustomerStore(db)
	for _, item := range []domainpharma.Customer{
		{ID: "customer-owner", Code: "C-OWNER", Name: "Owner customer", Region: "East", OwnerID: "owner-1", OrganizationID: "org-2", Status: domainpharma.CustomerStatusActive, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}},
		{ID: "customer-org", Code: "C-ORG", Name: "Org customer", Region: "East", OwnerID: "owner-2", OrganizationID: "org-1", Status: domainpharma.CustomerStatusActive, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}},
		{ID: "customer-denied", Code: "C-DENIED", Name: "Denied customer", Region: "West", OwnerID: "owner-2", OrganizationID: "org-2", Status: domainpharma.CustomerStatusActive, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}},
		{ID: "customer-outside", Code: "C-OUTSIDE", Name: "Outside customer", Region: "East", OwnerID: "owner-3", OrganizationID: "org-3", Status: domainpharma.CustomerStatusActive, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now}},
	} {
		item := item
		if err := customers.Upsert(ctx, &item); err != nil {
			t.Fatalf("upsert customer %s: %v", item.Code, err)
		}
	}
	scoped, err := customers.List(ctx, pharmaoarepo.ListFilter{
		Region: "east", OrganizationID: "org-1", OwnerID: "owner-1", ScopeAny: true,
	})
	if err != nil || len(scoped) != 2 {
		t.Fatalf("scoped customer list = %+v, err=%v", scoped, err)
	}
	treeScoped, err := customers.List(ctx, pharmaoarepo.ListFilter{OrganizationIDs: []shared.ID{"org-1", "org-2"}, ScopeAny: true})
	if err != nil || len(treeScoped) != 3 {
		t.Fatalf("tree-scoped customer list = %+v, err=%v", treeScoped, err)
	}
}

func TestPharmaMasterMigrationScriptsMatchCurrentSchema(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	for _, dialect := range []string{"mysql", "postgres"} {
		path := filepath.Join(root, "migrations", dialect, "20260718_000019_create_pharma_oa_master_data.sql")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s migration: %v", dialect, err)
		}
		content := strings.ToLower(string(raw))
		for _, table := range []string{"pharma_oa_employees", "pharma_oa_products", "pharma_oa_suppliers", "pharma_oa_customers", "pharma_oa_warehouses"} {
			if !strings.Contains(content, table) {
				t.Fatalf("%s migration missing table %s", dialect, table)
			}
		}
		for _, index := range []string{"uk_pharma_employees_code", "uk_pharma_products_approval", "idx_pharma_customers_scope_status", "idx_pharma_warehouses_status_name"} {
			if !strings.Contains(content, index) {
				t.Fatalf("%s migration missing index %s", dialect, index)
			}
		}
		if strings.Contains(content, "drop table") {
			t.Fatalf("%s normal migration must not contain destructive rollback", dialect)
		}
	}
}

func TestPharmaMasterGORMModelsMatchSchemaBaseline(t *testing.T) {
	db := TestDB(t)
	for _, table := range pharmaoarepo.SchemaBaseline() {
		if !strings.HasPrefix(table.Owner, "master.") {
			continue
		}
		for _, column := range table.Columns {
			if !db.Migrator().HasColumn(table.Name, column) {
				t.Fatalf("GORM model %s missing schema column %s", table.Name, column)
			}
		}
		for _, index := range table.Indexes {
			if !db.Migrator().HasIndex(table.Name, index.Name) {
				t.Fatalf("GORM model %s missing schema index %s", table.Name, index.Name)
			}
		}
	}
}

func TestPharmaMasterRepositoriesMySQLContract(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SKOLL_TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("SKOLL_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysqldriver.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open MySQL contract database: %v", err)
	}
	runExternalPharmaMasterContract(t, db, "mysql")
}

func TestPharmaMasterRepositoriesPostgreSQLContract(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SKOLL_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("SKOLL_TEST_POSTGRES_DSN is not set")
	}
	db, err := gorm.Open(postgresdriver.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL contract database: %v", err)
	}
	runExternalPharmaMasterContract(t, db, "postgres")
}

type pharmaMasterFixtures struct {
	employee  domainpharma.Employee
	product   domainpharma.Product
	supplier  domainpharma.Supplier
	customer  domainpharma.Customer
	warehouse domainpharma.Warehouse
}

func seedPharmaMasterContract(t *testing.T, db *gorm.DB, suffix string) pharmaMasterFixtures {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	fixtures := pharmaMasterFixtures{
		employee: domainpharma.Employee{
			ID: shared.ID("employee-" + suffix), Code: "E-" + suffix, Name: "张敏", DepartmentID: "dept-quality", PositionID: "position-qa",
			Phone: "13800000000", Email: "zhang@example.test", Status: domainpharma.EmployeeStatusActive,
			Certificates: []domainpharma.EmployeeCertificate{{ID: "cert-1", Name: "执业药师证", Number: "CERT-001", ExpiresAt: now.AddDate(1, 0, 0)}},
			Meta:         shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
		},
		product: domainpharma.Product{
			ID: shared.ID("product-" + suffix), Code: "P-" + suffix, Name: "冷链药品", Spec: "20mg", DosageForm: "注射剂",
			Manufacturer: "Skoll Pharma", ApprovalNumber: "APP-" + suffix,
			Temperature: domainpharma.ProductTemperature{Required: true, MinCelsius: 2, MaxCelsius: 8},
			Status:      domainpharma.ProductStatusDisabled, DisableReason: "测试停用", Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
		},
		supplier: domainpharma.Supplier{
			ID: shared.ID("supplier-" + suffix), Code: "S-" + suffix, Name: "华东供应商", Rating: 5, Status: domainpharma.SupplierStatusActive,
			Contacts: []domainpharma.SupplierContact{{ID: "contact-1", Name: "李四", Phone: "13900000000", Primary: true}},
			Qualifications: []domainpharma.SupplierQualification{{
				ID: "qualification-1", Name: "药品经营许可证", Number: "SUP-001", ExpiresAt: now.AddDate(1, 0, 0),
				Attachments: []domainpharma.SupplierAttachment{{FileID: "file-supplier", FileName: "supplier.pdf", MimeType: "application/pdf", Size: 1024}},
			}}, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
		},
		customer: domainpharma.Customer{
			ID: shared.ID("customer-" + suffix), Code: "C-" + suffix, Name: "第一医院", Region: "华东", OrganizationID: "org-east", OwnerID: "owner-1",
			Rating: 4, Status: domainpharma.CustomerStatusDisabled, DisableReason: "资质复核",
			Contacts: []domainpharma.CustomerContact{{ID: "contact-1", Name: "王五", Phone: "13700000000"}},
			Qualifications: []domainpharma.CustomerQualification{{
				ID: "qualification-1", Name: "医疗机构执业许可证", Number: "CUS-001", ExpiresAt: now.AddDate(1, 0, 0),
				Attachments: []domainpharma.CustomerAttachment{{FileID: "file-customer", FileName: "customer.pdf", Size: 2048}},
			}}, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
		},
		warehouse: domainpharma.Warehouse{
			ID: shared.ID("warehouse-" + suffix), Code: "W-" + suffix, Name: "上海冷链仓", Region: "华东", Status: domainpharma.WarehouseStatusEnabled,
			Temperature: domainpharma.WarehouseTemperature{Controlled: true, MinCelsius: 2, MaxCelsius: 8},
			Areas: []domainpharma.WarehouseArea{{
				ID: "area-1", Code: "COLD", Name: "冷藏区", Status: domainpharma.WarehouseStatusEnabled,
				Locations: []domainpharma.WarehouseLocation{{ID: "location-1", Code: "COLD-01", Name: "冷藏库位 01", Status: domainpharma.WarehouseStatusEnabled}},
			}}, Meta: shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
		},
	}

	for name, save := range map[string]func() error{
		"employee":  func() error { return NewPharmaEmployeeStore(db).Upsert(ctx, &fixtures.employee) },
		"product":   func() error { return NewPharmaProductStore(db).Upsert(ctx, &fixtures.product) },
		"supplier":  func() error { return NewPharmaSupplierStore(db).Upsert(ctx, &fixtures.supplier) },
		"customer":  func() error { return NewPharmaCustomerStore(db).Upsert(ctx, &fixtures.customer) },
		"warehouse": func() error { return NewPharmaWarehouseStore(db).Upsert(ctx, &fixtures.warehouse) },
	} {
		if err := save(); err != nil {
			t.Fatalf("save %s fixture: %v", name, err)
		}
	}
	return fixtures
}

func assertPharmaMasterContract(t *testing.T, db *gorm.DB, fixtures pharmaMasterFixtures) {
	t.Helper()
	ctx := context.Background()
	employee, err := NewPharmaEmployeeStore(db).Get(ctx, fixtures.employee.ID)
	if err != nil || employee == nil || employee.Phone != fixtures.employee.Phone || len(employee.Certificates) != 1 {
		t.Fatalf("employee round trip = %+v, err=%v", employee, err)
	}
	employeePage, err := NewPharmaEmployeeStore(db).ListPage(ctx, pharmaoarepo.ListFilter{Keyword: fixtures.employee.Code, Limit: 1})
	if err != nil || employeePage.Total != 1 || len(employeePage.Items) != 1 || employeePage.Items[0].ID != fixtures.employee.ID {
		t.Fatalf("employee page contract = %+v, err=%v", employeePage, err)
	}
	product, err := NewPharmaProductStore(db).GetByCode(ctx, strings.ToLower(fixtures.product.Code))
	if err != nil || product == nil || product.DisableReason != fixtures.product.DisableReason || !product.Temperature.Required {
		t.Fatalf("product round trip = %+v, err=%v", product, err)
	}
	supplier, err := NewPharmaSupplierStore(db).Get(ctx, fixtures.supplier.ID)
	if err != nil || supplier == nil || len(supplier.Qualifications) != 1 || len(supplier.Qualifications[0].Attachments) != 1 {
		t.Fatalf("supplier attachment round trip = %+v, err=%v", supplier, err)
	}
	customer, err := NewPharmaCustomerStore(db).Get(ctx, fixtures.customer.ID)
	if err != nil || customer == nil || customer.OrganizationID != "org-east" || len(customer.Qualifications) != 1 || len(customer.Qualifications[0].Attachments) != 1 {
		t.Fatalf("customer round trip = %+v, err=%v", customer, err)
	}
	warehouse, err := NewPharmaWarehouseStore(db).Get(ctx, fixtures.warehouse.ID)
	if err != nil || warehouse == nil || len(warehouse.Areas) != 1 || len(warehouse.Areas[0].Locations) != 1 {
		t.Fatalf("warehouse round trip = %+v, err=%v", warehouse, err)
	}

	supplier.Qualifications[0].Attachments[0].FileName = "mutated.pdf"
	again, err := NewPharmaSupplierStore(db).Get(ctx, fixtures.supplier.ID)
	if err != nil || again.Qualifications[0].Attachments[0].FileName != "supplier.pdf" {
		t.Fatalf("supplier repository did not return defensive data: %+v, err=%v", again, err)
	}
}

func openPharmaSQLite(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path+"?_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		if isSQLiteCGODisabledError(err) {
			t.Skipf("sqlite test requires cgo: %v", err)
		}
		t.Fatalf("open SQLite contract database: %v", err)
	}
	if err := db.AutoMigrate(pharmaMasterModels()...); err != nil {
		t.Fatalf("migrate SQLite contract database: %v", err)
	}
	return db
}

func runExternalPharmaMasterContract(t *testing.T, db *gorm.DB, dialect string) {
	t.Helper()
	if err := db.AutoMigrate(pharmaMasterModels()...); err != nil {
		t.Fatalf("migrate %s contract database: %v", dialect, err)
	}
	suffix := fmt.Sprintf("%s-%d", dialect, time.Now().UnixNano())
	fixtures := seedPharmaMasterContract(t, db, suffix)
	t.Cleanup(func() {
		for model, id := range map[any]string{
			&PharmaEmployeeModel{}: fixtures.employee.ID.String(), &PharmaProductModel{}: fixtures.product.ID.String(),
			&PharmaSupplierModel{}: fixtures.supplier.ID.String(), &PharmaCustomerModel{}: fixtures.customer.ID.String(),
			&PharmaWarehouseModel{}: fixtures.warehouse.ID.String(),
		} {
			_ = db.Delete(model, "id = ?", id).Error
		}
	})
	assertPharmaMasterContract(t, db, fixtures)
}

func pharmaMasterModels() []any {
	return []any{&PharmaEmployeeModel{}, &PharmaProductModel{}, &PharmaSupplierModel{}, &PharmaCustomerModel{}, &PharmaWarehouseModel{}}
}

func closeTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql database: %v", err)
	}
}
