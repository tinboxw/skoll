# M3-runtime-env-overrides

## Basic Info

- Milestone: M3-runtime-env-overrides
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Add environment-level runtime configuration for shutdown/drain timing.
- Keep CLI flags as primary explicit controls while supporting deployment defaults.
- Preserve safe startup behavior via strict parsing and validation.

## Non-Goals

- Runtime hot reload of configuration.
- New business or domain capabilities.
- Orchestrator-specific adapters.

## Delivered

- Added environment fallback parsing in `cmd/skoll/main.go`:
  - `SKOLL_SHUTDOWN_TIMEOUT`
  - `SKOLL_DRAIN_TIME`
- Updated flag descriptions to expose related env keys.
- Added `durationFromEnv` helper with explicit parse error context.
- Added unit tests for env parsing fallback/valid/invalid scenarios.

## Validation Evidence

Commands:

- go fmt ./...
- go test ./...
- go test -race ./...
- go test -run ^$ -bench "." -benchmem -count 3 ./internal/app
- go test -run ^$ -bench "." -benchmem -count 3 ./internal/service

Key output summary:

- `go test ./...`: PASS (all packages green)
- `go test -race ./...`: PASS (no race reported)
- `cmd/skoll/main_test.go`: PASS
- `BenchmarkHealthEndpoint-16`: `3967 ns/op`, `1094 B/op`, `12 allocs/op` (median of `-count 3`)
- `BenchmarkSystemServiceHealthParallel-16`: `6.855 ns/op`, `0 B/op`, `0 allocs/op` (median of `-count 3`)

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 4469 | 3967 | -11.23% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| service BenchmarkSystemServiceHealthParallel ns/op | 7.777 | 6.855 | -11.86% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | Acceptable |
| service BenchmarkSystemServiceHealthParallel B/op | 0 | 0 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | Acceptable |
| service BenchmarkSystemServiceHealthParallel allocs/op | 0 | 0 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | Acceptable |

## DoD Review (5 points)

- [x] Code compiles and tests pass
- [x] Package boundaries are clear
- [x] Runnable entry/example exists
- [x] Documentation is updated
- [x] Risks/assumptions are documented

Score: 5/5

## Regression Risks

- High: None identified.
- Medium: Invalid env duration now fails fast at startup (intended strict behavior).
- Low: Runtime behavior unchanged when env keys are not set.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
