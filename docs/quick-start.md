# Skoll Quick Start

> Goal: start Skoll, log in, and complete the core admin flows in about 30 minutes on a fresh developer machine.

## 1. Prerequisites

| Tool | Recommended | Notes |
|---|---|---|
| Go | 1.24.x | `go.mod` declares Go 1.24 and toolchain 1.24.1. |
| Node.js | 20.x | CI uses Node 20. |
| npm | bundled with Node | Use `npm ci` for reproducible frontend installs. |
| PowerShell | 7.x or Windows PowerShell | Local helper scripts are PowerShell scripts. |

## 2. Clone and Install

```powershell
git clone <repo-url> skoll
cd skoll
go mod download
cd web
npm ci
cd ..
```

## 3. Start Backend and Frontend

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart
```

Stop local services:

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/dev-down.ps1
```

## 4. Verify Health and Docs

```powershell
Invoke-RestMethod http://127.0.0.1:8080/skoll/health
```

- Swagger UI: `http://127.0.0.1:8080/skoll/docs/swagger`
- OpenAPI YAML: `http://127.0.0.1:8080/skoll/docs/openapi.yaml`
- Frontend: `http://127.0.0.1:5173/skoll`

## 5. Log In

| Field | Value |
|---|---|
| Account | `admin` |
| Password | `Admin@123456` |

Change the password after first login in any persistent environment.

## 6. Core Flow Checklist

| Flow | Expected Result |
|---|---|
| Dashboard | Main admin shell opens without route guard loops. |
| Users | User list loads with pagination controls visible. |
| Roles | Role list opens; role edit and grant paths are reachable for admin. |
| Permissions/Menu | Permission catalog and menu tree load. |
| Plugins | Plugin list opens; installed plugin state and risk/config panels are visible. |
| Audit | Audit query page opens and filter/export controls are reachable. |
| Settings | Settings page opens and setting reset/update controls are visible to admin. |

## 7. Plugin Flow

Install a local plugin through the API:

```powershell
$body = @{ path = "plugins/demo" } | ConvertTo-Json
Invoke-RestMethod `
  -Method Post `
  -Uri http://127.0.0.1:8080/skoll/v1/plugins/install `
  -ContentType "application/json" `
  -Body $body
```

Then use the plugin page to confirm the plugin appears, enable or disable it, and review risk/config/log details.

## 8. Generated Module Smoke

```powershell
go test ./...
cd web
npm run typecheck
npm run build
```

Expected result:

- Go tests pass.
- Frontend typecheck passes.
- Frontend build passes. Existing Sass legacy-js-api and Rollup PURE annotation warnings are known baseline warnings.

## 9. Common Blockers

| Symptom | Check |
|---|---|
| Backend port unavailable | Stop stale services with `scripts/dev-down.ps1`, then restart. |
| Login fails | Confirm seeded account `admin` / `Admin@123456`; check backend logs. |
| Frontend redirects to login | Confirm backend is running and API prefix is `/skoll`. |
| Plugin install fails | Validate `plugin.yaml` and confirm the path is under an allowed plugin root. |

## 10. Next Reading

- Architecture overview: `docs/architecture/README.md`
- API overview: `docs/api/README.md`
- Plugin development: `docs/development/plugin-guide.md`
- Operations and deployment: `docs/user/deployment.md`
- Current task board: `docs/refactor/task_board.md`
