# M20-dashboard-jwt-session-provenance-exception-governance

## Summary

This milestone slice adds exception governance matrix and expiry revalidation workflow for JWT provenance threshold-control operations.

## Delivered

- Added exception governance and revalidation guide:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_GOVERNANCE.md`
- Included:
  - exception governance matrix by risk and validity window
  - exception record template
  - expiry revalidation workflow and decision rules
  - governance and metric suggestions
- Linked exception governance from review automation, threshold-change log, and SLO policy docs.
- Synced roadmap, feature sequence, and README CN/EN indexes.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M20-dashboard-jwt-session-provenance-exception-sample-log
