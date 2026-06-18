package gormrepo

import (
	"context"
	"strings"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
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

func (s *AuditStore) AppendEvent(ctx context.Context, event *domainaudit.Event) error {
	if event == nil || strings.TrimSpace(event.ID.String()) == "" {
		return nil
	}
	row := AuditEventModelFromDomain(event)
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	})
}

func (s *AuditStore) GetEventByID(ctx context.Context, id shared.ID) (*domainaudit.Event, error) {
	key := strings.TrimSpace(id.String())
	if key == "" {
		return nil, nil
	}
	var row AuditEventModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("id = ?", key).First(&row).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row.ToDomain()
}

func (s *AuditStore) ListEvents(ctx context.Context, filter auditrepo.EventFilter) ([]*domainaudit.Event, error) {
	var rows []AuditEventModel
	if err := withDBRetry(func() error {
		return applyAuditEventFilter(s.db.WithContext(ctx).Model(&AuditEventModel{}), filter).
			Order("occurred_at desc, id asc").
			Find(&rows).Error
	}); err != nil {
		return nil, err
	}
	out := make([]*domainaudit.Event, 0, len(rows))
	for _, row := range rows {
		event, err := row.ToDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, nil
}

func (s *AuditStore) ExportEventSourceData(ctx context.Context, filter auditrepo.EventFilter) ([]auditrepo.EventSourceData, error) {
	var rows []AuditEventModel
	if err := withDBRetry(func() error {
		return applyAuditEventFilter(s.db.WithContext(ctx).Model(&AuditEventModel{}), filter).
			Order("occurred_at desc, id asc").
			Find(&rows).Error
	}); err != nil {
		return nil, err
	}
	out := make([]auditrepo.EventSourceData, 0, len(rows))
	for _, row := range rows {
		sourceData, err := unmarshalAuditMap(row.SourceJSON)
		if err != nil {
			return nil, err
		}
		out = append(out, auditrepo.EventSourceData{
			EventID:    shared.ID(row.ID),
			SourceData: sourceData,
		})
	}
	return out, nil
}

func applyAuditEventFilter(q *gorm.DB, filter auditrepo.EventFilter) *gorm.DB {
	if filter.Type != "" {
		q = q.Where("event_type = ?", string(filter.Type))
	}
	if !filter.ActorID.IsZero() {
		q = q.Where("actor_id = ?", filter.ActorID.String())
	}
	if filter.Action != "" {
		q = q.Where("action = ?", string(filter.Action))
	}
	if strings.TrimSpace(filter.ResourceType) != "" {
		q = q.Where("resource_type = ?", strings.TrimSpace(strings.ToLower(filter.ResourceType)))
	}
	if strings.TrimSpace(filter.ResourceID) != "" {
		q = q.Where("resource_id = ?", strings.TrimSpace(filter.ResourceID))
	}
	if filter.Result != "" {
		q = q.Where("result = ?", string(filter.Result))
	}
	if filter.Risk != "" {
		q = q.Where("risk = ?", string(filter.Risk))
	}
	if filter.TimeRange.IsValid() {
		q = q.Where("occurred_at >= ? AND occurred_at <= ?", filter.TimeRange.From, filter.TimeRange.To)
	}
	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}
	return q
}
