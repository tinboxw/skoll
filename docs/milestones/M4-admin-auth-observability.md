# M4-admin-auth-observability

## Basic Info

- Milestone: M4-admin-auth-observability
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Add admin auth verification observability for runtime operations.
- Expose failure-reason counters for faster auth incident triage.
- Keep existing verifier contract and auth mode behavior unchanged.

## Non-Goals

- Per-tenant or per-user auth metrics dimensions.
- Dynamic failure-reason taxonomy at runtime.
- External metrics backend integration.

## Delivered

- Added admin auth metrics counters in `internal/integration/adminauth/metrics.go`:
  - `skoll_admin_auth_verifications_total{mode,result}`
  - `skoll_admin_auth_failures_total{mode,reason}`
- Instrumented verifier paths with explicit reason tracking:
  - static-token: `missing_token`, `invalid_token`
  - hmac-sha256: `missing_headers`, `invalid_timestamp`, `timestamp_skew`, `replay_nonce`, `invalid_body_hash`, `invalid_signature`
- Added `MetricsPrometheus()` exporter in admin auth integration.
- Added extensible metrics collector hook in `internal/app/server.go`.
- Wired admin auth metrics into `/metrics` in `cmd/skoll/main.go`.
- Added tests for:
  - metrics reason counters (`internal/integration/adminauth/metrics_test.go`)
  - server external collector aggregation (`internal/app/server_test.go`)

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
- app benchmark median (`count 3`): `4114 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 3`): `7.382 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 885.3 | 4114 | +364.70% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | Needs follow-up |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.439 | 7.382 | +202.67% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | Needs follow-up |
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
- Medium: benchmark `ns/op` medians are significantly higher than prior M4 snapshot and require follow-up sampling/benchstat confirmation.
- Low: auth failure reason labels are intentionally bounded; future reason additions should be reviewed for cardinality impact.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
