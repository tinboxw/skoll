# Dashboard JWT Middleware Bridge Contract

## Purpose

Define interoperability and rollout expectations for middleware-provided verified JWT claims that are surfaced through `GET /admin/v1/system/dashboard` under `jwt_session_bootstrap.middleware_bridge`.

## Scope

This contract governs bridge metadata only. It does not define JWT signing algorithms, token issuance, or authorization policy semantics.

## Bridge Fields

`jwt_session_bootstrap` includes the following bridge fields:

- `claims_trusted` (bool)
- `middleware_bridge.present` (bool)
- `middleware_bridge.verified` (bool)
- `middleware_bridge.source` (string)
- `middleware_bridge.subject` (optional string)
- `middleware_bridge.role_id` (optional string)
- `middleware_bridge.claims_version` (optional string)

Current source modes:

- `none`: no bridge metadata found
- `header`: metadata provided by HTTP headers

## Header Mapping (Current Implementation)

Bridge extraction currently maps from these request headers:

- `X-Admin-JWT-Verified` -> `middleware_bridge.verified`
- `X-Admin-JWT-Subject` -> `middleware_bridge.subject`
- `X-Admin-Role-ID` -> `middleware_bridge.role_id`
- `X-Admin-JWT-Claims-Version` -> `middleware_bridge.claims_version`

`middleware_bridge.present` is true when any mapped bridge header is provided.

`claims_trusted` is aligned with `middleware_bridge.verified` in current milestone scope.

## Compatibility Rules

1. Additive fields in `middleware_bridge` are backward-compatible for `contract.version=v1`.
2. Removing or renaming existing bridge fields requires a contract version bump.
3. Type changes for existing bridge fields require a contract version bump.
4. Clients must tolerate missing optional bridge fields.

## Trust and UX Guidance

1. `claims_trusted=true` indicates middleware has marked claims as verified for bootstrap trust messaging.
2. `claims_trusted=false` means parsed JWT hints remain non-authoritative for privileged UX decisions.
3. Frontend should gate privileged actions on backend authorization responses, not bridge fields alone.
4. Frontend may use bridge fields to reduce identity flicker during bootstrap.

## Rollout Guidance

1. Deploy middleware headers in canary environments first.
2. Validate dashboard payload parity before broad rollout.
3. Track mismatches between bridge subject/role and backend authorization outcomes.
4. Document middleware claim-version evolution through `claims_version`.

## Future Evolution

- Add non-header bridge source types (for example context-injected metadata).
- Align bridge subject normalization with planned JWT verified-claims adapter.
- Add explicit provenance markers when multiple middleware layers contribute claims.
