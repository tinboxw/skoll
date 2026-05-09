package role

import "testing"

func TestValidateName(t *testing.T) {
	if err := ValidateName("x"); err == nil {
		t.Fatalf("expected error for too short name")
	}
	if err := ValidateName("operator"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
