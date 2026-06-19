# FE6 Table Performance Rules

## Metadata

- Work Item: FE6-03
- Date: 2026-06-19
- Executor: Codex
- Scope: table/list performance rules for core admin pages
- Status: Accepted rule baseline

## Source Scan

Commands used:

```powershell
rg -n "el-table|el-pagination|filteredRows|slice\(|max-height|row-key|v-loading" web/src/views -g "*.vue"
```

Observed table-heavy pages:

- `web/src/views/User/list.vue`
- `web/src/views/User/batch.vue`
- `web/src/views/User/edit.vue`
- `web/src/views/Role/list.vue`
- `web/src/views/Role/edit.vue`
- `web/src/views/Permission/index.vue`
- `web/src/views/Menu/index.vue`
- `web/src/views/Plugin/index.vue`
- `web/src/views/Audit/index.vue`
- `web/src/views/Dictionary/index.vue`
- `web/src/views/Organization/index.vue`
- `web/src/views/Setting/index.vue`

## Mandatory Rules

### Server-Side Pagination

Use server-side pagination for data that can exceed one normal page of admin work.

Required for:

- User list
- Audit events
- Plugin marketplace/runtime inventory when backend count can grow
- Operation/log style tables
- Any generated CRUD list

Acceptance:

- Query params include page and page size, or the task records why the source is bounded.
- Pagination state is part of the store/API query boundary, not only a local `slice`.
- Filter changes reset or validate the current page.
- Export/download actions use the active filter query, not only the currently rendered page.

### Client-Side Bounded Lists

Client-side filtering is allowed only when the data source is bounded and documented.

Allowed examples:

- Role permission options
- Menu tree drafts
- Organization departments/positions when seeded as bounded reference data
- Setting groups when loaded as bounded system configuration
- Dictionary item lists when dictionary item count is bounded by configuration

Acceptance:

- The page records why the list is bounded.
- Filtering is computed from a stable source array.
- The table has `row-key` when rows have stable identifiers.
- The rendered row count is visible in summary or pagination/state text when filters are active.

### Virtualization Trigger

Introduce virtualization or a virtualized table/list when any one of these is true:

- More than 500 rows can render in a single table.
- A permission matrix can exceed 100 resources or 1000 rendered permission cells.
- A tree table can exceed 300 flattened nodes.
- A detail drawer can render more than 200 log/task/history rows.
- Browser smoke or manual review shows scroll jank, layout shift, or delayed interaction.

If virtualization is not implemented immediately, the work item must record `Blocked` or a follow-up with the exact page, dataset size, and reason.

### Stable Dimensions

Every table-heavy page must avoid layout jumps.

Required:

- `row-key` for selectable, editable, expandable, or frequently updated rows.
- Fixed action-column width for row operations.
- `min-width` or `width` on every important column.
- `show-overflow-tooltip` or equivalent truncation for long IDs, emails, paths, traces, plugin IDs, and error text.
- `max-height` or bounded panel height for drawer/tab tables that live inside secondary panels.
- Empty/loading/no-permission states must preserve the table region instead of collapsing the page.

### Loading Behavior

Loading states must represent real pending work and must not hide repeated full-table recomputation.

Required:

- `v-loading` should be tied to API/store loading, save, import, export, or binding state.
- Local filters should not set global page loading.
- Bulk actions should disable selection/action controls while operating.
- Repeated refreshes must be guarded against stale writes or duplicate updates in FE6-04.

## Page Risk Matrix

| Page | Current pattern | Risk | Rule |
|---|---|---|---|
| User list | `filteredRows`, `row-key`, fixed action column | Client-side filtering can grow with user count | Move to server-side pagination when backend count is unbounded; keep row-key and fixed widths. |
| User batch | Excel import rows and results tables | Large imported workbook can render too many rows | Cap preview rows or virtualize over 500 rows; keep `xlsx` route-owned. |
| User edit | Role assignment table | Bounded by roles, but selectable rows need stable identity | Keep row-key and disable selection during binding. |
| Role list | `filteredRows` client-side table | Role count usually bounded | Document bounded source; server paginate if role count becomes tenant-scale. |
| Role edit | Permission/user tables | Permission/user associations can grow | Permission table needs virtualization threshold; associated users should paginate when unbounded. |
| Permission | Permission matrix and policy tables | Matrix cell count can grow quickly | Virtualize or group resources above 100 resources/1000 cells. |
| Menu | Tree/draft table | Flattened tree can become large | Virtualize or collapse tree above 300 nodes; preserve row-key. |
| Plugin | Inventory, risk, Dev Portal task tables | Multiple secondary tables in heavy panels | Keep heavy panels deferred; use max-height in panels; record request count in FE6-05. |
| Audit | Local paged items with `slice` | Audit logs are unbounded | Prefer server-side pagination; export must use active query. |
| Dictionary | Dictionary and item tables | Usually bounded config data | Document bounded source; virtualize item list if config exceeds 500 items. |
| Organization | Department/position tables | Tree/reference data can grow | Server paginate or virtualize for tenant-scale organizations. |
| Setting | Grouped setting tables | Bounded system config | Keep bounded groups and stable columns; avoid rendering giant raw values inline. |

## Acceptance Checklist For Future Table Work

Each table task must record:

- Data source: server-paginated, bounded client list, or generated fixture
- Row count risk: expected max rows and threshold
- Pagination: server-side, local bounded, or not applicable
- Virtualization: implemented, not needed, or follow-up/Blocked
- Stable dimensions: row-key, column widths, fixed actions, overflow behavior
- Loading behavior: API/store loading, local filter, bulk operation, or drawer/panel loading
- Narrow viewport: horizontal scroll, column hiding, or drawer fallback
- Export/download behavior when applicable

## Acceptance Outcome

- Server-side pagination rule is defined.
- Client-side bounded-list rule is defined.
- Virtualization trigger is defined.
- Stable dimensions rule is defined.
- Page-by-page table risk matrix is recorded.
