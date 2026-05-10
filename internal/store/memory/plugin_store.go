package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/tinboxw/skoll/internal/plugin"
)

type PluginStore struct {
	mu    sync.RWMutex
	items map[string]plugin.Info
}

func NewPluginStore() *PluginStore {
	return &PluginStore{items: make(map[string]plugin.Info)}
}

func (s *PluginStore) Get(_ context.Context, pluginID string) (*plugin.Info, error) {
	key := strings.TrimSpace(pluginID)
	if key == "" {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	result := item
	return &result, nil
}

func (s *PluginStore) List(_ context.Context) ([]plugin.Info, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]plugin.Info, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (s *PluginStore) Save(_ context.Context, info plugin.Info) error {
	key := strings.TrimSpace(info.ID)
	if key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = info
	return nil
}

func (s *PluginStore) Delete(_ context.Context, pluginID string) error {
	key := strings.TrimSpace(pluginID)
	if key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}
