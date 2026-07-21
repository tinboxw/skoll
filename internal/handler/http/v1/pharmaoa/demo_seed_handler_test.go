package pharmaoa

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

type demoSeedHTTPService struct{ actors []string }

func (s *demoSeedHTTPService) Status(_ context.Context, actorID string) (pharmaoasvc.DemoSeedSnapshot, error) {
	s.actors = append(s.actors, actorID)
	return pharmaoasvc.DemoSeedSnapshot{State: pharmaoasvc.DemoSeedStateReady, Stage: "ready", Counts: map[string]int{}}, nil
}

func (s *demoSeedHTTPService) Apply(_ context.Context, actorID string) (pharmaoasvc.DemoSeedSnapshot, error) {
	s.actors = append(s.actors, actorID)
	return pharmaoasvc.DemoSeedSnapshot{State: pharmaoasvc.DemoSeedStateApplied, Stage: "complete", Applied: true, ActorID: actorID, Counts: map[string]int{"employees": 1}}, nil
}

func TestDemoSeedHTTPUsesJWTActor(t *testing.T) {
	service := &demoSeedHTTPService{}
	mux := http.NewServeMux()
	RegisterDemoSeedRoutes(mux, service)

	apply := demoSeedHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/demo-seed/apply", `{"actorId":"spoofed"}`)
	if apply.Code != http.StatusOK || !bytes.Contains(apply.Body.Bytes(), []byte(`"state":"applied"`)) {
		t.Fatalf("apply status=%d body=%s", apply.Code, apply.Body.String())
	}
	status := demoSeedHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/demo-seed/status", "")
	if status.Code != http.StatusOK || !bytes.Contains(status.Body.Bytes(), []byte(`"state":"ready"`)) {
		t.Fatalf("status=%d body=%s", status.Code, status.Body.String())
	}
	if len(service.actors) != 2 || service.actors[0] != "seed-admin" || service.actors[1] != "seed-admin" {
		t.Fatalf("handler did not derive actor from JWT: %+v", service.actors)
	}
}

func demoSeedHTTPRequest(handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "seed-admin", Role: "admin"}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
