package storageadapter

import (
	"reflect"
	"testing"

	"github.com/tinboxw/skoll/internal/module/audit"
)

func TestInMemoryAdapterContract(t *testing.T) {
	runAdapterContract(t, func() Adapter { return NewInMemoryAdapter() })
}

func runAdapterContract(t *testing.T, factory func() Adapter) {
	t.Helper()

	a := factory()

	u1 := a.Users().Create("alice", "alice@example.com")
	u2 := a.Users().Create("bob", "bob@example.com")
	if u1.ID <= 0 || u2.ID <= 0 || u2.ID <= u1.ID {
		t.Fatalf("unexpected user IDs: u1=%d u2=%d", u1.ID, u2.ID)
	}
	users := a.Users().List()
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	r1 := a.Roles().Create("admin", []string{"user.read", "user.write"})
	if _, err := a.Roles().Get(r1.ID); err != nil {
		t.Fatalf("role get failed: %v", err)
	}

	m1 := a.Menus().Create("Dashboard", "/dashboard", 20)
	m2 := a.Menus().Create("System", "/system", 10)
	menus := a.Menus().List()
	if len(menus) != 2 || menus[0].ID != m2.ID || menus[1].ID != m1.ID {
		t.Fatalf("unexpected menu ordering: %#v", menus)
	}

	boundMenus := a.RBAC().SetRoleMenus(r1.ID, []int64{m1.ID, m2.ID, m1.ID})
	if !reflect.DeepEqual(boundMenus, []int64{m1.ID, m2.ID}) && !reflect.DeepEqual(boundMenus, []int64{m2.ID, m1.ID}) {
		// The service guarantees sorted output; this assertion keeps contract resilient
		// if future adapters return deterministic but different ordering policies.
		t.Fatalf("unexpected role menus: %v", boundMenus)
	}
	roleMenus := a.RBAC().GetRoleMenus(r1.ID)
	if len(roleMenus) != 2 {
		t.Fatalf("expected 2 role menus, got %d", len(roleMenus))
	}

	a.APIs().RegisterMany([]string{"GET /admin/v1/users", "POST:/admin/v1/users"})
	if !a.APIs().Exists("GET:/admin/v1/users") {
		t.Fatalf("expected registered api to exist")
	}

	apis := a.RBAC().SetRoleAPIs(r1.ID, []string{"POST:/admin/v1/users", "GET:/admin/v1/users", "GET:/admin/v1/users"})
	if len(apis) != 2 {
		t.Fatalf("expected 2 role apis after dedupe, got %d", len(apis))
	}

	a.Audit().Append("system", "bind-role-api", "role:1")
	a.Audit().Append("system", "bind-role-menu", "role:1")
	recent := a.Audit().Recent(1)
	if len(recent) != 1 {
		t.Fatalf("expected 1 recent audit item, got %d", len(recent))
	}
	query := a.Audit().Query(audit.Query{Page: 1, Size: 10, Action: "bind-role-api"})
	if query.Total != 1 || len(query.Items) != 1 {
		t.Fatalf("unexpected audit query result: %+v", query)
	}

	a.Configs().Set("system.theme", "aurora", "ui theme")
	cfg, err := a.Configs().Get("system.theme")
	if err != nil {
		t.Fatalf("config get failed: %v", err)
	}
	if cfg.Value != "aurora" {
		t.Fatalf("unexpected config value: %s", cfg.Value)
	}

	a.Dictionaries().Create("status", "Enabled", "1", 10, true)
	a.Dictionaries().Create("status", "Disabled", "0", 20, true)
	statusItems := a.Dictionaries().ListByType("status")
	if len(statusItems) != 2 {
		t.Fatalf("expected 2 dictionary items, got %d", len(statusItems))
	}

	uploaded, err := a.Files().Upload("contract.txt", []byte("contract"))
	if err != nil {
		t.Fatalf("file upload failed: %v", err)
	}
	if uploaded.ID <= 0 {
		t.Fatalf("expected generated file id")
	}
	_, content, err := a.Files().Download(uploaded.ID)
	if err != nil {
		t.Fatalf("file download failed: %v", err)
	}
	if string(content) != "contract" {
		t.Fatalf("unexpected downloaded content: %s", string(content))
	}

	job := a.Jobs().Create("daily-sync", "0 0 * * *")
	if job.ID <= 0 {
		t.Fatalf("expected generated job id")
	}
	run, err := a.Jobs().Run(job.ID)
	if err != nil {
		t.Fatalf("run job failed: %v", err)
	}
	if run.Status != "success" {
		t.Fatalf("unexpected run status: %s", run.Status)
	}
	history := a.Jobs().History(job.ID, 10)
	if len(history) != 1 {
		t.Fatalf("expected 1 job history item, got %d", len(history))
	}

	generated, err := a.Generators().Generate("contractmodule")
	if err != nil {
		t.Fatalf("generate module failed: %v", err)
	}
	if generated.Module != "contractmodule" || len(generated.Artifacts) == 0 {
		t.Fatalf("unexpected generator output: %+v", generated)
	}

	plugin := a.Plugins().Install("contract-plugin", "1.0.0", []string{"on_boot"})
	if !plugin.Enabled {
		t.Fatalf("expected plugin enabled on install")
	}
	disabled, err := a.Plugins().Disable("contract-plugin")
	if err != nil {
		t.Fatalf("disable plugin failed: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("expected plugin disabled")
	}
}
