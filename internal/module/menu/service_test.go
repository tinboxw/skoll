package menu

import "testing"

func TestServiceCreateGetList(t *testing.T) {
	svc := NewService()
	m1 := svc.Create("Dashboard", "/dashboard", 20)
	m2 := svc.Create("System", "/system", 10)

	got, err := svc.Get(m1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Title != "Dashboard" {
		t.Fatalf("unexpected menu title: %s", got.Title)
	}

	list := svc.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 menu items, got %d", len(list))
	}
	if list[0].ID != m2.ID || list[1].ID != m1.ID {
		t.Fatalf("expected menu sorted by order")
	}
}

func BenchmarkServiceList(b *testing.B) {
	svc := NewService()
	for i := 0; i < 1000; i++ {
		svc.Create("menu", "/path", i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.List()
	}
}
