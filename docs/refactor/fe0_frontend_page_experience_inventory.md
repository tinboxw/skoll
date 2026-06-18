# FE0 Frontend Page Experience Inventory

> Work Item: FE0-01  
> Date: 2026-06-19  
> Scope: Dashboard, User, Role, Permission, Menu, Plugin, Audit, Setting

## Summary

The current admin console already has usable operational surfaces for the core pages, but state handling and visual density are uneven. Audit is the strongest reference page after M2. Dashboard is the thinnest page. Plugin and Setting have rich functionality but need tighter information hierarchy. Permission, Menu, Role, and User pages need consistent empty/error/no-permission/narrow-width acceptance before broad FE3 polish starts.

## Priority Legend

| Priority | Meaning |
| --- | --- |
| P0 | Blocks reliable acceptance or repeated admin use. |
| P1 | Important UX polish or consistency issue. |
| P2 | Improvement that can wait behind FE1/FE3 standards. |

## Page Inventory

| Page | Current state | Strengths | Gaps | Priority |
| --- | --- | --- | --- | --- |
| Dashboard | Basic status cards for loaded plugins, enabled plugins, auth status. | Fast first render; low complexity. | Too sparse for admin operations; no system health, recent audit, permission/plugin warnings, or empty/error states. | P1 |
| User | List/add/edit/batch pages with permissions, table actions, role assignment, destructive confirmations. | Practical CRUD coverage; button permission checks; batch import path. | Filtering/search is light; no explicit no-permission page state; error/empty/narrow states need shared checklist. | P1 |
| Role | Role list/edit pages with permission editing and linked user context. | Role-permission workflow exists; destructive actions guarded. | Permission matrix relationship is split across Role and Permission pages; table scan and empty/error states need consistency review. | P1 |
| Permission | Permission matrix, policy rules, permission check tool. | Good operational fit for RBAC admins; grant/revoke confirmation exists. | Dense surface needs clearer grouping, sticky context, loading/empty/error/no-permission checks, and narrow layout review. | P0 |
| Menu | Menu registry editor for tree/order/visibility/permissions. | Directly supports M1 registry outcomes; dangerous saves are confirmed. | Tree editing density and responsive behavior need review; empty/default/custom state messaging needs standardization. | P0 |
| Plugin | Plugin inventory, lifecycle actions, config, logs, dev portal workflows. | Rich plugin-aware console; lifecycle actions guarded; config uses schema form. | Very large page mixes inventory and dev portal; needs information architecture split, loading/error/no-permission standardization, and heavy panel lazy strategy. | P0 |
| Audit | Type tabs, filters, quick suggestions, table, detail drawer, export, empty/error/no-permission states. | Best current reference for M2 operational page standard. | Strict fixture E2E still requires seeded data; browser smoke screenshots not yet captured. | P1 |
| Setting | Schema-driven common settings plus raw setting editor. | Reuses schema form; covers reset/fetch/save; grouped settings list. | Sensitive-setting treatment, validation copy, no-permission state, and narrow layout need review. | P1 |

## Cross-Page Findings

| Finding | Impact | Suggested task |
| --- | --- | --- |
| Loading/empty/error/no-permission states are not uniformly documented per page. | Acceptance can pass on build while visible workflows remain brittle. | FE0-05 and FE5-03 should reuse the audit baseline checklist. |
| Page density varies widely. | Dashboard feels too thin while Plugin feels overloaded. | FE1 layout standards, then FE3 page upgrades. |
| Permission-aware UI exists but visible no-permission states are inconsistent. | Restricted roles may see missing controls without a clear explanation. | FE5-04 permission-state checklist. |
| Dangerous actions mostly use confirmation but not yet centrally inventoried. | Reviewers cannot quickly verify destructive workflows. | FE2-05 confirm action helper standard. |
| Responsive behavior is page-local. | Narrow screens may regress without a shared standard. | FE1-07 responsive minimum standard. |

## Recommended Order

1. P0: Plugin, Permission, and Menu IA/state baseline.
2. P1: Dashboard content density and User/Role/Setting state alignment.
3. P1: Audit browser smoke screenshots once fixture seed exists.
4. P2: Copy polish, table microcopy, and secondary visual hierarchy after FE1 standards.

## Verification

```powershell
rg -n "Dashboard|User|Role|Permission|Menu|Plugin|Audit|Setting|P0|P1|P2" docs/refactor/fe0_frontend_page_experience_inventory.md
rg -n "const DashboardPage|const AuditPage|const MenuPage|const PermissionPage|const PluginPage|const RoleListPage|const SettingPage|const UserListPage" web/src/router/index.ts
```

Result: Passed.
