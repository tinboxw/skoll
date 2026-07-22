# Skoll Plugin Runtime Acceptance Log

> Batch: `skoll-plugin-runtime-2026-07-21`
> Append evidence only after the Work Item reaches `Review` and passes independent acceptance.
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

## PR0-00 Establish Official Plugin Runtime Batch

- Date: 2026-07-21
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Create the official parent board, atomic Work Item table, acceptance log, and current document indexes.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Batch files | Pass | Board, Work Items, and acceptance log exist under `docs/refactor/current/` |
| Atomic structure | Pass | Every Work Item includes priority, Skill, description, dependencies, deliverables, acceptance, verification, and status |
| Milestones | Pass | PR1-PR5 cover runtime, platform services, generator, frontend experience, and independent-industry proof |
| Collaboration | Pass | Task claim, review, failure retry, acceptance, and one-commit rules are explicit |
| Project direction | Pass | The batch forbids compatibility layers, legacy formats, dual paths, and transition adapters |

### Verification Commands

Result: document links, Work Item field/count/status checks, archive boundary check, and `git diff --check` passed.

### Impact Review

- API/OpenAPI: no runtime contract changed.
- Permission/audit: planned as explicit Work Items.
- Migration/seed: planned as explicit Work Items.
- Frontend/i18n: planned as explicit Work Items.
- Documentation: current execution indexes now point to this batch.
- Compatibility: none; current contracts only.

### Commit

`PR0-00: establish plugin runtime batch`

## PR1-01 Execute Declared External Plugin Routes

- Date: 2026-07-21
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Replace placeholder route success with declared-route validation and real HTTP execution against `service_base_url`.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Real execution | Pass | Full Router test reaches an `httptest` plugin backend through the `/skoll` prefix and manifest-declared route |
| Request preservation | Pass | Method, base-path join, declared path, query, body, and ordinary request header assertions pass |
| Response preservation | Pass | Backend HTTP 201, response header, and raw JSON body reach the caller unchanged |
| Fail closed | Pass | Missing backend returns 503, unreachable backend returns 502, disabled plugin returns 503, and missing executor returns 502; no placeholder 200 remains |
| Declaration boundary | Pass | A method/path pair absent from `api.routes` is not executed |
| Contract documentation | Pass | Chinese-default and English plugin API references define `service_base_url`, forwarding behavior, and stable failure codes |
| Quality gates | Pass | Focused tests, race tests, vet, full Go suite, diff check, and CodeGraph sync pass |

### Verification Commands

```powershell
go test ./internal/bootstrap ./internal/handler/http/... ./internal/plugin/... -count=1
go test -race ./internal/bootstrap ./internal/handler/http ./internal/plugin/... -count=1
go vet ./internal/bootstrap ./internal/handler/http/... ./internal/plugin/...
go test ./... -count=1
codegraph sync .
git diff --check
```

Result: all commands passed; CodeGraph synchronized 4 changed Go files.

### Impact Review

- API/OpenAPI: no new path or schema; execution semantics for existing declared plugin routes changed from placeholder success to real backend response.
- Permission/audit: existing JWT, route permission, and audit middleware remain in front of the executor.
- Migration/seed: no impact.
- Frontend/i18n: no client change; stable Chinese-default runtime error messages added.
- Documentation: Chinese and English plugin API contracts updated.
- Compatibility: none; current manifest contracts only.

### Commit

`PR1-01: execute external plugin routes`

## PR1-02 Retry Record

| Attempt | Status | Failed Gate | Retry Action |
| --- | --- | --- | --- |
| 1 | Failed | Focused Go packages could not compile because `%TEMP%/go-build*` ran out of disk space | Clean Go build/test caches and verified temporary Go build directories, restore the same Work Item to `Doing`, then rerun the focused gate |
| 2 | Failed | The first D-drive cold-cache build exceeded the 180-second command timeout before producing a test result | Keep the warmed D-drive cache, split focused packages, extend the timeout, restore the same Work Item to `Doing`, and rerun |
| 3 | Failed | `TestInstallPreflightServicePassesWithCompleteImpactSummary` used a base URL without the now-required health URL | Update the fixture to the current paired service contract, add explicit incomplete-pair rejection coverage, restore `Doing`, and rerun |
| 4 | Failed | Existing external-route tests had no health checker, so the new fail-closed gate returned `health_checker_unavailable` | Inject an explicit healthy checker into route-isolation fixtures and add separate real unhealthy/readiness tests before rerunning |
| 5 | Failed | The new health E2E sent GET to a manifest route declared as POST and correctly received HTTP 405 before the health gate | Parameterize the test request method, use POST for business traffic and GET for diagnostics, restore `Doing`, and rerun |
| 6 | Failed | Combined CodeGraph, race, vet, and full-suite command exceeded 360 seconds without yielding per-stage exit codes | Split every final gate into a separate command with its own timeout and evidence, restore `Doing`, and rerun |

Retry status: Go caches were cleaned; subsequent gates use D-drive `GOCACHE` and `GOTMPDIR`; `PR1-02` returned to `Doing`.

## PR1-02 Define And Enforce Plugin Health Contracts

- Date: 2026-07-21
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Add current-format plugin health/readiness contracts, real probes, fail-closed traffic control, diagnostics, bounded caching, and synchronized API documentation.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Manifest contract | Pass | `service_base_url` and `service_health_url` must be declared together; incomplete pairs and invalid URLs fail validation |
| Probe policy | Pass | Shared HTTP checker uses unauthenticated GET, a two-second timeout, no redirects, 2xx success, stable failure codes, and bounded response draining |
| Traffic control | Pass | A 503 health response prevents the declared business handler from receiving any request; healthy recovery restores execution |
| Host readiness | Pass | `/ready` returns 503 with `ready=false` while an enabled service plugin is unhealthy and returns 200 after recovery |
| Plugin diagnostics | Pass | `/v1/plugins/{id}/health` reports one plugin with stable 200/404/503 behavior |
| Secret safety | Pass | Reports contain no service URL, query token, network error text, or user credential |
| Performance | Pass | Business requests reuse health results for at most five seconds; explicit diagnostics and readiness refresh them; probe-count assertion prevents per-request probing |
| API contract | Pass | Both OpenAPI copies define readiness/health responses and remain byte-synchronized with resolvable references |
| Quality gates | Pass | Focused tests, race, vet, full Go suite, diff check, and CodeGraph sync pass |

### Verification Commands

```powershell
$env:GOCACHE='D:\.cache\skoll-go'
$env:GOTMPDIR='D:\.tmp\skoll-go'
go test ./internal/plugin ./internal/bootstrap ./internal/handler/http ./internal/handler/http/v1/plugin -count=1
go test -race ./internal/plugin ./internal/bootstrap ./internal/handler/http ./internal/handler/http/v1/plugin -count=1
go vet ./internal/plugin ./internal/bootstrap ./internal/handler/http/...
go test ./... -count=1
codegraph sync .
git diff --check
```

Result: all final gates passed. Six failed attempts were recorded and resolved on the same Work Item before acceptance.

### Impact Review

- API/OpenAPI: `/ready` now exposes structured plugin readiness and may return 503; `GET /v1/plugins/{id}/health` is new; both OpenAPI copies are synchronized.
- Permission/audit: `/ready` remains the bounded public readiness endpoint; single-plugin diagnostics remain behind the existing authenticated plugin-management boundary; no mutation audit is required.
- Migration/seed: no impact.
- Frontend/i18n: no frontend client consumes the diagnostic yet; Chinese-default API messages and English developer reference are complete.
- Documentation: current Chinese and English plugin API contracts define paired service fields, probe, cache, traffic, diagnostics, and error behavior.
- Compatibility: none; incomplete service declarations are rejected under the current contract.

### Commit

`PR1-02: enforce plugin health readiness`

## PR1-03 Retry Record

| Attempt | Status | Failed Gate | Retry Action |
| --- | --- | --- | --- |
| 1 | Failed | The lifecycle smoke returned HTTP 404 for an installed plugin because the Router registered only routes that were enabled at host startup | Keep the same Work Item in `Doing`, replace startup-only route exposure with a stable dynamic dispatch boundary, and rerun the lifecycle and race gates |
| 2 | Failed | The focused packages did not compile because `pluginExtensionSnapshotProvider` is also the live OpenAPI aggregation contract | Restore the shared provider type only, keep dynamic route registration independent from it, and rerun the same focused gate |

Retry status: `PR1-03` returned to `Doing` after the static route-registration gap was identified.

## PR1-03 Synchronize Plugin Route Lifecycle

- Date: 2026-07-21
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Replace startup-only plugin business-route registration with a current-format dynamic dispatcher and synchronize route execution, lifecycle mutations, permission snapshots, extension snapshots, and health-cache invalidation.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Dynamic dispatch | Pass | One stable `/v1/plugins/{pluginId}/api/*` boundary delegates the actual plugin ID, method, and stripped manifest path on every request; the Router is not rebuilt |
| Enable visibility | Pass | An installed plugin returns 503, then the same Router returns the backend 200 immediately after enable |
| Atomic publication | Pass | Enable publishes runtime state, route permissions, extension snapshots, and cleared health state before releasing the lifecycle write gate |
| Concurrent disable | Pass | Disable waits for an in-flight proxied request to finish and prevents later requests from reaching the backend |
| Snapshot closure | Pass | Route permission and extension resolution both disappear when disable returns and reappear after re-enable |
| Health freshness | Pass | A fresh stale-unhealthy cache entry is discarded by enable, so historical health cannot delay route recovery |
| Current-only boundary | Pass | Dynamic routing is limited to `/v1/plugins/{pluginId}/api/*`; no legacy route or compatibility branch was added |
| Failure behavior | Pass | Unsupported methods and missing executors fail closed; disabled and undeclared routes cannot produce placeholder success |
| Quality gates | Pass | Focused tests, relevant package tests, full race tests, vet, full Go suite, CodeGraph sync, and diff check pass |

### Verification Commands

```powershell
$env:GOCACHE='D:\.cache\skoll-go'
$env:GOTMPDIR='D:\.tmp\skoll-go'
go test ./internal/plugin/... ./internal/handler/http/... ./internal/bootstrap -count=1
go test -race ./internal/plugin/... ./internal/handler/http/... ./internal/bootstrap -count=1
go vet ./internal/plugin/... ./internal/handler/http/... ./internal/bootstrap
go test ./... -count=1
codegraph sync .
git diff --check
```

Result: all final gates passed. Two failed attempts were recorded and resolved on the same Work Item before acceptance.

### Impact Review

- API/OpenAPI: no schema change; current plugin business routes now use one stable dynamic dispatch boundary and runtime OpenAPI aggregation remains live.
- Permission/audit: lifecycle publication and permission snapshots are synchronized; existing JWT, permission, and route-audit middleware remain unchanged.
- Migration/seed: no impact.
- Frontend/i18n: no frontend API change; Chinese default runtime messages remain stable and the bilingual developer contract documents lifecycle behavior.
- Performance/concurrency: lifecycle writes drain in-flight plugin requests; normal requests share a read gate and health cache is cleared only on lifecycle mutations.
- Compatibility: none; only the current `/v1/plugins/{pluginId}/api/*` contract is routed dynamically.

### Commit

`PR1-03: synchronize plugin route lifecycle`

## PR1-04 Supervise Plugin Service Lifecycle

- Date: 2026-07-21
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Add current service-launcher contracts, deterministic supervision, an external HTTP-service adapter, lifecycle audit events, bounded graceful/forced shutdown, plugin-manager integration, and host cleanup.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Launcher contract | Pass | `ServiceLauncher` returns only after readiness; `ServiceHandle` owns completion, graceful stop, and mandatory force-stop behavior |
| State model | Pass | `not_applicable`, `starting`, `ready`, `stopping`, `stopped`, and `failed` transitions expose stable codes and UTC timestamps |
| External adapter | Pass | Initial health must pass before Ready; periodic health failure is detected as a crash without leaking endpoint or network-error details |
| Enable integration | Pass | External service readiness completes before plugin state, route permissions, and extension snapshots are published |
| Disable/uninstall | Pass | Business traffic closes first, then the service handle stops; lifecycle failures cannot reopen routes |
| Metadata reload | Pass | Enabled services stop and restart around current metadata; reload/readiness failure rolls the plugin back to Disabled |
| Timeout policy | Pass | Start and stop use five-second defaults; graceful-stop timeout is audited and followed by mandatory force release |
| Crash handling | Pass | Unexpected handle completion and health-monitor failure enter Failed and emit stable audited transitions |
| Host shutdown | Pass | Runner closes plugin runtime before the event bus and returns cleanup errors; supervisor deterministically stops every owned handle |
| No-service plugins | Pass | Plugins without `service_base_url` are `not_applicable` and create no service handle |
| Quality gates | Pass | Focused tests, full relevant package tests, race tests, vet, full Go suite, CodeGraph sync, and diff check pass |

### Verification Commands

```powershell
$env:GOCACHE='D:\.cache\skoll-go'
$env:GOTMPDIR='D:\.tmp\skoll-go'
go test ./internal/plugin/... ./internal/bootstrap -count=1
go test -race ./internal/plugin/... ./internal/bootstrap -count=1
go vet ./internal/plugin/... ./internal/bootstrap
go test ./... -count=1
codegraph sync .
git diff --check
```

Result: all gates passed on the first acceptance cycle.

### Impact Review

- API/OpenAPI: no HTTP schema change; service lifecycle is an internal runtime contract.
- Permission/audit: lifecycle transitions write stable `plugin_service.<state>` audit actions with codes only; route authorization remains unchanged.
- Migration/seed: no impact.
- Frontend/i18n: no client change; Chinese and English developer contracts and deployment guidance describe the runtime behavior.
- Runtime/deployment: plugin enable now requires readiness; Runner cleanup owns supervised handles and surfaces cleanup failures.
- Compatibility: none; only current paired service declarations participate in supervision.

### Commit

`PR1-04: supervise plugin services`

## PR1-05 Retry Record 1

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: focused migration and lifecycle test batch
- Evidence: `go test ./internal/plugin/... ./internal/store/sql/gormrepo ./internal/store ./internal/bootstrap ./internal/handler/cli` exceeded the 120-second command limit before producing a package result.
- Retry action: split the batch by package, identify a compile or test deadlock independently, correct the affected boundary, and rerun the complete acceptance matrix.

## PR1-05 Execute Plugin Migrations Transactionally

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Replace file-only migration state with real SQL execution, a persistent migration ledger, lifecycle gates, checksum validation, MySQL DDL compensation, and explicit uninstall data policy.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Migration contract | Pass | Planner requires ordered `{version}_{name}.up.sql` and `.down.sql` pairs under the manifest directory; empty, missing, duplicate, or orphaned versions fail closed |
| Persistent ledger | Pass | `sk_plugin_migrations` records plugin ID, version, name, SHA-256, and applied time with a unique plugin/version constraint; GORM and MySQL/PostgreSQL schema assets are registered |
| Atomic apply | Pass | Pending SQL and ledger records share one transaction on SQLite/PostgreSQL; MySQL additionally prevalidates down SQL and compensates attempted DDL in reverse order after apply failure |
| Idempotency and drift | Pass | Repeated apply executes no SQL; modifying an already applied up file produces a checksum error and blocks lifecycle progress |
| Enable ordering | Pass | Target and dependency migrations run in topological order before service start, state publication, route permissions, or extension snapshots |
| Upgrade behavior | Pass | Metadata reload applies only pending migrations; successful schema upgrade keeps the plugin enabled, while failed upgrade rolls back the pending batch and leaves the plugin Disabled |
| Failure isolation | Pass | Failed fresh migration leaves the plugin Installed, creates no ledger row, leaks no SQLite schema, and never starts or exposes the plugin |
| Uninstall policy | Pass | `retain` and `archive` preserve data and ledger; `drop` executes every down migration in reverse order and removes ledger rows before metadata deletion |
| File safety | Pass | Dev Portal refuses to remove plugin files when uninstall or destructive rollback fails, preserving down scripts for retry |
| Persistence | Pass | Plugin migration version and data manifest survive insert and update through the SQL plugin repository |
| Database matrix | Pass with environment gate | SQLite apply, rollback, failure atomicity, lifecycle, and policy tests ran. The same MySQL/PostgreSQL success and failure-cleanup tests compile and are enabled by `SKOLL_TEST_MYSQL_DSN` / `SKOLL_TEST_POSTGRES_DSN`; local ports 13306/15432 were unavailable, so live execution is explicitly not claimed |
| Current-only boundary | Pass | The local `.skoll/migration-state.json` path and fake apply/rollback behavior were removed; CLI apply/rollback cannot bypass the configured lifecycle store |
| Quality gates | Pass | Focused tests, full relevant packages, race tests, vet, full Go suite, CodeGraph sync, gofmt, and diff checks pass |

### Verification Commands

```powershell
$env:GOCACHE='D:\.cache\skoll-go-pr105'
$env:GOTMPDIR='D:\.tmp\skoll-go-pr105'
go test ./internal/plugin/... ./internal/store/sql/gormrepo ./internal/store ./internal/bootstrap ./internal/handler/cli -count=1
go test -race ./internal/plugin/... ./internal/store/sql/gormrepo ./internal/bootstrap ./internal/handler/cli ./internal/handler/http/v1/plugin -count=1
go vet ./internal/plugin/... ./internal/store/... ./internal/bootstrap ./internal/handler/cli ./internal/handler/http/v1/plugin
go test ./... -count=1
codegraph sync .
git diff --check
```

Result: all locally executable gates passed. The first combined test command exceeded its time limit while SQLite CGO was cold-compiling in a shared cache; Retry 1 isolated the cache, completed compilation, and passed every gate. Live MySQL/PostgreSQL tests remain explicit environment-gated checks.

### Impact Review

- API/OpenAPI: no HTTP schema change; existing lifecycle endpoints now surface migration failures instead of publishing an enabled plugin.
- Permission/audit: migration start/success/failure is recorded through the plugin migration audit sink; permission publication remains after successful migration.
- Migration/seed: adds the `sk_plugin_migrations` model and MySQL/PostgreSQL `20260722_000023` schema assets; no seed change.
- Frontend/i18n: no frontend contract or locale change.
- Runtime/deployment: data-bearing plugins require a configured SQL migration store; memory mode does not emulate SQL or create a second state path.
- Compatibility: none; the previous file state and non-executing CLI behavior are removed rather than bridged.

### Commit

`PR1-05: execute plugin migrations transactionally`

## PR1-06 Retry Record 1

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: focused event, plugin, and bootstrap test batch
- Evidence: `go test ./internal/event ./internal/plugin/... ./internal/bootstrap -count=1` exceeded the 120-second command limit before producing a package result.
- Retry action: split the event, plugin, and bootstrap packages, preserve the isolated PR1-06 cache, fix any package-specific failure, and rerun the complete acceptance matrix.

## PR1-06 Retry Record 2

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: focused bootstrap package tests
- Evidence: `go test ./internal/bootstrap -count=1` exceeded 180 seconds after the plugin package had already passed from the same warm cache.
- Retry action: rerun bootstrap with Go's short test timeout to capture blocked goroutine stacks, correct lifecycle lock ordering, then rerun the package and complete matrix.

## PR1-06 Retry Record 3

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: combined event, plugin, and bootstrap race test batch
- Evidence: `go test -race ./internal/event ./internal/plugin/... ./internal/bootstrap -count=1 -timeout 120s` exceeded the 300-second command limit without a package result during the first isolated race build.
- Retry action: split each package into an independent race gate using the warmed PR1-06 cache, then rerun all non-race and repository-wide gates.

## PR1-06 Deliver Host Events To Enabled Plugin Handlers

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Connect manifest event subscriptions to the host business event bus, add stable delivery identity and retry policies, deliver a current HTTP envelope to external plugin services, and synchronize subscriptions with plugin lifecycle state.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Event identity | Pass | Every business event requires a stable non-empty ID; delivery IDs are deterministic for event ID, plugin ID, and handler |
| Declared subscribers | Pass | The runtime registers only manifest subscriptions and rechecks enabled state plus the current declaration immediately before delivery |
| Lifecycle registration | Pass | Enable registers target and newly enabled dependency subscriptions; disable, uninstall, reload, and host close remove them under the lifecycle gate |
| In-flight behavior | Pass | Event handlers take the lifecycle read gate, so disable/reload waits for active delivery and stale snapshots cannot deliver after state changes |
| HTTP contract | Pass | External services receive `POST /_skoll/events` with a typed envelope, stable `Idempotency-Key`, plugin ID, and handler headers |
| Transport safety | Pass | Delivery accepts only HTTP(S), rejects URL credentials/query/fragment, never follows redirects, bounds response draining, and does not expose response bodies |
| Host idempotency | Pass | Initial and concurrent duplicate publication atomically claims one delivery; successful, failed, running, and dead-letter records suppress duplicate side effects |
| Retry policies | Pass | `none`, `standard`, and `aggressive` map to 1/3/5 maximum attempts and 0/1s/250ms retry delays; each failure record retains its policy |
| Failure observability | Pass | Failed attempts, retry success, attempt count, last error, next run, and terminal dead-letter state remain queryable from the runtime |
| Retry behavior | Pass | Due work is atomically claimed, retries keep the same delivery identity, successful retry closes the record, and missing handlers dead-letter visibly |
| Developer contract | Pass | Chinese-default and English documents define the sole current endpoint, envelope, headers, idempotency, lifecycle, and retry semantics |
| Current-only boundary | Pass | No legacy endpoint, envelope, compatibility flag, or dual delivery path was added |
| Quality gates | Pass | Focused tests, lifecycle tests, split race tests, vet, full Go suite, CodeGraph sync, gofmt, and diff checks pass |

### Verification Commands

```powershell
$env:GOCACHE='D:\.cache\skoll-go-pr106'
$env:GOTMPDIR='D:\.tmp\skoll-go-pr106'
go test ./internal/event ./internal/plugin/... ./internal/bootstrap -count=1 -timeout 90s
go test -race ./internal/event -count=1 -timeout 90s
go test -race ./internal/plugin/... -count=1 -timeout 120s
go test -race ./internal/bootstrap -count=1 -timeout 120s
go vet ./internal/event ./internal/plugin/... ./internal/bootstrap
go test ./... -count=1
codegraph sync .
git diff --check
```

Result: all gates passed after three recorded retries. The first combined package command and first combined race command exceeded their execution limits during cold builds; split warm-cache gates produced explicit package results. The bootstrap retry exposed and corrected eager event-bus validation in manually constructed managers before the complete matrix passed.

### Impact Review

- API/OpenAPI: no public admin endpoint changed; external plugin services gain the sole current `POST /_skoll/events` host callback contract.
- Permission/audit: event registration follows plugin enabled state and current manifest declarations; delivery failures are represented by runtime delivery/retry records without response-body leakage.
- Migration/seed: no database migration or seed impact; durable job persistence remains outside PR1-06.
- Frontend/i18n: no frontend runtime change; the developer contract is available in default Chinese and English.
- Runtime/concurrency: lifecycle writes drain in-flight delivery; delivery and retry claims are atomic in the current runtime store; host shutdown removes all subscriptions before service shutdown.
- Compatibility: none; only the current event endpoint and envelope are implemented.

### Commit

`PR1-06: deliver host events to plugins`

## PR1-07 Retry Record 1

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: focused plugin, CLI, HTTP, RBAC, and config tests
- Evidence: deletion of the compatibility parser removed `normalizeSemver`, while `internal/plugin/local_marketplace.go` still referenced that helper for release sorting and keys; packages depending on `internal/plugin` did not compile.
- Retry action: use the current marketplace-specific version normalizer at both call sites, format only existing Go files, and rerun the focused acceptance batch.

## PR1-07 Retry Record 2

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: focused plugin contract tests
- Evidence: strict top-level manifest validation rejected the current `vendor_id` signature field, and removing the manifest field correctly invalidated the demo plugin's recorded `plugin.yaml` digest.
- Retry action: declare and parse the complete current signature/vendor field set, align the developer guide with the flat manifest contract, refresh the demo manifest digest, and rerun the focused acceptance batch.

## PR1-07 Retry Record 3

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: combined full Go, frontend typecheck, schema, and portal syntax batch
- Evidence: the parallel command exceeded its 300-second outer limit before returning per-command results, so no child result was accepted as evidence.
- Retry action: execute frontend/schema checks independently, then rerun the warmed full Go suite with an explicit Go test timeout and a larger outer command limit.

## PR1-07 Remove Plugin Compatibility Mode And Legacy Format Code

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Remove plugin/core version compatibility constraints, aliases, compatibility fixtures, and project-level transition fallbacks; publish one strict current manifest and marketplace contract.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Runtime model | Pass | `CompatibilitySkoll`, `ValidateCompatibility`, the constraint parser, and core-version preflight input were deleted |
| Manifest parser | Pass | Only canonical snake-case fields are parsed; dotted/camel aliases are gone and unknown top-level fields fail closed |
| Current schema | Pass | Manifest schema is strict and declares bilingual names, vendor/signature, data, API, and event contracts; every current plugin manifest top-level field is covered |
| Scaffolding and portal | Pass | CLI, HTTP scaffold, generator template, and developer portal emit/edit the same current manifest without a compatibility field |
| Marketplace | Pass | Index releases no longer carry core-version or deprecation fields; version identity preserves distinct prereleases while normalizing only an optional leading `v` |
| Examples and signatures | Pass | Current plugin manifests and fixtures were updated; the demo manifest signature digest matches the new content |
| Project transition paths | Pass | Old API-prefix environment fallback, tab storage fallback, default-home compatibility writer, and abbreviated data-scope aliases were removed |
| Documentation | Pass | Developer guide, API/event contracts, marketplace reference, architecture docs, OpenAPI enum, and plugin READMEs describe current contracts only |
| Repository scan | Pass | Removed compatibility symbols and fields have no hit in runtime, tests, plugins, frontend, active schemas, or developer docs |
| Backend quality | Pass | Focused packages, `go vet ./...`, and `go test ./... -count=1 -timeout 300s` pass |
| Frontend quality | Pass | i18n, accessibility, large-list checks, Vue typecheck, portal syntax check, and production build pass |
| CodeGraph | Pass | Index synced; manifest load/preflight/scaffold/package callers still converge on one `MetadataLoader.Load` path |

### Verification Commands

```powershell
go test ./internal/plugin ./internal/handler/cli ./internal/handler/http/v1/plugin ./internal/domain/rbac ./internal/service/rbac ./pkg/config
go test ./... -count=1 -timeout 300s
go vet ./...
cd web; npm run typecheck; npm run build
node --check plugins/developer-portal/static/app.js
Get-Content docs/schemas/plugin-manifest.schema.json -Raw | ConvertFrom-Json
Get-Content docs/schemas/plugin-marketplace-index.schema.json -Raw | ConvertFrom-Json
codegraph sync .
codegraph status .
git diff --check
```

Result: all acceptance gates passed after three recorded retries. The production build retained only the existing third-party Rollup annotation and Sass API deprecation warnings.

### Impact Review

- API/OpenAPI: public endpoint paths are unchanged in PR1-07; data-scope values are now only `self`, `department`, `department_tree`, `custom`, and `all`.
- Plugin contract: `plugin.yaml` is the sole manifest format; strict schema/parser behavior rejects undeclared fields instead of interpreting aliases.
- Marketplace: release identity is plugin ID plus plugin version; core-version constraints and deprecation flags are not part of the current index.
- Configuration/frontend state: only `SKOLL_API_BASE_PREFIX`, the current tabs-state key, and typed default-home targets remain.
- Migration/seed: no database migration is required; built-in RBAC seeds now use canonical data-scope constants.
- Compatibility: none; no bridge, fallback, dual parser, or transition write path remains in the implemented surfaces.

### Commit

`PR1-07: remove plugin compatibility code`

## PR1-08 Retry Record 1

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: plugin, bootstrap, and HTTP package compile gate
- Evidence: the new Pharma OA backend factory declared `notification.Service` as a value, but five Pharma OA constructors require `*notification.Service` or notifier interfaces implemented by its pointer receiver.
- Retry action: make the plugin factory dependency pointer-accurate, rerun the same compile gate, and continue only after it passes.

## PR1-08 Retry Record 2

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: focused Pharma OA manifest and lifecycle tests
- Evidence: the implementation correctly changed Pharma OA to a monolith plugin and parameterized API paths, while legacy test assertions still required `frontend_only` and flat update/action paths.
- Retry action: replace fixed legacy route assertions with current monolith placement checks and an exhaustive manifest-to-handler route contract gate, then rerun the focused tests.

## PR1-08 Retry Record 3

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: bootstrap package tests
- Evidence: the new in-process lifecycle fixture omitted its route permission, while two middleware tests still expected the deleted host-owned Pharma OA permission table and one resolver fixture declared a flat approval path.
- Retry action: complete the fixture contract and assert the current boundary: host permission mapping ignores Pharma OA paths, and the manifest resolver authorizes parameterized plugin routes.

## PR1-08 Move Pharma OA Behind The Public Plugin Boundary

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Move Pharma OA construction, HTTP execution, frontend routes, permissions, API clients, OpenAPI, and lifecycle ownership behind the current plugin manifest boundary.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Core ownership | Pass | Bootstrap and the host router no longer construct Pharma OA services, carry Pharma OA dependencies, register business routes, or own its permission table |
| In-process runtime | Pass | A thread-safe lazy backend factory serves only enabled plugins and releases its handler on disable, uninstall, reload, and shutdown |
| Repository ownership | Pass | Pharma OA repositories are exposed as one lazy plugin-owned aggregate; normal core startup does not construct them |
| API namespace | Pass | All 119 Pharma OA handler routes, manifest routes, OpenAPI paths, clients, scripts, and integration fixtures use `/v1/plugins/pharma_oa/api/...` |
| Contract consistency | Pass | Exhaustive tests prove exact manifest-to-handler equality and require every manifest method/path in both OpenAPI documents |
| Route parameters | Pass | Manifest permission resolution supports named path segments and rejects conflicting parameterized route shapes |
| Backend lifecycle | Pass | Enabled requests lazily create the real Pharma OA backend; disabled requests cannot execute it; re-enable creates a fresh handler instance |
| Frontend lifecycle | Pass | Pharma OA pages are absent from the static router and mount only from an enabled backend plugin record; disable, uninstall, and sync failure remove routes and menu state |
| Boundary smoke | Pass | The real backend serves the plugin namespace and returns 404 for the removed host business namespace |
| Documentation | Pass | Plugin runtime and developer guides describe one manifest namespace, in-process/external execution, and dynamic integrated routes without a compatibility path |
| Backend quality | Pass | Focused packages, all Go packages, `go vet`, lifecycle smoke, E2E smoke, and performance/permission smoke pass |
| Frontend quality | Pass | i18n, accessibility, large-list checks, Vue typecheck, and production build pass |
| Repository scan | Pass | The removed host API namespace appears only in explicit negative tests; no core Pharma OA service construction or static business route remains |
| CodeGraph | Pass | The refreshed index resolves plugin requests through `HandlePluginRoute` and `RegisterInProcessBackend`; impact review covers bootstrap, router, runtime, store, manifest, clients, and frontend routing |

### Verification Commands

```powershell
go test ./internal/plugin ./internal/plugin/pharmaoa ./internal/bootstrap ./internal/store ./internal/handler/http ./internal/handler/http/v1/pharmaoa ./tests/integration -count=1
go test ./... -count=1 -timeout 300s
go vet ./...
cd web; npm run typecheck; npm run build
powershell -ExecutionPolicy Bypass -File scripts/smoke-pharma-oa-plugin.ps1 -Locale en-US
powershell -ExecutionPolicy Bypass -File scripts/smoke-pharma-oa-e2e.ps1 -Locale en-US
powershell -ExecutionPolicy Bypass -File scripts/smoke-pharma-oa-performance-permission.ps1 -Locale en-US
rg -n "/v1/pharma-oa|/v1/pharma_oa|registerPharma|Pharma.*Service" internal web/src plugins/pharma_oa docs/development scripts tests --glob '!docs/refactor/old/**'
codegraph sync .
codegraph status .
git diff --check
```

Result: all acceptance gates passed after three recorded retries. The frontend build retained only existing third-party Rollup annotation and Sass legacy API warnings. The performance smoke measured the 600-row common query at 44.3532 ms and passed its pagination, scope, permission, and benchmark gates.

### Impact Review

- API/OpenAPI: Pharma OA has one current namespace, `/v1/plugins/pharma_oa/api/...`; the removed `/v1/pharma-oa/...` host namespace has no runtime alias.
- Permission/audit: authorization resolves from the enabled plugin's manifest route, including parameterized paths; audit actions remain attached to the same manifest contract.
- Plugin lifecycle: first API access creates the in-process backend; disable, uninstall, reload, and shutdown release the instance and prevent further execution.
- Migration/data: migration execution remains plugin lifecycle owned; repositories are lazy and persistent data is retained across handler recreation.
- Frontend/i18n: integrated routes and menu records follow backend plugin state; existing locale, accessibility, and large-list baselines pass.
- Runtime/concurrency: factory creation and reset are serialized per plugin, while lifecycle locking prevents a disable operation from racing an admitted route execution.
- Compatibility: none; no bridge, fallback, duplicate route, or static host registration remains.

### Commit

`PR1-08: move Pharma OA behind plugin boundary`

## PR1-09 Retry Record 1

- Date: 2026-07-22
- Status: Failed -> Doing
- Failed gate: real plugin runtime milestone E2E
- Evidence: external service startup passed the installed plugin state to the real HTTP health checker, which correctly returned `plugin_not_enabled`; the enable endpoint failed with `service_start_failed` before the runtime could enter enabled state.
- Retry action: probe startup readiness with the pending enabled state while preserving the committed lifecycle transition order, then rerun the same install/migrate/start/auth/route/audit/event/disable/uninstall test.

## PR1-09 Close Real Plugin Runtime Milestone

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Close the first plugin-runtime milestone with one real external plugin fixture that crosses metadata, migration, service supervision, HTTP health, JWT, RBAC, route dispatch, audit, event delivery, disable, stop, and destructive uninstall boundaries.

### Lifecycle E2E

| Step | Result | Evidence |
| --- | --- | --- |
| Install | Pass | Current manifest loads into `installed` state without creating runtime data tables |
| Migrate | Pass | Enable applies the SQLite migration and creates the plugin-owned table transactionally |
| Start | Pass | External service launcher uses the real HTTP health checker and reaches `ready` |
| Enable | Pass | HTTP lifecycle endpoint enables the plugin, publishes route permissions, registers events, and records lifecycle audit |
| Authenticate | Pass | Missing and invalid JWT requests return 401 before the plugin backend is called |
| Authorize | Pass | Denied users, undeclared routes, and undeclared methods return 403 without backend execution; authorized and super-admin requests pass |
| Execute | Pass | Query parameters reach the external service and its exact JSON response returns through the runtime proxy; call counts prove no placeholder response |
| Audit | Pass | Successful enable/route/disable/uninstall actions and the denied permission attempt exist in the audit-event test store |
| Event | Pass | One declared event reaches `/_skoll/events` with the current envelope and headers; duplicate event ID is idempotent |
| Disable | Pass | HTTP lifecycle endpoint removes route permissions and event subscriptions immediately |
| Stop | Pass | Disable stops the supervised service and records `stopped` state |
| Uninstall | Pass | HTTP lifecycle endpoint runs the `drop` policy, removes the table and ledger, and leaves runtime state `uninstalled` |

### Security Matrix

| Scenario | Expected | Result |
| --- | --- | --- |
| No bearer token | 401; backend calls unchanged | Pass |
| Invalid bearer token | 401; backend calls unchanged | Pass |
| Valid user without route permission | 403; security denial audited; backend calls unchanged | Pass |
| Valid user with route permission | 200; exact external response; plugin-route audit recorded | Pass |
| Super admin on declared route | 200; manifest declaration still required | Pass |
| Super admin on undeclared path | 403; backend calls unchanged | Pass |
| Super admin using undeclared method | 403; backend calls unchanged | Pass |
| Authorized user after disable | 403; backend calls unchanged | Pass |

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Real runtime fixture | Pass | `TestPluginRuntimeMilestoneEndToEnd` uses `httptest.Server`, real HTTP health/event clients, reverse proxy, JWT parser, permission checker, audit store, and SQLite migration store |
| Startup ordering | Pass | Service readiness probes use the pending enabled state while the durable runtime state changes only after startup succeeds |
| Runtime execution | Pass | Declared enabled routes execute the external backend; missing backend, undeclared, unauthorized, denied, disabled, and uninstalled states fail closed |
| Concurrency | Pass | Bootstrap, all plugin packages, and the full HTTP stack pass the race detector, including lifecycle and event tests |
| Repository scan | Pass | Runtime execution paths contain no placeholder or mock-success response; remaining hits are config-schema placeholder fields and future generator scaffold text outside runtime execution |
| Backend quality | Pass | Focused runtime packages, all Go packages, and `go vet ./...` pass |
| CodeGraph | Pass | Impact analysis for `pluginManagerWithExtensions.Enable` covers bootstrap assembly and every direct lifecycle test; the refreshed index is current |
| Milestone dependencies | Pass | PR1-01 through PR1-08 are Done and their acceptance records remain linked in this batch log |

### Verification Commands

```powershell
go test ./internal/bootstrap -run TestPluginRuntimeMilestoneEndToEnd -count=1 -v
go test ./internal/bootstrap ./internal/plugin/... ./internal/handler/http/... -count=1 -timeout 300s
go test -race ./internal/bootstrap ./internal/plugin/... ./internal/handler/http/... -count=1 -timeout 300s
go test ./... -count=1 -timeout 300s
go vet ./...
rg -n -i "placeholder|mock success|not implemented|todo" internal/plugin internal/bootstrap/di.go internal/handler/http/router.go internal/handler/http/v1/plugin --glob '*.go'
codegraph impact pluginManagerWithExtensions.Enable
codegraph sync .
codegraph status .
git diff --check
```

Result: all gates passed after one recorded retry. The race batch completed successfully across Bootstrap, every plugin package, and the HTTP stack. The E2E retry exposed and fixed a real startup-order bug that fixed health-check fixtures did not detect.

### Impact Review

- Lifecycle: external startup now probes readiness as the pending enabled state; the manager still commits enabled state only after service startup and keeps the existing stop-on-failure compensation.
- API/security: no endpoint or permission contract changed; the new matrix proves current JWT, RBAC, manifest, and disabled-state behavior.
- Audit/events: no schema changed; tests verify current lifecycle, route, denial, and event envelope contracts together.
- Migration/data: no production migration added; the isolated fixture proves apply and `drop` uninstall against SQLite.
- Frontend: no frontend source changed; PR1-08 already passed typecheck, build, i18n, accessibility, and large-list gates for the same runtime boundary.
- Compatibility: none; no bridge, fallback, dual path, legacy fixture, or placeholder business response was added.

### Commit

`PR1-09: close plugin runtime milestone`

## PR2-01 Replace In-Memory Workflow Repository with SQL Persistence

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Persist workflow definitions and instances as queryable SQL relations, make aggregate writes and reads transactional, and inject the configured repository into the runtime.

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Relational schema | Pass | Seven GORM models and matching MySQL/PostgreSQL migrations cover definitions, nodes, node assignees, transitions, instances, tasks, and action history |
| Aggregate transaction | Pass | Definition and instance saves update the parent and replace ordered children in one transaction; duplicate task insertion proves parent and children roll back together |
| Restart recovery | Pass | A file-backed SQL restart restores definition version/status, node order, assignees, transitions, instance state, tasks, decisions, actor names, comments, and completion timestamps |
| Runtime injection | Pass | Memory mode provides the explicit memory repository; MySQL/PostgreSQL adapters provide `WorkflowStore`; Bootstrap consumes only `bundle.Workflow` |
| Error contract | Pass | Missing definitions and instances preserve the current service-facing not-found messages |
| Dialect contract | Pass | Both migration scripts contain every relation, foreign-key boundary, uniqueness rule, and required query index; aggregate JSON fallback is forbidden by test |
| Concurrency safety | Pass | Workflow service, store, GORM repository, and Bootstrap pass the race detector |
| Repository quality | Pass | Focused packages, all Go packages, and `go vet ./...` pass |

### Verification Commands

```powershell
go test ./internal/service/workflow ./internal/store/... ./internal/bootstrap -count=1
go test -race ./internal/service/workflow ./internal/store ./internal/store/sql/gormrepo ./internal/bootstrap -count=1 -timeout 300s
go test ./... -count=1 -timeout 300s
go vet ./...
codegraph sync .
codegraph status .
git diff --check
```

Result: all locally executable acceptance gates passed without retry. Live MySQL/PostgreSQL DSNs were not present; their migration contracts are statically verified, while restart and rollback behavior is executed against a file-backed SQL database.

### Impact Review

- Workflow data: definitions, instances, tasks, actors, decisions, and complete action history now survive process and database connection restarts in SQL modes.
- Transactionality: aggregate replacement and aggregate reconstruction execute inside database transactions; partial child writes cannot become visible as successful state.
- Runtime: Bootstrap no longer creates a private in-memory workflow repository and follows the configured store mode.
- Migrations: version `000024` adds the same current relational model for MySQL and PostgreSQL with explicit indexes and foreign keys.
- Frontend/API: no request or response contract changed.
- Compatibility: none; no legacy table reader, dual write, fallback, or migration bridge was added.

### Commit

`PR2-01: persist workflow aggregates in SQL`

## PR2-02 Prove Workflow Concurrency, Idempotency, and Recovery

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Make workflow decisions atomic across repository implementations, deduplicate semantic retries, serialize competing transitions, and prove deterministic recovery after a committed response interruption.

### Concurrency And Recovery Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Duplicate terminal decision | Pass | 32 concurrent approvals in memory mode all return success and persist one approval action; the test passes 50 repeated runs |
| Duplicate non-terminal action | Pass | 12 concurrent SQL copy actions to the same target persist one copied task and one action |
| Independent non-terminal actions | Pass | Concurrent copies to distinct targets at the same timestamp both persist with unique stable action IDs |
| Competing terminal transitions | Pass | Concurrent SQL approve/reject calls serialize; exactly one commits, one receives a domain conflict, and no database lock error leaks |
| Committed response interruption | Pass | A repository fixture commits approval and drops the response; after closing and reopening the database, retry returns the original comment, timestamp, task completion, and single action |
| Failed candidate write | Pass | A duplicate-task constraint failure rolls back parent, tasks, and timeline; no candidate state becomes visible |
| Transient database contention | Pass | SQLite lock, MySQL deadlock/lock wait, and PostgreSQL serialization signatures are retryable; constraint violations are not |
| Race safety | Pass | Domain, service, SQL repository, and workflow HTTP handler pass the race detector |

### Verification Commands

```powershell
go test ./internal/service/workflow -run TestWorkflowServiceDeduplicatesConcurrentDecision -count=50
go test ./internal/store/sql/gormrepo -run "TestWorkflowStoreSerializesCompetingDecisions|TestWorkflowStoreDeduplicatesConcurrentNonTerminalActions|TestWorkflowDecisionRecoversAfterCommittedResponseInterruption|TestWorkflowStoreFailedAtomicUpdateLeavesCommittedState|TestRetryableDBErrorIncludesConcurrencyFailures" -count=20
go test -race ./internal/domain/workflow ./internal/service/workflow ./internal/store/sql/gormrepo ./internal/handler/http/v1/workflow -count=1 -timeout 300s
go test ./... -count=1 -timeout 300s
go vet ./...
codegraph impact UpdateInstance
codegraph status .
git diff --check
```

Result: all acceptance gates passed without retry. The repeated concurrency suites remained deterministic, the race detector found no shared-memory violation, and the full repository test and vet gates passed.

### Impact Review

- Repository contract: all workflow decisions use `UpdateInstance`; memory mode locks the aggregate and SQL mode locks the parent row while reconstructing and replacing children in one transaction.
- Idempotency: approval, rejection, withdrawal, transfer, and copy match persisted action type, task, actor, and target; a retry returns the first committed aggregate without changing its evidence.
- Action identity: stable SHA-256-derived IDs distinguish independent actions even when timestamps are equal and reinforce the same semantic boundary used by the service.
- Recovery: only database state is authoritative; a lost response does not require process memory to determine whether the decision committed.
- API/schema: no wire contract or database column changed.
- Compatibility: none; no old read path, dual transition path, fallback mutation, or migration bridge was added.

### Commit

`PR2-02: serialize workflow decisions`

## PR2-03 Retry Record 1

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: notification migration contract parity
- Evidence: the PostgreSQL `000025` migration used an inline `REFERENCES` clause, while the cross-dialect acceptance test requires an explicit `FOREIGN KEY` declaration; restart, inbox state, reminder idempotency, delivery retry, concurrent deduplication, memory repository, and bundle tests passed.
- Retry action: express the PostgreSQL delivery-to-item relation as a named foreign-key constraint and rerun the same focused acceptance command.

## PR2-03 Persist Notification Inbox And Delivery State

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Persist notification inbox items, completion state, reminder rules, and channel-delivery attempts with restart-safe idempotency and retry semantics.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Inbox restart recovery | Pass | File-backed SQL reopen preserves notification content, ownership, due time, and completion state |
| Reminder restart idempotency | Pass | Stable reminder IDs prevent duplicate inbox rows across repeated scans and repository restart |
| Delivery duplicate suppression | Pass | Notification, channel, and idempotency key form a unique delivery boundary; 12 concurrent attempts persist one row |
| Failed-channel retry | Pass | A failed attempt remains retryable with a new key and increments the attempt number after restart |
| Successful-channel suppression | Pass | Once a channel succeeds, later keys return the committed success without another delivery row |
| Repository parity | Pass | Memory and SQL repositories implement the same explicit notification contract |
| Runtime wiring | Pass | Bootstrap receives the configured notification repository from memory, MySQL, or PostgreSQL bundles |
| Migration parity | Pass after retry | MySQL and PostgreSQL `000025` migrations expose equivalent tables, indexes, uniqueness, and explicit foreign keys |

### Verification Commands

```powershell
go test ./internal/service/notification ./internal/store/sql/gormrepo ./internal/store ./internal/bootstrap -run "TestNotification|TestAllModels|TestNewBundle" -count=1 -v
go test ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./internal/plugin/pharmaoa ./tests/integration -count=1
go test ./internal/store/sql/gormrepo -run "TestNotificationStoreDeduplicatesConcurrentDeliveryKey|TestNotificationStorePersistsInboxRulesAndDeliveryAcrossRestart" -count=20
go test -race ./internal/service/notification ./internal/store/sql/gormrepo ./internal/service/pharmaoa ./internal/bootstrap -count=1 -timeout 300s
go test ./... -count=1 -timeout 300s
go vet ./...
codegraph sync .
codegraph status .
git diff --check
```

Result: all locally executable acceptance gates passed after one documented migration-contract retry. Live MySQL and PostgreSQL DSNs were not available; dialect migrations are statically verified, and persistence, restart, concurrency, and rollback behavior execute against file-backed SQL.

### Impact Review

- Notification state: inbox items, completion state, reminder rules, and delivery attempts survive process and database connection restarts in SQL modes.
- Idempotency: explicit notification IDs deduplicate creation; deterministic reminder IDs deduplicate scans; delivery keys deduplicate concurrent channel attempts.
- Retry: failed channels remain retryable with a fresh delivery key, while the first success closes the channel-delivery boundary.
- Runtime: bootstrap no longer creates private notification memory and follows the configured store bundle.
- Migrations: version `000025` adds the same current relational model for MySQL and PostgreSQL.
- API/frontend: no wire contract or user interface changed.
- Compatibility: none; no legacy constructor, old read path, dual write, fallback, or migration bridge was added.

### Commit

`PR2-03: persist notification delivery state`

## PR2-04 Retry Record 1

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: full repository test suite
- Evidence: `TestPluginRuntimeMilestoneEndToEnd` intermittently retained only three of four expected plugin audit events; 20 focused repetitions reproduced two failures, while all job persistence, restart, lease, retry, dead-letter, soak, and race tests passed.
- Root cause: middleware audit IDs used only `UnixNano()`. On Windows, separate route and lifecycle events can receive the same timestamp ID, and the audit repository correctly treats the second write as the same identity, replacing evidence.
- Retry action: use one middleware audit ID generator with a process-wide atomic sequence, cover fixed-timestamp concurrent uniqueness, rerun the milestone test repeatedly, and then rerun all PR2-04 and repository gates.

## PR2-04 Retry Record 2

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused audit-ID regression build
- Evidence: replacing the plugin lifecycle ID generator left an unused `shared` import in `internal/handler/http/v1/plugin/handler.go`; compilation stopped before behavioral tests.
- Retry action: remove the obsolete import, format the affected packages, and rerun the exact focused regression command before broader gates.

## PR2-04 Retry Record 3

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused audit-ID regression build
- Evidence: the next package compilation found the same obsolete `shared` import in `internal/bootstrap/middleware.go`; no behavioral test executed.
- Retry action: remove the bootstrap import, run a one-count compile/test gate first, and only then run the 30-count timing regression.

## PR2-04 Persist Scheduled Jobs, Retries, And Dead Letters

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Add one durable host job model for scheduled work, atomic worker leases, bounded retries, terminal dead letters, restart recovery, and queryable execution evidence.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Scheduled job restart recovery | Pass | File-backed SQL reopen restores payload, schedule, attempts, status, and active lease ownership |
| Atomic leasing | Pass | 16 SQL workers and 24 memory workers racing for one job produce exactly one lease |
| Lease fencing | Pass | Only the current unexpired lease token can complete or fail a task; an expired worker cannot commit |
| Worker interruption recovery | Pass | An expired lease is reclaimed with a new token and incremented attempt count after restart |
| Retry scheduling | Pass | Non-terminal failures move to `retry_wait` with an explicit next-run time |
| Retry exhaustion | Pass | Final failure or final-attempt lease expiry moves the job to `dead_letter` with error and timestamp evidence |
| Idempotent scheduling | Pass | Namespace and idempotency key return the persisted job even when a retry supplies a new request ID; null keys allow independent jobs |
| Successful completion | Pass | Result JSON and completion time persist, and the consumed lease cannot commit twice |
| Repository/runtime wiring | Pass | Memory, MySQL, and PostgreSQL bundles expose the same repository; bootstrap constructs the host job service from the configured bundle |
| Migration parity | Pass | MySQL and PostgreSQL `000026` migrations contain equivalent job, lease, retry, idempotency, and dead-letter columns and indexes |
| Audit evidence uniqueness | Pass after retries | Route, lifecycle, request, error, and security audit producers share a timestamp-plus-atomic-sequence ID generator; fixed-time concurrent generation and 30 milestone repetitions pass |

### Verification Commands

```powershell
go test ./internal/service/job ./internal/store/sql/gormrepo ./internal/store ./internal/bootstrap -run "TestService|TestMemoryRepositoryLeasesJobOnceConcurrently|TestJobStore|TestJobMigration|TestAllModels|TestNewBundle" -count=1 -v
go test ./internal/store/sql/gormrepo -run "TestJobStorePersistsLeaseRetryAndDeadLetterAcrossRestart|TestJobStoreLeasesDueJobOnceConcurrently" -count=30 -timeout 300s
go test ./internal/handler/middleware ./internal/bootstrap -run "TestNewAuditIDIsUniqueAtFixedTimestamp|TestPluginRuntimeMilestoneEndToEnd" -count=30 -timeout 300s
go test -race ./internal/service/job ./internal/store/sql/gormrepo ./internal/store ./internal/handler/middleware ./internal/bootstrap -run "TestService|TestMemoryRepositoryLeasesJobOnceConcurrently|TestJobStore|TestAllModels|TestNewBundle|TestNewAuditID|TestPluginRuntimeMilestoneEndToEnd" -count=1 -timeout 300s
go test ./... -count=1 -timeout 300s
go vet ./...
codegraph sync .
codegraph status .
codegraph impact JobStore
codegraph query NewAuditID
git diff --check
```

Result: all locally executable acceptance gates passed after three documented retries. The first full-suite failure exposed a real timestamp-ID collision in existing audit evidence; the next two focused builds exposed obsolete imports in that correction. After repair, 30 job soak rounds, 30 audit milestone rounds, race detection, the full repository suite, vet, CodeGraph synchronization, and diff validation all pass. Live MySQL and PostgreSQL DSNs were not available; dialect schemas are statically verified and restart/concurrency behavior executes against file-backed SQL.

### Impact Review

- Job contract: host and plugin work share one explicit lifecycle: `scheduled`, `running`, `retry_wait`, `succeeded`, and `dead_letter`.
- Execution safety: atomic conditional updates issue opaque lease tokens; stale or expired workers cannot overwrite a newer execution.
- Recovery: schedules, active leases, attempt counters, retry times, results, errors, and dead-letter timestamps use the configured repository as the only authority.
- Observability: namespace, kind, status, error, result, completion, and dead-letter filters expose operational state without reconstructing logs.
- Runtime: store bundles and SQL adapters expose one current job repository; bootstrap no longer needs a private scheduler state path.
- Audit integrity: one process-wide sequence prevents same-timestamp audit events from replacing each other in repositories keyed by event ID.
- Migrations: version `000026` adds the same current job model and operational indexes for MySQL and PostgreSQL.
- API/frontend: no wire contract or user interface changed.
- Compatibility: none; no legacy queue adapter, in-memory fallback, dual write, old-state reader, or migration bridge was added.

### Commit

`PR2-04: persist scheduled jobs and dead letters`

## PR2-05 Retry Record 1

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: full repository test suite
- Evidence: `tests/integration/pharma_data_scope_matrix_test.go` still passed the internal RBAC service directly to `RegisterCustomerRoutes`; the new public `pluginsdk.DataScopeService` contract rejected the stale dependency at compile time. Focused transaction, scope, SQL, Pharma OA, bootstrap, repeated, and race gates passed.
- Retry action: construct the host data-scope adapter from the integration fixture's RBAC and organization dependencies, inject only the public plugin SDK port, then rerun the full PR2-05 acceptance sequence.

## PR2-05 Publish Host Transaction And Trusted Data-Scope Services

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Publish stable Go plugin SDK ports for host transactions and trusted data scope, provide strict host adapters, and move the Pharma OA customer boundary to the public contract.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Atomic plugin operation | Pass | File-backed SQL contract writes a header and line in one host transaction and proves both roll back on callback failure |
| Transaction context propagation | Pass | Unit of Work publishes the active GORM transaction through callback context and nested calls reuse the same boundary |
| Missing transaction dependency | Pass | Host service construction and common transaction manager fail closed without a Unit of Work or callback |
| Trusted identity | Pass | Data scope reads JWT claims only from authenticated context; missing subject or non-super tenant is rejected |
| Self scope | Pass | Owner is pinned to the authenticated subject and ignores request/resolver user IDs |
| Tenant boundary | Pass | Non-super all scope is bounded to the trusted tenant root and its organization descendants |
| Request narrowing | Pass | Immutable predicates intersect tenant, owner, and organization filters; forged dimensions become denied and cannot authorize a record |
| Global scope | Pass | Only `super_admin` receives all tenants, owners, and organizations |
| Pharma OA integration | Pass after retry | Customer routes receive `pluginsdk.DataScopeService`; the complete self/department/tree/all/denied/super-admin matrix passes through the host adapter |
| Runtime wiring | Pass | Bootstrap constructs and validates one `HostServices` value from configured Unit of Work, RBAC, and organization dependencies |
| Public developer contract | Pass | `docs/development/plugin-host-services.md` documents injection, transactions, trusted scope, narrowing, and mandatory conformance tests |

### Verification Commands

```powershell
go test ./pkg/pluginsdk ./internal/plugin/hostservice ./internal/store/sql ./internal/service/common ./internal/handler/http/v1/pharmaoa ./internal/plugin/pharmaoa ./internal/bootstrap ./tests/integration -run "TestScope|TestHostServices|TestDataScope|TestTransaction|TestUnitOfWork|TestCustomer|TestBackend|TestPharmaCustomerOrganizationScopeMatrix" -count=20 -timeout 300s
go test -race ./pkg/pluginsdk ./internal/plugin/hostservice ./internal/store/sql ./internal/service/common ./internal/handler/http/v1/pharmaoa ./internal/plugin/pharmaoa ./internal/bootstrap ./tests/integration -run "TestScope|TestHostServices|TestDataScope|TestTransaction|TestUnitOfWork|TestCustomer|TestBackend|TestPharmaCustomerOrganizationScopeMatrix" -count=1 -timeout 300s
go test ./... -count=1 -timeout 300s
go vet ./...
codegraph sync .
codegraph status .
codegraph impact NewTransactionService
codegraph impact ScopePredicate
git diff --check
```

Result: all acceptance gates passed after one documented retry. The retry removed the final integration-test dependency on the internal RBAC service; repeated contract tests, race detection, the full repository suite, vet, CodeGraph synchronization, impact review, and diff validation now pass.

### Impact Review

- Public SDK: `pkg/pluginsdk` owns transaction, permission, host-service, trusted-scope, narrowing, and record-predicate contracts without exposing host internals.
- Transactions: SQL callbacks use GORM's transaction API, nested calls reuse the active context, and absent boundaries fail closed instead of executing a callback without atomicity.
- Scope security: tenant, owner, and organization dimensions are immutable defensive copies and combine with AND semantics.
- Identity: request payloads and query strings cannot select authorization identity; only verified JWT context determines subject and tenant.
- Pharma OA: customer handlers consume the public scope port and preserve the existing organization-scope integration matrix.
- Runtime: the current host services are mandatory dependencies; no legacy RBAC injection, dual path, fallback, or compatibility adapter remains.

### Commit

`PR2-05: publish trusted plugin host services`

## PR2-06 Publish File, Audit, Config, And Secret Services

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Publish bounded file, audit, config, and secret SDK ports; bind them to one trusted plugin identity; remove direct file and audit service injection from the Pharma OA plugin boundary.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Complete host contract | Pass | `HostServices.Validate` rejects an empty plugin identity or any missing transaction, scope, file, audit, config, or secret port |
| File ownership | Pass | Keys are fixed under `plugins/{pluginID}/`; list/get/download/delete recheck source plugin ownership and reject cross-plugin access |
| File content policy | Pass | Host computes size, MIME, and SHA-256; executable content, executable names, path traversal, public visibility, and oversized bodies are rejected |
| Trusted file identity | Pass | Owners and audit actors come from verified JWT context or the bound background plugin actor, never plugin input |
| Audit namespace | Pass | Actions and resources are bound to the plugin namespace; the Pharma OA adapter normalizes existing business names before recording |
| Audit redaction | Pass | Nested secrets, passwords, tokens, credentials, authorization headers, cookies, and sessions are recursively redacted; metadata is limited to 64 KiB |
| Config validation | Pass | Config is validated against the current manifest Schema; sensitive keys are rejected and a declared but unavailable Schema fails closed without saving |
| Secret isolation | Pass | Secret names use a plugin-specific hashed namespace; AES-256-GCM ciphertext is stored as encrypted settings and audit details never contain plaintext |
| Pharma OA boundary | Pass | Plugin dependencies expose only public `HostServices`; internal file and audit service fields and bootstrap injection are absent |
| Existing attachment flow | Pass | Contract and complaint uploads declare `sourcePluginId=pharma_oa`; bounded reads preserve the current owner/source workflow |
| Runtime wiring | Pass | Bootstrap constructs all ports from current host services and refuses an undersized master secret |
| Public developer contract | Pass | `docs/development/plugin-host-services.md` documents file, audit, config, secret, identity, failure, and conformance rules |

### Verification Commands

```powershell
go test ./internal/plugin/hostservice ./pkg/pluginsdk ./internal/plugin/pharmaoa -count=20
go test -race ./internal/plugin/hostservice ./pkg/pluginsdk ./internal/plugin/pharmaoa -count=1
go test ./... -count=1
go vet ./...
codegraph sync .
codegraph impact FileService
codegraph impact HostServices
codegraph status .
rg -n "Audit\s+auditsvc\.Service|File\s+filesvc\.Service|deps\.Audit|deps\.File" internal/plugin/pharmaoa internal/bootstrap/di.go
git diff --check
```

Result: all acceptance gates passed without retry. Repeated security contracts, race detection, the complete repository suite, vet, CodeGraph synchronization and impact review, direct dependency scanning, and diff validation pass.

### Impact Review

- Public SDK: plugins receive explicit file, audit, config, and secret ports without importing host service, repository, store, or encryption implementations.
- Files: source, key prefix, owner, actor, MIME, hash, visibility, size, and executable policy are host-owned decisions.
- Audit: persisted identity is derived from trusted context; claimed business actors remain evidence only and cannot select the audit principal.
- Config and secrets: ordinary config remains Schema-validated and redacted, while encrypted secret storage is isolated by plugin and key.
- Pharma OA: contracts, complaints, report exports, and all business audits flow through public host ports using one-way plugin adapters.
- Runtime: every port is mandatory; no legacy injection, permissive fallback, dual path, or compatibility bridge remains.

### Commit

`PR2-06: publish bounded plugin host services`

## PR2-07 Retry Record 1

- Date: 2026-07-22
- Status: `Review -> Failed -> Doing`
- Failed gate: namespace side-effect review after the initial PR2-07 acceptance sequence
- Evidence: job lease candidates were filtered by namespace, but both memory and SQL repositories expired exhausted leases globally before selecting candidates. A plugin worker could not lease another plugin's job, yet it could advance another namespace from `running` to `dead_letter`.
- Retry action: bind exhausted-lease transition to the requested namespace in both repositories, add memory and SQL cross-namespace regression tests, then rerun the complete PR2-07 acceptance sequence.

## PR2-07 Freeze Current Plugin SDK Contracts

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Failed -> Doing -> Review -> Done`
- Scope: Freeze the current public plugin contract for lifecycle, workflow, jobs, transactions, data scope, files, audit, configuration, and secrets; prove it with a third-party conformance plugin that imports no host internals.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Public SDK completeness | Pass | `HostServices.Validate` requires transaction, scope, file, audit, config, secret, workflow, and job ports |
| Third-party import boundary | Pass | `plugins/sdk-conformance` imports only the Go standard library and `pkg/pluginsdk`; dependency and AST checks reject host internals |
| Plugin lifecycle | Pass | The conformance manifest installs, enables, disables, and uninstalls through the current lifecycle without fallback paths |
| Shared host services | Pass | The conformance plugin executes transaction, scope, file, audit, config, and secret operations through public SDK ports |
| Workflow contract | Pass | Definition, publication, instance start, and approval pass with plugin-bound identifiers, business types, trusted actors, and cross-plugin denial |
| Job contract | Pass after retry | Schedule, lease, completion, retry, query, and dead-letter transitions are bound to `plugin.{pluginID}` in memory and SQL stores |
| Cross-plugin side effects | Pass after retry | A namespace cannot lease or expire another namespace's running job; dedicated memory and SQLite regressions pass repeatedly |
| Pharma OA boundary | Pass | Pharma OA receives workflow through public `HostServices`; direct workflow, file, and audit service injection is absent |
| Developer reference | Pass | The SDK reference documents all ports, lifecycle, workflow, job, identity, failure, and conformance rules and is linked from plugin guides |
| Current-only architecture | Pass | No legacy injection, compatibility adapter, dual path, or permissive fallback is present |

### Verification Commands

```powershell
go test ./internal/plugin/hostservice ./pkg/pluginsdk ./plugins/sdk-conformance ./internal/service/job ./internal/store/sql/gormrepo -run "Conformance|HostServices|WorkflowService|JobService|JobStore|Service.*Job|Namespace" -count=20
go test -race ./internal/plugin/hostservice ./pkg/pluginsdk ./plugins/sdk-conformance ./internal/service/job ./internal/store/sql/gormrepo -run "Conformance|HostServices|WorkflowService|JobService|JobStore|Service.*Job|Namespace" -count=1
go test ./... -count=1
go vet ./...
go list -deps ./plugins/sdk-conformance
codegraph sync .
codegraph impact WorkflowService
codegraph impact JobService
codegraph impact LeaseDue
codegraph impact HostServices
codegraph status .
git diff --check
```

Result: all acceptance gates passed after one documented retry. The retry removed the last cross-plugin dead-letter side effect; repeated conformance and namespace tests, race detection, the full repository suite, vet, dependency checks, CodeGraph synchronization and impact review, documentation checks, boundary scans, and diff validation now pass.

### Impact Review

- SDK: workflow and durable job contracts are public, mandatory, and independent of host implementation packages.
- Isolation: workflow identifiers and business types are plugin-bound; durable jobs require an explicit namespace for every lease and expiration transition.
- Identity: workflow actors and audit principals come from trusted context or the bound background plugin identity.
- Reference plugin: one executable conformance scenario demonstrates the supported third-party development surface end to end.
- Pharma OA: the business plugin consumes the same public workflow contract available to external plugins.
- Architecture: only the current contract remains; no compatibility bridge or direct internal service path was introduced.

### Commit

`PR2-07: freeze plugin SDK contracts`

## PR3-01 Retry Record 1

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused generator package compilation
- Evidence: the memory repository test renderer was accidentally inserted inside the raw generated store template, so the template dispatcher could not resolve `renderMemoryStoreTest`.
- Retry action: move the renderer into generator source scope, rerun formatting and focused compilation, then execute the complete PR3-01 acceptance sequence from the beginning.

## PR3-01 Retry Record 2

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: generated temporary module `go test ./...`
- Evidence: the domain template exported `id` as `Id`, while memory repository and service templates referenced a hard-coded `ID` field. Syntax checks passed, but generated packages did not compile together.
- Retry action: implement Go initialism-aware exported names, derive repository and service identity access from the declared primary field, and restart focused and generated-module tests.

## PR3-01 Retry Record 3

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused Demo Product backend contract suite
- Evidence: semantic GORM mapping output and the generated temporary-module tests passed, but two content assertions required one exact spacing form that `gofmt` expanded for field alignment.
- Retry action: assert stable mapping expressions instead of formatter-owned whitespace and rerun the complete generator package suite.

## PR3-01 Generate Backend Domain, Application, Repository, And Tests

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Generate a current Go backend vertical slice containing domain, repository port, memory repository, GORM model/repository, application service, and executable generated tests without manual correction.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Metadata identity contract | Pass | Specs require exactly one required `id`-typed primary field; missing, optional, multiple, and non-ID primary definitions are rejected |
| Go naming | Pass after retry | Initialisms including ID, API, IP, and URL use idiomatic exported names consistently across domain, repository, service, persistence, and tests |
| Domain generation | Pass | Entity, constructor, validation, audit metadata, package doc, and constructor tests are generated and formatted |
| Repository generation | Pass | Repository port and concurrency-safe memory CRUD store are generated from the declared primary field |
| GORM persistence | Pass after retry | Models map every field and audit timestamp in both directions, propagate reconstruction errors, and query the declared primary column |
| Application generation | Pass | CRUD service port and implementation preserve IDs and creation time while generating complete audit action constants |
| Generated tests | Pass | Domain validation, memory CRUD, GORM full-entity round trip, and service CRUD tests are emitted as first-class candidates |
| Clean-module compilation | Pass after retry | Demo Product backend output is written to a clean temporary module and its generated `go test ./...` succeeds without edits |
| Golden and regeneration | Pass | The 24-file deterministic snapshot, content hashes, dry-run conflict classification, history, and rollback tests pass repeatedly |
| Module boundaries | Pass | Generated domain, repository, store, and service packages compile together with dependencies flowing inward through repository contracts |
| Developer example | Pass | Demo Product backend acceptance and 24-file output matrix document generated tests and clean-module verification |
| Current-only architecture | Pass | Invalid legacy primary-key shapes are rejected; no compatibility mode, alternate template path, or fallback output exists |

### Verification Commands

```powershell
go test ./internal/domain/generator ./internal/service/generator -count=3
go test -race ./internal/domain/generator ./internal/service/generator -count=1
go test ./... -count=1
go vet ./...
codegraph sync .
codegraph impact NewGeneratorSpec
codegraph impact buildCandidates
codegraph impact renderGORMStore
codegraph impact exportedName
codegraph status .
git diff --check
```

Result: all acceptance gates passed after three documented retries. The completed suite proves generated source syntax, cross-package compilation, generated test execution, persistence round trips, deterministic golden output, race safety, full repository behavior, static analysis, CodeGraph impact review, documentation consistency, and clean diffs.

### Impact Review

- Metadata: current generator specs now match the repository's canonical `shared.ID` contract and fail before rendering unsupported identity shapes.
- Domain: generated constructors own validation and audit metadata initialization.
- Persistence: memory and GORM stores share the declared primary field; GORM decoding cannot silently return a zero entity.
- Application: generated services preserve immutable identity and creation metadata across updates.
- Testing: generated code carries its own domain, repository, mapping, and service regressions into downstream projects.
- Architecture: the generator emits one current backend shape; no compatibility template, manual patch phase, or hidden fallback remains.

### Commit

`PR3-01: generate tested backend slices`

## PR3-02 Retry Record 1

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused Demo Product generator contract
- Evidence: the output safety matrix rejected the new module-specific `internal/bootstrap/generated_<module>_catalog.go` path because it still allowed only the former shared seed file.
- Retry action: replace shared-file allowances with explicit current module prefixes, update path contracts and example matrices, then restart focused generator tests.

## PR3-02 Retry Record 2

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused Demo Product catalog contract
- Evidence: the generated catalog correctly used typed permission inputs and `domainmenu.NewNode`, while the existing assertion still required the removed `DemoProductGeneratedMenu` map.
- Retry action: validate the current typed catalog registration function, menu constructor, path, component, and required permission, then restart focused tests.

## PR3-02 Retry Record 3

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: generator golden snapshot
- Evidence: the complete deterministic output changed after adding the current HTTP routes, typed catalog, OpenAPI operations, handler tests, and response-envelope alignment, while the expected snapshot hash still represented the previous template set.
- Retry action: replace only the expected hash with the full snapshot hash emitted by the strict golden test, retain deterministic content and idempotency assertions, and restart the complete PR3-02 acceptance sequence.

## PR3-02 Generate Handlers, Routes, Permissions, Audit, And OpenAPI

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Generate a current HTTP and API contract surface for each backend slice, with executable route declarations, platform response envelopes, typed permission/menu registration, mutation audit metadata, and one deterministic OpenAPI definition.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| HTTP routes | Pass | Generated handlers expose list/get/create/update/delete routes and validate IDs, JSON bodies, offsets, and bounded limits |
| Route declarations | Pass | Five generated `RouteContract` entries bind method, path, permission, and mutation audit action without shared router-file collisions |
| Response envelope | Pass | Handlers use platform `code/message/data`; list emits `items/offset/limit`, create/update emit `item`, and generated frontend clients consume those exact shapes |
| Permission catalog | Pass after retry | Typed `permissionsvc.RegisterResourceInput` entries register read/create/update/delete/manage resources with explicit risks |
| Menu catalog | Pass after retry | Module-specific bootstrap code builds a current `domainmenu.Node` and merges it through `menusvc.Service` |
| Audit declarations | Pass | Create, update, and delete routes each declare a non-empty generated audit action; generated contract tests reject omissions |
| OpenAPI uniqueness | Pass | Parsed YAML contains list/get/create/update/delete exactly once with request/response schemas and permission/audit extensions on mutations |
| Generated tests | Pass | Emitted handler tests execute create/list/bad-query flows and contract tests verify all route security declarations |
| Clean-module compilation | Pass | Generated backend, response helper, HTTP handler tests, and route wrapper compile and pass in a temporary clean module |
| Determinism | Pass after retry | The 25-file golden snapshot and idempotent regeneration suite pass repeatedly using the strict current hash |
| Impact review | Pass | CodeGraph limits renderer changes to candidate dispatch and `DryRun`; no unexpected runtime or domain callers were introduced |
| Current-only architecture | Pass | Output uses one typed catalog, one response contract, and module-specific route/catalog paths; no legacy seed, fallback, or compatibility branch remains |

### Verification Commands

```powershell
go test ./internal/domain/generator ./internal/service/generator -count=3
go test -race ./internal/domain/generator ./internal/service/generator -count=1
go test ./... -count=1
go vet ./...
codegraph sync .
codegraph impact renderHandler
codegraph impact renderOpenAPI
codegraph impact renderPermissionSeed
codegraph impact renderFrontendAPI
codegraph impact buildCandidates
codegraph status .
git diff --check
```

Result: all acceptance gates passed after three documented retries. Generated mutations now carry enforceable permission and audit declarations, the current OpenAPI contains every generated operation once, generated HTTP tests execute without manual edits, and frontend clients share the platform response contract.

### Commit

`PR3-02: generate secured HTTP contracts`

## PR3-03 Retry Record 1

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused generator package compilation
- Evidence: the generated plugin README raw-string template contained an unescaped nested backtick around the table placeholder, which ended the Go literal before the remaining template content.
- Retry action: keep the table name in plain Markdown text inside the raw template, rerun formatting, and restart the focused metadata and generator tests.

## PR3-03 Retry Record 2

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused generator package compilation
- Evidence: after removing the table-name backticks, the same raw README template still contained newly added backticks around rollback and uninstall policy placeholders.
- Retry action: remove the remaining nested backticks from the raw template and restart formatting plus the complete focused test set.

## PR3-03 Retry Record 3

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: generated plugin lifecycle Loader validation
- Evidence: text-level manifest assertions passed, but the current `FileLoader` rejected both generated retain-policy and drop-policy manifests as invalid.
- Retry action: include the materialized manifest in failure evidence, identify the exact current-contract violation, fix the renderer, and restart the full focused metadata/generator/lifecycle matrix.

## PR3-03 Retry Record 4

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: generator golden snapshot
- Evidence: all metadata, Loader, migration lifecycle, and seed idempotency tests passed, while the strict snapshot still represented migrations without primary-key and audit timestamp declarations.
- Retry action: replace only the expected hash with the complete deterministic hash emitted by the golden test and restart the full PR3-03 acceptance sequence.

## PR3-03 Generate Current Migrations, Seed, And Lifecycle Metadata

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Generate executable current-format database migrations and plugin lifecycle metadata, enforce plugin-owned namespaces before rendering, and prove seed and migration lifecycle idempotency without compatibility output.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Metadata policies | Pass | Plugin specs normalize and validate explicit `retain/drop/archive` uninstall and `manual/automatic/none` rollback policies |
| Data ownership | Pass after retry | Plugin tables, indexes, permissions, and menu keys must use the plugin data namespace before any file is rendered |
| Core migrations | Pass | MySQL and PostgreSQL output creates tables idempotently with the declared primary key, required audit timestamps, field uniqueness, and indexes |
| Executable SQL | Pass | Both dialect outputs execute against independent SQLite databases; schema inspection proves primary key, audit columns, and index, and down SQL removes the table |
| Current manifest | Pass after retry | Generated plugin manifests pass the real `FileLoader` with namespaced table metadata and three-segment plugin audit actions |
| Fresh install | Pass | The current `PluginMigrationHook` discovers the generated up/down pair, executes one fresh step, and records one ledger entry |
| Repeated upgrade | Pass | Reapplying the generated current version executes no SQL and leaves one migration ledger entry |
| Seed idempotency | Pass | Permission and menu services receive the generated catalog shape twice and retain exactly five permissions and one menu node |
| Automatic rollback | Pass | The generated automatic policy executes the down migration and clears the ledger in reverse lifecycle flow |
| Retain uninstall | Pass | The generated retain policy performs no destructive SQL and preserves the applied migration ledger |
| Drop uninstall | Pass | A generated drop-policy manifest executes the down migration and removes its ledger entry |
| Determinism | Pass after retry | The strict 25-file golden snapshot reflects the complete migration contract and remains idempotent across repeated dry runs |
| Current-only architecture | Pass | Invalid unnamespaced plugin metadata is rejected; no legacy manifest key, migration parser, fallback policy, or dual output path was added |

### Verification Commands

```powershell
go test ./internal/domain/generator ./internal/service/generator -count=3
go test -race ./internal/domain/generator ./internal/service/generator -count=1
go test ./... -count=1
go vet ./...
codegraph sync .
codegraph impact PluginSpec
codegraph impact validatePluginDataOwnership
codegraph impact renderMigration
codegraph impact renderPluginManifest
codegraph impact pluginAPIRoutes
codegraph status .
git diff --check
```

Result: all acceptance gates passed after four documented retries. Generated migrations now match generated persistence models, generated plugins are accepted by the current runtime contract, catalog seeds remain idempotent, and install/upgrade/rollback/uninstall policies execute as declared.

### Commit

`PR3-03: generate plugin lifecycle data contracts`

## PR3-04 Retry Record 1

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused generator contract suite
- Evidence: the output safety matrix blocked the new generated locale and route paths; previous frontend assertions also required the removed array-only list return and raw Element Plus page structure.
- Retry action: allow the explicit current i18n/router prefixes, preserve the plugin page ownership marker on `PageShell`, update assertions to the typed page and shared-kit contracts, and restart focused tests.

## PR3-04 Retry Record 2

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: focused Demo Product frontend contract
- Evidence: generated stores correctly imported the shared `toErrorMessage`, while two assertions still required a page-local duplicate helper; the strict golden hash still represented the previous 25-file set.
- Retry action: assert shared helper reuse, defer the golden update until all behavior gates stabilized, and restart the focused suite.

## PR3-04 Retry Record 3

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: generated frontend `vue-tsc`
- Evidence: the Windows test process changed its working directory before launching a relative `vue-tsc.cmd`, so the executable could not be resolved.
- Retry action: resolve the web workspace and frontend toolchain to absolute paths before materialization and rerun the executable frontend gate.

## PR3-04 Retry Record 4

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: generated frontend browser smoke
- Evidence: the combined test exceeded its outer timeout because killing the Vite command wrapper did not stop its child Node preview process during cleanup.
- Retry action: launch Vite through its Node entry point, add hard Chrome subprocess timeouts and independent browser profiles, terminate the exact preview process, and restart the complete generated frontend gate.

## PR3-04 Retry Record 5

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: desktop Chrome DOM smoke
- Evidence: typecheck and production build passed, but temporary shared-component fixtures used runtime template strings unavailable in the production Vue runtime, so Chrome rendered an empty root.
- Retry action: replace fixture templates with explicit render functions, retain nonblank DOM and narrow screenshot assertions, and rerun typecheck, build, desktop, and mobile smoke.

## PR3-04 Generate Element Plus Frontend Workflows

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Generate a current Element Plus frontend workflow containing typed list/detail/mutation clients, durable Pinia state, permission-aware lazy routes, Chinese/English locale keys, and shared-kit list/form/detail UI with executable frontend and browser acceptance.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Typed API | Pass | Generated clients expose typed page/get/create/update/delete operations, encode IDs, and consume the platform response envelope |
| Store states | Pass | Pinia output separates list, detail, and mutation status/errors; preserves query/page state; and provides retry, selection, mutation, and removal actions |
| Permission route | Pass after retry | Each module emits one lazy authenticated route with its read permission; plugin routes use the manifest frontend entry |
| Bilingual locale | Pass | Generated module-local `zh-CN` and `en-US` dictionaries cover page, actions, fields, validation, empty, permission, success, and destructive copy |
| Shared UI kit | Pass | Generated pages compose `PageShell`, `FilterBar`, `DataTable`, `DetailDrawer`, and `ConfirmAction` with Element Plus controls and Lucide icons |
| Loading/empty/error | Pass | Initial loading/error, stale-list error, detail loading/error, mutation error, and empty table states have explicit render paths |
| Permission state | Pass | Read denial blocks data loading and page content; create/update/delete actions bind their generated permission declarations |
| Validation/save | Pass | Element Plus form rules, typed controls, saving state, backend error preservation, and success feedback execute without hand edits |
| Destructive action | Pass | Delete requires shared confirmation, exposes progress, updates the store, and reports success/failure |
| Responsive layout | Pass | Stable controls and pagination include an explicit 760px narrow layout; Chrome produces a non-empty 390x844 screenshot |
| Generated typecheck/build | Pass after retry | The exact five frontend outputs materialize into a clean current workspace and pass `vue-tsc` plus Vite production build |
| Browser render | Pass after retry | Headless Chrome verifies nonblank desktop DOM markers and narrow viewport pixels against the production bundle |
| Root frontend regression | Pass | i18n, accessibility, large-list, `vue-tsc`, and production build gates pass for the existing console |
| Determinism | Pass | The strict 27-file golden snapshot and repeated dry runs pass using the current hash |
| Current-only architecture | Pass | One typed page contract, one route shape, and one shared-kit UI path are emitted; no compatibility route, alternate UI template, or fallback output was added |

### Verification Commands

```powershell
go test ./internal/domain/generator ./internal/service/generator -count=3
go test -race ./internal/domain/generator ./internal/service/generator -count=1
$env:SKOLL_GENERATOR_FRONTEND_BUILD='1'; go test ./internal/service/generator -run TestGeneratedFrontendTypechecksAndBuilds -count=1 -v
cd web; npm run typecheck; npm run build
go test ./... -count=1
go vet ./...
codegraph sync .
codegraph impact buildCandidates
codegraph impact renderFrontendAPI
codegraph impact renderFrontendStore
codegraph impact renderFrontendView
codegraph impact renderFrontendRoute
codegraph impact renderFrontendLocale
codegraph status .
git diff --check
```

Result: all acceptance gates passed after five documented retries. Generated frontend slices now compile and render as complete current workflows, reuse the platform UI kit, expose permission and bilingual contracts, and cover the required async, validation, destructive, and responsive states without hand edits.

### Impact Review

- Candidate output: every module gains locale and route files; plugin targets receive the same current files under their own frontend root.
- API/store: list responses retain paging metadata and detail calls have an independent state channel.
- UI: generated pages use the same operational components, Element Plus controls, and Lucide actions as the host console.
- Routing/i18n: core routes use the menu path, plugin routes use the manifest frontend entry, and visible generated copy resolves through module-local Chinese/English dictionaries.
- Testing: build-heavy browser acceptance is explicit through `SKOLL_GENERATOR_FRONTEND_BUILD=1`; normal Go and race suites retain deterministic source-level coverage without repeatedly starting browsers.

### Commit

`PR3-04: generate Element Plus frontend workflows`

## PR3-05 Retry Record 1

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: manual package/install smoke command
- Evidence: the acceptance command hard-coded demo version `1.0.0`, while the current manifest generated `demo-0.2.0.zip`; packaging succeeded and checksum verification correctly rejected the nonexistent path.
- Retry action: resolve the current artifact from the isolated output directory and restart the complete package, verify, install, and dev sequence.

## PR3-05 Retry Record 2

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing`
- Failed gate: manual package/install smoke command
- Evidence: Windows PowerShell rejected the unsupported `Select-Object -Single` parameter before checksum verification.
- Retry action: use `Select-Object -First 1` for the isolated single-artifact directory and restart the complete sequence.

## PR3-05 Generate Packaging And Local Development Commands

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Generate executable PowerShell and Unix plugin commands backed by one deterministic, checksummed, path-safe package contract shared by CLI, developer portal, formal installation, and local development.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Generated commands | Pass | Each generated plugin gains `plugin.ps1` and `plugin.sh` commands for package, verify, install, and dev operations |
| Deterministic package | Pass | Sorted files, fixed ZIP metadata, stable modes, and repeated builds produce the same SHA-256 digest |
| Current manifest validation | Pass | Source, extracted staging directory, and published install directory are all accepted by the current `plugin.yaml` loader |
| Checksum contract | Pass | Every ZIP has a strict `<artifact>.sha256` sidecar; missing, malformed, renamed, and tampered artifacts fail verification |
| Safe extraction | Pass | Absolute paths, traversal, backslashes, symlinks, duplicate targets, oversized files, and oversized packages fail closed |
| Runtime assets | Pass | Built frontend `dist` assets are packaged while `.git`, `node_modules`, and in-source package output are excluded |
| Formal install | Pass | `install-package` verifies, stages, validates, atomically publishes, and passes the resulting directory to `RuntimeManager.Install` |
| Development install | Pass after retry | `dev` invokes the same build, checksum, extraction, loader, and manager chain as formal installation |
| Developer portal | Pass | Dev package and pipeline endpoints use `BuildPackage`, return checksum evidence, and no longer retain a direct ZIP implementation |
| CLI smoke | Pass after retry | The demo plugin packages, verifies, installs, and dev-installs in isolated roots; both installed manifests exist |
| Regression | Pass | Focused race suites, full `go test ./...`, and `go vet ./...` pass |
| Current-only architecture | Pass | One package shape and one install chain remain; no legacy archive parser, direct-source dev install, compatibility flag, or fallback path was added |

### Verification Commands

```powershell
go test ./internal/plugin ./internal/handler/cli ./internal/service/generator ./internal/handler/http/v1/plugin ./cmd/skoll-plugin
go test -race ./internal/plugin ./internal/handler/cli ./internal/service/generator ./internal/handler/http/v1/plugin -count=1
go run ./cmd/skoll-plugin package plugins/demo <isolated-dist>
go run ./cmd/skoll-plugin verify-package <artifact> <artifact.sha256>
go run ./cmd/skoll-plugin install-package <artifact> <artifact.sha256> <isolated-plugins-root>
go run ./cmd/skoll-plugin dev plugins/demo <isolated-dev-dist> <isolated-dev-plugins-root>
go test ./... -count=1
go vet ./...
codegraph sync .
codegraph impact BuildPackage
codegraph impact InstallPackage
codegraph impact buildCandidates
codegraph status .
git diff --check
```

Result: all acceptance gates passed after two documented command retries. Generated plugins now expose reproducible package workflows, all package entry points emit and enforce the same checksum contract, and local development installs the exact artifact shape consumed by the runtime instead of using a separate source-directory path.

### Impact Review

- Package core: deterministic archive construction and guarded extraction are centralized in `internal/plugin/package.go`.
- Runtime CLI: `cmd/skoll-plugin` and plugin CLI handlers expose package, verify, install, and dev commands without starting the host server.
- Developer portal: package and pipeline endpoints now return the artifact path, checksum path, and SHA-256 digest from the centralized builder.
- Generator: plugin candidates include cross-platform command files and README instructions that state the shared installed/dev contract.
- Testing: package determinism, tamper rejection, traversal rejection, CLI parity, HTTP checksum evidence, generator output, and full regressions are covered.

### Commit

`PR3-05: generate plugin packaging commands`

## PR3-06 Retry Records

- Date: 2026-07-22
- Status: `Doing -> Failed -> Doing` for every failed gate below; the complete gate restarted after each correction.

| Retry | Failed gate | Evidence | Correction |
| --- | --- | --- | --- |
| 1 | Generated command contract | The assertion required a direct `go run` path after the command was changed to build artifacts first | Assert the current build-backed command contract |
| 2 | Source immutability scan | A Windows `node_modules` directory link was treated as a source file | Classify directory links before hashing source files |
| 3 | Generated backend compile | A log placeholder rendered invalid double-quoted Go source | Render one valid formatted log call |
| 4 | Generated frontend typecheck | The standalone UI support emitted invalid nested TypeScript render syntax | Emit explicit valid render functions |
| 5 | Generated frontend typecheck | `ImportMeta.env` lacked the Vite client declaration | Add the generated Vite type reference |
| 6 | PowerShell package command | A parameter default referenced `$PSScriptRoot` before script parameter binding completed | Resolve script-relative defaults after parameter parsing |
| 7 | Package artifact assertion | The failed command produced no artifact and insufficient diagnostics | Capture generated command output and artifact directory evidence |
| 8 | Package command execution | The host CLI was launched from the generated plugin Go module and native command failures did not terminate PowerShell | Run the CLI from `SKOLL_REPO_ROOT` and fail fast on every native exit code |
| 9 | Package source scan | A Windows `node_modules` junction reached the generic unsupported-link rejection | Skip reserved `.git` and `node_modules` entries before fail-closed link handling |
| 10 | Focused generator test | A semantic assertion depended on `gofmt` constant alignment | Assert the API constant and value independently of whitespace |
| 11 | Race gate | The service exit watcher could overwrite `service_force_stopped` with `service_exited` after a stop timeout | Keep the lifecycle in `stopping` through force-stop, retain timeout audit evidence, and let one terminal transition win |

## PR3-06 Prove Generated Plugin Zero-Edit E2E

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: Generate a standalone current-format full-stack plugin and prove its build, package, runtime, business API, lifecycle, migration, and uninstall flow without editing generated files or host core source.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Zero-edit generation | Pass after retry | The generator emits a Go module/backend, standalone Vue project, manifest, migrations, tests, and cross-platform commands into an isolated temporary root |
| Source immutability | Pass after retry | Hashes of every generated source file are identical before and after build/package/install execution |
| Backend build | Pass after retry | Generated Go tests pass and the generated backend compiles into the package runtime directory |
| Frontend build | Pass after retry | Generated Vue, TypeScript, Element Plus, shared support, and Vite output pass `vue-tsc` and production build |
| Package integrity | Pass after retry | The generated command builds one deterministic ZIP and SHA-256 sidecar accepted by the centralized verifier and installer |
| Safe package content | Pass after retry | Built backend and `web/dist` are installed; `.git`, `node_modules`, output recursion, traversal, and unsupported links remain excluded or rejected |
| Migration lifecycle | Pass | The generated up migration executes before enable; drop uninstall executes down migration and clears the ledger |
| Service lifecycle | Pass after retry | The installed generated service becomes healthy, and timeout/force-stop terminal states cannot be overwritten by its exit watcher |
| Runtime lifecycle | Pass | `RuntimeManager` installs, enables, disables, and uninstalls the generated manifest using only current public runtime contracts |
| Business execution | Pass | The installed backend executes create, list, and detail requests through its declared plugin API namespace |
| Disable boundary | Pass | Disabled state rejects plugin access before uninstall |
| Full regression | Pass | Focused tests, race tests, full `go test ./...`, `go vet ./...`, frontend typecheck, and frontend production build pass |
| Current-only architecture | Pass | One generated package/runtime path exists; no legacy output, compatibility mode, source-install fallback, or dual lifecycle was introduced |

### Verification Commands

```powershell
go test ./internal/domain/generator ./internal/service/generator ./internal/plugin ./internal/handler/cli ./internal/handler/http/v1/plugin -count=1
go test -race ./internal/domain/generator ./internal/service/generator ./internal/plugin -count=1
$env:SKOLL_GENERATOR_PLUGIN_E2E='1'; go test ./internal/service/generator -run TestGeneratedPluginBuildPackageAndInstallWithoutSourceEdits -count=1 -v
go test ./... -count=1
go vet ./...
cd web; npm run typecheck; npm run build
codegraph sync .
codegraph impact PluginSpec
codegraph impact buildCandidates
codegraph impact ServiceSupervisor
codegraph status .
git diff --check
```

Result: all PR3-06 acceptance gates passed after eleven documented retries. A generated plugin now builds and executes as an independently packaged full-stack unit, exercises real lifecycle and business calls, and leaves its generated source and host core unchanged.

### Impact Review

- Generator contract: plugin specs may declare the one service base/health pair used by the current manifest and runtime.
- Generated runtime: every plugin target includes a standalone Go CRUD service and Vue production project instead of frontend-only fragments.
- Commands: generated build/package/dev commands compile both runtime halves and invoke the host package tool from the host module.
- Packaging: explicit reserved dependency directories are skipped before generic link rejection; all other unsupported links still fail closed.
- Lifecycle: stop timeout audit evidence is retained while force-stop owns the single terminal state transition.
- Performance handoff: the generated sample currently emits about 1.03 MB minified JavaScript and 358 KB CSS before gzip; PR4-06 must introduce and enforce route/package budgets rather than suppressing the Vite warning.

### Commit

`PR3-06: prove generated plugin zero-edit E2E`

## PR4-01 Audit Current UI And Freeze The Experience Target

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Audit the complete current Vue console, visually inspect the auth entry at desktop and narrow widths, classify architecture and experience gaps, and freeze a measurable current-only target for PR4-02 through PR4-06.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Frontend architecture | Pass | CodeGraph maps the app shell, 35 views, shared page/table/state components, theme store, plugin bridge, and high-impact consumers |
| Element Plus decision | Pass | Element Plus is frozen as the only control system; the baseline records 24 native buttons, nine native inputs, one native table, and duplicate local patterns for removal |
| Shared component inventory | Pass | App shell, page, toolbar, table, form, drawer, confirmation, state, plugin workspace, and test ownership are explicit |
| Theme decision | Pass | Theme is defined as independent `light|dark` scheme and `comfortable|compact` density axes; incomplete preset semantics are rejected directly |
| Primary workflows | Pass | Sixteen platform, plugin, workflow, IAM, foundation, and Pharma OA workflow entries identify gaps, owner, target components, and downstream Work Item |
| State matrix | Pass | Loading, empty, error/retry, forbidden, degraded, saving, success, destructive, and narrow requirements are frozen by workflow group |
| Inconsistency register | Pass | Fifteen P0-P2 findings have evidence, owner, target, measurable acceptance, and PR4 assignment |
| Responsive visual review | Pass | Real Chrome renders prove the login desktop baseline and reproduce 390x844 horizontal overflow |
| Production render review | Pass | Asset requests prove `vite preview` base mismatch and the blank `/skoll/login` production preview; UI-002 assigns the executable fix to PR4-06 |
| Performance baseline | Pass | Host initial and generated-plugin bundle evidence becomes enforceable PR4-06 budgets rather than warning suppression |
| Documentation routing | Pass | Current refactor README and PR4-01 deliverable link resolve with zero broken local links |
| Current-only architecture | Pass | The target explicitly forbids old theme migration, compatibility controls/routes, dual UI paths, and fallback design |

### Verification Commands

```powershell
codegraph explore "Vue frontend architecture primary admin workflows shared components Element Plus themes responsive states"
codegraph impact PageShell
codegraph impact DataTable
codegraph impact ThemeMode
rg --files web/src
rg -n --glob '*.vue' '<(button|input|table)\b' web/src
rg -n --glob '*.vue' --glob '*.scss' '#[0-9A-Fa-f]{3,8}\b|rgba?\(' web/src
npm run typecheck
npm run build
git diff --check
```

Result: PR4-01 acceptance passed. The new frontend experience target is the executable design source for PR4-02 through PR4-06; it records that the present UI is functional but not yet unified, theme-complete, mobile-safe, browser-tested, or performance-gated.

### Commit

`PR4-01: freeze frontend experience target`

## PR4-02 Unify Interactive Controls And Shared UI

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> (Failed -> Doing) x4 -> Review -> Done`
- Scope: Replace native and ad-hoc controls with Element Plus, make the mobile navigation reachable, and extend the shared UI kit with behavior-level component tests.

### Retry Records

| Retry | Failed gate | Evidence | Correction |
| --- | --- | --- | --- |
| 1 | Frontend typecheck | A dynamic `unpin` or `removeRecent` event name did not satisfy the typed Vue emit overload | Branch explicitly by tab kind so each event keeps its own typed contract |
| 2 | Component test runtime | Importing the complete Element Plus plugin made Node attempt to load theme CSS as a module | Test SKOLL event contracts through narrow Element component stubs; production build remains the real integration gate |
| 3 | Full frontend typecheck | New component fixtures omitted required `PinnedTab.id` and `SidebarItem.order/source` fields | Make fixtures conform to the production domain contracts and rerun the complete gate |
| 4 | 390px visual review | The login card used `width: 100%` without including its padding, which expanded the grid minimum width | Apply border-box sizing and verify a real 390x844 CDP viewport has `scrollWidth = innerWidth = 390` |

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Element Plus controls | Pass | Login, profile, uploads, batch import, workflow filters, todo filters, compliance source selection, and form-builder actions use Element Plus controls |
| Native control removal | Pass | `rg -n --glob '*.vue' '<(button|input|table)\b' web/src` returns no source matches |
| Shared layout | Pass | Plugin workspace uses `PageShell` and the new responsive `MetricStrip`; the unused native `Table` component is removed |
| Mobile navigation | Pass | App shell provides an Element drawer and a mobile Sidebar variant; navigation closes the drawer through an explicit event |
| Header and tabs | Pass | Locale, theme, profile, logout, menu, open, pin, and close actions use stable Element controls, Lucide icons, labels, and typed events |
| Forms and errors | Pass | Login and profile retain label, validation, disabled, loading, success, and error semantics through Element forms and alerts |
| Component behavior | Pass after retry | Vitest runs four behavior tests covering header contracts, tab events, mobile navigation, and metric warning/wide states |
| Responsive browser review | Pass after retry | Real Chrome CDP at 390x844 renders the login without horizontal overflow; authenticated shell evidence keeps the menu control in the first viewport |
| Frontend quality gates | Pass after retry | i18n, accessibility, large-list, `vue-tsc`, and Vite production build pass |
| Current-only architecture | Pass | One Element Plus/shared-kit interaction path remains; no native fallback, compatibility control, legacy component, or dual UI path was introduced |

### Verification Commands

```powershell
cd web
npm run test:components
npm run typecheck
npm run build
rg -n --glob '*.vue' '<(button|input|table)\b' src
codegraph sync ..
codegraph impact MetricStrip
codegraph impact Header
codegraph impact PinnedTabs
codegraph status ..
git diff --check
```

Result: PR4-02 passed after four documented retries. Primary console controls now share Element Plus behavior, the mobile navigation has an executable entry, shared operational metrics replace page-local summary cards, and component tests protect the most important interaction contracts.

### Commit

`PR4-02: unify frontend interaction controls`

## PR4-03 Complete Token-Driven Themes

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: Replace the mutually exclusive theme preset with independent color-scheme and density axes, publish the same current contract to plugins, and move all product colors into semantic tokens.

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Current theme model | Pass | Store state is exactly `colorScheme: light\|dark` plus `density: comfortable\|compact`; setters and storage keys are independent |
| No compatibility path | Pass | The removed `ThemeMode`, `data-theme-mode`, `skoll.ui.theme`, and compact-as-color contract have zero production references; the old key is ignored rather than migrated |
| Plugin theme contract | Pass | Host bridge sends only `colorScheme`, `density`, and resolved semantic tokens; iframe updates apply the same two root attributes |
| Semantic token coverage | Pass | Background, surface, border, text, primary, status, sidebar, code, loading, shadow, radius, spacing, control, Element Plus overlay, fill, and mask values have light/dark definitions |
| Product raw-color scan | Pass | Automated scan covers 101 TypeScript, Vue, SCSS, and CSS files; raw hex/RGB/gradient values exist only in `variables.scss` |
| Density behavior | Pass | Browser-computed Element input height changes from 36px to 32px and card radius from 8px to 6px without changing color scheme |
| Light contrast | Pass | Real Chrome computed light text/surface contrast at 15.97:1 |
| Dark contrast | Pass | Real Chrome computed dark text/surface contrast at 14.06:1 |
| Element state colors | Pass | Dashboard matrix resolves distinct light/dark overlay, warning, danger, success, primary, fill, and border tokens |
| Desktop matrix | Pass | Light/comfortable, light/compact, dark/comfortable, and dark/compact render at 1280x900 with no horizontal overflow |
| Mobile matrix | Pass | Dark/compact dashboard at 390x844 has no horizontal overflow; all three segmented controls and the menu action remain reachable |
| Automated behavior | Pass | Ten Vitest tests cover all four theme combinations, storage, root attributes, plugin payload, host script, and layout controls |
| Frontend quality gates | Pass | i18n, accessibility, large-list, theme scan, `vue-tsc`, and Vite production build pass |

### Verification Commands

```powershell
cd web
npm run test:components
npm run check:theme
npm run typecheck
npm run build
codegraph sync ..
codegraph impact ThemeBridgePayload
codegraph impact useThemeStore
codegraph status ..
git diff --check
```

Result: PR4-03 passed without acceptance retries. Theme and density are now orthogonal current contracts across the host and plugin bridge, product screens consume semantic tokens, and the four visual combinations have executable contrast, size, overflow, and state-color evidence.

### Commit

`PR4-03: complete token-driven themes`

## PR4-04 Complete Chinese/English Copy And Accessibility

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> (Review -> Failed -> Doing) x10 -> Review -> Done`
- Scope: Move all production Vue UI copy into the current Chinese/English locale contract, synchronize document language, close keyboard/ARIA/focus gaps, and enforce the result with static, component, backend-route, and real-browser gates.

### Retry Records

| Retry | Failed gate | Evidence | Correction |
| --- | --- | --- | --- |
| 1 | Frontend typecheck | `Object.hasOwn` was outside the configured TypeScript library target | Use `Object.prototype.hasOwnProperty.call` for interpolation keys |
| 2 | Browser login | Selenium `clear()` left the Element Plus model value and produced `adminadmin` | Select the full controlled value with Ctrl+A before entering credentials |
| 3 | Browser login route | `/skoll/v1/auth/login` returned 404 because declared system-builtin routes were not mounted by the HTTP router | Register enabled system-builtin extension routes as canonical endpoints and add router integration coverage |
| 4 | Locale semantics | Chinese copy rendered while `document.documentElement.lang` remained `en` | Apply the active locale to the root document on initialization and every locale change |
| 5 | Runtime ARIA scan | An input wrapped by a native label was incorrectly reported unnamed | Resolve both explicit and wrapping labels in the browser accessibility probe |
| 6 | Reduced-motion check | Chrome serialized `0.00001s` as `1e-05s` | Compare parsed duration seconds against the budget instead of string formatting |
| 7 | Customer filters | Status and region combobox inputs had no explicit accessible names | Add localized ARIA labels to customer keyword and select controls |
| 8 | Qualification filters | Subject and status combobox inputs had no explicit accessible names | Add localized ARIA labels to all qualification filters |
| 9 | Mobile keyboard traversal | The first focusable dashboard date input had no stable accessible name | Add a localized dashboard-window ARIA label and richer focus diagnostics |
| 10 | Final restricted login | Hidden retained SPA inputs inflated the login input count | Target only visible username and current-password inputs |

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Full production locale inventory | Pass | Static gate scans 53 production Vue files, resolves 1,645 references against 1,800 bilingual keys, and reports zero hard-coded visible strings |
| Platform and workflow copy | Pass | Form Builder, Todo Center, Workflow, plugin operations, IAM pages, shared confirmations, sidebar, statuses, and errors use the current locale contract |
| Locale behavior | Pass after retry | Locale switching updates copy, persisted state, interpolation, known errors, and the root `lang` attribute |
| Notification demo contract | Pass | Callers must provide localized demo copy; no optional copy or fallback compatibility path remains |
| Static accessibility | Pass | All 53 production UI files pass icon-name, image/iframe, click semantics, positive-tabindex, localized confirmation, focus, and reduced-motion checks |
| Keyboard and focus | Pass after retry | First tab stops have accessible names and visible 3px outlines; destructive dialogs receive focus |
| ARIA and page structure | Pass after retry | Page shells reference visible headings; tested inputs and comboboxes expose localized accessible names; no unnamed control remains |
| Contrast and motion | Pass after retry | Browser matrix enforces body contrast >= 4.5:1 and transition/animation durations <= 0.00001s under reduced motion |
| Error and permission states | Pass | Localized network errors and restricted-user denial states render as explicit status/alert surfaces |
| Browser matrix | Pass after retries | 28 scenarios cover Chinese/English, 1440x1000/390x844, responsive/loading/empty/error/destructive/saving/no-permission states with no overflow |
| Canonical auth route | Pass after retry | Enabled system-builtin extension routes execute directly; disabled builtins and app plugins are not directly mounted |
| Automated tests | Pass | Six Vitest files and 13 tests cover i18n, notifications, theme, host SDK, metrics, and layout controls |
| Full quality gates | Pass | `npm run typecheck`, `npm run build`, and `go test ./...` pass |
| Current-only architecture | Pass | Locale, notification, route, and accessibility behavior have one current contract with no compatibility branch, fallback, or dual path |

### Verification Commands

```powershell
cd web
npm run test:components
npm run typecheck
npm run build
cd ..
go test ./...
python scripts/h4-browser-matrix.py --base-url http://127.0.0.1:5174 --output "$env:TEMP/skoll-pr4-04-browser"
codegraph sync .
codegraph status .
git diff --check
```

Result: PR4-04 passed after ten documented retries. The complete production UI now uses one bilingual locale contract, document language and accessible names follow the active locale, and the 28-state real-browser matrix protects keyboard, focus, ARIA, contrast, reduced motion, error, and permission behavior.

### Commit

`PR4-04: complete frontend localization and accessibility`

## PR4-05 Complete Responsive Layouts And UI State Coverage

- Date: 2026-07-22
- Owner: Codex
- Status flow: `Todo -> Doing -> (Review -> Failed -> Doing) x2 -> Review -> Done`
- Scope: Establish a real Playwright matrix for the current business UI, verify primary platform work surfaces, and enforce responsive state geometry for Chinese/English at desktop and 390px mobile widths.

### Retry Records

| Retry | Failed gate | Evidence | Correction |
| --- | --- | --- | --- |
| 1 | Playwright state matrix | Customer list and qualification reminders issued concurrent matching requests, but the test retained only one release callback; error refresh was also located outside the page action boundary | Give every matching request the same bounded delay and scope refresh actions to `PageShell` |
| 2 | Primary-surface matrix | Login completed before backend plugin synchronization, so immediate navigation entered a dynamic-route rebuild window and rendered an empty workspace | Treat the localized `Synced` header state as part of login readiness before visiting plugin routes |

### Acceptance Matrix

| Scenario | Result | Evidence |
| --- | --- | --- |
| Playwright foundation | Pass | `@playwright/test` 1.61.1, a current Chrome-channel configuration, deterministic one-worker execution, JSON results, retained failure traces, and failure screenshots are wired through `npm run test:states` |
| Locale and viewport matrix | Pass | Chinese and English each pass at 1440x1000 desktop and 390x844 mobile widths |
| Primary work surfaces | Pass after retry | Plugin workspace, Form Builder, and Developer Portal render after explicit plugin synchronization in all four locale/viewport projects |
| Responsive | Pass | Pharma dashboard body renders with no horizontal overflow, clipped visible text, control escape, header/content overlap, or post-stabilization movement |
| Loading | Pass after retry | Delayed customer APIs expose a stable PageShell skeleton without geometry failure |
| Empty | Pass | A no-match customer filter exposes the localized table empty state without overflow or overlap |
| Error | Pass after retry | Aborted customer requests expose the localized alert state and recover through the scoped retry action |
| Offline | Pass | Browser-level offline mode exposes the localized network error, remains stable, and recovers after connectivity is restored |
| Destructive | Pass | Qualification expiry scan opens the localized guarded dialog without viewport or text violations |
| Saving | Pass | A delayed real scan request keeps the action loading state stable at desktop and mobile widths |
| Success | Pass | The completed real scan exposes a visible success message and stable geometry |
| No permission | Pass | `dept_admin` is redirected to the explicit forbidden state in both locales and viewports |
| Layout stability | Pass | Every captured state compares page-shell, header, body, state, loading, dialog, and success landmarks over 180ms with <=1px movement |
| Layout containment | Pass | Every captured state enforces <=1px document overflow, zero selected text overflows, zero control viewport escapes, and no header/content overlap |
| Frontend quality gates | Pass | Six Vitest files / 13 tests, i18n/a11y/large-list/theme scans, `vue-tsc`, and Vite production build pass |
| Current-only architecture | Pass | Tests exercise the current routes and real APIs; no demo route, fallback fixture page, compatibility path, or duplicate UI implementation was added |

### Verification Commands

```powershell
cd web
npm run test:states
npm run test:components
npm run typecheck
npm run build
cd ..
codegraph sync .
codegraph status .
git diff --check
```

Result: PR4-05 passed after two documented retries. The current platform and Pharma OA work surfaces now have an executable Chinese/English desktop/mobile Playwright matrix covering responsive, loading, empty, error, offline, destructive, saving, success, and no-permission states with measurable containment and stability gates.

### Commit

`PR4-05: enforce responsive UI state matrix`
