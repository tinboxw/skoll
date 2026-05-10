package audit

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

var ErrInvalidTimeRange = errors.New("invalid time range")

type Record struct {
	ID         shared.ID
	ActorID    shared.ID
	Action     string
	Resource   string
	ResourceID string
	Detail     map[string]any
	OccurredAt time.Time
}

func NewRecord(id, actorID shared.ID, action, resource, resourceID string, detail map[string]any, now time.Time) (*Record, error) {
	if id.IsZero() || actorID.IsZero() {
		return nil, fmt.Errorf("record id and actor id are required")
	}
	if strings.TrimSpace(action) == "" || strings.TrimSpace(resource) == "" {
		return nil, fmt.Errorf("action and resource are required")
	}
	if detail == nil {
		detail = map[string]any{}
	}
	return &Record{
		ID:         id,
		ActorID:    actorID,
		Action:     strings.TrimSpace(action),
		Resource:   strings.TrimSpace(resource),
		ResourceID: strings.TrimSpace(resourceID),
		Detail:     detail,
		OccurredAt: now,
	}, nil
}
