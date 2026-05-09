package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	h := Auth()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}

	reqBearerInvalid := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	reqBearerInvalid.Header.Set("Authorization", "Basic abc")
	respBearerInvalid := httptest.NewRecorder()
	h.ServeHTTP(respBearerInvalid, reqBearerInvalid)
	if respBearerInvalid.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-bearer auth, got %d", respBearerInvalid.Code)
	}

	reqBearer := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	reqBearer.Header.Set("Authorization", "Bearer test-token")
	respBearer := httptest.NewRecorder()
	h.ServeHTTP(respBearer, reqBearer)
	if respBearer.Code != http.StatusOK {
		t.Fatalf("expected 200 for bearer auth, got %d", respBearer.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp2 := httptest.NewRecorder()
	h.ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusOK {
		t.Fatalf("health should bypass auth, got %d", resp2.Code)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/v1/plugins?enabled=true", nil)
	resp3 := httptest.NewRecorder()
	h.ServeHTTP(resp3, req3)
	if resp3.Code != http.StatusOK {
		t.Fatalf("plugin sync path should bypass auth, got %d", resp3.Code)
	}

	req4 := httptest.NewRequest(http.MethodPost, "/v1/plugins/demo/enable", nil)
	resp4 := httptest.NewRecorder()
	h.ServeHTTP(resp4, req4)
	if resp4.Code != http.StatusOK {
		t.Fatalf("plugin sub path should bypass auth, got %d", resp4.Code)
	}

	req5 := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
	resp5 := httptest.NewRecorder()
	h.ServeHTTP(resp5, req5)
	if resp5.Code != http.StatusOK {
		t.Fatalf("auth login should bypass auth, got %d", resp5.Code)
	}
}
