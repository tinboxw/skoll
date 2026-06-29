package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
)

type UserStore struct {
	mu    sync.RWMutex
	items map[shared.ID]*user.User
}

func NewUserStore() *UserStore {
	return &UserStore{items: make(map[shared.ID]*user.User)}
}

func (s *UserStore) GetByID(_ context.Context, id shared.ID) (*user.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items[id], nil
}

func (s *UserStore) GetByAccount(_ context.Context, account string) (*user.User, error) {
	target := strings.TrimSpace(strings.ToLower(account))
	if target == "" {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.items {
		if u != nil && strings.ToLower(strings.TrimSpace(u.Account)) == target {
			return u, nil
		}
	}
	return nil, nil
}

func (s *UserStore) GetByEmail(_ context.Context, email user.Email) (*user.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.items {
		if u != nil && u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (s *UserStore) List(_ context.Context, offset, limit int) ([]*user.User, error) {
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

	end := offset + limit
	if offset > len(ids) {
		return []*user.User{}, nil
	}
	if end > len(ids) {
		end = len(ids)
	}

	out := make([]*user.User, 0, end-offset)
	for _, id := range ids[offset:end] {
		out = append(out, s.items[shared.ID(id)])
	}
	return out, nil
}

func (s *UserStore) ListFiltered(ctx context.Context, filter userrepo.ListFilter, offset, limit int) ([]*user.User, error) {
	if filter.Empty() {
		return s.List(ctx, offset, limit)
	}

	userIDs := make(map[shared.ID]struct{})
	for _, id := range filter.NormalizedUserIDs() {
		userIDs[id] = struct{}{}
	}
	departmentIDs := make(map[string]struct{})
	for _, id := range filter.NormalizedDepartmentIDs() {
		departmentIDs[id] = struct{}{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.items))
	for id, u := range s.items {
		if u == nil {
			continue
		}
		if _, ok := userIDs[id]; ok {
			ids = append(ids, id.String())
			continue
		}
		if _, ok := departmentIDs[u.DepartmentID]; ok {
			ids = append(ids, id.String())
		}
	}
	sort.Strings(ids)

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(ids)
	}

	end := offset + limit
	if offset > len(ids) {
		return []*user.User{}, nil
	}
	if end > len(ids) {
		end = len(ids)
	}

	out := make([]*user.User, 0, end-offset)
	for _, id := range ids[offset:end] {
		out = append(out, s.items[shared.ID(id)])
	}
	return out, nil
}

func (s *UserStore) Save(_ context.Context, entity *user.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[entity.ID] = entity
	return nil
}

func (s *UserStore) Delete(_ context.Context, id shared.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)
	return nil
}
