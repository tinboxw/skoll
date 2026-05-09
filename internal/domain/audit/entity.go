package audit

import (
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type Entry struct {
	ID        shared.ID
	ActorID   shared.ID
	Action    string
	Resource  string
	Result    string
	Detail    string
	CreatedAt time.Time
}
