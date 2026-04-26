# M3-observability-and-hardening

## Basic Info

- Milestone: M3-observability-and-hardening
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Add minimal observability surface for runtime traffic visibility.
- Harden HTTP runtime defaults for safer production behavior.
- Keep the service simple and backward compatible for existing probes.

## Non-Goals

- Full tracing pipeline integration.
- External metrics backends.
- AuthN/AuthZ and business feature expansion.

## Delivered

- Added HTTP metrics endpoint: `/metrics` in Prometheus text format.
- Added request counters by total, path, and status code.
- Added HTTP middleware instrumentation for request observation.
- Added server hardening parameters: read/write/idle timeouts and max header bytes.
- Added metrics endpoint tests with counter assertions.

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
  - internal/app BenchmarkHealthEndpoint-16: 883.6 ns/op, 1094 B/op, 12 allocs/op
  - internal/service BenchmarkSystemServiceHealthParallel-16: 2.376 ns/op, 0 B/op, 0 allocs/op

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 858.8 | 883.6 | +24.8 (+2.89%) | go test -bench "." -benchmem ./internal/app | Pass |
| app BenchmarkHealthEndpoint B/op | 1070 | 1094 | +24 (+2.24%) | go test -bench "." -benchmem ./internal/app | Pass |
| app BenchmarkHealthEndpoint allocs/op | 11 | 12 | +1 (+9.09%) | go test -bench "." -benchmem ./internal/app | Pass |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.567 | 2.376 | -0.191 (-7.44%) | go test -bench "." -benchmem ./internal/service | Pass |
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
- Medium: Metrics map growth depends on path cardinality if unbounded path variants are introduced.
- Low: Additional middleware layer adds small per-request overhead on HTTP path.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
