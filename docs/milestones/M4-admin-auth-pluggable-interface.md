# M4-admin-auth-pluggable-interface

## Basic Info

- Milestone: M4-admin-auth-pluggable-interface
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Upgrade admin auth from single hardcoded middleware to a pluggable verifier contract.
- Keep existing static-token behavior fully backward compatible.
- Prepare extension point for future auth mechanisms (e.g. JWT or upstream gateway assertions).

## Non-Goals

- Full RBAC/permission matrix.
- Introducing business module authorization.
- Third-party identity provider integration.

## Delivered

- Added dedicated integration package: `internal/integration/adminauth`.
- Introduced verifier abstraction:
  - `Verifier` interface (`Mode()`, `Verify(*http.Request) bool`)
  - `ResolveVerifier(mode, token)` for startup-time auth config resolution.
- Added first concrete verifier implementation:
  - `static-token` using constant-time compare.
- Added reusable middleware wrapper:
  - `WithVerifier(next, verifier)` returns 401 JSON on auth failure.
- Updated startup config in `cmd/skoll/main.go`:
  - new flag/env: `-admin-auth-mode` / `SKOLL_ADMIN_AUTH_MODE`
  - supported modes: `auto|none|static-token`
  - runtime log snapshot now includes `admin-auth-mode`.
- Added focused tests for resolver and middleware behavior.

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
- app benchmark median (`count 3`): `901.3 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 3`): `2.555 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 874.1 | 901.3 | +3.11% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.454 | 2.555 | +4.12% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
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
- Medium: Mode parsing now controls startup behavior; invalid mode fails fast by design.
- Low: Static-token remains backward compatible under `auto` mode with token configured.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
