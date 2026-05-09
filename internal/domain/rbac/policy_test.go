package rbac

import "testing"

func TestPolicyRuleAllowsWildcard(t *testing.T) {
	r := PolicyRule{Resource: "user:*", Action: "read", Effect: EffectAllow, Scope: DataScopeAll}
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if !r.Allows("user:profile", "read") {
		t.Fatalf("expected rule to allow")
	}
	if r.Allows("role:item", "read") {
		t.Fatalf("expected rule not to match other resource")
	}
}

func TestDataScopeValidate(t *testing.T) {
	if err := DataScopeDeptTree.Validate(); err != nil {
		t.Fatalf("unexpected scope error: %v", err)
	}
	if err := DataScope("bad_scope").Validate(); err == nil {
		t.Fatalf("expected scope validation error")
	}
}
