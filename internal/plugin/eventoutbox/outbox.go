package eventoutbox

import (
	"context"
	"errors"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

var ErrIdentityConflict = errors.New("plugin event identity conflicts with another publication")

type Status string

const (
	StatusPending Status = "pending"
)

type Record struct {
	Envelope       pluginsdk.EventEnvelope
	IdempotencyKey string
	RequestHash    string
	Status         Status
	CreatedAt      time.Time
}

type Store interface {
	Enqueue(context.Context, Record) (Record, bool, error)
	Get(context.Context, string) (Record, error)
}
