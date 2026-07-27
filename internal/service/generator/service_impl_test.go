package generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"gopkg.in/yaml.v3"
)

func TestDryRunBuildsDeterministicFileList(t *testing.T) {
	svc := NewService()
	spec := mustSpec(t)

	result, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "batch-1",
		MigrationTimestamp: "20260629_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	if result.BatchID != "batch-1" || result.SpecID != "spec-product" {
		t.Fatalf("unexpected identity: %+v", result)
	}
	if result.Summary.Total != len(result.Files) || result.Summary.Create != len(result.Files) {
		t.Fatalf("summary = %+v files=%d", result.Summary, len(result.Files))
	}
	assertPlanPath(t, result.Files, "internal/domain/product/entity.go", FileStatusCreate)
	assertPlanPath(t, result.Files, "internal/repository/product/product_repo.go", FileStatusCreate)
	assertPlanPath(t, result.Files, "migrations/mysql/20260629_010203_create_products.sql", FileStatusCreate)
	assertPlanPath(t, result.Files, "web/src/views/Product/index.vue", FileStatusCreate)
	if plan := findPlan(t, result.Files, "internal/domain/product/entity.go"); !strings.Contains(plan.Diff, "--- /dev/null") || !strings.Contains(plan.Diff, "+++ generated/internal/domain/product/entity.go") {
		t.Fatalf("create diff is not readable: %q", plan.Diff)
	}
}

func TestDryRunClassifiesExistingFiles(t *testing.T) {
	svc := NewService()
	spec := mustSpec(t)
	base, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		MigrationTimestamp: "20260629_010203",
	})
	if err != nil {
		t.Fatalf("DryRun(base) error = %v", err)
	}
	domainPlan := findPlan(t, base.Files, "internal/domain/product/entity.go")
	repoPlan := findPlan(t, base.Files, "internal/repository/product/product_repo.go")
	storePlan := findPlan(t, base.Files, "internal/store/memory/product_store.go")
	handlerPlan := findPlan(t, base.Files, "internal/handler/http/v1/product/handler.go")

	result, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		MigrationTimestamp: "20260629_010203",
		ExistingFiles: []FileSnapshot{
			{Path: domainPlan.Path, CurrentHash: domainPlan.ContentHash},
			{Path: repoPlan.Path, CurrentHash: "old-generated-hash", PreviousGeneratedHash: "old-generated-hash", CurrentContent: "old repository content"},
			{Path: storePlan.Path, CurrentHash: "user-edited-hash", PreviousGeneratedHash: "old-generated-hash", CurrentContent: "user edited store"},
			{Path: handlerPlan.Path, CurrentHash: "unknown-existing-hash"},
		},
	})
	if err != nil {
		t.Fatalf("DryRun(existing) error = %v", err)
	}
	assertPlanPath(t, result.Files, domainPlan.Path, FileStatusUnchanged)
	assertPlanPath(t, result.Files, repoPlan.Path, FileStatusUpdateClean)
	assertPlanPath(t, result.Files, storePlan.Path, FileStatusConflict)
	assertPlanPath(t, result.Files, handlerPlan.Path, FileStatusConflict)
	if plan := findPlan(t, result.Files, repoPlan.Path); !strings.Contains(plan.Diff, "--- current/") || !strings.Contains(plan.Diff, "-old repository content") || !strings.Contains(plan.Diff, "+type ProductRepository interface") {
		t.Fatalf("update diff is not readable: %q", plan.Diff)
	}
	if plan := findPlan(t, result.Files, storePlan.Path); !strings.Contains(plan.Diff, "-user edited store") || !strings.Contains(plan.Reason, "differs") {
		t.Fatalf("conflict diff is not readable: %+v", plan)
	}
	if result.Summary.Unchanged != 1 || result.Summary.UpdateClean != 1 || result.Summary.Conflict != 2 {
		t.Fatalf("summary = %+v", result.Summary)
	}
}

func TestDryRunRendersBackendTemplatesAsValidGo(t *testing.T) {
	svc := NewService()
	spec := mustSpec(t)
	result, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		MigrationTimestamp: "20260629_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	goPaths := []string{
		"internal/domain/product/doc.go",
		"internal/domain/product/entity.go",
		"internal/domain/product/entity_test.go",
		"internal/repository/product/product_repo.go",
		"internal/store/memory/product_store.go",
		"internal/store/memory/product_store_test.go",
		"internal/store/sql/gormrepo/product_model.go",
		"internal/store/sql/gormrepo/product_store.go",
		"internal/store/sql/gormrepo/product_store_test.go",
		"internal/service/product/service.go",
		"internal/service/product/service_impl.go",
		"internal/service/product/service_impl_test.go",
		"internal/handler/http/v1/product/handler.go",
		"internal/handler/http/v1/product/handler_test.go",
		"internal/handler/http/generated_product_routes.go",
		"internal/bootstrap/generated_product_catalog.go",
	}
	for _, path := range goPaths {
		plan := findPlan(t, result.Files, path)
		if _, err := parser.ParseFile(token.NewFileSet(), path, plan.GeneratedContent, parser.AllErrors); err != nil {
			t.Fatalf("generated Go for %s is invalid: %v\n%s", path, err, plan.GeneratedContent)
		}
	}
	mysql := findPlan(t, result.Files, "migrations/mysql/20260629_010203_create_products.sql")
	if !strings.Contains(mysql.GeneratedContent, "CREATE TABLE IF NOT EXISTS products") ||
		!strings.Contains(mysql.GeneratedContent, "PRIMARY KEY (id)") ||
		!strings.Contains(mysql.GeneratedContent, "created_at TIMESTAMP NOT NULL") ||
		!strings.Contains(mysql.GeneratedContent, "CREATE INDEX idx_products_name") {
		t.Fatalf("mysql migration content = %q", mysql.GeneratedContent)
	}
	postgres := findPlan(t, result.Files, "migrations/postgres/20260629_010203_create_products.sql")
	if !strings.Contains(postgres.GeneratedContent, "CREATE TABLE IF NOT EXISTS products") || !strings.Contains(postgres.GeneratedContent, "name VARCHAR(255) NOT NULL") {
		t.Fatalf("postgres migration content = %q", postgres.GeneratedContent)
	}
	openapi := findPlan(t, result.Files, "docs/api/openapi.yaml")
	assertGeneratedOpenAPI(t, openapi.GeneratedContent, "Product")
	seed := findPlan(t, result.Files, "internal/bootstrap/generated_product_catalog.go")
	if !strings.Contains(seed.GeneratedContent, "ProductGeneratedPermissions") || !strings.Contains(seed.GeneratedContent, "product.manage") {
		t.Fatalf("permission seed content = %q", seed.GeneratedContent)
	}
	api := findPlan(t, result.Files, "web/src/api/product.ts")
	if !strings.Contains(api.GeneratedContent, "apiGet") ||
		!strings.Contains(api.GeneratedContent, "export type Product") ||
		!strings.Contains(api.GeneratedContent, "createProduct") ||
		!strings.Contains(api.GeneratedContent, "getProduct") ||
		!strings.Contains(api.GeneratedContent, "Promise<ProductListPage>") ||
		strings.Count(api.GeneratedContent, "return resp.data.item;") != 3 {
		t.Fatalf("frontend api content = %q", api.GeneratedContent)
	}
	store := findPlan(t, result.Files, "web/src/stores/product.ts")
	if !strings.Contains(store.GeneratedContent, "defineStore") || !strings.Contains(store.GeneratedContent, "useProductStore") || !strings.Contains(store.GeneratedContent, "async retry") || !strings.Contains(store.GeneratedContent, "detailStatus") {
		t.Fatalf("frontend store content = %q", store.GeneratedContent)
	}
	locale := findPlan(t, result.Files, "web/src/i18n/generated_product.ts")
	if !strings.Contains(locale.GeneratedContent, "translateProduct") || !strings.Contains(locale.GeneratedContent, `"zh-CN"`) || !strings.Contains(locale.GeneratedContent, `"en-US"`) {
		t.Fatalf("frontend locale content = %q", locale.GeneratedContent)
	}
	route := findPlan(t, result.Files, "web/src/router/generated_product.ts")
	if !strings.Contains(route.GeneratedContent, "productRoutes") || !strings.Contains(route.GeneratedContent, `permissions: ["product.read"]`) {
		t.Fatalf("frontend route content = %q", route.GeneratedContent)
	}
	view := findPlan(t, result.Files, "web/src/views/Product/index.vue")
	if !strings.Contains(view.GeneratedContent, "<DataTable") ||
		!strings.Contains(view.GeneratedContent, "<DetailDrawer") ||
		!strings.Contains(view.GeneratedContent, "<ConfirmAction") ||
		!strings.Contains(view.GeneratedContent, "v-permission=\"createPermission\"") ||
		!strings.Contains(view.GeneratedContent, "initialError") ||
		!strings.Contains(view.GeneratedContent, "useProductStore") {
		t.Fatalf("frontend view content = %q", view.GeneratedContent)
	}
}

func TestGeneratedBackendCompilesAndPassesGeneratedTests(t *testing.T) {
	spec := loadDemoProductSpec(t)
	result, err := NewService().DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "pr3-backend-compile",
		ActorID:            "generator-test",
		MigrationTimestamp: "20260722_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}

	root := t.TempDir()
	moduleFile := `module github.com/tinboxw/skoll

go 1.24

require gorm.io/gorm v1.31.1
`
	writeTestFile(t, root, "go.mod", moduleFile)
	sharedSource, err := os.ReadFile(filepath.Join("..", "..", "domain", "shared", "types.go"))
	if err != nil {
		t.Fatalf("read shared domain source: %v", err)
	}
	writeTestFile(t, root, "internal/domain/shared/types.go", string(sharedSource))
	responseSource, err := os.ReadFile(filepath.Join("..", "..", "handler", "http", "v1", "response.go"))
	if err != nil {
		t.Fatalf("read HTTP response source: %v", err)
	}
	writeTestFile(t, root, "internal/handler/http/v1/response.go", string(responseSource))

	backendPaths := []string{
		"internal/domain/demo_product/doc.go",
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
		"internal/handler/http/v1/demo_product/handler_test.go",
		"internal/handler/http/generated_demo_product_routes.go",
	}
	for _, path := range backendPaths {
		plan := findPlan(t, result.Files, path)
		writeTestFile(t, root, path, plan.GeneratedContent)
	}

	cmd := exec.Command("go", "test", "./...", "-count=1", "-mod=mod")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated go test ./... failed: %v\n%s", err, output)
	}
}

func assertGeneratedOpenAPI(t *testing.T, content, domainName string) {
	t.Helper()
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		t.Fatalf("generated OpenAPI is invalid YAML: %v\n%s", err, content)
	}
	paths, ok := document["paths"].(map[string]any)
	if !ok || len(paths) != 2 {
		t.Fatalf("OpenAPI paths = %#v", document["paths"])
	}
	for _, operation := range []string{"list", "get", "create", "update", "delete"} {
		operationID := "operationId: " + operation + domainName
		if count := strings.Count(content, operationID); count != 1 {
			t.Fatalf("%s count = %d\n%s", operationID, count, content)
		}
	}
	for _, marker := range []string{
		"x-permission: product.create",
		"x-audit-action: product.create",
		"x-permission: product.update",
		"x-audit-action: product.update",
		"x-permission: product.delete",
		"x-audit-action: product.delete",
	} {
		if !strings.Contains(content, marker) {
			t.Fatalf("OpenAPI missing %q\n%s", marker, content)
		}
	}
}

func TestDryRunRendersBusinessPluginTemplates(t *testing.T) {
	svc := NewService()
	spec := mustPluginSpec(t)
	result, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "batch-plugin",
		ActorID:            "actor-1",
		MigrationTimestamp: "20260704_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/plugin.yaml", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/go.mod", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/backend/main.go", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/migrations/001_create_pharma_oa_products.up.sql", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/migrations/001_create_pharma_oa_products.down.sql", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/web/src/api/product.ts", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/web/src/stores/product.ts", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/web/src/i18n/generated_product.ts", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/web/src/skoll-host.ts", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/web/src/views/Product/index.vue", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/web/package.json", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/web/src/main.ts", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/plugin.ps1", FileStatusCreate)
	assertPlanPath(t, result.Files, "examples/plugins/pharma-oa/plugin.sh", FileStatusCreate)
	testPlan := findPlan(t, result.Files, "examples/plugins/pharma-oa/plugin_acceptance_test.go")
	if _, err := parser.ParseFile(token.NewFileSet(), testPlan.Path, testPlan.GeneratedContent, parser.AllErrors); err != nil {
		t.Fatalf("generated plugin acceptance test is invalid: %v\n%s", err, testPlan.GeneratedContent)
	}

	manifest := findPlan(t, result.Files, "examples/plugins/pharma-oa/plugin.yaml").GeneratedContent
	for _, value := range []string{
		"id: pharma-oa",
		"ui_menu:",
		"data:",
		"namespace: pharma_oa",
		"name: pharma_oa_products",
		"uninstall_policy: retain",
		"rollback_policy: automatic",
		"path: /v1/plugins/pharma-oa/api/products",
		"permission: pharma_oa.product.read",
		"audit_action: pharma_oa.product.create",
		"name: approval-completed",
	} {
		if !strings.Contains(manifest, value) {
			t.Fatalf("plugin manifest missing %q\n%s", value, manifest)
		}
	}
	api := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/api/product.ts").GeneratedContent
	if !strings.Contains(api, `const basePath = "/v1/plugins/pharma-oa/api/products"`) || !strings.Contains(api, "createProduct") {
		t.Fatalf("plugin frontend api content = %q", api)
	}
	locale := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/i18n/generated_product.ts").GeneratedContent
	if !strings.Contains(locale, "translateProduct") || !strings.Contains(locale, `"generated.product.field.name"`) {
		t.Fatalf("plugin frontend locale content = %q", locale)
	}
	host := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/skoll-host.ts").GeneratedContent
	if !strings.Contains(host, `getPluginHost`) || !strings.Contains(host, `pluginId: "pharma-oa"`) || !strings.Contains(host, `"skoll:theme"`) {
		t.Fatalf("plugin frontend host content = %q", host)
	}
	view := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/views/Product/index.vue").GeneratedContent
	if !strings.Contains(view, "plugin-generated-page") || !strings.Contains(view, `from "@skoll/business-ui/core"`) || strings.Contains(view, "v-permission") {
		t.Fatalf("plugin frontend view content = %q", view)
	}
	powerShell := findPlan(t, result.Files, "examples/plugins/pharma-oa/plugin.ps1").GeneratedContent
	for _, marker := range []string{"verify-package", "install-package", "Invoke-SkollPlugin", "pharma-oa-1.0.0.zip"} {
		if !strings.Contains(filepath.ToSlash(powerShell), marker) {
			t.Fatalf("plugin PowerShell command missing %q\n%s", marker, powerShell)
		}
	}
	shell := findPlan(t, result.Files, "examples/plugins/pharma-oa/plugin.sh").GeneratedContent
	if !strings.Contains(shell, `dev) build_plugin; (cd "$repo_root" && go run ./cmd/skoll-plugin dev`) {
		t.Fatalf("plugin shell command does not use package-backed dev flow\n%s", shell)
	}
	backend := findPlan(t, result.Files, "examples/plugins/pharma-oa/backend/main.go").GeneratedContent
	for _, marker := range []string{`apiPath`, `"/v1/plugins/pharma-oa/api/products"`, `GET /health`, "http.ListenAndServe"} {
		if !strings.Contains(backend, marker) {
			t.Fatalf("plugin backend missing %q\n%s", marker, backend)
		}
	}
}

func TestGeneratorGoldenSnapshotAndIdempotency(t *testing.T) {
	svc := NewService()
	spec := mustSpec(t)
	first, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "batch-golden",
		ActorID:            "actor-1",
		MigrationTimestamp: "20260629_010203",
	})
	if err != nil {
		t.Fatalf("DryRun(first) error = %v", err)
	}
	second, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "batch-golden",
		ActorID:            "actor-1",
		MigrationTimestamp: "20260629_010203",
	})
	if err != nil {
		t.Fatalf("DryRun(second) error = %v", err)
	}
	firstSnapshot := goldenSnapshot(first)
	secondSnapshot := goldenSnapshot(second)
	if firstSnapshot != secondSnapshot {
		t.Fatalf("dry-run is not idempotent\nfirst=%s\nsecond=%s", firstSnapshot, secondSnapshot)
	}
	const expectedSnapshotHash = "419c206932d2f6928e89752962d3e7f87186ec634006c20e3c96008f10575f4a"
	if got := sha256Hex(firstSnapshot); got != expectedSnapshotHash {
		t.Fatalf("golden snapshot hash = %s, want %s\n%s", got, expectedSnapshotHash, firstSnapshot)
	}
}

func TestDryRunRejectsMissingSpecAndBadSnapshots(t *testing.T) {
	svc := NewService()
	if _, err := svc.DryRun(context.Background(), DryRunInput{}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "spec") {
		t.Fatalf("expected missing spec error, got %v", err)
	}
	if _, err := svc.DryRun(context.Background(), DryRunInput{Spec: mustSpec(t), ExistingFiles: []FileSnapshot{{Path: " "}}}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "path") {
		t.Fatalf("expected snapshot path error, got %v", err)
	}
}

func TestRecordAndGetGenerationHistory(t *testing.T) {
	svc := NewService()
	spec := mustSpec(t)
	dryRun, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "batch-history",
		ActorID:            "actor-1",
		MigrationTimestamp: "20260629_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	createdAt := time.Date(2026, 6, 29, 2, 0, 0, 0, time.UTC)
	history, err := svc.RecordHistory(context.Background(), RecordHistoryInput{
		DryRun:    dryRun,
		Spec:      spec,
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("RecordHistory() error = %v", err)
	}
	if history.BatchID != "batch-history" || history.ActorID != "actor-1" || history.SpecID != "spec-product" || history.SpecHash == "" || history.SpecSnapshot == "" {
		t.Fatalf("history identity = %+v", history)
	}
	if !history.CreatedAt.Equal(createdAt) {
		t.Fatalf("createdAt = %s", history.CreatedAt)
	}
	if len(history.Files) != len(dryRun.Files) {
		t.Fatalf("history files = %d, dry-run files = %d", len(history.Files), len(dryRun.Files))
	}
	for _, file := range history.Files {
		if file.Path == "" || file.TemplateID == "" || file.Hash == "" {
			t.Fatalf("incomplete file record: %+v", file)
		}
	}
	got, err := svc.GetHistory(context.Background(), "batch-history")
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	got.Files[0].Path = "mutated"
	again, err := svc.GetHistory(context.Background(), "batch-history")
	if err != nil {
		t.Fatalf("GetHistory(second) error = %v", err)
	}
	if again.Files[0].Path == "mutated" {
		t.Fatalf("history store returned mutable file slice")
	}
}

func TestRecordHistoryValidation(t *testing.T) {
	svc := NewService()
	spec := mustSpec(t)
	if _, err := svc.RecordHistory(context.Background(), RecordHistoryInput{Spec: spec}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "dry-run") {
		t.Fatalf("expected dry-run required error, got %v", err)
	}
	dryRun, err := svc.DryRun(context.Background(), DryRunInput{Spec: spec, BatchID: "batch-no-actor"})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	if _, err := svc.RecordHistory(context.Background(), RecordHistoryInput{DryRun: dryRun, Spec: spec}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "actor") {
		t.Fatalf("expected actor required error, got %v", err)
	}
	if _, err := svc.GetHistory(context.Background(), "missing"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestPlanRollback(t *testing.T) {
	svc := NewService()
	spec := mustSpec(t)
	base, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "batch-rollback",
		ActorID:            "actor-1",
		MigrationTimestamp: "20260629_010203",
	})
	if err != nil {
		t.Fatalf("DryRun(base) error = %v", err)
	}
	repoPlan := findPlan(t, base.Files, "internal/repository/product/product_repo.go")
	storePlan := findPlan(t, base.Files, "internal/store/memory/product_store.go")
	dryRun, err := svc.DryRun(context.Background(), DryRunInput{
		Spec:               spec,
		BatchID:            "batch-rollback",
		ActorID:            "actor-1",
		MigrationTimestamp: "20260629_010203",
		ExistingFiles: []FileSnapshot{
			{Path: repoPlan.Path, CurrentHash: "old-repo-hash", PreviousGeneratedHash: "old-repo-hash", CurrentContent: "old repository content"},
			{Path: storePlan.Path, CurrentHash: "user-edited-hash", PreviousGeneratedHash: "old-store-hash", CurrentContent: "user edited store"},
		},
	})
	if err != nil {
		t.Fatalf("DryRun(update) error = %v", err)
	}
	if _, err := svc.RecordHistory(context.Background(), RecordHistoryInput{DryRun: dryRun, Spec: spec}); err != nil {
		t.Fatalf("RecordHistory() error = %v", err)
	}
	createFile := findPlan(t, dryRun.Files, "internal/domain/product/entity.go")
	updateFile := findPlan(t, dryRun.Files, repoPlan.Path)
	conflictFile := findPlan(t, dryRun.Files, storePlan.Path)

	plan, err := svc.PlanRollback(context.Background(), RollbackInput{
		BatchID: "batch-rollback",
		CurrentFiles: []FileSnapshot{
			{Path: createFile.Path, CurrentHash: createFile.ContentHash},
			{Path: updateFile.Path, CurrentHash: updateFile.ContentHash},
			{Path: conflictFile.Path, CurrentHash: conflictFile.ContentHash},
		},
	})
	if err != nil {
		t.Fatalf("PlanRollback() error = %v", err)
	}
	assertRollbackAction(t, plan, createFile.Path, RollbackActionDelete)
	restore := assertRollbackAction(t, plan, updateFile.Path, RollbackActionRestore)
	if restore.RestoredHash != "old-repo-hash" || restore.RestoredContent != "old repository content" {
		t.Fatalf("restore plan = %+v", restore)
	}
	manual := assertRollbackAction(t, plan, conflictFile.Path, RollbackActionManual)
	if !strings.Contains(manual.Reason, "not written") {
		t.Fatalf("manual plan = %+v", manual)
	}

	modified, err := svc.PlanRollback(context.Background(), RollbackInput{
		BatchID: "batch-rollback",
		CurrentFiles: []FileSnapshot{
			{Path: createFile.Path, CurrentHash: "user-modified-after-generation"},
		},
	})
	if err != nil {
		t.Fatalf("PlanRollback(modified) error = %v", err)
	}
	conflict := assertRollbackAction(t, modified, createFile.Path, RollbackActionConflict)
	if !strings.Contains(conflict.Reason, "differs") {
		t.Fatalf("conflict plan = %+v", conflict)
	}
}

func assertRollbackAction(t *testing.T, plan *RollbackPlan, path string, action RollbackAction) RollbackFilePlan {
	t.Helper()
	for _, file := range plan.Files {
		if file.Path == path {
			if file.Action != action {
				t.Fatalf("rollback action for %s = %s, want %s: %+v", path, file.Action, action, file)
			}
			return file
		}
	}
	t.Fatalf("missing rollback file %s in %+v", path, plan.Files)
	return RollbackFilePlan{}
}

func assertPlanPath(t *testing.T, files []FilePlan, path string, status FileStatus) {
	t.Helper()
	plan := findPlan(t, files, path)
	if plan.Status != status {
		t.Fatalf("path %s status = %s, want %s: %+v", path, plan.Status, status, plan)
	}
	if plan.ContentHash == "" {
		t.Fatalf("path %s missing content hash", path)
	}
}

func findPlan(t *testing.T, files []FilePlan, path string) FilePlan {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("missing plan path %s in %+v", path, files)
	return FilePlan{}
}

func goldenSnapshot(result *DryRunResult) string {
	var b strings.Builder
	for _, file := range result.Files {
		b.WriteString(file.Path)
		b.WriteString("|")
		b.WriteString(file.TemplateID)
		b.WriteString("|")
		b.WriteString(string(file.Status))
		b.WriteString("|")
		b.WriteString(file.ContentHash)
		b.WriteString("\n")
	}
	return b.String()
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func writeTestFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create generated directory for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write generated file %s: %v", relativePath, err)
	}
}

func mustSpec(t *testing.T) *domaingenerator.GeneratorSpec {
	t.Helper()
	now := time.Date(2026, 6, 29, 1, 2, 3, 0, time.UTC)
	spec, err := domaingenerator.NewGeneratorSpec(domaingenerator.GeneratorSpecInput{
		ID: shared.ID("spec-product"),
		Module: domaingenerator.ModuleSpec{
			Name:        "product",
			Package:     "product",
			DisplayName: "Product",
		},
		Table: domaingenerator.TableSpec{
			Name:           "products",
			DomainName:     "Product",
			CollectionName: "Products",
		},
		Fields: []domaingenerator.FieldSpec{
			{Name: "id", ColumnName: "id", Label: "ID", Type: domaingenerator.FieldTypeID, GoType: "string", TypeScript: "string", PrimaryKey: true, Required: true, ListVisible: true, FormVisible: false},
			{Name: "name", ColumnName: "name", Label: "Name", Type: domaingenerator.FieldTypeString, GoType: "string", TypeScript: "string", Required: true, Filterable: true, Sortable: true, ListVisible: true, FormVisible: true},
		},
		Indexes: []domaingenerator.IndexSpec{
			{Name: "idx_products_name", Fields: []string{"name"}},
		},
		Permissions: domaingenerator.PermissionSpec{
			Resource:  "product",
			ReadKey:   "product.read",
			CreateKey: "product.create",
			UpdateKey: "product.update",
			DeleteKey: "product.delete",
			ManageKey: "product.manage",
		},
		Menu: domaingenerator.MenuSpec{
			Key:                 "product",
			Path:                "/products",
			Component:           "Product/index",
			RequiredPermissions: []string{"product.read"},
		},
		Page: domaingenerator.PageSpec{
			Title:     "Products",
			RouteName: "product.list",
			List:      domaingenerator.PageListSpec{Columns: []string{"id", "name"}, Filters: []string{"name"}, Actions: []string{"create", "update", "delete"}},
			Form:      domaingenerator.PageFormSpec{Fields: []string{"name"}, Mode: "drawer"},
		},
		Audit: domaingenerator.AuditSpec{
			Resource: "product",
			Actions:  []string{"create", "update", "delete"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("NewGeneratorSpec() error = %v", err)
	}
	return spec
}

func mustPluginSpec(t *testing.T) *domaingenerator.GeneratorSpec {
	t.Helper()
	in := validServiceGeneratorSpecInput()
	in.Plugin = domaingenerator.PluginSpec{
		Enabled:       true,
		ID:            "pharma-oa",
		Name:          "Pharma OA",
		Description:   "Generated pharma OA business plugin",
		DataNamespace: "pharma-oa",
		EventSubscriptions: []domaingenerator.PluginEventSubscriptionSpec{
			{Name: "approval-completed", Handler: "onApprovalCompleted"},
		},
	}
	in.Table.Name = "pharma_oa_products"
	in.Indexes[0].Name = "idx_pharma_oa_products_name"
	namespaceServicePluginInput(&in, "pharma_oa")
	spec, err := domaingenerator.NewGeneratorSpec(in)
	if err != nil {
		t.Fatalf("NewGeneratorSpec() error = %v", err)
	}
	return spec
}

func namespaceServicePluginInput(in *domaingenerator.GeneratorSpecInput, namespace string) {
	in.Permissions = domaingenerator.PermissionSpec{
		Resource:  namespace + ".product",
		ReadKey:   namespace + ".product.read",
		CreateKey: namespace + ".product.create",
		UpdateKey: namespace + ".product.update",
		DeleteKey: namespace + ".product.delete",
		ManageKey: namespace + ".product.manage",
	}
	in.Menu.Key = namespace + ".product"
	in.Menu.RequiredPermissions = []string{in.Permissions.ReadKey}
}

func validServiceGeneratorSpecInput() domaingenerator.GeneratorSpecInput {
	now := time.Date(2026, 6, 29, 1, 2, 3, 0, time.UTC)
	return domaingenerator.GeneratorSpecInput{
		ID: shared.ID("spec-product"),
		Module: domaingenerator.ModuleSpec{
			Name:        "product",
			Package:     "product",
			DisplayName: "Product",
		},
		Table: domaingenerator.TableSpec{
			Name:           "products",
			DomainName:     "Product",
			CollectionName: "Products",
			Comment:        "Product table",
		},
		Fields: []domaingenerator.FieldSpec{
			{Name: "id", ColumnName: "id", Label: "ID", Type: domaingenerator.FieldTypeID, GoType: "string", TypeScript: "string", PrimaryKey: true, Required: true, ListVisible: true, FormVisible: false},
			{Name: "name", ColumnName: "name", Label: "Name", Type: domaingenerator.FieldTypeString, GoType: "string", TypeScript: "string", Required: true, Filterable: true, Sortable: true, ListVisible: true, FormVisible: true},
		},
		Indexes: []domaingenerator.IndexSpec{
			{Name: "idx_products_name", Fields: []string{"name"}},
		},
		Permissions: domaingenerator.PermissionSpec{
			Resource:  "product",
			ReadKey:   "product.read",
			CreateKey: "product.create",
			UpdateKey: "product.update",
			DeleteKey: "product.delete",
			ManageKey: "product.manage",
		},
		Menu: domaingenerator.MenuSpec{
			Key:                 "product",
			Path:                "/products",
			Component:           "Product/index",
			RequiredPermissions: []string{"product.read"},
		},
		Page: domaingenerator.PageSpec{
			Title:     "Products",
			RouteName: "product.list",
			List:      domaingenerator.PageListSpec{Columns: []string{"id", "name"}, Filters: []string{"name"}, Actions: []string{"create", "update", "delete"}},
			Form:      domaingenerator.PageFormSpec{Fields: []string{"name"}, Mode: "drawer"},
		},
		Audit: domaingenerator.AuditSpec{
			Resource: "product",
			Actions:  []string{"create", "update", "delete"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
