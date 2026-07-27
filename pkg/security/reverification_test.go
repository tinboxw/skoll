package security

import (
	"strings"
	"testing"
	"time"
)

func TestReverificationProofBindsIdentityAudienceAndExpiry(t *testing.T) {
	service, err := NewReverificationProofService("test-reverification-secret")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	token, issued, err := service.IssueReverificationProof(
		"user-1", "plugin:medical_oa", ReverificationPurposeWorkflowSignature, "password", now, 2*time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := service.VerifyReverificationProof(token, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if verified != issued || !strings.HasPrefix(verified.ID, "verify-") {
		t.Fatalf("verified claims = %+v, issued = %+v", verified, issued)
	}
	if _, err := service.VerifyReverificationProof(token, issued.ExpiresAt); err == nil {
		t.Fatal("expired proof must fail")
	}
}

func TestReverificationProofRejectsTamperingAndInvalidPolicy(t *testing.T) {
	service, err := NewReverificationProofService("test-reverification-secret")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	token, _, err := service.IssueReverificationProof(
		"user-1", "core", ReverificationPurposeWorkflowSignature, "password", now, time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	tampered := token[:len(token)-1] + "0"
	if _, err := service.VerifyReverificationProof(tampered, now); err == nil {
		t.Fatal("tampered proof must fail")
	}
	if _, _, err := service.IssueReverificationProof("user-1", "core", "login", "password", now, time.Minute); err == nil {
		t.Fatal("unsupported purpose must fail")
	}
	if _, _, err := service.IssueReverificationProof("user-1", "core", ReverificationPurposeWorkflowSignature, "password", now, 6*time.Minute); err == nil {
		t.Fatal("overlong proof lifetime must fail")
	}
}
