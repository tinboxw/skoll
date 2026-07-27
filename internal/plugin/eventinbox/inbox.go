package eventinbox

import (
	"context"
	"errors"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

var ErrLeaseLost = errors.New("plugin event inbox lease was lost")

type Status string

const (
	StatusProcessing Status = "processing"
	StatusSucceeded  Status = "succeeded"
	StatusFailed     Status = "failed"
)

type Record struct {
	Delivery       pluginsdk.EventDelivery
	Status         Status
	AttemptCount   int
	LeaseOwner     string
	LeaseToken     string
	LeaseExpiresAt *time.Time
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ProcessedAt    *time.Time
}

type ClaimInput struct {
	Delivery      pluginsdk.EventDelivery
	WorkerID      string
	Now           time.Time
	LeaseDuration time.Duration
}

type TransitionInput struct {
	DeliveryID string
	LeaseToken string
	Error      string
	Now        time.Time
}

type Store interface {
	Claim(context.Context, ClaimInput) (Record, bool, error)
	Complete(context.Context, TransitionInput) (Record, error)
	Fail(context.Context, TransitionInput) (Record, error)
	Get(context.Context, string) (Record, error)
}
