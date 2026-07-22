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
