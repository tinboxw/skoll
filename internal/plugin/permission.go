package plugin

import "sync"

type RolePermission struct {
	AllowedAPIs []string
	DeniedAPIs  []string
}

type PluginPermission struct {
	PluginID  string
	RoleRules map[string]RolePermission
}

type PermissionChecker interface {
	Set(rule PluginPermission)
	Allow(pluginID, api, role string) bool
}

type RuleBasedPermissionChecker struct {
	mu    sync.RWMutex
	rules map[string]PluginPermission
}

func NewRuleBasedPermissionChecker() *RuleBasedPermissionChecker {
	return &RuleBasedPermissionChecker{rules: make(map[string]PluginPermission)}
}

func (c *RuleBasedPermissionChecker) Set(rule PluginPermission) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules[rule.PluginID] = rule
}

func (c *RuleBasedPermissionChecker) Allow(pluginID, api, role string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rule, ok := c.rules[pluginID]
	if !ok {
		return false
	}

	roleRule, ok := rule.RoleRules[role]
	if !ok {
		return false
	}

	if contains(roleRule.DeniedAPIs, api) || contains(roleRule.DeniedAPIs, "*") {
		return false
	}

	return contains(roleRule.AllowedAPIs, api) || contains(roleRule.AllowedAPIs, "*")
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
