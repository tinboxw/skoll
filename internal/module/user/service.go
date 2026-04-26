package user

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID        int64
	Name      string
	Email     string
	Active    bool
	CreatedAt time.Time
}

type Service struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]User
}

func NewService() *Service {
	return &Service{nextID: 1, items: make(map[int64]User)}
}

func (s *Service) Create(name, email string) User {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := User{
		ID:        s.nextID,
		Name:      name,
		Email:     email,
		Active:    true,
		CreatedAt: time.Now().UTC(),
	}
	s.nextID++
	s.items[u.ID] = u
	return u
}

func (s *Service) Get(id int64) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.items[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (s *Service) List() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]User, 0, len(s.items))
	for _, u := range s.items {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
