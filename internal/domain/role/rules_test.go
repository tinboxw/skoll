package role

import (
	"reflect"
	"testing"
)

func TestNormalizePermissions(t *testing.T) {
	in := []string{" user:read ", "USER:READ", "user:write", ""}
	got := NormalizePermissions(in)
	want := []string{"user:read", "user:write"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizePermissions() = %#v, want %#v", got, want)
	}
}

func TestValidateKey(t *testing.T) {
	if err := ValidateKey("admin_root"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateKey("Bad-Key"); err == nil {
		t.Fatalf("expected invalid key error")
	}
}
