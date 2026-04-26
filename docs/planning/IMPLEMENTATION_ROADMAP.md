# Skoll Implementation Roadmap

## Milestone Plan

- M0-project-baseline: Initialize minimal runnable Go service baseline.
- M1-core-domain: Introduce core domain modules and service boundaries.
- M2-concurrency-and-performance: Add concurrency primitives and benchmark-driven tuning.
- M3-observability-and-hardening: Add logging/metrics/tracing and production hardening.

## M0 Scope

### Goal

Deliver a minimal baseline that can be built, run, tested, and benchmarked.

### Deliverables

- Go module initialization with stable package boundaries.
- Runnable entry point under cmd.
- Internal HTTP server with health and readiness probes.
- Basic tests plus benchmark snapshot command support.
- README CN/EN updates with consistent run/test commands.

### Non-Goals

- Business domain features.
- Persistent storage and external integrations.
- Full observability stack.

## Validation Gates

- go fmt ./...
- go test ./...
- go test -race ./... (for concurrent behavior changes)
- go test -bench=. -benchmem ./...

## Risks

- Early over-design may reduce iteration speed.
- Lack of baseline benchmarks can hide regressions.

## Mitigation

- Keep M0 intentionally small.
- Capture benchmark output on every milestone checkpoint.

## Next Steps Checklist (post-M3 slices)

1. [x] M3-shutdown-sequence-test: Add integration test for `ready -> not_ready -> drain -> shutdown` sequence.
2. [x] M3-signal-drain-context: Replace timer wait with context-aware wait to react to forced termination signals.
3. [x] M3-metrics-route-template: Add optional route templating/whitelist extension policy for future admin endpoints.
4. [x] M3-startup-config-snapshot: Print effective runtime config (flag/env resolved) at startup for operability.
5. [x] M4-admin-auth-skeleton: Introduce minimal auth boundary skeleton for admin route group (no business modules yet).
6. [x] M4-release-checklist: Consolidate benchmark/race evidence and publish release readiness notes.
7. [x] M4-admin-auth-pluggable-interface: Extract admin auth verifier contract and keep static-token as first implementation.
8. [x] M4-admin-auth-hmac-sha256: Add HMAC signature verifier as second pluggable admin auth mode.
9. [x] M4-admin-auth-hmac-replay-bodyhash: Add nonce replay protection and optional body hash validation for HMAC mode.
10. [x] M4-admin-auth-nonce-store-interface: Add pluggable nonce replay store interface while keeping in-memory default.
11. [x] M4-admin-auth-shared-nonce-store-wiring: Wire a shared nonce store implementation for multi-instance replay defense.
12. [x] M4-admin-auth-observability: Add auth verification metrics and failure-reason counters.
13. [x] M4-admin-auth-security-runbook: Publish secret rotation and incident response runbook for admin auth.
14. [x] M4-admin-auth-prod-static-token-guard: Block static-token in prod mode by default unless explicitly allowed.
15. [x] M4-final-signoff-review: Complete final DoD/risk/performance signoff with latest gate recheck evidence.
16. [x] M4-production-env-template: Add production baseline env template with Redis shared nonce store defaults.
17. [x] M4-benchmark-toolchain-policy: Add pinned benchstat execution path via fixed Go toolchain strategy.
18. [x] M4-open-source-release-note: Publish public release note with capability matrix, compatibility strategy, and upgrade notes.
19. [x] M5-initial-module-scaffolds: Add first module scaffolds (user/role/menu/audit) with tests and benchmark baselines.
20. [x] M5-benchmark-baseline: Capture initial module benchmark snapshots for future regression comparison.
21. [x] Community-contribution-kit: Add CONTRIBUTING and issue templates.
22. [x] Community-versioning-changelog-flow: Add version policy and changelog process documentation.
