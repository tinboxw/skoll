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

- `GET /health`
- `GET /v1/plugins`
- `POST /v1/plugins/install` (body: `{ "path": "plugins/demo" }`)
- `POST /v1/plugins/validate` (body: `{ "path": "plugins/demo" }`)
- `POST /v1/plugins/{id}/enable`
- `POST /v1/plugins/{id}/disable`
- `DELETE /v1/plugins/{id}`
- `GET /v1/plugins/{id}/debug`
- `GET /v1/plugins/{id}/logs`
