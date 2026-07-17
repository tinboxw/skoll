package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
	"gopkg.in/yaml.v3"
)

func TestRouterOpenAPIAggregatesEnabledPluginSecurityMetadata(t *testing.T) {
	const routePath = "/v1/plugins/demo/api/reports"
	manager := &fakePluginManager{
		items: []plugin.Info{{ID: "demo", State: plugin.StateEnabled}},
		snapshots: map[string]plugin.RegistrySnapshot{
			"demo": {Routes: []plugin.RouteExtension{{
				Method:      http.MethodGet,
				Path:        routePath,
				Summary:     "List demo reports",
				Permission:  "demo.report.read",
				AuditAction: "demo.report.read",
				Source:      "plugin.demo",
			}}},
		},
	}
	if rendered := renderPluginOpenAPIPaths(manager); !strings.Contains(rendered, routePath) {
		t.Fatalf("enabled plugin route was not rendered: %q", rendered)
	}
	router := NewRouter(Dependencies{PluginManager: manager})

	document := requestOpenAPIDocument(t, router)
	components := openAPIMap(t, document["components"], "components")
	securitySchemes := openAPIMap(t, components["securitySchemes"], "components.securitySchemes")
	bearerAuth := openAPIMap(t, securitySchemes["bearerAuth"], "components.securitySchemes.bearerAuth")
	if bearerAuth["type"] != "http" || bearerAuth["scheme"] != "bearer" {
		t.Fatalf("unexpected bearerAuth scheme: %+v", bearerAuth)
	}

	paths := openAPIMap(t, document["paths"], "paths")
	pathItem := openAPIMap(t, paths[routePath], routePath)
	operation := openAPIMap(t, pathItem["get"], routePath+".get")
	if operation["x-skoll-plugin-id"] != "demo" {
		t.Fatalf("plugin id metadata = %v", operation["x-skoll-plugin-id"])
	}
	if operation["x-skoll-permission"] != "demo.report.read" {
		t.Fatalf("permission metadata = %v", operation["x-skoll-permission"])
	}
	if operation["x-skoll-audit-action"] != "demo.report.read" {
		t.Fatalf("audit metadata = %v", operation["x-skoll-audit-action"])
	}
	security, ok := operation["security"].([]any)
	if !ok || len(security) != 1 {
		t.Fatalf("security metadata = %#v", operation["security"])
	}
	if _, ok := openAPIMap(t, security[0], routePath+".get.security")["bearerAuth"]; !ok {
		t.Fatalf("bearerAuth requirement missing: %#v", security[0])
	}

	if err := manager.Disable("demo"); err != nil {
		t.Fatalf("disable plugin: %v", err)
	}
	disabledDocument := requestOpenAPIDocument(t, router)
	disabledPaths := openAPIMap(t, disabledDocument["paths"], "paths")
	if _, exists := disabledPaths[routePath]; exists {
		t.Fatalf("disabled plugin route remains in aggregated OpenAPI: %s", routePath)
	}
}

func TestRouterOpenAPISkipsInvalidPluginPermissionContract(t *testing.T) {
	const routePath = "/v1/plugins/demo/api/invalid"
	manager := &fakePluginManager{
		items: []plugin.Info{{ID: "demo", State: plugin.StateEnabled}},
		snapshots: map[string]plugin.RegistrySnapshot{
			"demo": {Routes: []plugin.RouteExtension{{
				Method: http.MethodGet,
				Path:   routePath,
				Source: "plugin.demo",
			}}},
		},
	}

	document := requestOpenAPIDocument(t, NewRouter(Dependencies{PluginManager: manager}))
	paths := openAPIMap(t, document["paths"], "paths")
	if _, exists := paths[routePath]; exists {
		t.Fatalf("invalid plugin permission contract entered OpenAPI: %s", routePath)
	}
}

func requestOpenAPIDocument(t *testing.T, router http.Handler) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/skoll/docs/openapi.yaml", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("OpenAPI status=%d body=%s", resp.Code, resp.Body.String())
	}
	var document map[string]any
	if err := yaml.Unmarshal(resp.Body.Bytes(), &document); err != nil {
		t.Fatalf("parse aggregated OpenAPI: %v", err)
	}
	return document
}
