package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewPluginLifecycleAuditEvent(t *testing.T) {
	now := time.Date(2026, 7, 18, 13, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/reports/enable", nil)
	req.Header.Set("X-Request-Id", "req-lifecycle-1")
	event, err := NewPluginLifecycleAuditEvent(PluginLifecycleAuditInput{
		ID:         shared.ID("audit-plugin-lifecycle-1"),
		ActorID:    "alice",
		ActorName:  "super_admin",
		Operation:  "enable",
		PluginID:   "reports",
		Succeeded:  true,
		Request:    req,
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("NewPluginLifecycleAuditEvent() error = %v", err)
	}
	if event.Type != domainaudit.EventTypePlugin || event.Action != domainaudit.AuditAction("plugin.lifecycle.enable") || event.Result != domainaudit.EventResultSuccess || event.Risk != domainaudit.EventRiskHigh {
		t.Fatalf("unexpected lifecycle audit identity: %+v", event)
	}
	if event.Actor.ID != shared.ID("alice") || event.Resource.Type != "plugin" || event.Resource.ID != "reports" || event.Trace.RequestID != "req-lifecycle-1" {
		t.Fatalf("unexpected lifecycle audit references: %+v", event)
	}
}

func TestNewPluginLifecycleAuditEventFailure(t *testing.T) {
	cause := errors.New("system builtin plugin cannot be disabled")
	event, err := NewPluginLifecycleAuditEvent(PluginLifecycleAuditInput{
		ID:         shared.ID("audit-plugin-lifecycle-failure"),
		Operation:  "disable",
		PluginID:   "builtin-auth",
		Error:      cause.Error(),
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("NewPluginLifecycleAuditEvent() error = %v", err)
	}
	if event.Result != domainaudit.EventResultFailure || event.Metadata["error"] != cause.Error() {
		t.Fatalf("unexpected failed lifecycle audit: %+v", event)
	}
}
