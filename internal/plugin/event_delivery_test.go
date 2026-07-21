package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/event"
)

func TestHTTPEventDeliveryClientSendsCurrentEnvelope(t *testing.T) {
	var received EventDeliveryEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/plugin/_skoll/events" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		if r.Header.Get("Idempotency-Key") != received.DeliveryID || r.Header.Get("X-Skoll-Plugin-ID") != "reports" || r.Header.Get("X-Skoll-Event-Handler") != "onApprovalCompleted" {
			t.Fatalf("unexpected delivery headers: %+v", r.Header)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	occurredAt := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	evt := event.BusinessEvent{
		ID: "event-approval-1", EventName: event.BusinessEventApprovalCompleted, Source: "workflow",
		SubjectType: "approval", SubjectID: "approval-1", Payload: map[string]any{"result": "approved"},
		Metadata: map[string]string{"traceId": "trace-1"}, OccurredAt: occurredAt,
	}
	client := NewHTTPEventDeliveryClient(nil, time.Second)
	if err := client.Deliver(context.Background(), Info{ID: "reports", ServiceBaseURL: server.URL + "/plugin/"}, EventSubscription{Handler: "onApprovalCompleted"}, evt); err != nil {
		t.Fatalf("deliver event: %v", err)
	}
	wantDeliveryID := event.BusinessDeliveryID(evt.ID, "reports:onApprovalCompleted")
	if received.DeliveryID != wantDeliveryID || received.PluginID != "reports" || received.EventID != evt.ID || received.Subject.ID != "approval-1" {
		t.Fatalf("unexpected envelope: %+v", received)
	}
}

func TestHTTPEventDeliveryClientRejectsFailureAndRedirect(t *testing.T) {
	redirectTargetCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirected" {
			redirectTargetCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, "/redirected", http.StatusTemporaryRedirect)
	}))
	defer server.Close()

	client := NewHTTPEventDeliveryClient(nil, time.Second)
	err := client.Deliver(context.Background(), Info{ID: "reports", ServiceBaseURL: server.URL}, EventSubscription{Handler: "handler"}, event.BusinessEvent{ID: "event-1", EventName: event.BusinessEventInboundCompleted})
	if err == nil || !strings.Contains(err.Error(), "status 307") {
		t.Fatalf("expected observable redirect failure, got %v", err)
	}
	if redirectTargetCalled {
		t.Fatal("event delivery followed a redirect")
	}
}

func TestHTTPEventDeliveryClientRejectsInvalidServiceURL(t *testing.T) {
	client := NewHTTPEventDeliveryClient(nil, time.Second)
	err := client.Deliver(context.Background(), Info{ID: "reports", ServiceBaseURL: "file:///tmp/plugin"}, EventSubscription{Handler: "handler"}, event.BusinessEvent{ID: "event-1", EventName: event.BusinessEventInboundCompleted})
	if err != ErrPluginManifestBroken {
		t.Fatalf("expected broken manifest, got %v", err)
	}
}
