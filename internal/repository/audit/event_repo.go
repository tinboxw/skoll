package audit

import (
	"context"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type EventFilter struct {
	Type         domainaudit.EventType
	ActorID      shared.ID
	Action       domainaudit.AuditAction
	ResourceType string
	ResourceID   string
	Result       domainaudit.EventResult
	Risk         domainaudit.EventRisk
	TimeRange    shared.TimeRange
	Offset       int
	Limit        int
}

type EventSourceData struct {
	EventID    shared.ID
	SourceData map[string]any
}

type EventRepository interface {
	AppendEvent(ctx context.Context, event *domainaudit.Event) error
	GetEventByID(ctx context.Context, id shared.ID) (*domainaudit.Event, error)
	ListEvents(ctx context.Context, filter EventFilter) ([]*domainaudit.Event, error)
	ExportEventSourceData(ctx context.Context, filter EventFilter) ([]EventSourceData, error)
}
