# ADMIN Subpackage Object Migration Plan

## Scope
- Target: `internal/app` admin control-plane routing and handler implementation.
- Goal: migrate from monolithic package-level handler functions to domain subpackages with object-style invocation.
- Non-goal: protocol redesign in the same iteration.

## Background
- Current state is multi-file under one package (`internal/app`), which reduced file size pressure but still leaves high coupling in route assembly and DTO ownership.
- Desired state is domain isolation with stable façade at root app package.

## Mandatory Rules (Execution Constraints)
- Keep external API compatibility during migration:
  - Do not change route path/method.
  - Do not change request/response JSON field names or semantic behavior.
- One-domain-per-change policy:
  - A single migration step should focus on one domain package to reduce blast radius.
- Structural refactor only by default:
  - Feature changes must be separated into follow-up commits.
- Gate per step:
  - `go fmt ./...`
  - `go test ./internal/app`
  - `go test ./...`
  - `go test -race ./...` when concurrency-sensitive logic is touched.

## Target Architecture
- Root façade (`internal/app`):
  - Keep `MountAdminModuleRoutes(...)` as compatibility entry.
  - Delegate domain registrations to subpackage handlers.
- Domain package template:
  - `internal/app/admin/<domain>/handler.go`
  - `type Handler struct { deps ... }`
  - `func NewHandler(...) *Handler`
  - `func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry)`
- Shared utilities:
  - Only generic helpers (respond/json, path parser, pagination parser) may be centralized.
  - Domain DTOs stay in domain package unless reuse is proven and stable.

## Migration Order
1. Jobs domain (M1) - completed baseline sample
2. Auth and session domain (M2)
3. RBAC and policy governance domain (M3)
4. Config and dictionary domain (M4)
5. System dashboard domain (M5)
6. Plugin governance domain (M6)
7. Database ops governance domain (M7)
8. Hardening and release-governance routes cleanup (M8)
9. Shared utility extraction and final façade cleanup (M9)

## Detailed Steps

### M1 Jobs (Done)
- Status: completed.
- Deliverables:
  - `internal/app/admin/jobs/handler.go`
  - `MountAdminModuleRoutes` delegates jobs routes to object handler.
- Verification:
  - `go fmt ./...`
  - `go test ./internal/app`
  - `go test ./...`

### M2 Auth and Session (Done)
- Status: completed.
- Deliverables:
  - `internal/app/admin/auth/handler.go`
  - `MountAdminModuleRoutes` delegates auth/session routes to object handler.
  - legacy auth/session implementation file removed.
- Verification:
  - `go fmt ./...`
  - `go test ./internal/app`
  - `go test ./...`
  - `go test -race ./...`

### M3 RBAC and Policy (Done)
- Status: completed.
- Deliverables:
  - `internal/app/admin/rbac/handler.go`
  - `MountAdminModuleRoutes` delegates RBAC/policy routes to object handler.
  - legacy RBAC implementation file removed.
- Verification:
  - `go fmt ./...`
  - `go test ./internal/app`
  - `go test ./...`
  - `go test -race ./...`

### M4 Config and Dictionary (Done)
- Status: completed.
- Deliverables:
  - `internal/app/admin/configdict/handler.go`
  - `MountAdminModuleRoutes` delegates config/dictionary routes to object handler.
  - legacy config/dictionary implementation file removed.
- Verification:
  - `go fmt ./...`
  - `go test ./internal/app`
  - `go test ./...`
  - `go test -race ./...`

### M5 System Dashboard (Done)
- Status: completed.
- Deliverables:
  - `internal/app/admin/dashboard/handler.go`
  - `internal/app/admin/dashboard/handler_test.go`
  - `MountAdminModuleRoutes` delegates system dashboard routes to object handler.
- Verification:
  - `go fmt ./...`
  - `go test ./internal/app/admin/dashboard`
  - `go test ./internal/app`
  - `go test ./...`
  - `go test -race ./...`

### M6 Plugins (Done)
- Status: completed.
- Deliverables:
  - `internal/app/admin/plugins/handler.go`
  - `MountAdminModuleRoutes` delegates all 25 plugin routes to object handler.
  - `internal/app/admin_plugins.go` deleted.
- Verification:
  - `go fmt ./...`
  - `go test ./internal/app`
  - `go test ./...`
  - `go test -race ./...`

### M7 Database Ops (Done)
- Status: completed.
- Deliverables:
  - `internal/app/admin/dbops/handler.go`
  - `MountAdminModuleRoutes` delegates all 8 db-ops routes to object handler.
  - `internal/app/admin_db_ops.go` deleted.
- Verification:
  - `go fmt ./...`
  - `go test ./internal/app`
  - `go test ./...`
  - `go test -race ./...`

### M8 Hardening and Release Governance — **Done**
- Created `internal/app/admin/hardening/handler.go` (`adminhardening` package).
  - `Handler` struct owns `hardeningState` (profiles, alertProfiles, runbooks, drills, nextDrillID).
  - `NewHandler(auditSvc)` initialises maps and nextDrillID=1.
  - `Register(mux, wrapper, apis)` mounts 8 routes (guardrails PUT/GET, alert-profiles PUT/GET, runbooks PUT/GET, fault-drills POST/GET).
  - `Snapshot() PostureSnapshot` replaces `collectDashboardHardeningPosture()`.
- Created `internal/app/admin/releasegov/handler.go` (`releasegov` package).
  - `Handler` struct owns `releaseClosureState` (checkpoints, policy).
  - `NewHandler(svc, auditSvc)` sets default blocking policy.
  - `Register(mux, wrapper, apis)` guards on nil svc, mounts 8 routes.
- Updated `admin_modules.go`:
  - Added imports for `adminhardening` and `adminreleasegov`.
  - `hardeningH` created before dashboard callback; callback uses `hardeningH.Snapshot()`.
  - `dashboardAggregateResponse.HardeningPosture` type changed to `adminhardening.PostureSnapshot`.
  - Removed `dashboardHardeningPosture` struct definition.
  - Replaced 8 direct hardening `handle(...)` calls with `hardeningH.Register(...)`.
  - Replaced `if services.Releases != nil { ... }` block with `adminreleasegov.NewHandler(...).Register(...)`.
- Deleted `admin_hardening.go` and `admin_release_governance.go`.
- Gates: `go build ./...` clean; `go test -race ./...` all green.

### M9 Final Cleanup — **Done**
- Created 5 new subpackages for remaining domain routes:
  - `internal/app/admin/menu/handler.go` (`adminmenu`) — 3 menu routes.
  - `internal/app/admin/auditlog/handler.go` (`adminauditlog`) — 4 audit-log routes, local `parseAuditQuery`/`auditQueryMaxSize`.
  - `internal/app/admin/files/handler.go` (`adminfiles`) — 4 file routes with download support.
  - `internal/app/admin/generator/handler.go` (`admingenerator`) — 1 module-generation route.
  - `internal/app/admin/apiregistry/handler.go` (`adminapiregistry`) — 1 API-listing route (2-param Register, no apis arg).
- Updated `admin_modules.go`:
  - Added 5 new subpackage imports.
  - Replaced all remaining inline handler closures with subpackage `Register()` calls.
  - Removed 13 old handler functions (createMenuHandler → listRegisteredAPIsHandler).
  - Removed orphaned DTOs: `createMenuRequest`, `appendAuditLogRequest`, `auditQueryProfileResponse`, `adminOpsControlProfileResponse`, `generateModuleRequest`, `listAPIsResponse`.
  - Removed orphaned helpers: `parseAuditQuery`, `auditQueryMaxSize` (now live in `adminauditlog`).
  - Removed unused stdlib imports: `encoding/json`, `io`, `path/filepath`.
- Gates: `go fmt ./...` clean; `go build ./...` clean; `go test ./internal/app/...` green; `go test -race ./...` all green.

### M10 Layered Migration (In Progress)
- Status: in progress.
- Objective: finish layering for remaining admin-related code still under `internal/app` root.
- Layering plan:
  - P1 contracts layer:
    - create `internal/app/admin/contracts` and migrate `AdminModuleServices` plus service interfaces out of root façade.
    - keep root compatibility via type alias to avoid behavior change.
  - P2 security/middleware layer:
    - move JWT claims adapter and RBAC API authorizer into `internal/app/admin/security` (or `middleware`) with stable exported entry points.
  - P3 dashboard bootstrap layer:
    - move dashboard aggregate/bootstrap structs and collectors into `internal/app/admin/dashboard` subfiles.
    - keep `MountAdminModuleRoutes` as composition-only registration.
  - P4 residual cleanup:
    - remove orphan helper/DTO definitions from root if no longer referenced.
    - keep only façade + wiring in root.
- Acceptance gates per phase:
  - `go fmt ./...`
  - `go build ./...`
  - `go test ./internal/app/...`
  - `go test -race ./...`
- Current progress:
  - P1 completed:
    - created `internal/app/admin/contracts/contracts.go`.
    - `internal/app/admin_modules.go` now consumes contracts via compatibility type aliases.
    - gates green: `go fmt ./...`, `go build ./...`, `go test ./internal/app/...`, `go test -race ./...`.
  - P2 completed:
    - created `internal/app/admin/security/claims.go` and `internal/app/admin/security/authorizer.go`.
    - root compatibility wrappers retained in `internal/app/admin_jwt_claims_adapter.go` and `internal/app/admin_rbac_authorizer.go`.
    - external symbols unchanged (`WithRoleAPIAuthorizer`, header constants, claim resolver behavior).
    - gates green: `go fmt ./...`, `go build ./...`, `go test ./internal/app/...`, `go test -race ./...`.
  - P3 in progress (phase 1 complete):
    - created `internal/app/admin/dashboard/types.go` as dashboard DTO owner.
    - moved aggregate/bootstrap DTO ownership from root by introducing compatibility aliases in `internal/app/admin_modules.go`.
    - removed duplicated dashboard DTO declarations from root files and kept package-level compatibility.
    - gates green: `go fmt ./...`, `go build ./...`, `go test ./internal/app/...`, `go test -race ./...`.
  - Migration mode update (direct refactor):
    - compatibility bridge files removed from root app package:
      - `internal/app/admin_jwt_claims_adapter.go`
      - `internal/app/admin_rbac_authorizer.go`
      - `internal/app/admin_jwt_claims_adapter_test.go`
    - call sites switched to direct package usage:
      - bootstrap and tests now consume `internal/app/admin/security` directly.
      - service wiring now consumes `internal/app/admin/contracts` directly.
    - no historical compatibility file retained for the removed paths.

## Deliverable Checklist per Milestone
- Domain subpackage exists and compiles.
- Legacy route patterns are unchanged.
- Legacy DTO JSON tags are unchanged.
- Root route assembly delegates to domain object register.
- Old duplicated handlers removed.
- Gates executed and recorded.

## Risk and Rollback
- Risk: accidental API drift in route path or JSON tags.
  - Mitigation: copy DTO fields/tags verbatim before any refactor optimization.
- Risk: hidden coupling to root helper functions.
  - Mitigation: introduce local helper first, then consolidate shared helpers in M9.
- Rollback approach:
  - Revert one milestone commit only.
  - Keep each milestone commit focused on one domain.

## Tracking and Reporting Format
- Milestone report format:
  - Milestone: `M<idx>-<short-title>`
  - Changes: domain package + façade wiring + legacy removal
  - Validation commands and outcomes
  - Benchmark note (`-benchmem`) if performance-sensitive
  - README parity status

## Immediate Next Action
- Continue M10-P3 phase 2: migrate dashboard collector logic (auth/jwt/bootstrap aggregation functions) from root `internal/app` into `internal/app/admin/dashboard` without introducing compatibility bridge files.
