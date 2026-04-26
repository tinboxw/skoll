# Dashboard JWT Provenance SLO Alert Rules

## Purpose

Define alert rule templates and rollout guardrails for JWT provenance SLO/error-budget signals.

These rules complement:

- `docs/community/DASHBOARD_JWT_PROVENANCE_SLO_POLICY.md`
- `docs/community/DASHBOARD_JWT_PROVENANCE_OPS_RUNBOOK.md`

## Signals

Primary metrics:

- `skoll_dashboard_jwt_provenance_exports_total`
- `skoll_dashboard_jwt_provenance_verification_states_total`
- `skoll_dashboard_jwt_provenance_alert_hints_total`

## Alert Templates (PromQL)

### 1) Invalid Verification Surge (High)

```promql
sum(rate(skoll_dashboard_jwt_provenance_verification_states_total{state="invalid"}[5m]))
/
clamp_min(sum(rate(skoll_dashboard_jwt_provenance_exports_total{enabled="true"}[5m])), 1)
> 0.02
```

Intent:

- Detect sudden parser/verification failures above 2% in recent traffic.

### 2) Burn Rate Critical (Critical)

```promql
(
  sum(rate(skoll_dashboard_jwt_provenance_verification_states_total{state=~"invalid|unverified"}[5m]))
  /
  clamp_min(sum(rate(skoll_dashboard_jwt_provenance_exports_total{enabled="true"}[5m])), 1)
)
/
0.01
> 2
```

Intent:

- Alert when SLO burn rate exceeds 2x budget.

### 3) Burn Rate At Risk (Warning)

```promql
(
  sum(rate(skoll_dashboard_jwt_provenance_verification_states_total{state=~"invalid|unverified"}[15m]))
  /
  clamp_min(sum(rate(skoll_dashboard_jwt_provenance_exports_total{enabled="true"}[15m])), 1)
)
/
0.01
> 1
```

Intent:

- Early warning when budget burn exceeds 1x.

### 4) Claims Version Missing Drift (Medium)

```promql
sum(rate(skoll_dashboard_jwt_provenance_alert_hints_total{hint="claims_version_missing"}[10m]))
/
clamp_min(sum(rate(skoll_dashboard_jwt_provenance_exports_total{enabled="true"}[10m])), 1)
> 0.1
```

Intent:

- Detect rollout inconsistency where claims version propagation regresses.

### 5) Chain Depth Drift (Medium)

```promql
sum(rate(skoll_dashboard_jwt_provenance_alert_hints_total{hint="provenance_chain_depth_high"}[10m]))
/
clamp_min(sum(rate(skoll_dashboard_jwt_provenance_exports_total{enabled="true"}[10m])), 1)
> 0.05
```

Intent:

- Detect middleware topology anomalies and duplicated hops.

## Rollout Guardrails

1. Canary-first policy:
   - Enable alert routes for canary slice before full rollout.
2. Guardrail gate:
   - Block full rollout when burn-rate critical alert fires in canary for >= 10m.
3. Version gate:
   - Block rollout if `claims_version_missing` drift remains > 10% for >= 15m.
4. Freeze policy:
   - If error-budget consumed ratio >= 2, enforce deployment freeze until mitigation evidence is recorded.
5. Escalation policy:
   - For sustained at-risk burn (>1x for >= 30m), page service owner and security on-call.

## Alert Routing Suggestions

- Critical: Pager + incident channel + deployment pipeline stop.
- High: Pager + ops channel.
- Medium: Ops channel + issue creation.
- Warning: Dashboard annotation + daily review.

## Tuning Guidance

1. Start with conservative thresholds in staging/canary.
2. Recalibrate per traffic profile and seasonality.
3. Keep alert sensitivity and suppression windows version-controlled.
4. Every threshold change must include before/after baseline evidence.
