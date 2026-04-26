# M2-concurrency-and-performance

## Basic Info

- Milestone: M2-concurrency-and-performance
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Add explicit concurrency primitives on a hot service path.
- Introduce benchmark-driven tuning for health path under parallel access.
- Keep API behavior unchanged while improving high-concurrency stability.

## Non-Goals

- External dependency integration.
- Full observability stack (planned in M3).
- Business domain expansion.

## Delivered

- Added `sync.RWMutex` guarded health cache in `SystemService`.
- Added health cache TTL (200ms) to reduce repeated health recomputation under high QPS.
- Added cache TTL behavior test.
- Added concurrent safety test for parallel `Health()` calls.
- Added parallel benchmark for `SystemService.Health()`.

## Validation Evidence

Commands:

- go fmt ./...
- go test ./...
- go test -race ./...
- go test -bench=. -benchmem ./...

Key output summary:

- go test ./...: pass
- go test -race ./...: pass
- benchmark snapshot:
	- internal/app BenchmarkHealthEndpoint-16: 858.8 ns/op, 1070 B/op, 11 allocs/op
	- internal/service BenchmarkSystemServiceHealthParallel-16: 2.567 ns/op, 0 B/op, 0 allocs/op

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
|---|---:|---:|---:|---|---|
| app BenchmarkHealthEndpoint ns/op | 952.1 | 858.8 | -93.3 (-9.80%) | go test -bench "." -benchmem ./internal/app | Pass |
| app BenchmarkHealthEndpoint B/op | 1072 | 1070 | -2 (-0.19%) | go test -bench "." -benchmem ./internal/app | Pass |
| app BenchmarkHealthEndpoint allocs/op | 11 | 11 | 0 (0%) | go test -bench "." -benchmem ./internal/app | Pass |
| service BenchmarkSystemServiceHealthParallel ns/op | N/A | 2.567 | N/A | go test -bench "." -benchmem ./internal/service | Pass |
| service BenchmarkSystemServiceHealthParallel B/op | N/A | 0 | N/A | go test -bench "." -benchmem ./internal/service | Pass |
| service BenchmarkSystemServiceHealthParallel allocs/op | N/A | 0 | N/A | go test -bench "." -benchmem ./internal/service | Pass |

Notes:

- `BenchmarkSystemServiceHealthParallel` is newly introduced in M2, so baseline is `N/A`.

## DoD Review (5 points)

- [x] Code compiles and tests pass
- [x] Package boundaries are clear
- [x] Runnable entry/example exists
- [x] Documentation is updated
- [x] Risks/assumptions are documented

Score: 5/5

## Regression Risks

- High: None identified.
- Medium: Health cache TTL introduces bounded staleness for uptime string.
- Low: Cache validity relies on wall-clock time; abrupt system time jumps can shorten/extend effective TTL.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
