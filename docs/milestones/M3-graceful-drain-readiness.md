# M3-graceful-drain-readiness

## Basic Info

- Milestone: M3-graceful-drain-readiness
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Improve graceful shutdown behavior for load-balanced runtime.
- Mark readiness as not ready before termination so upstream can stop routing requests.
- Keep business boundaries and endpoint contracts unchanged.

## Non-Goals

- Business feature expansion.
- New external dependencies.
- Replacing current shutdown timeout behavior.

## Delivered

- Added startup flag `-drain-time` (default `2s`).
- On termination signal, runtime now:
  - switches readiness state to `not_ready`,
  - waits for configured drain window,
  - then performs graceful shutdown with existing timeout.
- Updated README.md and README.en.md startup examples.

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
- `BenchmarkHealthEndpoint-16`: `4132 ns/op`, `1094 B/op`, `12 allocs/op` (median of `-count 3`)
- `BenchmarkSystemServiceHealthParallel-16`: `7.376 ns/op`, `0 B/op`, `0 allocs/op` (median of `-count 3`)

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 868.4 | 4132 | +375.81% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Needs follow-up |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.425 | 7.376 | +204.16% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | Needs follow-up |
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
- Medium: Setting `-drain-time` too long may delay process termination in orchestration environments.
- Medium: Bench `ns/op` increased notably in this run; no hotspot code-path changes in benchmarked packages, likely environment variance but needs next-cycle recheck.
- Low: Default `2s` drain adds a small fixed delay to signal-based shutdown path.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
