package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	documentnumbersvc "github.com/tinboxw/skoll/internal/service/documentnumber"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DocumentNumberStore struct {
	db *gorm.DB
}

func NewDocumentNumberStore(db *gorm.DB) *DocumentNumberStore {
	if db == nil {
		panic("document number database is required")
	}
	return &DocumentNumberStore{db: db}
}

func (s *DocumentNumberStore) Last(ctx context.Context, key documentnumbersvc.SequenceKey) (documentnumbersvc.Sequence, bool, error) {
	var row DocumentNumberSequenceModel
	err := storesql.ResolveDB(ctx, s.db).Where(documentNumberKey(key)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return documentnumbersvc.Sequence{}, false, nil
	}
	return documentnumbersvc.Sequence{LastValue: row.LastValue, RuleHash: row.RuleHash}, err == nil, err
}

func (s *DocumentNumberStore) Reserve(ctx context.Context, request documentnumbersvc.ReservationRequest) (documentnumbersvc.Reservation, error) {
	db := storesql.DBFromContext(ctx)
	if db == nil {
		return documentnumbersvc.Reservation{}, documentnumbersvc.ErrTransactionRequired
	}
	now := time.Now().UTC()
	seed := DocumentNumberSequenceModel{
		PluginID: request.Key.PluginID, TenantID: request.Key.TenantID, DocumentType: request.Key.DocumentType,
		PeriodKey: request.Key.PeriodKey, RuleHash: request.RuleHash, LastValue: request.Rule.Start - 1, UpdatedAt: now,
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
		return documentnumbersvc.Reservation{}, err
	}
	where := documentNumberKey(request.Key)
	if err := db.Model(&DocumentNumberSequenceModel{}).Where(where).
		UpdateColumn("last_value", gorm.Expr("last_value")).Error; err != nil {
		return documentnumbersvc.Reservation{}, err
	}
	var sequence DocumentNumberSequenceModel
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where(where).First(&sequence).Error; err != nil {
		return documentnumbersvc.Reservation{}, err
	}
	var prior DocumentNumberIssueModel
	err := db.Where("plugin_id = ? AND tenant_id = ? AND document_type = ? AND idempotency_key = ?",
		request.Key.PluginID, request.Key.TenantID, request.Key.DocumentType, request.IdempotencyKey).First(&prior).Error
	if err == nil {
		if prior.RequestHash != request.RequestHash {
			return documentnumbersvc.Reservation{}, documentnumbersvc.ErrConflict
		}
		return reservationFromIssue(prior, true), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return documentnumbersvc.Reservation{}, err
	}
	if sequence.RuleHash != request.RuleHash {
		return documentnumbersvc.Reservation{}, documentnumbersvc.ErrRuleConflict
	}
	next := sequence.LastValue + 1
	number, err := documentnumbersvc.Format(request.Rule, request.Key.PeriodKey, next)
	if err != nil {
		return documentnumbersvc.Reservation{}, err
	}
	result := db.Model(&DocumentNumberSequenceModel{}).Where(where).Where("last_value = ?", sequence.LastValue).
		Updates(map[string]any{"last_value": next, "updated_at": now})
	if result.Error != nil {
		return documentnumbersvc.Reservation{}, result.Error
	}
	if result.RowsAffected != 1 {
		return documentnumbersvc.Reservation{}, fmt.Errorf("document number sequence changed concurrently")
	}
	issue := DocumentNumberIssueModel{
		PluginID: request.Key.PluginID, TenantID: request.Key.TenantID, DocumentType: request.Key.DocumentType,
		IdempotencyKey: request.IdempotencyKey, PeriodKey: request.Key.PeriodKey, SequenceValue: next,
		Number: number, RequestHash: request.RequestHash, IssuedAt: request.OccurredAt.UTC(),
	}
	if err := db.Create(&issue).Error; err != nil {
		return documentnumbersvc.Reservation{}, err
	}
	return reservationFromIssue(issue, false), nil
}

func documentNumberKey(key documentnumbersvc.SequenceKey) map[string]any {
	return map[string]any{
		"plugin_id": key.PluginID, "tenant_id": key.TenantID,
		"document_type": key.DocumentType, "period_key": key.PeriodKey,
	}
}

func reservationFromIssue(row DocumentNumberIssueModel, duplicate bool) documentnumbersvc.Reservation {
	return documentnumbersvc.Reservation{
		Sequence: row.SequenceValue, Number: row.Number, PeriodKey: row.PeriodKey,
		Duplicate: duplicate, RequestHash: row.RequestHash, IssuedAt: row.IssuedAt,
	}
}

var _ documentnumbersvc.Repository = (*DocumentNumberStore)(nil)
