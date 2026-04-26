# M4-admin-auth-hmac-sha256

## Basic Info

- Milestone: M4-admin-auth-hmac-sha256
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Add a stronger second verifier implementation on top of pluggable admin auth.
- Keep static-token mode unchanged and backward compatible.
- Add replay-window protection for signed admin probe requests.

## Non-Goals

- Full JWT/RBAC integration.
- Persistent nonce storage for strict anti-replay guarantees.
- Cross-service shared key orchestration.

## Delivered

- Added `hmac-sha256` auth mode in `internal/integration/adminauth`.
- Added request headers contract:
  - `X-Admin-Timestamp` (Unix seconds)
  - `X-Admin-Signature` (hex HMAC-SHA256)
- Added signature payload format: `METHOD + "\\n" + PATH + "\\n" + TIMESTAMP`.
- Added ±5 minute timestamp skew validation to reject stale/future signatures.
- Added main runtime config support:
  - `-admin-auth-hmac-secret`
  - `SKOLL_ADMIN_AUTH_HMAC_SECRET`
- Extended resolver mode matrix:
  - `auto` now chooses `static-token` first, else `hmac-sha256` if HMAC secret is present.
- Added focused tests for:
  - resolver behavior,
  - valid/invalid HMAC signature,
  - stale timestamp rejection.
- Added integration test over go-admin probe handler wrapped by HMAC verifier.
- Added operator/client contract doc:
  - `docs/planning/ADMIN_AUTH_SIGNATURE_CONTRACT.md`

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
- app benchmark median (`count 3`): `1006 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 3`): `2.476 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 901.3 | 1006 | +11.62% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.555 | 2.476 | -3.09% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
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
- Medium: HMAC mode relies on caller/server clock alignment within skew window.
- Low: Request body is not included in signature payload in this phase (current admin probe is GET).

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
