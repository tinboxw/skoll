package plugin

import "testing"

func TestInfoMenuNodesDefaultsFromPluginManifest(t *testing.T) {
	info := Info{
		ID:            "report-plugin",
		Name:          "Report Plugin",
		Version:       "1.0.0",
		FrontendEntry: "/skoll/plugins/report-plugin",
		UIMenu: &UIMenu{
			Label:               "Reports",
			Icon:                "chart",
			Order:               90,
			RequiredRoles:       []string{" manager "},
			RequiredPermissions: []string{" report.read "},
		},
	}

	nodes, err := info.MenuNodes()
	if err != nil {
		t.Fatalf("MenuNodes error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("len = %d", len(nodes))
	}
	node := nodes[0]
	if node.Key() != "plugin.report-plugin" || node.Source() != "plugin.report-plugin" {
		t.Fatalf("unexpected identity: %+v", node.Identity)
	}
	if node.Name() != "Reports" || node.Path() != "/skoll/plugins/report-plugin" || node.Icon() != "chart" {
		t.Fatalf("unexpected view: %+v", node.View)
	}
	if len(node.RequiredRoles) != 1 || node.RequiredRoles[0] != "manager" {
		t.Fatalf("unexpected roles: %+v", node.RequiredRoles)
	}
	if len(node.RequiredPermissions) != 1 || node.RequiredPermissions[0] != "report.read" {
		t.Fatalf("unexpected permissions: %+v", node.RequiredPermissions)
	}
}

func TestInfoMenuNodesRejectsInvalidMenu(t *testing.T) {
	info := Info{
		ID:      "report-plugin",
		Name:    "Report Plugin",
		Version: "1.0.0",
		UIMenu: &UIMenu{
			Key:   "bad key",
			Label: "Reports",
			Path:  "/reports",
		},
	}
	if _, err := info.MenuNodes(); err == nil {
		t.Fatal("expected invalid menu node error")
	}
}
