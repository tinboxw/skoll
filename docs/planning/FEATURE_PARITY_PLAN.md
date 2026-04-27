# Skoll Feature Parity Plan (HisiPHP + Gin-Vue-Admin)

## Goal

Build an open-source admin framework parity roadmap for Skoll by extracting common capabilities from HisiPHP and Gin-Vue-Admin, then implementing them in phased Go milestones.

## Source Baseline

- Gin-Vue-Admin README and docs highlight: JWT auth, Casbin RBAC, dynamic menus/routes, API permission management, user/role/menu management, dictionary/config management, upload/download, code generator, form generator, and operation features around system tools.
- HisiPHP README highlights: built-in permission management, module management, plugin management, hook management, database management, online upgrade, and application/plugin marketplace ecosystem.

## Capability Matrix

| Capability | HisiPHP | Gin-Vue-Admin | Skoll Current | Skoll Plan |
| --- | --- | --- | --- | --- |
| Admin authentication | Yes | Yes (JWT) | Yes (static-token/HMAC pluggable) | Add JWT session and refresh flow |
| RBAC (role-permission) | Yes | Yes (Casbin) | Scaffold only (in-memory role service) | Introduce policy engine + persistent grants |
| Dynamic menu and route policy | Yes | Yes | Scaffold only (in-memory menu service) | Add role-menu binding and route export API |
| User management | Yes | Yes | Scaffold only (in-memory user service) | Add CRUD + lifecycle + password policy |
| API resource permission | Yes | Yes | No | Add API registry and permission binding |
| Dictionary and config center | Partial | Yes | No | Add key-value config and dict APIs |
| Audit/operation log | Yes | Yes | Scaffold only (in-memory audit service) | Add durable audit pipeline and query filters |
| File upload/download | Yes | Yes | No | Add local/S3-compatible file service |
| Code/form generator | Plugin/extension | Yes | No | Add template-based module generator |
| Plugin/module marketplace model | Yes | Plugin ecosystem | No | Define module manifest and lifecycle hooks |
| Job scheduling | Common plugin capability | Common enterprise capability | No | Add scheduler and execution audit |
| System observability dashboard | Yes | Yes | Partial (/metrics) | Add admin system status APIs |
| Multi-instance safety | N/A | Common deployment pattern | Partial (Redis nonce store for admin auth) | Extend shared-state strategy for sessions/jobs |

## Milestone Plan

### M6-foundation-admin-api-and-rbac

- Deliverables:
  - Unified admin API namespace `/admin/v1/*` with consistent DTO and error envelopes.
  - User/role/menu/audit baseline APIs from module scaffolds.
  - Role-menu and role-api binding model (in-memory first, then storage adapter).
  - API registry introspection endpoint for permission assignment.
- Acceptance:
  - Core admin CRUD APIs reachable and covered by tests.
  - Admin auth wrapper can protect all `/admin/*` business routes.

### M7-persistence-config-and-audit

- Deliverables:
  - Storage abstraction and first persistent adapter (SQLite/MySQL selectable).
  - Config center and dictionary APIs.
  - Durable audit logs with paging/filter/search.
  - Operation-level audit event enrichment.
- Acceptance:
  - In-memory and persistent adapters pass the same contract tests.
  - Config/dict/audit endpoints documented and benchmarked.

### M8-assets-jobs-and-generator

- Deliverables:
  - File upload/download API (local backend first, object storage adapter later).
  - Scheduler API and execution history.
  - Starter code generator for module CRUD skeletons.
- Acceptance:
  - Generated module passes `go test ./...` baseline.
  - Job execution audit visible via admin API.

### M9-plugin-and-ecosystem-readiness

- Deliverables:
  - Module/plugin manifest spec and lifecycle hook contracts.
  - Marketplace-ready packaging baseline (install, enable, disable, version check).
  - Public extension guide and compatibility matrix.
- Acceptance:
  - Reference plugin can be loaded and managed through admin API.
  - Backward-compatibility strategy documented.

## Execution Sequence (Current Turn)

1. Completed: M6-step1 admin module API baseline.
2. Completed: M6-step2 role-menu and role-api binding endpoints.
3. Completed: M6-step3 API registry + permission assignment workflow.
4. Completed: M7-step1 storage adapter contract and in-memory adapter baseline.
5. Completed: M7-step2 config center and dictionary APIs with persistence-ready boundaries.
6. Completed: M7-step3 durable audit log query/filter/paging baseline.
7. Completed: M8-step1 file service baseline with pluggable backend.
8. Completed: M8-step2 job scheduler baseline with execution history.
9. Completed: M8-step3 module generator baseline.
10. Completed: M9-step1 plugin manifest lifecycle baseline.
11. Completed: M9-step2 extension packaging and lifecycle operations.
12. Completed: M9-step3 ecosystem docs and compatibility strategy.
13. Completed: M10-step1 system status API baseline.
14. Completed: M10-step2 runtime metrics snapshot API.
15. Completed: M10-step3 node and dependency health summary API.
16. Completed: M10-step4 dashboard aggregation API for UI bootstrap.
17. Completed: M10-step5 dashboard UI bootstrap contract definition.
18. Completed: M11-step1 dashboard auth/session alignment plan.
19. Completed: M11-step2 dashboard auth/session policy docs.
20. Completed: M11-step3 dashboard auth/session observability fields.
21. Completed: M11-step4 dashboard auth/session actionability fields.
22. Completed: M12-step1 dashboard JWT session bootstrap fields.
23. Completed: M12-step2 dashboard JWT refresh/expiry hints.
24. Completed: M12-step3 dashboard JWT verification-state hints.
25. Completed: M12-step4 dashboard JWT middleware bridge fields.
26. Completed: M12-step5 dashboard JWT middleware bridge contract docs.
27. Completed: M13-step1 dashboard JWT verified-claims adapter alignment.
28. Completed: M13-step2 dashboard JWT claims normalization policy.
29. Completed: M13-step3 dashboard JWT claim source provenance fields.
30. Completed: M14-step1 dashboard JWT provenance audit export fields.
31. Completed: M14-step2 dashboard JWT provenance audit export docs and SIEM mapping guidance.
32. Completed: M15-step1 dashboard JWT provenance export operational metrics and alerting hints.
33. Completed: M15-step2 dashboard JWT provenance ops runbook and alert triage guidance.
34. Completed: M16-step1 dashboard JWT provenance SLO dashboards and error-budget policy.
35. Completed: M16-step2 dashboard JWT provenance SLO alert rule templates and rollout guardrails.
36. Completed: M17-step1 dashboard JWT provenance baseline recalibration workflow and review cadence.
37. Completed: M17-step2 dashboard JWT provenance recalibration evidence template and approval checklist.
38. Completed: M18-step1 dashboard JWT provenance threshold-change log format and monthly review archive workflow.
39. Completed: M18-step2 dashboard JWT provenance monthly archive sample and review checklist execution example.
40. Completed: M19-step1 dashboard JWT provenance monthly review automation checklist and ownership rotation guidance.
41. Completed: M19-step2 dashboard JWT provenance quarterly rotation roster sample and escalation handoff template.
42. Completed: M20-step1 dashboard JWT provenance exception governance matrix and expiry revalidation workflow.
43. Completed: M20-step2 dashboard JWT provenance sample exception records and revalidation decision log template.
44. Completed: M21-step1 dashboard JWT provenance exception governance observability metrics and monthly trend dashboard fields.
45. Completed: M21-step2 dashboard JWT provenance governance metric alert profiles and escalation thresholds.
46. Completed: M22-step1 dashboard JWT provenance governance scorecard template and decision readiness indicators.
47. Completed: M22-step2 dashboard JWT provenance governance scorecard sample and review sign-off example.
48. Completed: startup orchestration refactor moved bootstrap wiring out of `cmd/skoll/main.go` into `internal/bootstrap`.
49. Completed: E1-step1 policy engine and data-scope authorization baseline.
50. Completed: E2-step1 dynamic route/menu/button permission contract versioning baseline.
51. Completed: E3-step1 account/session security hardening baseline.
52. Completed: E4-step1 generator ecosystem depth baseline.
53. Completed: E5-step1 plugin market and online upgrade safety baseline.
54. Completed: E6-step1 database ops governance baseline.
55. Completed: E7-step1 multi-instance consistency hardening baseline.
56. Completed: E8-step1 release governance closure baseline.
57. Completed: E9-step0 enhancement backlog prioritization and iteration planning baseline.
58. Next: E9-step1 persistence adapter rollout design.
59. Next: E9-step2 production hardening plan.
60. Next: E9-step3 benchmark regression governance policy.
61. Planned: E10 auth and session parity.
62. Planned: E11 RBAC and permission governance parity.
63. Planned: E12 admin domain operational parity.
64. Planned: E13 hook and module governance parity.
65. Planned: E14 plugin marketplace depth parity.
66. Planned: E15 database management depth parity.
67. Planned: E16 distributed scheduler and job reliability parity.
68. Planned: E17 observability and operational hardening parity.
69. Planned: E18 release governance and parity closure.

## Risks and Mitigation

- Risk: feature breadth slows core stabilization.
  - Mitigation: ship thin vertical slices with strict gate checks per milestone.
- Risk: API shape churn between in-memory and persistent backends.
  - Mitigation: lock transport DTO and validate with adapter contract tests.
- Risk: auth and permission model divergence.
  - Mitigation: keep auth (identity) and RBAC (authorization) interfaces explicitly separated.
