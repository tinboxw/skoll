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
- API: `POST /admin/v1/job-dispatch-claims/{execution_key}/renew`
- API: `GET /admin/v1/job-dispatch-claims/{execution_key}`
- API: `PUT /admin/v1/jobs/{id}/retry-policy`
- API: `POST /admin/v1/jobs/{id}/retries/schedule`
- API: `POST /admin/v1/jobs/{id}/dead-letters`
- API: `GET /admin/v1/jobs/dead-letters`
- API: `POST /admin/v1/jobs/dead-letters/{execution_key}/replay`
- API: `GET /admin/v1/jobs/reliability/metrics`
- API: `GET /admin/v1/system/dashboard` (`scheduler_reliability` section)
- Rules:
  - `execution_key` is treated as an idempotency key.
  - Repeated claims from the same instance are accepted as idempotent.
  - Claims from other instances with the same `execution_key` are blocked.
  - Lease renewal must be performed by the claim owner instance.
  - Retry scheduling must respect per-job bounded backoff policy.
  - Dead-letter replay is idempotent and auditable.
- Audit signals:
  - `job_dispatch_claim`
  - `job_dispatch_duplicate_blocked`
  - `job_dispatch_claim_renew`
  - `retry_scheduled`
  - `dead_letter_marked`
  - `dead_letter_replayed`

## Operational Guidance

- Use a deterministic `execution_key` format (for example: `job_id:scheduled_at`).
- Alert on repeated `session_consistency_conflict` events for the same session.
- Alert on `job_dispatch_duplicate_blocked` spikes as an early sign of scheduler drift.
- Track `retry_scheduled` backlog and escalation to `dead_letter_marked` for reliability degradation.
