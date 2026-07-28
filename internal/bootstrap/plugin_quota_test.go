package bootstrap

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/plugin/quota"
)

func TestPluginRequestQuotaIsolatesConcurrentPluginsAndRecovers(t *testing.T) {
	policy := testPluginQuotaPolicy()
	policy.Request.MaxConcurrent = 1
	controller, err := quota.NewController(policy)
	if err != nil {
		t.Fatal(err)
	}
	const path = "/v1/plugins/demo/api/items"
	started := make(chan struct{})
	release := make(chan struct{})
	manager := &pluginManagerWithExtensions{
		quotas: controller,
		routeHandlers: map[string]http.HandlerFunc{
			pluginRouteKey("demo", http.MethodGet, path): func(w http.ResponseWriter, _ *http.Request) {
				close(started)
				<-release
				w.WriteHeader(http.StatusNoContent)
			},
			pluginRouteKey("other", http.MethodGet, path): func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
		},
	}

	firstDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		manager.HandlePluginRoute("demo", http.MethodGet, path, recorder, request)
		firstDone <- recorder
	}()
	<-started

	rejected := httptest.NewRecorder()
	manager.HandlePluginRoute("demo", http.MethodGet, path, rejected, httptest.NewRequest(http.MethodGet, path, nil))
	if rejected.Code != http.StatusTooManyRequests ||
		rejected.Header().Get("X-Skoll-Quota-Resource") != string(quota.ResourceRequest) ||
		rejected.Header().Get("Retry-After") == "" {
		t.Fatalf("concurrent request was not backpressured: status=%d headers=%v body=%s", rejected.Code, rejected.Header(), rejected.Body.String())
	}

	isolated := httptest.NewRecorder()
	manager.HandlePluginRoute("other", http.MethodGet, path, isolated, httptest.NewRequest(http.MethodGet, path, nil))
	if isolated.Code != http.StatusNoContent {
		t.Fatalf("one plugin exhausted another plugin capacity: %d", isolated.Code)
	}

	close(release)
	if first := <-firstDone; first.Code != http.StatusNoContent {
		t.Fatalf("first request failed: %d", first.Code)
	}
}

func TestPluginRequestQuotaBoundsBodiesResponsesAndTimeout(t *testing.T) {
	policy := testPluginQuotaPolicy()
	policy.MaxRequestBytes = 4
	policy.MaxResponseBytes = 4
	policy.RequestTimeout = 10 * time.Millisecond
	controller, err := quota.NewController(policy)
	if err != nil {
		t.Fatal(err)
	}
	const base = "/v1/plugins/demo/api/"
	manager := &pluginManagerWithExtensions{
		quotas: controller,
		routeHandlers: map[string]http.HandlerFunc{
			pluginRouteKey("demo", http.MethodPost, base+"upload"): func(w http.ResponseWriter, r *http.Request) {
				_, _ = r.Body.Read(make([]byte, 16))
				w.WriteHeader(http.StatusNoContent)
			},
			pluginRouteKey("demo", http.MethodGet, base+"response"): func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("oversized"))
			},
			pluginRouteKey("demo", http.MethodGet, base+"timeout"): func(w http.ResponseWriter, r *http.Request) {
				<-r.Context().Done()
				w.WriteHeader(http.StatusNoContent)
			},
		},
	}

	tooLarge := httptest.NewRecorder()
	manager.HandlePluginRoute("demo", http.MethodPost, base+"upload", tooLarge, httptest.NewRequest(http.MethodPost, base+"upload", bytes.NewReader([]byte("12345"))))
	if tooLarge.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request was accepted: %d", tooLarge.Code)
	}

	response := httptest.NewRecorder()
	manager.HandlePluginRoute("demo", http.MethodGet, base+"response", response, httptest.NewRequest(http.MethodGet, base+"response", nil))
	if response.Code != http.StatusBadGateway || !bytes.Contains(response.Body.Bytes(), []byte("plugin_response_too_large")) {
		t.Fatalf("oversized response was not rejected: status=%d body=%s", response.Code, response.Body.String())
	}

	timeout := httptest.NewRecorder()
	manager.HandlePluginRoute("demo", http.MethodGet, base+"timeout", timeout, httptest.NewRequest(http.MethodGet, base+"timeout", nil))
	if timeout.Code != http.StatusGatewayTimeout || !bytes.Contains(timeout.Body.Bytes(), []byte("plugin_request_timeout")) {
		t.Fatalf("timed out request was not rejected: status=%d body=%s", timeout.Code, timeout.Body.String())
	}
}

var _ plugin.Manager = (*pluginManagerWithExtensions)(nil)
