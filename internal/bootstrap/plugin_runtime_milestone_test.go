package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/event"
	httpHandler "github.com/tinboxw/skoll/internal/handler/http"
	"github.com/tinboxw/skoll/internal/plugin"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/logging"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestPluginRuntimeMilestoneEndToEnd(t *testing.T) {
	const (
		pluginID = "milestone_plugin"
		route    = "/v1/plugins/milestone_plugin/api/items/item-1"
		prefix   = "/skoll"
		secret   = "plugin-runtime-milestone-secret"
	)

	var businessCalls atomic.Int32
	var eventCalls atomic.Int32
	deliveries := make(chan plugin.EventDeliveryEnvelope, 2)
	pluginServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		case r.Method == http.MethodGet && r.URL.Path == route:
			businessCalls.Add(1)
			if r.URL.Query().Get("expand") != "owner" {
				http.Error(w, "query was not preserved", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"source":"plugin","id":"item-1"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/_skoll/events":
			var envelope plugin.EventDeliveryEnvelope
			if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
				http.Error(w, "invalid event", http.StatusBadRequest)
				return
			}
			if r.Header.Get("Idempotency-Key") != envelope.DeliveryID || r.Header.Get("X-Skoll-Plugin-ID") != pluginID {
				http.Error(w, "invalid event headers", http.StatusBadRequest)
				return
			}
			eventCalls.Add(1)
			deliveries <- envelope
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(pluginServer.Close)

	db := gormrepo.TestDB(t)
	pluginDir := writeRuntimeMilestonePlugin(t, pluginID, pluginServer.URL)
	healthChecker := plugin.NewHTTPHealthChecker(time.Second)
	supervisor := plugin.NewServiceSupervisor(
		&readyTestServiceLauncher{},
		nil,
		time.Second,
		time.Second,
	)
	dataDirectories, err := plugin.NewPluginDataDirectories(filepath.Join(t.TempDir(), "plugin-data"))
	if err != nil {
		t.Fatal(err)
	}
	manager := &pluginManagerWithExtensions{
		Manager:            plugin.NewRuntimeManager(plugin.NewFileLoader(), plugin.NewTopologicalResolver()),
		builtinInfos:       map[string]plugin.Info{},
		extensions:         map[string]plugin.RegistrySnapshot{},
		routeHandlers:      map[string]http.HandlerFunc{},
		routePermissions:   mustEmptyRoutePermissionRegistry(),
		healthChecker:      healthChecker,
		healthCache:        make(map[string]plugin.HealthReport),
		healthTTL:          time.Second,
		serviceSupervisor:  supervisor,
		migrationHook:      plugin.NewPluginMigrationHook(gormrepo.NewPluginMigrationStore(db), nil),
		dataDirectories:    dataDirectories,
		businessEvents:     event.NewBusinessEventBus(nil),
		eventDelivery:      plugin.NewHTTPEventDeliveryClient(nil, time.Second),
		eventSubscriptions: map[string][]func(){},
	}

	installed, err := manager.Install(pluginDir)
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	if installed.State != plugin.StateInstalled || db.Migrator().HasTable("milestone_plugin_items") {
		t.Fatalf("unexpected install state=%s table=%v", installed.State, db.Migrator().HasTable("milestone_plugin_items"))
	}

	auditEvents := memory.NewAuditEventStore()
	auditService := auditsvc.NewEventService(auditEvents)
	router := httpHandler.NewRouter(httpHandler.Dependencies{
		PluginManager:     manager,
		AuditEventService: auditService,
		APIPrefix:         prefix,
	})
	checker := &fakePermissionChecker{allowed: map[string]bool{
		"user:alice:milestone_plugin.items:read": true,
	}}
	handler := buildMiddlewareChain(
		router,
		logging.Discard(),
		AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{}},
		prefix,
		secret,
		checker,
		manager,
		auditService,
	)

	rootToken := signMilestoneToken(t, secret, "root", "super_admin")
	aliceToken := signMilestoneToken(t, secret, "alice", "operator")
	bobToken := signMilestoneToken(t, secret, "bob", "viewer")

	response := milestoneRequest(handler, http.MethodPost, prefix+"/v1/plugins/"+pluginID+"/enable", rootToken)
	if response.Code != http.StatusOK {
		t.Fatalf("enable plugin status=%d body=%s", response.Code, response.Body.String())
	}
	if !db.Migrator().HasTable("milestone_plugin_items") {
		t.Fatal("enable did not apply plugin migration")
	}
	serviceState, ok := supervisor.Snapshot(pluginID)
	if !ok || serviceState.State != plugin.ServiceStateReady {
		t.Fatalf("service did not reach ready: state=%+v exists=%v", serviceState, ok)
	}

	unauthenticated := milestoneRequest(handler, http.MethodGet, prefix+route+"?expand=owner", "")
	if unauthenticated.Code != http.StatusUnauthorized || businessCalls.Load() != 0 {
		t.Fatalf("unauthenticated status=%d calls=%d body=%s", unauthenticated.Code, businessCalls.Load(), unauthenticated.Body.String())
	}
	invalidToken := milestoneRequest(handler, http.MethodGet, prefix+route+"?expand=owner", "invalid-token")
	if invalidToken.Code != http.StatusUnauthorized || businessCalls.Load() != 0 {
		t.Fatalf("invalid token status=%d calls=%d body=%s", invalidToken.Code, businessCalls.Load(), invalidToken.Body.String())
	}
	denied := milestoneRequest(handler, http.MethodGet, prefix+route+"?expand=owner", bobToken)
	if denied.Code != http.StatusForbidden || businessCalls.Load() != 0 {
		t.Fatalf("denied status=%d calls=%d body=%s", denied.Code, businessCalls.Load(), denied.Body.String())
	}
	allowed := milestoneRequest(handler, http.MethodGet, prefix+route+"?expand=owner", aliceToken)
	if allowed.Code != http.StatusOK || allowed.Body.String() != `{"source":"plugin","id":"item-1"}` || businessCalls.Load() != 1 {
		t.Fatalf("allowed status=%d calls=%d body=%s", allowed.Code, businessCalls.Load(), allowed.Body.String())
	}
	superAllowed := milestoneRequest(handler, http.MethodGet, prefix+route+"?expand=owner", rootToken)
	if superAllowed.Code != http.StatusOK || businessCalls.Load() != 2 {
		t.Fatalf("super admin status=%d calls=%d body=%s", superAllowed.Code, businessCalls.Load(), superAllowed.Body.String())
	}
	undeclared := milestoneRequest(handler, http.MethodGet, prefix+"/v1/plugins/"+pluginID+"/api/private", rootToken)
	if undeclared.Code != http.StatusForbidden || businessCalls.Load() != 2 {
		t.Fatalf("undeclared route status=%d calls=%d body=%s", undeclared.Code, businessCalls.Load(), undeclared.Body.String())
	}
	wrongMethod := milestoneRequest(handler, http.MethodPost, prefix+route+"?expand=owner", rootToken)
	if wrongMethod.Code != http.StatusForbidden || businessCalls.Load() != 2 {
		t.Fatalf("wrong method status=%d calls=%d body=%s", wrongMethod.Code, businessCalls.Load(), wrongMethod.Body.String())
	}

	evt := event.BusinessEvent{
		ID:          "milestone-event-1",
		EventName:   event.BusinessEventApprovalCompleted,
		Source:      "milestone-test",
		SubjectType: "approval",
		SubjectID:   "approval-1",
		Payload:     map[string]any{"status": "approved"},
		OccurredAt:  time.Now().UTC(),
	}
	if err := manager.PublishBusinessEvent(context.Background(), evt); err != nil {
		t.Fatalf("publish plugin event: %v", err)
	}
	if err := manager.PublishBusinessEvent(context.Background(), evt); err != nil {
		t.Fatalf("publish duplicate plugin event: %v", err)
	}
	select {
	case envelope := <-deliveries:
		if envelope.PluginID != pluginID || envelope.Handler != "onApprovalCompleted" || envelope.EventID != evt.ID {
			t.Fatalf("unexpected event envelope: %+v", envelope)
		}
	case <-time.After(time.Second):
		t.Fatal("plugin event was not delivered")
	}
	if eventCalls.Load() != 1 {
		t.Fatalf("duplicate event delivery calls=%d want=1", eventCalls.Load())
	}

	disabled := milestoneRequest(handler, http.MethodPost, prefix+"/v1/plugins/"+pluginID+"/disable", rootToken)
	if disabled.Code != http.StatusOK {
		t.Fatalf("disable plugin status=%d body=%s", disabled.Code, disabled.Body.String())
	}
	serviceState, ok = supervisor.Snapshot(pluginID)
	if !ok || serviceState.State != plugin.ServiceStateStopped {
		t.Fatalf("service did not stop: state=%+v exists=%v", serviceState, ok)
	}
	closed := milestoneRequest(handler, http.MethodGet, prefix+route+"?expand=owner", aliceToken)
	if closed.Code != http.StatusForbidden || businessCalls.Load() != 2 {
		t.Fatalf("disabled route status=%d calls=%d body=%s", closed.Code, businessCalls.Load(), closed.Body.String())
	}
	if err := manager.PublishBusinessEvent(context.Background(), event.BusinessEvent{
		ID: "milestone-event-disabled", EventName: event.BusinessEventApprovalCompleted, Source: "milestone-test", OccurredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("publish while disabled: %v", err)
	}
	if eventCalls.Load() != 1 {
		t.Fatalf("disabled plugin received event: calls=%d", eventCalls.Load())
	}

	uninstalled := milestoneRequest(handler, http.MethodDelete, prefix+"/v1/plugins/"+pluginID, rootToken)
	if uninstalled.Code != http.StatusOK {
		t.Fatalf("uninstall plugin status=%d body=%s", uninstalled.Code, uninstalled.Body.String())
	}
	if db.Migrator().HasTable("milestone_plugin_items") {
		t.Fatal("drop uninstall policy retained plugin table")
	}
	ledger, err := gormrepo.NewPluginMigrationStore(db).ListApplied(context.Background(), pluginID)
	if err != nil || len(ledger) != 0 {
		t.Fatalf("uninstall migration ledger=%d err=%v", len(ledger), err)
	}
	info, err := manager.Get(pluginID)
	if err != nil || info.State != plugin.StateUninstalled {
		t.Fatalf("uninstall state=%s err=%v", info.State, err)
	}

	assertRuntimeMilestoneAudit(t, auditEvents)
}

func writeRuntimeMilestonePlugin(t *testing.T, pluginID, serviceURL string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), pluginID)
	migrations := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migrations, 0o755); err != nil {
		t.Fatalf("create plugin migrations: %v", err)
	}
	manifest := fmt.Sprintf(`id: %s
name: Plugin Runtime Milestone
version: 1.0.0
service_base_url: %s
service_health_url: %s/health
permissions:
  - key: milestone_plugin.items.read
    type: api
    module: milestone_plugin
    name: Read milestone items
api:
  routes:
    - method: GET
      path: /v1/plugins/milestone_plugin/api/items/{id}
      summary: Get milestone item
      permission: milestone_plugin.items.read
      audit_action: milestone_plugin.items.read
events:
  subscriptions:
    - name: approval-completed
      handler: onApprovalCompleted
      retry_policy: standard
data:
  namespace: milestone_plugin
  migration_version: v1.0.0
  migration_directory: migrations
  uninstall_policy: drop
  rollback_policy: automatic
`, pluginID, serviceURL, serviceURL)
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrations, "001_create.up.sql"), []byte("CREATE TABLE milestone_plugin_items (id TEXT PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("write up migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrations, "001_create.down.sql"), []byte("DROP TABLE milestone_plugin_items;"), 0o600); err != nil {
		t.Fatalf("write down migration: %v", err)
	}
	return dir
}

func signMilestoneToken(t *testing.T, secret, subject, role string) string {
	t.Helper()
	token, err := security.SignJWT(secret, security.JWTIdentity{Subject: subject, Role: role, Roles: []string{role}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign %s token: %v", subject, err)
	}
	return token
}

func milestoneRequest(handler http.Handler, method, target, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func assertRuntimeMilestoneAudit(t *testing.T, events *memory.AuditEventStore) {
	t.Helper()
	pluginEvents, err := events.ListEvents(context.Background(), auditrepo.EventFilter{Type: domainaudit.EventTypePlugin})
	if err != nil {
		t.Fatalf("list plugin audit events: %v", err)
	}
	wantPluginActions := map[domainaudit.AuditAction]bool{
		"plugin.lifecycle.enable":     false,
		"plugin.lifecycle.disable":    false,
		"plugin.lifecycle.uninstall":  false,
		"milestone_plugin.items.read": false,
	}
	for _, item := range pluginEvents {
		if _, ok := wantPluginActions[item.Action]; ok && item.Result == domainaudit.EventResultSuccess {
			wantPluginActions[item.Action] = true
		}
	}
	for action, found := range wantPluginActions {
		if !found {
			t.Errorf("missing successful plugin audit action %s: %+v", action, pluginEvents)
		}
	}

	securityEvents, err := events.ListEvents(context.Background(), auditrepo.EventFilter{Type: domainaudit.EventTypeSecurity})
	if err != nil {
		t.Fatalf("list security audit events: %v", err)
	}
	denied := false
	for _, item := range securityEvents {
		if item.Actor.ID.String() == "bob" && item.Metadata["reason"] == "permission_denied" {
			denied = true
			break
		}
	}
	if !denied {
		t.Errorf("missing permission-denied security audit: %+v", securityEvents)
	}
}
