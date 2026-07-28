package hostservice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/eventoutbox"
	"github.com/tinboxw/skoll/internal/plugin/quota"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type blockingQuotaDataStore struct {
	started chan struct{}
	release chan struct{}
}

func (s *blockingQuotaDataStore) Query(context.Context, pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	close(s.started)
	<-s.release
	return pluginsdk.DataPage{}, nil
}

func (s *blockingQuotaDataStore) Mutate(context.Context, pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	return pluginsdk.DataMutationResult{}, nil
}

func (s *blockingQuotaDataStore) Aggregate(context.Context, pluginsdk.DataAggregateQuery) (pluginsdk.DataAggregatePage, error) {
	return pluginsdk.DataAggregatePage{}, nil
}

func TestQuotaDataStoreBackpressuresConcurrentQueriesAndRecovers(t *testing.T) {
	policy := hostServiceTestQuotaPolicy()
	policy.Query.MaxConcurrent = 1
	controller, err := quota.NewController(policy)
	if err != nil {
		t.Fatal(err)
	}
	base := &blockingQuotaDataStore{started: make(chan struct{}), release: make(chan struct{})}
	service := &quotaDataStore{DataStoreService: base, pluginID: "reports", quotas: controller}
	done := make(chan error, 1)
	go func() {
		_, queryErr := service.Query(context.Background(), pluginsdk.DataQuery{})
		done <- queryErr
	}()
	<-base.started
	if _, err = service.Query(context.Background(), pluginsdk.DataQuery{}); !quotaErrorFor(err, quota.ResourceQuery) {
		t.Fatalf("concurrent query was not rejected: %v", err)
	}
	close(base.release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}

type quotaJobMemory struct {
	mu    sync.Mutex
	items map[string]pluginsdk.Job
}

func (s *quotaJobMemory) Schedule(_ context.Context, input pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.items[input.ID]; ok {
		return item, nil
	}
	item := pluginsdk.Job{ID: input.ID, Kind: input.Kind, Status: pluginsdk.JobStatusScheduled}
	s.items[input.ID] = item
	return item, nil
}

func (s *quotaJobMemory) LeaseDue(context.Context, pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	return nil, nil
}
func (s *quotaJobMemory) Complete(context.Context, pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, nil
}
func (s *quotaJobMemory) Fail(context.Context, pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, nil
}
func (s *quotaJobMemory) Get(_ context.Context, id string) (pluginsdk.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return pluginsdk.Job{}, jobsvc.ErrNotFound
	}
	return item, nil
}
func (s *quotaJobMemory) List(_ context.Context, query pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]pluginsdk.Job, 0)
	for _, item := range s.items {
		if query.Status == "" || item.Status == query.Status {
			out = append(out, item)
		}
	}
	return out, nil
}

func TestQuotaJobServiceBoundsPendingJobsAndAllowsIdempotentReplay(t *testing.T) {
	policy := hostServiceTestQuotaPolicy()
	policy.MaxPendingJobs = 1
	controller, err := quota.NewController(policy)
	if err != nil {
		t.Fatal(err)
	}
	base := &quotaJobMemory{items: map[string]pluginsdk.Job{
		"existing": {ID: "existing", Status: pluginsdk.JobStatusScheduled},
	}}
	service := &quotaJobService{JobService: base, pluginID: "reports", quotas: controller}
	if _, err = service.Schedule(context.Background(), pluginsdk.JobScheduleInput{ID: "new"}); !quotaErrorFor(err, quota.ResourceJob) {
		t.Fatalf("new job exceeded pending quota without rejection: %v", err)
	}
	if _, err = service.Schedule(context.Background(), pluginsdk.JobScheduleInput{ID: "existing"}); err != nil {
		t.Fatalf("idempotent replay must remain available at capacity: %v", err)
	}
}

type quotaOutboxMemory struct {
	items map[string]eventoutbox.Record
}

func (s *quotaOutboxMemory) Enqueue(_ context.Context, item eventoutbox.Record) (eventoutbox.Record, bool, error) {
	if existing, ok := s.items[item.Envelope.ID]; ok {
		return existing, false, nil
	}
	s.items[item.Envelope.ID] = item
	return item, true, nil
}
func (s *quotaOutboxMemory) Get(_ context.Context, id string) (eventoutbox.Record, error) {
	item, ok := s.items[id]
	if !ok {
		return eventoutbox.Record{}, errors.New("event not found")
	}
	return item, nil
}
func (s *quotaOutboxMemory) LeaseDue(context.Context, eventoutbox.LeaseInput) ([]eventoutbox.Record, error) {
	return nil, nil
}
func (s *quotaOutboxMemory) Ack(context.Context, eventoutbox.AckInput) (eventoutbox.Record, error) {
	return eventoutbox.Record{}, nil
}
func (s *quotaOutboxMemory) Fail(context.Context, eventoutbox.FailInput) (eventoutbox.Record, error) {
	return eventoutbox.Record{}, nil
}
func (s *quotaOutboxMemory) Replay(context.Context, string, time.Time) (eventoutbox.Record, error) {
	return eventoutbox.Record{}, nil
}
func (s *quotaOutboxMemory) List(_ context.Context, filter eventoutbox.Filter) ([]eventoutbox.Record, error) {
	out := make([]eventoutbox.Record, 0)
	for _, item := range s.items {
		if (filter.Publisher == "" || item.Envelope.Publisher == filter.Publisher) &&
			(filter.Status == "" || item.Status == filter.Status) {
			out = append(out, item)
		}
	}
	return out, nil
}

type quotaEventBase struct {
	published int
}

func (s *quotaEventBase) Publish(_ context.Context, publication pluginsdk.EventPublication) (pluginsdk.EventEnvelope, error) {
	s.published++
	return pluginsdk.EventEnvelope{ID: quotaEventID("reports", publication.IdempotencyKey)}, nil
}

func TestQuotaEventServiceBoundsPersistentBacklog(t *testing.T) {
	policy := hostServiceTestQuotaPolicy()
	policy.MaxPendingEvents = 1
	controller, err := quota.NewController(policy)
	if err != nil {
		t.Fatal(err)
	}
	existingID := quotaEventID("reports", "existing")
	store := &quotaOutboxMemory{items: map[string]eventoutbox.Record{
		existingID: {Envelope: pluginsdk.EventEnvelope{ID: existingID, Publisher: "reports"}, Status: eventoutbox.StatusPending},
	}}
	service := &quotaEventService{EventService: &quotaEventBase{}, pluginID: "reports", quotas: controller, store: store}
	if _, err = service.Publish(context.Background(), pluginsdk.EventPublication{IdempotencyKey: "new"}); !quotaErrorFor(err, quota.ResourceEvent) {
		t.Fatalf("event backlog quota was not enforced: %v", err)
	}
	if _, err = service.Publish(context.Background(), pluginsdk.EventPublication{IdempotencyKey: "existing"}); err != nil {
		t.Fatalf("idempotent event replay must remain available at capacity: %v", err)
	}
}

type quotaFileMemory struct {
	pluginsdk.FileService
	items []pluginsdk.FileObject
}

func (s *quotaFileMemory) Store(_ context.Context, input pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	item := pluginsdk.FileObject{ID: input.Key, Size: int64(len(input.Content))}
	s.items = append(s.items, item)
	return item, nil
}

func (s *quotaFileMemory) List(_ context.Context, query pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	if query.Offset >= len(s.items) {
		return []pluginsdk.FileObject{}, nil
	}
	end := min(query.Offset+query.Limit, len(s.items))
	return append([]pluginsdk.FileObject(nil), s.items[query.Offset:end]...), nil
}

func TestQuotaFileServiceEnforcesFileAndTotalStorageCapacity(t *testing.T) {
	policy := hostServiceTestQuotaPolicy()
	policy.MaxFileBytes = 512
	policy.MaxStorageBytes = 1024
	controller, err := quota.NewController(policy)
	if err != nil {
		t.Fatal(err)
	}
	base := &quotaFileMemory{items: []pluginsdk.FileObject{{ID: "existing", Size: 900}}}
	service := &quotaFileService{FileService: base, pluginID: "reports", quotas: controller}
	if _, err = service.Store(context.Background(), pluginsdk.FileWrite{Key: "too-large", Content: make([]byte, 513)}); !quotaErrorFor(err, quota.ResourceStorage) {
		t.Fatalf("per-file quota was not enforced: %v", err)
	}
	if _, err = service.Store(context.Background(), pluginsdk.FileWrite{Key: "total", Content: make([]byte, 125)}); !quotaErrorFor(err, quota.ResourceStorage) {
		t.Fatalf("total storage quota was not enforced: %v", err)
	}
	if _, err = service.Store(context.Background(), pluginsdk.FileWrite{Key: "fits", Content: make([]byte, 124)}); err != nil {
		t.Fatalf("storage within quota was rejected: %v", err)
	}
}

func quotaErrorFor(err error, resource quota.Resource) bool {
	var quotaErr *quota.Error
	return errors.As(err, &quotaErr) && quotaErr.Resource == resource
}
