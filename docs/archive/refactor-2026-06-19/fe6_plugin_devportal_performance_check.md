# FE6-05 Plugin / Dev Portal Performance Check

- Work Item: FE6-05
- Date: 2026-06-19
- Status: Source and documentation check passed; browser evidence blocked
- Scope: `web/src/views/Plugin/index.vue`, `web/src/plugins/index.ts`, `web/src/stores/plugins.ts`

## Source Scan

```powershell
rg -n "v-if=|v-show=|detailDrawerOpen|openPluginDetail|risk|visibleRiskRows|devTask|devRelease|devRollout|openDevTaskDrawer|iframe|createRemotePluginView|syncBackendPlugins|max-height|el-table|refreshDevPortal|fetch\(|apiGet|apiPost" web/src/views/Plugin/index.vue web/src/plugins/index.ts web/src/stores/plugins.ts
rg -n "activeHeavyPanel|v-if=""activeHeavyPanel|refreshDevPortal|openDevTaskDrawer|createRemotePluginView|max-height=""" web/src/views/Plugin/index.vue web/src/plugins/index.ts
```

## First Paint

- `activeHeavyPanel` defaults to `inventory`.
- `/skoll/plugin` is already covered by FE6-02 route lazy loading, so the plugin page view is not bundled into the initial admin shell.
- The default inventory panel renders first; risk analysis and Dev Portal panels are not rendered until the user switches the heavy panel selector.

## Deferred Heavy Panels

- Risk analysis is guarded by `v-if="activeHeavyPanel === 'risk'"`.
- Dev Portal is guarded by `v-if="activeHeavyPanel === 'devportal'"`.
- Dev task drawer is additionally guarded by `v-if="activeHeavyPanel === 'devportal'"` and opens only from Dev Portal actions.
- Plugin detail drawer opens through `detailDrawerOpen`; log, config, and debug content are tied to explicit detail actions instead of page first paint.

## Request Behavior

- Plugin inventory synchronization is owned by `syncBackendPlugins` and `usePluginStore`.
- Risk rows are computed from existing plugin store and Dev Portal task state; the risk panel does not need its own first-paint request.
- Dev Portal request surfaces are concentrated in `refreshDevPortal` and explicit actions:
  - `/v1/plugins/dev/config`
  - `/v1/plugins/dev/projects`
  - `/v1/plugins/dev/release-orders`
  - `/v1/plugins/dev/release-tasks`
  - `/v1/plugins/dev/rollout-tasks`
  - task detail and log endpoints opened by `openDevTaskDrawer`
- Remote plugin iframe content is created by `createRemotePluginView` and app plugin routes in `web/src/plugins/index.ts`; the iframe fetch happens when those route components mount, not during plugin inventory first paint.

## Table And Panel Constraints

- Dev Portal tables use bounded panel dimensions, including `max-height="260"` on release and task tables.
- Plugin inventory tables keep action columns fixed and use tooltip/cell constraints for dense text.
- Risk analysis is deferred, but the risk table should get a max-height or virtualization guard if `visibleRiskRows` can exceed 200 rows in real deployments.

## Browser Evidence

- Result: Blocked
- Reason: this thread has no callable browser tool, and the local Playwright runtime needed for browser smoke is not available.
- Follow-up: rerun FE6-05 with browser automation and collect first-paint request count, heavy-panel request count, and screenshot evidence for inventory, risk, and Dev Portal panels.

## Acceptance Outcome

| Check | Result | Evidence |
|---|---|---|
| Plugin route lazy loading linkage | Passed | FE6-02 covers `/skoll/plugin`; this check confirms plugin host behavior. |
| Heavy panel deferred rendering | Passed | `activeHeavyPanel` and `v-if` guards keep risk and Dev Portal off first paint. |
| Request count behavior documented | Passed | Dev Portal and iframe request surfaces are isolated from inventory first paint in source. |
| Table/panel constraints documented | Passed | Dev Portal table height constraints are present; risk table follow-up recorded. |
| Browser smoke evidence | Blocked | No browser automation tool is callable in this thread. |

## Follow-Ups

- Add request-counter assertions to the FE5 browser smoke suite once Playwright/browser tooling is available.
- Add a bounded height or virtualization rule to the risk table if real risk rows exceed 200.
- Consider stale-write guards around `refreshDevPortal` if rapid project switches or repeated refresh clicks cause overlapping requests.
- FE6-06 should include fields for route chunk, first-paint requests, deferred panel requests, table constraints, and browser evidence status.
