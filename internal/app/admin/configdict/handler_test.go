package configdict

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/config"
	"github.com/tinboxw/skoll/internal/module/dictionary"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

type testAuditSpy struct {
	count int
}

func (s *testAuditSpy) Append(actor, action, target string) audit.Record {
	s.count++
	return audit.Record{}
}

func TestHandler_RegisterAndMainFlow(t *testing.T) {
	mux := http.NewServeMux()
	registry := &testAPIRegistry{}
	auditSpy := &testAuditSpy{}
	h := NewHandler(config.NewService(), dictionary.NewService(), auditSpy)
	h.Register(mux, nil, registry)

	if len(registry.entries) != 10 {
		t.Fatalf("expected 10 registered routes, got %d", len(registry.entries))
	}

	configCreateReq := httptest.NewRequest(http.MethodPost, "/admin/v1/configs", strings.NewReader(`{"key":"system.theme","value":"aurora","description":"ui"}`))
	configCreateReq.Header.Set("Content-Type", "application/json")
	configCreateRR := httptest.NewRecorder()
	mux.ServeHTTP(configCreateRR, configCreateReq)
	if configCreateRR.Code != http.StatusCreated {
		t.Fatalf("expected config create status 201, got %d body=%s", configCreateRR.Code, configCreateRR.Body.String())
	}

	configQueryReq := httptest.NewRequest(http.MethodGet, "/admin/v1/configs/query?key_prefix=system.&page=1&size=10", nil)
	configQueryRR := httptest.NewRecorder()
	mux.ServeHTTP(configQueryRR, configQueryReq)
	if configQueryRR.Code != http.StatusOK {
		t.Fatalf("expected config query status 200, got %d body=%s", configQueryRR.Code, configQueryRR.Body.String())
	}
	if !strings.Contains(configQueryRR.Body.String(), `"total":1`) {
		t.Fatalf("expected config query total=1, got %s", configQueryRR.Body.String())
	}

	dictCreateReq := httptest.NewRequest(http.MethodPost, "/admin/v1/dictionaries", strings.NewReader(`{"type":"status","label":"Enabled","value":"1","sort":10,"enabled":true}`))
	dictCreateReq.Header.Set("Content-Type", "application/json")
	dictCreateRR := httptest.NewRecorder()
	mux.ServeHTTP(dictCreateRR, dictCreateReq)
	if dictCreateRR.Code != http.StatusCreated {
		t.Fatalf("expected dictionary create status 201, got %d body=%s", dictCreateRR.Code, dictCreateRR.Body.String())
	}

	dictQueryReq := httptest.NewRequest(http.MethodGet, "/admin/v1/dictionaries/query?type=status&q=enabled&enabled=true", nil)
	dictQueryRR := httptest.NewRecorder()
	mux.ServeHTTP(dictQueryRR, dictQueryReq)
	if dictQueryRR.Code != http.StatusOK {
		t.Fatalf("expected dictionary query status 200, got %d body=%s", dictQueryRR.Code, dictQueryRR.Body.String())
	}
	if !strings.Contains(dictQueryRR.Body.String(), `"Label":"Enabled"`) {
		t.Fatalf("expected dictionary query contains label, got %s", dictQueryRR.Body.String())
	}
}

func TestHandler_BulkEndpointsAuditAndValidation(t *testing.T) {
	mux := http.NewServeMux()
	auditSpy := &testAuditSpy{}
	h := NewHandler(config.NewService(), dictionary.NewService(), auditSpy)
	h.Register(mux, nil, &testAPIRegistry{})

	bulkConfigReq := httptest.NewRequest(http.MethodPost, "/admin/v1/configs/bulk", strings.NewReader(`{"items":[{"key":"feature.a","value":"on","description":"a"}]}`))
	bulkConfigReq.Header.Set("Content-Type", "application/json")
	bulkConfigRR := httptest.NewRecorder()
	mux.ServeHTTP(bulkConfigRR, bulkConfigReq)
	if bulkConfigRR.Code != http.StatusCreated {
		t.Fatalf("expected bulk config status 201, got %d body=%s", bulkConfigRR.Code, bulkConfigRR.Body.String())
	}

	bulkDictReq := httptest.NewRequest(http.MethodPost, "/admin/v1/dictionaries/bulk", strings.NewReader(`{"items":[{"type":"status","label":"Archived","value":"2","sort":30}]}`))
	bulkDictReq.Header.Set("Content-Type", "application/json")
	bulkDictRR := httptest.NewRecorder()
	mux.ServeHTTP(bulkDictRR, bulkDictReq)
	if bulkDictRR.Code != http.StatusCreated {
		t.Fatalf("expected bulk dictionary status 201, got %d body=%s", bulkDictRR.Code, bulkDictRR.Body.String())
	}

	if auditSpy.count != 2 {
		t.Fatalf("expected audit append count 2 for bulk endpoints, got %d", auditSpy.count)
	}

	invalidEnabledReq := httptest.NewRequest(http.MethodGet, "/admin/v1/dictionaries/query?enabled=maybe", nil)
	invalidEnabledRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidEnabledRR, invalidEnabledReq)
	if invalidEnabledRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid enabled filter status 400, got %d", invalidEnabledRR.Code)
	}

	invalidConfigReq := httptest.NewRequest(http.MethodPost, "/admin/v1/configs", strings.NewReader(`{"key":"","value":"on"}`))
	invalidConfigReq.Header.Set("Content-Type", "application/json")
	invalidConfigRR := httptest.NewRecorder()
	mux.ServeHTTP(invalidConfigRR, invalidConfigReq)
	if invalidConfigRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid config payload status 400, got %d", invalidConfigRR.Code)
	}
}
