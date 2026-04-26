package role

import "testing"

func TestServiceCreateGetList(t *testing.T) {
	svc := NewService()
	r1 := svc.Create("admin", []string{"read", "write"})
	r2 := svc.Create("viewer", []string{"read"})

	got, err := svc.Get(r1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Name != "admin" {
		t.Fatalf("unexpected role name: %s", got.Name)
	}

	list := svc.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(list))
	}
	if list[0].ID != r1.ID || list[1].ID != r2.ID {
		t.Fatalf("expected roles sorted by ID")
	}
}

func BenchmarkServiceList(b *testing.B) {
	svc := NewService()
	for i := 0; i < 1000; i++ {
		svc.Create("role", []string{"read"})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.List()
	}
}
