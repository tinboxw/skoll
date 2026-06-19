# FE6 Performance Acceptance Template

- Work Item: FE6-06
- Date: 2026-06-19
- Status: Canonical template
- Scope: frontend performance validation records for FE6 and later frontend tasks

## Purpose

Use this template whenever a frontend task can affect bundle size, route loading, table rendering, request count, heavy panels, or page responsiveness. It is the canonical FE6 acceptance format. Regression checklists should reference this document instead of duplicating another template.

## When To Use

- A route, page, drawer, table, plugin panel, or store request flow changes.
- A dependency, import, build config, or large component changes.
- A browser smoke or Playwright check captures page responsiveness or request behavior.
- A task is documentation-only but changes performance rules or baselines.

## Required Record

Copy the block below into `docs/refactor/acceptance_log.md` for each applicable task.

```markdown
### Performance Acceptance

- Page/route:
- Change type: Code | Docs | Build config | Test only
- Baseline reference:
- Typecheck:
- Build:
- Bundle impact:
- Route lazy loading:
- Heavy table/list risk:
- Request count or request behavior:
- Loading behavior:
- Deferred panels:
- Browser/performance evidence:
- Blocked evidence reason:
- Follow-up threshold:
```

## Field Rules

| Field | Required value |
|---|---|
| Page/route | Name every affected route, panel, drawer, iframe route, or shared store. |
| Change type | Use `Code`, `Docs`, `Build config`, or `Test only`; combine values when needed. |
| Baseline reference | Link to FE6-01 bundle baseline or the task-specific baseline used for comparison. |
| Typecheck | Record command and result, or `N/A` with reason. |
| Build | Record command and result, or `N/A` with reason. |
| Bundle impact | Record chunk delta, `No production bundle change`, or `Blocked` with reason. |
| Route lazy loading | State `Passed`, `N/A`, or `Failed`; eager loading needs a documented reason. |
| Heavy table/list risk | State pagination, max-height, virtualization trigger, stable dimensions, or `N/A`. |
| Request count or request behavior | Record first-paint requests, cached/reused requests, stale-write guard, or `Blocked`. |
| Loading behavior | Confirm loading state is meaningful and does not hide repeated requests. |
| Deferred panels | Confirm heavy panels, logs, drawers, and iframe content load on demand where applicable. |
| Browser/performance evidence | Provide command, screenshot path, trace path, or manual observation path. |
| Blocked evidence reason | Required when browser/tooling/seed data is missing; never claim browser evidence without a runnable tool. |
| Follow-up threshold | Define the number or condition that triggers follow-up work. |

## Result Vocabulary

- `Passed`: verified by command, source scan, browser evidence, or documented baseline.
- `Failed`: verification ran and the acceptance rule did not hold.
- `Blocked`: verification could not run because tooling, credentials, seed data, or environment was missing.
- `N/A`: the field is unrelated to the task and the reason is stated.

## Failure Policy

Mark the task `Failed` and retry before moving on when:

- bundle growth is unmeasured or unexplained;
- a route or heavy panel becomes eager without a documented reason;
- an unbounded table/list has no pagination, max-height, virtualization trigger, or stable size rule;
- repeated requests or stale writes are ignored;
- browser evidence is claimed without a runnable browser tool, trace, screenshot, or manual record.

Mark the evidence field `Blocked` when browser tooling, credentials, seed data, or runtime dependencies are unavailable. A task can pass source/documentation acceptance while browser evidence remains blocked only when the blocked reason and follow-up threshold are recorded.

## Canonical Validation

Use these searches to confirm a task recorded performance evidence:

```powershell
rg -n "### Performance Acceptance|Bundle impact|Route lazy loading|Request count|Deferred panels|Browser/performance evidence|Blocked evidence reason|Follow-up threshold" docs/refactor/acceptance_log.md
rg -n "FE6 Performance Acceptance Template|Required Record|Field Rules|Failure Policy|Canonical Validation" docs/refactor/fe6_performance_acceptance_template.md
```

## Example

```markdown
### Performance Acceptance

- Page/route: `/skoll/plugin`, Dev Portal panel
- Change type: Docs
- Baseline reference: `docs/refactor/fe6_bundle_baseline.md`
- Typecheck: N/A, docs-only task
- Build: N/A, docs-only task
- Bundle impact: No production bundle change
- Route lazy loading: Passed, `/skoll/plugin` covered by FE6-02
- Heavy table/list risk: Passed, Dev Portal tables use bounded height
- Request count or request behavior: Passed, request surfaces documented
- Loading behavior: N/A, no UI code changed
- Deferred panels: Passed, `activeHeavyPanel` guards risk and Dev Portal
- Browser/performance evidence: Blocked
- Blocked evidence reason: no callable browser tool in this thread
- Follow-up threshold: rerun browser smoke when Playwright/browser tool is available
```
