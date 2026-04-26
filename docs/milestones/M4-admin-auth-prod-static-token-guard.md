# M4-admin-auth-prod-static-token-guard

## Basic Info

- Milestone: M4-admin-auth-prod-static-token-guard
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Close release blocker around static-token usage in production mode.
- Keep static-token available for internal/dev scenarios.
- Require explicit operator opt-in when static-token must run in prod mode.

## Non-Goals

- Replacing static-token implementation.
- Removing static-token mode from resolver matrix.
- Introducing new auth algorithm.

## Delivered

- Added startup policy guard in `cmd/skoll/main.go`:
  - when `go-admin-enabled=true` and `go-admin-mode=prod`, effective admin auth mode `static-token` is rejected by default.
- Added explicit escape hatch:
  - flag: `-admin-auth-allow-static-token-in-prod`
  - env: `SKOLL_ADMIN_AUTH_ALLOW_STATIC_TOKEN_IN_PROD`
- Added effective-mode policy validation helper and unit tests.
- Extended runtime config snapshot log to include:
  - `admin-auth-effective-mode`
  - `admin-auth-allow-static-token-in-prod`

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
- app benchmark median (`count 10`): `888.7 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 10`): `2.4885 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 917.5 | 888.7 | -3.14% | go test -run ^$ -bench "." -benchmem -count 10 ./internal/app | PASS |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.589 | 2.4885 | -3.88% | go test -run ^$ -bench "." -benchmem -count 10 ./internal/service | PASS |
| service BenchmarkSystemServiceHealthParallel B/op | 0 | 0 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
| service BenchmarkSystemServiceHealthParallel allocs/op | 0 | 0 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |

Notes:

- Additional stability check was run with fresh `count 3` and `count 10` samples.
- `count 10` medians were used for final verdict to reduce short-window benchmark noise.

## DoD Review (5 points)

- [x] Code compiles and tests pass
- [x] Package boundaries are clear
- [x] Runnable entry/example exists
- [x] Documentation is updated
- [x] Risks/assumptions are documented

Score: 5/5

## Regression Risks

- High: none identified.
- Medium: operator may bypass policy by explicitly enabling static-token in prod; should be tracked and audited.
- Low: startup policy check is evaluated only at process boot.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
