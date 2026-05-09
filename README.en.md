# Skoll

Skoll v2 is a Go framework baseline reimplemented with strict layered architecture, focusing on clear responsibilities, low coupling, high cohesion, testability, and extensibility.

## Architecture Layers

- `cmd/skoll`: process entrypoint
- `internal/bootstrap`: configuration and runtime wiring
- `pkg/config`: configuration loading, validation, and watcher
- `pkg/logging`: structured logging abstraction
- `pkg/errors`: unified error model and HTTP mapping
- `pkg/security`: JWT, password hashing, and encryption utilities

## Quick Start

1. Memory mode (default)

```powershell
go run ./cmd/skoll
```

2. MySQL mode

```powershell
$env:SKOLL_STORE_MODE="mysql"
$env:SKOLL_STORE_DSN="user:pass@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local"
go run ./cmd/skoll
```

3. PostgreSQL mode

```powershell
$env:SKOLL_STORE_MODE="postgres"
$env:SKOLL_STORE_DSN="host=127.0.0.1 user=postgres password=postgres dbname=skoll port=5432 sslmode=disable"
go run ./cmd/skoll
```

## Tests

```powershell
go fmt ./...
go test ./...
go test -race ./...
```

## Design Docs

- `docs/planning/FRAMEWORK_REDESIGN_BLUEPRINT_2026-05-09.md`
- `docs/planning/FRAMEWORK_REDESIGN_EXECUTION_PLAN_2026-05-09.md`

## API

- `GET /health`
