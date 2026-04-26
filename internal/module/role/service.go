package role

import (
	"errors"
	"sort"
	"sync"
)

var ErrRoleNotFound = errors.New("role not found")

type Role struct {
	ID          int64
	Name        string
	Permissions []string
}

type Service struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]Role
}

func NewService() *Service {
	return &Service{nextID: 1, items: make(map[int64]Role)}
}

func (s *Service) Create(name string, permissions []string) Role {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := Role{ID: s.nextID, Name: name, Permissions: append([]string(nil), permissions...)}
	s.nextID++
	s.items[r.ID] = r
	return r
}

func (s *Service) Get(id int64) (Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.items[id]
	if !ok {
		return Role{}, ErrRoleNotFound
	}
	return r, nil
}

func (s *Service) List() []Role {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Role, 0, len(s.items))
	for _, r := range s.items {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
