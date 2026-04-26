package pluginmgr

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrPluginNotFound = errors.New("plugin not found")

type Manifest struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Hooks       []string  `json:"hooks"`
	Enabled     bool      `json:"enabled"`
	InstalledAt time.Time `json:"installed_at"`
}

type Service struct {
	mu    sync.RWMutex
	items map[string]Manifest
}

func NewService() *Service {
	return &Service{items: make(map[string]Manifest)}
}

func (s *Service) Install(name, version string, hooks []string) Manifest {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	item := Manifest{
		Name:        name,
		Version:     version,
		Hooks:       append([]string(nil), hooks...),
		Enabled:     true,
		InstalledAt: time.Now().UTC(),
	}
	s.items[name] = item
	return item
}

func (s *Service) Get(name string) (Manifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[strings.TrimSpace(name)]
	if !ok {
		return Manifest{}, ErrPluginNotFound
	}
	return item, nil
}

func (s *Service) List() []Manifest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Manifest, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) Enable(name string) (Manifest, error) {
	return s.setEnabled(name, true)
}

func (s *Service) Disable(name string) (Manifest, error) {
	return s.setEnabled(name, false)
}

func (s *Service) setEnabled(name string, enabled bool) (Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.TrimSpace(name)
	item, ok := s.items[key]
	if !ok {
		return Manifest{}, ErrPluginNotFound
	}
	item.Enabled = enabled
	s.items[key] = item
	return item, nil
}
