# E9 Enhancement Iteration Plan

## Milestone

- ID: `E9`
- Name: `enhancement backlog prioritization and production hardening`
- Status: `step3 benchmark regression governance policy completed (E9 closed)`

## Scope

- Prioritize post-E8 enhancement backlog by risk and delivery value.
- Design persistent/shared-state adapter rollout for consistency-critical modules.
- Define production hardening plan for admin and control-plane APIs.
- Establish benchmark regression governance policy for milestone gating.

## Non-Goals

- No immediate transport contract break.
- No one-shot migration to all persistent backends.
- No replacement of existing in-memory adapter in the same step.

## Execution Sequence

### E9-step0 Backlog Prioritization Baseline

- Deliverables:
  - Prioritized backlog grouped by `P0/P1/P2`.
  - Dependency map for storage, session consistency, job dispatch, and release governance.
  - Milestone split for E9-step1/step2/step3.
- Acceptance:
  - Top 5 items have owner, acceptance, and rollback strategy.
  - Planning results are reflected in roadmap and README CN/EN.
- Suggested commands:
  - `go test ./...`
- Risks:
  - Planning drift from runtime constraints.
- Mitigation:
  - Keep API contracts fixed and validate against existing route/adapter tests.

### E9-step1 Persistence Adapter Rollout Design

- Deliverables:
  - Adapter rollout design for `user`, `jobscheduler`, `releasegov` modules.
  - Storage topology decision (`MySQL/PostgreSQL + Redis`) and migration path.
  - Contract coverage matrix (in-memory vs persistent).
- Acceptance:
  - Persistent adapter interfaces can pass existing contract tests with no transport changes.
  - Data migration and rollback strategy are documented.
- Suggested commands:
  - `go test ./internal/module/storageadapter ./internal/module/user ./internal/module/jobscheduler ./internal/module/releasegov`
  - `go test -race ./internal/module/user ./internal/module/jobscheduler`
- Risks:
  - Consistency regressions under multi-instance concurrency.
- Mitigation:
  - Add idempotency and version-conflict tests before enabling persistent writes by default.

### E9-step2 Production Hardening Plan

- Deliverables:
  - Hardening checklist: timeout, rate limit, audit retention, dangerous-operation controls.
  - Failure-drill scenarios for backup/restore, dispatch conflicts, and release governance gates.
  - Alert mapping for high-risk admin actions.
- Acceptance:
  - Each hardening item has measurable gate and runbook entry.
  - At least one drill scenario is executable in staging.
- Suggested commands:
  - `go test ./...`
  - `go test -race ./...`
- Risks:
  - Over-hardening can reduce operability.
- Mitigation:
  - Introduce feature flags and staged rollout windows.

### E9-step3 Benchmark Regression Governance Policy

- Deliverables:
  - Baseline snapshot policy and benchmark command whitelist.
  - Regression threshold policy by metric type.
  - Milestone-level pass/fail rule template for scorecard evidence.
- Acceptance:
  - Every enhancement milestone includes baseline/current delta evidence.
  - Regression over threshold becomes a blocking item.
- Suggested commands:
  - `go test -bench=. -benchmem ./...`
  - `go test -bench=BenchmarkAdminUsersListEndpoint -benchmem ./internal/app`
  - `go test -bench=BenchmarkServiceClaimRunConsistency -benchmem ./internal/module/jobscheduler`
- Risks:
  - Noisy benchmark data causes false blocking.
- Mitigation:
  - Use fixed environment assumptions and compare rolling medians.

## Prioritized Backlog (Initial)

- P0:
  - Persistent adapter design for session/job/release governance states.
  - Production hardening checklist and drill scripts.
- P1:
  - Benchmark regression policy and evidence automation alignment.
  - Alert profile tuning for consistency conflict and dangerous ops signals.
- P2:
  - Optional dashboard governance trend views for scorecard history.

## Validation Gate for E9 Planning Checkpoint

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...` (for concurrency-impacting changes)
- `go test -bench=. -benchmem ./...` (for benchmark-policy updates)

## Exit Criteria for E9

- E9-step1/step2/step3 each has committed milestone records.
- README CN/EN keep parity for commands, doc links, and milestone status.
- Regression policy is enforceable in milestone review.
