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
