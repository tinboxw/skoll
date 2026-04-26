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

func TestServiceRolePolicies(t *testing.T) {
	svc := NewService()
	rules := svc.SetRolePolicies(3, []PolicyRule{
		{API: "GET:/admin/v1/users", Effect: "ALLOW", RequireVerified: true, RequireClaimsVersion: "V2"},
		{API: "GET:/admin/v1/users", Effect: "allow", RequireVerified: true, RequireClaimsVersion: "v2"},
		{API: "POST:/admin/v1/users", Effect: "deny"},
		{API: "", Effect: "allow"},
		{API: "GET:/admin/v1/users", Effect: "unknown"},
	})

	if len(rules) != 2 {
		t.Fatalf("expected 2 normalized policy rules, got %d", len(rules))
	}
	if rules[0].Effect != "allow" || rules[0].RequireClaimsVersion != "v2" {
		t.Fatalf("expected normalized allow rule, got %+v", rules[0])
	}
	if rules[1].Effect != "deny" {
		t.Fatalf("expected deny rule, got %+v", rules[1])
	}

	read := svc.GetRolePolicies(3)
	if !reflect.DeepEqual(read, rules) {
		t.Fatalf("unexpected role policies: got %v want %v", read, rules)
	}
}

func TestServiceRoleDataScope(t *testing.T) {
	svc := NewService()
	scope := svc.SetRoleDataScope(4, DataScope{TenantIDs: []string{"tenant-b", "tenant-a", "tenant-a", ""}, RequireOwnerMatch: true})

	if !reflect.DeepEqual(scope.TenantIDs, []string{"tenant-a", "tenant-b"}) {
		t.Fatalf("unexpected normalized tenant ids: got %v", scope.TenantIDs)
	}
	if !scope.RequireOwnerMatch {
		t.Fatalf("expected require owner match true")
	}

	read := svc.GetRoleDataScope(4)
	if !reflect.DeepEqual(read, scope) {
		t.Fatalf("unexpected role data scope: got %v want %v", read, scope)
	}
}

func TestServiceRoleRoutePermissions(t *testing.T) {
	svc := NewService()
	svc.SetRoleMenus(9, []int64{1, 2})

	contract := svc.SetRoleRoutePermissions(9, "V2", []RoutePermissionItem{
		{MenuID: 2, Route: "/system/users", Buttons: []string{"create", "delete", "create"}},
		{MenuID: 2, Route: "/system/users", Buttons: []string{"create", "delete"}},
		{MenuID: 3, Route: "/system/roles", Buttons: []string{"assign"}},
	})

	if contract.Version != "v2" {
		t.Fatalf("expected normalized contract version v2, got %s", contract.Version)
	}
	if len(contract.Items) != 2 {
		t.Fatalf("expected deduplicated route permission items, got %d", len(contract.Items))
	}

	read := svc.GetRoleRoutePermissions(9)
	if !reflect.DeepEqual(read, contract) {
		t.Fatalf("unexpected route permission contract: got %+v want %+v", read, contract)
	}

	consistency := svc.CheckRoleRoutePermissionConsistency(9)
	if consistency.Passed {
		t.Fatalf("expected consistency check to fail because menu 3 is not role-bound")
	}
	if len(consistency.Problems) == 0 {
		t.Fatalf("expected consistency problems")
	}

	fixed := svc.SetRoleRoutePermissions(9, "", []RoutePermissionItem{{MenuID: 1, Route: "/system/dashboard", Buttons: []string{"view"}}})
	if fixed.Version != "v1" {
		t.Fatalf("expected default contract version v1, got %s", fixed.Version)
	}

	consistency = svc.CheckRoleRoutePermissionConsistency(9)
	if !consistency.Passed {
		t.Fatalf("expected consistency check passed, got %+v", consistency)
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
