package workflow

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	"github.com/tinboxw/skoll/internal/service/job"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/security"
)

type signatureEvidenceResolver struct {
	references []domainworkflow.EvidenceReference
}

func (r signatureEvidenceResolver) ResolveWorkflowEvidence(_ context.Context, _ domainworkflow.Actor, ids []shared.ID) ([]domainworkflow.EvidenceReference, error) {
	if len(ids) != len(r.references) {
		return nil, context.Canceled
	}
	return append([]domainworkflow.EvidenceReference(nil), r.references...), nil
}

func TestSignedDecisionVerifiesBindingMeaningAndEvidence(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	proofs, err := security.NewReverificationProofService("workflow-signature-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(NewMemoryRepository(), Options{
		Jobs:       job.NewService(job.NewMemoryRepository(), func() time.Time { return now }),
		UnitOfWork: storesql.NewUnitOfWork(), Now: func() time.Time { return now },
		Proofs: proofs, Evidence: signatureEvidenceResolver{references: []domainworkflow.EvidenceReference{
			{FileID: "file-1", Name: "decision.pdf", Hash: strings.Repeat("a", 64), Size: 42, MIME: "application/pdf"},
		}},
	})
	definition, err := service.CreateDefinition(ctx, CreateDefinitionInput{
		ID: "signed-definition", Key: "regulated.release", Name: "Regulated release", Version: 1,
		Nodes: []domainworkflow.Node{
			{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
			{
				ID: "approve", Key: "approve", Name: "Approve", Type: domainworkflow.NodeApproval,
				Assignees: []shared.ID{"approver-1"}, Decision: domainworkflow.DecisionRule{Strategy: domainworkflow.DecisionAny, Quorum: 1},
				Signature: &domainworkflow.SignaturePolicy{Meaning: "I approve this release.", RequireEvidence: true},
			},
			{ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
		},
		Transitions: []domainworkflow.Transition{{From: "start", To: "approve"}, {From: "approve", To: "end"}}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.PublishDefinition(ctx, definition.ID, now); err != nil {
		t.Fatal(err)
	}
	instance, err := service.Start(ctx, StartInput{
		ID: "signed-instance", DefinitionID: definition.ID, BusinessType: "release", BusinessID: "batch-1",
		Title: "Release", Starter: domainworkflow.Actor{ID: "starter"}, Variables: map[string]domainworkflow.Value{}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	proof, _, err := proofs.IssueReverificationProof(
		"approver-1", "plugin:medical_oa", security.ReverificationPurposeWorkflowSignature, "password", now, 2*time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	action := TaskActionInput{
		InstanceID: instance.ID, TaskID: instance.Tasks[0].ID, Actor: domainworkflow.Actor{ID: "approver-1"},
		Comment: "approved", Now: now.Add(time.Minute),
		Signature: &DecisionSignatureInput{
			Proof: proof, Meaning: "I approve this release.", Audience: "plugin:medical_oa", EvidenceIDs: []shared.ID{"file-1"},
		},
	}
	approved, err := service.Approve(ctx, action)
	if err != nil {
		t.Fatal(err)
	}
	if len(approved.Receipts) != 1 || approved.Receipts[0].Evidence[0].Hash != strings.Repeat("a", 64) {
		t.Fatalf("unexpected signed result: %+v", approved.Receipts)
	}

	second, err := service.Start(ctx, StartInput{
		ID: "signed-instance-2", DefinitionID: definition.ID, BusinessType: "release", BusinessID: "batch-2",
		Title: "Release 2", Starter: domainworkflow.Actor{ID: "starter"}, Variables: map[string]domainworkflow.Value{}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	action.InstanceID = second.ID
	action.TaskID = second.Tasks[0].ID
	action.Signature.Meaning = "I merely reviewed this release."
	if _, err := service.Approve(ctx, action); err == nil {
		t.Fatal("wrong signing meaning must fail")
	}
	action.Signature.Meaning = "I approve this release."
	action.Signature.Audience = "plugin:other"
	if _, err := service.Approve(ctx, action); err == nil {
		t.Fatal("wrong proof audience must fail")
	}
	action.Signature.Audience = "plugin:medical_oa"
	if _, err := service.Approve(ctx, action); err == nil {
		t.Fatal("one reverification proof must not authorize another workflow decision")
	}
}
