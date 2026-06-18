package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewPermissionDeniedAuditEvent(t *testing.T) {
	now := time.Date(2026, time.June, 19, 13, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodDelete, "/skoll/v1/roles/r1", nil)
	req.RemoteAddr = "198.51.100.20:5678"
	req.Header.Set("X-Request-Id", "req-denied")
	req.Header.Set("User-Agent", "denied-test")

	event, err := NewPermissionDeniedAuditEvent(PermissionDeniedAuditInput{
		ID:         shared.ID("event-denied"),
		ActorID:    "user-1",
		ActorName:  "editor",
		Resource:   "role",
		Action:     "delete",
		Reason:     "permission_denied",
		Request:    req,
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("NewPermissionDeniedAuditEvent error: %v", err)
	}
	if event.ID != shared.ID("event-denied") ||
		event.Type != domainaudit.EventTypeSecurity ||
		event.Action != domainaudit.AuditAction("system.security.deny") ||
		event.Result != domainaudit.EventResultDenied ||
		event.Risk != domainaudit.EventRiskHigh ||
		event.Actor.ID != shared.ID("user-1") ||
		event.Resource.Type != "role" ||
		event.Resource.ID != "delete" ||
		event.Trace.RequestID != "req-denied" ||
		event.Trace.IP != "198.51.100.20" ||
		!event.OccurredAt.Equal(now) {
		t.Fatalf("unexpected event: %+v", event)
	}
	if event.Metadata["resource"] != "role" || event.Metadata["action"] != "delete" || event.SourceData["kind"] != "permission_denied" {
		t.Fatalf("unexpected metadata/source data: metadata=%+v source=%+v", event.Metadata, event.SourceData)
	}
}
