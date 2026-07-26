package http

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEmbeddedOpenAPIReferencesResolve(t *testing.T) {
	var document map[string]any
	if err := yaml.Unmarshal([]byte(openAPIYAMLDocument), &document); err != nil {
		t.Fatalf("parse embedded OpenAPI: %v", err)
	}
	paths := openAPIMap(t, document["paths"], "paths")
	for path := range paths {
		if !strings.HasPrefix(path, "/") {
			t.Fatalf("invalid OpenAPI path key %q", path)
		}
		if strings.HasPrefix(path, "/v1/plugins/pharma_oa/api/") {
			t.Fatalf("embedded core OpenAPI owns Pharma OA business path %q", path)
		}
	}
	components := openAPIMap(t, document["components"], "components")
	securitySchemes := openAPIMap(t, components["securitySchemes"], "components.securitySchemes")
	bearerAuth := openAPIMap(t, securitySchemes["bearerAuth"], "components.securitySchemes.bearerAuth")
	if bearerAuth["type"] != "http" || bearerAuth["scheme"] != "bearer" {
		t.Fatalf("invalid bearerAuth security scheme: %+v", bearerAuth)
	}
	schemas := openAPIMap(t, components["schemas"], "components.schemas")
	for name := range schemas {
		if strings.HasPrefix(strings.TrimSpace(name), "+") {
			t.Fatalf("invalid OpenAPI schema key %q", name)
		}
		if strings.HasPrefix(name, "Pharma") {
			t.Fatalf("embedded core OpenAPI owns Pharma OA schema %q", name)
		}
	}
	refs := make([]string, 0)
	collectOpenAPIRefs(document, &refs)
	missing := make(map[string]struct{})
	for _, ref := range refs {
		const prefix = "#/components/schemas/"
		if !strings.HasPrefix(ref, prefix) {
			continue
		}
		name := strings.TrimPrefix(ref, prefix)
		if _, ok := schemas[name]; !ok {
			missing[ref] = struct{}{}
		}
	}
	if len(missing) > 0 {
		items := make([]string, 0, len(missing))
		for ref := range missing {
			items = append(items, ref)
		}
		sort.Strings(items)
		t.Fatalf("unresolved OpenAPI schema references: %s", strings.Join(items, ", "))
	}
}

func openAPIMap(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI %s is %T, want object", name, value)
	}
	return result
}

func collectOpenAPIRefs(value any, refs *[]string) {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			if key == "$ref" {
				if ref, ok := child.(string); ok {
					*refs = append(*refs, ref)
				} else {
					*refs = append(*refs, fmt.Sprint(child))
				}
			}
			collectOpenAPIRefs(child, refs)
		}
	case []any:
		for _, child := range node {
			collectOpenAPIRefs(child, refs)
		}
	}
}
