package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
)

func renderCandidateContent(templateID string, spec domaingenerator.GeneratorSpec) string {
	switch templateID {
	case "backend.domain.doc":
		return formatGoTemplate(renderDomainDoc(spec))
	case "backend.domain.entity":
		return formatGoTemplate(renderDomainEntity(spec))
	case "backend.domain.test":
		return formatGoTemplate(renderDomainTest(spec))
	case "backend.repository":
		return formatGoTemplate(renderRepository(spec))
	case "backend.store.memory":
		return formatGoTemplate(renderMemoryStore(spec))
	case "backend.store.memory.test":
		return formatGoTemplate(renderMemoryStoreTest(spec))
	case "backend.store.gorm.model":
		return formatGoTemplate(renderGORMModel(spec))
	case "backend.store.gorm.repo":
		return formatGoTemplate(renderGORMStore(spec))
	case "backend.store.gorm.test":
		return formatGoTemplate(renderGORMStoreTest(spec))
	case "backend.migration.mysql":
		return renderMigration(spec, "mysql")
	case "backend.migration.postgres":
		return renderMigration(spec, "postgres")
	case "backend.service":
		return formatGoTemplate(renderService(spec))
	case "backend.service.impl":
		return formatGoTemplate(renderServiceImpl(spec))
	case "backend.service.test":
		return formatGoTemplate(renderServiceTest(spec))
	case "backend.handler":
		return formatGoTemplate(renderHandler(spec))
	case "backend.handler.test":
		return formatGoTemplate(renderHandlerTest(spec))
	case "backend.router":
		return formatGoTemplate(renderRouter(spec))
	case "backend.openapi.docs", "backend.openapi.runtime":
		return renderOpenAPI(spec)
	case "backend.permission.seed":
		return formatGoTemplate(renderPermissionSeed(spec))
	case "frontend.api":
		return renderFrontendAPI(spec)
	case "frontend.store":
		return renderFrontendStore(spec)
	case "frontend.locale":
		return renderFrontendLocale(spec)
	case "frontend.route":
		return renderFrontendRoute(spec)
	case "frontend.view":
		return renderFrontendView(spec)
	case "plugin.manifest":
		return renderPluginManifest(spec)
	case "plugin.go.mod":
		return renderPluginGoMod(spec)
	case "plugin.backend.server":
		if spec.Document != nil {
			return formatGoTemplate(renderDocumentPluginBackendServer(spec))
		}
		return formatGoTemplate(renderPluginBackendServer(spec))
	case "plugin.migration.up":
		return renderMigration(spec, "postgres")
	case "plugin.migration.down":
		return renderPluginMigrationDown(spec)
	case "plugin.frontend.api":
		return renderPluginFrontendAPI(spec)
	case "plugin.frontend.store":
		if spec.Document != nil {
			return renderDocumentPluginFrontendStore(spec)
		}
		return renderFrontendStore(spec)
	case "plugin.frontend.locale":
		return renderFrontendLocale(spec)
	case "plugin.frontend.route":
		return renderPluginFrontendRoute(spec)
	case "plugin.frontend.view":
		return renderPluginFrontendView(spec)
	case "plugin.frontend.package":
		return renderPluginFrontendPackage(spec)
	case "plugin.frontend.tsconfig":
		return renderPluginFrontendTSConfig(spec)
	case "plugin.frontend.vite":
		return renderPluginFrontendVite(spec)
	case "plugin.frontend.index":
		return renderPluginFrontendIndex(spec)
	case "plugin.frontend.main":
		return renderPluginFrontendMain(spec)
	case "plugin.frontend.styles":
		return renderPluginFrontendStyles()
	case "plugin.frontend.api-support":
		return renderPluginFrontendAPISupport(spec)
	case "plugin.frontend.common-support":
		return renderPluginFrontendCommonSupport()
	case "plugin.frontend.i18n-support":
		return renderPluginFrontendI18nSupport()
	case "plugin.frontend.permission-support":
		return renderPluginFrontendPermissionSupport()
	case "plugin.frontend.ui-support":
		return renderPluginFrontendUISupport()
	case "plugin.document.schema":
		return renderDocumentSchemaJSON(spec)
	case "plugin.frontend.document-schema":
		return renderDocumentFrontendSchema(spec)
	case "plugin.acceptance.test":
		return formatGoTemplate(renderPluginAcceptanceTest(spec))
	case "plugin.command.powershell":
		return renderPluginPowerShellCommand(spec)
	case "plugin.command.shell":
		return renderPluginShellCommand(spec)
	case "plugin.readme":
		return renderPluginREADME(spec)
	default:
		return fmt.Sprintf("// template %s for %s\n", templateID, spec.Module.Package)
	}
}

func renderDomainDoc(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf("package %s\n\n// Package %s contains generated domain code for %s.\n", spec.Module.Package, spec.Module.Package, spec.Table.CollectionName)
}

func renderDomainEntity(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	pkg := spec.Module.Package
	domainName := spec.Table.DomainName
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	fmt.Fprintf(&b, "import (\n\t\"fmt\"\n\t\"time\"\n\n\t\"github.com/tinboxw/skoll/internal/domain/shared\"\n)\n\n")
	fmt.Fprintf(&b, "type %s struct {\n", domainName)
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "\t%s %s `json:\"%s\"`\n", exportedName(field.Name), goFieldType(field), field.Name)
	}
	fmt.Fprintf(&b, "\tMeta shared.AuditMeta `json:\"meta\"`\n")
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "type %sInput struct {\n", domainName)
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "\t%s %s `json:\"%s\"`\n", exportedName(field.Name), goFieldType(field), field.Name)
	}
	fmt.Fprintf(&b, "\tCreatedAt time.Time\n\tUpdatedAt time.Time\n")
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "func New%s(in %sInput) (*%s, error) {\n", domainName, domainName, domainName)
	fmt.Fprintf(&b, "\tif err := validate%sInput(in); err != nil {\n\t\treturn nil, err\n\t}\n", domainName)
	fmt.Fprintf(&b, "\tif in.UpdatedAt.IsZero() {\n\t\tin.UpdatedAt = in.CreatedAt\n\t}\n")
	fmt.Fprintf(&b, "\treturn &%s{\n", domainName)
	for _, field := range spec.Fields {
		name := exportedName(field.Name)
		fmt.Fprintf(&b, "\t\t%s: in.%s,\n", name, name)
	}
	fmt.Fprintf(&b, "\t\tMeta: shared.AuditMeta{CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt},\n\t}, nil\n}\n\n")
	fmt.Fprintf(&b, "func validate%sInput(in %sInput) error {\n", domainName, domainName)
	for _, field := range spec.Fields {
		name := exportedName(field.Name)
		if field.Required || field.PrimaryKey {
			switch field.Type {
			case domaingenerator.FieldTypeString, domaingenerator.FieldTypeText, domaingenerator.FieldTypeID:
				fmt.Fprintf(&b, "\tif fmt.Sprint(in.%s) == \"\" {\n\t\treturn fmt.Errorf(\"%s is required\")\n\t}\n", name, field.Name)
			}
		}
	}
	fmt.Fprintf(&b, "\tif in.CreatedAt.IsZero() {\n\t\treturn fmt.Errorf(\"created time is required\")\n\t}\n")
	fmt.Fprintf(&b, "\treturn nil\n}\n")
	return b.String()
}

func renderDomainTest(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	domainName := spec.Table.DomainName
	primary := primaryField(spec)
	fmt.Fprintf(&b, "package %s\n\n", spec.Module.Package)
	fmt.Fprintf(&b, "import (\n\t\"testing\"\n\t\"time\"\n)\n\n")
	fmt.Fprintf(&b, "func TestNew%sValidatesAndBuildsEntity(t *testing.T) {\n", domainName)
	fmt.Fprintf(&b, "\tnow := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)\n")
	fmt.Fprintf(&b, "\titem, err := New%s(%sInput{\n", domainName, domainName)
	renderSampleAssignments(&b, spec, "now", true)
	fmt.Fprintf(&b, "\t\tCreatedAt: now,\n\t})\n")
	fmt.Fprintf(&b, "\tif err != nil {\n\t\tt.Fatalf(\"New%s() error = %%v\", err)\n\t}\n", domainName)
	fmt.Fprintf(&b, "\tif item.%s == %s || item.Meta.CreatedAt != now || item.Meta.UpdatedAt != now {\n", exportedName(primary.Name), zeroFieldValue(primary))
	fmt.Fprintf(&b, "\t\tt.Fatalf(\"entity = %%+v\", item)\n\t}\n")
	fmt.Fprintf(&b, "\tif _, err := New%s(%sInput{CreatedAt: now}); err == nil {\n", domainName, domainName)
	fmt.Fprintf(&b, "\t\tt.Fatal(\"expected required-field validation error\")\n\t}\n}\n")
	return b.String()
}

func renderRepository(spec domaingenerator.GeneratorSpec) string {
	pkg := spec.Module.Package
	domainName := spec.Table.DomainName
	alias := "domain" + pkg
	return fmt.Sprintf(`package %s

import (
	"context"

	%s "github.com/tinboxw/skoll/internal/domain/%s"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type ListFilter struct {
	Keyword string
}

type %sRepository interface {
	Create(ctx context.Context, item %s.%s) error
	Update(ctx context.Context, item %s.%s) error
	Get(ctx context.Context, id shared.ID) (*%s.%s, error)
	List(ctx context.Context, filter ListFilter, offset int, limit int) ([]%s.%s, error)
	Delete(ctx context.Context, id shared.ID) error
}
`, pkg, alias, pkg, domainName, alias, domainName, alias, domainName, alias, domainName, alias, domainName)
}

func renderMemoryStore(spec domaingenerator.GeneratorSpec) string {
	pkg := spec.Module.Package
	domainName := spec.Table.DomainName
	storeName := domainName + "Store"
	alias := "domain" + pkg
	repoAlias := pkg + "repo"
	primaryName := exportedName(primaryField(spec).Name)
	content := fmt.Sprintf(`package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	%s "github.com/tinboxw/skoll/internal/domain/%s"
	"github.com/tinboxw/skoll/internal/domain/shared"
	%s "github.com/tinboxw/skoll/internal/repository/%s"
)

type %s struct {
	mu    sync.RWMutex
	items map[shared.ID]%s.%s
}

func New%s() *%s {
	return &%s{items: make(map[shared.ID]%s.%s)}
}

func (s *%s) Create(_ context.Context, item %s.%s) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[item.{{PRIMARY}}]; ok {
		return fmt.Errorf("%s already exists")
	}
	s.items[item.{{PRIMARY}}] = item
	return nil
}

func (s *%s) Update(_ context.Context, item %s.%s) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[item.{{PRIMARY}}]; !ok {
		return fmt.Errorf("%s not found")
	}
	s.items[item.{{PRIMARY}}] = item
	return nil
}

func (s *%s) Get(_ context.Context, id shared.ID) (*%s.%s, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("%s not found")
	}
	return &item, nil
}

func (s *%s) List(_ context.Context, filter %s.ListFilter, offset int, limit int) ([]%s.%s, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]%s.%s, 0, len(s.items))
	for _, item := range s.items {
		if filter.Keyword != "" && !strings.Contains(strings.ToLower(fmt.Sprint(item)), strings.ToLower(filter.Keyword)) {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool { return fmt.Sprint(items[i].{{PRIMARY}}) < fmt.Sprint(items[j].{{PRIMARY}}) })
	if offset > len(items) {
		return []%s.%s{}, nil
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return append([]%s.%s(nil), items[offset:end]...), nil
}

func (s *%s) Delete(_ context.Context, id shared.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("%s not found")
	}
	delete(s.items, id)
	return nil
}
`, alias, pkg, repoAlias, pkg, storeName, alias, domainName, storeName, storeName, storeName, alias, domainName, storeName, alias, domainName, pkg, storeName, alias, domainName, pkg, storeName, alias, domainName, pkg, storeName, repoAlias, alias, domainName, alias, domainName, alias, domainName, alias, domainName, storeName, pkg)
	content = strings.ReplaceAll(content, "{{PRIMARY}}", primaryName)
	return content
}

func renderMemoryStoreTest(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	pkg := spec.Module.Package
	domainName := spec.Table.DomainName
	alias := "domain" + pkg
	primary := primaryField(spec)
	fmt.Fprintf(&b, "package memory\n\n")
	fmt.Fprintf(&b, "import (\n\t\"context\"\n\t\"testing\"\n\t\"time\"\n\n")
	fmt.Fprintf(&b, "\t%s \"github.com/tinboxw/skoll/internal/domain/%s\"\n", alias, pkg)
	fmt.Fprintf(&b, "\t%srepo \"github.com/tinboxw/skoll/internal/repository/%s\"\n)\n\n", pkg, pkg)
	fmt.Fprintf(&b, "func Test%sStoreCRUD(t *testing.T) {\n", domainName)
	fmt.Fprintf(&b, "\tctx := context.Background()\n\tnow := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)\n")
	fmt.Fprintf(&b, "\titem, err := %s.New%s(%s.%sInput{\n", alias, domainName, alias, domainName)
	renderSampleAssignments(&b, spec, "now", true)
	fmt.Fprintf(&b, "\t\tCreatedAt: now,\n\t})\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n")
	fmt.Fprintf(&b, "\tstore := New%sStore()\n", domainName)
	fmt.Fprintf(&b, "\tif err := store.Create(ctx, *item); err != nil {\n\t\tt.Fatal(err)\n\t}\n")
	fmt.Fprintf(&b, "\tgot, err := store.Get(ctx, item.%s)\n\tif err != nil || got.%s != item.%s {\n\t\tt.Fatalf(\"Get() = %%+v, %%v\", got, err)\n\t}\n", exportedName(primary.Name), exportedName(primary.Name), exportedName(primary.Name))
	fmt.Fprintf(&b, "\titem.Meta.UpdatedAt = now.Add(time.Minute)\n\tif err := store.Update(ctx, *item); err != nil {\n\t\tt.Fatal(err)\n\t}\n")
	fmt.Fprintf(&b, "\titems, err := store.List(ctx, %srepo.ListFilter{}, 0, 10)\n", pkg)
	fmt.Fprintf(&b, "\tif err != nil || len(items) != 1 {\n\t\tt.Fatalf(\"List() = %%+v, %%v\", items, err)\n\t}\n")
	fmt.Fprintf(&b, "\tif err := store.Delete(ctx, item.%s); err != nil {\n\t\tt.Fatal(err)\n\t}\n", exportedName(primary.Name))
	fmt.Fprintf(&b, "\tif _, err := store.Get(ctx, item.%s); err == nil {\n\t\tt.Fatal(\"expected deleted entity to be absent\")\n\t}\n}\n", exportedName(primary.Name))
	return b.String()
}

func renderGORMModel(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "package gormrepo\n\nimport \"time\"\n\n")
	fmt.Fprintf(&b, "type %sModel struct {\n", spec.Table.DomainName)
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "\t%s %s `gorm:\"column:%s%s\"`\n", exportedName(field.Name), gormFieldType(field), field.ColumnName, gormFieldTags(field))
	}
	fmt.Fprintf(&b, "\tCreatedAt time.Time `gorm:\"column:created_at\"`\n\tUpdatedAt time.Time `gorm:\"column:updated_at\"`\n")
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "func (%sModel) TableName() string { return %q }\n", spec.Table.DomainName, spec.Table.Name)
	return b.String()
}

func renderGORMStore(spec domaingenerator.GeneratorSpec) string {
	pkg := spec.Module.Package
	domainName := spec.Table.DomainName
	alias := "domain" + pkg
	repoAlias := pkg + "repo"
	primary := primaryField(spec)
	var b bytes.Buffer
	fmt.Fprintf(&b, "package gormrepo\n\n")
	fmt.Fprintf(&b, "import (\n\t\"context\"\n\t\"fmt\"\n\n")
	fmt.Fprintf(&b, "\t%s \"github.com/tinboxw/skoll/internal/domain/%s\"\n", alias, pkg)
	fmt.Fprintf(&b, "\t\"github.com/tinboxw/skoll/internal/domain/shared\"\n")
	fmt.Fprintf(&b, "\t%s \"github.com/tinboxw/skoll/internal/repository/%s\"\n", repoAlias, pkg)
	fmt.Fprintf(&b, "\t\"gorm.io/gorm\"\n)\n\n")
	fmt.Fprintf(&b, "type %sStore struct {\n\tdb *gorm.DB\n}\n\n", domainName)
	fmt.Fprintf(&b, "func New%sStore(db *gorm.DB) *%sStore {\n\treturn &%sStore{db: db}\n}\n\n", domainName, domainName, domainName)
	fmt.Fprintf(&b, "func (s *%sStore) Create(ctx context.Context, item %s.%s) error {\n\treturn s.db.WithContext(ctx).Create(to%sModel(item)).Error\n}\n\n", domainName, alias, domainName, domainName)
	fmt.Fprintf(&b, "func (s *%sStore) Update(ctx context.Context, item %s.%s) error {\n\treturn s.db.WithContext(ctx).Save(to%sModel(item)).Error\n}\n\n", domainName, alias, domainName, domainName)
	fmt.Fprintf(&b, "func (s *%sStore) Get(ctx context.Context, id shared.ID) (*%s.%s, error) {\n", domainName, alias, domainName)
	fmt.Fprintf(&b, "\tvar model %sModel\n", domainName)
	fmt.Fprintf(&b, "\tif err := s.db.WithContext(ctx).First(&model, \"%s = ?\", id.String()).Error; err != nil {\n\t\tif err == gorm.ErrRecordNotFound {\n\t\t\treturn nil, fmt.Errorf(\"%s not found\")\n\t\t}\n\t\treturn nil, err\n\t}\n", primary.ColumnName, pkg)
	fmt.Fprintf(&b, "\titem, err := from%sModel(model)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn &item, nil\n}\n\n", domainName)
	fmt.Fprintf(&b, "func (s *%sStore) List(ctx context.Context, filter %s.ListFilter, offset int, limit int) ([]%s.%s, error) {\n", domainName, repoAlias, alias, domainName)
	fmt.Fprintf(&b, "\tvar models []%sModel\n\tquery := s.db.WithContext(ctx).Model(&%sModel{})\n", domainName, domainName)
	fmt.Fprintf(&b, "\tif filter.Keyword != \"\" {\n\t\tquery = query.Where(\"id LIKE ?\", \"%%\"+filter.Keyword+\"%%\")\n\t}\n\tif offset > 0 {\n\t\tquery = query.Offset(offset)\n\t}\n\tif limit > 0 {\n\t\tquery = query.Limit(limit)\n\t}\n\tif err := query.Find(&models).Error; err != nil {\n\t\treturn nil, err\n\t}\n")
	fmt.Fprintf(&b, "\titems := make([]%s.%s, 0, len(models))\n\tfor _, model := range models {\n\t\titem, err := from%sModel(model)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\titems = append(items, item)\n\t}\n\treturn items, nil\n}\n\n", alias, domainName, domainName)
	fmt.Fprintf(&b, "func (s *%sStore) Delete(ctx context.Context, id shared.ID) error {\n\treturn s.db.WithContext(ctx).Delete(&%sModel{}, \"%s = ?\", id.String()).Error\n}\n\n", domainName, domainName, primary.ColumnName)
	fmt.Fprintf(&b, "func to%sModel(item %s.%s) %sModel {\n\treturn %sModel{\n", domainName, alias, domainName, domainName, domainName)
	for _, field := range spec.Fields {
		name := exportedName(field.Name)
		value := "item." + name
		if field.Type == domaingenerator.FieldTypeID {
			value += ".String()"
		}
		fmt.Fprintf(&b, "\t\t%s: %s,\n", name, value)
	}
	fmt.Fprintf(&b, "\t\tCreatedAt: item.Meta.CreatedAt,\n\t\tUpdatedAt: item.Meta.UpdatedAt,\n\t}\n}\n\n")
	fmt.Fprintf(&b, "func from%sModel(model %sModel) (%s.%s, error) {\n", domainName, domainName, alias, domainName)
	fmt.Fprintf(&b, "\titem, err := %s.New%s(%s.%sInput{\n", alias, domainName, alias, domainName)
	for _, field := range spec.Fields {
		name := exportedName(field.Name)
		value := "model." + name
		if field.Type == domaingenerator.FieldTypeID {
			value = "shared.ID(" + value + ")"
		}
		fmt.Fprintf(&b, "\t\t%s: %s,\n", name, value)
	}
	fmt.Fprintf(&b, "\t\tCreatedAt: model.CreatedAt,\n\t\tUpdatedAt: model.UpdatedAt,\n\t})\n")
	fmt.Fprintf(&b, "\tif err != nil {\n\t\treturn %s.%s{}, fmt.Errorf(\"decode %s model: %%w\", err)\n\t}\n", alias, domainName, pkg)
	fmt.Fprintf(&b, "\treturn *item, nil\n}\n")
	return b.String()
}

func renderGORMStoreTest(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	pkg := spec.Module.Package
	domainName := spec.Table.DomainName
	alias := "domain" + pkg
	fmt.Fprintf(&b, "package gormrepo\n\n")
	fmt.Fprintf(&b, "import (\n\t\"reflect\"\n\t\"testing\"\n\t\"time\"\n\n")
	fmt.Fprintf(&b, "\t%s \"github.com/tinboxw/skoll/internal/domain/%s\"\n)\n\n", alias, pkg)
	fmt.Fprintf(&b, "func Test%sModelRoundTrip(t *testing.T) {\n", domainName)
	fmt.Fprintf(&b, "\tnow := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)\n")
	fmt.Fprintf(&b, "\twant, err := %s.New%s(%s.%sInput{\n", alias, domainName, alias, domainName)
	renderSampleAssignments(&b, spec, "now", true)
	fmt.Fprintf(&b, "\t\tCreatedAt: now,\n\t})\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n")
	fmt.Fprintf(&b, "\tmodel := to%sModel(*want)\n", domainName)
	fmt.Fprintf(&b, "\tgot, err := from%sModel(model)\n", domainName)
	fmt.Fprintf(&b, "\tif err != nil || !reflect.DeepEqual(got, *want) {\n")
	fmt.Fprintf(&b, "\t\tt.Fatalf(\"round trip = %%+v, %%v\", got, err)\n\t}\n}\n")
	return b.String()
}

func renderMigration(spec domaingenerator.GeneratorSpec, dialect string) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s (\n", spec.Table.Name)
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "  %s %s%s%s,\n", field.ColumnName, sqlFieldType(field, dialect), sqlNullability(field), sqlFieldConstraints(field))
	}
	fmt.Fprintf(&b, "  created_at TIMESTAMP NOT NULL,\n")
	fmt.Fprintf(&b, "  updated_at TIMESTAMP NOT NULL,\n")
	fmt.Fprintf(&b, "  PRIMARY KEY (%s)\n", primaryColumn(spec))
	fmt.Fprintf(&b, ");\n")
	for _, index := range spec.Indexes {
		unique := ""
		if index.Unique {
			unique = "UNIQUE "
		}
		columns := make([]string, 0, len(index.Fields))
		for _, fieldName := range index.Fields {
			if field, ok := spec.FieldByName(fieldName); ok {
				columns = append(columns, field.ColumnName)
			}
		}
		fmt.Fprintf(&b, "CREATE %sINDEX %s ON %s (%s);\n", unique, index.Name, spec.Table.Name, strings.Join(columns, ", "))
	}
	return b.String()
}

func renderPluginMigrationDown(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", spec.Table.Name)
}

func renderPluginManifest(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "id: %s\n", spec.Plugin.ID)
	fmt.Fprintf(&b, "name: %s\n", spec.Plugin.Name)
	fmt.Fprintf(&b, "version: %s\n", spec.Plugin.Version)
	fmt.Fprintf(&b, "description: %s\n", spec.Plugin.Description)
	fmt.Fprintf(&b, "api_version: v1\n")
	fmt.Fprintf(&b, "ui_mode: %s\n", spec.Plugin.UIMode)
	fmt.Fprintf(&b, "frontend_entry: %s\n", spec.Plugin.FrontendEntry)
	if spec.Plugin.ServiceBaseURL != "" {
		fmt.Fprintf(&b, "service_base_url: %s\n", spec.Plugin.ServiceBaseURL)
		fmt.Fprintf(&b, "service_health_url: %s\n", spec.Plugin.ServiceHealthURL)
	}
	fmt.Fprintf(&b, "i18n_locales:\n  - zh-CN\n  - en-US\n")
	fmt.Fprintf(&b, "permissions:\n")
	for _, key := range pluginPermissionKeys(spec) {
		fmt.Fprintf(&b, "  - key: %s\n    type: api\n    module: %s\n    name: %s\n    risk: medium\n", key, spec.Module.Package, permissionDisplayName(key))
	}
	fmt.Fprintf(&b, "ui_menu:\n")
	fmt.Fprintf(&b, "  key: %s\n", spec.Menu.Key)
	if spec.Menu.ParentKey != "" {
		fmt.Fprintf(&b, "  parent_key: %s\n", spec.Menu.ParentKey)
	}
	fmt.Fprintf(&b, "  label: %s\n  path: %s\n  component: %s\n  icon: %s\n  order: %d\n", spec.Page.Title, spec.Plugin.FrontendEntry, spec.Menu.Component, spec.Menu.Icon, spec.Menu.Order)
	fmt.Fprintf(&b, "  required_permissions:\n")
	for _, permission := range spec.Menu.RequiredPermissions {
		fmt.Fprintf(&b, "    - %s\n", permission)
	}
	fmt.Fprintf(&b, "data:\n")
	fmt.Fprintf(&b, "  namespace: %s\n  migration_version: %s\n  migration_directory: %s\n  uninstall_policy: %s\n  rollback_policy: %s\n", spec.Plugin.DataNamespace, spec.Plugin.Version, spec.Plugin.MigrationDirectory, spec.Plugin.UninstallPolicy, spec.Plugin.RollbackPolicy)
	fmt.Fprintf(&b, "  tables:\n    - name: %s\n      description: %s\n      primary_key: %s\n      columns: %s\n", spec.Table.Name, spec.Table.Comment, primaryColumn(spec), strings.Join(fieldColumns(spec), ", "))
	if len(spec.Indexes) > 0 {
		fmt.Fprintf(&b, "      indexes: %s\n", pluginIndexList(spec))
	}
	fmt.Fprintf(&b, "api:\n  routes:\n")
	for _, route := range pluginAPIRoutes(spec) {
		fmt.Fprintf(&b, "    - method: %s\n      path: %s\n      summary: %s\n      permission: %s\n      audit_action: %s\n", route.method, route.path, route.summary, route.permission, route.auditAction)
	}
	if len(spec.Plugin.EventSubscriptions) > 0 {
		fmt.Fprintf(&b, "events:\n  subscriptions:\n")
		for _, subscription := range spec.Plugin.EventSubscriptions {
			fmt.Fprintf(&b, "    - name: %s\n      handler: %s\n      retry_policy: %s\n", subscription.Name, subscription.Handler, subscription.RetryPolicy)
		}
	}
	return b.String()
}

type pluginRoute struct {
	method      string
	path        string
	summary     string
	permission  string
	auditAction string
}

func pluginAPIRoutes(spec domaingenerator.GeneratorSpec) []pluginRoute {
	base := pluginAPIBasePath(spec)
	auditResource := pluginAuditResource(spec)
	if spec.Document != nil {
		return []pluginRoute{
			{method: "GET", path: base, summary: "Search " + spec.Table.CollectionName, permission: spec.Permissions.ReadKey, auditAction: auditResource + ".read"},
			{method: "GET", path: base + "/{id}", summary: "Get " + spec.Table.DomainName, permission: spec.Permissions.ReadKey, auditAction: auditResource + ".read"},
			{method: "POST", path: base, summary: "Create " + spec.Table.DomainName + " draft", permission: spec.Permissions.CreateKey, auditAction: auditResource + ".create"},
			{method: "POST", path: base + "/{id}/submit", summary: "Submit " + spec.Table.DomainName, permission: spec.Permissions.UpdateKey, auditAction: auditResource + ".submit"},
			{method: "POST", path: base + "/{id}/approve", summary: "Approve " + spec.Table.DomainName, permission: spec.Permissions.ManageKey, auditAction: auditResource + ".approve"},
			{method: "GET", path: base + "/export", summary: "Export " + spec.Table.CollectionName, permission: spec.Permissions.ManageKey, auditAction: auditResource + ".export"},
		}
	}
	return []pluginRoute{
		{method: "GET", path: base, summary: "List " + spec.Table.CollectionName, permission: spec.Permissions.ReadKey, auditAction: auditResource + ".read"},
		{method: "GET", path: base + "/{id}", summary: "Get " + spec.Table.DomainName, permission: spec.Permissions.ReadKey, auditAction: auditResource + ".read"},
		{method: "POST", path: base, summary: "Create " + spec.Table.DomainName, permission: spec.Permissions.CreateKey, auditAction: auditResource + ".create"},
		{method: "PUT", path: base + "/{id}", summary: "Update " + spec.Table.DomainName, permission: spec.Permissions.UpdateKey, auditAction: auditResource + ".update"},
		{method: "DELETE", path: base + "/{id}", summary: "Delete " + spec.Table.DomainName, permission: spec.Permissions.DeleteKey, auditAction: auditResource + ".delete"},
	}
}

func pluginAuditResource(spec domaingenerator.GeneratorSpec) string {
	return spec.Plugin.DataNamespace + "." + strings.TrimPrefix(spec.Audit.Resource, spec.Plugin.DataNamespace+".")
}

func pluginAPIBasePath(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf("/v1/plugins/%s/api/%s", spec.Plugin.ID, strings.TrimPrefix(spec.Menu.Path, "/"))
}

func pluginPermissionKeys(spec domaingenerator.GeneratorSpec) []string {
	return []string{spec.Permissions.ReadKey, spec.Permissions.CreateKey, spec.Permissions.UpdateKey, spec.Permissions.DeleteKey, spec.Permissions.ManageKey}
}

func permissionDisplayName(key string) string {
	return strings.Title(strings.ReplaceAll(strings.ReplaceAll(key, ".", " "), "_", " "))
}

func fieldColumns(spec domaingenerator.GeneratorSpec) []string {
	out := make([]string, 0, len(spec.Fields))
	for _, field := range spec.Fields {
		out = append(out, field.ColumnName)
	}
	return out
}

func primaryColumn(spec domaingenerator.GeneratorSpec) string {
	for _, field := range spec.Fields {
		if field.PrimaryKey {
			return field.ColumnName
		}
	}
	return "id"
}

func pluginIndexList(spec domaingenerator.GeneratorSpec) string {
	parts := make([]string, 0, len(spec.Indexes))
	for _, index := range spec.Indexes {
		prefix := ""
		if index.Unique {
			prefix = "unique "
		}
		columns := make([]string, 0, len(index.Fields))
		for _, fieldName := range index.Fields {
			if field, ok := spec.FieldByName(fieldName); ok {
				columns = append(columns, field.ColumnName)
			}
		}
		parts = append(parts, fmt.Sprintf("%s%s(%s)", prefix, index.Name, strings.Join(columns, ", ")))
	}
	return strings.Join(parts, "; ")
}

func renderPluginFrontendAPI(spec domaingenerator.GeneratorSpec) string {
	if spec.Document != nil {
		return renderDocumentPluginFrontendAPI(spec)
	}
	content := renderFrontendAPI(spec)
	return strings.Replace(content, fmt.Sprintf("const basePath = %q;", spec.Menu.Path), fmt.Sprintf("const basePath = %q;", pluginAPIBasePath(spec)), 1)
}

func renderPluginFrontendView(spec domaingenerator.GeneratorSpec) string {
	if spec.Document != nil {
		return renderDocumentPluginFrontendView(spec)
	}
	content := renderFrontendView(spec)
	content = strings.ReplaceAll(content, "generated-page", "plugin-generated-page")
	content = strings.ReplaceAll(content, "generated-toolbar", "plugin-generated-toolbar")
	content = strings.ReplaceAll(content, "generated-filters", "plugin-generated-filters")
	return content
}

func renderPluginFrontendRoute(spec domaingenerator.GeneratorSpec) string {
	content := renderFrontendRoute(spec)
	return strings.Replace(content, fmt.Sprintf("\t\tpath: %q,", spec.Menu.Path), fmt.Sprintf("\t\tpath: %q,", spec.Plugin.FrontendEntry), 1)
}

func renderPluginAcceptanceTest(spec domaingenerator.GeneratorSpec) string {
	pkg := strings.ReplaceAll(spec.Plugin.ID, "-", "_")
	return fmt.Sprintf(`package %s

import (
	"strings"
	"testing"
)

func TestGeneratedPluginManifestContract(t *testing.T) {
	required := []string{
		"id: %s",
		"data:",
		"namespace: %s",
		"name: %s",
		"uninstall_policy: %s",
		"rollback_policy: %s",
		"api:",
		"ui_menu:",
		"permission: %s",
		"audit_action: %s.create",
	}
	for _, value := range required {
		if !strings.Contains(generatedManifest, value) {
			t.Fatalf("manifest is missing %%q", value)
		}
	}
}

const generatedManifest = %q
`, pkg, spec.Plugin.ID, spec.Plugin.DataNamespace, spec.Table.Name, spec.Plugin.UninstallPolicy, spec.Plugin.RollbackPolicy, spec.Permissions.ReadKey, pluginAuditResource(spec), renderPluginManifest(spec))
}

func renderPluginREADME(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`# %s

Generated business plugin for %s.

## Generated Surface

- Manifest: permissions, menu, data manifest, API routes, audit actions, and event subscriptions.
- Migration: creates and drops the namespaced %s table under the plugin migration directory.
- Lifecycle: %s rollback and %s uninstall policies are declared in the current manifest.
- UI: API client, Pinia store, responsive list/form page, loading/error/save states, and permission-gated actions.
- Acceptance: plugin manifest contract test.
- Tooling: deterministic package, SHA-256 verification, package installation, and local development commands.

## Commands

`+"```powershell"+`
./plugin.ps1 package
./plugin.ps1 verify
./plugin.ps1 dev
`+"```"+`

`+"```bash"+`
sh ./plugin.sh package
sh ./plugin.sh verify
sh ./plugin.sh dev
`+"```"+`

Both `+"`dev`"+` and `+"`install`"+` verify and extract the same current package before the runtime loader installs it.
`, spec.Plugin.Name, spec.Table.CollectionName, spec.Table.Name, spec.Plugin.RollbackPolicy, spec.Plugin.UninstallPolicy)
}

func renderPluginPowerShellCommand(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`param(
    [ValidateSet("build", "package", "verify", "install", "dev")]
    [string]$Action = "package",
    [string]$DistDir = "",
    [string]$PluginsRoot = ""
)

$ErrorActionPreference = "Stop"
$DistDir = if ($DistDir) { $DistDir } else { Join-Path $PSScriptRoot "dist" }
$PluginsRoot = if ($PluginsRoot) { $PluginsRoot } else { Join-Path $PSScriptRoot ".skoll-dev" }
$RepoRoot = if ($env:SKOLL_REPO_ROOT) { (Resolve-Path $env:SKOLL_REPO_ROOT).Path } else { (Resolve-Path (Join-Path $PSScriptRoot "../../..")).Path }
$Artifact = Join-Path $DistDir "%s-%s.zip"
$Checksum = "$Artifact.sha256"
$Backend = Join-Path $PSScriptRoot "backend/bin/%s-server.exe"

function Build-Plugin {
    New-Item -ItemType Directory -Force (Split-Path $Backend) | Out-Null
    go build -o $Backend (Join-Path $PSScriptRoot "backend")
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    npm --prefix (Join-Path $PSScriptRoot "web") run build
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

function Invoke-SkollPlugin([string[]]$ToolArgs) {
    Push-Location $RepoRoot
    try {
        & go run ./cmd/skoll-plugin @ToolArgs
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }
}

switch ($Action) {
    "build" { Build-Plugin }
    "package" { Build-Plugin; Invoke-SkollPlugin @("package", $PSScriptRoot, $DistDir) }
    "verify" { Invoke-SkollPlugin @("verify-package", $Artifact, $Checksum) }
    "install" { Invoke-SkollPlugin @("install-package", $Artifact, $Checksum, $PluginsRoot) }
    "dev" { Build-Plugin; Invoke-SkollPlugin @("dev", $PSScriptRoot, $DistDir, $PluginsRoot) }
}
`, spec.Plugin.ID, spec.Plugin.Version, spec.Plugin.ID)
}

func renderPluginShellCommand(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`#!/usr/bin/env sh
set -eu

action="${1:-package}"
plugin_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$plugin_dir/../../.." && pwd)
dist_dir="${SKOLL_PLUGIN_DIST:-$plugin_dir/dist}"
plugins_root="${SKOLL_DEV_PLUGINS_ROOT:-$plugin_dir/.skoll-dev}"
artifact="$dist_dir/%s-%s.zip"
checksum="$artifact.sha256"
backend="$plugin_dir/backend/bin/%s-server"

build_plugin() {
  mkdir -p "$(dirname "$backend")"
  go build -o "$backend" "$plugin_dir/backend"
  npm --prefix "$plugin_dir/web" run build
}

case "$action" in
  build) build_plugin ;;
  package) build_plugin; (cd "$repo_root" && go run ./cmd/skoll-plugin package "$plugin_dir" "$dist_dir") ;;
  verify) (cd "$repo_root" && go run ./cmd/skoll-plugin verify-package "$artifact" "$checksum") ;;
  install) (cd "$repo_root" && go run ./cmd/skoll-plugin install-package "$artifact" "$checksum" "$plugins_root") ;;
  dev) build_plugin; (cd "$repo_root" && go run ./cmd/skoll-plugin dev "$plugin_dir" "$dist_dir" "$plugins_root") ;;
  *) echo "usage: $0 {build|package|verify|install|dev}" >&2; exit 2 ;;
esac
`, spec.Plugin.ID, spec.Plugin.Version, spec.Plugin.ID)
}

func renderService(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	pkg := spec.Module.Package
	alias := "domain" + pkg
	domainName := spec.Table.DomainName
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	fmt.Fprintf(&b, "import (\n\t\"context\"\n\n\t%s \"github.com/tinboxw/skoll/internal/domain/%s\"\n\t\"github.com/tinboxw/skoll/internal/domain/shared\"\n)\n\n", alias, pkg)
	fmt.Fprintf(&b, "type Service interface {\n")
	fmt.Fprintf(&b, "\tCreate(ctx context.Context, in CreateInput) (*%s.%s, error)\n", alias, domainName)
	fmt.Fprintf(&b, "\tUpdate(ctx context.Context, id shared.ID, in UpdateInput) (*%s.%s, error)\n", alias, domainName)
	fmt.Fprintf(&b, "\tGet(ctx context.Context, id shared.ID) (*%s.%s, error)\n", alias, domainName)
	fmt.Fprintf(&b, "\tList(ctx context.Context, in ListInput) ([]%s.%s, error)\n", alias, domainName)
	fmt.Fprintf(&b, "\tDelete(ctx context.Context, id shared.ID) error\n")
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "type CreateInput struct {\n")
	renderServiceInputFields(&b, spec, false)
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "type UpdateInput struct {\n")
	renderServiceInputFields(&b, spec, true)
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "type ListInput struct {\n\tKeyword string\n\tOffset int\n\tLimit int\n}\n")
	return b.String()
}

func renderServiceImpl(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	pkg := spec.Module.Package
	domainName := spec.Table.DomainName
	alias := "domain" + pkg
	repoAlias := pkg + "repo"
	primaryName := exportedName(primaryField(spec).Name)
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	fmt.Fprintf(&b, "import (\n\t\"context\"\n\t\"time\"\n\n")
	fmt.Fprintf(&b, "\t%s \"github.com/tinboxw/skoll/internal/domain/%s\"\n\t\"github.com/tinboxw/skoll/internal/domain/shared\"\n", alias, pkg)
	fmt.Fprintf(&b, "\t%s \"github.com/tinboxw/skoll/internal/repository/%s\"\n)\n\n", repoAlias, pkg)
	fmt.Fprintf(&b, "const (\n")
	for _, action := range generatedAuditActions(spec) {
		fmt.Fprintf(&b, "\tAuditAction%s%s = \"%s.%s\"\n", domainName, exportedName(action), spec.Audit.Resource, action)
	}
	fmt.Fprintf(&b, ")\n\n")
	fmt.Fprintf(&b, "type serviceImpl struct {\n\trepo %s.%sRepository\n\tauditFn func(ctx context.Context, action string, resourceID string) error\n}\n\n", repoAlias, domainName)
	fmt.Fprintf(&b, "func NewService(repo %s.%sRepository) Service {\n\treturn &serviceImpl{repo: repo}\n}\n\n", repoAlias, domainName)
	fmt.Fprintf(&b, "func (s *serviceImpl) Create(ctx context.Context, in CreateInput) (*%s.%s, error) {\n", alias, domainName)
	fmt.Fprintf(&b, "\tnow := time.Now().UTC()\n\titem, err := %s.New%s(%s.%sInput{\n", alias, domainName, alias, domainName)
	renderConstructorAssignments(&b, spec, "in", false)
	fmt.Fprintf(&b, "\t\tCreatedAt: now,\n\t\tUpdatedAt: now,\n\t})\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tif err := s.repo.Create(ctx, *item); err != nil {\n\t\treturn nil, err\n\t}\n\ts.audit(ctx, AuditAction%sCreate, item.%s.String())\n\treturn item, nil\n}\n\n", domainName, primaryName)
	fmt.Fprintf(&b, "func (s *serviceImpl) Update(ctx context.Context, id shared.ID, in UpdateInput) (*%s.%s, error) {\n", alias, domainName)
	fmt.Fprintf(&b, "\tcurrent, err := s.repo.Get(ctx, id)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tnow := time.Now().UTC()\n\titem, err := %s.New%s(%s.%sInput{\n", alias, domainName, alias, domainName)
	renderConstructorAssignments(&b, spec, "in", true)
	fmt.Fprintf(&b, "\t\tCreatedAt: current.Meta.CreatedAt,\n\t\tUpdatedAt: now,\n\t})\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tif err := s.repo.Update(ctx, *item); err != nil {\n\t\treturn nil, err\n\t}\n\ts.audit(ctx, AuditAction%sUpdate, item.%s.String())\n\treturn item, nil\n}\n\n", domainName, primaryName)
	fmt.Fprintf(&b, "func (s *serviceImpl) Get(ctx context.Context, id shared.ID) (*%s.%s, error) {\n\treturn s.repo.Get(ctx, id)\n}\n\n", alias, domainName)
	fmt.Fprintf(&b, "func (s *serviceImpl) List(ctx context.Context, in ListInput) ([]%s.%s, error) {\n\treturn s.repo.List(ctx, %s.ListFilter{Keyword: in.Keyword}, in.Offset, in.Limit)\n}\n\n", alias, domainName, repoAlias)
	fmt.Fprintf(&b, "func (s *serviceImpl) Delete(ctx context.Context, id shared.ID) error {\n\tif err := s.repo.Delete(ctx, id); err != nil {\n\t\treturn err\n\t}\n\ts.audit(ctx, AuditAction%sDelete, id.String())\n\treturn nil\n}\n\n", domainName)
	fmt.Fprintf(&b, "func (s *serviceImpl) audit(ctx context.Context, action string, resourceID string) {\n\tif s.auditFn != nil {\n\t\t_ = s.auditFn(ctx, action, resourceID)\n\t}\n}\n")
	return b.String()
}

func renderServiceTest(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	pkg := spec.Module.Package
	domainName := spec.Table.DomainName
	primary := primaryField(spec)
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	fmt.Fprintf(&b, "import (\n\t\"context\"\n\t\"testing\"\n")
	if hasFieldType(spec, domaingenerator.FieldTypeTime) {
		fmt.Fprintf(&b, "\t\"time\"\n")
	}
	fmt.Fprintf(&b, "\n\t\"github.com/tinboxw/skoll/internal/store/memory\"\n)\n\n")
	fmt.Fprintf(&b, "func TestServiceCRUD(t *testing.T) {\n")
	fmt.Fprintf(&b, "\tctx := context.Background()\n\tsvc := NewService(memory.New%sStore())\n", domainName)
	fmt.Fprintf(&b, "\tcreated, err := svc.Create(ctx, CreateInput{\n")
	renderSampleAssignments(&b, spec, "time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)", true)
	fmt.Fprintf(&b, "\t})\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n")
	fmt.Fprintf(&b, "\tgot, err := svc.Get(ctx, created.%s)\n", exportedName(primary.Name))
	fmt.Fprintf(&b, "\tif err != nil || got.%s != created.%s {\n\t\tt.Fatalf(\"Get() = %%+v, %%v\", got, err)\n\t}\n", exportedName(primary.Name), exportedName(primary.Name))
	fmt.Fprintf(&b, "\tupdated, err := svc.Update(ctx, created.%s, UpdateInput{\n", exportedName(primary.Name))
	renderSampleAssignments(&b, spec, "time.Date(2026, 1, 2, 3, 5, 5, 0, time.UTC)", false)
	fmt.Fprintf(&b, "\t})\n\tif err != nil || updated.%s != created.%s {\n\t\tt.Fatalf(\"Update() = %%+v, %%v\", updated, err)\n\t}\n", exportedName(primary.Name), exportedName(primary.Name))
	fmt.Fprintf(&b, "\titems, err := svc.List(ctx, ListInput{Limit: 10})\n\tif err != nil || len(items) != 1 {\n\t\tt.Fatalf(\"List() = %%+v, %%v\", items, err)\n\t}\n")
	fmt.Fprintf(&b, "\tif err := svc.Delete(ctx, created.%s); err != nil {\n\t\tt.Fatal(err)\n\t}\n", exportedName(primary.Name))
	fmt.Fprintf(&b, "\tif _, err := svc.Get(ctx, created.%s); err == nil {\n\t\tt.Fatal(\"expected deleted entity to be absent\")\n\t}\n}\n", exportedName(primary.Name))
	return b.String()
}

func renderHandler(spec domaingenerator.GeneratorSpec) string {
	content := `package {{PACKAGE}}

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/shared"
	apiv1 "github.com/tinboxw/skoll/internal/handler/http/v1"
	{{PACKAGE}}svc "github.com/tinboxw/skoll/internal/service/{{PACKAGE}}"
)

type RouteContract struct {
	Method      string
	Path        string
	Permission  string
	AuditAction string
}

var {{DOMAIN}}RouteContracts = []RouteContract{
	{Method: http.MethodGet, Path: "{{ROUTE}}", Permission: "{{READ_PERMISSION}}", AuditAction: "{{AUDIT_RESOURCE}}.read"},
	{Method: http.MethodGet, Path: "{{ROUTE}}/{id}", Permission: "{{READ_PERMISSION}}", AuditAction: "{{AUDIT_RESOURCE}}.read"},
	{Method: http.MethodPost, Path: "{{ROUTE}}", Permission: "{{CREATE_PERMISSION}}", AuditAction: "{{AUDIT_RESOURCE}}.create"},
	{Method: http.MethodPut, Path: "{{ROUTE}}/{id}", Permission: "{{UPDATE_PERMISSION}}", AuditAction: "{{AUDIT_RESOURCE}}.update"},
	{Method: http.MethodDelete, Path: "{{ROUTE}}/{id}", Permission: "{{DELETE_PERMISSION}}", AuditAction: "{{AUDIT_RESOURCE}}.delete"},
}

func RouteContracts() []RouteContract {
	return append([]RouteContract(nil), {{DOMAIN}}RouteContracts...)
}

type Handler struct {
	service {{PACKAGE}}svc.Service
}

func NewHandler(service {{PACKAGE}}svc.Service) *Handler {
	if service == nil {
		panic("{{PACKAGE}} service is required")
	}
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	if mux == nil {
		panic("{{PACKAGE}} route mux is required")
	}
	mux.HandleFunc("GET {{ROUTE}}", h.list)
	mux.HandleFunc("GET {{ROUTE}}/{id}", h.get)
	mux.HandleFunc("POST {{ROUTE}}", h.create)
	mux.HandleFunc("PUT {{ROUTE}}/{id}", h.update)
	mux.HandleFunc("DELETE {{ROUTE}}/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	in, err := listInput(r)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	items, err := h.service.List(r.Context(), in)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "offset": in.Offset, "limit": in.Limit})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), shared.ID(strings.TrimSpace(r.PathValue("id"))))
	if err != nil {
		apiv1.WriteError(w, http.StatusNotFound, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in {{PACKAGE}}svc.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Create(r.Context(), in)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusCreated, map[string]any{"item": item})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in {{PACKAGE}}svc.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Update(r.Context(), shared.ID(strings.TrimSpace(r.PathValue("id"))), in)
	if err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), shared.ID(strings.TrimSpace(r.PathValue("id")))); err != nil {
		apiv1.WriteError(w, http.StatusBadRequest, err)
		return
	}
	apiv1.WriteMessage(w, http.StatusOK, "ok", "ok")
}

func listInput(r *http.Request) ({{PACKAGE}}svc.ListInput, error) {
	query := r.URL.Query()
	offset, err := parseWindowValue(query.Get("offset"), 0, 0, 0)
	if err != nil {
		return {{PACKAGE}}svc.ListInput{}, fmt.Errorf("offset: %w", err)
	}
	limit, err := parseWindowValue(query.Get("limit"), 20, 1, 100)
	if err != nil {
		return {{PACKAGE}}svc.ListInput{}, fmt.Errorf("limit: %w", err)
	}
	return {{PACKAGE}}svc.ListInput{Keyword: strings.TrimSpace(query.Get("keyword")), Offset: offset, Limit: limit}, nil
}

func parseWindowValue(raw string, defaultValue, minimum, maximum int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || maximum > 0 && value > maximum {
		return 0, fmt.Errorf("must be an integer between %d and %d", minimum, maximum)
	}
	return value, nil
}
`
	return strings.NewReplacer(
		"{{PACKAGE}}", spec.Module.Package,
		"{{DOMAIN}}", spec.Table.DomainName,
		"{{ROUTE}}", spec.Menu.Path,
		"{{READ_PERMISSION}}", spec.Permissions.ReadKey,
		"{{CREATE_PERMISSION}}", spec.Permissions.CreateKey,
		"{{UPDATE_PERMISSION}}", spec.Permissions.UpdateKey,
		"{{DELETE_PERMISSION}}", spec.Permissions.DeleteKey,
		"{{AUDIT_RESOURCE}}", spec.Audit.Resource,
	).Replace(content)
}

func renderHandlerTest(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	pkg := spec.Module.Package
	domainName := spec.Table.DomainName
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	fmt.Fprintf(&b, "import (\n\t\"bytes\"\n\t\"encoding/json\"\n\t\"net/http\"\n\t\"net/http/httptest\"\n\t\"strings\"\n\t\"testing\"\n")
	if hasFieldType(spec, domaingenerator.FieldTypeTime) {
		fmt.Fprintf(&b, "\t\"time\"\n")
	}
	fmt.Fprintf(&b, "\n\t%sservice \"github.com/tinboxw/skoll/internal/service/%s\"\n", pkg, pkg)
	fmt.Fprintf(&b, "\t\"github.com/tinboxw/skoll/internal/store/memory\"\n)\n\n")
	fmt.Fprintf(&b, "func TestRouteContractsDeclarePermissionAndAudit(t *testing.T) {\n")
	fmt.Fprintf(&b, "\tcontracts := RouteContracts()\n\tif len(contracts) != 5 {\n\t\tt.Fatalf(\"route contracts = %%+v\", contracts)\n\t}\n")
	fmt.Fprintf(&b, "\tmutations := 0\n\tfor _, contract := range contracts {\n\t\tif strings.TrimSpace(contract.Permission) == \"\" {\n\t\t\tt.Fatalf(\"missing permission: %%+v\", contract)\n\t\t}\n\t\tswitch contract.Method {\n\t\tcase http.MethodPost, http.MethodPut, http.MethodDelete:\n\t\t\tmutations++\n\t\t\tif strings.TrimSpace(contract.AuditAction) == \"\" {\n\t\t\t\tt.Fatalf(\"missing mutation audit: %%+v\", contract)\n\t\t\t}\n\t\t}\n\t}\n\tif mutations != 3 {\n\t\tt.Fatalf(\"mutation contracts = %%d\", mutations)\n\t}\n}\n\n")
	fmt.Fprintf(&b, "func TestHandlerUsesCurrentResponseEnvelope(t *testing.T) {\n")
	fmt.Fprintf(&b, "\tmux := http.NewServeMux()\n\tNewHandler(%sservice.NewService(memory.New%sStore())).Register(mux)\n", pkg, domainName)
	fmt.Fprintf(&b, "\tpayload, err := json.Marshal(map[string]any{\n")
	renderJSONSampleAssignments(&b, spec)
	fmt.Fprintf(&b, "\t})\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n")
	fmt.Fprintf(&b, "\tcreate := httptest.NewRecorder()\n\tmux.ServeHTTP(create, httptest.NewRequest(http.MethodPost, %q, bytes.NewReader(payload)))\n", spec.Menu.Path)
	fmt.Fprintf(&b, "\tassertEnvelope(t, create, http.StatusCreated, \"ok\")\n")
	fmt.Fprintf(&b, "\tlist := httptest.NewRecorder()\n\tmux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, %q+\"?offset=0&limit=20\", nil))\n", spec.Menu.Path)
	fmt.Fprintf(&b, "\tassertEnvelope(t, list, http.StatusOK, \"ok\")\n")
	fmt.Fprintf(&b, "\tinvalid := httptest.NewRecorder()\n\tmux.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, %q+\"?offset=-1\", nil))\n", spec.Menu.Path)
	fmt.Fprintf(&b, "\tassertEnvelope(t, invalid, http.StatusBadRequest, \"error\")\n}\n\n")
	fmt.Fprintf(&b, "func assertEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {\n\tt.Helper()\n\tif recorder.Code != status || !strings.HasPrefix(recorder.Header().Get(\"Content-Type\"), \"application/json\") {\n\t\tt.Fatalf(\"response = %%d %%s %%s\", recorder.Code, recorder.Header().Get(\"Content-Type\"), recorder.Body.String())\n\t}\n\tvar body struct { Code string `json:\"code\"`; Message string `json:\"message\"`; Data any `json:\"data\"` }\n\tif err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Code != code || body.Message == \"\" {\n\t\tt.Fatalf(\"envelope = %%+v, %%v\", body, err)\n\t}\n}\n")
	return b.String()
}

func renderRouter(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`package httpserver

import (
	"net/http"

	%shttp "github.com/tinboxw/skoll/internal/handler/http/v1/%s"
	%ssvc "github.com/tinboxw/skoll/internal/service/%s"
)

func RegisterGenerated%sRoutes(mux *http.ServeMux, service %ssvc.Service) []%shttp.RouteContract {
	%shttp.NewHandler(service).Register(mux)
	return %shttp.RouteContracts()
}
`, spec.Module.Package, spec.Module.Package, spec.Module.Package, spec.Module.Package, spec.Table.DomainName, spec.Module.Package, spec.Module.Package, spec.Module.Package, spec.Module.Package)
}

func renderOpenAPI(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	route := spec.Menu.Path
	domainName := spec.Table.DomainName
	fmt.Fprintf(&b, "openapi: 3.0.3\npaths:\n")
	fmt.Fprintf(&b, "  %s:\n", route)
	renderOpenAPIOperation(&b, "get", "list"+domainName, spec.Module.Name, spec.Permissions.ReadKey, spec.Audit.Resource+".read", "200", "#/components/schemas/"+domainName+"ListResponse", "")
	renderOpenAPIOperation(&b, "post", "create"+domainName, spec.Module.Name, spec.Permissions.CreateKey, spec.Audit.Resource+".create", "201", "#/components/schemas/"+domainName+"ItemResponse", "#/components/schemas/"+domainName+"CreateInput")
	fmt.Fprintf(&b, "  %s/{id}:\n", route)
	fmt.Fprintf(&b, "    parameters:\n      - name: id\n        in: path\n        required: true\n        schema:\n          type: string\n")
	renderOpenAPIOperation(&b, "get", "get"+domainName, spec.Module.Name, spec.Permissions.ReadKey, spec.Audit.Resource+".read", "200", "#/components/schemas/"+domainName+"ItemResponse", "")
	renderOpenAPIOperation(&b, "put", "update"+domainName, spec.Module.Name, spec.Permissions.UpdateKey, spec.Audit.Resource+".update", "200", "#/components/schemas/"+domainName+"ItemResponse", "#/components/schemas/"+domainName+"UpdateInput")
	renderOpenAPIOperation(&b, "delete", "delete"+domainName, spec.Module.Name, spec.Permissions.DeleteKey, spec.Audit.Resource+".delete", "200", "#/components/schemas/MessageResponse", "")
	fmt.Fprintf(&b, "components:\n  schemas:\n")
	renderOpenAPIEntitySchema(&b, spec)
	renderOpenAPIInputSchema(&b, spec, domainName+"CreateInput", true)
	renderOpenAPIInputSchema(&b, spec, domainName+"UpdateInput", false)
	fmt.Fprintf(&b, "    %sItemResponse:\n      allOf:\n        - $ref: '#/components/schemas/ResponseEnvelope'\n        - type: object\n          properties:\n            data:\n              type: object\n              properties:\n                item:\n                  $ref: '#/components/schemas/%s'\n", domainName, domainName)
	fmt.Fprintf(&b, "    %sListResponse:\n      allOf:\n        - $ref: '#/components/schemas/ResponseEnvelope'\n        - type: object\n          properties:\n            data:\n              type: object\n              properties:\n                items:\n                  type: array\n                  items:\n                    $ref: '#/components/schemas/%s'\n                offset:\n                  type: integer\n                limit:\n                  type: integer\n", domainName, domainName)
	fmt.Fprintf(&b, "    ResponseEnvelope:\n      type: object\n      required: [code, message]\n      properties:\n        code:\n          type: string\n        message:\n          type: string\n        data: {}\n")
	fmt.Fprintf(&b, "    MessageResponse:\n      $ref: '#/components/schemas/ResponseEnvelope'\n")
	return b.String()
}

func renderOpenAPIOperation(b *bytes.Buffer, method, operationID, tag, permission, auditAction, status, responseRef, requestRef string) {
	fmt.Fprintf(b, "    %s:\n      operationId: %s\n      tags: [%s]\n      x-permission: %s\n      x-audit-action: %s\n", method, operationID, tag, permission, auditAction)
	if requestRef != "" {
		fmt.Fprintf(b, "      requestBody:\n        required: true\n        content:\n          application/json:\n            schema:\n              $ref: '%s'\n", requestRef)
	}
	fmt.Fprintf(b, "      responses:\n        %q:\n          description: %s\n          content:\n            application/json:\n              schema:\n                $ref: '%s'\n", status, httpStatusDescription(status), responseRef)
}

func renderOpenAPIEntitySchema(b *bytes.Buffer, spec domaingenerator.GeneratorSpec) {
	fmt.Fprintf(b, "    %s:\n      type: object\n      required:\n", spec.Table.DomainName)
	for _, field := range spec.Fields {
		if field.Required || field.PrimaryKey {
			fmt.Fprintf(b, "        - %s\n", field.Name)
		}
	}
	fmt.Fprintf(b, "      properties:\n")
	for _, field := range spec.Fields {
		renderOpenAPIField(b, field, 8)
	}
}

func renderOpenAPIInputSchema(b *bytes.Buffer, spec domaingenerator.GeneratorSpec, name string, includePrimary bool) {
	fmt.Fprintf(b, "    %s:\n      type: object\n      required:\n", name)
	for _, field := range spec.Fields {
		if field.PrimaryKey && !includePrimary {
			continue
		}
		if field.Required || field.PrimaryKey {
			fmt.Fprintf(b, "        - %s\n", field.Name)
		}
	}
	fmt.Fprintf(b, "      properties:\n")
	for _, field := range spec.Fields {
		if field.PrimaryKey && !includePrimary {
			continue
		}
		renderOpenAPIField(b, field, 8)
	}
}

func renderOpenAPIField(b *bytes.Buffer, field domaingenerator.FieldSpec, indent int) {
	spaces := strings.Repeat(" ", indent)
	fmt.Fprintf(b, "%s%s:\n", spaces, field.Name)
	switch field.Type {
	case domaingenerator.FieldTypeInt:
		fmt.Fprintf(b, "%s  type: integer\n", spaces)
	case domaingenerator.FieldTypeDecimal:
		fmt.Fprintf(b, "%s  type: number\n%s  format: double\n", spaces, spaces)
	case domaingenerator.FieldTypeBool:
		fmt.Fprintf(b, "%s  type: boolean\n", spaces)
	case domaingenerator.FieldTypeTime:
		fmt.Fprintf(b, "%s  type: string\n%s  format: date-time\n", spaces, spaces)
	case domaingenerator.FieldTypeJSON:
		fmt.Fprintf(b, "%s  type: object\n%s  additionalProperties: true\n", spaces, spaces)
	default:
		fmt.Fprintf(b, "%s  type: string\n", spaces)
	}
}

func httpStatusDescription(status string) string {
	switch status {
	case "201":
		return "Created"
	default:
		return "OK"
	}
}

func renderPermissionSeed(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	domainName := spec.Table.DomainName
	fmt.Fprintf(&b, "package bootstrap\n\n")
	fmt.Fprintf(&b, "import (\n\t\"context\"\n\t\"fmt\"\n\n")
	fmt.Fprintf(&b, "\tdomainmenu \"github.com/tinboxw/skoll/internal/domain/menu\"\n")
	fmt.Fprintf(&b, "\tdomainpermission \"github.com/tinboxw/skoll/internal/domain/permission\"\n")
	fmt.Fprintf(&b, "\tmenusvc \"github.com/tinboxw/skoll/internal/service/menu\"\n")
	fmt.Fprintf(&b, "\tpermissionsvc \"github.com/tinboxw/skoll/internal/service/permission\"\n)\n\n")
	fmt.Fprintf(&b, "var %sGeneratedPermissions = []permissionsvc.RegisterResourceInput{\n", domainName)
	permissions := []struct {
		key, action, risk string
	}{
		{spec.Permissions.ReadKey, "Read", "Low"},
		{spec.Permissions.CreateKey, "Create", "Medium"},
		{spec.Permissions.UpdateKey, "Update", "Medium"},
		{spec.Permissions.DeleteKey, "Delete", "High"},
		{spec.Permissions.ManageKey, "Manage", "High"},
	}
	for _, permission := range permissions {
		fmt.Fprintf(&b, "\t{Key: %q, Type: domainpermission.ResourceTypeAPI, Module: %q, Source: \"system\", Name: %q, Risk: domainpermission.RiskLevel%s},\n", permission.key, spec.Module.Package, permission.action+" "+spec.Table.CollectionName, permission.risk)
	}
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "func Register%sGeneratedCatalog(ctx context.Context, permissionService permissionsvc.Service, menuService menusvc.Service) error {\n", domainName)
	fmt.Fprintf(&b, "\tif permissionService == nil || menuService == nil {\n\t\treturn fmt.Errorf(\"generated %s catalog services are required\")\n\t}\n", spec.Module.Package)
	fmt.Fprintf(&b, "\tfor _, item := range %sGeneratedPermissions {\n\t\tif _, err := permissionService.RegisterResource(ctx, item); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n", domainName)
	fmt.Fprintf(&b, "\tnode, err := domainmenu.NewNode(\n")
	fmt.Fprintf(&b, "\t\tdomainmenu.NodeIdentity{Key: %q, ParentKey: %q, Source: \"system\"},\n", spec.Menu.Key, spec.Menu.ParentKey)
	fmt.Fprintf(&b, "\t\tdomainmenu.NodeView{Name: %q, Path: %q, Component: %q, Icon: %q},\n", spec.Page.Title, spec.Menu.Path, spec.Menu.Component, spec.Menu.Icon)
	fmt.Fprintf(&b, "\t\t%d,\n\t)\n\tif err != nil {\n\t\treturn err\n\t}\n", spec.Menu.Order)
	fmt.Fprintf(&b, "\tnode.RequiredPermissions = %#v\n", spec.Menu.RequiredPermissions)
	fmt.Fprintf(&b, "\t_, err = menuService.MergeNodes(ctx, menusvc.MergeNodesInput{Nodes: []domainmenu.MenuNode{node}})\n\treturn err\n}\n")
	return b.String()
}

func renderFrontendAPI(spec domaingenerator.GeneratorSpec) string {
	typeName := spec.Table.DomainName
	var b bytes.Buffer
	fmt.Fprintf(&b, "import { apiDelete, apiGet, apiPost, apiPut, type ApiResponse } from \"../utils/api\";\n\n")
	fmt.Fprintf(&b, "export type %s = {\n", typeName)
	fmt.Fprintf(&b, "\t[key: string]: unknown;\n")
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "\t%s: %s;\n", field.Name, tsFieldType(field))
	}
	fmt.Fprintf(&b, "};\n\n")
	fmt.Fprintf(&b, "export type %sListQuery = {\n\tkeyword?: string;\n\toffset?: number;\n\tlimit?: number;\n};\n\n", typeName)
	fmt.Fprintf(&b, "export type %sListPage = {\n\titems: %s[];\n\toffset: number;\n\tlimit: number;\n\thasMore: boolean;\n};\n\n", typeName, typeName)
	fmt.Fprintf(&b, "export type %sInput = Partial<Omit<%s, \"id\">>;\n\n", typeName, typeName)
	fmt.Fprintf(&b, "const basePath = %q;\n\n", spec.Menu.Path)
	fmt.Fprintf(&b, "export async function list%s(query: %sListQuery = {}): Promise<%sListPage> {\n", typeName, typeName, typeName)
	fmt.Fprintf(&b, "\tconst params = new URLSearchParams();\n\tif (query.keyword) params.set(\"keyword\", query.keyword);\n\tif (query.offset !== undefined) params.set(\"offset\", String(query.offset));\n\tif (query.limit !== undefined) params.set(\"limit\", String(query.limit));\n\tconst suffix = params.toString();\n\tconst resp = await apiGet<ApiResponse<{ items: %s[]; offset: number; limit: number }>>(`${basePath}${suffix ? `?${suffix}` : \"\"}`);\n\treturn { ...resp.data, hasMore: resp.data.items.length >= resp.data.limit };\n}\n\n", typeName)
	fmt.Fprintf(&b, "export async function get%s(id: string): Promise<%s> {\n\tconst resp = await apiGet<ApiResponse<{ item: %s }>>(`${basePath}/${encodeURIComponent(id)}`);\n\treturn resp.data.item;\n}\n\n", typeName, typeName, typeName)
	fmt.Fprintf(&b, "export async function create%s(input: %sInput): Promise<%s> {\n\tconst resp = await apiPost<ApiResponse<{ item: %s }>>(basePath, input);\n\treturn resp.data.item;\n}\n\n", typeName, typeName, typeName, typeName)
	fmt.Fprintf(&b, "export async function update%s(id: string, input: %sInput): Promise<%s> {\n\tconst resp = await apiPut<ApiResponse<{ item: %s }>>(`${basePath}/${encodeURIComponent(id)}`, input);\n\treturn resp.data.item;\n}\n\n", typeName, typeName, typeName, typeName)
	fmt.Fprintf(&b, "export async function delete%s(id: string): Promise<void> {\n\tawait apiDelete<ApiResponse<null>>(`${basePath}/${encodeURIComponent(id)}`);\n}\n", typeName)
	return b.String()
}

func renderFrontendStore(spec domaingenerator.GeneratorSpec) string {
	module := spec.Module.Package
	typeName := spec.Table.DomainName
	storeName := exportedName(module)
	var b bytes.Buffer
	fmt.Fprintf(&b, "import { defineStore } from \"pinia\";\n")
	fmt.Fprintf(&b, "import { create%s, delete%s, get%s, list%s, update%s, type %s, type %sInput, type %sListQuery } from \"../api/%s\";\n", typeName, typeName, typeName, typeName, typeName, typeName, typeName, typeName, module)
	fmt.Fprintf(&b, "import { toErrorMessage } from \"../utils/common\";\n\n")
	fmt.Fprintf(&b, "type LoadStatus = \"idle\" | \"loading\" | \"success\" | \"error\";\n\n")
	fmt.Fprintf(&b, "type %sState = {\n\titems: %s[];\n\tselected: %s | null;\n\tlistStatus: LoadStatus;\n\tdetailStatus: LoadStatus;\n\tmutationStatus: LoadStatus;\n\tlistError: string;\n\tdetailError: string;\n\tmutationError: string;\n\tlastQuery: %sListQuery;\n\toffset: number;\n\tlimit: number;\n\thasMore: boolean;\n};\n\n", typeName, typeName, typeName, typeName)
	fmt.Fprintf(&b, "export const use%sStore = defineStore(\"%s\", {\n", storeName, module)
	fmt.Fprintf(&b, "\tstate: (): %sState => ({\n\t\titems: [],\n\t\tselected: null,\n\t\tlistStatus: \"idle\",\n\t\tdetailStatus: \"idle\",\n\t\tmutationStatus: \"idle\",\n\t\tlistError: \"\",\n\t\tdetailError: \"\",\n\t\tmutationError: \"\",\n\t\tlastQuery: {},\n\t\toffset: 0,\n\t\tlimit: 20,\n\t\thasMore: false\n\t}),\n", typeName)
	fmt.Fprintf(&b, "\tgetters: {\n\t\tisLoading: (state): boolean => state.listStatus === \"loading\" || state.detailStatus === \"loading\" || state.mutationStatus === \"loading\"\n\t},\n")
	fmt.Fprintf(&b, "\tactions: {\n\t\tasync load(query: %sListQuery = {}): Promise<void> {\n\t\t\tthis.listStatus = \"loading\";\n\t\t\tthis.listError = \"\";\n\t\t\tthis.lastQuery = query;\n\t\t\ttry {\n\t\t\t\tconst page = await list%s(query);\n\t\t\t\tthis.items = page.items;\n\t\t\t\tthis.offset = page.offset;\n\t\t\t\tthis.limit = page.limit;\n\t\t\t\tthis.hasMore = page.hasMore;\n\t\t\t\tthis.listStatus = \"success\";\n\t\t\t} catch (error) {\n\t\t\t\tthis.listStatus = \"error\";\n\t\t\t\tthis.listError = toErrorMessage(error);\n\t\t\t\tthrow error;\n\t\t\t}\n\t\t},\n", typeName, typeName)
	fmt.Fprintf(&b, "\t\tasync retry(): Promise<void> {\n\t\t\tawait this.load(this.lastQuery);\n\t\t},\n")
	fmt.Fprintf(&b, "\t\tasync loadOne(id: string): Promise<%s> {\n\t\t\tthis.detailStatus = \"loading\";\n\t\t\tthis.detailError = \"\";\n\t\t\ttry {\n\t\t\t\tconst item = await get%s(id);\n\t\t\t\tthis.selected = item;\n\t\t\t\tthis.detailStatus = \"success\";\n\t\t\t\treturn item;\n\t\t\t} catch (error) {\n\t\t\t\tthis.detailStatus = \"error\";\n\t\t\t\tthis.detailError = toErrorMessage(error);\n\t\t\t\tthrow error;\n\t\t\t}\n\t\t},\n", typeName, typeName)
	fmt.Fprintf(&b, "\t\tasync create(input: %sInput): Promise<%s> {\n\t\t\treturn this.runMutation(() => create%s(input));\n\t\t},\n", typeName, typeName, typeName)
	fmt.Fprintf(&b, "\t\tasync update(id: string, input: %sInput): Promise<%s> {\n\t\t\treturn this.runMutation(() => update%s(id, input));\n\t\t},\n", typeName, typeName, typeName)
	fmt.Fprintf(&b, "\t\tasync remove(id: string): Promise<void> {\n\t\t\tthis.mutationStatus = \"loading\";\n\t\t\tthis.mutationError = \"\";\n\t\t\ttry {\n\t\t\t\tawait delete%s(id);\n\t\t\t\tthis.items = this.items.filter((item) => item.id !== id);\n\t\t\t\tif (this.selected?.id === id) this.selected = null;\n\t\t\t\tthis.mutationStatus = \"success\";\n\t\t\t} catch (error) {\n\t\t\t\tthis.mutationStatus = \"error\";\n\t\t\t\tthis.mutationError = toErrorMessage(error);\n\t\t\t\tthrow error;\n\t\t\t}\n\t\t},\n", typeName)
	fmt.Fprintf(&b, "\t\tasync runMutation(operation: () => Promise<%s>): Promise<%s> {\n\t\t\tthis.mutationStatus = \"loading\";\n\t\t\tthis.mutationError = \"\";\n\t\t\ttry {\n\t\t\t\tconst item = await operation();\n\t\t\t\tconst index = this.items.findIndex((current) => current.id === item.id);\n\t\t\t\tif (index >= 0) this.items.splice(index, 1, item); else this.items.unshift(item);\n\t\t\t\tthis.selected = item;\n\t\t\t\tthis.mutationStatus = \"success\";\n\t\t\t\treturn item;\n\t\t\t} catch (error) {\n\t\t\t\tthis.mutationStatus = \"error\";\n\t\t\t\tthis.mutationError = toErrorMessage(error);\n\t\t\t\tthrow error;\n\t\t\t}\n\t\t},\n", typeName, typeName)
	fmt.Fprintf(&b, "\t\tclearMutationState(): void {\n\t\t\tthis.mutationStatus = \"idle\";\n\t\t\tthis.mutationError = \"\";\n\t\t}\n\t}\n});\n")
	return b.String()
}

func renderFrontendLocale(spec domaingenerator.GeneratorSpec) string {
	module := spec.Module.Package
	name := spec.Table.DomainName
	prefix := "generated." + module
	var b bytes.Buffer
	fmt.Fprintf(&b, "export type %sLocale = \"zh-CN\" | \"en-US\";\n\n", name)
	fmt.Fprintf(&b, "const messages: Record<%sLocale, Record<string, string>> = {\n", name)
	for _, locale := range []string{"zh-CN", "en-US"} {
		fmt.Fprintf(&b, "\t%q: {\n", locale)
		labels := [][2]string{
			{"title", spec.Page.Title}, {"description", spec.Module.Description}, {"search", "Search"}, {"searchPlaceholder", "Search " + spec.Table.CollectionName},
			{"refresh", "Refresh"}, {"create", "Create"}, {"edit", "Edit"}, {"view", "View"}, {"delete", "Delete"}, {"save", "Save"}, {"cancel", "Cancel"},
			{"actions", "Actions"}, {"empty", "No " + spec.Table.CollectionName + " found"}, {"detail", spec.Table.DomainName + " details"},
			{"createTitle", "Create " + spec.Table.DomainName}, {"editTitle", "Edit " + spec.Table.DomainName}, {"saved", spec.Table.DomainName + " saved"},
			{"deleted", spec.Table.DomainName + " deleted"}, {"deleteTitle", "Delete " + spec.Table.DomainName}, {"deleteConfirm", "This action permanently deletes the selected record."},
			{"noPermission", "You do not have permission to view this page."}, {"retry", "Retry"}, {"previous", "Previous"}, {"next", "Next"},
		}
		if locale == "zh-CN" {
			labels = [][2]string{
				{"title", spec.Page.Title}, {"description", spec.Module.Description}, {"search", "搜索"}, {"searchPlaceholder", "搜索" + spec.Table.CollectionName},
				{"refresh", "刷新"}, {"create", "新建"}, {"edit", "编辑"}, {"view", "查看"}, {"delete", "删除"}, {"save", "保存"}, {"cancel", "取消"},
				{"actions", "操作"}, {"empty", "暂无" + spec.Table.CollectionName}, {"detail", spec.Table.DomainName + "详情"},
				{"createTitle", "新建" + spec.Table.DomainName}, {"editTitle", "编辑" + spec.Table.DomainName}, {"saved", spec.Table.DomainName + "已保存"},
				{"deleted", spec.Table.DomainName + "已删除"}, {"deleteTitle", "删除" + spec.Table.DomainName}, {"deleteConfirm", "此操作将永久删除所选记录。"},
				{"noPermission", "当前账号无权访问此页面。"}, {"retry", "重试"}, {"previous", "上一页"}, {"next", "下一页"},
			}
		}
		for _, label := range labels {
			fmt.Fprintf(&b, "\t\t%q: %q,\n", prefix+"."+label[0], label[1])
		}
		for _, field := range spec.Fields {
			fmt.Fprintf(&b, "\t\t%q: %q,\n", prefix+".field."+field.Name, field.Label)
			fmt.Fprintf(&b, "\t\t%q: %q,\n", prefix+".validation."+field.Name, field.Label+" is invalid")
		}
		fmt.Fprintf(&b, "\t},\n")
	}
	fmt.Fprintf(&b, "};\n\n")
	fmt.Fprintf(&b, "export function translate%s(locale: string, key: string): string {\n\tconst selected: %sLocale = locale === \"en-US\" ? \"en-US\" : \"zh-CN\";\n\treturn messages[selected][key] ?? key;\n}\n\n", name, name)
	fmt.Fprintf(&b, "export const %sMessages = messages;\n", module)
	return b.String()
}

func renderFrontendRoute(spec domaingenerator.GeneratorSpec) string {
	name := spec.Table.DomainName
	module := spec.Module.Package
	return fmt.Sprintf(`import type { RouteRecordRaw } from "vue-router";

const %sPage = () => import("../views/%s/index.vue");

export const %sRoutes: RouteRecordRaw[] = [
	{
		path: %q,
		name: %q,
		component: %sPage,
		meta: { requiresAuth: true, permissions: [%q] }
	}
];
`, name, name, module, spec.Menu.Path, spec.Page.RouteName, name, spec.Permissions.ReadKey)
}

func renderFrontendView(spec domaingenerator.GeneratorSpec) string {
	module := spec.Module.Package
	typeName := spec.Table.DomainName
	storeName := exportedName(module)
	prefix := "generated." + module
	var b bytes.Buffer
	fmt.Fprintf(&b, "<template>\n")
	fmt.Fprintf(&b, "\t<PageShell\n\t\tclass=\"generated-page\"\n\t\t:title=\"t('%s.title')\"\n\t\t:description=\"t('%s.description')\"\n\t\t:forbidden=\"!canRead\"\n\t\t:forbidden-description=\"t('%s.noPermission')\"\n\t\t:loading=\"initialLoading\"\n\t\t:error=\"initialError\"\n\t>\n", prefix, prefix, prefix)
	fmt.Fprintf(&b, "\t\t<template #actions>\n\t\t\t<el-button :icon=\"RefreshCw\" :loading=\"store.listStatus === 'loading'\" @click=\"loadPage(store.offset)\">{{ t('%s.refresh') }}</el-button>\n\t\t\t<el-button v-permission=\"createPermission\" type=\"primary\" :icon=\"Plus\" @click=\"openCreate\">{{ t('%s.create') }}</el-button>\n\t\t</template>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t\t<template #stateActions>\n\t\t\t<el-button v-if=\"canRead\" :icon=\"RefreshCw\" @click=\"loadPage(0)\">{{ t('%s.retry') }}</el-button>\n\t\t</template>\n", prefix)
	fmt.Fprintf(&b, "\t\t<FilterBar>\n\t\t\t<el-form-item :label=\"t('%s.search')\">\n\t\t\t\t<el-input v-model=\"filters.keyword\" clearable :placeholder=\"t('%s.searchPlaceholder')\" @keyup.enter=\"loadPage(0)\" />\n\t\t\t</el-form-item>\n\t\t\t<template #actions>\n\t\t\t\t<el-button type=\"primary\" :icon=\"Search\" :loading=\"store.listStatus === 'loading'\" @click=\"loadPage(0)\">{{ t('%s.search') }}</el-button>\n\t\t\t</template>\n\t\t</FilterBar>\n", prefix, prefix, prefix)
	fmt.Fprintf(&b, "\t\t<DataTable\n\t\t\t:rows=\"store.items\"\n\t\t\t:columns=\"columns\"\n\t\t\t:loading=\"store.listStatus === 'loading'\"\n\t\t\t:error=\"tableError\"\n\t\t\t:empty-text=\"t('%s.empty')\"\n\t\t\t:aria-label=\"t('%s.title')\"\n\t\t>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t<template #actions=\"{ row }\">\n\t\t\t\t<el-tooltip :content=\"t('%s.view')\">\n\t\t\t\t\t<el-button circle text :icon=\"Eye\" :aria-label=\"t('%s.view')\" @click=\"openDetail(row)\" />\n\t\t\t\t</el-tooltip>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t\t<el-tooltip :content=\"t('%s.edit')\">\n\t\t\t\t\t<el-button v-permission=\"updatePermission\" circle text type=\"primary\" :icon=\"Pencil\" :aria-label=\"t('%s.edit')\" @click=\"openEdit(row)\" />\n\t\t\t\t</el-tooltip>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t\t<ConfirmAction\n\t\t\t\t\tv-permission=\"deletePermission\"\n\t\t\t\t\t:label=\"t('%s.delete')\"\n\t\t\t\t\t:title=\"t('%s.deleteTitle')\"\n\t\t\t\t\t:message=\"t('%s.deleteConfirm')\"\n\t\t\t\t\t:loading=\"deletingId === String(row.id)\"\n\t\t\t\t\tsize=\"small\"\n\t\t\t\t\tplain\n\t\t\t\t\t@confirm=\"remove(String(row.id))\"\n\t\t\t\t/>\n\t\t\t</template>\n", prefix, prefix, prefix)
	fmt.Fprintf(&b, "\t\t\t<template #pagination>\n\t\t\t\t<div class=\"generated-pagination\">\n\t\t\t\t\t<el-button :disabled=\"store.offset === 0 || store.listStatus === 'loading'\" @click=\"loadPage(Math.max(0, store.offset - store.limit))\">{{ t('%s.previous') }}</el-button>\n\t\t\t\t\t<span>{{ store.offset + 1 }} - {{ store.offset + store.items.length }}</span>\n\t\t\t\t\t<el-button :disabled=\"!store.hasMore || store.listStatus === 'loading'\" @click=\"loadPage(store.offset + store.limit)\">{{ t('%s.next') }}</el-button>\n\t\t\t\t</div>\n\t\t\t</template>\n\t\t</DataTable>\n", prefix, prefix)
	fmt.Fprintf(&b, "\t</PageShell>\n\n")
	fmt.Fprintf(&b, "\t<DetailDrawer v-model=\"detailOpen\" :title=\"t('%s.detail')\" :loading=\"store.detailStatus === 'loading'\">\n\t\t<el-alert v-if=\"store.detailError\" type=\"error\" :title=\"store.detailError\" show-icon :closable=\"false\" />\n\t\t<el-descriptions v-else-if=\"store.selected\" :column=\"1\" border>\n", prefix)
	for _, fieldName := range spec.Page.List.Columns {
		if field, ok := spec.FieldByName(fieldName); ok {
			fmt.Fprintf(&b, "\t\t\t<el-descriptions-item :label=\"t('%s.field.%s')\">{{ displayValue(store.selected.%s) }}</el-descriptions-item>\n", prefix, field.Name, field.Name)
		}
	}
	fmt.Fprintf(&b, "\t\t</el-descriptions>\n\t</DetailDrawer>\n\n")
	fmt.Fprintf(&b, "\t<DetailDrawer v-model=\"formOpen\" :title=\"formTitle\" :loading=\"false\">\n\t\t<el-alert v-if=\"store.mutationError\" type=\"error\" :title=\"store.mutationError\" show-icon :closable=\"false\" />\n\t\t<el-form ref=\"formRef\" :model=\"form\" :rules=\"rules\" label-position=\"top\" @submit.prevent>\n")
	for _, fieldName := range spec.Page.Form.Fields {
		if field, ok := spec.FieldByName(fieldName); ok {
			renderGeneratedFormControl(&b, field, prefix)
		}
	}
	fmt.Fprintf(&b, "\t\t</el-form>\n\t\t<template #footer>\n\t\t\t<el-button @click=\"formOpen = false\">{{ t('%s.cancel') }}</el-button>\n\t\t\t<el-button type=\"primary\" :icon=\"Save\" :loading=\"store.mutationStatus === 'loading'\" @click=\"save\">{{ t('%s.save') }}</el-button>\n\t\t</template>\n\t</DetailDrawer>\n", prefix, prefix)
	fmt.Fprintf(&b, "</template>\n\n")
	fmt.Fprintf(&b, "<script setup lang=\"ts\">\n")
	fmt.Fprintf(&b, "import { computed, onMounted, reactive, ref } from \"vue\";\n")
	fmt.Fprintf(&b, "import { ElMessage, type FormInstance, type FormRules } from \"element-plus\";\n")
	fmt.Fprintf(&b, "import { Eye, Pencil, Plus, RefreshCw, Save, Search } from \"lucide-vue-next\";\n\n")
	fmt.Fprintf(&b, "import { ConfirmAction, DataTable, DetailDrawer, FilterBar, PageShell, type DataTableColumn } from \"../../components/Common\";\n")
	fmt.Fprintf(&b, "import { useI18n } from \"../../i18n\";\n")
	fmt.Fprintf(&b, "import { translate%s } from \"../../i18n/generated_%s\";\n", typeName, module)
	fmt.Fprintf(&b, "import { useButtonAccess } from \"../../permissions/button\";\n")
	fmt.Fprintf(&b, "import { use%sStore } from \"../../stores/%s\";\n", storeName, module)
	fmt.Fprintf(&b, "import type { %s, %sInput } from \"../../api/%s\";\n\n", typeName, typeName, module)
	fmt.Fprintf(&b, "const store = use%sStore();\n", storeName)
	fmt.Fprintf(&b, "const { locale } = useI18n();\nconst buttonAccess = useButtonAccess();\nconst t = (key: string): string => translate%s(locale.value, key);\n", typeName)
	fmt.Fprintf(&b, "const readPermission = %q;\nconst createPermission = %q;\nconst updatePermission = %q;\nconst deletePermission = %q;\n", spec.Permissions.ReadKey, spec.Permissions.CreateKey, spec.Permissions.UpdateKey, spec.Permissions.DeleteKey)
	fmt.Fprintf(&b, "const canRead = computed(() => buttonAccess.can(readPermission));\nconst initialLoading = computed(() => store.listStatus === \"loading\" && store.items.length === 0);\nconst initialError = computed(() => store.items.length === 0 ? store.listError : \"\");\nconst tableError = computed(() => store.items.length > 0 ? store.listError : \"\");\n")
	fmt.Fprintf(&b, "const filters = reactive({ keyword: \"\" });\nconst detailOpen = ref(false);\nconst formOpen = ref(false);\nconst editingId = ref<string | null>(null);\nconst deletingId = ref(\"\");\nconst formRef = ref<FormInstance>();\nconst form = reactive<%sInput>({});\n", typeName)
	fmt.Fprintf(&b, "const formTitle = computed(() => t(editingId.value ? %q : %q));\n", prefix+".editTitle", prefix+".createTitle")
	fmt.Fprintf(&b, "const columns: DataTableColumn[] = [\n")
	for _, fieldName := range spec.Page.List.Columns {
		if field, ok := spec.FieldByName(fieldName); ok {
			fmt.Fprintf(&b, "\t{ key: %q, label: t(%q), minWidth: 140 },\n", field.Name, prefix+".field."+field.Name)
		}
	}
	fmt.Fprintf(&b, "];\nconst rules: FormRules = {\n")
	for _, fieldName := range spec.Page.Form.Fields {
		if field, ok := spec.FieldByName(fieldName); ok && (field.Required || len(field.Validation) > 0) {
			fmt.Fprintf(&b, "\t%s: [", field.Name)
			if field.Required {
				fmt.Fprintf(&b, "{ required: true, message: t(%q), trigger: \"blur\" }", prefix+".validation."+field.Name)
			}
			for _, rule := range field.Validation {
				if rule.Type == domaingenerator.ValidationPattern && strings.TrimSpace(rule.Value) != "" {
					if field.Required {
						fmt.Fprintf(&b, ", ")
					}
					fmt.Fprintf(&b, "{ pattern: new RegExp(%q), message: t(%q), trigger: \"blur\" }", rule.Value, prefix+".validation."+field.Name)
				}
			}
			fmt.Fprintf(&b, "],\n")
		}
	}
	fmt.Fprintf(&b, "};\n\n")
	fmt.Fprintf(&b, "async function loadPage(offset: number): Promise<void> {\n\ttry {\n\t\tawait store.load({ keyword: filters.keyword.trim() || undefined, offset, limit: store.limit });\n\t} catch {\n\t\t// The shared page and table states render the store error.\n\t}\n}\n\n")
	fmt.Fprintf(&b, "async function openDetail(row: Record<string, unknown>): Promise<void> {\n\tdetailOpen.value = true;\n\ttry {\n\t\tawait store.loadOne(String(row.id));\n\t} catch {\n\t\t// The detail drawer renders the store error.\n\t}\n}\n\n")
	fmt.Fprintf(&b, "function openCreate(): void {\n\teditingId.value = null;\n\tresetForm();\n\tstore.clearMutationState();\n\tformOpen.value = true;\n}\n\n")
	fmt.Fprintf(&b, "function openEdit(row: Record<string, unknown>): void {\n\teditingId.value = String(row.id);\n\tresetForm();\n\tObject.assign(form, row);\n\tstore.clearMutationState();\n\tformOpen.value = true;\n}\n\n")
	fmt.Fprintf(&b, "async function save(): Promise<void> {\n\tif (!await formRef.value?.validate().catch(() => false)) return;\n\ttry {\n\t\tif (editingId.value) await store.update(editingId.value, { ...form }); else await store.create({ ...form });\n\t\tformOpen.value = false;\n\t\tElMessage.success(t(%q));\n\t} catch {\n\t\t// The form drawer keeps the backend error visible for correction.\n\t}\n}\n\n", prefix+".saved")
	fmt.Fprintf(&b, "async function remove(id: string): Promise<void> {\n\tdeletingId.value = id;\n\ttry {\n\t\tawait store.remove(id);\n\t\tElMessage.success(t(%q));\n\t} catch {\n\t\tElMessage.error(store.mutationError);\n\t} finally {\n\t\tdeletingId.value = \"\";\n\t}\n}\n\n", prefix+".deleted")
	fmt.Fprintf(&b, "function resetForm(): void {\n\tfor (const key of Object.keys(form)) delete form[key];\n")
	for _, fieldName := range spec.Page.Form.Fields {
		if field, ok := spec.FieldByName(fieldName); ok {
			fmt.Fprintf(&b, "\tform.%s = %s;\n", field.Name, tsFieldDefault(field))
		}
	}
	fmt.Fprintf(&b, "}\n\nfunction displayValue(value: unknown): string {\n\tif (value === null || value === undefined || value === \"\") return \"-\";\n\tif (typeof value === \"object\") return JSON.stringify(value);\n\treturn String(value);\n}\n\nonMounted(() => { if (canRead.value) void loadPage(0); });\n")
	fmt.Fprintf(&b, "</script>\n\n")
	fmt.Fprintf(&b, "<style scoped>\n.generated-pagination { display: flex; align-items: center; justify-content: flex-end; gap: 8px; min-height: 32px; }\n:deep(.el-form-item) { margin-bottom: 18px; }\n:deep(.el-input-number), :deep(.el-date-editor) { width: 100%%; }\n@media (max-width: 760px) { .generated-pagination { justify-content: space-between; width: 100%%; } }\n</style>\n")
	return b.String()
}

func renderGeneratedFormControl(b *bytes.Buffer, field domaingenerator.FieldSpec, prefix string) {
	fmt.Fprintf(b, "\t\t\t<el-form-item :label=\"t('%s.field.%s')\" prop=\"%s\">\n", prefix, field.Name, field.Name)
	switch field.Type {
	case domaingenerator.FieldTypeBool:
		fmt.Fprintf(b, "\t\t\t\t<el-switch v-model=\"form.%s\" />\n", field.Name)
	case domaingenerator.FieldTypeInt, domaingenerator.FieldTypeDecimal:
		fmt.Fprintf(b, "\t\t\t\t<el-input-number v-model=\"form.%s\" controls-position=\"right\" />\n", field.Name)
	case domaingenerator.FieldTypeTime:
		fmt.Fprintf(b, "\t\t\t\t<el-date-picker v-model=\"form.%s\" type=\"datetime\" value-format=\"YYYY-MM-DDTHH:mm:ss[Z]\" />\n", field.Name)
	case domaingenerator.FieldTypeText, domaingenerator.FieldTypeJSON:
		fmt.Fprintf(b, "\t\t\t\t<el-input v-model=\"form.%s\" type=\"textarea\" :rows=\"4\" />\n", field.Name)
	default:
		fmt.Fprintf(b, "\t\t\t\t<el-input v-model=\"form.%s\" />\n", field.Name)
	}
	fmt.Fprintf(b, "\t\t\t</el-form-item>\n")
}

func tsFieldDefault(field domaingenerator.FieldSpec) string {
	switch field.Type {
	case domaingenerator.FieldTypeBool:
		return "false"
	case domaingenerator.FieldTypeInt, domaingenerator.FieldTypeDecimal:
		return "0"
	case domaingenerator.FieldTypeJSON:
		return `"{}"`
	default:
		return `""`
	}
}

func renderServiceInputFields(b *bytes.Buffer, spec domaingenerator.GeneratorSpec, update bool) {
	for _, field := range spec.Fields {
		if update && field.PrimaryKey {
			continue
		}
		fmt.Fprintf(b, "\t%s %s\n", exportedName(field.Name), goFieldType(field))
	}
}

func renderConstructorAssignments(b *bytes.Buffer, spec domaingenerator.GeneratorSpec, inputName string, update bool) {
	for _, field := range spec.Fields {
		name := exportedName(field.Name)
		if update && field.PrimaryKey {
			fmt.Fprintf(b, "\t\t%s: id,\n", name)
			continue
		}
		fmt.Fprintf(b, "\t\t%s: %s.%s,\n", name, inputName, name)
	}
}

func renderSampleAssignments(b *bytes.Buffer, spec domaingenerator.GeneratorSpec, nowExpression string, includePrimary bool) {
	for _, field := range spec.Fields {
		if field.PrimaryKey && !includePrimary {
			continue
		}
		fmt.Fprintf(b, "\t\t%s: %s,\n", exportedName(field.Name), sampleFieldValue(field, nowExpression))
	}
}

func renderJSONSampleAssignments(b *bytes.Buffer, spec domaingenerator.GeneratorSpec) {
	for _, field := range spec.Fields {
		fmt.Fprintf(b, "\t\t%q: %s,\n", field.Name, sampleJSONValue(field))
	}
}

func sampleJSONValue(field domaingenerator.FieldSpec) string {
	switch field.Type {
	case domaingenerator.FieldTypeInt:
		return "7"
	case domaingenerator.FieldTypeDecimal:
		return "12.5"
	case domaingenerator.FieldTypeBool:
		return "true"
	case domaingenerator.FieldTypeTime:
		return `time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)`
	case domaingenerator.FieldTypeJSON:
		return `map[string]any{"key": "value"}`
	default:
		return fmt.Sprintf("%q", "generated-"+field.Name)
	}
}

func sampleFieldValue(field domaingenerator.FieldSpec, nowExpression string) string {
	switch field.Type {
	case domaingenerator.FieldTypeID:
		return fmt.Sprintf("%q", "generated-"+field.Name)
	case domaingenerator.FieldTypeString, domaingenerator.FieldTypeText:
		return fmt.Sprintf("%q", "sample "+field.Name)
	case domaingenerator.FieldTypeInt:
		return "7"
	case domaingenerator.FieldTypeDecimal:
		return "12.5"
	case domaingenerator.FieldTypeBool:
		return "true"
	case domaingenerator.FieldTypeTime:
		return nowExpression
	case domaingenerator.FieldTypeJSON:
		fieldType := goFieldType(field)
		switch fieldType {
		case "string":
			return `"{"key":"value"}"`
		case "map[string]any", "map[string]interface{}":
			return `map[string]any{"key": "value"}`
		default:
			if strings.HasPrefix(fieldType, "[]") || strings.HasPrefix(fieldType, "map[") {
				return fieldType + "{}"
			}
			return "*new(" + fieldType + ")"
		}
	default:
		return "*new(" + goFieldType(field) + ")"
	}
}

func zeroFieldValue(field domaingenerator.FieldSpec) string {
	switch field.Type {
	case domaingenerator.FieldTypeString, domaingenerator.FieldTypeText, domaingenerator.FieldTypeID:
		return `""`
	case domaingenerator.FieldTypeBool:
		return "false"
	case domaingenerator.FieldTypeTime:
		return "time.Time{}"
	case domaingenerator.FieldTypeJSON:
		if goFieldType(field) == "string" {
			return `""`
		}
		return "nil"
	default:
		return "0"
	}
}

func primaryField(spec domaingenerator.GeneratorSpec) domaingenerator.FieldSpec {
	for _, field := range spec.Fields {
		if field.PrimaryKey {
			return field
		}
	}
	return spec.Fields[0]
}

func hasFieldType(spec domaingenerator.GeneratorSpec, fieldType domaingenerator.FieldType) bool {
	for _, field := range spec.Fields {
		if field.Type == fieldType {
			return true
		}
	}
	return false
}

func generatedAuditActions(spec domaingenerator.GeneratorSpec) []string {
	actions := []string{"create", "update", "delete"}
	seen := map[string]struct{}{"create": {}, "update": {}, "delete": {}}
	for _, action := range spec.Audit.Actions {
		if _, ok := seen[action]; ok {
			continue
		}
		seen[action] = struct{}{}
		actions = append(actions, action)
	}
	return actions
}

func formatGoTemplate(source string) string {
	out, err := format.Source([]byte(source))
	if err != nil {
		return source
	}
	return string(out)
}

func exportedName(value string) string {
	parts := strings.Split(value, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		switch strings.ToLower(part) {
		case "api":
			parts[i] = "API"
		case "id":
			parts[i] = "ID"
		case "ip":
			parts[i] = "IP"
		case "url":
			parts[i] = "URL"
		default:
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func goFieldType(field domaingenerator.FieldSpec) string {
	if field.Type == domaingenerator.FieldTypeID {
		return "shared.ID"
	}
	if strings.TrimSpace(field.GoType) != "" {
		return strings.TrimSpace(field.GoType)
	}
	switch field.Type {
	case domaingenerator.FieldTypeInt:
		return "int"
	case domaingenerator.FieldTypeDecimal:
		return "float64"
	case domaingenerator.FieldTypeBool:
		return "bool"
	case domaingenerator.FieldTypeTime:
		return "time.Time"
	default:
		return "string"
	}
}

func gormFieldType(field domaingenerator.FieldSpec) string {
	if field.Type == domaingenerator.FieldTypeID {
		return "string"
	}
	return goFieldType(field)
}

func gormFieldTags(field domaingenerator.FieldSpec) string {
	var tags []string
	if field.PrimaryKey {
		tags = append(tags, "primaryKey")
	}
	if field.Unique {
		tags = append(tags, "unique")
	}
	if field.Required {
		tags = append(tags, "not null")
	}
	if len(tags) == 0 {
		return ""
	}
	return ";" + strings.Join(tags, ";")
}

func sqlFieldType(field domaingenerator.FieldSpec, dialect string) string {
	switch field.Type {
	case domaingenerator.FieldTypeInt:
		return "INTEGER"
	case domaingenerator.FieldTypeDecimal:
		return "DECIMAL(18,4)"
	case domaingenerator.FieldTypeBool:
		return "BOOLEAN"
	case domaingenerator.FieldTypeTime:
		return "TIMESTAMP"
	case domaingenerator.FieldTypeText, domaingenerator.FieldTypeJSON:
		return "TEXT"
	default:
		if dialect == "postgres" {
			return "VARCHAR(255)"
		}
		return "VARCHAR(255)"
	}
}

func sqlNullability(field domaingenerator.FieldSpec) string {
	if field.Required || field.PrimaryKey {
		return " NOT NULL"
	}
	return ""
}

func sqlFieldConstraints(field domaingenerator.FieldSpec) string {
	if field.Unique && !field.PrimaryKey {
		return " UNIQUE"
	}
	return ""
}

func tsFieldType(field domaingenerator.FieldSpec) string {
	if strings.TrimSpace(field.TypeScript) != "" {
		return strings.TrimSpace(field.TypeScript)
	}
	switch field.Type {
	case domaingenerator.FieldTypeInt, domaingenerator.FieldTypeDecimal:
		return "number"
	case domaingenerator.FieldTypeBool:
		return "boolean"
	case domaingenerator.FieldTypeJSON:
		return "Record<string, unknown>"
	default:
		return "string"
	}
}
