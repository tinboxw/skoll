# E9-step2 Production Hardening Plan

## Objective

Define executable hardening controls for admin and control-plane paths before enabling any persistent adapter in production.

## Hardening Checklist

- Runtime resilience
  - Enforce request timeout and graceful shutdown bounds.
  - Define per-endpoint retry and idempotency boundaries.
- Access and operation safety
  - Strengthen high-risk operation guardrails (db ops, plugin upgrade, release evidence mutations).
  - Require audit context completeness for sensitive actions.
- Data safety
  - Define backup/restore cadence and verification checkpoints.
  - Define recovery-time and data-loss objective assumptions for rollout windows.
- Observability
  - Standardize control-plane metrics labels for route, actor, action, and result.
  - Add alert thresholds for conflict spikes, failed claims, and release scorecard regressions.

## Drill Scenarios

1. Backup/restore drill
   - Inject synthetic data, perform backup, recover to clean environment, verify scorecard/session/job consistency.
2. Dispatch conflict drill
   - Simulate duplicated dispatch claims under concurrent workers and verify idempotent claim status.
3. Release governance gate drill
   - Submit regression evidence and verify scorecard blocks rollout decisions.

## Alert Mapping

- `session_consistency_conflict_total` high-rate alert.
- `job_dispatch_claim_duplicate_total` burst alert.
- `release_scorecard_regression_block_total` non-zero alert.
- `admin_dangerous_operation_denied_total` sustained trend alert.

## Rollout Strategy

- Stage 1: shadow alerts, no blocking.
- Stage 2: alerts + runbook enforced on-call workflow.
- Stage 3: release gate blocking for unresolved critical alerts.

## Validation Commands

- `go test ./...`
- `go test -race ./...`
- `go test ./internal/app ./internal/module/user ./internal/module/jobscheduler ./internal/module/releasegov`
