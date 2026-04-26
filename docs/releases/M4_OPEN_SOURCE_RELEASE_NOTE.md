# M4 Open Source Release Note

## Overview

This release packages the Skoll framework baseline from M0 to M4, focused on runtime hardening, admin auth security, and release readiness.

## Functional Matrix

| Area | Capability | Status | Notes |
| --- | --- | --- | --- |
| Runtime | Graceful drain and shutdown sequence | Ready | supports readiness switch and signal-aware drain interruption |
| Runtime | Runtime config snapshot logging | Ready | startup logs include effective runtime controls |
| Health | `/health` and `/ready` probes | Ready | stable contracts for orchestration |
| Observability | `/metrics` and request counters | Ready | bounded route labels and status counters |
| Admin | Optional `/admin/ping` probe | Ready | enabled only when go-admin integration is on |
| Admin Auth | `static-token` and `hmac-sha256` | Ready | pluggable verifier contract |
| Admin Auth | Replay defense and optional body hash | Ready | nonce one-time check and body hash validation |
| Admin Auth | Shared nonce store wiring | Ready | Redis-backed replay defense for multi-instance setup |
| Security Ops | Incident and rotation runbook | Ready | operation baseline documented |

## Compatibility Strategy

1. Endpoint compatibility

- Existing probe contracts stay stable (`/health`, `/ready`, `/metrics`).
- `/admin/ping` remains optional and tied to go-admin integration toggle.

1. Auth compatibility

- `auto` mode keeps backward behavior for token-first fallback.
- Production guard blocks `static-token` by default in prod mode unless explicitly allowed.

1. Configuration compatibility

- Flag and environment-based controls are both supported.
- Invalid critical runtime config fails fast at startup.

## Upgrade Notes

1. Production deployments should use HMAC mode and shared Redis nonce store baseline.
2. If you previously used static token in prod mode, explicitly migrate to HMAC mode or set a temporary allow flag during transition.
3. Review runtime startup logs after upgrade to confirm effective config values.
4. Re-run benchmark checks after deployment using the benchmark toolchain policy.

## Validation Snapshot

- `go fmt ./...`: pass
- `go test ./...`: pass
- `go test -race ./...`: pass
- benchmark evidence recorded in `docs/milestones/M4-release-checklist.md`

## Known Risks

- Shared nonce store remains a required deployment control in multi-instance production.
- Benchmark outputs can jitter; release decisions should use median and sample range evidence.

## References

- `docs/milestones/M4-release-checklist.md`
- `docs/milestones/M4-final-signoff-review.md`
- `docs/planning/ADMIN_AUTH_SECURITY_RUNBOOK.md`
- `docs/planning/BENCHMARK_TOOLCHAIN_POLICY.md`
- `docs/planning/PRODUCTION_ENV_TEMPLATE.md`
