package workflowaudit

import (
	"testing"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
)

func TestWorkflowAuditFixtureQueryableAndReplayable(t *testing.T) {
	fixture, err := BuildApprovalChainFixture()
	if err != nil {
		t.Fatalf("BuildApprovalChainFixture returned error: %v", err)
	}
	if fixture.Instance.Status != domainworkflow.InstanceApproved {
		t.Fatalf("expected approved instance, got %s", fixture.Instance.Status)
	}
	if len(fixture.Events) != 4 {
		t.Fatalf("expected 4 audit events, got %d", len(fixture.Events))
	}

	approvals := FindEvents(fixture.Events, Query{
		ActorID:      "approver-2",
		Action:       domainaudit.AuditAction("workflow.task.approve"),
		ResourceType: "workflow_instance",
		ResourceID:   fixture.Instance.ID.String(),
		Result:       domainaudit.EventResultSuccess,
	})
	if len(approvals) != 1 {
		t.Fatalf("expected one approval event for approver-2, got %d", len(approvals))
	}
	if approvals[0].SourceData["kind"] != "workflow_audit_replay" {
		t.Fatalf("approval event missing replay source data: %+v", approvals[0].SourceData)
	}

	delegations := FindEvents(fixture.Events, Query{Action: domainaudit.AuditAction("workflow.task.delegate")})
	if len(delegations) != 1 || delegations[0].Metadata["targetActorId"] != "approver-2" {
		t.Fatalf("expected delegation event to target approver-2, got %+v", delegations)
	}

	replay, err := ReplayApprovalChain(fixture.Events)
	if err != nil {
		t.Fatalf("ReplayApprovalChain returned error: %v", err)
	}
	if replay.InstanceID != fixture.Instance.ID.String() || replay.Terminal != "approved" {
		t.Fatalf("unexpected replay result: %+v", replay)
	}
	gotActions := make([]string, 0, len(replay.Records))
	for _, record := range replay.Records {
		gotActions = append(gotActions, record.Action)
	}
	want := []string{"start", "copy", "delegate", "approve"}
	for idx := range want {
		if gotActions[idx] != want[idx] {
			t.Fatalf("expected replay action %s at %d, got %s", want[idx], idx, gotActions[idx])
		}
	}
}

func TestWorkflowAuditReplayRejectsMissingOrReorderedEvents(t *testing.T) {
	fixture, err := BuildApprovalChainFixture()
	if err != nil {
		t.Fatalf("BuildApprovalChainFixture returned error: %v", err)
	}
	if _, err := ReplayApprovalChain(fixture.Events[:2]); err == nil {
		t.Fatalf("expected replay to reject incomplete chain")
	}
	reordered := []*domainaudit.Event{fixture.Events[1], fixture.Events[0], fixture.Events[2], fixture.Events[3]}
	reordered[0].OccurredAt = fixture.Events[0].OccurredAt.Add(-1)
	if _, err := ReplayApprovalChain(reordered); err == nil {
		t.Fatalf("expected replay to reject reordered chain")
	}
}

func TestWorkflowAuditReplaySmoke(t *testing.T) {
	fixture, err := BuildApprovalChainFixture()
	if err != nil {
		t.Fatalf("BuildApprovalChainFixture returned error: %v", err)
	}
	replay, err := ReplayApprovalChain(fixture.Events)
	if err != nil {
		t.Fatalf("ReplayApprovalChain returned error: %v", err)
	}
	t.Logf("workflow audit replay passed: instance=%s terminal=%s records=%d", replay.InstanceID, replay.Terminal, len(replay.Records))
}
