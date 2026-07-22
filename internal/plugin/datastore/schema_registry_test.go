package datastore

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func TestSchemaRegistryRegistersOwnedNamespaceAndAllowlist(t *testing.T) {
	registry := NewSchemaRegistry()
	registered, err := registry.Register(validPluginSchema("medical_oa"))
	if err != nil {
		t.Fatal(err)
	}
	if registered.Namespace == "" || len(registered.Namespace) != 20 || len(registered.Tables) != 1 {
		t.Fatalf("unexpected registration: %+v", registered)
	}
	table, err := registry.Resolve("medical_oa", "products")
	if err != nil {
		t.Fatal(err)
	}
	if table.PluginID != "medical_oa" || table.Namespace != registered.Namespace || table.PhysicalName != registered.Namespace+"_products" {
		t.Fatalf("unexpected table: %+v", table)
	}
	for _, field := range []string{FieldTenantID, FieldOrganizationID, FieldOwnerID, FieldVersion, FieldCreatedAt, FieldUpdatedAt, "id", "name", "status"} {
		if _, exists := table.Fields[field]; !exists {
			t.Fatalf("missing field %s: %+v", field, table.Fields)
		}
	}
	if !table.Fields[FieldTenantID].HostManaged || table.Fields["name"].HostManaged {
		t.Fatalf("host ownership not preserved: %+v", table.Fields)
	}
}

func TestPluginNamespaceIsDeterministicAndPhysicalNamesFitDatabaseLimits(t *testing.T) {
	first, err := PluginNamespace("medical_oa")
	if err != nil {
		t.Fatal(err)
	}
	second, err := PluginNamespace(" medical_oa ")
	if err != nil {
		t.Fatal(err)
	}
	other, err := PluginNamespace("equipment_maintenance")
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first == other || !strings.HasPrefix(first, "skp_") || len(first) != 20 {
		t.Fatalf("unexpected namespaces: first=%q second=%q other=%q", first, second, other)
	}

	schema := withTableName(validPluginSchema("medical_oa"), strings.Repeat("a", MaxLogicalTableLen))
	registered, err := NewSchemaRegistry().Register(schema)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(registered.Tables[0].PhysicalName); got != 63 {
		t.Fatalf("physical table length=%d want=63", got)
	}
}

func TestSchemaRegistryRejectsReservedAndUndeclaredIdentifiers(t *testing.T) {
	tests := []struct {
		name   string
		schema PluginSchema
		field  string
	}{
		{name: "reserved plugin", schema: validPluginSchema("system"), field: "pluginId"},
		{name: "reserved table", schema: withTableName(validPluginSchema("medical_oa"), "sk_users"), field: "tables[0].name"},
		{name: "long table", schema: withTableName(validPluginSchema("medical_oa"), strings.Repeat("a", MaxLogicalTableLen+1)), field: "tables[0].name"},
		{name: "reserved field", schema: withField(validPluginSchema("medical_oa"), FieldTenantID), field: "tables[0].fields[3].name"},
		{name: "duplicate field", schema: withField(validPluginSchema("medical_oa"), "name"), field: "tables[0].fields[3].name"},
		{name: "null storage", schema: withFieldType(validPluginSchema("medical_oa"), pluginsdk.DataValueNull), field: "tables[0].fields[1].type"},
		{name: "mutable key", schema: withMutablePrimaryKey(validPluginSchema("medical_oa")), field: "tables[0].primaryKey[0]"},
		{name: "unknown key", schema: withPrimaryKey(validPluginSchema("medical_oa"), "missing"), field: "tables[0].primaryKey[0]"},
		{name: "unqueryable index", schema: withUnqueryableIndex(validPluginSchema("medical_oa")), field: "tables[0].indexes[1].fields[0]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewSchemaRegistry().Register(test.schema)
			assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, test.field)
		})
	}

	registry := NewSchemaRegistry()
	if _, err := registry.Register(validPluginSchema("medical_oa")); err != nil {
		t.Fatal(err)
	}
	_, err := registry.Resolve("medical_oa", "customers")
	assertStoreError(t, err, pluginsdk.DataStoreErrorNotFound, "table")
}

func TestSchemaRegistryRejectsDuplicateAndCrossPluginNamespace(t *testing.T) {
	registry := NewSchemaRegistry()
	medical, err := registry.Register(validPluginSchema("medical_oa"))
	if err != nil {
		t.Fatal(err)
	}
	equipment, err := registry.Register(validPluginSchema("equipment_maintenance"))
	if err != nil {
		t.Fatal(err)
	}
	if medical.Namespace == equipment.Namespace {
		t.Fatal("plugin namespaces collided")
	}
	if _, err := registry.Register(validPluginSchema("medical_oa")); err == nil {
		t.Fatal("duplicate registration succeeded")
	} else {
		assertStoreError(t, err, pluginsdk.DataStoreErrorConflict, "pluginId")
	}
	_, err = registry.ResolveNamespace("medical_oa", equipment.Namespace)
	assertStoreError(t, err, pluginsdk.DataStoreErrorForbidden, "namespace")
	_, err = registry.ResolveNamespace("INVALID", medical.Namespace)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "pluginId")
	owned, err := registry.ResolveNamespace("medical_oa", medical.Namespace)
	if err != nil || owned.PluginID != "medical_oa" {
		t.Fatalf("owned namespace=%+v err=%v", owned, err)
	}
}

func TestSchemaRegistrySnapshotsAreImmutableAndUnregisterClosesAccess(t *testing.T) {
	registry := NewSchemaRegistry()
	registered, err := registry.Register(validPluginSchema("medical_oa"))
	if err != nil {
		t.Fatal(err)
	}
	registered.Tables[0].Fields["name"] = FieldSchema{Name: "tampered"}
	registered.Tables[0].PrimaryKey[0] = "tampered"

	snapshot, exists := registry.Snapshot("medical_oa")
	if !exists || snapshot.Tables[0].Fields["name"].Name != "name" || snapshot.Tables[0].PrimaryKey[0] != "id" {
		t.Fatalf("registry was mutated through returned value: %+v", snapshot)
	}
	snapshot.Tables[0].Indexes[0].Fields[0] = "tampered"
	second, _ := registry.Snapshot("medical_oa")
	if second.Tables[0].Indexes[0].Fields[0] != "status" {
		t.Fatalf("registry index was mutated: %+v", second.Tables[0].Indexes)
	}
	if !registry.Unregister("medical_oa") || registry.Unregister("medical_oa") {
		t.Fatal("unexpected unregister result")
	}
	_, err = registry.Resolve("medical_oa", "products")
	assertStoreError(t, err, pluginsdk.DataStoreErrorNotFound, "pluginId")
}

func TestSchemaRegistryConcurrentOwnership(t *testing.T) {
	registry := NewSchemaRegistry()
	const plugins = 32
	var wait sync.WaitGroup
	errorsByPlugin := make(chan error, plugins)
	for index := range plugins {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			pluginID := fmt.Sprintf("plugin_%02d", index)
			registered, err := registry.Register(validPluginSchema(pluginID))
			if err != nil {
				errorsByPlugin <- err
				return
			}
			if _, err = registry.Resolve(pluginID, "products"); err != nil {
				errorsByPlugin <- err
				return
			}
			if _, err = registry.ResolveNamespace(pluginID, registered.Namespace); err != nil {
				errorsByPlugin <- err
			}
		}(index)
	}
	wait.Wait()
	close(errorsByPlugin)
	for err := range errorsByPlugin {
		t.Fatal(err)
	}
}

func TestSchemaRegistryEnforcesCollectionLimits(t *testing.T) {
	schema := validPluginSchema("medical_oa")
	schema.Tables = make([]TableSchema, MaxTablesPerPlugin+1)
	_, err := NewSchemaRegistry().Register(schema)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "tables")

	schema = validPluginSchema("medical_oa")
	schema.Tables[0].Fields = make([]FieldSchema, MaxFieldsPerTable+1)
	_, err = NewSchemaRegistry().Register(schema)
	assertStoreError(t, err, pluginsdk.DataStoreErrorInvalidRequest, "tables[0].fields")
}

func validPluginSchema(pluginID string) PluginSchema {
	return PluginSchema{PluginID: pluginID, Tables: []TableSchema{{
		Name: "products",
		Fields: []FieldSchema{
			{Name: "id", Type: pluginsdk.DataValueString, Filterable: true, Sortable: true},
			{Name: "name", Type: pluginsdk.DataValueString, Mutable: true, Filterable: true, Sortable: true},
			{Name: "status", Type: pluginsdk.DataValueString, Mutable: true, Filterable: true, Sortable: true},
		},
		PrimaryKey: []string{"id"},
		Indexes:    []IndexSchema{{Fields: []string{"status", "name"}}},
	}}}
}

func withTableName(schema PluginSchema, name string) PluginSchema {
	schema.Tables[0].Name = name
	return schema
}

func withField(schema PluginSchema, name string) PluginSchema {
	schema.Tables[0].Fields = append(schema.Tables[0].Fields, FieldSchema{Name: name, Type: pluginsdk.DataValueString})
	return schema
}

func withFieldType(schema PluginSchema, kind pluginsdk.DataValueType) PluginSchema {
	schema.Tables[0].Fields[1].Type = kind
	return schema
}

func withMutablePrimaryKey(schema PluginSchema) PluginSchema {
	schema.Tables[0].Fields[0].Mutable = true
	return schema
}

func withPrimaryKey(schema PluginSchema, field string) PluginSchema {
	schema.Tables[0].PrimaryKey = []string{field}
	return schema
}

func withIndex(schema PluginSchema, index IndexSchema) PluginSchema {
	schema.Tables[0].Indexes = append(schema.Tables[0].Indexes, index)
	return schema
}

func withUnqueryableIndex(schema PluginSchema) PluginSchema {
	schema.Tables[0].Fields[0].Filterable = false
	schema.Tables[0].Fields[0].Sortable = false
	return withIndex(schema, IndexSchema{Fields: []string{"id"}})
}

func assertStoreError(t *testing.T, err error, code pluginsdk.DataStoreErrorCode, field string) {
	t.Helper()
	var target *pluginsdk.DataStoreError
	if !errors.As(err, &target) || target.Code != code || !strings.HasPrefix(target.Field, field) {
		t.Fatalf("error=%v want code=%s field=%s", err, code, field)
	}
}
