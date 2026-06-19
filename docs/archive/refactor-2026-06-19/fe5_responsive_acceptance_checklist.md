# FE5 Responsive Acceptance Checklist

> Work Item: FE5-05  
> Date: 2026-06-19  
> Scope: desktop and narrow viewport checks for core Skoll Admin pages

## Purpose

Use this checklist to verify that Skoll Admin pages remain usable across desktop and narrow viewports. It turns the FE1 responsive standard into a repeatable FE5 acceptance gate.

## Required Viewports

| Viewport | Size | Purpose |
|---|---|---|
| Desktop | 1440 x 900 or wider | Primary admin workstation layout. |
| Tablet/Narrow desktop | 768 x 1024 or similar | Two-column layouts should collapse cleanly. |
| Minimum narrow | 390 x 844 or similar | Minimum manual acceptance width; no overlap or clipped primary controls. |

If browser tooling is unavailable, mark the run as `Blocked` and record the missing tool. Do not replace responsive acceptance with build-only evidence.

## Required Commands

```powershell
cd web
npm run typecheck
npm run build
```

## Global Checks

| Area | Required check | Expected result |
|---|---|---|
| Page header | Title, description, status chips, and page actions at all viewports. | Header actions wrap below title without covering text. |
| Filters | Search, select, segmented controls, date inputs, reset actions. | Controls stack or wrap; labels remain attached to inputs. |
| Tables | Dense columns, fixed action columns, pagination, batch actions. | Table scrolls horizontally or degrades; action buttons stay reachable. |
| Forms | SchemaForm, edit forms, raw editors, sliders, inputs. | Fields become single column; validation text does not overlap controls. |
| Drawers/dialogs | Detail, config, logs, release, confirm dialogs. | Surface fits viewport; close/cancel controls are visible. |
| State blocks | Loading, empty, error, forbidden, success/failure. | State copy and actions fit without clipping or overlap. |
| Dangerous actions | Delete, disable, reset, clear, rollback confirmations. | Confirmation content fits and cancel path remains reachable. |
| Plugin panels | Inventory, risk report, DevPortal, task drawers. | Heavy panels do not force first screen horizontal overflow. |

## Page Matrix

| Page | Route | Desktop check | Narrow check | Failure examples |
|---|---|---|---|---|
| Dashboard | `/skoll/dashboard` | Summary cards, quick links, risk list align in readable bands. | Cards and lists stack; quick-link text and icons do not collide. | Overlapping risk items, clipped quick links. |
| User | `/skoll/user` | Summary, filters, table, batch assignment fit workstation width. | Filters and bulk assignment stack; table actions remain reachable. | Filter buttons overflow; action column unreachable. |
| Role | `/skoll/role` and role edit | List, edit form, users, permissions panels remain scannable. | Edit panels stack; grant/revoke/save controls wrap safely. | Permission tags overlap; save controls clipped. |
| Permission | `/skoll/permission` | Matrix filters and table remain dense but readable. | Filters stack; matrix table scrolls without hiding row actions. | Matrix cells crush labels; save button inaccessible. |
| Menu | `/skoll/menu` | Tree table and editor controls remain aligned. | Editor and tree table stack; visibility/order controls remain usable. | Tree table spills over shell; delete confirm clipped. |
| Plugin | `/skoll/plugin` | Inventory, detail drawer, risk report, DevPortal fit desktop. | Heavy panel switcher wraps; tables scroll; drawer/tags fit viewport. | DevPortal controls overflow; detail drawer close button hidden. |
| Audit | `/skoll/audit` | Tabs, filters, table, detail drawer, export actions align. | Filters stack; detail drawer and export controls fit. | Export button clipped; detail tabs overlap. |
| Setting | `/skoll/setting` | SchemaForm, raw setting list, summary and audit hint align. | Form/list stack; sensitive labels and reset confirmation fit. | Validation text overlaps input; reset dialog clips buttons. |

## Evidence

Record for each page:

- Route.
- Role.
- Desktop viewport result.
- Tablet/narrow desktop result.
- Minimum narrow result.
- Screenshot paths when available.
- Failed expected/actual text.
- Blocked reason when a viewport cannot be tested.

## Failure Policy

Mark responsive acceptance as `Failed` if any page has:

- Text overlapping adjacent controls.
- Buttons or icon-only actions clipped beyond reach.
- Drawer/dialog close or cancel controls off-screen.
- Table action columns inaccessible with no horizontal scroll.
- Loading, empty, error, or forbidden state clipped or hidden.
- Dangerous confirmation that cannot be canceled at 390px width.

Mark as `Blocked` only when browser tooling, seed data, or a required role is unavailable.

## Browser Tooling Status

Current Codex thread browser smoke status: Blocked.

Reason: the thread has not exposed a callable in-app browser tool, and the bundled Playwright runtime previously lacked `playwright-core`.

Until browser tooling is restored, use this checklist as manual acceptance guidance and keep Blocked entries explicit in `acceptance_log.md`.

## Verification

```powershell
rg -n "Desktop|Tablet/Narrow desktop|Minimum narrow|390|Page header|Filters|Tables|Forms|Drawers/dialogs|Dangerous actions|Dashboard|User|Role|Permission|Menu|Plugin|Audit|Setting|Failure Policy|Browser Tooling Status" docs/refactor/fe5_responsive_acceptance_checklist.md
```
