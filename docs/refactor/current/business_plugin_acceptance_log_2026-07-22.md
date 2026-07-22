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
