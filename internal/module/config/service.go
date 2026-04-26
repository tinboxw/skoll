package config

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrConfigNotFound = errors.New("config not found")

type Entry struct {
	Key         string
	Value       string
	Description string
	UpdatedAt   time.Time
}

type Service struct {
	mu    sync.RWMutex
	items map[string]Entry
}

func NewService() *Service {
	return &Service{items: make(map[string]Entry)}
}

func (s *Service) Set(key, value, description string) Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := Entry{
		Key:         key,
		Value:       value,
		Description: description,
		UpdatedAt:   time.Now().UTC(),
	}
	s.items[key] = item
	return item
}

func (s *Service) Get(key string) (Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[key]
	if !ok {
		return Entry{}, ErrConfigNotFound
	}
	return item, nil
}

func (s *Service) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Entry, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
