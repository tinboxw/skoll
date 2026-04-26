# Dashboard JWT Claims Normalization Policy

## Purpose

Define stable normalization rules for JWT bridge claim extraction so dashboard bootstrap and RBAC authorization consume equivalent identity context.

## Scope

Applies to shared adapter logic used by:

- dashboard JWT middleware bridge extraction
- RBAC role-id extraction for admin API authorization

## Normalized Fields

- `verified`
- `subject`
- `role_id`
- `claims_version`

## Rules

1. `verified`:
   - true aliases: `true`, `1`, `yes`, `y`, `on` (case-insensitive)
   - any other value is treated as false
2. `subject`:
   - trim surrounding whitespace only
3. `role_id`:
   - accepts positive integer strings only
   - normalized to canonical decimal (`001` -> `1`)
   - precedence: `X-Admin-JWT-Role-ID` then `X-Admin-Role-ID`
   - invalid JWT-specific role values fall back to legacy role header if valid
4. `claims_version`:
   - trim whitespace and normalize to lowercase

## Compatibility Notes

1. Normalization output shape must remain stable within `dashboard-ui-bootstrap v1`.
2. Expanding accepted alias sets is backward-compatible.
3. Changing role-id precedence or type rules requires explicit rollout notice and compatibility review.

## Security Notes

1. Normalization does not imply trust.
2. Trust semantics continue to rely on verification-state fields (`claims_trusted`, `verification_state`).
3. Authorization decisions must still enforce backend RBAC checks.

## Testing Expectations

- Adapter unit tests must cover precedence, fallback, alias parsing, and canonical role-id output.
- RBAC integration tests must verify authorization behavior with normalized JWT role-id inputs.
