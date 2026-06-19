# FE5 Playwright Browser Automation Plan

## Metadata

- Work Item: FE5-06
- Date: 2026-06-19
- Executor: Codex
- Scope: evaluate Playwright/browser automation for FE5 acceptance
- Status: Decision recorded

## Current State

- `web/package.json` has `typecheck`, `build`, and Node-based smoke scripts.
- No Playwright dependency or browser test script is currently present.
- The current Codex thread does not expose a callable in-app browser control tool.
- `ADJ-TAIL-20260619-02` recorded the minimum browser smoke set as `Blocked` because browser automation was unavailable.

## Decision

Introduce Playwright for browser-level smoke checks.

Rationale:

- FE5 requires visible workflow validation, not only `typecheck` and `build`.
- Existing Node smoke scripts validate API and integration paths, but they cannot detect route rendering, hidden controls, text overlap, drawer/dialog behavior, or narrow viewport navigation.
- Browser smoke must produce durable pass/fail evidence with screenshots or traces when a route fails.

Non-goals:

- Do not replace existing Node smoke scripts.
- Do not make Playwright a full end-to-end regression suite in the first step.
- Do not mark browser smoke passed unless a real browser run completes.

## Proposed Dependencies And Scripts

Add in the implementation work item that follows this evaluation:

```powershell
cd web
npm install -D @playwright/test
npx playwright install chromium
```

Proposed scripts:

```json
{
  "test:browser:smoke": "playwright test -c playwright.config.ts --project=chromium --grep @smoke",
  "test:browser:smoke:headed": "playwright test -c playwright.config.ts --project=chromium --grep @smoke --headed",
  "test:browser:report": "playwright show-report"
}
```

Recommended evidence paths:

- `web/test-results/`
- `web/playwright-report/`
- `docs/refactor/evidence/fe5/` for curated screenshots copied into acceptance records when needed

## Coverage Scope

Minimum smoke coverage:

| Scenario | Route/action | Required evidence |
|---|---|---|
| Login | Open login page, submit admin credentials, assert authenticated shell | screenshot after successful route transition |
| Permission denied | Sign in as restricted role and open protected route/action | screenshot of denial state and assertion on visible forbidden copy or disabled action |
| Plugin operation | Open plugin page, inspect list/detail/config safe path | screenshot of plugin table and one safe operation state |
| Audit export | Open audit page, apply filters, trigger export path | assertion that export control is reachable and failure/success feedback is visible |
| Narrow navigation | Re-run authenticated navigation at 390px viewport | screenshot showing reachable navigation and no primary-content overlap |

Expanded page coverage after the minimum smoke is stable:

- Dashboard
- User
- Role
- Permission
- Menu
- Plugin
- Audit
- Setting

## Fixture And Environment Contract

Required environment variables:

```powershell
$env:SKOLL_E2E_BASE_URL = "http://127.0.0.1:5173"
$env:SKOLL_E2E_ADMIN_USER = "admin"
$env:SKOLL_E2E_ADMIN_PASSWORD = "<secret>"
$env:SKOLL_E2E_RESTRICTED_USER = "restricted"
$env:SKOLL_E2E_RESTRICTED_PASSWORD = "<secret>"
```

Required runtime state:

- Frontend dev server is running.
- Backend API is running with predictable seed data.
- Admin account has all core page permissions.
- Restricted account lacks at least one protected route/action.
- Plugin and audit sample data exist.

If any fixture is missing, the browser smoke result must be `Blocked`, not `Passed`.

## Execution Commands

Local minimum gate:

```powershell
cd web
npm run typecheck
npm run build
npm run test:browser:smoke
```

Debug run:

```powershell
cd web
npm run test:browser:smoke:headed
npm run test:browser:report
```

CI gate after stabilization:

```powershell
cd web
npm ci
npm run typecheck
npm run build
npx playwright install --with-deps chromium
npm run test:browser:smoke
```

## Failure Record Format

Every failed or blocked browser scenario must record:

- Scenario name
- Route
- User role
- Viewport
- Action
- Expected result
- Actual result
- Result: `Passed`, `Failed`, or `Blocked`
- Screenshot or trace path when available

## Rollout Order

1. Add Playwright dependency, config, and smoke script.
2. Implement login helper and fixture guard.
3. Automate the five minimum scenarios from `ADJ-TAIL-20260619-02`.
4. Record browser result fields in `acceptance_log.md`.
5. Expand coverage only after the minimum suite is stable.

## Acceptance Outcome

- Whether to introduce Playwright: yes, introduce it for browser-level smoke checks.
- Coverage scope: five minimum scenarios first, then core page expansion.
- Execution commands: documented for local, debug, and CI use.
- Current limitation: this work item is a plan; no browser run is claimed as passed.
