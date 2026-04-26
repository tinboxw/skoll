# Skoll Implementation Roadmap

## Milestone Plan

- M0-project-baseline: Initialize minimal runnable Go service baseline.
- M1-core-domain: Introduce core domain modules and service boundaries.
- M2-concurrency-and-performance: Add concurrency primitives and benchmark-driven tuning.
- M3-observability-and-hardening: Add logging/metrics/tracing and production hardening.
- M6-admin-api-and-rbac-foundation: Align baseline admin capabilities with HisiPHP/Gin-Vue-Admin common set.
- M7-persistence-config-and-audit: Move admin modules from scaffold state to persistent operational baseline.
- M8-assets-jobs-and-generator: Add file capabilities, scheduler, and module generation support.
- M9-plugin-ecosystem-readiness: Build extension lifecycle and marketplace-facing compatibility surface.

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
23. [x] M6-admin-module-api-baseline: Expose `/admin/v1` APIs for user/role/menu/audit scaffolds and wire optional admin auth wrapper.
24. [x] M6-role-menu-binding: Add role-menu binding APIs and DTO contract tests.
25. [x] M6-role-api-binding: Add API resource registry and role-api permission binding APIs.
26. [x] M6-api-registry-and-permission-assignment: Add API registry listing endpoint and enforce registered-api binding policy.
27. [x] M6-rbac-e2e-smoke: Add end-to-end tests for `identity -> role -> route/api authorization` baseline flow.
28. [x] M7-storage-adapter-contract: Introduce storage adapter interfaces and shared contract tests.
29. [x] M7-config-and-dictionary: Implement config center and dictionary APIs with persistence.
30. [x] M7-durable-audit-log: Add paginated/filterable persistent audit logs.
31. [x] M8-file-service-baseline: Add file upload/download module with pluggable backend.
32. [x] M8-job-scheduler-baseline: Add scheduler APIs with execution history and observability.
33. [x] M8-module-generator: Add CRUD module generator with template governance.
34. [x] M9-plugin-manifest-lifecycle: Define plugin manifest and lifecycle hooks.
35. [x] M9-extension-packaging: Add install/enable/disable/version-check flow for extension packages.
36. [x] M9-ecosystem-docs: Publish extension developer guide and compatibility policy.
37. [x] M10-system-status-api: Add `/admin/v1/system/status` observability baseline endpoint.
38. [x] M10-dashboard-runtime-metrics: Add admin dashboard runtime metric snapshot endpoint.
39. [x] M10-dashboard-node-health: Add admin node/dependency health summary endpoint.
40. [x] M10-dashboard-aggregation: Add combined dashboard aggregate endpoint for UI bootstrap.
41. [x] M10-dashboard-ui-bootstrap-contract: Define stable dashboard payload contract for frontend integration.
42. [x] M11-dashboard-auth-session-alignment: Align dashboard bootstrap with auth/session capability plan.
43. [x] M11-dashboard-auth-session-policy-docs: Publish auth/session compatibility and rollout guidance.
44. [x] M11-dashboard-auth-session-observability: Add auth/session observability fields and failure counters to dashboard payload.
45. [x] M11-dashboard-auth-session-actionability: Add actionable auth/session guidance fields for dashboard UX.
46. [x] M12-dashboard-jwt-session-bootstrap: Introduce JWT session bootstrap fields for dashboard identity context.
47. [x] M12-dashboard-jwt-session-refresh-hints: Add JWT refresh and expiry hint fields for dashboard UX.
48. [x] M12-dashboard-jwt-session-verification-state: Add JWT verification-state hints for dashboard trust messaging.
49. [x] M12-dashboard-jwt-session-middleware-bridge: Expose verified-claims bridge fields for middleware integration.
50. [x] M12-dashboard-jwt-session-bridge-contract-docs: Publish middleware bridge rollout and compatibility docs.
51. [x] M13-dashboard-jwt-session-verified-claims-adapter: Align verified claims adapter interfaces for dashboard and RBAC bridge.
52. [x] M13-dashboard-jwt-session-claims-normalization: Add shared claim normalization policy for role/subject extraction.
53. [x] M13-dashboard-jwt-session-source-provenance: Add claim source provenance fields for multi-middleware tracing.
54. [x] M14-dashboard-jwt-session-provenance-audit-export: Add provenance audit export fields for security operations.
55. [x] M14-dashboard-jwt-session-provenance-audit-docs: Publish provenance audit export compatibility and SIEM mapping guidance.
56. [x] M15-dashboard-jwt-session-provenance-ops-metrics: Add provenance export operational metrics and alerting hints.
57. [x] M15-dashboard-jwt-session-provenance-ops-runbook: Add provenance ops runbook and alert triage guidance.
58. [x] M16-dashboard-jwt-session-provenance-slo-dashboards: Add provenance SLO dashboards and error-budget policy.
59. [x] M16-dashboard-jwt-session-provenance-slo-alert-rules: Add provenance SLO alert rule templates and rollout guardrails.
60. [x] M17-dashboard-jwt-session-provenance-baseline-recalibration: Add periodic baseline recalibration workflow and review cadence.
61. [x] M17-dashboard-jwt-session-provenance-recalibration-evidence-template: Add reusable recalibration evidence template and approval checklist.
62. [x] M18-dashboard-jwt-session-provenance-threshold-change-log: Add threshold-change log format and monthly review archive workflow.
63. [x] M18-dashboard-jwt-session-provenance-archive-sample: Add first monthly archive sample and review checklist execution example.
64. [x] M19-dashboard-jwt-session-provenance-review-automation: Add monthly review automation checklist and ownership rotation guidance.
65. [ ] M19-dashboard-jwt-session-provenance-rotation-roster-sample: Add quarterly rotation roster sample and escalation handoff template.
