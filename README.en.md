# Skoll

Skoll v2 is a Go framework baseline reimplemented with strict layered architecture, focusing on clear responsibilities, low coupling, high cohesion, testability, and extensibility.

## Architecture Layers

- `cmd/skoll`: process entrypoint
- `internal/bootstrap`: configuration and runtime wiring
- `pkg/config`: configuration loading, validation, and watcher
- `pkg/logging`: structured logging abstraction
- `pkg/errors`: unified error model and HTTP mapping
- `pkg/security`: JWT, password hashing, and encryption utilities

## Storage Layer Notes

- `internal/store/sql/gormrepo/model`: shared model definitions and mapping for MySQL/PostgreSQL (one file per table).
- `internal/store/sql/gormrepo/store`: shared repositories for MySQL/PostgreSQL (split by user/role/system/rbac).
- `internal/store/sql/gormrepo/stores.go`: root facade exports to keep upper-layer calls stable.
- `internal/store/sql/mysql`: MySQL dialect entrypoint (DSN parsing, connection, migration, key normalization).
- `internal/store/sql/postgres`: PostgreSQL dialect entrypoint (connection, migration, key normalization).
- `internal/store/sql/transaction.go`: SQL-mode transaction boundary (`UnitOfWork`).

## Quick Start

Default load precedence: `SKOLL_* environment variables > config file > built-in defaults`.

The loader checks these config files automatically (in order):
- `./skoll.yaml`
- `./skoll.yml`
- `./config/skoll.yaml`
- `./config/skoll.yml`
- `./configs/skoll.yaml`
- `./configs/skoll.yml`

The default profile is `mysql + memory cache` (see `configs/skoll.yaml`).
When `store.mode=mysql`, audit records are persisted in MySQL and survive backend restarts.

Recommended local bootstrap (force MySQL mode and restart both services):

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart
```

`dev-up.ps1` now runs post-start checks by default:
- frontend health check: `/skoll/health`
- login for token acquisition (default account `admin`)
- developer portal endpoint checks: `/skoll/v1/plugins/dev/config`, `/skoll/v1/plugins/dev/permission-catalog`

Common options:

```powershell
# Skip post-start checks
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart -SkipPostChecks

# Fail script with non-zero exit when checks fail (CI/gate use)
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart -FailOnCheckError

# Override account used for post-check login
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart -AdminAccount admin -AdminPassword "Admin@123456"
```

Stop local backend/frontend listeners:

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/dev-down.ps1
```

You can switch between these preset combinations (1/2/3/4):
1. `configs/skoll.mysql-memory.yaml`
2. `configs/skoll.mysql-redis.yaml`
3. `configs/skoll.postgres-memory.yaml`
4. `configs/skoll.postgres-redis.yaml`

1. Config-file mode (recommended for local development)

```powershell
go run ./cmd/skoll
```

2. Specify a config file path

```powershell
$env:SKOLL_CONFIG_FILE="D:/workspace/3rdsrc/tinbox/skoll/configs/skoll.yaml"
go run ./cmd/skoll
```

3. Use profile 1 (mysql + memory cache)

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.mysql-memory.yaml"
go run ./cmd/skoll
```

4. Use profile 2 (mysql + redis cache)

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.mysql-redis.yaml"
go run ./cmd/skoll
```

5. Use profile 3 (postgres + memory cache)

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.postgres-memory.yaml"
go run ./cmd/skoll
```

6. Use profile 4 (postgres + redis cache)

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.postgres-redis.yaml"
go run ./cmd/skoll
```

7. Override any profile via env (example: override Redis address)

```powershell
$env:SKOLL_CONFIG_FILE="./configs/skoll.mysql-redis.yaml"
$env:SKOLL_CACHE_REDIS_ADDR="10.0.0.5:6379"
go run ./cmd/skoll
```

8. Unified logs to a single file (recommended)

```powershell
$env:SKOLL_LOG_DIR="log"
$env:SKOLL_LOG_FILE="skoll.log"
go run ./cmd/skoll
```

When `SKOLL_LOG_FILE` is set, both application logs and plugin operation logs are written to `log/skoll.log`.

9. Default stdout mode (no log file)

```powershell
$env:SKOLL_LOG_FILE=""
$env:SKOLL_LOG_PLUGIN_PER_FILE="false"
go run ./cmd/skoll
```

By default logs go to stdout, and plugin logs include the `plugin_id` field.

10. Per-plugin log files

```powershell
$env:SKOLL_LOG_FILE=""
$env:SKOLL_LOG_PLUGIN_PER_FILE="true"
$env:SKOLL_LOG_DIR="log"
go run ./cmd/skoll
```

Plugin logs are written to `log/<pluginId>.log`.

## Default Login Account

- Built-in admin account: `admin`
- Default password: `Admin@123456`
- The built-in seed password policy requires at least 8 characters, so `admin/admin` is not supported.
- With persistent MySQL data, startup now reconciles built-in accounts to the defaults above (change the password after first login).

## Tests

```powershell
go fmt ./...
go test ./...
go test -race ./...

# Optional: local MySQL integration validation
$env:SKOLL_TEST_MYSQL_DSN="root:root@tcp(127.0.0.1:3306)/skoll?parseTime=true"
go test ./...
```

## Design Docs

- `docs/planning/FRAMEWORK_REDESIGN_BLUEPRINT_2026-05-09.md`
- `docs/planning/FRAMEWORK_REDESIGN_EXECUTION_PLAN_2026-05-09.md`

## API

- `GET /skoll/health`
- `GET /skoll/v1/plugins`
- `POST /skoll/v1/plugins/install` (body: `{ "path": "plugins/demo" }`)
- `POST /skoll/v1/plugins/validate` (body: `{ "path": "plugins/demo" }`)
- `POST /skoll/v1/plugins/{id}/enable`
- `POST /skoll/v1/plugins/{id}/disable`
- `DELETE /skoll/v1/plugins/{id}`
- `GET /skoll/v1/plugins/{id}/debug`
- `GET /skoll/v1/plugins/{id}/logs`
- `POST /skoll/v1/plugins/dev/scaffold` (super_admin only, body: `{ "pluginsRoot": "plugins", "pluginId": "crm-order", "pluginName": "CRM Order", "appId": "crm", "mode": "workspace|repository" }`)
- `POST /skoll/v1/plugins/dev/validate-all` (super_admin only, body: `{ "pluginsRoot": "plugins" }`)
- `GET /skoll/v1/plugins/dev/config` (super_admin only)
- `GET /skoll/v1/plugins/dev/manifest?pluginsRoot=plugins&pluginId=crm-order` (super_admin only)
- `POST /skoll/v1/plugins/dev/manifest/validate` (super_admin only, body: `{ "pluginsRoot": "plugins", "pluginId": "crm-order", "manifest": "...yaml..." }`)
- `PUT /skoll/v1/plugins/dev/manifest` (super_admin only, body: `{ "pluginsRoot": "plugins", "pluginId": "crm-order", "manifest": "...yaml..." }`)
- `POST /skoll/v1/plugins/dev/projects` (super_admin only, body: `{ "pluginsRoot": "plugins" }`)
- `POST /skoll/v1/plugins/dev/remove` (super_admin only, body: `{ "pluginsRoot": "plugins", "pluginId": "crm-order", "removeFiles": true }`)
- `POST /skoll/v1/plugins/dev/package` (super_admin only, body: `{ "pluginsRoot": "plugins", "pluginId": "crm-order", "outputDir": "plugins/_dist" }`)
- `POST /skoll/v1/plugins/dev/pipeline` (super_admin only, body: `{ "pluginsRoot": "plugins", "pluginId": "crm-order", "outputDir": "plugins/_dist" }`)
- `POST /skoll/v1/plugins/dev/rollout` (super_admin only, body: `{ "pluginId": "crm-order", "rolloutPercent": 20 }`)
- `POST /skoll/v1/plugins/dev/rollback` (super_admin only, body: `{ "pluginId": "crm-order" }`)
- `POST /skoll/v1/plugins/dev/release-orders` (super_admin only, body: `{ "pluginsRoot": "plugins", "pluginId": "crm-order", "releaseVersion": "1.2.0", "changelog": "..." }`)
- `GET /skoll/v1/plugins/dev/release-orders?pluginsRoot=plugins&pluginId=crm-order` (super_admin only)
- `POST /skoll/v1/plugins/dev/release-orders/{orderId}/approve` (super_admin only, body: `{ "pluginsRoot": "plugins", "pluginId": "crm-order", "comment": "approved" }`)
- `POST /skoll/v1/plugins/dev/release-orders/{orderId}/reject` (super_admin only, body: `{ "pluginsRoot": "plugins", "pluginId": "crm-order", "comment": "need more checks" }`)

Developer page plugin (installable/removable):
- Plugin directory: `plugins/developer-portal`
- Install: `POST /skoll/v1/plugins/install` with body `{"path":"plugins/developer-portal"}`
- Access: enable it in Plugin Management, then click Open Page for that plugin
- Production recommendation: uninstall this plugin when developer tooling is not required

Dev Portal response contract (`data` payload):
- scaffold: `{ operation, status, pluginsRoot, pluginId, pluginName, pluginDir, mode }`
- validate-all: `{ operation, status, pluginsRoot, summary:{total,valid,invalid}, results:[...] }`
- config: `{ enabled, defaultRoot, allowedRoots[] }`
- manifest_get/manifest_put/manifest_validate: `{ operation, status, pluginsRoot, pluginId, pluginDir, manifestPath, manifest, validation, error? }`
- projects: `{ operation, status, pluginsRoot, projects:[{ pluginId,name,version,path,mode,installed,enabled,previewUrl,buildHint,packageHint,publishHint,lastUpdatedAt }] }`
- remove: `{ operation, status, pluginsRoot, pluginId, pluginDir, uninstalled, filesRemoved }`
- package: `{ operation, status, pluginsRoot, pluginId, pluginDir, artifactPath }`
- pipeline: `{ operation, status, pluginsRoot, pluginId, pluginDir, startedAt, finishedAt, steps:[{name,status,message?,artifactPath?,durationMs}] }`
- rollout/rollback: `{ operation, status, pluginId, rolloutPercent, persisted, message? }`
- release order create/list/approve/reject: `{ operation, status, order:{ orderId, pluginId, releaseVersion, orderStatus, changelog?, createdBy, createdAt, approvedBy?, approvedAt?, rejectedBy?, rejectedAt?, reviewComment? }, orders?[] }`

## Infrastructure Additions

- Unified API base prefix: `SKOLL_API_BASE_PREFIX` (default: `/skoll`, injected to frontend at build time and used by backend at runtime)
- Frontend page base path: `SKOLL_WEB_BASE_PATH` (default: `/skoll`, admin UI is mounted under this path)
- Dev Portal toggle: `SKOLL_DEV_PORTAL_ENABLED` (default: `false`; when enabled, `/v1/plugins/dev/*` routes are registered)
- Dev Portal allowlisted roots: `SKOLL_DEV_PLUGINS_ROOT` (default: `plugins`; supports multiple roots separated by `;` or `,`; request `pluginsRoot` must match one allowlisted root)
- Event bus mode: `SKOLL_EVENT_MODE` (`memory` or `redis`)
- Redis Pub/Sub address: `SKOLL_EVENT_REDIS_ADDR` (required when `SKOLL_EVENT_MODE=redis`)
- Swagger UI: `GET {API_BASE_PREFIX}/docs/swagger`
- OpenAPI document: `GET {API_BASE_PREFIX}/docs/openapi.yaml` (`servers.url` is generated from the current prefix)
