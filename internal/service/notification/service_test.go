package notification

import (
	"context"
	"testing"
	"time"
)

func TestNotificationServiceTodoMessageAndDone(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	seq := 0
	svc := NewService(func() time.Time {
		now = now.Add(time.Minute)
		return now
	}, func(prefix string) string {
		seq++
		return prefix + "-id"
	})

	todo, err := svc.Create(context.Background(), CreateInput{
		Category: CategoryTodo,
		Title:    "Approve purchase request",
		ActorID:  "approver-1",
		Target:   Target{Type: "workflow", ID: "wf-1", Path: "/skoll/workflow?instance=wf-1"},
	})
	if err != nil {
		t.Fatalf("Create todo returned error: %v", err)
	}
	if todo.Status != StatusPending || todo.Target.Path == "" {
		t.Fatalf("todo not initialized correctly: %#v", todo)
	}

	message, err := svc.Create(context.Background(), CreateInput{
		Category: CategoryMessage,
		Title:    "Workflow copied",
		ActorID:  "approver-1",
		Target:   Target{Type: "workflow", ID: "wf-1", Path: "/skoll/workflow?instance=wf-1"},
	})
	if err != nil {
		t.Fatalf("Create message returned error: %v", err)
	}

	items, err := svc.List(context.Background(), Filter{ActorID: "approver-1", Status: StatusPending})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 pending items, got %d", len(items))
	}

	done, err := svc.Complete(context.Background(), todo.ID, "approver-1")
	if err != nil {
		t.Fatalf("Complete todo returned error: %v", err)
	}
	if done.Status != StatusDone {
		t.Fatalf("expected todo done, got %s", done.Status)
	}
	read, err := svc.Complete(context.Background(), message.ID, "approver-1")
	if err != nil {
		t.Fatalf("Complete message returned error: %v", err)
	}
	if read.Status != StatusRead {
		t.Fatalf("expected message read, got %s", read.Status)
	}
}

func TestNotificationServiceReminderRulesAndOwnership(t *testing.T) {
	now := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	svc := NewService(func() time.Time { return now }, func(prefix string) string { return prefix + "-1" })

	if err := svc.UpsertReminderRule(context.Background(), ReminderRule{
		ID:      "qual-expiring",
		Name:    "Qualification expiring",
		ActorID: "quality-user",
		Title:   "Supplier qualification expires soon",
		Body:    "Review the supplier file before purchase approval.",
		Target:  Target{Type: "supplier", ID: "sup-1", Path: "/skoll/pharma-oa/suppliers/sup-1"},
		DueAt:   now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("UpsertReminderRule returned error: %v", err)
	}

	emitted, err := svc.EmitDueReminders(context.Background())
	if err != nil {
		t.Fatalf("EmitDueReminders returned error: %v", err)
	}
	if len(emitted) != 1 || emitted[0].Category != CategoryReminder || emitted[0].Target.Path == "" {
		t.Fatalf("unexpected emitted reminders: %#v", emitted)
	}

	if _, err := svc.Complete(context.Background(), emitted[0].ID, "other-user"); err == nil {
		t.Fatalf("expected ownership error when another actor completes reminder")
	}

	done, err := svc.Complete(context.Background(), emitted[0].ID, "quality-user")
	if err != nil {
		t.Fatalf("Complete reminder returned error: %v", err)
	}
	if done.Status != StatusDone {
		t.Fatalf("expected reminder done, got %s", done.Status)
	}
}
