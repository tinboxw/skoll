package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/pkg/security"
)

type fakeAuditEventSink struct {
	err    error
	events []*domainaudit.Event
}

func (f *fakeAuditEventSink) AppendEvent(_ context.Context, event *domainaudit.Event) error {
	f.events = append(f.events, event)
	return f.err
}

func TestRequestAuditRecordsSuccess(t *testing.T) {
	sink := &fakeAuditEventSink{}
	now := time.Date(2026, time.June, 19, 12, 0, 0, 0, time.UTC)
	handler := RequestAudit(sink,
		WithRequestAuditClock(func() time.Time { return now }),
		WithRequestAuditID(func() shared.ID { return shared.ID("event-1") }),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/skoll/v1/users?active=true", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("User-Agent", "skoll-test")
	req.Header.Set("X-Request-Id", "req-1")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "user-1", Role: "admin"}))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected response status 200, got %d", resp.Code)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(sink.events))
	}
	event := sink.events[0]
	if event.ID != shared.ID("event-1") ||
		event.Type != domainaudit.EventTypeOperation ||
		event.Action != domainaudit.AuditAction("http.request.post") ||
		event.Result != domainaudit.EventResultSuccess ||
		event.Risk != domainaudit.EventRiskLow ||
		event.Actor.ID != shared.ID("user-1") ||
		event.Actor.Name != "admin" ||
		event.Resource.Type != "http_request" ||
		event.Resource.ID != "POST /skoll/v1/users" ||
		!event.OccurredAt.Equal(now) {
		t.Fatalf("unexpected event: %+v", event)
	}
	if event.Trace.RequestID != "req-1" || event.Trace.IP != "192.0.2.10" || event.Trace.UserAgent != "skoll-test" {
		t.Fatalf("unexpected trace: %+v", event.Trace)
	}
	if event.Metadata["status"] != http.StatusOK || event.SourceData["query"] != "active=true" {
		t.Fatalf("unexpected metadata/source data: metadata=%+v source=%+v", event.Metadata, event.SourceData)
	}
}

func TestRequestAuditRecordsFailureAndUnauthorized(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantResult domainaudit.EventResult
	}{
		{name: "failure", statusCode: http.StatusInternalServerError, wantResult: domainaudit.EventResultFailure},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, wantResult: domainaudit.EventResultDenied},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sink := &fakeAuditEventSink{}
			handler := RequestAudit(sink,
				WithRequestAuditID(func() shared.ID { return shared.ID("event-" + tc.name) }),
			)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/skoll/v1/users", nil))

			if len(sink.events) != 1 {
				t.Fatalf("expected one audit event, got %d", len(sink.events))
			}
			if sink.events[0].Result != tc.wantResult {
				t.Fatalf("expected result %s, got %+v", tc.wantResult, sink.events[0])
			}
			if sink.events[0].Actor.Type != "anonymous" || sink.events[0].Actor.ID != shared.ID("anonymous") {
				t.Fatalf("expected anonymous actor, got %+v", sink.events[0].Actor)
			}
		})
	}
}

func TestRequestAuditSkipPaths(t *testing.T) {
	sink := &fakeAuditEventSink{}
	handler := RequestAudit(sink,
		WithRequestAuditSkipPaths("/skoll/health", "/skoll/docs"),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/skoll/docs/swagger", nil))

	if len(sink.events) != 0 {
		t.Fatalf("expected skipped audit event, got %+v", sink.events)
	}
}

func TestRequestAuditIgnoresSinkError(t *testing.T) {
	sink := &fakeAuditEventSink{err: errors.New("sink failed")}
	handler := RequestAudit(sink,
		WithRequestAuditID(func() shared.ID { return shared.ID("event-1") }),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodPost, "/skoll/v1/roles", nil))

	if resp.Code != http.StatusCreated {
		t.Fatalf("audit sink error should not change response, got %d", resp.Code)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected attempted audit event, got %d", len(sink.events))
	}
}
