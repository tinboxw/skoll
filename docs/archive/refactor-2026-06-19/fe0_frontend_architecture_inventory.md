# FE0 Frontend Architecture Inventory

> Work Item: FE0-02  
> Date: 2026-06-19  
> Scope: API clients, stores, router, permissions, SchemaForm, Common components

## Summary

The frontend has a workable Vue 3 + TypeScript architecture with shared API wrappers, Pinia stores, route guards, permission helpers, and a reusable `SchemaForm`. The strongest boundaries are audit, navigation/menu, permissions, route access, and button access. The weakest boundary is still page-local API and type handling in large operational pages, especially Plugin and system CRUD pages.

## Architecture Map

| Area | Current owner | Boundary | Status |
| --- | --- | --- | --- |
| API transport | `web/src/utils/api.ts` | `apiGet/apiPost/apiPut/apiPatch/apiDelete`, `ApiResponse<T>`, `ApiError`, 401 redirect handling | Stable |
| Audit API client | `web/src/audit/api.ts` | Typed list/detail/export client and query builder | Stable |
| Navigation/Menu API client | `web/src/navigation/api.ts` | Typed menu tree/save/reorder/visibility client | Stable |
| Permission API client | `web/src/permissions/api.ts` | Typed permission catalog list/detail/enable/disable/diff client | Stable |
| Page-local API calls | `web/src/views/**` | User, Role, Setting, Dictionary, Organization, Plugin dev portal requests | Needs consolidation |
| Stores | `web/src/stores` | app/navigation/permissions/plugins/tabs/user shared state | Stable but uneven per domain |
| Router | `web/src/router/index.ts` | Static routes, plugin home routing, auth redirect, route permission guard | Stable |
| Route permissions | `web/src/permissions/route.ts` | `canAccessRoute`, `isPublicRoute`, route meta normalization | Stable |
| Button permissions | `web/src/permissions/button.ts` | `BUTTON_ACCESS`, `useButtonAccess`, shared permission checks | Stable |
| Generic access | `web/src/permissions/access.ts` | role/permission rule evaluation with implied permissions | Stable |
| Schema forms | `web/src/components/Common/SchemaForm.vue` | Plugin/system settings schema rendering and validation | Stable |
| Common shell components | `web/src/components/Layout` | header/sidebar/main content/pinned tabs | Stable |
| Common low-level components | `web/src/components/Common` | Table/Form/Dialog wrappers | Underused and underspecified |

## Reuse Points

| Reuse point | Used by | Keep using for |
| --- | --- | --- |
| `utils/api.ts` | All API calls and plugin auth | Standard envelope, auth header, 401 handling, visible `ApiError`. |
| `useButtonAccess` | User, Role, Plugin, other guarded page actions | Button visibility/disabled behavior and page-local access checks. |
| `canAccessRoute` | Router guard | Static page and plugin route permission gating. |
| `useNavigationStore` | App shell/sidebar | Backend menu registry and route/sidebar alignment. |
| `usePermissionStore` | Permission page and access planning | Permission catalog cache and enabled items. |
| `usePluginStore` | App shell, Dashboard, Plugin page, route default home | Plugin inventory, default home validation, plugin tabs. |
| `SchemaForm` | Plugin config, Setting common settings | Schema-driven admin forms. |
| `confirmAction` | Dangerous User/Role/Plugin/Menu/Setting/Audit workflows | Standard destructive confirmation UX. |

## Gaps

| Gap | Impact | Recommended follow-up |
| --- | --- | --- |
| Plugin page owns many raw `/v1/plugins/dev/*` requests and ad hoc response shapes. | Hard to test, hard to split UI, repeated `Record<string, unknown>`. | FE2-01 should introduce typed plugin/dev portal clients before FE3/FE4 Plugin redesign. |
| User/Role/Setting/Dictionary/Organization pages keep DTO normalization locally. | CRUD pages repeat request and normalization patterns. | FE2-01 and FE2-02 should define API/store conventions. |
| Common `Table.vue`, `Form.vue`, `Dialog.vue` are documented but lightly used. | Page polish will remain inconsistent if wrappers stay underspecified. | FE1/FE2 should decide whether to formalize or retire each wrapper. |
| Route meta permissions use hardcoded static permissions. | Backend menu registry and static route access can drift. | FE2-03 should document menu/route/button permission source-of-truth behavior. |
| `v-permission` appears in Plugin page, while most pages use `useButtonAccess`. | Mixed permission style makes audits slower. | FE2-03 should standardize directive vs composable usage. |
| Store coverage is domain-specific but incomplete for CRUD pages. | Loading/error/retry patterns are page-local. | FE2-02 should define Pinia async state contract. |

## Recommended Architecture Rules

1. Add typed API clients before adding or expanding page-local requests.
2. Use Pinia stores for shared inventory, navigation, permission, plugin, and authenticated user state.
3. Keep one-page-only form state inside the page, but move reusable DTO normalization to clients or stores.
4. Use route meta for page access, `useButtonAccess` for script logic, and `v-permission` only for simple template-only visibility.
5. Use `SchemaForm` for backend/plugin-defined configuration fields instead of hardcoded plugin-specific forms.
6. Do not add legacy route aliases or duplicate stores when a current route/store already owns the behavior.

## Validation

```powershell
rg "defineStore|SchemaForm|canAccess|v-permission" web/src
rg -n "apiGet|apiPost|apiPut|apiDelete|createRouter|router.beforeEach|BUTTON_ACCESS|useButtonAccess" web/src
```

Result: Passed.
