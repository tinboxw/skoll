# Plugin Control Center

This directory owns the operator-facing plugin fleet and one routed workspace per plugin. Business plugin pages remain under `/skoll/plugins/<plugin-id>` and do not share this namespace.

## Routes

| Route | Owner | Purpose |
| --- | --- | --- |
| `/skoll/plugin-center` | `index.vue` | Search and inspect the installed plugin fleet |
| `/skoll/plugin-center/install` | `pages/Install.vue` | Validate, review, and install a package |
| `/skoll/plugin-center/marketplace` | `pages/Marketplace.vue` | Select a local package for installation |
| `/skoll/plugin-center/:pluginId/overview` | `pages/Overview.vue` | Identity, summary, and blocking issues |
| `/skoll/plugin-center/:pluginId/runtime` | `pages/Runtime.vue` | Runtime health and observation freshness |
| `/skoll/plugin-center/:pluginId/capabilities` | `pages/Capabilities.vue` | Routes, permissions, services, dependencies, and extensions |
| `/skoll/plugin-center/:pluginId/data` | `pages/Data.vue` | Host-managed schemas, physical tables, storage size, and data policy |
| `/skoll/plugin-center/:pluginId/migrations` | `pages/Migrations.vue` | Durable migration ledger, pending steps, policy gates, and controlled rollback |
| `/skoll/plugin-center/:pluginId/jobs` | `pages/Jobs.vue` | Namespaced job state, attempts, dead letters, and controlled retry |
| `/skoll/plugin-center/:pluginId/audit` | `pages/Audit.vue` | Host records and HTTP audit events with correlation navigation |
| `/skoll/plugin-center/:pluginId/diagnostics` | `pages/Diagnostics.vue` | Process, route, job, audit, and error correlation |
| `/skoll/plugin-center/:pluginId/settings` | `pages/Settings.vue` | Schema-driven plugin configuration |

`components/PluginWorkspaceShell.vue` owns the runtime/capability snapshot and lifecycle commands for all workspace routes. Child pages consume the injected workspace from `workspace.ts`; data and migration pages use `data-control.ts`, while jobs, audit, and diagnostics pages use `diagnostics.ts` to read their separate authoritative contracts.

## Authoritative Data

- Fleet inventory: `GET /v1/plugins`
- Workspace snapshot: `GET /v1/plugins/{id}/control`
- Data lifecycle snapshot: `GET /v1/plugins/{id}/data-control`
- Migration rollback: `POST /v1/plugins/{id}/migrations/rollback`
- Correlated diagnostics: `GET /v1/plugins/{id}/diagnostics`
- Dead-letter retry: `POST /v1/plugins/{id}/jobs/{jobId}/retry`
- Configuration: `GET|PUT /v1/plugins/{id}/config`
- Lifecycle: `POST /v1/plugins/{id}/enable`, `POST /v1/plugins/{id}/disable`, and `DELETE /v1/plugins/{id}`

The control snapshot includes `capturedAt` and `staleAfter`. A failed health probe is data inside the snapshot, not a reason to hide the rest of the plugin controls. The data lifecycle snapshot comes from the host schema registry and migration ledger; the frontend does not infer migration state. Diagnostics aggregate persistent job and audit stores with current health observations. Dead-letter retry creates a new job and preserves the failed source record. Rollback and retry require exact confirmations, `plugin.manage`, and the `super_admin` role.

## Verification

```powershell
cd web
npm run typecheck
npm run test:components
$env:SKOLL_E2E_BASE_URL = "http://127.0.0.1:5173"
npm run test:plugin-center
npm run build
npm run check:bundle
```

The browser suite runs both 1440px desktop and 390px mobile projects and verifies fleet access, all workspace routes, lifecycle, migration and dead-letter action state, correlation rendering, forbidden access, and horizontal overflow.
