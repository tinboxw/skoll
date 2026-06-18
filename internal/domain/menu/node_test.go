package menu

import "testing"

func TestNewNodeDefaultsVisible(t *testing.T) {
	node, err := NewNode(NodeIdentity{
		Key:       " System.User ",
		ParentKey: " System ",
		Source:    " System ",
	}, NodeView{
		Name:      " Users ",
		Path:      " /system/users ",
		Component: " UserList ",
		Icon:      " Users ",
	}, 20)
	if err != nil {
		t.Fatalf("NewNode() error = %v", err)
	}

	if !node.Visible {
		t.Fatal("new menu node should be visible by default")
	}
	if node.Key() != "system.user" {
		t.Fatalf("key = %q", node.Key())
	}
	if node.ParentKey() != "system" {
		t.Fatalf("parent = %q", node.ParentKey())
	}
	if node.Source() != "system" {
		t.Fatalf("source = %q", node.Source())
	}
	if node.Path() != "/system/users" {
		t.Fatalf("path = %q", node.Path())
	}
	if node.Component() != "UserList" {
		t.Fatalf("component = %q", node.Component())
	}
	if node.Icon() != "Users" {
		t.Fatalf("icon = %q", node.Icon())
	}
	if node.Sort != 20 {
		t.Fatalf("sort = %d", node.Sort)
	}
}
