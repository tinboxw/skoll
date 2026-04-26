# M16-dashboard-jwt-session-provenance-slo-alert-rules

## Summary

This milestone slice adds SLO alert rule templates and rollout guardrails for dashboard JWT provenance reliability signals.

## Delivered

- Added alert rule template guide:
  - `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_ALERT_RULES.md`
- Added PromQL templates for:
  - invalid verification surge
  - burn-rate critical / at-risk
  - claims-version-missing drift
  - provenance chain-depth drift
- Added rollout guardrails for canary gating, freeze policy, and escalation thresholds.
- Linked alert-rules doc from SLO policy.
- Updated roadmap, feature sequence, and README CN/EN documentation indexes.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Next

- M17-dashboard-jwt-session-provenance-baseline-recalibration
