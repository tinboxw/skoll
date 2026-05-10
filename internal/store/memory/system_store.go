package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/system"
)

type SystemStore struct {
	mu    sync.RWMutex
	items map[shared.ID]*system.Setting
	byKey map[string]shared.ID
}

func NewSystemStore() *SystemStore {
	return &SystemStore{
		items: make(map[shared.ID]*system.Setting),
		byKey: make(map[string]shared.ID),
	}
}

func (s *SystemStore) GetSettingByID(_ context.Context, id shared.ID) (*system.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items[id], nil
}

func (s *SystemStore) GetSettingByKey(_ context.Context, key string) (*system.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byKey[normalizeSettingKey(key)]
	if !ok {
		return nil, nil
	}
	return s.items[id], nil
}

func (s *SystemStore) ListSettings(_ context.Context, offset, limit int) ([]*system.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.byKey))
	for key := range s.byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(keys)
	}
	if offset > len(keys) {
		return []*system.Setting{}, nil
	}
	end := offset + limit
	if end > len(keys) {
		end = len(keys)
	}

	out := make([]*system.Setting, 0, end-offset)
	for _, key := range keys[offset:end] {
		out = append(out, s.items[s.byKey[key]])
	}
	return out, nil
}

func (s *SystemStore) SaveSetting(_ context.Context, setting *system.Setting) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[setting.ID] = setting
	s.byKey[normalizeSettingKey(setting.Key)] = setting.ID
	return nil
}

func (s *SystemStore) DeleteSetting(_ context.Context, id shared.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if ok && item != nil {
		delete(s.byKey, normalizeSettingKey(item.Key))
	}
	delete(s.items, id)
	return nil
}

func normalizeSettingKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
