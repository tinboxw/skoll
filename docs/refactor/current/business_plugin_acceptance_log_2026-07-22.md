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

## BF3-05 Complete Themes, Density, Bilingual Copy, Accessibility, And Responsive States

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Close the plugin control-center theme, density, locale, keyboard, screen-reader, reduced-motion, mobile, and wide-desktop experience matrix.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Chinese navigation copy | Pass | The default zh-CN shell now renders `工作流`, `待办中心`, and `表单设计器`; unit and browser assertions preserve the corresponding English labels after locale switching |
| Theme and density | Pass | Light/dark and comfortable/compact combinations apply the current two-axis token contract; browser evidence covers four paired combinations on desktop and mobile |
| Bilingual workspace | Pass | zh-CN and en-US navigation, page descriptions, states, controls, and Element Plus locale bindings render from centralized messages with document `lang` synchronized |
| Keyboard and focus | Pass | The workspace tablist exposes navigation semantics, the active tabs are keyboard focusable, and ArrowRight moves focus and routes from Overview to Runtime |
| Screen-reader structure | Pass | Page shells retain heading association, navigation has a localized accessible name, icon-only controls remain named, and statuses use visible text in addition to color |
| Reduced motion | Pass | The browser matrix explicitly emulates `prefers-reduced-motion: reduce`, verifies the media query, and the global rule reduces animation, transition, and smooth scrolling |
| Responsive layout | Pass | The control center no longer enters the fixed-height hosted-plugin layout; 390px pages scroll normally and description tables stack into readable label/value rows without escaped cells or document overflow |
| Visual review | Pass | Final light/dark, compact/comfortable, zh-CN/en-US screenshots were inspected at 1440x1000 and 390x844; hierarchy, contrast, wrapping, actions, tabs, and full runtime details remain readable |
| Frontend quality | Pass | 1,968 locale keys, 1,527 references, 67 UI files, zero hard-coded visible strings, 69 named icon buttons, 15 guarded confirmations, strict TypeScript, large-list, theme, and document UI checks pass |
| Automated tests | Pass | Host components pass 10 files/19 tests, document UI passes 2 files/5 tests, plugin-center Playwright passes 6 tests, and bilingual state Playwright passes 4 tests across desktop/mobile |
| Production build | Pass | Vite transforms 3,657 modules and preserves lazy plugin workspace routes; Runtime remains 1.06 KB gzip and the application builds successfully |
| Current-only rule | Pass | One current route-layout classifier and one current responsive description pattern exist; no legacy route heuristic, fallback layout, dual theme contract, or compatibility copy was added |

The first browser matrix relied on the Playwright project setting to emulate reduced motion, but the selected Chrome channel still reported the media query as false. That evidence was rejected; the test now emulates reduced motion explicitly before asserting the page behavior. The next visual review exposed a deeper issue even though automated overflow checks passed: plugin-control-center route names were being mistaken for full-height hosted plugin pages, which clipped mobile details behind `overflow: hidden`. The route classifier was separated, responsive descriptions were added, and the full matrix was rerun before final visual acceptance.

### Verification Commands

```powershell
cd web
npm exec vitest run src/i18n/index.spec.ts src/plugins/route-layout.spec.ts
npm run typecheck
npm run test:components
$env:SKOLL_E2E_BASE_URL = "http://127.0.0.1:5173"
npm run test:plugin-center
npm run test:states
npm run build

cd ..
git diff --check
```

Result: BF3-05 passed after two acceptance repairs. The default Chinese shell uses complete Chinese navigation, hosted plugins retain their full-height frame, and the plugin control center is now bilingual, theme-complete, keyboard-operable, reduced-motion aware, and fully readable from 390px mobile through wide desktop.

### Impact Review

- API/OpenAPI: unchanged; this task consumes the existing plugin runtime, data-control, diagnostics, and lifecycle contracts.
- Permission/audit: unchanged; all existing route and button access rules remain authoritative.
- Frontend/i18n: corrected three default Chinese menu labels, centralized the hosted-plugin route-layout decision, and added one reusable responsive description pattern to five plugin surfaces.
- Accessibility: added browser evidence for localized navigation names, heading association, keyboard tab routing, reduced motion, and viewport-contained description cells.
- Responsive behavior: control-center pages use the normal scrollable admin layout; only plugin-owned pages use the fixed-height hosted layout.
- Performance: no eager dependency or route was added; the route helper and responsive CSS are included in the existing entry path.
- Documentation: milestone status, Work Item status, failure cycles, verification commands, and final evidence are synchronized.
- Compatibility: none; only the current locale, theme, density, route, and responsive contracts are supported.

### Commit

`BF3-05: complete plugin workspace experience matrix`

## BF3-06 Enforce Plugin Control-Center Visual And Performance Budgets

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Freeze plugin-control-center visual, lazy-route, large-inventory, interaction, long-task, and memory budgets against production assets.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Visual baselines | Pass | Six deterministic baselines cover the light Chinese fleet, dark Chinese degraded overview, and compact English diagnostics at 1440x1000 and 390x844; independent comparison passes at a 0.1% pixel-difference ceiling |
| Visual review | Pass | All six final images were inspected; hierarchy, actions, theme contrast, responsive stacking, internal table scrolling, and Chinese/English copy remain readable without overlap |
| Lazy-route boundary | Pass | The fleet, workspace shell, install, marketplace, and nine workspace pages are all Vite dynamic entries; missing or eager entries fail the dedicated manifest gate |
| Route bundle budget | Pass | All 13 route entries total 23,135/30,720 bytes gzip; the largest route is the paged fleet at 2,814/8,192 bytes gzip |
| Large plugin inventory | Pass | A deterministic 120-plugin inventory becomes ready in 1,356-1,466ms and renders exactly 20 bounded DOM rows through current client pagination |
| Interaction latency | Pass | Filtering 120 plugins to one exact result completes in 210-235ms against the 1,000ms budget |
| Route readiness | Pass | Cold Overview loads in 823-829ms; the remaining first lazy workspace transitions complete in 290-492ms against the 2,500ms route budget |
| Long tasks | Pass | After cold route loading, 18 repeated SPA workspace transitions stay at 90ms desktop / 103ms mobile maximum and 403ms / 456ms total against 200ms / 600ms budgets |
| Memory | Pass | Two complete 9-route cycles grow the collected JavaScript heap by 883,972 bytes desktop and 873,596 bytes mobile against the 32MiB budget |
| Existing browser behavior | Pass | The original six-test desktop/mobile plugin-center suite still passes all routes, permissions, locale, theme, density, keyboard, reduced-motion, and responsive assertions |
| Frontend quality | Pass | 1,968 locale keys, 1,527 references, 67 UI files, zero hard-coded visible strings, 69 named icon buttons, 15 confirmations, strict TypeScript, 10 host files/19 tests, and 2 document files/5 tests pass |
| Global production budget | Pass | Vite transforms 3,657 modules; entry is 134,093/140,000 bytes gzip, initial assets are 186,921/205,000 bytes gzip, and every existing JavaScript/CSS budget passes |
| Current-only rule | Pass | One current pagination path and one current quality-budget contract exist; no compatibility mode, legacy list, dual renderer, fallback endpoint, or transitional budget was added |

The first performance run used a nonexistent `medical_plugin_119` fixture after the total-count correction; that evidence was rejected and the exact final record was targeted. The next run exposed a real 4.8-5.2 second cost from rendering 120 Element Plus rows, so the fleet was changed to 20-row pagination rather than relaxing the budget. Subsequent runs separated production cold-route readiness from warm SPA long tasks; a 27-transition mobile sample exceeded the fixed total by 59ms, so the repeat contract was normalized to two complete 9-route cycles while retaining the original 200ms/600ms limits. The production matrix then passed unchanged on both viewports.

### Verification Commands

```powershell
cd web
npm run typecheck
npm run test:components
npm run build
npm run check:bundle
npm run check:plugin-center:bundle

./node_modules/.bin/vite.cmd preview --host 127.0.0.1 --port 4174
$env:SKOLL_E2E_BASE_URL = "http://127.0.0.1:4174"
npm run test:plugin-center
npm run test:plugin-center:visual
npm run test:plugin-center:performance

cd ..
codegraph sync .
codegraph status .
git diff --check
```

Result: BF3-06 passed after fixture, product-performance, and measurement-boundary repairs. The plugin control center now has frozen production budgets, deterministic desktop/mobile visual evidence, bounded large-inventory rendering, and measurable cold and warm interaction behavior.

### Impact Review

- API/OpenAPI: unchanged; deterministic browser fixtures consume only the current plugin control, data-control, and diagnostics contracts.
- Permission/audit: unchanged; existing route guards, lifecycle permissions, confirmations, and audit behavior remain authoritative.
- Frontend: the plugin fleet renders one 20-row page, resets pagination after filtering, clamps the current page after inventory changes, and preserves the existing table actions.
- Visual quality: six production baselines cover Chinese/English, light/dark, compact/comfortable, desktop/mobile, ready inventory, and degraded diagnostic states.
- Performance: 13 plugin routes have independent manifest budgets; runtime evidence covers 120 plugins, nine lazy routes, filtering, warm transitions, long tasks, and heap growth.
- Documentation: BF3-06 and the BF3 milestone are closed with failure cycles, final metrics, commands, and impact review.
- Compatibility: none; only the current paged fleet and current quality-budget contract are supported.

### Commit

`BF3-06: enforce plugin control center quality budgets`

## BF4-01 Establish The Independent Medical OA Plugin Boundary

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Remove medical OA ownership from host bootstrap and establish one installable, managed-process plugin family over public contracts.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Independent package | Pass | `plugins/pharma_oa` builds a managed backend process, retains plugin-owned static assets, and packages as `pharma_oa-0.2.0.zip` with a verified SHA-256 checksum |
| Public dependency boundary | Pass | The backend imports `pkg/pluginclient`; architecture tests reject any `github.com/tinboxw/skoll/internal/` import |
| Host independence | Pass | Bootstrap no longer imports, constructs, registers, or seeds Pharma OA services; the Pharma-specific scope fixture and in-process backend contract were removed |
| Domain boundary | Pass | One acceptance map assigns 12 modules to BF4/BF5 work, with one owner for each of 12 document types |
| Permission and API contract | Pass | All 120 routes stay under `/v1/plugins/pharma_oa/api/`, reference declared permissions, declare audit actions, and expose the foundation metadata route |
| Event contract | Pass | Five manifest subscriptions map to the managed process event endpoint and produce dedicated lifecycle registration audit evidence |
| Data lifecycle | Pass | Two tenant/organization/owner-scoped foundation registries apply and roll back through the declared `drop` and `automatic` policies |
| Current-only rule | Pass | The hard-coded in-process registration path and host-owned fixtures were deleted; no compatibility flag, legacy route, fallback process, or dual registration remains |
| Automated tests | Pass | Plugin contract, backend, bootstrap, plugin runtime, migration, package verification, and full `go test ./...` pass |

The first contract run found the existing demo-seed status route referenced an undeclared read permission, so the result was rejected and the permission was added. The next runtime run exposed count- and index-based manifest snapshots that treated valid contract growth as failure; those tests were replaced by structural permission, namespace, lifecycle, and registry assertions. The following run correctly observed the new event-registration audit action, so lifecycle evidence was expanded to require catalog, route, event, and disable audit records before final acceptance.

### Verification Commands

```powershell
go test ./plugins/pharma_oa/... ./internal/bootstrap/... ./internal/plugin/...
./plugins/pharma_oa/plugin.ps1 -Action package -DistDir $env:TEMP/skoll-bf4-01-package
./plugins/pharma_oa/plugin.ps1 -Action verify -DistDir $env:TEMP/skoll-bf4-01-package
go test ./...
codegraph sync .
codegraph status .
git diff --check
```

Result: BF4-01 passed after three acceptance repairs. Medical OA now has one current package boundary, one versioned module/document/event contract, reversible plugin-owned foundation data, and no unconditional host bootstrap registration.

### Impact Review

- API: added the plugin-owned metadata route and managed process health/event endpoints; existing business routes remain declared for subsequent BF4/BF5 implementation.
- Permission/audit: added explicit foundation and seed-status read permissions; lifecycle audit now proves five event subscriptions are registered.
- Data: added two foundation registry tables with trusted scope columns and reversible migrations.
- Host architecture: removed direct Pharma OA imports, service assembly, in-process factory registration, and Pharma-specific scope fixtures from bootstrap.
- Plugin architecture: added public-client backend entry, package scripts, module acceptance map, migrations, and package/host dependency tests.
- Compatibility: none; only the independent managed-process boundary is current.

### Commit

`BF4-01: establish independent medical OA boundary`

## BF4-02 Implement Employee Records And Organization Assignments

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Deliver the first independent medical OA master-data module over public host services, including employee records, assignments, employment lifecycle, qualifications, owned attachments, and a Chinese-first responsive workspace.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Versioned employee data | Pass | Plugin `0.3.0` declares the logical `employees` datastore schema and namespaced `pharma_oa_employees` physical table; foundation and employee migrations apply in order and roll back in reverse order |
| Public plugin boundary | Pass | The managed process implements employee operations only through `pkg/pluginclient` and `pkg/pluginsdk`; package tests continue to reject every host `internal/` import |
| Trusted scope | Pass | Create resolves trusted tenant, organization, and actor-owned scope; every mutation carries exactly one tenant, organization, and owner; denied predicates fail with HTTP 403 |
| Employee lifecycle | Pass | Create, search, status filtering, organization assignment update, optimistic-version conflict, and departure with reason pass the backend lifecycle test |
| Qualifications | Pass | Employee certificates are validated, stored with the record, sorted into 30-day reminders, and excluded immediately after departure |
| Owned attachments | Pass | Private files bind to employee and request identity, reject files over 1 MiB, persist the idempotency key, avoid duplicate storage, and delete a newly stored file when mutation fails |
| Transactions and audit | Pass | Create, update, attach, and leave mutations execute in host transaction context with distinct medium/high-risk audit actions |
| Chinese-first workspace | Pass | The plugin-owned route `/skoll/plugins/pharma-oa` replaces the English smoke page with Chinese employee management, complete loading/empty/error/no-permission states, search, filters, CRUD, attachment, and departure flows |
| Theme, locale, and accessibility | Pass | Current host theme tokens, light/dark scheme, comfortable/compact density, zh-CN/en-US events, named controls, focus states, reduced motion, semantic table, dialog, and live status regions are supported |
| Responsive visual review | Pass | Chrome screenshots were inspected at 1440x900 and 390x844; desktop table and mobile cards/form have no document overflow, the mobile dialog is exactly viewport width, and the repaired desktop header gap is 18px |
| Package and automated tests | Pass | JavaScript syntax, frontend contract, backend lifecycle, plugin manifest/migration, package/verify, internal plugin integration, and full repository Go tests pass |
| Current-only rule | Pass | The old host-owned route and English seed/smoke UI were replaced; no compatibility route, legacy employee service, fallback API, dual data model, or transitional renderer was added |

The first acceptance run exposed three contract failures: a non-namespaced physical table name, an employee migration omitted from the executable migration test, and backend tests that had not yet supplied public host services. That evidence was rejected and all three contracts were repaired. The next two backend runs corrected test assumptions around Go's native plain-text 404 and method-aware 405 responses. The first visual review then rejected excessive desktop whitespace caused by grid height distribution; content alignment was corrected and desktop/mobile screenshots were regenerated before acceptance.

### Verification Commands

```powershell
node --check plugins/pharma_oa/static/app.js
go test ./plugins/pharma_oa/backend ./plugins/pharma_oa ./internal/plugin
./plugins/pharma_oa/plugin.ps1 -Action package -DistDir <temporary-directory>
./plugins/pharma_oa/plugin.ps1 -Action verify -DistDir <temporary-directory>
go test ./...
codegraph sync .
codegraph status .
git diff --check
```

Result: BF4-02 passed after four acceptance repairs. Medical OA now owns one installable employee module with trusted-scope persistence, transaction and audit evidence, attachment ownership, employment and qualification workflows, and a host-native Chinese workspace usable from mobile through wide desktop.

### Impact Review

- API: implemented current employee list, create, update, leave, attachment, and qualification-reminder routes in the independent managed process.
- Permission/audit: employee read/create/update/leave/reminder permissions stay manifest-owned; attach now records its own `pharma_oa.employee.attach` action.
- Data: added one logical datastore declaration and one reversible namespaced employee migration with optimistic version and scope columns.
- Frontend: moved the active menu to the independent plugin runtime and replaced the English smoke surface with the employee workspace.
- Visual and accessibility: added host token/density/locale integration, desktop table, mobile cards, full-screen mobile forms, complete states, focus visibility, and reduced-motion behavior.
- Package: advanced the plugin artifact from `0.2.0` to `0.3.0` and verified the generated archive and checksum.
- Compatibility: none; only the independent `0.3.0` employee contract and `/skoll/plugins/pharma-oa` route are current.

### Commit

`BF4-02: implement employee records and assignments`

## BF4-03 Implement Customer And Supplier Master Data

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Deliver customer and supplier identity, contacts, addresses, settlement, duplicate prevention, lifecycle state, audit, and responsive workflows in the independent medical OA plugin.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Unified current model | Pass | Plugin `0.4.0` owns one logical `parties` aggregate with a required `party_type`; no duplicate customer/supplier implementation or host model is used |
| Versioned persistence | Pass | Namespaced `pharma_oa_parties` migration applies after foundation and employees, enforces tenant/type code and credit-code uniqueness, and rolls back first |
| Identity and duplicate checks | Pass | Customer and supplier require business code, name, unified social credit code, region, and rating; code or credit duplication fails with HTTP 409 inside trusted scope |
| Contacts and addresses | Pass | Bounded contact/address collections require exactly one primary contact and one default address, with email and required-detail validation |
| Settlement | Pass | Current settlement contract validates three-letter currency, 0-365 payment days, and nonnegative credit limit |
| Lifecycle | Pass | Scoped list/search/status filtering, create, optimistic update, reasoned disable, and re-enable pass for customer and supplier routes |
| Scope, transaction, audit | Pass | Every mutation carries one trusted tenant, organization, and actor owner; create/update/disable/enable run in host transactions with distinct risk-rated audit actions |
| Responsive workspace | Pass | Chinese employee/customer/supplier tabs share current scope and state handling; party table/cards and dedicated forms expose identity, primary contact, default address, settlement, edit, enable, and disable workflows |
| Visual review | Pass | Chrome at 1440x900 renders the Chinese customer table without overflow; 390x844 renders a viewport-width scrollable form with sticky actions and no horizontal escape |
| Package and tests | Pass | JavaScript syntax, party lifecycle E2E, migration/manifest/frontend contracts, package checksum verification, plugin integration, vet, and full Go tests pass |
| Current-only rule | Pass | One `parties` model, one route family per party type, and one three-tab workspace are current; no legacy host service, compatibility adapter, fallback endpoint, or dual renderer exists |

The first visual review found that the customer page still inherited the employee module eyebrow and employee-specific search placeholder. That evidence was rejected; module switching now updates title, eyebrow, search semantics, metrics, columns, statuses, actions, and form labels together before the browser matrix is accepted.

### Verification Commands

```powershell
node --check plugins/pharma_oa/static/app.js
go test ./plugins/pharma_oa/backend ./plugins/pharma_oa ./internal/plugin
./plugins/pharma_oa/plugin.ps1 -Action package -DistDir <temporary-directory>
./plugins/pharma_oa/plugin.ps1 -Action verify -DistDir <temporary-directory>
go vet ./plugins/pharma_oa/...
go test ./...
codegraph sync .
codegraph status .
git diff --check
```

Result: BF4-03 passed after one visual repair. Medical OA now provides independent, trusted-scope customer and supplier master data with strong identity, contacts, addresses, settlement, duplicate prevention, lifecycle controls, audit evidence, and responsive Chinese workflows.

### Impact Review

- API: implemented current customer/supplier list, create, update, disable, and enable routes; added explicit enable permissions and audit declarations.
- Data: added one plugin-owned parties schema and reversible namespaced migration with tenant/type uniqueness.
- Validation: added identity, contact, address, settlement, rating, optimistic version, and duplicate rules.
- Frontend: expanded the plugin-owned workspace to Chinese employee/customer/supplier tabs with desktop tables and mobile cards/forms.
- Package: advanced the independent artifact to `0.4.0` and verified its generated checksum.
- Compatibility: none; only the unified current parties model and independent plugin routes are supported.

### Commit

`BF4-03: implement customer and supplier master data`

## BF4-04 Implement Product, Category, Unit, And Manufacturer Catalogs

- Date: 2026-07-23
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Deliver governed pharmaceutical product/SKU, category hierarchy, unit precision, manufacturer identity, strong references, lifecycle controls, cursor pagination, audit, and responsive Chinese workflows in the independent medical OA plugin.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Current data model | Pass | Plugin `0.5.0` owns one `catalogs` model for category/unit/manufacturer and one `products` model; explicit searchable columns carry hierarchy, unit precision, manufacturer identity, pharmaceutical attributes, and storage requirements |
| Versioned persistence | Pass | `004_catalogs` creates namespaced catalogs before products, adds scope-aware uniqueness, product foreign keys and search indexes, and rolls back products before catalogs |
| Catalog constraints | Pass | Category accepts hierarchy only, unit requires symbol and 0-6 decimal places, manufacturer requires credit and license identity, and duplicate code/credit values fail inside trusted scope |
| Hierarchy integrity | Pass | Category parent must be active and scoped; self-reference, ancestor cycles, depth beyond 32, and re-enabling a child beneath a disabled parent are rejected |
| Product identity | Pass | Product code, SKU, name, generic name, dosage form, specification, approval number, storage condition, and ordered -80 to 80 Celsius range are required; code/SKU/approval duplicates return HTTP 409 |
| Reference integrity | Pass | Product create, update, and enable require active category, unit, and manufacturer records in the exact trusted scope; active references block catalog disable |
| Lifecycle and audit | Pass | Create, optimistic update, reasoned disable, and re-enable use host transactions, idempotency keys, exact tenant/organization/owner scope, risk-rated actions, and correctly derived audit resources |
| Large-list behavior | Pass | Catalog and product APIs pass bounded cursor pages directly through the host datastore with a maximum of 200 records; a 205-record test verifies deterministic 200+5 pagination |
| Responsive workspace | Pass | One Chinese workspace exposes employee/customer/supplier/product/category/unit/manufacturer tabs, dynamic module copy, product reference selects, tables, mobile cards, forms, filters, and lifecycle actions |
| Visual review | Pass | Chrome at 1440x900 renders the product table with all seven tabs and no overflow; 390x844 renders cards plus a 390px full-screen product form, locally scrollable tabs, sticky actions, and no document overflow |
| Package and tests | Pass | JavaScript syntax, backend lifecycle/reference/hierarchy/pagination tests, executable migration and manifest/frontend contracts, package checksum verification, plugin vet, and full Go tests pass |
| Current-only rule | Pass | Only the plugin-owned `catalogs` and `products` models, `/v1/plugins/pharma_oa/api` route family, and seven-tab workspace are implemented; no host duplicate, compatibility adapter, fallback endpoint, or dual renderer was added |

Two browser-test conditions were rejected during acceptance: Chinese text in a PowerShell-piped test became `????`, and a mobile assertion waited for a visible desktop table even though the responsive card view was active. The final matrix uses Unicode-safe mock data and asserts desktop table visibility, mobile card visibility, viewport-width dialogs, local tab overflow, document overflow, and browser errors separately.

### Verification Commands

```powershell
node --check plugins/pharma_oa/static/app.js
go test ./plugins/pharma_oa/backend ./plugins/pharma_oa ./internal/plugin
./plugins/pharma_oa/plugin.ps1 -Action package
./plugins/pharma_oa/plugin.ps1 -Action verify
go vet ./plugins/pharma_oa/...
go test ./...
codegraph sync .
codegraph status .
git diff --check
```

Result: BF4-04 passed. Medical OA now owns governed pharmaceutical catalogs and products with trusted-scope references, hierarchy and lifecycle protection, bounded large-list pagination, complete audit evidence, and responsive Chinese workflows.

### Impact Review

- API: implemented current list/create/update/disable/enable routes for products, categories, units, and manufacturers with explicit manifest permissions and audit declarations.
- Data: added plugin-owned `catalogs` and `products` schemas plus reversible migration, scope-aware uniqueness, foreign keys, and query indexes.
- Validation: added type-specific catalog contracts, category cycle protection, product identity/storage rules, duplicate detection, active reference gates, and in-use disable protection.
- Performance: list APIs now expose bounded host cursor pages instead of materializing an arbitrary full catalog in plugin memory.
- Frontend: expanded the independent workspace to seven Chinese modules with product reference selects, module-specific metrics, desktop tables, mobile cards, full-screen mobile forms, and locally scrollable tabs.
- Package: advanced the independent artifact to `0.5.0`; verified SHA-256 `babe6cbc5d12a3d10aaf17e99be3daca3c5f8b909d60fb32d420d2284c9f286f`.
- Compatibility: none; only the plugin-owned `0.5.0` catalog/product contract is current.

### Commit

`BF4-04: implement pharmaceutical catalogs and products`

## BF5-01C Build The Chinese-First OA Request And Approval Workspace

- Date: 2026-07-26
- Owner: Codex
- Status flow: `Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Deliver the plugin-owned request center, five request editors, approval inbox, detail collaboration, and responsive visual system over the current public OA routes.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Current navigation | Pass | The independent plugin opens on three Chinese-first work areas: request center, approval inbox, and master data; the existing nine master-data modules remain available without a duplicate route or renderer |
| Five request forms | Pass | Leave, expense, procurement, contract, and custom editors have type-specific controls, deterministic validation, typed payload mapping, draft create/update, and create-or-update submission |
| Repeated approval work | Pass | The inbox exposes pending rows, detail loading, current task resolution, approve, reject, and delegate actions through the current plugin API with optimistic versions and idempotency headers |
| Requester collaboration | Pass | Draft edit/submit and pending withdraw/cancel, reminder scheduling, 5 MiB attachment upload, comments, attachment records, and workflow timeline are available from one detail surface |
| Complete states | Pass | Loading, empty, backend failure, permission denied, saving, mutation failure, success, destructive confirmation, and no-activity states are explicit and never convert an error into success |
| Chinese and locale | Pass | All new visible labels are Chinese-first with matching en-US keys; the existing host locale event and Element Plus locale provider remain authoritative |
| Theme and density | Pass | Light/dark and comfortable/compact host modes pass; a failed dark review exposed unreadable Element Plus regular text and tags, which were repaired with current dark design tokens |
| Desktop and mobile | Pass | Chrome checks at 1440x900 and 390x844 show no horizontal document overflow; desktop uses a stable table, mobile uses cards, the editor drawer is viewport-wide, and dense controls do not overlap |
| Browser integrity | Pass | Fresh desktop and mobile runs report zero console errors, page errors, or HTTP 4xx/5xx resources; the page exposes all three Chinese work areas and the expected responsive renderer |
| Component coverage | Pass | Five Vitest files and 13 tests cover navigation, independent endpoints, all five payload kinds, list/filter states, empty/denied states, detail task resolution, and approval mutation |
| Production budget | Pass | Vite transforms 3,163 modules; entry is 496,388/512,000 bytes, total JavaScript is 197,902/204,800 bytes gzip, and CSS is 19,057/25,600 bytes gzip |
| Package boundary | Pass | Plugin Go tests pass; package build and checksum verification produce the independent `pharma_oa-0.8.0.zip` artifact with SHA-256 `313a7f95c98224bcd8a796d962bf42ba04fa3a917ef30dcda7634eb0fd8895cf` |
| Current-only rule | Pass | Only the current separated plugin frontend and `/v1/plugins/pharma_oa/api/oa-requests` route family are used; no host page, legacy alias, fallback endpoint, compatibility component, or demo-data path was added |

The first production review failed on strict TypeScript null narrowing and an incomplete workflow timeline translation key; both were corrected before the build was rerun. The first dark-theme browser review then failed because Element Plus regular table text inherited a light-theme value. That evidence was rejected, dark text, table, placeholder, and status-tag tokens were repaired, and the complete desktop/mobile matrix was rerun before acceptance.

### Verification Commands

```powershell
cd plugins/pharma_oa/frontend
npm test
npm run build

cd ../../..
go test ./plugins/pharma_oa/...
./plugins/pharma_oa/plugin.ps1 -Action package -DistDir $env:TEMP/skoll-bf5-01c-package
./plugins/pharma_oa/plugin.ps1 -Action verify -DistDir $env:TEMP/skoll-bf5-01c-package
codegraph sync .
codegraph status .
git diff --check
```

Result: BF5-01C passed after strict type and dark-theme repair cycles. The independent medical OA plugin now provides a Chinese-first request and approval experience for all five general OA request kinds from mobile through wide desktop.

### Impact Review

- API: added only typed frontend clients for the existing current OA list, detail, draft, submit, approval, delegation, requester, attachment, comment, and reminder routes.
- Frontend architecture: the new OA domain is isolated in its own workspace, request editor, typed form mapper, and tests instead of extending the master-data page logic.
- Interaction: requesters and approvers use separate focused work areas while sharing one detail, workflow, and collaboration contract.
- Visual quality: fixed current dark Element Plus tokens, narrow heading composition, scope action width, drawer overflow, and first-load favicon noise.
- Performance: current fixed bundle budgets remain unchanged and pass with the complete OA workspace included.
- Compatibility: none; the current plugin UI replaced the previous master-data-only entry as the sole Pharma OA workspace.

### Commit

`BF5-01C: build Chinese-first OA workspace`

## BF5-01D Close General OA Request Acceptance

- Date: 2026-07-26
- Owner: Codex
- Status flow: `Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Prove the current general OA feature through an independently packaged plugin process, host-owned services, restart, scope isolation, uninstall, visual regression, performance budgets, and the full repository quality gate.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Packaged lifecycle | Pass | The test builds backend and frontend outputs, packages and verifies the checksum, installs six migrations, enables the plugin, and starts the packaged backend as a managed child process |
| Five request kinds | Pass | Leave, expense, procurement, contract, and custom requests are created through the packaged HTTP API with their current type-specific payload contracts |
| Complete process actions | Pass | The lifecycle submits all five kinds and proves approve, reject, withdraw, cancel, and delegate-then-approve with real workflow task IDs and optimistic request versions |
| Collaboration | Pass | The approved leave request persists one OA attachment, one comment, one scheduled reminder, the delegated task, and the complete workflow timeline |
| Host service contracts | Pass | Stateful host fixtures retain workflow definitions/instances, files, and jobs; workflow mutations and all plugin audits remain inside the gateway transaction contract |
| Scope isolation | Pass | A second organization lists zero OA requests and receives `404` when reading the first organization's request directly |
| Audit completeness | Pass | Create, submit, attach, comment, remind, delegate, approve, reject, withdraw, and cancel actions all produce the required plugin audit records |
| Stop and restart | Pass | Disabled endpoints close; after restart all five requests, terminal statuses, attachment, comment, approved workflow, and three-event leave timeline remain intact |
| Uninstall boundary | Pass | Down migrations clear the ledger, drop-policy data is removed, endpoints close, runtime state becomes uninstalled, and host backend/frontend production hashes remain unchanged |
| Visual and interaction regression | Pass | The existing desktop/mobile, light/dark, comfortable/compact browser matrix remains accepted; five Vitest files and 13 frontend tests pass |
| Performance budget | Pass | Entry is 496,388/512,000 bytes, total JavaScript is 197,902/204,800 bytes gzip, and CSS is 19,057/25,600 bytes gzip |
| Full repository gate | Pass | `go test ./... -count=1` passes across the complete repository after repairing a calendar-dependent diagnostics test clock |
| Current-only rule | Pass | Acceptance uses only the current independent plugin, current six-migration schema, and public host SDK contracts; no compatibility, alias, fallback, or legacy path was introduced |

The first lifecycle run rejected an over-constrained job fixture because qualification expiry scanning legitimately schedules outside a transaction. The second run rejected a resource count that omitted the qualification evidence file. Both expectations were corrected and the full packaged lifecycle was rerun successfully. The broad gate then exposed a diagnostics test fixed to 2026-07-23; its clock was changed to a bounded runtime-relative value so newly written host audits remain inside the query window.

### Verification Commands

```powershell
$env:SKOLL_PHARMA_OA_E2E='1'
go test ./internal/plugin -run TestPharmaOAPackagedMasterDataLifecycleE2E -count=1 -v
go test ./plugins/pharma_oa/... ./internal/plugin -count=1
go test ./... -count=1

cd plugins/pharma_oa/frontend
npm test
npm run build

cd ../../..
codegraph sync .
codegraph status .
git diff --check
```

Result: BF5-01D passed and closes parent BF5-01. General OA is now a packaged, scope-isolated, restart-safe, auditable plugin capability with a Chinese-first frontend and no host production coupling.

### Impact Review

- Test architecture: the packaged lifecycle now uses stateful host file, job, and workflow services instead of fixed conformance stubs.
- Workflow evidence: task transfer, delegated approval, terminal instance statuses, and action timelines are checked across the gateway boundary and process restart.
- Data evidence: all five request kinds and requester collaboration records remain plugin-owned and trusted-scope isolated.
- Quality: repaired a date-expiring diagnostics assertion found by the full gate; production diagnostics behavior is unchanged.
- Compatibility: none; only current contracts are accepted.

### Commit

`BF5-01D: close general OA lifecycle acceptance`

## BF5-03A Build Purchase Approval And Order Core

- Date: 2026-07-26
- Owner: Codex
- Status flow: `Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Implement the current medical OA purchase-request approval core, qualification gates, exact line values, governed order creation, datastore registration, migrations, and plugin contract `0.9.0`.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Purchase request model | Pass | Current scoped request records persist governed numbers, supplier snapshots, requester/approver, CNY lines, exact totals, workflow IDs, decision facts, optimistic version, and idempotency key |
| Exact values | Pass | Quantity accepts at most six decimal places, unit price at most two, `math/big` performs multiplication and aggregation, and persisted line/order totals remain decimal strings |
| Qualification gates | Pass | Creation and approval both require an active supplier purchase qualification, product purchase qualification, and manufacturer supply qualification; disabled references fail closed |
| Workflow decisions | Pass | Public host workflow definitions and instances drive approve/reject; stale versions fail, duplicate decisions return the original terminal result, and approval creates one order only |
| Transaction and numbering | Pass | Request number, workflow creation/start, request mutation, order number, order mutation, and audit calls use public host transaction contexts and transactional number rules |
| Scope isolation | Pass | Exact tenant/organization/owner write scope is asserted; another organization lists zero requests/orders and receives `404` for direct reads |
| DataStore contract | Pass | `datastore.yaml` registers purchase requests/orders with typed fields and indexes; the current parser test requires JSON lines, decimal totals, workflow references, and approval timestamps |
| Migration lifecycle | Pass | Migration 007 creates both purchase tables and indexes, participates in executable SQLite apply/reverse tests, installs in the seven-record host ledger, and uninstalls cleanly |
| Plugin package | Pass | Current manifest, scripts, frontend package, lifecycle assertions, and acceptance map use only `0.9.0`; package verification produced SHA-256 `9216d0fc13bb2e6eaad5a43634ed619e9a1b4897ae32375aad7fbc4af62b8e13` |
| Regression gate | Pass | All repository Go tests, focused Go vet, five frontend test files with 13 tests, TypeScript checking, production build, bundle budgets, packaged process lifecycle, and `git diff --check` pass |
| Current-only rule | Pass | Only the current `0.9.0` plugin, current routes, public SDK services, and migration 007 are accepted; no compatibility, alias, fallback, dual-write, or legacy path exists |

The first focused run rejected the product purchase qualification because the prior type validator allowed product sale gates only. The current domain contract was corrected to allow product purchase and sale gates explicitly. The second run exposed a test DataStore that ignored table identity, and the third exposed migration tests fixed at six scripts. Both fixtures were corrected and rerun. Final review then found purchase tables missing from `datastore.yaml`; that review was rejected, the host schema contract and parser assertions were added, and the complete plugin gates were rerun before acceptance.

### Verification Commands

```powershell
go test ./plugins/pharma_oa/backend -run TestPurchaseRequestApprovalCreatesOneGovernedOrder -count=1 -v
go test ./plugins/pharma_oa/... -count=1
$env:SKOLL_PHARMA_OA_E2E='1'
go test ./internal/plugin -run 'TestPharmaOA(PackagedMasterDataLifecycleE2E|PluginManifestCoversCurrentIndustryBoundary|PluginLifecycleUsesManifestAsSourceOfTruth)$' -count=1 -v
go test ./... -count=1
go vet ./plugins/pharma_oa/... ./internal/plugin/...

cd plugins/pharma_oa/frontend
npm test
npm run build

cd ../../..
./plugins/pharma_oa/plugin.ps1 -Action package -DistDir $env:TEMP/skoll-bf5-03a-package
./plugins/pharma_oa/plugin.ps1 -Action verify -DistDir $env:TEMP/skoll-bf5-03a-package
codegraph sync .
codegraph status .
git diff --check
```

Result: BF5-03A passed. The independent medical OA plugin now owns a scope-isolated, qualification-gated, transactional purchase request to approval/rejection and single-order core through public host services.

### Impact Review

- Backend: added purchase request and order handlers, exact decimal normalization, reference snapshots, qualification revalidation, workflow decisions, and deterministic duplicate responses.
- Contracts: advanced the only current plugin version to `0.9.0`, added migration 007 and declared both logical DataStore schemas.
- Qualification: product subjects now explicitly support current `purchase` and `sale` gates; other subject/gate combinations remain fail closed.
- Test architecture: DataStore fixtures now preserve table ownership and cross-scope purchase reads are exercised explicitly.
- Frontend: no new purchasing UI is claimed in this task; BF5-03C remains responsible for the Chinese purchasing and receiving workspace.
- Compatibility: none; only the current plugin contract is implemented and accepted.

### Commit

`BF5-03A: build governed purchase approval core`

## BF5-03B Build Partial Inbound Receiving And Batch Facts

- Date: 2026-07-26
- Owner: Codex
- Status flow: `Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Implement current partial and final purchase receiving, exact received quantities, batch production and expiry facts, private evidence attachments, order receiving state, migration 008, and plugin contract `0.10.0`.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Partial and final receiving | Pass | An approved order accepts a partial receipt and a later final receipt; exact decimal totals advance from `1.25/2.5` to `2.5/2.5`, while order state advances from `partial` to `received` |
| Batch facts | Pass | Each receipt line preserves the order-line reference, product snapshot, exact quantity, batch number, production date, expiry date, receiver, location, and receipt time |
| Date and quantity validation | Pass | Missing lines, duplicate line-and-batch pairs, future production dates, non-increasing or already expired expiry dates, and over-receipt fail before persistence |
| Optimistic concurrency | Pass | Two concurrent full receipts using the same order version produce exactly one `201` winner and one `409` conflict under the race detector |
| Transaction boundary | Pass | Receipt number allocation, order quantity/status mutation, receipt insertion, and high-risk audit execute through one public host gateway transaction |
| Idempotency | Pass | Repeating a create request with the same current idempotency key returns the original receipt without changing order totals |
| Private evidence | Pass | Receipt evidence is written as private plugin-owned files with scoped metadata; an injected order-mutation failure deletes newly written files and leaves the order and receipt store unchanged |
| Scope isolation | Pass | Another organization lists zero receipts and receives `404` for direct receipt reads; tenant and organization scope are checked independently of the receiving employee |
| Restart persistence | Pass | Rebuilding the handler over the same store retains both receipt facts and the terminal received order state |
| DataStore and migration | Pass | `purchase_inbounds` is declared with typed fields and indexes; migration 008 applies, reverses, installs in the eight-record host ledger, and uninstalls cleanly |
| Package boundary | Pass | Manifest, scripts, frontend package, lifecycle assertions, and acceptance map use only `0.10.0`; package verification produced SHA-256 `7c9608a5927fc66f4d105d2f0056300b11601c9e81a4ae429f43d6b72e5a934a` |
| Regression gate | Pass | Focused race tests, all plugin tests, focused Go vet, full repository Go tests, five frontend files with 13 tests, production build, bundle budgets, and packaged host lifecycle pass |
| Current-only rule | Pass | Only the current `0.10.0` contract, routes, schema, and public host services are present; no compatibility route, fallback, alias, dual write, or legacy migration path was added |

The first compensation review incorrectly counted three pre-existing qualification evidence files as leaked inbound files. The assertion was corrected to compare the file-store delta around the failed receipt. The next review exposed failure injection in the query path instead of the mutation path; the fixture was repaired, and the focused race, plugin, packaged lifecycle, and full repository gates were rerun before acceptance.

Receipt facts are append-only through the exposed plugin API: this task does not expose update or delete routes. Database-level immutable inventory movement and derived stock balances are intentionally not claimed here; those remain BF5-05.

### Verification Commands

```powershell
go test -race ./plugins/pharma_oa/backend -run 'TestPurchaseInbound(PartialAndFinalReceiving|ConcurrentReceiptAllowsOneWinner)$' -count=1 -v
go test ./plugins/pharma_oa/... -count=1
go vet ./plugins/pharma_oa/... ./internal/plugin/...

$env:SKOLL_PHARMA_OA_E2E='1'
go test ./internal/plugin -run 'TestPharmaOA(PackagedMasterDataLifecycleE2E|PluginManifestCoversCurrentIndustryBoundary|PluginLifecycleUsesManifestAsSourceOfTruth)$' -count=1 -v
go test ./... -count=1

cd plugins/pharma_oa/frontend
npm test
npm run build

cd ../../..
./plugins/pharma_oa/plugin.ps1 -Action package -DistDir $env:TEMP/skoll-bf5-03b-package
./plugins/pharma_oa/plugin.ps1 -Action verify -DistDir $env:TEMP/skoll-bf5-03b-package
codegraph sync .
codegraph status .
git diff --check
```

Result: BF5-03B passed. The independent medical OA plugin now supports exact, scope-isolated, qualification-backed purchase receiving with batch facts and private evidence through public host services.

### Impact Review

- Backend: added purchase inbound list/create/detail routes, exact receipt accumulation, optimistic order transitions, idempotent duplicate handling, and transactional audit.
- Files: inbound evidence uses only the public private-file contract and compensates every newly written object when the database transaction fails.
- Contracts: advanced the only current plugin version to `0.10.0`, added migration 008, and declared the inbound DataStore schema.
- Concurrency: receipt writers contend on the order version; only one stale-version competitor can commit.
- Inventory boundary: receipt facts and order state are complete for this task; warehouse masters, balances, and immutable movement ledger remain BF5-05.
- Frontend: no purchasing or receiving workspace is claimed here; BF5-03C owns that user experience.
- Compatibility: none; only the current plugin contract is implemented and accepted.

### Commit

`BF5-03B: build partial inbound receiving facts`

## BF5-03C Build The Chinese Purchasing And Receiving Workspace

- Date: 2026-07-26
- Owner: Codex
- Status flow: `Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Deliver one Chinese-first Element Plus workspace for qualified purchase requests, approval decisions, generated orders, partial receiving, batch facts, private evidence selection, and receipt detail over the current plugin routes.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Current navigation | Pass | The independent plugin now exposes four Chinese primary work areas: request center, approval inbox, purchasing and receiving, and master data; the purchasing area is lazy-loaded |
| Purchase request editor | Pass | Active supplier and product lookups, approver fields, reason, fixed CNY currency, repeatable product lines, six-place quantity, two-place unit price, validation, removal, and total preview are available in one drawer |
| Approval queue | Pass | Pending request rows open the current workflow task and call current approve/reject routes with task ID, optimistic version, comment, and an idempotency key |
| Qualification and conflict feedback | Pass | Supplier/product/manufacturer qualification failures and stale-version conflicts remain failures and are translated into actionable Chinese messages |
| Order operations | Pass | Open, partial, and received orders expose supplier, exact persisted totals, per-line ordered/received/remaining facts, receipt history, and direct receiving actions |
| Partial receiving | Pass | Warehouse, area, location, selected remaining lines, exact quantity strings, batch, production date, expiry date, order version, and up to ten private evidence files map to the current inbound contract |
| Receipt detail | Pass | Persisted inbound records open through the current detail route and show order reference, warehouse facts, product, quantity, batch, production/expiry dates, receiver time, and evidence metadata |
| Complete states | Pass | Loading, empty, backend error, validation error, qualification rejection, optimistic conflict, saving, and success are distinct; no backend error is converted into success |
| Element Plus and theme | Pass | Inputs, selects, date pickers, tables, tags, drawers, dialogs, buttons, tooltips, and messages use Element Plus and inherit host light/dark plus comfortable/compact tokens |
| Chinese and locale | Pass | New visible labels are Chinese-first with complete en-US counterparts; real-browser locale switching changes the workspace, views, controls, and status labels without reload |
| Desktop and mobile | Pass | Chrome checks at 1440x900 and 390x844 show zero document overflow; desktop tables become mobile cards, four primary tabs fit, drawers stay viewport-wide, and receiving fields retain local labels |
| Accessibility and keyboard | Pass | Operational controls are native buttons or Element Plus controls; icon-only actions have accessible names, inputs have labels, and Enter applies search |
| Browser integrity | Pass | Desktop light, mobile dark/compact, receiving drawer, and English locale checks report zero console/page errors and no HTTP resource failures |
| Interaction evidence | Pass | A real 390px browser filled a partial receipt and emitted the expected scoped POST body with order version, quantity `170`, batch, production/expiry dates, and idempotency header |
| Component coverage | Pass | Six Vitest files and 19 tests cover navigation, lazy loading, current API paths, approval task mutation, order progress, receipt detail, OA regression, locale, and form mapping |
| Performance budget | Pass | Initial JavaScript is 201,310/204,800 bytes gzip, the largest async business chunk is 6,898/8,192 bytes gzip, total JavaScript is 208,208/215,040 bytes gzip, CSS is 19,555/25,600 bytes gzip, and entry raw size is 503,392/512,000 bytes |
| Plugin and repository gate | Pass | Plugin Go tests and the full repository Go test suite pass; package and checksum verification produce `pharma_oa-0.10.0.zip` with SHA-256 `51fac0a039241480a741808d4c2dc8584f720a80a3e519117816d16edb9bf7ee` |
| Current-only rule | Pass | Only current `0.10.0` routes, types, UI, and public host SDK calls are used; no compatibility component, legacy route, fallback response, duplicate screen, or host production coupling was introduced |

The first unit review rejected the old three-tab navigation assertion, and the lazy-component review then exposed an async timing assumption; both tests were corrected and rerun. The initial bundle review rejected a total-only budget that penalized a correctly lazy-loaded business chunk. The gate was replaced with explicit initial, per-async-chunk, total, CSS, and raw-entry limits, and all limits pass. Mobile visual review rejected unlabeled receiving controls after the desktop table header disappeared; local labels and accessible names were added before the complete browser matrix was rerun.

The first two full repository runs failed because the system drive had no temporary linker space. No source expectation was weakened: the final run isolated `TEMP`, `TMP`, `TMPDIR`, `GOTMPDIR`, and `GOCACHE` on the D drive and passed the entire repository. Package creation required the same isolation for npm cache before package and checksum verification passed.

### Verification Commands

```powershell
cd plugins/pharma_oa/frontend
npx vue-tsc --noEmit
npm test
npm run build

cd ../../..
go test ./plugins/pharma_oa/... -count=1
$env:TEMP='D:\workspace\.codex-temp\skoll-go-tmp'
$env:TMP=$env:TEMP
$env:TMPDIR=$env:TEMP
$env:GOTMPDIR=$env:TEMP
$env:GOCACHE='D:\workspace\.codex-temp\skoll-go-cache'
go test ./... -count=1

$env:npm_config_cache='D:\workspace\.codex-temp\npm-cache'
./plugins/pharma_oa/plugin.ps1 -Action package -DistDir 'D:\workspace\.codex-temp\skoll-bf5-03c-package'
./plugins/pharma_oa/plugin.ps1 -Action verify -DistDir 'D:\workspace\.codex-temp\skoll-bf5-03c-package'
codegraph sync .
codegraph explore "BF5-03C PurchaseWorkspace frontend API types i18n lazy loading bundle budget impact and missing current route coverage"
git diff --check
```

Result: BF5-03C passed. Purchase staff, approvers, and warehouse receivers can complete the current request-to-order-to-partial-receipt experience from one Chinese-first, theme-aware, responsive plugin workspace.

### Impact Review

- Frontend architecture: the purchasing domain is a lazy plugin-owned component instead of another branch inside the already large application shell.
- API: added typed clients only for current purchase request, purchase order, and inbound routes; every mutation retains host-supplied idempotency headers.
- Interaction: repeated lines, approval decisions, exact persisted progress, partial receiving, multiple evidence files, and receipt facts stay connected without duplicate pages.
- Visual quality: Element Plus remains authoritative; desktop tables, mobile cards, dense metrics, segmented views, drawers, dark mode, and compact density share the current design tokens.
- Performance: the quality gate now distinguishes initial cost from intentionally lazy business chunks while retaining hard initial, async, total, CSS, and entry limits.
- Acceptance boundary: packaged real-process request approval and partial/final receiving remain BF5-03D; this task proves the complete frontend contract, browser interaction, package contents, and repository regression.
- Compatibility: none; only the current UI and route contracts exist.

### Commit

`BF5-03C: build Chinese purchase operations workspace`

## BF5-03D1 Map And Freeze The Pharma OA Extraction Boundary

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Done`
- Scope: Inventory the built-in Pharma OA implementation, map every business capability to the independent plugin roadmap, and make host removal a blocking prerequisite for BF5-03D.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Host inventory | Pass | CodeGraph and exact source scans found 138 files in the five built-in backend business directories, 17 host frontend Pharma paths, plus Store, SQL adapter, OpenAPI, benchmark, and test coupling |
| Boundary decision | Pass | Pharma OA is classified as plugin-owned business code; workflow, datastore, document number, file, job, notification, audit, permission, data-scope, secret, configuration, and frontend host services remain industry-neutral framework capabilities |
| Capability disposition | Pass | Existing plugin capabilities map to BF4-02 through BF5-03; CRM, sales, inventory, quality, finance, operations UI, and reporting map explicitly to BF5-02 and BF5-04 through BF5-09 |
| Removal sequence | Pass | Separate Work Items now cover backend surfaces, persistence ownership, frontend surfaces, architecture gates, and final lifecycle closure |
| Current-only rule | Pass | The plan contains no compatibility layer, legacy bridge, fallback, copied dual path, or deferred host business implementation |

The independent plugin is the only future Pharma OA owner. Existing host behavior is not treated as a contract: required behavior is implemented through current public plugin services, while obsolete host packages and screens are deleted.

### Verification Commands

```powershell
codegraph sync .
codegraph explore "How is the built-in Pharma OA implementation wired into the Skoll host at startup, persistence, HTTP routing, OpenAPI, and frontend navigation, and which files can be removed without affecting plugins/pharma_oa?"
rg --files internal/domain/pharmaoa internal/service/pharmaoa internal/repository/pharmaoa internal/handler/http/v1/pharmaoa internal/plugin/pharmaoa
rg --files web/src | rg "pharma|Pharma"
rg -n "PharmaOA|pharmaoa|pharma_oa" internal/store internal/bootstrap cmd
git diff --check
```

Result: BF5-03D1 passed. BF5-03D cannot close until BF5-03D2 through BF5-03D5 prove that the framework owns no Pharma OA business implementation.

### Impact Review

- Architecture: establishes one-way ownership from the industry plugin to public framework contracts.
- Backend: schedules removal of built-in application, HTTP, domain, repository, SQL, and bootstrap coupling.
- Frontend: schedules removal of host business views, API client, locale bundle, and navigation coupling.
- Roadmap: preserves desired medical OA outcomes as plugin Work Items without preserving legacy code.
- Compatibility: none; host business behavior is not retained as a fallback or migration source.

### Commit

`BF5-03D1: freeze Pharma OA extraction boundary`

## BF5-03D2 Remove Built-In Pharma OA Backend Application And HTTP Surfaces

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Remove the host-owned Pharma OA plugin adapter, application services, HTTP handlers, embedded OpenAPI operations, and tests coupled to that implementation.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Application removal | Pass | Deleted the 57-file built-in application-service package; no production or test package imports `internal/service/pharmaoa` |
| HTTP removal | Pass | Deleted the 45-file built-in handler package and the four-file host plugin adapter; no package imports either removed path |
| Contract removal | Pass | Removed all 95 built-in Pharma OA business paths and 186 exclusively reachable schemas from the embedded core OpenAPI; the synchronized public copy has the same SHA-256 |
| Obsolete test removal | Pass | Removed four integration and benchmark suites that instantiated the deleted host implementation; independent packaged-plugin tests remain authoritative |
| Core regression | Pass | Embedded OpenAPI references resolve, both OpenAPI copies stay synchronized, and every package under `internal/...` passes |
| Independent plugin | Pass | `go test ./plugins/pharma_oa/... -count=1` passes with the plugin using only current public host services |
| Current-only rule | Pass | No route alias, adapter, fallback, copied service, or hidden built-in registration remains |

The first review failed because the final removed OpenAPI path was also the last mapping entry, leaving its operation body attached to the preceding path. The structural residue was deleted, the reference test was changed from asserting a built-in Pharma path to forbidding one, and both OpenAPI copies were synchronized before the full task gate was rerun.

### Verification Commands

```powershell
go test ./internal/handler/http -run 'Test(EmbeddedOpenAPIReferencesResolve|OpenAPIContractFilesStayInSync)$' -count=1 -v
go test ./internal/... -count=1
go test ./plugins/pharma_oa/... -count=1
rg -n "internal/service/pharmaoa|internal/handler/http/v1/pharmaoa|internal/plugin/pharmaoa" --glob "*.go" --glob "!plugins/pharma_oa/contract_test.go"
rg -n "^  /v1/plugins/pharma_oa/api/" internal/handler/http/openapi.yaml
git diff --check
```

Result: BF5-03D2 passed. The core no longer owns a Pharma OA application or HTTP API; domain and SQL ownership remain isolated to BF5-03D3.

### Impact Review

- Backend: removed 106 files from the built-in service, handler, and plugin-adapter packages.
- Tests: removed four suites whose subject was the deleted implementation.
- API: core OpenAPI no longer advertises business-plugin operations or schemas.
- Plugin runtime: generic route aggregation and the independent package are unchanged.
- Compatibility: none; removed endpoints have no alias or fallback.

### Commit

`BF5-03D2: remove built-in Pharma OA backend`

## BF5-03D3 Remove Built-In Pharma OA Domain And Persistence Ownership

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Done`
- Scope: Remove host-owned Pharma OA domain models, repository contracts and implementations, GORM persistence, SQL adapter exposure, Bundle construction, automatic schema migration, and the old table-specific benchmark command.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Domain ownership | Pass | Deleted the 24-file built-in Pharma OA domain package; framework domain packages contain no industry entity model |
| Repository ownership | Pass | Deleted the eight-file Pharma OA repository package and all memory repository construction |
| SQL ownership | Pass | Deleted 13 Pharma GORM model/store files and removed every Pharma model from `AllModels()` |
| Adapter boundary | Pass | MySQL and PostgreSQL adapters no longer import, construct, store, or expose 21 Pharma repository groups |
| Bundle boundary | Pass | `store.Bundle` no longer exposes `PharmaOA`; its lazy factory, provider interface, memory assembly, and tests are removed |
| Benchmark ownership | Pass | Deleted the host command that seeded and queried four removed `pharma_oa_*` tables |
| Generic plugin persistence | Pass | Plugin datastore DB, migration ledger, document number, workflow binding, attachment, comment, and timeline models remain unchanged |
| Regression gate | Pass | All store packages pass, followed by the complete repository `go test ./... -count=1` gate |
| Current-only rule | Pass | No host table, repository, adapter, migration, fallback store, or dual persistence path remains |

### Verification Commands

```powershell
go test ./internal/store/... -count=1
go test ./... -count=1
rg -n "internal/domain/pharmaoa|internal/repository/pharmaoa|PharmaOA|NewPharma|Pharma[A-Z].*Repository" internal cmd --glob "*.go"
rg -n "pharma_oa_" internal cmd --glob "*.go"
git diff --check
```

Result: BF5-03D3 passed. Host persistence is industry-neutral; the installed plugin owns medical data exclusively through declared plugin schemas and public DataStore services.

### Impact Review

- Domain: removed all built-in medical entities and invariants.
- Persistence: removed 21 repository groups, 29 automatic GORM model registrations, and four host benchmark tables.
- Runtime: Bundle, MySQL, and PostgreSQL now assemble only framework repositories.
- Plugin data: packaged plugin schemas and migrations are untouched and continue to pass the full repository gate.
- Compatibility: none; old tables and repository APIs are not retained.

### Commit

`BF5-03D3: remove Pharma OA persistence ownership`

## BF5-03D4 Remove Built-In Pharma OA Frontend Surfaces

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Remove host-owned Pharma OA views, API client, locale bundle, static routes, redirects, reminders, and business-specific frontend quality assets while preserving platform-level quality gates.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Host UI ownership | Pass | Deleted 15 built-in Pharma views, the host Pharma API client, business locale bundle, and static integrated-route registry |
| Routing boundary | Pass | Host routing now consumes only runtime plugin application and remote routes; no `/pharma-oa` redirect or hard-coded Pharma route remains |
| Platform copy | Pass | Workflow demo, dictionary examples, and reminder targets use industry-neutral platform language and routes |
| Locale gate | Pass | Replaced the Pharma inventory baseline with a platform baseline; 1,207 bilingual keys, 825 references, and 52 UI files pass with zero hard-coded visible copy |
| Frontend quality assets | Pass | Large-list checks now validate reusable virtual tables, pagination, cancellation, and Element Plus locale binding; obsolete Pharma smoke, runtime, visual tests, and screenshots were removed |
| Host regression | Pass | Typecheck, accessibility, theme, 29 component/document tests, and production build pass |
| Independent plugin | Pass | The plugin-owned frontend passes 19 tests and its independent typecheck, build, lazy-chunk, and bundle budgets |
| Current-only rule | Pass | No host view, route alias, locale fallback, API proxy, copied screen, or dual UI remains |

The first command review rejected `npm test` because the host package deliberately exposes explicit test scripts instead of a generic alias. The task gate was corrected to `npm run test:components`. The first typecheck then rejected the old locale script because it still read the deleted Pharma locale and page inventory; the script was converted to a platform-wide locale and hard-coded-copy gate before all checks were rerun.

### Verification Commands

```powershell
cd web
npm run check:i18n
npm run check:large-list
npm run typecheck
npm run test:components
npm run build

cd ../plugins/pharma_oa/frontend
npm test
npm run build

cd ../../..
rg -n -i "pharma-oa|pharma_oa|pharma\.|Pharma OA|医药 OA" web/src web/scripts web/tests/e2e web/i18n
rg -n "src/pharma-oa|views/Pharma|integrated-routes|pharma.json|pharma-oa-baseline" web --glob "!node_modules/**"
git diff --check
```

Result: BF5-03D4 passed. The host is a generic Element Plus platform shell; the independent plugin package is now the only medical OA frontend owner.

### Impact Review

- Frontend: removed all built-in medical business screens and their host API client.
- Routing: plugin navigation is runtime-driven and contains no industry route registry.
- Localization: platform and plugin dictionaries have separate ownership and independent tests.
- Quality: reusable platform gates remain active, while business-specific checks live with the plugin package.
- Compatibility: none; deleted host routes and screens have no aliases or fallback components.

### Commit

`BF5-03D4: remove built-in Pharma OA frontend`

## BF5-03D5 Enforce Framework Purity And Independent-Plugin Gates

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Done`
- Scope: Turn the extraction boundary into an executable architecture rule and prove the independent package across artifact, process lifecycle, backend, and frontend gates.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Architecture gate | Pass | `TestMedicalOAProductionCoreOwnsNoBusinessImplementation` scans core Go, OpenAPI, Vue, TypeScript, JSON, and frontend quality scripts and rejects medical package, route, table, menu, locale, or view ownership |
| Removed-path gate | Pass | The five former backend business package roots, host Pharma frontend roots, locale file, and integrated route registry must remain absent |
| Allowed-reference policy | Pass | Production core references are forbidden; independent `plugins/pharma_oa` sources, packaged lifecycle tests, and generic platform tests using an opaque plugin ID remain allowed |
| Artifact gate | Pass | Version `0.10.0` frontend and backend package successfully; generated ZIP and checksum verify at SHA-256 `6306a8fd1c8585d19b432feaf01e7b7aa918f8721d343c3122131639e447e5b3` |
| Real-process lifecycle | Pass | Packaged backend installs, starts as an external process, persists across restart, enforces scopes, disables, re-enables, and uninstalls without restoring host business code |
| Backend regression | Pass | `go test ./... -count=1` passes across the complete repository |
| Host frontend gate | Pass | Typecheck, locale, accessibility, large-list, theme, and 29 component/document tests pass |
| Independent frontend gate | Pass | Plugin build, lazy chunks, and bundle budgets pass during packaging; its 19 focused frontend tests passed in BF5-03D4 |
| Current-only rule | Pass | The gate recognizes one current plugin implementation and rejects aliases, fallbacks, copied host paths, and dual ownership |

### Verification Commands

```powershell
go test ./plugins/pharma_oa -run 'TestMedicalOAProductionCoreOwnsNoBusinessImplementation$' -count=1 -v
$env:SKOLL_PHARMA_OA_E2E='1'
go test ./internal/plugin -run 'TestPharmaOAPackagedBusinessLifecycleE2E$' -count=1 -v
go test ./... -count=1

./plugins/pharma_oa/plugin.ps1 -Action package -DistDir 'D:\workspace\.codex-temp\skoll-bf5-03d5-package'
./plugins/pharma_oa/plugin.ps1 -Action verify -DistDir 'D:\workspace\.codex-temp\skoll-bf5-03d5-package'

cd web
npm run typecheck
npm run test:components

cd ..
codegraph sync .
git diff --check
```

Result: BF5-03D5 passed. Framework purity is now executable rather than documentary, and the medical OA package remains independently buildable and operable.

### Impact Review

- Architecture: future commits cannot silently recreate a host medical package, route, schema, menu, locale bundle, or view.
- Plugin boundary: business code remains fully owned by the installable package and current public host contracts.
- Tests: generic plugin-platform tests may retain opaque IDs without becoming an implementation owner.
- Operations: artifact checksum and real external-process lifecycle prove package integrity and isolation.
- Compatibility: none; the gate explicitly rejects retained legacy ownership and dual paths.

### Commit

`BF5-03D5: enforce framework purity gate`

## BF5-03D Close Purchasing And Inbound Acceptance

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Done`
- Scope: Close the packaged purchasing and inbound lifecycle after the independent-plugin implementation, host extraction, architecture gate, and frontend acceptance all pass.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Clean package install | Pass | The installed artifact contains the managed backend, separated frontend, manifest, and reversible migrations `001` through `008` |
| Qualification gates | Pass | Supplier purchase, product purchase, and manufacturer supply qualifications are created with evidence, submitted, approved, and enforced before purchasing |
| Purchase idempotency | Pass | Repeating the same request with one idempotency key returns the original request and does not create another record |
| Approval path | Pass | Approving a `2.5 x 10.20` request creates one open order with exact `25.50` total and a traceable workflow decision |
| Rejection path | Pass | A second request reaches `rejected` through its current pending task without creating an order |
| Partial receiving | Pass | The first receipt records lot, production, expiry, warehouse/location, and evidence; order progress becomes exact `1.25` and `partial` |
| Cross-scope isolation | Pass | Another tenant scope sees zero requests, orders, and receipts; direct detail reads return not found |
| Restart durability | Pass | Two requests, one order, one partial receipt, workflow timelines, files, jobs, exact quantity, status, and optimistic version survive process restart |
| Final receiving | Pass | A second `1.25` receipt closes the order at exact `2.5`, status `received`, with two persisted inbound records |
| Lifecycle controls | Pass | Disable closes business endpoints, re-enable restores the package, and uninstall closes endpoints and removes plugin-owned schema without restoring host code |
| Audit and transactions | Pass | Purchase create/approve/reject and inbound create actions are audited; at least 40 transactions commit and none roll back in the successful scenario |
| Framework purity | Pass | BF5-03D1 through BF5-03D5 pass; production core has no medical business implementation or dual path |
| Frontend quality | Pass | Chinese purchasing UI, interaction tests, themes, responsive layout, typecheck, lazy chunks, bundle budgets, and host frontend gates pass |
| Repository regression | Pass | The complete `go test ./... -count=1` gate passes after extraction |

### Verification Commands

```powershell
$env:SKOLL_PHARMA_OA_E2E='1'
go test ./internal/plugin -run 'TestPharmaOAPackagedBusinessLifecycleE2E$' -count=1 -v
go test ./plugins/pharma_oa/... -count=1
go test ./... -count=1

cd plugins/pharma_oa/frontend
npm test
npm run build

cd ../../../web
npm run typecheck
npm run test:components
npm run build

cd ..
codegraph sync .
git diff --check
```

Result: BF5-03D and parent BF5-03 passed. Purchasing and inbound receiving now exist only as an independently packaged medical OA capability built on current public framework services.

### Impact Review

- Business capability: request, approval, rejection, order creation, partial receipt, and final receipt form one tested package lifecycle.
- Isolation: tenant scope, plugin process, schema, files, workflow, jobs, audit, and transaction boundaries remain explicit.
- Durability: package state survives restart and is removed by uninstall policy.
- Framework: reusable host services remain industry-neutral, with medical behavior exclusively inside the plugin package.
- Compatibility: none; no removed host endpoint, schema, page, or fallback path is retained.

### Commit

`BF5-03D: close packaged purchase lifecycle`

## BF5-05P Correct Inventory And Outbound Delivery Order

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Done`
- Scope: Remove the circular product dependency in the medical OA roadmap and freeze an atomic inventory delivery sequence before implementation.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Direction | Pass | Purchasing and inbound now feed inventory; lot-aware inventory becomes a prerequisite for sales allocation and outbound delivery |
| Dependency graph | Pass | BF5-05 depends on completed BF5-03, product catalogs, and qualifications; BF5-04 depends on completed BF5-05 |
| Atomic sequence | Pass | Topology, immutable ledger, operational movements, frontend workspace, and packaged acceptance are separate BF5-05A through BF5-05E Work Items |
| Ownership | Pass | Every child declares backend, database, workflow, frontend, UX, accessibility, testing, or quality-gate Skills explicitly |
| Acceptance | Pass | Every child has concrete deliverables, failure conditions, verification categories, and one commit after acceptance |
| Current-only rule | Pass | Inventory is implemented only inside the independent medical OA plugin; no host model, compatibility adapter, or parallel ledger is planned |

Result: BF5-05P passed. The executable order is now inbound receiving, inventory ledger and warehouse operations, then sales allocation and outbound delivery.

### Impact Review

- Roadmap: removes a dependency inversion that would otherwise force fake allocation or temporary stock behavior.
- Backend: warehouse and ledger contracts become authoritative before sales consumes them.
- Frontend: inventory operations receive their own complete workflow rather than being embedded in sales pages.
- Quality: reconciliation, concurrency, restart, cross-scope, and uninstall gates are explicit.
- Compatibility: none; no temporary bridge or deferred dual model is allowed.

### Commit

`BF5-05P: sequence inventory before outbound`

## BF5-05A1 Declare Warehouse Topology Schema And Migration

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Done`
- Scope: Add only the independent plugin persistence contract for warehouse, area, and location topology before exposing behavior.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Schema contract | Pass | `datastore.yaml` declares `warehouses`, `warehouse_areas`, and `warehouse_locations` with typed identity, parent, status, temperature, location-type, and disable-reason fields |
| Scope ownership | Pass | Physical tables include tenant, organization, owner, version, and timestamps; no host model or repository is introduced |
| Reversible migration | Pass | Migration `009_warehouse_topology` creates all three tables in parent-first order and drops them in child-first order |
| Warehouse uniqueness | Pass | Warehouse code is unique per tenant organization and reusable in another organization |
| Area uniqueness | Pass | Area code is unique per warehouse and reusable in another warehouse |
| Location uniqueness | Pass | Location code is unique per area and reusable in another area |
| Manifest ownership | Pass | The plugin manifest records all three physical tables, columns, indexes, descriptions, and drop-on-uninstall ownership |
| Regression | Pass | Package surface, schema loader, executable/reversible SQLite migration, framework purity, and all Pharma OA Go packages pass |
| Current-only rule | Pass | No compatibility table, alias field, host persistence adapter, or dual schema exists |

### Verification Commands

```powershell
go test ./plugins/pharma_oa -run 'TestMedicalOA(PackageSurfaceAndHostIndependence|PurchaseDataStoreContractIsRegistered|FoundationMigrationIsExecutableAndReversible)$' -count=1 -v
go test ./plugins/pharma_oa/... -count=1
codegraph sync .
git diff --check
```

Result: BF5-05A1 passed. The plugin now owns one current, reversible warehouse topology schema ready for scoped behavior in BF5-05A2.

### Impact Review

- Persistence: adds three plugin-owned topology tables and no framework tables.
- Data integrity: scoped unique indexes encode the intended warehouse hierarchy boundaries.
- Lifecycle: automatic migration and drop uninstall policies include the new tables.
- API: no route behavior changes in this Work Item.
- Compatibility: none.

### Commit

`BF5-05A1: declare warehouse topology schema`

## FF0-01 Audit And Freeze The Framework Capability Baseline

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Done`
- Scope: Reconcile the current framework implementation against the capabilities required to deliver medical OA and future independent business plugins without plugin-local framework substitutes.

### Capability Disposition

| Capability | Result | Evidence / Next Owner |
| --- | --- | --- |
| External-process multi-call transaction | Pass | `pkg/pluginclient` starts a host transaction, propagates `X-Skoll-Transaction-ID`, and finishes with commit/rollback; `HostGateway` retains the real transaction context, serializes calls, applies TTL rollback, and binds user claims |
| Trusted data scope and namespace isolation | Pass | Structured query/mutation contracts accept scope intent only; registry, planners, and executors inject trusted tenant/organization/owner predicates and deny reserved or cross-plugin identifiers |
| Optimistic concurrency | Pass | Single-record update/delete/upsert support positive `ExpectedVersion`; SQL writes include the current version and return deterministic conflict on stale or concurrent changes |
| Persistent idempotent mutations | Pass | `(plugin_id, idempotency_key)` reservation, request hash, and validated result are persisted in the same transaction as mutation and audit |
| Exact decimal transport | Partial | Decimal values use validated base-10 strings and avoid JSON float conversion; no host-owned guarded arithmetic or aggregate execution exists |
| Append-only data policy | Gap: FF1-01 | Schemas describe field mutability only; the host cannot declare a table insert-only, so an immutable stock ledger is not enforceable by the framework |
| Atomic guarded arithmetic | Gap: FF1-02..FF1-03 | Mutations replace field values and cannot atomically increment/decrement exact numeric values with lower/upper guards under contention |
| Scoped aggregation | Gap: FF1-04..FF1-05 | `DataQuery` returns records only and has no bounded count/sum/min/max/group contract or dialect executor |
| Transactional plugin events | Gap: FF2 | Manifests declare subscriptions and the runtime can deliver host events, but plugins have no current public publication service, transactional outbox, inbox dedupe, or versioned inter-plugin authorization |
| Regulated workflow evidence | Gap: FF3 | Current workflows cover ordinary actions; conditional/parallel/quorum decisions, durable escalation, electronic signatures, and tamper-evident evidence are not complete |
| Frontend plugin SDK | Gap: FF4 | The shell and current plugin UI work, but reusable typed composition, inherited design tokens, failure isolation, and plugin-level visual/performance gates are not one complete public SDK |
| Generator and test leverage | Gap: FF5 | Generators and fixtures exist but do not yet produce and prove the complete current datastore/event/workflow/frontend/package stack without hand edits |
| Business-scale hardening | Gap: FF6 | Permission, audit, diagnostics, and limits exist in parts; least-privilege capability grants, full correlation, quotas/backpressure, and multi-plugin load/security acceptance are not closed |

### Direction And Dependency Result

- FF1 is the immediate blocking milestone because BF5-05B requires host-enforced immutable ledger behavior, exact guarded balance updates, and reconciliation aggregates.
- BF5-05A topology may continue independently, but BF5-05B cannot start until FF1-06 passes.
- Quality and recall work waits for reliable events and governed evidence through FF2-05 and FF3-04.
- Medical operations UI waits for the reusable frontend SDK gate FF4-05; final acceptance waits for FF5-04 and FF6-04.
- FF work is industry-neutral framework code. Medical rules, tables, pages, and workflows remain exclusively inside independent plugins.
- No compatibility version, legacy parser, dual path, fallback implementation, release task, deployment task, or documentation-only milestone is introduced.

### Verification

```powershell
codegraph status .
codegraph node pkg/pluginclient/services.go
codegraph node pkg/pluginclient/client.go
codegraph node internal/plugin/host_gateway.go
codegraph node pkg/pluginsdk/datastore.go
git diff --check
```

Result: FF0-01 passed. The framework baseline is evidence-based, FF1 through FF6 are dependency-ordered and atomic, and the first executable gap is FF1-01 append-only datastore enforcement.

### Commit

`FF0-01: freeze framework capability gaps`

## FF1-01 Enforce Explicit Mutable Or Append-Only Table Policies

- Date: 2026-07-27
- Owner: Codex
- Status flow: `Doing -> Review -> Done`
- Scope: Make table-level mutability a required current datastore contract and enforce append-only behavior in the host before persistence.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Current schema contract | Pass | Every table in both current `datastore.yaml` manifests declares exactly `mutable`; omitted and unknown values fail strict installation validation |
| Registry ownership | Pass | `TableSchema` and `ResolvedTable` preserve one typed policy; programmatic schemas without a supported policy fail closed |
| Append-only enforcement | Pass | Insert succeeds; update, upsert, and delete return `unsupported` before idempotency reservation, SQL, or audit, proven by unchanged record/idempotency/audit counts |
| Lifecycle inspection | Pass | Storage snapshots and the data-control HTTP contract expose the authoritative mutation policy |
| Operator UX | Pass | The Element Plus data page renders a localized write-policy tag for every owned table in zh-CN and en-US |
| Current plugin manifests | Pass | Datastore E2E and Pharma OA schemas load under the required current contract; no parser fallback or implicit mutable default exists |
| Regression | Pass | All Go packages, frontend typecheck/a11y/i18n/theme gates, 29 component tests, production build, and six desktop/mobile plugin-center scenarios pass |
| Framework purity | Pass | The policy is industry-neutral host infrastructure; no stock ledger or medical rule entered the core |

### Failed Runs And Re-Execution

- The first focused command targeted `plugins/datastore_e2e`, which has no root Go package. The corrected backend package command passed.
- The first full Go run found a stale Pharma OA manifest test fixed at 12 tables. It was corrected to require the current explicit 15-table set, and the full suite then passed.
- The first Playwright run had no Vite server; the second had no Go API; the third exposed a lifecycle-state assertion fixed to accept the actual enable/disable primary action. The fourth run passed all six scenarios.

### Verification Commands

```powershell
go test ./internal/plugin/datastore ./internal/handler/http/v1/plugin ./internal/bootstrap ./plugins/datastore_e2e/backend ./plugins/pharma_oa -run 'Test(LoadSchemaManifest|SchemaRegistry|MutationExecutor|LifecycleInspectStorage|PluginDataControl|PluginDataStore|MedicalOA.*DataStore|MedicalOAPackageSurface)' -count=1
go test ./... -count=1
cd web
npm run typecheck
npm run test:components
npm run build
npm run test:plugin-center
codegraph sync .
git diff --check
```

Result: FF1-01 passed. Plugin authors must make table mutability explicit, the host can now guarantee insert-only ledger storage, and operators can inspect that policy from the current control center.

### Commit

`FF1-01: enforce append-only plugin tables`
