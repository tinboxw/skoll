package plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestInProcessBackendRegistryBuildsLazilyAndResets(t *testing.T) {
	registry := NewInProcessBackendRegistry()
	var builds atomic.Int32
	if err := registry.Register("pharma_oa", func() (http.Handler, error) {
		builds.Add(1)
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}), nil
	}); err != nil {
		t.Fatalf("register backend: %v", err)
	}
	if builds.Load() != 0 {
		t.Fatal("backend must not be built during registration")
	}

	for range 2 {
		recorder := httptest.NewRecorder()
		handled, err := registry.ServeHTTP("pharma_oa", recorder, httptest.NewRequest(http.MethodGet, "/v1/plugins/pharma_oa/api/employees", nil))
		if err != nil || !handled || recorder.Code != http.StatusNoContent {
			t.Fatalf("serve backend: handled=%v status=%d err=%v", handled, recorder.Code, err)
		}
	}
	if builds.Load() != 1 {
		t.Fatalf("expected one lazy build, got %d", builds.Load())
	}
	if err := registry.Reset("pharma_oa"); err != nil {
		t.Fatalf("reset backend: %v", err)
	}
	_, _ = registry.ServeHTTP("pharma_oa", httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if builds.Load() != 2 {
		t.Fatalf("expected rebuild after reset, got %d", builds.Load())
	}
}

func TestInProcessBackendRegistryReportsFactoryFailure(t *testing.T) {
	registry := NewInProcessBackendRegistry()
	want := errors.New("build failed")
	if err := registry.Register("broken", func() (http.Handler, error) { return nil, want }); err != nil {
		t.Fatalf("register backend: %v", err)
	}
	handled, err := registry.ServeHTTP("broken", httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !handled || !errors.Is(err, want) {
		t.Fatalf("expected handled factory failure, handled=%v err=%v", handled, err)
	}
}
