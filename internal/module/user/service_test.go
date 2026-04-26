package user

import "testing"

func TestServiceCreateGetList(t *testing.T) {
	svc := NewService()
	u1 := svc.Create("alice", "alice@example.com")
	u2 := svc.Create("bob", "bob@example.com")

	got, err := svc.Get(u1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Fatalf("unexpected user email: %s", got.Email)
	}

	list := svc.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list))
	}
	if list[0].ID != u1.ID || list[1].ID != u2.ID {
		t.Fatalf("expected users sorted by ID")
	}
}

func BenchmarkServiceList(b *testing.B) {
	svc := NewService()
	for i := 0; i < 1000; i++ {
		svc.Create("name", "mail@example.com")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.List()
	}
}
