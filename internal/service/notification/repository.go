package notification

import (
	"context"
	"time"
)

type Repository interface {
	CreateItem(ctx context.Context, item Item) (stored Item, created bool, err error)
	CompleteItem(ctx context.Context, id, actorID string, now time.Time) (Item, error)
	ListItems(ctx context.Context, filter Filter) ([]Item, error)
	UpsertReminderRule(ctx context.Context, rule ReminderRule) error
	ListReminderRules(ctx context.Context) ([]ReminderRule, error)
	RecordDeliveryAttempt(ctx context.Context, attempt DeliveryAttempt) (stored DeliveryAttempt, created bool, err error)
	ListDeliveryAttempts(ctx context.Context, notificationID, channel string) ([]DeliveryAttempt, error)
}
