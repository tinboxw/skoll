package rbac

import (
	"strings"

	"github.com/tinboxw/skoll/internal/module/rbac"
)

func toRolePermissionContractItems(items []rbac.RoutePermissionItem) []rolePermissionContractItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]rolePermissionContractItem, len(items))
	for i, item := range items {
		out[i] = rolePermissionContractItem{MenuID: item.MenuID, Route: item.Route, Buttons: append([]string(nil), item.Buttons...)}
	}
	return out
}

func toRolePolicyRuleItems(rules []rbac.PolicyRule) []rolePolicyRuleItem {
	if len(rules) == 0 {
		return nil
	}
	out := make([]rolePolicyRuleItem, len(rules))
	for i, rule := range rules {
		out[i] = rolePolicyRuleItem{
			API:                  rule.API,
			Effect:               rule.Effect,
			RequireVerified:      rule.RequireVerified,
			RequireClaimsVersion: rule.RequireClaimsVersion,
		}
	}
	return out
}

func toRolePolicyRules(items []rolePolicyRuleItem) []rbac.PolicyRule {
	if len(items) == 0 {
		return nil
	}
	out := make([]rbac.PolicyRule, len(items))
	for i, item := range items {
		out[i] = rbac.PolicyRule{
			API:                  strings.TrimSpace(item.API),
			Effect:               strings.TrimSpace(item.Effect),
			RequireVerified:      item.RequireVerified,
			RequireClaimsVersion: strings.TrimSpace(item.RequireClaimsVersion),
		}
	}
	return out
}

func toRoutePermissionItems(items []rolePermissionContractItem) []rbac.RoutePermissionItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]rbac.RoutePermissionItem, len(items))
	for i, item := range items {
		out[i] = rbac.RoutePermissionItem{MenuID: item.MenuID, Route: strings.TrimSpace(item.Route), Buttons: append([]string(nil), item.Buttons...)}
	}
	return out
}

func toRolePermissionDiffResponse(roleID int64, diff rbac.PermissionDiff) rolePermissionDiffResponse {
	return rolePermissionDiffResponse{
		RoleID:           roleID,
		AddedMenus:       append([]int64(nil), diff.AddedMenus...),
		RemovedMenus:     append([]int64(nil), diff.RemovedMenus...),
		AddedAPIs:        append([]string(nil), diff.AddedAPIs...),
		RemovedAPIs:      append([]string(nil), diff.RemovedAPIs...),
		AddedRules:       toRolePolicyRuleItems(diff.AddedPolicies),
		RemovedRules:     toRolePolicyRuleItems(diff.RemovedPolicies),
		DataScopeChanged: diff.DataScopeChanged,
		RouteChanged:     diff.RouteChanged,
	}
}
