# Dashboard UI Bootstrap Contract

## Purpose

Define a stable response contract for `GET /admin/v1/system/dashboard` so frontend dashboard bootstrap logic can evolve independently from backend internals.

## Contract Identity

- `name`: `dashboard-ui-bootstrap`
- `version`: `v1`
- `stability`: `stable`

## Response Shape

Top-level fields:

- `contract`: contract descriptor
- `generated_at_unix_sec`: response generation timestamp
- `auth_session`: request auth/session context snapshot
- `auth_observability`: auth verification counters and failure reasons summary
- `auth_actionability`: actionable guidance for auth/session operation and UX handling
- `jwt_session_bootstrap`: unverified JWT bootstrap metadata from bearer token context
	- includes refresh/expiry hints: `session_state`, `expires_in_sec`, `refresh_recommended`, `refresh_reason`, `refresh_after_unix_sec`
	- includes verification-state hints: `verification_state`, `verification_hint`, `trust_level`, `trust_message`
	- includes middleware bridge hints: `claims_trusted`, `middleware_bridge.present`, `middleware_bridge.verified`, `middleware_bridge.source`, `middleware_bridge.subject`, `middleware_bridge.role_id`, `middleware_bridge.claims_version`
- `status`: module/resource counters
- `runtime_metrics`: process runtime snapshot
- `node_health`: service dependency health summary

## Required Sections

`contract.required_sections` must contain:

1. `status`
2. `runtime_metrics`
3. `node_health`
4. `auth_session`
5. `auth_observability`
6. `auth_actionability`
7. `jwt_session_bootstrap`

Consumers should treat unknown additive fields as forward-compatible.

## Compatibility Rules

- Additive fields are allowed in `v1`.
- Removing or renaming existing fields requires contract version bump.
- Type changes for existing fields require contract version bump.

## Client Guidance

- Validate `contract.name` equals `dashboard-ui-bootstrap`.
- Branch by `contract.version` for parsing logic.
- Implement tolerant parsing for additional fields.
