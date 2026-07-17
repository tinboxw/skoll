package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewPluginRouteAuditEvent(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodPost, "/skoll/v1/plugins/reports/api/exports", nil)
	req.Header.Set("X-Request-Id", "req-plugin-1")
	req.Header.Set("X-Trace-Id", "trace-plugin-1")

	event, err := NewPluginRouteAuditEvent(PluginRouteAuditInput{
		ID:          shared.ID("audit-plugin-route-1"),
		ActorID:     "alice",
		ActorName:   "editor",
		Source:      "plugin.reports",
		Permission:  "reports.exports.create",
		AuditAction: "reports.exports.create",
		StatusCode:  http.StatusCreated,
		Request:     req,
		OccurredAt:  now,
	})
	if err != nil {
		t.Fatalf("NewPluginRouteAuditEvent() error = %v", err)
	}
	if event.Type != domainaudit.EventTypePlugin || event.Action != domainaudit.AuditAction("reports.exports.create") || event.Result != domainaudit.EventResultSuccess {
		t.Fatalf("unexpected plugin route audit identity: %+v", event)
	}
	if event.Actor.ID != shared.ID("alice") || event.Resource.Type != "plugin_route" || event.Resource.Name != "plugin.reports" {
		t.Fatalf("unexpected plugin route audit references: %+v", event)
	}
	if event.Metadata["permission"] != "reports.exports.create" || event.Metadata["status"] != http.StatusCreated || event.Trace.RequestID != "req-plugin-1" {
		t.Fatalf("unexpected plugin route audit evidence: %+v", event)
	}
}

func TestNewPluginRouteAuditEventRejectsInvalidAction(t *testing.T) {
	_, err := NewPluginRouteAuditEvent(PluginRouteAuditInput{
		ID:          shared.ID("audit-plugin-route-invalid"),
		AuditAction: "invalid",
		OccurredAt:  time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected invalid audit action error")
	}
}
