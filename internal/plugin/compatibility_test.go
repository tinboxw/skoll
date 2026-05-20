package plugin

import "testing"

func TestInfoValidateCompatibility(t *testing.T) {
	info := Info{CompatibilitySkoll: ">=1.0.0 <2.0.0"}
	if err := info.ValidateCompatibility("1.4.2"); err != nil {
		t.Fatalf("expected compatibility pass, got %v", err)
	}
	if err := info.ValidateCompatibility("v2.0.0"); err == nil {
		t.Fatalf("expected compatibility failure for 2.0.0")
	}
}

func TestInfoValidateCompatibilityExpressionError(t *testing.T) {
	info := Info{CompatibilitySkoll: ">=1.0"}
	if err := info.ValidateCompatibility("1.0.0"); err == nil {
		t.Fatalf("expected invalid compatibility expression error")
	}
}

func TestInfoValidateCompatibilityInvalidCoreVersion(t *testing.T) {
	info := Info{CompatibilitySkoll: ">=1.0.0 <2.0.0"}
	if err := info.ValidateCompatibility("dev"); err == nil {
		t.Fatalf("expected invalid core version error")
	}
}
