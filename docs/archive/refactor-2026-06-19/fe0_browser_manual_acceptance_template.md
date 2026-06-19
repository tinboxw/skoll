# FE0 Browser Manual Acceptance Template

> Work Item: FE0-05  
> Date: 2026-06-19  
> Scope: browser/manual acceptance for visible frontend workflows

## Purpose

Use this template for every visible frontend task until automated browser smoke coverage is introduced. A build passing is not enough for visible UI changes.

## Run Metadata

```text
Task:
Date:
Browser:
Viewport:
Backend mode:
User/role:
Route(s):
Build command:
Typecheck command:
Screenshots/log paths:
```

## Required State Matrix

| State | Required check | Expected result | Result | Evidence |
| --- | --- | --- | --- | --- |
| Normal data | Open the route with a seeded or real data set. | Primary table/form/panel renders real data without layout shift. | Pending | |
| Loading | Trigger refresh/save/export on a slow or throttled backend. | Loading indicator appears; duplicate actions are disabled. | Pending | |
| Empty | Use filters or seed state that returns no rows/items. | Empty state is clear and stale data is removed. | Pending | |
| Backend error | Stop backend or force a failing response. | Error message is visible; retry path remains available. | Pending | |
| No permission | Use a restricted role or remove required permission. | Page or controls show no-permission behavior, not silent disappearance. | Pending | |
| Narrow viewport | Check at 390px width and one desktop width. | Text, buttons, filters, tables, drawers, and dialogs do not overlap. | Pending | |
| Dangerous action | Trigger delete/disable/reset/clear/rollback paths. | Confirmation appears before action; cancel is safe; failure is visible. | Pending | |
| Save success/failure | Submit a form or config change. | Success and failure states are visible and not confused. | Pending | |
| Detail/drawer/dialog | Open secondary detail surfaces. | Focused content fits, closes cleanly, and preserves list context. | Pending | |

## Page-Specific Additions

| Page family | Extra checks |
| --- | --- |
| Dashboard | Cards reflect loaded plugin/auth/system data; empty plugin inventory does not look broken. |
| User/Role | Bulk actions, role assignment, permission grant/revoke, and destructive actions are guarded. |
| Permission/Menu | Dense matrix/tree views remain scannable and editable on desktop; narrow viewport has no overlap. |
| Plugin | Lifecycle actions, config SchemaForm, logs/debug drawers, dev portal heavy panels, default home behavior. |
| Audit | Type tabs, filters, quick suggestions, detail drawer, CSV export, no-permission state. |
| Setting | SchemaForm validation, raw setting editor, reset confirmation, sensitive setting handling. |

## Evidence Rules

1. Record exact route, user role, and viewport.
2. Record command output path or screenshot path when available.
3. Record failed expected/actual text for every failed state.
4. Do not mark a visible task done when only `npm run build` passed.
5. When a state cannot be tested yet, mark it `Blocked` with the missing seed, role, fixture, or browser automation reason.

## Acceptance Log Snippet

```text
Browser:
Viewport desktop:
Viewport narrow:
Role:
Route:
Normal:
Loading:
Empty:
Error:
No permission:
Dangerous action:
Save/export/detail:
Warnings:
Decision:
```

## Verification

```powershell
rg -n "Normal data|Loading|Empty|Backend error|No permission|Narrow viewport|Dangerous action|Acceptance Log Snippet" docs/refactor/fe0_browser_manual_acceptance_template.md
```

Result: Passed.
