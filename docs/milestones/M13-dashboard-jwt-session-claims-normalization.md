# M13-dashboard-jwt-session-claims-normalization

## Summary

This milestone slice adds a shared JWT claims normalization policy so dashboard bootstrap and RBAC authorization consume aligned role/subject context.

## Delivered

- Extended shared verified-claims adapter with normalization rules for:
  - `verified` alias parsing
  - `subject` trim normalization
  - `role_id` canonicalization and precedence fallback
  - `claims_version` lowercase normalization
- Added JWT role-id normalization behavior:
  - `X-Admin-JWT-Role-ID` preferred
  - invalid preferred value falls back to `X-Admin-Role-ID` when valid
- Updated RBAC E2E to verify authorization via normalized JWT role-id input.
- Updated middleware bridge contract with explicit normalization rules.
- Added dedicated normalization policy document.
- Updated roadmap, feature sequence, and README CN/EN references.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M13-dashboard-jwt-session-source-provenance
