package hostservice

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/eventoutbox"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type eventBusinessProbe struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

func TestEventServicePersistsTransactionBoundIdempotentOutbox(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "event-outbox.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), TranslateError: true,
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve database connection: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(&gormrepo.PluginEventOutboxModel{}, &eventBusinessProbe{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	store := gormrepo.NewPluginEventOutboxStore(db)
	declaration := pluginsdk.EventPublicationDeclaration{
		Name: "inventory-changed", SchemaVersion: 1,
		PayloadType: "inventory.change", Scope: pluginsdk.EventScopeTenant,
	}
	resolver := func(pluginID string) ([]pluginsdk.EventPublicationDeclaration, error) {
		if pluginID != "warehouse" {
			return nil, errors.New("unexpected plugin")
		}
		return []pluginsdk.EventPublicationDeclaration{declaration}, nil
	}
	now := time.Date(2026, 7, 27, 10, 30, 0, 0, time.UTC)
	events, err := NewEventService("warehouse", resolver, store, func() time.Time { return now })
	if err != nil {
		t.Fatalf("create event service: %v", err)
	}
	transactions, err := NewTransactionService(storesql.NewUnitOfWorkWithDB(db))
	if err != nil {
		t.Fatalf("create transaction service: %v", err)
	}
	publication := pluginsdk.EventPublication{
		IdempotencyKey: "movement-42", Name: declaration.Name, SchemaVersion: 1,
		Scope:         pluginsdk.EventScope{TenantID: "tenant-a", OrganizationID: "org-a"},
		CorrelationID: "request-42",
		Subject:       pluginsdk.EventSubject{Type: "inventory.item", ID: "item-42"},
		Payload:       pluginsdk.EventPayload{"quantity": {Type: pluginsdk.DataValueInteger, Value: "3"}},
	}

	forced := errors.New("force business rollback")
	err = transactions.Within(context.Background(), func(tx pluginsdk.Transaction) error {
		if err := storesql.ResolveDB(tx.Context(), db).Create(&eventBusinessProbe{ID: "probe-rollback", Name: "rolled back"}).Error; err != nil {
			return err
		}
		if _, err := events.Publish(tx.Context(), publication); err != nil {
			return err
		}
		return forced
	})
	if !errors.Is(err, forced) {
		t.Fatalf("expected forced rollback, got %v", err)
	}
	assertEventOutboxCounts(t, db, 0, 0)

	var first pluginsdk.EventEnvelope
	operationCtx, err := pluginsdk.WithOperationContext(context.Background(), pluginsdk.OperationContext{
		CorrelationID: "operation-42",
		RequestID:     "request-42",
		TraceID:       "trace-42",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = transactions.Within(operationCtx, func(tx pluginsdk.Transaction) error {
		if err := storesql.ResolveDB(tx.Context(), db).Create(&eventBusinessProbe{ID: "probe-commit", Name: "committed"}).Error; err != nil {
			return err
		}
		var publishErr error
		first, publishErr = events.Publish(tx.Context(), publication)
		return publishErr
	})
	if err != nil {
		t.Fatalf("commit business mutation and event: %v", err)
	}
	if first.CorrelationID != "operation-42" {
		t.Fatalf("event correlation=%q", first.CorrelationID)
	}
	assertEventOutboxCounts(t, db, 1, 1)

	var eventErr *pluginsdk.EventError
	if _, err = events.Publish(context.Background(), publication); !errors.As(err, &eventErr) || eventErr.Code != pluginsdk.EventErrorTransactionRequired {
		t.Fatalf("expected transaction requirement, got %v", err)
	}
	var duplicate pluginsdk.EventEnvelope
	err = transactions.Within(operationCtx, func(tx pluginsdk.Transaction) error {
		var publishErr error
		duplicate, publishErr = events.Publish(tx.Context(), publication)
		return publishErr
	})
	if err != nil {
		t.Fatalf("repeat idempotent event: %v", err)
	}
	if !reflect.DeepEqual(first, duplicate) {
		t.Fatalf("expected byte-stable event identity, got first=%+v duplicate=%+v", first, duplicate)
	}
	assertEventOutboxCounts(t, db, 1, 1)

	conflict := publication
	conflict.Payload = pluginsdk.EventPayload{"quantity": {Type: pluginsdk.DataValueInteger, Value: "4"}}
	err = transactions.Within(operationCtx, func(tx pluginsdk.Transaction) error {
		_, publishErr := events.Publish(tx.Context(), conflict)
		return publishErr
	})
	if !errors.As(err, &eventErr) || eventErr.Code != pluginsdk.EventErrorConflict {
		t.Fatalf("expected deterministic identity conflict, got %v", err)
	}
	assertEventOutboxCounts(t, db, 1, 1)

	restarted, err := NewEventService("warehouse", resolver, gormrepo.NewPluginEventOutboxStore(db), func() time.Time {
		return now.Add(time.Hour)
	})
	if err != nil {
		t.Fatalf("restart event service: %v", err)
	}
	var afterRestart pluginsdk.EventEnvelope
	err = transactions.Within(operationCtx, func(tx pluginsdk.Transaction) error {
		var publishErr error
		afterRestart, publishErr = restarted.Publish(tx.Context(), publication)
		return publishErr
	})
	if err != nil || !reflect.DeepEqual(first, afterRestart) {
		t.Fatalf("expected durable idempotency after restart, envelope=%+v err=%v", afterRestart, err)
	}

	undeclared := publication
	undeclared.Name = "inventory-deleted"
	if _, err = events.Publish(context.Background(), undeclared); !errors.As(err, &eventErr) || eventErr.Code != pluginsdk.EventErrorUndeclaredPublication {
		t.Fatalf("expected undeclared publication denial, got %v", err)
	}
}

func assertEventOutboxCounts(t *testing.T, db *gorm.DB, probes, events int64) {
	t.Helper()
	var probeCount int64
	if err := db.Model(&eventBusinessProbe{}).Count(&probeCount).Error; err != nil {
		t.Fatalf("count business probes: %v", err)
	}
	var eventCount int64
	if err := db.Model(&gormrepo.PluginEventOutboxModel{}).Count(&eventCount).Error; err != nil {
		t.Fatalf("count outbox events: %v", err)
	}
	if probeCount != probes || eventCount != events {
		t.Fatalf("unexpected transaction counts: probes=%d/%d events=%d/%d", probeCount, probes, eventCount, events)
	}
}

var _ eventoutbox.Store = (*gormrepo.PluginEventOutboxStore)(nil)
