package job

import (
	"context"
	"time"
)

type Repository interface {
	Schedule(ctx context.Context, item Job) (stored Job, created bool, err error)
	LeaseDue(ctx context.Context, namespace, workerID string, now, leaseUntil time.Time, limit int) ([]Job, error)
	Complete(ctx context.Context, id, leaseToken string, result []byte, now time.Time) (Job, error)
	Fail(ctx context.Context, id, leaseToken, message string, retryAt, now time.Time) (Job, error)
	Get(ctx context.Context, id string) (Job, error)
	List(ctx context.Context, filter Filter) ([]Job, error)
}
