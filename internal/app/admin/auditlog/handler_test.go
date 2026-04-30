package auditlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/module/audit"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func newAuditMux() (*http.ServeMux, *audit.Service) {
	mux := http.NewServeMux()
	auditSvc := audit.NewService()
	NewHandler(auditSvc).Register(mux, nil, &testAPIRegistry{})
	return mux, auditSvc
}

func TestAdminAuditRoutes_RecentWithLimit(t *testing.T) {
	mux, _ := newAuditMux()

	for i := 0; i < 3; i++ {
		payload := []byte(`{"actor":"system","action":"create","target":"user"}`)
		req := httptest.NewRequest(http.MethodPost, "/admin/v1/audit-logs", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected audit append status 201, got %d", rr.Code)
		}
	}

	recentReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs?limit=2", nil)
	recentRR := httptest.NewRecorder()
	mux.ServeHTTP(recentRR, recentReq)
	if recentRR.Code != http.StatusOK {
		t.Fatalf("expected recent status 200, got %d", recentRR.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recentRR.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal recent response failed: %v", err)
	}
	items, ok := payload["items"].([]any)
	if !ok {
		t.Fatalf("expected items array payload, got %v", payload)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 recent records, got %d", len(items))
	}

	filteredReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs?page=1&size=10&actor=system&action=create&q=user", nil)
	filteredRR := httptest.NewRecorder()
	mux.ServeHTTP(filteredRR, filteredReq)
	if filteredRR.Code != http.StatusOK {
		t.Fatalf("expected filtered status 200, got %d", filteredRR.Code)
	}
	if !strings.Contains(filteredRR.Body.String(), `"total":3`) {
		t.Fatalf("expected filtered total field in response, got %s", filteredRR.Body.String())
	}

	profileReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs/profile", nil)
	profileRR := httptest.NewRecorder()
	mux.ServeHTTP(profileRR, profileReq)
	if profileRR.Code != http.StatusOK {
		t.Fatalf("expected audit profile status 200, got %d", profileRR.Code)
	}
	if !strings.Contains(profileRR.Body.String(), `"max_size":200`) {
		t.Fatalf("expected audit profile max size field, got %s", profileRR.Body.String())
	}

	controlProfileReq := httptest.NewRequest(http.MethodGet, "/admin/v1/admin-ops/control-profile", nil)
	controlProfileRR := httptest.NewRecorder()
	mux.ServeHTTP(controlProfileRR, controlProfileReq)
	if controlProfileRR.Code != http.StatusOK {
		t.Fatalf("expected control profile status 200, got %d", controlProfileRR.Code)
	}
	if !strings.Contains(controlProfileRR.Body.String(), `"audit_retention_days":180`) {
		t.Fatalf("expected retention profile payload, got %s", controlProfileRR.Body.String())
	}
}
