# M21-dashboard-jwt-session-provenance-exception-metrics

## Summary

This milestone slice adds exception governance observability metrics and monthly trend dashboard fields for JWT provenance operations.

## Delivered

- Added metrics and trend-field guide:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_EXCEPTION_METRICS.md`
- Included:
  - metric definitions and label recommendations
  - monthly trend dashboard field contract
  - derived ratio formulas and payload example
  - alerting hints and data quality rules
- Linked metrics guide from exception governance and SLO policy docs.
- Synced roadmap, feature sequence, and README CN/EN indexes.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M21-dashboard-jwt-session-provenance-exception-alert-profiles
