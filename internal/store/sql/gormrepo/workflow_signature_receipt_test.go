package gormrepo

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
)

func TestWorkflowSignatureReceiptSurvivesRestartRejectsReuseAndDetectsTampering(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	dsn := filepath.Join(t.TempDir(), "workflow-signature.db")
	db := openWorkflowTestDB(t, dsn)
	repo := NewWorkflowStore(db)
	definition := signedPersistenceDefinition(now)
	if err := repo.SaveDefinition(ctx, definition); err != nil {
		t.Fatal(err)
	}
	first := signedPersistenceInstance(t, definition, "instance-signed-1", "verify-once", now)
	if err := repo.SaveInstance(ctx, first); err != nil {
		t.Fatal(err)
	}
	second := signedPersistenceInstance(t, definition, "instance-signed-2", "verify-once", now.Add(time.Minute))
	if err := repo.SaveInstance(ctx, second); err == nil {
		t.Fatal("verification identity must be globally single use")
	}
	closeWorkflowTestDB(t, db)

	restarted := openWorkflowTestDB(t, dsn)
	t.Cleanup(func() { closeWorkflowTestDB(t, restarted) })
	restartedRepo := NewWorkflowStore(restarted)
	restored, err := restartedRepo.GetInstance(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Receipts) != 1 || restored.Receipts[0].VerificationID != "verify-once" {
		t.Fatalf("receipt was not restored: %+v", restored.Receipts)
	}
	if err := restored.Receipts[0].VerifyDigest(); err != nil {
		t.Fatal(err)
	}
	secondSignature := &domainworkflow.DecisionSignature{
		VerificationID: "verify-second-approver", Audience: "plugin:medical_oa", Method: "password",
		Meaning: "I approve.", VerifiedAt: now.Add(2 * time.Minute),
		Evidence: []domainworkflow.EvidenceReference{
			{FileID: "file-2", Name: "quality-review.pdf", Hash: strings.Repeat("b", 64), Size: 128, MIME: "application/pdf"},
		},
	}
	if err := restored.Approve(
		definition,
		restored.Tasks[1].ID,
		domainworkflow.Actor{ID: "quality-approver"},
		"quality approved",
		secondSignature,
		now.Add(3*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if err := restartedRepo.SaveInstance(ctx, *restored); err != nil {
		t.Fatalf("append second signed decision after restart: %v", err)
	}
	restored, err = restartedRepo.GetInstance(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Status != domainworkflow.InstanceApproved || len(restored.Receipts) != 2 || len(restored.Timeline) != 3 {
		t.Fatalf("signed all-approver workflow did not continue append-only: %+v", restored)
	}
	firstActionID := restored.Receipts[0].ActionID.String()
	if err := restarted.Model(&WorkflowActionModel{}).
		Where("id = ?", firstActionID).
		Update("comment", "tampered action").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := restartedRepo.GetInstance(ctx, first.ID); err == nil || !strings.Contains(err.Error(), "action correlation") {
		t.Fatalf("tampered workflow action error = %v", err)
	}
	if err := restarted.Model(&WorkflowActionModel{}).
		Where("id = ?", firstActionID).
		Update("comment", "approved").Error; err != nil {
		t.Fatal(err)
	}
	if err := restarted.Model(&WorkflowSignatureReceiptModel{}).
		Where("id = ?", restored.Receipts[0].ID.String()).
		Update("meaning", "tampered meaning").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := restartedRepo.GetInstance(ctx, first.ID); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("tampered receipt error = %v", err)
	}
}

func signedPersistenceDefinition(now time.Time) domainworkflow.Definition {
	return domainworkflow.Definition{
		ID: "signed-definition", Key: "regulated.signature", Name: "Signed decision", Version: 1,
		Status: domainworkflow.DefinitionPublished,
		Nodes: []domainworkflow.Node{
			{ID: "start", Key: "start", Name: "Start", Type: domainworkflow.NodeStart},
			{
				ID: "approve", Key: "approve", Name: "Approve", Type: domainworkflow.NodeApproval,
				Assignees: []shared.ID{"approver", "quality-approver"}, Decision: domainworkflow.DecisionRule{Strategy: domainworkflow.DecisionAll, Quorum: 2},
				Signature: &domainworkflow.SignaturePolicy{Meaning: "I approve.", RequireEvidence: true},
			},
			{ID: "end", Key: "end", Name: "End", Type: domainworkflow.NodeEnd},
		},
		Transitions: []domainworkflow.Transition{{From: "start", To: "approve"}, {From: "approve", To: "end"}},
		Meta:        shared.AuditMeta{CreatedAt: now, UpdatedAt: now},
	}
}

func signedPersistenceInstance(t *testing.T, definition domainworkflow.Definition, id, verificationID string, now time.Time) domainworkflow.Instance {
	t.Helper()
	instance, err := domainworkflow.Start(domainworkflow.StartInput{
		ID: shared.ID(id), Definition: definition, BusinessType: "release", BusinessID: id,
		Title: "Signed release", Starter: domainworkflow.Actor{ID: "starter"}, Variables: map[string]domainworkflow.Value{}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	signature := &domainworkflow.DecisionSignature{
		VerificationID: shared.ID(verificationID), Audience: "plugin:medical_oa", Method: "password",
		Meaning: "I approve.", VerifiedAt: now,
		Evidence: []domainworkflow.EvidenceReference{
			{FileID: "file-1", Name: "evidence.pdf", Hash: strings.Repeat("a", 64), Size: 64, MIME: "application/pdf"},
		},
	}
	if err := instance.Approve(definition, instance.Tasks[0].ID, domainworkflow.Actor{ID: "approver"}, "approved", signature, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	return *instance
}
