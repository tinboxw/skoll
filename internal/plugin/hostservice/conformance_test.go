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
	"time"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	documentnumbersvc "github.com/tinboxw/skoll/internal/service/documentnumber"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"github.com/tinboxw/skoll/internal/service/rbac"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store"
	objectstore "github.com/tinboxw/skoll/internal/store/object"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
	sdkconformance "github.com/tinboxw/skoll/plugins/sdk-conformance"
)

type conformanceDataStore struct {
	record *pluginsdk.DataRecord
}

func (s *conformanceDataStore) Query(context.Context, pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	if s.record == nil {
		return pluginsdk.DataPage{}, nil
	}
	return pluginsdk.DataPage{Records: []pluginsdk.DataRecord{*s.record}}, nil
}
func (s *conformanceDataStore) Mutate(_ context.Context, mutation pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	record := pluginsdk.DataRecord{Values: map[string]pluginsdk.DataValue{
		"id": mutation.Key["id"], "name": mutation.Values["name"],
	}, Version: 1}
	s.record = &record
	return pluginsdk.DataMutationResult{RowsAffected: 1, Record: &record}, nil
}
func (s *conformanceDataStore) Aggregate(context.Context, pluginsdk.DataAggregateQuery) (pluginsdk.DataAggregatePage, error) {
	return pluginsdk.DataAggregatePage{}, nil
}

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
	dataStore := &conformanceDataStore{}
	jobs := jobsvc.NewService(bundle.Jobs, nil)
	host, err := NewHostServices(HostServicesDependencies{
		PluginID: "sdk_conformance", Transactions: transactions, DataScopes: scopes,
		DataStore: func(string, pluginsdk.DataScopeService, pluginsdk.AuditService) (pluginsdk.DataStoreService, error) {
			return dataStore, nil
		},
		EventPublications: func(string) ([]pluginsdk.EventPublicationDeclaration, error) {
			return []pluginsdk.EventPublicationDeclaration{{
				Name: "conformance-event", SchemaVersion: 1,
				PayloadType: "conformance.event", Scope: pluginsdk.EventScopeTenant,
			}}, nil
		},
		EventOutbox: gormrepo.NewPluginEventOutboxStore(bundle.PluginDataDB),
		Files:       filesvc.NewService(bundle.Files, objects, filesvc.Options{}), Audit: auditsvc.NewService(bundle.Audit),
		DocumentNumbers:   documentnumbersvc.NewService(gormrepo.NewDocumentNumberStore(bundle.PluginDataDB)),
		DocumentWorkflows: gormrepo.NewDocumentWorkflowStore(bundle.PluginDataDB),
		ConfigStore:       configStore, System: systemsvc.NewService(bundle.System), MasterSecret: "sdk-conformance-master-secret",
		Workflow: workflowsvc.NewService(bundle.Workflow, workflowsvc.Options{
			Jobs: jobs, UnitOfWork: bundle.UnitOfWork, Now: func() time.Time { return time.Now().UTC() },
		}), Jobs: jobs,
	})
	if err != nil {
		t.Fatalf("NewHostServices error: %v", err)
	}
	ctx := security.WithJWTClaimsContext(context.Background(), &security.JWTClaims{Subject: "conformance-user", Role: "super_admin", Roles: []string{"super_admin"}})
	report, err := sdkconformance.Run(ctx, host)
	if err != nil {
		t.Fatalf("SDK conformance error: %v", err)
	}
	if !report.Transaction || !report.Scope || !report.DataStore || !report.Event || !report.Document || !report.Collaboration || !report.DocumentQuery || !report.File || !report.Audit || !report.Config || !report.Secret || !report.Workflow || !report.Job {
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
