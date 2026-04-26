# M17-dashboard-jwt-session-provenance-recalibration-evidence-template

## Summary

This milestone slice adds a reusable evidence template and approval checklist for dashboard JWT provenance recalibration changes.

## Delivered

- Added recalibration evidence template document:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_RECALIBRATION_EVIDENCE_TEMPLATE.md`
- Included:
  - standard evidence record structure
  - approval checklist (service owner + security on-call)
  - post-rollout verification template (D+7)
  - governance notes for version-controlled approvals
- Linked template from related policy/workflow docs.
- Synced roadmap, feature sequence, and README CN/EN indexes.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M18-dashboard-jwt-session-provenance-threshold-change-log
