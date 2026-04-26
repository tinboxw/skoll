# Skoll Parity Enhancement Phase Plan (Post-M22)

## Goal

After completing the current JWT provenance governance documentation track (through M22-step2), start a new enhancement phase to narrow the remaining parity gap against Gin-Vue-Admin and selected HisiPHP operational capabilities.

## Scope

This phase focuses on feature-depth enhancement and production-governance hardening:

- authorization depth (policy engine and data-scope model)
- dynamic route/menu/button permission contract evolution
- account/session security hardening
- generator ecosystem depth (including form model support)
- plugin marketplace safety and upgrade rollback
- database operations governance
- multi-instance consistency for sessions/jobs
- release governance and performance-regression gates

## Milestone Packages

### E1-policy-engine-and-data-scope

- Feature package:
  - strengthen policy model (RBAC + resource/data scope)
  - introduce policy snapshot/reload and audit linkage
- Acceptance:
  - no privilege escalation in e2e authorization cases
  - policy change events traceable in audit records
- Primary risks:
  - policy migration compatibility and authorization regressions

### E2-dynamic-route-menu-button-permission

- Feature package:
  - dynamic route/menu/button-level permission contract
  - contract versioning for frontend compatibility
- Acceptance:
  - menu-route-button consistency checks pass
  - backward-compatible contract behavior validated
- Primary risks:
  - contract drift between backend payload and frontend assumptions

### E3-account-session-security-hardening

- Feature package:
  - password policy and rotation constraints
  - failed-login lock strategy
  - mfa extension point baseline
  - session revoke and anomaly handling contract
- Acceptance:
  - lock/revoke scenarios covered by tests
  - security events appear in audit pipeline
- Primary risks:
  - over-strict controls reducing usability

### E4-generator-ecosystem-depth

- Feature package:
  - form schema support for generator workflows
  - template governance and compatibility metadata
- Acceptance:
  - generated modules and form payloads pass baseline tests
  - template compatibility policy documented
- Primary risks:
  - template evolution breaking generated code compatibility

### E5-plugin-market-and-online-upgrade-safety

- Feature package:
  - plugin signature verification baseline
  - dependency and version precheck contracts
  - install/upgrade rollback procedure
- Acceptance:
  - failed upgrade rollback evidence produced
  - plugin lifecycle tests cover precheck + rollback path
- Primary risks:
  - inconsistent state across partial upgrade failures

### E6-database-ops-governance

- Feature package:
  - migration orchestration baseline
  - backup/restore workflow contract
  - controlled SQL execution and operation audit
- Acceptance:
  - backup/restore drill evidence documented
  - dangerous operations require explicit safeguards
- Primary risks:
  - data integrity and recovery-time uncertainty

### E7-multi-instance-consistency-hardening

- Feature package:
  - shared-state strategy for sessions/jobs
  - idempotency and duplicate-execution prevention patterns
- Acceptance:
  - multi-instance execution consistency scenarios validated
  - race and stress checks report no critical defects
- Primary risks:
  - distributed lock contention and timing edge cases

### E8-release-governance-closure

- Feature package:
  - release scorecard and gating automation contracts
  - performance regression baseline and trend tracking
- Acceptance:
  - each release candidate includes measurable gate evidence
  - rollback/degradation criteria documented
- Primary risks:
  - noisy metrics causing incorrect release decisions

## Commit Batches

Recommended one-milestone-one-commit sequence:

1. `E1-policy-engine-and-data-scope`
2. `E2-dynamic-route-menu-button-permission`
3. `E3-account-session-security-hardening`
4. `E4-generator-ecosystem-depth`
5. `E5-plugin-market-and-online-upgrade-safety`
6. `E6-database-ops-governance`
7. `E7-multi-instance-consistency-hardening`
8. `E8-release-governance-closure`

## Validation Baseline

Each milestone must include:

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...` (required for concurrent/state-sharing changes)
- benchmark or measurable performance data for changed critical paths

## Documentation and Parity Rules

For each milestone:

- update both `README.md` and `README.en.md` if usage/structure/commands change
- add one milestone record under `docs/milestones/`
- keep roadmap and feature plan in sync with completion state
