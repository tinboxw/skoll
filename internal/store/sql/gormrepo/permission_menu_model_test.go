package gormrepo

import (
	"testing"

	"github.com/tinboxw/skoll/internal/domain/menu"
	"github.com/tinboxw/skoll/internal/domain/permission"
)

func TestPermissionResourceModelRoundTrip(t *testing.T) {
	resource, err := permission.NewResourceWithMetadata(permission.ResourceIdentity{
		Key:    "system:user:list",
		Type:   permission.ResourceTypeAPI,
		Module: "system",
		Source: "system",
	}, "List users", permission.RiskLevelMedium, map[string]string{"owner": "iam"})
	if err != nil {
		t.Fatalf("NewResourceWithMetadata() error = %v", err)
	}

	model := PermissionResourceModelFromDomain(resource)
	if model.TableName() != "sk_permission_resources" {
		t.Fatalf("table = %q", model.TableName())
	}

	got, err := model.ToDomain()
	if err != nil {
		t.Fatalf("ToDomain() error = %v", err)
	}
	if got.Key() != resource.Key() || got.Type() != resource.Type() || got.Module() != resource.Module() || got.Source() != resource.Source() {
		t.Fatalf("identity mismatch: %+v", got.Identity)
	}
	if got.Name != "List users" || got.Risk != permission.RiskLevelMedium || got.Metadata["owner"] != "iam" || !got.Enabled {
		t.Fatalf("resource mismatch: %+v", got)
	}
}

func TestMenuNodeModelRoundTrip(t *testing.T) {
	node, err := menu.NewNode(menu.NodeIdentity{
		Key:       "system.users",
		ParentKey: "system",
		Source:    "system",
	}, menu.NodeView{
		Name:      "Users",
		Path:      "/system/users",
		Component: "UserList",
		Icon:      "Users",
	}, 20)
	if err != nil {
		t.Fatalf("NewNode() error = %v", err)
	}
	node.RequiredRoles = []string{"admin"}
	node.RequiredPermissions = []string{"system:user:list"}

	model := MenuNodeModelFromDomain(node)
	if model.TableName() != "sk_menu_nodes" {
		t.Fatalf("table = %q", model.TableName())
	}

	got, err := model.ToDomain()
	if err != nil {
		t.Fatalf("ToDomain() error = %v", err)
	}
	if got.Key() != node.Key() || got.ParentKey() != node.ParentKey() || got.Source() != node.Source() {
		t.Fatalf("identity mismatch: %+v", got.Identity)
	}
	if got.Path() != node.Path() || got.Component() != node.Component() || got.Sort != node.Sort || !got.Visible {
		t.Fatalf("node mismatch: %+v", got)
	}
	if len(got.RequiredRoles) != 1 || got.RequiredRoles[0] != "admin" {
		t.Fatalf("roles = %+v", got.RequiredRoles)
	}
	if len(got.RequiredPermissions) != 1 || got.RequiredPermissions[0] != "system:user:list" {
		t.Fatalf("permissions = %+v", got.RequiredPermissions)
	}
}
