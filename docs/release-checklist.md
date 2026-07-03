# Skoll Release Checklist

> Scope: release candidate preparation and final verification for the current Skoll contracts.

## 1. Release Identity

| Check | Command or Evidence | Result |
|---|---|---|
| Release version is selected. | Record tag/version, for example `v0.x.y`. | [ ] |
| Changelog or release notes are drafted. | Link issue, PR, or release note draft. | [ ] |
| Commit range is known. | `git log --oneline <previous>..HEAD` | [ ] |
| No legacy compatibility work slipped in. | Review PRs and final inspection. | [ ] |

## 2. Quality Gates

| Gate | Command | Result |
|---|---|---|
| Go format | CI `gofmt` check or local `gofmt -l` | [ ] |
| Go tests | `go test ./...` | [ ] |
| Go coverage | `go test ./... -covermode=atomic -coverprofile=artifacts/coverage.out` | [ ] |
| OpenAPI sync | `go test ./internal/handler/http -run TestOpenAPIContractFilesStayInSync -count=1` | [ ] |
| Plugin manifests | `go test ./internal/plugin -run TestValidatePluginManifestsUnderPluginsDir -count=1` | [ ] |
| Frontend typecheck | `cd web && npm run typecheck` | [ ] |
| Frontend build | `cd web && npm run build` | [ ] |
| Browser smoke minimum | `cd web && node scripts/fe5-browser-smoke-minimum.mjs --json` | [ ] |

## 3. API and Schema Artifacts

| Artifact | Check | Result |
|---|---|---|
| OpenAPI docs | `docs/api/openapi.yaml` matches `internal/handler/http/openapi.yaml`. | [ ] |
| Plugin manifest schema | `docs/schemas/plugin-manifest.schema.json` is current. | [ ] |
| Marketplace schema | `docs/schemas/plugin-marketplace-index.schema.json` is current. | [ ] |
| Generated examples | `examples/demo_product` spec, generated file matrix, backend/frontend acceptance records are current. | [ ] |
| Demo plugin | `plugins/demo` manifest, signature asset coverage, and lifecycle acceptance are current. | [ ] |

## 4. Deployment and Configuration

| Area | Check | Result |
|---|---|---|
| Docker image | `docker build -f deploy/docker/Dockerfile -t skoll:<version> .` | [ ] |
| Docker Compose | `docker compose -f deploy/compose/docker-compose.yaml config` | [ ] |
| Kubernetes manifests | Review `deploy/k8s/deployment.yaml` and `deploy/k8s/service.yaml`. | [ ] |
| Runtime config | Required `SKOLL_*` values are recorded for target environment. | [ ] |
| Secrets | `SKOLL_SECURITY_JWT_SECRET`, database credentials, and plugin secrets are not committed. | [ ] |
| Health endpoint | `/skoll/health` returns ok in target environment. | [ ] |
| Logs | `SKOLL_LOG_DIR`, `SKOLL_LOG_FILE`, and retention policy are documented. | [ ] |

## 5. Database, Files, and Plugin Assets

| Area | Check | Result |
|---|---|---|
| Migration plan | SQL migrations and store behavior are reviewed for the release. | [ ] |
| Backup before release | MySQL or PostgreSQL backup command from `docs/user/operations.md` has been executed. | [ ] |
| Restore rehearsal | Backup restore was tested in a non-production environment. | [ ] |
| File/object storage | File/object store data and plugin package artifacts are included in backup policy. | [ ] |
| Plugin packages | Local marketplace package hashes and signature/risk metadata are reviewed. | [ ] |

## 6. Rollback Plan

| Area | Check | Result |
|---|---|---|
| App rollback | Previous image/tag and config are recorded. | [ ] |
| Database rollback | Restore point and migration rollback limits are documented. | [ ] |
| Plugin rollback | Plugin release order, rollback plan, and catalog restoration behavior are validated. | [ ] |
| Communication | Release owner and rollback decision owner are named. | [ ] |

## 7. Post-Release Verification

| Area | Check | Result |
|---|---|---|
| Login | Admin login succeeds with expected role and permissions. | [ ] |
| Core pages | Dashboard, Users, Roles, Permissions/Menu, Plugins, Audit, Settings open. | [ ] |
| Plugin lifecycle | Demo plugin install/enable/disable path remains healthy. | [ ] |
| Audit | Login failure, permission denial, plugin lifecycle, and export events are visible. | [ ] |
| Performance | Backend and frontend performance baselines are not newly regressed. | [ ] |
| Known warnings | Sass/Rollup frontend warnings match the documented baseline. | [ ] |

## 8. Final Record

Before publishing, update:

- `docs/refactor/work_items.md`
- `docs/refactor/task_board.md`
- `docs/refactor/acceptance_log.md`
- release notes or GitHub release draft

Record the final commands, results, commit SHA, release tag, and any accepted residual risk.
