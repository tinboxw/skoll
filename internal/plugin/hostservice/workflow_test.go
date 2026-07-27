package hostservice

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

type workflowSignatureAudit struct {
	entries []pluginsdk.AuditEntry
}

func (a *workflowSignatureAudit) Record(_ context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	a.entries = append(a.entries, entry)
	return pluginsdk.AuditReceipt{ID: "audit-receipt", OccurredAt: time.Now().UTC()}, nil
}

type workflowSignatureEvidence struct{}

func (workflowSignatureEvidence) ResolveWorkflowEvidence(
	_ context.Context,
	_ domainworkflow.Actor,
	fileIDs []shared.ID,
) ([]domainworkflow.EvidenceReference, error) {
	items := make([]domainworkflow.EvidenceReference, 0, len(fileIDs))
	for _, id := range fileIDs {
		items = append(items, domainworkflow.EvidenceReference{
			FileID: id, Name: "quality-certificate.pdf", Hash: "sha256:certificate", Size: 1024, MIME: "application/pdf",
		})
	}
	return items, nil
}

func TestWorkflowServiceBindsIdentityAndNamespace(t *testing.T) {
	backend := workflowsvc.NewService(workflowsvc.NewMemoryRepository(), workflowsvc.Options{
		Jobs:       jobsvc.NewService(jobsvc.NewMemoryRepository(), func() time.Time { return time.Now().UTC() }),
		UnitOfWork: storesql.NewUnitOfWork(), Now: func() time.Time { return time.Now().UTC() },
	})
	audit, err := NewAuditService("plugin_a", auditsvc.NewService(clickhouse.NewAuditStore()))
	if err != nil {
		t.Fatalf("NewAuditService error: %v", err)
	}
	service, err := NewWorkflowService("plugin_a", backend, audit)
	if err != nil {
		t.Fatalf("NewWorkflowService error: %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "approver-1"})
	definition, err := service.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: "approval", Key: "approval", Name: "Approval", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{
				ID: "review", Key: "review", Name: "Review", Type: pluginsdk.WorkflowNodeApproval,
				AssigneeIDs: []string{"approver-1"}, Decision: &pluginsdk.WorkflowDecisionRule{Strategy: pluginsdk.WorkflowDecisionAny, Quorum: 1},
				Escalation: &pluginsdk.WorkflowEscalationRule{AfterSeconds: 3600, Target: pluginsdk.WorkflowActor{ID: "director-1"}},
			},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "review"}, {From: "review", To: "end"}},
	})
	if err != nil || definition.ID != "approval" || definition.Key != "approval" {
		t.Fatalf("CreateDefinition item=%+v err=%v", definition, err)
	}
	if definition.Nodes[1].Escalation == nil || definition.Nodes[1].Escalation.Target.ID != "director-1" {
		t.Fatalf("workflow escalation contract was not preserved: %+v", definition.Nodes[1])
	}
	substitution, err := service.CreateSubstitution(ctx, pluginsdk.WorkflowSubstitutionInput{
		ID: "absence-1", Substitute: pluginsdk.WorkflowActor{ID: "backup-1"},
		StartsAt: time.Now().UTC().Add(-time.Hour), EndsAt: time.Now().UTC().Add(time.Hour), Reason: "planned absence",
	})
	if err != nil {
		t.Fatalf("CreateSubstitution error: %v", err)
	}
	if substitution.ID != "absence-1" || substitution.Principal.ID != "approver-1" || substitution.Substitute.ID != "backup-1" {
		t.Fatalf("workflow substitution namespace or identity is invalid: %+v", substitution)
	}
	otherActor := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "approver-2"})
	if _, err := service.RevokeSubstitution(otherActor, substitution.ID); err == nil {
		t.Fatal("different actor revoked another principal's substitution")
	}
	if _, err := service.RevokeSubstitution(ctx, substitution.ID); err != nil {
		t.Fatalf("RevokeSubstitution error: %v", err)
	}
	if _, err = service.PublishDefinition(ctx, definition.ID); err != nil {
		t.Fatalf("PublishDefinition error: %v", err)
	}
	instance, err := service.Start(ctx, pluginsdk.WorkflowStartInput{ID: "request", DefinitionID: definition.ID, BusinessType: "request", BusinessID: "1", Title: "Request"})
	if err != nil || instance.Starter.ID != "approver-1" || len(instance.Tasks) != 1 {
		t.Fatalf("Start item=%+v err=%v", instance, err)
	}
	approved, err := service.Approve(ctx, pluginsdk.WorkflowTaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[0].ID})
	if err != nil || approved.Status != pluginsdk.WorkflowInstanceApproved {
		t.Fatalf("Approve item=%+v err=%v", approved, err)
	}
	otherAudit, _ := NewAuditService("plugin_b", auditsvc.NewService(clickhouse.NewAuditStore()))
	other, err := NewWorkflowService("plugin_b", backend, otherAudit)
	if err != nil {
		t.Fatalf("NewWorkflowService other error: %v", err)
	}
	if _, err := other.GetDefinition(ctx, "approval"); err == nil {
		t.Fatal("another plugin read a foreign workflow definition")
	}
}

func TestWorkflowServiceElectronicSignatureCorrelatesImmutableAuditEvidence(t *testing.T) {
	now := time.Date(2026, time.July, 27, 10, 0, 0, 0, time.UTC)
	proofs, err := security.NewReverificationProofService("workflow-signature-test-secret")
	if err != nil {
		t.Fatalf("NewReverificationProofService error: %v", err)
	}
	backend := workflowsvc.NewService(workflowsvc.NewMemoryRepository(), workflowsvc.Options{
		Jobs:       jobsvc.NewService(jobsvc.NewMemoryRepository(), func() time.Time { return now }),
		UnitOfWork: storesql.NewUnitOfWork(),
		Now:        func() time.Time { return now },
		Proofs:     proofs,
		Evidence:   workflowSignatureEvidence{},
	})
	audit := &workflowSignatureAudit{}
	serviceAPI, err := NewWorkflowService("medical_oa", backend, audit)
	if err != nil {
		t.Fatalf("NewWorkflowService error: %v", err)
	}
	service := serviceAPI.(*workflowService)
	service.now = func() time.Time { return now }
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "quality-director"})
	definition, err := service.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: "release", Key: "release", Name: "Batch Release", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{
				ID: "quality", Key: "quality", Name: "Quality", Type: pluginsdk.WorkflowNodeApproval,
				AssigneeIDs: []string{"quality-director"},
				Decision:    &pluginsdk.WorkflowDecisionRule{Strategy: pluginsdk.WorkflowDecisionAny, Quorum: 1},
				Signature:   &pluginsdk.WorkflowSignaturePolicy{Meaning: "I approve this batch release", RequireEvidence: true},
			},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "quality"}, {From: "quality", To: "end"}},
	})
	if err != nil {
		t.Fatalf("CreateDefinition error: %v", err)
	}
	if _, err := service.PublishDefinition(ctx, definition.ID); err != nil {
		t.Fatalf("PublishDefinition error: %v", err)
	}
	instance, err := service.Start(ctx, pluginsdk.WorkflowStartInput{
		ID: "batch-001", DefinitionID: definition.ID, BusinessType: "batch-release", BusinessID: "B001", Title: "Batch B001",
	})
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	proof, _, err := proofs.IssueReverificationProof(
		"quality-director", "plugin:medical_oa", security.ReverificationPurposeWorkflowSignature, "password", now, 2*time.Minute,
	)
	if err != nil {
		t.Fatalf("IssueReverificationProof error: %v", err)
	}
	approved, err := service.Approve(ctx, pluginsdk.WorkflowTaskActionInput{
		InstanceID: instance.ID,
		TaskID:     instance.Tasks[0].ID,
		Comment:    "release approved",
		Signature: &pluginsdk.WorkflowDecisionSignature{
			Proof: proof, Meaning: "I approve this batch release", EvidenceIDs: []string{"quality-certificate"},
		},
	})
	if err != nil {
		t.Fatalf("Approve error: %v", err)
	}
	if len(approved.Receipts) != 1 || approved.Timeline[len(approved.Timeline)-1].ReceiptID != approved.Receipts[0].ID {
		t.Fatalf("signature receipt was not linked to action: %+v", approved)
	}
	receipt := approved.Receipts[0]
	if receipt.EvidenceDigest == "" || receipt.AuditCorrelationID != receipt.ID || len(receipt.Evidence) != 1 {
		t.Fatalf("signature receipt evidence is incomplete: %+v", receipt)
	}
	lastAudit := audit.entries[len(audit.entries)-1]
	if lastAudit.Detail["receiptId"] != receipt.ID ||
		lastAudit.Detail["evidenceDigest"] != receipt.EvidenceDigest ||
		lastAudit.Detail["auditCorrelationId"] != receipt.AuditCorrelationID {
		t.Fatalf("workflow audit is not correlated to signature receipt: %+v", lastAudit.Detail)
	}

	second, err := service.Start(ctx, pluginsdk.WorkflowStartInput{
		ID: "batch-002", DefinitionID: definition.ID, BusinessType: "batch-release", BusinessID: "B002", Title: "Batch B002",
	})
	if err != nil {
		t.Fatalf("Start second instance error: %v", err)
	}
	if _, err := service.Approve(ctx, pluginsdk.WorkflowTaskActionInput{
		InstanceID: second.ID,
		TaskID:     second.Tasks[0].ID,
		Signature: &pluginsdk.WorkflowDecisionSignature{
			Proof: proof, Meaning: "I approve this batch release", EvidenceIDs: []string{"quality-certificate"},
		},
	}); err == nil {
		t.Fatal("reused reverification proof was accepted across workflow instances")
	}
}
