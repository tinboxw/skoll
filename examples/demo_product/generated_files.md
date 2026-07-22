# Demo Product Generated File Matrix

> Work Item: M7-05-01
> Source spec: `examples/demo_product/spec.json`
> Dry-run batch used by tests: `demo-product-acceptance`

## Output Matrix

The generator dry-run should produce 27 file plans for `demo_product` with no conflicts and no blocked paths.

| Layer | Template ID | Path | Expected status in clean repo |
|---|---|---|---|
| Domain | `backend.domain.doc` | `internal/domain/demo_product/doc.go` | create |
| Domain | `backend.domain.entity` | `internal/domain/demo_product/entity.go` | create |
| Domain test | `backend.domain.test` | `internal/domain/demo_product/entity_test.go` | create |
| Repository port | `backend.repository` | `internal/repository/demo_product/demo_product_repo.go` | create |
| Memory store | `backend.store.memory` | `internal/store/memory/demo_product_store.go` | create |
| Memory store test | `backend.store.memory.test` | `internal/store/memory/demo_product_store_test.go` | create |
| GORM model | `backend.store.gorm.model` | `internal/store/sql/gormrepo/demo_product_model.go` | create |
| GORM repository | `backend.store.gorm.repo` | `internal/store/sql/gormrepo/demo_product_store.go` | create |
| GORM mapping test | `backend.store.gorm.test` | `internal/store/sql/gormrepo/demo_product_store_test.go` | create |
| GORM registry | `backend.store.gorm.registry` | `internal/store/sql/gormrepo/all_models.go` | update-clean or conflict if user-edited |
| Store factory | `backend.store.factory` | `internal/store/factory.go` | update-clean or conflict if user-edited |
| MySQL migration | `backend.migration.mysql` | `migrations/mysql/20260629_030000_create_demo_products.sql` | create |
| PostgreSQL migration | `backend.migration.postgres` | `migrations/postgres/20260629_030000_create_demo_products.sql` | create |
| Service port | `backend.service` | `internal/service/demo_product/service.go` | create |
| Service implementation | `backend.service.impl` | `internal/service/demo_product/service_impl.go` | create |
| Service test | `backend.service.test` | `internal/service/demo_product/service_impl_test.go` | create |
| HTTP handler | `backend.handler` | `internal/handler/http/v1/demo_product/handler.go` | create |
| HTTP handler test | `backend.handler.test` | `internal/handler/http/v1/demo_product/handler_test.go` | create |
| Router registration | `backend.router` | `internal/handler/http/generated_demo_product_routes.go` | create |
| OpenAPI docs | `backend.openapi.docs` | `docs/api/openapi.yaml` | update-clean or conflict if user-edited |
| OpenAPI runtime | `backend.openapi.runtime` | `internal/handler/http/openapi.yaml` | update-clean or conflict if user-edited |
| Permission/menu catalog | `backend.permission.seed` | `internal/bootstrap/generated_demo_product_catalog.go` | create |
| Frontend API | `frontend.api` | `web/src/api/demo_product.ts` | create |
| Frontend store | `frontend.store` | `web/src/stores/demo_product.ts` | create |
| Frontend locale | `frontend.locale` | `web/src/i18n/generated_demo_product.ts` | create |
| Frontend route | `frontend.route` | `web/src/router/generated_demo_product.ts` | create |
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

## Migration Contract

- Both dialect outputs create the table idempotently and declare the metadata primary key.
- `created_at` and `updated_at` are required because the generated GORM model persists domain audit metadata.
- Declared indexes are emitted after table creation and are covered by executable migration tests.
- Plugin targets require namespaced tables, indexes, permissions, and menus before rendering.
- Plugin manifests declare one current migration directory plus explicit rollback and uninstall policies.

## Acceptance Commands

```powershell
go test ./internal/domain/generator/... ./internal/service/generator/...
$env:SKOLL_GENERATOR_FRONTEND_BUILD='1'; go test ./internal/service/generator -run TestGeneratedFrontendTypechecksAndBuilds -count=1 -v
rg -n "demo_product.read|demo_product.create|demo_product.update|demo_product.delete|demo_product.manage|DemoProduct/index|/demo-products" examples/demo_product/spec.json examples/demo_product/README.md examples/demo_product/generated_files.md
```

Expected result:

- `TestDemoProductGenerationAcceptance` passes.
- The generated Go files parse successfully in dry-run tests.
- OpenAPI, frontend API/store/locale/route/view, permissions, menu declaration, history recording, and rollback planning are covered by generator tests.
- The generated frontend is typechecked, production-built, opened in headless Chrome, and rendered at desktop and 390x844 viewports without hand edits.
