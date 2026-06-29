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
	case "backend.repository":
		return formatGoTemplate(renderRepository(spec))
	case "backend.store.memory":
		return formatGoTemplate(renderMemoryStore(spec))
	case "backend.store.gorm.model":
		return formatGoTemplate(renderGORMModel(spec))
	case "backend.store.gorm.repo":
		return formatGoTemplate(renderGORMStore(spec))
	case "backend.migration.mysql":
		return renderMigration(spec, "mysql")
	case "backend.migration.postgres":
		return renderMigration(spec, "postgres")
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
		fmt.Fprintf(&b, "\t%s %s\n", exportedName(field.Name), goFieldType(field))
	}
	fmt.Fprintf(&b, "\tMeta shared.AuditMeta\n")
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "type %sInput struct {\n", domainName)
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "\t%s %s\n", exportedName(field.Name), goFieldType(field))
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
	return fmt.Sprintf(`package memory

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
	if _, ok := s.items[item.ID]; ok {
		return fmt.Errorf("%s already exists")
	}
	s.items[item.ID] = item
	return nil
}

func (s *%s) Update(_ context.Context, item %s.%s) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[item.ID]; !ok {
		return fmt.Errorf("%s not found")
	}
	s.items[item.ID] = item
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
	sort.SliceStable(items, func(i, j int) bool { return fmt.Sprint(items[i].ID) < fmt.Sprint(items[j].ID) })
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
	fmt.Fprintf(&b, "\tif err := s.db.WithContext(ctx).First(&model, \"id = ?\", id.String()).Error; err != nil {\n\t\tif err == gorm.ErrRecordNotFound {\n\t\t\treturn nil, fmt.Errorf(\"%s not found\")\n\t\t}\n\t\treturn nil, err\n\t}\n", pkg)
	fmt.Fprintf(&b, "\titem := from%sModel(model)\n\treturn &item, nil\n}\n\n", domainName)
	fmt.Fprintf(&b, "func (s *%sStore) List(ctx context.Context, filter %s.ListFilter, offset int, limit int) ([]%s.%s, error) {\n", domainName, repoAlias, alias, domainName)
	fmt.Fprintf(&b, "\tvar models []%sModel\n\tquery := s.db.WithContext(ctx).Model(&%sModel{})\n", domainName, domainName)
	fmt.Fprintf(&b, "\tif filter.Keyword != \"\" {\n\t\tquery = query.Where(\"id LIKE ?\", \"%%\"+filter.Keyword+\"%%\")\n\t}\n\tif offset > 0 {\n\t\tquery = query.Offset(offset)\n\t}\n\tif limit > 0 {\n\t\tquery = query.Limit(limit)\n\t}\n\tif err := query.Find(&models).Error; err != nil {\n\t\treturn nil, err\n\t}\n")
	fmt.Fprintf(&b, "\titems := make([]%s.%s, 0, len(models))\n\tfor _, model := range models {\n\t\titems = append(items, from%sModel(model))\n\t}\n\treturn items, nil\n}\n\n", alias, domainName, domainName)
	fmt.Fprintf(&b, "func (s *%sStore) Delete(ctx context.Context, id shared.ID) error {\n\treturn s.db.WithContext(ctx).Delete(&%sModel{}, \"id = ?\", id.String()).Error\n}\n\n", domainName, domainName)
	fmt.Fprintf(&b, "func to%sModel(item %s.%s) %sModel {\n\treturn %sModel{}\n}\n\n", domainName, alias, domainName, domainName, domainName)
	fmt.Fprintf(&b, "func from%sModel(model %sModel) %s.%s {\n\titem, _ := %s.New%s(%s.%sInput{})\n\treturn *item\n}\n", domainName, domainName, alias, domainName, alias, domainName, alias, domainName)
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
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
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
