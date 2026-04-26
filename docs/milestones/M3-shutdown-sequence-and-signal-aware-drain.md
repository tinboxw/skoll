# M3-shutdown-sequence-and-signal-aware-drain

## Basic Info

- Milestone: M3-shutdown-sequence-and-signal-aware-drain
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Make graceful shutdown sequence explicitly testable.
- Support interrupting drain wait with a second termination signal.
- Improve operability with startup runtime configuration snapshot logging.

## Non-Goals

- Redesigning transport/service architecture.
- Introducing external orchestrator dependencies.
- Changing endpoint contracts.

## Delivered

- Refactored shutdown path in `cmd/skoll/main.go` into testable helper:
  - `performGracefulStop`
  - `waitForDrain`
- Signal handling changed to explicit signal channel:
  - first signal starts graceful sequence,
  - second signal can interrupt drain wait.
- Added startup runtime configuration snapshot log via `formatRuntimeConfigLog`.
- Added unit tests for:
  - shutdown call order,
  - second-signal drain interruption,
  - runtime config snapshot format.

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
- `BenchmarkHealthEndpoint-16`: `953.3 ns/op`, `1095 B/op`, `12 allocs/op` (median of `-count 3`)
- `BenchmarkSystemServiceHealthParallel-16`: `6.751 ns/op`, `0 B/op`, `0 allocs/op` (median of `-count 3`)

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 3967 | 953.3 | -75.97% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| app BenchmarkHealthEndpoint B/op | 1094 | 1095 | +0.09% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| service BenchmarkSystemServiceHealthParallel ns/op | 6.855 | 6.751 | -1.52% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | Acceptable |
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
- Medium: second signal now short-circuits drain window by design.
- Low: startup config snapshot adds one log line at startup only.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
