# M3-drain-config-guard

## Basic Info

- Milestone: M3-drain-config-guard
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Add startup-time guardrails for graceful drain-related parameters.
- Prevent accidental misconfiguration that can delay or break shutdown behavior.
- Keep runtime behavior backward compatible for valid configurations.

## Non-Goals

- Shutdown orchestration redesign.
- Endpoint contract changes.
- New external dependencies.

## Delivered

- Added runtime validation in `cmd/skoll/main.go`:
  - `shutdown-timeout` must be `> 0`
  - `drain-time` must be `>= 0`
  - `drain-time` must be `<= 30s`
- Added unit tests in `cmd/skoll/main_test.go` for valid and invalid configurations.
- Updated README.md and README.en.md with parameter constraints.

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
- `BenchmarkHealthEndpoint-16`: `4469 ns/op`, `1094 B/op`, `12 allocs/op` (median of `-count 3`)
- `BenchmarkSystemServiceHealthParallel-16`: `7.777 ns/op`, `0 B/op`, `0 allocs/op` (median of `-count 3`)

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 4132 | 4469 | +8.16% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Acceptable |
| service BenchmarkSystemServiceHealthParallel ns/op | 7.376 | 7.777 | +5.44% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | Acceptable |
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
- Medium: The `30s` upper bound may require adjustment for specific deployment environments with longer drain windows.
- Low: Invalid startup parameters now fail fast at process startup.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
