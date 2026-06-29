# Demo Product Generator Example

This example is the M5 generator acceptance fixture. It describes a single-table `demo_product` CRUD module and is consumed by generator tests.

The fixture proves:

- `GeneratorSpec` can be loaded from an example file.
- dry-run emits backend domain/repository/store/service/handler/OpenAPI templates.
- dry-run emits frontend API/store/list-form page templates.
- generated file hashes can be recorded in history and used by rollback planning.

Validation:

```powershell
go test ./internal/service/generator/...
```
