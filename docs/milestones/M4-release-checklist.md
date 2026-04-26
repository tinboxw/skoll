# M4-release-checklist

## Release Readiness Summary

- Scope: M0 through current M4 slices.
- Status: Complete (all mandatory gates passed with linked evidence).

## Mandatory Gates

- [x] Code format: `go fmt ./...`
- [x] Unit tests: `go test ./...`
- [x] Race tests: `go test -race ./...`
- [x] Bench evidence: `go test -run ^$ -bench "." -benchmem -count 3 ./internal/app`
- [x] Bench evidence: `go test -run ^$ -bench "." -benchmem -count 3 ./internal/service`
- [x] README bilingual parity checked

## Validation Output Snapshot

- final gate recheck (2026-04-24):
  - `go fmt ./...`: PASS
  - `go test ./...`: PASS
  - `go test -race ./...`: PASS
- final benchmark recheck (`count 3`, 2026-04-24):
  - app samples: `944.6`, `1033`, `930.6 ns/op`; median: `944.6 ns/op`
  - service samples: `2.404`, `2.422`, `2.429 ns/op`; median: `2.422 ns/op`
- `go test ./...`: PASS (`cmd/skoll`, `internal/app`, `internal/integration/goadmin`, `internal/service`)
- `go test -race ./...`: PASS
- app benchmark samples (`count 10`):
  - `889.1`, `850.6`, `916.6`, `881.8`, `930.8`, `878.6`, `857.7`, `888.3`, `908.2`, `899.4 ns/op`
- app benchmark median: `888.7 ns/op`, `1094 B/op`, `12 allocs/op`
- app sample range: `850.6 ~ 930.8 ns/op`
- service benchmark samples (`count 10`):
  - `2.316`, `2.401`, `2.478`, `2.405`, `2.487`, `2.529`, `2.490`, `2.550`, `2.612`, `2.543 ns/op`
- service benchmark median: `2.4885 ns/op`, `0 B/op`, `0 allocs/op`
- service sample range: `2.316 ~ 2.612 ns/op`
- stability cross-check (`count 3` vs `count 10`):
  - app median delta: `-13.04%`
  - service median delta: `-9.67%`
- reproducibility note:
  - `benchstat@latest` currently requires a newer Go toolchain in this environment.
  - this checkpoint uses `count 10` medians and sample ranges as the release evidence baseline.

## Key Milestone Evidence

- M0 baseline: `docs/milestones/M0-project-baseline.md`
- M1 core domain: `docs/milestones/M1-core-domain.md`
- M2 concurrency/performance: `docs/milestones/M2-concurrency-and-performance.md`
- M3 observability/hardening: `docs/milestones/M3-observability-and-hardening.md`
- M3 go-admin minimal integration: `docs/milestones/M3-go-admin-minimal-integration.md`
- M3 go-admin probe hook: `docs/milestones/M3-go-admin-probe-hook.md`
- M3 graceful drain readiness: `docs/milestones/M3-graceful-drain-readiness.md`
- M3 drain config guard: `docs/milestones/M3-drain-config-guard.md`
- M3 runtime env overrides: `docs/milestones/M3-runtime-env-overrides.md`
- M3 signal-aware drain sequence: `docs/milestones/M3-shutdown-sequence-and-signal-aware-drain.md`
- M3 metrics route template: `docs/milestones/M3-metrics-route-template.md`
- M4 admin auth skeleton: `docs/milestones/M4-admin-auth-skeleton.md`
- M4 admin pluggable auth interface: `docs/milestones/M4-admin-auth-pluggable-interface.md`
- M4 admin HMAC auth: `docs/milestones/M4-admin-auth-hmac-sha256.md`
- M4 admin HMAC replay/body-hash hardening: `docs/milestones/M4-admin-auth-hmac-replay-bodyhash.md`
- M4 admin nonce-store interface: `docs/milestones/M4-admin-auth-nonce-store-interface.md`
- M4 admin shared nonce-store wiring: `docs/milestones/M4-admin-auth-shared-nonce-store-wiring.md`
- M4 admin auth observability: `docs/milestones/M4-admin-auth-observability.md`
- M4 admin auth security runbook: `docs/milestones/M4-admin-auth-security-runbook.md`
- M4 admin prod static-token guard: `docs/milestones/M4-admin-auth-prod-static-token-guard.md`
- M4 final signoff review: `docs/milestones/M4-final-signoff-review.md`

## Current Risks

- Medium: multi-instance replay protection still requires explicit shared nonce store configuration in deployment.
- Medium: microbenchmark output still has short-window jitter; release decision is based on `count 10` medians and range evidence.
- Low: admin auth failure counters add bounded reason labels and should stay stable unless reason taxonomy changes.
- Low: route templating policy currently covers admin prefix only.
- Low: runbook requires periodic ownership review to remain operationally accurate.

## Release Decision

- Current decision: Pass.
- Blocking items before final release sign-off:
  1. None.
