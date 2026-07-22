package equipmentmaintenance

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type proofManifest struct {
	ID               string `yaml:"id"`
	APIVersion       string `yaml:"api_version"`
	UIMode           string `yaml:"ui_mode"`
	Level            string `yaml:"level"`
	AppID            string `yaml:"app_id"`
	FrontendEntry    string `yaml:"frontend_entry"`
	ServiceBaseURL   string `yaml:"service_base_url"`
	ServiceHealthURL string `yaml:"service_health_url"`
	UIMenu           struct {
		RequiredPermissions []string `yaml:"required_permissions"`
	} `yaml:"ui_menu"`
	Permissions []struct {
		Key string `yaml:"key"`
	} `yaml:"permissions"`
	Data struct {
		Namespace string `yaml:"namespace"`
		Tables    []struct {
			Name    string `yaml:"name"`
			Columns string `yaml:"columns"`
		} `yaml:"tables"`
	} `yaml:"data"`
	API struct {
		Routes []struct {
			Method      string `yaml:"method"`
			Path        string `yaml:"path"`
			Permission  string `yaml:"permission"`
			AuditAction string `yaml:"audit_action"`
		} `yaml:"routes"`
	} `yaml:"api"`
	Events struct {
		Subscriptions []struct {
			Name    string `yaml:"name"`
			Handler string `yaml:"handler"`
		} `yaml:"subscriptions"`
	} `yaml:"events"`
}

type acceptanceMap struct {
	SchemaVersion  int    `json:"schemaVersion"`
	PluginID       string `json:"pluginId"`
	PublicContract struct {
		APIVersion    string   `json:"apiVersion"`
		APIPrefix     string   `json:"apiPrefix"`
		BackendClient string   `json:"backendClient"`
		HostServices  []string `json:"hostServices"`
	} `json:"publicContract"`
	Domains []struct {
		ID          string   `json:"id"`
		Tables      []string `json:"tables"`
		Permissions []string `json:"permissions"`
		Routes      []string `json:"routes"`
		Acceptance  string   `json:"acceptance"`
	} `json:"domains"`
	Workflows []struct {
		HostService     string `json:"hostService"`
		TriggerRoute    string `json:"triggerRoute"`
		CompletionEvent string `json:"completionEvent"`
	} `json:"workflows"`
	Events []struct {
		Name    string `json:"name"`
		Handler string `json:"handler"`
	} `json:"events"`
	Jobs []struct {
		HostService  string `json:"hostService"`
		TriggerRoute string `json:"triggerRoute"`
	} `json:"jobs"`
	AcceptanceScenarios []struct {
		ID     string `json:"id"`
		Domain string `json:"domain"`
		Result string `json:"result"`
	} `json:"acceptanceScenarios"`
}

func TestProofPluginContractIsCompleteAndPublic(t *testing.T) {
	manifest := loadManifest(t)
	contract := loadAcceptanceMap(t)

	if manifest.ID != "equipment_maintenance" || manifest.AppID != manifest.ID || contract.PluginID != manifest.ID {
		t.Fatalf("plugin identity mismatch: manifest=%q app=%q contract=%q", manifest.ID, manifest.AppID, contract.PluginID)
	}
	if manifest.APIVersion != "v1" || contract.SchemaVersion != 1 || contract.PublicContract.APIVersion != manifest.APIVersion {
		t.Fatalf("current contract version mismatch: manifest=%q map=%q schema=%d", manifest.APIVersion, contract.PublicContract.APIVersion, contract.SchemaVersion)
	}
	if manifest.UIMode != "separated" || manifest.Level != "app" || manifest.FrontendEntry == "" || manifest.ServiceBaseURL == "" || manifest.ServiceHealthURL == "" {
		t.Fatalf("proof plugin must declare a complete separated runtime: %+v", manifest)
	}
	if contract.PublicContract.BackendClient != "github.com/tinboxw/skoll/pkg/pluginclient" {
		t.Fatalf("unexpected public backend client: %q", contract.PublicContract.BackendClient)
	}

	permissions := make(map[string]struct{}, len(manifest.Permissions))
	for _, permission := range manifest.Permissions {
		if permission.Key == "" {
			t.Fatal("manifest contains an empty permission")
		}
		if _, exists := permissions[permission.Key]; exists {
			t.Fatalf("duplicate manifest permission: %s", permission.Key)
		}
		permissions[permission.Key] = struct{}{}
	}
	for _, permission := range manifest.UIMenu.RequiredPermissions {
		assertContains(t, permissions, permission, "menu permission")
	}

	routes := make(map[string]struct{}, len(manifest.API.Routes))
	for _, route := range manifest.API.Routes {
		key := strings.ToUpper(route.Method) + " " + route.Path
		if !strings.HasPrefix(route.Path, contract.PublicContract.APIPrefix) {
			t.Fatalf("route escapes plugin API namespace: %s", key)
		}
		if _, exists := routes[key]; exists {
			t.Fatalf("duplicate manifest route: %s", key)
		}
		routes[key] = struct{}{}
		assertContains(t, permissions, route.Permission, "route permission")
		if len(strings.Split(route.AuditAction, ".")) != 3 {
			t.Fatalf("route %s has invalid audit action %q", key, route.AuditAction)
		}
	}

	tables := make(map[string]struct{}, len(manifest.Data.Tables))
	for _, table := range manifest.Data.Tables {
		tables[table.Name] = struct{}{}
		columns := csvSet(table.Columns)
		for _, required := range []string{"tenant_id", "organization_id", "owner_id"} {
			assertContains(t, columns, required, "scope column in "+table.Name)
		}
	}

	wantDomains := []string{"assets", "dashboards", "inspections", "spare_parts", "work_orders"}
	gotDomains := make([]string, 0, len(contract.Domains))
	for _, domain := range contract.Domains {
		gotDomains = append(gotDomains, domain.ID)
		if strings.TrimSpace(domain.Acceptance) == "" || len(domain.Routes) == 0 || len(domain.Permissions) == 0 {
			t.Fatalf("domain acceptance is incomplete: %+v", domain)
		}
		for _, table := range domain.Tables {
			assertContains(t, tables, table, "domain table")
		}
		for _, permission := range domain.Permissions {
			assertContains(t, permissions, permission, "domain permission")
		}
		for _, route := range domain.Routes {
			assertContains(t, routes, route, "domain route")
		}
	}
	sort.Strings(gotDomains)
	if strings.Join(gotDomains, ",") != strings.Join(wantDomains, ",") {
		t.Fatalf("domain boundary = %v, want %v", gotDomains, wantDomains)
	}

	wantServices := []string{"Audit", "Config", "DataScopes", "Files", "Jobs", "Secrets", "Transactions", "Workflows"}
	gotServices := append([]string(nil), contract.PublicContract.HostServices...)
	sort.Strings(gotServices)
	if strings.Join(gotServices, ",") != strings.Join(wantServices, ",") {
		t.Fatalf("host services = %v, want %v", gotServices, wantServices)
	}

	events := make(map[string]struct{}, len(manifest.Events.Subscriptions))
	for _, event := range manifest.Events.Subscriptions {
		events[event.Name+"::"+event.Handler] = struct{}{}
	}
	for _, event := range contract.Events {
		assertContains(t, events, event.Name+"::"+event.Handler, "event subscription")
	}
	for _, workflow := range contract.Workflows {
		if workflow.HostService != "Workflows" {
			t.Fatalf("workflow uses non-public service: %+v", workflow)
		}
		assertContains(t, routes, workflow.TriggerRoute, "workflow trigger route")
		if !hasEventName(manifest, workflow.CompletionEvent) {
			t.Fatalf("workflow completion event is not subscribed: %s", workflow.CompletionEvent)
		}
	}
	for _, job := range contract.Jobs {
		if job.HostService != "Jobs" {
			t.Fatalf("job uses non-public service: %+v", job)
		}
		assertContains(t, routes, job.TriggerRoute, "job trigger route")
	}
	if len(contract.AcceptanceScenarios) < 6 {
		t.Fatalf("acceptance scenario count = %d, want at least 6", len(contract.AcceptanceScenarios))
	}
}

func loadManifest(t *testing.T) proofManifest {
	t.Helper()
	raw, err := os.ReadFile("plugin.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var manifest proofManifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode plugin manifest: %v", err)
	}
	return manifest
}

func loadAcceptanceMap(t *testing.T) acceptanceMap {
	t.Helper()
	raw, err := os.ReadFile("contract/acceptance-map.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract acceptanceMap
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("decode acceptance map: %v", err)
	}
	return contract
}

func csvSet(raw string) map[string]struct{} {
	items := make(map[string]struct{})
	for _, item := range strings.Split(raw, ",") {
		items[strings.TrimSpace(item)] = struct{}{}
	}
	return items
}

func assertContains(t *testing.T, values map[string]struct{}, value string, label string) {
	t.Helper()
	if _, ok := values[value]; !ok {
		t.Fatalf("%s %q is not declared", label, value)
	}
}

func hasEventName(manifest proofManifest, name string) bool {
	for _, event := range manifest.Events.Subscriptions {
		if event.Name == name {
			return true
		}
	}
	return false
}
