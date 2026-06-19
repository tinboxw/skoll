# FE0 Audit Page UX Baseline

> Work Item: ADJ-20260619-03  
> Date: 2026-06-19  
> Scope: `web/src/views/Audit/index.vue`

## Purpose

This baseline defines the minimum UX acceptance standard for the audit page before later frontend refactor tasks continue. It is a checklist, not a replacement for implementation tests.

## User Workflow

1. Open the audit page from the admin sidebar.
2. Scan the current audit event set by type, actor, action, resource, risk, and time range.
3. Narrow results with tabs, filters, and quick suggestions.
4. Open an event detail drawer and inspect trace, metadata, diff, and source data.
5. Export the current filtered result set as CSV.
6. Handle empty, backend error, and no-permission states without losing filter context.

## Acceptance Matrix

| Area | Required behavior | Current anchor | Status |
| --- | --- | --- | --- |
| Page header | Title, description, refresh, export, and clear actions are visible without instructional filler. | `page-header`, `header-actions` | Passed |
| Type tabs | `all`, `operation`, `login`, `error`, `plugin`, and `security` are available and reload data on change. | `AUDIT_TYPE_TABS`, `el-tabs` | Passed |
| Filters | Actor, action, resource type, resource ID, risk, time range, and limit are available. | `filters`, `buildAuditEventQuery` | Passed |
| Quick filters | Actor, action, and resource suggestions can be applied without typing repeated values. | `quickActors`, `quickActions`, `quickResources` | Passed |
| Table scan | Stable columns show id, actor, action, resource, and occurred time; long IDs use overflow tooltip. | `el-table`, `el-table-column` | Passed |
| Pagination | Page size and page navigation are stable and visible when rows exist. | `el-pagination` | Passed |
| Detail drawer | Drawer shows summary fields plus trace, metadata, diff, and source data. | `el-drawer`, `detail-section` | Passed |
| Export | Export uses the same typed query as the list and reports success/error. | `exportCSV`, `exportAuditEvents(buildAuditEventQuery())` | Passed |
| Dangerous action | Clear by time range uses confirmation before deleting. | `clearByRange`, `confirmAction` | Passed |
| Loading | Table and detail drawer show loading states during async calls. | `v-loading`, `el-skeleton` | Passed |
| Empty | Empty state is visible when a successful query returns no rows. | `el-empty` | Passed |
| Error | Backend errors surface as visible alert text. | `el-alert v-if="error"` | Passed |
| No permission | 403 is mapped to a no-permission result state. | `forbidden`, `el-result` | Passed |
| Narrow viewport | Header, actions, and filters collapse to grid layout under 960px. | `@media (max-width: 960px)` | Passed |

## Manual Smoke Checklist

Use this checklist after any visible audit page change:

| Scenario | Steps | Expected result |
| --- | --- | --- |
| Default data state | Open `/audit` as an allowed admin user. | Page loads, tabs and filters render, table or empty state is visible. |
| Type switch | Click each audit type tab. | Query reloads and URL/filter state stays coherent. |
| Filter search | Enter actor/action/resource/risk/time filters and refresh. | Query uses the entered filters and repeated values become suggestions. |
| Detail drawer | Click a table row. | Drawer opens with summary, trace, metadata, diff, and source data; skeleton appears while loading. |
| Export | Click export after applying filters. | CSV download starts and success alert appears; failures show an error alert. |
| Empty state | Apply a filter that returns no rows. | Empty state text is visible and no stale rows remain. |
| Backend error | Simulate a non-403 API failure. | Error alert is visible and controls remain usable for retry. |
| No permission | Open as a restricted user without `audit.read`. | No-permission result appears, not an empty table. |
| Narrow viewport | Check at 390px width. | Header actions and filter controls do not overlap or overflow. |

## Verification

```powershell
rg -n "AUDIT_TYPE_TABS|el-tabs|el-form|el-result|el-empty|el-drawer|exportCSV|clearByRange|@media" web/src/views/Audit/index.vue
cd web
npm run typecheck
npm run build
```

Result: Passed.
