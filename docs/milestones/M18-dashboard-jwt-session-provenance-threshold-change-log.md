# M18-dashboard-jwt-session-provenance-threshold-change-log

## Summary

This milestone slice adds a standard threshold-change log format and monthly archive workflow for dashboard JWT provenance alert tuning governance.

## Delivered

- Added threshold-change log standard:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_THRESHOLD_CHANGE_LOG.md`
- Included:
  - required immutable fields for every threshold change
  - YAML example record
  - monthly review archive workflow
  - monthly archive markdown template
  - governance rules for append-only corrections
- Linked threshold-change log doc from policy/workflow documents.
- Synced roadmap, feature sequence, and README CN/EN indexes.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M18-dashboard-jwt-session-provenance-archive-sample
