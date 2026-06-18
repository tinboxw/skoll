package memory

import (
	"context"
	"testing"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	menurepo "github.com/tinboxw/skoll/internal/repository/menu"
)

func TestMenuStoreUpsertAndTree(t *testing.T) {
	ctx := context.Background()
	store := NewMenuStore()

	_ = store.Upsert(ctx, mustMenuNode(t, "system", "", "/system", 20))
	_ = store.Upsert(ctx, mustMenuNode(t, "dashboard", "", "/dashboard", 10))
	_ = store.Upsert(ctx, mustMenuNode(t, "system.users", "system", "/system/users", 20))
	_ = store.Upsert(ctx, mustMenuNode(t, "system.roles", "system", "/system/roles", 10))

	nodes, err := store.Tree(ctx, menurepo.ListFilter{})
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}

	assertMenuKeys(t, nodes, []string{"dashboard", "system", "system.roles", "system.users"})
}

func TestMenuStoreReorder(t *testing.T) {
	ctx := context.Background()
	store := NewMenuStore()

	_ = store.Upsert(ctx, mustMenuNode(t, "system.users", "system", "/system/users", 10))
	_ = store.Upsert(ctx, mustMenuNode(t, "system.roles", "system", "/system/roles", 20))

	if err := store.Reorder(ctx, "system", []string{"system.roles", "system.users"}); err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}

	nodes, err := store.List(ctx, menurepo.ListFilter{ParentKey: "system"}, 0, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	assertMenuKeys(t, nodes, []string{"system.roles", "system.users"})
	if nodes[0].Sort != 0 || nodes[1].Sort != 10 {
		t.Fatalf("sorts = %d, %d", nodes[0].Sort, nodes[1].Sort)
	}
}

func TestMenuStoreVisibilityFilter(t *testing.T) {
	ctx := context.Background()
	store := NewMenuStore()

	visible := mustMenuNode(t, "system.users", "system", "/system/users", 10)
	hidden := mustMenuNode(t, "system.audit", "system", "/system/audit", 20)
	hidden.Visible = false
	_ = store.Upsert(ctx, visible)
	_ = store.Upsert(ctx, hidden)

	wantVisible := true
	nodes, err := store.List(ctx, menurepo.ListFilter{Visible: &wantVisible}, 0, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	assertMenuKeys(t, nodes, []string{"system.users"})
}

func mustMenuNode(t *testing.T, key string, parent string, path string, sort int) domainmenu.MenuNode {
	t.Helper()
	node, err := domainmenu.NewNode(domainmenu.NodeIdentity{
		Key:       key,
		ParentKey: parent,
		Source:    "system",
	}, domainmenu.NodeView{
		Name: key,
		Path: path,
	}, sort)
	if err != nil {
		t.Fatalf("NewNode(%q) error = %v", key, err)
	}
	return node
}

func assertMenuKeys(t *testing.T, nodes []domainmenu.MenuNode, want []string) {
	t.Helper()
	if len(nodes) != len(want) {
		t.Fatalf("len = %d, want %d: %+v", len(nodes), len(want), nodes)
	}
	for i, node := range nodes {
		if node.Key() != want[i] {
			t.Fatalf("nodes[%d].Key() = %q, want %q", i, node.Key(), want[i])
		}
	}
}
