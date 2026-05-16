package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

type fakeAuditRepo struct {
	appendErr          error
	getByIDErr         error
	listByActorErr     error
	listByTimeRangeErr error
	deleteErr          error

	appendCalled          bool
	listByTimeRangeCalled bool
	deleteByTimeRangeCall bool
}

func (f *fakeAuditRepo) Append(_ context.Context, _ *domainaudit.Record) error {
	f.appendCalled = true
	return f.appendErr
}

func (f *fakeAuditRepo) GetByID(_ context.Context, _ shared.ID) (*domainaudit.Record, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	return nil, nil
}

func (f *fakeAuditRepo) ListByActor(_ context.Context, _ shared.ID, _ int) ([]*domainaudit.Record, error) {
	if f.listByActorErr != nil {
		return nil, f.listByActorErr
	}
	return nil, nil
}

func (f *fakeAuditRepo) ListByTimeRange(_ context.Context, _ shared.TimeRange, _ int) ([]*domainaudit.Record, error) {
	f.listByTimeRangeCalled = true
	if f.listByTimeRangeErr != nil {
		return nil, f.listByTimeRangeErr
	}
	return nil, nil
}

func (f *fakeAuditRepo) DeleteByTimeRange(_ context.Context, _ shared.TimeRange) (int, error) {
	f.deleteByTimeRangeCall = true
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	return 0, nil
}

func TestAuditServiceAppendAndList(t *testing.T) {
	svc := NewService(clickhouse.NewAuditStore())

	rec, err := svc.Append(context.Background(), "actor-1", "create", "user", "u-1", map[string]any{"ip": "127.0.0.1"})
	if err != nil {
		t.Fatalf("Append error: %v", err)
	}
	if rec.ID == "" {
		t.Fatalf("expected record id")
	}

	items, err := svc.ListByActor(context.Background(), "actor-1", 10)
	if err != nil {
		t.Fatalf("ListByActor error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected audit items")
	}

	got, err := svc.GetByID(context.Background(), rec.ID.String())
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if got == nil || got.ID != rec.ID {
		t.Fatalf("expected matched audit record, got %+v", got)
	}

	from := rec.OccurredAt.Add(-time.Second)
	to := rec.OccurredAt.Add(time.Second)
	rangeItems, err := svc.ListByTimeRange(context.Background(), from, to, 10)
	if err != nil {
		t.Fatalf("ListByTimeRange error: %v", err)
	}
	if len(rangeItems) == 0 {
		t.Fatalf("expected range items")
	}

	deleted, err := svc.ClearByTimeRange(context.Background(), from, to)
	if err != nil {
		t.Fatalf("ClearByTimeRange error: %v", err)
	}
	if deleted <= 0 {
		t.Fatalf("expected deleted count > 0, got %d", deleted)
	}
}

func TestAuditServiceAppendValidationError(t *testing.T) {
	repo := &fakeAuditRepo{}
	svc := NewService(repo)

	_, err := svc.Append(context.Background(), "actor-1", "", "user", "u-1", nil)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if repo.appendCalled {
		t.Fatalf("append should not be called when input is invalid")
	}
}

func TestAuditServiceAppendRepoError(t *testing.T) {
	repo := &fakeAuditRepo{appendErr: errors.New("append failed")}
	svc := NewService(repo)

	_, err := svc.Append(context.Background(), "actor-1", "create", "user", "u-1", nil)
	if err == nil {
		t.Fatalf("expected append repo error")
	}
	if !repo.appendCalled {
		t.Fatalf("expected append to be called")
	}
}

func TestAuditServiceGetByIDRepoError(t *testing.T) {
	repo := &fakeAuditRepo{getByIDErr: errors.New("get failed")}
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), "audit-1")
	if err == nil {
		t.Fatalf("expected get-by-id repo error")
	}
}

func TestAuditServiceListByActorRepoError(t *testing.T) {
	repo := &fakeAuditRepo{listByActorErr: errors.New("list failed")}
	svc := NewService(repo)

	_, err := svc.ListByActor(context.Background(), "actor-1", 10)
	if err == nil {
		t.Fatalf("expected list-by-actor repo error")
	}
}

func TestAuditServiceListByTimeRangeInvalidRange(t *testing.T) {
	repo := &fakeAuditRepo{}
	svc := NewService(repo)

	from := time.Now().UTC()
	to := from.Add(-time.Minute)

	_, err := svc.ListByTimeRange(context.Background(), from, to, 10)
	if !errors.Is(err, domainaudit.ErrInvalidTimeRange) {
		t.Fatalf("expected ErrInvalidTimeRange, got %v", err)
	}
	if repo.listByTimeRangeCalled {
		t.Fatalf("list by time range should not call repo for invalid range")
	}
}

func TestAuditServiceListByTimeRangeRepoError(t *testing.T) {
	repo := &fakeAuditRepo{listByTimeRangeErr: errors.New("time range failed")}
	svc := NewService(repo)

	from := time.Now().UTC().Add(-time.Minute)
	to := time.Now().UTC()

	_, err := svc.ListByTimeRange(context.Background(), from, to, 10)
	if err == nil {
		t.Fatalf("expected list-by-time-range repo error")
	}
}

func TestAuditServiceClearByTimeRangeInvalidRange(t *testing.T) {
	repo := &fakeAuditRepo{}
	svc := NewService(repo)

	from := time.Now().UTC()
	to := from.Add(-time.Minute)

	_, err := svc.ClearByTimeRange(context.Background(), from, to)
	if !errors.Is(err, domainaudit.ErrInvalidTimeRange) {
		t.Fatalf("expected ErrInvalidTimeRange, got %v", err)
	}
	if repo.deleteByTimeRangeCall {
		t.Fatalf("delete by time range should not call repo for invalid range")
	}
}

func TestAuditServiceClearByTimeRangeRepoError(t *testing.T) {
	repo := &fakeAuditRepo{deleteErr: errors.New("delete failed")}
	svc := NewService(repo)

	from := time.Now().UTC().Add(-time.Minute)
	to := time.Now().UTC()

	_, err := svc.ClearByTimeRange(context.Background(), from, to)
	if err == nil {
		t.Fatalf("expected clear-by-time-range repo error")
	}
}
