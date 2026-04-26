# M4-admin-auth-skeleton

## Basic Info

- Milestone: M4-admin-auth-skeleton
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Introduce a minimal authorization boundary for admin route group.
- Keep feature rollout controlled and optional by configuration.
- Avoid coupling auth logic into domain/service packages.

## Non-Goals

- Full RBAC model.
- Session/JWT implementation.
- Business module authorization matrix.

## Delivered

- Added optional startup flag and env default:
  - `-admin-auth-token`
  - `SKOLL_ADMIN_AUTH_TOKEN`
- Added admin middleware skeleton in `cmd/skoll/main.go`:
  - validates request header `X-Admin-Token`
  - returns `401` with JSON error when token mismatch/missing
- Mounted `/admin/ping` under optional middleware when token configured.
- Added tests for middleware allow/reject behavior.
- Added runtime config snapshot field `admin-auth-enabled`.

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
- app benchmark median (`count 3`): `874.1 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 3`): `2.454 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 953.3 | 874.1 | -8.31% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint B/op | 1095 | 1094 | -0.09% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 6.751 | 2.454 | -63.65% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
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
- Medium: Header token skeleton is static-token based; production rollout should migrate to stronger auth model.
- Low: Middleware is active only when token is configured.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
