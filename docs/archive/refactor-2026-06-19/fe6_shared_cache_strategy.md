# FE6 Shared Data Cache Strategy

## Metadata

- Work Item: FE6-04
- Date: 2026-06-19
- Executor: Codex
- Scope: shared frontend data cache and invalidation strategy
- Status: Accepted strategy baseline

## Source Scan

Commands used:

```powershell
rg -n "defineStore|load|refresh|syncStatus|lastLoadedAt|lastQuery|lastSyncedAt|localStorage|force" web/src/stores web/src/views -g "*.ts" -g "*.vue"
```

Observed shared data areas:

- `web/src/stores/permissions.ts`
- `web/src/stores/navigation.ts`
- `web/src/stores/plugins.ts`
- `web/src/stores/user.ts`
- Page-local dictionaries in `web/src/views/Dictionary/index.vue`
- Page-local organization options in User add/edit/list pages
- Page-local audit query results in `web/src/views/Audit/index.vue`

## Current Baseline

| Data area | Current owner | Current cache behavior | Gap |
|---|---|---|---|
| Permission catalog | `usePermissionStore` | Query-normalized cache with `lastQuery`, `lastLoadedAt`, `force`, `refresh`, and `clear`. | Needs stale response guard if concurrent loads are introduced. |
| Menu tree | `useNavigationStore` | Store-owned state with `syncStatus`, `lastError`; save/reorder/visibility mutate or reload. | No `lastLoadedAt` or skip-if-fresh rule; repeated route consumers can trigger reloads if callers are added. |
| Plugin inventory | `usePluginStore` plus `syncBackendPlugins` | Store-owned state with `syncedFromServer`, `syncStatus`, `syncAttempts`, `lastSyncedAt`, `degradedMode`. | Sync is guarded by bootstrap task, but explicit refresh policy and stale response guard should be documented in future code work. |
| User session/permissions | `useUserStore` and localStorage | Session is persisted; `hydrateProfile` preserves local session on temporary `/auth/me` failure. | Auth changes must invalidate permission/menu/plugin derived state. |
| Dictionaries | Page-local `Dictionary/index.vue` | Loaded and saved inside the page. | Shared dictionary reads should move to a store when used by multiple pages or generated forms. |
| Organization options | Page-local User add/edit/list loaders | Departments/positions are loaded per page. | Repeated option fetches should move to a bounded reference-data cache. |
| Audit query results | Page-local `Audit/index.vue` | Filtered query result is page state. | Keep page-local; do not cache unbounded audit logs globally. Add stale request guard if rapid filter changes are automated. |

## Cache Classes

### Session Cache

Used for:

- Auth profile
- Auth permissions
- Default home preference

Rules:

- Persist only user/session scoped data.
- Clear on logout.
- Rehydrate before protected route checks when possible.
- Do not keep stale session permissions after login/logout/account switch.

### Catalog Cache

Used for:

- Permission catalog
- Menu tree
- Plugin inventory
- Dictionaries once shared outside the dictionary page
- Organization departments/positions once shared by multiple user screens

Rules:

- Store owner records `syncStatus`, `lastError`, `lastLoadedAt`, and the normalized query or scope.
- `load()` skips work when current data is fresh for the same query/scope.
- `refresh()` forces a network call.
- Mutations patch local state only when the returned entity is authoritative; otherwise reload the scope.
- Store exposes `clear()` for auth logout, tenant switch, or destructive reset.

### Page Query Cache

Used for:

- Audit query results
- Table page results
- Detail drawer fetches

Rules:

- Keep unbounded result sets page-local.
- Query key includes filters, page, page size, sort, and current permission scope when relevant.
- Rapid filter changes must not allow stale responses to overwrite newer results.
- Export/download must use the same active query object as the table.

## Invalidation Matrix

| Event | Permission catalog | Menu tree | Plugin inventory | Dictionaries | Organization options | Page query results |
|---|---|---|---|---|---|---|
| Login | Refresh after session is set | Refresh after session is set | Sync backend plugins | Load on demand | Load on demand | Clear page state |
| Logout | Clear | Clear or fall back to defaults | Clear backend records / mark unsynced | Clear shared cache when introduced | Clear shared cache when introduced | Clear page state |
| Permission enable/disable | Patch returned resource; refresh if query excludes it | No direct invalidation unless menu permissions change | No direct invalidation | N/A | N/A | Refresh affected protected page if visible |
| Menu save/reorder/visibility | N/A | Patch or reload authoritative tree | May affect plugin/default home access; validate default home | N/A | N/A | N/A |
| Plugin sync/install/disable | Refresh permission catalog if plugin permissions changed | Refresh menu tree if plugin menus changed | Refresh plugin inventory | N/A | N/A | Refresh plugin page state |
| Dictionary save | N/A | N/A | N/A | Patch authoritative payload or refresh dictionary scope | N/A | Refresh generated forms that use dictionary options |
| Organization update | N/A | N/A | N/A | N/A | Refresh departments/positions scope | Refresh User filters/forms if visible |
| Tenant/org scope switch | Clear | Clear | Clear | Clear | Clear | Clear |

## Stale Write Guard

Any shared `load()` or page query that can overlap must use one of these guards:

- Request sequence number: only the latest sequence may write state.
- AbortController: cancel the previous request when a newer query starts.
- Query key compare: response can write only if its normalized query matches the current active query.

Acceptance:

- The chosen guard is documented in the store or page acceptance log.
- Loading state belongs to the active request, not any older request.
- Failed older requests must not replace a newer success with an error state.

## Request Count Rules

- Opening Dashboard should not independently refetch menu, plugin, and permission catalogs if bootstrap already loaded them.
- Navigating between User add/edit/list should not repeatedly fetch the same department/position option lists once a shared organization cache exists.
- Permission pages should use `usePermissionStore.load()` with query normalization instead of raw repeated catalog calls.
- Menu pages should use `useNavigationStore` as the single menu tree owner.
- Plugin pages should use `usePluginStore` and explicit plugin sync actions rather than duplicate inventory fetches.
- Audit remains query-local and should not be globally cached.

## Acceptance Checklist For Future Cache Work

Every cache-related task must record:

- Data owner: store or page
- Cache scope: session, catalog, or page query
- Freshness rule: skip-if-fresh, force refresh, TTL, or no cache
- Invalidation events
- Stale write guard
- Loading/error state ownership
- Request count before/after when measurable
- Browser or manual evidence path, or `Blocked` reason

## Acceptance Outcome

- Menu cache and invalidation rules are defined.
- Permission cache and invalidation rules are defined.
- Dictionary cache promotion rule is defined.
- Organization option cache promotion rule is defined.
- Plugin inventory/cache rules are defined.
- Stale write and request count policies are defined.
