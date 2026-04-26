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

func TestServiceQuery_FilterAndPaging(t *testing.T) {
	svc := NewService()
	svc.Append("alice", "create", "user:1")
	svc.Append("alice", "update", "user:2")
	svc.Append("bob", "create", "role:1")
	svc.Append("alice", "create", "user:3")

	filtered := svc.Query(Query{Actor: "alice", Action: "create"})
	if filtered.Total != 2 {
		t.Fatalf("expected 2 filtered records, got %d", filtered.Total)
	}

	paged := svc.Query(Query{Page: 2, Size: 1, Actor: "alice"})
	if paged.Total != 3 {
		t.Fatalf("expected total 3 records for alice, got %d", paged.Total)
	}
	if len(paged.Items) != 1 {
		t.Fatalf("expected page size 1, got %d", len(paged.Items))
	}
	if paged.Items[0].Action != "update" || paged.Items[0].Target != "user:2" {
		t.Fatalf("unexpected paged item: %+v", paged.Items[0])
	}

	keyword := svc.Query(Query{Q: "role"})
	if keyword.Total != 1 {
		t.Fatalf("expected 1 keyword match, got %d", keyword.Total)
	}
	if keyword.Items[0].Target != "role:1" {
		t.Fatalf("unexpected keyword match item: %+v", keyword.Items[0])
	}
}
