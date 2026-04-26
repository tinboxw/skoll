package modgenerator

import "testing"

func TestServiceGenerate(t *testing.T) {
	svc := NewService()
	result, err := svc.Generate("account")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if result.Module != "account" {
		t.Fatalf("unexpected module name: %s", result.Module)
	}
	if len(result.Artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(result.Artifacts))
	}
}

func TestServiceGenerate_InvalidName(t *testing.T) {
	svc := NewService()
	if _, err := svc.Generate("Account-Module"); err == nil {
		t.Fatalf("expected invalid module name error")
	}
}
