# Dashboard Auth/Session Policy

## Scope

This policy defines compatibility, rollout, and client integration expectations for dashboard authentication/session data returned by `GET /admin/v1/system/dashboard`.

## Current Contract Surface

The `auth_session` section currently includes:

- `authenticated`
- `auth_mode_hint`
- `role_id` (optional)
- `has_role_binding`
- `token_header_present`
- `signature_header_present`

## Compatibility Rules

1. Additive fields are backward-compatible in the same contract version.
2. Removing or renaming existing fields requires contract version bump.
3. Type changes for existing fields require contract version bump.
4. `auth_session` must remain listed in `contract.required_sections`.

## Security and Interpretation Notes

1. `auth_mode_hint` is a transport hint for dashboard bootstrapping and does not replace server-side authorization.
2. `authenticated=true` reflects request header presence/alignment context in current milestone scope.
3. Clients must not infer permission grants from `authenticated` alone; authorization remains RBAC/API-binding driven.

## Rollout Guidance

1. Frontend should gate parser behavior by `contract.name` and `contract.version`.
2. Frontend should tolerate unknown additive fields in `auth_session`.
3. Backend rollout should preserve existing field semantics until version bump.
4. Release notes should explicitly call out any contract changes before deployment.

## Future Evolution

- Bind `auth_session` to stronger identity/session claims once JWT/session flow is introduced.
- Add explicit claim provenance metadata when available.
