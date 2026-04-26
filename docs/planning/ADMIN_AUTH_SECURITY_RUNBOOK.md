# Admin Auth Security Runbook

## Scope

This runbook defines operational security practices for admin auth in Skoll, including secret lifecycle, incident response, and deployment guardrails.

## Runtime Modes and Recommended Use

- `none`
  - Use only for local development in trusted environments.
- `static-token`
  - Use only for internal/dev bootstrap scenarios.
  - Not recommended as a long-term production control plane auth mode.
- `hmac-sha256`
  - Recommended baseline for production admin probe protection.

## Required Deployment Baseline

- Enforce HTTPS for all admin routes.
- Keep host clocks synchronized via NTP.
- Restrict admin route network exposure (ingress allowlist/private network).
- Use the production env template as baseline input:
  - `.env.production.example`
  - `docs/planning/PRODUCTION_ENV_TEMPLATE.md`
- For multi-instance deployments, use shared nonce store:
  - `SKOLL_ADMIN_AUTH_NONCE_STORE=redis`
  - `SKOLL_ADMIN_AUTH_NONCE_REDIS_ADDR=<redis-host:port>`
- Monitor these metrics at minimum:
  - `skoll_admin_auth_verifications_total{mode,result}`
  - `skoll_admin_auth_failures_total{mode,reason}`

## Secret Rotation Policy

### Rotation Frequency

- Rotate `SKOLL_ADMIN_AUTH_HMAC_SECRET` every 90 days or earlier based on risk signals.
- Immediate rotation is required after any suspected leak or unauthorized admin request burst.

### Planned Rotation Procedure

1. Generate a new high-entropy secret in your secret manager.
2. Roll out to all instances via deployment pipeline.
3. Restart or reload services so all instances use the new secret.
4. Validate health and auth behavior:
   - `curl /health` and `curl /ready`
   - Signed `curl /admin/ping` with new secret.
5. Monitor auth failure reasons for anomalies for at least one replay-window cycle.
6. Revoke old secret from secret manager and CI/CD variables.

### Emergency Rotation Procedure

1. Declare incident and freeze non-essential deployments.
2. Rotate secret immediately in secret manager.
3. Roll out updated secret to all regions/instances.
4. Verify new signed requests pass and old signatures fail.
5. Retain evidence (timeline, logs, metrics snapshots) for postmortem.

## Incident Response Playbook

### Trigger Conditions

- Sudden rise in `skoll_admin_auth_failures_total`.
- Repeated `replay_nonce` failures from multiple source addresses.
- Unexpected spikes in `invalid_signature` or `missing_headers`.
- Signs of leaked secret or unauthorized admin route access.

### Triage Steps

1. Confirm impact scope:
   - affected environments, regions, and time window.
2. Inspect reason-specific counters and request logs.
3. Correlate with recent deploy/config changes.
4. Verify Redis nonce store health for multi-instance setups.

### Containment Steps

1. Restrict admin route traffic at ingress/WAF.
2. Rotate `SKOLL_ADMIN_AUTH_HMAC_SECRET`.
3. If needed, temporarily disable admin integration in non-essential environments.

### Recovery Steps

1. Restore intended ingress policy.
2. Re-enable normal admin probe workflow.
3. Confirm failure counters return to expected baseline.

### Post-Incident Actions

- Record root cause and remediation in internal incident tracker.
- Add detection/alert tuning for the observed failure pattern.
- Review whether static-token usage remains in any sensitive environment.

## Alerting Recommendations

- Alert on sustained auth failure ratio growth:
  - `failure / (success + failure)` above configured threshold.
- Alert on sudden `replay_nonce` spikes.
- Alert on `missing_headers` spikes that may indicate probing/scanning.
- Use rate-based windows (for example 5m/15m) to reduce noise.

## Configuration Checklist

- `SKOLL_ADMIN_AUTH_MODE=hmac-sha256` in production.
- `SKOLL_ADMIN_AUTH_HMAC_SECRET` stored in secret manager only.
- `SKOLL_ADMIN_AUTH_NONCE_STORE=redis` for multi-instance deployments.
- `SKOLL_ADMIN_AUTH_NONCE_REDIS_ADDR` points to highly available Redis.
- `SKOLL_ADMIN_AUTH_NONCE_REDIS_PASSWORD` configured when required.
- No admin secrets committed to repository or logged in plain text.

## Operational Verification Commands

```bash
# Basic probes
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/metrics

# Signed admin probe (sample)
TS=$(date +%s)
NONCE="nonce-$(openssl rand -hex 8)"
PAYLOAD="GET\n/admin/ping\n${TS}\n${NONCE}\n"
SIG=$(printf "%b" "$PAYLOAD" | openssl dgst -sha256 -hmac "${SKOLL_ADMIN_AUTH_HMAC_SECRET}" -binary | xxd -p -c 256)

curl \
  -H "X-Admin-Timestamp: ${TS}" \
  -H "X-Admin-Nonce: ${NONCE}" \
  -H "X-Admin-Signature: ${SIG}" \
  http://localhost:8080/admin/ping
```

## Ownership

- Security policy owner: platform/security team.
- Runtime implementation owner: skoll contributors.
- On-call responders: service SRE/operations roster.
