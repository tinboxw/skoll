# Plugin API and OpenAPI Contract

> Default documentation: [简体中文](plugin-api-contract.md). This page is the English reference.

## Scope

This document explains the current business-plugin API declaration and how Skoll aggregates enabled plugin routes into `/skoll/docs/openapi.yaml`. Only the current `plugin.yaml` format is supported; there is no legacy manifest or route compatibility path.

## Manifest Declaration

Every business route must live under `/v1/plugins/{pluginId}/api/` and declare an assignable permission key. Add a three-segment `audit_action` when successful and failed execution must be audited.

```yaml
id: pharma_oa
version: 1.0.0
service_base_url: http://127.0.0.1:18090
api:
  routes:
    - method: GET
      path: /v1/plugins/pharma_oa/api/employees
      summary: List Pharma OA employees
      permission: pharma_oa.employee.read
      audit_action: pharma_oa.employee.read
```

Install preflight validates the method, path, permission, audit action, and source. Invalid declarations do not enter the route-permission registry or aggregated OpenAPI document.

`service_base_url` is the plugin backend HTTP(S) base URL. Skoll executes only method/path pairs declared in `api.routes` and appends the declared path to this base. For example, the request above is sent to `http://127.0.0.1:18090/v1/plugins/pharma_oa/api/employees`. A path prefix in the base URL is preserved.

## Runtime Behavior

| Plugin state | Business route execution | Aggregated OpenAPI |
| --- | --- | --- |
| Installed | Not executable | Hidden |
| Enabled | Enforced by JWT and declared permission | Included |
| Disabled | Denied | Removed |
| Uninstalled | Not executable | Removed |

Plugin pages and static assets use a separate bounded public policy. `/v1/plugins/{pluginId}/api/*` is always a protected business API. A missing route resolver, permission declaration, or permission checker fails closed.

After authentication and authorization, Skoll forwards the request method, declared path, query, body, and ordinary request headers to the plugin backend. The backend status, response headers, and body are returned to the caller. An unreachable backend or missing execution configuration never produces placeholder success.

| HTTP | Code | Meaning |
| --- | --- | --- |
| 502 | `plugin_route_unavailable` | The host has no available plugin route executor |
| 502 | `plugin_backend_unavailable` | The plugin backend connection or execution failed |
| 503 | `plugin_backend_not_configured` | A business route is declared without a valid backend URL |
| 503 | `plugin_not_enabled` | The plugin is not enabled |

## Aggregated Metadata

Each enabled plugin operation includes:

| Field | Meaning |
| --- | --- |
| `security: [{ bearerAuth: [] }]` | A Skoll JWT is required |
| `x-skoll-plugin-id` | Plugin ID |
| `x-skoll-plugin-source` | Route and permission source |
| `x-skoll-permission` | Permission enforced by the server |
| `x-skoll-audit-action` | Optional declared audit action |

The baseline specification defines `components.securitySchemes.bearerAuth` as an HTTP Bearer JWT scheme. Runtime aggregation adds only paths from enabled plugins and does not rewrite the two baseline files on disk.

## Inspect and Verify

```powershell
# Inspect the aggregated specification after starting Skoll
Invoke-WebRequest http://127.0.0.1:8080/skoll/docs/openapi.yaml | Select-Object -ExpandProperty Content

# Verify embedded/public sync, references, and aggregated security metadata
go test ./internal/handler/http -run 'Test(RouterOpenAPI|OpenAPIContractFilesStayInSync|EmbeddedOpenAPIReferencesResolve)' -count=1
```

Use an access token issued by the login endpoint and grant the current user or role the permission named by `x-skoll-permission`. Menu visibility is not a substitute for backend authorization.

## Change Impact

When adding, removing, or changing `api.routes`, synchronize the permission catalog, audit actions, tests, and developer documentation. This contract adds no migration, seed, or frontend page route.
