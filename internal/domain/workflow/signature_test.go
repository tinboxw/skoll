package workflow

import (
	"strings"
	"testing"
	"time"
)

func TestRequiredSignatureCreatesVerifiableImmutableReceipt(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	definition, err := sampleDefinition(now)
	if err != nil {
		t.Fatal(err)
	}
	for index := range definition.Nodes {
		if definition.Nodes[index].Type == NodeApproval {
			definition.Nodes[index].Signature = &SignaturePolicy{
				Meaning:         "I approve this regulated decision.",
				RequireEvidence: true,
			}
		}
	}
	instance, err := Start(StartInput{
		ID: "signed-instance", Definition: *definition, BusinessType: "regulated",
		BusinessID: "record-1", Title: "Signed approval", Starter: Actor{ID: "starter"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	task := instance.Tasks[0]
	if err := instance.Approve(*definition, task.ID, task.Assignee, "approved", nil, now.Add(time.Minute)); err == nil {
		t.Fatal("required signature must fail without evidence")
	}
	signature := &DecisionSignature{
		VerificationID: "verify-1", Audience: "plugin:medical_oa", Method: "password",
		Meaning: "I approve this regulated decision.", VerifiedAt: now.Add(30 * time.Second),
		Evidence: []EvidenceReference{{FileID: "file-1", Name: "evidence.pdf", Hash: strings.Repeat("a", 64), Size: 128, MIME: "application/pdf"}},
	}
	if err := instance.Approve(*definition, task.ID, task.Assignee, "approved", signature, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(instance.Receipts) != 1 || instance.Timeline[len(instance.Timeline)-1].ReceiptID != instance.Receipts[0].ID {
		t.Fatalf("signature correlation is incomplete: %+v", instance)
	}
	receipt := instance.Receipts[0]
	if receipt.AuditCorrelationID != receipt.ID || receipt.VerificationID != "verify-1" || len(receipt.Evidence) != 1 {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if err := receipt.VerifyDigest(); err != nil {
		t.Fatal(err)
	}
	if err := instance.ValidateSignatureEvidence(); err != nil {
		t.Fatal(err)
	}
	instance.Timeline[len(instance.Timeline)-1].Comment = "tampered action"
	if err := instance.ValidateSignatureEvidence(); err == nil {
		t.Fatal("tampered workflow action must fail signature correlation")
	}
	receipt.Meaning = "tampered"
	if err := receipt.VerifyDigest(); err == nil {
		t.Fatal("tampered receipt must fail digest verification")
	}
}

func TestSignaturePolicyRejectsWrongMeaningAndReusedVerification(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	definition, err := sampleDefinition(now)
	if err != nil {
		t.Fatal(err)
	}
	for index := range definition.Nodes {
		if definition.Nodes[index].Type == NodeApproval {
			definition.Nodes[index].Signature = &SignaturePolicy{Meaning: "I approve."}
		}
	}
	instance, err := Start(StartInput{
		ID: "signed-quorum", Definition: *definition, BusinessType: "regulated",
		BusinessID: "record-2", Title: "Signed quorum", Starter: Actor{ID: "starter"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	first := &DecisionSignature{VerificationID: "verify-shared", Audience: "core", Method: "password", Meaning: "wrong", VerifiedAt: now}
	if err := instance.Approve(*definition, instance.Tasks[0].ID, instance.Tasks[0].Assignee, "", first, now.Add(time.Minute)); err == nil {
		t.Fatal("wrong signing meaning must fail")
	}
	first.Meaning = "I approve."
	if err := instance.Approve(*definition, instance.Tasks[0].ID, instance.Tasks[0].Assignee, "", first, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(instance.Tasks) < 2 {
		t.Skip("definition has only one approval task")
	}
	if err := instance.Approve(*definition, instance.Tasks[1].ID, instance.Tasks[1].Assignee, "", first, now.Add(2*time.Minute)); err == nil {
		t.Fatal("one reverification proof must not authorize two decisions")
	}
}
