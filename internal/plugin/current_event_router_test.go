package plugin_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/plugin/eventinbox"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCurrentEventRouterAuthorizesAndDeduplicatesConsumption(t *testing.T) {
	store, closeDB := openEventInboxTestStore(t)
	defer closeDB()
	catalog := currentEventCatalogStub{items: []plugin.Info{
		currentPublisher(plugin.StateEnabled),
		currentSubscriber("analytics", plugin.StateEnabled, []uint32{1}),
		currentSubscriber("disabled", plugin.StateDisabled, []uint32{1}),
	}}
	client := &currentDeliveryClientStub{}
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	router, err := plugin.NewCurrentEventRouter(catalog, store, client, plugin.CurrentEventRouterOptions{
		WorkerID: "router-a", LeaseDuration: time.Minute, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new current event router: %v", err)
	}
	envelope := currentTenantEnvelope(1)
	var wg sync.WaitGroup
	var firstErr, secondErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		firstErr = router.Deliver(context.Background(), envelope)
	}()
	go func() {
		defer wg.Done()
		secondErr = router.Deliver(context.Background(), envelope)
	}()
	wg.Wait()
	if firstErr != nil || secondErr != nil {
		t.Fatalf("duplicate routing errors: first=%v second=%v", firstErr, secondErr)
	}
	calls := client.snapshot()
	if len(calls) != 1 {
		t.Fatalf("delivery calls=%d want=1", len(calls))
	}
	delivery := calls[0]
	if delivery.Subscriber != "analytics" || delivery.Envelope.Scope.TenantID != "tenant-a" ||
		delivery.Envelope.Publisher != "warehouse" || delivery.Envelope.SchemaVersion != 1 {
		t.Fatalf("tenant or contract routing changed: %+v", delivery)
	}
	record, err := store.Get(context.Background(), delivery.DeliveryID)
	if err != nil || record.Status != eventinbox.StatusSucceeded || record.AttemptCount != 1 {
		t.Fatalf("inbox record=%+v err=%v", record, err)
	}
}

func TestCurrentEventRouterRetriesFailedInboxAndRejectsForgedContracts(t *testing.T) {
	store, closeDB := openEventInboxTestStore(t)
	defer closeDB()
	catalog := currentEventCatalogStub{items: []plugin.Info{
		currentPublisher(plugin.StateEnabled),
		currentSubscriber("analytics", plugin.StateEnabled, []uint32{1}),
	}}
	client := &currentDeliveryClientStub{failures: 1}
	now := time.Date(2026, 7, 27, 13, 0, 0, 0, time.UTC)
	router, err := plugin.NewCurrentEventRouter(catalog, store, client, plugin.CurrentEventRouterOptions{
		WorkerID: "router-a", LeaseDuration: time.Minute, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new current event router: %v", err)
	}
	envelope := currentTenantEnvelope(1)
	if err := router.Deliver(context.Background(), envelope); err == nil {
		t.Fatal("expected first consumer failure")
	}
	now = now.Add(time.Second)
	if err := router.Deliver(context.Background(), envelope); err != nil {
		t.Fatalf("retry consumer delivery: %v", err)
	}
	calls := client.snapshot()
	if len(calls) != 2 {
		t.Fatalf("retry delivery calls=%d want=2", len(calls))
	}
	record, err := store.Get(context.Background(), calls[0].DeliveryID)
	if err != nil || record.Status != eventinbox.StatusSucceeded || record.AttemptCount != 2 {
		t.Fatalf("retried inbox=%+v err=%v", record, err)
	}

	before := len(client.snapshot())
	for name, alter := range map[string]func(*pluginsdk.EventEnvelope){
		"forged publisher": func(item *pluginsdk.EventEnvelope) { item.Publisher = "forged" },
		"scope mismatch":   func(item *pluginsdk.EventEnvelope) { item.Scope = pluginsdk.EventScope{} },
		"unknown version":  func(item *pluginsdk.EventEnvelope) { item.SchemaVersion = 2 },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := currentTenantEnvelope(1)
			alter(&invalid)
			if err := router.Deliver(context.Background(), invalid); err == nil {
				t.Fatalf("%s was accepted", name)
			}
			if len(client.snapshot()) != before {
				t.Fatalf("%s reached consumer", name)
			}
		})
	}
}

type currentEventCatalogStub struct {
	items []plugin.Info
}

func (s currentEventCatalogStub) Get(id string) (plugin.Info, error) {
	for _, item := range s.items {
		if item.ID == id {
			return item, nil
		}
	}
	return plugin.Info{}, plugin.ErrPluginNotFound
}

func (s currentEventCatalogStub) List() []plugin.Info {
	return append([]plugin.Info(nil), s.items...)
}

type currentDeliveryClientStub struct {
	mu       sync.Mutex
	calls    []pluginsdk.EventDelivery
	failures int
}

func (s *currentDeliveryClientStub) Deliver(_ context.Context, _ plugin.Info, delivery pluginsdk.EventDelivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, delivery)
	if s.failures > 0 {
		s.failures--
		return errors.New("consumer unavailable")
	}
	return nil
}

func (s *currentDeliveryClientStub) snapshot() []pluginsdk.EventDelivery {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]pluginsdk.EventDelivery(nil), s.calls...)
}

func currentPublisher(state plugin.State) plugin.Info {
	return plugin.Info{
		ID: "warehouse", State: state,
		EventContract: &plugin.EventContract{Publications: []plugin.EventPublication{{
			Name: "inventory.received", SchemaVersion: 1, PayloadType: "inventory.received", Scope: pluginsdk.EventScopeTenant,
		}}},
	}
}

func currentSubscriber(id string, state plugin.State, versions []uint32) plugin.Info {
	return plugin.Info{
		ID: id, State: state, ServiceBaseURL: "http://plugin.invalid",
		EventContract: &plugin.EventContract{Subscriptions: []plugin.EventSubscription{{
			Publisher: "warehouse", Name: "inventory.received", SchemaVersions: versions,
			Handler: "onInventoryReceived", RetryPolicy: "standard",
		}}},
	}
}

func currentTenantEnvelope(version uint32) pluginsdk.EventEnvelope {
	return pluginsdk.EventEnvelope{
		ID: "event-inventory-1", Publisher: "warehouse", Name: "inventory.received",
		SchemaVersion: version, PayloadType: "inventory.received",
		Scope:         pluginsdk.EventScope{TenantID: "tenant-a", OrganizationID: "org-a"},
		CorrelationID: "correlation-1", Subject: pluginsdk.EventSubject{Type: "inventory_lot", ID: "lot-1"},
		Payload:    pluginsdk.EventPayload{"quantity": {Type: pluginsdk.DataValueInteger, Value: "10"}},
		OccurredAt: time.Date(2026, 7, 27, 11, 0, 0, 0, time.UTC),
	}
}

func openEventInboxTestStore(t *testing.T) (*gormrepo.PluginEventInboxStore, func()) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "event-inbox.db")+"?_busy_timeout=5000&_journal_mode=WAL"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open event inbox database: %v", err)
	}
	if err := db.AutoMigrate(&gormrepo.PluginEventInboxModel{}); err != nil {
		t.Fatalf("migrate event inbox database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("unwrap event inbox database: %v", err)
	}
	return gormrepo.NewPluginEventInboxStore(db), func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close event inbox database: %v", err)
		}
	}
}
