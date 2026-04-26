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
- `middleware_bridge.source_provenance` (string[])
- `middleware_bridge.subject` (optional string)
- `middleware_bridge.subject_source` (optional string)
- `middleware_bridge.role_id` (optional string)
- `middleware_bridge.role_source` (optional string)
- `middleware_bridge.claims_version` (optional string)
- `middleware_bridge.claims_version_source` (optional string)
- `middleware_bridge.verified_source` (optional string)

`jwt_session_bootstrap.provenance_audit_export` includes:

- `enabled` (bool)
- `source_provenance` (string[])
- `source_path` (optional string, `>` joined chain)
- `source` (optional string)
- `verified` (bool)
- `claims_trusted` (bool)
- `verification_state` (string)
- `subject` (optional string)
- `role_id` (optional string)
- `claims_version` (optional string)
- `role_source` (optional string)
- `subject_source` (optional string)
- `claims_version_source` (optional string)
- `verified_source` (optional string)

Current source modes:

- `none`: no bridge metadata found
- `header`: metadata provided by HTTP headers

When `X-Admin-JWT-Source` is provided, `middleware_bridge.source` uses that normalized value.

## Header Mapping (Current Implementation)

Bridge extraction currently maps from these request headers:

- `X-Admin-JWT-Verified` -> `middleware_bridge.verified`
- `X-Admin-JWT-Subject` -> `middleware_bridge.subject`
- `X-Admin-JWT-Role-ID` -> `middleware_bridge.role_id` (preferred)
- `X-Admin-Role-ID` -> `middleware_bridge.role_id`
- `X-Admin-JWT-Claims-Version` -> `middleware_bridge.claims_version`

Role ID precedence in current implementation:

1. `X-Admin-JWT-Role-ID`
2. `X-Admin-Role-ID`

## Normalization Rules

Current adapter normalization policy:

1. `role_id` accepts positive integer strings only and is normalized to canonical decimal form.
2. Invalid JWT-specific role IDs fall back to legacy role header when valid.
3. `subject` is trim-normalized.
4. `claims_version` is normalized to lowercase.
5. `verified` accepts aliases: `true/1/yes/y/on`.

See detailed policy:

- `docs/community/DASHBOARD_JWT_CLAIMS_NORMALIZATION_POLICY.md`

`middleware_bridge.present` is true when any mapped bridge header is provided.

`claims_trusted` is aligned with `middleware_bridge.verified` in current milestone scope.

`middleware_bridge.source_provenance` is populated from:

- `X-Admin-JWT-Source-Provenance` (comma-separated chain, deduplicated)
- fallback to `[source]` when no chain is provided and source is not `none`

`provenance_audit_export` is derived from `middleware_bridge` plus `verification_state` and is intended for security operations export pipelines.

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

For provenance audit export compatibility and SIEM field mapping, see:

- `docs/community/DASHBOARD_JWT_PROVENANCE_AUDIT_EXPORT_GUIDE.md`
