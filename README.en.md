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

This repository already includes a starter config: `configs/skoll.yaml`.

1. Config-file mode (recommended for local development)

```powershell
go run ./cmd/skoll
```

2. Specify a config file path

```powershell
$env:SKOLL_CONFIG_FILE="D:/workspace/3rdsrc/tinbox/skoll/configs/skoll.yaml"
go run ./cmd/skoll
```

3. Memory mode (env overrides config)

```powershell
go run ./cmd/skoll
```

4. MySQL mode

```powershell
$env:SKOLL_STORE_MODE="mysql"
$env:SKOLL_STORE_DSN="user:pass@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local"
go run ./cmd/skoll
```

5. PostgreSQL mode

```powershell
$env:SKOLL_STORE_MODE="postgres"
$env:SKOLL_STORE_DSN="host=127.0.0.1 user=postgres password=postgres dbname=skoll port=5432 sslmode=disable"
go run ./cmd/skoll
```

6. Redis Event Bus (optional, distributed events)

```powershell
$env:SKOLL_EVENT_MODE="redis"
$env:SKOLL_EVENT_REDIS_ADDR="127.0.0.1:6379"
# Optional: defaults to skoll.events
$env:SKOLL_EVENT_CHANNEL_PREFIX="skoll.events"
go run ./cmd/skoll
```

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
