package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewErrorAuditEventForHandledError(t *testing.T) {
	now := time.Date(2026, time.June, 19, 14, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodGet, "/skoll/v1/users?offset=0", nil)
	req.Header.Set("X-Request-Id", "req-error")

	event, err := NewErrorAuditEvent(ErrorAuditInput{
		ID:         shared.ID("event-error"),
		LogID:      shared.ID("error-log"),
		Request:    req,
		StatusCode: http.StatusInternalServerError,
		ErrorCode:  "handler_error",
		Summary:    "handler returned error status",
		Message:    "database unavailable",
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("NewErrorAuditEvent error: %v", err)
	}
	if event.Type != domainaudit.EventTypeError ||
		event.Action != domainaudit.AuditAction("error.request.handled") ||
		event.Result != domainaudit.EventResultFailure ||
		event.Risk != domainaudit.EventRiskMedium ||
		event.Trace.RequestID != "req-error" ||
		event.SourceData["kind"] != "error_log" ||
		event.SourceData["errorCode"] != "handler_error" ||
		event.SourceData["level"] != "error" {
		t.Fatalf("unexpected handled error event: %+v", event)
	}
}

func TestErrorAuditRecordsHandledError(t *testing.T) {
	sink := &fakeAuditEventSink{}
	handler := ErrorAudit(sink,
		WithErrorAuditID(sequenceAuditIDs("event-1", "log-1")),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/skoll/v1/users", nil)
	req.Header.Set("X-Request-Id", "req-500")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected one error event, got %d", len(sink.events))
	}
	if sink.events[0].Action != domainaudit.AuditAction("error.request.handled") || sink.events[0].Trace.RequestID != "req-500" {
		t.Fatalf("unexpected event: %+v", sink.events[0])
	}
}

func TestErrorAuditRecordsPanic(t *testing.T) {
	sink := &fakeAuditEventSink{}
	handler := ErrorAudit(sink,
		WithErrorAuditID(sequenceAuditIDs("event-panic", "log-panic")),
	)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("explode")
	}))

	req := httptest.NewRequest(http.MethodGet, "/skoll/v1/roles", nil)
	req.Header.Set("X-Trace-Id", "trace-panic")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected one panic event, got %d", len(sink.events))
	}
	event := sink.events[0]
	if event.Action != domainaudit.AuditAction("error.request.panic") ||
		event.Risk != domainaudit.EventRiskCritical ||
		event.Trace.TraceID != "trace-panic" ||
		event.SourceData["level"] != "critical" ||
		event.SourceData["message"] != "explode" {
		t.Fatalf("unexpected panic event: %+v", event)
	}
}

func sequenceAuditIDs(ids ...string) AuditIDFunc {
	index := 0
	return func() shared.ID {
		if index >= len(ids) {
			return shared.ID(ids[len(ids)-1])
		}
		id := ids[index]
		index++
		return shared.ID(id)
	}
}
