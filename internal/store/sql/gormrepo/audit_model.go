package gormrepo

import (
	"encoding/json"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type AuditRecordModel struct {
	ID         string    `gorm:"primaryKey;size:64"`
	ActorID    string    `gorm:"column:actor_id;size:64;index:idx_audit_actor_occurred,priority:1"`
	Action     string    `gorm:"column:action;size:64;index"`
	Resource   string    `gorm:"column:resource;size:64;index"`
	ResourceID string    `gorm:"column:resource_id;size:128"`
	DetailJSON string    `gorm:"column:detail_json;type:longtext"`
	OccurredAt time.Time `gorm:"column:occurred_at;index:idx_audit_actor_occurred,priority:2;index:idx_audit_occurred"`
}

func (AuditRecordModel) TableName() string { return "sk_audit_records" }

func AuditRecordModelFromDomain(record *domainaudit.Record) AuditRecordModel {
	detailJSON := "{}"
	if record != nil && len(record.Detail) > 0 {
		if raw, err := json.Marshal(record.Detail); err == nil {
			detailJSON = string(raw)
		}
	}
	if record == nil {
		return AuditRecordModel{DetailJSON: detailJSON}
	}
	return AuditRecordModel{
		ID:         record.ID.String(),
		ActorID:    record.ActorID.String(),
		Action:     record.Action,
		Resource:   record.Resource,
		ResourceID: record.ResourceID,
		DetailJSON: detailJSON,
		OccurredAt: record.OccurredAt,
	}
}

func (m AuditRecordModel) ToDomain() *domainaudit.Record {
	detail := map[string]any{}
	if raw := m.DetailJSON; raw != "" {
		_ = json.Unmarshal([]byte(raw), &detail)
	}
	return &domainaudit.Record{
		ID:         shared.ID(m.ID),
		ActorID:    shared.ID(m.ActorID),
		Action:     m.Action,
		Resource:   m.Resource,
		ResourceID: m.ResourceID,
		Detail:     detail,
		OccurredAt: m.OccurredAt,
	}
}
