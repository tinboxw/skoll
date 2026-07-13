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
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
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
