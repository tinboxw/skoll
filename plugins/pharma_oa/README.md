# Pharma OA Plugin

`pharma_oa` is the industry plugin entry for the Pharma OA roadmap. The current plugin package declares the installable app, menu, permissions, API route contracts, config schema, and the employee/customer/warehouse management entries.

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
- Product route contracts:
  - `GET /v1/plugins/pharma_oa/api/products`
  - `POST /v1/plugins/pharma_oa/api/products`
  - `PUT /v1/plugins/pharma_oa/api/products`
  - `POST /v1/plugins/pharma_oa/api/products/disable`
  - `POST /v1/plugins/pharma_oa/api/products/import`
- Supplier route contracts:
  - `GET /v1/plugins/pharma_oa/api/suppliers`
  - `POST /v1/plugins/pharma_oa/api/suppliers`
  - `PUT /v1/plugins/pharma_oa/api/suppliers`
  - `POST /v1/plugins/pharma_oa/api/suppliers/disable`
  - `GET /v1/plugins/pharma_oa/api/suppliers/qualification-reminders`
  - `GET /v1/plugins/pharma_oa/api/suppliers/purchase-eligibility`
- Customer route contracts:
  - `GET /v1/plugins/pharma_oa/api/customers`
  - `POST /v1/plugins/pharma_oa/api/customers`
  - `PUT /v1/plugins/pharma_oa/api/customers`
  - `POST /v1/plugins/pharma_oa/api/customers/disable`
  - `GET /v1/plugins/pharma_oa/api/customers/qualification-reminders`
  - `GET /v1/plugins/pharma_oa/api/customers/sales-eligibility`
- Warehouse route contracts:
  - `GET /v1/plugins/pharma_oa/api/warehouses`
  - `POST /v1/plugins/pharma_oa/api/warehouses`
  - `PUT /v1/plugins/pharma_oa/api/warehouses`
  - `POST /v1/plugins/pharma_oa/api/warehouses/disable`
  - `GET /v1/plugins/pharma_oa/api/warehouses/movement-eligibility`
- Master data import/export route contracts:
  - `GET /v1/plugins/pharma_oa/api/master-data/template`
  - `POST /v1/plugins/pharma_oa/api/master-data/import`
  - `GET /v1/plugins/pharma_oa/api/master-data/export`
- Purchase route contracts:
  - `GET /v1/plugins/pharma_oa/api/purchase-requests`
  - `POST /v1/plugins/pharma_oa/api/purchase-requests`
  - `GET /v1/plugins/pharma_oa/api/purchase-requests/detail`
  - `POST /v1/plugins/pharma_oa/api/purchase-requests/approve`
  - `POST /v1/plugins/pharma_oa/api/purchase-requests/reject`
  - `GET /v1/plugins/pharma_oa/api/purchase-orders`
  - `GET /v1/plugins/pharma_oa/api/purchase-orders/detail`
- Demo seed route contracts:
  - `GET /v1/plugins/pharma_oa/api/demo-seed/status`
  - `POST /v1/plugins/pharma_oa/api/demo-seed/apply`

The current employee module uses the core `/v1/pharma-oa/employees` API and integrated route `/skoll/pharma-oa/employees`. The product module uses the core `/v1/pharma-oa/products` API for drug master data, disable flow, and import validation. The supplier module uses `/v1/pharma-oa/suppliers` for supplier records, contacts, qualification attachment metadata, reminders, and purchase eligibility checks. The customer module uses `/v1/pharma-oa/customers` and `/skoll/pharma-oa/customers` for customer contacts, region ownership, qualification attachment metadata, organization/owner isolation, reminders, and sales eligibility checks. The warehouse module uses `/v1/pharma-oa/warehouses` for warehouse, area, location, temperature attributes, disable flow, and inbound/outbound movement location eligibility. The master data exchange module uses `/v1/pharma-oa/master-data/template`, `/v1/pharma-oa/master-data/import`, and `/v1/pharma-oa/master-data/export` for employee, product, supplier, and customer templates, row-level import reports, and export jobs. The purchase module uses `/v1/pharma-oa/purchase-requests` and `/v1/pharma-oa/purchase-orders` for supplier-qualified requests, workflow approval, rejection, and idempotent purchase-order generation.

## Lifecycle Smoke

The plugin is validated by `go test ./internal/plugin/...`. The dedicated test installs, enables, disables, checks permission/menu catalog effects, route extension registration, catalog audit events, and duplicate-install failure handling.
