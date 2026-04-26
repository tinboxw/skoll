# M17-dashboard-jwt-session-provenance-baseline-recalibration

## Summary

This milestone slice adds the periodic baseline recalibration workflow and review cadence for dashboard JWT provenance SLO alerting.

## Delivered

- Added baseline recalibration workflow and governance document:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_BASELINE_RECALIBRATION.md`
- Defined review cadence:
  - weekly light review
  - monthly baseline review
  - quarterly policy review
  - event-driven review after architecture changes
- Defined threshold change workflow:
  - evidence collection
  - segmented analysis
  - safety checks
  - canary trial
  - joint sign-off
  - staged rollout and post-rollout verification
- Added decision rules, rollback triggers, and evidence checklist.
- Synced roadmap, feature plan, and README CN/EN indexes.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M17-dashboard-jwt-session-provenance-recalibration-evidence-template
