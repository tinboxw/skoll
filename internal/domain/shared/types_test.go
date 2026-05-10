package shared

import (
	"testing"
	"time"
)

func TestID(t *testing.T) {
	id := ID("user-1")
	if id.String() != "user-1" {
		t.Fatalf("unexpected id string: %s", id.String())
	}
	if id.IsZero() {
		t.Fatalf("id should not be zero")
	}
	if !ID("").IsZero() {
		t.Fatalf("empty id should be zero")
	}
}

func TestTimeRangeIsValid(t *testing.T) {
	now := time.Date(2026, time.May, 10, 12, 0, 0, 0, time.UTC)

	if !(TimeRange{From: now, To: now.Add(time.Minute)}).IsValid() {
		t.Fatalf("expected valid time range")
	}
	if (TimeRange{From: now, To: now.Add(-time.Minute)}).IsValid() {
		t.Fatalf("range with To before From should be invalid")
	}
	if (TimeRange{}).IsValid() {
		t.Fatalf("zero range should be invalid")
	}
}

func TestAuditMetaTouch(t *testing.T) {
	m := AuditMeta{}
	t1 := time.Date(2026, time.May, 10, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Minute)

	m.Touch(t1)
	if !m.CreatedAt.Equal(t1) || !m.UpdatedAt.Equal(t1) {
		t.Fatalf("unexpected timestamps after first touch: %+v", m)
	}

	m.Touch(t2)
	if !m.CreatedAt.Equal(t1) {
		t.Fatalf("CreatedAt should keep first touch time: %v", m.CreatedAt)
	}
	if !m.UpdatedAt.Equal(t2) {
		t.Fatalf("UpdatedAt should move to latest touch time: %v", m.UpdatedAt)
	}
}
