# Skoll

Skoll 北欧・巨狼｜Go high-performance, high-concurrency framework.

## Project Status

Skoll has completed M4 release readiness, started M5 initial generic module scaffolds (user/role/menu/audit), and delivered M6/M7 foundations, full M8 (file/job/generator), full M9 (plugin lifecycle/packaging/ecosystem docs), full M10 (dashboard aggregation and UI bootstrap contract), full M11 (dashboard auth/session capabilities), full M12 (dashboard JWT session bridge capabilities), and M13-step2 (JWT claims normalization policy) baseline.

## Structure

```text
.
├── cmd/skoll/                 # Executable entrypoint
├── internal/app/              # HTTP transport and bootstrap orchestration
├── internal/domain/           # Core domain models
├── internal/service/          # Core service layer
├── pkg/version/               # Public version utilities
├── docs/planning/             # Planning documents
└── docs/milestones/           # Milestone records
```

## Quick Start

### Run the service

```bash
go run ./cmd/skoll -addr :8080
# Optional: enable minimal go-admin bootstrap
go run ./cmd/skoll -addr :8080 -go-admin-enabled -go-admin-mode dev
# Optional: switch to not_ready and drain for 3 seconds before shutdown
go run ./cmd/skoll -addr :8080 -drain-time 3s
# Optional: provide default drain/shutdown values via environment variables
SKOLL_DRAIN_TIME=3s SKOLL_SHUTDOWN_TIMEOUT=12s go run ./cmd/skoll -addr :8080
# Optional: enable pluggable admin auth skeleton for admin routes (current implementation: static-token)
SKOLL_ADMIN_AUTH_MODE=static-token SKOLL_ADMIN_AUTH_TOKEN=secret go run ./cmd/skoll -addr :8080 -go-admin-enabled
# Optional: enable HMAC signature auth mode
SKOLL_ADMIN_AUTH_MODE=hmac-sha256 SKOLL_ADMIN_AUTH_HMAC_SECRET=secret go run ./cmd/skoll -addr :8080 -go-admin-enabled
# Optional: if static-token must be used in prod mode, explicitly allow it (blocked by default)
SKOLL_ADMIN_AUTH_MODE=static-token SKOLL_ADMIN_AUTH_TOKEN=secret SKOLL_ADMIN_AUTH_ALLOW_STATIC_TOKEN_IN_PROD=true go run ./cmd/skoll -addr :8080 -go-admin-enabled -go-admin-mode prod
# Optional: switch nonce replay protection to Redis shared store for multi-instance deployments
SKOLL_ADMIN_AUTH_MODE=hmac-sha256 SKOLL_ADMIN_AUTH_HMAC_SECRET=secret SKOLL_ADMIN_AUTH_NONCE_STORE=redis SKOLL_ADMIN_AUTH_NONCE_REDIS_ADDR=127.0.0.1:6379 go run ./cmd/skoll -addr :8080 -go-admin-enabled
# Recommended: use the production environment template as deployment baseline
# Template file: .env.production.example
# Constraints: shutdown-timeout must be > 0, drain-time must be in [0s, 30s]
# Behavior: first termination signal starts graceful sequence; second signal can interrupt drain wait and enter shutdown immediately
# Behavior: in go-admin prod mode, static-token is denied by default unless admin-auth-allow-static-token-in-prod is explicitly enabled
# Observability: runtime config snapshot is printed on startup
```

### Probe endpoints

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/metrics
# If admin auth is enabled, /metrics additionally includes skoll_admin_auth_verifications_total and skoll_admin_auth_failures_total
# M6: admin module API baseline (minimal user/role/menu/audit read/write)
curl -X POST http://localhost:8080/admin/v1/users -H "Content-Type: application/json" -d '{"name":"alice","email":"alice@example.com"}'
curl http://localhost:8080/admin/v1/users
curl http://localhost:8080/admin/v1/users/1
curl -X POST http://localhost:8080/admin/v1/roles -H "Content-Type: application/json" -d '{"name":"ops","permissions":["user.read"]}'
curl -X PUT http://localhost:8080/admin/v1/roles/1/menus -H "Content-Type: application/json" -d '{"menu_ids":[1,2]}'
curl http://localhost:8080/admin/v1/roles/1/menus
curl -X PUT http://localhost:8080/admin/v1/roles/1/apis -H "Content-Type: application/json" -d '{"apis":["GET:/admin/v1/users","POST:/admin/v1/users"]}'
curl http://localhost:8080/admin/v1/roles/1/apis
curl http://localhost:8080/admin/v1/apis
curl -X POST http://localhost:8080/admin/v1/menus -H "Content-Type: application/json" -d '{"title":"Dashboard","path":"/dashboard","order":1}'
curl -X POST http://localhost:8080/admin/v1/audit-logs -H "Content-Type: application/json" -d '{"actor":"system","action":"create","target":"user"}'
curl http://localhost:8080/admin/v1/audit-logs?limit=20
curl "http://localhost:8080/admin/v1/audit-logs?page=1&size=20&actor=system&action=create&q=user"
curl -X POST http://localhost:8080/admin/v1/configs -H "Content-Type: application/json" -d '{"key":"system.theme","value":"aurora","description":"ui theme"}'
curl http://localhost:8080/admin/v1/configs
curl http://localhost:8080/admin/v1/configs/system.theme
curl -X POST http://localhost:8080/admin/v1/dictionaries -H "Content-Type: application/json" -d '{"type":"status","label":"Enabled","value":"1","sort":10}'
curl http://localhost:8080/admin/v1/dictionaries
curl http://localhost:8080/admin/v1/dictionaries?type=status
curl -X POST http://localhost:8080/admin/v1/files -F "file=@./README.md"
curl http://localhost:8080/admin/v1/files
curl http://localhost:8080/admin/v1/files/1
curl -L http://localhost:8080/admin/v1/files/1/download -o downloaded.bin
curl -X POST http://localhost:8080/admin/v1/jobs -H "Content-Type: application/json" -d '{"name":"daily-sync","schedule":"0 0 * * *"}'
curl http://localhost:8080/admin/v1/jobs
curl -X POST http://localhost:8080/admin/v1/jobs/1/run
curl http://localhost:8080/admin/v1/jobs/1/history?limit=20
curl -X POST http://localhost:8080/admin/v1/generator/modules -H "Content-Type: application/json" -d '{"module":"billing"}'
curl -X POST http://localhost:8080/admin/v1/plugins/manifests -H "Content-Type: application/json" -d '{"name":"audit-ext","version":"1.0.0","hooks":["on_boot"]}'
curl http://localhost:8080/admin/v1/plugins
curl http://localhost:8080/admin/v1/plugins/audit-ext
curl -X POST http://localhost:8080/admin/v1/plugins/packages/install -H "Content-Type: application/json" -d '{"name":"audit-ext","version":"1.1.0","package_url":"https://example.com/plugins/audit-ext-1.1.0.tgz","package_hash":"sha256:abcd","hooks":["on_boot"]}'
curl -X POST http://localhost:8080/admin/v1/plugins/audit-ext/disable
curl -X POST http://localhost:8080/admin/v1/plugins/audit-ext/enable
curl -X POST http://localhost:8080/admin/v1/plugins/audit-ext/version-check -H "Content-Type: application/json" -d '{"latest_version":"1.2.0"}'
curl http://localhost:8080/admin/v1/system/status
curl http://localhost:8080/admin/v1/system/runtime-metrics
curl http://localhost:8080/admin/v1/system/node-health
curl http://localhost:8080/admin/v1/system/dashboard
curl -H "X-Admin-Token: secret" -H "X-Admin-Role-ID: 1" http://localhost:8080/admin/v1/system/dashboard
# Available only when minimal go-admin integration is enabled
curl http://localhost:8080/admin/ping
# If admin auth skeleton is enabled, include the auth header
curl -H "X-Admin-Token: secret" http://localhost:8080/admin/ping
# If hmac-sha256 is enabled, include X-Admin-Timestamp (Unix seconds), X-Admin-Nonce, and X-Admin-Signature (HMAC-SHA256 hex)
# Optional: X-Admin-Body-SHA256 (sha256 hex of request body)
```

## Quality Gates and Milestone Rules

1. Milestone labels must use `M0/M1/M2/M3-short-topic` (example: `M1-auth-rbac`).
2. Every PR must include performance comparison data (baseline, current, delta, command).
3. Baseline validation command: `go test ./...`.
4. For concurrency-related changes, additionally run: `go test -race ./...`.
5. Performance snapshot command: `go test -bench=. -benchmem ./...`.
6. CI workflow is defined in `.github/workflows/ci.yml` and enforces checks after module initialization.

## Local Development Commands

```bash
go fmt ./...
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

## Documentation

- Implementation roadmap: `docs/planning/IMPLEMENTATION_ROADMAP.md`
- Feature parity plan: `docs/planning/FEATURE_PARITY_PLAN.md`
- Production deployment env template: `docs/planning/PRODUCTION_ENV_TEMPLATE.md`
- Benchmark toolchain policy: `docs/planning/BENCHMARK_TOOLCHAIN_POLICY.md`
- Admin HMAC signature contract: `docs/planning/ADMIN_AUTH_SIGNATURE_CONTRACT.md`
- Admin auth security runbook: `docs/planning/ADMIN_AUTH_SECURITY_RUNBOOK.md`
- Open source release note (M4): `docs/releases/M4_OPEN_SOURCE_RELEASE_NOTE.md`
- Versioning policy: `docs/community/VERSIONING_POLICY.md`
- Changelog process: `docs/community/CHANGELOG_PROCESS.md`
- Extension developer guide: `docs/community/EXTENSION_DEVELOPER_GUIDE.md`
- Extension compatibility policy: `docs/community/EXTENSION_COMPATIBILITY_POLICY.md`
- Dashboard UI bootstrap contract: `docs/community/DASHBOARD_UI_BOOTSTRAP_CONTRACT.md`
- Dashboard auth/session policy: `docs/community/DASHBOARD_AUTH_SESSION_POLICY.md`
- Dashboard JWT middleware bridge contract: `docs/community/DASHBOARD_JWT_MIDDLEWARE_BRIDGE_CONTRACT.md`
- Dashboard JWT claims normalization policy: `docs/community/DASHBOARD_JWT_CLAIMS_NORMALIZATION_POLICY.md`
- Contribution guide: `CONTRIBUTING.md`
- M0 baseline record: `docs/milestones/M0-project-baseline.md`
- M1 core-domain record: `docs/milestones/M1-core-domain.md`
- M2 concurrency/performance record: `docs/milestones/M2-concurrency-and-performance.md`
- M3 observability/hardening record: `docs/milestones/M3-observability-and-hardening.md`
- M3 go-admin minimal integration record: `docs/milestones/M3-go-admin-minimal-integration.md`
- M3 go-admin probe hook record: `docs/milestones/M3-go-admin-probe-hook.md`
- M3 graceful drain readiness record: `docs/milestones/M3-graceful-drain-readiness.md`
- M3 drain config guard record: `docs/milestones/M3-drain-config-guard.md`
- M3 runtime env overrides record: `docs/milestones/M3-runtime-env-overrides.md`
- M3 signal-aware drain and shutdown sequence record: `docs/milestones/M3-shutdown-sequence-and-signal-aware-drain.md`
- M3 metrics route template record: `docs/milestones/M3-metrics-route-template.md`
- M4 admin auth skeleton record: `docs/milestones/M4-admin-auth-skeleton.md`
- M4 admin pluggable auth interface record: `docs/milestones/M4-admin-auth-pluggable-interface.md`
- M4 admin HMAC auth record: `docs/milestones/M4-admin-auth-hmac-sha256.md`
- M4 admin nonce-store interface record: `docs/milestones/M4-admin-auth-nonce-store-interface.md`
- M4 admin shared nonce-store wiring record: `docs/milestones/M4-admin-auth-shared-nonce-store-wiring.md`
- M4 admin auth observability record: `docs/milestones/M4-admin-auth-observability.md`
- M4 admin auth security runbook record: `docs/milestones/M4-admin-auth-security-runbook.md`
- M4 admin prod static-token guard record: `docs/milestones/M4-admin-auth-prod-static-token-guard.md`
- M4 release checklist record: `docs/milestones/M4-release-checklist.md`
- M5 initial module scaffold record: `docs/milestones/M5-initial-module-scaffolds.md`
- M6 admin module API baseline record: `docs/milestones/M6-admin-module-api-baseline.md`
- M6 role binding capability record: `docs/milestones/M6-role-bindings.md`
- M6 API registry and permission assignment record: `docs/milestones/M6-api-registry-and-permission-assignment.md`
- M6 RBAC end-to-end smoke record: `docs/milestones/M6-rbac-e2e-smoke.md`
- M7 storage adapter contract record: `docs/milestones/M7-storage-adapter-contract.md`
- M7 config center and dictionary record: `docs/milestones/M7-config-and-dictionary.md`
- M7 durable audit log query record: `docs/milestones/M7-durable-audit-log.md`
- M8 file service baseline record: `docs/milestones/M8-file-service-baseline.md`
- M8 job scheduler baseline record: `docs/milestones/M8-job-scheduler-baseline.md`
- M8 module generator record: `docs/milestones/M8-module-generator.md`
- M9 plugin manifest lifecycle record: `docs/milestones/M9-plugin-manifest-lifecycle.md`
- M9 extension packaging and version-check record: `docs/milestones/M9-extension-packaging.md`
- M9 ecosystem docs and compatibility strategy record: `docs/milestones/M9-ecosystem-docs.md`
- M10 system status API record: `docs/milestones/M10-system-status-api.md`
- M10 runtime metrics snapshot API record: `docs/milestones/M10-dashboard-runtime-metrics.md`
- M10 node and dependency health summary API record: `docs/milestones/M10-dashboard-node-health.md`
- M10 dashboard aggregation API record: `docs/milestones/M10-dashboard-aggregation.md`
- M10 dashboard UI bootstrap contract record: `docs/milestones/M10-dashboard-ui-bootstrap-contract.md`
- M11 dashboard auth/session alignment record: `docs/milestones/M11-dashboard-auth-session-alignment.md`
- M11 dashboard auth/session policy docs record: `docs/milestones/M11-dashboard-auth-session-policy-docs.md`
- M11 dashboard auth/session observability record: `docs/milestones/M11-dashboard-auth-session-observability.md`
- M11 dashboard auth/session actionability record: `docs/milestones/M11-dashboard-auth-session-actionability.md`
- M12 dashboard JWT session bootstrap record: `docs/milestones/M12-dashboard-jwt-session-bootstrap.md`
- M12 dashboard JWT refresh/expiry hints record: `docs/milestones/M12-dashboard-jwt-session-refresh-hints.md`
- M12 dashboard JWT verification-state hints record: `docs/milestones/M12-dashboard-jwt-session-verification-state.md`
- M12 dashboard JWT middleware bridge record: `docs/milestones/M12-dashboard-jwt-session-middleware-bridge.md`
- M12 dashboard JWT middleware bridge contract docs record: `docs/milestones/M12-dashboard-jwt-session-bridge-contract-docs.md`
- M13 dashboard verified JWT claims adapter alignment record: `docs/milestones/M13-dashboard-jwt-session-verified-claims-adapter.md`
- M13 dashboard JWT claims normalization policy record: `docs/milestones/M13-dashboard-jwt-session-claims-normalization.md`
- Milestone template: `docs/milestones/MILESTONE_LOG_TEMPLATE.md`

## Contribution

1. Fork the repository
2. Create a feature branch
3. Commit code with validation evidence
4. Open a Pull Request
