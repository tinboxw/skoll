package gormrepo

import (
	"context"
	"strings"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuditStore struct {
	db *gorm.DB
}

func NewAuditStore(db *gorm.DB) *AuditStore {
	return &AuditStore{db: db}
}

func (s *AuditStore) Append(ctx context.Context, record *domainaudit.Record) error {
	if record == nil || strings.TrimSpace(record.ID.String()) == "" {
		return nil
	}
	row := AuditRecordModelFromDomain(record)
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (s *AuditStore) GetByID(ctx context.Context, id shared.ID) (*domainaudit.Record, error) {
	key := strings.TrimSpace(id.String())
	if key == "" {
		return nil, nil
	}
	var row AuditRecordModel
	err := s.db.WithContext(ctx).Where("id = ?", key).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row.ToDomain(), nil
}

func (s *AuditStore) ListByActor(ctx context.Context, actorID shared.ID, limit int) ([]*domainaudit.Record, error) {
	key := strings.TrimSpace(actorID.String())
	if key == "" {
		return []*domainaudit.Record{}, nil
	}
	q := s.db.WithContext(ctx).Where("actor_id = ?", key).Order("occurred_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []AuditRecordModel
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainaudit.Record, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *AuditStore) ListByTimeRange(ctx context.Context, tr shared.TimeRange, limit int) ([]*domainaudit.Record, error) {
	q := s.db.WithContext(ctx).
		Where("occurred_at >= ? AND occurred_at <= ?", tr.From, tr.To).
		Order("occurred_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []AuditRecordModel
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainaudit.Record, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ToDomain())
	}
	return out, nil
}

func (s *AuditStore) DeleteByTimeRange(ctx context.Context, tr shared.TimeRange) (int, error) {
	q := s.db.WithContext(ctx).Where("occurred_at >= ? AND occurred_at <= ?", tr.From, tr.To)
	result := q.Delete(&AuditRecordModel{})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}
