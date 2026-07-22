package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"gorm.io/gorm"
)

var migrationTableBindingPattern = regexp.MustCompile(`\{\{table:([a-z][a-z0-9_]{0,41})\}\}`)

type SchemaCandidate struct {
	Schema  PluginSchema
	Present bool
}

type Lifecycle struct {
	db       *gorm.DB
	registry *SchemaRegistry
}

type StorageSnapshot struct {
	PluginID       string
	Namespace      string
	Tables         []StorageTable
	TotalSizeBytes int64
	SizeKnown      bool
}

type StorageTable struct {
	LogicalName  string
	PhysicalName string
	Fields       []string
	PrimaryKey   []string
	IndexCount   int
	Exists       bool
	SizeBytes    int64
	SizeKnown    bool
}

func NewLifecycle(db *gorm.DB, registry *SchemaRegistry) (*Lifecycle, error) {
	if db == nil || registry == nil {
		return nil, errors.New("plugin datastore lifecycle requires database and schema registry")
	}
	return &Lifecycle{db: db, registry: registry}, nil
}

func (l *Lifecycle) Prepare(pluginID, pluginDir string) (SchemaCandidate, error) {
	if l == nil || l.registry == nil {
		return SchemaCandidate{}, errors.New("plugin datastore lifecycle is not configured")
	}
	schema, present, err := LoadSchemaManifest(pluginID, pluginDir)
	if err != nil {
		return SchemaCandidate{}, err
	}
	if !present {
		schema.PluginID = strings.TrimSpace(pluginID)
	}
	return SchemaCandidate{Schema: schema, Present: present}, nil
}

func (l *Lifecycle) Activate(candidate SchemaCandidate) error {
	if l == nil || l.registry == nil {
		return errors.New("plugin datastore lifecycle is not configured")
	}
	if !candidate.Present {
		if strings.TrimSpace(candidate.Schema.PluginID) != "" {
			l.registry.Unregister(candidate.Schema.PluginID)
		}
		return nil
	}
	_, err := l.registry.Replace(candidate.Schema)
	return err
}

func (l *Lifecycle) Snapshot(pluginID string) (RegisteredSchema, bool) {
	if l == nil || l.registry == nil {
		return RegisteredSchema{}, false
	}
	return l.registry.Snapshot(pluginID)
}

func (l *Lifecycle) InspectStorage(ctx context.Context, pluginID string) (StorageSnapshot, bool, error) {
	if l == nil || l.db == nil || l.registry == nil {
		return StorageSnapshot{}, false, errors.New("plugin datastore lifecycle is not configured")
	}
	registered, exists := l.registry.Snapshot(strings.TrimSpace(pluginID))
	if !exists {
		return StorageSnapshot{}, false, nil
	}
	storage, err := l.inspectRegisteredStorage(ctx, registered)
	return storage, true, err
}

func (l *Lifecycle) InspectCandidateStorage(ctx context.Context, candidate SchemaCandidate) (StorageSnapshot, bool, error) {
	if l == nil || l.db == nil {
		return StorageSnapshot{}, false, errors.New("plugin datastore lifecycle is not configured")
	}
	if !candidate.Present {
		return StorageSnapshot{}, false, nil
	}
	registered, err := buildRegisteredSchema(candidate.Schema)
	if err != nil {
		return StorageSnapshot{}, true, err
	}
	storage, err := l.inspectRegisteredStorage(ctx, registered)
	return storage, true, err
}

func (l *Lifecycle) inspectRegisteredStorage(ctx context.Context, registered RegisteredSchema) (StorageSnapshot, error) {
	dialect, err := lifecycleDialect(l.db)
	if err != nil {
		return StorageSnapshot{}, err
	}
	db := l.db.WithContext(normalizeLifecycleContext(ctx))
	out := StorageSnapshot{PluginID: registered.PluginID, Namespace: registered.Namespace, Tables: make([]StorageTable, 0, len(registered.Tables)), SizeKnown: true}
	for _, table := range registered.Tables {
		fields := make([]string, 0, len(table.Fields))
		for name := range table.Fields {
			fields = append(fields, name)
		}
		sort.Strings(fields)
		item := StorageTable{
			LogicalName: table.LogicalName, PhysicalName: table.PhysicalName, Fields: fields,
			PrimaryKey: append([]string(nil), table.PrimaryKey...), IndexCount: len(table.Indexes),
			Exists: db.Migrator().HasTable(table.PhysicalName),
		}
		if item.Exists {
			item.SizeBytes, item.SizeKnown = inspectTableSize(db, dialect, table.PhysicalName)
		} else {
			item.SizeKnown = true
		}
		if item.SizeKnown {
			out.TotalSizeBytes += item.SizeBytes
		} else {
			out.SizeKnown = false
		}
		out.Tables = append(out.Tables, item)
	}
	return out, nil
}

func inspectTableSize(db *gorm.DB, dialect SQLDialect, table string) (int64, bool) {
	var size sql.NullInt64
	var result *gorm.DB
	switch dialect {
	case DialectMySQL:
		result = db.Raw("SELECT COALESCE(data_length + index_length, 0) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", table).Scan(&size)
	case DialectPostgreSQL:
		result = db.Raw("SELECT COALESCE(pg_total_relation_size(to_regclass(?)), 0)", table).Scan(&size)
	case DialectSQLite:
		result = db.Raw("SELECT COALESCE(SUM(pgsize), 0) FROM dbstat WHERE name = ?", table).Scan(&size)
	default:
		return 0, false
	}
	if result.Error != nil || !size.Valid || size.Int64 < 0 {
		return 0, false
	}
	return size.Int64, true
}

func (l *Lifecycle) MigrationTransformer(candidate SchemaCandidate) (func(string) (string, error), error) {
	if !candidate.Present {
		return nil, nil
	}
	registered, err := buildRegisteredSchema(candidate.Schema)
	if err != nil {
		return nil, err
	}
	dialect, err := lifecycleDialect(l.db)
	if err != nil {
		return nil, err
	}
	bindings := make(map[string]string, len(registered.Tables))
	for _, table := range registered.Tables {
		bindings[table.LogicalName] = quoteSQLIdentifier(dialect, table.PhysicalName)
	}
	return func(sql string) (string, error) {
		var transformErr error
		expanded := migrationTableBindingPattern.ReplaceAllStringFunc(sql, func(token string) string {
			match := migrationTableBindingPattern.FindStringSubmatch(token)
			physical, exists := bindings[match[1]]
			if !exists {
				transformErr = fmt.Errorf("migration references undeclared logical table %s", match[1])
				return token
			}
			return physical
		})
		if transformErr != nil {
			return "", transformErr
		}
		if strings.Contains(expanded, "{{table:") {
			return "", errors.New("migration contains malformed table binding")
		}
		return expanded, nil
	}, nil
}

func (l *Lifecycle) ValidateStorage(ctx context.Context, candidate SchemaCandidate) error {
	if l == nil || l.db == nil {
		return errors.New("plugin datastore lifecycle is not configured")
	}
	if !candidate.Present {
		return nil
	}
	registered, err := buildRegisteredSchema(candidate.Schema)
	if err != nil {
		return err
	}
	db := l.db.WithContext(normalizeLifecycleContext(ctx))
	for _, table := range registered.Tables {
		if !db.Migrator().HasTable(table.PhysicalName) {
			return fmt.Errorf("plugin datastore table %s was not created by migrations", table.LogicalName)
		}
		columns, columnErr := db.Migrator().ColumnTypes(table.PhysicalName)
		if columnErr != nil {
			return fmt.Errorf("inspect plugin datastore table %s: %w", table.LogicalName, columnErr)
		}
		available := make(map[string]struct{}, len(columns))
		for _, column := range columns {
			available[strings.ToLower(strings.TrimSpace(column.Name()))] = struct{}{}
		}
		for field := range table.Fields {
			if _, exists := available[field]; !exists {
				return fmt.Errorf("plugin datastore table %s is missing field %s", table.LogicalName, field)
			}
		}
	}
	return nil
}

func (l *Lifecycle) Uninstall(ctx context.Context, pluginID string, policy plugin.DataUninstallPolicy) error {
	if l == nil || l.db == nil || l.registry == nil {
		return errors.New("plugin datastore lifecycle is not configured")
	}
	pluginID = strings.TrimSpace(pluginID)
	registered, exists := l.registry.Snapshot(pluginID)
	if policy != plugin.DataUninstallDrop {
		l.registry.Unregister(pluginID)
		return nil
	}
	if !exists {
		return nil
	}
	ctx = normalizeLifecycleContext(ctx)
	if err := l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, table := range registered.Tables {
			if err := tx.Migrator().DropTable(table.PhysicalName); err != nil {
				return fmt.Errorf("drop plugin datastore table %s: %w", table.LogicalName, err)
			}
		}
		return tx.Where("plugin_id = ?", pluginID).Delete(&gormrepo.PluginDataMutationModel{}).Error
	}); err != nil {
		return err
	}
	for _, table := range registered.Tables {
		if l.db.WithContext(ctx).Migrator().HasTable(table.PhysicalName) {
			return fmt.Errorf("plugin datastore table %s remains after drop uninstall", table.LogicalName)
		}
	}
	l.registry.Unregister(pluginID)
	return nil
}

func normalizeLifecycleContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func lifecycleDialect(db *gorm.DB) (SQLDialect, error) {
	if db == nil || db.Dialector == nil {
		return "", errors.New("plugin datastore database dialect is unavailable")
	}
	switch strings.ToLower(strings.TrimSpace(db.Dialector.Name())) {
	case "sqlite":
		return DialectSQLite, nil
	case "postgres":
		return DialectPostgreSQL, nil
	case "mysql":
		return DialectMySQL, nil
	default:
		return "", fmt.Errorf("unsupported plugin datastore database dialect %q", db.Dialector.Name())
	}
}
