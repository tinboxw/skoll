# M7 Backend Performance Baseline

> Updated: 2026-07-03
> Scope: M7-03-01 backend API baseline script and result capture.

## Script

- Tool: k6
- Script: `tests/performance/skoll-backend-baseline.k6.mjs`
- Default target: `http://127.0.0.1:18080/skoll`
- Default load: 5 virtual users for 1 minute

## Covered Flows

| Flow | Endpoint | Metric |
|---|---|---|
| Login | `POST /v1/auth/login` | `skoll_login_latency` |
| User list | `GET /v1/users?offset=0&limit=20` | `skoll_user_list_latency` |
| Role list | `GET /v1/roles?offset=0&limit=20` | `skoll_role_list_latency` |
| Plugin list | `GET /v1/plugins?offset=0&limit=20` | `skoll_plugin_list_latency` |
| Audit query | `GET /v1/audit?offset=0&limit=20` | `skoll_audit_query_latency` |

## Run Command

```powershell
$env:SKOLL_BASE_URL = "http://127.0.0.1:18080/skoll"
$env:SKOLL_ADMIN_USER = "admin"
$env:SKOLL_ADMIN_PASSWORD = "admin123"
$env:SKOLL_PERF_VUS = "5"
$env:SKOLL_PERF_DURATION = "1m"
k6 run tests/performance/skoll-backend-baseline.k6.mjs
```

## Thresholds

| Metric | P95 | P99 |
|---|---:|---:|
| Overall HTTP request duration | < 1000 ms | < 2000 ms |
| Login | < 1000 ms | < 2000 ms |
| User list | < 800 ms | < 1500 ms |
| Role list | < 800 ms | < 1500 ms |
| Plugin list | < 800 ms | < 1500 ms |
| Audit query | < 1200 ms | < 2500 ms |

The k6 summary also reports throughput through `http_reqs` and iteration rate. Record QPS, P95, and P99 from the k6 summary for each baseline run.

## Result Capture Template

| Date | Commit | Base URL | VUs | Duration | HTTP QPS | Login P95/P99 | User List P95/P99 | Role List P95/P99 | Plugin List P95/P99 | Audit Query P95/P99 | Result |
|---|---|---|---:|---|---:|---|---|---|---|---|---|
| TBD | TBD | TBD | 5 | 1m | TBD | TBD | TBD | TBD | TBD | TBD | Pending local service |

## Notes

- The script requires a running Skoll backend and valid admin credentials.
- The login token is reused for list/query flows when the API returns `data.token` or `data.accessToken`.
- Keep this as a baseline script first; do not tune query/index/cache behavior until a run identifies a bottleneck.
- If k6 is unavailable locally, validate syntax with `node --check tests/performance/skoll-backend-baseline.k6.mjs` and run k6 in CI or a developer machine with the backend stack available.
