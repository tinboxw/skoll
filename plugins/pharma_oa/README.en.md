# Pharma OA Industry Plugin

English | [简体中文](README.md)

`pharma_oa` is Skoll's pharmaceutical industry sample plugin. Its installable contract brings master data, approvals, purchase/sales/inventory, compliance, CRM, analytics, and an end-to-end demo together while reusing Skoll permissions, audit, workflow, notification, file, and frontend host capabilities.

> Current position: an executable, testable, extensible industry sample, not a production GSP/GMP system with regulatory validation. Read the [Release, Regulatory, and Support Boundaries](../../docs/user/release-boundaries.en.md) before adoption.

## Quick Start

### 1. Prepare the environment

Follow the [Skoll Quick Start](../../docs/quick-start.md) to install Go 1.24, Node.js 20, and project dependencies. Run the remaining commands from the repository root.

### 2. Start Skoll

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart
```

At startup, the backend scans `plugins/*/plugin.yaml`, then installs and enables this plugin automatically. Verify:

- Frontend: `http://127.0.0.1:5173/skoll`
- Plugin manager: sign in and confirm Pharma OA is enabled
- Pharma OA entry: `http://127.0.0.1:5173/skoll/pharma-oa/employees`
- Business dashboard: `http://127.0.0.1:5173/skoll/pharma-oa/dashboard`

The development account is `admin` with password `Admin@123456`. Change it immediately in a persistent environment.

### 3. Initialize demo data

```powershell
$base = "http://127.0.0.1:8080/skoll"
$login = Invoke-RestMethod -Method Post -Uri "$base/v1/auth/login" `
  -ContentType "application/json" `
  -Body (@{ account = "admin"; password = "Admin@123456" } | ConvertTo-Json)
$headers = @{ Authorization = "Bearer $($login.data.token)" }

Invoke-RestMethod -Method Post `
  -Uri "$base/v1/pharma-oa/demo-seed/apply" `
  -Headers $headers
```

One apply creates an employee and qualification, product, qualified supplier and customer, cold-chain warehouse, purchase approval, inbound, sales outbound, and customer follow-up. Reapplying in the same process reuses the scenario without duplicating inventory ledger entries.

### 4. Run end-to-end acceptance

```powershell
.\scripts\smoke-pharma-oa-e2e.ps1 -Locale en-US
```

The default script output is Chinese. The smoke covers onboarding, purchase approval, inbound, sales outbound, qualification alerts, customer follow-up, inventory evidence, and seed idempotency.

## Plugin Lifecycle

Repository development should use startup discovery. Run the independent lifecycle smoke when validating the package:

```powershell
.\scripts\smoke-pharma-oa-plugin.ps1 -Locale en-US
```

It validates preflight, install, duplicate-install failure, enable, disable, permission and menu catalogs, route registration, audit, and failure states. A running instance also exposes enable and disable actions in the plugin manager.

When the directory is added after startup, call `POST /skoll/v1/plugins/preflight`, followed by `POST /skoll/v1/plugins/install` and `POST /skoll/v1/plugins/pharma_oa/enable`. Use `plugins/pharma_oa` as the path and do not reinstall an instance already discovered at startup.

## Feature Scope

| Domain | Implemented capability | Main entry |
| --- | --- | --- |
| Master data | Employees, products, suppliers, customers, warehouses, qualifications, import/export | `/skoll/pharma-oa/employees`, `/customers`, and `/v1/pharma-oa/*` |
| Workflow and purchase | Purchase requests, approve/reject, purchase orders, inbound | `/skoll/pharma-oa/purchase-inbounds` and purchase APIs |
| Sales and inventory | Sales orders, batch outbound, stocktakes, transfers, ledger, alerts | `/skoll/pharma-oa/sales` and inventory APIs |
| Collaboration and compliance | Announcements, contracts, qualifications, complaints, recalls, cold-chain, compliance dashboard | `/skoll/pharma-oa/announcements`, `/contracts`, `/qualifications`, `/quality-complaints`, `/drug-recalls`, `/cold-chain`, `/compliance-dashboard` |
| CRM and finance collaboration | Follow-ups, opportunities, payment plans, invoices, overdue reminders | `/skoll/pharma-oa/customer-follow-ups`, `/sales-opportunities`, `/payment-invoices` |
| Analytics and demo | Metrics, responsive dashboard, asynchronous CSV reports, full demo data | `/skoll/pharma-oa/dashboard`, demo-seed and report APIs |

The [OpenAPI contract](../../docs/api/openapi.yaml) is canonical. Host pages call `/v1/pharma-oa/*`; the `/v1/plugins/pharma_oa/api/*` entries in `plugin.yaml` declare plugin catalog, permission, and extension routes. These declared routes are the current contract.

## Regulatory And Data Boundary

- Compliance, qualification, recall, cold-chain, and audit are sample capability labels. They do not assert conformity with any jurisdiction, validation practice, electronic-record rule, or quality system.
- Built-in `DEMO-*` people, organizations, medicines, qualifications, and transactions are fictional. Do not replace or combine them with real personal data, patient data, production credentials, or legal records while continuing to treat the seed as a demo.
- Production adopters own regulatory analysis, system validation, data classification, least privilege, retention/deletion, backup/recovery, incident response, and go-live approval. Maintainers provide no regulatory certification or production SLA.
- The [English boundary](../../docs/user/release-boundaries.en.md) and [Chinese-default counterpart](../../docs/user/release-boundaries.md) are authoritative for license, security, data, and support scope.

## Runtime Boundaries

- `plugin.yaml` is the single install contract for menus, config, permissions, routes, audit actions, and events.
- The current business services are wired into the Skoll host from `internal/domain/pharmaoa`, `internal/service/pharmaoa`, and `internal/handler/http/v1/pharmaoa`.
- Business pages are host-integrated. Routes live in `web/src/router/index.ts` and the typed client in `web/src/pharma-oa/api.ts`. `static/` is a plugin page/lifecycle fixture, not the full business console.
- `mysql` / `postgres` modes persist Pharma OA business records in SQL and rediscover the applied demo seed after restart. `memory` remains a process-local development fixture and loses data on restart. Report files use the existing private object store.
- Disabling the plugin removes its catalog menu, permissions, and extensions, but does not hot-unload host services already wired at startup. It is not a process sandbox.
- Only the current API, data, plugin, and page-route contracts are supported.

## Localization

- The plugin declares `zh-CN` and `en-US`; the host defaults to `zh-CN` when no preference is stored.
- Manifest names, menu labels, and config labels provide both Chinese and English values.
- [README.md](README.md) is the default Chinese guide; this file is its English counterpart.
- New frontend copy must use host `useI18n()` and `web/src/i18n/index.ts`; do not add single-language hard-coded component text.
- Visible main-flow copy has passed Chinese-default and English-switch acceptance. New copy must update both locales and retain complete state and responsive coverage.

## Extend a Domain

1. Define domain objects and invariants in `internal/domain/pharmaoa`, then implement use cases in `internal/service/pharmaoa`.
2. Add JWT identity, permissions, error states, and audit coverage in `internal/handler/http/v1/pharmaoa`, and update both OpenAPI copies.
3. Declare matching permissions, routes, audit actions, menus, or events in `plugin.yaml`. Use `pharma_oa.<resource>.<action>` keys.
4. Add a typed client in `web/src/pharma-oa/api.ts`, a host page under `web/src/views/Pharma*`, and wire its route/menu.
5. Add Chinese and English i18n entries together, with Chinese as the default. Cover loading, empty, error, no-permission, saving/destructive, and responsive states.
6. Add service, HTTP, permission, audit, manifest, frontend, and end-to-end tests. Any persistent table also requires migration/seed impact and multi-database validation in the same Work Item.

See the general [Plugin Development Guide](../../docs/development/plugin-guide.md) for platform contracts and tooling.

## Verification

```powershell
go test ./internal/plugin -run TestPharmaOA -count=1
.\scripts\smoke-pharma-oa-plugin.ps1 -Locale en-US
.\scripts\smoke-pharma-oa-e2e.ps1 -Locale en-US
go test ./...
cd web
npm run typecheck
npm run build
```

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Pharma OA is absent from the plugin list | Start from the repository root, verify `plugins/pharma_oa/plugin.yaml`, and inspect backend startup logs |
| Menu is hidden | Enable the plugin and grant `pharma_oa.menu.read`; `super_admin` can verify directly |
| API returns 401/403 | Refresh the JWT and verify the corresponding `pharma_oa.*` permission |
| Demo data disappeared | Check whether `memory` mode is active; for persistent modes inspect the DSN, migrations, and startup logs, and verify a backup before recovery |
| Duplicate install fails | Startup discovery already installed it; use list, enable, or disable instead |
| Business API is not found | Use `/skoll/v1/pharma-oa/*` and check the OpenAPI contract |
