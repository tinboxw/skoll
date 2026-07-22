package pharmaoa

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

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
	Permissions      []struct {
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
	if manifest.Version != "0.2.0" || manifest.MigrationVersion != manifest.Version || contract.ContractVersion != manifest.Version || contract.Migrations.Version != manifest.Version {
		t.Fatalf("contract version mismatch: manifest=%q migration=%q map=%q", manifest.Version, manifest.MigrationVersion, contract.ContractVersion)
	}
	if manifest.APIVersion != "v1" || contract.SchemaVersion != 1 || contract.PublicContract.APIVersion != manifest.APIVersion {
		t.Fatalf("API contract mismatch: manifest=%q map=%q schema=%d", manifest.APIVersion, contract.PublicContract.APIVersion, contract.SchemaVersion)
	}
	if manifest.ServiceBaseURL == "" || manifest.ServiceHealthURL == "" || contract.PublicContract.BackendClient != "github.com/tinboxw/skoll/pkg/pluginclient" {
		t.Fatal("managed process and public backend client are required")
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
		"backend/main.go", "backend/server.go", "plugin.ps1", "plugin.sh", "static/index.html",
		"contract/acceptance-map.json", "migrations/001_foundation.up.sql", "migrations/001_foundation.down.sql",
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
	up, err := os.ReadFile("migrations/001_foundation.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(up)).Error; err != nil {
		t.Fatalf("apply foundation migration: %v", err)
	}
	manifest := loadPharmaManifest(t)
	wantTables := make(map[string]struct{}, len(manifest.Data.Tables))
	for _, table := range manifest.Data.Tables {
		wantTables[table.Name] = struct{}{}
		if !db.Migrator().HasTable(table.Name) {
			t.Fatalf("migration did not create %s", table.Name)
		}
		columns := pharmaCSVSet(table.Columns)
		for _, column := range []string{"tenant_id", "organization_id", "owner_id"} {
			assertPharmaContains(t, columns, column, "scope column in "+table.Name)
		}
	}
	for _, table := range loadPharmaAcceptanceMap(t).Migrations.Tables {
		assertPharmaContains(t, wantTables, table, "contract migration table")
	}
	down, err := os.ReadFile("migrations/001_foundation.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(down)).Error; err != nil {
		t.Fatalf("rollback foundation migration: %v", err)
	}
	for table := range wantTables {
		if db.Migrator().HasTable(table) {
			t.Fatalf("rollback retained %s", table)
		}
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
