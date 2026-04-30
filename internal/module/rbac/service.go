package rbac

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var ErrPolicySnapshotNotFound = errors.New("policy snapshot not found")

type Service struct {
	mu          sync.RWMutex
	roleMenu    map[int64][]int64
	roleAPI     map[int64][]string
	policies    map[int64][]PolicyRule
	snapshotSeq map[int64]int64
	snapshots   map[int64][]PolicySnapshot
	data        map[int64]DataScope
	route       map[int64]RoutePermissionContract
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

type RoutePermissionItem struct {
	MenuID  int64
	Route   string
	Buttons []string
}

type RoutePermissionContract struct {
	Version string
	Items   []RoutePermissionItem
}

type RoutePermissionConsistency struct {
	Passed   bool
	Problems []string
}

type PolicySnapshot struct {
	RoleID  int64
	Version string
	Rules   []PolicyRule
}

func NewService() *Service {
	return &Service{
		roleMenu:    make(map[int64][]int64),
		roleAPI:     make(map[int64][]string),
		policies:    make(map[int64][]PolicyRule),
		snapshotSeq: make(map[int64]int64),
		snapshots:   make(map[int64][]PolicySnapshot),
		data:        make(map[int64]DataScope),
		route:       make(map[int64]RoutePermissionContract),
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

func (s *Service) CreateRolePolicySnapshot(roleID int64) PolicySnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	rules := clonePolicyRules(s.policies[roleID])
	next := s.snapshotSeq[roleID] + 1
	s.snapshotSeq[roleID] = next
	snapshot := PolicySnapshot{RoleID: roleID, Version: "v" + strconv.FormatInt(next, 10), Rules: rules}
	s.snapshots[roleID] = append(s.snapshots[roleID], snapshot)
	return clonePolicySnapshot(snapshot)
}

func (s *Service) ListRolePolicySnapshots(roleID int64) []PolicySnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	raw := s.snapshots[roleID]
	if len(raw) == 0 {
		return nil
	}
	out := make([]PolicySnapshot, len(raw))
	for i := range raw {
		out[i] = clonePolicySnapshot(raw[i])
	}
	return out
}

func (s *Service) RollbackRolePolicies(roleID int64, version string) ([]PolicyRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	version = strings.ToLower(strings.TrimSpace(version))
	for _, snapshot := range s.snapshots[roleID] {
		if snapshot.Version != version {
			continue
		}
		rules := clonePolicyRules(snapshot.Rules)
		s.policies[roleID] = rules
		return clonePolicyRules(rules), nil
	}
	return nil, ErrPolicySnapshotNotFound
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

func (s *Service) SetRoleRoutePermissions(roleID int64, version string, items []RoutePermissionItem) RoutePermissionContract {
	normalized := normalizeRouteContract(version, items)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.route[roleID] = normalized
	return cloneRouteContract(normalized)
}

func (s *Service) GetRoleRoutePermissions(roleID int64) RoutePermissionContract {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneRouteContract(s.route[roleID])
}

func (s *Service) CheckRoleRoutePermissionConsistency(roleID int64) RoutePermissionConsistency {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contract := s.route[roleID]
	menus := s.roleMenu[roleID]
	menuSet := make(map[int64]struct{}, len(menus))
	for _, id := range menus {
		menuSet[id] = struct{}{}
	}

	problems := make([]string, 0)
	for _, item := range contract.Items {
		if item.MenuID > 0 {
			if _, ok := menuSet[item.MenuID]; !ok {
				problems = append(problems, "menu_id "+strconv.FormatInt(item.MenuID, 10)+" is not bound to role")
			}
		}
		if item.Route == "" {
			problems = append(problems, "route cannot be empty")
		}
	}

	return RoutePermissionConsistency{Passed: len(problems) == 0, Problems: problems}
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

func clonePolicySnapshot(snapshot PolicySnapshot) PolicySnapshot {
	return PolicySnapshot{RoleID: snapshot.RoleID, Version: snapshot.Version, Rules: clonePolicyRules(snapshot.Rules)}
}

func cloneDataScope(scope DataScope) DataScope {
	return DataScope{
		TenantIDs:         append([]string(nil), scope.TenantIDs...),
		RequireOwnerMatch: scope.RequireOwnerMatch,
	}
}

func normalizeRouteContract(version string, items []RoutePermissionItem) RoutePermissionContract {
	version = strings.ToLower(strings.TrimSpace(version))
	if version == "" {
		version = "v1"
	}
	if version != "v1" && version != "v2" {
		version = "v1"
	}

	seen := make(map[string]struct{}, len(items))
	out := make([]RoutePermissionItem, 0, len(items))
	for _, item := range items {
		route := strings.TrimSpace(item.Route)
		if route == "" {
			continue
		}
		buttons := normalizeStringList(item.Buttons)
		key := strconv.FormatInt(item.MenuID, 10) + "|" + route + "|" + strings.Join(buttons, ",")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, RoutePermissionItem{MenuID: item.MenuID, Route: route, Buttons: buttons})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MenuID == out[j].MenuID {
			return out[i].Route < out[j].Route
		}
		return out[i].MenuID < out[j].MenuID
	})

	return RoutePermissionContract{Version: version, Items: out}
}

func cloneRouteContract(contract RoutePermissionContract) RoutePermissionContract {
	out := RoutePermissionContract{Version: contract.Version}
	if len(contract.Items) == 0 {
		return out
	}
	out.Items = make([]RoutePermissionItem, len(contract.Items))
	for i, item := range contract.Items {
		out.Items[i] = RoutePermissionItem{
			MenuID:  item.MenuID,
			Route:   item.Route,
			Buttons: append([]string(nil), item.Buttons...),
		}
	}
	return out
}
