package memory

import (
	"context"
	"testing"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
)

func TestPermissionStoreRegisterIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := NewPermissionStore()

	first := mustPermission(t, "system:user:list", "List users", "system")
	second := mustPermission(t, "system:user:list", "List users updated", "system")

	if err := store.Register(ctx, first); err != nil {
		t.Fatalf("Register(first) error = %v", err)
	}
	if err := store.Register(ctx, second); err != nil {
		t.Fatalf("Register(second) error = %v", err)
	}

	items, err := store.List(ctx, permissionrepo.ListFilter{}, 0, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].Name != "List users updated" {
		t.Fatalf("Name = %q", items[0].Name)
	}
}

func TestPermissionStoreListFiltersBySource(t *testing.T) {
	ctx := context.Background()
	store := NewPermissionStore()

	_ = store.Register(ctx, mustPermission(t, "system:user:list", "List users", "system"))
	_ = store.Register(ctx, mustPermission(t, "plugin.demo:report:list", "List reports", "plugin.demo"))

	items, err := store.List(ctx, permissionrepo.ListFilter{Source: "PLUGIN.DEMO"}, 0, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].Source() != "plugin.demo" {
		t.Fatalf("items = %+v", items)
	}
}

func TestPermissionStoreSetEnabled(t *testing.T) {
	ctx := context.Background()
	store := NewPermissionStore()
	resource := mustPermission(t, "system:user:list", "List users", "system")

	_ = store.Register(ctx, resource)
	if err := store.SetEnabled(ctx, resource.Key(), false); err != nil {
		t.Fatalf("SetEnabled(false) error = %v", err)
	}

	disabled := false
	items, err := store.List(ctx, permissionrepo.ListFilter{Enabled: &disabled}, 0, 0)
	if err != nil {
		t.Fatalf("List(disabled) error = %v", err)
	}
	if len(items) != 1 || items[0].Enabled {
		t.Fatalf("items = %+v", items)
	}

	got, err := store.Get(ctx, resource.Key())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == nil || got.Enabled {
		t.Fatalf("got = %+v", got)
	}
}

func mustPermission(t *testing.T, key string, name string, source string) domainpermission.PermissionResource {
	t.Helper()
	resource, err := domainpermission.NewResource(domainpermission.ResourceIdentity{
		Key:    key,
		Type:   domainpermission.ResourceTypeAPI,
		Module: "system",
		Source: source,
	}, name)
	if err != nil {
		t.Fatalf("NewResource() error = %v", err)
	}
	return resource
}
