# M4-admin-auth-security-runbook

## Basic Info

- Milestone: M4-admin-auth-security-runbook
- Date range: 2026-04
- Owner: skoll contributors
- Related PRs/issues: TBD

## Objectives

- Publish a practical security runbook for admin auth operations.
- Define secret rotation and incident response procedures.
- Clarify production baseline and mode policy for admin auth.

## Non-Goals

- Replacing existing verifier implementations.
- Introducing new auth protocols.
- Adding external incident tooling integration.

## Delivered

- Added runbook document:
  - `docs/planning/ADMIN_AUTH_SECURITY_RUNBOOK.md`
- Documented:
  - mode policy (`none`, `static-token`, `hmac-sha256`)
  - production baseline controls (HTTPS, NTP, ingress restrictions, shared nonce store)
  - planned/emergency secret rotation procedures
  - incident trigger, triage, containment, recovery, postmortem actions
  - alerting guidance using auth metrics
- Synchronized roadmap, release checklist, and README CN/EN references.

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
- app benchmark median (`count 3`): `917.5 ns/op`, `1094 B/op`, `12 allocs/op`
- service benchmark median (`count 3`): `2.589 ns/op`, `0 B/op`, `0 allocs/op`

## Performance Comparison Data (required)

| Metric | Baseline | Current | Delta | Command | Verdict |
| --- | ---: | ---: | ---: | --- | --- |
| app BenchmarkHealthEndpoint ns/op | 4114 | 917.5 | -77.70% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint B/op | 1094 | 1094 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| app BenchmarkHealthEndpoint allocs/op | 12 | 12 | 0.00% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/app | PASS |
| service BenchmarkSystemServiceHealthParallel ns/op | 7.382 | 2.589 | -64.93% | go test -run ^$ -bench "." -benchmem -count 3 ./internal/service | PASS |
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
- Medium: static-token mode may still be used in non-production internal contexts and should remain explicitly scoped.
- Low: runbook quality depends on periodic review by security/operations owners.

## README Parity

- README.md status: synced
- README.en.md status: synced
- Parity gap: none
