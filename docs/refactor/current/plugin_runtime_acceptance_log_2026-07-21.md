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
