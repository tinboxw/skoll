# Demo Product Generated File Matrix

> Work Item: M7-05-01
> Source spec: `examples/demo_product/spec.json`
> Dry-run batch used by tests: `demo-product-acceptance`

## Output Matrix

The generator dry-run should produce 20 file plans for `demo_product` with no conflicts and no blocked paths.

| Layer | Template ID | Path | Expected status in clean repo |
|---|---|---|---|
| Domain | `backend.domain.doc` | `internal/domain/demo_product/doc.go` | create |
| Domain | `backend.domain.entity` | `internal/domain/demo_product/entity.go` | create |
| Repository port | `backend.repository` | `internal/repository/demo_product/demo_product_repo.go` | create |
| Memory store | `backend.store.memory` | `internal/store/memory/demo_product_store.go` | create |
| GORM model | `backend.store.gorm.model` | `internal/store/sql/gormrepo/demo_product_model.go` | create |
| GORM repository | `backend.store.gorm.repo` | `internal/store/sql/gormrepo/demo_product_store.go` | create |
| GORM registry | `backend.store.gorm.registry` | `internal/store/sql/gormrepo/all_models.go` | update-clean or conflict if user-edited |
| Store factory | `backend.store.factory` | `internal/store/factory.go` | update-clean or conflict if user-edited |
| MySQL migration | `backend.migration.mysql` | `migrations/mysql/20260629_030000_create_demo_products.sql` | create |
| PostgreSQL migration | `backend.migration.postgres` | `migrations/postgres/20260629_030000_create_demo_products.sql` | create |
| Service port | `backend.service` | `internal/service/demo_product/service.go` | create |
| Service implementation | `backend.service.impl` | `internal/service/demo_product/service_impl.go` | create |
| HTTP handler | `backend.handler` | `internal/handler/http/v1/demo_product/handler.go` | create |
| Router registration | `backend.router` | `internal/handler/http/v1/router.go` | update-clean or conflict if user-edited |
| OpenAPI docs | `backend.openapi.docs` | `docs/api/openapi.yaml` | update-clean or conflict if user-edited |
| OpenAPI runtime | `backend.openapi.runtime` | `internal/handler/http/openapi.yaml` | update-clean or conflict if user-edited |
| Permission/menu seed | `backend.permission.seed` | `internal/bootstrap/permission_menu_seed.go` | update-clean or conflict if user-edited |
| Frontend API | `frontend.api` | `web/src/api/demo_product.ts` | create |
| Frontend store | `frontend.store` | `web/src/stores/demo_product.ts` | create |
| Frontend view | `frontend.view` | `web/src/views/DemoProduct/index.vue` | create |

## Permission and Menu Declarations

| Declaration | Value |
|---|---|
| Permission resource | `demo_product` |
| Read | `demo_product.read` |
| Create | `demo_product.create` |
| Update | `demo_product.update` |
| Delete | `demo_product.delete` |
| Manage | `demo_product.manage` |
| Menu key | `demo_product` |
| Parent menu key | `system` |
| Route path | `/demo-products` |
| Component | `DemoProduct/index` |
| Icon | `Package` |
| Order | `60` |
| Required permissions | `demo_product.read` |

## Acceptance Commands

```powershell
go test ./internal/domain/generator/... ./internal/service/generator/...
rg -n "demo_product.read|demo_product.create|demo_product.update|demo_product.delete|demo_product.manage|DemoProduct/index|/demo-products" examples/demo_product/spec.json examples/demo_product/README.md examples/demo_product/generated_files.md
```

Expected result:

- `TestDemoProductGenerationAcceptance` passes.
- The generated Go files parse successfully in dry-run tests.
- OpenAPI, frontend API/store/view, permissions, menu declaration, history recording, and rollback planning are covered by generator tests.
