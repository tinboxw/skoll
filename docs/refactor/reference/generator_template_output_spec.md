# Generator Template Output Spec

> Work Item: M5-02-01
> Scope: Skoll generator v1 single-table CRUD template output.
> Rule: generated code follows the current architecture directly. No compatibility adapters, no legacy route aliases, and no dual output layouts.

## Purpose

This document defines the canonical output paths, naming rules, and conflict policy for generator templates before the dry-run and diff service is implemented.

The generator reads a validated `GeneratorSpec` from `internal/domain/generator` and produces a deterministic file plan. The write path must be gated by dry-run, diff, conflict detection, and generated-file hash checks.

## Output Path Matrix

| Area | Canonical output | Notes |
|---|---|---|
| Domain | `internal/domain/<module>/` | Entity/value object definitions and domain validation only. No service, store, HTTP, or template dependency. |
| Repository | `internal/repository/<module>/` | Service-facing repository contract. No GORM dependency. |
| Memory store | `internal/store/memory/<module>_store.go` | In-memory implementation for tests and local runtime parity. |
| SQL model/store | `internal/store/sql/gormrepo/<module>_model.go`, `internal/store/sql/gormrepo/<module>_store.go` | GORM model and repository implementation. |
| SQL registration | `internal/store/sql/gormrepo/all_models.go` | Adds generated model to the current model registry. |
| Store factory | `internal/store/factory.go`, database adapters when needed | Wires the generated repository into existing store bundles. |
| Migrations | `migrations/mysql/<timestamp>_create_<table>.sql`, `migrations/postgres/<timestamp>_create_<table>.sql` | Timestamp comes from the generation batch and remains stable inside the batch. |
| Service | `internal/service/<module>/` | Use-case logic, audit calls, permission checks, and repository orchestration. |
| HTTP handler | `internal/handler/http/v1/<module>/` | Request parsing, response mapping, and service invocation only. |
| HTTP routing | current v1 router registration file | Registers only the canonical v1 route. |
| OpenAPI | `docs/api/openapi.yaml`, `internal/handler/http/openapi.yaml` | Request/response schemas, errors, pagination, and filters match handler behavior. |
| Permission catalog | current permission seed/catalog registration file | Uses generated permission keys from the spec. |
| Menu registry | current menu seed/registry registration file | Uses generated menu key, route, component, and required permissions from the spec. |
| Audit actions | current audit action registration location or service constants | Uses generated audit resource and action names from the spec. |
| Frontend API client | `web/src/api/<module>.ts` | Typed request/response functions, using current web API helper conventions. |
| Frontend store | `web/src/stores/<module>.ts` | Pinia store with loading, empty, error, retry, pagination, and save state. |
| Frontend views | `web/src/views/<Module>/index.vue`, optional child form components under the same directory | List, filter, create, edit, delete, permission-gated actions, and responsive layout. |
| Frontend router/menu binding | current router/menu integration points | Only canonical route paths generated from the spec are registered. |
| Generator history | generator history store/service output in later M5 tasks | Records actor, spec snapshot, file list, content hash, and generated-at time. |

## Naming Rules

- `module`, Go package names, table names, field names, column names, index names, route names, and audit resources must already pass `GeneratorSpec` validation.
- Go packages use lower snake case from `spec.Module.Package`.
- Go exported types use `spec.Module.DomainName`, for example `Product`.
- Collection/list names use `spec.Module.CollectionName`, for example `Products`.
- Database tables use `spec.Table.Name` exactly after validation.
- Migration filenames use `create_<table>` for first-generation table creation.
- Permission keys come from `spec.Permissions` and should normally follow `<module>.<action>`, for example `product.read`, `product.create`, `product.update`, `product.delete`.
- Menu keys come from `spec.Menu.Key`; route paths come from `spec.Menu.RoutePath`; frontend component paths come from `spec.Menu.Component`.
- Audit actions come from `spec.Audit.Actions` and should follow the existing audit action naming convention.
- Generated test fixtures and golden files must include the work item or fixture name in the path so they are traceable to the generator scenario.

## Conflict Policy

The generator must not overwrite user edits. Every write candidate is classified before disk writes:

| Status | Meaning | Write behavior |
|---|---|---|
| `create` | File does not exist. | Safe to write after dry-run approval. |
| `unchanged` | Existing content hash equals the generated content hash. | No write needed. |
| `update-clean` | Existing file has a recorded generated hash and still matches that recorded hash. | Safe to replace with new generated content after dry-run approval. |
| `conflict` | Existing file differs from the recorded generated hash or has no generated ownership record. | Do not write. Report current hash, expected generated hash, new generated hash, and path. |
| `blocked` | Path is outside the allowed output matrix or the spec references an unsupported output surface. | Do not write. Report the rule that blocked the path. |

Conflict reports must include:

- canonical path
- status
- previous generated hash when known
- current file hash when the file exists
- new generated hash
- short reason
- suggested human action

## No Overwrite

Generated ownership is established only when a file is created by the generator or updated from a matching recorded generated hash. A file without generated ownership is treated as user-owned even when its path matches the output matrix.

Rollback can only restore or remove files whose current hash still matches a generated history record. If a file changed after generation, rollback reports a conflict and leaves it untouched.

## Dry Run

Dry-run is mandatory before writes. It must return:

- normalized spec identity
- generation batch id
- ordered file plan
- output path
- status (`create`, `unchanged`, `update-clean`, `conflict`, `blocked`)
- summary counts by status
- diff for `create` and `update-clean`
- conflict report for `conflict` and `blocked`

The write command may only execute when dry-run contains no `conflict` or `blocked` entries unless a future explicit manual resolution flow records the decision. M5 v1 does not define a force overwrite mode.

## Hash

Hashes are content hashes of normalized generated file bytes. They are stored with:

- generation batch id
- actor
- generated-at timestamp
- spec snapshot hash
- path
- content hash
- template version or template id
- operation (`create`, `update`, `unchanged`)

Hash checks are the source of truth for idempotency, update safety, and rollback eligibility.

## Canonical Validation

The document-level acceptance for M5-02-01 is:

```powershell
rg -n "Generator Template Output Spec|Output Path Matrix|Naming Rules|Conflict Policy|No Overwrite|Dry Run|Hash|Canonical Validation" docs/refactor/generator_template_output_spec.md
git diff --check
```

Expected result: both commands pass. Later implementation tasks must keep their template output behavior aligned with this file.
