package audit

import "testing"

func TestServiceAppendRecent(t *testing.T) {
	svc := NewService()
	r1 := svc.Append("alice", "create", "user:1")
	r2 := svc.Append("bob", "update", "role:1")

	if r1.ID == 0 || r2.ID == 0 {
		t.Fatalf("expected generated IDs")
	}

	recent := svc.Recent(1)
	if len(recent) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recent))
	}
	if recent[0].ID != r2.ID {
		t.Fatalf("expected latest record")
	}
}

func BenchmarkServiceRecent(b *testing.B) {
	svc := NewService()
	for i := 0; i < 10000; i++ {
		svc.Append("actor", "action", "target")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.Recent(100)
	}
}
