package audit

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type EventResult string

const (
	EventResultSuccess EventResult = "success"
	EventResultFailure EventResult = "failure"
	EventResultDenied  EventResult = "denied"
)

type EventRisk string

const (
	EventRiskLow      EventRisk = "low"
	EventRiskMedium   EventRisk = "medium"
	EventRiskHigh     EventRisk = "high"
	EventRiskCritical EventRisk = "critical"
)

type ActorRef struct {
	Type string
	ID   shared.ID
	Name string
}

type ResourceRef struct {
	Type string
	ID   string
	Name string
}

type TraceContext struct {
	TraceID   string
	RequestID string
	Method    string
	Path      string
	IP        string
	UserAgent string
}

type EventInput struct {
	ID         shared.ID
	Type       EventType
	Action     AuditAction
	Actor      ActorRef
	Resource   ResourceRef
	Result     EventResult
	Trace      TraceContext
	Risk       EventRisk
	Metadata   map[string]any
	SourceData map[string]any
	OccurredAt time.Time
}

type Event struct {
	ID         shared.ID
	Type       EventType
	Action     AuditAction
	Actor      ActorRef
	Resource   ResourceRef
	Result     EventResult
	Trace      TraceContext
	Risk       EventRisk
	Metadata   map[string]any
	SourceData map[string]any
	OccurredAt time.Time
}

func NewEvent(in EventInput) (*Event, error) {
	in.Actor = normalizeActor(in.Actor)
	in.Resource = normalizeResource(in.Resource)
	in.Trace = normalizeTrace(in.Trace)
	if in.Risk == "" {
		in.Risk = EventRiskLow
	}
	if err := validateEventInput(in); err != nil {
		return nil, err
	}
	return &Event{
		ID:         in.ID,
		Type:       in.Type,
		Action:     in.Action,
		Actor:      in.Actor,
		Resource:   in.Resource,
		Result:     in.Result,
		Trace:      in.Trace,
		Risk:       in.Risk,
		Metadata:   copyMetadata(in.Metadata),
		SourceData: copyMetadata(in.SourceData),
		OccurredAt: in.OccurredAt,
	}, nil
}

func validateEventInput(in EventInput) error {
	if in.ID.IsZero() {
		return fmt.Errorf("audit event id is required")
	}
	if err := in.Type.Validate(); err != nil {
		return err
	}
	if err := in.Action.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(in.Actor.Type) == "" {
		return fmt.Errorf("audit event actor type is required")
	}
	if strings.TrimSpace(in.Resource.Type) == "" {
		return fmt.Errorf("audit event resource type is required")
	}
	if err := in.Result.Validate(); err != nil {
		return err
	}
	if err := in.Risk.Validate(); err != nil {
		return err
	}
	if in.OccurredAt.IsZero() {
		return fmt.Errorf("audit event occurred time is required")
	}
	return nil
}

func (r EventResult) Validate() error {
	switch r {
	case EventResultSuccess, EventResultFailure, EventResultDenied:
		return nil
	default:
		return fmt.Errorf("audit event result is invalid")
	}
}

func (r EventRisk) Validate() error {
	switch r {
	case EventRiskLow, EventRiskMedium, EventRiskHigh, EventRiskCritical:
		return nil
	default:
		return fmt.Errorf("audit event risk is invalid")
	}
}

func normalizeActor(actor ActorRef) ActorRef {
	return ActorRef{
		Type: strings.TrimSpace(strings.ToLower(actor.Type)),
		ID:   shared.ID(strings.TrimSpace(actor.ID.String())),
		Name: strings.TrimSpace(actor.Name),
	}
}

func normalizeResource(resource ResourceRef) ResourceRef {
	return ResourceRef{
		Type: strings.TrimSpace(strings.ToLower(resource.Type)),
		ID:   strings.TrimSpace(resource.ID),
		Name: strings.TrimSpace(resource.Name),
	}
}

func normalizeTrace(trace TraceContext) TraceContext {
	return TraceContext{
		TraceID:   strings.TrimSpace(trace.TraceID),
		RequestID: strings.TrimSpace(trace.RequestID),
		Method:    strings.TrimSpace(strings.ToUpper(trace.Method)),
		Path:      strings.TrimSpace(trace.Path),
		IP:        strings.TrimSpace(trace.IP),
		UserAgent: strings.TrimSpace(trace.UserAgent),
	}
}

func copyMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		out[normalizedKey] = value
	}
	return out
}
