package notification

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
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

type DeliveryStatus string

const (
	DeliverySucceeded DeliveryStatus = "succeeded"
	DeliveryFailed    DeliveryStatus = "failed"
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

type DeliveryAttempt struct {
	ID             string
	NotificationID string
	Channel        string
	IdempotencyKey string
	Status         DeliveryStatus
	Attempt        int
	Error          string
	CreatedAt      time.Time
}

type DeliveryAttemptInput struct {
	ID             string
	NotificationID string
	Channel        string
	IdempotencyKey string
	Status         DeliveryStatus
	Error          string
}

type Service struct {
	repo     Repository
	now      func() time.Time
	generate func(prefix string) string
}

func NewService(repo Repository, now func() time.Time, generate func(prefix string) string) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if generate == nil {
		generate = func(prefix string) string { return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000") }
	}
	return &Service{repo: repo, now: now, generate: generate}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	if s == nil || s.repo == nil {
		return Item{}, errors.New("notification repository is not configured")
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
	now := s.now().UTC()
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = s.generate(string(category))
	}
	item := Item{
		ID: id, Category: category, Status: initialStatus(category), Title: strings.TrimSpace(in.Title), Body: strings.TrimSpace(in.Body),
		ActorID: strings.TrimSpace(in.ActorID), Target: normalizeTarget(in.Target), DueAt: in.DueAt, CreatedAt: now, UpdatedAt: now,
	}
	stored, created, err := s.repo.CreateItem(ctx, item)
	if err != nil {
		return Item{}, err
	}
	if !created && !sameItemContract(stored, item) {
		return Item{}, errors.New("notification id conflicts with existing item")
	}
	return stored, nil
}

func (s *Service) Complete(ctx context.Context, id string, actorID string) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	if s == nil || s.repo == nil {
		return Item{}, errors.New("notification repository is not configured")
	}
	return s.repo.CompleteItem(ctx, strings.TrimSpace(id), strings.TrimSpace(actorID), s.now().UTC())
}

func (s *Service) List(ctx context.Context, filter Filter) ([]Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("notification repository is not configured")
	}
	filter.ActorID = strings.TrimSpace(filter.ActorID)
	return s.repo.ListItems(ctx, filter)
}

func (s *Service) UpsertReminderRule(ctx context.Context, rule ReminderRule) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.repo == nil {
		return errors.New("notification repository is not configured")
	}
	if strings.TrimSpace(rule.ID) == "" || strings.TrimSpace(rule.ActorID) == "" || strings.TrimSpace(rule.Title) == "" {
		return errors.New("reminder rule id, actor, and title are required")
	}
	if strings.TrimSpace(rule.Target.Path) == "" {
		return errors.New("reminder target path is required")
	}
	rule.ID = strings.TrimSpace(rule.ID)
	rule.Name = strings.TrimSpace(rule.Name)
	rule.ActorID = strings.TrimSpace(rule.ActorID)
	rule.Title = strings.TrimSpace(rule.Title)
	rule.Body = strings.TrimSpace(rule.Body)
	rule.Target = normalizeTarget(rule.Target)
	return s.repo.UpsertReminderRule(ctx, rule)
}

func (s *Service) EmitDueReminders(ctx context.Context) ([]Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("notification repository is not configured")
	}
	rules, err := s.repo.ListReminderRules(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	out := make([]Item, 0)
	for _, rule := range rules {
		if rule.DueAt.IsZero() || rule.DueAt.After(now) {
			continue
		}
		item, err := s.Create(ctx, CreateInput{
			ID: reminderItemID(rule), Category: CategoryReminder, Title: rule.Title, Body: rule.Body,
			ActorID: rule.ActorID, Target: rule.Target, DueAt: rule.DueAt,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) RecordDeliveryAttempt(ctx context.Context, in DeliveryAttemptInput) (DeliveryAttempt, error) {
	if err := ctx.Err(); err != nil {
		return DeliveryAttempt{}, err
	}
	if s == nil || s.repo == nil {
		return DeliveryAttempt{}, errors.New("notification repository is not configured")
	}
	in.NotificationID = strings.TrimSpace(in.NotificationID)
	in.Channel = strings.TrimSpace(strings.ToLower(in.Channel))
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.NotificationID == "" || in.Channel == "" || in.IdempotencyKey == "" {
		return DeliveryAttempt{}, errors.New("notification delivery identity is incomplete")
	}
	if in.Status != DeliverySucceeded && in.Status != DeliveryFailed {
		return DeliveryAttempt{}, errors.New("notification delivery status is invalid")
	}
	if in.Status == DeliveryFailed && strings.TrimSpace(in.Error) == "" {
		return DeliveryAttempt{}, errors.New("failed notification delivery requires an error")
	}
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = s.generate("delivery")
	}
	attempt := DeliveryAttempt{
		ID: id, NotificationID: in.NotificationID, Channel: in.Channel, IdempotencyKey: in.IdempotencyKey,
		Status: in.Status, Error: strings.TrimSpace(in.Error), CreatedAt: s.now().UTC(),
	}
	stored, _, err := s.repo.RecordDeliveryAttempt(ctx, attempt)
	return stored, err
}

func (s *Service) ListDeliveryAttempts(ctx context.Context, notificationID, channel string) ([]DeliveryAttempt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, errors.New("notification repository is not configured")
	}
	return s.repo.ListDeliveryAttempts(ctx, strings.TrimSpace(notificationID), strings.TrimSpace(strings.ToLower(channel)))
}

func initialStatus(Category) Status { return StatusPending }

func normalizeTarget(target Target) Target {
	return Target{Type: strings.TrimSpace(target.Type), ID: strings.TrimSpace(target.ID), Path: strings.TrimSpace(target.Path)}
}

func sameItemContract(left, right Item) bool {
	return left.ID == right.ID && left.Category == right.Category && left.Title == right.Title && left.Body == right.Body &&
		left.ActorID == right.ActorID && left.Target == right.Target && left.DueAt.Equal(right.DueAt)
}

func reminderItemID(rule ReminderRule) string {
	identity := fmt.Sprintf("%s\x00%d", rule.ID, rule.DueAt.UTC().UnixNano())
	digest := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("reminder-%x", digest[:12])
}
