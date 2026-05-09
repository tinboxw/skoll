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

	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp2 := httptest.NewRecorder()
	h.ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusOK {
		t.Fatalf("health should bypass auth, got %d", resp2.Code)
	}
}
