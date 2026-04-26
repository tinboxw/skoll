# M4-admin-auth-nonce-store-interface

## Basic Info

- Milestone: M4-admin-auth-nonce-store-interface
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Extract nonce replay defense into a pluggable store contract.
- Keep default in-memory behavior unchanged.
- Enable future multi-instance shared nonce backends without changing verifier contract.

## Non-Goals

- Introducing a concrete Redis/DB dependency in this milestone.
- Changing admin auth flags or runtime mode names.
- Redesigning existing HMAC payload/signature rules.

## Delivered

- Added `ReplayNonceStore` interface in admin auth boundary.
- Added `ResolveVerifierWithNonceStore(...)` to support dependency injection for HMAC verifier.
- Kept `ResolveVerifier(...)` as backward-compatible default entrypoint.
- Refactored in-memory implementation to `memoryNonceStore` with same one-time consume semantics.
- Added tests to verify injected nonce store participates in verification.
- Updated roadmap/release checklist and README CN/EN milestone links.

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
- app benchmark median (`count 3`): `879.1 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 3`): `2.542 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 876.5 | 879.1 | +0.30% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 2.525 | 2.542 | +0.67% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
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
- Medium: Shared backend wiring is not enabled by runtime config yet (injection point is available for integration layer).
- Low: One extra interface indirection in HMAC verifier path.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
