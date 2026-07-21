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
	if err := DataScopeDepartmentTree.Validate(); err != nil {
		t.Fatalf("unexpected scope error: %v", err)
	}
	if got := NormalizeDataScope("department"); got != DataScopeDepartment {
		t.Fatalf("expected department scope, got %s", got)
	}
	if err := DataScope("bad_scope").Validate(); err == nil {
		t.Fatalf("expected scope validation error")
	}
}

func TestPolicyRuleNormalized(t *testing.T) {
	rule := PolicyRule{Resource: " user:* ", Action: " read ", Effect: EffectAllow, Scope: DataScope("department")}
	normalized := rule.Normalized()
	if normalized.Resource != "user:*" || normalized.Action != "read" || normalized.Scope != DataScopeDepartment {
		t.Fatalf("unexpected normalized rule: %+v", normalized)
	}
}
