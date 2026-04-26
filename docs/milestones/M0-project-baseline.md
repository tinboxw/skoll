# M0-project-baseline

## Basic Info

- Milestone: M0-project-baseline
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Initialize a minimal runnable Go baseline service.
- Establish package boundaries: cmd/internal/pkg.
- Add baseline health/readiness endpoints and tests.

## Non-Goals

- Complex business modules.
- Persistent data layer.
- Advanced distributed components.

## Planned Deliverables

- go.mod initialized.
- cmd/skoll entrypoint with graceful shutdown.
- internal/app HTTP server with /health and /ready.
- pkg/version for build version string.
- Unit tests and benchmark for baseline endpoint path.
- README.md and README.en.md synchronized for run/test usage.

## Validation Commands

- go fmt ./...
- go test ./...
- go test -race ./...
- go test -bench "." -benchmem ./internal/app

## Validation Output Summary

- go test ./...: pass
- go test -race ./...: pass
- benchmark snapshot:
	- BenchmarkHealthEndpoint-16: 879.1 ns/op, 1072 B/op, 11 allocs/op

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
|---|---:|---:|---:|---|---|
| p95 latency | N/A | TBD | N/A | N/A | TBD |
| throughput/QPS | N/A | TBD | N/A | N/A | TBD |
| bench ns/op | N/A | 879.1 | N/A | go test -bench "." -benchmem ./internal/app | Pass |

## DoD Review (5 points)

- [x] Code compiles and tests pass
- [x] Package boundaries are clear
- [x] Runnable entry/example exists
- [x] Documentation is updated
- [x] Risks/assumptions are documented

Score: 5/5

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
