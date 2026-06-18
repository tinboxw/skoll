package audit

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewEventBuildsCompleteAuditEvent(t *testing.T) {
	now := time.Date(2026, time.June, 19, 10, 0, 0, 0, time.UTC)
	metadata := map[string]any{" reason ": "manual"}
	sourceData := map[string]any{" account ": "admin"}

	event, err := NewEvent(EventInput{
		ID:     shared.ID("audit-1"),
		Type:   EventTypeOperation,
		Action: AuditAction("menu.node.update"),
		Actor: ActorRef{
			Type: " User ",
			ID:   shared.ID(" actor-1 "),
			Name: " Admin ",
		},
		Resource: ResourceRef{
			Type: " Menu ",
			ID:   " menu-1 ",
			Name: " Settings ",
		},
		Result: EventResultSuccess,
		Trace: TraceContext{
			TraceID:   " trace-1 ",
			RequestID: " req-1 ",
			Method:    " post ",
			Path:      " /skoll/v1/menus ",
			IP:        " 127.0.0.1 ",
			UserAgent: " test-agent ",
		},
		Metadata:   metadata,
		SourceData: sourceData,
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	if event.Type != EventTypeOperation || event.Action != AuditAction("menu.node.update") {
		t.Fatalf("unexpected event type/action: %+v", event)
	}
	if event.Actor.Type != "user" || event.Actor.ID != shared.ID("actor-1") || event.Actor.Name != "Admin" {
		t.Fatalf("unexpected actor: %+v", event.Actor)
	}
	if event.Resource.Type != "menu" || event.Resource.ID != "menu-1" || event.Resource.Name != "Settings" {
		t.Fatalf("unexpected resource: %+v", event.Resource)
	}
	if event.Result != EventResultSuccess || event.Risk != EventRiskLow {
		t.Fatalf("unexpected result/risk: result=%q risk=%q", event.Result, event.Risk)
	}
	if event.Trace.Method != "POST" || event.Trace.Path != "/skoll/v1/menus" || event.Trace.RequestID != "req-1" {
		t.Fatalf("unexpected trace: %+v", event.Trace)
	}
	if event.Metadata["reason"] != "manual" {
		t.Fatalf("unexpected metadata: %+v", event.Metadata)
	}
	metadata["reason"] = "changed"
	if event.Metadata["reason"] != "manual" {
		t.Fatalf("metadata should be copied: %+v", event.Metadata)
	}
	if event.SourceData["account"] != "admin" {
		t.Fatalf("unexpected source data: %+v", event.SourceData)
	}
	sourceData["account"] = "changed"
	if event.SourceData["account"] != "admin" {
		t.Fatalf("source data should be copied: %+v", event.SourceData)
	}
}

func TestNewEventAcceptsExplicitRiskAndDeniedResult(t *testing.T) {
	event, err := NewEvent(validEventInput(func(in *EventInput) {
		in.Type = EventTypeSecurity
		in.Result = EventResultDenied
		in.Risk = EventRiskHigh
	}))
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	if event.Result != EventResultDenied || event.Risk != EventRiskHigh {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestNewEventRejectsIncompleteFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*EventInput)
	}{
		{name: "missing id", mutate: func(in *EventInput) { in.ID = "" }},
		{name: "invalid type", mutate: func(in *EventInput) { in.Type = EventType("trace") }},
		{name: "invalid action", mutate: func(in *EventInput) { in.Action = AuditAction("login_failed") }},
		{name: "missing actor type", mutate: func(in *EventInput) { in.Actor.Type = "" }},
		{name: "missing resource type", mutate: func(in *EventInput) { in.Resource.Type = "" }},
		{name: "invalid result", mutate: func(in *EventInput) { in.Result = EventResult("ok") }},
		{name: "invalid risk", mutate: func(in *EventInput) { in.Risk = EventRisk("warn") }},
		{name: "missing time", mutate: func(in *EventInput) { in.OccurredAt = time.Time{} }},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEvent(validEventInput(tt.mutate))
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestEventResultAndRiskValidation(t *testing.T) {
	for _, result := range []EventResult{EventResultSuccess, EventResultFailure, EventResultDenied} {
		if err := result.Validate(); err != nil {
			t.Fatalf("result %q should be valid: %v", result, err)
		}
	}
	if err := EventResult("partial").Validate(); err == nil {
		t.Fatal("expected invalid result")
	}

	for _, risk := range []EventRisk{EventRiskLow, EventRiskMedium, EventRiskHigh, EventRiskCritical} {
		if err := risk.Validate(); err != nil {
			t.Fatalf("risk %q should be valid: %v", risk, err)
		}
	}
	if err := EventRisk("warning").Validate(); err == nil {
		t.Fatal("expected invalid risk")
	}
}

func validEventInput(mutate func(*EventInput)) EventInput {
	in := EventInput{
		ID:     shared.ID("audit-1"),
		Type:   EventTypeOperation,
		Action: AuditAction("user.account.create"),
		Actor: ActorRef{
			Type: "user",
			ID:   shared.ID("actor-1"),
		},
		Resource: ResourceRef{
			Type: "user",
			ID:   "user-1",
		},
		Result:     EventResultSuccess,
		OccurredAt: time.Date(2026, time.June, 19, 10, 0, 0, 0, time.UTC),
	}
	if mutate != nil {
		mutate(&in)
	}
	return in
}
