package plugin

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRoutePermissionRegistryNormalizesSortsAndResolves(t *testing.T) {
	registry, err := NewRoutePermissionRegistry([]RouteExtension{
		{Method: " post ", Path: "v1/plugins/demo/api/items/", Permission: " Demo.Items.Create ", AuditAction: " Demo.Items.Create ", Source: " Plugin.Demo "},
		{Method: "get", Path: "//v1/plugins/demo/api/items", Permission: "Demo.Items.Read", AuditAction: "Demo.Items.Read", Source: "Plugin.Demo"},
	})
	if err != nil {
		t.Fatalf("create route permission registry: %v", err)
	}

	want := []RoutePermissionDescriptor{
		{Method: "GET", Path: "/v1/plugins/demo/api/items", Permission: "demo.items.read", AuditAction: "demo.items.read", Source: "plugin.demo"},
		{Method: "POST", Path: "/v1/plugins/demo/api/items", Permission: "demo.items.create", AuditAction: "demo.items.create", Source: "plugin.demo"},
	}
	if got := registry.Descriptors(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected descriptors: %#v", got)
	}
	resolved, ok := registry.ResolveRoutePermission(" get ", "/v1/plugins/demo/api/items/")
	if !ok || !reflect.DeepEqual(resolved, want[0]) {
		t.Fatalf("unexpected resolved descriptor: %#v, ok=%v", resolved, ok)
	}
	if _, ok := registry.ResolveRoutePermission("DELETE", "/v1/plugins/demo/api/items"); ok {
		t.Fatal("unexpected descriptor for undeclared method")
	}

	snapshot := registry.Descriptors()
	snapshot[0].Permission = "changed"
	if got := registry.Descriptors()[0].Permission; got != "demo.items.read" {
		t.Fatalf("descriptor snapshot mutated registry: %s", got)
	}
}

func TestRoutePermissionRegistryRejectsMalformedDescriptorsWithLocalizedDiagnostics(t *testing.T) {
	tests := []struct {
		name  string
		route RouteExtension
		code  RoutePermissionErrorCode
	}{
		{name: "method", route: RouteExtension{Method: "OPTIONS", Path: "/v1/plugins/demo/api/items", Permission: "demo.items.read", Source: "plugin.demo"}, code: RoutePermissionMethodInvalid},
		{name: "path", route: RouteExtension{Method: "GET", Path: "/v1/plugins/demo/api/../items", Permission: "demo.items.read", Source: "plugin.demo"}, code: RoutePermissionPathInvalid},
		{name: "permission", route: RouteExtension{Method: "GET", Path: "/v1/plugins/demo/api/items", Permission: "Bad Permission", Source: "plugin.demo"}, code: RoutePermissionKeyInvalid},
		{name: "audit", route: RouteExtension{Method: "GET", Path: "/v1/plugins/demo/api/items", Permission: "demo.items.read", AuditAction: "Bad Audit", Source: "plugin.demo"}, code: RoutePermissionAuditInvalid},
		{name: "source", route: RouteExtension{Method: "GET", Path: "/v1/plugins/demo/api/items", Permission: "demo.items.read", Source: "Plugin Demo"}, code: RoutePermissionSourceInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRoutePermissionRegistry([]RouteExtension{tt.route})
			var diagnostic *RoutePermissionError
			if !errors.As(err, &diagnostic) {
				t.Fatalf("expected route permission diagnostic, got %v", err)
			}
			if diagnostic.Code != tt.code {
				t.Fatalf("code = %s, want %s", diagnostic.Code, tt.code)
			}
			if got := diagnostic.Error(); got == "" || strings.HasPrefix(got, "invalid ") {
				t.Fatalf("default diagnostic should be Chinese, got %q", got)
			}
			if got := diagnostic.Message("en-US"); !strings.Contains(got, "invalid") {
				t.Fatalf("English diagnostic = %q", got)
			}
		})
	}
}

func TestRoutePermissionRegistryRejectsNormalizedRouteConflict(t *testing.T) {
	_, err := NewRoutePermissionRegistry([]RouteExtension{
		{Method: "get", Path: "v1/plugins/demo/api/items/", Permission: "demo.items.read", Source: "plugin.alpha"},
		{Method: " GET ", Path: "//v1/plugins/demo/api/items", Permission: "demo.items.list", Source: "plugin.beta"},
	})
	var diagnostic *RoutePermissionError
	if !errors.As(err, &diagnostic) {
		t.Fatalf("expected conflict diagnostic, got %v", err)
	}
	if diagnostic.Code != RoutePermissionRouteConflict || diagnostic.Method != "GET" || diagnostic.Path != "/v1/plugins/demo/api/items" {
		t.Fatalf("unexpected conflict diagnostic: %#v", diagnostic)
	}
	if !strings.Contains(diagnostic.Error(), "插件路由冲突") || !strings.Contains(diagnostic.Message("en-US"), "conflicting plugin route") {
		t.Fatalf("unexpected localized conflict diagnostics: zh=%q en=%q", diagnostic.Error(), diagnostic.Message("en-US"))
	}
}

func TestNilRoutePermissionRegistryFailsClosed(t *testing.T) {
	var registry *RoutePermissionRegistry
	if _, ok := registry.ResolveRoutePermission("GET", "/v1/plugins/demo/api/items"); ok {
		t.Fatal("nil resolver must not resolve a route")
	}
	if got := registry.Descriptors(); len(got) != 0 {
		t.Fatalf("nil registry descriptors = %#v", got)
	}
}

func TestRoutePermissionRegistryResolvesParameterizedPath(t *testing.T) {
	registry, err := NewRoutePermissionRegistry([]RouteExtension{{
		Method: "POST", Path: "/v1/plugins/pharma_oa/api/drug-recalls/{id}/tasks/{taskId}/complete",
		Permission: "pharma_oa.drug_recall.complete", Source: "plugin.pharma_oa",
	}})
	if err != nil {
		t.Fatalf("build registry: %v", err)
	}
	descriptor, ok := registry.ResolveRoutePermission("POST", "/v1/plugins/pharma_oa/api/drug-recalls/recall-1/tasks/task-1/complete")
	if !ok || descriptor.Permission != "pharma_oa.drug_recall.complete" {
		t.Fatalf("unexpected parameterized route resolution: ok=%v descriptor=%+v", ok, descriptor)
	}
}

func TestRoutePermissionRegistryRejectsEquivalentParameterizedRoutes(t *testing.T) {
	_, err := NewRoutePermissionRegistry([]RouteExtension{
		{Method: "GET", Path: "/v1/plugins/reports/api/items/{id}", Permission: "reports.item.read", Source: "plugin.reports"},
		{Method: "GET", Path: "/v1/plugins/reports/api/items/{itemId}", Permission: "reports.item.read", Source: "plugin.reports"},
	})
	if err == nil {
		t.Fatal("expected equivalent parameterized routes to conflict")
	}
}
