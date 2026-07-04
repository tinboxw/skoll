# Pharma OA Plugin Skeleton

`pharma_oa` is the industry plugin entry for the Pharma OA roadmap. This Work Item adds the installable skeleton only: manifest, menu declaration, permissions, API route contract, config schema, and a static integrated page.

## Contract

- Plugin ID: `pharma_oa`
- App ID: `pharma_oa`
- Frontend entry: `/skoll/plugins/pharma_oa`
- Menu key: `plugin.pharma_oa`
- Required menu permission: `pharma_oa.menu.read`
- Demo seed route contracts:
  - `GET /v1/plugins/pharma_oa/api/demo-seed/status`
  - `POST /v1/plugins/pharma_oa/api/demo-seed/apply`

The skeleton intentionally does not create pharma master-data tables yet. Employee, product, supplier, customer, warehouse, and import/export modules are covered by later F8 Work Items.

## Lifecycle Smoke

The plugin is validated by `go test ./internal/plugin/...`. The dedicated test installs, enables, disables, checks permission/menu catalog effects, route extension registration, catalog audit events, and duplicate-install failure handling.
