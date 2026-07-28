package hostservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tinboxw/skoll/internal/plugin/eventoutbox"
	"github.com/tinboxw/skoll/internal/plugin/quota"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type quotaDataStore struct {
	pluginsdk.DataStoreService
	pluginID string
	quotas   *quota.Controller
}

func (s *quotaDataStore) Query(ctx context.Context, query pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	lease, err := s.quotas.Acquire(s.pluginID, quota.ResourceQuery)
	if err != nil {
		return pluginsdk.DataPage{}, err
	}
	defer lease.Release()
	return s.DataStoreService.Query(ctx, query)
}

func (s *quotaDataStore) Aggregate(ctx context.Context, query pluginsdk.DataAggregateQuery) (pluginsdk.DataAggregatePage, error) {
	lease, err := s.quotas.Acquire(s.pluginID, quota.ResourceQuery)
	if err != nil {
		return pluginsdk.DataAggregatePage{}, err
	}
	defer lease.Release()
	return s.DataStoreService.Aggregate(ctx, query)
}

func (s *quotaDataStore) Mutate(ctx context.Context, mutation pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	lease, err := s.quotas.Acquire(s.pluginID, quota.ResourceMutation)
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	defer lease.Release()
	return s.DataStoreService.Mutate(ctx, mutation)
}

type quotaEventService struct {
	pluginsdk.EventService
	pluginID string
	quotas   *quota.Controller
	store    eventoutbox.Store
}

func (s *quotaEventService) Publish(ctx context.Context, publication pluginsdk.EventPublication) (pluginsdk.EventEnvelope, error) {
	lease, err := s.quotas.Acquire(s.pluginID, quota.ResourceEvent)
	if err != nil {
		return pluginsdk.EventEnvelope{}, err
	}
	defer lease.Release()

	eventID := quotaEventID(s.pluginID, publication.IdempotencyKey)
	if _, getErr := s.store.Get(ctx, eventID); getErr == nil {
		return s.EventService.Publish(ctx, publication)
	}
	pending, err := pendingEventCount(ctx, s.store, s.pluginID, s.quotas.Policy().MaxPendingEvents)
	if err != nil {
		return pluginsdk.EventEnvelope{}, err
	}
	reservation, err := s.quotas.Reserve(s.pluginID, quota.ResourceEvent, int64(pending), 1, int64(s.quotas.Policy().MaxPendingEvents))
	if err != nil {
		return pluginsdk.EventEnvelope{}, err
	}
	defer reservation.Release()
	return s.EventService.Publish(ctx, publication)
}

func quotaEventID(pluginID, idempotencyKey string) string {
	identityHash := sha256.Sum256([]byte(pluginID + "\x00" + idempotencyKey))
	return "event-" + hex.EncodeToString(identityHash[:])
}

func pendingEventCount(ctx context.Context, store eventoutbox.Store, pluginID string, maximum int) (int, error) {
	total := 0
	for _, status := range []eventoutbox.Status{eventoutbox.StatusPending, eventoutbox.StatusRunning, eventoutbox.StatusRetryWait} {
		items, err := store.List(ctx, eventoutbox.Filter{Publisher: pluginID, Status: status, Limit: maximum + 1})
		if err != nil {
			return 0, err
		}
		total += len(items)
		if total >= maximum {
			return total, nil
		}
	}
	return total, nil
}

type quotaJobService struct {
	pluginsdk.JobService
	pluginID string
	quotas   *quota.Controller
}

func (s *quotaJobService) Schedule(ctx context.Context, input pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	lease, err := s.quotas.Acquire(s.pluginID, quota.ResourceJob)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	defer lease.Release()
	if _, getErr := s.JobService.Get(ctx, input.ID); getErr == nil {
		return s.JobService.Schedule(ctx, input)
	} else if !errors.Is(getErr, jobsvc.ErrNotFound) {
		return pluginsdk.Job{}, getErr
	}
	maximum := s.quotas.Policy().MaxPendingJobs
	pending, err := pendingJobCount(ctx, s.JobService, maximum)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	reservation, err := s.quotas.Reserve(s.pluginID, quota.ResourceJob, int64(pending), 1, int64(maximum))
	if err != nil {
		return pluginsdk.Job{}, err
	}
	defer reservation.Release()
	return s.JobService.Schedule(ctx, input)
}

func pendingJobCount(ctx context.Context, jobs pluginsdk.JobService, maximum int) (int, error) {
	total := 0
	for _, status := range []pluginsdk.JobStatus{pluginsdk.JobStatusScheduled, pluginsdk.JobStatusRunning, pluginsdk.JobStatusRetryWait} {
		items, err := jobs.List(ctx, pluginsdk.JobQuery{Status: status, Limit: maximum + 1})
		if err != nil {
			return 0, err
		}
		total += len(items)
		if total >= maximum {
			return total, nil
		}
	}
	return total, nil
}

type quotaDocumentService struct {
	pluginsdk.DocumentService
	pluginID string
	quotas   *quota.Controller
}

func (s *quotaDocumentService) Export(ctx context.Context, input pluginsdk.DocumentExportInput) (pluginsdk.Job, error) {
	lease, err := s.quotas.Acquire(s.pluginID, quota.ResourceExport)
	if err != nil {
		return pluginsdk.Job{}, err
	}
	defer lease.Release()
	return s.DocumentService.Export(ctx, input)
}

type quotaFileService struct {
	pluginsdk.FileService
	pluginID string
	quotas   *quota.Controller
}

func (s *quotaFileService) Store(ctx context.Context, input pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	policy := s.quotas.Policy()
	size := int64(len(input.Content))
	if size > policy.MaxFileBytes {
		return pluginsdk.FileObject{}, &quota.Error{PluginID: s.pluginID, Resource: quota.ResourceStorage}
	}
	lease, err := s.quotas.Acquire(s.pluginID, quota.ResourceStorage)
	if err != nil {
		return pluginsdk.FileObject{}, err
	}
	defer lease.Release()
	current, err := pluginStorageBytes(ctx, s.FileService, policy.MaxStorageBytes)
	if err != nil {
		return pluginsdk.FileObject{}, err
	}
	reservation, err := s.quotas.Reserve(s.pluginID, quota.ResourceStorage, current, size, policy.MaxStorageBytes)
	if err != nil {
		return pluginsdk.FileObject{}, err
	}
	defer reservation.Release()
	return s.FileService.Store(ctx, input)
}

func pluginStorageBytes(ctx context.Context, files pluginsdk.FileService, maximum int64) (int64, error) {
	var total int64
	for offset := 0; ; offset += 100 {
		items, err := files.List(ctx, pluginsdk.FileQuery{Offset: offset, Limit: 100})
		if err != nil {
			return 0, err
		}
		for _, item := range items {
			if item.Size < 0 {
				return 0, fmt.Errorf("plugin file size is invalid")
			}
			total += item.Size
			if total >= maximum {
				return total, nil
			}
		}
		if len(items) < 100 {
			return total, nil
		}
	}
}

func wrapHostServicesWithQuotas(host pluginsdk.HostServices, quotas *quota.Controller, outbox eventoutbox.Store) (pluginsdk.HostServices, error) {
	if quotas == nil || outbox == nil {
		return pluginsdk.HostServices{}, fmt.Errorf("plugin host quota dependencies are required")
	}
	pluginID := host.PluginID
	host.DataStore = &quotaDataStore{DataStoreService: host.DataStore, pluginID: pluginID, quotas: quotas}
	host.Events = &quotaEventService{EventService: host.Events, pluginID: pluginID, quotas: quotas, store: outbox}
	host.Jobs = &quotaJobService{JobService: host.Jobs, pluginID: pluginID, quotas: quotas}
	host.Documents = &quotaDocumentService{DocumentService: host.Documents, pluginID: pluginID, quotas: quotas}
	host.Files = &quotaFileService{FileService: host.Files, pluginID: pluginID, quotas: quotas}
	return host, nil
}

var (
	_ pluginsdk.DataStoreService = (*quotaDataStore)(nil)
	_ pluginsdk.EventService     = (*quotaEventService)(nil)
	_ pluginsdk.JobService       = (*quotaJobService)(nil)
	_ pluginsdk.DocumentService  = (*quotaDocumentService)(nil)
	_ pluginsdk.FileService      = (*quotaFileService)(nil)
)
