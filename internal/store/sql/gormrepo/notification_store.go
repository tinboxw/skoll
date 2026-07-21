package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationStore struct {
	db *gorm.DB
}

func NewNotificationStore(db *gorm.DB) *NotificationStore {
	return &NotificationStore{db: db}
}

func (s *NotificationStore) CreateItem(ctx context.Context, item notificationsvc.Item) (notificationsvc.Item, bool, error) {
	if s == nil || s.db == nil {
		return notificationsvc.Item{}, false, fmt.Errorf("notification repository is required")
	}
	row := notificationItemRow(item)
	var stored NotificationItemModel
	created := false
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
			if result.Error != nil {
				return result.Error
			}
			created = result.RowsAffected == 1
			return tx.Where("id = ?", row.ID).First(&stored).Error
		})
	})
	if err != nil {
		return notificationsvc.Item{}, false, err
	}
	return notificationItemFromRow(stored), created, nil
}

func (s *NotificationStore) CompleteItem(ctx context.Context, id, actorID string, now time.Time) (notificationsvc.Item, error) {
	if s == nil || s.db == nil {
		return notificationsvc.Item{}, fmt.Errorf("notification repository is required")
	}
	var stored NotificationItemModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", strings.TrimSpace(id)).First(&stored).Error; err != nil {
				return err
			}
			if stored.ActorID != strings.TrimSpace(actorID) {
				return fmt.Errorf("notification actor does not own item")
			}
			next := string(notificationsvc.StatusDone)
			if stored.Category == string(notificationsvc.CategoryMessage) {
				next = string(notificationsvc.StatusRead)
			}
			if stored.Status == next {
				return nil
			}
			stored.Status = next
			stored.UpdatedAt = now
			return tx.Model(&NotificationItemModel{}).Where("id = ?", stored.ID).Updates(map[string]any{"status": stored.Status, "updated_at": stored.UpdatedAt}).Error
		})
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notificationsvc.Item{}, fmt.Errorf("notification item not found")
	}
	if err != nil {
		return notificationsvc.Item{}, err
	}
	return notificationItemFromRow(stored), nil
}

func (s *NotificationStore) ListItems(ctx context.Context, filter notificationsvc.Filter) ([]notificationsvc.Item, error) {
	query := s.db.WithContext(ctx).Model(&NotificationItemModel{})
	if filter.ActorID != "" {
		query = query.Where("actor_id = ?", filter.ActorID)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", string(filter.Category))
	}
	if filter.Status != "" {
		query = query.Where("status = ?", string(filter.Status))
	}
	var rows []NotificationItemModel
	if err := withDBRetry(func() error { return query.Order("updated_at DESC").Order("id ASC").Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]notificationsvc.Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, notificationItemFromRow(row))
	}
	return out, nil
}

func (s *NotificationStore) UpsertReminderRule(ctx context.Context, rule notificationsvc.ReminderRule) error {
	row := notificationRuleRow(rule)
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "actor_id", "title", "body", "target_type", "target_id", "target_path", "due_at", "interval_nanos", "updated_at",
			}),
		}).Create(&row).Error
	})
}

func (s *NotificationStore) ListReminderRules(ctx context.Context) ([]notificationsvc.ReminderRule, error) {
	var rows []NotificationReminderRuleModel
	if err := withDBRetry(func() error { return s.db.WithContext(ctx).Order("id ASC").Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]notificationsvc.ReminderRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, notificationRuleFromRow(row))
	}
	return out, nil
}

func (s *NotificationStore) RecordDeliveryAttempt(ctx context.Context, attempt notificationsvc.DeliveryAttempt) (notificationsvc.DeliveryAttempt, bool, error) {
	if s == nil || s.db == nil {
		return notificationsvc.DeliveryAttempt{}, false, fmt.Errorf("notification repository is required")
	}
	var stored NotificationDeliveryAttemptModel
	created := false
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var item NotificationItemModel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", attempt.NotificationID).First(&item).Error; err != nil {
				return err
			}
			query := tx.Where("notification_id = ? AND channel = ? AND idempotency_key = ?", attempt.NotificationID, attempt.Channel, attempt.IdempotencyKey)
			if err := query.First(&stored).Error; err == nil {
				return nil
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Where("notification_id = ? AND channel = ? AND status = ?", attempt.NotificationID, attempt.Channel, string(notificationsvc.DeliverySucceeded)).Order("attempt ASC").First(&stored).Error; err == nil {
				return nil
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			var maxAttempt int
			if err := tx.Model(&NotificationDeliveryAttemptModel{}).Where("notification_id = ? AND channel = ?", attempt.NotificationID, attempt.Channel).Select("COALESCE(MAX(attempt), 0)").Scan(&maxAttempt).Error; err != nil {
				return err
			}
			attempt.Attempt = maxAttempt + 1
			stored = notificationDeliveryRow(attempt)
			if err := tx.Create(&stored).Error; err != nil {
				return err
			}
			created = true
			return nil
		})
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notificationsvc.DeliveryAttempt{}, false, fmt.Errorf("notification item not found")
	}
	if err != nil {
		return notificationsvc.DeliveryAttempt{}, false, err
	}
	return notificationDeliveryFromRow(stored), created, nil
}

func (s *NotificationStore) ListDeliveryAttempts(ctx context.Context, notificationID, channel string) ([]notificationsvc.DeliveryAttempt, error) {
	query := s.db.WithContext(ctx).Where("notification_id = ?", strings.TrimSpace(notificationID))
	if channel != "" {
		query = query.Where("channel = ?", strings.TrimSpace(strings.ToLower(channel)))
	}
	var rows []NotificationDeliveryAttemptModel
	if err := withDBRetry(func() error { return query.Order("channel ASC").Order("attempt ASC").Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]notificationsvc.DeliveryAttempt, 0, len(rows))
	for _, row := range rows {
		out = append(out, notificationDeliveryFromRow(row))
	}
	return out, nil
}

func notificationItemRow(item notificationsvc.Item) NotificationItemModel {
	return NotificationItemModel{
		ID: item.ID, Category: string(item.Category), Status: string(item.Status), Title: item.Title, Body: item.Body, ActorID: item.ActorID,
		TargetType: item.Target.Type, TargetID: item.Target.ID, TargetPath: item.Target.Path, DueAt: nullableTime(item.DueAt), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func notificationItemFromRow(row NotificationItemModel) notificationsvc.Item {
	return notificationsvc.Item{
		ID: row.ID, Category: notificationsvc.Category(row.Category), Status: notificationsvc.Status(row.Status), Title: row.Title, Body: row.Body, ActorID: row.ActorID,
		Target: notificationsvc.Target{Type: row.TargetType, ID: row.TargetID, Path: row.TargetPath}, DueAt: timeValue(row.DueAt), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func notificationRuleRow(rule notificationsvc.ReminderRule) NotificationReminderRuleModel {
	now := time.Now().UTC()
	return NotificationReminderRuleModel{
		ID: rule.ID, Name: rule.Name, ActorID: rule.ActorID, Title: rule.Title, Body: rule.Body,
		TargetType: rule.Target.Type, TargetID: rule.Target.ID, TargetPath: rule.Target.Path, DueAt: nullableTime(rule.DueAt),
		IntervalNanos: int64(rule.Interval), CreatedAt: now, UpdatedAt: now,
	}
}

func notificationRuleFromRow(row NotificationReminderRuleModel) notificationsvc.ReminderRule {
	return notificationsvc.ReminderRule{
		ID: row.ID, Name: row.Name, ActorID: row.ActorID, Title: row.Title, Body: row.Body,
		Target: notificationsvc.Target{Type: row.TargetType, ID: row.TargetID, Path: row.TargetPath}, DueAt: timeValue(row.DueAt), Interval: time.Duration(row.IntervalNanos),
	}
}

func notificationDeliveryRow(attempt notificationsvc.DeliveryAttempt) NotificationDeliveryAttemptModel {
	return NotificationDeliveryAttemptModel{
		ID: attempt.ID, NotificationID: attempt.NotificationID, Channel: attempt.Channel, IdempotencyKey: attempt.IdempotencyKey,
		Status: string(attempt.Status), Attempt: attempt.Attempt, Error: attempt.Error, CreatedAt: attempt.CreatedAt,
	}
}

func notificationDeliveryFromRow(row NotificationDeliveryAttemptModel) notificationsvc.DeliveryAttempt {
	return notificationsvc.DeliveryAttempt{
		ID: row.ID, NotificationID: row.NotificationID, Channel: row.Channel, IdempotencyKey: row.IdempotencyKey,
		Status: notificationsvc.DeliveryStatus(row.Status), Attempt: row.Attempt, Error: row.Error, CreatedAt: row.CreatedAt,
	}
}

func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	value = value.UTC()
	return &value
}

func timeValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

var _ notificationsvc.Repository = (*NotificationStore)(nil)
