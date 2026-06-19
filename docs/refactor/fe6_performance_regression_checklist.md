# FE6 Performance Regression Checklist

- Work Item: ADJ-FE-20260619-08
- Date: 2026-06-19
- Status: Accepted regression checklist
- Canonical acceptance template: `docs/refactor/fe6_performance_acceptance_template.md`

## Purpose

Use this checklist before closing FE6 or when a later frontend task touches bundle size, route loading, heavy tables, shared requests, plugin heavy panels, or browser performance evidence. This document defines what to check; FE6-06 defines how to record the result in `docs/refactor/acceptance_log.md`.

## Regression Matrix

| Area | Baseline / command | Threshold | Evidence path | Failure policy |
|---|---|---|---|---|
| Bundle baseline | `cd web; npm run build`; compare with `docs/refactor/fe6_bundle_baseline.md` | New large chunk or dependency growth must be explained; existing warnings must not grow without reason. | `docs/refactor/acceptance_log.md` using FE6-06 `Bundle impact` field. | Failed if growth is unmeasured or unexplained. |
| Route lazy loading | `rg -n "const .*Page = \\(\\) => import|component: .*Page|from \"\\.\\./views|from '\\.\\./views" web/src/router/index.ts web/src` | Major admin pages remain lazy-loaded; eager routes require a documented reason. | `docs/refactor/acceptance_log.md` using FE6-06 `Route lazy loading` field. | Failed if a heavy route becomes eager without reason. |
| Heavy tables/lists | `rg -n "el-table|el-pagination|max-height|virtual|row-key|slice\\(" web/src/views -g "*.vue"` plus `docs/refactor/fe6_table_performance_rules.md` | Unbounded lists need server pagination, max-height, virtualization trigger, or stable dimensions. | `docs/refactor/acceptance_log.md` using FE6-06 `Heavy table/list risk` field. | Failed if a large list has no bounded rendering rule. |
| Request count and cache behavior | `rg -n "defineStore|load|refresh|syncStatus|lastLoadedAt|lastQuery|AbortController|requestId|force" web/src/stores web/src/views -g "*.ts" -g "*.vue"` plus `docs/refactor/fe6_shared_cache_strategy.md` | Shared catalogs have owner, invalidation, and stale-write behavior; repeated requests are cached, guarded, or documented. | `docs/refactor/acceptance_log.md` using FE6-06 `Request count or request behavior` field. | Failed if duplicated requests or stale writes are ignored. |
| Plugin heavy panels | `rg -n "activeHeavyPanel|v-if=\"activeHeavyPanel|detailDrawerOpen|openDevTaskDrawer|createRemotePluginView|max-height" web/src/views/Plugin/index.vue web/src/plugins/index.ts` plus `docs/refactor/fe6_plugin_devportal_performance_check.md` | Risk, Dev Portal, detail/log/drawer, and iframe content stay on demand. | `docs/refactor/acceptance_log.md` using FE6-06 `Deferred panels` and `Browser/performance evidence` fields. | Failed if heavy plugin panels block first paint without reason. |
| Browser/performance evidence | FE5 browser smoke minimum set or equivalent Playwright/browser trace when tooling is available. | Browser evidence is captured, or `Blocked` includes the exact missing tool, credentials, seed data, or runtime dependency. | Screenshot, trace, smoke output, or `Blocked evidence reason` in `docs/refactor/acceptance_log.md`. | Failed if browser evidence is claimed without a runnable tool or fixture. |

## Required Closeout Steps

1. Run the relevant command from the matrix.
2. Record the result with the FE6-06 `### Performance Acceptance` block.
3. Use `Passed`, `Failed`, `Blocked`, or `N/A` exactly as defined by `docs/refactor/fe6_performance_acceptance_template.md`.
4. Retry before moving on when the result is `Failed`.
5. Record a follow-up threshold when evidence is `Blocked` or a risk is accepted temporarily.

## FE6 Closeout Checklist

| Item | Status | Evidence |
|---|---|---|
| Bundle baseline exists | Passed | `docs/refactor/fe6_bundle_baseline.md` |
| Route lazy loading coverage exists | Passed | `docs/refactor/fe6_route_lazy_loading_coverage.md` |
| Heavy table rules exist | Passed | `docs/refactor/fe6_table_performance_rules.md` |
| Shared cache/request strategy exists | Passed | `docs/refactor/fe6_shared_cache_strategy.md` |
| Plugin/Dev Portal performance check exists | Passed | `docs/refactor/fe6_plugin_devportal_performance_check.md` |
| Canonical acceptance template exists | Passed | `docs/refactor/fe6_performance_acceptance_template.md` |
| Browser tool gap is explicit | Passed | FE6-05 records browser evidence as `Blocked`; this checklist requires the same honesty. |

## Canonical Validation

```powershell
rg -n "Regression Matrix|Bundle baseline|Route lazy loading|Heavy tables/lists|Request count and cache behavior|Plugin heavy panels|Browser/performance evidence|FE6-06|Failure policy|Canonical Validation" docs/refactor/fe6_performance_regression_checklist.md
rg -n "FE6 Performance Acceptance Template|### Performance Acceptance|Bundle impact|Route lazy loading|Request count|Deferred panels|Blocked evidence reason" docs/refactor/fe6_performance_acceptance_template.md docs/refactor/acceptance_log.md
```

## Next Step

After this checklist is accepted, run `ADJ-TAIL-20260619-04` to record the tail threshold snapshot and confirm the remaining FE6 closeout order.
