# Multi-Instance Consistency Baseline

## Scope

This baseline defines consistency controls for session ownership and job dispatch when Skoll runs across multiple instances.

## Session Consistency

- API: `POST /admin/v1/sessions/consistency/heartbeat`
- API: `GET /admin/v1/sessions/{session_id}/consistency`
- Rules:
  - Session heartbeat versions must be monotonic.
  - Lower-version heartbeats are rejected as stale.
  - Same-version writes from different instances are flagged as writer conflicts.
- Audit signals:
  - `session_consistency_heartbeat`
  - `session_consistency_conflict`

## Job Dispatch Consistency

- API: `POST /admin/v1/jobs/{id}/dispatch-claim`
- API: `GET /admin/v1/job-dispatch-claims/{execution_key}`
- Rules:
  - `execution_key` is treated as an idempotency key.
  - Repeated claims from the same instance are accepted as idempotent.
  - Claims from other instances with the same `execution_key` are blocked.
- Audit signals:
  - `job_dispatch_claim`
  - `job_dispatch_duplicate_blocked`

## Operational Guidance

- Use a deterministic `execution_key` format (for example: `job_id:scheduled_at`).
- Alert on repeated `session_consistency_conflict` events for the same session.
- Alert on `job_dispatch_duplicate_blocked` spikes as an early sign of scheduler drift.
