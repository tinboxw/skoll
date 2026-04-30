package persistent_test

import (
	"errors"
	"testing"

	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
)

func newRBAC(t *testing.T) *persistent.RBACRepository {
	t.Helper()
	gdb := newTestDB(t)
	// Migrate schema first; the constructor performs hydration which requires
	// tables to exist.
	if err := db.Migrate(gdb, (&persistent.RBACRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewRBACRepository(gdb)
	if err != nil {
		t.Fatalf("new rbac repo: %v", err)
	}
	return repo
}

func TestRBACRepository_RoleMenusAndAPIs(t *testing.T) {
	repo := newRBAC(t)
	roleID := int64(7)

	out := repo.SetRoleMenus(roleID, []int64{3, 1, 2, 1})
	if len(out) != 3 || out[0] != 1 || out[1] != 2 || out[2] != 3 {
		t.Fatalf("unexpected menus: %+v", out)
	}
	if got := repo.GetRoleMenus(roleID); len(got) != 3 {
		t.Fatalf("get menus: %+v", got)
	}

	apis := repo.SetRoleAPIs(roleID, []string{"GET:/a", "POST:/b", "GET:/a"})
	if len(apis) != 2 {
		t.Fatalf("unexpected apis: %+v", apis)
	}
}

func TestRBACRepository_PoliciesSnapshotsRollback(t *testing.T) {
	repo := newRBAC(t)
	roleID := int64(11)

	repo.SetRolePolicies(roleID, []rbac.PolicyRule{{API: "GET:/x", Effect: "allow"}})
	v1 := repo.CreateRolePolicySnapshot(roleID)
	if v1.Version == "" {
		t.Fatalf("expected version: %+v", v1)
	}

	repo.SetRolePolicies(roleID, []rbac.PolicyRule{{API: "POST:/y", Effect: "deny"}})
	v2 := repo.CreateRolePolicySnapshot(roleID)

	snaps := repo.ListRolePolicySnapshots(roleID)
	if len(snaps) != 2 || snaps[0].Version == snaps[1].Version {
		t.Fatalf("unexpected snapshots: %+v", snaps)
	}

	rolled, err := repo.RollbackRolePolicies(roleID, v1.Version)
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if len(rolled) != 1 || rolled[0].API != "GET:/x" {
		t.Fatalf("unexpected rolled rules: %+v", rolled)
	}

	if _, err := repo.RollbackRolePolicies(roleID, "v999"); !errors.Is(err, rbac.ErrPolicySnapshotNotFound) {
		t.Fatalf("expected snapshot-not-found, got %v", err)
	}

	_ = v2
}

func TestRBACRepository_DataScopeAndRoute(t *testing.T) {
	repo := newRBAC(t)
	roleID := int64(21)

	scope := repo.SetRoleDataScope(roleID, rbac.DataScope{TenantIDs: []string{"t2", "t1", "t1"}, RequireOwnerMatch: true})
	if len(scope.TenantIDs) != 2 || scope.TenantIDs[0] != "t1" || scope.TenantIDs[1] != "t2" {
		t.Fatalf("unexpected normalized scope: %+v", scope)
	}

	route := repo.SetRoleRoutePermissions(roleID, "v1", []rbac.RoutePermissionItem{
		{MenuID: 1, Route: "/a", Buttons: []string{"x"}},
	})
	if route.Version != "v1" || len(route.Items) != 1 {
		t.Fatalf("unexpected route: %+v", route)
	}

	repo.SetRoleMenus(roleID, []int64{1})
	cons := repo.CheckRoleRoutePermissionConsistency(roleID)
	if !cons.Passed {
		t.Fatalf("expected consistency pass, got %+v", cons)
	}
}

func TestRBACRepository_HydrateAcrossInstances(t *testing.T) {
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.RBACRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewRBACRepository(gdb)
	if err != nil {
		t.Fatalf("first new: %v", err)
	}

	roleID := int64(101)
	repo.SetRoleMenus(roleID, []int64{1, 2, 3})
	repo.SetRoleAPIs(roleID, []string{"GET:/a"})
	repo.SetRolePolicies(roleID, []rbac.PolicyRule{{API: "GET:/a", Effect: "allow"}})
	repo.SetRoleDataScope(roleID, rbac.DataScope{TenantIDs: []string{"t1"}, RequireOwnerMatch: true})
	repo.SetRoleRoutePermissions(roleID, "v1", []rbac.RoutePermissionItem{{MenuID: 1, Route: "/a"}})
	first := repo.CreateRolePolicySnapshot(roleID)

	// Recreate the repo over the same database.
	repo2, err := persistent.NewRBACRepository(gdb)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if got := repo2.GetRoleMenus(roleID); len(got) != 3 {
		t.Fatalf("hydrated menus: %+v", got)
	}
	if got := repo2.GetRoleAPIs(roleID); len(got) != 1 || got[0] != "GET:/a" {
		t.Fatalf("hydrated apis: %+v", got)
	}
	if got := repo2.GetRolePolicies(roleID); len(got) != 1 {
		t.Fatalf("hydrated policies: %+v", got)
	}
	scope := repo2.GetRoleDataScope(roleID)
	if !scope.RequireOwnerMatch || len(scope.TenantIDs) != 1 {
		t.Fatalf("hydrated scope: %+v", scope)
	}
	route := repo2.GetRoleRoutePermissions(roleID)
	if route.Version != "v1" || len(route.Items) != 1 {
		t.Fatalf("hydrated route: %+v", route)
	}
	snaps := repo2.ListRolePolicySnapshots(roleID)
	if len(snaps) != 1 || snaps[0].Version != first.Version {
		t.Fatalf("hydrated snapshots: %+v", snaps)
	}

	// New snapshot must continue the version sequence.
	next := repo2.CreateRolePolicySnapshot(roleID)
	if next.Version == first.Version {
		t.Fatalf("expected new version after hydrate, got duplicate %s", next.Version)
	}
}
