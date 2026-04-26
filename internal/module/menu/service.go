package menu

import (
	"errors"
	"sort"
	"sync"
)

var ErrMenuNotFound = errors.New("menu not found")

type Item struct {
	ID     int64
	Title  string
	Path   string
	Order  int
	Hidden bool
}

type Service struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]Item
}

func NewService() *Service {
	return &Service{nextID: 1, items: make(map[int64]Item)}
}

func (s *Service) Create(title, path string, order int) Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	it := Item{ID: s.nextID, Title: title, Path: path, Order: order}
	s.nextID++
	s.items[it.ID] = it
	return it
}

func (s *Service) Get(id int64) (Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	it, ok := s.items[id]
	if !ok {
		return Item{}, ErrMenuNotFound
	}
	return it, nil
}

func (s *Service) List() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Item, 0, len(s.items))
	for _, it := range s.items {
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].ID < out[j].ID
		}
		return out[i].Order < out[j].Order
	})
	return out
}
