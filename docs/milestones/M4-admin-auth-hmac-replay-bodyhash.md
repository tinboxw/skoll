# M4-admin-auth-hmac-replay-bodyhash

## Basic Info

- Milestone: M4-admin-auth-hmac-replay-bodyhash
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Add replay protection to HMAC verifier via nonce tracking.
- Extend HMAC payload to include nonce and optional body hash.
- Keep compatibility for current admin probe use case while improving security posture.

## Non-Goals

- Distributed nonce store across multiple instances.
- Persistent anti-replay ledger across restarts.
- Full request canonicalization framework.

## Delivered

- Added nonce header contract and one-time use validation:
  - `X-Admin-Nonce`
- Added optional body hash validation:
  - `X-Admin-Body-SHA256`
  - server verifies provided hash matches actual request body.
- Updated HMAC payload format:
  - `method + path + timestamp + nonce + bodyHash` (newline-separated)
- Added in-memory nonce store with TTL cleanup and one-time consume semantics.
- Updated unit tests for:
  - nonce replay rejection,
  - valid/invalid body hash paths.
- Updated go-admin integration test to include nonce in signed request.
- Updated signature contract documentation:
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
- app benchmark median (`count 3`): `876.5 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 3`): `2.525 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 1006 | 876.5 | -12.87% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.476 | 2.525 | +1.98% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
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
- Medium: Nonce replay protection is process-local; multi-instance deployments need shared replay strategy if strict global guarantees are required.
- Low: Optional body hash verification adds small overhead only when header is supplied.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
