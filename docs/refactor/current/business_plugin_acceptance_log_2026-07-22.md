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

## BF1-08 Prove Plugin Datastore Lifecycle E2E

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Prove the public datastore and lifecycle contracts with an independent packaged plugin running as a real managed process.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Independent boundary | Pass | `plugins/datastore_e2e/backend` imports public `pkg/pluginclient` and `pkg/pluginsdk` contracts and contains no core plugin registration code |
| Clean package | Pass | The test builds the plugin binary, packages source declarations and migrations, installs into an empty plugin root, and verifies every required artifact |
| Real process | Pass | The managed-process launcher starts the packaged binary, provisions host credentials, and reaches its HTTP API over an ephemeral loopback address |
| Scoped write and query | Pass | A JWT-bound employee creates and lists a record through a real host gateway, transaction service, SQLite adapter, and datastore registry |
| Cross-scope rejection | Pass | The same process submits a forged tenant predicate and receives the public forbidden datastore response |
| Restart durability | Pass | Disable stops the process and makes the endpoint unavailable; re-enable restarts it and preserves the scoped record |
| Explicit rollback | Pass | Disabled rollback removes the active schema and physical table; re-enable reapplies migration and presents a clean datastore |
| Drop uninstall | Pass | Uninstall stops the process, removes the physical table and schema registration, and deletes plugin idempotency rows |
| Audit attribution | Pass | Successful plugin mutations publish the concrete `datastore.insert` action through the host audit contract |
| Full quality gate | Pass | Focused real-process E2E, race, full Go tests, full vet, dependency-boundary inspection, and diff checks pass |
| Current-only rule | Pass | The sample uses one current manifest, client, SDK, schema declaration, migration binding, and lifecycle path; no compatibility or fallback path exists |

The first acceptance run exposed a missing empty map literal in the E2E fixture. The second completed the full lifecycle and showed that the audit contract records the concrete mutation action `datastore.insert`; the assertion was corrected from a non-contract generic action name. The third focused run passed.

### Verification Commands

```powershell
go test ./plugins/datastore_e2e/backend ./internal/bootstrap -run TestIndependentPluginDataStoreProcessLifecycleE2E -count=1 -v
go test -race ./internal/bootstrap -run TestIndependentPluginDataStoreProcessLifecycleE2E -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Result: BF1-08 and milestone BF1 passed. A clean independent process now exercises the complete public datastore path through package, runtime, scope, transaction, restart, rollback, audit, and uninstall boundaries.

### Impact Review

- API/OpenAPI: no browser route change; the sample exposes only plugin-owned routes declared in its manifest.
- Permission/audit: host-issued JWT identity remains authoritative, forged tenant scope fails closed, and successful mutations are audited.
- Migration/seed: the sample uses one current `datastore.yaml` and logical migration bindings; the E2E creates and removes its isolated SQLite namespace.
- Frontend/i18n: none.
- Documentation: datastore reference, Work Item state, milestone state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF1-08: prove datastore lifecycle end to end`

## BF2-01 Define Reusable Business-Document Schemas And States

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Publish one strict plugin-facing contract for business-document schemas, typed values, records, state transitions, metadata, and optimistic action versions.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Public boundary | Pass | All contracts live in `pkg/pluginsdk` and depend only on the Go standard library; plugins do not import internal form, workflow, pharma OA, or database packages |
| Schema identity | Pass | Lowercase type keys, positive schema versions, unique header/line/state/action keys, bounded collections, and one declared non-terminal initial state are enforced |
| Header and lines | Pass | Required/unknown header fields, named line groups, row limits, line IDs, duplicate rows, required line fields, and unknown line groups fail with exact paths |
| Lossless values | Pass | Integer and decimal strings are canonical; money carries uppercase currency; quantity carries unit; dates, UTC datetimes, booleans, references, strings, text, and JSON are explicit |
| Validation rules | Pass | Numeric min/max use exact rational comparison; string length and regex rules are bounded, type-compatible, unique, and deterministically applied |
| State graph | Pass | Terminal sources, self-transitions, unknown states, duplicate actions, and non-terminal dead ends are rejected; `NextState` enforces availability and required comments |
| Record metadata | Pass | Document type/schema version, number, title, current state, positive record version, UTC timestamps, actor attribution, and bounded unique tags validate |
| Optimistic action | Pass | `DocumentActionInput` requires one document, declared-shaped action key, and positive `expectedVersion`; no last-write-wins input exists |
| Deterministic errors | Pass | Map keys are sorted before unknown-field validation and `DocumentContractError` publishes stable field/message data for UI binding |
| Wire contract | Pass | JSON uses explicit tagged values and camelCase fields including `schemaVersion`, `createdAt`, `createdBy`, currency, unit, and typed references |
| Full quality gate | Pass | Focused contract tests, focused race tests, full Go tests, full vet, CodeGraph review, and diff checks pass |
| Current-only rule | Pass | No dynamic-form translation, float business value, inferred type, undeclared-field preservation, alternate state path, legacy format, or fallback exists |

All focused acceptance tests passed on the first run. CodeGraph review confirmed that the contract sits above datastore persistence and workflow execution without replacing or importing either internal implementation.

### Verification Commands

```powershell
go test ./pkg/pluginsdk -run Document -count=1
go test -race ./pkg/pluginsdk -run Document -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Result: BF2-01 passed. Independent plugins can now describe and deterministically validate business-document headers, lines, exact business values, custom fields, state machines, metadata, and optimistic versions through one public contract.

### Impact Review

- API/OpenAPI: no HTTP route was added; the JSON-tagged SDK types establish the wire shape for later host services.
- Permission/audit: no authorization path changed; metadata requires actor attribution while trusted tenant and organization scope remains host-owned.
- Migration/seed: none.
- Frontend/i18n: no component changed; stable field paths and labels are ready for schema-driven forms in BF2-06.
- Documentation: plugin business-document contract, Work Item state, milestone state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts only.

### Commit

`BF2-01: define business document contracts`

## BF2-02 Implement Tenant-Safe Document Numbering

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Publish one host-owned numbering service for tenant-safe preview and transactional issuance across in-process and independent plugins.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Public boundary | Pass | `pkg/pluginsdk` exposes strict rule, input, result, service, and typed-error contracts; `pkg/pluginclient` preserves them across the managed process boundary |
| Namespace isolation | Pass | Sequences are keyed by plugin, tenant, document type, and UTC period; tenant/type/period matrix tests each begin independently at one |
| Concurrent uniqueness | Pass | 32 concurrent issue attempts produce 32 unique, continuous sequence values; SQL row locking, optimistic update checks, and a unique value index protect issuance |
| Transactional gap policy | Pass | Formal issue requires a host transaction; rollback removes sequence and idempotency writes and the rolled-back number is reused |
| Idempotency | Pass | Same key and input returns the committed number with `duplicate=true`; changed input returns a stable conflict and does not advance the sequence |
| Rule stability | Pass | The first issue persists a rule hash; preview and issue reject rule changes within the active tenant/type/period namespace |
| Restart continuity | Pass | A newly constructed service over the same database previews the next committed number |
| Authorization | Pass | Host service resolves the declared permission, constrains trusted scope to exactly one requested tenant, and rejects cross-tenant access before the backend |
| Audit | Pass | First committed issuance writes `document_number.issue`; idempotent replay does not create a duplicate audit entry |
| Process gateway | Pass | Public client preview and issue succeed through the loopback gateway; conflict and transaction-required errors retain code and field |
| Persistence | Pass | GORM models and MySQL/PostgreSQL migration 27 create sequence and issue tables with matching keys, rule hash, timestamps, and unique constraints |
| Full quality gate | Pass | Focused tests, focused race tests, full Go tests, full vet, CodeGraph review, migration review, and diff checks pass |
| Current-only rule | Pass | No legacy counter, alternate gap policy, local plugin counter, compatibility adapter, dual write, or fallback path exists |

The first host-service acceptance run exposed an incomplete test scope fixture and failed. The fixture was corrected to model one trusted tenant with unrestricted owner and organization dimensions, and the failed acceptance suite plus the full quality gate were re-executed successfully.

### Verification Commands

```powershell
go test ./internal/plugin/hostservice ./internal/plugin -run DocumentNumber -count=1 -v
go test ./... -count=1
go test -race ./pkg/pluginsdk ./pkg/pluginclient ./internal/store/sql/gormrepo ./internal/plugin/hostservice ./internal/plugin -run DocumentNumber -count=1
go vet ./...
git diff --check
```

Result: BF2-02 passed. Independent plugins can now preview and atomically issue tenant-safe business-document numbers with explicit rollback, idempotency, authorization, audit, restart, and rule-freezing semantics.

### Impact Review

- API/OpenAPI: no public application HTTP route was added; managed plugin host capability `document-numbers` now exposes `preview` and `issue` operations.
- Permission/audit: every call carries a permission and tenant; first issuance is audited under the authenticated host context.
- Migration/seed: migration 27 creates two current-only tables for MySQL and PostgreSQL; memory mode migrates the same GORM models; no seed data.
- Frontend/i18n: no UI changed; typed field errors and preview output are ready for BF2-06 schema-driven forms.
- Documentation: plugin numbering guide, migration notes, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; host-owned numbering is the only supported implementation.

### Commit

`BF2-02: implement tenant-safe document numbering`

## BF2-03 Bind Documents To Workflow And Approval Actions

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Publish one tenant-scoped document workflow service that commits document state, workflow state, optimistic versions, and idempotency results through the current plugin transaction boundary.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Public contract | Pass | `pkg/pluginsdk` defines submit, get, approve, reject, withdraw, delegate, cancel, result, validation, and typed-error contracts; `HostServices.Validate` requires the `Documents` port |
| Trusted boundary | Pass | Host service binds plugin identity and actor from trusted context, constrains every request to exactly one authorized tenant, and rejects cross-tenant access before persistence |
| Action validity | Pass | Document schema transitions are validated before workflow mutation; an integration test deliberately swallows an invalid-action error and proves neither state advances |
| Action matrix | Pass | Real SQL-backed integration covers submit, approve, reject, withdraw, delegate, cancel, and delegated approval |
| Idempotency | Pass | Repeating an identical approval returns the committed document/workflow result with `duplicate=true`; changed input returns conflict |
| Optimistic concurrency | Pass | Decisions require the current positive document version; the repository locks the binding and rechecks idempotency after lock acquisition |
| Atomicity | Pass | Workflow persistence reuses the ambient host transaction; submit rollback removes both records and action rollback restores both document and workflow state |
| Process gateway | Pass | `pkg/pluginclient` exposes all document operations and preserves stable document workflow error code, field, message, and retryability through the loopback gateway |
| Public conformance | Pass | `plugins/sdk-conformance` creates, submits, and approves a typed document using only public contracts |
| Audit | Pass | First submit and decision record document audit actions; idempotent replay does not emit a duplicate document audit entry |
| Persistence | Pass | GORM models and MySQL/PostgreSQL migration 28 create tenant/plugin bindings, schema/document snapshots, workflow uniqueness, and immutable idempotency actions |
| Full quality gate | Pass | Focused tests/race, full Go tests/race, vet, CodeGraph impact review, migration review, and diff checks pass |
| Current-only rule | Pass | No legacy document state, alternate workflow, compatibility adapter, dual write, local plugin transaction, or fallback exists |

The first full race run failed in the pre-existing notification delivery uniqueness test. That exact test then passed 20 consecutive race runs, and two later full race runs passed. No notification code was changed. Final review additionally found and fixed workflow-first validation; the final complete gate was rerun after that correction.

### Verification Commands

```powershell
go test ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/service/documentworkflow ./internal/store/sql/gormrepo -run "DocumentWorkflow|HostServices|ThirdPartyPlugin" -count=1
go test -race ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/service/documentworkflow ./internal/store/sql/gormrepo -run "DocumentWorkflow|HostServices|ThirdPartyPlugin" -count=1
go test -race ./internal/store/sql/gormrepo -run TestNotificationStoreDeduplicatesConcurrentDeliveryKey -count=20
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
git diff --check
codegraph impact HostServices
```

Result: BF2-03 passed. Independent plugins can submit typed business documents and execute workflow-bound decisions with trusted identity, tenant isolation, stable errors, idempotent replay, and one atomic persistence boundary.

### Impact Review

- API/OpenAPI: no application HTTP route was added; managed plugin host capability `documents` now exposes `submit`, `act`, and `get` operations.
- Permission/audit: every call carries a permission and tenant; actors come from the verified context; first writes emit namespaced document and workflow audit evidence.
- Migration/seed: migration 28 creates two current-only tables for MySQL and PostgreSQL; memory mode migrates the same models; no seed data.
- Frontend/i18n: no UI changed; the stable action, state, version, and typed-error model is ready for BF2-06 components.
- Documentation: host capability reference, document workflow guide, migration notes, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts and current persistence only.

### Commit

`BF2-03: bind documents to workflow actions`

## BF2-04 Add Attachments, Comments, And Immutable Timelines

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Publish tenant-scoped document attachments, immutable comments, and one ordered append-only activity timeline through the current in-process and managed-plugin host boundary.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Public contract | Pass | `DocumentService` extends document workflows with add/remove/list attachment, add/list comment, and cursor-paged timeline operations plus bounded request and strict response validation |
| File ownership | Pass | The Host resolves every attachment through the plugin- and actor-scoped `FileService`; a user cannot attach another user's private file and receives stable `forbidden` on `fileId` |
| Tenant isolation | Pass | Every collaboration operation resolves the declared permission and constrains it to exactly one trusted tenant before document access |
| Attachment lifecycle | Pass | Add stores an immutable file snapshot and actor; remove is an attributed soft unlink that preserves the file snapshot and add/remove sequence history |
| Comment immutability | Pass | Comments are append-only, attributed, byte-bounded, and expose no update or delete service or persistence lifecycle columns |
| Timeline ordering | Pass | Submit, workflow actions, attachment add/remove, and comment add share one per-document monotonic sequence with a database unique constraint and strict cursor paging |
| Transaction atomicity | Pass | Collaboration writes require a host transaction; explicit rollback leaves no attachment, comment, or timeline event while retaining the committed submit event |
| Audit | Pass | Attachment add/remove and comment add emit attributed audit actions linked to immutable event IDs and sequences without copying comment bodies into audit detail |
| Stable errors | Pass | Duplicate comments, duplicate/removed attachments, missing resources, missing transactions, forbidden files, and malformed external responses retain operation-specific typed code and field data |
| Process gateway | Pass | All six collaboration operations round-trip through `pkg/pluginclient`; mismatched document IDs, cursors, attribution, event shapes, and ordering fail closed as `unavailable` response errors |
| Public conformance | Pass | `plugins/sdk-conformance` stores a file, submits a document, attaches the file, comments, approves, and reads all collaboration history using only public packages |
| Persistence | Pass | GORM models and MySQL/PostgreSQL migration 29 create scoped attachment, comment, and timeline tables with binding foreign keys and matching indexes |
| Full quality gate | Pass | Focused tests/race, full Go tests/race, vet, CodeGraph impact review, migration review, and diff checks pass after final error-contract review |
| Current-only rule | Pass | No legacy attachment table, mutable comment path, alternate timeline, compatibility adapter, dual write, or fallback exists |

### Verification Commands

```powershell
go test ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/store/sql/gormrepo -run "DocumentWorkflow|DocumentCollaboration|HostGatewayClientConformance|ThirdPartyPlugin" -count=1
go test -race ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/store/sql/gormrepo -run "DocumentWorkflow|DocumentCollaboration|HostGatewayClientConformance|ThirdPartyPlugin" -count=1
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
git diff --check
codegraph impact DocumentService
```

Result: BF2-04 passed. Independent plugins can collaborate on approval-backed business documents with host-verified file ownership, immutable attribution, ordered history, transaction rollback, stable process-boundary errors, and one current persistence model.

### Impact Review

- API/OpenAPI: no application HTTP route was added; managed plugin host capability `documents` adds six collaboration operations to the current public client.
- Permission/audit: tenant and file access remain host-owned; attachment and comment mutations emit resource-linked audit evidence under the verified actor.
- Migration/seed: migration 29 adds three current-only tables for MySQL and PostgreSQL; memory mode migrates matching GORM models; no seed data.
- Frontend/i18n: no UI changed; attributed attachments, comments, stable cursors, and timeline event kinds are ready for BF2-06 document components.
- Documentation: plugin document workflow guide, migration notes, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts and current persistence only.

### Commit

`BF2-04: add document collaboration timeline`

## BF2-05 Add Document Search, Cursor Paging, Print, And Export

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Publish tenant-scoped document discovery, stable keyset paging, permission-aware print snapshots, and bounded durable export plans through the current plugin Host boundary.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Public contract | Pass | `DocumentService` exposes validated search, print, and export types with explicit sort, cursor, format, row, sensitive-field, and typed-error bounds |
| Tenant isolation | Pass | Host authorization constrains every search, print, and export request to exactly one declared tenant before repository or job access; cross-tenant tests fail closed |
| Search filtering | Pass | SQL search supports bounded type, state, creator, created-time, number/title text, and plugin/tenant filters; `%`, `_`, and `!` are escaped as literal input |
| Stable cursor | Pass | Keyset paging orders by an allowlisted field plus document ID; the cursor binds the normalized query hash and last key; filter changes invalidate it |
| Concurrent insertion | Pass | Acceptance inserts a document before the first-page cursor and proves the next page neither repeats nor drifts into the new preceding row |
| Query projection | Pass | Migration 28 and GORM models persist number, title, creator, and updater projections with scoped updated-time, created-time, number, and type/state indexes |
| Print policy | Pass | Print deep-copies the document and removes schema fields marked `sensitive` by default; only a separately authorized sensitive permission reveals them |
| Durable export | Pass | Export freezes the authorized query, actor, format, row bound, and sensitive decision in a `document_export` job capped at 50,000 rows; no synchronous large export path exists |
| Export idempotency | Pass | Identical job ID/key/input replays the persisted job; a fresh SQL Job service reconstructs the same plan; changed or unauthorized sensitive input fails |
| Process gateway | Pass | Search, print, and export round-trip through the managed gateway; malformed, mismatched, oversized, or cross-filter responses are rejected as retryable `unavailable` response errors |
| Public conformance | Pass | `plugins/sdk-conformance` searches, prints redacted and sensitive snapshots, schedules, leases, and completes an export using only public SDK services |
| Full quality gate | Pass | Focused tests/race, full Go tests/race, vet, CodeGraph impact review, migration review, and diff checks pass |
| Current-only rule | Pass | No offset compatibility mode, legacy search, alternate export path, dual projection, schema backfill, or fallback exists |

The first gateway conformance run correctly rejected an invalid print fixture whose schema omitted required header fields. The fixture was repaired and the failed suite was rerun. Final review also removed client-side reimplementation of database text collation and number ordering; stability remains proven at the SQL service boundary without introducing MySQL/PostgreSQL false rejections.

### Verification Commands

```powershell
go test ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/store/sql/gormrepo -run "Document(Query|Search|Print|Export|Workflow|Collaboration)|HostGatewayClientConformance|ThirdPartyPlugin" -count=1
go test -race ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/store/sql/gormrepo -run "Document(Query|Search|Print|Export|Workflow|Collaboration)|HostGatewayClientConformance|ThirdPartyPlugin" -count=1
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
git diff --check
codegraph impact DocumentQueryService
```

Result: BF2-05 passed. Independent plugins can discover approval-backed documents with stable scoped cursors, produce permission-aware print snapshots, and schedule bounded durable exports without owning host tables or bypassing trusted scope.

### Impact Review

- API/OpenAPI: no application HTTP route was added; the managed `documents` capability now exposes `search`, `print`, and `export` through the public client.
- Permission/audit: every query carries a tenant permission; sensitive fields require a second explicit permission; print and export scheduling emit audit evidence under the verified actor.
- Migration/seed: the current migration 28 directly defines query projections and indexes for MySQL/PostgreSQL; memory mode migrates the same model; no compatibility migration or seed data exists.
- Frontend/i18n: no UI changed; BF2-06 can consume stable summaries, cursors, print snapshots, redaction paths, and export job IDs.
- Documentation: plugin document workflow guide, migration notes, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; current contracts, current projections, and current durable jobs only.

### Commit

`BF2-05: add document query and export`

## BF2-06 Build Schema-Driven Document UI Components

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Publish a host-independent Element Plus document UI package covering schema-driven list, form, detail, approval, collaboration, timeline, and redaction-safe print workflows.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Public package | Pass | `@skoll/document-ui` has an independent package manifest, lockfile, source boundary, peer dependencies, build, type declarations, tests, README, and packable exports |
| Host independence | Pass | Package source imports no host view, workflow client, internal component, or host source path; plugins consume only the public package and browser host bridge |
| Schema model | Pass | Header/line schemas, exact string-backed number types, money, quantity, dates, references, JSON, state/action metadata, document records, workflow state, and collaboration records are typed |
| Forms and validation | Pass | Required fields, line bounds, numeric precision, JSON, dates, references, stable field paths, immutable updates, save, submit, cancel, add, and remove controls are covered |
| Document workflows | Pass | Detail composition includes approve, reject, delegate, withdraw, cancel, attachments, immutable comments, ordered timeline, print, export, and sensitive-field redaction |
| Locale and time | Pass | zh-CN/en-US messages and overrides cover visible kit copy; boolean and attachment actions are localized; selected date-time values emit RFC3339 UTC |
| Responsive states | Pass | Desktop/mobile list, form, detail, approval, and print layouts have no page overflow; wide tables use explicit horizontal work surfaces |
| Theme and density | Pass | Light/dark x comfortable/compact consume the current semantic token bridge without raw product colors or a package-owned theme fork |
| State matrix | Pass | Ready, loading, empty, error, forbidden, form, detail, and redaction-safe print fixtures pass in desktop and mobile Playwright projects |
| Package budget | Pass | Library output is 58.34 KB raw/12.44 KB gzip JavaScript and 12.86 KB raw/2.29 KB gzip CSS; Vue, Element Plus, and Lucide remain peer dependencies |
| Package consumption | Pass | `npm pack --dry-run` includes README, JavaScript, CSS, and declarations; a runtime import verifies all eight primary component exports |
| Component tests | Pass | Host component suite passes 13 tests and the independent document package passes 5 tests |
| Full frontend gate | Pass | Host typecheck/static checks, package typecheck, production build, and all bundle budgets pass |
| Current-only rule | Pass | No host-source import, legacy component path, alternate payload, compatibility mode, dual implementation, or fallback UI exists |

The first visual run exposed five mobile overflow/boundary failures in form and detail work surfaces. Grid minimum widths and explicit table scroll boundaries were corrected before the 20-case matrix passed. After extraction into the public package, a later run passed 16/20 but lost stable error/forbidden semantic selectors when the host `StateBlock` dependency was removed; package-owned state hooks were added and the complete 20-case matrix passed on retry.

### Verification Commands

```powershell
cd packages/skoll-document-ui
npm run typecheck
npm test
npm run build
npm pack --dry-run
node -e "import('./dist/index.js').then(...)"

cd ../../web
npm run test:documents:browser
npm run typecheck
npm run test:components
npm run build
npm run check:bundle

cd ..
codegraph sync .
codegraph status .
codegraph impact DocumentList
rg -n "web/src|\.\./workflow|\.\./components|src/business-documents" packages/skoll-document-ui web/tests docs/development/plugin-business-document-ui.en.md
git diff --check
```

Result: BF2-06 passed after two acceptance retries. Independent plugins now have a compact, packable, schema-driven business document UI package that consumes current host contracts without importing or modifying host production pages.

### Impact Review

- API/OpenAPI: no HTTP route changed; the public TypeScript component and workflow-view types mirror the current document and workflow contracts.
- Permission/audit: components accept explicit action and visibility decisions; trusted authorization, sensitive-field filtering, audit, and mutation remain backend-owned.
- Migration/seed: none.
- Frontend/i18n: one public package provides controlled components, two locales, token-driven themes/density, responsive states, and browser fixtures.
- Documentation: package README, developer package guide, development index, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; current package and current contracts only.

### Commit

`BF2-06: add schema-driven document UI kit`

## BF2-07 Generate And Prove A Zero-Edit Business-Document Plugin

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Add an explicit document generator target and prove an independently packaged plugin through the public Go SDK, public Vue document package, and complete plugin lifecycle.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Generator model | Pass | `DocumentSpec` explicitly declares schema key/name, approval definition, number prefix, and title field; invalid plugin or field bindings are rejected |
| Public backend boundary | Pass | Generated backend imports only `pkg/pluginclient` and `pkg/pluginsdk`, forwards the request bearer token through the public client, and imports no host `internal/` package, handler, service, repository, or source path |
| Public frontend boundary | Pass | Generated Vue source depends on `@skoll/document-ui`, requires the injected Host SDK bridge, emits a typed schema module, and composes document list, form, detail, approval, and export surfaces without direct-fetch fallback |
| Document contract | Pass | Generated JSON schema validates and declares `draft -> submitted -> approved/rejected` states and actions using current public contract types |
| Workflow and transaction | Pass | Generated startup ensures the approval definition through the public workflow service; submit and approve run through the public transaction and document services |
| Manifest and permissions | Pass | Search, detail, create, submit, approve, and export routes carry plugin-owned permission and audit declarations |
| Zero-edit build | Pass | Generated Go tests, Vue typecheck, and Vue production build run from emitted source without editing a generated file |
| Package integrity | Pass | PowerShell package flow builds backend/frontend, creates SHA-256 evidence, verifies the archive, installs it, and preserves all generated source hashes |
| Runtime lifecycle | Pass | Installed plugin enables, starts under the managed supervisor, creates a draft, submits, searches, reads detail, approves, schedules export, disables execution, rolls back owned migration data, and uninstalls |
| Generator contract tests | Pass | Generated Go parses, schema validates, SDK/UI dependency boundaries and manifest routes are asserted deterministically |
| Backend quality gate | Pass | Generator domain/service, public SDK/client, and plugin runtime packages pass their full test suites |
| Frontend quality gate | Pass | Public UI typecheck/tests/build and host typecheck/component tests/build/bundle budgets pass |
| Generated UI size | Pass | Element Plus component registration reduced generated JavaScript from 1,049.05 KB raw/342.32 KB gzip to 562.57 KB raw/187.88 KB gzip |
| Current-only rule | Pass | No legacy route, alternate payload, compatibility adapter, host-source import, dual implementation, or runtime fallback was added |

The first lifecycle run failed because the backend template referenced a non-existent document sort constant. The template was corrected and the complete lifecycle restarted. The second run passed Go and Vue builds but failed when an incorrect repository-relative Go replacement leaked into the package tool; the generated module path and isolated test workspace were corrected, then the complete lifecycle passed twice, including after frontend bundle optimization.

### Verification Commands

```powershell
go test ./internal/domain/generator ./internal/service/generator ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin -count=1
go test ./internal/service/generator -run TestDocumentPluginTargetUsesOnlyPublicContracts -count=1
$env:SKOLL_GENERATOR_PLUGIN_E2E = "1"
go test ./internal/service/generator -run TestGeneratedPluginBuildPackageAndInstallWithoutSourceEdits -count=1 -v

cd packages/skoll-document-ui
npm run typecheck
npm test -- --run
npm run build

cd ../../web
npm run typecheck
npm run test:components
npm run build
npm run check:bundle

cd ..
codegraph sync .
codegraph status .
codegraph impact DocumentSpec
codegraph impact renderDocumentPluginBackendServer
git diff --check
```

Result: BF2-07 passed after two acceptance retries. A generated business-document plugin now crosses the public SDK and UI boundaries and completes the required package/runtime/business lifecycle without hand edits.

### Impact Review

- API/OpenAPI: generated plugin manifests declare current document routes; no host application route was added.
- Permission/audit: every generated operation carries an explicit plugin-owned permission and manifest audit action; host services remain the enforcement boundary.
- Migration/data: generated plugin tables and migrations remain plugin-owned; uninstall follows the one declared current policy.
- Frontend/i18n: generated UI uses the public document kit and Element Plus component registration; host application pages are unchanged.
- Generator: CRUD and business-document targets are explicit generator products; document output adds schema artifacts and public SDK/UI dependencies.
- Documentation: generation guide, development index, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; only current contracts and generated outputs exist.

### Commit

`BF2-07: generate zero-edit document plugins`

## BF3-01 Redesign The Plugin Control-Center Information Architecture

- Date: 2026-07-23
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Audit the current plugin console and freeze one current-only route, ownership, state, component, and interaction contract for the control center.

### Acceptance Result

| Check | Result | Evidence |
| --- | --- | --- |
| Current audit | Pass | The 2,994-line plugin view, local heavy-panel state, global inspector, duplicated inventory, inferred health, broad detail drawer, and embedded developer portal are mapped to concrete replacement owners |
| Scannable hierarchy | Pass | Fleet navigation and an eight-surface plugin workspace give install, runtime, capabilities, data, migrations, jobs, audit, and errors one stable location |
| Ownership | Pass | Every required concern has one canonical route, source, mutation owner, and BF3 implementation Work Item |
| State model | Pass | Install, lifecycle, data, job, and request states are finite; stale observations remain visibly distinct from current health |
| Component contract | Pass | Twelve focused component responsibilities and forbidden ownership boundaries replace page-global state and duplicated behavior |
| Interaction review | Pass | Deep links, browser history, install/lifecycle commands, confirmation, durable evidence, correlation navigation, responsive navigation, keyboard, and focus behavior are specified |
| Current-only rule | Pass | The old monolithic route, alternate detail drawer, compatibility tabs, fallback sources, and mixed developer/operator navigation are explicitly excluded |

### Verification

```powershell
codegraph status .
codegraph node "web/src/views/Plugin/index.vue"
codegraph impact openPluginDetail
git diff --check
```

Result: BF3-01 passed. BF3-02 through BF3-06 now have one implementation boundary for an operational, routed, independently testable plugin control center.

### Impact Review

- API/OpenAPI: no runtime API changed; required snapshot timestamps and durable operation/correlation IDs are frozen as frontend consumption requirements for BF3-02 through BF3-04.
- Permission/audit: route permissions and mutation ownership are explicit; detailed permission and audit implementation remains in the owning Work Items.
- Migration/seed: none.
- Frontend/i18n: route and component architecture is frozen; implementation begins in BF3-02.
- Documentation: current README, Work Item status, information architecture, and acceptance evidence are synchronized.
- Compatibility: none; the current monolithic route is replaced directly when BF3-02 lands.

### Commit

`BF3-01: define plugin control center architecture`

## BF3-02 Build Runtime Health And Capability Views

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing` across six repair cycles, then `Review -> Done`
- Scope: Replace the plugin-console monolith with an operational fleet, routed plugin workspace, authoritative runtime snapshot, capability inventory, lifecycle commands, schema settings, and responsive browser coverage.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Authoritative API | Pass | `GET /v1/plugins/{id}/control` returns identity, capture/stale timestamps, lifecycle state, health, service URLs, host services, routes, permissions, dependencies, and extension counts in one response |
| Failure inspectability | Pass | Health-provider failures remain a successful inspectable snapshot with `health_unavailable`; manager absence, invalid IDs, and missing plugins retain explicit HTTP outcomes |
| Runtime state model | Pass | Loading, ready, degraded, crashed, disabled, and stale are derived and component-tested; route authorization produces the distinct global forbidden state |
| Lifecycle truth | Pass | Enable, disable, and uninstall commands are selected from the current backend state and protected for system plugins; the browser proves enabled-plugin actions without mutating the fixture |
| Routed ownership | Pass | Fleet, install, marketplace, overview, runtime, capabilities, and settings use `/skoll/plugin-center`; business plugins retain the separate `/skoll/plugins/<id>` namespace |
| Workspace ownership | Pass | One provider fetches and owns the control snapshot; lazy child routes consume it without duplicate runtime fetches |
| Fleet usability | Pass | Search/state filtering, row navigation, explicit control and business-entry commands, empty/error/loading states, and current lifecycle labels are present |
| Configuration and install | Pass | Schema-driven settings and validate-review-install flow were retained as focused routes; marketplace selection enters the install flow |
| Permission behavior | Pass | `admin` reaches every control route; `dept_admin` is redirected to the explicit forbidden surface on desktop and mobile |
| Responsive browser matrix | Pass | Playwright passes fleet plus overview/runtime/capabilities/settings at 1440x1000 and 390x844 with zero document overflow; screenshots were visually inspected |
| i18n and accessibility | Pass | 1,853 locale keys, 1,441 references, zero hard-coded visible strings, 65 named icon buttons, and 15 guarded confirmations pass automated gates |
| Component coverage | Pass | Host frontend passes 8 files/15 tests, including state derivation and fleet navigation; public document UI passes 2 files/5 tests |
| Production performance | Pass | Vite transforms 3,639 modules; control-center route chunks remain about 1.05-2.08 KB gzip and all bundle budgets pass |
| API and module docs | Pass | OpenAPI defines the complete control snapshot and response envelope; the module README documents routes, owners, authoritative data, and verification commands |
| Current-only rule | Pass | The 2,994-line page and old route are replaced directly; no compatibility route, alternate payload, dual workspace, or fallback data source exists |

The first backend run exposed an incomplete test fixture. Frontend checks then found a missing locale key, one hard-coded label, unnamed icon buttons, and an Element Plus fixed-column test assumption. Those were corrected before the full gate. The first browser run failed because the new test referenced the Overview CSS class instead of its existing `data-testid`; the selector was corrected and all four browser cases were rerun. The final strict typecheck then found an implicit row parameter type; navigation was moved to a typed handler and the complete backend, type, component, build, browser, and bundle gates passed again. Repository-wide Redocly lint still reports the pre-existing OpenAPI baseline errors outside this contract; all nine new `PluginControl*` references resolve to declared schemas.

### Verification Commands

```powershell
go test ./internal/handler/http/v1/plugin ./internal/handler/http/v1/system -count=1

cd web
npm run typecheck
npm run test:components
$env:SKOLL_E2E_BASE_URL = "http://127.0.0.1:5173"
npm run test:plugin-center
npm run build
npm run check:bundle

cd ..
codegraph sync .
codegraph status .
codegraph impact control
codegraph impact deriveRuntimeState
git diff --check
```

Result: BF3-02 passed after six repair cycles. Operators now have one current, permission-aware, responsive control center for plugin inventory, health, capabilities, lifecycle, installation, and configuration.

### Impact Review

- API/OpenAPI: one authenticated control snapshot endpoint and matching OpenAPI/TypeScript contracts were added; existing lifecycle and configuration payloads are unchanged.
- Permission/audit: all control routes require `plugin.read`; install requires `plugin.manage`; lifecycle handlers retain their existing audit boundary.
- Migration/data: none.
- Frontend/i18n: the monolithic plugin page is replaced by a fleet and lazy routed workspace with complete zh-CN/en-US copy.
- Performance: control routes are independent async chunks and pass current entry, initial, async, total JavaScript, CSS, and chunk-count budgets.
- Documentation: information architecture, OpenAPI, module README, Work Item state, and acceptance evidence are synchronized.
- Compatibility: none; only the current route namespace and current snapshot contract exist.

### Commit

`BF3-02: build plugin runtime control center`

## BF3-03 Build Datastore And Migration Lifecycle Views

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Add authoritative plugin data/schema inspection, durable migration-ledger inspection, policy-aware rollback, and responsive Data/Migrations workspaces.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Authoritative data API | Pass | `GET /v1/plugins/{id}/data-control` reads the live plugin state, datastore schema registry or inactive declared schema, physical table existence/size, migration ledger, policies, and allowed actions |
| Existing engine reuse | Pass | Inspection and rollback reuse `MigrationPlanner`, `MigrationStore`, `PluginMigrationHook`, and `datastore.Lifecycle`; no second migration engine or browser-derived state exists |
| Disabled-schema inspectability | Pass | A schema unregistered by explicit rollback remains inspectable from the sole current `datastore.yaml` declaration, including physical table presence and metadata, while `registered=false` preserves runtime truth |
| Rollback safety | Pass | `POST /v1/plugins/{id}/migrations/rollback` requires `super_admin`, exact plugin-ID confirmation, a positive bounded limit, a disabled plugin, automatic rollback policy, and an applied-step ceiling |
| Durable feedback | Pass | Successful responses include operation ID, completion time, rolled-back step count, and refreshed snapshot; the existing migration audit sink and route audit record the operation and policy |
| Permission behavior | Pass | Handler tests prove non-super-admin denial and exact confirmation; browser UI additionally requires `plugin.manage` and `super_admin` before rendering the action |
| Operator UI | Pass | Routed Data and Migrations pages expose schema namespace, logical/physical tables, fields, keys, indexes, sizes, versions, applied/pending steps, policy consequences, blocked reasons, and explicit confirmation |
| Responsive browser matrix | Pass | Playwright passes fleet plus six workspace routes at 1440x1000 and 390x844 with zero document overflow; final loaded-state screenshots were visually inspected after the first skeleton-timing evidence was rejected |
| Backend tests | Pass | `go test ./internal/handler/http/... ./internal/plugin/... ./internal/bootstrap/...` passes, including API, permission, migration inspection, inactive schema inspection, and lifecycle coverage |
| Frontend quality | Pass | 1,907 locale keys, 1,486 references, zero hard-coded visible strings, accessibility, large-list, theme, strict TypeScript, 8 host files/15 tests, and 2 document files/5 tests pass |
| Production performance | Pass | Vite transforms 3,646 modules; Data is 1.48 KB gzip, Migrations is 2.15 KB gzip, entry is 132,349/140,000 bytes gzip, and all bundle budgets pass |
| API/docs sync | Pass | `docs/api/openapi.yaml` and embedded `internal/handler/http/openapi.yaml` share SHA256 `805EC2113928A267EC4454ACA4A1DBFEE392587DCCD6CE2B5D2C1E083DAC2279`; route, request, response, policy, and action schemas parse |
| Current-only rule | Pass | Only the current API, datastore declaration, migration ledger, and routed workspaces exist; no compatibility payload, alternate schema source, fallback state, or dual route was added |

The first browser evidence captured the final route while its skeleton was still visible. That evidence was rejected, the test was changed to wait for the rendered policy block, and the complete desktop/mobile suite was rerun before visual inspection. A later lifecycle review found that explicit rollback intentionally unregisters active datastore access; candidate-schema inspection was added so disabled plugin data remains observable without reactivating access.

### Verification Commands

```powershell
go test ./internal/handler/http/... ./internal/plugin/... ./internal/bootstrap/...

cd web
npm run typecheck
npm run test:components
$env:SKOLL_E2E_BASE_URL = "http://127.0.0.1:5173"
npm run test:plugin-center
npm run build
npm run check:bundle

cd ..
Compare-Object (Get-Content -Encoding UTF8 docs/api/openapi.yaml) (Get-Content -Encoding UTF8 internal/handler/http/openapi.yaml)
codegraph sync .
codegraph status .
git diff --check
```

Result: BF3-03 passed after rejecting premature visual evidence. Operators can now inspect plugin-owned storage and migration truth and perform only policy-allowed rollback with explicit consequences and durable feedback.

### Impact Review

- API/OpenAPI: one read-only data lifecycle snapshot and one guarded rollback endpoint were added with synchronized runtime/document contracts.
- Permission/audit: read access remains under the plugin workspace; rollback requires `plugin.manage` in the UI and `super_admin` at the authoritative handler, with durable migration and route audit records.
- Migration/data: no migration format changed; existing planner, ledger, transformer, policies, and down scripts are reused.
- Frontend/i18n: two lazy workspace routes and complete zh-CN/en-US, loading, empty, error, blocked, success, mobile, and desktop states were added.
- Performance: both new pages remain independent small async chunks and all existing budgets pass.
- Documentation: module README, OpenAPI, Work Item status, and this acceptance evidence are synchronized.
- Compatibility: none; only the current data-control contract and current routed workspaces are supported.

### Commit

`BF3-03: add plugin data lifecycle control`

## BF3-04 Build Jobs, Audit, Errors, And Diagnostics Views

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Add one plugin-scoped diagnostics contract, namespaced job inspection, combined Host/HTTP audit timelines, derived process/route/job/audit errors, and controlled dead-letter retry.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Authoritative diagnostics API | Pass | `GET /v1/plugins/{id}/diagnostics` aggregates the current health provider, persistent job service, Host audit records, and HTTP audit events; the browser does not infer operational state |
| Plugin isolation | Pass | Job queries are fixed to `plugin.{id}` and full task IDs remain `plugin:{id}:{localId}`; service tests verify only the requested plugin namespace is exposed |
| Correlation chain | Pass | Process, route, job, audit, request, trace, and error IDs are preserved in typed correlation records and navigable from the audit page into filtered diagnostics |
| Error derivation | Pass | Unhealthy process checks, job errors/dead letters, denied routes, failed routes, and failed Host audit records produce separately owned error signals with stable source IDs |
| Dead-letter retry | Pass | `POST /v1/plugins/{id}/jobs/{jobId}/retry` accepts only a dead-letter source, creates a new scheduled task, and leaves the original failed task unchanged |
| Permission and confirmation | Pass | Retry requires `super_admin` plus exact plugin-ID and job-ID confirmations; handler tests cover role denial, confirmation mismatch, successful execution, and durable audit evidence |
| Operator UI | Pass | Lazy Jobs, Audit, and Diagnostics routes provide filters, explicit empty/loading/error states, attempt/error context, retry feedback, audit resources, summary signals, and compact correlation tags |
| Responsive browser matrix | Pass | Playwright passes all nine plugin workspace routes at 1440x1000 and 390x844, validates retry action state and linked error IDs, and reports zero document horizontal overflow |
| Visual review | Pass | Final desktop and mobile screenshots were inspected after loading; desktop exposes the full investigation hierarchy, while mobile keeps navigation and wide tables internally scrollable without overlap |
| Backend tests | Pass | `go test ./internal/handler/http/... ./internal/plugin/... ./internal/bootstrap/...` passes, including diagnostics aggregation, namespace isolation, correlation, retry immutability, role, confirmation, and audit tests |
| Frontend quality | Pass | 1,968 locale keys, 1,524 references, 67 UI files, zero hard-coded visible strings, accessibility, large-list, theme, strict TypeScript, 9 host files/17 tests, and 2 document files/5 tests pass |
| Production performance | Pass | Vite transforms 3,656 modules; Jobs, Audit, and Diagnostics remain lazy chunks at 2.14 KB, 1.88 KB, and 1.92 KB gzip; all bundle budgets pass with entry 134,075/140,000 bytes gzip |
| API/docs sync | Pass | `docs/api/openapi.yaml` and embedded `internal/handler/http/openapi.yaml` share SHA256 `4DEC7B3EC36976818B3707ACC99B1F0BB6C5E68675CB520E69ED320A2E4BE036` and both parse successfully |
| Current-only rule | Pass | Only the current diagnostics and retry contracts exist; no compatibility payload, alternate task store, fallback endpoint, legacy route, or dual execution path was added |

The first browser run exposed an incorrect test assumption: the `admin` fixture is a super administrator, so its retry action was correctly enabled. The ordinary-admin denial remained covered by the authoritative handler test; the browser assertion was corrected and the complete desktop/mobile suite was rerun. Visual evidence was then scrolled to the correlated error table so mobile reachability was inspected rather than inferred from the summary viewport.

### Verification Commands

```powershell
go test ./internal/handler/http/... ./internal/plugin/... ./internal/bootstrap/...

cd web
npm run typecheck
npm run test:components
$env:SKOLL_E2E_BASE_URL = "http://127.0.0.1:5173"
npm run test:plugin-center
npm run build
npm run check:bundle

cd ..
python -c "import yaml; yaml.safe_load(open('docs/api/openapi.yaml', encoding='utf-8')); yaml.safe_load(open('internal/handler/http/openapi.yaml', encoding='utf-8'))"
codegraph sync .
codegraph status .
git diff --check
```

Result: BF3-04 passed after one browser expectation repair. Operators can now investigate one plugin from process health through route, job, audit, and error identifiers, then perform only confirmed and durably audited dead-letter retry.

### Impact Review

- API/OpenAPI: one read-only correlated diagnostics endpoint and one guarded dead-letter retry endpoint were added with synchronized Go, OpenAPI, and TypeScript contracts.
- Permission/audit: diagnostics remain inside the authenticated plugin workspace; retry requires `super_admin`, exact dual confirmation, and writes the source job, retry job, operation, plugin, and actor to durable audit storage.
- Jobs/data: no task schema or worker lifecycle changed; retry creates a new task through the existing persistent job service and preserves the source dead letter.
- Frontend/i18n: three lazy bilingual routes and complete loading, empty, error, filter, success, permission, mobile, and desktop states were added.
- Performance: the three new routes are independent small async chunks and all current JavaScript/CSS budgets pass.
- Documentation: module README, OpenAPI, Work Item status, and this acceptance evidence are synchronized.
- Compatibility: none; only the current diagnostics aggregate and current retry operation are supported.

### Commit

`BF3-04: build plugin diagnostics workspace`
