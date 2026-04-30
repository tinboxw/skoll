# E13 Hook and Module Governance Parity Plan

## Objective

Deliver HisiPHP-style hook/event and module lifecycle governance parity with isolation, ordering, and recoverability.

## Scope

- Hook/event registry:
  - register, enable/disable, order, version
  - namespace isolation and timeout budget
- Module lifecycle governance:
  - install, enable, disable, remove, compatibility check
  - idempotent retries and rollback contracts
- Runtime isolation:
  - hook failure containment (no control-plane crash)
  - bounded retries and dead-letter diagnostics

## Acceptance

- Hook execution failures are isolated and observable.
- Module lifecycle APIs are idempotent and recoverable.
- Governance audit logs cover module state transitions.
- Compatibility checks block unsafe module combinations.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -bench=. -benchmem ./internal/module/pluginmgr`
