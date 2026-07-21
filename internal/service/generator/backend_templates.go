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
	case "frontend.view":
		return renderFrontendView(spec)
	case "plugin.manifest":
		return renderPluginManifest(spec)
	case "plugin.migration.up":
		return renderMigration(spec, "postgres")
	case "plugin.migration.down":
		return renderPluginMigrationDown(spec)
	case "plugin.frontend.api":
		return renderPluginFrontendAPI(spec)
	case "plugin.frontend.store":
		return renderFrontendStore(spec)
	case "plugin.frontend.view":
		return renderPluginFrontendView(spec)
	case "plugin.acceptance.test":
		return formatGoTemplate(renderPluginAcceptanceTest(spec))
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
	fmt.Fprintf(&b, "CREATE TABLE %s (\n", spec.Table.Name)
	for idx, field := range spec.Fields {
		separator := ","
		if idx == len(spec.Fields)-1 {
			separator = ""
		}
		fmt.Fprintf(&b, "  %s %s%s%s\n", field.ColumnName, sqlFieldType(field, dialect), sqlNullability(field), separator)
	}
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
	fmt.Fprintf(&b, "  namespace: %s\n  migration_version: %s\n  migration_directory: %s\n  uninstall_policy: retain\n  rollback_policy: manual\n", spec.Plugin.DataNamespace, spec.Plugin.Version, spec.Plugin.MigrationDirectory)
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
	return []pluginRoute{
		{method: "GET", path: base, summary: "List " + spec.Table.CollectionName, permission: spec.Permissions.ReadKey, auditAction: spec.Audit.Resource + ".read"},
		{method: "POST", path: base, summary: "Create " + spec.Table.DomainName, permission: spec.Permissions.CreateKey, auditAction: spec.Audit.Resource + ".create"},
		{method: "PUT", path: base + "/{id}", summary: "Update " + spec.Table.DomainName, permission: spec.Permissions.UpdateKey, auditAction: spec.Audit.Resource + ".update"},
		{method: "DELETE", path: base + "/{id}", summary: "Delete " + spec.Table.DomainName, permission: spec.Permissions.DeleteKey, auditAction: spec.Audit.Resource + ".delete"},
	}
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
	content := renderFrontendAPI(spec)
	return strings.Replace(content, fmt.Sprintf("const basePath = %q;", spec.Menu.Path), fmt.Sprintf("const basePath = %q;", pluginAPIBasePath(spec)), 1)
}

func renderPluginFrontendView(spec domaingenerator.GeneratorSpec) string {
	content := renderFrontendView(spec)
	content = strings.ReplaceAll(content, "generated-page", "plugin-generated-page")
	content = strings.ReplaceAll(content, "generated-toolbar", "plugin-generated-toolbar")
	content = strings.ReplaceAll(content, "generated-filters", "plugin-generated-filters")
	return content
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
`, pkg, spec.Plugin.ID, spec.Permissions.ReadKey, spec.Audit.Resource, renderPluginManifest(spec))
}

func renderPluginREADME(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf(`# %s

Generated business plugin for %s.

## Generated Surface

- Manifest: permissions, menu, data manifest, API routes, audit actions, and event subscriptions.
- Migration: creates and drops %s under the plugin migration directory.
- UI: API client, Pinia store, responsive list/form page, loading/error/save states, and permission-gated actions.
- Acceptance: plugin manifest contract test.

## Verification

`+"```powershell"+`
go test ./internal/domain/generator/... ./internal/service/generator/...
cd web; npm run build
`+"```"+`
`, spec.Plugin.Name, spec.Table.CollectionName, spec.Table.Name)
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
	module := spec.Module.Package
	typeName := spec.Table.DomainName
	var b bytes.Buffer
	fmt.Fprintf(&b, "import { apiDelete, apiGet, apiPost, apiPut, type ApiResponse } from \"../utils/api\";\n\n")
	fmt.Fprintf(&b, "export type %s = {\n", typeName)
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "\t%s: %s;\n", field.Name, tsFieldType(field))
	}
	fmt.Fprintf(&b, "};\n\n")
	fmt.Fprintf(&b, "export type %sListQuery = {\n\tkeyword?: string;\n\toffset?: number;\n\tlimit?: number;\n};\n\n", typeName)
	fmt.Fprintf(&b, "export type %sInput = Partial<Omit<%s, \"id\">>;\n\n", typeName, typeName)
	fmt.Fprintf(&b, "const basePath = %q;\n\n", spec.Menu.Path)
	fmt.Fprintf(&b, "export async function list%s(query: %sListQuery = {}): Promise<%s[]> {\n", typeName, typeName, typeName)
	fmt.Fprintf(&b, "\tconst params = new URLSearchParams();\n\tif (query.keyword) params.set(\"keyword\", query.keyword);\n\tif (query.offset !== undefined) params.set(\"offset\", String(query.offset));\n\tif (query.limit !== undefined) params.set(\"limit\", String(query.limit));\n\tconst suffix = params.toString();\n\tconst resp = await apiGet<ApiResponse<{ items: %s[]; offset: number; limit: number }>>(`${basePath}${suffix ? `?${suffix}` : \"\"}`);\n\treturn resp.data.items;\n}\n\n", typeName)
	fmt.Fprintf(&b, "export async function create%s(input: %sInput): Promise<%s> {\n\tconst resp = await apiPost<ApiResponse<{ item: %s }>>(basePath, input);\n\treturn resp.data.item;\n}\n\n", typeName, typeName, typeName, typeName)
	fmt.Fprintf(&b, "export async function update%s(id: string, input: %sInput): Promise<%s> {\n\tconst resp = await apiPut<ApiResponse<{ item: %s }>>(`${basePath}/${id}`, input);\n\treturn resp.data.item;\n}\n\n", typeName, typeName, typeName, typeName)
	fmt.Fprintf(&b, "export async function delete%s(id: string): Promise<void> {\n\tawait apiDelete<ApiResponse<null>>(`${basePath}/${id}`);\n}\n", typeName)
	_ = module
	return b.String()
}

func renderFrontendStore(spec domaingenerator.GeneratorSpec) string {
	module := spec.Module.Package
	typeName := spec.Table.DomainName
	storeName := exportedName(module)
	var b bytes.Buffer
	fmt.Fprintf(&b, "import { defineStore } from \"pinia\";\n")
	fmt.Fprintf(&b, "import { create%s, delete%s, list%s, update%s, type %s, type %sInput, type %sListQuery } from \"../api/%s\";\n\n", typeName, typeName, typeName, typeName, typeName, typeName, typeName, module)
	fmt.Fprintf(&b, "type LoadStatus = \"idle\" | \"loading\" | \"success\" | \"error\";\n\n")
	fmt.Fprintf(&b, "type %sState = {\n\titems: %s[];\n\tlistStatus: LoadStatus;\n\tmutationStatus: LoadStatus;\n\tlastError: string | null;\n\tlastQuery: %sListQuery;\n};\n\n", typeName, typeName, typeName)
	fmt.Fprintf(&b, "export const use%sStore = defineStore(\"%s\", {\n", storeName, module)
	fmt.Fprintf(&b, "\tstate: (): %sState => ({\n\t\titems: [],\n\t\tlistStatus: \"idle\",\n\t\tmutationStatus: \"idle\",\n\t\tlastError: null,\n\t\tlastQuery: {}\n\t}),\n", typeName)
	fmt.Fprintf(&b, "\tgetters: {\n\t\tisLoading: (state): boolean => state.listStatus === \"loading\" || state.mutationStatus === \"loading\",\n\t\thasError: (state): boolean => state.listStatus === \"error\" || state.mutationStatus === \"error\"\n\t},\n")
	fmt.Fprintf(&b, "\tactions: {\n\t\tasync load(query: %sListQuery = {}): Promise<void> {\n\t\t\tthis.listStatus = \"loading\";\n\t\t\tthis.lastQuery = query;\n\t\t\ttry {\n\t\t\t\tthis.items = await list%s(query);\n\t\t\t\tthis.listStatus = \"success\";\n\t\t\t\tthis.lastError = null;\n\t\t\t} catch (error) {\n\t\t\t\tthis.listStatus = \"error\";\n\t\t\t\tthis.lastError = toErrorMessage(error);\n\t\t\t\tthrow error;\n\t\t\t}\n\t\t},\n", typeName, typeName)
	fmt.Fprintf(&b, "\t\tasync retry(): Promise<void> {\n\t\t\tawait this.load(this.lastQuery);\n\t\t},\n")
	fmt.Fprintf(&b, "\t\tasync create(input: %sInput): Promise<%s> {\n\t\t\tthis.mutationStatus = \"loading\";\n\t\t\ttry {\n\t\t\t\tconst item = await create%s(input);\n\t\t\t\tthis.items = [item, ...this.items];\n\t\t\t\tthis.mutationStatus = \"success\";\n\t\t\t\tthis.lastError = null;\n\t\t\t\treturn item;\n\t\t\t} catch (error) {\n\t\t\t\tthis.mutationStatus = \"error\";\n\t\t\t\tthis.lastError = toErrorMessage(error);\n\t\t\t\tthrow error;\n\t\t\t}\n\t\t},\n", typeName, typeName, typeName)
	fmt.Fprintf(&b, "\t\tasync update(id: string, input: %sInput): Promise<%s> {\n\t\t\tthis.mutationStatus = \"loading\";\n\t\t\ttry {\n\t\t\t\tconst item = await update%s(id, input);\n\t\t\t\tthis.items = this.items.map((current) => current.id === id ? item : current);\n\t\t\t\tthis.mutationStatus = \"success\";\n\t\t\t\tthis.lastError = null;\n\t\t\t\treturn item;\n\t\t\t} catch (error) {\n\t\t\t\tthis.mutationStatus = \"error\";\n\t\t\t\tthis.lastError = toErrorMessage(error);\n\t\t\t\tthrow error;\n\t\t\t}\n\t\t},\n", typeName, typeName, typeName)
	fmt.Fprintf(&b, "\t\tasync remove(id: string): Promise<void> {\n\t\t\tthis.mutationStatus = \"loading\";\n\t\t\ttry {\n\t\t\t\tawait delete%s(id);\n\t\t\t\tthis.items = this.items.filter((item) => item.id !== id);\n\t\t\t\tthis.mutationStatus = \"success\";\n\t\t\t\tthis.lastError = null;\n\t\t\t} catch (error) {\n\t\t\t\tthis.mutationStatus = \"error\";\n\t\t\t\tthis.lastError = toErrorMessage(error);\n\t\t\t\tthrow error;\n\t\t\t}\n\t\t}\n\t}\n});\n\n", typeName)
	fmt.Fprintf(&b, "function toErrorMessage(error: unknown): string {\n\treturn error instanceof Error && error.message.trim() !== \"\" ? error.message : \"%s_request_failed\";\n}\n", module)
	return b.String()
}

func renderFrontendView(spec domaingenerator.GeneratorSpec) string {
	module := spec.Module.Package
	typeName := spec.Table.DomainName
	storeName := exportedName(module)
	keywordField := "keyword"
	if len(spec.Page.List.Filters) > 0 {
		keywordField = spec.Page.List.Filters[0]
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "<template>\n")
	fmt.Fprintf(&b, "\t<section class=\"generated-page\">\n")
	fmt.Fprintf(&b, "\t\t<header class=\"generated-toolbar\">\n\t\t\t<h1>%s</h1>\n\t\t\t<el-button v-permission=\"createPermission\" type=\"primary\" :loading=\"store.mutationStatus === 'loading'\" @click=\"openCreate\">Create</el-button>\n\t\t</header>\n", spec.Page.Title)
	fmt.Fprintf(&b, "\t\t<div class=\"generated-filters\">\n\t\t\t<el-input v-model=\"filters.keyword\" clearable placeholder=\"Search\" @keyup.enter=\"load\" />\n\t\t\t<el-button :loading=\"store.listStatus === 'loading'\" @click=\"load\">Search</el-button>\n\t\t</div>\n")
	fmt.Fprintf(&b, "\t\t<el-alert v-if=\"store.hasError\" type=\"error\" :title=\"store.lastError || 'Load failed'\" show-icon />\n")
	fmt.Fprintf(&b, "\t\t<el-table v-loading=\"store.listStatus === 'loading'\" :data=\"store.items\" empty-text=\"No data\">\n")
	for _, fieldName := range spec.Page.List.Columns {
		if field, ok := spec.FieldByName(fieldName); ok {
			fmt.Fprintf(&b, "\t\t\t<el-table-column prop=\"%s\" label=\"%s\" />\n", field.Name, field.Label)
		}
	}
	fmt.Fprintf(&b, "\t\t\t<el-table-column fixed=\"right\" label=\"Actions\" width=\"180\">\n\t\t\t\t<template #default=\"{ row }\">\n\t\t\t\t\t<el-button v-permission=\"updatePermission\" text type=\"primary\" @click=\"openEdit(row)\">Edit</el-button>\n\t\t\t\t\t<el-button v-permission=\"deletePermission\" text type=\"danger\" @click=\"remove(row.id)\">Delete</el-button>\n\t\t\t\t</template>\n\t\t\t</el-table-column>\n")
	fmt.Fprintf(&b, "\t\t</el-table>\n")
	fmt.Fprintf(&b, "\t\t<el-drawer v-model=\"drawerOpen\" :title=\"editingId ? 'Edit' : 'Create'\" size=\"420px\">\n\t\t\t<el-form label-position=\"top\" @submit.prevent>\n")
	for _, fieldName := range spec.Page.Form.Fields {
		if field, ok := spec.FieldByName(fieldName); ok {
			renderGeneratedFormControl(&b, field)
		}
	}
	fmt.Fprintf(&b, "\t\t\t\t<el-button type=\"primary\" :loading=\"store.mutationStatus === 'loading'\" @click=\"save\">Save</el-button>\n\t\t\t</el-form>\n\t\t</el-drawer>\n")
	fmt.Fprintf(&b, "\t</section>\n</template>\n\n")
	fmt.Fprintf(&b, "<script setup lang=\"ts\">\n")
	fmt.Fprintf(&b, "import { onMounted, reactive, ref } from \"vue\";\n")
	fmt.Fprintf(&b, "import { use%sStore } from \"../../stores/%s\";\n", storeName, module)
	fmt.Fprintf(&b, "import type { %s, %sInput } from \"../../api/%s\";\n\n", typeName, typeName, module)
	fmt.Fprintf(&b, "const store = use%sStore();\n", storeName)
	fmt.Fprintf(&b, "const createPermission = %q;\nconst updatePermission = %q;\nconst deletePermission = %q;\n", spec.Permissions.CreateKey, spec.Permissions.UpdateKey, spec.Permissions.DeleteKey)
	fmt.Fprintf(&b, "const filters = reactive({ keyword: \"\" });\nconst drawerOpen = ref(false);\nconst editingId = ref<string | null>(null);\nconst form = reactive<%sInput>({});\n\n", typeName)
	fmt.Fprintf(&b, "async function load(): Promise<void> {\n\tawait store.load({ keyword: filters.keyword || undefined, offset: 0, limit: 20 });\n}\n\n")
	fmt.Fprintf(&b, "function openCreate(): void {\n\teditingId.value = null;\n\tresetForm();\n\tdrawerOpen.value = true;\n}\n\n")
	fmt.Fprintf(&b, "function openEdit(row: %s): void {\n\teditingId.value = row.id;\n\tObject.assign(form, row);\n\tdrawerOpen.value = true;\n}\n\n", typeName)
	fmt.Fprintf(&b, "async function save(): Promise<void> {\n\tif (editingId.value) {\n\t\tawait store.update(editingId.value, form);\n\t} else {\n\t\tawait store.create(form);\n\t}\n\tdrawerOpen.value = false;\n}\n\n")
	fmt.Fprintf(&b, "async function remove(id: string): Promise<void> {\n\tawait store.remove(id);\n}\n\n")
	fmt.Fprintf(&b, "function resetForm(): void {\n")
	for _, fieldName := range spec.Page.Form.Fields {
		if field, ok := spec.FieldByName(fieldName); ok {
			fmt.Fprintf(&b, "\tform.%s = undefined;\n", field.Name)
		}
	}
	fmt.Fprintf(&b, "}\n\nonMounted(load);\n")
	fmt.Fprintf(&b, "</script>\n\n")
	fmt.Fprintf(&b, "<style scoped>\n.generated-page { display: grid; gap: 16px; }\n.generated-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }\n.generated-toolbar h1 { margin: 0; font-size: 20px; font-weight: 650; }\n.generated-filters { display: flex; gap: 8px; max-width: 520px; }\n@media (max-width: 720px) { .generated-toolbar, .generated-filters { align-items: stretch; flex-direction: column; } }\n</style>\n")
	_ = keywordField
	return b.String()
}

func renderGeneratedFormControl(b *bytes.Buffer, field domaingenerator.FieldSpec) {
	fmt.Fprintf(b, "\t\t\t\t<el-form-item label=\"%s\">\n", field.Label)
	switch field.Type {
	case domaingenerator.FieldTypeBool:
		fmt.Fprintf(b, "\t\t\t\t\t<el-switch v-model=\"form.%s\" />\n", field.Name)
	case domaingenerator.FieldTypeInt, domaingenerator.FieldTypeDecimal:
		fmt.Fprintf(b, "\t\t\t\t\t<el-input-number v-model=\"form.%s\" :min=\"0\" controls-position=\"right\" />\n", field.Name)
	default:
		fmt.Fprintf(b, "\t\t\t\t\t<el-input v-model=\"form.%s\" />\n", field.Name)
	}
	fmt.Fprintf(b, "\t\t\t\t</el-form-item>\n")
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
