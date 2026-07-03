# Skoll Operations Guide

> Scope: configuration, deployment, backup, logs, and common recovery paths for running Skoll.

## 1. Runtime Configuration

Skoll reads configuration in this order:

1. `SKOLL_*` environment variables
2. `SKOLL_CONFIG_FILE`
3. default config file search paths
4. built-in defaults

Common config files:

| File | Use |
|---|---|
| `configs/skoll.yaml` | Default local config. |
| `configs/skoll.mysql-memory.yaml` | MySQL store with memory cache. |
| `configs/skoll.mysql-redis.yaml` | MySQL store with Redis cache. |
| `configs/skoll.postgres-memory.yaml` | PostgreSQL store with memory cache. |
| `configs/skoll.postgres-redis.yaml` | PostgreSQL store with Redis cache. |

Minimal production variables:

| Variable | Required | Notes |
|---|---|---|
| `SKOLL_API_BASE_PREFIX` | recommended | Defaults to `/skoll`. Keep backend and frontend aligned. |
| `SKOLL_STORE_MODE` | yes | `mysql` or `postgres` for persistent environments. |
| `SKOLL_STORE_DSN` | yes | Database DSN. Keep credentials in a secret manager. |
| `SKOLL_SECURITY_JWT_SECRET` | yes | Must be changed from development defaults. |
| `SKOLL_LOG_LEVEL` | recommended | `info` by default; use `debug` only during diagnosis. |
| `SKOLL_LOG_DIR` | recommended | Directory for file logs. |
| `SKOLL_LOG_FILE` | recommended | Use one file such as `skoll.log` for simple operations. |

Example:

```powershell
$env:SKOLL_CONFIG_FILE = "./configs/skoll.mysql-memory.yaml"
$env:SKOLL_SECURITY_JWT_SECRET = "<replace-with-secret>"
$env:SKOLL_LOG_DIR = "log"
$env:SKOLL_LOG_FILE = "skoll.log"
go run ./cmd/skoll
```

## 2. Deployment Options

### Local or VM

```powershell
go build -o skoll.exe ./cmd/skoll
.\skoll.exe
```

Health check:

```powershell
Invoke-RestMethod http://127.0.0.1:18080/skoll/health
```

### Docker

```powershell
docker build -f deploy/docker/Dockerfile -t skoll:latest .
docker run --rm -p 18080:8080 `
  -e SKOLL_STORE_MODE=memory `
  -e SKOLL_SECURITY_JWT_SECRET="<replace-with-secret>" `
  skoll:latest
```

### Docker Compose

```powershell
docker compose -f deploy/compose/docker-compose.yaml up --build
docker compose -f deploy/compose/docker-compose.yaml down
```

### Kubernetes

```powershell
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
kubectl get pods
kubectl logs deploy/skoll
```

Before production use, review image tag, secrets, resource requests/limits, ingress, TLS, and persistent database connectivity.

## 3. Backup and Restore

Skoll stores durable business data in the configured SQL database. File/object storage and plugin package artifacts should be backed up separately if enabled by deployment policy.

### MySQL Backup

```powershell
mysqldump --single-transaction --routines --triggers `
  -u <user> -p skoll > skoll-backup.sql
```

Restore:

```powershell
mysql -u <user> -p skoll < skoll-backup.sql
```

### PostgreSQL Backup

```powershell
pg_dump --format=custom --file=skoll-backup.dump skoll
```

Restore:

```powershell
pg_restore --clean --if-exists --dbname=skoll skoll-backup.dump
```

### Backup Checklist

| Item | Check |
|---|---|
| Database | Backup completes and restore is tested in a non-production environment. |
| Config | Runtime config and secret references are documented. |
| Logs | Log retention policy is known. |
| Plugins | Installed plugin packages, generated artifacts, and external plugin references are recoverable. |
| OpenAPI/schema | Release artifacts include matching API and schema docs. |

## 4. Logs and Audit

Recommended file logging:

```powershell
$env:SKOLL_LOG_DIR = "log"
$env:SKOLL_LOG_FILE = "skoll.log"
$env:SKOLL_LOG_PLUGIN_PER_FILE = "false"
```

Plugin-per-file logging:

```powershell
$env:SKOLL_LOG_DIR = "log"
$env:SKOLL_LOG_FILE = ""
$env:SKOLL_LOG_PLUGIN_PER_FILE = "true"
```

Operational checks:

| Signal | Where to Look |
|---|---|
| Startup failure | Process logs and `SKOLL_CONFIG_FILE` path. |
| Login failure | Auth logs, seeded account state, JWT secret changes. |
| Permission denied | Audit events and role/permission grants. |
| Plugin lifecycle failure | Plugin page logs, plugin manifest, release order state. |
| API errors | Error logs and trace/request IDs where available. |

## 5. Health, Readiness, and API Docs

| Endpoint | Purpose |
|---|---|
| `/skoll/health` | Liveness/readiness health check. |
| `/skoll/docs/swagger` | Swagger UI. |
| `/skoll/docs/openapi.yaml` | OpenAPI YAML. |

If `SKOLL_API_BASE_PREFIX` changes, use the configured prefix in all endpoint paths.

## 6. Common Failure Paths

| Symptom | Likely Cause | Recovery |
|---|---|---|
| Service does not start | Invalid config file path or DSN. | Check `SKOLL_CONFIG_FILE`, `SKOLL_STORE_MODE`, and `SKOLL_STORE_DSN`; restart after fixing. |
| Health endpoint fails | Backend not listening or wrong prefix. | Check process port, `SKOLL_SERVER_ADDRESS`, and `SKOLL_API_BASE_PREFIX`. |
| Login redirects repeatedly | Backend auth failed or frontend cannot reach API. | Check backend health, browser network errors, and seeded admin account. |
| Database migration/store errors | Database unavailable or schema drift. | Verify database connectivity, run migrations, restore from known-good backup if needed. |
| Plugin install fails | Invalid manifest, blocked root, or duplicate plugin. | Validate `plugin.yaml`, confirm allowed plugin root, inspect plugin logs. |
| Audit export is slow | Large query range. | Narrow filters and review audit table indexing before increasing limits. |
| Frontend build fails | Dependency or type error. | Run `cd web; npm run typecheck; npm run build` and fix the first error. |

## 7. Release-Day Checks

```powershell
go test ./...
cd web
npm run typecheck
npm run build
```

Also verify:

- `docs/api/openapi.yaml` matches the embedded OpenAPI document.
- Plugin manifests validate under `plugins/`.
- Backup and restore commands were tested for the target database.
- Current image/tag and config values are recorded.
- Known warnings are documented and not new regressions.
