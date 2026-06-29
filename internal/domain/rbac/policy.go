package rbac

import (
	"fmt"
	"strings"
)

type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

type PolicyRule struct {
	Resource string
	Action   string
	Effect   Effect
	Scope    DataScope
}

func (p PolicyRule) Validate() error {
	if strings.TrimSpace(p.Resource) == "" {
		return fmt.Errorf("resource is required")
	}
	if strings.TrimSpace(p.Action) == "" {
		return fmt.Errorf("action is required")
	}
	if p.Effect != EffectAllow && p.Effect != EffectDeny {
		return fmt.Errorf("effect must be allow or deny")
	}
	return NormalizeDataScope(p.Scope).Validate()
}

func (p PolicyRule) Normalized() PolicyRule {
	p.Resource = strings.TrimSpace(p.Resource)
	p.Action = strings.TrimSpace(p.Action)
	p.Scope = NormalizeDataScope(p.Scope)
	return p
}

func (p PolicyRule) Matches(resource, action string) bool {
	return wildcardMatch(p.Resource, strings.TrimSpace(resource)) && wildcardMatch(p.Action, strings.TrimSpace(action))
}

func (p PolicyRule) Allows(resource, action string) bool {
	return p.Effect == EffectAllow && p.Matches(resource, action)
}

func wildcardMatch(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	value = strings.TrimSpace(value)
	if pattern == "*" {
		return true
	}
	if pattern == value {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}
	return false
}
