# M3-go-admin-minimal-integration

## Basic Info

- Milestone: M3-go-admin-minimal-integration
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Enter go-admin introduction phase with minimal, optional integration.
- Keep existing app/domain/service boundaries unchanged.
- Add metrics path-cardinality protection to prevent long-term label growth.

## Non-Goals

- Replacing current HTTP/service architecture.
- Introducing go-admin business modules into current runtime path.
- Auth/RBAC or admin panel feature rollout.

## Delivered

- Added isolated go-admin bootstrap adapter in `internal/integration/goadmin`.
- Added startup flags:
  - `-go-admin-enabled` (default `false`)
  - `-go-admin-mode` (`dev|test|prod`)
- Kept default behavior unchanged when adapter is disabled.
- Added metrics path whitelist normalization with fallback label `__other__`.
- Added tests for bootstrap adapter and metrics normalization.

## Validation Evidence

Commands:

- go fmt ./...
- go test ./...
- go test -race ./...
- go test -bench "." -benchmem ./internal/app
- go test -bench "." -benchmem ./internal/service

Key output summary:

- go test ./...: pass
- go test -race ./...: pass
- benchmark snapshot:
  - internal/app BenchmarkHealthEndpoint-16: 953.9 ns/op, 1094 B/op, 12 allocs/op
  - internal/service BenchmarkSystemServiceHealthParallel-16: 2.748 ns/op, 0 B/op, 0 allocs/op

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 883.6 | 953.9 | +70.3 (+7.96%) | go test -bench "." -benchmem ./internal/app | Pass |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0 (0%) | go test -bench "." -benchmem ./internal/app | Pass |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0 (0%) | go test -bench "." -benchmem ./internal/app | Pass |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.376 | 2.748 | +0.372 (+15.66%) | go test -bench "." -benchmem ./internal/service | Pass |
| service BenchmarkSystemServiceHealthParallel B/op | 0 | 0 | 0 (0%) | go test -bench "." -benchmem ./internal/service | Pass |
| service BenchmarkSystemServiceHealthParallel allocs/op | 0 | 0 | 0 (0%) | go test -bench "." -benchmem ./internal/service | Pass |

## DoD Review (5 points)

- [x] Code compiles and tests pass
- [x] Package boundaries are clear
- [x] Runnable entry/example exists
- [x] Documentation is updated
- [x] Risks/assumptions are documented

Score: 5/5

## Regression Risks

- High: None identified.
- Medium: go-admin dependency tree is large; should stay optional until deeper integration stage.
- Low: unknown routes now aggregate into `__other__`, reducing per-path visibility by design.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
