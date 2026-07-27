package generator

import (
	"context"
	"encoding/json"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/plugin/datastore"
)

func TestGeneratedFullStackPluginUsesOneCurrentContract(t *testing.T) {
	spec := generatedPluginE2ESpec(t, "http://127.0.0.1:19090")
	result, err := NewService().DryRun(context.Background(), DryRunInput{
		Spec: spec, BatchID: "ff5-fullstack-contract", ActorID: "generator-test", MigrationTimestamp: "20260722_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	for _, file := range result.Files {
		if !strings.HasPrefix(file.Path, "examples/plugins/pharma-oa/") {
			t.Fatalf("plugin generation crossed into core code: %s", file.Path)
		}
	}
	pluginDir := materializeGeneratedPlugin(t, result, spec.Plugin.ID)

	info, err := pluginruntime.NewFileLoader().Load(pluginDir)
	if err != nil {
		t.Fatalf("load generated current manifest: %v", err)
	}
	schema, present, err := datastore.LoadSchemaManifest(spec.Plugin.ID, pluginDir)
	if err != nil || !present {
		t.Fatalf("load generated datastore schema: present=%v err=%v", present, err)
	}

	var contract generatedPluginContract
	contractPath := filepath.Join(pluginDir, "contract.json")
	raw := findPlan(t, result.Files, "examples/plugins/pharma-oa/contract.json").GeneratedContent
	if err := json.Unmarshal([]byte(raw), &contract); err != nil {
		t.Fatalf("decode generated contract %s: %v", contractPath, err)
	}
	if contract.SchemaVersion != 1 || contract.Plugin.ID != spec.Plugin.ID || contract.Plugin.DataNamespace != spec.Plugin.DataNamespace {
		t.Fatalf("plugin identity contract = %+v", contract.Plugin)
	}
	if contract.Data.LogicalTable != "products" || contract.Data.PrimaryField != "id" ||
		len(schema.Tables) != 1 || schema.Tables[0].Name != contract.Data.LogicalTable ||
		len(info.DataManifest.Tables) != 1 || info.DataManifest.Tables[0].Name != contract.Data.LogicalTable {
		t.Fatalf("data contracts disagree: contract=%+v schema=%+v", contract.Data, schema)
	}
	if contract.API.BasePath != pluginAPIBasePath(*spec) || len(contract.API.Routes) != len(info.APIContract.Routes) {
		t.Fatalf("API contracts disagree: contract=%+v manifest=%+v", contract.API, info.APIContract)
	}
	for index, route := range contract.API.Routes {
		manifestRoute := info.APIContract.Routes[index]
		if route.Method != manifestRoute.Method || route.Path != manifestRoute.Path ||
			route.Permission != manifestRoute.Permission || route.AuditAction != manifestRoute.AuditAction {
			t.Fatalf("route %d disagrees: contract=%+v manifest=%+v", index, route, manifestRoute)
		}
	}
	if contract.Permissions.ReadKey != spec.Permissions.ReadKey ||
		contract.UI.FrontendEntry != info.FrontendEntry ||
		contract.UI.MenuKey != info.UIMenu.Key ||
		strings.Join(contract.UI.RequiredPermissions, ",") != strings.Join(info.UIMenu.RequiredPermissions, ",") {
		t.Fatalf("permission/UI contracts disagree: contract=%+v/%+v manifest=%+v", contract.Permissions, contract.UI, info)
	}
	if info.EventContract == nil || len(info.EventContract.Publications) != 1 || len(info.EventContract.Subscriptions) != 1 ||
		len(contract.Events.Publications) != 1 || len(contract.Events.Subscriptions) != 1 ||
		contract.Events.Publications[0].Name != info.EventContract.Publications[0].Name ||
		contract.Events.Subscriptions[0].Publisher != info.EventContract.Subscriptions[0].Publisher {
		t.Fatalf("event contracts disagree: contract=%+v manifest=%+v", contract.Events, info.EventContract)
	}
	if contract.Workflow == nil || contract.Workflow.DefinitionID != spec.Document.DefinitionID ||
		contract.Workflow.SchemaKey != spec.Document.SchemaKey || contract.Workflow.BusinessType != spec.Document.SchemaKey {
		t.Fatalf("workflow contract = %+v", contract.Workflow)
	}
	if strings.Join(contract.Locales, ",") != strings.Join(info.I18nLocales, ",") {
		t.Fatalf("locale contracts disagree: contract=%v manifest=%v", contract.Locales, info.I18nLocales)
	}

	up := findPlan(t, result.Files, "examples/plugins/pharma-oa/migrations/001_create_products.up.sql").GeneratedContent
	down := findPlan(t, result.Files, "examples/plugins/pharma-oa/migrations/001_create_products.down.sql").GeneratedContent
	if !strings.Contains(up, "CREATE TABLE {{table:products}}") ||
		!strings.Contains(down, "DROP TABLE IF EXISTS {{table:products}}") ||
		strings.Contains(up, "CREATE TABLE pharma_oa_products") {
		t.Fatalf("migration does not use the current logical table binding:\n%s\n%s", up, down)
	}

	for _, path := range []string{
		"examples/plugins/pharma-oa/backend/contract.go",
		"examples/plugins/pharma-oa/backend/main.go",
		"examples/plugins/pharma-oa/plugin_acceptance_test.go",
	} {
		file := findPlan(t, result.Files, path)
		if _, err := parser.ParseFile(token.NewFileSet(), path, file.GeneratedContent, parser.AllErrors); err != nil {
			t.Fatalf("generated Go contract is invalid at %s: %v\n%s", path, err, file.GeneratedContent)
		}
	}
	backend := findPlan(t, result.Files, "examples/plugins/pharma-oa/backend/main.go").GeneratedContent
	backendContract := findPlan(t, result.Files, "examples/plugins/pharma-oa/backend/contract.go").GeneratedContent
	frontendAPI := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/api/product.ts").GeneratedContent
	frontendView := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/views/Product/index.vue").GeneratedContent
	frontendContract := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/contract.ts").GeneratedContent
	for _, marker := range []string{
		`"/v1/plugins/pharma-oa/api/products"`,
		`generatedLogicalTable`,
		`host.Documents.Submit`,
		`pluginContract.api.basePath`,
		`pluginContract.permissions`,
		`"definitionId": "product-request-approval"`,
	} {
		if !strings.Contains(backend+backendContract+frontendAPI+frontendView+frontendContract, marker) {
			t.Fatalf("generated full-stack contract missing %q", marker)
		}
	}
	for _, forbidden := range []string{"memoryStore", "offset?: number", "CREATE TABLE pharma_oa_products"} {
		if strings.Contains(backend+frontendAPI+up, forbidden) {
			t.Fatalf("generated current contract contains obsolete marker %q", forbidden)
		}
	}

	repeated, err := NewService().DryRun(context.Background(), DryRunInput{
		Spec: spec, BatchID: "ff5-fullstack-contract-repeat", ActorID: "generator-test", MigrationTimestamp: "20260722_010203",
	})
	if err != nil {
		t.Fatalf("repeated DryRun() error = %v", err)
	}
	for _, path := range []string{
		"examples/plugins/pharma-oa/contract.json",
		"examples/plugins/pharma-oa/datastore.yaml",
		"examples/plugins/pharma-oa/backend/contract.go",
		"examples/plugins/pharma-oa/web/src/contract.ts",
	} {
		first := findPlan(t, result.Files, path)
		second := findPlan(t, repeated.Files, path)
		if first.ContentHash != second.ContentHash || first.GeneratedContent != second.GeneratedContent {
			t.Fatalf("generated contract is not deterministic: %s", path)
		}
	}
}

func TestGeneratedCRUDPluginUsesHostDatastoreContract(t *testing.T) {
	result, err := NewService().DryRun(context.Background(), DryRunInput{
		Spec: mustPluginSpec(t), BatchID: "ff5-datastore-contract", ActorID: "generator-test", MigrationTimestamp: "20260722_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	backend := findPlan(t, result.Files, "examples/plugins/pharma-oa/backend/main.go").GeneratedContent
	store := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/stores/product.ts").GeneratedContent
	api := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/api/product.ts").GeneratedContent
	for _, marker := range []string{
		"pluginclient.FromEnvironment",
		"host.DataStore.Query",
		"host.DataStore.Mutate",
		"host.Transactions.Within",
		"pluginsdk.DataPageRequest",
		"cursor?: string",
		"cursorHistory",
	} {
		if !strings.Contains(backend+store+api, marker) {
			t.Fatalf("generated CRUD datastore contract missing %q", marker)
		}
	}
	for _, forbidden := range []string{"memoryStore", "offset?: number", "map[string]map[string]any"} {
		if strings.Contains(backend+store+api, forbidden) {
			t.Fatalf("generated CRUD datastore contract contains obsolete marker %q", forbidden)
		}
	}
}
