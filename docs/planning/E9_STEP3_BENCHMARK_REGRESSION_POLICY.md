# E9-step3 Benchmark Regression Governance Policy

## Objective

Define benchmark evidence and blocking thresholds for enhancement milestones.

## Benchmark Command Whitelist

- `go test -bench=. -benchmem ./...`
- `go test -bench=BenchmarkAdminUsersListEndpoint -benchmem ./internal/app`
- `go test -bench=BenchmarkServiceClaimRunConsistency -benchmem ./internal/module/jobscheduler`
- `go test -bench=BenchmarkServiceSubmitEvidence -benchmem ./internal/module/releasegov`

## Baseline Policy

- Keep one approved baseline snapshot per milestone stream.
- Baselines must include machine profile, Go version, and command used.
- Evidence must capture median of at least 5 samples for noisy benchmarks.

## Threshold Policy

- Latency-oriented benchmark (`ns/op`):
  - Warning: regression >= 8%
  - Block: regression >= 15%
- Allocation count (`allocs/op`):
  - Warning: increase >= 1 allocation
  - Block: increase >= 3 allocations
- Memory footprint (`B/op`):
  - Warning: regression >= 10%
  - Block: regression >= 20%

## Milestone Gate Rule

- Every enhancement milestone must attach baseline vs current benchmark evidence.
- Any block-threshold regression requires explicit exception approval and rollback plan.
- Unapproved block-threshold regression prevents milestone completion sign-off.

## Reporting Template

- Benchmark ID
- Baseline value
- Current value
- Delta (% or absolute)
- Threshold status (`pass`/`warning`/`block`)
- Exception ticket (if any)

## Validation Commands

- `go test -bench=. -benchmem ./...`
- `go test ./...`
