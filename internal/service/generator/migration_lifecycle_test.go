package generator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	menusvc "github.com/tinboxw/skoll/internal/service/menu"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
	"github.com/tinboxw/skoll/internal/store/memory"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGeneratedPluginLifecycleArtifacts(t *testing.T) {
	for _, uninstallPolicy := range []string{"retain", "drop"} {
		t.Run(uninstallPolicy, func(t *testing.T) {
			spec := mustPluginLifecycleSpec(t, uninstallPolicy)
			result, err := NewService().DryRun(context.Background(), DryRunInput{
				Spec:               spec,
				BatchID:            "generated-plugin-lifecycle-" + uninstallPolicy,
				ActorID:            "generator-test",
				MigrationTimestamp: "20260722_010203",
			})
			if err != nil {
				t.Fatalf("DryRun() error = %v", err)
			}

			pluginDir := materializeGeneratedPlugin(t, result, spec.Plugin.ID)
			info, err := pluginruntime.NewFileLoader().Load(pluginDir)
			if err != nil {
				manifest, readErr := os.ReadFile(filepath.Join(pluginDir, "plugin.yaml"))
				t.Fatalf("load generated manifest: %v (read error: %v)\n%s", err, readErr, manifest)
			}
			assertGeneratedDataManifest(t, info, spec, uninstallPolicy)

			plan, err := pluginruntime.NewMigrationPlanner(pluginDir, info.DataManifest.MigrationDirectory).Plan(nil)
			if err != nil {
				t.Fatalf("plan generated migrations: %v", err)
			}
			if len(plan.Applied) != 0 || len(plan.Pending) != 1 || plan.Pending[0].Version != 1 {
				t.Fatalf("fresh migration plan = %+v", plan)
			}

			store := newGeneratedMigrationStore()
			hook := pluginruntime.NewPluginMigrationHook(store, nil)
			installed, err := hook.Run(context.Background(), pluginruntime.PluginMigrationHookInput{
				PluginID:           info.ID,
				PluginDir:          pluginDir,
				MigrationDirectory: info.DataManifest.MigrationDirectory,
				Action:             pluginruntime.PluginMigrationInstall,
				ToVersion:          info.DataManifest.MigrationVersion,
			})
			if err != nil || len(installed) != 1 || len(store.records) != 1 || len(store.executed) != 1 {
				t.Fatalf("fresh install = steps:%+v records:%+v sql:%+v err:%v", installed, store.records, store.executed, err)
			}
			if !strings.Contains(store.executed[0], "CREATE TABLE {{table:products}}") ||
				!strings.Contains(store.executed[0], "id TEXT PRIMARY KEY") ||
				!strings.Contains(store.executed[0], "created_at DATETIME NOT NULL") {
				t.Fatalf("generated migration SQL = %q", store.executed[0])
			}

			upgraded, err := hook.Run(context.Background(), pluginruntime.PluginMigrationHookInput{
				PluginID:           info.ID,
				PluginDir:          pluginDir,
				MigrationDirectory: info.DataManifest.MigrationDirectory,
				Action:             pluginruntime.PluginMigrationUpgrade,
				FromVersion:        info.DataManifest.MigrationVersion,
				ToVersion:          info.DataManifest.MigrationVersion,
			})
			if err != nil || len(upgraded) != 0 || len(store.records) != 1 || len(store.executed) != 1 {
				t.Fatalf("idempotent upgrade = steps:%+v records:%+v sql:%+v err:%v", upgraded, store.records, store.executed, err)
			}

			if uninstallPolicy == "retain" {
				retained, err := hook.Run(context.Background(), pluginruntime.PluginMigrationHookInput{
					PluginID:           info.ID,
					PluginDir:          pluginDir,
					MigrationDirectory: info.DataManifest.MigrationDirectory,
					Action:             pluginruntime.PluginMigrationUninstall,
					FromVersion:        info.DataManifest.MigrationVersion,
					UninstallPolicy:    info.DataManifest.UninstallPolicy,
				})
				if err != nil || len(retained) != 0 || len(store.records) != 1 || len(store.executed) != 1 {
					t.Fatalf("retain uninstall = steps:%+v records:%+v sql:%+v err:%v", retained, store.records, store.executed, err)
				}
				rolledBack, err := hook.Run(context.Background(), pluginruntime.PluginMigrationHookInput{
					PluginID:           info.ID,
					PluginDir:          pluginDir,
					MigrationDirectory: info.DataManifest.MigrationDirectory,
					Action:             pluginruntime.PluginMigrationDowngrade,
					FromVersion:        info.DataManifest.MigrationVersion,
					RollbackPolicy:     info.DataManifest.RollbackPolicy,
				})
				if err != nil || len(rolledBack) != 1 || len(store.records) != 0 || !strings.Contains(store.executed[1], "DROP TABLE IF EXISTS {{table:products}}") {
					t.Fatalf("automatic rollback = steps:%+v records:%+v sql:%+v err:%v", rolledBack, store.records, store.executed, err)
				}
				return
			}

			dropped, err := hook.Run(context.Background(), pluginruntime.PluginMigrationHookInput{
				PluginID:           info.ID,
				PluginDir:          pluginDir,
				MigrationDirectory: info.DataManifest.MigrationDirectory,
				Action:             pluginruntime.PluginMigrationUninstall,
				FromVersion:        info.DataManifest.MigrationVersion,
				UninstallPolicy:    info.DataManifest.UninstallPolicy,
			})
			if err != nil || len(dropped) != 1 || len(store.records) != 0 || !strings.Contains(store.executed[1], "DROP TABLE IF EXISTS {{table:products}}") {
				t.Fatalf("drop uninstall = steps:%+v records:%+v sql:%+v err:%v", dropped, store.records, store.executed, err)
			}
		})
	}
}

func TestGeneratedCatalogSeedServicesAreIdempotent(t *testing.T) {
	spec := mustPluginSpec(t)
	permissionService := permissionsvc.NewService(memory.NewPermissionStore())
	menuService := menusvc.NewService(memory.NewMenuStore())
	permissions := []permissionsvc.RegisterResourceInput{
		{Key: spec.Permissions.ReadKey, Type: domainpermission.ResourceTypeAPI, Module: spec.Module.Package, Source: "system", Name: "Read " + spec.Table.CollectionName, Risk: domainpermission.RiskLevelLow},
		{Key: spec.Permissions.CreateKey, Type: domainpermission.ResourceTypeAPI, Module: spec.Module.Package, Source: "system", Name: "Create " + spec.Table.CollectionName, Risk: domainpermission.RiskLevelMedium},
		{Key: spec.Permissions.UpdateKey, Type: domainpermission.ResourceTypeAPI, Module: spec.Module.Package, Source: "system", Name: "Update " + spec.Table.CollectionName, Risk: domainpermission.RiskLevelMedium},
		{Key: spec.Permissions.DeleteKey, Type: domainpermission.ResourceTypeAPI, Module: spec.Module.Package, Source: "system", Name: "Delete " + spec.Table.CollectionName, Risk: domainpermission.RiskLevelHigh},
		{Key: spec.Permissions.ManageKey, Type: domainpermission.ResourceTypeAPI, Module: spec.Module.Package, Source: "system", Name: "Manage " + spec.Table.CollectionName, Risk: domainpermission.RiskLevelHigh},
	}
	node, err := domainmenu.NewNode(
		domainmenu.NodeIdentity{Key: spec.Menu.Key, ParentKey: spec.Menu.ParentKey, Source: "system"},
		domainmenu.NodeView{Name: spec.Page.Title, Path: spec.Menu.Path, Component: spec.Menu.Component, Icon: spec.Menu.Icon},
		spec.Menu.Order,
	)
	if err != nil {
		t.Fatalf("create generated menu seed: %v", err)
	}
	node.RequiredPermissions = append([]string(nil), spec.Menu.RequiredPermissions...)

	for attempt := 0; attempt < 2; attempt++ {
		for _, permission := range permissions {
			if _, err := permissionService.RegisterResource(context.Background(), permission); err != nil {
				t.Fatalf("register generated permission on attempt %d: %v", attempt+1, err)
			}
		}
		if _, err := menuService.MergeNodes(context.Background(), menusvc.MergeNodesInput{Nodes: []domainmenu.MenuNode{node}}); err != nil {
			t.Fatalf("merge generated menu on attempt %d: %v", attempt+1, err)
		}
	}

	resources, err := permissionService.ListResources(context.Background(), permissionsvc.ListResourcesInput{Source: "system"})
	if err != nil || len(resources) != 5 {
		t.Fatalf("repeated permission seed = items:%+v err:%v", resources, err)
	}
	nodes, err := menuService.Tree(context.Background(), menusvc.TreeInput{Source: "system"})
	if err != nil || len(nodes) != 1 || nodes[0].Key() != spec.Menu.Key {
		t.Fatalf("repeated menu seed = items:%+v err:%v", nodes, err)
	}
}

func TestGeneratedMigrationExecutesAndRollsBack(t *testing.T) {
	spec := mustSpec(t)
	for _, dialect := range []string{"mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file:generated_"+dialect+"?mode=memory&cache=shared"), &gorm.Config{})
			if err != nil {
				t.Fatalf("open migration database: %v", err)
			}
			if err := db.Exec(renderMigration(*spec, dialect)).Error; err != nil {
				t.Fatalf("execute generated %s migration: %v", dialect, err)
			}

			type columnInfo struct {
				Name string `gorm:"column:name"`
				PK   int    `gorm:"column:pk"`
			}
			var columns []columnInfo
			if err := db.Raw("PRAGMA table_info(products)").Scan(&columns).Error; err != nil {
				t.Fatalf("read generated table columns: %v", err)
			}
			byName := make(map[string]columnInfo, len(columns))
			for _, column := range columns {
				byName[column.Name] = column
			}
			if byName["id"].PK != 1 || byName["created_at"].Name == "" || byName["updated_at"].Name == "" {
				t.Fatalf("generated table columns = %+v", columns)
			}

			var indexCount int64
			if err := db.Raw("SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = ?", "idx_products_name").Scan(&indexCount).Error; err != nil || indexCount != 1 {
				t.Fatalf("generated index count = %d, err = %v", indexCount, err)
			}
			if err := db.Exec(renderPluginMigrationDown(*spec)).Error; err != nil {
				t.Fatalf("execute generated rollback: %v", err)
			}
			if db.Migrator().HasTable("products") {
				t.Fatal("generated rollback retained products table")
			}
		})
	}
}

func mustPluginLifecycleSpec(t *testing.T, uninstallPolicy string) *domaingenerator.GeneratorSpec {
	t.Helper()
	in := validServiceGeneratorSpecInput()
	in.Indexes[0].Name = "idx_pharma_oa_products_name"
	namespaceServicePluginInput(&in, "pharma_oa")
	in.Plugin = domaingenerator.PluginSpec{
		Enabled:         true,
		ID:              "pharma-oa",
		Name:            "Pharma OA",
		Description:     "Generated pharma OA business plugin",
		DataNamespace:   "pharma-oa",
		UninstallPolicy: uninstallPolicy,
		RollbackPolicy:  "automatic",
	}
	spec, err := domaingenerator.NewGeneratorSpec(in)
	if err != nil {
		t.Fatalf("NewGeneratorSpec() error = %v", err)
	}
	return spec
}

func materializeGeneratedPlugin(t *testing.T, result *DryRunResult, pluginID string) string {
	t.Helper()
	workspace := generatedPluginTestWorkspace(t)
	root := filepath.Join(workspace, "examples", "plugins", pluginID)
	prefix := filepath.ToSlash(filepath.Join("examples", "plugins", pluginID)) + "/"
	for _, file := range result.Files {
		path := filepath.ToSlash(file.Path)
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		relative := strings.TrimPrefix(path, prefix)
		target := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("create generated plugin directory: %v", err)
		}
		if err := os.WriteFile(target, []byte(file.GeneratedContent), 0o644); err != nil {
			t.Fatalf("write generated plugin file %s: %v", relative, err)
		}
	}
	return root
}

func generatedPluginTestWorkspace(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "windows" {
		return t.TempDir()
	}
	workspace := filepath.Join(os.TempDir(), fmt.Sprintf("skoll-generated-%d-%d", os.Getpid(), time.Now().UnixNano()))
	cmd := exec.Command(
		"powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		`$ErrorActionPreference = "Stop"; New-Item -ItemType Directory -Path $env:SKOLL_GENERATED_WORKSPACE | Out-Null`,
	)
	cmd.Env = append(os.Environ(), "SKOLL_GENERATED_WORKSPACE="+workspace)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create generated plugin test workspace %s: %v\n%s", workspace, err, output)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(workspace); err != nil {
			t.Errorf("remove generated plugin test workspace %s: %v", workspace, err)
		}
	})
	return workspace
}

func assertGeneratedDataManifest(t *testing.T, info pluginruntime.Info, spec *domaingenerator.GeneratorSpec, uninstallPolicy string) {
	t.Helper()
	if info.DataManifest == nil {
		t.Fatal("generated manifest has no data contract")
	}
	data := info.DataManifest
	if data.Namespace != spec.Plugin.DataNamespace ||
		data.MigrationVersion != spec.Plugin.Version ||
		data.MigrationDirectory != spec.Plugin.MigrationDirectory ||
		string(data.UninstallPolicy) != uninstallPolicy ||
		data.RollbackPolicy != pluginruntime.DataRollbackAutomatic {
		t.Fatalf("generated data manifest = %+v", data)
	}
	if len(data.Tables) != 1 || data.Tables[0].Name != "products" || data.Tables[0].PrimaryKey != "id" {
		t.Fatalf("generated data tables = %+v", data.Tables)
	}
}

type generatedMigrationStore struct {
	records  []pluginruntime.MigrationRecord
	executed []string
}

type generatedMigrationTransaction struct {
	records  []pluginruntime.MigrationRecord
	executed []string
}

func newGeneratedMigrationStore() *generatedMigrationStore {
	return &generatedMigrationStore{records: []pluginruntime.MigrationRecord{}, executed: []string{}}
}

func (s *generatedMigrationStore) ListApplied(_ context.Context, pluginID string) ([]pluginruntime.MigrationRecord, error) {
	out := make([]pluginruntime.MigrationRecord, 0, len(s.records))
	for _, record := range s.records {
		if record.PluginID == pluginID {
			out = append(out, record)
		}
	}
	return out, nil
}

func (s *generatedMigrationStore) WithTransaction(_ context.Context, fn func(pluginruntime.MigrationTransaction) error) error {
	tx := &generatedMigrationTransaction{
		records:  append([]pluginruntime.MigrationRecord(nil), s.records...),
		executed: append([]string(nil), s.executed...),
	}
	if err := fn(tx); err != nil {
		return err
	}
	s.records = tx.records
	s.executed = tx.executed
	return nil
}

func (*generatedMigrationStore) RequiresApplyCompensation() bool { return false }

func (tx *generatedMigrationTransaction) ExecSQL(sql string) error {
	tx.executed = append(tx.executed, sql)
	return nil
}

func (tx *generatedMigrationTransaction) MarkApplied(record pluginruntime.MigrationRecord) error {
	tx.records = append(tx.records, record)
	return nil
}

func (tx *generatedMigrationTransaction) RemoveApplied(pluginID string, version int) error {
	for index, record := range tx.records {
		if record.PluginID == pluginID && record.Version == version {
			tx.records = append(tx.records[:index], tx.records[index+1:]...)
			return nil
		}
	}
	return nil
}
