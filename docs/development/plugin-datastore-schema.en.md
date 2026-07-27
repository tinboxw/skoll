# Plugin Datastore Schema And Lifecycle

Skoll plugins declare structured relational access in one optional file named `datastore.yaml` at the plugin package root. Its presence enables the datastore capability; absence means that the plugin declares no datastore schema. There is no legacy schema file, inferred column type, raw database handle, or alternate registration path.

## Schema File

```yaml
version: 1
tables:
  - name: records
    mutation_policy: mutable
    primary_key: [id]
    fields:
      - name: id
        type: string
        filterable: true
        sortable: true
      - name: amount
        type: decimal
        mutable: true
        filterable: true
      - name: approved_at
        type: timestamp
        nullable: true
        mutable: true
        sortable: true
    indexes:
      - fields: [amount]
```

The parser accepts exactly one YAML document, rejects unknown fields, and supports schema version `1` only. Every table must declare `mutation_policy: mutable` or `mutation_policy: append_only`; omission and unknown values fail installation. Append-only tables accept inserts and reject update, upsert, and delete before SQL execution or audit. Every table requires one to four immutable, non-null plugin fields as its primary key. Supported field types are `string`, `integer`, `decimal`, `boolean`, `timestamp`, `bytes`, and `json`. JSON and byte fields cannot be filterable, sortable, or indexed.

Skoll adds `tenant_id`, `organization_id`, `owner_id`, `version`, `created_at`, and `updated_at` to the registered field set. Migration DDL must create these host-managed columns, but the plugin must not repeat them in `datastore.yaml` or write them through `DataStoreService`.

## Migration Bindings

A datastore plugin must also declare migration metadata in `plugin.yaml`. Migration SQL references logical tables only:

```sql
CREATE TABLE {{table:records}} (
    id TEXT PRIMARY KEY,
    amount DECIMAL(18, 2) NOT NULL,
    approved_at TIMESTAMP NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

The host validates each `{{table:logical_name}}` binding against the candidate schema and expands it to a quoted, opaque `skp_<hash>_<table>` identifier for the configured SQL dialect. Unknown or malformed bindings fail before migration execution. Expanded SQL determines the migration checksum, so planning, execution, MySQL compensation, and rollback use the same physical statement.

## Lifecycle Rules

- Install parses and validates `datastore.yaml` before persisting usable plugin metadata.
- Enable and restart validate the candidate schema, apply pending migrations, inspect every required physical table and column, and then atomically publish the schema.
- Upgrade keeps the previous schema active until all migrations and storage checks succeed. Failed migrations leave the prior registry and records unchanged.
- Rollback is explicit, requires a disabled plugin and a positive migration limit, and removes the active schema registration until the current package is enabled and migrated again.
- `retain` and `archive` uninstall policies revoke schema access while preserving relational records. `drop` rolls migrations back, drops every registered physical table, deletes plugin idempotency records, verifies absence, and then unregisters the namespace.

Plugins never receive physical names, SQL connections, GORM objects, host credentials, or access to another plugin namespace.

## Independent Process Example

`plugins/datastore_e2e` is the current executable reference. Its backend imports only `pkg/pluginclient` and `pkg/pluginsdk`, starts from the environment provided by the managed-process launcher, and performs scoped queries and transactional mutations through `HostServices.DataStore`.

The always-run bootstrap E2E builds and packages this plugin from source, installs and enables it, rejects a forged tenant scope, verifies restart durability, performs an explicit rollback and clean recreation, and verifies that drop uninstall removes the physical namespace and idempotency rows.

## Verification

```powershell
go test ./internal/plugin/datastore ./internal/bootstrap -run "TestLoadSchemaManifest|TestPluginDataStore" -count=1
go test -race ./internal/plugin/datastore ./internal/bootstrap -run "TestSchemaRegistry|TestLoadSchemaManifest|TestPluginDataStore" -count=1
go test ./... -count=1
go vet ./...
```
