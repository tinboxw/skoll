package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestJobStorePersistsLeaseRetryAndDeadLetterAcrossRestart(t *testing.T) {
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "jobs.db")
	now := time.Date(2026, 7, 22, 20, 0, 0, 0, time.UTC)
	db := openJobTestDB(t, dsn)
	service := jobsvc.NewService(NewJobStore(db), func() time.Time { return now })
	created, err := service.Schedule(ctx, jobsvc.ScheduleInput{
		ID: "job-restart", Namespace: "plugin.pharma_oa", Kind: "qualification-scan", IdempotencyKey: "daily-2026-07-22",
		Payload: []byte(`{"days":30}`), MaxAttempts: 2,
	})
	if err != nil {
		t.Fatalf("Schedule error: %v", err)
	}
	retried, err := service.Schedule(ctx, jobsvc.ScheduleInput{
		ID: "job-restart-new-request", Namespace: "plugin.pharma_oa", Kind: "qualification-scan", IdempotencyKey: "daily-2026-07-22",
		Payload: []byte(`{"days":30}`), MaxAttempts: 2,
	})
	if err != nil || retried.ID != created.ID {
		t.Fatalf("durable idempotency key created a second job: item=%+v err=%v", retried, err)
	}
	for _, id := range []string{"independent-1", "independent-2"} {
		if _, err := service.Schedule(ctx, jobsvc.ScheduleInput{ID: id, Namespace: "plugin.pharma_oa", Kind: "independent", RunAt: now.Add(time.Hour), MaxAttempts: 1}); err != nil {
			t.Fatalf("schedule job without idempotency key %s: %v", id, err)
		}
	}
	first, err := service.LeaseDue(ctx, jobsvc.LeaseInput{Namespace: "plugin.pharma_oa", WorkerID: "worker-1", Limit: 1, LeaseDuration: time.Minute})
	if err != nil || len(first) != 1 || first[0].AttemptCount != 1 {
		t.Fatalf("first lease error=%v items=%+v", err, first)
	}
	firstToken := first[0].LeaseToken
	closeJobTestDB(t, db)

	restartedDB := openJobTestDB(t, dsn)
	t.Cleanup(func() { closeJobTestDB(t, restartedDB) })
	restarted := jobsvc.NewService(NewJobStore(restartedDB), func() time.Time { return now })
	stored, err := restarted.Get(ctx, created.ID)
	if err != nil || stored.Status != jobsvc.StatusRunning || stored.LeaseToken != firstToken {
		t.Fatalf("lease did not survive restart: item=%+v err=%v", stored, err)
	}
	if duplicate, err := restarted.LeaseDue(ctx, jobsvc.LeaseInput{Namespace: "plugin.pharma_oa", WorkerID: "worker-2", Limit: 1, LeaseDuration: time.Minute}); err != nil || len(duplicate) != 0 {
		t.Fatalf("restart ignored active lease: items=%+v err=%v", duplicate, err)
	}

	now = now.Add(time.Minute)
	second, err := restarted.LeaseDue(ctx, jobsvc.LeaseInput{Namespace: "plugin.pharma_oa", WorkerID: "worker-2", Limit: 1, LeaseDuration: time.Minute})
	if err != nil || len(second) != 1 || second[0].AttemptCount != 2 || second[0].LeaseToken == firstToken {
		t.Fatalf("expired lease was not recovered: items=%+v err=%v", second, err)
	}
	if _, err := restarted.Complete(ctx, jobsvc.CompleteInput{JobID: created.ID, LeaseToken: firstToken}); !errors.Is(err, jobsvc.ErrLeaseLost) {
		t.Fatalf("stale lease completed recovered job: %v", err)
	}
	dead, err := restarted.Fail(ctx, jobsvc.FailInput{JobID: created.ID, LeaseToken: second[0].LeaseToken, Error: "qualification service unavailable", RetryAfter: time.Minute})
	if err != nil || dead.Status != jobsvc.StatusDeadLetter || dead.DeadLetteredAt == nil {
		t.Fatalf("retry exhaustion did not dead-letter job: item=%+v err=%v", dead, err)
	}
	deadLetters, err := restarted.List(ctx, jobsvc.Filter{Namespace: "plugin.pharma_oa", Status: jobsvc.StatusDeadLetter})
	if err != nil || len(deadLetters) != 1 || deadLetters[0].LastError == "" {
		t.Fatalf("dead letter not observable: items=%+v err=%v", deadLetters, err)
	}
}

func TestJobStoreLeasesDueJobOnceConcurrently(t *testing.T) {
	ctx := context.Background()
	path := filepath.ToSlash(filepath.Join(t.TempDir(), "jobs-concurrent.db"))
	db := openJobTestDB(t, fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", path))
	t.Cleanup(func() { closeJobTestDB(t, db) })
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	now := time.Date(2026, 7, 22, 21, 0, 0, 0, time.UTC)
	service := jobsvc.NewService(NewJobStore(db), func() time.Time { return now })
	if _, err := service.Schedule(ctx, jobsvc.ScheduleInput{ID: "job-concurrent", Namespace: "system", Kind: "health-scan", MaxAttempts: 1}); err != nil {
		t.Fatalf("Schedule error: %v", err)
	}

	const workers = 16
	start := make(chan struct{})
	counts := make(chan int, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			items, err := service.LeaseDue(ctx, jobsvc.LeaseInput{Namespace: "system", WorkerID: fmt.Sprintf("worker-%d", i), Limit: 1, LeaseDuration: time.Minute})
			counts <- len(items)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(counts)
	close(errs)
	total := 0
	for count := range counts {
		total += count
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent lease error: %v", err)
		}
	}
	if total != 1 {
		t.Fatalf("expected one durable lease, got %d", total)
	}

	now = now.Add(time.Minute)
	if items, err := service.LeaseDue(ctx, jobsvc.LeaseInput{Namespace: "system", WorkerID: "observer", Limit: 1, LeaseDuration: time.Minute}); err != nil || len(items) != 0 {
		t.Fatalf("exhausted expired job was leased again: items=%+v err=%v", items, err)
	}
	stored, err := service.Get(ctx, "job-concurrent")
	if err != nil || stored.Status != jobsvc.StatusDeadLetter || stored.LastError == "" {
		t.Fatalf("expired final lease was not observable as dead letter: item=%+v err=%v", stored, err)
	}
}

func TestJobStoreExpiresOnlyRequestedNamespace(t *testing.T) {
	ctx := context.Background()
	db := openJobTestDB(t, filepath.Join(t.TempDir(), "job-namespace.db"))
	t.Cleanup(func() { closeJobTestDB(t, db) })
	now := time.Date(2026, 7, 22, 22, 0, 0, 0, time.UTC)
	service := jobsvc.NewService(NewJobStore(db), func() time.Time { return now })
	for _, namespace := range []string{"plugin.a", "plugin.b"} {
		if _, err := service.Schedule(ctx, jobsvc.ScheduleInput{ID: namespace, Namespace: namespace, Kind: "sync", MaxAttempts: 1}); err != nil {
			t.Fatalf("Schedule %s error: %v", namespace, err)
		}
		if items, err := service.LeaseDue(ctx, jobsvc.LeaseInput{Namespace: namespace, WorkerID: namespace, Limit: 1, LeaseDuration: time.Minute}); err != nil || len(items) != 1 {
			t.Fatalf("LeaseDue %s items=%+v err=%v", namespace, items, err)
		}
	}
	now = now.Add(time.Minute)
	if _, err := service.LeaseDue(ctx, jobsvc.LeaseInput{Namespace: "plugin.a", WorkerID: "plugin.a", Limit: 1, LeaseDuration: time.Minute}); err != nil {
		t.Fatalf("expire plugin.a error: %v", err)
	}
	other, err := service.Get(ctx, "plugin.b")
	if err != nil || other.Status != jobsvc.StatusRunning {
		t.Fatalf("plugin.a changed plugin.b state: item=%+v err=%v", other, err)
	}
}

func TestJobMigrationScriptsCoverLeasesRetriesAndDeadLetters(t *testing.T) {
	required := []string{
		"sk_jobs", "idx_job_idempotency", "idx_job_due", "lease_token", "lease_expires_at", "max_attempts", "attempt_count", "dead_lettered_at",
	}
	root := filepath.Join("..", "..", "..", "..", "migrations")
	for _, dialect := range []string{"mysql", "postgres"} {
		path := filepath.Join(root, dialect, "20260722_000026_create_job_persistence.sql")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s job migration: %v", dialect, err)
		}
		text := strings.ToLower(string(body))
		for _, token := range required {
			if !strings.Contains(text, token) {
				t.Fatalf("%s job migration missing %q", dialect, token)
			}
		}
	}
}

func openJobTestDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		if isSQLiteCGODisabledError(err) {
			t.Skipf("sqlite test requires cgo: %v", err)
		}
		t.Fatalf("open job test database: %v", err)
	}
	if err := db.AutoMigrate(&JobModel{}); err != nil {
		t.Fatalf("migrate job test database: %v", err)
	}
	return db
}

func closeJobTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve job sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close job test database: %v", err)
	}
}
