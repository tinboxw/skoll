package audit

import (
	"fmt"
	"strings"
)

type EventType string

const (
	EventTypeOperation EventType = "operation"
	EventTypeLogin     EventType = "login"
	EventTypeError     EventType = "error"
	EventTypePlugin    EventType = "plugin"
	EventTypeSecurity  EventType = "security"
)

var allEventTypes = []EventType{
	EventTypeOperation,
	EventTypeLogin,
	EventTypeError,
	EventTypePlugin,
	EventTypeSecurity,
}

func AllEventTypes() []EventType {
	return append([]EventType(nil), allEventTypes...)
}

func ParseEventType(value string) (EventType, error) {
	eventType := EventType(strings.TrimSpace(strings.ToLower(value)))
	if err := eventType.Validate(); err != nil {
		return "", err
	}
	return eventType, nil
}

func (t EventType) Validate() error {
	switch t {
	case EventTypeOperation, EventTypeLogin, EventTypeError, EventTypePlugin, EventTypeSecurity:
		return nil
	default:
		return fmt.Errorf("audit event type is invalid")
	}
}

func (t EventType) String() string {
	return string(t)
}
