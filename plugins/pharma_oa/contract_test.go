package pharmaoa

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin/datastore"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type pharmaManifest struct {
	ID               string `yaml:"id"`
	Version          string `yaml:"version"`
	APIVersion       string `yaml:"api_version"`
	MigrationVersion string `yaml:"migration_version"`
	AppID            string `yaml:"app_id"`
	ServiceBaseURL   string `yaml:"service_base_url"`
	ServiceHealthURL string `yaml:"service_health_url"`
	FrontendEntry    string `yaml:"frontend_entry"`
	UIMenu           struct {
		Path string `yaml:"path"`
	} `yaml:"ui_menu"`
	Permissions []struct {
		Key string `yaml:"key"`
	} `yaml:"permissions"`
	Data struct {
		Namespace          string `yaml:"namespace"`
		MigrationVersion   string `yaml:"migration_version"`
		MigrationDirectory string `yaml:"migration_directory"`
		UninstallPolicy    string `yaml:"uninstall_policy"`
		RollbackPolicy     string `yaml:"rollback_policy"`
		Tables             []struct {
			Name    string `yaml:"name"`
			Columns string `yaml:"columns"`
		} `yaml:"tables"`
	} `yaml:"data"`
	API struct {
		Routes []struct {
			Method      string `yaml:"method"`
			Path        string `yaml:"path"`
			Permission  string `yaml:"permission"`
			AuditAction string `yaml:"audit_action"`
		} `yaml:"routes"`
	} `yaml:"api"`
	Events struct {
		Subscriptions []struct {
			Name    string `yaml:"name"`
			Handler string `yaml:"handler"`
		} `yaml:"subscriptions"`
	} `yaml:"events"`
}

type pharmaAcceptanceMap struct {
	SchemaVersion   int    `json:"schemaVersion"`
	PluginID        string `json:"pluginId"`
	ContractVersion string `json:"contractVersion"`
	PublicContract  struct {
		APIVersion    string   `json:"apiVersion"`
		APIPrefix     string   `json:"apiPrefix"`
		BackendClient string   `json:"backendClient"`
		HostServices  []string `json:"hostServices"`
	} `json:"publicContract"`
	Modules []struct {
		ID            string   `json:"id"`
		WorkItems     []string `json:"workItems"`
		DocumentTypes []string `json:"documentTypes"`
		Acceptance    string   `json:"acceptance"`
	} `json:"modules"`
	DocumentTypes []string `json:"documentTypes"`
	Events        []struct {
		Name    string `json:"name"`
		Handler string `json:"handler"`
	} `json:"events"`
	Migrations struct {
		Version string   `json:"version"`
		Tables  []string `json:"tables"`
	} `json:"migrations"`
	AcceptanceScenarios []struct {
		ID     string `json:"id"`
		Result string `json:"result"`
	} `json:"acceptanceScenarios"`
}

func TestMedicalOABoundaryUsesOneCurrentPublicContract(t *testing.T) {
	manifest := loadPharmaManifest(t)
	contract := loadPharmaAcceptanceMap(t)
	if manifest.ID != "pharma_oa" || manifest.AppID != manifest.ID || contract.PluginID != manifest.ID {
		t.Fatalf("plugin identity mismatch: manifest=%q app=%q contract=%q", manifest.ID, manifest.AppID, contract.PluginID)
	}
	if manifest.Version != "0.10.0" || manifest.MigrationVersion != manifest.Version || contract.ContractVersion != manifest.Version || contract.Migrations.Version != manifest.Version {
		t.Fatalf("contract version mismatch: manifest=%q migration=%q map=%q", manifest.Version, manifest.MigrationVersion, contract.ContractVersion)
	}
	if manifest.APIVersion != "v1" || contract.SchemaVersion != 1 || contract.PublicContract.APIVersion != manifest.APIVersion {
		t.Fatalf("API contract mismatch: manifest=%q map=%q schema=%d", manifest.APIVersion, contract.PublicContract.APIVersion, contract.SchemaVersion)
	}
	if manifest.ServiceBaseURL == "" || manifest.ServiceHealthURL == "" || contract.PublicContract.BackendClient != "github.com/tinboxw/skoll/pkg/pluginclient" {
		t.Fatal("managed process and public backend client are required")
	}
	if manifest.FrontendEntry != "/skoll/plugins/pharma-oa" || manifest.UIMenu.Path != manifest.FrontendEntry {
		t.Fatalf("independent plugin UI entry mismatch: frontend=%q menu=%q", manifest.FrontendEntry, manifest.UIMenu.Path)
	}
	if manifest.Data.Namespace != manifest.ID || manifest.Data.MigrationVersion != manifest.Version || manifest.Data.MigrationDirectory != "migrations" || manifest.Data.UninstallPolicy != "drop" || manifest.Data.RollbackPolicy != "automatic" {
		t.Fatalf("data lifecycle is incomplete: %+v", manifest.Data)
	}

	wantModules := []string{"analytics", "catalog", "crm", "finance", "inventory", "office", "parties", "purchasing", "qualifications", "quality", "sales", "workforce"}
	gotModules := make([]string, 0, len(contract.Modules))
	documentOwners := make(map[string]string)
	for _, module := range contract.Modules {
		if module.ID == "" || len(module.WorkItems) == 0 || strings.TrimSpace(module.Acceptance) == "" {
			t.Fatalf("incomplete module boundary: %+v", module)
		}
		gotModules = append(gotModules, module.ID)
		for _, documentType := range module.DocumentTypes {
			if owner, exists := documentOwners[documentType]; exists {
				t.Fatalf("document type %s has two owners: %s and %s", documentType, owner, module.ID)
			}
			documentOwners[documentType] = module.ID
		}
	}
	sort.Strings(gotModules)
	if strings.Join(gotModules, ",") != strings.Join(wantModules, ",") {
		t.Fatalf("module boundary=%v want=%v", gotModules, wantModules)
	}
	for _, documentType := range contract.DocumentTypes {
		if documentOwners[documentType] == "" {
			t.Fatalf("document type %s has no module owner", documentType)
		}
	}
	if len(documentOwners) != len(contract.DocumentTypes) || len(contract.DocumentTypes) != 12 {
		t.Fatalf("document ownership=%d types=%d want=12", len(documentOwners), len(contract.DocumentTypes))
	}

	permissions := make(map[string]struct{}, len(manifest.Permissions))
	for _, permission := range manifest.Permissions {
		if _, exists := permissions[permission.Key]; exists {
			t.Fatalf("duplicate permission: %s", permission.Key)
		}
		permissions[permission.Key] = struct{}{}
	}
	routes := make(map[string]struct{}, len(manifest.API.Routes))
	for _, route := range manifest.API.Routes {
		key := strings.ToUpper(route.Method) + " " + route.Path
		if !strings.HasPrefix(route.Path, contract.PublicContract.APIPrefix) {
			t.Fatalf("route escapes plugin namespace: %s", key)
		}
		if _, exists := routes[key]; exists {
			t.Fatalf("duplicate route: %s", key)
		}
		routes[key] = struct{}{}
		assertPharmaContains(t, permissions, route.Permission, "route permission")
		if len(strings.Split(route.AuditAction, ".")) < 3 {
			t.Fatalf("route %s has invalid audit action %q", key, route.AuditAction)
		}
	}
	assertPharmaContains(t, routes, "GET /v1/plugins/pharma_oa/api/meta", "foundation route")

	events := make(map[string]struct{}, len(manifest.Events.Subscriptions))
	for _, event := range manifest.Events.Subscriptions {
		events[event.Name+"::"+event.Handler] = struct{}{}
	}
	for _, event := range contract.Events {
		assertPharmaContains(t, events, event.Name+"::"+event.Handler, "event subscription")
	}
	if len(events) != 5 || len(contract.AcceptanceScenarios) < 5 {
		t.Fatalf("events=%d scenarios=%d", len(events), len(contract.AcceptanceScenarios))
	}
}

func TestMedicalOAPackageSurfaceAndHostIndependence(t *testing.T) {
	for _, path := range []string{
		"backend/main.go", "backend/server.go", "backend/employee.go", "backend/party.go", "backend/catalog.go", "backend/qualification.go", "backend/oa_request.go", "backend/purchase.go", "backend/inbound.go", "plugin.ps1", "plugin.sh", "frontend/index.html", "frontend/package.json", "frontend/src/App.vue", "frontend/src/api.ts", "frontend/src/i18n.ts", "frontend/src/styles.css", "datastore.yaml",
		"contract/acceptance-map.json", "migrations/001_foundation.up.sql", "migrations/001_foundation.down.sql",
		"migrations/002_employees.up.sql", "migrations/002_employees.down.sql",
		"migrations/003_parties.up.sql", "migrations/003_parties.down.sql",
		"migrations/004_catalogs.up.sql", "migrations/004_catalogs.down.sql",
		"migrations/005_qualifications.up.sql", "migrations/005_qualifications.down.sql",
		"migrations/006_oa_requests.up.sql", "migrations/006_oa_requests.down.sql",
		"migrations/007_purchases.up.sql", "migrations/007_purchases.down.sql",
		"migrations/008_purchase_inbounds.up.sql", "migrations/008_purchase_inbounds.down.sql",
		"migrations/009_warehouse_topology.up.sql", "migrations/009_warehouse_topology.down.sql",
	} {
		if info, err := os.Stat(filepath.FromSlash(path)); err != nil || info.IsDir() {
			t.Fatalf("required package file %q is unavailable: %v", path, err)
		}
	}
	if err := filepath.WalkDir("backend", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".go" {
			return err
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		privatePrefix := strings.Join([]string{"github.com/tinboxw/skoll", "internal"}, "/") + "/"
		if strings.Contains(string(raw), privatePrefix) {
			t.Fatalf("backend imports private host source: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	root := filepath.Clean(filepath.Join("..", ".."))
	if err := filepath.WalkDir(filepath.Join(root, "internal", "bootstrap"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		source := string(raw)
		if strings.Contains(source, "internal/plugin/pharmaoa") || strings.Contains(source, "pharmaoaplugin") || strings.Contains(source, "scopeMatrixFixtures") {
			t.Fatalf("host bootstrap owns medical OA source: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestMedicalOAProductionCoreOwnsNoBusinessImplementation(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	for _, relativePath := range []string{
		"internal/domain/pharmaoa",
		"internal/service/pharmaoa",
		"internal/repository/pharmaoa",
		"internal/handler/http/v1/pharmaoa",
		"internal/plugin/pharmaoa",
		"web/src/pharma-oa",
		"web/src/i18n/pharma.json",
		"web/src/plugins/integrated-routes.ts",
	} {
		path := filepath.Join(root, filepath.FromSlash(relativePath))
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("production core still owns medical OA path %q: %v", relativePath, err)
		}
	}

	viewsRoot := filepath.Join(root, "web", "src", "views")
	views, err := os.ReadDir(viewsRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, view := range views {
		if view.IsDir() && strings.HasPrefix(strings.ToLower(view.Name()), "pharma") {
			t.Fatalf("production core still owns medical OA view %q", view.Name())
		}
	}

	for _, relativeRoot := range []string{"cmd", "internal", "pkg", "web/src", "web/scripts", "web/i18n"} {
		scanRoot := filepath.Join(root, filepath.FromSlash(relativeRoot))
		if err := filepath.WalkDir(scanRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			name := strings.ToLower(entry.Name())
			if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".spec.ts") || strings.HasSuffix(name, ".test.ts") {
				return nil
			}
			switch filepath.Ext(name) {
			case ".go", ".ts", ".vue", ".json", ".mjs", ".yaml", ".yml":
			default:
				return nil
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			source := strings.ToLower(string(raw))
			for _, forbidden := range []string{"pharma_oa", "pharmaoa", "pharma-oa", "pharma oa", "医药 oa"} {
				if strings.Contains(source, forbidden) {
					relative, _ := filepath.Rel(root, path)
					t.Fatalf("production core contains medical OA ownership marker %q in %s", forbidden, filepath.ToSlash(relative))
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMedicalOAPurchaseDataStoreContractIsRegistered(t *testing.T) {
	schema, present, err := datastore.LoadSchemaManifest("pharma_oa", ".")
	if err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Fatal("datastore schema is missing")
	}
	tables := make(map[string]datastore.TableSchema, len(schema.Tables))
	for _, table := range schema.Tables {
		tables[table.Name] = table
	}
	for tableName, requiredFields := range map[string]map[string]pluginsdk.DataValueType{
		"purchase_requests": {
			"number": pluginsdk.DataValueString, "supplier_id": pluginsdk.DataValueString,
			"lines": pluginsdk.DataValueJSON, "total_amount": pluginsdk.DataValueDecimal,
			"status": pluginsdk.DataValueString, "workflow_instance_id": pluginsdk.DataValueString,
		},
		"purchase_orders": {
			"number": pluginsdk.DataValueString, "purchase_request_id": pluginsdk.DataValueString,
			"lines": pluginsdk.DataValueJSON, "total_amount": pluginsdk.DataValueDecimal,
			"status": pluginsdk.DataValueString, "approved_at": pluginsdk.DataValueTimestamp,
		},
		"purchase_inbounds": {
			"number": pluginsdk.DataValueString, "purchase_order_id": pluginsdk.DataValueString,
			"lines": pluginsdk.DataValueJSON, "attachments": pluginsdk.DataValueJSON,
			"status": pluginsdk.DataValueString, "received_at": pluginsdk.DataValueTimestamp,
		},
		"warehouses": {
			"code": pluginsdk.DataValueString, "name": pluginsdk.DataValueString,
			"address": pluginsdk.DataValueString, "status": pluginsdk.DataValueString,
		},
		"warehouse_areas": {
			"warehouse_id": pluginsdk.DataValueString, "code": pluginsdk.DataValueString,
			"temperature_min": pluginsdk.DataValueDecimal, "temperature_max": pluginsdk.DataValueDecimal,
		},
		"warehouse_locations": {
			"warehouse_id": pluginsdk.DataValueString, "area_id": pluginsdk.DataValueString,
			"code": pluginsdk.DataValueString, "location_type": pluginsdk.DataValueString,
		},
	} {
		table, ok := tables[tableName]
		if !ok {
			t.Fatalf("datastore schema is missing %s", tableName)
		}
		fields := make(map[string]pluginsdk.DataValueType, len(table.Fields))
		for _, field := range table.Fields {
			fields[field.Name] = field.Type
		}
		for field, wantType := range requiredFields {
			if got := fields[field]; got != wantType {
				t.Fatalf("%s.%s type=%q want=%q", tableName, field, got, wantType)
			}
		}
	}
}

func TestMedicalOAFoundationMigrationIsExecutableAndReversible(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migration.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, path := range []string{"migrations/001_foundation.up.sql", "migrations/002_employees.up.sql", "migrations/003_parties.up.sql", "migrations/004_catalogs.up.sql", "migrations/005_qualifications.up.sql", "migrations/006_oa_requests.up.sql", "migrations/007_purchases.up.sql", "migrations/008_purchase_inbounds.up.sql", "migrations/009_warehouse_topology.up.sql"} {
		migration, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		sql := strings.ReplaceAll(string(migration), "{{table:employees}}", "pharma_oa_employees")
		sql = strings.ReplaceAll(sql, "{{table:parties}}", "pharma_oa_parties")
		sql = strings.ReplaceAll(sql, "{{table:catalogs}}", "pharma_oa_catalogs")
		sql = strings.ReplaceAll(sql, "{{table:products}}", "pharma_oa_products")
		sql = strings.ReplaceAll(sql, "{{table:qualification_types}}", "pharma_oa_qualification_types")
		sql = strings.ReplaceAll(sql, "{{table:qualifications}}", "pharma_oa_qualifications")
		sql = strings.ReplaceAll(sql, "{{table:oa_requests}}", "pharma_oa_oa_requests")
		sql = strings.ReplaceAll(sql, "{{table:purchase_requests}}", "pharma_oa_purchase_requests")
		sql = strings.ReplaceAll(sql, "{{table:purchase_orders}}", "pharma_oa_purchase_orders")
		sql = strings.ReplaceAll(sql, "{{table:purchase_inbounds}}", "pharma_oa_purchase_inbounds")
		sql = strings.ReplaceAll(sql, "{{table:warehouses}}", "pharma_oa_warehouses")
		sql = strings.ReplaceAll(sql, "{{table:warehouse_areas}}", "pharma_oa_warehouse_areas")
		sql = strings.ReplaceAll(sql, "{{table:warehouse_locations}}", "pharma_oa_warehouse_locations")
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	manifest := loadPharmaManifest(t)
	wantTables := make(map[string]struct{}, len(manifest.Data.Tables))
	for _, table := range manifest.Data.Tables {
		physicalTable := manifest.Data.Namespace + "_" + table.Name
		wantTables[physicalTable] = struct{}{}
		if !db.Migrator().HasTable(physicalTable) {
			t.Fatalf("migration did not create %s for logical table %s", physicalTable, table.Name)
		}
		columns := pharmaCSVSet(table.Columns)
		for _, column := range []string{"tenant_id", "organization_id", "owner_id"} {
			assertPharmaContains(t, columns, column, "scope column in "+table.Name)
		}
	}
	for _, table := range loadPharmaAcceptanceMap(t).Migrations.Tables {
		assertPharmaContains(t, wantTables, table, "contract migration table")
	}
	assertWarehouseTopologyScopedUniqueness(t, db)
	for _, path := range []string{"migrations/009_warehouse_topology.down.sql", "migrations/008_purchase_inbounds.down.sql", "migrations/007_purchases.down.sql", "migrations/006_oa_requests.down.sql", "migrations/005_qualifications.down.sql", "migrations/004_catalogs.down.sql", "migrations/003_parties.down.sql", "migrations/002_employees.down.sql", "migrations/001_foundation.down.sql"} {
		migration, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		sql := strings.ReplaceAll(string(migration), "{{table:employees}}", "pharma_oa_employees")
		sql = strings.ReplaceAll(sql, "{{table:parties}}", "pharma_oa_parties")
		sql = strings.ReplaceAll(sql, "{{table:catalogs}}", "pharma_oa_catalogs")
		sql = strings.ReplaceAll(sql, "{{table:products}}", "pharma_oa_products")
		sql = strings.ReplaceAll(sql, "{{table:qualification_types}}", "pharma_oa_qualification_types")
		sql = strings.ReplaceAll(sql, "{{table:qualifications}}", "pharma_oa_qualifications")
		sql = strings.ReplaceAll(sql, "{{table:oa_requests}}", "pharma_oa_oa_requests")
		sql = strings.ReplaceAll(sql, "{{table:purchase_requests}}", "pharma_oa_purchase_requests")
		sql = strings.ReplaceAll(sql, "{{table:purchase_orders}}", "pharma_oa_purchase_orders")
		sql = strings.ReplaceAll(sql, "{{table:purchase_inbounds}}", "pharma_oa_purchase_inbounds")
		sql = strings.ReplaceAll(sql, "{{table:warehouses}}", "pharma_oa_warehouses")
		sql = strings.ReplaceAll(sql, "{{table:warehouse_areas}}", "pharma_oa_warehouse_areas")
		sql = strings.ReplaceAll(sql, "{{table:warehouse_locations}}", "pharma_oa_warehouse_locations")
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("rollback %s: %v", path, err)
		}
	}
	for table := range wantTables {
		if db.Migrator().HasTable(table) {
			t.Fatalf("rollback retained %s", table)
		}
	}
}

func assertWarehouseTopologyScopedUniqueness(t *testing.T, db *gorm.DB) {
	t.Helper()
	const timestamp = "2026-07-27T00:00:00Z"
	insertWarehouse := func(id, organization string) error {
		return db.Exec(`INSERT INTO pharma_oa_warehouses
			(id, code, name, address, contact_name, contact_phone, status, tenant_id, organization_id, owner_id, version, created_at, updated_at)
			VALUES (?, 'WH-001', 'Main warehouse', '1 Storage Road', 'Owner', '13800000000', 'active', 'tenant-a', ?, 'owner-a', 1, ?, ?)`,
			id, organization, timestamp, timestamp).Error
	}
	if err := insertWarehouse("warehouse-a", "org-a"); err != nil {
		t.Fatal(err)
	}
	if err := insertWarehouse("warehouse-duplicate", "org-a"); err == nil {
		t.Fatal("warehouse code must be unique inside one tenant organization")
	}
	if err := insertWarehouse("warehouse-b", "org-b"); err != nil {
		t.Fatalf("warehouse code should be reusable in another organization: %v", err)
	}

	insertArea := func(id, warehouse string) error {
		return db.Exec(`INSERT INTO pharma_oa_warehouse_areas
			(id, warehouse_id, code, name, status, tenant_id, organization_id, owner_id, version, created_at, updated_at)
			VALUES (?, ?, 'AREA-001', 'Qualified area', 'active', 'tenant-a', 'org-a', 'owner-a', 1, ?, ?)`,
			id, warehouse, timestamp, timestamp).Error
	}
	if err := insertArea("area-a", "warehouse-a"); err != nil {
		t.Fatal(err)
	}
	if err := insertArea("area-duplicate", "warehouse-a"); err == nil {
		t.Fatal("area code must be unique inside one warehouse")
	}
	if err := insertArea("area-b", "warehouse-b"); err != nil {
		t.Fatalf("area code should be reusable in another warehouse: %v", err)
	}

	insertLocation := func(id, warehouse, area string) error {
		return db.Exec(`INSERT INTO pharma_oa_warehouse_locations
			(id, warehouse_id, area_id, code, name, location_type, status, tenant_id, organization_id, owner_id, version, created_at, updated_at)
			VALUES (?, ?, ?, 'LOC-001', 'Qualified location', 'standard', 'active', 'tenant-a', 'org-a', 'owner-a', 1, ?, ?)`,
			id, warehouse, area, timestamp, timestamp).Error
	}
	if err := insertLocation("location-a", "warehouse-a", "area-a"); err != nil {
		t.Fatal(err)
	}
	if err := insertLocation("location-duplicate", "warehouse-a", "area-a"); err == nil {
		t.Fatal("location code must be unique inside one area")
	}
	if err := insertLocation("location-b", "warehouse-b", "area-b"); err != nil {
		t.Fatalf("location code should be reusable in another area: %v", err)
	}
}

func TestMedicalOAWorkspaceIsElementNativeChineseFirstAndComplete(t *testing.T) {
	html := pharmaReadText(t, "frontend/index.html")
	app := pharmaReadText(t, "frontend/src/App.vue")
	api := pharmaReadText(t, "frontend/src/api.ts")
	i18n := pharmaReadText(t, "frontend/src/i18n.ts")
	styles := pharmaReadText(t, "frontend/src/styles.css")
	manifest := pharmaReadText(t, "plugin.yaml")
	if !strings.Contains(html, `<html lang="zh-CN">`) || !strings.Contains(manifest, "ui_mode: separated") {
		t.Fatal("the current separated Chinese-first frontend contract is missing")
	}
	for _, value := range []string{"员工管理", "客户管理", "供应商管理", "药品管理", "药品分类", "计量单位", "生产企业", "资质台账", "资质类型", "扫描到期资质"} {
		if !strings.Contains(i18n, value) {
			t.Fatalf("localized workspace is missing %q", value)
		}
	}
	for _, value := range []string{"el-config-provider", "el-table", "el-dialog", "el-drawer", "mobile-records", "statusActions", "scanExpiry", "uploadAttachment", "saveEditor"} {
		if !strings.Contains(app, value) {
			t.Fatalf("Element Plus workspace is missing %q", value)
		}
	}
	for _, value := range []string{`import { getPluginHost } from "@skoll/plugin-sdk"`, `getPluginHost({ pluginId: "pharma_oa"`, `"Idempotency-Key"`, `customer: "/customers"`, `supplier: "/suppliers"`, `product: "/products"`, `qualification: "/qualifications"`, `qualificationType: "/qualification-types"`, "scanQualificationExpiry"} {
		if !strings.Contains(api, value) {
			t.Fatalf("host-native API workflow is missing %q", value)
		}
	}
	if strings.Contains(api, "window.__SKOLL_HOST__") {
		t.Fatal("business plugin must use the current plugin SDK instead of reading the host global directly")
	}
	for _, value := range []string{"skoll:locale", "skoll:host-ready", "Master Data Workspace", "主数据工作台"} {
		if !strings.Contains(i18n, value) {
			t.Fatalf("runtime locale contract is missing %q", value)
		}
	}
	for _, value := range []string{"--color-surface", "--color-primary", `[data-density="compact"]`, "var(--color-bg)", "@media (max-width: 900px)", "@media (max-width: 620px)", "prefers-reduced-motion", ":focus-visible"} {
		if !strings.Contains(styles, value) {
			t.Fatalf("responsive theme contract is missing %q", value)
		}
	}
	if _, err := os.Stat("static/index.html"); !os.IsNotExist(err) {
		t.Fatalf("legacy static frontend entry must not remain: %v", err)
	}
}

func loadPharmaManifest(t *testing.T) pharmaManifest {
	t.Helper()
	raw, err := os.ReadFile("plugin.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var manifest pharmaManifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return manifest
}

func pharmaReadText(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func loadPharmaAcceptanceMap(t *testing.T) pharmaAcceptanceMap {
	t.Helper()
	raw, err := os.ReadFile("contract/acceptance-map.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract pharmaAcceptanceMap
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("decode acceptance map: %v", err)
	}
	return contract
}

func pharmaCSVSet(raw string) map[string]struct{} {
	items := make(map[string]struct{})
	for _, item := range strings.Split(raw, ",") {
		items[strings.TrimSpace(item)] = struct{}{}
	}
	return items
}

func assertPharmaContains(t *testing.T, values map[string]struct{}, value string, label string) {
	t.Helper()
	if _, ok := values[value]; !ok {
		t.Fatalf("%s %q is missing", label, value)
	}
}
