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

## F7-01: Implement Workflow Domain

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/workflow/types.go`
- `internal/domain/workflow/workflow_test.go`
- `internal/service/workflow/service.go`
- `internal/service/workflow/service_impl.go`
- `internal/service/workflow/memory_repository.go`
- `internal/service/workflow/service_impl_test.go`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Workflow definition model | Passed | `Definition`, `Node`, and `Transition` validate published workflow shape, node types, assignees, and exactly one start/end node |
| Workflow instance model | Passed | `Instance`, `Task`, `Action`, timeline, starter, current node, business target, and status fields are represented in the domain |
| Start action | Passed | Published definitions can start instances and create pending approval tasks; draft definitions cannot start |
| Approve/reject actions | Passed | Pending task assignees can approve or reject; terminal instances reject repeated actions |
| Withdraw action | Passed | Starter can withdraw a running instance and pending tasks become canceled |
| Transfer/copy actions | Passed | Transfer closes the original pending task and creates a pending task for the target; copy records a completed copied task without consuming the original |
| Service orchestration | Passed | `internal/service/workflow` provides repository interfaces, in-memory repository, and use-case service for create/publish/start/approve/reject/withdraw/transfer/copy |
| API/OpenAPI impact | Passed | No HTTP endpoint or response contract was added in this Work Item; F7-02 owns Workflow API/OpenAPI |
| Permission, audit, migration/seed impact | Passed | No runtime permission seed, audit sink, migration, or seed data is introduced; domain timeline actions prepare audit inputs for later API/service integration |

### Verification Commands

```powershell
go test ./internal/domain/workflow/... ./internal/service/workflow/...
go test ./internal/domain/... ./internal/service/workflow/...
git diff --check
```

Result: Passed after retry.

### Failure And Retry

- Initial failure: `TestWorkflowDefinitionValidationAndPublish` expected the missing-end-node invariant, but the fixture also omitted approval assignees and hit the earlier assignee invariant.
- Fix: added the approval assignee to that invalid-definition fixture.
- Retry result: Passed.

Notes:

- `git diff --check` exits successfully and prints only existing CRLF conversion warnings from the working tree.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F7-02` from `docs/refactor/current/pharma_oa_work_items.md`.

## F7-02: Implement Workflow API

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/handler/http/v1/workflow/handler.go`
- `internal/handler/http/v1/workflow/handler_test.go`
- `internal/handler/http/router.go`
- `internal/bootstrap/di.go`
- `internal/service/workflow/service.go`
- `internal/service/workflow/service_impl.go`
- `docs/api/openapi.yaml`
- `internal/handler/http/openapi.yaml`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Workflow definition API | Passed | Added create, get, and publish endpoints under `/v1/workflows/definitions` |
| Workflow instance API | Passed | Added start, get, approve, reject, withdraw, transfer, and copy endpoints under `/v1/workflows/instances` |
| Service support | Passed | Workflow service now exposes get-definition and get-instance methods for API readback |
| Permission records | Passed | Router registers workflow API permission resources for definition management, instance start/read, and task actions |
| Audit records | Passed | Handler test runs through router with request-audit middleware and verifies successful audit events for approve, reject, and transfer requests |
| Three workflow scenarios | Passed | API test covers approve, reject, and copy-transfer-approve scenarios end to end |
| OpenAPI sync | Passed | `docs/api/openapi.yaml` and `internal/handler/http/openapi.yaml` are identical and include workflow request/response schemas |
| Migration/seed impact | Passed | No core database migration or seed file is introduced; runtime permission catalog records are registered through the existing permission service |

### Verification Commands

```powershell
go test ./internal/handler/http/...
go test ./internal/service/workflow/...
go test ./internal/bootstrap/...
git diff --no-index docs/api/openapi.yaml internal/handler/http/openapi.yaml
git diff --check
```

Result: Passed after retry.

### Failure And Retry

- Initial failure: workflow handler test imported the root HTTP router from the same package test and created an import cycle.
- Fix: moved the test to external package `workflow_test`.
- Second failure: audit assertion expected 11 events, but the three API scenarios perform 10 workflow API calls.
- Fix: corrected the assertion to require one audit event per actual workflow API call.
- Retry result: Passed.

Notes:

- `git diff --check` exits successfully and prints only existing CRLF conversion warnings from the working tree.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F7-03` from `docs/refactor/current/pharma_oa_work_items.md`.

## F7-03: Implement Workflow UI

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `web/src/workflow/api.ts`
- `web/src/views/Workflow/index.vue`
- `web/src/router/index.ts`
- `web/src/navigation/menu.ts`
- `web/src/components/Layout/Sidebar.vue`
- `web/src/i18n/index.ts`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Workflow API client | Passed | Added typed frontend client for definition create/publish, instance start/get, approve, reject, withdraw, transfer, and copy |
| Workflow list views | Passed | Workflow page provides pending, approved, initiated-by-me, and copied-to-me summary filters over locally tracked instances refreshed from API |
| Launch page | Passed | Launch dialog initializes the demo definition, starts an instance through F7-02 API, and records it in the local workflow index |
| Approval page | Passed | Row actions open approve/reject dialogs for assigned pending tasks and update instance state through API |
| Timeline drawer | Passed | Detail drawer shows timeline entries and exposes transfer, copy, and withdraw actions |
| Required UI states | Passed | Browser smoke covered empty, API error, no-permission, saving action entry, destructive reject confirmation, successful approval, and 390px responsive layout |
| Route/menu integration | Passed | `/skoll/workflow` route and sidebar entry are available with workflow icon and label |

### Verification Commands

```powershell
cd web
npm run typecheck
npm run build
```

Browser smoke result:

```json
{
  "emptyVisible": true,
  "launchedRows": 1,
  "drawerVisible": true,
  "approvedVisible": true,
  "forbiddenVisible": true,
  "mobileOverflowCount": 0,
  "destructivePresent": true,
  "errorVisible": true
}
```

Result: Passed after retry.

### Failure And Retry

- Initial failure: `vue-tsc` rejected the page-local generic `DataTableColumn<WorkflowRow>[]` because `DataTable` exposes `DataTableColumn[]`.
- Fix: aligned the page column definition with the shared UI Kit prop type.
- Browser smoke found that failed instance refresh silently became an empty table when all indexed instances failed to load.
- Fix: show the page error state when stored workflow IDs exist and none can be refreshed.
- Retry result: Passed.

Notes:

- `npm run build` prints the existing Sass legacy JS API and VueUse/Rollup annotation warnings, then exits successfully.
- Browser smoke used Playwright with local Chrome and mocked `/skoll/v1/workflows/**` responses.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F7-04` from `docs/refactor/current/pharma_oa_work_items.md`.

## F7-04: Implement Form Builder Schema

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/form/schema.go`
- `internal/domain/form/schema_test.go`
- `internal/service/form/service.go`
- `internal/service/form/service_impl.go`
- `internal/service/form/memory_repository.go`
- `internal/service/form/service_impl_test.go`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Form schema model | Passed | Added `form.Schema` with ID, key, name, version, business type, description, fields, and audit metadata |
| Field types | Passed | Supports string, textarea, number, boolean, date, datetime, select, multi-select, dictionary, user, department, attachment, and detail-table fields |
| Validation metadata | Passed | Supports required, min, max, min/max length, pattern, enum, min-items, and max-items rules with pattern compilation checks |
| Detail table field | Passed | Detail-table config validates row limits, required columns, duplicate column keys, and rejects nested detail tables |
| Attachment field | Passed | Attachment config validates max files, max size, accepted extensions, and required flag |
| Pharma OA fixtures | Passed | Leave, reimbursement, purchase request, purchase inbound, and customer qualification fixtures are all expressible |
| Service layer | Passed | Added schema create/get/get-by-key service and clone-safe in-memory repository |
| API/OpenAPI impact | Passed | No HTTP endpoint or response contract was added; later F7 form API work can expose this schema through OpenAPI |
| Permission, audit, migration/seed impact | Passed | No permission seed, audit sink, core migration, or seed data is introduced in this Work Item |

### Verification Commands

```powershell
go test ./internal/domain/form/... ./internal/service/form/...
go test ./internal/domain/... ./internal/service/...
git diff --check
```

Result: Passed.

Notes:

- `git diff --check` exits successfully and prints only existing CRLF conversion warnings from the working tree.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F7-05` from `docs/refactor/current/pharma_oa_work_items.md`.

## F7-05: Implement Form Builder UI

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `web/src/form-builder/types.ts`
- `web/src/views/FormBuilder/index.vue`
- `web/src/views/Workflow/index.vue`
- `web/src/router/index.ts`
- `web/src/navigation/menu.ts`
- `web/src/components/Layout/Sidebar.vue`
- `web/src/components/Common/DataTable.vue`
- `web/src/i18n/index.ts`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Form designer | Passed | `/skoll/form-builder` provides schema metadata editing, field table, field drawer, required flag, select/dictionary/attachment/detail-table settings, and purchase request template |
| Preview | Passed | Preview tab renders saved/current fields, required marks, attachment hints, detail-table columns, and validation readiness |
| Save | Passed | Save validates schema and persists the current form schema to `skoll.formBuilder.schemas` |
| Version management | Passed | New version duplicates the current schema with an incremented version and saves alongside earlier versions |
| Workflow launch usage | Passed | `/skoll/workflow` launch dialog loads saved form schemas, applies selected form business type, and displays selected form fields before launch |
| Required UI states | Passed | Browser smoke covered loading path, empty state, corrupted-storage error state, no-permission state, save/saving path, destructive delete confirmation, and 390px responsive layout |
| Route/menu integration | Passed | `/skoll/form-builder` route and sidebar entry are available with form icon and i18n label |
| API/OpenAPI impact | Passed | No HTTP endpoint or response contract was added; OpenAPI remains unchanged because F7-05 is UI-only over local schema persistence |
| Permission, audit, migration/seed impact | Passed | Uses current frontend permission rule `form.schema.manage`; no backend permission seed, audit sink, core migration, or seed data is introduced in this Work Item |

### Verification Commands

```powershell
cd web
npm run typecheck
npm run build
git diff --check
```

Browser smoke result:

```json
{
  "emptyVisible": true,
  "previewVisible": true,
  "savedSchemas": 1,
  "versionCount": 2,
  "deleteConfirmVisible": true,
  "workflowFormPreview": true,
  "workflowBusinessType": "oa.purchase",
  "noPermissionVisible": true,
  "errorVisible": true,
  "mobile": {
    "viewport": 390,
    "documentScrollWidth": 390,
    "bodyScrollWidth": 390
  }
}
```

Result: Passed after retry.

### Failure And Retry

- Initial browser smoke failed to select the form in Workflow because each Playwright page used an isolated browser context and did not share localStorage.
- Retry: injected saved form schemas explicitly into the workflow smoke page.
- Initial mobile smoke treated table internals as page overflow.
- Fix: made shared `DataTable` constrain itself to its parent with explicit horizontal overflow handling and rechecked page-level scroll width at 390px.
- Retry result: Passed.

Notes:

- `npm run build` prints the existing Sass legacy JS API and VueUse/Rollup annotation warnings, then exits successfully.
- Browser smoke used Playwright with local Chrome and localStorage-backed form schemas.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F7-06` from `docs/refactor/current/pharma_oa_work_items.md`.

## F7-06: Implement Todo/Notification Center

- Status: Passed
- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/service/notification/service.go`
- `internal/service/notification/service_test.go`
- `web/src/notifications/types.ts`
- `web/src/views/TodoCenter/index.vue`
- `web/src/views/Workflow/index.vue`
- `web/src/router/index.ts`
- `web/src/navigation/menu.ts`
- `web/src/components/Layout/Sidebar.vue`
- `web/src/i18n/index.ts`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Todo service | Passed | `internal/service/notification` creates, lists, completes, and filters todo/message/reminder items with target paths and actor ownership checks |
| Reminder rules | Passed | Service supports reminder rules and emits due reminder items with business target links |
| Todo Center UI | Passed | `/skoll/todo` provides pending, done, message, and reminder views with summary filters, search, mark-done/read, target jump, demo reminder, and destructive clear confirmation |
| Workflow integration | Passed | Workflow launch creates a workflow todo; approve/reject/transfer closes the actor's workflow todo; transfer creates a target todo and copy creates a message |
| Target jump | Passed | Browser smoke opened a notification target and landed on `/skoll/workflow?instance=wf-1` |
| Required UI states | Passed | Browser smoke covered empty, data, error, no-permission, mark-done, destructive clear confirmation, and 390px responsive layout |
| Route/menu integration | Passed | `/skoll/todo` route and sidebar entry are available with notification icon and i18n label |
| API/OpenAPI impact | Passed | No HTTP endpoint or response contract was added; OpenAPI remains unchanged because this Work Item exposes service tests and a local frontend center only |
| Permission, audit, migration/seed impact | Passed | Uses current frontend permission rules `notification.read` and `notification.act`; no backend permission seed, audit sink, core migration, or seed data is introduced in this Work Item |

### Verification Commands

```powershell
go test ./internal/service/...
cd web
npm run typecheck
npm run build
git diff --check
```

Browser smoke result:

```json
{
  "emptyVisible": true,
  "pendingVisible": true,
  "jumpToWorkflow": true,
  "doneCount": 1,
  "messageVisible": true,
  "reminderVisible": true,
  "clearConfirmVisible": true,
  "noPermissionVisible": true,
  "errorVisible": true,
  "mobile": {
    "viewport": 390,
    "documentScrollWidth": 390,
    "bodyScrollWidth": 390
  }
}
```

Result: Passed after retry.

### Failure And Retry

- Initial browser smoke could not locate icon-only row action buttons by accessible name.
- Fix: added `aria-label` values for Todo Center open-target and mark-done actions.
- Second smoke failure was a broad test selector for `Messages` matching both description text and the summary button.
- Retry: used role-based button selectors; result passed.

Notes:

- `npm run build` prints the existing Sass legacy JS API and VueUse/Rollup annotation warnings, then exits successfully.
- Browser smoke used Playwright with local Chrome and localStorage-backed notification fixtures.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F7-07` from `docs/refactor/current/pharma_oa_work_items.md`.

## F7-07 Add Workflow Audit Fixtures

- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/testing/workflowaudit/fixture.go`
- `internal/testing/workflowaudit/replay.go`
- `internal/testing/workflowaudit/fixture_test.go`
- `scripts/smoke-workflow-audit-replay.ps1`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Workflow audit fixture | Passed | Fixture builds a real workflow definition, instance, tasks, timeline, and terminal approved approval-chain state |
| Audit event generation | Passed | Fixture emits start, copy, transfer, and approve audit events with actor, target, action, resource, result, and source data |
| Queryability | Passed | Fixture helpers query audit events by actor, action, resource type/id, and result |
| Replayability | Passed | Replay validates ordered start -> copy -> transfer -> approve records for the same workflow instance and returns terminal `approved` |
| Failure behavior | Passed | Replay rejects missing and reordered audit chains |
| Smoke script | Passed | `scripts/smoke-workflow-audit-replay.ps1` runs the replay smoke test through `go test` |
| API/OpenAPI impact | Passed | No HTTP endpoint, request, response, or OpenAPI contract was changed |
| Permission, audit, migration/seed impact | Passed | Adds audit fixture and replay test assets only; no runtime permission catalog, migration, or seed data is introduced |

### Verification Commands

```powershell
go test ./internal/testing/workflowaudit -v
go test ./...
powershell -ExecutionPolicy Bypass -File scripts/smoke-workflow-audit-replay.ps1
git diff --check
```

Smoke result:

```text
Workflow audit replay smoke passed.
```

Result: Passed after retry.

### Failure And Retry

- Initial smoke script passed `-count=$Count` as a single PowerShell token and Go reported `invalid count "$Count"`.
- Fix: changed the script to build a `go test` argument array and pass `-count` plus the resolved numeric value separately.
- Retry: workflow audit replay smoke passed.

Notes:

- `go test ./...` passed before the smoke script retry fix; the changed script was then independently validated by the smoke command.
- `git diff --check` reports only existing CRLF conversion warnings for tracked Markdown files.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F8-01` from `docs/refactor/current/pharma_oa_work_items.md`.

## F8-01 Create `pharma_oa` Plugin Skeleton

- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `plugins/pharma_oa/plugin.yaml`
- `plugins/pharma_oa/README.md`
- `plugins/pharma_oa/static/index.html`
- `plugins/pharma_oa/static/style.css`
- `plugins/pharma_oa/static/app.js`
- `internal/plugin/pharma_oa_manifest_test.go`
- `scripts/smoke-pharma-oa-plugin.ps1`
- `docs/refactor/current/pharma_oa_task_board.md`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Manifest skeleton | Passed | `plugins/pharma_oa/plugin.yaml` defines app-level `pharma_oa`, frontend entry `/skoll/plugins/pharma_oa`, config schema, i18n locales, and compatibility metadata |
| Menu group | Passed | Lifecycle smoke imports visible `plugin.pharma_oa` menu at `/skoll/plugins/pharma_oa` guarded by `pharma_oa.menu.read` |
| Permissions | Passed | Lifecycle smoke imports menu, plugin manage, seed apply, and API-derived seed read/apply permissions, then disables them on plugin disable |
| Routes | Passed | Route registry imports `GET /v1/plugins/pharma_oa/api/demo-seed/status` and `POST /v1/plugins/pharma_oa/api/demo-seed/apply` |
| Demo seed entry | Passed | Manifest declares seed config fields and demo seed API route contracts; static page exposes a reserved seed action entry |
| Static plugin page states | Passed | Static page includes loading, empty, error, no-permission, saving, destructive, ready, and responsive states for the integrated plugin entry |
| Lifecycle | Passed | Dedicated test installs, enables, disables, and blocks duplicate install for `pharma_oa` |
| Audit | Passed | Catalog audit sink records catalog import, route import, and catalog disable events with permission/menu/route/audit-action counts |
| API/OpenAPI impact | Passed | Plugin route contract is declared in manifest and exposed through route extension/preflight preview; core OpenAPI file is unchanged because no core HTTP handler was added |
| Permission, audit, migration/seed impact | Passed | Permission catalog and audit impacts are manifest-driven and verified; no migration is added; seed is a declared skeleton entry for later F12 data implementation |

### Verification Commands

```powershell
go test ./internal/plugin/...
powershell -ExecutionPolicy Bypass -File scripts/smoke-pharma-oa-plugin.ps1
git diff --check
```

Smoke result:

```text
Pharma OA plugin lifecycle smoke passed.
```

Result: Passed.

### Failure And Retry

- None.

Notes:

- `git diff --check` reports only existing CRLF conversion warnings for tracked Markdown files.
- Parent task board was updated from F7 `Todo` to `Done` and F8 `Todo` to `Doing` to match completed F7 work items and the active F8 batch.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F8-02` from `docs/refactor/current/pharma_oa_work_items.md`.

## F8-02 Implement Employee Management

- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/pharmaoa/employee.go`
- `internal/service/pharmaoa/employee_service.go`
- `internal/service/pharmaoa/employee_service_test.go`
- `internal/handler/http/v1/pharmaoa/employee_handler.go`
- `internal/handler/http/v1/pharmaoa/employee_handler_test.go`
- `internal/bootstrap/di.go`
- `internal/handler/http/router.go`
- `internal/handler/http/openapi.yaml`
- `docs/api/openapi.yaml`
- `internal/plugin/pharma_oa_manifest_test.go`
- `plugins/pharma_oa/plugin.yaml`
- `plugins/pharma_oa/README.md`
- `web/src/pharma-oa/api.ts`
- `web/src/views/PharmaEmployee/index.vue`
- `web/src/router/index.ts`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Employee records | Passed | Domain and service create/update/list employee records with code, name, department, position, phone, email, status, and certificate data |
| Department and position | Passed | API, service, and Vue table/drawer preserve department and position fields; browser smoke created `Quality` / `QA Specialist` |
| Certificate reminders | Passed | `QualificationReminders` returns active employees with certificates expiring within 30 days; browser smoke showed `Expiring certs` changing to `1` after create |
| Status flow | Passed | `MarkLeft` changes employee status and clears reminder eligibility; browser smoke completed destructive leave confirmation and showed Active `0`, Left `1`, Expiring certs `0` |
| API/OpenAPI | Passed | Core employee endpoints are registered under `/v1/pharma-oa/employees` and synced in both `internal/handler/http/openapi.yaml` and `docs/api/openapi.yaml` |
| Permissions | Passed | `RegisterEmployeePermissions` registers read/create/update/leave/reminder permissions; plugin manifest declares employee permission and route contracts |
| Audit | Passed | Service appends `pharma_oa.employee.create`, `pharma_oa.employee.update`, and `pharma_oa.employee.leave` audit records with resource `pharma_oa_employee` |
| Plugin lifecycle impact | Passed | Plugin manifest and manifest test cover employee menu entry, permission catalog impact, route catalog impact, disable handling, and failure state coverage inherited from lifecycle smoke |
| Frontend states | Passed | Employee page covers loading, empty, error, no-permission, saving, destructive confirmation, and responsive layout states using shared UI kit components |
| Migration and seed impact | Passed | No database migration is added for this work item; current service is in-memory for the vertical slice; demo seed remains manifest-only until later F12 data work |

### Verification Commands

```powershell
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/plugin
go test ./...
cd web; npm run typecheck
cd web; npm run build
git diff --check
```

Browser smoke:

- Started local backend/frontend temporarily, logged in through the real login page with seeded admin credentials.
- Opened `/skoll/pharma-oa/employees`.
- Verified title, toolbar, empty table state, create drawer, save flow, certificate reminder count, leave dialog, destructive confirmation, and mobile viewport `390x844`.
- Temporary backend/frontend processes were stopped after validation.

Result: Passed.

### Failure And Retry

- Initial targeted Go test failed because `internal/plugin/pharma_oa_manifest_test.go` still expected the F8-01 seed-only manifest counts.
- Fix: updated the manifest test to assert employee permissions, employee route contracts, and API-derived catalog counts.
- Retry: targeted Go test, full Go test, frontend typecheck, frontend build, diff check, and browser smoke all passed.

Notes:

- `npm run build` reports existing Sass legacy JS API and Rollup annotation warnings, but exits successfully.
- `git diff --check` reports only Windows CRLF conversion warnings for tracked files.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F8-03` from `docs/refactor/current/pharma_oa_work_items.md`.

## F8-03 Implement Product/Drug Master Data

- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/pharmaoa/product.go`
- `internal/service/pharmaoa/product_service.go`
- `internal/service/pharmaoa/product_service_test.go`
- `internal/handler/http/v1/pharmaoa/product_handler.go`
- `internal/handler/http/v1/pharmaoa/product_handler_test.go`
- `internal/bootstrap/di.go`
- `internal/handler/http/router.go`
- `internal/handler/http/openapi.yaml`
- `docs/api/openapi.yaml`
- `internal/plugin/pharma_oa_manifest_test.go`
- `plugins/pharma_oa/plugin.yaml`
- `plugins/pharma_oa/README.md`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Product records | Passed | `Product` domain and `ProductService` create/list/update records with code, name, spec, dosage form, manufacturer, approval number, status, and audit metadata |
| Drug attributes | Passed | Service tests cover spec, dosage form, manufacturer, approval number, and temperature range `2-8` Celsius |
| Temperature validation | Passed | Domain validation rejects required temperature ranges where min exceeds max and normalizes non-required ranges |
| CRUD | Passed | Handler test creates, lists, imports, and disables products through `/v1/pharma-oa/products` routes |
| Disable flow | Passed | `Disable` moves products to `disabled`, records reason, and emits `pharma_oa.product.disable` audit action |
| Import validation | Passed | Import fixture test accepts valid rows, rejects duplicate import codes, rejects incomplete rows, and reports row numbers/messages |
| API/OpenAPI | Passed | Product list/create/update/disable/import routes are registered and synced in both `internal/handler/http/openapi.yaml` and `docs/api/openapi.yaml` |
| Permissions | Passed | Product read/create/update/disable/import permissions are registered in HTTP startup and declared in plugin manifest |
| Audit | Passed | Product create/update/disable/import append audit records with resource `pharma_oa_product`; plugin lifecycle audit counts include product route/permission metadata |
| Plugin lifecycle impact | Passed | Manifest test validates install preflight, enable, disable, permission catalog, route registry, audit action counts, and duplicate-install failure state after product additions |
| Migration and seed impact | Passed | No database migration is added for this work item; product master data is an in-memory vertical slice; demo seed remains manifest-only until F12 data work |
| Frontend impact | Passed | No frontend route/page was changed in F8-03; frontend build was not required by the work item and no static UI was introduced |

### Verification Commands

```powershell
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/plugin
go test ./...
git diff --check
codegraph sync .
```

Result: Passed.

### Failure And Retry

- None.

Notes:

- `git diff --check` reports only Windows CRLF conversion warnings for tracked files.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F8-04` from `docs/refactor/current/pharma_oa_work_items.md`.

## F8-04 Implement Supplier Management

- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/pharmaoa/supplier.go`
- `internal/service/pharmaoa/supplier_service.go`
- `internal/service/pharmaoa/supplier_service_test.go`
- `internal/handler/http/v1/pharmaoa/supplier_handler.go`
- `internal/handler/http/v1/pharmaoa/supplier_handler_test.go`
- `internal/bootstrap/di.go`
- `internal/handler/http/router.go`
- `internal/handler/http/openapi.yaml`
- `docs/api/openapi.yaml`
- `internal/plugin/pharma_oa_manifest_test.go`
- `plugins/pharma_oa/plugin.yaml`
- `plugins/pharma_oa/README.md`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Supplier records | Passed | `Supplier` domain and `SupplierService` create/list/update/disable records with code, name, rating, status, contacts, qualifications, attachments, and audit metadata |
| Contacts | Passed | Service and handler tests preserve supplier contact name, phone, email, and position fields |
| Qualification attachments | Passed | Domain validation accepts safe metadata and rejects missing file IDs/names, negative file size, path traversal, path separators, and executable file extensions |
| Rating and status | Passed | Rating defaults to `3`, validates `0-5`, and disabled suppliers are excluded from active purchase eligibility |
| Expiry reminders | Passed | `QualificationReminders` returns active suppliers with qualifications expiring before the requested deadline |
| Purchase blocking | Passed | `ValidatePurchaseSupplier` and `/v1/pharma-oa/suppliers/{id}/purchase-eligibility` block disabled or expired suppliers from purchase usage |
| API/OpenAPI | Passed | Supplier list/create/update/disable/reminder/purchase-eligibility routes are registered and synced in both `internal/handler/http/openapi.yaml` and `docs/api/openapi.yaml` |
| Permissions | Passed | Supplier read/create/update/disable/reminder/purchase permissions are registered in HTTP startup and declared in the plugin manifest |
| Audit | Passed | Supplier create/update/disable append audit records with resource `pharma_oa_supplier`; plugin lifecycle audit counts include supplier route and permission metadata |
| Plugin lifecycle impact | Passed | Manifest test validates install preflight, enable, disable, permission catalog, route registry, audit action counts, and duplicate-install failure state after supplier additions |
| Migration and seed impact | Passed | No database migration is added for this work item; supplier management is an in-memory vertical slice; demo seed remains manifest-only until F12 data work |
| Frontend impact | Passed | No frontend route/page was changed in F8-04; frontend build was not required by the work item and no static UI was introduced |

### Verification Commands

```powershell
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/plugin
go test ./...
git diff --check
codegraph sync .
```

Result: Passed after retry.

### Failure And Retry

- Initial targeted Go test failed because plugin manifest validation rejected permission key `pharma_oa.supplier.purchase.validate` as too long.
- Fix: shortened the permission key to `pharma_oa.supplier.purchase` across handler registration, plugin manifest, and manifest assertions.
- Retry: targeted Go test, full Go test, diff check, and CodeGraph sync all passed.

Notes:

- `git diff --check` reports only Windows CRLF conversion warnings for tracked files.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F8-05` from `docs/refactor/current/pharma_oa_work_items.md`.

## F8-05 Implement Customer Management

- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/pharmaoa/customer.go`
- `internal/service/pharmaoa/customer_service.go`
- `internal/service/pharmaoa/customer_service_test.go`
- `internal/handler/http/v1/pharmaoa/customer_handler.go`
- `internal/handler/http/v1/pharmaoa/customer_handler_test.go`
- `internal/bootstrap/di.go`
- `internal/handler/http/router.go`
- `internal/handler/http/openapi.yaml`
- `docs/api/openapi.yaml`
- `internal/plugin/pharma_oa_manifest_test.go`
- `plugins/pharma_oa/plugin.yaml`
- `plugins/pharma_oa/README.md`
- `web/src/pharma-oa/api.ts`
- `web/src/views/PharmaCustomer/index.vue`
- `web/src/router/index.ts`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Customer records | Passed | `Customer` domain and `CustomerService` create/list/update/disable records with code, name, region, organization, owner, rating, contacts, qualifications, attachments, and audit metadata |
| Contacts | Passed | Service, handler, and browser smoke preserve primary contact name/phone/email fields |
| Qualification attachments | Passed | Domain validation accepts safe metadata and rejects missing file IDs/names, negative size, path traversal, path separators, and executable extensions |
| Region ownership | Passed | Customer records require `region`, `organizationId`, and `ownerId`; list filters support region and service tests cover owner/org scope filtering |
| Organization/owner isolation | Passed | Service and handler tests reject cross-owner/cross-organization update attempts and hide records outside requested scope |
| Expired qualification sales block | Passed | `ValidateSalesCustomer` and `/v1/pharma-oa/customers/{id}/sales-eligibility` block disabled or expired customers from sales usage |
| API/OpenAPI | Passed | Customer list/create/update/disable/reminder/sales-eligibility routes are registered and synced in both `internal/handler/http/openapi.yaml` and `docs/api/openapi.yaml` |
| Permissions | Passed | Customer read/create/update/disable/reminder/sales permissions are registered in HTTP startup and declared in plugin manifest |
| Audit | Passed | Customer create/update/disable append audit records with resource `pharma_oa_customer`; plugin lifecycle audit counts include customer route and permission metadata |
| Plugin lifecycle impact | Passed | Manifest test validates install preflight, enable, disable, permission catalog, route registry, audit action counts, and duplicate-install failure state after customer additions |
| Frontend states | Passed | `/skoll/pharma-oa/customers` covers loading, empty, error, saving, destructive disable confirmation, sales eligibility action, and 390px responsive layout; route guard blocks users without `pharma_oa.customer.read` before page entry |
| Migration and seed impact | Passed | No database migration is added for this work item; customer management is an in-memory vertical slice; demo seed remains manifest-only until F12 data work |

### Verification Commands

```powershell
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/plugin
go test ./...
cd web; npm run typecheck
cd web; npm run build
git diff --check
codegraph sync .
```

Browser smoke:

```json
{
  "emptyVisible": true,
  "createdVisible": true,
  "disabledVisible": true,
  "errorState": "Request failed / service unavailable",
  "mobileWidth": { "body": 390, "doc": 390, "viewport": 390 }
}
```

Result: Passed after retry.

### Failure And Retry

- Initial browser smoke used overly broad selectors for `Name` and `Disabled`; selectors were narrowed to exact labels/table status.
- Initial no-permission browser attempt did not show the page-local no-permission block because the route guard correctly redirects before entering protected routes; this was accepted as route-level no-permission enforcement for the customer route.
- Retry: targeted Go tests, full Go tests, frontend typecheck, frontend build, browser smoke, diff check, and CodeGraph sync passed.

Notes:

- `npm run build` reports existing Sass legacy JS API and Rollup annotation warnings, but exits successfully.
- `git diff --check` reports only Windows CRLF conversion warnings for tracked files.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F8-06` from `docs/refactor/current/pharma_oa_work_items.md`.

## F8-06 Implement Warehouse And Location

- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/pharmaoa/warehouse.go`
- `internal/service/pharmaoa/warehouse_service.go`
- `internal/service/pharmaoa/warehouse_service_test.go`
- `internal/handler/http/v1/pharmaoa/warehouse_handler.go`
- `internal/handler/http/v1/pharmaoa/warehouse_handler_test.go`
- `internal/bootstrap/di.go`
- `internal/handler/http/router.go`
- `internal/handler/http/openapi.yaml`
- `docs/api/openapi.yaml`
- `internal/plugin/pharma_oa_manifest_test.go`
- `plugins/pharma_oa/plugin.yaml`
- `plugins/pharma_oa/README.md`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Warehouse records | Passed | `Warehouse` domain and `WarehouseService` create/list/update/disable records with code, name, region, status, audit metadata, and nested areas/locations |
| Area and location records | Passed | Domain normalization validates area/location IDs, codes, names, status values, and duplicate IDs/codes |
| Temperature attributes | Passed | Warehouse, area, and location temperature attributes normalize non-controlled storage and reject controlled ranges where `minCelsius` exceeds `maxCelsius` |
| Inbound/outbound selection guard | Passed | `ValidateMovementLocation` and `/v1/pharma-oa/warehouses/{id}/movement-eligibility` allow only enabled warehouses, enabled areas, and enabled locations |
| API/OpenAPI | Passed | Warehouse list/create/update/disable/movement-eligibility routes are registered and synced in both `internal/handler/http/openapi.yaml` and `docs/api/openapi.yaml` |
| Permissions | Passed | Warehouse read/create/update/disable/movement permissions are registered in HTTP startup and declared in the plugin manifest |
| Audit | Passed | Warehouse create/update/disable append audit records with resource `pharma_oa_warehouse`; plugin lifecycle audit counts include warehouse route and permission metadata |
| Plugin lifecycle impact | Passed | Manifest test validates install preflight, enable, disable, permission catalog, route registry, audit action counts, and duplicate-install failure state after warehouse additions |
| Migration and seed impact | Passed | No database migration is added for this work item; warehouse management is an in-memory vertical slice; demo seed remains manifest-only until F12 data work |
| Frontend impact | Passed | No frontend route/page was changed in F8-06; frontend build was not required by the work item and no static UI was introduced |

### Verification Commands

```powershell
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/plugin
go test ./...
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
git diff --check
codegraph sync .
```

Result: Passed.

### Failure And Retry

- None.

Notes:

- `git diff --check` reports only Windows CRLF conversion warnings for tracked files.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F8-07` from `docs/refactor/current/pharma_oa_work_items.md`.

## F8-07 Implement Master Data Import/Export

- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/service/pharmaoa/master_data_exchange_service.go`
- `internal/service/pharmaoa/master_data_exchange_service_test.go`
- `internal/handler/http/v1/pharmaoa/master_data_exchange_handler.go`
- `internal/handler/http/v1/pharmaoa/master_data_exchange_handler_test.go`
- `internal/bootstrap/di.go`
- `internal/handler/http/router.go`
- `internal/handler/http/openapi.yaml`
- `docs/api/openapi.yaml`
- `internal/plugin/pharma_oa_manifest_test.go`
- `plugins/pharma_oa/plugin.yaml`
- `plugins/pharma_oa/README.md`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Excel templates | Passed | `MasterDataExchangeService.Template` returns `.xlsx` filename, Excel MIME type, header rows, and base64 OpenXML workbook content for employees, products, suppliers, and customers |
| Import validation | Passed | Import validates each row through the existing employee/product/supplier/customer services and preserves domain validation rules |
| Error report | Passed | Invalid rows return `row`, `code`, and clear `message`; tests cover missing code and missing employee name |
| Export job | Passed | Export returns a completed job with `.xlsx` filename, Excel MIME type, base64 workbook content, headers, and exported rows |
| Four resource coverage | Passed | Fixture tests import and export employees, products, suppliers, and customers |
| API/OpenAPI | Passed | Master data template/import/export routes are registered under `/v1/pharma-oa/master-data/*` and synced in both OpenAPI files |
| Permissions | Passed | Master data template/import/export permissions are registered in HTTP startup and declared in the plugin manifest |
| Audit and plugin lifecycle impact | Passed | Plugin lifecycle test validates install preflight, enable, disable, route registry, permission catalog, and audit counts after master-data route additions |
| Migration and seed impact | Passed | No database migration or seed is added; exchange service uses current in-memory master data services |
| Frontend impact | Passed | No frontend route/page was changed in F8-07; no static UI was introduced |

### Verification Commands

```powershell
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/plugin
go test ./...
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
git diff --check
codegraph sync .
```

Result: Passed.

### Failure And Retry

- Initial implementation returned CSV-oriented template/export metadata.
- Fix: added a minimal OpenXML `.xlsx` generator and base64 workbook payloads for templates and export jobs.
- Retry: targeted Go tests, full Go tests, OpenAPI sync check, diff check, and CodeGraph sync all passed.

Notes:

- `git diff --check` reports only Windows CRLF conversion warnings for tracked files.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F9-01` from `docs/refactor/current/pharma_oa_work_items.md`.

## F9-01 Implement Inventory Domain Model

- Date: 2026-07-04
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/pharmaoa/inventory.go`
- `internal/domain/pharmaoa/inventory_test.go`
- `internal/service/pharmaoa/inventory_service.go`
- `internal/service/pharmaoa/inventory_service_test.go`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Stock balance model | Passed | `StockBalance` tracks product, warehouse, area, location, batch, quantity, locked quantity, and available quantity |
| Stock ledger model | Passed | `StockLedgerEntry` validates operation, position, non-zero delta, balance-after, reference, and occurrence time |
| Batch and expiry model | Passed | `StockBatch` tracks product, batch number, production date, and expiry; domain test rejects expiry before production date |
| Stock lock model | Passed | `StockLock` supports active/released status; service lock reduces available stock and blocks outbound when insufficient |
| Inbound ledger | Passed | `InventoryService.Inbound` creates batch/balance as needed, increases stock, and appends an inbound ledger entry |
| Outbound ledger | Passed | `InventoryService.Outbound` checks available stock, decreases quantity, and appends an outbound ledger entry |
| Stocktake ledger | Passed | `InventoryService.Stocktake` calculates quantity delta from actual count and appends a stocktake ledger entry |
| Transfer ledger | Passed | `InventoryService.Transfer` decreases source, increases destination, and appends transfer-out plus transfer-in ledger entries |
| Immutable ledger behavior | Passed | `ListLedger` returns copies; mutating a returned entry does not alter stored ledger history |
| API/OpenAPI impact | Passed | No HTTP endpoint is added in F9-01; OpenAPI remains unchanged |
| Permission, audit, migration/seed impact | Passed | Service emits audit hooks when audit service is present; no new permission, migration, or seed is added for the in-memory domain slice |

### Verification Commands

```powershell
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa
go test ./...
git diff --check
codegraph sync .
```

Result: Passed.

### Failure And Retry

- None.

Notes:

- `git diff --check` reports only Windows CRLF conversion warnings for tracked files.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F9-02` from `docs/refactor/current/pharma_oa_work_items.md`.

## F9-02 Implement Purchase Request And Purchase Order

- Date: 2026-07-13
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/pharmaoa/purchase.go`
- `internal/domain/pharmaoa/purchase_test.go`
- `internal/service/pharmaoa/purchase_service.go`
- `internal/service/pharmaoa/purchase_service_test.go`
- `internal/handler/http/v1/pharmaoa/purchase_handler.go`
- `internal/handler/http/v1/pharmaoa/purchase_handler_test.go`
- `internal/bootstrap/di.go`
- `internal/handler/http/router.go`
- `internal/handler/http/openapi.yaml`
- `docs/api/openapi.yaml`
- `internal/plugin/pharma_oa_manifest_test.go`
- `plugins/pharma_oa/plugin.yaml`
- `plugins/pharma_oa/README.md`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Purchase request model | Passed | Domain model validates request identity, supplier, requester, approver, non-empty positive-quantity lines, totals, and pending/approved/rejected lifecycle |
| Workflow approval | Passed | Creating a request publishes and starts a dedicated approval workflow bound to business type `pharma_oa.purchase_request`; only the configured assignee can approve or reject |
| Purchase order generation | Passed | Approved workflow generates one open purchase order linked to the request with copied supplier, lines, amount, approver, and approval time |
| Duplicate execution safety | Passed | Sequential and concurrent repeated approval return the same order; service test confirms only one order is stored |
| Supplier qualification | Passed | Supplier purchase eligibility is checked before workflow creation and rechecked before approval; expired or disabled suppliers are blocked |
| API/OpenAPI | Passed | Seven request/order routes are registered; explicit request/response schemas are identical in `internal/handler/http/openapi.yaml` and `docs/api/openapi.yaml` |
| Permissions | Passed | Purchase read/create/approve/reject and order-read resources are registered at startup and declared in the plugin manifest |
| Audit | Passed | Request create, approval, rejection, and order creation append `pharma_oa_purchase` audit records; plugin lifecycle audit counts cover 39 routes and 37 unique audit actions |
| Plugin lifecycle | Passed | Manifest test covers preflight, install, enable, disable, permission catalog, route registry, audit counts, and duplicate-install failure with purchase additions |
| Migration and seed impact | Passed | No migration or seed is added; this Work Item follows the current in-memory pharma OA vertical-slice pattern |
| Frontend impact | Passed | No frontend route/page was added in F9-02; no static UI or incomplete frontend state was introduced |
| Local runtime | Passed | Rebuilt backend returns 200 from `/skoll/health`; served OpenAPI contains purchase request and purchase order paths |

### Verification Commands

```powershell
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/plugin
go test ./...
go build -o .\tmp\skoll-f9-02.exe .\cmd\skoll
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
git diff --check
codegraph sync .
```

Result: Passed after retries.

### Failure And Retry

- First plugin lifecycle run failed because 39 routes contain 37 unique audit actions; corrected the preflight audit-action assertion.
- Second run failed because existing demo-seed route assertions retained their old indexes after seven purchase routes were inserted; shifted them to the new manifest positions.
- Third run exposed that catalog audit `Permissions` counts declaration-to-route bindings (38 + 39 = 77), while `AuditActions` remains deduplicated at 37; corrected the audit count assertion.
- Initial local runtime check inspected the response as bytes and falsely reported absent purchase paths; decoded the served UTF-8 OpenAPI and confirmed both paths are present after rebuilding the binary.
- Retry: targeted tests, full Go tests, build, health/OpenAPI runtime smoke, OpenAPI sync, diff check, and CodeGraph sync passed.

Notes:

- `git diff --check` reports only Windows CRLF conversion warnings for tracked files.
- Existing unrelated working-tree changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included in this Work Item.

### Next Step

- Claim `F9-03` from `docs/refactor/current/pharma_oa_work_items.md`.

## F9-03 Implement Purchase Inbound

- Date: 2026-07-13
- Executor: Codex
- Commit: pending

### Changed Files

- `internal/domain/pharmaoa/purchase_inbound.go`
- `internal/service/pharmaoa/purchase_inbound_service.go`
- `internal/service/pharmaoa/purchase_inbound_service_test.go`
- `internal/handler/http/v1/pharmaoa/purchase_inbound_handler.go`
- `internal/bootstrap/di.go`
- `internal/handler/http/router.go`
- `internal/handler/http/openapi.yaml`
- `docs/api/openapi.yaml`
- `internal/plugin/pharma_oa_manifest_test.go`
- `plugins/pharma_oa/plugin.yaml`
- `plugins/pharma_oa/README.md`
- `web/src/pharma-oa/api.ts`
- `web/src/views/PharmaPurchaseInbound/index.vue`
- `web/src/router/index.ts`
- `docs/refactor/current/pharma_oa_work_items.md`
- `docs/refactor/current/pharma_oa_acceptance_log.md`

### Acceptance

| Item | Result | Evidence |
| --- | --- | --- |
| Inbound order | Passed | Completed inbound records link approved purchase orders to warehouse/area/location, receiver, time, lines, and attachments |
| Purchase validation | Passed | Service rejects missing orders, products outside the order, non-whole order quantities, and quantities exceeding the purchase order |
| Warehouse validation | Passed | Inbound requires an enabled warehouse, area, and location through `ValidateMovementLocation` |
| Batch and expiry | Passed | Each line requires batch number, production date, and expiry after production; inventory creates the stock batch and returns its batch ID |
| Inventory and ledger | Passed | Service fixture receives five units, increases one stock balance to five, and writes one immutable inbound ledger linked to the inbound ID |
| Attachments | Passed | Attachment metadata requires safe file ID/name and non-negative size; path traversal and separators are rejected |
| Audit | Passed | Inventory writes `pharma_oa.inventory.inbound`; inbound completion writes `pharma_oa.inbound.create` with order, line, and attachment counts |
| API/OpenAPI/permissions | Passed | Three routes, explicit schemas, read/create permissions, startup registration, and synchronized OpenAPI files are present |
| Plugin lifecycle | Passed | Manifest lifecycle covers 40 declarations, 42 routes, 39 unique audit actions, enable/disable, catalogs, and duplicate install failure |
| Frontend states | Passed | Page covers loading, empty, API error, route/page no-permission, saving, disabled create without orders, and responsive layout |
| Browser | Passed | Dedicated 5174 frontend against the F9-03 backend rendered the empty state; 390px viewport had no horizontal overflow or button overlap |
| Migration and seed impact | Passed | No migration or seed is added; inbound uses current in-memory purchase, warehouse, and inventory services |

### Verification Commands

```powershell
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/plugin
go test ./...
cd web; npm run typecheck
cd web; npm run build
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
git diff --check
codegraph sync .
```

Result: Passed after retry.

### Failure And Retry

- Initial targeted test failed because warehouse eligibility was called with three IDs instead of `WarehouseMovementLocationInput`; corrected the service call and reran successfully.
- Initial browser run used the existing 5173 proxy bound to another developer's 8080 backend and returned 404; started an isolated 5174 frontend pointing at the validated 18080 backend and reran successfully.

Notes:

- Frontend build reports existing Sass legacy API and Rollup annotation warnings but exits successfully.
- `git diff --check` reports only Windows CRLF conversion warnings.
- Existing unrelated changes in `docs/README.md`, `docs/collaboration.md`, `.codegraph/`, `.vscode/`, and `AGENTS.md` were not included.

### Next Step

- Claim `F9-04` from `docs/refactor/current/pharma_oa_work_items.md`.

## F9-04 Implement Sales Order And Sales Outbound

- Date: 2026-07-13
- Executor: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Dependencies: F9-01 and F8-05 are Done.

### Delivery

- Added sales orders with customer qualification validation, line totals, immutable order snapshots, list/detail APIs, audit actions, and least-privilege permissions.
- Added sales outbound with a second customer qualification check, warehouse location validation, order quantity validation, batch stock availability preflight, inventory deduction, and outbound ledger references.
- Added denial tests proving expired customer qualification and insufficient stock cannot change inventory or append outbound ledger entries.
- Added `/v1/pharma-oa/sales-orders` and `/v1/pharma-oa/sales-outbounds` API contracts, synchronized embedded OpenAPI, plugin route declarations, lifecycle permissions, and catalog audit counts.
- Added `/skoll/pharma-oa/sales` with order/outbound tabs and loading, empty, error, no-permission, saving, disabled-action, and responsive states.
- Migration and seed impact: none; the current milestone uses the existing in-memory customer, warehouse, and inventory services.

### Retry Evidence

1. Plugin validation initially failed because four-segment sales audit actions violated the manifest's three-segment audit action rule. Audit actions were renamed to three-segment forms and plugin installation/lifecycle tests passed on retry.
2. Browser acceptance initially found `New order` enabled when no eligible customers were available. The action now disables for an empty customer list and passed desktop/mobile browser revalidation.

### Acceptance

| Check | Result | Evidence |
| --- | --- | --- |
| Customer qualification | Passed | Expired qualification blocks outbound without changing the seeded balance |
| Inventory and ledger | Passed | Successful outbound reduces stock from 10 to 6 and appends a `-4` ledger entry with balance-after 6 |
| Failure atomicity | Passed | Insufficient stock is rejected before mutation and leaves only the original inbound ledger |
| API and OpenAPI | Passed | Sales order/outbound list, create, detail routes are wired and both OpenAPI files are byte-equivalent |
| Permissions and plugin lifecycle | Passed | Four sales permissions and six routes pass install, enable, disable, audit-count, and duplicate-install tests |
| Frontend states | Passed | Both tabs render empty state; customer/order absence disables create; saving/error/no-permission states are implemented |
| Responsive browser | Passed | Desktop and 390px checks show no horizontal overflow, button overlap, or clipped button text |

### Verification Commands

```text
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http
go test ./internal/plugin ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa
go test ./...
cd web; npm run typecheck
cd web; npm run build
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
codegraph sync .
git diff --check
```

## F9-05 Implement Stocktake And Transfer

- Date: 2026-07-13
- Executor: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Dependencies: F9-01 is Done.

### Delivery

- Added stocktake orders that snapshot system stock, require a non-zero physical-count difference, create an assigned approval workflow, and defer inventory mutation until approval.
- Added idempotent approval and rejection handling, stale-balance protection, immutable stocktake ledger references, and create/approve/reject audit actions.
- Added atomic cross-warehouse transfers with source and destination validation, preserved total stock, paired outbound/inbound ledger references, and transfer audit actions.
- Added list/detail/create/action APIs, least-privilege permissions, synchronized OpenAPI contracts, plugin extension routes, and lifecycle catalog assertions.
- Added fixture coverage for unauthorized approval, duplicate approval, stale stock, paired ledgers, and invalid-input side-effect prevention.
- Migration and seed impact: none; the current milestone uses the existing in-memory workflow, warehouse, and inventory services.

### Retry Evidence

1. The first handler/bootstrap validation run failed because the prior F9-04 backend still occupied port `18081`. That owned process was identified and stopped; the same command passed on retry.
2. Pre-commit review found invalid stocktake or transfer input could reach side-effecting work before final domain validation. Validation was moved ahead of workflow/inventory calls, creation critical sections were serialized, a no-side-effects fixture was added, and targeted plus full tests passed on retry.

### Acceptance

| Check | Result | Evidence |
| --- | --- | --- |
| Difference approval | Passed | A stocktake with system quantity 10 and actual quantity 8 remains pending with stock at 10 until the assigned approver acts |
| Approval idempotency | Passed | Approval posts one `-2` stocktake ledger entry; repeated approval returns the same ledger ID without another mutation |
| Rejection and stale stock | Passed | Rejection does not mutate inventory; a changed balance blocks approval while the order and workflow remain pending |
| Cross-warehouse transfer | Passed | Moving four units creates distinct `transfer_out` and `transfer_in` ledger entries and preserves total stock at 10 |
| Failure atomicity | Passed | Invalid stocktake input creates no workflow definition, and invalid transfer input creates no inventory or ledger changes |
| API and OpenAPI | Passed | Eight stocktake/transfer routes and explicit request/response schemas are wired; both OpenAPI files are byte-equivalent |
| Permissions and audit | Passed | Six least-privilege permissions and six operation audit actions are registered with startup and plugin catalogs |
| Plugin lifecycle | Passed | Manifest tests cover 50 declarations, 56 routes, 49 unique audit actions, enable/disable, catalog effects, and duplicate-install failure |
| Runtime health | Passed | Isolated memory-mode backend returned HTTP 200 from `http://127.0.0.1:18085/skoll/health` and was stopped after verification |
| Migration and seed impact | Passed | No migration or seed changes are required for the current in-memory milestone implementation |

### Verification Commands

```text
go test ./internal/service/pharmaoa -run 'TestStocktake|TestCrossWarehouseTransfer|TestInvalidInventoryOperationInput' -count=1 -v
go test ./internal/handler/http/... ./internal/plugin/... ./internal/service/pharmaoa
go test ./...
go build -o tmp/skoll-f9-05.exe ./cmd/skoll
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
git diff --check
codegraph sync .
```

Result: Passed after retry.

### Next Step

- Claim `F9-06` from `docs/refactor/current/pharma_oa_work_items.md`.

## F9-06 Implement Inventory Alerts

- Date: 2026-07-13
- Executor: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Dependencies: F9-01 is Done.

### Delivery

- Added retryable inventory alert scan jobs with pending, running, succeeded, and failed lifecycle states, timestamps, counters, error details, and execution logs.
- Added near-expiry, low-stock, and over-stock rules over inventory balances and batch expiry data, including active/resolved alert lifecycle and deterministic duplicate suppression.
- Added notification-center reminders owned by the configured recipient and actionable links carrying warehouse, balance, and batch context.
- Added success, alert creation, resolution, and failure audit records; hardened the shared audit ID generator with a process-wide atomic sequence for burst-write uniqueness.
- Added alert/job list, scan, and failed-job retry APIs, synchronized OpenAPI schemas, two least-privilege permissions, plugin routes, and lifecycle catalog assertions.
- Migration and seed impact: none; job, alert, and notification state use the current in-memory milestone services.

### Retry Evidence

1. The first plugin lifecycle run failed because existing demo-seed route assertions retained their old runtime snapshot indexes after four alert routes were inserted. The assertions were moved to indexes 58 and 59; plugin lifecycle validation passed on retry.
2. The first audit fixture found only one of three burst `pharma_oa.alert.create` records because Windows returned equal nanosecond timestamps and the audit ID generator used that timestamp alone. A process-wide atomic suffix and a 100-record burst test were added; all audit and alert fixtures passed on retry.

### Acceptance

| Check | Result | Evidence |
| --- | --- | --- |
| Alert rules | Passed | Fixtures independently produce near-expiry, low-stock, and over-stock alerts from batch expiry and quantity thresholds |
| Notification center | Passed | Three active conditions create three pending reminder items for the configured recipient with actionable target paths |
| Stock detail links | Passed | Each reminder targets `/skoll/pharma-oa/warehouses` with encoded `warehouseId`, `balanceId`, and `batchId` context |
| Duplicate safety | Passed | A second identical scan matches all three conditions but creates zero new alerts or notifications |
| Resolution | Passed | Replenishing low stock resolves the alert and marks its notification done on the next scan |
| Failure and retry | Passed | Reader failure leaves a queryable failed job with error log; retry reuses the job, increments retry count, and succeeds |
| Audit traceability | Passed | `system` audit queries return all alert creation and successful run actions; 100 burst appends retain 100 unique records |
| API and OpenAPI | Passed | Alert list, job list, run, and retry routes are wired with explicit schemas; both OpenAPI files are byte-equivalent |
| Permissions and plugin lifecycle | Passed | Read/run permissions and four routes pass install, enable, disable, catalog audit, and duplicate-install checks |
| Runtime health | Passed | Isolated memory-mode backend returned HTTP 200 from `http://127.0.0.1:18086/skoll/health` and was stopped after verification |
| Migration and seed impact | Passed | No migration or seed changes are required for the current in-memory implementation |

### Verification Commands

```text
go test ./internal/service/audit ./internal/service/pharmaoa -run 'TestAuditService|TestInventoryAlert' -count=1 -v
go test ./internal/service/audit ./internal/service/notification ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/bootstrap ./internal/plugin
go test ./...
go build -o tmp/skoll-f9-06.exe ./cmd/skoll
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
git diff --check
codegraph sync .
```

Result: Passed after retry.

### Next Step

- Claim `F9-07` from `docs/refactor/current/pharma_oa_work_items.md`.

## F9-07 Add Inventory End-To-End Smoke

- Date: 2026-07-13
- Executor: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Dependencies: F9-02, F9-03, F9-04, F9-05, and F9-06 are Done.

### Delivery

- Added a repeatable integration fixture that creates product, qualified supplier/customer, and two-warehouse master data before exercising the real pharma OA services.
- Added one closed-loop scenario covering purchase request approval, generated purchase order, batch inbound, sales order/outbound, stocktake approval, and paired cross-warehouse transfer.
- Added final balance and ordered immutable-ledger assertions: 20 units inbound, 6 outbound, 1 stocktake reduction, and 3 transferred leave 10 units in the source warehouse and 3 in the target warehouse.
- Added inventory alert assertions for two near-expiry balances and one low-stock balance, including notification-center ownership and actionable balance/batch target links.
- Added `scripts/smoke-pharma-inventory.ps1` as the dedicated acceptance command and documented it in the integration test guide.
- API/OpenAPI, permission, audit contract, migration, and seed impact: none; this work item validates existing F9 behavior without changing runtime contracts or persistent data.

### Retry Evidence

1. The first smoke build failed because the notification service's second constructor argument was incorrectly treated as an audit service; it is an ID generator. The fixture now uses the default notification ID generator while inventory alert audit remains wired through the alert service, and the same script passed on retry.

### Acceptance

| Check | Result | Evidence |
| --- | --- | --- |
| Purchase approval | Passed | A qualified supplier request for 20 units is approved into one purchase order with the expected request relation and total amount |
| Purchase inbound | Passed | The approved order creates a completed inbound record, batch ID, attachment metadata, balance, and inbound ledger reference |
| Sales outbound | Passed | A qualified customer order for six units completes outbound and decreases source inventory with an outbound ledger reference |
| Stocktake and transfer | Passed | Assigned approval posts one `-1` stocktake entry; moving three units creates distinct paired transfer entries |
| Balance accuracy | Passed | Final balances are source warehouse 10 and target warehouse 3, preserving the post-stocktake total of 13 |
| Ledger integrity | Passed | Five immutable entries appear in order: inbound, outbound, stocktake, transfer out, and transfer in |
| Inventory alerts | Passed | The scan creates two near-expiry alerts and one low-stock alert, each with a pending notification and actionable balance/batch link |
| Repeatability | Passed | The dedicated smoke command passed three consecutive runs with isolated in-memory fixture state |
| Repository tests | Passed | `go test ./...` completed successfully across backend, plugin, HTTP, and integration packages |
| Static and runtime quality | Passed | `go vet ./...`, binary build, and isolated memory-mode health check on port 18087 passed |
| Contract and data impact | Passed | No runtime API/OpenAPI, permission, audit schema, migration, or seed change is required for this test-only work item |

### Verification Commands

```text
.\scripts\smoke-pharma-inventory.ps1 -Count 3
go test ./...
go vet ./...
go build -o tmp/skoll-f9-07.exe ./cmd/skoll
Invoke-WebRequest http://127.0.0.1:18087/skoll/health
git diff --check
codegraph sync .
```

Result: Passed after retry.

### Next Step

- Claim `F10-01` from `docs/refactor/current/pharma_oa_work_items.md`.

## F10-01 Implement Announcements

- Date: 2026-07-13
- Executor: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Dependencies: F7 is Done.

### Delivery

- Added announcement and policy-document domain models, organization and role audiences, draft and published lifecycle, attachment validation for policies, and idempotent read confirmations.
- Added create, publish, audience-filtered list/detail, read-confirmation, and receipt-query services with create, publish, and read audit records.
- Added authenticated HTTP routes, least-privilege permissions, synchronized OpenAPI contracts, plugin routes, menu visibility, audit declarations, and lifecycle catalog assertions.
- Added the `/skoll/pharma-oa/announcements` console with loading, empty, error, no-permission, saving, publish-confirmation, receipt, and responsive states.
- Migration and seed impact: none; this milestone uses the current in-memory announcement service and existing plugin registration path.

### Retry Evidence

1. The first frontend validation was run from the repository root and failed because `package.json` lives under `web`. The command was rerun from `web`, and typecheck plus production build passed.
2. The first handler fixture passed a JWT claims value where middleware expects a pointer. The fixture was corrected to use `*JWTClaims`, and targeted plus full Go tests passed on retry.
3. Mobile browser acceptance found that the teleported Element Plus detail drawer bypassed its scoped responsive selector and that long mode labels clipped. The selector was made global, labels were shortened, and the 390px recheck passed without overflow or clipping.

### Acceptance

| Check | Result | Evidence |
| --- | --- | --- |
| Organization targeting | Passed | Published announcements are visible only when the caller organization intersects the configured organization audience |
| Role targeting | Passed | JWT role claims participate in audience filtering; matching roles can read while non-matching roles cannot |
| Policy attachments | Passed | Policy documents require at least one document reference before creation succeeds |
| Audience denial | Passed | Users outside all configured organization and role audiences cannot view or confirm the announcement |
| Read confirmation | Passed | Confirmation is idempotent, records the reader and timestamp, and is queryable through the receipt endpoint |
| Audit traceability | Passed | Create, publish, and read actions are present in the audit service fixture |
| API and OpenAPI | Passed | Five operations are wired with explicit request/response schemas and both OpenAPI files are byte-equivalent |
| Permissions and plugin lifecycle | Passed | Five permissions and five routes pass install, enable, disable, menu, audit-catalog, and duplicate-install assertions |
| Frontend states | Passed | The console covers loading, empty, error, no-permission, saving, destructive publish confirmation, detail, and receipt states |
| Responsive browser | Passed | Desktop and 390px browser checks show no document overflow, drawer overflow, overlap, or clipped mode labels |
| Runtime health | Passed | Memory-mode backend returned HTTP 200 from `http://127.0.0.1:8080/skoll/health` during browser acceptance |
| Migration and seed impact | Passed | No migration or seed update is required for the current in-memory milestone implementation |

### Verification Commands

```text
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/bootstrap ./internal/plugin -count=1
go test ./internal/plugin -run 'TestPharmaOA' -count=1
go test ./...
go vet ./...
cd web; npm run typecheck
cd web; npm run build
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
codegraph sync .
git diff --check
```

Result: Passed after retry.

### Next Step

- Claim `F10-02` from `docs/refactor/current/pharma_oa_work_items.md`.

## F10-02 Implement Contract Archive

- Date: 2026-07-13
- Executor: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Dependencies: F7, F8-04, and F8-05 are Done.

### Delivery

- Added supplier/customer contract records with immutable attachment metadata, effective and expiry dates, amount/currency, owner, approver, and lifecycle state.
- Added file-access validation before workflow creation, single-approver workflow instances, assignment-enforced approval/rejection, and idempotent terminal actions.
- Added expiry scans that create owner notifications with actionable contract links, persist reminder evidence, mark elapsed active contracts expired, and avoid duplicate notifications.
- Added six authenticated HTTP operations, five least-privilege permissions, synchronized OpenAPI schemas, plugin routes/audit declarations, and lifecycle catalog assertions.
- Added `/skoll/pharma-oa/contracts` with loading, empty, error, no-permission, saving, destructive confirmation, real file upload, detail, and responsive states.
- Migration and seed impact: none; this milestone uses the current in-memory contract service and existing file, workflow, notification, and plugin infrastructure.

### Retry Evidence

1. The initial service build referenced a non-existent `SubjectTypeUser` RBAC constant. It was corrected to `SubjectUser`, and service tests passed on retry.
2. The initial HTTP fixture had an unused import and fixed calendar dates that could expire. The import was removed, dates were made relative to the test clock, and handler tests passed on retry.
3. The first combined frontend validation exceeded the 120-second command limit during Vite output and ended with `EPIPE`. Typecheck and build were rerun independently with sufficient timeouts; both passed.
4. Final review found that request `actorId` values could override an authenticated JWT subject. Contract actions now prefer the JWT subject, a spoofed approver test is rejected, and targeted plus full Go validation passed on retry.

### Acceptance

| Check | Result | Evidence |
| --- | --- | --- |
| Party relation | Passed | Supplier and customer contracts resolve an existing master-data record and retain its ID, type, and display name |
| File attachment control | Passed | Creation requires at least one available file accessible to the authenticated owner; inaccessible files fail before workflow or contract persistence |
| Workflow relation | Passed | Each contract starts a linked single-approver workflow and stores its workflow instance ID |
| Approval security | Passed | Only the assigned workflow actor can approve or reject; authenticated JWT subject wins over spoofed request actor data |
| Terminal idempotence | Passed | Repeated approval or rejection returns the existing terminal contract without duplicating workflow actions or audit records |
| Expiry reminders | Passed | Active or elapsed contracts inside the scan window create one owner notification with a contract detail target; repeated scans create none |
| Audit traceability | Passed | Create, approve, reject, expiry-reminder, and expiry-scan actions are appended with contract/workflow/notification context |
| API and OpenAPI | Passed | Six operations use explicit request/response schemas and both OpenAPI files have identical SHA-256 hashes |
| Permissions and plugin lifecycle | Passed | Five permissions and six routes pass install, enable, disable, menu, audit-catalog, and duplicate-install assertions |
| Frontend states | Passed | The console covers loading, empty, error, no-permission, saving, approval/rejection/scan confirmation, upload, and detail states |
| Responsive browser | Passed | Desktop and 390x844 browser checks show no document, drawer, or button overflow; the contract drawer remains usable |
| Runtime health | Passed | The memory-mode backend returned HTTP 200 from `http://127.0.0.1:8080/skoll/health` during browser acceptance |
| Migration and seed impact | Passed | No migration or seed update is required for the current in-memory milestone implementation |

### Verification Commands

```text
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/bootstrap ./internal/plugin -count=1
go test ./internal/handler/http/v1/pharmaoa -run TestContractHTTPCreateApproveAndExpiryScan -count=1 -v
go test ./...
go vet ./...
go build -o tmp/skoll-f10-02-check.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
Get-FileHash docs/api/openapi.yaml; Get-FileHash internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
codegraph sync .
git diff --check
```

Result: Passed after retry.

### Next Step

- Claim `F10-03` from `docs/refactor/current/pharma_oa_work_items.md`.

## F10-03 Implement Qualification Management

- Date: 2026-07-13
- Executor: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Dependencies: F8-02, F8-04, and F8-05 are Done.

### Delivery

- Added one read model over employee certificates plus supplier and customer qualifications, with valid, expiring, expired, and permanent status calculation.
- Added configurable expiry scans that create deterministic, idempotent notifications with actionable links to the owning employee, supplier, or customer ledger.
- Added qualification-block audit events to critical purchase-request creation/approval and sales-order creation when a supplier or customer qualification is expired.
- Added authenticated list and expiry-scan HTTP operations, two least-privilege permissions, synchronized OpenAPI schemas, plugin routes/audit declarations, and lifecycle catalog assertions.
- Added `/skoll/pharma-oa/qualifications` with loading, empty, error, no-permission, scanning confirmation/progress, filtering, summary, subject navigation, and responsive states.
- Migration and seed impact: none; this milestone composes the existing employee, supplier, customer, notification, audit, and plugin services without introducing persistence schema or seed changes.

### Retry Evidence

1. The first binary acceptance build reached the Windows linker but returned only `link.exe: exit status 1`. The Work Item moved through `Failed -> Doing`; a clean output path with verbose linking succeeded on retry.
2. The first browser acceptance connected to a stale Vite process that had not loaded the new static route, leaving only the console shell and a route warning. The Work Item moved through `Failed -> Doing`; the workspace Vite process was restarted and desktop plus 375x844 checks passed on retry.
3. Final review found that the POST scan handler did not enforce the OpenAPI `1..365` day range and that two OpenAPI keys retained a literal patch prefix. The Work Item moved through `Failed -> Doing`; request validation, a POST boundary test, and both OpenAPI files were corrected before the full gate was rerun.

### Acceptance

| Check | Result | Evidence |
| --- | --- | --- |
| Unified ledger | Passed | Employee certificates and supplier/customer qualifications are normalized into one filterable record shape with subject identity and source-ledger target |
| Qualification status | Passed | Tests cover valid, expiring, expired, and permanent calculation against a fixed clock and configurable day window |
| Expiry reminders | Passed | Eligible expired/expiring records create recipient notifications with actionable subject-ledger links |
| Reminder idempotence | Passed | Deterministic reminder IDs make a repeated scan report zero new reminders while retaining prior evidence |
| Purchase blocking | Passed | Expired supplier qualification blocks purchase request creation and approval before critical state changes |
| Sales blocking | Passed | Expired customer qualification blocks sales order creation before stock or order state changes |
| Audit traceability | Passed | Qualification block, reminder creation, and scan summary actions retain actor, subject, operation, notification, and result context |
| API and OpenAPI | Passed | List and expiry-scan operations use explicit schemas and enforce the documented day range; both OpenAPI files have SHA-256 `97582748A8115A58B7283E716C651A2C598B532CB485233F78824F76C01B0A02` |
| Permissions and plugin lifecycle | Passed | Read and high-risk scan permissions plus two routes pass manifest, install, enable, disable, menu/catalog, audit-action, and duplicate-install assertions |
| Frontend states | Passed | The console covers loading, empty, error, no-permission, scanning confirmation/progress, filters, summary, reminder state, and subject navigation |
| Responsive browser | Passed | Desktop and 375x844 runtime checks show the empty ledger and controls without document overflow, overlap, or clipped text |
| Inventory regression | Passed | The Pharma OA inventory end-to-end smoke still passes after purchase and sales qualification audit changes |
| Runtime health | Passed | The rebuilt memory-mode backend returned HTTP 200 with `code=ok` from `http://127.0.0.1:8080/skoll/health` |
| Migration and seed impact | Passed | No migration or seed update is required because existing master-data qualification records are composed at service level |

### Verification Commands

```text
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa -run 'TestQualification' -count=1 -v
go test ./internal/plugin -run 'TestPharmaOA' -count=1 -v
.\scripts\smoke-pharma-inventory.ps1 -Count 1
go test ./...
go vet ./...
go build -x -o tmp/skoll-f10-03.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
Get-FileHash docs/api/openapi.yaml; Get-FileHash internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
codegraph sync .
git diff --check
```

Result: Passed after retry.

### Next Step

- Claim `F10-04` from `docs/refactor/current/pharma_oa_work_items.md`.

## F10-04 Implement Quality Complaints

- Date: 2026-07-13
- Executor: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
- Dependencies: F7, F8-03, and F8-05 are Done.

### Delivery

- Added quality complaint registration related to customer, active product, inventory batch, authenticated reporter, assigned handler, and accessible file evidence.
- Added a linked single-handler workflow with required resolution or rejection conclusions, terminal idempotence, and create/resolve/reject audit events.
- Added complaint list, detail, product-batch selector, create, resolve, and reject HTTP operations with four least-privilege permissions and synchronized OpenAPI contracts.
- Added plugin menu, permission, route, and audit declarations with install, enable, disable, duplicate-install, and catalog assertions.
- Added `/skoll/pharma-oa/quality-complaints` with loading, empty, error, no-permission, saving, destructive confirmation, upload, detail, filtering, and responsive states.
- Migration and seed impact: none; this milestone uses the current in-memory complaint service and existing workflow, file, product, customer, inventory, permission, and audit services.

### Retry Evidence

1. The first frontend type gate used the non-existent command `npm run type-check`. The Work Item moved through `Failed -> Doing`; the repository-defined `npm run typecheck` command was then used and passed together with the production build.
2. A final duplicate-number regression assertion exposed hidden coupling in a pre-existing audit assertion. The first retry stopped counting all actor records but still assumed event return order, so it failed again. The Work Item moved through `Failed -> Doing` for each attempt; the final assertion verifies exactly one resolve and one reject event without depending on unrelated audit volume or storage order.

### Acceptance

| Check | Result | Evidence |
| --- | --- | --- |
| Complaint registration | Passed | Registration retains the customer, active product, matching inventory batch, authenticated reporter, assigned handler, complaint content, and unique case number |
| Product batch relation | Passed | The batch selector filters inventory batches by product, and service validation rejects a batch belonging to another product |
| File attachment control | Passed | At least one available file accessible to the authenticated reporter is required; blank, denied, or unavailable files fail before persistence |
| Workflow relation | Passed | Every complaint publishes and starts a linked single-handler workflow and stores its workflow instance ID |
| Actor security | Passed | Only the assigned handler can complete the workflow, and authenticated JWT subject wins over spoofed request actor data |
| Conclusion and idempotence | Passed | Resolve and reject require a conclusion; repeated terminal actions return the existing result without duplicate workflow or audit changes |
| Audit traceability | Passed | Create, resolve, and reject actions retain complaint, workflow, actor, and conclusion context |
| API and OpenAPI | Passed | Six operations use explicit schemas; both OpenAPI files have SHA-256 `665B9060549070F53E646858ACD2EAA4AD34C10F7DCE05A23B8FD48A2BA00A7C` |
| Permissions and plugin lifecycle | Passed | Four complaint permissions and six routes pass install, enable, disable, menu/catalog, audit-action, and duplicate-install assertions; the plugin totals 68 permissions, 79 routes, and 70 unique audit actions |
| Frontend states | Passed | The console covers loading, empty, error, no-permission, saving, upload, detail, filtering, and resolve/reject confirmation states |
| Responsive browser | Passed | Desktop and 375x844 runtime checks show no document overflow, clipped text, drawer overlap, or inaccessible footer actions |
| Plugin regression | Passed | The Pharma OA plugin lifecycle smoke passes after complaint capabilities are declared |
| Inventory regression | Passed | The Pharma OA inventory end-to-end smoke passes after complaint batch lookup is introduced |
| Runtime health | Passed | The rebuilt memory-mode backend returned HTTP 200 with `code=ok` from `http://127.0.0.1:8080/skoll/health` |
| Migration and seed impact | Passed | No migration or seed update is required for the current in-memory milestone implementation |

### Verification Commands

```text
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa -run 'TestQualityComplaint' -count=1 -v
go test ./internal/plugin -run 'TestPharmaOA' -count=1 -v
.\scripts\smoke-pharma-oa-plugin.ps1 -Count 1
.\scripts\smoke-pharma-inventory.ps1 -Count 1
go test ./...
go vet ./...
go build -o tmp/skoll-f10-04-final.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
Get-FileHash docs/api/openapi.yaml; Get-FileHash internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
codegraph sync .
git diff --check
```

Result: Passed after retry.

### Next Step

- Claim `F10-05` from `docs/refactor/current/pharma_oa_work_items.md`.

## F10-05 Implement Drug Recall

- Date: 2026-07-13
- Executor: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Dependencies: F9-01 and F10-04 are Done.

### Delivery

- Added recall orders with immutable affected-customer tasks, product and batch evidence, optional source complaints, processing trace, terminal idempotence, and concurrency-safe recall-number uniqueness.
- Added exact product-and-batch tracing over completed sales outbounds, grouped customer scope, outbound document IDs, and recalled quantities.
- Added recall list, batch selector, scope preview, detail, create, and customer-task completion HTTP operations with three least-privilege permissions and synchronized OpenAPI contracts.
- Added create, customer-task completion, and recall completion audit events plus plugin install, enable, disable, menu, permission, route, audit catalog, and duplicate-install assertions.
- Added `/skoll/pharma-oa/drug-recalls` with loading, empty, error, no-permission, saving, destructive confirmation, batch scope preview, complaint relation, processing trace, and responsive states.
- Migration and seed impact: none; this milestone composes the existing in-memory sales outbound, inventory batch, product, customer, complaint, permission, audit, and plugin services.

### Retry Evidence

1. The first plugin lifecycle gate retained the pre-recall expected route and audit-action totals even though the manifest loaded the new declarations. The Work Item moved through `Failed -> Doing`; the expected catalog totals were updated to 85 routes and 76 unique audit actions, and the lifecycle suite passed on retry.
2. The first browser acceptance connected to a stale Vite process and did not resolve the new recall route. The Work Item moved through `Failed -> Doing`; after an initial background start failed to listen, the workspace Vite server was restarted successfully and desktop plus 375x844 browser checks passed.

### Acceptance

| Check | Result | Evidence |
| --- | --- | --- |
| Batch trace | Passed | Scope discovery matches both product and batch against immutable completed sales-outbound lines and retains outbound IDs as evidence |
| Affected customers | Passed | Matching outbound lines are grouped by customer with customer identity, outbound documents, and summed recalled quantity |
| Complaint relation | Passed | An optional source complaint must reference the same product and inventory batch as the recall |
| Recall task flow | Passed | Each affected customer receives one pending task; completion requires actor and note, is idempotent, and the final task automatically completes the recall |
| Recall-number concurrency | Passed | Case-insensitive uniqueness is protected across concurrent creates and an eight-worker regression test leaves exactly one recall |
| Actor security | Passed | Authenticated JWT subject overrides spoofed request actor data for create and task completion |
| Audit traceability | Passed | Create, task-complete, and automatic recall-complete actions retain recall, customer, task, actor, and result context |
| API and OpenAPI | Passed | Six operations use explicit schemas; both OpenAPI files have SHA-256 `C45F7BFEDF57D535F6BF80B0694DE782206D0AD8EA2B0208136C71ED3C9BD25E` |
| Permissions and plugin lifecycle | Passed | Three recall permissions and six routes pass install, enable, disable, menu/catalog, audit-action, and duplicate-install assertions; the plugin totals 71 permissions, 85 routes, 76 unique audit actions, and 156 resource events |
| Frontend states | Passed | The console covers loading, empty, error, no-permission, saving, destructive confirmation, filters, scope preview, detail, and task progress states |
| Responsive browser | Passed | Desktop and 375x844 runtime checks show no document overflow, clipped text, drawer overlap, or inaccessible footer actions |
| Plugin regression | Passed | The Pharma OA plugin lifecycle smoke passes after recall capabilities are declared |
| Inventory regression | Passed | The Pharma OA inventory end-to-end smoke passes after outbound trace composition is introduced |
| Runtime health | Passed | The final rebuilt memory-mode backend returned HTTP 200 with `code=ok` from `http://127.0.0.1:8080/skoll/health` |
| Migration and seed impact | Passed | No migration or seed update is required for the current in-memory milestone implementation |

### Verification Commands

```text
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa -run 'TestDrugRecall' -count=1 -v
go test ./internal/plugin -run 'TestPharmaOA' -count=1 -v
.\scripts\smoke-pharma-oa-plugin.ps1 -Count 1
.\scripts\smoke-pharma-inventory.ps1 -Count 1
go test ./...
go vet ./...
go build -o tmp/skoll-f10-05-final.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
Get-FileHash docs/api/openapi.yaml; Get-FileHash internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
codegraph sync .
git diff --check
```

Result: Passed after retry.

### Next Step

- Claim `F10-06` from `docs/refactor/current/pharma_oa_work_items.md`.

## F10-06 Cold-Chain Records

- Date: 2026-07-13
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: immutable temperature and humidity readings, stock-batch context, threshold evaluation, anomaly reminders, retryable scan jobs, permissions, audit declarations, OpenAPI contracts, and a responsive operator console.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Reading integrity | Passed | Records require an enabled warehouse, area, location, controlled temperature threshold, positive inventory balance, and an exact product/batch/location context; frozen thresholds and authenticated actor identity are retained with every immutable reading |
| Anomaly evaluation | Passed | Latest readings create, update, and resolve deterministic stock-balance anomalies for temperature and humidity deviations; duplicate scans reuse anomaly and notification identities |
| Job and retry behavior | Passed | Scan jobs expose pending/running/succeeded/failed states, retry count, logs, policy, actor, result counts, completion time, and an explicit failed-job retry path |
| Scale regression | Passed | The scan evaluates the full internal reading snapshot and the 501-position regression test proves that the public list limit cannot hide an anomaly |
| Notification trace | Passed | Active anomalies create notification-center reminders carrying recipient, risk, batch, location, reasons, and a deep link back to the cold-chain console; later normal readings resolve the same reminder |
| API and identity | Passed | Seven JWT-protected routes cover contexts, readings, anomalies, jobs, scan, and retry; authenticated JWT subjects override spoofed request actors |
| OpenAPI | Passed | Public and embedded OpenAPI files are byte-identical with SHA-256 `0622CC64CDB6381F369FEDB099E37E585ECE6771FF72A6063FB68F317372C8F0` and define all cold-chain request, response, filter, and job schemas |
| Permissions and audit | Passed | `pharma_oa.cold_chain.read`, `.create`, and `.run` are declared with API/button risk levels; record, anomaly create/update/resolve, run, and failure actions are emitted or declared for audit coverage |
| Plugin lifecycle | Passed | The Pharma OA manifest exposes 74 permissions, 92 routes, 83 unique audit actions, and 166 resource events; install, enable, disable, catalog, route, audit, and duplicate-install assertions pass |
| Frontend states | Passed | The console covers loading, empty, error, no-permission, saving, destructive confirmation, scan progress, failed retry, filtering, details, and disabled actions when no eligible stock or recipient exists |
| Responsive browser | Passed after retry | Real authenticated Chrome checks at 1366x900 and 375x844 show no application console errors, page errors, failed requests, document overflow, drawer clipping, or inaccessible actions; screenshots were reviewed manually |
| Plugin regression | Passed | `smoke-pharma-oa-plugin.ps1` passes after cold-chain capabilities are declared |
| Inventory regression | Passed | `smoke-pharma-inventory.ps1` passes after cold-chain stock-balance integration is introduced |
| Runtime health | Passed | The final memory-mode binary returned HTTP 200 with `code=ok` from `http://127.0.0.1:8080/skoll/health` |
| Migration and seed impact | Passed | No migration or seed update is required for the current in-memory milestone implementation |

### Retry Record

1. The first product assertion measured the drawer while its slide transition was still active. F10-06 moved to `Failed`, the check was changed to wait for the settled drawer geometry, and the task returned to `Doing`.
2. The next run exposed an unqualified browser 404. F10-06 moved to `Failed`, response URL collection was added, and the task returned to `Doing`.
3. The 404 was identified as Chrome's implicit `/favicon.ico` request, not an application resource. F10-06 moved to `Failed`, the exclusion was limited to that exact browser-default URL while all API/script/style errors remained fatal, and the full desktop/mobile suite passed after returning to `Doing`.

### Verification Commands

```text
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa -run 'TestColdChain' -count=1 -v
go test ./internal/service/pharmaoa -run 'TestColdChainScanDoesNotInheritPublicRecordLimit' -count=1 -v
go test ./internal/plugin -run 'TestPharmaOA' -count=1 -v
.\scripts\smoke-pharma-oa-plugin.ps1 -Count 1
.\scripts\smoke-pharma-inventory.ps1 -Count 1
go test ./...
go vet ./...
go build -o tmp/skoll-f10-06.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
node tmp/f10-06-browser.cjs
Get-FileHash docs/api/openapi.yaml, internal/handler/http/openapi.yaml
Invoke-RestMethod http://127.0.0.1:8080/skoll/health
codegraph sync .
codegraph status .
git diff --check
```

Result: Passed after retries.

### Next Step

- Claim `F10-07` from `docs/refactor/current/pharma_oa_work_items.md`.

## F10-07 Compliance Audit Dashboard

- Date: 2026-07-16
- Status flow: `Todo -> Doing -> (Failed -> Doing) x14 -> Review -> Blocked -> Review -> Done`
- Scope: active qualification, quality-complaint, drug-recall, and cold-chain risk aggregation; filters, trace links, CSV evidence export, permissions, audit events, OpenAPI, plugin declarations, and a responsive console.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Risk aggregation | Passed | Expired and expiring qualifications, pending complaints, active recalls, and active cold-chain anomalies are normalized without duplicating source state; terminal and valid records are excluded |
| Priority and filters | Passed | High risk sorts before medium risk, oldest observed or due items sort first, and keyword, source, risk, and bounded limit filters are independently tested |
| Actionability and trace | Passed | Every risk retains its source ID, deep link, subject, batch context, open-action counts, observed or due time, and source-specific evidence; the browser trace drawer and source navigation pass |
| CSV evidence | Passed | Authenticated export uses the same filters, UTF-8 BOM, deterministic fields, formula-injection protection, download state, and an auditable export event |
| Audit and actor identity | Passed | View and export events record normalized filters and counts; JWT subject is the only HTTP actor and spoofed actor query data is ignored |
| API and OpenAPI | Passed | Dashboard and export operations use explicit schemas; all local OpenAPI references resolve and public plus embedded contracts are byte-identical with SHA-256 `78314C6A156B56784E64F43F87DF112561796769F8FD762324ADF06C9A25E5A2` |
| Shared API metadata | Passed | `AuditMeta` now serializes as `createdAt` and `updatedAt`, is declared in OpenAPI, and has a regression test |
| Permissions and plugin lifecycle | Passed | Read and export permissions, two routes, and view/export audit actions pass install, enable, disable, catalog, route, audit, and duplicate-install assertions; the plugin totals 76 permissions, 94 routes, 85 unique audit actions, and 170 resource events |
| Frontend states | Passed after retries | Loading, empty, normalized error, read denial, export denial, exporting, filters, trace detail, and source navigation are covered; icon actions expose accessible names |
| Responsive browser | Passed after retries | Authenticated Chrome at 1366x900 and 375x844 has no document overflow, clipped controls, console errors, page errors, or unexpected HTTP failures; both screenshots were reviewed manually |
| Full quality gate | Passed | `go test ./...`, `go vet ./...`, backend build, frontend type check, frontend production build, focused review tests, plugin smoke, and inventory smoke all pass |
| Runtime health | Passed | The rebuilt backend and Vite app returned HTTP 200 before browser acceptance |
| Migration and seed impact | Passed | No migration or seed update is required because the dashboard is a read-only composition over current milestone services |

### Retry Record

1. Service-level retries corrected a CSV expectation and made audit assertions independent of storage order.
2. OpenAPI retries corrected the embedded variable name, detected inherited malformed cold-chain keys, added complete local-reference validation, and supplied the previously missing `AuditMeta` schema.
3. The first browser run coincided with Vite dependency pre-bundling and correctly kept aborted module requests fatal; the warmed server passed that gate on retry.
4. Browser review found unnamed icon actions. F10-07 returned to `Doing`, added `aria-label` values for refresh, clear, trace, and source actions, and passed type check plus build again.
5. Test-harness retries corrected duplicate trace-text selection, a transient MySQL connection abort, and parent-shell service expiry without hiding those failed runs.
6. Error-state acceptance was isolated to prevent Playwright route state interference and verified the normalized service-unavailable message from one intercepted dashboard request.
7. Permission acceptance intercepted `/auth/me` with a read-only profile so runtime hydration could not replace the intended restricted session; read/export separation and complete read denial then passed.

### Verification Commands

```text
go test ./...
go vet ./...
go build -o tmp/skoll-f10-07.exe ./cmd/skoll
go test ./internal/domain/shared ./internal/service/pharmaoa ./internal/handler/http ./internal/handler/http/v1/pharmaoa ./internal/plugin -run 'Test(ComplianceDashboard|AuditMeta|EmbeddedOpenAPI|OpenAPIContractFilesStayInSync|PharmaOA)' -count=1
go test ./internal/plugin -run TestPharmaOA -count=1 -v
.\scripts\smoke-pharma-oa-plugin.ps1 -Count 1
.\scripts\smoke-pharma-inventory.ps1 -Count 1
node node_modules/vue-tsc/bin/vue-tsc.js --noEmit
node node_modules/vite/bin/vite.js build
node tmp/f10-07-browser.cjs
node tmp/f10-07-error-diagnostic.cjs
Get-FileHash docs/api/openapi.yaml, internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
codegraph sync .
codegraph status .
git diff --check
```

Result: Passed after retries. The temporary `.git/index.lock` permission blocker was cleared before the required Work Item commit.

### Next Step

- Claim `F11-01` from `docs/refactor/current/pharma_oa_work_items.md`.

## F11-01 Customer Follow-Up

- Date: 2026-07-16
- Status flow: `Todo -> Doing -> Review -> (Failed -> Doing) x2 -> Review -> Done`
- Scope: customer visit planning, owned and authorized data scope, attachments, completion and cancellation, permissions, audit events, OpenAPI contracts, plugin declarations, and a responsive sales console.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Data scope | Passed | The service supports trusted own, organization, and all-customer scopes; HTTP derives owner identity exclusively from JWT subject and ignores spoofed actor or scope inputs |
| Follow-up lifecycle | Passed | Planned visits can be created and edited, then completed or cancelled through explicit state transitions with invalid transitions rejected |
| Attachment safety | Passed after retry | Real file upload, safe filename normalization, executable-extension rejection, attachment metadata, and empty-array serialization are covered by regression tests and browser acceptance |
| Audit and identity | Passed after retry | Create, list, update, complete, and cancel events retain authenticated actor identity; assertions validate the emitted action set independently of audit storage order |
| API and OpenAPI | Passed | Five JWT-protected operations and all request, response, filter, attachment, and transition schemas resolve; public and embedded contracts are byte-identical with SHA-256 `4BEF2844B59374710B7DBCC98B0DA3D29BCCE961769E0E2CDDB491CA06267A19` |
| Permissions and plugin lifecycle | Passed after retry | Read, create, update, complete, and cancel permissions pass install, enable, disable, catalog, route, audit, and duplicate-install assertions; the plugin exposes 81 permissions, 99 routes, 90 unique audit actions, and 180 resource events |
| Frontend states | Passed | The console covers loading, empty, normalized error, no-permission, saving, destructive confirmation, filters, visit planning, detail, attachments, completion, and cancellation |
| Responsive browser | Passed after retry | Authenticated Chrome checks cover desktop, drawer, and narrow viewport geometry with no application errors or horizontal overflow; the real UI create-to-complete flow passes |
| Full quality gate | Passed | Full Go tests, vet, backend build, focused service/HTTP/plugin/OpenAPI tests, frontend type check, frontend production build, CodeGraph sync, and diff checks pass |
| Runtime health | Passed | The rebuilt backend returned HTTP 200 with `code=ok` from `http://127.0.0.1:8080/skoll/health` |
| Migration and seed impact | Passed | No migration or seed update is required for the current in-memory milestone implementation |

### Retry Record

1. Plugin lifecycle assertions still expected the previous manifest totals. Counts were updated to 81 permissions, 99 routes, 90 audit actions, and 180 resource events, then the suite passed.
2. A transient login 500 occurred during the first browser run; the immediate retry returned 200 and continued through the acceptance flow.
3. The first audit assertion depended on storage order. It was replaced with a set-based action assertion and passed.
4. The first create-to-complete browser script waited on hidden Element Plus drawer DOM after the transition. It was changed to wait for visible drawer and row state, then passed.
5. Browser diagnostics exposed `attachments: null` for records without files. F11-01 moved to `Failed`, cloning was corrected to return `[]`, and a serialization regression test was added.
6. The first regression-test compile used the wrong local variable. F11-01 moved to `Failed` again, the assertion was corrected, and targeted plus full suites passed.

### Verification Commands

```text
go test ./...
go vet ./...
go build -o tmp/skoll-f11-01.exe ./cmd/skoll
go test ./internal/service/pharmaoa ./internal/handler/http ./internal/handler/http/v1/pharmaoa ./internal/plugin -run 'Test(CustomerFollowUp|EmbeddedOpenAPI|OpenAPIContractFilesStayInSync|PharmaOA)' -count=1
cd web; npm run typecheck
cd web; npm run build
python <F11-01 Selenium desktop/drawer/mobile and create-to-complete acceptance>
Get-FileHash docs/api/openapi.yaml, internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
codegraph sync .
codegraph status .
git diff --check
```

Result: Passed after retries.

### Next Step

- Claim `F11-02` from `docs/refactor/current/pharma_oa_work_items.md`.

## F11-02 Sales Opportunities

- Date: 2026-07-16
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: owned sales opportunities, customer and product snapshots, exact expected amounts, controlled stage transitions, funnel statistics, permissions, audit events, OpenAPI contracts, plugin declarations, and a responsive sales console.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Opportunity lifecycle | Passed | Opportunities advance only through `lead -> qualified -> proposal -> negotiation -> won`; stage skipping and terminal-state mutation are rejected, while loss is an explicit terminal transition with a required reason |
| Exact amount and statistics | Passed | Expected amounts use integer cents; statistics expose all six stages plus total, open, won, lost, and aggregate expected amount without floating-point loss |
| Customer and product relation | Passed | Creation validates an authorized sales customer and enabled products, then retains customer and product snapshots so later master-data changes do not corrupt opportunity history |
| Data scope and identity | Passed | Trusted service calls support own, organization, and all scopes; HTTP derives the owner exclusively from the JWT subject and ignores spoofed actor or scope inputs |
| Audit coverage | Passed | Create, list, statistics, update, and advance operations emit declared `pharma_oa.sales_opportunity.*` audit actions with authenticated actor identity |
| API and OpenAPI | Passed | Five JWT-protected operations and all request, response, filter, transition, timeline, and statistics schemas resolve; public and embedded contracts are byte-identical with SHA-256 `77022BD39ED3F7C7EC44056356A61091CC9E30A4D52033878D8F80E6C0DB87FE` |
| Permissions and plugin lifecycle | Passed after retry | Read, create, update, and advance permissions pass install, enable, disable, catalog, route, audit, and duplicate-install assertions; the plugin exposes 85 permissions, 104 routes, 95 unique audit actions, and 189 resource events |
| Frontend states | Passed after retry | The console covers loading, empty, normalized error, no-permission, saving, destructive loss confirmation, filters, create/edit, stage advance, statistics, detail, and timeline states |
| Real closed loop | Passed | Authenticated browser acceptance creates an opportunity from real customer/product prerequisites, advances it from Lead to Qualified, verifies the statistics delta and timeline, and confirms destructive cancellation leaves state unchanged |
| Responsive browser | Passed after retry | Selenium Chrome checks at 1366x900 and 500x844 show no application console errors, unexpected HTTP failures, horizontal overflow, clipped controls, or inaccessible drawer content; both screenshots were manually reviewed |
| Full quality gate | Passed | Full Go tests, vet, backend build, focused service/HTTP/plugin/OpenAPI tests, frontend type check, frontend production build, CodeGraph sync, and diff checks pass |
| Runtime health | Passed | The backend health endpoint and Vite opportunity route both returned HTTP 200 after the final build |
| Migration and seed impact | Passed | No migration or persistent seed update is required for the current in-memory milestone implementation; browser prerequisites are isolated acceptance setup data |

### Retry Record

1. The initial browser runner could not resolve a root Playwright installation. F11-02 moved to `Failed`, switched to the repository's available Selenium runtime, and returned to `Doing`.
2. The first Selenium diagnostic exposed an async table-render crash when stage data was temporarily undefined. F11-02 moved to `Failed`, made stage labels null-safe, added the UI fix, and returned to `Doing`.
3. Plugin assertions inherited older route indices and manifest totals. Route checks were updated to the declared paths and the resource-event expectation was corrected to permission plus route count.
4. Browser setup initially had no eligible customer or product, correctly disabling opportunity creation. The acceptance runner now creates protected API prerequisites when absent and remains idempotent through unique titles and baseline statistics.
5. Element Plus teleported and hidden transition DOM required visible-instance selectors for stage options, rows, and drawers; the checks were tightened without suppressing application errors.
6. Destructive-cancel acceptance found an uncaught confirmation rejection. Advance and loss prompts now handle cancellation explicitly, and the browser passes with a clean console.
7. No-permission acceptance initially used a super-admin session. It now uses a restricted user role and intercepts profile hydration so read denial is proven without issuing opportunity API requests.

### Verification Commands

```text
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa -run TestSalesOpportunity -count=1
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa -run TestSalesOpportunity -count=1
go test ./internal/handler/http -run 'Test(EmbeddedOpenAPI|OpenAPIContractFilesStayInSync)' -count=1
go test ./internal/plugin -run TestPharmaOA -count=1
go test ./internal/service/pharmaoa ./internal/handler/http ./internal/handler/http/v1/pharmaoa ./internal/plugin -run 'Test(SalesOpportunity|EmbeddedOpenAPI|OpenAPIContractFilesStayInSync|PharmaOA)' -count=1
go test ./...
go vet ./...
go build -o tmp/skoll-f11-02.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
python tmp/f11-02-browser.py
Get-FileHash docs/api/openapi.yaml, internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
codegraph sync .
codegraph status .
git diff --check
```

Result: Passed after retries.

### Next Step

- Claim `F11-03` from `docs/refactor/current/pharma_oa_work_items.md`.

## F11-03 Payment And Invoice Records

- Date: 2026-07-17
- Status flow: `Todo -> Doing -> Review -> (Failed -> Doing) x7 -> Review -> Done`
- Scope: sales-order payment plans, partial and complete receipts, invoice issue and void lifecycle, private financial attachments, overdue reminders, observable retryable scan jobs, permissions, audit events, OpenAPI contracts, plugin declarations, and a responsive finance console.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Sales-order relation and exact amounts | Passed | Payment plans and invoices retain order snapshots and integer-cent amounts; cumulative plans and active invoices cannot exceed the order total |
| Payment lifecycle | Passed | Positive receipts cannot exceed the remaining plan amount; partial and paid transitions are deterministic, and paid plans complete outstanding reminder notifications |
| Invoice lifecycle | Passed | Invoice numbers are unique, issue totals are bounded by the order, void requires a reason, and voiding releases the amount for later invoices |
| Attachment safety | Passed | Plans, receipts, and invoices use authenticated private uploads; metadata validation rejects traversal, invalid sizes, missing identity, and executable extensions |
| Data scope and identity | Passed | Trusted service calls support own and all scopes; HTTP derives actor identity exclusively from JWT subject and ignores spoofed actor or scope input |
| Overdue reminders | Passed | Due unpaid plans become overdue, notification IDs are deterministic per plan, deep links target the selected plan, repeated scans are idempotent, and payment completion closes the reminder |
| Job lifecycle and audit | Passed | Scan jobs expose pending, running, succeeded, and failed states with logs and retry count; only failed jobs retry, and payment, invoice, scan, retry, and failure actions are audited |
| API and OpenAPI | Passed | Nine JWT-protected operations and all payment, invoice, attachment, filter, receipt, void, job, and retry schemas resolve; public and embedded contracts are byte-identical with SHA-256 `2428AC8807E4A258270B60DD19D4FA35680CF6CB1FD0DDA62A7F8021F9D457A2` |
| Permissions and plugin lifecycle | Passed | Six finance permissions and nine routes pass install, enable, disable, catalog, route, audit, and duplicate-install assertions; the plugin exposes 91 permissions, 113 routes, 104 unique audit actions, and 204 resource events |
| Frontend states | Passed after retries | Loading, empty, normalized error, no-permission, saving, destructive invoice void, payment plan, receipt, invoice, reminder scan, job detail, and deep-link selection are covered |
| Responsive browser | Passed after retries | Authenticated Selenium Chrome acceptance at desktop and 500x844 mobile dimensions has no application errors, unexpected requests, horizontal overflow, clipped controls, or inaccessible drawer content; both screenshots were manually reviewed |
| Full quality gate | Passed | Full Go tests, vet, backend build, focused domain/service/HTTP/plugin/OpenAPI tests, frontend type check, frontend production build, CodeGraph sync, and diff checks pass |
| Runtime health | Passed | The rebuilt backend health endpoint and Vite finance route both returned HTTP 200 before final acceptance |
| Migration and seed impact | Passed | No migration or persistent seed update is required for the current in-memory milestone implementation; browser prerequisites are isolated acceptance setup data |

### Retry Record

1. A transient login 500 stopped the first browser attempt before product assertions; a clean runtime returned 200.
2. Browser setup used a spoofed sales-order actor, which correctly failed F11-03 ownership checks; setup now uses the authenticated actor.
3. Teleported hidden drawer inputs caused the receipt amount to target a stale field; selectors now operate on the visible drawer instance.
4. The scan drawer initially matched the toolbar action behind it; the submission assertion is scoped to the visible drawer.
5. Mobile detail initially selected an overdue plan sharing the same order; the row selector now requires the partial status.
6. The mobile reference assertion required a node whose entire text equaled the reference even though the UI intentionally renders `reference · actor`; it now validates contained visible text.
7. Accumulated in-memory acceptance data caused a later login 500; the final complete run used a clean backend process and passed all 13 states without suppressing diagnostics.

### Verification Commands

```text
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa -run TestPayment -count=1
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa -run TestPayment -count=1
go test ./internal/domain/pharmaoa ./internal/service/pharmaoa ./internal/handler/http ./internal/handler/http/v1/pharmaoa ./internal/plugin -run 'Test(Payment|EmbeddedOpenAPI|OpenAPIContractFilesStayInSync|PharmaOA)' -count=1
go test ./internal/plugin -run TestPharmaOA -count=1
go test ./...
go vet ./...
go build -o tmp/skoll-f11-03.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
python tmp/f11-03-browser.py
Get-FileHash docs/api/openapi.yaml, internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
codegraph sync .
codegraph status .
git diff --check
```

Result: Passed after retries.

### Next Step

- Claim `F11-04` from `docs/refactor/current/pharma_oa_work_items.md`.

## F11-04 Business Metrics API

- Date: 2026-07-17
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: bounded operational metrics query, stock alerts, qualification expiry, purchase approval efficiency, customer follow-up outcomes, exact cent-based sales trends, audit, permission, OpenAPI, plugin declarations, and typed frontend client.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Metric sources | Passed | The service composes the existing F9/F10/F11 sources without duplicating source state: active inventory alerts, active-subject qualifications, purchase requests, customer follow-ups, and sales orders |
| Query bounds | Passed | Paired inclusive `from`/`to` values accept RFC3339 or date input, windows over 366 days are rejected, qualification horizon is limited to 1-365 days, and buckets are restricted to day/week/month |
| Stock and qualification metrics | Passed | Active low-stock, over-stock, and near-expiry alerts are counted by type; expired and expiring qualifications exclude inactive employees, suppliers, and customers and retain subject breakdowns |
| Approval and follow-up metrics | Passed | Purchase approval totals, terminal rate, and average completion hours derive from immutable audit timestamps; follow-up totals, state counts, overdue plans, and completion rate respect the requested window |
| Sales trend | Passed | Sales orders aggregate with rounded integer cents into continuous day/week/month series, including empty buckets, total order count, and total amount |
| Performance and concurrency | Passed | Five independent sources load concurrently; a 10,000-order, 30-day sample completed in about 0.11 seconds under the Go race detector, with no race findings |
| Identity and audit | Passed | HTTP ignores an `actorId` query value and uses JWT subject; successful reads append `pharma_oa.business_metrics.read` evidence for the authenticated actor |
| API and OpenAPI | Passed | One JWT-protected GET operation and all query, window, metric, series, and envelope schemas resolve; public and embedded contracts are byte-identical with SHA-256 `5599BC92BD730FA7FAA08D86E9824E20E094E76E6BFC0E7D46E74E475C501C45` |
| Permissions and plugin lifecycle | Passed | The read permission, route, and audit action pass install, enable, disable, catalog, route, audit, and duplicate-install assertions; the plugin exposes 92 permissions, 114 routes, 105 unique audit actions, and 206 resource events |
| Frontend contract | Passed | TypeScript query and response types plus `getBusinessMetrics` align with the Go/OpenAPI contract; type check and production build pass without adding a premature Dashboard page |
| Full quality gate | Passed | Full Go tests, vet, backend build, focused race/service/HTTP/plugin/OpenAPI tests, frontend type check and build, CodeGraph sync, and diff checks pass |
| Runtime HTTP | Passed | Rebuilt backend returned health 200, authenticated metrics 200, over-limit query 400, and unauthenticated query 401 |
| Migration and seed impact | Passed | No migration or seed update is required because F11-04 is a read-only composition over current milestone services |

### Verification Commands

```text
go test -race ./internal/service/pharmaoa -run TestBusinessMetrics -count=1 -v
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/plugin -run 'Test(BusinessMetrics|EmbeddedOpenAPI|OpenAPIContractFilesStayInSync|PharmaOA)' -count=1
go test ./...
go vet ./...
go build -o tmp/skoll-f11-04.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
Get-FileHash docs/api/openapi.yaml, internal/handler/http/openapi.yaml
Invoke-WebRequest http://127.0.0.1:8080/skoll/health
Invoke-WebRequest -Headers <auth> http://127.0.0.1:8080/skoll/v1/pharma-oa/business-metrics
codegraph sync .
codegraph status .
git diff --check
```

Result: Passed.

### Next Step

- Claim `F11-05` from `docs/refactor/current/pharma_oa_work_items.md`.

## F11-05 Pharma OA Dashboard

- Date: 2026-07-17
- Status flow: `Todo -> Doing -> (Failed -> Doing) x2 -> Review -> Done`
- Scope: lazy-loaded Pharma OA dashboard route, bounded business-metrics filters, operational metric widgets, exact cent-based sales trend visualization, source deep links, permission gating, request single-flight, shared state handling, and responsive layout.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Dashboard metrics | Passed | Stock alerts, qualification risk, approval completion, customer follow-up completion, and sales trend render from one typed F11-04 snapshot |
| Filters and refresh | Passed | Inclusive date range, day/week/month bucket, and 1-365 day qualification horizon are sent only on explicit apply or refresh; a double trigger issues one request while loading |
| Performance | Passed | The route is lazy-loaded into a 10.98 KiB JavaScript chunk (3.87 KiB gzip); authenticated first render against the real backend completed in 1737 ms |
| Loading and data states | Passed | Delayed metrics response exposes the shared animated skeleton before all widgets and the accessible sales chart render |
| Empty and error states | Passed after retries | Zero activity renders the shared empty state without stale widgets; a 503 renders the shared normalized error title and a non-empty locale-aware description |
| Permission state | Passed | A session without `pharma_oa.business_metrics.read` is redirected by the route guard and issues no business-metrics request |
| Responsive browser | Passed | Selenium Chrome at 1366x900 and 500x844 has no console errors, horizontal overflow, clipped controls, or overlapping dashboard content; both screenshots were manually reviewed |
| Frontend quality gate | Passed | Vue TypeScript check, 3598-module production build, and diff whitespace validation pass |
| API, permission, audit, migration, and seed impact | Passed | F11-05 consumes the existing F11-04 API and read permission without changing backend contracts, audit behavior, migrations, or seed data |

### Retry Record

1. The first run expected the raw 503 message, but shared error normalization intentionally replaces server details; the assertion was aligned with the frontend contract.
2. The second run bound the normalized error description to English while the active locale differed; the final assertion verifies the locale-independent error title, non-empty description, and absence of stale widgets.

### Verification Commands

```text
cd web; npm run typecheck
cd web; npm run build
python tmp/f11-05-browser.py
git diff --check
```

Result: Passed after retries.

### Next Step

- Claim `F11-06` from `docs/refactor/current/pharma_oa_work_items.md`.

## F11-06 Report Export

- Date: 2026-07-17
- Status flow: `Todo -> Doing -> (Failed -> Doing) x2 -> Review -> Done`
- Scope: asynchronous report export jobs, exact cent-based CSV generation, private file storage, owner isolation, retry and failure evidence, permissions, audit, OpenAPI, plugin declarations, and typed frontend client.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Report coverage | Passed | Business metrics, sales trend, and operational risk report types produce deterministic CSV output from the bounded F11-04 metrics query |
| Amount precision | Passed | Monetary values remain integer cents through aggregation and are formatted exactly once for CSV output |
| Async lifecycle | Passed | Queue returns a pending job, background execution records running and terminal states, and each job retains ordered execution logs |
| File storage and ownership | Passed | Successful output is written through the file service as a private `pharma_oa` object with SHA-256 metadata; list, detail, retry, and download enforce authenticated ownership |
| Failure and retry | Passed | Failed jobs retain their error and may be retried; retrying a succeeded job returns conflict without starting duplicate work |
| Identity and HTTP states | Passed | Request actor fields are ignored in favor of JWT subject; unauthenticated access returns 401, invalid ranges return 400, missing/foreign jobs are isolated, and invalid retries return 409 |
| Audit chain | Passed | Queue, run, complete, fail, list, detail, retry, and download have dedicated `pharma_oa.report_export.*` actions with actor and job evidence |
| API and OpenAPI | Passed after retries | Five JWT-protected operations and all request, job, log, file, and envelope schemas resolve; public and embedded contracts are byte-identical with SHA-256 `4BF692D4A83916FF9EED7269487DDBA8DE208D1425CA82A6BBC7623AA761098C` |
| Permissions and plugin lifecycle | Passed | Read, create, retry, and download permissions plus five routes pass install, enable, disable, catalog, audit, and duplicate-install assertions; the plugin exposes 96 permissions, 119 routes, 110 unique audit actions, and 215 resource events |
| Frontend contract | Passed | Typed list, queue, detail, retry, and download clients align with the Go and OpenAPI contracts; frontend type check and production build pass |
| Full quality gate | Passed | Focused service/HTTP/race/plugin/OpenAPI tests, full Go tests, vet, backend build, frontend type check and build, runtime HTTP checks, CodeGraph sync, and diff checks pass |
| Runtime HTTP | Passed | A real queued job completed with 14 rows and 442 CSV bytes; authenticated list/detail/download and private object bytes matched, retry returned 409, invalid range returned 400, unauthenticated access returned 401, and the audit chain was present |
| Migration and seed impact | Passed | No migration or seed change is required because job state is process-local and result files use the existing file metadata and object storage contracts |

### Retry Record

1. The first OpenAPI validation found accidental leading `+` characters in the new path block; the contract was repaired and regenerated in both copies.
2. The second validation found the same patch artifact in the schema block; it was removed before focused and full contract validation passed.

### Verification Commands

```text
go test ./internal/service/pharmaoa -run TestReportExport -count=1
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa -run TestReportExport -count=1
go test -race ./internal/service/pharmaoa -run TestReportExport -count=1
go test ./internal/plugin -run TestPharmaOA -count=1
go test ./internal/handler/http -run 'Test(EmbeddedOpenAPI|OpenAPIContractFilesStayInSync)' -count=1
go test ./...
go vet ./...
go build -o tmp/skoll-f11-06.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
python tmp/f11-06-runtime.py
Get-FileHash docs/api/openapi.yaml, internal/handler/http/openapi.yaml
codegraph sync .
codegraph status .
git diff --check
```

Result: Passed after retries.

### Next Step

- Claim `F12-01` from `docs/refactor/current/pharma_oa_work_items.md`.

## F12-01 Demo Data Set

- Date: 2026-07-17
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: one-command Pharma OA demo initialization, employee qualification, drug master data, qualified supplier and customer, cold-chain warehouse, purchase approval workflow, inbound batch inventory, sales outbound, customer follow-up, status reporting, idempotency, permissions, audit, OpenAPI, and typed frontend client.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| One-command initialization | Passed | `POST /v1/pharma-oa/demo-seed/apply` creates the complete scenario through existing domain services and returns all entity references in one response |
| Master data | Passed | One active employee with a 20-day qualification, one temperature-controlled product, one qualified supplier, one qualified customer, and one cold-chain warehouse/area/location are queryable through their normal APIs |
| Workflow and purchase | Passed | A purchase request launches and completes a real approval workflow, generates a purchase order, and retains the workflow instance ID |
| Inventory chain | Passed | Purchase inbound creates one batch and inbound ledger entry; sales outbound consumes 12 of 100 units and creates the second immutable ledger entry, leaving 88 units in one stock balance |
| CRM and alert readiness | Passed | A planned customer follow-up is owned by the authenticated seed actor, and the employee certificate produces one 30-day qualification reminder |
| Idempotency and concurrency | Passed | A process-wide apply lock serializes execution; a second apply returns the same references with `reused=true` and does not duplicate stock ledger entries |
| Failure visibility | Passed | Status exposes ready, applying, applied, and failed states plus the terminal stage and error; failed initialization appends `pharma_oa.seed.fail` evidence |
| Identity, permission, and audit | Passed | HTTP derives the actor from JWT claims; read/apply permissions are registered, unauthenticated status returns 401, and read/apply audit actions are queryable for the actor |
| API and OpenAPI | Passed | Two JWT-protected operations and status/entity/count schemas resolve; public and embedded contracts are byte-identical with SHA-256 `ACF22A6C7867204895F098D4C73524B41ED5CA9161C00AAC18218451446516F0` |
| Plugin and frontend contract | Passed | Existing plugin seed routes and permissions pass lifecycle smoke; plugin docs describe the live host endpoints, and typed status/apply clients pass TypeScript and production build checks |
| Full quality gate | Passed | Focused service/HTTP/plugin/OpenAPI tests, race detector, all Go tests, vet, backend build, frontend type check and 3598-module production build, CodeGraph sync, and diff checks pass |
| Runtime HTTP | Passed after retry | New binary on port 8081 initialized the scenario, returned `reused=true` on repeat, exposed all references across 10 normal business list APIs, and returned both seed audit actions |
| Migration and persistent seed impact | Passed | No migration or SQL seed changes are required because current Pharma OA repositories are process-local; the implementation intentionally composes existing domain contracts without introducing a second persistence path |

### Retry Record

1. The first runtime verification called a nonexistent `/v1/pharma-oa/inventory/balances` endpoint after seed execution had succeeded. The Work Item moved `Doing -> Failed -> Doing`; the script was corrected to validate the returned inventory counts plus the 10 supported business list APIs, and the rerun passed.
2. Local policy rejected attempts to replace the existing port-8080 process or launch a detached process. The accepted rerun hosted the rebuilt binary in a managed foreground session on port 8081, leaving the user's current browser service untouched.

### Verification Commands

```text
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa -run TestDemoSeed -count=1
go test -race ./internal/service/pharmaoa -run TestDemoSeed -count=1
go test ./internal/plugin -run TestPharmaOA -count=1
go test ./internal/handler/http -run 'Test(EmbeddedOpenAPI|OpenAPIContractFilesStayInSync)' -count=1
go test ./...
go vet ./...
go build -o tmp/skoll-f12-01.exe ./cmd/skoll
cd web; npm run typecheck
cd web; npm run build
Invoke-RestMethod -Method Post http://127.0.0.1:8081/skoll/v1/pharma-oa/demo-seed/apply -Headers <auth>
Get-FileHash docs/api/openapi.yaml, internal/handler/http/openapi.yaml
codegraph sync .
codegraph status .
git diff --check
```

Result: Passed after retry.

### Next Step

- Claim `F12-02` from `docs/refactor/current/pharma_oa_work_items.md`.

## F12-02 End-To-End Acceptance Script

- Date: 2026-07-17
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: independent Pharma OA acceptance command, employee onboarding, purchase approval, purchase inbound, sales outbound, qualification alert, customer follow-up, workflow and inventory evidence, demo-seed idempotency, and Chinese/English command output.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Independent command | Passed | `scripts/smoke-pharma-oa-e2e.ps1` runs the dedicated integration acceptance test without requiring an externally hosted service |
| Employee onboarding | Passed | The seeded active employee and qualification certificate are present and referenced by the acceptance result |
| Purchase workflow | Passed | A purchase request is linked to a completed approval workflow and a generated purchase order |
| Purchase inbound | Passed | Approved purchase data produces the expected inbound batch, balance, and inventory ledger evidence |
| Sales outbound | Passed | The sales order consumes 12 units from the seeded batch and leaves the expected balance of 88 |
| Qualification alert | Passed | The 20-day employee qualification produces exactly one 30-day reminder |
| Customer follow-up | Passed | The planned customer follow-up is present and owned by the seeded actor |
| Inventory evidence and idempotency | Passed | Exactly two immutable ledger entries remain after the full chain, and a second seed apply reuses the scenario without duplication |
| Multilingual command output | Passed after retry | The script defaults to Chinese and accepts `-Locale en-US`; both modes pass under Windows PowerShell 5 |
| Full quality gate | Passed | Repeated smoke, English smoke, race detector, full Go tests, vet, PowerShell parser validation, formatting, CodeGraph status, and diff checks pass |
| Production impact | Passed | The Work Item adds acceptance automation only and reuses the F12-01 contracts; no API/OpenAPI, permission, audit, migration, seed, or frontend production changes are required |

### Retry Record

1. The first combined quality command exceeded 300 seconds after the acceptance test had passed. Verification was split into bounded commands, and every required gate passed independently.
2. The first multilingual script embedded raw Chinese in a UTF-8 file without a BOM, which Windows PowerShell 5 misparsed. The Work Item moved `Doing -> Failed -> Doing`; the script was made ASCII-safe and decodes UTF-8 messages at runtime, after which both Chinese and English modes passed.

### Verification Commands

```text
.\scripts\smoke-pharma-oa-e2e.ps1 -Count 2
.\scripts\smoke-pharma-oa-e2e.ps1 -Locale en-US
go test -race ./tests/integration -run '^TestPharmaOAEndToEndAcceptanceSmoke$' -count=1
go test ./...
go vet ./...
[System.Management.Automation.Language.Parser]::ParseFile((Resolve-Path 'scripts/smoke-pharma-oa-e2e.ps1'), [ref]$null, [ref]$null)
gofmt -d tests/integration/pharma_oa_acceptance_smoke_test.go
codegraph status .
git diff --check
```

Result: Passed after retries.

### Next Step

- Claim `F12-03` from `docs/refactor/current/pharma_oa_work_items.md`.

## F12-03 Industry Plugin README

- Date: 2026-07-17
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: default-Chinese and English industry-plugin guides, executable startup and demo-seed instructions, feature and runtime boundaries, extension workflow, troubleshooting, localized manifest metadata, and multilingual lifecycle smoke output.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Install and run guide | Passed | The default README starts from prerequisites, repository-root startup, automatic plugin discovery, login, concrete browser routes, authenticated demo seeding, and independent acceptance commands |
| Feature boundary | Passed | F8-F12 domains are grouped by capability and entry point; the guide distinguishes host APIs from manifest extension routes, integrated pages from the static lifecycle fixture, and sample capability from regulatory validation |
| Runtime honesty | Passed | Process-local repositories, restart behavior, private report storage, catalog disable semantics, and the absence of process hot-unload or legacy compatibility are explicit |
| Extension guide | Passed | Domain, service, HTTP, permission, audit, OpenAPI, manifest, typed frontend, i18n, UI-state, test, and migration impacts are sequenced for new developers |
| Multilingual docs | Passed | `README.md` is the default Chinese guide and `README.en.md` provides the same install, boundary, extension, verification, and troubleshooting structure in English |
| Multilingual manifest | Passed | Chinese plugin name, menu label, config title, and all config field labels now have real Chinese values while English fields remain unchanged; manifest tests prevent regression |
| Multilingual smoke | Passed | The lifecycle smoke defaults to `zh-CN`, supports `en-US`, remains ASCII-safe for Windows PowerShell 5, and passes in both modes |
| Documentation links | Passed | Every local Markdown link in both guides resolves from the owning file |
| Existing behavior | Passed | Plugin manifest/lifecycle tests, the Pharma OA end-to-end smoke, and all Go tests pass after the documentation and localization changes |
| API, permission, audit, migration, seed, and frontend impact | Passed | No contract, permission key, audit action, migration, seed behavior, frontend route, or frontend client changed; only existing manifest display metadata and verification output were localized |

### Verification Commands

```text
go test ./internal/plugin -run TestPharmaOA -count=1
.\scripts\smoke-pharma-oa-plugin.ps1
.\scripts\smoke-pharma-oa-plugin.ps1 -Locale en-US
.\scripts\smoke-pharma-oa-e2e.ps1
go test ./...
PowerShell local Markdown link check for plugins/pharma_oa/README.md and README.en.md
PowerShell parser check for scripts/smoke-pharma-oa-plugin.ps1
gofmt -d internal/plugin/pharma_oa_manifest_test.go
codegraph sync .
codegraph status .
git diff --check
```

Result: Passed.

### Next Step

- Claim `F12-04` from `docs/refactor/current/pharma_oa_work_items.md`.

## F12-04 Performance And Permission Validation

- Date: 2026-07-17
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: large-list pagination and common-query samples, customer data scope, authenticated actor identity, approval and inventory RBAC, plugin public-path boundary, OpenAPI synchronization, multilingual acceptance command, and residual risk inventory.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Large-list pagination | Passed | 600 employees and 600 customers verify default 50, maximum 200, stable offset, keyword/status filters, region filter, and owner scope |
| Performance sample | Passed | Representative query set completed in 3.5-6.1 ms; 1000-record fixed-iteration benchmarks are recorded in `pharma_oa_performance_permission_report.md` and `tests/benchmark/README.md` |
| Authentication boundary | Passed | `/v1/plugins` no longer exposes management or business API descendants; only the exact catalog route plus plugin `page` and `assets` remain public |
| Approval permission | Passed | Purchase, contract, and quality complaint approval/rejection routes enforce matching backend RBAC resources/actions on host and plugin-alias paths |
| Inventory permission | Passed | Purchase inbound, sales outbound, stocktake, and transfer routes enforce read/create/approve/reject RBAC as applicable |
| Permission denial and bypass | Passed | Missing tokens return `401`, ungranted non-super users return `403`, granted users pass, `super_admin` retains the documented bypass, and denial audit behavior remains covered |
| Customer data scope | Passed | Authenticated non-super users are forced to their JWT subject; query/body `ownerId`, `organizationId`, and `includeAll` cannot broaden scope; `super_admin` receives full scope |
| Actor identity | Passed | JWT subject takes precedence over request `actorId`, protecting audit and workflow actors from request spoofing |
| API contract | Passed | Both OpenAPI files have identical hashes, parse successfully, document offset/limit and `403`, and no longer advertise client-controlled customer read scope |
| Multilingual command | Passed | `scripts/smoke-pharma-oa-performance-permission.ps1` defaults to Chinese, supports `en-US`, is Windows PowerShell 5-safe, and passes in both locales |
| Quality gate | Passed | Focused race tests, `go test ./...`, `go vet ./...`, PowerShell parsing, OpenAPI tests, formatting, CodeGraph sync, and diff checks pass |
| Migration/seed/frontend impact | Passed | No migration, seed, permission key, frontend route, or frontend UI state changed; existing frontend keys continue to match RBAC resource/action pairs |

### Verification Commands

```text
.\scripts\smoke-pharma-oa-performance-permission.ps1
.\scripts\smoke-pharma-oa-performance-permission.ps1 -Locale en-US
go test -race ./internal/bootstrap ./internal/handler/http/v1/pharmaoa ./internal/service/pharmaoa -run 'Test(LoadAuthPolicy|AuthPolicy|PharmaOACritical|AuthGuardMiddlewareEnforces|ActorIDFromRequest|CustomerScopeFromRequest|NormalizePagination|PharmaOALargeList)' -count 1
go test ./...
go vet ./...
go test ./internal/handler/http -run 'Test(OpenAPI|RouterServes)' -count 1
[System.Management.Automation.Language.Parser]::ParseFile((Resolve-Path 'scripts/smoke-pharma-oa-performance-permission.ps1'), [ref]$null, [ref]$null)
Get-FileHash docs/api/openapi.yaml
Get-FileHash internal/handler/http/openapi.yaml
codegraph sync .
codegraph status .
git diff --check
```

### Retry Record

1. The first staged diff check found two trailing spaces in the report header. The Work Item moved `Review -> Failed -> Doing`; the whitespace was removed before repeating the staged diff and final status checks.

Result: Passed after retry.

### Next Step

- Claim the next `Todo` Work Item from `docs/refactor/current/pharma_oa_work_items.md` after rereading the mandatory current documents.
