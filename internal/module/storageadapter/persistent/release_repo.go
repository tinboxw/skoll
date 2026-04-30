package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/tinboxw/skoll/internal/module/releasegov"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// ReleaseEvidenceModel persists a single submitted evidence row. The full
// EvidenceInput is stored as JSON to remain forward-compatible with new
// optional fields without schema migrations.
type ReleaseEvidenceModel struct {
	ID                 int64  `gorm:"primaryKey;autoIncrement"`
	Milestone          string `gorm:"size:128;index;not null"`
	Payload            string `gorm:"type:text;not null"`
	CollectedAtUnixSec int64  `gorm:"not null;index"`
}

func (ReleaseEvidenceModel) TableName() string { return "skoll_release_evidence" }

// ReleaseRepository is a SQL-backed implementation of
// contracts.ReleaseRepository using a write-through pattern.
type ReleaseRepository struct {
	db    *gorm.DB
	mu    sync.Mutex
	inner *releasegov.Service
}

// NewReleaseRepository constructs a repository, hydrating evidence from the
// database. Tables must be migrated before construction.
func NewReleaseRepository(database *gorm.DB) (*ReleaseRepository, error) {
	if database == nil {
		return nil, fmt.Errorf("persistent: nil db")
	}
	r := &ReleaseRepository{db: database, inner: releasegov.NewService()}
	if err := r.hydrate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Models returns the gorm models managed by this repository.
func (r *ReleaseRepository) Models() []any {
	return []any{&ReleaseEvidenceModel{}}
}

func (r *ReleaseRepository) ctx() context.Context { return context.Background() }

func (r *ReleaseRepository) hydrate() error {
	var rows []ReleaseEvidenceModel
	if err := r.db.WithContext(r.ctx()).Order("id asc").Find(&rows).Error; err != nil {
		return fmt.Errorf("hydrate release evidence: %w", err)
	}
	out := make([]releasegov.Evidence, 0, len(rows))
	for _, row := range rows {
		var ev releasegov.Evidence
		if err := json.Unmarshal([]byte(row.Payload), &ev); err != nil {
			return fmt.Errorf("decode release evidence id=%d: %w", row.ID, err)
		}
		out = append(out, ev)
	}
	r.inner.ImportEvidence(out)
	return nil
}

// SubmitEvidence delegates to the inner service then persists the resulting
// evidence row.
func (r *ReleaseRepository) SubmitEvidence(input releasegov.EvidenceInput, now time.Time) (releasegov.Evidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ev, err := r.inner.SubmitEvidence(input, now)
	if err != nil {
		return ev, err
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return ev, fmt.Errorf("encode release evidence: %w", err)
	}
	row := ReleaseEvidenceModel{
		Milestone:          ev.Milestone,
		Payload:            string(payload),
		CollectedAtUnixSec: ev.CollectedAtUnixSec,
	}
	if err := r.db.WithContext(r.ctx()).Create(&row).Error; err != nil {
		return ev, fmt.Errorf("persist release evidence: %w", err)
	}
	return ev, nil
}

// Scorecard delegates to the inner service.
func (r *ReleaseRepository) Scorecard(milestone string, allowedRegression float64) releasegov.Scorecard {
	return r.inner.Scorecard(milestone, allowedRegression)
}

var _ contracts.ReleaseRepository = (*ReleaseRepository)(nil)
