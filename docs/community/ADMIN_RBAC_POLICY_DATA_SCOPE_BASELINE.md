# Admin RBAC Policy and Data-Scope Baseline

## Purpose

Define the E1 baseline for policy-engine style authorization and data-scope enforcement on admin APIs.

This baseline extends role API bindings with:

- policy rules (`allow` / `deny`) per API resource
- optional claim constraints (`require_verified`, `require_claims_version`)
- data-scope constraints (`tenant_ids`, `require_owner_match`)

## API Endpoints

Role policy management:

- `PUT /admin/v1/roles/{id}/policies`
- `GET /admin/v1/roles/{id}/policies`

Role data-scope management:

- `PUT /admin/v1/roles/{id}/data-scope`
- `GET /admin/v1/roles/{id}/data-scope`

## Payload Contracts

### Set role policies

```json
{
  "rules": [
    {
      "api": "GET:/admin/v1/users",
      "effect": "allow",
      "require_verified": true,
      "require_claims_version": "v2"
    },
    {
      "api": "POST:/admin/v1/users",
      "effect": "deny"
    }
  ]
}
```

Rules are normalized and deduplicated by API/effect/constraint tuple.

### Set role data scope

```json
{
  "tenant_ids": ["tenant-a", "tenant-b"],
  "require_owner_match": true
}
```

Tenant list is normalized and deduplicated.

## Authorization Evaluation Order

For each admin API request:

1. Verify admin identity headers and resolve role ID.
2. Verify role exists.
3. Check role API binding (`GET:/path`, `POST:/path`, etc.).
4. Evaluate policy rules for the resolved API:
   - any matching `deny` => reject
   - if no matching policy => pass (baseline compatibility)
   - if matching `allow` rules exist, at least one must satisfy claim constraints
5. Evaluate data scope:
   - if `tenant_ids` present, `X-Data-Tenant-ID` must exist and be in scope
   - if `require_owner_match=true`, `X-Resource-Owner` must equal JWT subject (`X-Admin-JWT-Subject`)

## Headers Used by Data Scope

- `X-Data-Tenant-ID`
- `X-Resource-Owner`
- `X-Admin-JWT-Subject`

## Governance Notes

- Baseline is intentionally strict on explicit deny and scope mismatch.
- Policy/data-scope changes should be version-controlled and reviewed.
- This is a compatibility-first baseline; richer policy expressions can be added in E1 follow-up slices if needed.
