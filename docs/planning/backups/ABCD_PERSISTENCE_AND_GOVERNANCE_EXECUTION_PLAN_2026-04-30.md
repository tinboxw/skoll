# Backup - ABCD Persistence and Governance Execution Plan

Backup snapshot date: 2026-04-30
Source: docs/planning/ABCD_PERSISTENCE_AND_GOVERNANCE_EXECUTION_PLAN.md

---

# ABCD Persistence and Governance Execution Plan

## Meta

- Created at: 2026-04-30
- Scope: close currently known post-E18 functional gaps with implementation-first sequencing.
- Source gaps:
  - persistent adapters (`mysql`, `postgres`) are still planned-only.
  - release-governance checkpoint/policy state is in-memory.
  - external plugin marketplace compatibility snapshot matrix is not yet published.

## Iteration Plan (A/B/C/D)

### A - Adapter mode rollout and subpackage refactor (P0)

- Objective:
  - deliver storage-adapter mode rollout baseline and implementation layering that supports persistent backends.
- Deliverables:
  - contracts/memory/persistent subpackages with stable root package facade.
  - mysql/postgres bootstrap config contract and unified mode factory entry.
  - memory mode migrated to subpackage implementation without public API break.
  - mysql/postgres runtime mode path enabled (transitional backend wiring).
- Acceptance:
  - `SKOLL_STORAGE_ADAPTER=mysql` runtime path works.
  - `SKOLL_STORAGE_ADAPTER=postgres` runtime path works.
  - `go test ./...` and `go test -race ./...` pass.
  - root `storageadapter` package API remains backward-compatible.

### B - Release-governance persistent state (P0)

- Objective:
  - remove in-memory closure state from admin release-governance path.
- Deliverables:
  - persistence-backed policy/checkpoint store in storage adapter boundary.
  - releasegov handler switches to repository-backed state access.
  - restart-survival tests and concurrent update tests.
- Acceptance:
  - checkpoint/policy survive process restart.
  - API contract remains backward compatible.

### C - Postgres adapter implementation (P1)

- Objective:
  - provide second relational adapter path for deployment portability.
- Deliverables:
  - postgres adapter bootstrap/config contract.
  - postgres repository implementations aligned with mysql behavior.
  - contract tests and parity report.
- Acceptance:
  - `SKOLL_STORAGE_ADAPTER=postgres` runtime path works.
  - full test gates pass with postgres mode.

### D - Plugin marketplace compatibility matrix publication (P1)

- Objective:
  - close external evidence/documentation gap for marketplace compatibility snapshots.
- Deliverables:
  - versioned compatibility matrix doc set.
  - update release evidence checklist to require matrix reference.
  - maintenance cadence and ownership model.
- Acceptance:
  - public matrix docs published with version tag/date.
  - release scorecard evidence references matrix snapshot.

## Suggested Execution Order

1. A (MySQL)
2. B (Release-governance persistence)
3. C (Postgres)
4. D (Marketplace matrix publication)

## Task Breakdown

### A Tasks

- A-Task01: persistent adapter bootstrap/config contract and factory extension scaffold.
- A-Task02: contracts/memory/persistent subpackage split and root package facade compatibility.
- A-Task03: mysql/postgres mode constructors and transitional runtime wiring.
- A-Task04: adapter tests update and rollout validation gate execution.

### B Tasks

- B-Task01: release-governance repository interface extraction in storage adapter boundary.
- B-Task02: handler decouple from in-memory map state.
- B-Task03: restart persistence and concurrent policy update tests.
- B-Task04: migration/backfill strategy doc and runbook updates.

### C Tasks

- C-Task01: postgres bootstrap/config contract.
- C-Task02: postgres repository implementations.
- C-Task03: cross-adapter parity contract suite.
- C-Task04: benchmark and compatibility evidence.

### D Tasks

- D-Task01: define compatibility matrix schema/template.
- D-Task02: publish initial marketplace compatibility snapshot.
- D-Task03: wire matrix evidence to release-governance checklist.
- D-Task04: monthly maintenance SOP and owner rotation.

## Tracking Board

| Item | Status | Notes |
| --- | --- | --- |
| A-Task01 | completed | persistent bootstrap config contract + factory scaffold landed |
| A-Task02 | completed | contracts/memory/persistent subpackages landed with root API aliases |
| A-Task03 | completed | mysql/postgres constructors wired via unified factory entry |
| A-Task04 | completed | gates passed: fmt + storageadapter tests + full test suite |
| B-Task01 | not-started |  |
| B-Task02 | not-started |  |
| B-Task03 | not-started |  |
| B-Task04 | not-started |  |
| C-Task01 | not-started |  |
| C-Task02 | not-started |  |
| C-Task03 | not-started |  |
| C-Task04 | not-started |  |
| D-Task01 | not-started |  |
| D-Task02 | not-started |  |
| D-Task03 | not-started |  |
| D-Task04 | not-started |  |

## Execution Log

- 2026-04-30: plan created, task decomposition completed, implementation started from A-Task01.
- 2026-04-30: A-Task01 completed.
  - Added persistent bootstrap config resolver for mysql/postgres DSN environment contract.
  - Extended storage adapter factory path to use persistent bootstrap scaffold before adapter construction.
  - Added and updated storage adapter tests for bootstrap config resolution and error-path validation.
  - Validation gates passed: `go fmt ./...`, `go test ./internal/module/storageadapter/...`, `go test ./...`.
- 2026-04-30: A iteration completed.
  - Migrated storage adapter into contracts/memory/persistent subpackages and retained root package compatibility aliases.
  - Moved memory implementation into dedicated subpackage and kept root factory behavior unchanged for callers.
  - Enabled mysql/postgres mode constructor path under unified factory (transitional backend wiring in this slice).
  - Validation gates passed: `go fmt ./...`, `go test ./internal/module/storageadapter/...`, `go test ./...`.
