# Pharma OA Acceptance Log

> Batch: pharma-oa-2026-07-04
> Record acceptance evidence here only for Work Items from `pharma_oa_work_items.md`.
> Completed M0-M7/FE/N0 evidence remains under `../old/` and must not be updated for this batch.

## F6-00: Create Official Pharma OA Task Files

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `docs/refactor/README.md`
- `docs/refactor/current/README.md`
- `docs/refactor/current/pharma_oa_task_board.md`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Official current files exist | Passed | Task board, work-item table, and acceptance log are under `docs/refactor/current/` |
| Candidate pool is not used as ledger | Passed | `pharma_oa_milestone_plan_2026-07-04.md` remains unchanged |
| Archived evidence not updated | Passed | No file under `docs/refactor/old/` is changed |
| Work Item schema complete | Passed | Each row includes ID, Skill, task description, dependencies, deliverables, acceptance criteria, verification commands, and status |
| Priority order respected | Passed | F6 is first, followed by F7, F8, and F9 before later pharma OA milestones |
| No-compatibility rule preserved | Passed | No legacy API/data/plugin/page compatibility task is added |

### Verification Commands

```powershell
rg -n "pharma_oa_task_board|pharma_oa_work_items|pharma_oa_acceptance_log" docs/refactor/README.md docs/refactor/current/README.md
rg -n "F6-00|F6-01|F12-05" docs/refactor/current/pharma_oa_work_items.md
rg -n "\| Todo \|" docs/refactor/current/pharma_oa_work_items.md docs/refactor/current/pharma_oa_task_board.md
git diff --check
```

Result: Passed.

### Next Step

- Claim `F6-01` from `docs/refactor/current/pharma_oa_work_items.md`.

## F6-01: Implement Theme Engine

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `web/src/stores/theme.ts`
- `web/src/main.ts`
- `web/src/App.vue`
- `web/src/components/Layout/Header.vue`
- `web/src/styles/variables.scss`
- `web/src/styles/global.scss`
- `web/src/i18n/index.ts`
- `web/src/plugins/index.ts`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Theme store | Passed | `useThemeStore` owns `light`, `dark`, and `compact`, persists `skoll.ui.theme`, and applies `data-theme`, `data-theme-mode`, and `data-density` to the root element |
| CSS tokens | Passed | `variables.scss` defines light/dark/compact tokens and Element Plus CSS variable mapping |
| Switch entry | Passed | Header exposes three theme buttons with localized labels and active state |
| Refresh persistence | Passed | Browser smoke kept compact mode after reload |
| Plugin page synchronization | Passed | Existing plugin iframe bridge now injects and posts `skoll:theme` payload with mode, density, color scheme, and tokens |
| Verification | Passed | Typecheck, build, and browser DOM smoke passed |

### Verification Commands

```powershell
cd web; npm run typecheck
cd web; npm run build
cd web; npm run dev
```

Browser smoke result:

```json
{
  "channel": "chrome",
  "hasThemeButtons": 3,
  "initial": { "theme": "light", "mode": "light", "density": "comfortable", "stored": null },
  "dark": { "theme": "dark", "mode": "dark", "density": "comfortable", "stored": "dark" },
  "compact": { "theme": "light", "mode": "compact", "density": "compact", "stored": "compact" },
  "persisted": { "theme": "light", "mode": "compact", "density": "compact", "stored": "compact" }
}
```

Notes:

- `npm run build` still prints the known Sass legacy JS API and VueUse/Rollup annotation warnings, then exits successfully.
- Browser smoke used a temporary local token to enter the frontend shell. Backend `127.0.0.1:8080` was not running, so Vite logged proxy errors for auth/plugin/menu requests; those API calls are outside this Work Item's acceptance scope.

### Next Step

- Claim `F6-02` from `docs/refactor/current/pharma_oa_work_items.md`.

## F6-02: Implement Plugin UI Kit

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `web/src/components/Common/PageShell.vue`
- `web/src/components/Common/PageToolbar.vue`
- `web/src/components/Common/FilterBar.vue`
- `web/src/components/Common/DataTable.vue`
- `web/src/components/Common/DetailDrawer.vue`
- `web/src/components/Common/ConfirmAction.vue`
- `web/src/components/Common/index.ts`
- `web/src/components/Common/README.md`
- `web/src/views/Dashboard/index.vue`
- `web/src/App.vue`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Page shell | Passed | `PageShell` provides title, description, meta/actions, loading, error, and no-permission states |
| Toolbar and filters | Passed | `PageToolbar` and `FilterBar` provide responsive action/filter layouts |
| Data table | Passed | `DataTable` covers loading, empty, error, no-permission, row actions, custom cells, and pagination slot |
| Detail and confirmation | Passed | `DetailDrawer` covers loading/responsive drawer layout; `ConfirmAction` guards risky actions through Element Plus confirmation |
| Main app usage | Passed | Dashboard now uses `PageShell` and keeps existing cards, quick links, risk states, and permission-aware empty state |
| Responsive smoke | Passed | Dashboard smoke has no overflowing elements at 390px width after retry |
| Documentation | Passed | `web/src/components/Common/README.md` lists UI Kit components, state coverage, usage rules, and validation commands |

### Verification Commands

```powershell
cd web; npm run typecheck
cd web; npm run build
cd web; npm run dev
```

Browser smoke result after retry:

```json
{
  "channel": "chrome",
  "desktop": {
    "hasShell": true,
    "hasHeader": true,
    "cards": 4,
    "statusChips": 2
  },
  "mobile": {
    "bodyWidth": 390,
    "overflowing": []
  }
}
```

### Failure And Retry

- Initial failure: browser smoke found `.content-area` overflowed on 390px width.
- Fix: added `box-sizing: border-box` and `min-width: 0` to `.content-area`.
- Retry result: Passed.

Notes:

- `npm run build` still prints the known Sass legacy JS API and VueUse/Rollup annotation warnings, then exits successfully.
- Browser smoke used a temporary local token to enter the frontend shell. Backend `127.0.0.1:8080` was not running, so Vite logged proxy errors for auth/plugin/menu requests; those API calls are outside this Work Item's acceptance scope.

### Next Step

- Claim `F6-03` from `docs/refactor/current/pharma_oa_work_items.md`.

## F6-03: Implement Plugin Host SDK

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/plugin/host_context.go`
- `internal/plugin/host_context_test.go`
- `web/src/plugins/host-sdk.ts`
- `web/src/plugins/index.ts`
- `web/src/plugins/README.md`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Backend PluginContext | Passed | `NewPluginContext` exposes plugin identity, API prefix, locale set, issued time, default capabilities, and endpoint contracts |
| Host capabilities | Passed | Context and frontend SDK cover `auth`, `user`, `organization`, `dictionary`, `file`, `audit`, `config`, and `permission` |
| Current API contracts | Passed | SDK and context use existing `/v1/auth/me`, `/v1/system/settings`, `/v1/system/dictionaries`, `/v1/files`, `/v1/audit`, `/v1/plugins/{id}/config`, and `/v1/permissions` paths |
| Frontend injection | Passed | Remote plugin pages and app-home plugin pages receive `window.__SKOLL_HOST__` with the real plugin ID, current locale, token, API prefix, and theme bridge |
| SDK usage sample | Passed | `web/src/plugins/README.md` documents the injected Host SDK and sample calls |
| API/OpenAPI impact | Passed | No new HTTP endpoint was added; OpenAPI remains unchanged because all SDK calls target existing current contracts |
| Permission, audit, migration impact | Passed | No new permission seed, audit action, database table, migration, or seed data is introduced in this Work Item |

### Verification Commands

```powershell
go test ./internal/plugin/...
cd web; npm run typecheck
cd web; npm run build
git diff --check
```

Result: Passed.

Notes:

- `npm run build` still prints the known Sass legacy JS API and VueUse/Rollup annotation warnings, then exits successfully.
- `git diff --check` exits successfully and prints only existing CRLF conversion warnings from the working tree.

### Next Step

- Claim `F6-04` from `docs/refactor/current/pharma_oa_work_items.md`.

## F6-04: Implement Plugin Data Contract

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/plugin/types.go`
- `internal/plugin/loader.go`
- `internal/plugin/install_preflight.go`
- `internal/plugin/local_marketplace.go`
- `internal/plugin/README.md`
- `internal/plugin/types_test.go`
- `internal/plugin/loader_test.go`
- `internal/plugin/install_preflight_test.go`
- `internal/plugin/local_marketplace_test.go`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Data manifest | Passed | `Info.DataManifest` supports namespace, migration version/directory, uninstall policy, rollback policy, tables, columns, and indexes |
| Namespace rules | Passed | Validation rejects `sk_` namespaces, tables outside the namespace, duplicate/invalid identifiers, unknown index columns, and invalid policies |
| Migration declaration | Passed | Data contract carries migration version and directory; existing plugin migrator remains the execution path |
| Uninstall and rollback strategy | Passed | `retain`, `archive`, and `drop` uninstall policies plus `manual`, `automatic`, and `none` rollback policies are validated |
| Install preflight risk | Passed | Preflight JSON now includes `data` summary and raises risk for data tables and destructive `drop` uninstall policy |
| Marketplace risk | Passed | Local marketplace risk summary exposes data namespace/table/uninstall impact |
| Docs/example | Passed | `internal/plugin/README.md` includes a current `data:` manifest example |
| API/OpenAPI impact | Passed | No new HTTP endpoint or response route was added; OpenAPI remains unchanged for this Work Item |
| Permission, audit, migration/seed impact | Passed | No new permission seed, audit action, Skoll core table, Skoll migration, or seed data is introduced; plugin-owned migrations are declared in manifest only |

### Verification Commands

```powershell
go test ./internal/plugin/...
go test ./internal/store/...
git diff --check -- internal/plugin docs/refactor/current/pharma_oa_work_items.md
```

Result: Passed.

Notes:

- `git diff --check` exits successfully and prints only existing CRLF conversion warnings from the working tree.
- `docs/refactor/old/` was not updated.

### Next Step

- Claim `F6-05` from `docs/refactor/current/pharma_oa_work_items.md`.

## F6-05: Implement Plugin API/OpenAPI Contract

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/plugin/types.go`
- `internal/plugin/loader.go`
- `internal/plugin/catalog.go`
- `internal/plugin/registry.go`
- `internal/plugin/manager.go`
- `internal/plugin/catalog_audit.go`
- `internal/plugin/install_preflight.go`
- `internal/plugin/local_marketplace.go`
- `internal/plugin/README.md`
- `internal/plugin/*_test.go`
- `docs/api/openapi.yaml`
- `internal/handler/http/openapi.yaml`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Route registry | Passed | `Info.RouteExtensions()` converts `api.routes` into `RouteExtension`; `RuntimeManager` can import plugin routes into `ExtensionRegistry` on enable |
| Manifest API contract | Passed | `api.routes` supports method, path, summary, permission, and audit action |
| Namespace rules | Passed | Plugin API paths must stay under `/v1/plugins/{pluginId}/api/` |
| Permission binding | Passed | API route permissions enter `CatalogPermissions()` and install preflight permission diff |
| Audit action declaration | Passed | Route audit actions are validated with `module.resource.action` and exposed through `AuditActions()` |
| OpenAPI aggregation preview | Passed | Install preflight exposes `api.openapiPaths`; OpenAPI schemas document the `api` response block |
| OpenAPI sync | Passed | `docs/api/openapi.yaml` and `internal/handler/http/openapi.yaml` are identical |
| Permission, audit, migration/seed impact | Passed | No new seed or core migration; existing plugin catalog audit now records route/audit-action counts |

### Verification Commands

```powershell
go test ./internal/plugin/...
go test ./internal/handler/http/...
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
git diff --check -- internal/plugin docs/api/openapi.yaml internal/handler/http/openapi.yaml docs/refactor/current/pharma_oa_work_items.md
```

Result: Passed after retry.

### Failure And Retry

- Initial failure: `TestInstallPreflightServicePassesWithCompleteImpactSummary` expected one permission add; API route permission correctly added a second permission.
- Fix: updated the test assertion to expect both manifest permission and API route permission.
- Retry result: Passed.

Notes:

- `git diff --check` exits successfully and prints only existing CRLF conversion warnings from the working tree.
- `docs/refactor/old/` was not updated.

### Next Step

- Claim `F6-06` from `docs/refactor/current/pharma_oa_work_items.md`.

## F6-06: Implement Business Event Bus

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/event/business_bus.go`
- `internal/event/business_bus_test.go`
- `internal/plugin/types.go`
- `internal/plugin/loader.go`
- `internal/plugin/catalog.go`
- `internal/plugin/manager.go`
- `internal/plugin/catalog_audit.go`
- `internal/plugin/install_preflight.go`
- `internal/plugin/README.md`
- `internal/plugin/*_test.go`
- `docs/api/openapi.yaml`
- `internal/handler/http/openapi.yaml`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Event model | Passed | `BusinessEvent` validates current event names and covers source, subject, payload, metadata, and occurrence time |
| Publisher/subscriber APIs | Passed | `BusinessEventBus.Subscribe` and `Publish` support named handlers for `approval-completed`, `inbound-completed`, `qualification-expiring`, and similar events |
| After-transaction events | Passed | `AfterCommitQueue` publishes only on `Commit` and drops queued events on `Rollback` |
| Retry records | Passed | Failed handlers create `BusinessRetryRecord`; `RetryDue` replays due failures and marks records succeeded or failed |
| Plugin subscriptions | Passed | `events.subscriptions` manifest block is parsed, normalized, validated, shown in install preflight, and imported into `ExtensionRegistry.Events` on enable |
| Plugin install/enable/disable/permissions/menu/audit/failure states | Passed | Existing lifecycle tests still cover install, enable, disable, permission/menu import, audit events, and invalid manifest failures; new tests add event import and invalid event contract failures |
| API/OpenAPI impact | Passed | Plugin install preflight response now includes `events`; `docs/api/openapi.yaml` and `internal/handler/http/openapi.yaml` are synchronized |
| Permission, audit, migration/seed impact | Passed | No new permission seed, core migration, or seed data; plugin catalog audit now records event subscription count |

### Verification Commands

```powershell
go test ./internal/event/... ./internal/plugin/...
git diff --no-index docs/api/openapi.yaml internal/handler/http/openapi.yaml
git diff --check
```

Result: Passed.

Notes:

- `git diff --check` exits successfully and prints only existing CRLF conversion warnings from the working tree.
- `docs/refactor/old/` was not updated.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F6-07` from `docs/refactor/current/pharma_oa_work_items.md`.

## F6-07: Implement Business Plugin Generator

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/generator/spec.go`
- `internal/domain/generator/spec_test.go`
- `internal/service/generator/service_impl.go`
- `internal/service/generator/backend_templates.go`
- `internal/service/generator/service_impl_test.go`
- `docs/refactor/current/pharma_oa_task_board.md`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Business plugin spec | Passed | `GeneratorSpec` now supports optional `PluginSpec` with plugin ID, version, data namespace, migration directory, frontend entry, UI mode, and event subscriptions |
| Current rule validation | Passed | Plugin ID, data namespace, frontend entry, UI mode, event names, handlers, duplicate subscriptions, and retry policies are validated without legacy compatibility output |
| Plugin output templates | Passed | Dry-run adds `examples/plugins/{plugin}/plugin.yaml`, plugin migration up/down SQL, plugin frontend API/store/view, README, and plugin acceptance test when `PluginSpec.Enabled` is true |
| API, UI, permissions, menu, migration, audit | Passed | Generated manifest declares permissions, UI menu, data manifest, plugin API routes under `/v1/plugins/{id}/api/...`, audit actions, and optional business event subscriptions |
| Tests that build/pass | Passed | Generator tests parse generated Go plugin acceptance test and validate plugin manifest/API/UI content; domain/service generator tests pass |
| Frontend build | Passed | Existing frontend build passes after generator changes |
| Permission, audit, migration/seed impact | Passed | No runtime permission seed, audit table, core migration, or seed data is introduced; this Work Item only extends generator templates and dry-run outputs |

### Verification Commands

```powershell
go test ./internal/domain/generator/... ./internal/service/generator/...
cd web; npm run build
git diff --check
```

Result: Passed.

Notes:

- `npm run build` prints the existing Sass legacy JS API and VueUse/Rollup annotation warnings, then exits successfully.
- `git diff --check` exits successfully and prints only existing CRLF conversion warnings from the working tree.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F7-01` from `docs/refactor/current/pharma_oa_work_items.md`.
