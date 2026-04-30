package persistent

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// AuditRecordModel is the gorm representation of audit.Record.
type AuditRecordModel struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	Actor     string `gorm:"size:128;index"`
	Action    string `gorm:"size:128;index"`
	Target    string `gorm:"size:255;index"`
	CreatedAt time.Time
}

func (AuditRecordModel) TableName() string { return "skoll_audit_records" }

func (m AuditRecordModel) toDomain() audit.Record {
	return audit.Record{
		ID:        m.ID,
		Actor:     m.Actor,
		Action:    m.Action,
		Target:    m.Target,
		CreatedAt: m.CreatedAt.UTC(),
	}
}

// AuditRepository is a SQL-backed implementation of contracts.AuditRepository.
type AuditRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewAuditRepository constructs a SQL-backed audit repository.
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

func (r *AuditRepository) ctx() context.Context { return context.Background() }

// Models returns gorm models owned by this repo for migration registration.
func (r *AuditRepository) Models() []any { return []any{&AuditRecordModel{}} }

// Append persists a new audit record and returns the stored value.
func (r *AuditRepository) Append(actor, action, target string) audit.Record {
	row := AuditRecordModel{
		Actor:     actor,
		Action:    action,
		Target:    target,
		CreatedAt: r.now(),
	}
	if err := r.db.WithContext(r.ctx()).Create(&row).Error; err != nil {
		// In-memory contract guarantees no error path; surface a panic in
		// development. Production handlers wrap the call site with their own
		// error context and should switch to a method that returns error
		// once the contracts interface is widened.
		panic(err)
	}
	return row.toDomain()
}

// Recent returns the latest entries up to the supplied limit.
func (r *AuditRepository) Recent(limit int) []audit.Record {
	out := r.Query(audit.Query{Page: 1, Size: limit})
	return out.Items
}

// Query applies filters/pagination and returns the matching rows ordered by id desc.
func (r *AuditRepository) Query(raw audit.Query) audit.QueryResult {
	q := normalizeAuditQuery(raw)

	tx := r.db.WithContext(r.ctx()).Model(&AuditRecordModel{})
	if q.Actor != "" {
		tx = tx.Where("LOWER(actor) = ?", q.Actor)
	}
	if q.Action != "" {
		tx = tx.Where("LOWER(action) = ?", q.Action)
	}
	if q.Target != "" {
		tx = tx.Where("LOWER(target) = ?", q.Target)
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		tx = tx.Where("LOWER(actor) LIKE ? OR LOWER(action) LIKE ? OR LOWER(target) LIKE ?", like, like, like)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		panic(err)
	}
	if total == 0 {
		return audit.QueryResult{Page: q.Page, Size: q.Size, Total: 0, Items: nil}
	}

	offset := (q.Page - 1) * q.Size
	if int64(offset) >= total {
		return audit.QueryResult{Page: q.Page, Size: q.Size, Total: int(total), Items: nil}
	}

	var rows []AuditRecordModel
	if err := tx.Order("id DESC").Offset(offset).Limit(q.Size).Find(&rows).Error; err != nil {
		panic(err)
	}

	items := make([]audit.Record, len(rows))
	for i, row := range rows {
		items[i] = row.toDomain()
	}
	return audit.QueryResult{Page: q.Page, Size: q.Size, Total: int(total), Items: items}
}

func normalizeAuditQuery(raw audit.Query) audit.Query {
	q := raw
	if q.Page <= 0 {
		q.Page = audit.DefaultPage
	}
	if q.Size <= 0 {
		q.Size = audit.DefaultSize
	}
	if q.Size > audit.MaxSize {
		q.Size = audit.MaxSize
	}
	q.Actor = strings.TrimSpace(strings.ToLower(q.Actor))
	q.Action = strings.TrimSpace(strings.ToLower(q.Action))
	q.Target = strings.TrimSpace(strings.ToLower(q.Target))
	q.Q = strings.TrimSpace(strings.ToLower(q.Q))
	return q
}

// Compile-time interface check.
var _ contracts.AuditRepository = (*AuditRepository)(nil)

// errAuditNotFound is reserved for future error-returning APIs; currently
// unused but kept to mirror config/dictionary error contracts.
var errAuditNotFound = errors.New("audit record not found")

var _ = errAuditNotFound
