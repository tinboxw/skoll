# Demo Product Frontend Acceptance

> Work Item: M7-05-03
> Scope: generated frontend contract for the `demo_product` example module.

## Tested Frontend Surface

| Area | Evidence |
|---|---|
| API client | `TestDemoProductFrontendAcceptanceContract` asserts typed page/get/create/update/delete functions, encoded IDs, and the `/demo-products` contract. |
| Pinia store | The test asserts independent list/detail/mutation states, retry, paging, selection, create/update/remove actions, and shared error narrowing. |
| Route | A permission-aware lazy route is generated with `requiresAuth` and `demo_product.read`. |
| Locale | Module-local `zh-CN` and `en-US` keys cover page copy, actions, fields, validation, empty states, and destructive confirmation. |
| List page | The generated page composes `PageShell`, `FilterBar`, and `DataTable` from the shared UI kit. |
| Loading/empty/error | Initial loading/error and stale-table error are separate; `DataTable` owns the empty state. |
| Permission state | Read permission blocks the page and create/update/delete actions use current permission declarations. |
| Detail state | `DetailDrawer` loads the selected record through the generated typed detail API and renders detail errors in place. |
| Form drawer | `DetailDrawer` and Element Plus `FormRules` cover create/edit, validation, saving, backend errors, and success feedback. |
| Typed controls | The test asserts string, decimal, and boolean fields render as `el-input`, `el-input-number`, and `el-switch`. |
| Destructive action | `ConfirmAction` requires confirmation, reports progress, and preserves backend failure feedback. |
| Responsive state | Stable pagination and full-width form controls have an explicit 760px narrow layout. |
| Executable build | `TestGeneratedFrontendTypechecksAndBuilds` materializes the exact dry-run output and runs `vue-tsc` plus Vite production build. |
| Browser smoke | Headless Chrome verifies rendered desktop DOM and a non-empty 390x844 screenshot. |

## Commands

```powershell
go test ./internal/domain/generator/... ./internal/service/generator/...
$env:SKOLL_GENERATOR_FRONTEND_BUILD='1'; go test ./internal/service/generator -run TestGeneratedFrontendTypechecksAndBuilds -count=1 -v
cd web
npm run typecheck
npm run build
```

Result on 2026-07-22: Passed.

## Boundary

The demo frontend is validated as exact generated dry-run output in an isolated current frontend workspace. Route mounting and package/install automation are owned by PR3-05; this task emits the complete route contract and does not add a manual or compatibility route path.
