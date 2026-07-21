package gormrepo

import "time"

type NotificationItemModel struct {
	ID         string `gorm:"size:128;primaryKey"`
	Category   string `gorm:"size:32;index:idx_notification_actor_status_updated,priority:2;index:idx_notification_category_status,priority:1"`
	Status     string `gorm:"size:32;index:idx_notification_actor_status_updated,priority:3;index:idx_notification_category_status,priority:2"`
	Title      string `gorm:"size:255"`
	Body       string `gorm:"type:text"`
	ActorID    string `gorm:"size:64;index:idx_notification_actor_status_updated,priority:1"`
	TargetType string `gorm:"size:64;index:idx_notification_target,priority:1"`
	TargetID   string `gorm:"size:128;index:idx_notification_target,priority:2"`
	TargetPath string `gorm:"size:512"`
	DueAt      *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time `gorm:"index:idx_notification_actor_status_updated,priority:4"`
}

func (NotificationItemModel) TableName() string { return "sk_notification_items" }

type NotificationReminderRuleModel struct {
	ID            string     `gorm:"size:128;primaryKey"`
	Name          string     `gorm:"size:255"`
	ActorID       string     `gorm:"size:64;index:idx_notification_rule_actor_due,priority:1"`
	Title         string     `gorm:"size:255"`
	Body          string     `gorm:"type:text"`
	TargetType    string     `gorm:"size:64"`
	TargetID      string     `gorm:"size:128"`
	TargetPath    string     `gorm:"size:512"`
	DueAt         *time.Time `gorm:"index:idx_notification_rule_actor_due,priority:2"`
	IntervalNanos int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (NotificationReminderRuleModel) TableName() string { return "sk_notification_reminder_rules" }

type NotificationDeliveryAttemptModel struct {
	ID             string `gorm:"size:128;primaryKey"`
	NotificationID string `gorm:"size:128;uniqueIndex:idx_notification_delivery_idempotency,priority:1;index:idx_notification_delivery_stream,priority:1;index:idx_notification_delivery_status,priority:1"`
	Channel        string `gorm:"size:64;uniqueIndex:idx_notification_delivery_idempotency,priority:2;index:idx_notification_delivery_stream,priority:2"`
	IdempotencyKey string `gorm:"size:128;uniqueIndex:idx_notification_delivery_idempotency,priority:3"`
	Status         string `gorm:"size:32;index:idx_notification_delivery_status,priority:2"`
	Attempt        int    `gorm:"index:idx_notification_delivery_stream,priority:3"`
	Error          string `gorm:"type:text"`
	CreatedAt      time.Time
}

func (NotificationDeliveryAttemptModel) TableName() string {
	return "sk_notification_delivery_attempts"
}
