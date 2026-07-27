package plugin_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/plugin/eventoutbox"
	"github.com/tinboxw/skoll/internal/plugin/hostservice"
	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReliableCurrentEventInteroperabilityEndToEnd(t *testing.T) {
	db, closeDB := openReliableEventDB(t)
	defer closeDB()
	outbox := gormrepo.NewPluginEventOutboxStore(db)
	inbox := gormrepo.NewPluginEventInboxStore(db)
	now := time.Date(2026, 7, 27, 14, 0, 0, 0, time.UTC)
	consumer := newReliableEventConsumer(t)
	defer consumer.Close()
	catalog := newReliableEventCatalog(consumer.URL)

	publisher, err := hostservice.NewEventService("warehouse", func(pluginID string) ([]pluginsdk.EventPublicationDeclaration, error) {
		info, resolveErr := catalog.Get(pluginID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return info.EventPublications(), nil
	}, outbox, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new event publisher: %v", err)
	}
	uow := storesql.NewUnitOfWorkWithDB(db)
	dispatcher := newReliableEventDispatcher(t, catalog, inbox, outbox, &now)

	first := reliableEventPublication("receive-1")
	firstEnvelope := publishReliableEvent(t, uow, publisher, first, false)
	if processed, err := dispatcher.DispatchBatch(context.Background(), "dispatcher-a", 10); err != nil || processed != 1 {
		t.Fatalf("dispatch first event processed=%d err=%v", processed, err)
	}
	if consumer.effects.Load() != 1 {
		t.Fatalf("first event effects=%d want=1", consumer.effects.Load())
	}
	duplicateEnvelope := publishReliableEvent(t, uow, publisher, first, false)
	if duplicateEnvelope.ID != firstEnvelope.ID {
		t.Fatalf("duplicate event identity changed: %s != %s", duplicateEnvelope.ID, firstEnvelope.ID)
	}
	if processed, err := dispatcher.DispatchBatch(context.Background(), "dispatcher-a", 10); err != nil || processed != 0 {
		t.Fatalf("duplicate dispatch processed=%d err=%v", processed, err)
	}
	if consumer.effects.Load() != 1 {
		t.Fatalf("duplicate event produced %d effects", consumer.effects.Load())
	}

	rolledBack := reliableEventPublication("receive-rollback")
	rollbackEnvelope := publishReliableEvent(t, uow, publisher, rolledBack, true)
	if _, err := outbox.Get(context.Background(), rollbackEnvelope.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("rolled-back event persisted: %v", err)
	}

	consumer.available.Store(false)
	poison := reliableEventPublication("receive-outage")
	poisonEnvelope := publishReliableEvent(t, uow, publisher, poison, false)
	for attempt := 1; attempt <= eventoutbox.DefaultMaxAttempts; attempt++ {
		if processed, err := dispatcher.DispatchBatch(context.Background(), "dispatcher-a", 10); err != nil || processed != 1 {
			t.Fatalf("outage attempt %d processed=%d err=%v", attempt, processed, err)
		}
		now = now.Add(time.Duration(1<<attempt) * time.Second)
	}
	dead, err := outbox.Get(context.Background(), poisonEnvelope.ID)
	if err != nil || dead.Status != eventoutbox.StatusDeadLetter || dead.AttemptCount != eventoutbox.DefaultMaxAttempts {
		t.Fatalf("outage dead letter=%+v err=%v", dead, err)
	}

	replayAudit := &reliableReplayAudit{}
	replayer, err := eventoutbox.NewReplayer(outbox, reliableReplayAuthorizer{}, replayAudit, eventoutbox.ReplayOptions{
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new event replayer: %v", err)
	}
	if _, err := replayer.Replay(context.Background(), "super-admin", poisonEnvelope.ID); err != nil {
		t.Fatalf("replay dead letter: %v", err)
	}
	if replayAudit.calls.Load() != 1 {
		t.Fatalf("replay audits=%d want=1", replayAudit.calls.Load())
	}
	consumer.available.Store(true)

	restartedOutbox := gormrepo.NewPluginEventOutboxStore(db)
	restartedInbox := gormrepo.NewPluginEventInboxStore(db)
	restartedDispatcher := newReliableEventDispatcher(t, catalog, restartedInbox, restartedOutbox, &now)
	if processed, err := restartedDispatcher.DispatchBatch(context.Background(), "dispatcher-restarted", 10); err != nil || processed != 1 {
		t.Fatalf("restart replay dispatch processed=%d err=%v", processed, err)
	}
	if consumer.effects.Load() != 2 {
		t.Fatalf("replayed event effects=%d want=2", consumer.effects.Load())
	}

	catalog.setSubscriberState(plugin.StateDisabled)
	disabledEnvelope := publishReliableEvent(t, uow, publisher, reliableEventPublication("receive-disabled"), false)
	if processed, err := restartedDispatcher.DispatchBatch(context.Background(), "dispatcher-restarted", 10); err != nil || processed != 1 {
		t.Fatalf("disabled dispatch processed=%d err=%v", processed, err)
	}
	disabledRecord, _ := restartedOutbox.Get(context.Background(), disabledEnvelope.ID)
	if disabledRecord.Status != eventoutbox.StatusSucceeded || consumer.effects.Load() != 2 {
		t.Fatalf("disabled subscriber event=%s effects=%d", disabledRecord.Status, consumer.effects.Load())
	}

	catalog.uninstallSubscriber()
	uninstalledEnvelope := publishReliableEvent(t, uow, publisher, reliableEventPublication("receive-uninstalled"), false)
	if processed, err := restartedDispatcher.DispatchBatch(context.Background(), "dispatcher-restarted", 10); err != nil || processed != 1 {
		t.Fatalf("uninstalled dispatch processed=%d err=%v", processed, err)
	}
	uninstalledRecord, _ := restartedOutbox.Get(context.Background(), uninstalledEnvelope.ID)
	if uninstalledRecord.Status != eventoutbox.StatusSucceeded || consumer.effects.Load() != 2 {
		t.Fatalf("uninstalled subscriber event=%s effects=%d", uninstalledRecord.Status, consumer.effects.Load())
	}
}

func newReliableEventDispatcher(
	t *testing.T,
	catalog *reliableEventCatalog,
	inbox *gormrepo.PluginEventInboxStore,
	outbox *gormrepo.PluginEventOutboxStore,
	now *time.Time,
) *eventoutbox.Dispatcher {
	t.Helper()
	router, err := plugin.NewCurrentEventRouter(
		catalog, inbox, plugin.NewHTTPEventDeliveryClient(nil, time.Second),
		plugin.CurrentEventRouterOptions{
			WorkerID: "router", LeaseDuration: time.Minute, Now: func() time.Time { return *now },
		},
	)
	if err != nil {
		t.Fatalf("new current event router: %v", err)
	}
	dispatcher, err := eventoutbox.NewDispatcher(outbox, router, eventoutbox.DispatcherOptions{
		LeaseDuration: time.Minute, RetryDelay: time.Second, Now: func() time.Time { return *now },
	})
	if err != nil {
		t.Fatalf("new current event dispatcher: %v", err)
	}
	return dispatcher
}

func publishReliableEvent(
	t *testing.T,
	uow *storesql.UnitOfWork,
	publisher pluginsdk.EventService,
	publication pluginsdk.EventPublication,
	rollback bool,
) pluginsdk.EventEnvelope {
	t.Helper()
	var envelope pluginsdk.EventEnvelope
	rollbackErr := errors.New("rollback publication")
	err := uow.Do(context.Background(), func(tx repository.Tx) error {
		var publishErr error
		envelope, publishErr = publisher.Publish(tx.Context(), publication)
		if publishErr != nil {
			return publishErr
		}
		if rollback {
			return rollbackErr
		}
		return nil
	})
	if rollback {
		if !errors.Is(err, rollbackErr) {
			t.Fatalf("rollback publication error=%v", err)
		}
		return envelope
	}
	if err != nil {
		t.Fatalf("publish event: %v", err)
	}
	return envelope
}

func reliableEventPublication(key string) pluginsdk.EventPublication {
	return pluginsdk.EventPublication{
		IdempotencyKey: key, Name: "inventory.received", SchemaVersion: 1,
		Scope:         pluginsdk.EventScope{TenantID: "tenant-a", OrganizationID: "org-a"},
		CorrelationID: "correlation-" + key,
		Subject:       pluginsdk.EventSubject{Type: "inventory_lot", ID: "lot-" + key},
		Payload:       pluginsdk.EventPayload{"quantity": {Type: pluginsdk.DataValueInteger, Value: "10"}},
	}
}

type reliableEventCatalog struct {
	mu    sync.RWMutex
	items map[string]plugin.Info
}

func newReliableEventCatalog(subscriberURL string) *reliableEventCatalog {
	return &reliableEventCatalog{items: map[string]plugin.Info{
		"warehouse": {
			ID: "warehouse", State: plugin.StateEnabled,
			EventContract: &plugin.EventContract{Publications: []plugin.EventPublication{{
				Name: "inventory.received", SchemaVersion: 1, PayloadType: "inventory.received", Scope: pluginsdk.EventScopeTenant,
			}}},
		},
		"analytics": {
			ID: "analytics", State: plugin.StateEnabled, ServiceBaseURL: subscriberURL,
			EventContract: &plugin.EventContract{Subscriptions: []plugin.EventSubscription{{
				Publisher: "warehouse", Name: "inventory.received", SchemaVersions: []uint32{1},
				Handler: "onInventoryReceived", RetryPolicy: "standard",
			}}},
		},
	}}
}

func (c *reliableEventCatalog) Get(id string) (plugin.Info, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[id]
	if !ok {
		return plugin.Info{}, plugin.ErrPluginNotFound
	}
	return item, nil
}

func (c *reliableEventCatalog) List() []plugin.Info {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := make([]plugin.Info, 0, len(c.items))
	for _, item := range c.items {
		items = append(items, item)
	}
	return items
}

func (c *reliableEventCatalog) setSubscriberState(state plugin.State) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item := c.items["analytics"]
	item.State = state
	c.items["analytics"] = item
}

func (c *reliableEventCatalog) uninstallSubscriber() {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, "analytics")
}

type reliableEventConsumer struct {
	*httptest.Server
	available atomic.Bool
	effects   atomic.Int64
	mu        sync.Mutex
	handled   map[string]struct{}
}

func newReliableEventConsumer(t *testing.T) *reliableEventConsumer {
	t.Helper()
	consumer := &reliableEventConsumer{handled: map[string]struct{}{}}
	consumer.available.Store(true)
	consumer.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/_skoll/events" {
			http.NotFound(w, r)
			return
		}
		if !consumer.available.Load() {
			http.Error(w, "consumer unavailable", http.StatusServiceUnavailable)
			return
		}
		var delivery pluginsdk.EventDelivery
		if err := json.NewDecoder(r.Body).Decode(&delivery); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		declaration := pluginsdk.EventSubscriptionDeclaration{
			Publisher: "warehouse", Name: "inventory.received", SchemaVersions: []uint32{1},
			Handler: "onInventoryReceived", RetryPolicy: "standard",
		}
		if delivery.Subscriber != "analytics" || delivery.Envelope.Scope.TenantID != "tenant-a" ||
			delivery.Validate(declaration) != nil {
			http.Error(w, "invalid delivery", http.StatusForbidden)
			return
		}
		consumer.mu.Lock()
		if _, duplicate := consumer.handled[delivery.DeliveryID]; !duplicate {
			consumer.handled[delivery.DeliveryID] = struct{}{}
			consumer.effects.Add(1)
		}
		consumer.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	return consumer
}

type reliableReplayAuthorizer struct{}

func (reliableReplayAuthorizer) AuthorizeEventReplay(context.Context, string, eventoutbox.Record) error {
	return nil
}

type reliableReplayAudit struct {
	calls atomic.Int64
}

func (a *reliableReplayAudit) RecordEventReplay(context.Context, string, eventoutbox.Record) error {
	a.calls.Add(1)
	return nil
}

func openReliableEventDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "reliable-events.db")+"?_busy_timeout=5000&_journal_mode=WAL"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open reliable event database: %v", err)
	}
	if err := db.AutoMigrate(&gormrepo.PluginEventOutboxModel{}, &gormrepo.PluginEventInboxModel{}); err != nil {
		t.Fatalf("migrate reliable event database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("unwrap reliable event database: %v", err)
	}
	return db, func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close reliable event database: %v", err)
		}
	}
}
