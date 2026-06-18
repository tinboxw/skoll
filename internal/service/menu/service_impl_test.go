package menu

import (
	"context"
	"testing"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	menurepo "github.com/tinboxw/skoll/internal/repository/menu"
	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestMergeNodesStableSortsAndUpserts(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMenuStore()
	svc := NewService(store)

	nodes, err := svc.MergeNodes(ctx, MergeNodesInput{Nodes: []domainmenu.MenuNode{
		mustServiceNode(t, "plugin.reports", "", "plugin.demo", "/reports", "Reports", 20),
		mustServiceNode(t, "system", "", "system", "/system", "System", 10),
		mustServiceNode(t, "generated.audit", "", "generated.audit", "/audit", "Audit", 30),
		mustServiceNode(t, "plugin.reports", "", "plugin.demo", "/reports", "Reports updated", 20),
		mustServiceNode(t, "system.users", "system", "system", "/system/users", "Users", 10),
	}})
	if err != nil {
		t.Fatalf("MergeNodes() error = %v", err)
	}

	assertServiceMenuKeys(t, nodes, []string{"system", "system.users", "plugin.reports", "generated.audit"})
	if nodes[2].Name() != "Reports updated" {
		t.Fatalf("duplicate key should update node, got %q", nodes[2].Name())
	}

	tree, err := store.Tree(ctx, menurepoFilter())
	if err != nil {
		t.Fatalf("store.Tree() error = %v", err)
	}
	assertServiceMenuKeys(t, tree, []string{"system", "system.users", "plugin.reports", "generated.audit"})
}

func mustServiceNode(t *testing.T, key string, parent string, source string, path string, name string, sort int) domainmenu.MenuNode {
	t.Helper()
	node, err := domainmenu.NewNode(domainmenu.NodeIdentity{
		Key:       key,
		ParentKey: parent,
		Source:    source,
	}, domainmenu.NodeView{
		Name: name,
		Path: path,
	}, sort)
	if err != nil {
		t.Fatalf("NewNode(%q) error = %v", key, err)
	}
	return node
}

func assertServiceMenuKeys(t *testing.T, nodes []domainmenu.MenuNode, want []string) {
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

func menurepoFilter() menurepo.ListFilter {
	return menurepo.ListFilter{}
}
