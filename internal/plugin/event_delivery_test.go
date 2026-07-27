package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func TestHTTPEventDeliveryClientSendsCurrentEnvelope(t *testing.T) {
	var received pluginsdk.EventDelivery
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

	delivery := pluginsdk.EventDelivery{
		DeliveryID: "delivery-approval-1", Subscriber: "reports", Handler: "onApprovalCompleted",
		Envelope: pluginsdk.EventEnvelope{
			ID: "event-approval-1", Publisher: "workflow", Name: "approval-completed",
			SchemaVersion: 1, PayloadType: "approval.completed", CorrelationID: "trace-1",
			Subject:    pluginsdk.EventSubject{Type: "approval", ID: "approval-1"},
			Payload:    pluginsdk.EventPayload{"result": {Type: pluginsdk.DataValueString, Value: "approved"}},
			OccurredAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC),
		},
	}
	client := NewHTTPEventDeliveryClient(nil, time.Second)
	if err := client.Deliver(context.Background(), Info{ID: "reports", ServiceBaseURL: server.URL + "/plugin/"}, delivery); err != nil {
		t.Fatalf("deliver event: %v", err)
	}
	if received.DeliveryID != delivery.DeliveryID || received.Subscriber != "reports" ||
		received.Envelope.ID != delivery.Envelope.ID || received.Envelope.Subject.ID != "approval-1" {
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
	err := client.Deliver(context.Background(), Info{ID: "reports", ServiceBaseURL: server.URL}, testCurrentDelivery())
	if err == nil || !strings.Contains(err.Error(), "status 307") {
		t.Fatalf("expected observable redirect failure, got %v", err)
	}
	if redirectTargetCalled {
		t.Fatal("event delivery followed a redirect")
	}
}

func TestHTTPEventDeliveryClientRejectsInvalidServiceURL(t *testing.T) {
	client := NewHTTPEventDeliveryClient(nil, time.Second)
	err := client.Deliver(context.Background(), Info{ID: "reports", ServiceBaseURL: "file:///tmp/plugin"}, testCurrentDelivery())
	if err != ErrPluginManifestBroken {
		t.Fatalf("expected broken manifest, got %v", err)
	}
}

func testCurrentDelivery() pluginsdk.EventDelivery {
	return pluginsdk.EventDelivery{
		DeliveryID: "delivery-1", Subscriber: "reports", Handler: "handler",
		Envelope: pluginsdk.EventEnvelope{
			ID: "event-1", Publisher: "warehouse", Name: "inbound-completed",
			SchemaVersion: 1, PayloadType: "inbound.completed", CorrelationID: "event-1",
			Payload: pluginsdk.EventPayload{}, OccurredAt: time.Now().UTC(),
		},
	}
}
