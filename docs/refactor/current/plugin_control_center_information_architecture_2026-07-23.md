# Plugin Control Center Information Architecture

> Work Item: `BF3-01`
> Date: 2026-07-23
> Status: Frozen implementation contract for `BF3-02` through `BF3-06`.
> Audience: product, frontend, backend, plugin authors, and QA.

## Decision

The plugin control center is an operator workspace, not a single settings page. Replace the current monolithic plugin view with routed work surfaces that share one plugin context and one status vocabulary.

The current contract is intentionally direct:

- no legacy plugin-console route
- no compatibility tab or duplicated screen
- no fallback data source
- no mixed operator and developer navigation
- no plugin-specific hard-coded business UI in the control center

The developer portal becomes a separate product area. Plugin installation, lifecycle, runtime, capabilities, data, migrations, jobs, diagnostics, and configuration remain in the operator control center.

## Current Audit

| Finding | Evidence | Consequence | Resolution owner |
| --- | --- | --- | --- |
| One route owns six product areas | `web/src/views/Plugin/index.vue` is 2,994 lines | Loading and errors collide; areas cannot be tested or loaded independently | BF3-02 through BF3-04 |
| Inventory, marketplace, risk, and developer tools share one local tab state | `activeHeavyPanel` selects four unrelated areas | URLs cannot preserve operator intent and browser navigation is weak | BF3-02 |
| Install preflight is permanently above inventory | The install panel renders before the primary workspace | Routine health inspection competes with an occasional workflow | BF3-02 |
| Plugin detail is a broad drawer | Permissions, menu, config, assets, logs, and release status are drawer tabs | Data, migrations, jobs, audit, and errors have no stable owner | BF3-02 through BF3-04 |
| Health is inferred from manifest fields | `pluginHealthStatus` combines multiple optional values | Stale, unknown, disabled, degraded, and crashed are not authoritative states | BF3-02 |
| Inspector state is page-global | Config, debug, and logs share `selectedPlugin` and `activeInspectorPanel` | One failed request can obscure a different workflow | BF3-02 |
| Tables and actions are duplicated | System and app inventories repeat columns and command logic | Visual and permission behavior can drift | BF3-02 |
| Developer release tools are embedded in operator inventory | Scaffold, package, rollout, rollback, and release review share the same page | Personas and permission boundaries are unclear | BF3-04 |

## Route Map

| Route | Surface | Primary owner | Required permission |
| --- | --- | --- | --- |
| `/skoll/plugin-center` | Installed plugin inventory and fleet attention queue | BF3-02 | `plugin.read` |
| `/skoll/plugin-center/install` | Source/package validation, impact review, and install | BF3-02 | `plugin.manage` |
| `/skoll/plugin-center/marketplace` | Discoverable packages, signature, risk, and install entry | BF3-02 | `plugin.read` |
| `/skoll/plugin-center/:pluginId/overview` | Identity, install state, health summary, version, and next action | BF3-02 | `plugin.read` |
| `/skoll/plugin-center/:pluginId/runtime` | Process, heartbeat, endpoints, routes, and runtime events | BF3-02 | `plugin.read` |
| `/skoll/plugin-center/:pluginId/capabilities` | Host services, permissions, menus, routes, and declared resources | BF3-02 | `plugin.read` |
| `/skoll/plugin-center/:pluginId/data` | Owned schemas, tables, size, scope, retention, and uninstall policy | BF3-03 | `plugin.read` |
| `/skoll/plugin-center/:pluginId/migrations` | Current version, steps, history, plan, and allowed actions | BF3-03 | `plugin.read` |
| `/skoll/plugin-center/:pluginId/jobs` | Scheduled, running, failed, completed, and dead-letter jobs | BF3-04 | `plugin.read` |
| `/skoll/plugin-center/:pluginId/diagnostics` | Correlated logs, audit events, errors, and request/task identifiers | BF3-04 | `plugin.read` |
| `/skoll/plugin-center/:pluginId/settings` | Schema-driven current configuration | BF3-02 | `plugin.read` |
| `/skoll/plugin-development/*` | Scaffold, validate, package, release, rollout, and rollback | BF3-04 | `plugin.manage` |

`/skoll/plugin-center/:pluginId` redirects directly to `overview`. The dedicated namespace cannot collide with business-plugin entry routes under `/skoll/plugins/<plugin-id>`. There is no second detail drawer or alternate plugin-detail route.

## Navigation Hierarchy

The fleet level has three commands: **Installed**, **Install**, and **Marketplace**. Risk is not a separate destination; actionable risk is an attention queue on Installed and a signal on each relevant work surface.

The plugin workspace uses a stable left or top sub-navigation in this order:

1. Overview
2. Runtime
3. Capabilities
4. Data
5. Migrations
6. Jobs
7. Diagnostics
8. Settings

Each item may display one compact semantic count. Counts represent actionable records only, such as failed jobs or unresolved errors. Empty counts are omitted.

## Ownership Map

| Concern | Canonical surface | Canonical source | Mutation owner |
| --- | --- | --- | --- |
| Install state | Overview | package/install registry | Install workflow |
| Runtime health | Runtime | managed-process health snapshot | Lifecycle command bar |
| Routes and host capabilities | Capabilities | resolved current manifest and host capability registry | Plugin package install |
| Permissions and menus | Capabilities | permission and menu registries | Plugin package install |
| Data ownership | Data | plugin datastore/schema registry | Migration lifecycle |
| Migration version and history | Migrations | migration planner and records | Migration lifecycle |
| Background jobs | Jobs | plugin job service | Job actions |
| Audit evidence | Diagnostics | audit store filtered by plugin ID | Host services and lifecycle |
| Runtime and request errors | Diagnostics | correlated runtime/error feed | Runtime and job retry actions |
| Configuration | Settings | plugin config service and current schema | Settings form |

No concern may be rendered from a second locally reconstructed source. Summary data links to its canonical surface.

## Status Model

### Install State

`not_installed | validating | blocked | installing | installed | uninstalling | failed`

### Lifecycle State

`disabled | starting | ready | degraded | stopping | crashed | stale`

### Data State

`unavailable | ready | migration_required | migrating | migration_failed | policy_blocked`

### Job State

`scheduled | queued | running | succeeded | failed | dead_letter | cancelled`

### Request State

Every routed surface independently owns:

`idle | loading | ready | empty | stale | forbidden | error`

`stale` keeps the last successful snapshot visible with its capture time and a retry action. It never presents stale data as current health.

## Page Contracts

### Fleet Inventory

The default view is one table, not separate system and app tables. Required columns are plugin identity, app/family, version, install state, runtime state, risk/attention, last observed time, and compact actions. Filters cover keyword, family, install state, lifecycle state, and attention-only.

Row selection opens the routed Overview surface. Lifecycle actions remain in a compact menu and use the same command component as Overview.

### Plugin Workspace Shell

The shell owns only:

- plugin identity and back navigation
- stable workspace navigation
- lifecycle status and last observation time
- lifecycle command bar
- route outlet

The shell does not fetch page-specific tables, logs, jobs, migration steps, or configuration.

### Overview

Overview answers four questions in one viewport: what is installed, is it ready, what requires attention, and what action is safe now. It summarizes other surfaces with links and never duplicates their detailed tables.

### Runtime And Capabilities

Runtime displays process truth and observation freshness. Capabilities displays declared and resolved contract truth. A missing declared capability is a capability error; a declared capability whose process is unavailable is a runtime error.

### Data And Migrations

Data describes current ownership and policy. Migrations describes version movement and evidence. Destructive data consequences must appear before confirmation and in the resulting audit entry.

### Jobs And Diagnostics

Jobs owns execution and retry/dead-letter commands. Diagnostics owns investigation. Job rows link to diagnostics by correlation ID; diagnostics links back to the originating route, process, job, audit event, or request.

## Component Contract

| Component | Responsibility | Must not own |
| --- | --- | --- |
| `PluginFleetTable` | Fleet filters, stable columns, row navigation | Install forms or detail data |
| `PluginWorkspaceShell` | Identity, navigation, lifecycle command placement | Child-page requests |
| `PluginStateSummary` | Compact install/runtime/data/job signals | State inference from arbitrary strings |
| `PluginLifecycleCommands` | Permission-aware enable, disable, restart, uninstall commands | Confirmation copy or API response rendering |
| `PluginInstallWizard` | Validate, review impact, confirm, execute, result | Marketplace search |
| `PluginCapabilityTable` | Routes, services, permissions, menus, resource declarations | Runtime health |
| `PluginDataInventory` | Schema/table/policy facts | Migration execution |
| `PluginMigrationTimeline` | Plan, current version, history, allowed commands | Data-size inference |
| `PluginJobTable` | Filtered jobs and allowed job commands | Raw process logs |
| `PluginDiagnosticsTimeline` | Correlated logs, audit, errors, and navigation | Job mutation |
| `PluginSettingsForm` | Schema-driven config validation and save | Raw JSON editing when a schema exists |
| `PluginSurfaceState` | Loading, empty, stale, forbidden, and error presentation | Business-specific recovery rules |

Route components target at most 500 lines and focused components at most 300 lines. No route component may exceed the frozen 800-line hard limit.

## Frontend Data Contract

Every control-center response uses a capture envelope:

```typescript
type PluginSnapshot<T> = {
  pluginId: string;
  capturedAt: string;
  staleAfter: string;
  data: T;
};
```

The frontend compares `staleAfter` with the current clock and does not invent health from menu reachability or enabled state. Commands return a durable operation or correlation ID so progress and failures can be followed on Jobs or Diagnostics.

## Interaction Rules

1. A plugin row always opens Overview; it never opens a local drawer.
2. Install is a validate, impact review, confirm, execute, result sequence.
3. Enable, disable, restart, uninstall, migration, retry, and dead-letter actions are permission-aware and duplicate-submit safe.
4. Destructive confirmation names the plugin, current state, data policy, and irreversible consequence.
5. A successful command refreshes only affected fleet/workspace summaries and links to durable evidence.
6. A failed command keeps operator context and exposes the correlation ID.
7. Browser back/forward preserves the active plugin and work surface.
8. Deep links render forbidden, missing-plugin, and unavailable states without redirecting to Inventory.

## Responsive And Accessibility Rules

- At 390px, workspace navigation becomes a labeled Element Plus select or drawer; it is not removed.
- Tables keep stable action access and use controlled horizontal scrolling inside the table boundary.
- Status is conveyed by text plus semantic color, never color alone.
- Every command has a keyboard path, visible focus, accessible name, and pending state.
- Focus moves to the result heading after install or lifecycle completion and returns to the invoking control after cancellation.
- Light/dark and comfortable/compact combinations use the same information hierarchy.

## Implementation Sequence

1. BF3-02 creates routed fleet, workspace shell, Overview, Runtime, Capabilities, Settings, and lifecycle commands, then removes the monolithic route.
2. BF3-03 fills Data and Migrations using authoritative datastore and lifecycle contracts.
3. BF3-04 fills Jobs and Diagnostics and moves developer tooling to `/skoll/plugin-development/*`.
4. BF3-05 completes theme, density, bilingual, accessibility, and responsive matrices.
5. BF3-06 freezes visual, bundle, interaction, and memory budgets.

## Acceptance Trace

| Required concern | One hierarchy | Owner identified | State model | Interaction reviewed |
| --- | --- | --- | --- | --- |
| Install | Fleet Install and Overview | BF3-02 | Yes | Yes |
| Runtime health | Runtime | BF3-02 | Yes | Yes |
| Capabilities | Capabilities | BF3-02 | Yes | Yes |
| Data | Data | BF3-03 | Yes | Yes |
| Migrations | Migrations | BF3-03 | Yes | Yes |
| Jobs | Jobs | BF3-04 | Yes | Yes |
| Audit | Diagnostics | BF3-04 | Yes | Yes |
| Errors | Diagnostics | BF3-04 | Yes | Yes |

BF3-01 passes when this route, ownership, state, component, and interaction contract is the only accepted implementation target for BF3-02 through BF3-06.
