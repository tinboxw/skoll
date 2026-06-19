# FE6 Route Lazy Loading Coverage

## Metadata

- Work Item: FE6-02
- Date: 2026-06-19
- Executor: Codex
- Scope: admin route lazy-loading coverage
- Status: Coverage recorded

## Commands

```powershell
rg -n "const .*Page = \(\) => import|component: .*Page" web/src/router/index.ts
Select-String -Path web/src/router/index.ts -Pattern 'from "../views','from ''../views','import "../views','import ''../views'
rg -n 'import \* as XLSX|from "xlsx"|from ''xlsx''' web/src -g "*.vue" -g "*.ts"
```

## Static Admin Routes

All static admin route components in `web/src/router/index.ts` are declared as lazy component factories:

| Route | Name | Component factory | Lazy-loaded |
|---|---|---|---|
| `/skoll/login` | `login` | `LoginPage` | Passed |
| `/skoll/dashboard` | `dashboard` | `DashboardPage` | Passed |
| `/skoll/plugin` | `plugin` | `PluginPage` | Passed |
| `/skoll/user` | `user-list` | `UserListPage` | Passed |
| `/skoll/user/add` | `user-add` | `UserAddPage` | Passed |
| `/skoll/user/batch-add` | `user-batch-add` | `UserBatchPage` | Passed |
| `/skoll/user/:id/edit` | `user-edit` | `UserEditPage` | Passed |
| `/skoll/profile` | `profile` | `ProfilePage` | Passed |
| `/skoll/role` | `role-list` | `RoleListPage` | Passed |
| `/skoll/role/:id/edit` | `role-edit` | `RoleEditPage` | Passed |
| `/skoll/permission` | `permission` | `PermissionPage` | Passed |
| `/skoll/menu` | `menu` | `MenuPage` | Passed |
| `/skoll/dictionary` | `dictionary` | `DictionaryPage` | Passed |
| `/skoll/organization` | `organization` | `OrganizationPage` | Passed |
| `/skoll/audit` | `audit` | `AuditPage` | Passed |
| `/skoll/setting` | `setting` | `SettingPage` | Passed |

Redirect-only routes (`/`, `/skoll`, `/skoll/`) do not own page components and are not counted as eager page loads.

## Static Import Check

No static `../views` component import was found in `web/src/router/index.ts`. The router imports shared guards, stores, and plugin bootstrap helpers eagerly, while page views use dynamic imports.

## Bundle Baseline Follow-Up

FE6-01 identified `assets/xlsx-DLNWaC59.js` as the largest JavaScript chunk. The only source-level `xlsx` import is in `web/src/views/User/batch.vue`, and that page is routed through `UserBatchPage = () => import("../views/User/batch.vue")`.

Current conclusion:

- Spreadsheet code is route-scoped to the user batch import page at source level.
- The build still emits a separate `xlsx` chunk, so later optimization should avoid making spreadsheet helpers shared by first-load routes.
- If export/import behavior expands, FE6 regression checks should verify `xlsx` remains lazy and route-owned.

## Plugin Routes

`/skoll/plugin` is a lazy static route. Backend plugin and app plugin pages are registered at runtime with iframe-based view components created in `web/src/plugins/index.ts`; remote plugin page content is fetched from the backend and is not bundled as local route view code.

Current follow-up:

- Built-in auth plugin host code is imported by plugin bootstrap and remains part of the shell/plugin host path.
- FE6-05 should measure plugin host and heavy panel behavior separately, including logs, risk report, release history, Dev Portal tasks, and iframe loading.

## Acceptance Outcome

- Major static admin routes are lazy-loaded.
- Redirect routes are documented as component-less.
- No router-level static `../views` imports are present.
- The largest `xlsx` chunk is tied to a lazy user batch route at source level.
- Plugin route behavior is documented for FE6-05 follow-up.
