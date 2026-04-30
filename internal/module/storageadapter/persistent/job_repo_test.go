package persistent_test

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/module/jobscheduler"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
)

func newJobRepo(t *testing.T) *persistent.JobRepository {
	t.Helper()
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.JobRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewJobRepository(gdb)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return repo
}

func TestJobRepository_CoreLifecycle(t *testing.T) {
	repo := newJobRepo(t)
	j := repo.Create("daily", "0 0 * * *")
	if j.ID == 0 {
		t.Fatalf("job: %+v", j)
	}
	got, err := repo.Get(j.ID)
	if err != nil || got.Name != "daily" {
		t.Fatalf("get: %+v err=%v", got, err)
	}
	if list := repo.List(); len(list) != 1 {
		t.Fatalf("list: %+v", list)
	}
	exec, err := repo.Run(j.ID)
	if err != nil || exec.JobID != j.ID || exec.Status == "" {
		t.Fatalf("run: %+v err=%v", exec, err)
	}
	if h := repo.History(j.ID, 10); len(h) != 1 {
		t.Fatalf("history: %+v", h)
	}
}

func TestJobRepository_ClaimAndRetryFlow(t *testing.T) {
	repo := newJobRepo(t)
	j := repo.Create("nightly", "")
	now := time.Unix(1_700_000_000, 0).UTC()

	c, err := repo.ClaimRun(j.ID, "exec-1", "node-A", now)
	if err != nil || !c.Claimed {
		t.Fatalf("claim: %+v err=%v", c, err)
	}
	c2 := repo.ClaimStatus("exec-1")
	if !c2.Claimed {
		t.Fatalf("status: %+v", c2)
	}
	c3, err := repo.RenewClaimLease("exec-1", "node-A", 60, now.Add(time.Second))
	if err != nil || c3.LeaseRenewalCount != 1 {
		t.Fatalf("renew: %+v err=%v", c3, err)
	}

	pol, err := repo.SetRetryPolicy(j.ID, jobscheduler.RetryPolicy{MaxRetries: 3, BackoffBaseMillis: 100, BackoffMaxMillis: 1000, JitterPercent: 10})
	if err != nil {
		t.Fatalf("set retry: %v", err)
	}
	_ = pol

	sch, err := repo.ScheduleRetry(j.ID, "exec-1", 1, now)
	if err != nil || sch.Attempt != 1 {
		t.Fatalf("schedule: %+v err=%v", sch, err)
	}

	dl, err := repo.MarkDeadLetter(j.ID, "exec-1", "boom", 3, now)
	if err != nil || dl.Status != "dead_lettered" {
		t.Fatalf("dead letter: %+v err=%v", dl, err)
	}
	if list := repo.ListDeadLetters(10); len(list) != 1 {
		t.Fatalf("list dl: %+v", list)
	}
	rep, err := repo.ReplayDeadLetter("exec-1", "ops", now.Add(time.Hour))
	if err != nil || rep.Status != "replayed" {
		t.Fatalf("replay: %+v err=%v", rep, err)
	}
}

func TestJobRepository_HydrateAcrossInstances(t *testing.T) {
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.JobRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewJobRepository(gdb)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	j := repo.Create("daily", "0 0 * * *")
	now := time.Unix(1_700_000_000, 0).UTC()
	if _, err := repo.Run(j.ID); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := repo.ClaimRun(j.ID, "exec-1", "node-A", now); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if _, err := repo.SetRetryPolicy(j.ID, jobscheduler.RetryPolicy{MaxRetries: 3, BackoffBaseMillis: 100, BackoffMaxMillis: 1000, JitterPercent: 10}); err != nil {
		t.Fatalf("policy: %v", err)
	}
	if _, err := repo.ScheduleRetry(j.ID, "exec-1", 1, now); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if _, err := repo.MarkDeadLetter(j.ID, "exec-1", "boom", 3, now); err != nil {
		t.Fatalf("dl: %v", err)
	}

	repo2, err := persistent.NewJobRepository(gdb)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if got := repo2.List(); len(got) != 1 || got[0].ID != j.ID {
		t.Fatalf("hydrated jobs: %+v", got)
	}
	if got := repo2.History(j.ID, 10); len(got) != 1 {
		t.Fatalf("hydrated executions: %+v", got)
	}
	if got := repo2.ClaimStatus("exec-1"); !got.Claimed {
		t.Fatalf("hydrated claim: %+v", got)
	}
	if got, err := repo2.GetRetryPolicy(j.ID); err != nil || got.MaxRetries != 3 {
		t.Fatalf("hydrated policy: %+v err=%v", got, err)
	}
	if got := repo2.ListDeadLetters(10); len(got) != 1 {
		t.Fatalf("hydrated dl: %+v", got)
	}

	// Continued counters: a new job & run advance past hydrated values.
	j2 := repo2.Create("hourly", "")
	if j2.ID <= j.ID {
		t.Fatalf("next job id did not advance: %d vs %d", j2.ID, j.ID)
	}
	e, err := repo2.Run(j2.ID)
	if err != nil || e.ID == 0 {
		t.Fatalf("run after hydrate: %+v err=%v", e, err)
	}
}
