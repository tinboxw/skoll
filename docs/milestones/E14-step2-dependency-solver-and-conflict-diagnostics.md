# E14-step2 Dependency Solver And Conflict Diagnostics

## Milestone

- ID: `E14-step2`
- Name: `dependency solver and conflict diagnostics`

## Delivered

- Added deterministic dependency solver in plugin manager:
  - computes stable resolved set (`name@version`)
  - emits actionable conflict diagnostics for
    - missing dependencies
    - invalid candidate/dependency versions
    - dependency min-version mismatch
    - candidate version collision
- Added admin API:
  - `POST /admin/v1/plugins/dependency-solver/resolve`
- Added audit linkage for solver execution.

## Files

- `internal/module/pluginmgr/service.go`
- `internal/module/pluginmgr/service_test.go`
- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `internal/module/storageadapter/adapter.go`

## Validation

- `go fmt ./...`
- `go test ./internal/module/pluginmgr ./internal/app ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`
