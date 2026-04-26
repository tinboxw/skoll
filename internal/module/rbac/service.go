package rbac

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Service struct {
	mu       sync.RWMutex
	roleMenu map[int64][]int64
	roleAPI  map[int64][]string
	policies map[int64][]PolicyRule
	data     map[int64]DataScope
}

type PolicyRule struct {
	API                  string
	Effect               string
	RequireVerified      bool
	RequireClaimsVersion string
}

type DataScope struct {
	TenantIDs         []string
	RequireOwnerMatch bool
}

func NewService() *Service {
	return &Service{
		roleMenu: make(map[int64][]int64),
		roleAPI:  make(map[int64][]string),
		policies: make(map[int64][]PolicyRule),
		data:     make(map[int64]DataScope),
	}
}

func (s *Service) SetRoleMenus(roleID int64, menuIDs []int64) []int64 {
	normalized := normalizeInt64List(menuIDs)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.roleMenu[roleID] = normalized
	return append([]int64(nil), normalized...)
}

func (s *Service) GetRoleMenus(roleID int64) []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]int64(nil), s.roleMenu[roleID]...)
}

func (s *Service) SetRoleAPIs(roleID int64, apis []string) []string {
	normalized := normalizeStringList(apis)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.roleAPI[roleID] = normalized
	return append([]string(nil), normalized...)
}

func (s *Service) GetRoleAPIs(roleID int64) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.roleAPI[roleID]...)
}

func (s *Service) SetRolePolicies(roleID int64, rules []PolicyRule) []PolicyRule {
	normalized := normalizePolicyRules(rules)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[roleID] = normalized
	return clonePolicyRules(normalized)
}

func (s *Service) GetRolePolicies(roleID int64) []PolicyRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clonePolicyRules(s.policies[roleID])
}

func (s *Service) SetRoleDataScope(roleID int64, scope DataScope) DataScope {
	normalized := normalizeDataScope(scope)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[roleID] = normalized
	return cloneDataScope(normalized)
}

func (s *Service) GetRoleDataScope(roleID int64) DataScope {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneDataScope(s.data[roleID])
}

func normalizeInt64List(raw []int64) []int64 {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(raw))
	out := make([]int64, 0, len(raw))
	for _, v := range raw {
		if v <= 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func normalizeStringList(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func normalizePolicyRules(raw []PolicyRule) []PolicyRule {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]PolicyRule, 0, len(raw))
	for _, rule := range raw {
		api := strings.TrimSpace(rule.API)
		if api == "" {
			continue
		}
		effect := strings.ToLower(strings.TrimSpace(rule.Effect))
		if effect != "allow" && effect != "deny" {
			continue
		}
		key := effect + "|" + api + "|" + strconv.FormatBool(rule.RequireVerified) + "|" + strings.ToLower(strings.TrimSpace(rule.RequireClaimsVersion))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, PolicyRule{
			API:                  api,
			Effect:               effect,
			RequireVerified:      rule.RequireVerified,
			RequireClaimsVersion: strings.ToLower(strings.TrimSpace(rule.RequireClaimsVersion)),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].API == out[j].API {
			return out[i].Effect < out[j].Effect
		}
		return out[i].API < out[j].API
	})
	return out
}

func normalizeDataScope(scope DataScope) DataScope {
	return DataScope{
		TenantIDs:         normalizeStringList(scope.TenantIDs),
		RequireOwnerMatch: scope.RequireOwnerMatch,
	}
}

func clonePolicyRules(raw []PolicyRule) []PolicyRule {
	if len(raw) == 0 {
		return nil
	}
	out := make([]PolicyRule, len(raw))
	copy(out, raw)
	return out
}

func cloneDataScope(scope DataScope) DataScope {
	return DataScope{
		TenantIDs:         append([]string(nil), scope.TenantIDs...),
		RequireOwnerMatch: scope.RequireOwnerMatch,
	}
}
