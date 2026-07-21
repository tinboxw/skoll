package pluginsdk

import (
	"context"
	"encoding/json"
	"time"
)

type JobStatus string

const (
	JobStatusScheduled  JobStatus = "scheduled"
	JobStatusRunning    JobStatus = "running"
	JobStatusRetryWait  JobStatus = "retry_wait"
	JobStatusSucceeded  JobStatus = "succeeded"
	JobStatusDeadLetter JobStatus = "dead_letter"
)

type Job struct {
	ID             string
	Kind           string
	IdempotencyKey string
	Payload        json.RawMessage
	Status         JobStatus
	RunAt          time.Time
	MaxAttempts    int
	AttemptCount   int
	LeaseOwner     string
	LeaseToken     string
	LeaseExpiresAt *time.Time
	LastError      string
	Result         json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
	DeadLetteredAt *time.Time
}

type JobScheduleInput struct {
	ID             string
	Kind           string
	IdempotencyKey string
	Payload        json.RawMessage
	RunAt          time.Time
	MaxAttempts    int
}

type JobLeaseInput struct {
	WorkerID      string
	Limit         int
	LeaseDuration time.Duration
}

type JobCompleteInput struct {
	JobID      string
	LeaseToken string
	Result     json.RawMessage
}

type JobFailInput struct {
	JobID      string
	LeaseToken string
	Error      string
	RetryAfter time.Duration
}

type JobQuery struct {
	Kind   string
	Status JobStatus
	Limit  int
}

type JobService interface {
	Schedule(ctx context.Context, input JobScheduleInput) (Job, error)
	LeaseDue(ctx context.Context, input JobLeaseInput) ([]Job, error)
	Complete(ctx context.Context, input JobCompleteInput) (Job, error)
	Fail(ctx context.Context, input JobFailInput) (Job, error)
	Get(ctx context.Context, id string) (Job, error)
	List(ctx context.Context, query JobQuery) ([]Job, error)
}
