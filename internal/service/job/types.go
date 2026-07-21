package job

import (
	"encoding/json"
	"errors"
	"time"
)

type Status string

const (
	StatusScheduled  Status = "scheduled"
	StatusRunning    Status = "running"
	StatusRetryWait  Status = "retry_wait"
	StatusSucceeded  Status = "succeeded"
	StatusDeadLetter Status = "dead_letter"
)

var (
	ErrNotFound  = errors.New("job not found")
	ErrConflict  = errors.New("job identity conflicts with persisted job")
	ErrLeaseLost = errors.New("job lease is no longer owned")
)

type Job struct {
	ID             string          `json:"id"`
	Namespace      string          `json:"namespace"`
	Kind           string          `json:"kind"`
	IdempotencyKey string          `json:"idempotencyKey,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	Status         Status          `json:"status"`
	RunAt          time.Time       `json:"runAt"`
	MaxAttempts    int             `json:"maxAttempts"`
	AttemptCount   int             `json:"attemptCount"`
	LeaseOwner     string          `json:"leaseOwner,omitempty"`
	LeaseToken     string          `json:"-"`
	LeaseExpiresAt *time.Time      `json:"leaseExpiresAt,omitempty"`
	LastError      string          `json:"lastError,omitempty"`
	Result         json.RawMessage `json:"result,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	CompletedAt    *time.Time      `json:"completedAt,omitempty"`
	DeadLetteredAt *time.Time      `json:"deadLetteredAt,omitempty"`
}

type ScheduleInput struct {
	ID             string
	Namespace      string
	Kind           string
	IdempotencyKey string
	Payload        json.RawMessage
	RunAt          time.Time
	MaxAttempts    int
}

type LeaseInput struct {
	WorkerID      string
	Limit         int
	LeaseDuration time.Duration
}

type CompleteInput struct {
	JobID      string
	LeaseToken string
	Result     json.RawMessage
}

type FailInput struct {
	JobID      string
	LeaseToken string
	Error      string
	RetryAfter time.Duration
}

type Filter struct {
	Namespace string
	Kind      string
	Status    Status
	Limit     int
}
