package rbac

import (
	"sort"
	"strings"
	"sync"
)

type Service struct {
	mu       sync.RWMutex
	roleMenu map[int64][]int64
	roleAPI  map[int64][]string
}

func NewService() *Service {
	return &Service{
		roleMenu: make(map[int64][]int64),
		roleAPI:  make(map[int64][]string),
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
