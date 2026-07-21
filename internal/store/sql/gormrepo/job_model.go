package gormrepo

import "time"

type JobModel struct {
	ID             string     `gorm:"column:id;type:varchar(128);primaryKey"`
	Namespace      string     `gorm:"column:namespace;type:varchar(128);not null;uniqueIndex:idx_job_idempotency,priority:1;index:idx_job_namespace_status,priority:1"`
	Kind           string     `gorm:"column:kind;type:varchar(128);not null;index:idx_job_kind_status,priority:1"`
	IdempotencyKey *string    `gorm:"column:idempotency_key;type:varchar(191);uniqueIndex:idx_job_idempotency,priority:2"`
	PayloadJSON    string     `gorm:"column:payload_json;type:text;not null"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;index:idx_job_namespace_status,priority:2;index:idx_job_kind_status,priority:2;index:idx_job_due,priority:1"`
	RunAt          time.Time  `gorm:"column:run_at;not null;index:idx_job_due,priority:2"`
	MaxAttempts    int        `gorm:"column:max_attempts;not null"`
	AttemptCount   int        `gorm:"column:attempt_count;not null"`
	LeaseOwner     string     `gorm:"column:lease_owner;type:varchar(128);not null"`
	LeaseToken     string     `gorm:"column:lease_token;type:varchar(64);not null"`
	LeaseExpiresAt *time.Time `gorm:"column:lease_expires_at;index:idx_job_lease_expiry"`
	LastError      string     `gorm:"column:last_error;type:text;not null"`
	ResultJSON     string     `gorm:"column:result_json;type:text;not null"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;not null"`
	CompletedAt    *time.Time `gorm:"column:completed_at"`
	DeadLetteredAt *time.Time `gorm:"column:dead_lettered_at;index:idx_job_dead_lettered"`
}

func (JobModel) TableName() string { return "sk_jobs" }
