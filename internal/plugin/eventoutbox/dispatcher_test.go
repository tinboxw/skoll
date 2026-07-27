package eventoutbox_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/tinboxw/skoll/internal/plugin/eventoutbox"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDispatcherLeasesEventsAcrossCompetingWorkers(t *testing.T) {
	store, closeDB := openEventOutboxTestStore(t)
	defer closeDB()
	now := time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC)
	const eventCount = 20
	for index := 0; index < eventCount; index++ {
		enqueueEvent(t, store, fmt.Sprintf("event-%02d", index), now, 3)
	}
	deliveries := map[string]int{}
	var mu sync.Mutex
	sink := DeliverySinkFunc(func(_ context.Context, envelope pluginsdk.EventEnvelope) error {
		mu.Lock()
		defer mu.Unlock()
		deliveries[envelope.ID]++
		return nil
	})
	first, err := NewDispatcher(store, sink, DispatcherOptions{LeaseDuration: time.Minute, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("new first dispatcher: %v", err)
	}
	second, err := NewDispatcher(store, sink, DispatcherOptions{LeaseDuration: time.Minute, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("new second dispatcher: %v", err)
	}
	var wg sync.WaitGroup
	var firstErr, secondErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, firstErr = first.DispatchBatch(context.Background(), "worker-a", eventCount)
	}()
	go func() {
		defer wg.Done()
		_, secondErr = second.DispatchBatch(context.Background(), "worker-b", eventCount)
	}()
	wg.Wait()
	if firstErr != nil || secondErr != nil {
		t.Fatalf("competing dispatch failed: first=%v second=%v", firstErr, secondErr)
	}
	if len(deliveries) != eventCount {
		t.Fatalf("delivered %d events, expected %d", len(deliveries), eventCount)
	}
	for id, count := range deliveries {
		if count != 1 {
			t.Fatalf("event %s delivered %d times during one lease window", id, count)
		}
	}
	succeeded, err := store.List(context.Background(), Filter{Status: StatusSucceeded, Limit: eventCount})
	if err != nil || len(succeeded) != eventCount {
		t.Fatalf("succeeded events=%d err=%v", len(succeeded), err)
	}
}

func TestDispatcherRecoversExpiredLeaseAfterRestart(t *testing.T) {
	store, closeDB := openEventOutboxTestStore(t)
	defer closeDB()
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	enqueueEvent(t, store, "restart-event", now, 3)
	leased, err := store.LeaseDue(context.Background(), LeaseInput{
		WorkerID: "stopped-worker", Limit: 1, Now: now, LeaseDuration: time.Second,
	})
	if err != nil || len(leased) != 1 {
		t.Fatalf("initial lease=%d err=%v", len(leased), err)
	}

	restartedStore := gormrepo.NewPluginEventOutboxStore(store.db)
	var delivered atomic.Int64
	dispatcher, err := NewDispatcher(restartedStore, DeliverySinkFunc(func(context.Context, pluginsdk.EventEnvelope) error {
		delivered.Add(1)
		return nil
	}), DispatcherOptions{LeaseDuration: time.Minute, Now: func() time.Time { return now.Add(2 * time.Second) }})
	if err != nil {
		t.Fatalf("new restarted dispatcher: %v", err)
	}
	processed, err := dispatcher.DispatchBatch(context.Background(), "restarted-worker", 1)
	if err != nil || processed != 1 || delivered.Load() != 1 {
		t.Fatalf("restart dispatch processed=%d delivered=%d err=%v", processed, delivered.Load(), err)
	}
	record, err := restartedStore.Get(context.Background(), "restart-event")
	if err != nil || record.Status != StatusSucceeded || record.AttemptCount != 2 {
		t.Fatalf("restart record=%+v err=%v", record, err)
	}
}

func TestDispatcherBoundsPoisonEventsAndReplaysWithAuthorizationAudit(t *testing.T) {
	store, closeDB := openEventOutboxTestStore(t)
	defer closeDB()
	now := time.Date(2026, 7, 27, 11, 0, 0, 0, time.UTC)
	enqueueEvent(t, store, "poison-event", now, 3)
	var failDelivery atomic.Bool
	failDelivery.Store(true)
	current := now
	dispatcher, err := NewDispatcher(store, DeliverySinkFunc(func(context.Context, pluginsdk.EventEnvelope) error {
		if failDelivery.Load() {
			return errors.New("subscriber unavailable")
		}
		return nil
	}), DispatcherOptions{
		LeaseDuration: time.Minute, RetryDelay: time.Second,
		Now: func() time.Time { return current },
	})
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		processed, dispatchErr := dispatcher.DispatchBatch(context.Background(), "worker", 1)
		if dispatchErr != nil || processed != 1 {
			t.Fatalf("attempt %d processed=%d err=%v", attempt, processed, dispatchErr)
		}
		current = current.Add(time.Duration(1<<attempt) * time.Second)
	}
	dead, err := store.Get(context.Background(), "poison-event")
	if err != nil || dead.Status != StatusDeadLetter || dead.AttemptCount != 3 ||
		dead.DeadLetteredAt == nil || dead.LastError != "subscriber unavailable" {
		t.Fatalf("dead letter=%+v err=%v", dead, err)
	}
	if metrics := dispatcher.Metrics(); metrics.Retried != 2 || metrics.DeadLetters != 1 {
		t.Fatalf("unexpected dispatcher metrics: %+v", metrics)
	}

	denied := errors.New("event replay denied")
	deniedReplayer, err := NewReplayer(store, replayAuthorizerStub{err: denied}, &replayAuditStub{}, ReplayOptions{})
	if err != nil {
		t.Fatalf("new denied replayer: %v", err)
	}
	if _, err := deniedReplayer.Replay(context.Background(), "operator", dead.Envelope.ID); !errors.Is(err, denied) {
		t.Fatalf("unauthorized replay error=%v", err)
	}
	stillDead, _ := store.Get(context.Background(), dead.Envelope.ID)
	if stillDead.Status != StatusDeadLetter {
		t.Fatalf("unauthorized replay changed status to %s", stillDead.Status)
	}

	audit := &replayAuditStub{}
	replayer, err := NewReplayer(store, replayAuthorizerStub{}, audit, ReplayOptions{Now: func() time.Time { return current }})
	if err != nil {
		t.Fatalf("new replayer: %v", err)
	}
	replayed, err := replayer.Replay(context.Background(), "operator", dead.Envelope.ID)
	if err != nil || replayed.Status != StatusPending || replayed.AttemptCount != 0 || audit.calls.Load() != 1 {
		t.Fatalf("replayed=%+v audits=%d err=%v", replayed, audit.calls.Load(), err)
	}
	failDelivery.Store(false)
	current = current.Add(time.Second)
	if processed, err := dispatcher.DispatchBatch(context.Background(), "worker", 1); err != nil || processed != 1 {
		t.Fatalf("replayed dispatch processed=%d err=%v", processed, err)
	}
	final, _ := store.Get(context.Background(), dead.Envelope.ID)
	if final.Status != StatusSucceeded {
		t.Fatalf("replayed event status=%s", final.Status)
	}
}

type eventOutboxTestStore struct {
	*gormrepo.PluginEventOutboxStore
	db *gorm.DB
}

func openEventOutboxTestStore(t *testing.T) (*eventOutboxTestStore, func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "event-outbox.db")
	db, err := gorm.Open(sqlite.Open(path+"?_busy_timeout=5000&_journal_mode=WAL"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open event outbox database: %v", err)
	}
	if err := db.AutoMigrate(&gormrepo.PluginEventOutboxModel{}); err != nil {
		t.Fatalf("migrate event outbox database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("unwrap event outbox database: %v", err)
	}
	return &eventOutboxTestStore{PluginEventOutboxStore: gormrepo.NewPluginEventOutboxStore(db), db: db}, func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close event outbox database: %v", err)
		}
	}
}

func enqueueEvent(t *testing.T, store Store, id string, now time.Time, maxAttempts int) {
	t.Helper()
	_, duplicate, err := store.Enqueue(context.Background(), Record{
		Envelope: pluginsdk.EventEnvelope{
			ID: id, Publisher: "warehouse", Name: "inventory.received", SchemaVersion: 1,
			PayloadType: "inventory.received", CorrelationID: "correlation-" + id,
			Payload:    pluginsdk.EventPayload{"quantity": {Type: pluginsdk.DataValueInteger, Value: "1"}},
			OccurredAt: now,
		},
		IdempotencyKey: id, RequestHash: "hash-" + id, Status: StatusPending,
		MaxAttempts: maxAttempts, NextAttemptAt: now, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || duplicate {
		t.Fatalf("enqueue %s duplicate=%t err=%v", id, duplicate, err)
	}
}

type replayAuthorizerStub struct {
	err error
}

func (s replayAuthorizerStub) AuthorizeEventReplay(context.Context, string, Record) error {
	return s.err
}

type replayAuditStub struct {
	calls atomic.Int64
}

func (s *replayAuditStub) RecordEventReplay(context.Context, string, Record) error {
	s.calls.Add(1)
	return nil
}
