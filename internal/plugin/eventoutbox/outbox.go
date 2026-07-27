package eventoutbox

import (
	"context"
	"errors"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

var (
	ErrIdentityConflict = errors.New("plugin event identity conflicts with another publication")
	ErrLeaseLost        = errors.New("plugin event outbox lease was lost")
	ErrNotDeadLetter    = errors.New("plugin event is not dead-lettered")
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusRunning    Status = "running"
	StatusRetryWait  Status = "retry_wait"
	StatusSucceeded  Status = "succeeded"
	StatusDeadLetter Status = "dead_letter"
)

type Record struct {
	Envelope       pluginsdk.EventEnvelope
	IdempotencyKey string
	RequestHash    string
	Status         Status
	AttemptCount   int
	MaxAttempts    int
	NextAttemptAt  time.Time
	LeaseOwner     string
	LeaseToken     string
	LeaseExpiresAt *time.Time
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeliveredAt    *time.Time
	DeadLetteredAt *time.Time
}

type LeaseInput struct {
	WorkerID      string
	Limit         int
	Now           time.Time
	LeaseDuration time.Duration
}

type AckInput struct {
	EventID    string
	LeaseToken string
	Now        time.Time
}

type FailInput struct {
	EventID    string
	LeaseToken string
	Error      string
	RetryAt    time.Time
	Now        time.Time
}

type Filter struct {
	Publisher string
	Status    Status
	Limit     int
}

type Store interface {
	Enqueue(context.Context, Record) (Record, bool, error)
	Get(context.Context, string) (Record, error)
	LeaseDue(context.Context, LeaseInput) ([]Record, error)
	Ack(context.Context, AckInput) (Record, error)
	Fail(context.Context, FailInput) (Record, error)
	Replay(context.Context, string, time.Time) (Record, error)
	List(context.Context, Filter) ([]Record, error)
}
