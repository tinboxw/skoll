# Skoll Business Plugin Foundation Acceptance Log

> Batch: `skoll-business-plugin-foundation-2026-07-22`
> Append evidence only after a Work Item reaches `Review` and passes acceptance.
> Failed acceptance must be recorded before the same Work Item returns to `Doing`.

## Entry Template

```text
## <Work Item ID> <Title>

- Date:
- Owner:
- Status flow:
- Scope:

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |

### Verification Commands

Result:

### Impact Review

- API/OpenAPI:
- Permission/audit:
- Migration/seed:
- Frontend/i18n:
- Documentation:
- Compatibility: none; current contracts only.

### Commit

`<work-item-id>: <short summary>`
```

## BF0-00 Establish The Official Business-Plugin Foundation Batch

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Convert the closed plugin-runtime capability gap into one official functionality/UI execution batch targeting complete medical OA delivery.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Batch files | Pass | Parent board, Work Item table, and acceptance log exist under `docs/refactor/current/` |
| Milestones | Pass | BF1-BF5 cover datastore, business documents, plugin UX, medical master data, and medical transaction/quality workflows |
| Atomic tasks | Pass | 39 Work Items carry priority, Skill, dependencies, deliverables, acceptance criteria, verification commands, and status |
| Direction | Pass | The critical path contains functionality, interaction, visual, and performance work; release and operations tasks are excluded |
| Medical OA scope | Pass | Employees, approvals, CRM, customers, suppliers, products, qualifications, purchase, sales, inventory, batches, quality, contracts, finance coordination, and reports are represented |
| Current-only rule | Pass | Batch rules reject compatibility modes, legacy formats, dual paths, transition adapters, and fallbacks |
| Collaboration | Pass | One-owner status flow, failed acceptance retry, acceptance evidence, and one-commit-per-item rules are explicit |
| Navigation | Pass | Both refactor indexes identify the active board, Work Item table, acceptance log, and latest closed batch |

### Verification Commands

```powershell
link/status/count checks
git diff --check
```

Result: BF0-00 passed. The next development path starts with the public scoped plugin datastore, then advances through reusable OA infrastructure and complete medical-industry workflows.

### Impact Review

- API/OpenAPI: no implementation change; BF1-01 owns the first contract change.
- Permission/audit: planned per vertical Work Item.
- Migration/seed: planned in BF1 and each medical module.
- Frontend/i18n: BF2, BF3, BF4, and BF5 contain explicit experience gates.
- Documentation: active indexes and official execution sources are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF0-00: establish business plugin foundation batch`

## BF1-01 Define The Current Public Plugin Datastore Contract

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Define one lossless structured datastore model for independent plugins without exposing SQL, GORM, host connections, internal packages, or a partially wired HostServices port.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Public service | Pass | `DataStoreService` exposes only context-bound `Query` and single-record `Mutate` operations |
| Lossless values | Pass | Null, string, canonical int64, decimal, boolean, RFC3339Nano timestamp, Base64 bytes, and JSON use explicit wire types and bounded validation |
| Structured query | Pass | Explicit table/fields, permission, scope narrowing, recursive filters, stable sort, cursor, and `1..200` limit are modeled and validated |
| Bounded complexity | Pass | Fields, sorts, filter depth/nodes/set values, cursor, scope IDs, mutation values, key width, payload bytes, and identifier forms have current hard limits |
| Trusted scope | Pass | `tenant_id`, `organization_id`, and `owner_id` are rejected from plugin keys, values, and ordinary filters; only `DataScopeIntent.Filter` can request narrowing |
| Mutation semantics | Pass | Insert/update/upsert/delete require keys and idempotency; values, returning fields, key overlap, reserved fields, and optimistic version rules are deterministic |
| Response contract | Pass | Records, pages, cursors, affected-row count, returned record, and versions have independent validation |
| Stable errors | Pass | Public errors carry inspectable code, field path, message, and retryability for invalid, forbidden, missing, conflict, limit, unsupported, and unavailable cases |
| Wire format | Pass | Permission and scope filter use tested camelCase JSON; map validation and overlap checks sort keys before reporting failures |
| Transaction boundary | Pass | The interface documents that `TransactionService` callback context carries the transaction; no transaction identifier or host handle is public |
| Current-only rule | Pass | No raw SQL escape hatch, compatibility API, dual wire shape, fallback store, or optional legacy behavior exists |
| Developer reference | Pass | Plugin guide, SDK reference, and dedicated datastore contract distinguish defined types from the BF1-05/BF1-06 runtime publication work |

### Verification Commands

```powershell
go test ./pkg/pluginsdk -count=1
go test -race ./pkg/pluginsdk -count=1
go test ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin -count=1
go test ./... -count=1
go vet ./...
codegraph impact Permission
codegraph impact ScopeFilter
relative Markdown link check
git diff --check
```

Result: BF1-01 passed without acceptance retries. The SDK now has one validated, lossless, scope-aware relational data contract ready for schema ownership and query-planning implementation.

### Impact Review

- API/OpenAPI: public Go SDK types and camelCase JSON tags were added; no host HTTP route or product OpenAPI endpoint is published in this item.
- Permission/audit: every query/mutation declares a permission and scope intent; enforcement and automatic mutation audit belong to BF1-03/BF1-04.
- Migration/seed: none; schema ownership and lifecycle begin in BF1-02/BF1-07.
- Frontend/i18n: none.
- Documentation: plugin guide, SDK reference, datastore contract, Work Item table, and acceptance evidence are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF1-01: define public plugin datastore contract`

## BF1-02 Register Plugin-Owned Schemas And Relational Namespaces

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Add the host-side schema ownership registry, deterministic relational namespace mapping, declared table and field allowlists, limits, isolation, and concurrency tests.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Namespace ownership | Pass | Every valid plugin ID maps deterministically to one opaque `skp_<hash>` namespace; cross-plugin resolution returns `forbidden` |
| Physical names | Pass | Logical names are validated before mapping and the maximum table name produces a 63-character physical identifier |
| Table allowlist | Pass | Only registered logical tables resolve; undeclared tables and unregistered plugins return `not_found` |
| Field ownership | Pass | Plugin fields are validated and merged with immutable host-managed scope, version, and timestamp fields |
| Reserved identifiers | Pass | Core, system database, host field, and reserved-prefix identifiers fail closed |
| Schema limits | Pass | Table, field, primary-key, index-count, and index-width limits are enforced before registration |
| Queryability | Pass | Indexes can reference only declared filterable or sortable fields; JSON and byte fields cannot become queryable |
| Concurrency | Pass | Parallel registration and resolution for 32 plugins passes the Go race detector |
| Defensive copies | Pass | Registration, snapshot, namespace resolution, and table resolution do not expose mutable registry state |
| Lifecycle boundary | Pass | Unregister immediately closes table and namespace access; manifest and migration binding remains isolated to BF1-07 |
| Current-only rule | Pass | No raw SQL, direct database handle, legacy namespace, dual registration path, or fallback storage was added |
| Developer reference | Pass | The datastore contract documents schema ownership, host fields, limits, and the BF1-07 lifecycle boundary |

The first focused run failed because the `unqueryable_index` fixture still marked its field as queryable. The fixture was corrected, duplicate-field and duplicate-index validation was tightened, and all focused and full gates passed on the retry.

### Verification Commands

```powershell
go test ./internal/plugin/datastore ./pkg/pluginsdk -count=1
go test -race ./internal/plugin/datastore -count=1
go vet ./internal/plugin/datastore ./pkg/pluginsdk
go test ./... -count=1
go vet ./...
git diff --check
```

Result: BF1-02 passed after one acceptance retry. Plugins now have isolated schema ownership metadata ready for trusted query planning.

### Impact Review

- API/OpenAPI: no HTTP or public SDK route change; the registry is host-internal.
- Permission/audit: host scope fields are reserved; trusted predicate injection is implemented in BF1-03 and mutation audit in BF1-04.
- Migration/seed: no database DDL is executed; lifecycle and migration binding belongs to BF1-07.
- Frontend/i18n: none.
- Documentation: datastore contract, BF1 milestone status, Work Item status, and acceptance evidence are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF1-02: register plugin datastore schemas`

## BF1-03 Implement Trusted Scoped Query Planning

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Compile validated plugin queries into deterministic, bound SQL using host-resolved permissions and data scope for SQLite, PostgreSQL, and MySQL.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Trusted identity | Pass | `QueryPlanner` resolves `DataScopeService` from the request context and never accepts final tenant, organization, or owner predicates from plugin input |
| Scope narrowing | Pass | Requested scope is intersected with trusted scope; outside, denied, unresolved, and incomplete scopes return stable `forbidden` errors |
| Permission ownership | Pass | Datastore permissions must belong to the current plugin resource namespace; foreign resources fail before scope resolution |
| Schema allowlist | Pass | Selected, filtered, and sorted fields must exist in the plugin-owned registered table and carry the required field capability |
| Bound SQL | Pass | Identifiers are host-quoted and every plugin or scope value is a bound argument; query plans reject more than 900 parameters |
| Dialects | Pass | Golden tests verify identifier quoting, placeholder numbering, predicates, ordering, and limits for SQLite, PostgreSQL, and MySQL |
| Structured filters | Pass | Nested all/any groups, sets, comparisons, literal contains/prefix escaping, null semantics, and value/schema type agreement compile deterministically |
| Stable pagination | Pass | Missing primary-key fields are appended to sort; opaque keyset cursor is bound to plugin, table, sort, and typed values |
| Cross-dialect ordering | Pass | Nullable sort fields fail closed, avoiding database-specific NULL ordering behavior |
| Result metadata | Pass | Plans separate requested fields from scan fields and automatically include version and cursor keys while fetching `limit + 1` rows |
| Current-only rule | Pass | No offset pagination, raw SQL, sort expression, physical-table input, legacy cursor, dual planner, or dialect fallback exists |
| Developer reference | Pass | The datastore contract documents trusted planning, parameter limits, stable cursor behavior, and the BF1-05 execution boundary |

### Verification Commands

```powershell
go test ./internal/plugin/datastore ./pkg/pluginsdk -count=1
go test -race ./internal/plugin/datastore -count=1
go vet ./internal/plugin/datastore ./pkg/pluginsdk
go test ./... -count=1
go vet ./...
git diff --check
```

Result: BF1-03 passed without acceptance retries. The generated plans are ready for BF1-04 mutation semantics and the BF1-05 SQL execution adapter.

### Impact Review

- API/OpenAPI: no HTTP route changed; the planner consumes the current public `DataQuery` contract.
- Permission/audit: permission ownership and trusted read scope are enforced; mutation audit remains BF1-04.
- Migration/seed: none.
- Frontend/i18n: none.
- Documentation: datastore query-planning rules, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF1-03: implement trusted datastore query planning`

## BF1-04 Implement Transactional Datastore Mutations

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Execute scoped insert, update, upsert, and delete operations with optimistic concurrency, durable idempotency, automatic audit, and shared transaction rollback.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Schema enforcement | Pass | Mutation keys must equal the declared primary key; values must be plugin-managed, mutable, and type-correct; returning fields must be declared |
| Exact write scope | Pass | Every mutation resolves one trusted tenant, organization, and owner; ambiguous all/multi scope requires explicit narrowing and outside scope is forbidden |
| Host fields | Pass | Insert injects trusted scope, version 1, and timestamps; plugins cannot write host-managed fields |
| Optimistic concurrency | Pass | Update, existing upsert, and delete bind the loaded version into the write condition; stale expected or concurrent versions return `conflict` |
| Upsert | Pass | Missing records insert, existing records update and increment version, and expected-version upsert cannot silently create a missing record |
| Durable idempotency | Pass | `sk_plugin_data_mutations` reserves plugin/key, stores canonical request hash and committed result, replays identical requests, and rejects changed requests |
| Idempotency privacy | Pass | Scope-set order is canonicalized and audit stores only a short idempotency hash, not the submitted key or business values |
| Atomic audit | Pass | SQL audit append now resolves the transaction-bound database; mutation, idempotency result, and audit share one commit boundary |
| Rollback | Pass | Audit failure and explicit outer `UnitOfWork` failure leave no business row, idempotency row, or audit row |
| Returned records | Pass | Insert, update, upsert, and delete map requested database fields to typed `DataRecord` values with the correct version |
| Race safety | Pass | Datastore and GORM repository suites pass the Go race detector |
| Production migration list | Pass | The host idempotency model is included in `gormrepo.AllModels` used by MySQL and PostgreSQL adapters |
| Current-only rule | Pass | No non-transaction batch API, partial commit, legacy idempotency record, direct database handle, or mutation fallback exists |

The first focused run completed all mutation assertions but failed test cleanup because Windows still held SQLite files, and the audit transaction test referenced an unavailable helper. The fixtures now close SQL connections explicitly and use the repository's current `TestDB`; all gates passed on retry.

### Verification Commands

```powershell
go test ./internal/plugin/datastore ./internal/store/sql/gormrepo ./internal/plugin/hostservice -count=1
go test -race ./internal/plugin/datastore ./internal/store/sql/gormrepo -count=1
go vet ./internal/plugin/datastore ./internal/store/sql/gormrepo ./internal/plugin/hostservice
go test ./... -count=1
go vet ./...
git diff --check
```

Result: BF1-04 passed after one acceptance retry. The datastore now has a transaction-safe mutation core ready to publish through the BF1-05 host gateway.

### Impact Review

- API/OpenAPI: no route changed; BF1-05 publishes the existing `DataMutation` contract.
- Permission/audit: plugin-owned permission, exact trusted scope, success audit, hashed identifiers, and transactional audit persistence are enforced.
- Migration/seed: `PluginDataMutationModel` is added to the production AutoMigrate model list; no seed data is required.
- Frontend/i18n: none.
- Documentation: datastore mutation rules, database schema, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF1-04: implement transactional datastore mutations`

## BF1-05 Publish Datastore Through The Host Gateway

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Execute typed datastore queries, bind the datastore to a plugin host identity, and publish query/mutation operations through the lifecycle-authenticated loopback gateway.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Host service contract | Pass | `HostServices` now requires `DataStore`; construction fails when its factory or service is absent |
| Query execution | Pass | The host executes planner-produced bound SQL, converts database values to typed records, returns stable keyset cursors, and honors transaction-bound connections |
| Mutation publication | Pass | `datastore.mutate` delegates to the scoped transactional executor without exposing SQL, GORM, database credentials, or physical table names |
| Plugin identity isolation | Pass | Each issued credential stores a host service bound to exactly one normalized plugin ID; schema resolution still requires that plugin's registered namespace |
| Lifecycle credentials | Pass | Disabled plugins cannot receive a credential from the host factory; revoked credentials immediately receive `401` and active transactions are rolled back |
| Request validation | Pass | Gateway accepts only loopback JSON `POST` v1 operations, rejects unknown/trailing JSON, validates query/mutation contracts again, and enforces the 32 MiB request limit |
| Structured errors | Pass | Datastore failures map to deterministic HTTP status plus `code`, `field`, `message`, and `retryable`; internal database errors remain redacted |
| Runtime assembly | Pass | Memory, MySQL, and PostgreSQL bundles expose one datastore database/dialect and bootstrap constructs the same current host port for in-process and managed plugins |
| Transaction visibility | Pass | A query can read a mutation inside the active host transaction and the record disappears after rollback |
| Race safety | Pass | Datastore, gateway, SDK, and client packages pass the Go race detector |
| Current-only rule | Pass | No raw SQL endpoint, legacy datastore API, alternate route, dual wire format, remote database credential, or fallback service was added |

### Verification Commands

```powershell
go test ./internal/plugin/datastore ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin/hostservice ./internal/plugin/pharmaoa ./internal/plugin -count=1
go test ./internal/store ./internal/bootstrap -count=1
go test -race ./internal/plugin/datastore ./internal/plugin ./pkg/pluginsdk ./pkg/pluginclient -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Result: BF1-05 passed without acceptance retries. The host now publishes one authenticated, bounded, typed datastore gateway; BF1-06 will complete external-client error reconstruction, cancellation/deadline, transaction propagation, and conformance evidence.

### Impact Review

- API/OpenAPI: adds process-host operations `POST /v1/datastore/query` and `POST /v1/datastore/mutate`; these are loopback credential APIs, not browser OpenAPI routes.
- Permission/audit: trusted scope and plugin ownership remain mandatory; mutations retain automatic transactional audit.
- Migration/seed: the memory runtime now initializes the existing idempotency model; plugin-owned schema lifecycle remains BF1-07. No seed is added.
- Frontend/i18n: none.
- Documentation: host-service contract, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF1-05: publish datastore through host gateway`

## BF1-06 Publish The External-Process Datastore Client

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Complete the public external-process datastore client with strict wire validation, typed error reconstruction, transaction propagation, cancellation/deadline behavior, and public SDK conformance.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Public service | Pass | `Client.HostServices()` exposes `DataStoreService.Query` and `Mutate` through public `pkg/pluginsdk` types only |
| Request validation | Pass | Query and mutation contracts are validated locally before transport; the host still validates independently |
| Typed values | Pass | String, decimal, JSON, keys, values, versions, returning records, scope filters, and structured predicates survive JSON round trips without number coercion |
| Page metadata | Pass | Records, versions, `hasMore`, and `nextCursor` round trip unchanged; returned record fields must exactly match the requested field set |
| Mutation response | Pass | Result shape is validated and returned record fields must exactly match `returning`; undeclared records fail closed |
| Stable errors | Pass | Valid datastore error responses reconstruct public `DataStoreError` code, field, message, and retryable; malformed or unknown host failures remain typed client transport errors |
| Transaction propagation | Pass | `TransactionService.Within` places the issued transaction ID on datastore requests and finish calls; the real host gateway observes the transaction-bound context |
| Cancellation and deadline | Pass | HTTP calls retain caller context and return errors matching `context.Canceled` and `context.DeadlineExceeded` |
| Strict response decoding | Pass | Unknown fields, trailing JSON, oversized responses, invalid page/result shape, and mismatched record fields are rejected |
| Public conformance | Pass | `plugins/sdk-conformance` mutates and queries a typed record through `HostServices.DataStore` while importing only public contracts |
| Race safety | Pass | Client, host-service, and plugin gateway suites pass the Go race detector |
| Current-only rule | Pass | No old client, alternate datastore transport, permissive decoder, legacy error shape, dual protocol, or fallback implementation exists |

The first focused run failed to compile because the test used `DataOperatorGreaterOrEqual` instead of the current public constant `DataOperatorGreaterOrEq`. After correction, the combined retry timed out because the deadline test waited indefinitely for a server-side request context cancellation on Windows. A 30-second diagnostic run identified the blocked `httptest.Server` connection; the fixture now emits a delayed valid response so the client deadline is tested without platform-dependent server cancellation. Focused, race, full test, and vet gates then passed.

### Verification Commands

```powershell
go test ./pkg/pluginclient ./internal/plugin/hostservice ./internal/plugin -count=1
go test -race ./pkg/pluginclient ./internal/plugin/hostservice ./internal/plugin -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Result: BF1-06 passed after two acceptance retries. External plugins now consume the current typed datastore contract with deterministic transport, transaction, error, cancellation, and response-validation semantics.

### Impact Review

- API/OpenAPI: no new route; this completes the client for BF1-05 loopback host operations.
- Permission/audit: user token, permission, scope intent, and transaction context are preserved; host authorization and audit remain authoritative.
- Migration/seed: none.
- Frontend/i18n: none.
- Documentation: host-service contract, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF1-06: publish external datastore client`

## BF1-07 Bind Datastore Schema And Records To Plugin Lifecycle

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Add one strict typed schema manifest, logical migration table bindings, atomic schema activation, restart and upgrade recovery, explicit rollback, and complete drop-uninstall cleanup.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Current schema contract | Pass | Optional root `datastore.yaml` version 1 is the only declaration; strict YAML rejects unknown fields, extra documents, invalid types, identifiers, keys, indexes, and limits |
| Hidden physical namespace | Pass | Migration SQL uses validated `{{table:logical_name}}` bindings; the host expands quoted opaque names per dialect and rejects unknown or malformed bindings |
| Install validation | Pass | A declared datastore schema requires plugin migration metadata and is fully validated before installed metadata becomes usable |
| Atomic activation | Pass | Candidate schema is prepared before migration and replaces the registry only after migrations and physical table/column inspection succeed |
| Restart durability | Pass | A fresh manager rebuilds the registry from the plugin package against the same database and preserves the existing business record |
| Upgrade durability | Pass | A successful migration adds the declared field while retaining records and atomically publishes the new schema |
| Failed migration | Pass | A multi-step SQLite upgrade failure rolls back the leaked column and leaves the prior schema registration active |
| Explicit rollback | Pass | Enabled-plugin rollback is rejected; disabled rollback requires a positive limit, runs down migrations, and revokes stale schema access until re-enable |
| Drop uninstall | Pass | Down migrations run with logical bindings, registered tables are dropped and verified absent, plugin idempotency rows are deleted, and the namespace is unregistered |
| Retain/archive | Pass | Non-drop policies revoke runtime schema access without deleting relational records |
| Full quality gate | Pass | Focused lifecycle tests, targeted race tests, full Go tests, full vet, and diff checks pass |
| Current-only rule | Pass | No inferred field type, legacy schema file, raw physical name contract, alternate registration API, direct database handle, or fallback storage path exists |

During final architecture review, migration fixtures were changed from precomputed physical names to logical table bindings so plugin packages never depend on host-internal namespace mapping. All acceptance gates passed without retry.

### Verification Commands

```powershell
go test ./internal/plugin ./internal/plugin/datastore ./internal/bootstrap -run "TestPluginMigration|TestLoadSchemaManifest|TestPluginDataStore" -count=1
go test -race ./internal/plugin/datastore ./internal/bootstrap -run "TestSchemaRegistry|TestLoadSchemaManifest|TestPluginDataStore" -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Result: BF1-07 passed. Datastore schemas and records now follow plugin install, enable, restart, upgrade, explicit rollback, and uninstall lifecycles without exposing physical storage internals.

### Impact Review

- API/OpenAPI: no browser HTTP route change; explicit rollback is an internal manager capability for the later operator API/UI task.
- Permission/audit: runtime DataStore access remains credential and scope bound; migration audit events retain lifecycle status and step counts.
- Migration/seed: introduces strict logical table binding expansion and validates the migrated physical shape before schema publication.
- Frontend/i18n: none.
- Documentation: lifecycle schema reference, HostServices reference, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF1-07: bind datastore to plugin lifecycle`
