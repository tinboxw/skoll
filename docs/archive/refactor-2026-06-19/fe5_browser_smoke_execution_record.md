# FE5 Browser Smoke Execution Record

## Metadata

- Work Item: ADJ-TAIL-20260619-02
- Date: 2026-06-19
- Executor: Codex
- Scope: minimum browser smoke execution record for FE5 acceptance
- Status: Recorded with blocked execution results

## Browser Tooling Status

The current thread does not expose a callable in-app browser control tool. A tool discovery check for browser control returned thread and automation tools only. Previous bundled Playwright attempts were also blocked because the local runtime did not provide `playwright-core`.

Browser tooling is unavailable in this thread.

This document is therefore an execution record, not a claim that the browser smoke scenarios passed. Each required scenario is recorded with an explicit `Blocked` result and retry conditions.

## Preconditions Expected For Retry

- Frontend dev server URL is available.
- Backend API is available with seeded data.
- Admin user can sign in successfully.
- Restricted role user exists and lacks at least one protected route/action.
- Plugin, audit, and navigation seed data are present.
- Browser automation can capture screenshots into an agreed evidence path.

## Minimum Smoke Matrix

| Scenario | Route/action | Expected | Actual | Result | Evidence / blocked reason |
|---|---|---|---|---|---|
| Login | Open login page, submit valid admin credentials, land in authenticated shell | User reaches the main admin layout and authenticated routes are accessible | Not executed | Blocked | No callable browser tool in current thread; no screenshot/click execution possible |
| Permission denied | Sign in as restricted role and open a protected page or action | UI denies access consistently through route guard/menu/action/API denial state | Not executed | Blocked | No callable browser tool in current thread; restricted role flow cannot be clicked or captured |
| Plugin operation | Open plugin management and perform one safe read/detail/config workflow | Plugin page remains usable and operation state is visible without layout or permission regressions | Not executed | Blocked | No callable browser tool in current thread; plugin workflow cannot be clicked or captured |
| Audit export | Open audit page and trigger export/download path with valid filters | Export command is reachable, state is visible, and no silent UI failure occurs | Not executed | Blocked | No callable browser tool in current thread; export flow cannot be clicked or captured |
| Narrow navigation | Re-run authenticated navigation at 390px viewport | Sidebar/top navigation remains reachable and route changes do not overlap or hide primary content | Not executed | Blocked | No callable browser tool in current thread; viewport resize and screenshot evidence cannot be captured |

## Retry Conditions

Retry conditions are listed here so this record can be replayed once browser automation is available.

Retry this smoke record when one of the following is true:

- The in-app browser control tool is available in the current thread.
- Playwright or an equivalent browser automation runtime is installed and callable.
- A maintained smoke script is added by FE5-06 or ADJ-FE-20260619-07.

The retry should replace each `Blocked` row with `Passed` or `Failed`, attach screenshot paths, and record the exact route, user role, viewport, and command used.

## Acceptance Outcome

- Required minimum scenarios are present: Login, Permission denied, Plugin operation, Audit export, Narrow navigation.
- Each scenario has a pass/fail/blocked result field.
- The current execution result is `Blocked` for all scenarios because browser tooling is unavailable.
- The task deliverable is complete as an honest execution record; product-level browser smoke remains pending a runnable browser tool.
