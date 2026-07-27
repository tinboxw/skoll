package bootstrap

import (
	"context"
	"testing"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestEnsureSystemPermissionCatalogSeedsFrameworkKeys(t *testing.T) {
	service := permissionsvc.NewService(memory.NewPermissionStore())

	ensureSystemPermissionCatalog(context.Background(), nil, service)
	ensureSystemPermissionCatalog(context.Background(), nil, service)

	items, err := service.ListResources(context.Background(), permissionsvc.ListResourcesInput{Source: "system"})
	if err != nil {
		t.Fatalf("ListResources error: %v", err)
	}

	got := map[string]domainpermission.PermissionResource{}
	for _, item := range items {
		got[item.Key()] = item
	}
	for _, key := range []string{
		"plugin.install", "plugin.enable", "plugin.disable", "plugin.uninstall",
		"permission.manage", "menu.read", "menu.manage",
	} {
		item, ok := got[key]
		if !ok {
			t.Fatalf("expected seeded permission %q, got keys=%v", key, got)
		}
		if !item.Enabled {
			t.Fatalf("expected %q to be enabled", key)
		}
	}
	if got["permission.manage"].Risk != domainpermission.RiskLevelHigh {
		t.Fatalf("permission.manage risk = %q", got["permission.manage"].Risk)
	}
	for _, key := range []string{"plugin.install", "plugin.enable", "plugin.disable", "plugin.uninstall"} {
		if got[key].Risk != domainpermission.RiskLevelHigh || got[key].Type() != domainpermission.ResourceTypePlugin {
			t.Fatalf("%s is not a high-risk plugin permission: %+v", key, got[key])
		}
	}
}
