package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/quota"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const (
	businessScaleRequestsPerPlugin = 64
	businessScaleMaxP95Latency     = 500 * time.Millisecond
)

type businessScaleConfig struct {
	pluginID string
	fail     atomic.Bool
	calls    atomic.Int64
	entered  chan<- struct{}
	release  <-chan struct{}
}

func (s *businessScaleConfig) Get(ctx context.Context) (map[string]any, error) {
	s.calls.Add(1)
	if s.fail.Load() {
		return nil, errors.New("simulated plugin dependency outage")
	}
	if s.entered != nil {
		select {
		case s.entered <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if s.release != nil {
		select {
		case <-s.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return map[string]any{"pluginId": s.pluginID}, nil
}

func (s *businessScaleConfig) Replace(context.Context, map[string]any) (map[string]any, error) {
	return nil, errors.New("configuration replacement is not declared")
}

func TestBusinessScaleFrameworkAcceptance(t *testing.T) {
	healthyPluginIDs := []string{"pharma_oa", "equipment", "crm", "employees", "inventory", "approvals"}
	allPluginIDs := append(append([]string(nil), healthyPluginIDs...), "abusive", "outage")

	releaseAbusive := make(chan struct{})
	abusiveEntered := make(chan struct{}, 4)
	configs := make(map[string]*businessScaleConfig, len(allPluginIDs))
	audits := make(map[string]*gatewayAuditRecorder, len(allPluginIDs))
	for _, pluginID := range allPluginIDs {
		configs[pluginID] = &businessScaleConfig{pluginID: pluginID}
		audits[pluginID] = &gatewayAuditRecorder{}
	}
	configs["abusive"].entered = abusiveEntered
	configs["abusive"].release = releaseAbusive

	policy := newPluginTestQuotaController().Policy()
	policy.HostCall = quota.Limit{RatePerSecond: 10000, Burst: 1024, MaxConcurrent: 4}
	quotas, err := quota.NewController(policy)
	if err != nil {
		t.Fatal(err)
	}
	gateway, err := NewHostGateway(func(pluginID string) (pluginsdk.HostServices, error) {
		configService, ok := configs[pluginID]
		if !ok {
			return pluginsdk.HostServices{}, fmt.Errorf("plugin %q is unavailable", pluginID)
		}
		host := gatewayHostWithCapabilities(pluginsdk.HostCapabilityConfigGet)
		host.PluginID = pluginID
		host.Config = configService
		host.Audit = audits[pluginID]
		return host, nil
	}, "business-scale-jwt-secret", quotas, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })

	tokens := make(map[string]string, len(allPluginIDs))
	for _, pluginID := range allPluginIDs {
		credential, issueErr := gateway.Issue(pluginID)
		if issueErr != nil {
			t.Fatalf("issue %s credential: %v", pluginID, issueErr)
		}
		tokens[pluginID] = credential.Token
	}

	var abusiveGroup sync.WaitGroup
	abusiveResponses := make(chan *httptest.ResponseRecorder, 4)
	for worker := 0; worker < 4; worker++ {
		abusiveGroup.Add(1)
		go func() {
			defer abusiveGroup.Done()
			abusiveResponses <- businessScaleGatewayCall(gateway, tokens["abusive"], "/v1/config/get", "{}")
		}()
	}
	for worker := 0; worker < 4; worker++ {
		select {
		case <-abusiveEntered:
		case <-time.After(2 * time.Second):
			t.Fatal("abusive plugin did not occupy its concurrency leases")
		}
	}
	rejected := businessScaleGatewayCall(gateway, tokens["abusive"], "/v1/config/get", "{}")
	assertGatewayError(t, rejected, http.StatusTooManyRequests, "plugin_quota_exceeded", "")

	type loadResult struct {
		pluginID string
		latency  time.Duration
		err      error
	}
	results := make(chan loadResult, len(healthyPluginIDs)*businessScaleRequestsPerPlugin)
	var healthyGroup sync.WaitGroup
	for _, pluginID := range healthyPluginIDs {
		pluginID := pluginID
		healthyGroup.Add(1)
		go func() {
			defer healthyGroup.Done()
			for requestIndex := 0; requestIndex < businessScaleRequestsPerPlugin; requestIndex++ {
				startedAt := time.Now()
				response := businessScaleGatewayCall(gateway, tokens[pluginID], "/v1/config/get", "{}")
				latency := time.Since(startedAt)
				if response.Code != http.StatusOK {
					results <- loadResult{pluginID: pluginID, latency: latency, err: fmt.Errorf("status=%d body=%s", response.Code, response.Body.String())}
					continue
				}
				var body map[string]any
				if decodeErr := json.Unmarshal(response.Body.Bytes(), &body); decodeErr != nil {
					results <- loadResult{pluginID: pluginID, latency: latency, err: decodeErr}
					continue
				}
				if body["pluginId"] != pluginID {
					results <- loadResult{pluginID: pluginID, latency: latency, err: fmt.Errorf("cross-plugin response=%v", body)}
					continue
				}
				results <- loadResult{pluginID: pluginID, latency: latency}
			}
		}()
	}
	healthyGroup.Wait()
	close(results)

	latencies := make([]time.Duration, 0, len(healthyPluginIDs)*businessScaleRequestsPerPlugin)
	for result := range results {
		if result.err != nil {
			t.Errorf("healthy plugin %s request failed: %v", result.pluginID, result.err)
		}
		latencies = append(latencies, result.latency)
	}
	if t.Failed() {
		t.FailNow()
	}
	p95 := businessScalePercentile(latencies, 0.95)
	if p95 > businessScaleMaxP95Latency {
		t.Fatalf("healthy plugin p95 latency=%s exceeds %s", p95, businessScaleMaxP95Latency)
	}
	for _, pluginID := range healthyPluginIDs {
		if calls := configs[pluginID].calls.Load(); calls != businessScaleRequestsPerPlugin {
			t.Fatalf("plugin %s calls=%d want=%d", pluginID, calls, businessScaleRequestsPerPlugin)
		}
	}

	close(releaseAbusive)
	abusiveGroup.Wait()
	close(abusiveResponses)
	for response := range abusiveResponses {
		if response.Code != http.StatusOK {
			t.Fatalf("leased abusive request status=%d body=%s", response.Code, response.Body.String())
		}
	}
	recovered := businessScaleGatewayCall(gateway, tokens["abusive"], "/v1/config/get", "{}")
	if recovered.Code != http.StatusOK {
		t.Fatalf("abusive plugin did not recover after pressure release: status=%d body=%s", recovered.Code, recovered.Body.String())
	}

	forged := businessScaleGatewayCall(gateway, "forged-credential", "/v1/config/get", "{}")
	assertGatewayError(t, forged, http.StatusUnauthorized, "plugin_identity_invalid", "")

	oversizedRequest := httptest.NewRequest(http.MethodPost, "/v1/config/get", strings.NewReader("{}"))
	oversizedRequest.RemoteAddr = "127.0.0.1:1234"
	oversizedRequest.Header.Set("Authorization", "Bearer "+tokens["pharma_oa"])
	oversizedRequest.Header.Set("Content-Type", "application/json")
	oversizedRequest.ContentLength = maxHostRequestBytes + 1
	oversized := httptest.NewRecorder()
	gateway.ServeHTTP(oversized, oversizedRequest)
	assertGatewayError(t, oversized, http.StatusRequestEntityTooLarge, "host_request_too_large", "")

	undeclared := businessScaleGatewayCall(gateway, tokens["equipment"], "/v1/secrets/get", `{"key":"private"}`)
	assertGatewayError(t, undeclared, http.StatusForbidden, "host_capability_denied", "")

	configs["outage"].fail.Store(true)
	failed := businessScaleGatewayCall(gateway, tokens["outage"], "/v1/config/get", "{}")
	assertGatewayError(t, failed, http.StatusUnprocessableEntity, "host_call_failed", "")
	configs["outage"].fail.Store(false)
	recovered = businessScaleGatewayCall(gateway, tokens["outage"], "/v1/config/get", "{}")
	if recovered.Code != http.StatusOK {
		t.Fatalf("failing plugin did not recover: status=%d body=%s", recovered.Code, recovered.Body.String())
	}

	revokedToken := tokens["crm"]
	gateway.Revoke(revokedToken)
	revoked := businessScaleGatewayCall(gateway, revokedToken, "/v1/config/get", "{}")
	assertGatewayError(t, revoked, http.StatusUnauthorized, "plugin_identity_invalid", "")
	unaffected := businessScaleGatewayCall(gateway, tokens["employees"], "/v1/config/get", "{}")
	if unaffected.Code != http.StatusOK {
		t.Fatalf("credential revocation affected another plugin: status=%d body=%s", unaffected.Code, unaffected.Body.String())
	}
	reissued, err := gateway.Issue("crm")
	if err != nil {
		t.Fatalf("reissue crm credential: %v", err)
	}
	recovered = businessScaleGatewayCall(gateway, reissued.Token, "/v1/config/get", "{}")
	if recovered.Code != http.StatusOK {
		t.Fatalf("reissued plugin credential did not recover: status=%d body=%s", recovered.Code, recovered.Body.String())
	}

	assertBusinessScaleAuditEvidence(t, audits["abusive"], "plugin_quota_exceeded")
	assertBusinessScaleAuditEvidence(t, audits["outage"], "host_call_failed")
	abusiveRejections := uint64(0)
	for _, snapshot := range quotas.Snapshot("abusive") {
		if snapshot.Resource == quota.ResourceHostCall {
			abusiveRejections = snapshot.Rejected
			break
		}
	}
	t.Logf(
		"business-scale gate passed: plugins=%d healthy_requests=%d p95=%s abusive_rejections=%d",
		len(allPluginIDs),
		len(latencies),
		p95,
		abusiveRejections,
	)
}

func businessScaleGatewayCall(gateway *HostGateway, token, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	gateway.ServeHTTP(response, request)
	return response
}

func businessScalePercentile(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]time.Duration(nil), values...)
	sort.Slice(ordered, func(left, right int) bool { return ordered[left] < ordered[right] })
	index := int(float64(len(ordered)-1) * percentile)
	return ordered[index]
}

func assertBusinessScaleAuditEvidence(t *testing.T, recorder *gatewayAuditRecorder, errorCode string) {
	t.Helper()
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	for _, entry := range recorder.entries {
		if entry.Result == pluginsdk.AuditResultFailure && entry.Detail["errorCode"] == errorCode {
			if entry.Detail["owner"] == "" || entry.Detail["stage"] == "" {
				t.Fatalf("audit evidence is missing owner or stage: %+v", entry)
			}
			return
		}
	}
	t.Fatalf("audit evidence does not contain failure code %q: %+v", errorCode, recorder.entries)
}
