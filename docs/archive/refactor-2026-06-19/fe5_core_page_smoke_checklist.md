# FE5 Core Page Smoke Checklist

> Work Item: FE5-03  
> Date: 2026-06-19  
> Scope: Dashboard/User/Role/Permission/Menu/Plugin/Audit/Setting browser smoke checklist

## Purpose

Use this checklist after `npm run typecheck` and `npm run build` pass. It is the repeatable manual smoke path for Skoll core admin pages until automated browser coverage is available.

## Run Metadata

Record these fields for every run:

| Field | Value |
|---|---|
| Date |  |
| Browser |  |
| Backend base URL |  |
| Frontend URL |  |
| Desktop viewport |  |
| Narrow viewport |  |
| Admin role/account |  |
| Restricted role/account |  |
| Screenshots/log path |  |
| Decision | Pending |

## Required Commands

```powershell
cd web
npm run typecheck
npm run build
```

## Global Smoke Rules

Each page must be checked at one desktop width and one narrow width.

| State | Required check | Expected result |
|---|---|---|
| Normal | Open the route with seeded or real data. | Primary content renders without overlap or stale placeholders. |
| Loading | Trigger refresh/save/export or throttle backend when possible. | Loading is visible and duplicate actions are disabled. |
| Empty | Use filters or seed data that returns no rows/items. | Empty state is explicit and previous data is not shown as current. |
| Error | Stop backend or force a failing response when safe. | Error is visible, retry or recovery remains available. |
| No permission | Use restricted role or remove route/button permission. | Page or control shows forbidden state instead of silently succeeding. |
| Narrow | Recheck at narrow viewport. | Text, filters, buttons, tables, drawers, and dialogs do not overlap. |
| Dangerous action | Trigger cancel path for delete/disable/reset/clear/rollback. | Confirmation appears and cancel leaves data unchanged. |

If a state cannot be exercised, mark it `Blocked` and record the missing seed, role, backend fixture, or browser automation reason.

## Page Checklist

| Page | Route | Smoke path | Must cover | Decision |
|---|---|---|---|---|
| Dashboard | `/skoll/dashboard` | Open page, review system summary, plugin summary, quick links, risk list. | Normal, loading, empty plugin/risk hints, permission-filtered quick links, narrow layout. | Pending |
| User | `/skoll/user` | Search/filter users, open add/edit/batch flows, cancel deactivate or assignment action. | Normal, loading, empty filter result, backend error, no permission, dangerous confirm, narrow layout. | Pending |
| Role | `/skoll/role` | Open list and edit page, filter roles/users/permissions, cancel grant/revoke/save where available. | Normal, loading, empty panels, save failure, no permission, confirm actions, narrow edit layout. | Pending |
| Permission | `/skoll/permission` | Filter permission matrix, toggle grant/revoke in cancel-safe path, review save confirmation. | Normal, matrix loading, empty filter result, save error, no permission, dense table scan, narrow filters. | Pending |
| Menu | `/skoll/menu` | Filter tree, edit visibility/order/permission fields, cancel save/delete confirmation. | Normal, loading, empty filter result, validation warning, no permission, dangerous confirm, narrow editor. | Pending |
| Plugin | `/skoll/plugin` | Review inventory, details, risk panel, DevPortal tab, lifecycle/config/log cancel-safe paths. | Normal, loading, empty filters, lifecycle/config error, no permission, install/disable confirmation, narrow panels. | Pending |
| Audit | `/skoll/audit` | Switch type tabs, apply filters, open detail drawer, trigger export/clear cancel-safe path. | Normal, loading, empty filters, detail/export error, no permission, export evidence, narrow detail. | Pending |
| Setting | `/skoll/setting` | Search settings, edit SchemaForm/raw value without saving, cancel reset, review audit hint. | Normal, loading, empty schema/filter, save/reset error, no permission, reset confirmation, narrow form. | Pending |

## Minimum Evidence

Record at least one evidence item per page:

- Screenshot path for desktop and narrow viewport when browser tooling is available.
- Exact route and role.
- Expected/actual text for any failed state.
- Command output summary for typecheck/build.
- Blocked reason for any state not tested.

## Browser Tooling Status

Current Codex thread browser smoke status: Blocked.

Reason: the thread has not exposed a callable in-app browser tool, and the bundled Playwright runtime previously lacked `playwright-core`.

Until browser tooling is restored, use this checklist as manual acceptance guidance and keep Blocked entries explicit in `acceptance_log.md`.

## Verification

```powershell
rg -n "Dashboard|User|Role|Permission|Menu|Plugin|Audit|Setting|Normal|Loading|Empty|Error|No permission|Narrow|Dangerous action|Browser Tooling Status" docs/refactor/fe5_core_page_smoke_checklist.md
```
