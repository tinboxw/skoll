# Operations Reproduction Record 2026-07-04

> Work Item: M7-04-04
> Scope: reproduce Quick Start plus Operations Guide on a local Windows development environment.

## Environment

| Item | Value |
|---|---|
| OS shell | PowerShell |
| Repo | `D:\workspace\3rdsrc\tinbox\skoll` |
| Backend mode | `scripts/dev-up.ps1` defaults |
| Backend health endpoint | `http://127.0.0.1:8080/skoll/health` |
| Frontend endpoint | `http://127.0.0.1:5173/skoll/` |

## Reproduction Steps

| Step | Command | Result |
|---|---|---|
| Stop stale services | `powershell -ExecutionPolicy Bypass -File ./scripts/dev-down.ps1` | Passed; stale frontend process on port 5173 was stopped. |
| Start backend/frontend | `powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart` | Passed; backend and frontend started in background. |
| Backend health | `Invoke-RestMethod http://127.0.0.1:8080/skoll/health` | Passed; returned `{"code":"ok","message":"ok"}`. |
| Frontend health | Built-in `dev-up.ps1` post check | Passed; frontend health returned ok. |
| Config/catalog check | Built-in `dev-up.ps1` post check | Passed; dev config ok and permission catalog framework count was 10. |
| Wrong documented port check | `Invoke-RestMethod http://127.0.0.1:18080/skoll/health` | Failed as expected; this exposed stale documentation and triggered a retry after doc correction. |
| Login/audit smoke | `powershell -ExecutionPolicy Bypass -File ./scripts/smoke-auth-audit.ps1` | Failed first because the script default fixture path pointed to a pre-archive location; passed after updating the default fixture path to the archived fixture. |
| Stop services | `powershell -ExecutionPolicy Bypass -File ./scripts/dev-down.ps1` | Passed; backend/frontend local processes stopped. |

## Retry Record

Initial validation failed because `docs/quick-start.md` and `docs/user/operations.md` used `127.0.0.1:18080` for local development health checks while `scripts/dev-up.ps1` starts the backend on `127.0.0.1:8080`.

Fix:

- Updated local development health, Swagger, OpenAPI, and plugin-install examples to use `8080`.
- Kept Docker examples using `18080:8080`, because that is the host-to-container mapping documented for Docker.
- Updated `scripts/smoke-auth-audit.ps1` so its default fixture path points to `docs/archive/refactor-2026-06-19/fixtures/m2_audit_smoke_events.json`.

Retry result: Passed.

## Operations Coverage

| Area | Evidence |
|---|---|
| Start | `dev-up.ps1 -ForceRestart` started backend and frontend. |
| Login | `smoke-auth-audit.ps1` authenticated with `admin` / `Admin@123456`. |
| Stop | `dev-down.ps1` stopped local processes. |
| Logs | `dev-up.ps1` printed backend and frontend log paths under `log/`. |
| Recovery | Stale port recovery was exercised by running `dev-down.ps1` before startup. |
| Backup/restore guidance | `docs/user/operations.md` documents MySQL and PostgreSQL backup/restore commands; not executed locally because no external SQL database was provisioned for this reproduction. |

## Follow-up Boundary

This record proves the local development path and documents the external database boundary. Production backup/restore should be re-run in the target environment with real MySQL or PostgreSQL credentials before release.
