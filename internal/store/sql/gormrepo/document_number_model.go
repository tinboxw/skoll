package gormrepo

import "time"

type DocumentNumberSequenceModel struct {
	PluginID     string    `gorm:"column:plugin_id;type:varchar(64);primaryKey"`
	TenantID     string    `gorm:"column:tenant_id;type:varchar(128);primaryKey"`
	DocumentType string    `gorm:"column:document_type;type:varchar(63);primaryKey"`
	PeriodKey    string    `gorm:"column:period_key;type:varchar(8);primaryKey"`
	RuleHash     string    `gorm:"column:rule_hash;type:char(64);not null"`
	LastValue    int64     `gorm:"column:last_value;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null"`
}

func (DocumentNumberSequenceModel) TableName() string { return "sk_document_number_sequences" }

type DocumentNumberIssueModel struct {
	PluginID       string    `gorm:"column:plugin_id;type:varchar(64);primaryKey;uniqueIndex:idx_document_number_value,priority:1"`
	TenantID       string    `gorm:"column:tenant_id;type:varchar(128);primaryKey;uniqueIndex:idx_document_number_value,priority:2"`
	DocumentType   string    `gorm:"column:document_type;type:varchar(63);primaryKey;uniqueIndex:idx_document_number_value,priority:3"`
	IdempotencyKey string    `gorm:"column:idempotency_key;type:varchar(128);primaryKey"`
	PeriodKey      string    `gorm:"column:period_key;type:varchar(8);not null;uniqueIndex:idx_document_number_value,priority:4"`
	SequenceValue  int64     `gorm:"column:sequence_value;not null;uniqueIndex:idx_document_number_value,priority:5"`
	Number         string    `gorm:"column:number;type:varchar(128);not null"`
	RequestHash    string    `gorm:"column:request_hash;type:char(64);not null"`
	IssuedAt       time.Time `gorm:"column:issued_at;not null"`
}

func (DocumentNumberIssueModel) TableName() string { return "sk_document_number_issues" }
