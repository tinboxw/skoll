package pluginsdk

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEventPublicationValidatesTypedBoundedEnvelope(t *testing.T) {
	declaration := EventPublicationDeclaration{
		Name: "inventory.changed", SchemaVersion: 2,
		PayloadType: "inventory.change", Scope: EventScopeTenant,
	}
	publication := EventPublication{
		IdempotencyKey: "inventory-42-v2",
		Name:           "inventory.changed", SchemaVersion: 2,
		Scope:         EventScope{TenantID: "tenant-1", OrganizationID: "org-1"},
		CorrelationID: "request-42", CausationID: "command-42",
		Subject: EventSubject{Type: "inventory.item", ID: "item-42"},
		Payload: EventPayload{
			"quantity": {Type: DataValueDecimal, Value: "12.50"},
			"released": {Type: DataValueBoolean, Value: "true"},
		},
	}
	if err := publication.Validate(declaration); err != nil {
		t.Fatalf("expected valid publication, got %v", err)
	}
	envelope := EventEnvelope{
		ID: "event-42", Publisher: "warehouse", Name: publication.Name,
		SchemaVersion: publication.SchemaVersion, PayloadType: declaration.PayloadType,
		Scope: publication.Scope, CorrelationID: publication.CorrelationID,
		CausationID: publication.CausationID, Subject: publication.Subject,
		Payload: publication.Payload, OccurredAt: time.Now().UTC(),
	}
	if err := envelope.Validate(declaration); err != nil {
		t.Fatalf("expected valid envelope, got %v", err)
	}
}

func TestEventPublicationRejectsUndeclaredVersionScopeAndPayload(t *testing.T) {
	declaration := EventPublicationDeclaration{
		Name: "inventory.changed", SchemaVersion: 1,
		PayloadType: "inventory.change", Scope: EventScopeTenant,
	}
	valid := EventPublication{
		IdempotencyKey: "event-1", Name: declaration.Name, SchemaVersion: 1,
		Scope: EventScope{TenantID: "tenant-1"}, CorrelationID: "request-1",
		Payload: EventPayload{"quantity": {Type: DataValueInteger, Value: "1"}},
	}
	tests := []struct {
		name string
		edit func(*EventPublication)
		code EventErrorCode
	}{
		{name: "name", edit: func(p *EventPublication) { p.Name = "inventory.deleted" }, code: EventErrorUndeclaredPublication},
		{name: "version", edit: func(p *EventPublication) { p.SchemaVersion = 2 }, code: EventErrorUnsupportedSchema},
		{name: "scope", edit: func(p *EventPublication) { p.Scope = EventScope{} }, code: EventErrorScopeMismatch},
		{name: "typed payload", edit: func(p *EventPublication) {
			p.Payload["quantity"] = DataValue{Type: DataValueInteger, Value: "1.0"}
		}, code: EventErrorInvalidRequest},
		{name: "payload bound", edit: func(p *EventPublication) {
			p.Payload["note"] = DataValue{Type: DataValueString, Value: strings.Repeat("x", MaxEventPayloadBytes)}
		}, code: EventErrorPayloadLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject := valid
			subject.Payload = EventPayload{}
			for key, value := range valid.Payload {
				subject.Payload[key] = value
			}
			tt.edit(&subject)
			var eventErr *EventError
			if err := subject.Validate(declaration); !errors.As(err, &eventErr) || eventErr.Code != tt.code {
				t.Fatalf("expected %s, got %v", tt.code, err)
			}
		})
	}
}

func TestEventSubscriptionDeclarationRequiresExactCapabilities(t *testing.T) {
	valid := EventSubscriptionDeclaration{
		Publisher: "workflow", Name: "approval.completed",
		SchemaVersions: []uint32{1, 2}, Handler: "onApprovalCompleted",
		RetryPolicy: "standard",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid subscription, got %v", err)
	}
	for _, invalid := range []EventSubscriptionDeclaration{
		{Name: valid.Name, SchemaVersions: []uint32{1}, Handler: valid.Handler, RetryPolicy: "standard"},
		{Publisher: valid.Publisher, Name: valid.Name, Handler: valid.Handler, RetryPolicy: "standard"},
		{Publisher: valid.Publisher, Name: valid.Name, SchemaVersions: []uint32{1, 1}, Handler: valid.Handler, RetryPolicy: "standard"},
		{Publisher: valid.Publisher, Name: valid.Name, SchemaVersions: []uint32{1}, Handler: valid.Handler, RetryPolicy: "forever"},
	} {
		if err := invalid.Validate(); err == nil {
			t.Fatalf("expected invalid declaration: %+v", invalid)
		}
	}
}

func TestEventDeliveryRejectsUndeclaredOrUnsupportedConsumption(t *testing.T) {
	declaration := EventSubscriptionDeclaration{
		Publisher: "warehouse", Name: "inventory.received", SchemaVersions: []uint32{1},
		Handler: "onInventoryReceived", RetryPolicy: "standard",
	}
	delivery := EventDelivery{
		DeliveryID: "delivery-1", Subscriber: "analytics", Handler: declaration.Handler,
		Envelope: EventEnvelope{
			ID: "event-1", Publisher: declaration.Publisher, Name: declaration.Name,
			SchemaVersion: 1, PayloadType: "inventory.received", CorrelationID: "correlation-1",
			Payload: EventPayload{}, OccurredAt: time.Now().UTC(),
		},
	}
	if err := delivery.Validate(declaration); err != nil {
		t.Fatalf("valid event delivery rejected: %v", err)
	}
	for name, alter := range map[string]func(*EventDelivery){
		"publisher": func(item *EventDelivery) { item.Envelope.Publisher = "forged" },
		"name":      func(item *EventDelivery) { item.Envelope.Name = "inventory.changed" },
		"version":   func(item *EventDelivery) { item.Envelope.SchemaVersion = 2 },
		"handler":   func(item *EventDelivery) { item.Handler = "privateHandler" },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := delivery
			alter(&invalid)
			if err := invalid.Validate(declaration); err == nil {
				t.Fatalf("invalid %s accepted", name)
			}
		})
	}
}
