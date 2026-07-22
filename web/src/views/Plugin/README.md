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
| `/skoll/plugin-center/:pluginId/settings` | `pages/Settings.vue` | Schema-driven plugin configuration |

`components/PluginWorkspaceShell.vue` owns the snapshot and lifecycle commands for all workspace routes. Child pages consume the injected workspace from `workspace.ts`; they must not refetch the control snapshot independently.

## Authoritative Data

- Fleet inventory: `GET /v1/plugins`
- Workspace snapshot: `GET /v1/plugins/{id}/control`
- Configuration: `GET|PUT /v1/plugins/{id}/config`
- Lifecycle: `POST /v1/plugins/{id}/enable`, `POST /v1/plugins/{id}/disable`, and `DELETE /v1/plugins/{id}`

The control snapshot includes `capturedAt` and `staleAfter`. A failed health probe is data inside the snapshot, not a reason to hide the rest of the plugin controls.

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

The browser suite runs both 1440px desktop and 390px mobile projects and verifies fleet access, all workspace routes, lifecycle command state, forbidden access, and horizontal overflow.
