package menu

import "testing"

func TestSortSiblingsIsStable(t *testing.T) {
	nodes := []MenuNode{
		mustNode(t, "reports.daily", "/reports/daily", 20),
		mustNode(t, "system.users", "/system/users", 10),
		mustNode(t, "reports.monthly", "/reports/monthly", 20),
	}

	sorted := SortSiblings(nodes)

	assertKeys(t, sorted, []string{"system.users", "reports.daily", "reports.monthly"})
	assertKeys(t, nodes, []string{"reports.daily", "system.users", "reports.monthly"})
}

func TestFilterVisibleRemovesHiddenNodes(t *testing.T) {
	visible := mustNode(t, "system.users", "/system/users", 10)
	hidden := mustNode(t, "system.roles", "/system/roles", 20)
	hidden.Visible = false

	filtered := FilterVisible([]MenuNode{hidden, visible})

	assertKeys(t, filtered, []string{"system.users"})
}

func TestFilterAuthorizedRequiresRolesAndPermissions(t *testing.T) {
	allowed := mustNode(t, "system.users", "/system/users", 10)
	allowed.RequiredRoles = []string{"Admin"}
	allowed.RequiredPermissions = []string{"system:user:list"}
	missingRole := mustNode(t, "system.roles", "/system/roles", 20)
	missingRole.RequiredRoles = []string{"owner"}
	missingPermission := mustNode(t, "system.audit", "/system/audit", 30)
	missingPermission.RequiredPermissions = []string{"system:audit:list"}
	public := mustNode(t, "dashboard.home", "/dashboard", 0)

	filtered := FilterAuthorized([]MenuNode{
		allowed,
		missingRole,
		missingPermission,
		public,
	}, NewAccessContext([]string{"admin"}, []string{"system:user:list"}))

	assertKeys(t, filtered, []string{"system.users", "dashboard.home"})
}

func mustNode(t *testing.T, key string, path string, sort int) MenuNode {
	t.Helper()
	node, err := NewNode(NodeIdentity{
		Key:    key,
		Source: "system",
	}, NodeView{
		Name: key,
		Path: path,
	}, sort)
	if err != nil {
		t.Fatalf("NewNode(%q) error = %v", key, err)
	}
	return node
}

func assertKeys(t *testing.T, nodes []MenuNode, want []string) {
	t.Helper()
	if len(nodes) != len(want) {
		t.Fatalf("len = %d, want %d", len(nodes), len(want))
	}
	for i, node := range nodes {
		if node.Key() != want[i] {
			t.Fatalf("nodes[%d].Key() = %q, want %q", i, node.Key(), want[i])
		}
	}
}
