# M2 Audit Write Path Inventory

- Work Item: M2-03-01
- Date: 2026-06-19
- Scope: `internal/handler`, `internal/service`, and bootstrap audit entry points that currently write or expose audit records.

## Current Write Entrypoints

| Area | File | Current action/resource shape | Current sink | Replacement target |
|---|---|---|---|---|
| Auth login/logout | `internal/bootstrap/auth_builtin_handler.go:53` `:60` `:71` `:101` `:117` `:138` | `login_failed`, `login`, `logout` on `auth` | `audit.Service.Append` old `Record` | M2-03-03 should write `EventTypeLogin` with `LoginLog` source data and explicit success/failure result. |
| Plugin catalog lifecycle | `internal/bootstrap/di.go:180` `:187` | plugin catalog event action on `plugin` | `audit.Service.Append` old `Record` | M2-03-02 or plugin lifecycle follow-up should route through unified event writer with plugin source data. |
| User service create/disable | `internal/service/user/service_impl.go:73` `:85` `:177` `:189` `:292` `:294` | `create`, `disable` on `user` | direct `auditrepo.AuditRepository.Append` | Replace direct repository writes with service-level event writer, or remove once handler/middleware writes authoritative events. |
| User HTTP operations | `internal/handler/http/v1/user/handler.go:58` `:95` `:147` `:185` `:195` `:210` `:249` `:253` `:261` | `create`, `batch_create`, `update_email`, `update`, `delete`, `disable`, `assign_role` on `user` | per-handler `appendAudit` helper | Migrate to shared request audit/event helper; normalize to catalog actions like `user.user.create`. |
| Role HTTP operations | `internal/handler/http/v1/role/handler.go:51` `:165` `:175` `:192` `:209` `:213` `:224` | `create`, `update`, `delete`, `grant`, `revoke` on `role` | per-handler `appendAudit` helper | Migrate to shared request audit/event helper; preserve before/after source data for update. |
| RBAC HTTP operations | `internal/handler/http/v1/rbac/handler.go:53` `:69` `:118` `:122` `:133` | `bind`, `set_policies`, `unbind` on `rbac` | per-handler `appendAudit` helper | Migrate to shared request audit/event helper; classify role binding/policy writes as operation events. |
| System HTTP operations | `internal/handler/http/v1/system/handler.go:180` `:190` `:233` `:295` `:384` `:722` `:733` | `upsert`, `reset`, `upsert_menus`, `upsert_dictionaries`, `upsert_departments`, `upsert_positions` on system resources | per-handler `appendAudit` helper | Migrate to shared request audit/event helper; keep setting/list deltas in source data. |
| Plugin HTTP operations | `internal/handler/http/v1/plugin/handler.go:720` `:727` `:738` `:1126` `:1260` `:1406` `:1482` `:1540` `:1597` `:1725` plus dev release files | `update_config`, `dev_*`, release order/task/rollout actions on plugin resources | per-handler `appendAudit` helper | Migrate plugin actions to catalog names and unified event writer; keep plugin operation payload in source data. |

## Current Read/Export Entrypoints To Replace Later

| Area | File | Current behavior | Replacement target |
|---|---|---|---|
| Audit HTTP API | `internal/handler/http/v1/audit/handler.go:75` `:97` `:124` `:148` `:162` | lists/exports/clears legacy `audit.Record`; filters are mostly in handler memory after range fetch | M2-04 should use `audit.EventService` list/detail/export and remove in-handler legacy record filtering. |
| Audit service legacy writer | `internal/service/audit/service_impl.go:19` `:29` `:34` | creates old `audit.Record` and appends to old repository | Keep only until all writers move to event model; then retire or wrap around event writer by explicit migration task. |

## Middleware Findings

- `internal/handler/middleware/auth*.go` currently handles authentication and bypass policy only; no audit event is written from middleware.
- Permission denial is currently returned by auth/permission flow without a unified forbidden audit event. M2-03-04 should add a denial path that captures actor, method, route/resource, action, trace/request id, and result `denied`.
- Panic/handler error capture is not currently wired to audit/error logs in handler middleware. M2-03-05 should add error event capture with trace context and `ErrorLog` source data.

## Replacement Checklist

1. Add one shared event writer for HTTP request operations that accepts actor, action, resource, resource id, result, risk, metadata, source data, and trace context.
2. Use the shared writer from the upcoming request audit middleware for success/failure cases instead of cloning `appendAudit` helpers per handler.
3. Move login success/failure/logout to `EventTypeLogin` and attach `LoginLog` source data.
4. Move permission denials to `EventTypeSecurity` or operation-denied events, using action catalog names instead of bare verbs.
5. Move panic/handler errors to `EventTypeError` and attach `ErrorLog` source data.
6. Remove direct `auditrepo.AuditRepository` writes from `internal/service/user` or make them call the unified event writer only where service-level audit is intentionally required.
7. Replace legacy audit list/export HTTP implementation with `audit.EventService` after event writes are populated.

## Validation Command

```powershell
rg -n "audit|operation|login" internal/handler internal/service
rg -n "h\.appendAudit\(|appendAuthAudit\(|audit\.Append\(|auditSvc\.Append\(|audit\.NewRecord\(" internal/handler internal/service internal/bootstrap -S
```
