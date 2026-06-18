# FE5 Permission State Acceptance Checklist

> Work Item: FE5-04  
> Date: 2026-06-19  
> Scope: admin and restricted-role permission-state checks for core frontend workflows

## Purpose

Use this checklist to verify that Skoll frontend permission behavior is consistent across routes, sidebar menus, buttons, forbidden page states, and API denial feedback.

This checklist complements `fe5_core_page_smoke_checklist.md`: FE5-03 checks page workflows; FE5-04 checks the same workflows under different permission sets.

## Required Roles

| Role | Required permissions | Expected use |
|---|---|---|
| Admin | `super_admin` or a role with `*` | Positive-path access to all core pages and management actions. |
| Restricted | A role with only read permissions, for example `user.read`, `role.read`, `plugin.read`, `audit.read`, `dict.read`, `org.read` | Negative-path checks for routes, sidebar items, management buttons, and API denied states. |

If a seeded restricted role is not available, record the run as `Blocked` and include the missing role or seed script.

## Required Commands

```powershell
cd web
npm run typecheck
npm run build
```

## Permission Entry Points

| Surface | Source of truth | Acceptance check |
|---|---|---|
| Route guard | `web/src/permissions/route.ts` / `canAccessRoute` | Unauthorized route access redirects or shows forbidden state; no protected page flashes as usable. |
| Sidebar/menu | `/v1/menus/tree` consumed by navigation store | Restricted role sees only allowed menu items. |
| Button/action | `web/src/permissions/button.ts` / `BUTTON_ACCESS` and `v-permission` | Restricted role cannot trigger create/update/delete/manage actions. |
| Page state | `StateBlock` forbidden or equivalent no-permission content | Forbidden state is visible and explains the access issue. |
| API denial | backend 403/`forbidden` surfaced by page state or toast | Denied operations do not look like empty success. |
| Audit signal | `system.security.deny` or permission-denied event when backend supports it | Denial path is traceable in audit smoke or acceptance notes. |

## Page Matrix

| Page | Admin check | Restricted check | Must record |
|---|---|---|---|
| Dashboard | Open `/skoll/dashboard`; quick links route to allowed pages. | Quick links requiring missing permissions are hidden or blocked. | Visible quick links, blocked links, role. |
| User | Open `/skoll/user`; create/edit/batch and destructive actions are available with confirmation. | `user.create`, `user.update`, `user.delete` actions are hidden/disabled or forbidden. | Button state, forbidden route for `/skoll/user/add` and `/skoll/user/edit/:id`. |
| Role | Open `/skoll/role`; edit, grant, revoke, and save paths are available with confirmation. | `role.update`, `role.manage`, permission grant/revoke actions are hidden/disabled or forbidden. | Edit route behavior, grant/revoke button state. |
| Permission | Open `/skoll/permission`; matrix management and save actions are available. | Missing `permission.manage` prevents route or shows forbidden state. | Route result and no-permission copy. |
| Menu | Open `/skoll/menu`; save/delete/visibility actions are available with confirmation. | Missing `system.manage` prevents route or shows forbidden state. | Route result and action state. |
| Plugin | Open `/skoll/plugin`; lifecycle, install, config, DevPortal manage actions are available. | `plugin.read` allows inventory; missing `plugin.manage` hides/disables install, lifecycle, config save, and DevPortal actions. | Inventory visibility, management button state, forbidden DevPortal state. |
| Audit | Open `/skoll/audit`; filters, detail, export, and clear actions follow permissions. | Missing `audit.read` prevents route; read-only role cannot clear if clear is protected. | Route result, export/clear button state, API denied feedback. |
| Setting | Open `/skoll/setting`; schema edit/save/reset actions are available. | Missing `system.manage` prevents route or shows forbidden state. | Route result, save/reset button state. |

## Negative-Path Evidence

For every restricted-role check, record:

- Role and permission list.
- Route or action attempted.
- Expected permission key.
- Expected result.
- Actual result.
- Screenshot/log path when available.
- Audit event or API error text when available.

## Failure Policy

Mark the permission-state run as `Failed` if any of these happen:

- Restricted role can execute a management action.
- Unauthorized route renders usable protected content.
- API denial is displayed as a successful empty state.
- Management buttons remain enabled without confirmation or feedback.
- Admin role cannot access a core action that the role should own.

Mark as `Blocked` only when the missing ingredient is explicit: seeded role, backend fixture, browser tooling, or audit fixture.

## Browser Tooling Status

Current Codex thread browser smoke status: Blocked.

Reason: the thread has not exposed a callable in-app browser tool, and the bundled Playwright runtime previously lacked `playwright-core`.

Until browser tooling is restored, use this checklist as manual acceptance guidance and keep Blocked entries explicit in `acceptance_log.md`.

## Verification

```powershell
rg -n "Admin|Restricted|Route guard|Sidebar/menu|Button/action|API denial|Audit signal|Dashboard|User|Role|Permission|Menu|Plugin|Audit|Setting|Negative-Path Evidence|Failure Policy|Browser Tooling Status" docs/refactor/fe5_permission_state_acceptance_checklist.md
```
