# E9-step1 Persistence Adapter Rollout Design

## Objective

Define the persistence rollout baseline for consistency-critical modules while keeping existing admin transport contracts stable.

## Target Modules

- `user` (session consistency and security state)
- `jobscheduler` (dispatch claims and execution history)
- `releasegov` (evidence and scorecard state)

## Storage Topology Decision

- Relational store (`MySQL` or `PostgreSQL`) for durable business entities and auditable records.
- Redis for low-latency distributed coordination and idempotency helpers where applicable.

## Contract Strategy

- Keep repository interfaces unchanged in `internal/module/storageadapter/adapter.go`.
- Preserve API payload contracts in `internal/app/*`.
- Enforce behavior parity via shared adapter contract tests.

## Rollout Phases

1. Phase A (done in step1):
   - Introduce adapter mode factory and runtime selector.
   - Keep `memory` as default mode.
   - Reject non-implemented persistent modes explicitly.
2. Phase B (next):
   - Introduce persistent adapter skeleton package and schema migration draft.
   - Add dual-run contract tests (memory vs persistent).
3. Phase C (later):
   - Add persistent write/read path by module with feature flag.
   - Add rollback and migration rehearsal evidence.

## Runtime Configuration Baseline

- New flag/env:
  - `-storage-adapter`
  - `SKOLL_STORAGE_ADAPTER`
- Allowed values for now:
  - `memory` (active)
  - `mysql` (planned, not implemented)
  - `postgres` (planned, not implemented)

## Risks

- API behavior drift during persistence migration.
- Consistency edge cases under distributed contention.
- Migration rollback complexity for mixed state.

## Mitigation

- Keep API and repository contracts stable.
- Require contract-test parity before enabling persistent mode.
- Gate persistent rollout behind explicit config and rehearsal evidence.

## Validation Commands

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test ./internal/module/storageadapter ./internal/bootstrap`
