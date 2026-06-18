package menu

import "testing"

func TestValidateNodeAcceptsValidValues(t *testing.T) {
	node, err := NewNode(NodeIdentity{
		Key:    "system.dashboard",
		Source: "system",
	}, NodeView{
		Name: "Dashboard",
		Path: "/dashboard",
	}, 0)
	if err != nil {
		t.Fatalf("NewNode() error = %v", err)
	}
	if node.Key() != "system.dashboard" {
		t.Fatalf("key = %q", node.Key())
	}
}

func TestValidateNodeRejectsInvalidPath(t *testing.T) {
	_, err := NewNode(validIdentity(), NodeView{
		Name: "Users",
		Path: "system/users",
	}, 1)
	if err == nil {
		t.Fatal("expected invalid path error")
	}
}

func TestValidateNodeRejectsBlankName(t *testing.T) {
	_, err := NewNode(validIdentity(), NodeView{
		Name: " ",
		Path: "/system/users",
	}, 1)
	if err == nil {
		t.Fatal("expected blank name error")
	}
}

func TestValidateNodeRejectsNegativeSort(t *testing.T) {
	_, err := NewNode(validIdentity(), NodeView{
		Name: "Users",
		Path: "/system/users",
	}, -1)
	if err == nil {
		t.Fatal("expected negative sort error")
	}
}

func TestValidateNodeRejectsInvalidSource(t *testing.T) {
	identity := validIdentity()
	identity.Source = "System Menu"
	_, err := NewNode(identity, NodeView{
		Name: "Users",
		Path: "/system/users",
	}, 1)
	if err == nil {
		t.Fatal("expected invalid source error")
	}
}

func validIdentity() NodeIdentity {
	return NodeIdentity{
		Key:    "system.users",
		Source: "system",
	}
}
