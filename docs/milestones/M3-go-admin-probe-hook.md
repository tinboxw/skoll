# M3-go-admin-probe-hook

## Basic Info

- Milestone: M3-go-admin-probe-hook
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Continue go-admin introduction with a minimal runtime hook.
- Keep existing app/domain/service boundaries unchanged.
- Preserve metrics path cardinality protection while adding integration observability.

## Non-Goals

- Business module migration to go-admin.
- Auth/RBAC rollout.
- Replacing existing health/readiness/metrics endpoints.

## Delivered

- Added optional admin probe route `/admin/ping` mounted only when `-go-admin-enabled=true`.
- Added go-admin probe response with integration mode and request id.
- Added request id header using go-admin SDK constant `TrafficKey`.
- Added server extension point `HandleFunc` to mount optional integration routes.
- Added `/admin/ping` into metrics path whitelist to avoid cardinality drift.
- Added tests for go-admin probe handler and metrics whitelist behavior.
- Refactored metrics hot path to lock-free atomic counters (path whitelist and status buckets) to avoid request-path contention.

## Validation Evidence

Commands:

- go fmt ./...
- go test ./...
- go test -race ./...
- go test -bench "." -benchmem ./internal/app
- go test -bench "." -benchmem ./internal/service

Key output summary:

- `go test ./...`: PASS (all packages green)
- `go test -race ./...`: PASS (no race reported)
- `BenchmarkHealthEndpoint-16`: `868.4 ns/op`, `1094 B/op`, `12 allocs/op`
- `BenchmarkSystemServiceHealthParallel-16`: `2.425 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 953.9 | 868.4 | -8.96% | go test -bench "." -benchmem ./internal/app | Acceptable |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -bench "." -benchmem ./internal/app | Acceptable |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -bench "." -benchmem ./internal/app | Acceptable |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.748 | 2.425 | -11.75% | go test -bench "." -benchmem ./internal/service | Acceptable |
| service BenchmarkSystemServiceHealthParallel B/op | 0 | 0 | 0.00% | go test -bench "." -benchmem ./internal/service | Acceptable |
| service BenchmarkSystemServiceHealthParallel allocs/op | 0 | 0 | 0.00% | go test -bench "." -benchmem ./internal/service | Acceptable |

## DoD Review (5 points)

- [x] Code compiles and tests pass
- [x] Package boundaries are clear
- [x] Runnable entry/example exists
- [x] Documentation is updated
- [x] Risks/assumptions are documented

Score: 5/5

## Regression Risks

- High: None identified.
- Medium: probe route should stay optional; enabling in production requires explicit policy.
- Low: admin probe adds one extra known path label in metrics.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
