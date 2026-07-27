package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
)

type generatedPluginContract struct {
	SchemaVersion int                               `json:"schemaVersion"`
	Plugin        generatedPluginIdentity           `json:"plugin"`
	Data          generatedPluginDataContract       `json:"data"`
	API           generatedPluginAPIContract        `json:"api"`
	UI            generatedPluginUIContract         `json:"ui"`
	Permissions   generatedPluginPermissionContract `json:"permissions"`
	Events        generatedPluginEventContract      `json:"events"`
	Workflow      *generatedPluginWorkflowContract  `json:"workflow,omitempty"`
	Locales       []string                          `json:"locales"`
}

type generatedPluginIdentity struct {
	ID            string `json:"id"`
	DataNamespace string `json:"dataNamespace"`
	Version       string `json:"version"`
}

type generatedPluginDataContract struct {
	LogicalTable       string `json:"logicalTable"`
	PrimaryField       string `json:"primaryField"`
	MigrationDirectory string `json:"migrationDirectory"`
}

type generatedPluginAPIContract struct {
	BasePath string                         `json:"basePath"`
	Routes   []generatedPluginRouteContract `json:"routes"`
}

type generatedPluginRouteContract struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Permission  string `json:"permission"`
	AuditAction string `json:"auditAction"`
}

type generatedPluginUIContract struct {
	FrontendEntry       string   `json:"frontendEntry"`
	MenuKey             string   `json:"menuKey"`
	RouteName           string   `json:"routeName"`
	RequiredPermissions []string `json:"requiredPermissions"`
}

type generatedPluginPermissionContract struct {
	Resource  string `json:"resource"`
	ReadKey   string `json:"readKey"`
	CreateKey string `json:"createKey"`
	UpdateKey string `json:"updateKey"`
	DeleteKey string `json:"deleteKey"`
	ManageKey string `json:"manageKey"`
}

type generatedPluginEventContract struct {
	Publications  []generatedPluginEventPublication  `json:"publications"`
	Subscriptions []generatedPluginEventSubscription `json:"subscriptions"`
}

type generatedPluginEventPublication struct {
	Name          string `json:"name"`
	SchemaVersion uint32 `json:"schemaVersion"`
	PayloadType   string `json:"payloadType"`
	Scope         string `json:"scope"`
}

type generatedPluginEventSubscription struct {
	Publisher      string   `json:"publisher"`
	Name           string   `json:"name"`
	SchemaVersions []uint32 `json:"schemaVersions"`
	Handler        string   `json:"handler"`
	RetryPolicy    string   `json:"retryPolicy"`
}

type generatedPluginWorkflowContract struct {
	DefinitionID string `json:"definitionId"`
	SchemaKey    string `json:"schemaKey"`
	BusinessType string `json:"businessType"`
}

func buildGeneratedPluginContract(spec domaingenerator.GeneratorSpec) generatedPluginContract {
	contract := generatedPluginContract{
		SchemaVersion: 1,
		Plugin: generatedPluginIdentity{
			ID: spec.Plugin.ID, DataNamespace: spec.Plugin.DataNamespace, Version: spec.Plugin.Version,
		},
		Data: generatedPluginDataContract{
			LogicalTable: pluginLogicalTable(spec),
			PrimaryField: primaryField(spec).Name, MigrationDirectory: spec.Plugin.MigrationDirectory,
		},
		API: generatedPluginAPIContract{BasePath: pluginAPIBasePath(spec), Routes: generatedPluginRoutes(spec)},
		UI: generatedPluginUIContract{
			FrontendEntry: spec.Plugin.FrontendEntry, MenuKey: spec.Menu.Key, RouteName: spec.Page.RouteName,
			RequiredPermissions: append([]string(nil), spec.Menu.RequiredPermissions...),
		},
		Permissions: generatedPluginPermissionContract{
			Resource: spec.Permissions.Resource, ReadKey: spec.Permissions.ReadKey,
			CreateKey: spec.Permissions.CreateKey, UpdateKey: spec.Permissions.UpdateKey,
			DeleteKey: spec.Permissions.DeleteKey, ManageKey: spec.Permissions.ManageKey,
		},
		Events:  generatedPluginEvents(spec),
		Locales: []string{"zh-CN", "en-US"},
	}
	if spec.Document != nil {
		contract.Workflow = &generatedPluginWorkflowContract{
			DefinitionID: spec.Document.DefinitionID,
			SchemaKey:    spec.Document.SchemaKey,
			BusinessType: spec.Document.SchemaKey,
		}
	}
	return contract
}

func generatedPluginEvents(spec domaingenerator.GeneratorSpec) generatedPluginEventContract {
	contract := generatedPluginEventContract{
		Publications:  make([]generatedPluginEventPublication, 0, len(spec.Plugin.EventPublications)),
		Subscriptions: make([]generatedPluginEventSubscription, 0, len(spec.Plugin.EventSubscriptions)),
	}
	for _, publication := range spec.Plugin.EventPublications {
		contract.Publications = append(contract.Publications, generatedPluginEventPublication{
			Name: publication.Name, SchemaVersion: publication.SchemaVersion,
			PayloadType: publication.PayloadType, Scope: publication.Scope,
		})
	}
	for _, subscription := range spec.Plugin.EventSubscriptions {
		contract.Subscriptions = append(contract.Subscriptions, generatedPluginEventSubscription{
			Publisher: subscription.Publisher, Name: subscription.Name,
			SchemaVersions: append([]uint32(nil), subscription.SchemaVersions...),
			Handler:        subscription.Handler, RetryPolicy: subscription.RetryPolicy,
		})
	}
	return contract
}

func generatedPluginRoutes(spec domaingenerator.GeneratorSpec) []generatedPluginRouteContract {
	routes := pluginAPIRoutes(spec)
	out := make([]generatedPluginRouteContract, 0, len(routes))
	for _, route := range routes {
		out = append(out, generatedPluginRouteContract{
			Method: route.method, Path: route.path, Permission: route.permission, AuditAction: route.auditAction,
		})
	}
	return out
}

func renderPluginContractJSON(spec domaingenerator.GeneratorSpec) string {
	raw, _ := json.MarshalIndent(buildGeneratedPluginContract(spec), "", "  ")
	return string(raw) + "\n"
}

func renderPluginFrontendContract(spec domaingenerator.GeneratorSpec) string {
	return "export const pluginContract = " + strings.TrimSpace(renderPluginContractJSON(spec)) + " as const;\n"
}

func renderPluginBackendContract(spec domaingenerator.GeneratorSpec) string {
	contract := buildGeneratedPluginContract(spec)
	var b bytes.Buffer
	fmt.Fprintf(&b, "package main\n\n")
	fmt.Fprintf(&b, "import \"github.com/tinboxw/skoll/pkg/pluginsdk\"\n\n")
	fmt.Fprintf(&b, "const (\n")
	fmt.Fprintf(&b, "\tgeneratedPluginID = %q\n", contract.Plugin.ID)
	fmt.Fprintf(&b, "\tgeneratedAPIBasePath = %q\n", contract.API.BasePath)
	fmt.Fprintf(&b, "\tgeneratedLogicalTable = %q\n", contract.Data.LogicalTable)
	fmt.Fprintf(&b, "\tgeneratedPrimaryField = %q\n", contract.Data.PrimaryField)
	if contract.Workflow != nil {
		fmt.Fprintf(&b, "\tgeneratedWorkflowDefinitionID = %q\n", contract.Workflow.DefinitionID)
		fmt.Fprintf(&b, "\tgeneratedWorkflowBusinessType = %q\n", contract.Workflow.BusinessType)
	}
	fmt.Fprintf(&b, ")\n\n")
	fmt.Fprintf(&b, "var generatedFields = []string{%s}\n\n", quotedStrings(spec.FieldNames()))
	fmt.Fprintf(&b, "var generatedFieldTypes = map[string]pluginsdk.DataValueType{\n")
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "\t%q: pluginsdk.%s,\n", field.Name, pluginDataValueConstant(field.Type))
	}
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "var generatedPermissions = map[string]pluginsdk.Permission{\n")
	for _, permission := range []struct {
		action string
		key    string
	}{
		{action: "read", key: spec.Permissions.ReadKey},
		{action: "create", key: spec.Permissions.CreateKey},
		{action: "update", key: spec.Permissions.UpdateKey},
		{action: "delete", key: spec.Permissions.DeleteKey},
		{action: "manage", key: spec.Permissions.ManageKey},
	} {
		resource, permissionAction := permissionParts(permission.key)
		fmt.Fprintf(&b, "\t%q: {Resource: %q, Action: %q},\n", permission.action, resource, permissionAction)
	}
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "var generatedEventPublications = []pluginsdk.EventPublicationDeclaration{\n")
	for _, publication := range contract.Events.Publications {
		fmt.Fprintf(
			&b,
			"\t{Name: %q, SchemaVersion: %d, PayloadType: %q, Scope: pluginsdk.EventScopeMode(%q)},\n",
			publication.Name,
			publication.SchemaVersion,
			publication.PayloadType,
			publication.Scope,
		)
	}
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

func renderPluginDatastoreSchema(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "version: 1\ntables:\n")
	fmt.Fprintf(&b, "  - name: %s\n    mutation_policy: mutable\n", pluginLogicalTable(spec))
	fmt.Fprintf(&b, "    primary_key: [%s]\n    fields:\n", primaryField(spec).Name)
	for _, field := range spec.Fields {
		fmt.Fprintf(&b, "      - name: %s\n        type: %s\n", field.Name, pluginDataValueType(field.Type))
		if !field.PrimaryKey {
			fmt.Fprintf(&b, "        mutable: true\n")
		}
		if field.Filterable || field.PrimaryKey {
			fmt.Fprintf(&b, "        filterable: true\n")
		}
		if field.Sortable || field.PrimaryKey {
			fmt.Fprintf(&b, "        sortable: true\n")
		}
		if field.Type == domaingenerator.FieldTypeInt || field.Type == domaingenerator.FieldTypeDecimal {
			fmt.Fprintf(&b, "        aggregatable: true\n")
		}
		if field.Type != domaingenerator.FieldTypeJSON && field.Type != domaingenerator.FieldTypeText {
			fmt.Fprintf(&b, "        groupable: true\n")
		}
	}
	if len(spec.Indexes) > 0 {
		fmt.Fprintf(&b, "    indexes:\n")
		for _, index := range spec.Indexes {
			fmt.Fprintf(&b, "      - fields: [%s]\n", strings.Join(index.Fields, ", "))
			if index.Unique {
				fmt.Fprintf(&b, "        unique: true\n")
			}
		}
	}
	return b.String()
}

func renderPluginDatastoreMigration(spec domaingenerator.GeneratorSpec) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "CREATE TABLE {{table:%s}} (\n", pluginLogicalTable(spec))
	for index, field := range spec.Fields {
		suffix := ","
		if index == len(spec.Fields)-1 {
			suffix = ","
		}
		fmt.Fprintf(&b, "    %s %s%s%s\n", field.Name, pluginSQLType(field.Type), pluginSQLConstraints(field), suffix)
	}
	fmt.Fprintf(&b, "    tenant_id TEXT NOT NULL,\n")
	fmt.Fprintf(&b, "    organization_id TEXT NOT NULL,\n")
	fmt.Fprintf(&b, "    owner_id TEXT NOT NULL,\n")
	fmt.Fprintf(&b, "    version INTEGER NOT NULL,\n")
	fmt.Fprintf(&b, "    created_at DATETIME NOT NULL,\n")
	fmt.Fprintf(&b, "    updated_at DATETIME NOT NULL\n")
	fmt.Fprintf(&b, ");\n")
	for _, index := range spec.Indexes {
		unique := ""
		if index.Unique {
			unique = "UNIQUE "
		}
		fmt.Fprintf(&b, "CREATE %sINDEX %s ON {{table:%s}} (%s);\n", unique, index.Name, pluginLogicalTable(spec), strings.Join(index.Fields, ", "))
	}
	return b.String()
}

func renderPluginDatastoreMigrationDown(spec domaingenerator.GeneratorSpec) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS {{table:%s}};\n", pluginLogicalTable(spec))
}

func pluginLogicalTable(spec domaingenerator.GeneratorSpec) string {
	return spec.Table.Name
}

func pluginDataValueType(fieldType domaingenerator.FieldType) string {
	switch fieldType {
	case domaingenerator.FieldTypeInt:
		return "integer"
	case domaingenerator.FieldTypeDecimal:
		return "decimal"
	case domaingenerator.FieldTypeBool:
		return "boolean"
	case domaingenerator.FieldTypeTime:
		return "timestamp"
	case domaingenerator.FieldTypeJSON:
		return "json"
	default:
		return "string"
	}
}

func pluginDataValueConstant(fieldType domaingenerator.FieldType) string {
	switch fieldType {
	case domaingenerator.FieldTypeInt:
		return "DataValueInteger"
	case domaingenerator.FieldTypeDecimal:
		return "DataValueDecimal"
	case domaingenerator.FieldTypeBool:
		return "DataValueBoolean"
	case domaingenerator.FieldTypeTime:
		return "DataValueTimestamp"
	case domaingenerator.FieldTypeJSON:
		return "DataValueJSON"
	default:
		return "DataValueString"
	}
}

func pluginSQLType(fieldType domaingenerator.FieldType) string {
	switch fieldType {
	case domaingenerator.FieldTypeInt:
		return "INTEGER"
	case domaingenerator.FieldTypeDecimal:
		return "NUMERIC"
	case domaingenerator.FieldTypeBool:
		return "BOOLEAN"
	case domaingenerator.FieldTypeTime:
		return "DATETIME"
	default:
		return "TEXT"
	}
}

func pluginSQLConstraints(field domaingenerator.FieldSpec) string {
	constraints := make([]string, 0, 3)
	if field.PrimaryKey {
		constraints = append(constraints, "PRIMARY KEY")
	}
	if field.Required && !field.PrimaryKey {
		constraints = append(constraints, "NOT NULL")
	}
	if field.Unique && !field.PrimaryKey {
		constraints = append(constraints, "UNIQUE")
	}
	if len(constraints) == 0 {
		return ""
	}
	return " " + strings.Join(constraints, " ")
}

func permissionParts(key string) (string, string) {
	index := strings.LastIndex(key, ".")
	if index <= 0 || index == len(key)-1 {
		return key, "access"
	}
	return key[:index], key[index+1:]
}

func quotedStrings(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, strconv.Quote(item))
	}
	return strings.Join(quoted, ", ")
}
