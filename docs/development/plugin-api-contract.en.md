# Plugin API and OpenAPI Contract

> Default documentation: [简体中文](plugin-api-contract.md). This page is the English reference.

## Scope

This document explains the current business-plugin API declaration and how Skoll aggregates enabled plugin routes into `/skoll/docs/openapi.yaml`. `plugin.yaml` is the sole supported declaration format.

## Manifest Declaration

Every business route must live under `/v1/plugins/{pluginId}/api/` and declare an assignable permission key. Add a three-segment `audit_action` when successful and failed execution must be audited.

```yaml
id: pharma_oa
version: 1.0.0
service_base_url: http://127.0.0.1:18090
service_health_url: http://127.0.0.1:18090/health
api:
  routes:
    - method: GET
      path: /v1/plugins/pharma_oa/api/employees
      summary: List Pharma OA employees
      permission: pharma_oa.employee.read
      audit_action: pharma_oa.employee.read
```

Install preflight validates the method, path, permission, audit action, and source. Invalid declarations do not enter the route-permission registry or aggregated OpenAPI document.

`service_base_url` is the managed backend loopback-HTTP base URL and must be declared together with a same-origin `service_health_url`. Skoll executes only method/path pairs declared in `api.routes` and appends the declared path to this base. For example, the request above is sent to `http://127.0.0.1:18090/v1/plugins/pharma_oa/api/employees`. A path prefix in the base URL is preserved. See the [Managed Plugin Process Contract](plugin-process-contract.en.md) for entry, address, and environment rules.

## Runtime Behavior

| Plugin state | Business route execution | Aggregated OpenAPI |
| --- | --- | --- |
| Installed | Not executable | Hidden |
| Enabled | Enforced by JWT and declared permission | Included |
| Disabled | Denied | Removed |
| Uninstalled | Not executable | Removed |

Plugin pages and static assets use a separate bounded public policy. `/v1/plugins/{pluginId}/api/*` is always a protected business API. A missing route resolver, permission declaration, or permission checker fails closed.

After authentication and authorization, Skoll forwards the request method, declared path, query, body, and ordinary request headers to the plugin backend. The backend status, response headers, and body are returned to the caller. An unreachable backend or missing execution configuration never produces placeholder success.

Enable, disable, uninstall, and metadata reload share one lifecycle gate with route execution. When enable returns, declared routes, extension snapshots, and permission snapshots are published together without restarting the host. Disable and uninstall first drain in-flight requests; after they return, new business requests, extension lookups, and permission resolution can no longer access that plugin. Re-enabling republishes the current declaration through the same Router instance.

Plugins that declare an external service are also owned by the service supervisor. Enable must pass initial readiness within five seconds or the plugin remains closed. Once ready, the existing health contract monitors the service every five seconds and moves an unhealthy or unexpectedly exited service to Failed. Disable, uninstall, and host shutdown close business traffic first and then allow five seconds for graceful stop; a timeout records a stable failure code and forces resource release. Every transition is audited without service URLs or underlying network errors. A plugin without `service_base_url` is `not_applicable` and receives no service handle.

Skoll probes `service_health_url` with an unauthenticated GET request. A 2xx response is healthy; redirects, non-2xx responses, timeouts, and connection errors are unhealthy. Unhealthy plugins receive no business traffic. Business requests reuse health results for at most five seconds; `GET /ready` and `GET /v1/plugins/{pluginId}/health` force a fresh probe. Reports contain only plugin ID, stable status code, timestamp, latency, and HTTP status; they never expose URLs, tokens, or underlying error text.

| HTTP | Code | Meaning |
| --- | --- | --- |
| 502 | `plugin_route_unavailable` | The host has no available plugin route executor |
| 502 | `plugin_backend_unavailable` | The plugin backend connection or execution failed |
| 503 | `plugin_backend_not_configured` | A business route is declared without a valid backend URL |
| 503 | `plugin_not_enabled` | The plugin is not enabled |
| 503 | `plugin_unhealthy` | The health probe failed and business traffic was blocked |

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
