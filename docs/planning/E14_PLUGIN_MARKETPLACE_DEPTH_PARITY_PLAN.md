# E14 Plugin Marketplace Depth Parity Plan

## Objective

Reach marketplace depth parity with signed index trust, dependency solving, and transactional upgrades.

## Scope

- Signed index ingestion:
  - trust root management
  - index signature verification and expiry policy
- Dependency solver:
  - conflict diagnosis output
  - deterministic resolution strategy
- Upgrade transaction:
  - checkpointed step execution
  - automatic rollback on failure
  - provenance and trust evidence retention

## Acceptance

- Unsigned/tampered index artifacts are rejected.
- Solver reports actionable conflict diagnostics.
- Upgrade flow supports rollback to previous stable state.
- Audit trail captures trust and provenance metadata.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -bench=. -benchmem ./internal/module/pluginmgr`
