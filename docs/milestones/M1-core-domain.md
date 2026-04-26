# M1-core-domain

## Basic Info

- Milestone: M1-core-domain
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Introduce explicit core domain and service boundaries.
- Move health/readiness business logic from transport layer into service layer.
- Keep HTTP handlers thin and focused on protocol mapping.

## Non-Goals

- New business modules beyond system health/readiness.
- Persistence or external dependencies.
- API surface expansion.

## Delivered

- Added `internal/domain` with domain-level health/readiness modeling.
- Added `internal/service` with `SystemService` encapsulating readiness state and uptime/version logic.
- Refactored `internal/app/server.go` to depend on service outputs.
- Added deterministic unit tests for service logic (`internal/service/system_service_test.go`).
- Updated README and README.en structure documentation in parity.

## Validation Evidence

Commands:

- go fmt ./...
- go test ./...
- go test -race ./...
- go test -bench "." -benchmem ./internal/app

Key output summary:

- go test ./...: pass
- go test -race ./...: pass
- benchmark snapshot:
	- BenchmarkHealthEndpoint-16: 952.1 ns/op, 1072 B/op, 11 allocs/op

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
|---|---:|---:|---:|---|---|
| bench ns/op | 879.1 | 952.1 | +73.0 | go test -bench "." -benchmem ./internal/app | Pass |
| B/op | 1072 | 1072 | 0 | go test -bench "." -benchmem ./internal/app | Pass |
| allocs/op | 11 | 11 | 0 | go test -bench "." -benchmem ./internal/app | Pass |

## DoD Review (5 points)

- [x] Code compiles and tests pass
- [x] Package boundaries are clear
- [x] Runnable entry/example exists
- [x] Documentation is updated
- [x] Risks/assumptions are documented

Score: 5/5

## Regression Risks

- High: None identified.
- Medium: Domain-service mapping may drift if response DTOs change without tests.
- Low: Uptime string format depends on Go duration string formatting.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
