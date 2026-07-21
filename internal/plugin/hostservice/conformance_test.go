package hostservice

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"github.com/tinboxw/skoll/internal/service/rbac"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store"
	objectstore "github.com/tinboxw/skoll/internal/store/object"
	"github.com/tinboxw/skoll/pkg/security"
	sdkconformance "github.com/tinboxw/skoll/plugins/sdk-conformance"
)

func TestThirdPartyPluginPassesPublicSDKConformance(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("NewBundle error: %v", err)
	}
	transactions, err := NewTransactionService(bundle.UnitOfWork)
	if err != nil {
		t.Fatalf("NewTransactionService error: %v", err)
	}
	scopes, err := NewDataScopeService(rbac.NewServiceWithOrganization(bundle.RBAC, bundle.Organization), bundle.Organization)
	if err != nil {
		t.Fatalf("NewDataScopeService error: %v", err)
	}
	objects, err := objectstore.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStore error: %v", err)
	}
	configStore := &hostConfigStore{info: pluginruntime.Info{ID: "sdk_conformance"}}
	host, err := NewHostServices(HostServicesDependencies{
		PluginID: "sdk_conformance", Transactions: transactions, DataScopes: scopes,
		Files: filesvc.NewService(bundle.Files, objects, filesvc.Options{}), Audit: auditsvc.NewService(bundle.Audit),
		ConfigStore: configStore, System: systemsvc.NewService(bundle.System), MasterSecret: "sdk-conformance-master-secret",
		Workflow: workflowsvc.NewService(bundle.Workflow), Jobs: jobsvc.NewService(bundle.Jobs, nil),
	})
	if err != nil {
		t.Fatalf("NewHostServices error: %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "conformance-user", Role: "super_admin", Roles: []string{"super_admin"}})
	report, err := sdkconformance.Run(ctx, host)
	if err != nil {
		t.Fatalf("SDK conformance error: %v", err)
	}
	if !report.Transaction || !report.Scope || !report.File || !report.Audit || !report.Config || !report.Secret || !report.Workflow || !report.Job {
		t.Fatalf("incomplete SDK conformance report: %+v", report)
	}
}

func TestSDKConformancePluginImportsOnlyPublicContracts(t *testing.T) {
	path := filepath.Join(conformanceRepositoryRoot(t), "plugins", "sdk-conformance", "plugin.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse conformance plugin: %v", err)
	}
	allowed := map[string]bool{
		"context": true, "encoding/json": true, "fmt": true, "time": true,
		"github.com/tinboxw/skoll/pkg/pluginsdk": true,
	}
	for _, spec := range file.Decls {
		declaration, ok := spec.(*ast.GenDecl)
		if !ok || declaration.Tok != token.IMPORT {
			continue
		}
		for _, item := range declaration.Specs {
			path, unquoteErr := strconv.Unquote(item.(*ast.ImportSpec).Path.Value)
			if unquoteErr != nil || !allowed[path] {
				t.Fatalf("conformance plugin imports non-public dependency %q", path)
			}
		}
	}
}

func TestSDKConformanceManifestLifecycle(t *testing.T) {
	root := conformanceRepositoryRoot(t)
	path := filepath.Join(root, "plugins", "sdk-conformance")
	manager := pluginruntime.NewRuntimeManager(pluginruntime.NewFileLoader(), pluginruntime.NewTopologicalResolver())
	info, err := manager.Install(path)
	if err != nil || info.ID != "sdk_conformance" {
		t.Fatalf("Install info=%+v err=%v", info, err)
	}
	if err = manager.Enable(info.ID); err != nil {
		t.Fatalf("Enable error: %v", err)
	}
	enabled, err := manager.Get(info.ID)
	if err != nil || enabled.State != pluginruntime.StateEnabled {
		t.Fatalf("enabled info=%+v err=%v", enabled, err)
	}
	if err = manager.Disable(info.ID); err != nil {
		t.Fatalf("Disable error: %v", err)
	}
	if err = manager.Uninstall(info.ID); err != nil {
		t.Fatalf("Uninstall error: %v", err)
	}
	uninstalled, err := manager.Get(info.ID)
	if err != nil || uninstalled.State != pluginruntime.StateUninstalled {
		t.Fatalf("uninstalled info=%+v err=%v", uninstalled, err)
	}
}

func conformanceRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", ".."))
}
