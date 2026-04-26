package apiregistry

import (
	"sort"
	"strings"
	"sync"
)

type Service struct {
	mu    sync.RWMutex
	items map[string]struct{}
}

func NewService() *Service {
	return &Service{items: make(map[string]struct{})}
}

func (s *Service) RegisterMany(entries []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, entry := range entries {
		normalized := Normalize(entry)
		if normalized == "" {
			continue
		}
		s.items[normalized] = struct{}{}
	}
}

func (s *Service) Exists(entry string) bool {
	normalized := Normalize(entry)
	if normalized == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.items[normalized]
	return ok
}

func (s *Service) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.items))
	for k := range s.items {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func Normalize(entry string) string {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return ""
	}

	if strings.Contains(entry, " ") {
		parts := strings.Fields(entry)
		if len(parts) != 2 {
			return ""
		}
		method := parts[0]
		path := parts[1]
		if !strings.HasPrefix(path, "/") {
			return ""
		}
		return strings.ToUpper(method) + ":" + path
	}

	if !strings.Contains(entry, ":") {
		return ""
	}
	parts := strings.SplitN(entry, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	method := strings.TrimSpace(parts[0])
	path := strings.TrimSpace(parts[1])
	if method == "" || !strings.HasPrefix(path, "/") {
		return ""
	}
	return strings.ToUpper(method) + ":" + path
}
