package gormrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/eventoutbox"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/gorm"
)

type PluginEventOutboxModel struct {
	ID             string    `gorm:"column:id;primaryKey;size:128"`
	Publisher      string    `gorm:"column:publisher;size:64;not null;uniqueIndex:uq_plugin_event_idempotency,priority:1;index:idx_plugin_event_pending,priority:2"`
	IdempotencyKey string    `gorm:"column:idempotency_key;size:128;not null;uniqueIndex:uq_plugin_event_idempotency,priority:2"`
	Name           string    `gorm:"column:event_name;size:128;not null"`
	SchemaVersion  uint32    `gorm:"column:schema_version;not null"`
	PayloadType    string    `gorm:"column:payload_type;size:128;not null"`
	TenantID       string    `gorm:"column:tenant_id;size:128;index:idx_plugin_event_pending,priority:3"`
	OrganizationID string    `gorm:"column:organization_id;size:128"`
	OwnerID        string    `gorm:"column:owner_id;size:128"`
	CorrelationID  string    `gorm:"column:correlation_id;size:128;not null;index"`
	CausationID    string    `gorm:"column:causation_id;size:128"`
	SubjectType    string    `gorm:"column:subject_type;size:128"`
	SubjectID      string    `gorm:"column:subject_id;size:128"`
	PayloadJSON    string    `gorm:"column:payload_json;type:text;not null"`
	RequestHash    string    `gorm:"column:request_hash;size:64;not null"`
	Status         string    `gorm:"column:status;size:24;not null;index:idx_plugin_event_pending,priority:1"`
	OccurredAt     time.Time `gorm:"column:occurred_at;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;index:idx_plugin_event_pending,priority:4"`
}

func (PluginEventOutboxModel) TableName() string { return "sk_plugin_event_outbox" }

type PluginEventOutboxStore struct {
	db *gorm.DB
}

func NewPluginEventOutboxStore(db *gorm.DB) *PluginEventOutboxStore {
	return &PluginEventOutboxStore{db: db}
}

func (s *PluginEventOutboxStore) Enqueue(ctx context.Context, record eventoutbox.Record) (eventoutbox.Record, bool, error) {
	if s == nil || s.db == nil {
		return eventoutbox.Record{}, false, errors.New("plugin event outbox database is required")
	}
	if existing, err := s.Get(ctx, record.Envelope.ID); err == nil {
		if existing.RequestHash != record.RequestHash {
			return eventoutbox.Record{}, false, eventoutbox.ErrIdentityConflict
		}
		return existing, true, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return eventoutbox.Record{}, false, err
	}
	row, err := eventOutboxModel(record)
	if err != nil {
		return eventoutbox.Record{}, false, err
	}
	if err = storesql.ResolveDB(ctx, s.db).Create(&row).Error; err == nil {
		return record, false, nil
	}
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		return eventoutbox.Record{}, false, fmt.Errorf("enqueue plugin event: %w", err)
	}
	existing, loadErr := s.Get(ctx, record.Envelope.ID)
	if loadErr != nil {
		return eventoutbox.Record{}, false, loadErr
	}
	if existing.RequestHash != record.RequestHash {
		return eventoutbox.Record{}, false, eventoutbox.ErrIdentityConflict
	}
	return existing, true, nil
}

func (s *PluginEventOutboxStore) Get(ctx context.Context, id string) (eventoutbox.Record, error) {
	if s == nil || s.db == nil {
		return eventoutbox.Record{}, errors.New("plugin event outbox database is required")
	}
	var row PluginEventOutboxModel
	if err := storesql.ResolveDB(ctx, s.db).Where("id = ?", id).First(&row).Error; err != nil {
		return eventoutbox.Record{}, err
	}
	return row.eventOutboxRecord()
}

func eventOutboxModel(record eventoutbox.Record) (PluginEventOutboxModel, error) {
	payload, err := json.Marshal(record.Envelope.Payload)
	if err != nil {
		return PluginEventOutboxModel{}, fmt.Errorf("encode plugin event payload: %w", err)
	}
	return PluginEventOutboxModel{
		ID: record.Envelope.ID, Publisher: record.Envelope.Publisher,
		IdempotencyKey: record.IdempotencyKey, Name: record.Envelope.Name,
		SchemaVersion: record.Envelope.SchemaVersion, PayloadType: record.Envelope.PayloadType,
		TenantID: record.Envelope.Scope.TenantID, OrganizationID: record.Envelope.Scope.OrganizationID,
		OwnerID: record.Envelope.Scope.OwnerID, CorrelationID: record.Envelope.CorrelationID,
		CausationID: record.Envelope.CausationID, SubjectType: record.Envelope.Subject.Type,
		SubjectID: record.Envelope.Subject.ID, PayloadJSON: string(payload),
		RequestHash: record.RequestHash, Status: string(record.Status),
		OccurredAt: record.Envelope.OccurredAt, CreatedAt: record.CreatedAt,
	}, nil
}

func (m PluginEventOutboxModel) eventOutboxRecord() (eventoutbox.Record, error) {
	payload := pluginsdk.EventPayload{}
	if err := json.Unmarshal([]byte(m.PayloadJSON), &payload); err != nil {
		return eventoutbox.Record{}, fmt.Errorf("decode plugin event payload: %w", err)
	}
	return eventoutbox.Record{
		Envelope: pluginsdk.EventEnvelope{
			ID: m.ID, Publisher: m.Publisher, Name: m.Name, SchemaVersion: m.SchemaVersion,
			PayloadType: m.PayloadType,
			Scope: pluginsdk.EventScope{
				TenantID: m.TenantID, OrganizationID: m.OrganizationID, OwnerID: m.OwnerID,
			},
			CorrelationID: m.CorrelationID, CausationID: m.CausationID,
			Subject: pluginsdk.EventSubject{Type: m.SubjectType, ID: m.SubjectID},
			Payload: payload, OccurredAt: m.OccurredAt,
		},
		IdempotencyKey: m.IdempotencyKey, RequestHash: m.RequestHash,
		Status: eventoutbox.Status(m.Status), CreatedAt: m.CreatedAt,
	}, nil
}

var _ eventoutbox.Store = (*PluginEventOutboxStore)(nil)
