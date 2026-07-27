package workflow

import (
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestSubstitutionAndDelegationRemainTaskScopedAndTraceable(t *testing.T) {
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	definition := assignmentTestDefinition(t, now, nil)
	instance, err := Start(StartInput{
		ID: "instance-assignment", Definition: definition, BusinessType: "leave", BusinessID: "LEAVE-1",
		Title: "Leave", Starter: Actor{ID: "employee-1"}, Variables: map[string]Value{}, Now: now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	window, err := NewSubstitutionWindow(
		"absence-manager-1",
		Actor{ID: "manager-1", Name: "Manager One"},
		Actor{ID: "backup-1", Name: "Backup One"},
		Actor{ID: "manager-1", Name: "Manager One"},
		now.Add(-time.Hour), now.Add(8*time.Hour), now, "annual leave",
	)
	if err != nil {
		t.Fatalf("NewSubstitutionWindow error: %v", err)
	}
	if err := instance.ApplySubstitutions([]SubstitutionWindow{*window}, now); err != nil {
		t.Fatalf("ApplySubstitutions error: %v", err)
	}
	if got := instance.Tasks[0]; got.Assignee.ID != "backup-1" || got.OriginalAssignee.ID != "manager-1" ||
		got.Assignment != AssignmentSubstituted || got.AuthorizationID != window.ID {
		t.Fatalf("substitution provenance is incomplete: %+v", got)
	}
	if len(instance.Timeline) != 2 || instance.Timeline[1].Type != ActionSubstitute {
		t.Fatalf("substitution timeline is incomplete: %+v", instance.Timeline)
	}
	if err := instance.Approve(definition, instance.Tasks[1].ID, Actor{ID: "backup-1"}, "", now.Add(time.Minute)); err == nil ||
		!strings.Contains(err.Error(), "assignee mismatch") {
		t.Fatalf("substitute broadened into another task: %v", err)
	}

	second, err := Start(StartInput{
		ID: "instance-delegation", Definition: definition, BusinessType: "leave", BusinessID: "LEAVE-2",
		Title: "Leave", Starter: Actor{ID: "employee-2"}, Variables: map[string]Value{}, Now: now,
	})
	if err != nil {
		t.Fatalf("second Start error: %v", err)
	}
	original := second.Tasks[0]
	if err := second.Delegate(original.ID, original.Assignee, Actor{ID: "delegate-1"}, "temporary delegation", now.Add(time.Minute)); err != nil {
		t.Fatalf("Delegate error: %v", err)
	}
	delegated := second.Tasks[len(second.Tasks)-1]
	if delegated.Assignment != AssignmentDelegated || delegated.OriginalAssignee.ID != original.Assignee.ID ||
		delegated.AuthorizationID.IsZero() || delegated.AuthorizedBy.ID != original.Assignee.ID {
		t.Fatalf("delegation provenance is incomplete: %+v", delegated)
	}
	if err := second.Approve(definition, second.Tasks[1].ID, Actor{ID: "delegate-1"}, "", now.Add(2*time.Minute)); err == nil ||
		!strings.Contains(err.Error(), "assignee mismatch") {
		t.Fatalf("delegate broadened into another task: %v", err)
	}
}

func TestEscalationIsBoundedAndIdempotent(t *testing.T) {
	now := time.Date(2026, 7, 27, 11, 0, 0, 0, time.UTC)
	escalation := &EscalationRule{After: time.Hour, Target: Actor{ID: "director-1", Name: "Director"}}
	definition := assignmentTestDefinition(t, now, escalation)
	instance, err := Start(StartInput{
		ID: "instance-escalation", Definition: definition, BusinessType: "purchase", BusinessID: "PO-1",
		Title: "Purchase", Starter: Actor{ID: "buyer-1"}, Variables: map[string]Value{}, Now: now,
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	taskID := instance.Tasks[0].ID
	for range 3 {
		if err := instance.Escalate(taskID, escalation.Target, now.Add(time.Hour)); err != nil {
			t.Fatalf("Escalate error: %v", err)
		}
	}
	actions := 0
	for _, action := range instance.Timeline {
		if action.Type == ActionEscalate {
			actions++
		}
	}
	if actions != 1 || len(instance.Tasks) != 3 {
		t.Fatalf("escalation executed more than once: actions=%d tasks=%+v", actions, instance.Tasks)
	}
	escalated := instance.Tasks[len(instance.Tasks)-1]
	if escalated.Assignment != AssignmentEscalated || escalated.Assignee.ID != "director-1" ||
		escalated.OriginalAssignee.ID != "manager-1" {
		t.Fatalf("escalation provenance is incomplete: %+v", escalated)
	}
	if err := (&EscalationRule{After: time.Second, Target: Actor{ID: "director-1"}}).Validate(); err == nil {
		t.Fatal("expected sub-minute escalation to be rejected")
	}
}

func assignmentTestDefinition(t *testing.T, now time.Time, escalation *EscalationRule) Definition {
	t.Helper()
	definition, err := NewDefinition("definition-assignment", "assignment.approval", "Assignment approval", 1, []Node{
		{ID: "start", Key: "start", Name: "Start", Type: NodeStart},
		{
			ID: "approval", Key: "approval", Name: "Approval", Type: NodeApproval,
			Assignees: []shared.ID{"manager-1", "manager-2"},
			Decision:  DecisionRule{Strategy: DecisionAll, Quorum: 2}, Escalation: escalation,
		},
		{ID: "end", Key: "end", Name: "End", Type: NodeEnd},
	}, []Transition{{From: "start", To: "approval"}, {From: "approval", To: "end"}}, now)
	if err != nil {
		t.Fatalf("NewDefinition error: %v", err)
	}
	if err := definition.Publish(now); err != nil {
		t.Fatalf("Publish error: %v", err)
	}
	return *definition
}
