package generator

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/module/modgenerator"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func newGeneratorMux() *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(modgenerator.NewService()).Register(mux, nil, &testAPIRegistry{})
	return mux
}

func TestGeneratorRoutes_ModuleScaffoldPreview(t *testing.T) {
	mux := newGeneratorMux()

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/generator/modules", strings.NewReader(`{"module":"billing","template_version":"v2","form_schema":{"version":"v2","fields":[{"name":"name","type":"string","required":true},{"name":"status","type":"select"}]}}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected generator status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"module":"billing"`) {
		t.Fatalf("expected module name in generator result, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"internal/module/billing/service.go"`) {
		t.Fatalf("expected generated service artifact path, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"template_version":"v2"`) {
		t.Fatalf("expected template compatibility metadata, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"form_schema"`) {
		t.Fatalf("expected form schema in generator response, got %s", rr.Body.String())
	}
}
