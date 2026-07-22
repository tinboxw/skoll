# Demo Product Generator Example

This example is the M5/M7 generator acceptance fixture. It describes a single-table `demo_product` CRUD module and is consumed by generator tests.

The fixture proves:

- `GeneratorSpec` can be loaded from an example file.
- dry-run emits backend domain/repository/store/service/handler/OpenAPI templates.
- dry-run emits typed frontend API/store, bilingual locale, permission-aware route, and shared-kit list/form/detail templates.
- generated file hashes can be recorded in history and used by rollback planning.

## Spec Contract

| Area | Value |
|---|---|
| Spec file | `examples/demo_product/spec.json` |
| Spec ID | `spec-demo-product` |
| Module | `demo_product` |
| Go package | `demo_product` |
| Table | `demo_products` |
| Domain type | `DemoProduct` |
| Collection | `DemoProducts` |
| Menu key | `demo_product` |
| Menu parent | `system` |
| Menu path | `/demo-products` |
| Menu component | `DemoProduct/index` |
| Required menu permission | `demo_product.read` |

## Permission Keys

| Operation | Permission key |
|---|---|
| Read/list/detail | `demo_product.read` |
| Create | `demo_product.create` |
| Update | `demo_product.update` |
| Delete | `demo_product.delete` |
| Manage/admin | `demo_product.manage` |

## Generated File List

See [generated_files.md](generated_files.md) for the dry-run output matrix, ownership expectations, and release checklist.

## Backend Acceptance

See [backend_acceptance.md](backend_acceptance.md) for the generated backend API, permission, audit, store, service, and test coverage record.

## Frontend Acceptance

See [frontend_acceptance.md](frontend_acceptance.md) for the generated frontend API, Pinia store, list/form page, state, permission, and build coverage record.

Validation:

```powershell
go test ./internal/service/generator/...
```

For the formal M7-05-01 acceptance gate, run:

```powershell
go test ./internal/domain/generator/... ./internal/service/generator/...
```
