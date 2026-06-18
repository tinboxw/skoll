package menu

import (
	"sort"
	"strings"
)

type AccessContext struct {
	roles       map[string]struct{}
	permissions map[string]struct{}
}

func NewAccessContext(roles []string, permissions []string) AccessContext {
	return AccessContext{
		roles:       keySet(roles),
		permissions: keySet(permissions),
	}
}

func SortSiblings(nodes []MenuNode) []MenuNode {
	sorted := append([]MenuNode(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Sort < sorted[j].Sort
	})
	return sorted
}

func FilterVisible(nodes []MenuNode) []MenuNode {
	filtered := make([]MenuNode, 0, len(nodes))
	for _, node := range nodes {
		if node.Visible {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func FilterAuthorized(nodes []MenuNode, access AccessContext) []MenuNode {
	filtered := make([]MenuNode, 0, len(nodes))
	for _, node := range nodes {
		if access.Allows(node) {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

func (a AccessContext) Allows(node MenuNode) bool {
	return containsAll(a.roles, node.RequiredRoles) &&
		containsAll(a.permissions, node.RequiredPermissions)
}

func keySet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		v := strings.TrimSpace(strings.ToLower(value))
		if v != "" {
			set[v] = struct{}{}
		}
	}
	return set
}

func containsAll(set map[string]struct{}, required []string) bool {
	for _, value := range required {
		v := strings.TrimSpace(strings.ToLower(value))
		if v == "" {
			continue
		}
		if _, ok := set[v]; !ok {
			return false
		}
	}
	return true
}
