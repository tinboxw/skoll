package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type RoleStore struct {
	mu    sync.RWMutex
	items map[shared.ID]*role.Role
}

func NewRoleStore() *RoleStore {
	return &RoleStore{items: make(map[shared.ID]*role.Role)}
}

func (s *RoleStore) GetByID(_ context.Context, id shared.ID) (*role.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items[id], nil
}

func (s *RoleStore) GetByKey(_ context.Context, key string) (*role.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	want := strings.ToLower(strings.TrimSpace(key))
	for _, r := range s.items {
		if r != nil && r.Key == want {
			return r, nil
		}
	}
	return nil, nil
}

func (s *RoleStore) List(_ context.Context, offset, limit int) ([]*role.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.items))
	for id := range s.items {
		ids = append(ids, id.String())
	}
	sort.Strings(ids)

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(ids)
	}
	if offset > len(ids) {
		return []*role.Role{}, nil
	}
	end := offset + limit
	if end > len(ids) {
		end = len(ids)
	}

	out := make([]*role.Role, 0, end-offset)
	for _, id := range ids[offset:end] {
		out = append(out, s.items[shared.ID(id)])
	}
	return out, nil
}

func (s *RoleStore) Save(_ context.Context, entity *role.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[entity.ID] = entity
	return nil
}

func (s *RoleStore) Delete(_ context.Context, id shared.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)
	return nil
}
