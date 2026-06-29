package generator

import (
	"context"
	"strings"
	"testing"
	"time"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
	"github.com/tinboxw/skoll/internal/domain/shared"
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
	if plan := findPlan(t, result.Files, repoPlan.Path); !strings.Contains(plan.Diff, "--- current/") || !strings.Contains(plan.Diff, "-old repository content") || !strings.Contains(plan.Diff, "+backend.repository") {
		t.Fatalf("update diff is not readable: %q", plan.Diff)
	}
	if plan := findPlan(t, result.Files, storePlan.Path); !strings.Contains(plan.Diff, "-user edited store") || !strings.Contains(plan.Reason, "differs") {
		t.Fatalf("conflict diff is not readable: %+v", plan)
	}
	if result.Summary.Unchanged != 1 || result.Summary.UpdateClean != 1 || result.Summary.Conflict != 2 {
		t.Fatalf("summary = %+v", result.Summary)
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
