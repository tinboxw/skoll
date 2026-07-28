package job

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	if repo == nil {
		panic("job repository is required")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{repo: repo, now: now}
}

func (s *Service) Schedule(ctx context.Context, in ScheduleInput) (Job, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.Namespace = strings.TrimSpace(in.Namespace)
	in.Kind = strings.TrimSpace(in.Kind)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.CorrelationID = strings.TrimSpace(in.CorrelationID)
	if in.ID == "" || in.Namespace == "" || in.Kind == "" {
		return Job{}, fmt.Errorf("job id, namespace, and kind are required")
	}
	if in.MaxAttempts <= 0 {
		return Job{}, fmt.Errorf("job max attempts must be positive")
	}
	payload := append([]byte(nil), in.Payload...)
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	if !json.Valid(payload) {
		return Job{}, fmt.Errorf("job payload must be valid json")
	}
	now := s.now().UTC()
	runAt := in.RunAt.UTC()
	if in.RunAt.IsZero() {
		runAt = now
	}
	candidate := Job{
		ID: in.ID, Namespace: in.Namespace, Kind: in.Kind, IdempotencyKey: in.IdempotencyKey,
		CorrelationID: in.CorrelationID, Payload: payload, Status: StatusScheduled, RunAt: runAt, MaxAttempts: in.MaxAttempts,
		CreatedAt: now, UpdatedAt: now,
	}
	stored, created, err := s.repo.Schedule(ctx, candidate)
	if err != nil {
		return Job{}, err
	}
	if !created && !sameSchedule(stored, candidate) {
		return Job{}, ErrConflict
	}
	return cloneJob(stored), nil
}

func (s *Service) LeaseDue(ctx context.Context, in LeaseInput) ([]Job, error) {
	in.Namespace = strings.TrimSpace(in.Namespace)
	in.WorkerID = strings.TrimSpace(in.WorkerID)
	if in.Namespace == "" || in.WorkerID == "" {
		return nil, fmt.Errorf("job namespace and worker id are required")
	}
	if in.Limit <= 0 || in.Limit > 100 {
		return nil, fmt.Errorf("job lease limit must be between 1 and 100")
	}
	if in.LeaseDuration <= 0 {
		return nil, fmt.Errorf("job lease duration must be positive")
	}
	now := s.now().UTC()
	items, err := s.repo.LeaseDue(ctx, in.Namespace, in.WorkerID, now, now.Add(in.LeaseDuration), in.Limit)
	return cloneJobs(items), err
}

func (s *Service) Complete(ctx context.Context, in CompleteInput) (Job, error) {
	in.JobID = strings.TrimSpace(in.JobID)
	in.LeaseToken = strings.TrimSpace(in.LeaseToken)
	if in.JobID == "" || in.LeaseToken == "" {
		return Job{}, fmt.Errorf("job id and lease token are required")
	}
	if len(in.Result) > 0 && !json.Valid(in.Result) {
		return Job{}, fmt.Errorf("job result must be valid json")
	}
	item, err := s.repo.Complete(ctx, in.JobID, in.LeaseToken, append([]byte(nil), in.Result...), s.now().UTC())
	return cloneJob(item), err
}

func (s *Service) Fail(ctx context.Context, in FailInput) (Job, error) {
	in.JobID = strings.TrimSpace(in.JobID)
	in.LeaseToken = strings.TrimSpace(in.LeaseToken)
	in.Error = strings.TrimSpace(in.Error)
	if in.JobID == "" || in.LeaseToken == "" || in.Error == "" {
		return Job{}, fmt.Errorf("job id, lease token, and error are required")
	}
	if in.RetryAfter < 0 {
		return Job{}, fmt.Errorf("job retry delay cannot be negative")
	}
	now := s.now().UTC()
	item, err := s.repo.Fail(ctx, in.JobID, in.LeaseToken, in.Error, now.Add(in.RetryAfter), now)
	return cloneJob(item), err
}

func (s *Service) Get(ctx context.Context, id string) (Job, error) {
	item, err := s.repo.Get(ctx, strings.TrimSpace(id))
	return cloneJob(item), err
}

func (s *Service) List(ctx context.Context, filter Filter) ([]Job, error) {
	filter.Namespace = strings.TrimSpace(filter.Namespace)
	filter.Kind = strings.TrimSpace(filter.Kind)
	if filter.Limit < 0 || filter.Limit > 500 {
		return nil, fmt.Errorf("job list limit must be between 0 and 500")
	}
	items, err := s.repo.List(ctx, filter)
	return cloneJobs(items), err
}

func sameSchedule(left, right Job) bool {
	sameIdentity := left.ID == right.ID || right.IdempotencyKey != "" && left.Namespace == right.Namespace && left.IdempotencyKey == right.IdempotencyKey
	return sameIdentity && left.Namespace == right.Namespace && left.Kind == right.Kind &&
		left.IdempotencyKey == right.IdempotencyKey && bytes.Equal(left.Payload, right.Payload) &&
		left.RunAt.Equal(right.RunAt) && left.MaxAttempts == right.MaxAttempts
}

func cloneJob(item Job) Job {
	item.Payload = append(json.RawMessage(nil), item.Payload...)
	item.Result = append(json.RawMessage(nil), item.Result...)
	if item.LeaseExpiresAt != nil {
		value := *item.LeaseExpiresAt
		item.LeaseExpiresAt = &value
	}
	if item.CompletedAt != nil {
		value := *item.CompletedAt
		item.CompletedAt = &value
	}
	if item.DeadLetteredAt != nil {
		value := *item.DeadLetteredAt
		item.DeadLetteredAt = &value
	}
	return item
}

func cloneJobs(items []Job) []Job {
	out := make([]Job, len(items))
	for i := range items {
		out[i] = cloneJob(items[i])
	}
	return out
}
