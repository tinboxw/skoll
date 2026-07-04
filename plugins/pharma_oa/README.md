# Pharma OA Plugin

`pharma_oa` is the industry plugin entry for the Pharma OA roadmap. The current plugin package declares the installable app, menu, permissions, API route contracts, config schema, and the employee management entry.

## Contract

- Plugin ID: `pharma_oa`
- App ID: `pharma_oa`
- Frontend entry: `/skoll/pharma-oa/employees`
- Menu key: `plugin.pharma_oa`
- Required menu permission: `pharma_oa.menu.read`
- Employee route contracts:
  - `GET /v1/plugins/pharma_oa/api/employees`
  - `POST /v1/plugins/pharma_oa/api/employees`
  - `PUT /v1/plugins/pharma_oa/api/employees`
  - `POST /v1/plugins/pharma_oa/api/employees/leave`
  - `GET /v1/plugins/pharma_oa/api/employees/qualification-reminders`
- Demo seed route contracts:
  - `GET /v1/plugins/pharma_oa/api/demo-seed/status`
  - `POST /v1/plugins/pharma_oa/api/demo-seed/apply`

The current employee module uses the core `/v1/pharma-oa/employees` API and integrated route `/skoll/pharma-oa/employees`. Product, supplier, customer, warehouse, and import/export modules are covered by later F8 Work Items.

## Lifecycle Smoke

The plugin is validated by `go test ./internal/plugin/...`. The dedicated test installs, enables, disables, checks permission/menu catalog effects, route extension registration, catalog audit events, and duplicate-install failure handling.
