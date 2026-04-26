# Dashboard JWT Provenance Audit Export Guide

## Purpose

Define compatibility boundaries and SIEM mapping guidance for `jwt_session_bootstrap.provenance_audit_export` in `GET /admin/v1/system/dashboard`.

This guide focuses on security operations export. It does not change authorization semantics.

## Payload Scope

`provenance_audit_export` currently includes:

- `enabled`
- `source_provenance`
- `source_path`
- `operational_metrics`
- `alerting_hints`
- `source`
- `verified`
- `claims_trusted`
- `verification_state`
- `subject`
- `role_id`
- `claims_version`
- `role_source`
- `subject_source`
- `claims_version_source`
- `verified_source`

`operational_metrics` currently includes:

- `exports_total`
- `enabled_total`
- `disabled_total`
- `verified_total`
- `unverified_total`
- `invalid_total`
- `alert_hints_total`

`alerting_hints` values currently include:

- `verification_state_invalid`
- `verification_state_unverified`
- `claims_not_verified`
- `claims_version_missing`
- `provenance_chain_depth_high`

## Compatibility Rules

1. Additive fields are backward-compatible under current dashboard contract `v1`.
2. Removing, renaming, or changing field types requires a contract version bump.
3. Consumers must tolerate optional fields being absent.
4. `source_provenance` order is significant and should be preserved by downstream pipelines.

## SIEM Mapping Guidance

Use a stable, product-scoped key namespace for exports to avoid collisions:

- Prefix: `skoll.jwt.provenance.*`

Recommended mapping:

| Dashboard field | Suggested audit key | Type | Notes |
| --- | --- | --- | --- |
| `enabled` | `skoll.jwt.provenance.enabled` | bool | Gate for whether provenance export is meaningful. |
| `source_provenance` | `skoll.jwt.provenance.chain` | array<string> | Preserve sequence; do not sort. |
| `source_path` | `skoll.jwt.provenance.path` | string | Optional compact chain representation (`>` joined). |
| `source` | `skoll.jwt.provenance.source` | string | Final normalized source hint. |
| `verified` | `skoll.jwt.provenance.verified` | bool | Middleware verification marker. |
| `claims_trusted` | `skoll.jwt.provenance.claims_trusted` | bool | Mirrors bootstrap trust posture. |
| `verification_state` | `skoll.jwt.provenance.verification_state` | string | `not_present`/`unverified`/`verified`/`invalid`. |
| `subject` | `skoll.jwt.provenance.subject` | string | Optional JWT subject hint. |
| `role_id` | `skoll.jwt.provenance.role_id` | string | Optional normalized role identifier. |
| `claims_version` | `skoll.jwt.provenance.claims_version` | string | Optional claim schema/version marker. |
| `role_source` | `skoll.jwt.provenance.role_source` | string | Header/source metadata for role extraction. |
| `subject_source` | `skoll.jwt.provenance.subject_source` | string | Header/source metadata for subject extraction. |
| `claims_version_source` | `skoll.jwt.provenance.claims_version_source` | string | Header/source metadata for version extraction. |
| `verified_source` | `skoll.jwt.provenance.verified_source` | string | Header/source metadata for verified extraction. |

## Export Example

```json
{
  "event_type": "dashboard_jwt_provenance",
  "skoll.jwt.provenance.enabled": true,
  "skoll.jwt.provenance.chain": ["edge-auth", "gateway"],
  "skoll.jwt.provenance.path": "edge-auth>gateway",
  "skoll.jwt.provenance.source": "gateway",
  "skoll.jwt.provenance.verified": true,
  "skoll.jwt.provenance.claims_trusted": true,
  "skoll.jwt.provenance.verification_state": "verified",
  "skoll.jwt.provenance.subject": "bridge-user",
  "skoll.jwt.provenance.role_id": "9",
  "skoll.jwt.provenance.claims_version": "v1",
  "skoll.jwt.provenance.role_source": "x-admin-role-id",
  "skoll.jwt.provenance.subject_source": "x-admin-jwt-subject",
  "skoll.jwt.provenance.claims_version_source": "x-admin-jwt-claims-version",
  "skoll.jwt.provenance.verified_source": "x-admin-jwt-verified"
}
```

## Operational Guidance

1. Store both `chain` and `path` when possible: chain for exact sequence analytics, path for quick filtering.
2. Treat `subject` and `role_id` as operational hints; enforce access decisions with backend authorization outcomes.
3. Alert on abrupt source-chain shifts (for example, missing expected upstream hop) as potential middleware regression.
4. Track `claims_version` transitions during middleware rollouts to detect mixed deployment states.

## Operations Runbook

For incident triage and mitigation workflow, see:

- `docs/community/DASHBOARD_JWT_PROVENANCE_OPS_RUNBOOK.md`
