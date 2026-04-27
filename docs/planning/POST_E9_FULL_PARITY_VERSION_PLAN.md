# Post-E9 Full Parity Version Plan (Skoll vs HisiPHP + Gin-Vue-Admin)

## Goal

Starting from E9, continue versioned iterations until Skoll reaches functional parity with the capability set referenced from HisiPHP and Gin-Vue-Admin.

Parity definition in this document:

- Capability parity means each key feature area has production-usable implementation, API contract, tests, and operation docs.
- Parity does not require identical internal architecture; it requires equivalent user-facing capability and governance outcomes.

## Scope of Full Parity

- Admin authentication with JWT login/refresh/revoke/session lifecycle.
- RBAC policy depth and persistent grants.
- Dynamic route/menu/button permission lifecycle.
- User/role/menu/API/dictionary/config/audit operational completeness.
- File service, scheduler, generator, plugin lifecycle and marketplace safety.
- Database operations governance and operational safety.
- Multi-instance consistency and shared-state reliability.
- Release governance with benchmark regression gates and evidence closure.
- HisiPHP-oriented extension points: module management and hook/event governance.

## Version Track (Post-E9)

### E10-auth-and-session-parity

- Target parity area:
  - Gin-Vue-Admin auth experience parity: login, refresh token lifecycle, revoke/logout, session introspection.
- Deliverables:
  - JWT login and refresh flow contract under admin APIs.
  - Session token revocation list and expiry strategy.
  - Auth failure reason normalization and alert hooks.
- Acceptance:
  - Auth flow e2e covers login -> refresh -> revoke -> denied access.
  - Race and baseline benchmarks for auth middleware paths.

### E11-rbac-and-permission-governance-parity

- Target parity area:
  - Casbin-style policy depth and operational governance.
- Deliverables:
  - Persistent role-policy grants and versioned policy snapshots.
  - Permission diff/check tooling for route/menu/button/API/data-scope.
  - Policy rollback contract and audit linkage.
- Acceptance:
  - Policy migration rollback passes contract tests.
  - No privilege-escalation in regression matrix.

### E12-admin-domain-operational-parity

- Target parity area:
  - Admin domain modules maturity (user/role/menu/config/dict/audit) to production operations level.
- Deliverables:
  - Bulk operations and safe validation contracts.
  - Query/filter/index strategy for audit and dictionary/config reads.
  - Admin operation SLA and retention policy docs.
- Acceptance:
  - All domain modules have persistence-backed adapter tests.
  - p95 query latency target documented and measured.

### E13-hook-and-module-governance-parity

- Target parity area:
  - HisiPHP-style module and hook management capability.
- Deliverables:
  - Hook/event registration, ordering, and isolation contract.
  - Module lifecycle governance: install, enable, disable, remove, compatibility check.
  - Hook execution audit and failure containment strategy.
- Acceptance:
  - Hook failures do not crash core control-plane path.
  - Module lifecycle operations are idempotent and recoverable.

### E14-plugin-marketplace-depth-parity

- Target parity area:
  - Marketplace and online upgrade depth parity.
- Deliverables:
  - Signed plugin index metadata ingestion and trust policy.
  - Dependency graph solver with conflict diagnostics.
  - Multi-step upgrade transaction with checkpoint rollback.
- Acceptance:
  - Simulated dependency conflict and rollback scenarios pass.
  - Provenance and trust evidence retained in audit trail.

### E15-database-management-depth-parity

- Target parity area:
  - HisiPHP database management depth.
- Deliverables:
  - Migration history inventory and drift detection.
  - Backup catalog metadata and restore rehearsal workflow.
  - Controlled SQL policy classes (read-only, write-guarded, destructive-confirmed).
- Acceptance:
  - Backup/restore drill success criteria and RTO evidence documented.
  - Destructive SQL safeguards enforce dual confirmation policy.

### E16-distributed-scheduler-and-job-reliability-parity

- Target parity area:
  - Enterprise job scheduling reliability under multi-instance deployment.
- Deliverables:
  - Distributed scheduler claim/lease renewal model.
  - Retry, backoff, dead-letter, and operator replay workflows.
  - Job execution observability and SLO metrics.
- Acceptance:
  - Duplicate execution prevention validated under concurrent stress.
  - Retry and dead-letter flows are operable from admin APIs.

### E17-observability-and-operational-hardening-parity

- Target parity area:
  - Production hardening and operations parity.
- Deliverables:
  - Control-plane rate limits, timeout policy, and circuit guards.
  - Operational runbooks for incident classes (auth, plugin, db, scheduler).
  - Alert profiles and escalation matrix.
- Acceptance:
  - Staging fault drills produce reproducible mitigation evidence.
  - Hardening rules are feature-flagged and rollback-ready.

### E18-release-governance-and-parity-closure

- Target parity area:
  - Final parity closure against both reference projects.
- Deliverables:
  - Capability-by-capability parity checklist with evidence links.
  - Release scorecard automation with benchmark regression blocking.
  - Public parity closure report and compatibility statement.
- Acceptance:
  - All parity checklist items marked complete with proof.
  - Milestone review outputs pass DoD and regression-risk criteria.

## Recommended Commit Sequence

1. E10-auth-and-session-parity
2. E11-rbac-and-permission-governance-parity
3. E12-admin-domain-operational-parity
4. E13-hook-and-module-governance-parity
5. E14-plugin-marketplace-depth-parity
6. E15-database-management-depth-parity
7. E16-distributed-scheduler-and-job-reliability-parity
8. E17-observability-and-operational-hardening-parity
9. E18-release-governance-and-parity-closure

## Milestone Validation Gates

For each post-E9 milestone:

- go fmt ./...
- go test ./...
- go test -race ./... (mandatory when state sharing/concurrency changed)
- go test -bench=. -benchmem ./... (or target benchmark suite)

Required evidence in milestone record:

- baseline vs current performance data with percentage delta
- regression risk list (critical/medium/low)
- README CN/EN parity status

## Exit Criteria (Full Parity)

- E10-E18 all completed and committed milestone-by-milestone.
- Capability matrix in feature plan has no remaining non-parity gap item.
- Release governance scorecard blocks threshold-breaking regressions.
- CN/EN documentation parity maintained for public usage and operations guidance.
