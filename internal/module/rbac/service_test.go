package rbac

import (
	"reflect"
	"testing"
)

func TestServiceRoleMenus(t *testing.T) {
	svc := NewService()
	got := svc.SetRoleMenus(1, []int64{3, 1, 2, 2, 0, -1})
	want := []int64{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected role menus: got %v want %v", got, want)
	}

	read := svc.GetRoleMenus(1)
	if !reflect.DeepEqual(read, want) {
		t.Fatalf("unexpected get role menus: got %v want %v", read, want)
	}
}

func TestServiceRoleAPIs(t *testing.T) {
	svc := NewService()
	got := svc.SetRoleAPIs(2, []string{"POST:/admin/v1/users", "", "GET:/admin/v1/users", "GET:/admin/v1/users"})
	want := []string{"GET:/admin/v1/users", "POST:/admin/v1/users"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected role apis: got %v want %v", got, want)
	}

	read := svc.GetRoleAPIs(2)
	if !reflect.DeepEqual(read, want) {
		t.Fatalf("unexpected get role apis: got %v want %v", read, want)
	}
}

func BenchmarkSetRoleMenus(b *testing.B) {
	svc := NewService()
	menuIDs := make([]int64, 100)
	for i := 0; i < 100; i++ {
		menuIDs[i] = int64((i % 20) + 1)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.SetRoleMenus(1, menuIDs)
	}
}
