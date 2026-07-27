package gormrepo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/eventoutbox"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/gorm"
)

type PluginEventOutboxModel struct {
	ID             string     `gorm:"column:id;primaryKey;size:128"`
	Publisher      string     `gorm:"column:publisher;size:64;not null;uniqueIndex:uq_plugin_event_idempotency,priority:1;index:idx_plugin_event_dispatch,priority:3"`
	IdempotencyKey string     `gorm:"column:idempotency_key;size:128;not null;uniqueIndex:uq_plugin_event_idempotency,priority:2"`
	Name           string     `gorm:"column:event_name;size:128;not null"`
	SchemaVersion  uint32     `gorm:"column:schema_version;not null"`
	PayloadType    string     `gorm:"column:payload_type;size:128;not null"`
	TenantID       string     `gorm:"column:tenant_id;size:128"`
	OrganizationID string     `gorm:"column:organization_id;size:128"`
	OwnerID        string     `gorm:"column:owner_id;size:128"`
	CorrelationID  string     `gorm:"column:correlation_id;size:128;not null;index"`
	CausationID    string     `gorm:"column:causation_id;size:128"`
	SubjectType    string     `gorm:"column:subject_type;size:128"`
	SubjectID      string     `gorm:"column:subject_id;size:128"`
	PayloadJSON    string     `gorm:"column:payload_json;type:text;not null"`
	RequestHash    string     `gorm:"column:request_hash;size:64;not null"`
	Status         string     `gorm:"column:status;size:24;not null;index:idx_plugin_event_dispatch,priority:1"`
	AttemptCount   int        `gorm:"column:attempt_count;not null"`
	MaxAttempts    int        `gorm:"column:max_attempts;not null"`
	NextAttemptAt  time.Time  `gorm:"column:next_attempt_at;not null;index:idx_plugin_event_dispatch,priority:2"`
	LeaseOwner     string     `gorm:"column:lease_owner;size:128;not null"`
	LeaseToken     string     `gorm:"column:lease_token;size:64;not null"`
	LeaseExpiresAt *time.Time `gorm:"column:lease_expires_at;index"`
	LastError      string     `gorm:"column:last_error;type:text;not null"`
	OccurredAt     time.Time  `gorm:"column:occurred_at;not null"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;not null"`
	DeliveredAt    *time.Time `gorm:"column:delivered_at"`
	DeadLetteredAt *time.Time `gorm:"column:dead_lettered_at"`
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

func (s *PluginEventOutboxStore) LeaseDue(ctx context.Context, input eventoutbox.LeaseInput) ([]eventoutbox.Record, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("plugin event outbox database is required")
	}
	if strings.TrimSpace(input.WorkerID) == "" || input.Limit <= 0 || input.Limit > 100 || input.LeaseDuration <= 0 {
		return nil, errors.New("valid plugin event lease input is required")
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	leaseUntil := now.Add(input.LeaseDuration)
	var leased []eventoutbox.Record
	err := withDBRetry(func() error {
		leased = nil
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := expireExhaustedPluginEvents(tx, now); err != nil {
				return err
			}
			var ids []string
			candidateLimit := input.Limit * 4
			if candidateLimit > 400 {
				candidateLimit = 400
			}
			eligible := tx.Model(&PluginEventOutboxModel{}).
				Where("attempt_count < max_attempts").
				Where("((status IN ? AND next_attempt_at <= ?) OR (status = ? AND lease_expires_at <= ?))",
					[]string{string(eventoutbox.StatusPending), string(eventoutbox.StatusRetryWait)}, now,
					string(eventoutbox.StatusRunning), now).
				Order("next_attempt_at ASC").Order("id ASC").Limit(candidateLimit)
			if err := eligible.Pluck("id", &ids).Error; err != nil {
				return err
			}
			for _, id := range ids {
				if len(leased) >= input.Limit {
					break
				}
				token, err := newPluginEventLeaseToken()
				if err != nil {
					return err
				}
				result := tx.Model(&PluginEventOutboxModel{}).
					Where("id = ? AND attempt_count < max_attempts", id).
					Where("((status IN ? AND next_attempt_at <= ?) OR (status = ? AND lease_expires_at <= ?))",
						[]string{string(eventoutbox.StatusPending), string(eventoutbox.StatusRetryWait)}, now,
						string(eventoutbox.StatusRunning), now).
					Updates(map[string]any{
						"status": string(eventoutbox.StatusRunning), "attempt_count": gorm.Expr("attempt_count + 1"),
						"lease_owner": input.WorkerID, "lease_token": token, "lease_expires_at": leaseUntil, "updated_at": now,
					})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					continue
				}
				var row PluginEventOutboxModel
				if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
					return err
				}
				record, err := row.eventOutboxRecord()
				if err != nil {
					return err
				}
				leased = append(leased, record)
			}
			return nil
		})
	})
	return leased, err
}

func (s *PluginEventOutboxStore) Ack(ctx context.Context, input eventoutbox.AckInput) (eventoutbox.Record, error) {
	if s == nil || s.db == nil {
		return eventoutbox.Record{}, errors.New("plugin event outbox database is required")
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var stored PluginEventOutboxModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			result := tx.Model(&PluginEventOutboxModel{}).
				Where("id = ? AND status = ? AND lease_token = ? AND lease_expires_at > ?",
					input.EventID, string(eventoutbox.StatusRunning), strings.TrimSpace(input.LeaseToken), now).
				Updates(map[string]any{
					"status": string(eventoutbox.StatusSucceeded), "delivered_at": now, "updated_at": now,
					"lease_owner": "", "lease_token": "", "lease_expires_at": nil, "last_error": "",
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return eventoutbox.ErrLeaseLost
			}
			return tx.Where("id = ?", input.EventID).First(&stored).Error
		})
	})
	if err != nil {
		return eventoutbox.Record{}, err
	}
	return stored.eventOutboxRecord()
}

func (s *PluginEventOutboxStore) Fail(ctx context.Context, input eventoutbox.FailInput) (eventoutbox.Record, error) {
	if s == nil || s.db == nil {
		return eventoutbox.Record{}, errors.New("plugin event outbox database is required")
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var stored PluginEventOutboxModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var current PluginEventOutboxModel
			if err := tx.Where("id = ?", input.EventID).First(&current).Error; err != nil {
				return err
			}
			if current.Status != string(eventoutbox.StatusRunning) || current.LeaseToken != strings.TrimSpace(input.LeaseToken) ||
				current.LeaseExpiresAt == nil || !current.LeaseExpiresAt.After(now) {
				return eventoutbox.ErrLeaseLost
			}
			status := eventoutbox.StatusRetryWait
			updates := map[string]any{
				"status": string(status), "next_attempt_at": input.RetryAt.UTC(),
				"last_error": strings.TrimSpace(input.Error), "updated_at": now,
				"lease_owner": "", "lease_token": "", "lease_expires_at": nil,
			}
			if current.AttemptCount >= current.MaxAttempts {
				status = eventoutbox.StatusDeadLetter
				updates["status"] = string(status)
				updates["dead_lettered_at"] = now
			}
			result := tx.Model(&PluginEventOutboxModel{}).
				Where("id = ? AND status = ? AND lease_token = ? AND lease_expires_at > ?",
					input.EventID, string(eventoutbox.StatusRunning), strings.TrimSpace(input.LeaseToken), now).
				Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return eventoutbox.ErrLeaseLost
			}
			return tx.Where("id = ?", input.EventID).First(&stored).Error
		})
	})
	if err != nil {
		return eventoutbox.Record{}, err
	}
	return stored.eventOutboxRecord()
}

func (s *PluginEventOutboxStore) Replay(ctx context.Context, id string, now time.Time) (eventoutbox.Record, error) {
	if s == nil || s.db == nil {
		return eventoutbox.Record{}, errors.New("plugin event outbox database is required")
	}
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var stored PluginEventOutboxModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			result := tx.Model(&PluginEventOutboxModel{}).
				Where("id = ? AND status = ?", id, string(eventoutbox.StatusDeadLetter)).
				Updates(map[string]any{
					"status": string(eventoutbox.StatusPending), "attempt_count": 0, "next_attempt_at": now,
					"lease_owner": "", "lease_token": "", "lease_expires_at": nil, "last_error": "",
					"dead_lettered_at": nil, "updated_at": now,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				var count int64
				if err := tx.Model(&PluginEventOutboxModel{}).Where("id = ?", id).Count(&count).Error; err != nil {
					return err
				}
				if count == 0 {
					return gorm.ErrRecordNotFound
				}
				return eventoutbox.ErrNotDeadLetter
			}
			return tx.Where("id = ?", id).First(&stored).Error
		})
	})
	if err != nil {
		return eventoutbox.Record{}, err
	}
	return stored.eventOutboxRecord()
}

func (s *PluginEventOutboxStore) List(ctx context.Context, filter eventoutbox.Filter) ([]eventoutbox.Record, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("plugin event outbox database is required")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		return nil, errors.New("plugin event outbox list limit exceeds 200")
	}
	query := storesql.ResolveDB(ctx, s.db).Model(&PluginEventOutboxModel{})
	if publisher := strings.TrimSpace(filter.Publisher); publisher != "" {
		query = query.Where("publisher = ?", publisher)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", string(filter.Status))
	}
	var rows []PluginEventOutboxModel
	if err := query.Order("updated_at DESC").Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	records := make([]eventoutbox.Record, 0, len(rows))
	for _, row := range rows {
		record, err := row.eventOutboxRecord()
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func eventOutboxModel(record eventoutbox.Record) (PluginEventOutboxModel, error) {
	payload, err := json.Marshal(record.Envelope.Payload)
	if err != nil {
		return PluginEventOutboxModel{}, fmt.Errorf("encode plugin event payload: %w", err)
	}
	if record.MaxAttempts <= 0 {
		record.MaxAttempts = eventoutbox.DefaultMaxAttempts
	}
	if record.NextAttemptAt.IsZero() {
		record.NextAttemptAt = record.CreatedAt
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
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
		AttemptCount: record.AttemptCount, MaxAttempts: record.MaxAttempts,
		NextAttemptAt: record.NextAttemptAt, LeaseOwner: record.LeaseOwner, LeaseToken: record.LeaseToken,
		LeaseExpiresAt: record.LeaseExpiresAt, LastError: record.LastError,
		OccurredAt: record.Envelope.OccurredAt, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
		DeliveredAt: record.DeliveredAt, DeadLetteredAt: record.DeadLetteredAt,
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
		Status: eventoutbox.Status(m.Status), AttemptCount: m.AttemptCount, MaxAttempts: m.MaxAttempts,
		NextAttemptAt: m.NextAttemptAt, LeaseOwner: m.LeaseOwner, LeaseToken: m.LeaseToken,
		LeaseExpiresAt: cloneSQLTime(m.LeaseExpiresAt), LastError: m.LastError,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		DeliveredAt: cloneSQLTime(m.DeliveredAt), DeadLetteredAt: cloneSQLTime(m.DeadLetteredAt),
	}, nil
}

func expireExhaustedPluginEvents(tx *gorm.DB, now time.Time) error {
	return tx.Model(&PluginEventOutboxModel{}).
		Where("status = ? AND lease_expires_at <= ? AND attempt_count >= max_attempts", string(eventoutbox.StatusRunning), now).
		Updates(map[string]any{
			"status": string(eventoutbox.StatusDeadLetter), "last_error": "plugin event lease expired after final attempt",
			"dead_lettered_at": now, "updated_at": now, "lease_owner": "", "lease_token": "", "lease_expires_at": nil,
		}).Error
}

func newPluginEventLeaseToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func cloneSQLTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

var _ eventoutbox.Store = (*PluginEventOutboxStore)(nil)
