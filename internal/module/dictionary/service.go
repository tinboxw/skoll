package dictionary

import (
	"errors"
	"sort"
	"sync"
)

var ErrItemNotFound = errors.New("dictionary item not found")

type Item struct {
	ID      int64
	Type    string
	Label   string
	Value   string
	Sort    int
	Enabled bool
}

type Service struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]Item
}

func NewService() *Service {
	return &Service{nextID: 1, items: make(map[int64]Item)}
}

func (s *Service) Create(itemType, label, value string, sortOrder int, enabled bool) Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := Item{
		ID:      s.nextID,
		Type:    itemType,
		Label:   label,
		Value:   value,
		Sort:    sortOrder,
		Enabled: enabled,
	}
	s.nextID++
	s.items[item.ID] = item
	return item
}

func (s *Service) Get(id int64) (Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[id]
	if !ok {
		return Item{}, ErrItemNotFound
	}
	return item, nil
}

func (s *Service) List() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	sortItems(out)
	return out
}

func (s *Service) ListByType(itemType string) []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		if item.Type != itemType {
			continue
		}
		out = append(out, item)
	}
	sortItems(out)
	return out
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Type == items[j].Type {
			if items[i].Sort == items[j].Sort {
				return items[i].ID < items[j].ID
			}
			return items[i].Sort < items[j].Sort
		}
		return items[i].Type < items[j].Type
	})
}
