# Demo Product Backend Acceptance

> Work Item: M7-05-02
> Scope: generated backend contract for the `demo_product` example module.

## Tested Backend Surface

| Area | Evidence |
|---|---|
| HTTP API | `TestDemoProductBackendAcceptanceContract` asserts generated list/get/create/update/delete handler routes for `/demo-products`. Each route declares a permission, and every mutation declares an audit action. |
| Response contract | Handler responses use the platform `code/message/data` envelope; list data contains `items/offset/limit`, while create/update data contains `item`. Generated frontend clients consume the same shape. |
| Service flow | The test asserts generated service implementation calls repository create/update/list/delete paths. |
| Store flow | The test asserts generated memory store contains create/update/get/list/delete methods. |
| GORM mapping | Generated round-trip tests verify every field and audit timestamp crosses the persistence boundary without empty placeholder models. |
| Migration | MySQL and PostgreSQL output declares an idempotent table create, the metadata primary key, audit timestamps, and declared indexes. |
| Generated tests | Domain validation, memory CRUD, GORM mapping, and service CRUD tests are emitted with the backend slice. |
| Compile acceptance | `TestGeneratedBackendCompilesAndPassesGeneratedTests` writes the current backend output to a clean temporary module and runs `go test ./...` without manual edits. |
| Audit | The test asserts generated audit constants for `demo_product.create`, `demo_product.update`, and `demo_product.delete`. |
| Permission seed | The test asserts a typed permission catalog registers read/create/update/delete/manage resources through `permissionsvc.Service`. |
| Menu seed | The test asserts a typed menu node is built with `domainmenu.NewNode` and merged through `menusvc.Service`. |
| OpenAPI | Parsed YAML contract tests require list/get/create/update/delete operations exactly once and require permission/audit extensions on every mutation. |
| Rollback/history | Existing demo generation acceptance records dry-run history and rollback planning. |

## Commands

```powershell
go test ./internal/domain/generator/... ./internal/service/generator/...
go test ./...
```

Result on 2026-07-04: Passed.

## Boundary

This task validates generated backend code through dry-run output and Go tests. M7-05-03 owns generated frontend typecheck/build acceptance, and M7-05-05 owns plugin lifecycle acceptance.
