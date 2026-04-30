# E12 Admin Domain Operational Parity Plan

## Objective

Raise user/role/menu/config/dictionary/audit modules to production operational parity with persistence-ready query and bulk-operation governance.

## Scope

- Bulk operations with safety validation:
  - user/role/menu bulk create-update-disable
  - config/dictionary bulk upsert with schema checks
- Query and index strategy:
  - audit paging/filter SLA profile
  - dictionary/config typed query contract
- Operational controls:
  - retention/archival policy
  - actor/action-level rate guard hints

## Acceptance

- Bulk operations are idempotent and provide partial-failure reports.
- Query APIs expose stable pagination and filtering contracts.
- Audit retention policy and operational SLA docs are complete.
- Persistence adapter contract tests cover all admin domain modules.

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`
- `go test -bench=. -benchmem ./internal/app`
