package pharmaoa

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store"
	"gopkg.in/yaml.v3"
)

func TestManifestRoutesMatchPharmaOAHandlers(t *testing.T) {
	root := repositoryRoot(t)
	info, err := pluginruntime.NewFileLoader().Load(filepath.Join(root, "plugins", PluginID))
	if err != nil {
		t.Fatalf("load Pharma OA manifest: %v", err)
	}
	manifestRoutes, err := info.RouteExtensions()
	if err != nil {
		t.Fatalf("load manifest routes: %v", err)
	}

	declared := make(map[string]struct{}, len(manifestRoutes))
	for _, route := range manifestRoutes {
		declared[route.Method+" "+route.Path] = struct{}{}
	}
	registered := registeredHandlerRoutes(t, filepath.Join(root, "internal", "handler", "http", "v1", "pharmaoa"))
	if len(declared) != len(registered) {
		t.Fatalf("manifest routes=%d handler routes=%d", len(declared), len(registered))
	}
	for route := range registered {
		if _, ok := declared[route]; !ok {
			t.Errorf("handler route missing from manifest: %s", route)
		}
	}
	for route := range declared {
		if _, ok := registered[route]; !ok {
			t.Errorf("manifest route has no handler: %s", route)
		}
	}
	assertOpenAPIRoutes(t, filepath.Join(root, "internal", "handler", "http", "openapi.yaml"), manifestRoutes)
	assertOpenAPIRoutes(t, filepath.Join(root, "docs", "api", "openapi.yaml"), manifestRoutes)
}

func TestFrontendUsesPluginOwnedRoutesAndAPI(t *testing.T) {
	root := repositoryRoot(t)
	routerSource := readContractFile(t, filepath.Join(root, "web", "src", "router", "index.ts"))
	if strings.Contains(routerSource, `name: "pharma-oa-`) || strings.Contains(routerSource, "const PharmaEmployeePage") {
		t.Fatal("host router still owns Pharma OA pages")
	}
	integratedRoutes := readContractFile(t, filepath.Join(root, "web", "src", "plugins", "integrated-routes.ts"))
	if got := strings.Count(integratedRoutes, `name: "pharma-oa-`); got != 15 {
		t.Fatalf("integrated Pharma OA routes=%d want=15", got)
	}
	if !strings.Contains(integratedRoutes, "pharma_oa: createPharmaOARoutes") {
		t.Fatal("Pharma OA routes are not registered by plugin id")
	}
	apiSource := readContractFile(t, filepath.Join(root, "web", "src", "pharma-oa", "api.ts"))
	if strings.Contains(apiSource, "/v1/pharma-oa") {
		t.Fatal("frontend still calls the host Pharma OA namespace")
	}
	if !strings.Contains(apiSource, "/v1/plugins/pharma_oa/api/") {
		t.Fatal("frontend does not call the Pharma OA plugin namespace")
	}
}

func TestBackendServesOnlyPluginNamespace(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("build memory stores: %v", err)
	}
	notifications := notificationsvc.NewService(nil, nil)
	handler, err := NewBackend(Dependencies{
		Stores:       bundle,
		Audit:        auditsvc.NewService(bundle.Audit),
		RBAC:         rbacsvc.NewServiceWithOrganization(bundle.RBAC, bundle.Organization),
		Workflow:     workflowsvc.NewService(workflowsvc.NewMemoryRepository()),
		Notification: notifications,
	})
	if err != nil {
		t.Fatalf("build Pharma OA backend: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/plugins/pharma_oa/api/employees", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("plugin API status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	legacy := httptest.NewRecorder()
	handler.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/v1/pharma-oa/employees", nil))
	if legacy.Code != http.StatusNotFound {
		t.Fatalf("host namespace status=%d want=%d", legacy.Code, http.StatusNotFound)
	}
}

func registeredHandlerRoutes(t *testing.T, dir string) map[string]struct{} {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read handler directory: %v", err)
	}
	pattern := regexp.MustCompile(`mux\.HandleFunc\("([A-Z]+) ([^"]+)"`)
	routes := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_handler.go") {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			t.Fatalf("read %s: %v", entry.Name(), readErr)
		}
		for _, match := range pattern.FindAllStringSubmatch(string(content), -1) {
			routes[match[1]+" "+match[2]] = struct{}{}
		}
	}
	return routes
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", ".."))
}

func readContractFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func assertOpenAPIRoutes(t *testing.T, path string, routes []pluginruntime.RouteExtension) {
	t.Helper()
	var document struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal([]byte(readContractFile(t, path)), &document); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	for _, route := range routes {
		operations := document.Paths[route.Path]
		if operations == nil || operations[strings.ToLower(route.Method)] == nil {
			t.Errorf("OpenAPI %s is missing %s %s", path, route.Method, route.Path)
		}
	}
}
