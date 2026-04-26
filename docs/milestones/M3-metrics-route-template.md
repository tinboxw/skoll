# M3-metrics-route-template

## Basic Info

- Milestone: M3-metrics-route-template
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Improve metrics path readability without sacrificing cardinality control.
- Keep unknown-path fallback behavior unchanged.
- Ensure admin routes can be grouped into a bounded template label.

## Non-Goals

- Dynamic route discovery.
- Full route taxonomy system.
- Changes to existing endpoint contracts.

## Delivered

- Added metrics path template policy in `internal/app/metrics.go`:
  - `/admin/*` -> `/admin/:path`
- Kept exact whitelist matching precedence (e.g. `/admin/ping` remains exact).
- Extended path counter buckets with explicit admin template bucket.
- Added tests for template normalization and template-bucket counting.

## Validation Evidence

Commands:

- go fmt ./...
- go test ./...
- go test -race ./...
- go test -run ^$ -bench "." -benchmem -count 3 ./internal/app
- go test -run ^$ -bench "." -benchmem -count 3 ./internal/service

Key output summary:

- `go test ./...`: PASS
- `go test -race ./...`: PASS
- app benchmark median (`count 3`): `874.1 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 3`): `2.454 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 953.3 | 874.1 | -8.31% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint B/op | 1095 | 1094 | -0.09% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 6.751 | 2.454 | -63.65% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
| service BenchmarkSystemServiceHealthParallel B/op | 0 | 0 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
| service BenchmarkSystemServiceHealthParallel allocs/op | 0 | 0 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |

## DoD Review (5 points)

- [x] Code compiles and tests pass
- [x] Package boundaries are clear
- [x] Runnable entry/example exists
- [x] Documentation is updated
- [x] Risks/assumptions are documented

Score: 5/5

## Regression Risks

- High: None identified.
- Medium: Template policy is currently prefix-based; new admin sub-route naming should follow stable conventions.
- Low: Exact whitelist has higher priority than template rules by design.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
