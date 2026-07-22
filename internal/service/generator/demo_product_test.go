package generator

import (
	"context"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
)

func TestDemoProductGenerationAcceptance(t *testing.T) {
	spec := loadDemoProductSpec(t)
	svc := NewService()
	result, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "demo-product-acceptance",
		ActorID:            "codex",
		MigrationTimestamp: "20260629_030000",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	if result.Summary.Conflict != 0 || result.Summary.Blocked != 0 {
		t.Fatalf("dry-run has conflicts or blocked files: %+v", result.Summary)
	}

	goPaths := []string{
		"internal/domain/demo_product/entity.go",
		"internal/domain/demo_product/entity_test.go",
		"internal/repository/demo_product/demo_product_repo.go",
		"internal/store/memory/demo_product_store.go",
		"internal/store/memory/demo_product_store_test.go",
		"internal/store/sql/gormrepo/demo_product_model.go",
		"internal/store/sql/gormrepo/demo_product_store.go",
		"internal/store/sql/gormrepo/demo_product_store_test.go",
		"internal/service/demo_product/service.go",
		"internal/service/demo_product/service_impl.go",
		"internal/service/demo_product/service_impl_test.go",
		"internal/handler/http/v1/demo_product/handler.go",
	}
	for _, path := range goPaths {
		plan := findPlan(t, result.Files, path)
		if _, err := parser.ParseFile(token.NewFileSet(), path, plan.GeneratedContent, parser.AllErrors); err != nil {
			t.Fatalf("generated Go for %s is invalid: %v\n%s", path, err, plan.GeneratedContent)
		}
	}

	assertGeneratedContains(t, result, "migrations/mysql/20260629_030000_create_demo_products.sql",
		"CREATE TABLE IF NOT EXISTS demo_products",
		"PRIMARY KEY (id)",
		"created_at TIMESTAMP NOT NULL",
		"updated_at TIMESTAMP NOT NULL",
		"idx_demo_products_name",
	)
	assertGeneratedContains(t, result, "docs/api/openapi.yaml", "/demo-products:", "operationId: listDemoProduct", "DemoProduct:")
	assertGeneratedContains(t, result, "web/src/api/demo_product.ts", "export type DemoProduct", "listDemoProduct", "createDemoProduct")
	assertGeneratedContains(t, result, "web/src/stores/demo_product.ts", "useDemoProductStore", "listError", "detailStatus", "async retry")
	assertGeneratedContains(t, result, "web/src/i18n/generated_demo_product.ts", "translateDemoProduct", `"zh-CN"`, `"en-US"`)
	assertGeneratedContains(t, result, "web/src/router/generated_demo_product.ts", "demo_productRoutes", `permissions: ["demo_product.read"]`)
	assertGeneratedContains(t, result, "web/src/views/DemoProduct/index.vue", "<DataTable", "<DetailDrawer", "demo_product.create")
	assertDemoProductBackendContract(t, result)
	assertDemoProductFrontendContract(t, result)

	history, err := svc.RecordHistory(context.Background(), RecordHistoryInput{
		DryRun:    result,
		Spec:      spec,
		CreatedAt: time.Date(2026, 6, 29, 3, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("RecordHistory() error = %v", err)
	}
	if history.SpecID != "spec-demo-product" || history.ActorID != "codex" || len(history.Files) != len(result.Files) {
		t.Fatalf("history = %+v", history)
	}

	current := make([]FileSnapshot, 0, len(result.Files))
	for _, file := range result.Files {
		current = append(current, FileSnapshot{Path: file.Path, CurrentHash: file.ContentHash})
	}
	rollback, err := svc.PlanRollback(context.Background(), RollbackInput{
		BatchID:      "demo-product-acceptance",
		CurrentFiles: current,
	})
	if err != nil {
		t.Fatalf("PlanRollback() error = %v", err)
	}
	if len(rollback.Conflicts) != 0 {
		t.Fatalf("rollback conflicts = %+v", rollback.Conflicts)
	}
}

func TestDemoProductBackendAcceptanceContract(t *testing.T) {
	spec := loadDemoProductSpec(t)
	svc := NewService()
	result, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "demo-product-backend-acceptance",
		ActorID:            "codex",
		MigrationTimestamp: "20260629_030000",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	assertDemoProductBackendContract(t, result)
}

func TestDemoProductFrontendAcceptanceContract(t *testing.T) {
	spec := loadDemoProductSpec(t)
	svc := NewService()
	result, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "demo-product-frontend-acceptance",
		ActorID:            "codex",
		MigrationTimestamp: "20260629_030000",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	assertDemoProductFrontendContract(t, result)
}

func loadDemoProductSpec(t *testing.T) *domaingenerator.GeneratorSpec {
	t.Helper()
	path := filepath.Join("..", "..", "..", "examples", "demo_product", "spec.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read demo product spec: %v", err)
	}
	var in domaingenerator.GeneratorSpecInput
	if err := json.Unmarshal(data, &in); err != nil {
		t.Fatalf("decode demo product spec: %v", err)
	}
	spec, err := domaingenerator.NewGeneratorSpec(in)
	if err != nil {
		t.Fatalf("NewGeneratorSpec() error = %v", err)
	}
	return spec
}

func assertGeneratedContains(t *testing.T, result *DryRunResult, path string, values ...string) {
	t.Helper()
	content := findPlan(t, result.Files, path).GeneratedContent
	for _, value := range values {
		if !strings.Contains(content, value) {
			t.Fatalf("%s does not contain %q\n%s", path, value, content)
		}
	}
}

func assertDemoProductBackendContract(t *testing.T, result *DryRunResult) {
	t.Helper()
	assertGeneratedContains(t, result, "internal/handler/http/v1/demo_product/handler.go",
		`mux.HandleFunc("GET /demo-products", h.list)`,
		`mux.HandleFunc("POST /demo-products", h.create)`,
		`mux.HandleFunc("PUT /demo-products/{id}", h.update)`,
		`mux.HandleFunc("DELETE /demo-products/{id}", h.delete)`,
		"func (h *Handler) create",
		"func (h *Handler) update",
		"func (h *Handler) delete",
	)
	assertGeneratedContains(t, result, "internal/service/demo_product/service_impl.go",
		`AuditActionDemoProductCreate = "demo_product.create"`,
		`AuditActionDemoProductUpdate = "demo_product.update"`,
		`AuditActionDemoProductDelete = "demo_product.delete"`,
		"s.repo.Create(ctx, *item)",
		"s.repo.Update(ctx, *item)",
		"s.repo.Delete(ctx, id)",
	)
	assertGeneratedContains(t, result, "internal/store/memory/demo_product_store.go",
		"func (s *DemoProductStore) Create",
		"func (s *DemoProductStore) Update",
		"func (s *DemoProductStore) Get",
		"func (s *DemoProductStore) List",
		"func (s *DemoProductStore) Delete",
	)
	assertGeneratedContains(t, result, "internal/store/sql/gormrepo/demo_product_store.go",
		"func toDemoProductModel",
		"item.ID.String()",
		"func fromDemoProductModel",
		"shared.ID(model.ID)",
		"CreatedAt: model.CreatedAt",
	)
	assertGeneratedContains(t, result, "internal/domain/demo_product/entity_test.go", "TestNewDemoProductValidatesAndBuildsEntity")
	assertGeneratedContains(t, result, "internal/store/memory/demo_product_store_test.go", "TestDemoProductStoreCRUD")
	assertGeneratedContains(t, result, "internal/store/sql/gormrepo/demo_product_store_test.go", "TestDemoProductModelRoundTrip")
	assertGeneratedContains(t, result, "internal/service/demo_product/service_impl_test.go", "TestServiceCRUD")
	assertGeneratedContains(t, result, "internal/bootstrap/generated_demo_product_catalog.go",
		"DemoProductGeneratedPermissions",
		`"demo_product.read"`,
		`"demo_product.create"`,
		`"demo_product.update"`,
		`"demo_product.delete"`,
		`"demo_product.manage"`,
		"RegisterDemoProductGeneratedCatalog",
		"domainmenu.NewNode",
		`Key: "demo_product"`,
		`Path: "/demo-products"`,
		`Component: "DemoProduct/index"`,
		`node.RequiredPermissions = []string{"demo_product.read"}`,
	)
}

func assertDemoProductFrontendContract(t *testing.T, result *DryRunResult) {
	t.Helper()
	assertGeneratedContains(t, result, "web/src/api/demo_product.ts",
		"export type DemoProduct",
		"export type DemoProductInput",
		"listDemoProduct",
		"createDemoProduct",
		"updateDemoProduct",
		"deleteDemoProduct",
		`const basePath = "/demo-products"`,
		"ApiResponse<{ items: DemoProduct[]; offset: number; limit: number }>",
		"Promise<DemoProductListPage>",
		"getDemoProduct",
		"hasMore: resp.data.items.length >= resp.data.limit",
		"ApiResponse<{ item: DemoProduct }>",
		"return resp.data.item",
	)
	assertGeneratedContains(t, result, "web/src/stores/demo_product.ts",
		"items: DemoProduct[]",
		"listStatus: LoadStatus",
		"mutationStatus: LoadStatus",
		"listError: string",
		"detailError: string",
		"mutationError: string",
		"async retry",
		"async loadOne",
		"async create",
		"async update",
		"async remove",
		`import { toErrorMessage } from "../utils/common"`,
	)
	assertGeneratedContains(t, result, "web/src/views/DemoProduct/index.vue",
		"<PageShell",
		"<FilterBar",
		"<DataTable",
		"<DetailDrawer",
		"<ConfirmAction",
		`v-permission="createPermission"`,
		`v-permission="updatePermission"`,
		`v-permission="deletePermission"`,
		`const createPermission = "demo_product.create"`,
		`const updatePermission = "demo_product.update"`,
		`const deletePermission = "demo_product.delete"`,
		`<el-input v-model="form.name" />`,
		`<el-input-number v-model="form.price"`,
		`<el-switch v-model="form.enabled" />`,
		"FormRules",
		"initialLoading",
		"initialError",
		"@media (max-width: 760px)",
	)
	assertGeneratedContains(t, result, "web/src/i18n/generated_demo_product.ts",
		"translateDemoProduct",
		`"generated.demo_product.title"`,
		`"generated.demo_product.field.name"`,
		`"zh-CN"`,
		`"en-US"`,
	)
	assertGeneratedContains(t, result, "web/src/router/generated_demo_product.ts",
		"demo_productRoutes",
		`path: "/demo-products"`,
		`name: "demo_product.list"`,
		`permissions: ["demo_product.read"]`,
	)
}
