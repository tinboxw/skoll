# M22-dashboard-jwt-session-provenance-governance-scorecard-template

## Summary

This milestone slice adds a monthly governance scorecard template and decision readiness indicators for JWT provenance exception governance.

## Delivered

- Added governance scorecard template and readiness rules:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_GOVERNANCE_SCORECARD_TEMPLATE.md`
- Included:
  - monthly scorecard sections and metric snapshot schema
  - readiness indicator pass/fail checks
  - decision board options (`proceed`, `proceed_with_guardrails`, `freeze_changes`)
  - evidence requirements and governance constraints
- Linked scorecard template from alert profile, review automation, and SLO policy docs.
- Synced roadmap, feature sequence, and README CN/EN indexes.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M22-dashboard-jwt-session-provenance-governance-scorecard-sample
