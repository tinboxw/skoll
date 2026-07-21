package gormrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestNotificationStorePersistsInboxRulesAndDeliveryAcrossRestart(t *testing.T) {
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "notification.db")
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.UTC)
	db := openNotificationTestDB(t, dsn)
	service := notificationsvc.NewService(NewNotificationStore(db), func() time.Time { return now }, nil)

	messageInput := notificationsvc.CreateInput{
		ID: "notification-1", Category: notificationsvc.CategoryMessage, Title: "Approval copied", Body: "Review the approval result.",
		ActorID: "employee-1", Target: notificationsvc.Target{Type: "workflow", ID: "workflow-1", Path: "/workflow/workflow-1"},
	}
	message, err := service.Create(ctx, messageInput)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	duplicate, err := service.Create(ctx, messageInput)
	if err != nil || duplicate.ID != message.ID || !duplicate.CreatedAt.Equal(message.CreatedAt) {
		t.Fatalf("duplicate notification was not idempotent: item=%+v err=%v", duplicate, err)
	}
	conflict := messageInput
	conflict.Title = "Conflicting title"
	if _, err := service.Create(ctx, conflict); err == nil {
		t.Fatal("expected conflicting duplicate notification id to fail")
	}
	read, err := service.Complete(ctx, message.ID, "employee-1")
	if err != nil || read.Status != notificationsvc.StatusRead {
		t.Fatalf("Complete message error=%v item=%+v", err, read)
	}

	if err := service.UpsertReminderRule(ctx, notificationsvc.ReminderRule{
		ID: "rule-qualification", Name: "Qualification", ActorID: "employee-1", Title: "Qualification expires",
		Target: notificationsvc.Target{Type: "supplier", ID: "supplier-1", Path: "/suppliers/supplier-1"}, DueAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("UpsertReminderRule error: %v", err)
	}
	emitted, err := service.EmitDueReminders(ctx)
	if err != nil || len(emitted) != 1 {
		t.Fatalf("EmitDueReminders error=%v items=%+v", err, emitted)
	}
	failed, err := service.RecordDeliveryAttempt(ctx, notificationsvc.DeliveryAttemptInput{
		ID: "delivery-1", NotificationID: message.ID, Channel: "email", IdempotencyKey: "send-1",
		Status: notificationsvc.DeliveryFailed, Error: "smtp unavailable",
	})
	if err != nil || failed.Attempt != 1 {
		t.Fatalf("Record failed delivery error=%v item=%+v", err, failed)
	}
	closeNotificationTestDB(t, db)

	restartedDB := openNotificationTestDB(t, dsn)
	t.Cleanup(func() { closeNotificationTestDB(t, restartedDB) })
	now = now.Add(time.Hour)
	restarted := notificationsvc.NewService(NewNotificationStore(restartedDB), func() time.Time { return now }, nil)
	items, err := restarted.List(ctx, notificationsvc.Filter{ActorID: "employee-1"})
	if err != nil || len(items) != 2 {
		t.Fatalf("List after restart error=%v items=%+v", err, items)
	}
	readFound := false
	for _, item := range items {
		if item.ID == message.ID && item.Status == notificationsvc.StatusRead {
			readFound = true
		}
	}
	if !readFound {
		t.Fatalf("read state did not survive restart: %+v", items)
	}
	reemitted, err := restarted.EmitDueReminders(ctx)
	if err != nil || len(reemitted) != 1 || reemitted[0].ID != emitted[0].ID {
		t.Fatalf("reminder rule was not restart-safe and idempotent: items=%+v err=%v", reemitted, err)
	}
	items, err = restarted.List(ctx, notificationsvc.Filter{ActorID: "employee-1"})
	if err != nil || len(items) != 2 {
		t.Fatalf("re-emission created duplicate inbox item: items=%+v err=%v", items, err)
	}

	duplicateFailure, err := restarted.RecordDeliveryAttempt(ctx, notificationsvc.DeliveryAttemptInput{
		ID: "delivery-duplicate", NotificationID: message.ID, Channel: "email", IdempotencyKey: "send-1",
		Status: notificationsvc.DeliverySucceeded,
	})
	if err != nil || duplicateFailure.ID != failed.ID || duplicateFailure.Status != notificationsvc.DeliveryFailed {
		t.Fatalf("same delivery key must return the first attempt: item=%+v err=%v", duplicateFailure, err)
	}
	succeeded, err := restarted.RecordDeliveryAttempt(ctx, notificationsvc.DeliveryAttemptInput{
		ID: "delivery-2", NotificationID: message.ID, Channel: "email", IdempotencyKey: "send-2",
		Status: notificationsvc.DeliverySucceeded,
	})
	if err != nil || succeeded.Attempt != 2 {
		t.Fatalf("failed delivery was not retryable: item=%+v err=%v", succeeded, err)
	}
	afterSuccess, err := restarted.RecordDeliveryAttempt(ctx, notificationsvc.DeliveryAttemptInput{
		ID: "delivery-3", NotificationID: message.ID, Channel: "email", IdempotencyKey: "send-3",
		Status: notificationsvc.DeliveryFailed, Error: "must not deliver again",
	})
	if err != nil || afterSuccess.ID != succeeded.ID {
		t.Fatalf("successful channel allowed duplicate delivery: item=%+v err=%v", afterSuccess, err)
	}
	attempts, err := restarted.ListDeliveryAttempts(ctx, message.ID, "email")
	if err != nil || len(attempts) != 2 || attempts[0].Status != notificationsvc.DeliveryFailed || attempts[1].Status != notificationsvc.DeliverySucceeded {
		t.Fatalf("delivery history was not restored: items=%+v err=%v", attempts, err)
	}
}

func TestNotificationStoreDeduplicatesConcurrentDeliveryKey(t *testing.T) {
	ctx := context.Background()
	path := filepath.ToSlash(filepath.Join(t.TempDir(), "notification-concurrent.db"))
	db := openNotificationTestDB(t, fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", path))
	t.Cleanup(func() { closeNotificationTestDB(t, db) })
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	now := time.Date(2026, 7, 22, 16, 0, 0, 0, time.UTC)
	service := notificationsvc.NewService(NewNotificationStore(db), func() time.Time { return now }, nil)
	item, err := service.Create(ctx, notificationsvc.CreateInput{
		ID: "notification-concurrent", Category: notificationsvc.CategoryMessage, Title: "Concurrent delivery", ActorID: "employee-1",
		Target: notificationsvc.Target{Type: "workflow", ID: "workflow-1", Path: "/workflow/workflow-1"},
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	const workers = 12
	start := make(chan struct{})
	results := make(chan notificationsvc.DeliveryAttempt, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		worker := worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			attempt, err := service.RecordDeliveryAttempt(ctx, notificationsvc.DeliveryAttemptInput{
				ID: fmt.Sprintf("delivery-%d", worker), NotificationID: item.ID, Channel: "webhook", IdempotencyKey: "event-1", Status: notificationsvc.DeliverySucceeded,
			})
			results <- attempt
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	winningID := ""
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent delivery error: %v", err)
		}
	}
	for attempt := range results {
		if winningID == "" {
			winningID = attempt.ID
		}
		if attempt.ID != winningID || attempt.Attempt != 1 {
			t.Fatalf("concurrent duplicate returned different effects: %+v winner=%s", attempt, winningID)
		}
	}
	attempts, err := service.ListDeliveryAttempts(ctx, item.ID, "webhook")
	if err != nil || len(attempts) != 1 {
		t.Fatalf("expected one persisted delivery attempt, items=%+v err=%v", attempts, err)
	}
}

func TestNotificationMigrationScriptsCoverInboxAndDeliveryState(t *testing.T) {
	required := []string{
		"sk_notification_items", "sk_notification_reminder_rules", "sk_notification_delivery_attempts",
		"idx_notification_delivery_idempotency", "idx_notification_actor_status_updated", "foreign key",
	}
	root := filepath.Join("..", "..", "..", "..", "migrations")
	for _, dialect := range []string{"mysql", "postgres"} {
		path := filepath.Join(root, dialect, "20260722_000025_create_notification_persistence.sql")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s notification migration: %v", dialect, err)
		}
		text := strings.ToLower(string(body))
		for _, token := range required {
			if !strings.Contains(text, token) {
				t.Fatalf("%s notification migration missing %q", dialect, token)
			}
		}
	}
}

func openNotificationTestDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		if isSQLiteCGODisabledError(err) {
			t.Skipf("sqlite test requires cgo: %v", err)
		}
		t.Fatalf("open notification test database: %v", err)
	}
	if err := db.AutoMigrate(&NotificationItemModel{}, &NotificationReminderRuleModel{}, &NotificationDeliveryAttemptModel{}); err != nil {
		t.Fatalf("migrate notification test database: %v", err)
	}
	return db
}

func closeNotificationTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve notification sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close notification test database: %v", err)
	}
}
