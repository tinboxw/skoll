package notification

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

type Category string

const (
	CategoryTodo     Category = "todo"
	CategoryMessage  Category = "message"
	CategoryReminder Category = "reminder"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusDone    Status = "done"
	StatusRead    Status = "read"
)

type Target struct {
	Type string
	ID   string
	Path string
}

type Item struct {
	ID        string
	Category  Category
	Status    Status
	Title     string
	Body      string
	ActorID   string
	Target    Target
	DueAt     time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateInput struct {
	ID       string
	Category Category
	Title    string
	Body     string
	ActorID  string
	Target   Target
	DueAt    time.Time
}

type Filter struct {
	ActorID  string
	Category Category
	Status   Status
}

type ReminderRule struct {
	ID       string
	Name     string
	ActorID  string
	Title    string
	Body     string
	Target   Target
	DueAt    time.Time
	Interval time.Duration
}

type Service struct {
	mu       sync.RWMutex
	items    map[string]Item
	rules    map[string]ReminderRule
	now      func() time.Time
	generate func(prefix string) string
}

func NewService(now func() time.Time, generate func(prefix string) string) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if generate == nil {
		generate = func(prefix string) string { return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000") }
	}
	return &Service{
		items:    map[string]Item{},
		rules:    map[string]ReminderRule{},
		now:      now,
		generate: generate,
	}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	if s == nil {
		return Item{}, errors.New("notification service is not configured")
	}
	category := in.Category
	if category == "" {
		category = CategoryTodo
	}
	if category != CategoryTodo && category != CategoryMessage && category != CategoryReminder {
		return Item{}, errors.New("notification category is invalid")
	}
	if strings.TrimSpace(in.Title) == "" {
		return Item{}, errors.New("notification title is required")
	}
	if strings.TrimSpace(in.ActorID) == "" {
		return Item{}, errors.New("notification actor is required")
	}
	if strings.TrimSpace(in.Target.Path) == "" {
		return Item{}, errors.New("notification target path is required")
	}
	now := s.now()
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = s.generate(string(category))
	}
	item := Item{
		ID:        id,
		Category:  category,
		Status:    initialStatus(category),
		Title:     strings.TrimSpace(in.Title),
		Body:      strings.TrimSpace(in.Body),
		ActorID:   strings.TrimSpace(in.ActorID),
		Target:    normalizeTarget(in.Target),
		DueAt:     in.DueAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.ID] = item
	return item, nil
}

func (s *Service) Complete(ctx context.Context, id string, actorID string) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[strings.TrimSpace(id)]
	if !ok {
		return Item{}, errors.New("notification item not found")
	}
	if item.ActorID != strings.TrimSpace(actorID) {
		return Item{}, errors.New("notification actor does not own item")
	}
	if item.Category == CategoryMessage {
		item.Status = StatusRead
	} else {
		item.Status = StatusDone
	}
	item.UpdatedAt = s.now()
	s.items[item.ID] = item
	return item, nil
}

func (s *Service) List(ctx context.Context, filter Filter) ([]Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		if filter.ActorID != "" && item.ActorID != filter.ActorID {
			continue
		}
		if filter.Category != "" && item.Category != filter.Category {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func (s *Service) UpsertReminderRule(ctx context.Context, rule ReminderRule) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(rule.ID) == "" || strings.TrimSpace(rule.ActorID) == "" || strings.TrimSpace(rule.Title) == "" {
		return errors.New("reminder rule id, actor, and title are required")
	}
	if strings.TrimSpace(rule.Target.Path) == "" {
		return errors.New("reminder target path is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rule.ID = strings.TrimSpace(rule.ID)
	rule.ActorID = strings.TrimSpace(rule.ActorID)
	rule.Title = strings.TrimSpace(rule.Title)
	rule.Target = normalizeTarget(rule.Target)
	s.rules[rule.ID] = rule
	return nil
}

func (s *Service) EmitDueReminders(ctx context.Context) ([]Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	rules := make([]ReminderRule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, rule)
	}
	s.mu.RUnlock()

	now := s.now()
	out := make([]Item, 0)
	for _, rule := range rules {
		if rule.DueAt.IsZero() || rule.DueAt.After(now) {
			continue
		}
		item, err := s.Create(ctx, CreateInput{
			ID:       s.generate("reminder"),
			Category: CategoryReminder,
			Title:    rule.Title,
			Body:     rule.Body,
			ActorID:  rule.ActorID,
			Target:   rule.Target,
			DueAt:    rule.DueAt,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func initialStatus(category Category) Status {
	if category == CategoryMessage {
		return StatusPending
	}
	return StatusPending
}

func normalizeTarget(target Target) Target {
	return Target{
		Type: strings.TrimSpace(target.Type),
		ID:   strings.TrimSpace(target.ID),
		Path: strings.TrimSpace(target.Path),
	}
}
