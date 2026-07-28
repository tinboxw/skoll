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
	ID               string   `yaml:"id"`
	Version          string   `yaml:"version"`
	APIVersion       string   `yaml:"api_version"`
	MigrationVersion string   `yaml:"migration_version"`
	AppID            string   `yaml:"app_id"`
	HostCapabilities []string `yaml:"host_capabilities"`
	ServiceBaseURL   string   `yaml:"service_base_url"`
	ServiceHealthURL string   `yaml:"service_health_url"`
	FrontendEntry    string   `yaml:"frontend_entry"`
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
	if manifest.Version != "0.12.0" || manifest.MigrationVersion != manifest.Version || contract.ContractVersion != manifest.Version || contract.Migrations.Version != manifest.Version {
		t.Fatalf("contract version mismatch: manifest=%q migration=%q map=%q", manifest.Version, manifest.MigrationVersion, contract.ContractVersion)
	}
	var frontendPackage struct {
		Version string `json:"version"`
	}
	packageRaw, err := os.ReadFile("frontend/package.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(packageRaw, &frontendPackage); err != nil {
		t.Fatal(err)
	}
	artifact := "pharma_oa-" + manifest.Version + ".zip"
	if frontendPackage.Version != manifest.Version ||
		!strings.Contains(pharmaReadText(t, "plugin.ps1"), artifact) ||
		!strings.Contains(pharmaReadText(t, "plugin.sh"), artifact) {
		t.Fatalf("package versions are not synchronized with plugin %s", manifest.Version)
	}
	if manifest.APIVersion != "v1" || contract.SchemaVersion != 1 || contract.PublicContract.APIVersion != manifest.APIVersion {
		t.Fatalf("API contract mismatch: manifest=%q map=%q schema=%d", manifest.APIVersion, contract.PublicContract.APIVersion, contract.SchemaVersion)
	}
	if manifest.ServiceBaseURL == "" || manifest.ServiceHealthURL == "" || contract.PublicContract.BackendClient != "github.com/tinboxw/skoll/pkg/pluginclient" {
		t.Fatal("managed process and public backend client are required")
	}
	capabilities := make(map[string]struct{}, len(manifest.HostCapabilities))
	for _, capability := range manifest.HostCapabilities {
		capabilities[capability] = struct{}{}
	}
	assertPharmaContains(t, capabilities, "events.publish", "inventory event capability")
	assertPharmaContains(t, capabilities, "datastore.aggregate", "inventory reconciliation capability")
	hostServices := make(map[string]struct{}, len(contract.PublicContract.HostServices))
	for _, service := range contract.PublicContract.HostServices {
		hostServices[service] = struct{}{}
	}
	assertPharmaContains(t, hostServices, "Events", "public host service")
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
	if len(documentOwners) != len(contract.DocumentTypes) || len(contract.DocumentTypes) != 14 {
		t.Fatalf("document ownership=%d types=%d want=14", len(documentOwners), len(contract.DocumentTypes))
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
		"backend/main.go", "backend/server.go", "backend/employee.go", "backend/party.go", "backend/catalog.go", "backend/qualification.go", "backend/oa_request.go", "backend/purchase.go", "backend/inbound.go", "backend/inventory.go", "backend/warehouse.go", "backend/warehouse_topology.go", "plugin.ps1", "plugin.sh", "frontend/index.html", "frontend/package.json", "frontend/src/App.vue", "frontend/src/api.ts", "frontend/src/i18n.ts", "frontend/src/styles.css", "datastore.yaml",
		"contract/acceptance-map.json", "migrations/001_foundation.up.sql", "migrations/001_foundation.down.sql",
		"migrations/002_employees.up.sql", "migrations/002_employees.down.sql",
		"migrations/003_parties.up.sql", "migrations/003_parties.down.sql",
		"migrations/004_catalogs.up.sql", "migrations/004_catalogs.down.sql",
		"migrations/005_qualifications.up.sql", "migrations/005_qualifications.down.sql",
		"migrations/006_oa_requests.up.sql", "migrations/006_oa_requests.down.sql",
		"migrations/007_purchases.up.sql", "migrations/007_purchases.down.sql",
		"migrations/008_purchase_inbounds.up.sql", "migrations/008_purchase_inbounds.down.sql",
		"migrations/009_warehouse_topology.up.sql", "migrations/009_warehouse_topology.down.sql",
		"migrations/010_inventory_ledger.up.sql", "migrations/010_inventory_ledger.down.sql",
		"migrations/011_inventory_movements.up.sql", "migrations/011_inventory_movements.down.sql",
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

func TestMedicalOAWarehouseTopologyPermissionsAndRoutes(t *testing.T) {
	manifest := loadPharmaManifest(t)
	permissions := make(map[string]struct{}, len(manifest.Permissions))
	for _, permission := range manifest.Permissions {
		permissions[permission.Key] = struct{}{}
	}
	for _, permission := range []string{
		"pharma_oa.warehouse.read", "pharma_oa.warehouse.create", "pharma_oa.warehouse.update",
		"pharma_oa.warehouse.disable", "pharma_oa.warehouse.enable", "pharma_oa.warehouse.movement",
		"pharma_oa.warehouse_area.read", "pharma_oa.warehouse_area.create", "pharma_oa.warehouse_area.update",
		"pharma_oa.warehouse_area.disable", "pharma_oa.warehouse_area.enable",
		"pharma_oa.warehouse_location.read", "pharma_oa.warehouse_location.create", "pharma_oa.warehouse_location.update",
		"pharma_oa.warehouse_location.disable", "pharma_oa.warehouse_location.enable",
	} {
		assertPharmaContains(t, permissions, permission, "warehouse topology permission")
	}

	routes := make(map[string]string, len(manifest.API.Routes))
	for _, route := range manifest.API.Routes {
		routes[strings.ToUpper(route.Method)+" "+route.Path] = route.Permission
	}
	for route, permission := range map[string]string{
		"GET /v1/plugins/pharma_oa/api/warehouses":                           "pharma_oa.warehouse.read",
		"POST /v1/plugins/pharma_oa/api/warehouses":                          "pharma_oa.warehouse.create",
		"PUT /v1/plugins/pharma_oa/api/warehouses/{id}":                      "pharma_oa.warehouse.update",
		"POST /v1/plugins/pharma_oa/api/warehouses/{id}/disable":             "pharma_oa.warehouse.disable",
		"POST /v1/plugins/pharma_oa/api/warehouses/{id}/enable":              "pharma_oa.warehouse.enable",
		"GET /v1/plugins/pharma_oa/api/warehouses/{id}/movement-eligibility": "pharma_oa.warehouse.movement",
		"GET /v1/plugins/pharma_oa/api/warehouse-areas":                      "pharma_oa.warehouse_area.read",
		"POST /v1/plugins/pharma_oa/api/warehouse-areas":                     "pharma_oa.warehouse_area.create",
		"PUT /v1/plugins/pharma_oa/api/warehouse-areas/{id}":                 "pharma_oa.warehouse_area.update",
		"POST /v1/plugins/pharma_oa/api/warehouse-areas/{id}/disable":        "pharma_oa.warehouse_area.disable",
		"POST /v1/plugins/pharma_oa/api/warehouse-areas/{id}/enable":         "pharma_oa.warehouse_area.enable",
		"GET /v1/plugins/pharma_oa/api/warehouse-locations":                  "pharma_oa.warehouse_location.read",
		"POST /v1/plugins/pharma_oa/api/warehouse-locations":                 "pharma_oa.warehouse_location.create",
		"PUT /v1/plugins/pharma_oa/api/warehouse-locations/{id}":             "pharma_oa.warehouse_location.update",
		"POST /v1/plugins/pharma_oa/api/warehouse-locations/{id}/disable":    "pharma_oa.warehouse_location.disable",
		"POST /v1/plugins/pharma_oa/api/warehouse-locations/{id}/enable":     "pharma_oa.warehouse_location.enable",
	} {
		if routes[route] != permission {
			t.Fatalf("warehouse topology route %q permission=%q want=%q", route, routes[route], permission)
		}
	}
}

func TestMedicalOAInventoryLedgerContractIsRegistered(t *testing.T) {
	manifest := loadPharmaManifest(t)
	contract := loadPharmaAcceptanceMap(t)

	permissions := make(map[string]struct{}, len(manifest.Permissions))
	for _, permission := range manifest.Permissions {
		permissions[permission.Key] = struct{}{}
	}
	for _, permission := range []string{"pharma_oa.inventory.read", "pharma_oa.inventory.receive"} {
		assertPharmaContains(t, permissions, permission, "inventory ledger permission")
	}

	type routeContract struct {
		permission  string
		auditAction string
	}
	routes := make(map[string]routeContract, len(manifest.API.Routes))
	for _, route := range manifest.API.Routes {
		routes[strings.ToUpper(route.Method)+" "+route.Path] = routeContract{
			permission: route.Permission, auditAction: route.AuditAction,
		}
	}
	for route, want := range map[string]routeContract{
		"GET /v1/plugins/pharma_oa/api/inventory-lots": {
			permission: "pharma_oa.inventory.read", auditAction: "pharma_oa.inventory_lot.read",
		},
		"GET /v1/plugins/pharma_oa/api/stock-ledger": {
			permission: "pharma_oa.inventory.read", auditAction: "pharma_oa.stock_ledger.read",
		},
		"GET /v1/plugins/pharma_oa/api/stock-balances": {
			permission: "pharma_oa.inventory.read", auditAction: "pharma_oa.stock_balance.read",
		},
		"GET /v1/plugins/pharma_oa/api/stock-reconciliation": {
			permission: "pharma_oa.inventory.read", auditAction: "pharma_oa.stock_reconciliation.read",
		},
	} {
		if got := routes[route]; got != want {
			t.Fatalf("inventory route %q=%+v want=%+v", route, got, want)
		}
	}

	inventoryWorkItems := map[string]struct{}{}
	for _, module := range contract.Modules {
		if module.ID != "inventory" {
			continue
		}
		for _, workItem := range module.WorkItems {
			inventoryWorkItems[workItem] = struct{}{}
		}
	}
	assertPharmaContains(t, inventoryWorkItems, "BF5-05B", "inventory work item")
	assertPharmaContains(t, inventoryWorkItems, "BF5-05C", "inventory movement work item")
	scenarios := make(map[string]struct{}, len(contract.AcceptanceScenarios))
	for _, scenario := range contract.AcceptanceScenarios {
		scenarios[scenario.ID] = struct{}{}
	}
	assertPharmaContains(t, scenarios, "POA-INVENTORY-02", "inventory ledger acceptance scenario")
	assertPharmaContains(t, scenarios, "POA-INVENTORY-03", "inventory movement acceptance scenario")

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
	type tableContract struct {
		policy       datastore.TableMutationPolicy
		fields       map[string]pluginsdk.DataValueType
		unique       []string
		aggregatable string
		groupable    []string
	}
	for tableName, want := range map[string]tableContract{
		"inventory_lots": {
			policy: datastore.TableMutationAppendOnly,
			fields: map[string]pluginsdk.DataValueType{
				"id": pluginsdk.DataValueString, "product_id": pluginsdk.DataValueString,
				"batch_no": pluginsdk.DataValueString, "production_date": pluginsdk.DataValueTimestamp,
				"expires_at": pluginsdk.DataValueTimestamp,
			},
			unique: []string{"product_id", "batch_no"},
		},
		"stock_ledger": {
			policy: datastore.TableMutationAppendOnly,
			fields: map[string]pluginsdk.DataValueType{
				"id": pluginsdk.DataValueString, "entry_type": pluginsdk.DataValueString,
				"product_id": pluginsdk.DataValueString, "lot_id": pluginsdk.DataValueString,
				"batch_no": pluginsdk.DataValueString, "warehouse_id": pluginsdk.DataValueString,
				"area_id": pluginsdk.DataValueString, "location_id": pluginsdk.DataValueString,
				"quantity_micros": pluginsdk.DataValueInteger, "source_document_type": pluginsdk.DataValueString,
				"source_document_id": pluginsdk.DataValueString, "source_document_number": pluginsdk.DataValueString,
				"source_document_line_id": pluginsdk.DataValueString, "occurred_at": pluginsdk.DataValueTimestamp,
			},
			unique:       []string{"source_document_type", "source_document_id", "source_document_line_id"},
			aggregatable: "quantity_micros",
			groupable:    []string{"product_id", "lot_id", "batch_no", "location_id"},
		},
		"stock_balances": {
			policy: datastore.TableMutationMutable,
			fields: map[string]pluginsdk.DataValueType{
				"id": pluginsdk.DataValueString, "product_id": pluginsdk.DataValueString,
				"lot_id": pluginsdk.DataValueString, "batch_no": pluginsdk.DataValueString,
				"warehouse_id": pluginsdk.DataValueString, "area_id": pluginsdk.DataValueString,
				"location_id": pluginsdk.DataValueString, "quantity_micros": pluginsdk.DataValueInteger,
			},
			unique:       []string{"product_id", "lot_id", "location_id"},
			aggregatable: "quantity_micros",
			groupable:    []string{"product_id", "lot_id", "batch_no", "location_id"},
		},
	} {
		table, ok := tables[tableName]
		if !ok {
			t.Fatalf("datastore schema is missing %s", tableName)
		}
		if table.MutationPolicy != want.policy {
			t.Fatalf("%s mutation policy=%q want framework policy %q", tableName, table.MutationPolicy, want.policy)
		}
		if len(table.PrimaryKey) != 1 || table.PrimaryKey[0] != "id" {
			t.Fatalf("%s primary key=%v want [id]", tableName, table.PrimaryKey)
		}
		if len(table.Fields) != len(want.fields) {
			t.Fatalf("%s fields=%d want exact set %d", tableName, len(table.Fields), len(want.fields))
		}
		fields := make(map[string]datastore.FieldSchema, len(table.Fields))
		for _, field := range table.Fields {
			fields[field.Name] = field
		}
		groupable := make(map[string]struct{}, len(want.groupable))
		for _, field := range want.groupable {
			groupable[field] = struct{}{}
		}
		for fieldName, fieldType := range want.fields {
			field, exists := fields[fieldName]
			if !exists || field.Type != fieldType {
				t.Fatalf("%s.%s=%+v want type %q", tableName, fieldName, field, fieldType)
			}
			if !field.Filterable {
				t.Fatalf("%s.%s must be filterable", tableName, fieldName)
			}
			if fieldName != "id" && !field.Mutable {
				t.Fatalf("%s.%s must accept its initial insert value", tableName, fieldName)
			}
			if field.Aggregatable != (fieldName == want.aggregatable) {
				t.Fatalf("%s.%s aggregatable=%t want=%t", tableName, fieldName, field.Aggregatable, fieldName == want.aggregatable)
			}
			_, wantGroupable := groupable[fieldName]
			if field.Groupable != wantGroupable {
				t.Fatalf("%s.%s groupable=%t want=%t", tableName, fieldName, field.Groupable, wantGroupable)
			}
		}
		if !pharmaHasUniqueIndex(table, want.unique) {
			t.Fatalf("%s is missing unique index %v", tableName, want.unique)
		}
	}

	up := pharmaReadText(t, "migrations/010_inventory_ledger.up.sql")
	if strings.Count(up, "quantity_micros BIGINT NOT NULL") != 2 {
		t.Fatal("inventory migration must store ledger and balance quantities as BIGINT")
	}
	requestHashAdd := strings.Index(up, "ALTER TABLE {{table:purchase_inbounds}} ADD COLUMN request_hash VARCHAR(64) NOT NULL DEFAULT '';")
	lotCreate := strings.Index(up, "CREATE TABLE {{table:inventory_lots}}")
	if requestHashAdd < 0 || lotCreate <= requestHashAdd {
		t.Fatal("inventory migration must add the receipt request hash before creating ledger tables")
	}
	down := pharmaReadText(t, "migrations/010_inventory_ledger.down.sql")
	balanceDrop := strings.Index(down, "{{table:stock_balances}}")
	ledgerDrop := strings.Index(down, "{{table:stock_ledger}}")
	lotDrop := strings.Index(down, "{{table:inventory_lots}}")
	requestHashDrop := strings.Index(down, "ALTER TABLE {{table:purchase_inbounds}} DROP COLUMN request_hash;")
	if balanceDrop < 0 || ledgerDrop <= balanceDrop || lotDrop <= ledgerDrop || requestHashDrop <= lotDrop {
		t.Fatal("inventory rollback must drop balance and ledger children before lots, then remove the receipt request hash")
	}
}

func TestMedicalOAInventoryMovementContractIsRegistered(t *testing.T) {
	manifest := loadPharmaManifest(t)
	contract := loadPharmaAcceptanceMap(t)

	permissions := make(map[string]struct{}, len(manifest.Permissions))
	for _, permission := range manifest.Permissions {
		permissions[permission.Key] = struct{}{}
	}
	for _, permission := range []string{
		"pharma_oa.stock_transfer.read", "pharma_oa.stock_transfer.create",
		"pharma_oa.stocktake.read", "pharma_oa.stocktake.create", "pharma_oa.stocktake.approve", "pharma_oa.stocktake.reject",
		"pharma_oa.stock_adjustment.read", "pharma_oa.stock_adjustment.create", "pharma_oa.stock_adjustment.approve", "pharma_oa.stock_adjustment.reject",
		"pharma_oa.stock_return.read", "pharma_oa.stock_return.create", "pharma_oa.stock_return.approve", "pharma_oa.stock_return.reject",
	} {
		assertPharmaContains(t, permissions, permission, "inventory movement permission")
	}
	for _, forbidden := range []string{"pharma_oa.stock_transfer.approve", "pharma_oa.stock_transfer.reject"} {
		if _, exists := permissions[forbidden]; exists {
			t.Fatalf("unimplemented stock transfer decision permission %q must not be declared", forbidden)
		}
	}

	type routeContract struct {
		permission  string
		auditAction string
	}
	wantRoutes := map[string]routeContract{
		"GET /v1/plugins/pharma_oa/api/stock-transfers":                 {permission: "pharma_oa.stock_transfer.read", auditAction: "pharma_oa.stock_transfer.read_list"},
		"GET /v1/plugins/pharma_oa/api/stock-transfers/{id}":            {permission: "pharma_oa.stock_transfer.read", auditAction: "pharma_oa.stock_transfer.read"},
		"POST /v1/plugins/pharma_oa/api/stock-transfers":                {permission: "pharma_oa.stock_transfer.create", auditAction: "pharma_oa.stock_transfer.create"},
		"GET /v1/plugins/pharma_oa/api/stocktakes":                      {permission: "pharma_oa.stocktake.read", auditAction: "pharma_oa.stocktake.read_list"},
		"GET /v1/plugins/pharma_oa/api/stocktakes/{id}":                 {permission: "pharma_oa.stocktake.read", auditAction: "pharma_oa.stocktake.read"},
		"POST /v1/plugins/pharma_oa/api/stocktakes":                     {permission: "pharma_oa.stocktake.create", auditAction: "pharma_oa.stocktake.create"},
		"POST /v1/plugins/pharma_oa/api/stocktakes/{id}/approve":        {permission: "pharma_oa.stocktake.approve", auditAction: "pharma_oa.stocktake.approve"},
		"POST /v1/plugins/pharma_oa/api/stocktakes/{id}/reject":         {permission: "pharma_oa.stocktake.reject", auditAction: "pharma_oa.stocktake.reject"},
		"GET /v1/plugins/pharma_oa/api/stock-adjustments":               {permission: "pharma_oa.stock_adjustment.read", auditAction: "pharma_oa.stock_adjustment.read_list"},
		"GET /v1/plugins/pharma_oa/api/stock-adjustments/{id}":          {permission: "pharma_oa.stock_adjustment.read", auditAction: "pharma_oa.stock_adjustment.read"},
		"POST /v1/plugins/pharma_oa/api/stock-adjustments":              {permission: "pharma_oa.stock_adjustment.create", auditAction: "pharma_oa.stock_adjustment.create"},
		"POST /v1/plugins/pharma_oa/api/stock-adjustments/{id}/approve": {permission: "pharma_oa.stock_adjustment.approve", auditAction: "pharma_oa.stock_adjustment.approve"},
		"POST /v1/plugins/pharma_oa/api/stock-adjustments/{id}/reject":  {permission: "pharma_oa.stock_adjustment.reject", auditAction: "pharma_oa.stock_adjustment.reject"},
		"GET /v1/plugins/pharma_oa/api/stock-returns":                   {permission: "pharma_oa.stock_return.read", auditAction: "pharma_oa.stock_return.read_list"},
		"GET /v1/plugins/pharma_oa/api/stock-returns/{id}":              {permission: "pharma_oa.stock_return.read", auditAction: "pharma_oa.stock_return.read"},
		"POST /v1/plugins/pharma_oa/api/stock-returns":                  {permission: "pharma_oa.stock_return.create", auditAction: "pharma_oa.stock_return.create"},
		"POST /v1/plugins/pharma_oa/api/stock-returns/{id}/approve":     {permission: "pharma_oa.stock_return.approve", auditAction: "pharma_oa.stock_return.approve"},
		"POST /v1/plugins/pharma_oa/api/stock-returns/{id}/reject":      {permission: "pharma_oa.stock_return.reject", auditAction: "pharma_oa.stock_return.reject"},
	}
	gotRoutes := make(map[string]routeContract)
	for _, route := range manifest.API.Routes {
		if !strings.Contains(route.Path, "/stock-transfers") &&
			!strings.Contains(route.Path, "/stocktakes") &&
			!strings.Contains(route.Path, "/stock-adjustments") &&
			!strings.Contains(route.Path, "/stock-returns") {
			continue
		}
		key := strings.ToUpper(route.Method) + " " + route.Path
		gotRoutes[key] = routeContract{permission: route.Permission, auditAction: route.AuditAction}
	}
	if len(gotRoutes) != len(wantRoutes) {
		t.Fatalf("inventory movement routes=%d want exact surface=%d: %+v", len(gotRoutes), len(wantRoutes), gotRoutes)
	}
	for route, want := range wantRoutes {
		if got := gotRoutes[route]; got != want {
			t.Fatalf("inventory movement route %q=%+v want=%+v", route, got, want)
		}
	}

	inventoryDocumentTypes := map[string]struct{}{}
	for _, module := range contract.Modules {
		if module.ID != "inventory" {
			continue
		}
		for _, documentType := range module.DocumentTypes {
			inventoryDocumentTypes[documentType] = struct{}{}
		}
	}
	for _, documentType := range []string{"stock_transfer", "stocktake", "stock_adjustment", "stock_return"} {
		assertPharmaContains(t, inventoryDocumentTypes, documentType, "inventory document type")
	}

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
	movement, exists := tables["inventory_movements"]
	if !exists || movement.MutationPolicy != datastore.TableMutationMutable ||
		len(movement.PrimaryKey) != 1 || movement.PrimaryKey[0] != "id" {
		t.Fatalf("inventory movement table contract is incomplete: %+v", movement)
	}
	movementTypes := map[string]pluginsdk.DataValueType{
		"id": pluginsdk.DataValueString, "number": pluginsdk.DataValueString,
		"movement_type": pluginsdk.DataValueString, "return_type": pluginsdk.DataValueString,
		"reason":              pluginsdk.DataValueString,
		"source_warehouse_id": pluginsdk.DataValueString, "source_area_id": pluginsdk.DataValueString,
		"source_location_id": pluginsdk.DataValueString, "destination_warehouse_id": pluginsdk.DataValueString,
		"destination_area_id": pluginsdk.DataValueString, "destination_location_id": pluginsdk.DataValueString,
		"legs": pluginsdk.DataValueJSON, "requester_id": pluginsdk.DataValueString,
		"requested_at": pluginsdk.DataValueTimestamp, "approver_id": pluginsdk.DataValueString,
		"approver_name": pluginsdk.DataValueString, "workflow_definition_id": pluginsdk.DataValueString,
		"workflow_instance_id": pluginsdk.DataValueString, "status": pluginsdk.DataValueString,
		"create_operation_key": pluginsdk.DataValueString, "request_hash": pluginsdk.DataValueString,
		"last_operation_key": pluginsdk.DataValueString, "last_operation_hash": pluginsdk.DataValueString,
		"decision_comment": pluginsdk.DataValueString, "decided_at": pluginsdk.DataValueTimestamp,
		"posted_at": pluginsdk.DataValueTimestamp,
	}
	if len(movement.Fields) != len(movementTypes) {
		t.Fatalf("inventory movement fields=%d want exact current set=%d", len(movement.Fields), len(movementTypes))
	}
	movementFields := make(map[string]datastore.FieldSchema, len(movement.Fields))
	for _, field := range movement.Fields {
		movementFields[field.Name] = field
	}
	for fieldName, fieldType := range movementTypes {
		field, ok := movementFields[fieldName]
		if !ok || field.Type != fieldType {
			t.Fatalf("inventory_movements.%s=%+v want type=%q", fieldName, field, fieldType)
		}
		if fieldName != "id" && !field.Mutable {
			t.Fatalf("inventory_movements.%s must accept its initial insert value", fieldName)
		}
	}
	for _, nullable := range []string{
		"return_type",
		"source_warehouse_id", "source_area_id", "source_location_id",
		"destination_warehouse_id", "destination_area_id", "destination_location_id",
		"approver_id", "approver_name", "workflow_definition_id", "workflow_instance_id",
		"decision_comment", "decided_at", "posted_at",
	} {
		if !movementFields[nullable].Nullable {
			t.Fatalf("inventory_movements.%s must be nullable across current movement types and states", nullable)
		}
	}
	if !pharmaHasUniqueIndex(movement, []string{"number"}) ||
		!pharmaHasUniqueIndex(movement, []string{"create_operation_key"}) {
		t.Fatal("inventory movement is missing its scoped number or create-operation uniqueness")
	}

	returnTotals, exists := tables["stock_return_totals"]
	if !exists || returnTotals.MutationPolicy != datastore.TableMutationMutable ||
		len(returnTotals.Fields) != 4 || !pharmaHasUniqueIndex(returnTotals, []string{"reference_ledger_entry_id", "return_type"}) {
		t.Fatalf("stock return total table contract is incomplete: %+v", returnTotals)
	}
	returnFields := make(map[string]datastore.FieldSchema, len(returnTotals.Fields))
	for _, field := range returnTotals.Fields {
		returnFields[field.Name] = field
	}
	for _, fieldName := range []string{"reference_ledger_entry_id", "return_type", "quantity_micros"} {
		if !returnFields[fieldName].Mutable || !returnFields[fieldName].Filterable {
			t.Fatalf("stock_return_totals.%s must support scoped initial insertion and lookup", fieldName)
		}
	}
	if quantity := returnFields["quantity_micros"]; quantity.Type != pluginsdk.DataValueInteger || !quantity.Aggregatable {
		t.Fatalf("stock return total quantity must be an exact aggregatable integer: %+v", quantity)
	}

	up := pharmaReadText(t, "migrations/011_inventory_movements.up.sql")
	movementCreate := strings.Index(up, "CREATE TABLE {{table:inventory_movements}}")
	totalCreate := strings.Index(up, "CREATE TABLE {{table:stock_return_totals}}")
	if movementCreate < 0 || totalCreate <= movementCreate ||
		!strings.Contains(up, "quantity_micros BIGINT NOT NULL") ||
		strings.Contains(up, "{{table:stock_ledger}}") {
		t.Fatal("movement migration must create the document before its integer return projection without changing stock_ledger")
	}
	down := pharmaReadText(t, "migrations/011_inventory_movements.down.sql")
	totalDrop := strings.Index(down, "{{table:stock_return_totals}}")
	movementDrop := strings.Index(down, "{{table:inventory_movements}}")
	if totalDrop < 0 || movementDrop <= totalDrop || strings.Contains(down, "{{table:stock_ledger}}") {
		t.Fatal("movement rollback must drop the return projection before movement documents without changing stock_ledger")
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
			"request_hash": pluginsdk.DataValueString,
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
	inboundFields := make(map[string]datastore.FieldSchema)
	for _, field := range tables["purchase_inbounds"].Fields {
		inboundFields[field.Name] = field
	}
	requestHash := inboundFields["request_hash"]
	if requestHash.Type != pluginsdk.DataValueString || !requestHash.Mutable || !requestHash.Filterable || requestHash.Nullable {
		t.Fatalf("purchase_inbounds.request_hash must be required, mutable, and filterable: %+v", requestHash)
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
	for _, path := range []string{"migrations/001_foundation.up.sql", "migrations/002_employees.up.sql", "migrations/003_parties.up.sql", "migrations/004_catalogs.up.sql", "migrations/005_qualifications.up.sql", "migrations/006_oa_requests.up.sql", "migrations/007_purchases.up.sql", "migrations/008_purchase_inbounds.up.sql", "migrations/009_warehouse_topology.up.sql", "migrations/010_inventory_ledger.up.sql", "migrations/011_inventory_movements.up.sql"} {
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
		sql = strings.ReplaceAll(sql, "{{table:inventory_lots}}", "pharma_oa_inventory_lots")
		sql = strings.ReplaceAll(sql, "{{table:stock_ledger}}", "pharma_oa_stock_ledger")
		sql = strings.ReplaceAll(sql, "{{table:stock_balances}}", "pharma_oa_stock_balances")
		sql = strings.ReplaceAll(sql, "{{table:inventory_movements}}", "pharma_oa_inventory_movements")
		sql = strings.ReplaceAll(sql, "{{table:stock_return_totals}}", "pharma_oa_stock_return_totals")
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	if !db.Migrator().HasColumn("pharma_oa_purchase_inbounds", "request_hash") {
		t.Fatal("inventory migration did not add purchase inbound request_hash")
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
	contractTables := make(map[string]struct{})
	for _, table := range loadPharmaAcceptanceMap(t).Migrations.Tables {
		assertPharmaContains(t, wantTables, table, "contract migration table")
		contractTables[table] = struct{}{}
	}
	if len(contractTables) != len(wantTables) {
		t.Fatalf("contract migration tables=%d want exact manifest set=%d", len(contractTables), len(wantTables))
	}
	for table := range wantTables {
		assertPharmaContains(t, contractTables, table, "manifest table in contract migration map")
	}
	assertWarehouseTopologyScopedUniqueness(t, db)
	assertInventoryLedgerScopedProjection(t, db)
	assertInventoryMovementScopedProjection(t, db)
	for _, path := range []string{"migrations/011_inventory_movements.down.sql", "migrations/010_inventory_ledger.down.sql", "migrations/009_warehouse_topology.down.sql", "migrations/008_purchase_inbounds.down.sql", "migrations/007_purchases.down.sql", "migrations/006_oa_requests.down.sql", "migrations/005_qualifications.down.sql", "migrations/004_catalogs.down.sql", "migrations/003_parties.down.sql", "migrations/002_employees.down.sql", "migrations/001_foundation.down.sql"} {
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
		sql = strings.ReplaceAll(sql, "{{table:inventory_lots}}", "pharma_oa_inventory_lots")
		sql = strings.ReplaceAll(sql, "{{table:stock_ledger}}", "pharma_oa_stock_ledger")
		sql = strings.ReplaceAll(sql, "{{table:stock_balances}}", "pharma_oa_stock_balances")
		sql = strings.ReplaceAll(sql, "{{table:inventory_movements}}", "pharma_oa_inventory_movements")
		sql = strings.ReplaceAll(sql, "{{table:stock_return_totals}}", "pharma_oa_stock_return_totals")
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("rollback %s: %v", path, err)
		}
		if path == "migrations/010_inventory_ledger.down.sql" && db.Migrator().HasColumn("pharma_oa_purchase_inbounds", "request_hash") {
			t.Fatal("inventory rollback retained purchase inbound request_hash")
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

func assertInventoryLedgerScopedProjection(t *testing.T, db *gorm.DB) {
	t.Helper()
	const (
		timestamp = "2026-07-28T00:00:00Z"
		expiresAt = "2027-07-28T00:00:00Z"
	)
	insertLot := func(id, organization string) error {
		return db.Exec(`INSERT INTO pharma_oa_inventory_lots
			(id, product_id, batch_no, production_date, expires_at, tenant_id, organization_id, owner_id, version, created_at, updated_at)
			VALUES (?, 'product-a', 'BATCH-001', ?, ?, 'tenant-a', ?, 'owner-a', 1, ?, ?)`,
			id, timestamp, expiresAt, organization, timestamp, timestamp).Error
	}
	if err := insertLot("lot-a", "org-a"); err != nil {
		t.Fatal(err)
	}
	if err := insertLot("lot-duplicate", "org-a"); err == nil {
		t.Fatal("product and batch must be unique inside one tenant organization")
	}
	if err := insertLot("lot-b", "org-b"); err != nil {
		t.Fatalf("product and batch should be reusable in another organization: %v", err)
	}

	insertLedger := func(id, sourceLineID string, quantityMicros int64) error {
		return db.Exec(`INSERT INTO pharma_oa_stock_ledger
			(id, entry_type, product_id, lot_id, batch_no, warehouse_id, area_id, location_id, quantity_micros,
			 source_document_type, source_document_id, source_document_number, source_document_line_id, occurred_at,
			 tenant_id, organization_id, owner_id, version, created_at, updated_at)
			VALUES (?, 'receipt', 'product-a', 'lot-a', 'BATCH-001', 'warehouse-a', 'area-a', 'location-a', ?,
			 'purchase_inbound', 'inbound-a', 'PI-001', ?, ?, 'tenant-a', 'org-a', 'owner-a', 1, ?, ?)`,
			id, quantityMicros, sourceLineID, timestamp, timestamp, timestamp).Error
	}
	if err := insertLedger("ledger-a", "line-a", 500_000); err != nil {
		t.Fatal(err)
	}
	if err := insertLedger("ledger-b", "line-b", 750_000); err != nil {
		t.Fatal(err)
	}
	if err := insertLedger("ledger-duplicate", "line-a", 1); err == nil {
		t.Fatal("one source document line must create at most one stock ledger entry")
	}

	insertBalance := func(id string, quantityMicros int64) error {
		return db.Exec(`INSERT INTO pharma_oa_stock_balances
			(id, product_id, lot_id, batch_no, warehouse_id, area_id, location_id, quantity_micros,
			 tenant_id, organization_id, owner_id, version, created_at, updated_at)
			VALUES (?, 'product-a', 'lot-a', 'BATCH-001', 'warehouse-a', 'area-a', 'location-a', ?,
			 'tenant-a', 'org-a', 'owner-a', 1, ?, ?)`,
			id, quantityMicros, timestamp, timestamp).Error
	}
	if err := insertBalance("balance-a", 1_250_000); err != nil {
		t.Fatal(err)
	}
	if err := insertBalance("balance-duplicate", 1_250_000); err == nil {
		t.Fatal("product, lot, and location must identify one balance projection")
	}

	var ledgerTotal int64
	if err := db.Raw(`SELECT COALESCE(SUM(quantity_micros), 0) FROM pharma_oa_stock_ledger
		WHERE tenant_id = 'tenant-a' AND organization_id = 'org-a'
		  AND product_id = 'product-a' AND lot_id = 'lot-a' AND location_id = 'location-a'`).
		Scan(&ledgerTotal).Error; err != nil {
		t.Fatal(err)
	}
	var balanceTotal int64
	if err := db.Raw(`SELECT COALESCE(SUM(quantity_micros), 0) FROM pharma_oa_stock_balances
		WHERE tenant_id = 'tenant-a' AND organization_id = 'org-a'
		  AND product_id = 'product-a' AND lot_id = 'lot-a' AND location_id = 'location-a'`).
		Scan(&balanceTotal).Error; err != nil {
		t.Fatal(err)
	}
	if ledgerTotal != 1_250_000 || balanceTotal != ledgerTotal {
		t.Fatalf("ledger total=%d balance total=%d want exact integer projection 1250000", ledgerTotal, balanceTotal)
	}
	for _, table := range []string{"pharma_oa_stock_ledger", "pharma_oa_stock_balances"} {
		var nonInteger int64
		if err := db.Raw("SELECT COUNT(*) FROM " + table + " WHERE typeof(quantity_micros) <> 'integer'").Scan(&nonInteger).Error; err != nil {
			t.Fatal(err)
		}
		if nonInteger != 0 {
			t.Fatalf("%s contains %d non-integer quantity values", table, nonInteger)
		}
	}
}

func assertInventoryMovementScopedProjection(t *testing.T, db *gorm.DB) {
	t.Helper()
	const timestamp = "2026-07-28T08:00:00Z"
	insertMovement := func(id, number, operationKey, organization string) error {
		return db.Exec(`INSERT INTO pharma_oa_inventory_movements
			(id, number, movement_type, reason, legs, requester_id, requested_at, status,
			 create_operation_key, request_hash, last_operation_key, last_operation_hash,
			 tenant_id, organization_id, owner_id, version, created_at, updated_at)
			VALUES (?, ?, 'stock_transfer', 'Replenish picking location', '[]', 'requester-a', ?, 'posted',
			 ?, 'request-hash', ?, 'operation-hash',
			 'tenant-a', ?, 'owner-a', 1, ?, ?)`,
			id, number, timestamp, operationKey, operationKey, organization, timestamp, timestamp).Error
	}
	if err := insertMovement("movement-a", "MV-001", "movement-a.create", "org-a"); err != nil {
		t.Fatal(err)
	}
	if err := insertMovement("movement-number-duplicate", "MV-001", "movement-b.create", "org-a"); err == nil {
		t.Fatal("movement number must be unique inside one tenant organization")
	}
	if err := insertMovement("movement-operation-duplicate", "MV-002", "movement-a.create", "org-a"); err == nil {
		t.Fatal("movement create operation must be unique inside one tenant organization")
	}
	if err := insertMovement("movement-other-organization", "MV-001", "movement-a.create", "org-b"); err != nil {
		t.Fatalf("movement identities should be reusable in another organization: %v", err)
	}

	insertReturnTotal := func(id, referenceID, returnType, organization string, quantity int64) error {
		return db.Exec(`INSERT INTO pharma_oa_stock_return_totals
			(id, reference_ledger_entry_id, return_type, quantity_micros,
			 tenant_id, organization_id, owner_id, version, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'tenant-a', ?, 'owner-a', 1, ?, ?)`,
			id, referenceID, returnType, quantity, organization, timestamp, timestamp).Error
	}
	if err := insertReturnTotal("return-total-a", "ledger-a", "supplier_return", "org-a", 100); err != nil {
		t.Fatal(err)
	}
	if err := insertReturnTotal("return-total-duplicate", "ledger-a", "supplier_return", "org-a", 1); err == nil {
		t.Fatal("one reference ledger entry and return type must have one cumulative projection")
	}
	if err := insertReturnTotal("return-total-type", "ledger-a", "customer_return", "org-a", 1); err != nil {
		t.Fatalf("different return types must have independent totals: %v", err)
	}
	if err := insertReturnTotal("return-total-organization", "ledger-a", "supplier_return", "org-b", 1); err != nil {
		t.Fatalf("return references should be reusable in another organization: %v", err)
	}

	operation := db.Exec(`UPDATE pharma_oa_stock_return_totals
		SET quantity_micros = quantity_micros + 600, version = version + 1, updated_at = ?
		WHERE id = 'return-total-a' AND tenant_id = 'tenant-a' AND organization_id = 'org-a'
		  AND quantity_micros + 600 >= 0 AND quantity_micros + 600 <= 1000`, timestamp)
	if operation.Error != nil || operation.RowsAffected != 1 {
		t.Fatalf("bounded return adjustment failed: rows=%d err=%v", operation.RowsAffected, operation.Error)
	}
	operation = db.Exec(`UPDATE pharma_oa_stock_return_totals
		SET quantity_micros = quantity_micros + 301, version = version + 1, updated_at = ?
		WHERE id = 'return-total-a' AND tenant_id = 'tenant-a' AND organization_id = 'org-a'
		  AND quantity_micros + 301 >= 0 AND quantity_micros + 301 <= 1000`, timestamp)
	if operation.Error != nil || operation.RowsAffected != 0 {
		t.Fatalf("return upper bound was not enforced: rows=%d err=%v", operation.RowsAffected, operation.Error)
	}
	var row struct {
		QuantityMicros int64
		Version        int64
		StorageType    string
	}
	if err := db.Raw(`SELECT quantity_micros, version, typeof(quantity_micros) AS storage_type
		FROM pharma_oa_stock_return_totals WHERE id = 'return-total-a'`).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.QuantityMicros != 700 || row.Version != 2 || row.StorageType != "integer" {
		t.Fatalf("return projection=%+v want exact integer quantity=700 version=2", row)
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

func pharmaHasUniqueIndex(table datastore.TableSchema, fields []string) bool {
	for _, index := range table.Indexes {
		if index.Unique && strings.Join(index.Fields, "\x00") == strings.Join(fields, "\x00") {
			return true
		}
	}
	return false
}
