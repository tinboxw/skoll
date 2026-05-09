package security

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("p@ssw0rd")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !VerifyPassword(hash, "p@ssw0rd") {
		t.Fatalf("expected password verification success")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatalf("expected password verification failure")
	}
}
