# FE5 Browser Smoke Minimum Set

## Metadata

- Work Item: ADJ-FE-20260619-07
- Date: 2026-06-19
- Executor: Codex
- Scope: minimum browser smoke set for FE5
- Status: Minimum set defined; browser execution remains blocked until runner and fixtures are available

## Command

List the minimum browser smoke matrix:

```powershell
cd web
npm run smoke:browser:minimum
```

Emit machine-readable output:

```powershell
cd web
npm run smoke:browser:minimum -- --json
```

Fail when required browser fixtures are missing:

```powershell
cd web
npm run smoke:browser:minimum -- --fail-on-blocked
```

## Required Environment

```powershell
$env:SKOLL_E2E_BASE_URL = "http://127.0.0.1:5173"
$env:SKOLL_E2E_ADMIN_USER = "admin"
$env:SKOLL_E2E_ADMIN_PASSWORD = "<secret>"
$env:SKOLL_E2E_RESTRICTED_USER = "restricted"
$env:SKOLL_E2E_RESTRICTED_PASSWORD = "<secret>"
```

If any required value is missing, the result must be recorded as `Blocked`, not `Passed`.

## Minimum Scenarios

| Scenario | Route | Role | Viewport | Required check |
|---|---|---|---|---|
| Login | `/login` | admin | desktop | Submit valid admin credentials and confirm the authenticated shell is reachable. |
| Permission denied | protected route/action | restricted | desktop | Confirm route guard, menu/action state, or API denial feedback is visible. |
| Plugin operation | `/plugin` | admin | desktop | Open plugin management and inspect one safe list/detail/config workflow. |
| Audit export | `/audit` | admin | desktop | Apply audit filters and trigger the export path with visible success/failure feedback. |
| Narrow navigation | `/dashboard` and one secondary route | admin | 390x844 | Confirm navigation remains reachable and primary content does not overlap or disappear. |

## Evidence Rules

Each scenario must record:

- Scenario name
- Route
- User role
- Viewport
- Action
- Expected result
- Actual result
- Result: `Passed`, `Failed`, or `Blocked`
- Screenshot, trace, report path, or blocked reason

Recommended evidence paths:

- `web/test-results/`
- `web/playwright-report/`
- `docs/refactor/evidence/fe5/`

## Current Runner Behavior

`web/scripts/fe5-browser-smoke-minimum.mjs` is a minimum-set recorder. It does not click the application yet. It validates that the five browser smoke scenarios are defined and that required browser fixture variables are present.

Current expected output in this thread is `Blocked` because browser automation and credentials are not available. The later Playwright implementation should keep the same scenario names and evidence fields, then replace `Blocked` rows with real `Passed` or `Failed` results.

## Acceptance Outcome

- Login is included.
- Permission denied is included.
- Plugin operation is included.
- Audit export is included.
- Narrow navigation is included.
- A repeatable script command exists.
- The blocked state is explicit when browser execution prerequisites are missing.
