# FE6 Performance Execution Order

## Metadata

- Work Item: ADJ-TAIL-20260619-03
- Date: 2026-06-19
- Executor: Codex
- Scope: FE6 performance execution order and acceptance priorities
- Status: Accepted planning baseline

## Purpose

FE6 must run as a measured performance sequence. Do not start with scattered optimizations. Measure bundle output first, verify route lazy loading second, then work through render cost, request behavior, plugin heavy panels, acceptance records, and final regression checks.

## Execution Order

| Order | Work Item | Focus | Why first/next | Acceptance standard |
|---:|---|---|---|---|
| 1 | FE6-01 | Bundle baseline | Establish current build output, chunk sizes, and warnings before changing performance behavior. | `npm run build` output, chunk list, and warnings are recorded; bundle growth in later tasks has a baseline. |
| 2 | FE6-02 | Route lazy loading | Route splitting determines first-load cost and must be known before page-level optimization. | Major admin pages use dynamic imports or documented exceptions; route-level eager loading is not hidden by fallback routes. |
| 3 | FE6-03 | Heavy table rules | Tables are the largest repeated UI cost across User, Role, Permission, Menu, Plugin, and Audit pages. | Server-side pagination, stable dimensions, large-list limits, and virtualization triggers are documented. |
| 4 | FE6-04 | Request count and shared cache strategy | Once render risks are known, shared catalogs and stale request behavior can be controlled. | Menu, permission, dictionary, and plugin/catalog cache invalidation rules are explicit; repeated/stale writes are guarded. |
| 5 | FE6-05 | Plugin heavy panels and Dev Portal | Plugin/Dev Portal has logs, risk reports, release history, and tasks that can block first paint if loaded too early. | Heavy panels are deferred, request counts are recorded, and browser smoke/performance evidence is captured or Blocked with reason. |
| 6 | FE6-06 | Performance acceptance template | After concrete checks exist, acceptance logs need a stable field set for future frontend tasks. | `acceptance_log.md` can record bundle delta, route loading, request count, table/render risk, and page responsiveness. |
| 7 | ADJ-FE-20260619-08 | Regression checklist | The final FE6 checklist closes gaps across bundle, lazy loading, tables, request count, and plugin heavy panels. | Regression checklist includes baseline command, threshold, evidence path, and failure policy for each FE6 risk area. |
| 8 | ADJ-TAIL-20260619-04 | Tail threshold check | Once FE6 planning/checklists are in motion, confirm remaining Todo count and whether task optimization must run. | Remaining tasks are counted and the next-round threshold rule is recorded. |

## Priority Rationale

### Bundle

- Build output is the first measurable source of truth.
- Later route, table, cache, and plugin changes must explain any bundle growth.
- Validation command: `cd web && npm run build`.

### Route Lazy Loading

- Route-level dynamic imports reduce initial admin shell cost.
- Eager routes are allowed only with a documented reason.
- Validation command: `rg "component:\\s*\\(\\)\\s*=>" web/src/router web/src`.

### Heavy Tables

- Heavy tables must not rely on loading spinners to hide render cost.
- User, Role, Permission, Menu, Plugin, and Audit pages need stable dimensions and bounded data.
- Acceptance must mention pagination, virtualization trigger, and layout-shift risk.

### Request Count

- Menus, permissions, dictionaries, plugin metadata, and audit filters need explicit cache/invalidation behavior.
- Repeated route transitions must not create stale writes or duplicated requests.
- Acceptance must record whether repeated requests are expected, cached, cancellable, or blocked.

### Plugin Heavy Panels

- Logs, risk report, release history, Dev Portal tasks, and config/detail panels must load on demand.
- Plugin first paint should not wait for heavy secondary panels.
- Browser evidence is preferred; if browser tooling is unavailable, record `Blocked` with the exact missing tool or fixture.

## FE6 Acceptance Fields

Every FE6 task should record:

- Build command and result
- Bundle impact
- Route lazy loading status
- Heavy table/list risk
- Request count or request behavior
- Loading behavior
- Deferred panels
- Browser/performance evidence path or blocked reason
- Follow-up threshold

## Failure Policy

Mark the task `Failed` when:

- Bundle growth is unmeasured or unexplained.
- A route or heavy panel is made eager without a documented reason.
- A large table/list has no pagination, virtualization trigger, or stable layout rule.
- Repeated requests or stale writes are ignored.
- Browser evidence is claimed without a runnable browser tool or fixture.

Mark the task `Blocked` when:

- Required browser tooling, credentials, or seed data is missing.
- Build/typecheck cannot run because dependencies are unavailable.
- A baseline cannot be measured due to an external environment failure.

## Acceptance Outcome

- Bundle is first in the sequence.
- Route lazy loading follows bundle baseline.
- Heavy table rules precede cache/request strategy.
- Request count and shared cache strategy are explicit.
- Plugin heavy panels are checked after general render/request risks.
- Performance acceptance and regression checklist close the sequence.
