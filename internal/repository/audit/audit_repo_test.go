package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

// MockAuditRepository implements AuditRepository for testing
type MockAuditRepository struct {
	records map[shared.ID]*audit.Record
	err     error
}

func NewMockAuditRepository() *MockAuditRepository {
	return &MockAuditRepository{
		records: make(map[shared.ID]*audit.Record),
	}
}

func (m *MockAuditRepository) WithError(err error) *MockAuditRepository {
	m.err = err
	return m
}

func (m *MockAuditRepository) WithRecords(records ...*audit.Record) *MockAuditRepository {
	for _, r := range records {
		m.records[r.ID] = r
	}
	return m
}

func (m *MockAuditRepository) Append(ctx context.Context, record *audit.Record) error {
	if m.err != nil {
		return m.err
	}
	if record.ID == "" {
		record.ID = shared.ID("generated-" + time.Now().Format("20060102150405"))
	}
	m.records[record.ID] = record
	return nil
}

func (m *MockAuditRepository) GetByID(ctx context.Context, id shared.ID) (*audit.Record, error) {
	if m.err != nil {
		return nil, m.err
	}
	r, ok := m.records[id]
	if !ok {
		return nil, errors.New("record not found")
	}
	return r, nil
}

func (m *MockAuditRepository) ListByActor(ctx context.Context, actorID shared.ID, limit int) ([]*audit.Record, error) {
	if m.err != nil {
		return nil, m.err
	}
	var records []*audit.Record
	for _, r := range m.records {
		if r.ActorID == actorID {
			records = append(records, r)
			if len(records) >= limit {
				break
			}
		}
	}
	return records, nil
}

func (m *MockAuditRepository) ListByTimeRange(ctx context.Context, tr shared.TimeRange, limit int) ([]*audit.Record, error) {
	if m.err != nil {
		return nil, m.err
	}
	var records []*audit.Record
	for _, r := range m.records {
		if (r.OccurredAt.After(tr.From) || r.OccurredAt.Equal(tr.From)) &&
			(r.OccurredAt.Before(tr.To) || r.OccurredAt.Equal(tr.To)) {
			records = append(records, r)
			if len(records) >= limit {
				break
			}
		}
	}
	return records, nil
}

func (m *MockAuditRepository) DeleteByTimeRange(ctx context.Context, tr shared.TimeRange) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	count := 0
	for id, r := range m.records {
		if (r.OccurredAt.After(tr.From) || r.OccurredAt.Equal(tr.From)) &&
			(r.OccurredAt.Before(tr.To) || r.OccurredAt.Equal(tr.To)) {
			delete(m.records, id)
			count++
		}
	}
	return count, nil
}

// Tests
func TestMockAuditRepository_Append(t *testing.T) {
	repo := NewMockAuditRepository()

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		record := &audit.Record{
			ID:         shared.ID("audit-1"),
			Action:     "user.login",
			ActorID:    shared.ID("user-1"),
			Resource:   "auth",
			ResourceID: "session-1",
			OccurredAt: time.Now(),
		}
		err := repo.Append(ctx, record)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, record.ID)
		if err != nil {
			t.Fatalf("failed to retrieve: %v", err)
		}
		if retrieved.Action != "user.login" {
			t.Errorf("expected action user.login, got %s", retrieved.Action)
		}
	})

	t.Run("with error", func(t *testing.T) {
		errRepo := repo.WithError(errors.New("storage error"))
		ctx := context.Background()
		err := errRepo.Append(ctx, &audit.Record{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMockAuditRepository_ListByActor(t *testing.T) {
	repo := NewMockAuditRepository()
	now := time.Now()
	records := []*audit.Record{
		{ID: shared.ID("a1"), Action: "create", ActorID: shared.ID("user-1"), Resource: "test", OccurredAt: now},
		{ID: shared.ID("a2"), Action: "update", ActorID: shared.ID("user-1"), Resource: "test", OccurredAt: now},
		{ID: shared.ID("a3"), Action: "delete", ActorID: shared.ID("user-2"), Resource: "test", OccurredAt: now},
	}
	repo.WithRecords(records...)

	t.Run("filter by actor", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.ListByActor(ctx, shared.ID("user-1"), 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 records for user-1, got %d", len(result))
		}
	})

	t.Run("with limit", func(t *testing.T) {
		ctx := context.Background()
		result, err := repo.ListByActor(ctx, shared.ID("user-1"), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 record with limit=1, got %d", len(result))
		}
	})
}

func TestMockAuditRepository_ListByTimeRange(t *testing.T) {
	repo := NewMockAuditRepository()
	baseTime := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	records := []*audit.Record{
		{ID: shared.ID("t1"), OccurredAt: baseTime.Add(-2 * time.Hour)},
		{ID: shared.ID("t2"), OccurredAt: baseTime.Add(-1 * time.Hour)},
		{ID: shared.ID("t3"), OccurredAt: baseTime},
		{ID: shared.ID("t4"), OccurredAt: baseTime.Add(1 * time.Hour)},
	}
	repo.WithRecords(records...)

	t.Run("filter by time range", func(t *testing.T) {
		ctx := context.Background()
		tr := shared.TimeRange{
			From: baseTime.Add(-90 * time.Minute),
			To:   baseTime.Add(90 * time.Minute),
		}
		result, err := repo.ListByTimeRange(ctx, tr, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should include t2, t3 (within -90min to +90min from baseTime)
		if len(result) < 2 {
			t.Errorf("expected at least 2 records in range, got %d", len(result))
		}
	})
}

func TestMockAuditRepository_DeleteByTimeRange(t *testing.T) {
	repo := NewMockAuditRepository()
	baseTime := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	records := []*audit.Record{
		{ID: shared.ID("d1"), OccurredAt: baseTime.Add(-30 * 24 * time.Hour)}, // 30 days ago
		{ID: shared.ID("d2"), OccurredAt: baseTime.Add(-7 * 24 * time.Hour)},  // 7 days ago
		{ID: shared.ID("d3"), OccurredAt: baseTime},                           // today
	}
	repo.WithRecords(records...)

	t.Run("delete old records", func(t *testing.T) {
		ctx := context.Background()
		tr := shared.TimeRange{
			From: time.Time{},                        // beginning of time
			To:   baseTime.Add(-14 * 24 * time.Hour), // older than 14 days
		}
		count, err := repo.DeleteByTimeRange(ctx, tr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 1 { // only d1 (30 days ago) should be deleted; d2 is only 7 days ago
			t.Errorf("expected to delete 1 record (30 days ago), got %d", count)
		}
	})
}
