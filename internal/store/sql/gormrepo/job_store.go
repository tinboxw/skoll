package gormrepo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type JobStore struct {
	db *gorm.DB
}

func NewJobStore(db *gorm.DB) *JobStore {
	return &JobStore{db: db}
}

func (s *JobStore) Schedule(ctx context.Context, item jobsvc.Job) (jobsvc.Job, bool, error) {
	if s == nil || s.db == nil {
		return jobsvc.Job{}, false, fmt.Errorf("job repository is required")
	}
	row := jobRow(item)
	var stored JobModel
	created := false
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
			if result.Error != nil {
				return result.Error
			}
			created = result.RowsAffected == 1
			if err := tx.Where("id = ?", row.ID).First(&stored).Error; err == nil {
				return nil
			} else if !errors.Is(err, gorm.ErrRecordNotFound) || row.IdempotencyKey == nil {
				return err
			}
			return tx.Where("namespace = ? AND idempotency_key = ?", row.Namespace, *row.IdempotencyKey).First(&stored).Error
		})
	})
	if err != nil {
		return jobsvc.Job{}, false, err
	}
	return jobFromRow(stored), created, nil
}

func (s *JobStore) LeaseDue(ctx context.Context, namespace, workerID string, now, leaseUntil time.Time, limit int) ([]jobsvc.Job, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("job repository is required")
	}
	var leased []jobsvc.Job
	err := withDBRetry(func() error {
		leased = nil
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := expireExhaustedJobs(tx, namespace, now); err != nil {
				return err
			}
			var ids []string
			candidateLimit := limit * 4
			if candidateLimit > 400 {
				candidateLimit = 400
			}
			eligible := tx.Model(&JobModel{}).
				Where("namespace = ?", namespace).
				Where("attempt_count < max_attempts").
				Where("((status IN ? AND run_at <= ?) OR (status = ? AND lease_expires_at <= ?))",
					[]string{string(jobsvc.StatusScheduled), string(jobsvc.StatusRetryWait)}, now, string(jobsvc.StatusRunning), now).
				Order("run_at ASC").Order("id ASC").Limit(candidateLimit)
			if err := eligible.Pluck("id", &ids).Error; err != nil {
				return err
			}
			for _, id := range ids {
				if len(leased) >= limit {
					break
				}
				token, err := newSQLLeaseToken()
				if err != nil {
					return err
				}
				result := tx.Model(&JobModel{}).
					Where("id = ? AND namespace = ? AND attempt_count < max_attempts", id, namespace).
					Where("((status IN ? AND run_at <= ?) OR (status = ? AND lease_expires_at <= ?))",
						[]string{string(jobsvc.StatusScheduled), string(jobsvc.StatusRetryWait)}, now, string(jobsvc.StatusRunning), now).
					Updates(map[string]any{
						"status": string(jobsvc.StatusRunning), "attempt_count": gorm.Expr("attempt_count + 1"),
						"lease_owner": workerID, "lease_token": token, "lease_expires_at": leaseUntil, "updated_at": now,
					})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					continue
				}
				var row JobModel
				if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
					return err
				}
				leased = append(leased, jobFromRow(row))
			}
			return nil
		})
	})
	return leased, err
}

func (s *JobStore) Complete(ctx context.Context, id, leaseToken string, result []byte, now time.Time) (jobsvc.Job, error) {
	if s == nil || s.db == nil {
		return jobsvc.Job{}, fmt.Errorf("job repository is required")
	}
	var stored JobModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			updates := map[string]any{
				"status": string(jobsvc.StatusSucceeded), "result_json": string(result), "completed_at": now, "updated_at": now,
				"lease_owner": "", "lease_token": "", "lease_expires_at": nil,
			}
			result := tx.Model(&JobModel{}).
				Where("id = ? AND status = ? AND lease_token = ? AND lease_expires_at > ?", id, string(jobsvc.StatusRunning), leaseToken, now).
				Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return jobTransitionError(tx, id)
			}
			return tx.Where("id = ?", id).First(&stored).Error
		})
	})
	if err != nil {
		return jobsvc.Job{}, err
	}
	return jobFromRow(stored), nil
}

func (s *JobStore) Fail(ctx context.Context, id, leaseToken, message string, retryAt, now time.Time) (jobsvc.Job, error) {
	if s == nil || s.db == nil {
		return jobsvc.Job{}, fmt.Errorf("job repository is required")
	}
	var stored JobModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var current JobModel
			if err := tx.Where("id = ?", id).First(&current).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return jobsvc.ErrNotFound
				}
				return err
			}
			if current.Status != string(jobsvc.StatusRunning) || current.LeaseToken != leaseToken || current.LeaseExpiresAt == nil || !current.LeaseExpiresAt.After(now) {
				return jobsvc.ErrLeaseLost
			}
			status := jobsvc.StatusRetryWait
			updates := map[string]any{
				"status": string(status), "last_error": message, "updated_at": now,
				"lease_owner": "", "lease_token": "", "lease_expires_at": nil,
			}
			if current.AttemptCount >= current.MaxAttempts {
				status = jobsvc.StatusDeadLetter
				updates["status"] = string(status)
				updates["dead_lettered_at"] = now
			} else {
				updates["run_at"] = retryAt
			}
			result := tx.Model(&JobModel{}).
				Where("id = ? AND status = ? AND lease_token = ? AND lease_expires_at > ?", id, string(jobsvc.StatusRunning), leaseToken, now).
				Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return jobsvc.ErrLeaseLost
			}
			return tx.Where("id = ?", id).First(&stored).Error
		})
	})
	if err != nil {
		return jobsvc.Job{}, err
	}
	return jobFromRow(stored), nil
}

func (s *JobStore) Get(ctx context.Context, id string) (jobsvc.Job, error) {
	if s == nil || s.db == nil {
		return jobsvc.Job{}, fmt.Errorf("job repository is required")
	}
	var row JobModel
	err := withDBRetry(func() error { return s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error })
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return jobsvc.Job{}, jobsvc.ErrNotFound
	}
	if err != nil {
		return jobsvc.Job{}, err
	}
	return jobFromRow(row), nil
}

func (s *JobStore) List(ctx context.Context, filter jobsvc.Filter) ([]jobsvc.Job, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("job repository is required")
	}
	query := s.db.WithContext(ctx).Model(&JobModel{})
	if filter.Namespace != "" {
		query = query.Where("namespace = ?", filter.Namespace)
	}
	if filter.Kind != "" {
		query = query.Where("kind = ?", filter.Kind)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", string(filter.Status))
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	var rows []JobModel
	if err := withDBRetry(func() error { return query.Order("updated_at DESC").Order("id ASC").Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]jobsvc.Job, 0, len(rows))
	for _, row := range rows {
		out = append(out, jobFromRow(row))
	}
	return out, nil
}

func expireExhaustedJobs(tx *gorm.DB, namespace string, now time.Time) error {
	return tx.Model(&JobModel{}).
		Where("namespace = ? AND status = ? AND lease_expires_at <= ? AND attempt_count >= max_attempts", namespace, string(jobsvc.StatusRunning), now).
		Updates(map[string]any{
			"status": string(jobsvc.StatusDeadLetter), "last_error": "job lease expired after final attempt",
			"dead_lettered_at": now, "updated_at": now, "lease_owner": "", "lease_token": "", "lease_expires_at": nil,
		}).Error
}

func jobTransitionError(tx *gorm.DB, id string) error {
	var count int64
	if err := tx.Model(&JobModel{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return jobsvc.ErrNotFound
	}
	return jobsvc.ErrLeaseLost
}

func jobRow(item jobsvc.Job) JobModel {
	var key *string
	if item.IdempotencyKey != "" {
		value := item.IdempotencyKey
		key = &value
	}
	return JobModel{
		ID: item.ID, Namespace: item.Namespace, Kind: item.Kind, IdempotencyKey: key, PayloadJSON: string(item.Payload),
		Status: string(item.Status), RunAt: item.RunAt, MaxAttempts: item.MaxAttempts, AttemptCount: item.AttemptCount,
		LeaseOwner: item.LeaseOwner, LeaseToken: item.LeaseToken, LeaseExpiresAt: item.LeaseExpiresAt, LastError: item.LastError,
		ResultJSON: string(item.Result), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, CompletedAt: item.CompletedAt, DeadLetteredAt: item.DeadLetteredAt,
	}
}

func jobFromRow(row JobModel) jobsvc.Job {
	key := ""
	if row.IdempotencyKey != nil {
		key = *row.IdempotencyKey
	}
	return jobsvc.Job{
		ID: row.ID, Namespace: row.Namespace, Kind: row.Kind, IdempotencyKey: key, Payload: []byte(row.PayloadJSON),
		Status: jobsvc.Status(row.Status), RunAt: row.RunAt, MaxAttempts: row.MaxAttempts, AttemptCount: row.AttemptCount,
		LeaseOwner: row.LeaseOwner, LeaseToken: row.LeaseToken, LeaseExpiresAt: row.LeaseExpiresAt, LastError: row.LastError,
		Result: []byte(row.ResultJSON), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, CompletedAt: row.CompletedAt, DeadLetteredAt: row.DeadLetteredAt,
	}
}

func newSQLLeaseToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

var _ jobsvc.Repository = (*JobStore)(nil)
