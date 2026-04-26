package jobscheduler

import "testing"

func TestServiceCreateRunHistory(t *testing.T) {
	svc := NewService()
	job := svc.Create("daily-sync", "0 0 * * *")
	if job.ID <= 0 {
		t.Fatalf("expected generated job id")
	}

	exec, err := svc.Run(job.ID)
	if err != nil {
		t.Fatalf("run job failed: %v", err)
	}
	if exec.Status != "success" {
		t.Fatalf("unexpected execution status: %s", exec.Status)
	}

	history := svc.History(job.ID, 10)
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if history[0].ID != exec.ID {
		t.Fatalf("unexpected execution id in history: %d", history[0].ID)
	}
}
