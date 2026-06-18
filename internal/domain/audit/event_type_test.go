package audit

import "testing"

func TestEventTypeValidateAcceptsKnownTypes(t *testing.T) {
	cases := []EventType{
		EventTypeOperation,
		EventTypeLogin,
		EventTypeError,
		EventTypePlugin,
		EventTypeSecurity,
	}

	for _, eventType := range cases {
		t.Run(eventType.String(), func(t *testing.T) {
			if err := eventType.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestEventTypeValidateRejectsUnknownTypes(t *testing.T) {
	for _, eventType := range []EventType{"", "audit", "log", "operation.login"} {
		t.Run(eventType.String(), func(t *testing.T) {
			if err := eventType.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestParseEventTypeNormalizesInput(t *testing.T) {
	got, err := ParseEventType(" Login ")
	if err != nil {
		t.Fatalf("ParseEventType() error = %v", err)
	}
	if got != EventTypeLogin {
		t.Fatalf("ParseEventType() = %q, want %q", got, EventTypeLogin)
	}
}

func TestParseEventTypeRejectsInvalidInput(t *testing.T) {
	if _, err := ParseEventType("trace"); err == nil {
		t.Fatal("expected invalid event type error")
	}
}

func TestAllEventTypesReturnsCopy(t *testing.T) {
	got := AllEventTypes()
	if len(got) != 5 {
		t.Fatalf("len(AllEventTypes()) = %d, want 5", len(got))
	}
	got[0] = EventType("changed")

	again := AllEventTypes()
	if again[0] != EventTypeOperation {
		t.Fatalf("AllEventTypes leaked internal slice: %+v", again)
	}
}
