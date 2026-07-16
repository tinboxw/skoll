package pharmaoa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

type businessMetricsHandlerFixture struct {
	input pharmaoasvc.BusinessMetricsInput
}

func (f *businessMetricsHandlerFixture) Get(_ context.Context, in pharmaoasvc.BusinessMetricsInput) (*pharmaoasvc.BusinessMetricsSnapshot, error) {
	f.input = in
	return &pharmaoasvc.BusinessMetricsSnapshot{GeneratedAt: time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)}, nil
}

func TestBusinessMetricsHTTPUsesJWTActorAndParsesBoundedQuery(t *testing.T) {
	fixture := &businessMetricsHandlerFixture{}
	mux := http.NewServeMux()
	RegisterBusinessMetricsRoutes(mux, fixture)
	req := httptest.NewRequest(http.MethodGet, "/v1/pharma-oa/business-metrics?from=2026-07-01&to=2026-07-07&bucket=week&qualificationDays=45&actorId=spoofed", nil)
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "manager-1", Role: "manager"}))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if fixture.input.ActorID != "manager-1" || fixture.input.Bucket != pharmaoasvc.BusinessMetricsBucketWeek || fixture.input.QualificationDays != 45 {
		t.Fatalf("unexpected input: %+v", fixture.input)
	}
	if got := fixture.input.To.UTC().Format(time.RFC3339Nano); got != "2026-07-07T23:59:59.999999999Z" {
		t.Fatalf("unexpected inclusive to: %s", got)
	}
}

func TestBusinessMetricsHTTPRejectsInvalidTime(t *testing.T) {
	fixture := &businessMetricsHandlerFixture{}
	mux := http.NewServeMux()
	RegisterBusinessMetricsRoutes(mux, fixture)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/pharma-oa/business-metrics?from=bad", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
