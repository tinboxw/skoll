package gormrepo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/plugin/eventinbox"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/gorm"
)

type PluginEventInboxModel struct {
	DeliveryID     string     `gorm:"column:delivery_id;primaryKey;size:128"`
	EventID        string     `gorm:"column:event_id;size:128;not null;index"`
	Subscriber     string     `gorm:"column:subscriber;size:64;not null;index:idx_plugin_event_inbox_status,priority:2"`
	Handler        string     `gorm:"column:handler;size:128;not null"`
	Publisher      string     `gorm:"column:publisher;size:64;not null"`
	EventName      string     `gorm:"column:event_name;size:128;not null"`
	SchemaVersion  uint32     `gorm:"column:schema_version;not null"`
	TenantID       string     `gorm:"column:tenant_id;size:128;not null"`
	DeliveryJSON   string     `gorm:"column:delivery_json;type:text;not null"`
	Status         string     `gorm:"column:status;size:24;not null;index:idx_plugin_event_inbox_status,priority:1"`
	AttemptCount   int        `gorm:"column:attempt_count;not null"`
	LeaseOwner     string     `gorm:"column:lease_owner;size:128;not null"`
	LeaseToken     string     `gorm:"column:lease_token;size:64;not null"`
	LeaseExpiresAt *time.Time `gorm:"column:lease_expires_at;index"`
	LastError      string     `gorm:"column:last_error;type:text;not null"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;not null"`
	ProcessedAt    *time.Time `gorm:"column:processed_at"`
}

func (PluginEventInboxModel) TableName() string { return "sk_plugin_event_inbox" }

type PluginEventInboxStore struct {
	db *gorm.DB
}

func NewPluginEventInboxStore(db *gorm.DB) *PluginEventInboxStore {
	return &PluginEventInboxStore{db: db}
}

func (s *PluginEventInboxStore) Claim(ctx context.Context, input eventinbox.ClaimInput) (eventinbox.Record, bool, error) {
	if s == nil || s.db == nil {
		return eventinbox.Record{}, false, errors.New("plugin event inbox database is required")
	}
	if strings.TrimSpace(input.WorkerID) == "" || input.LeaseDuration <= 0 {
		return eventinbox.Record{}, false, errors.New("valid plugin event inbox claim is required")
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	token, err := newPluginEventInboxLeaseToken()
	if err != nil {
		return eventinbox.Record{}, false, err
	}
	var stored PluginEventInboxModel
	claimed := false
	err = withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			err := tx.Where("delivery_id = ?", input.Delivery.DeliveryID).First(&stored).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				row, encodeErr := eventInboxModel(input.Delivery, input.WorkerID, token, now, input.LeaseDuration)
				if encodeErr != nil {
					return encodeErr
				}
				if createErr := tx.Create(&row).Error; createErr != nil {
					return createErr
				}
				stored, claimed = row, true
				return nil
			}
			if err != nil {
				return err
			}
			if stored.Status == string(eventinbox.StatusSucceeded) ||
				(stored.Status == string(eventinbox.StatusProcessing) && stored.LeaseExpiresAt != nil && stored.LeaseExpiresAt.After(now)) {
				return nil
			}
			result := tx.Model(&PluginEventInboxModel{}).
				Where("delivery_id = ? AND status <> ? AND (status = ? OR lease_expires_at IS NULL OR lease_expires_at <= ?)",
					input.Delivery.DeliveryID, string(eventinbox.StatusSucceeded), string(eventinbox.StatusFailed), now).
				Updates(map[string]any{
					"status": string(eventinbox.StatusProcessing), "attempt_count": gorm.Expr("attempt_count + 1"),
					"lease_owner": input.WorkerID, "lease_token": token, "lease_expires_at": now.Add(input.LeaseDuration),
					"last_error": "", "updated_at": now,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}
			claimed = true
			return tx.Where("delivery_id = ?", input.Delivery.DeliveryID).First(&stored).Error
		})
	})
	if err != nil {
		return eventinbox.Record{}, false, err
	}
	record, err := stored.eventInboxRecord()
	return record, claimed, err
}

func (s *PluginEventInboxStore) Complete(ctx context.Context, input eventinbox.TransitionInput) (eventinbox.Record, error) {
	return s.transition(ctx, input, eventinbox.StatusSucceeded)
}

func (s *PluginEventInboxStore) Fail(ctx context.Context, input eventinbox.TransitionInput) (eventinbox.Record, error) {
	return s.transition(ctx, input, eventinbox.StatusFailed)
}

func (s *PluginEventInboxStore) Get(ctx context.Context, id string) (eventinbox.Record, error) {
	if s == nil || s.db == nil {
		return eventinbox.Record{}, errors.New("plugin event inbox database is required")
	}
	var row PluginEventInboxModel
	if err := s.db.WithContext(ctx).Where("delivery_id = ?", id).First(&row).Error; err != nil {
		return eventinbox.Record{}, err
	}
	return row.eventInboxRecord()
}

func (s *PluginEventInboxStore) transition(ctx context.Context, input eventinbox.TransitionInput, status eventinbox.Status) (eventinbox.Record, error) {
	if s == nil || s.db == nil {
		return eventinbox.Record{}, errors.New("plugin event inbox database is required")
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	updates := map[string]any{
		"status": string(status), "updated_at": now, "lease_owner": "", "lease_token": "", "lease_expires_at": nil,
	}
	if status == eventinbox.StatusSucceeded {
		updates["processed_at"] = now
		updates["last_error"] = ""
	} else {
		updates["last_error"] = strings.TrimSpace(input.Error)
	}
	var stored PluginEventInboxModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			result := tx.Model(&PluginEventInboxModel{}).
				Where("delivery_id = ? AND status = ? AND lease_token = ? AND lease_expires_at > ?",
					input.DeliveryID, string(eventinbox.StatusProcessing), strings.TrimSpace(input.LeaseToken), now).
				Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return eventinbox.ErrLeaseLost
			}
			return tx.Where("delivery_id = ?", input.DeliveryID).First(&stored).Error
		})
	})
	if err != nil {
		return eventinbox.Record{}, err
	}
	return stored.eventInboxRecord()
}

func eventInboxModel(delivery pluginsdk.EventDelivery, workerID, token string, now time.Time, duration time.Duration) (PluginEventInboxModel, error) {
	encoded, err := json.Marshal(delivery)
	if err != nil {
		return PluginEventInboxModel{}, err
	}
	leaseExpiresAt := now.Add(duration).UTC()
	return PluginEventInboxModel{
		DeliveryID: delivery.DeliveryID, EventID: delivery.Envelope.ID, Subscriber: delivery.Subscriber,
		Handler: delivery.Handler, Publisher: delivery.Envelope.Publisher, EventName: delivery.Envelope.Name,
		SchemaVersion: delivery.Envelope.SchemaVersion, TenantID: delivery.Envelope.Scope.TenantID,
		DeliveryJSON: string(encoded), Status: string(eventinbox.StatusProcessing), AttemptCount: 1,
		LeaseOwner: workerID, LeaseToken: token, LeaseExpiresAt: &leaseExpiresAt,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (m PluginEventInboxModel) eventInboxRecord() (eventinbox.Record, error) {
	var delivery pluginsdk.EventDelivery
	if err := json.Unmarshal([]byte(m.DeliveryJSON), &delivery); err != nil {
		return eventinbox.Record{}, err
	}
	return eventinbox.Record{
		Delivery: delivery, Status: eventinbox.Status(m.Status), AttemptCount: m.AttemptCount,
		LeaseOwner: m.LeaseOwner, LeaseToken: m.LeaseToken, LeaseExpiresAt: cloneSQLTime(m.LeaseExpiresAt),
		LastError: m.LastError, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt, ProcessedAt: cloneSQLTime(m.ProcessedAt),
	}, nil
}

func newPluginEventInboxLeaseToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

var _ eventinbox.Store = (*PluginEventInboxStore)(nil)
