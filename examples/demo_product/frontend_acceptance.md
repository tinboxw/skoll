# Demo Product Frontend Acceptance

> Work Item: M7-05-03
> Scope: generated frontend contract for the `demo_product` example module.

## Tested Frontend Surface

| Area | Evidence |
|---|---|
| API client | `TestDemoProductFrontendAcceptanceContract` asserts list/create/update/delete functions and `/demo-products` base path. |
| Pinia store | The test asserts loading/mutation state, last error, retry, create/update/remove actions, and `unknown` error narrowing. |
| List page | The test asserts generated table output for `DemoProduct/index.vue`. |
| Empty state | The test asserts table `empty-text="No data"`. |
| Error state | The test asserts `el-alert` rendering from `store.hasError`. |
| Form drawer | The test asserts generated drawer and field controls. |
| Typed controls | The test asserts string, decimal, and boolean fields render as `el-input`, `el-input-number`, and `el-switch`. |
| Permission buttons | The test asserts create/update/delete permission directives and permission constants. |

## Commands

```powershell
go test ./internal/domain/generator/... ./internal/service/generator/...
cd web
npm run typecheck
npm run build
```

Result on 2026-07-04: Passed.

## Boundary

The demo frontend is validated as generated dry-run output and by the repository frontend typecheck/build gates. It is not mounted as a committed route in this task; the generated file matrix remains the source of truth for where the page will be emitted.
