package generator

import (
	"context"
	"encoding/json"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func TestDocumentPluginTargetUsesOnlyPublicContracts(t *testing.T) {
	spec := generatedPluginE2ESpec(t, "http://127.0.0.1:19090")
	result, err := NewService().DryRun(context.Background(), DryRunInput{
		Spec: spec, BatchID: "document-plugin-contract", ActorID: "generator-test", MigrationTimestamp: "20260722_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}

	backend := findPlan(t, result.Files, "examples/plugins/pharma-oa/backend/main.go").GeneratedContent
	if _, err := parser.ParseFile(token.NewFileSet(), "backend/main.go", backend, parser.AllErrors); err != nil {
		t.Fatalf("generated document backend is invalid: %v\n%s", err, backend)
	}
	for _, marker := range []string{
		`"github.com/tinboxw/skoll/pkg/pluginclient"`,
		`"github.com/tinboxw/skoll/pkg/pluginsdk"`,
		"host.Documents.Submit", "host.Documents.Act", "host.Documents.Export", "ensureWorkflowDefinition", "pluginclient.WithUserToken",
	} {
		if !strings.Contains(backend, marker) {
			t.Fatalf("generated backend missing %q", marker)
		}
	}
	if strings.Contains(backend, "github.com/tinboxw/skoll/internal/") {
		t.Fatal("generated plugin imports a host-internal package")
	}

	var schema pluginsdk.DocumentSchema
	schemaPlan := findPlan(t, result.Files, "examples/plugins/pharma-oa/document-schema.json")
	if err := json.Unmarshal([]byte(schemaPlan.GeneratedContent), &schema); err != nil {
		t.Fatalf("decode generated schema: %v", err)
	}
	if err := schema.Validate(); err != nil {
		t.Fatalf("generated schema is invalid: %v", err)
	}
	if schema.Key != "product_request" || schema.InitialState != "draft" || len(schema.Actions) != 3 {
		t.Fatalf("generated schema = %+v", schema)
	}

	goMod := findPlan(t, result.Files, "examples/plugins/pharma-oa/go.mod").GeneratedContent
	if !strings.Contains(goMod, "require github.com/tinboxw/skoll v0.0.0") || !strings.Contains(goMod, "replace github.com/tinboxw/skoll => ../../..") {
		t.Fatalf("generated go.mod does not bind the public SDK:\n%s", goMod)
	}
	packageJSON := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/package.json").GeneratedContent
	view := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/views/Product/index.vue").GeneratedContent
	hostAPI := findPlan(t, result.Files, "examples/plugins/pharma-oa/web/src/utils/api.ts").GeneratedContent
	for _, marker := range []string{"@skoll/document-ui", "DocumentList", "DocumentForm", "DocumentDetail", "approveDocument", "exportDocuments", "window.__SKOLL_HOST__", `pluginId !== "pharma-oa"`} {
		if !strings.Contains(packageJSON+view+hostAPI, marker) {
			t.Fatalf("generated frontend missing %q", marker)
		}
	}

	manifest := findPlan(t, result.Files, "examples/plugins/pharma-oa/plugin.yaml").GeneratedContent
	for _, route := range []string{"/submit", "/approve", "/export"} {
		if !strings.Contains(manifest, route) {
			t.Fatalf("generated manifest missing document route %q", route)
		}
	}
}
