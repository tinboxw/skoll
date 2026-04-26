# M4-admin-auth-shared-nonce-store-wiring

## Basic Info

- Milestone: M4-admin-auth-shared-nonce-store-wiring
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Wire a shared nonce replay store implementation for multi-instance deployments.
- Keep memory store as default to preserve backward compatibility.
- Avoid changing existing verifier contracts and auth mode semantics.

## Non-Goals

- Full distributed lock coordination.
- Key-management redesign.
- New admin auth protocol changes.

## Delivered

- Added Redis-backed nonce store implementation in admin auth integration.
- Added startup wiring for nonce store mode selection:
  - `admin-auth-nonce-store=auto|memory|redis`
- Added Redis startup config options:
  - `admin-auth-nonce-redis-addr`
  - `admin-auth-nonce-redis-password`
  - `admin-auth-nonce-redis-db`
  - `admin-auth-nonce-redis-key-prefix`
- Added startup-time Redis ping validation for fail-fast behavior.
- Kept default mode backward compatible:
  - `auto` + no Redis address => memory fallback.
- Added tests for:
  - nonce store resolver branches,
  - Redis nonce store use-once semantics.

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
- app benchmark median (`count 3`): `885.3 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 3`): `2.439 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 879.1 | 885.3 | +0.71% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.542 | 2.439 | -4.05% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
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

- High: Redis store misconfiguration now fails fast in HMAC flow when redis mode is selected.
- Medium: Redis availability affects HMAC verification when redis nonce store is enabled.
- Low: Additional startup config surface for nonce store wiring.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
