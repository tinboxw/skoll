package hostservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const maxPluginJobPayloadBytes = 1024 * 1024

type jobBackend interface {
	Schedule(context.Context, jobsvc.ScheduleInput) (jobsvc.Job, error)
	LeaseDue(context.Context, jobsvc.LeaseInput) ([]jobsvc.Job, error)
	Complete(context.Context, jobsvc.CompleteInput) (jobsvc.Job, error)
	Fail(context.Context, jobsvc.FailInput) (jobsvc.Job, error)
	Get(context.Context, string) (jobsvc.Job, error)
	List(context.Context, jobsvc.Filter) ([]jobsvc.Job, error)
}

type jobService struct {
	pluginID  string
	namespace string
	jobs      jobBackend
	audit     pluginsdk.AuditService
}

func NewJobService(pluginID string, jobs jobBackend, audit pluginsdk.AuditService) (pluginsdk.JobService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" {
		return nil, fmt.Errorf("plugin host identity is required")
	}
	if jobs == nil || audit == nil {
		return nil, fmt.Errorf("plugin host job dependencies are required")
	}
	return &jobService{pluginID: pluginID, namespace: "plugin." + pluginID, jobs: jobs, audit: audit}, nil
}

func (s *jobService) Schedule(ctx context.Context, input pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	id, err := s.boundID(input.ID)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	kind, err := validatePluginName(input.Kind, "job kind")
	if err != nil {
		return pluginsdk.Job{}, err
	}
	if len(input.Payload) > maxPluginJobPayloadBytes {
		return pluginsdk.Job{}, fmt.Errorf("plugin job payload exceeds %d bytes", maxPluginJobPayloadBytes)
	}
	if key := strings.TrimSpace(input.IdempotencyKey); len(key) > 256 {
		return pluginsdk.Job{}, fmt.Errorf("plugin job idempotency key is too long")
	}
	correlationID := ""
	if operation, ok := pluginsdk.OperationContextFromContext(ctx); ok {
		correlationID = operation.CorrelationID
	}
	item, err := s.jobs.Schedule(ctx, jobsvc.ScheduleInput{
		ID: id, Namespace: s.namespace, Kind: kind, IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
		CorrelationID: correlationID, Payload: append(json.RawMessage(nil), input.Payload...), RunAt: input.RunAt, MaxAttempts: input.MaxAttempts,
	})
	if err != nil {
		return pluginsdk.Job{}, err
	}
	if err := s.record(ctx, "job.schedule", input.ID, map[string]any{"kind": kind}); err != nil {
		return pluginsdk.Job{}, err
	}
	return s.job(item)
}

func (s *jobService) LeaseDue(ctx context.Context, input pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	workerID, err := validatePluginLocalID(input.WorkerID, "job worker")
	if err != nil {
		return nil, err
	}
	items, err := s.jobs.LeaseDue(ctx, jobsvc.LeaseInput{
		Namespace: s.namespace, WorkerID: s.workerPrefix() + workerID,
		Limit: input.Limit, LeaseDuration: input.LeaseDuration,
	})
	if err != nil {
		return nil, err
	}
	out := make([]pluginsdk.Job, 0, len(items))
	for _, item := range items {
		converted, convertErr := s.job(item)
		if convertErr != nil {
			return nil, convertErr
		}
		out = append(out, converted)
	}
	return out, nil
}

func (s *jobService) Complete(ctx context.Context, input pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	id, err := s.boundID(input.JobID)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	if len(input.Result) > maxPluginJobPayloadBytes {
		return pluginsdk.Job{}, fmt.Errorf("plugin job result exceeds %d bytes", maxPluginJobPayloadBytes)
	}
	item, err := s.jobs.Complete(ctx, jobsvc.CompleteInput{
		JobID: id, LeaseToken: strings.TrimSpace(input.LeaseToken), Result: append(json.RawMessage(nil), input.Result...),
	})
	if err != nil {
		return pluginsdk.Job{}, err
	}
	if err := s.record(ctx, "job.complete", input.JobID, map[string]any{"kind": item.Kind}); err != nil {
		return pluginsdk.Job{}, err
	}
	return s.job(item)
}

func (s *jobService) Fail(ctx context.Context, input pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	id, err := s.boundID(input.JobID)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	item, err := s.jobs.Fail(ctx, jobsvc.FailInput{
		JobID: id, LeaseToken: strings.TrimSpace(input.LeaseToken), Error: strings.TrimSpace(input.Error), RetryAfter: input.RetryAfter,
	})
	if err != nil {
		return pluginsdk.Job{}, err
	}
	if err := s.record(ctx, "job.fail", input.JobID, map[string]any{"kind": item.Kind, "status": item.Status}); err != nil {
		return pluginsdk.Job{}, err
	}
	return s.job(item)
}

func (s *jobService) Get(ctx context.Context, id string) (pluginsdk.Job, error) {
	bound, err := s.boundID(id)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	item, err := s.jobs.Get(ctx, bound)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	return s.job(item)
}

func (s *jobService) List(ctx context.Context, query pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	kind := strings.TrimSpace(query.Kind)
	var err error
	if kind != "" {
		kind, err = validatePluginName(kind, "job kind")
		if err != nil {
			return nil, err
		}
	}
	items, err := s.jobs.List(ctx, jobsvc.Filter{
		Namespace: s.namespace, Kind: kind, Status: jobsvc.Status(query.Status), Limit: query.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]pluginsdk.Job, 0, len(items))
	for _, item := range items {
		converted, convertErr := s.job(item)
		if convertErr != nil {
			return nil, convertErr
		}
		out = append(out, converted)
	}
	return out, nil
}

func (s *jobService) boundID(value string) (string, error) {
	value, err := validatePluginLocalID(value, "job")
	if err != nil {
		return "", err
	}
	return "plugin:" + s.pluginID + ":" + value, nil
}

func (s *jobService) localID(value string) (string, error) {
	prefix := "plugin:" + s.pluginID + ":"
	if !strings.HasPrefix(value, prefix) {
		return "", fmt.Errorf("job is outside plugin namespace")
	}
	return validatePluginLocalID(strings.TrimPrefix(value, prefix), "job")
}

func (s *jobService) workerPrefix() string {
	return "plugin:" + s.pluginID + ":worker:"
}

func (s *jobService) job(item jobsvc.Job) (pluginsdk.Job, error) {
	if item.Namespace != s.namespace {
		return pluginsdk.Job{}, fmt.Errorf("job is outside plugin namespace")
	}
	id, err := s.localID(item.ID)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	leaseOwner := ""
	if item.LeaseOwner != "" {
		if !strings.HasPrefix(item.LeaseOwner, s.workerPrefix()) {
			return pluginsdk.Job{}, fmt.Errorf("job lease owner is outside plugin namespace")
		}
		leaseOwner = strings.TrimPrefix(item.LeaseOwner, s.workerPrefix())
	}
	return pluginsdk.Job{
		ID: id, Kind: item.Kind, IdempotencyKey: item.IdempotencyKey, CorrelationID: item.CorrelationID,
		Payload: append(json.RawMessage(nil), item.Payload...), Status: pluginsdk.JobStatus(item.Status),
		RunAt: item.RunAt, MaxAttempts: item.MaxAttempts, AttemptCount: item.AttemptCount,
		LeaseOwner: leaseOwner, LeaseToken: item.LeaseToken, LeaseExpiresAt: cloneTime(item.LeaseExpiresAt),
		LastError: item.LastError, Result: append(json.RawMessage(nil), item.Result...),
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, CompletedAt: cloneTime(item.CompletedAt), DeadLetteredAt: cloneTime(item.DeadLetteredAt),
	}, nil
}

func (s *jobService) record(ctx context.Context, action, id string, detail map[string]any) error {
	_, err := s.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: action, Resource: "job", ResourceID: strings.TrimSpace(id), Risk: pluginsdk.AuditRiskMedium, Detail: detail,
	})
	return err
}

var _ pluginsdk.JobService = (*jobService)(nil)
