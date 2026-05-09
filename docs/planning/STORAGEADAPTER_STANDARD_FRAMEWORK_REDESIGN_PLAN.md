# StorageAdapter Standard Framework Redesign Plan

## Background

The current storage adapter has already introduced `contracts/memory/persistent` split, but persistent internals still carry transition-era patterns:

- several repositories are `write-through` wrappers around in-memory services
- DB errors in non-error-returning methods use panic path
- adapter bootstrap owns too many construction details in one place
- migration strategy is AutoMigrate-only and not versioned yet

This plan defines a standard, extensible framework-style redesign **without compatibility wrappers**.

## Design Goals

- make `memory`, `mysql`, `postgres` first-class and symmetric implementations of the same port
- remove root-level compatibility indirection and keep root package orchestration-only
- keep extensibility for future dialects and module repositories
- make transaction and error behavior explicit and consistent

## Target Architecture

```text
internal/module/storageadapter/
  contracts/            # storage port (single source of truth)
  factory.go            # mode resolve + implementation assembly only
  memory/               # in-memory adapter implementation
  persistent/
    db/                 # dialect open, tx, migration engine
    repo/               # pure SQL repositories by domain (future target)
    adapter.go          # composition root for SQL adapter
    factory.go          # mysql/pg constructor entry
```

### Layer Responsibilities

1. contracts
- define repository contracts only
- no implementation details

2. factory (root package)
- resolve mode and build adapter
- no type alias re-export, no compatibility wrappers

3. memory
- standalone implementation of contracts
- no dependency on persistent

4. persistent
- standalone SQL implementation of contracts
- no dependency on memory
- repository behavior defined by SQL + transaction policy, not by in-memory mirror behavior

## Mandatory Engineering Rules

- no compatibility wrapper files
- no hidden fallback from mysql/pg to memory
- all multi-row writes must use explicit transaction boundaries
- no implicit panic-based control flow for recoverable persistence errors
- dialect-specific behavior must stay inside `persistent/db` and repository internals

## Redesign Milestones

### R1 - Port and assembly cleanup (completed)

- remove root alias-based compatibility layer
- root `factory.go` returns `contracts.Adapter`
- bootstrap wiring depends on `contracts.Adapter` explicitly

### R2 - Repository implementation normalization

- replace write-through in-memory wrappers with direct SQL repositories (priority: release/rbac/user/job/plugin)
- move model ownership to repository-local model files
- add repository-level transaction methods for complex updates

### R3 - Error model normalization

- define storage-level error taxonomy in persistent package
- remove panic path from recoverable DB failures
- map not-found/constraint/conflict errors deterministically

### R4 - Migration framework upgrade

- introduce versioned migration runner (up/down or forward-only)
- keep AutoMigrate only for test harness or bootstrap compatibility mode
- add migration verification tests per dialect

### R5 - Dialect extension contract

- formalize dialect capability matrix (mysql/pg/sqlite-test)
- add extension hook for future dialect registration
- keep repository SQL portable unless feature is explicitly gated by capability

## Execution Plan

1. implement R2 for release-governance first (small surface, high value)
2. implement R2 for user/rbac/job
3. implement R3 globally
4. implement R4 migration system
5. implement R5 capability registry

## Validation Gates (per milestone)

- `go fmt ./...`
- `go test ./internal/module/storageadapter/...`
- `go test ./...`
- `go test -race ./...` (for shared-state/transaction changes)

## Current Status

- R1 completed in code: root compatibility alias removed, root assembly uses explicit contracts port.
- R2-R5 pending.
