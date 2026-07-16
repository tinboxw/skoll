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
- Purchase inbound route contracts:
  - `GET /v1/plugins/pharma_oa/api/purchase-inbounds`
  - `POST /v1/plugins/pharma_oa/api/purchase-inbounds`
  - `GET /v1/plugins/pharma_oa/api/purchase-inbounds/detail`
- Sales route contracts:
  - `GET /v1/plugins/pharma_oa/api/sales-orders`
  - `POST /v1/plugins/pharma_oa/api/sales-orders`
  - `GET /v1/plugins/pharma_oa/api/sales-orders/detail`
  - `GET /v1/plugins/pharma_oa/api/sales-outbounds`
  - `POST /v1/plugins/pharma_oa/api/sales-outbounds`
  - `GET /v1/plugins/pharma_oa/api/sales-outbounds/detail`
- Inventory operation route contracts:
  - `GET /v1/plugins/pharma_oa/api/stocktakes`
  - `POST /v1/plugins/pharma_oa/api/stocktakes`
  - `GET /v1/plugins/pharma_oa/api/stocktakes/detail`
  - `POST /v1/plugins/pharma_oa/api/stocktakes/approve`
  - `POST /v1/plugins/pharma_oa/api/stocktakes/reject`
  - `GET /v1/plugins/pharma_oa/api/transfers`
  - `POST /v1/plugins/pharma_oa/api/transfers`
  - `GET /v1/plugins/pharma_oa/api/transfers/detail`
- Inventory alert route contracts:
  - `GET /v1/plugins/pharma_oa/api/inventory-alerts`
  - `GET /v1/plugins/pharma_oa/api/inventory-alert-jobs`
  - `POST /v1/plugins/pharma_oa/api/inventory-alert-jobs`
  - `POST /v1/plugins/pharma_oa/api/inventory-alert-jobs/retry`
- Contract archive route contracts:
  - `GET /v1/plugins/pharma_oa/api/contracts`
  - `POST /v1/plugins/pharma_oa/api/contracts`
  - `GET /v1/plugins/pharma_oa/api/contracts/detail`
  - `POST /v1/plugins/pharma_oa/api/contracts/approve`
  - `POST /v1/plugins/pharma_oa/api/contracts/reject`
  - `POST /v1/plugins/pharma_oa/api/contracts/expiry-scan`
- Qualification route contracts:
  - `GET /v1/plugins/pharma_oa/api/qualifications`
  - `POST /v1/plugins/pharma_oa/api/qualifications/expiry-scan`
- Quality complaint route contracts:
  - `GET /v1/plugins/pharma_oa/api/quality-complaints`
  - `POST /v1/plugins/pharma_oa/api/quality-complaints`
  - `GET /v1/plugins/pharma_oa/api/quality-complaints/detail`
  - `POST /v1/plugins/pharma_oa/api/quality-complaints/resolve`
  - `POST /v1/plugins/pharma_oa/api/quality-complaints/reject`
  - `GET /v1/plugins/pharma_oa/api/quality-complaints/batches`
- Drug recall route contracts:
  - `GET /v1/plugins/pharma_oa/api/drug-recalls`
  - `POST /v1/plugins/pharma_oa/api/drug-recalls`
  - `GET /v1/plugins/pharma_oa/api/drug-recalls/detail`
  - `GET /v1/plugins/pharma_oa/api/drug-recalls/batches`
  - `GET /v1/plugins/pharma_oa/api/drug-recalls/scope`
  - `POST /v1/plugins/pharma_oa/api/drug-recalls/task-complete`
- Cold-chain route contracts:
  - `GET /v1/plugins/pharma_oa/api/cold-chain-contexts`
  - `GET /v1/plugins/pharma_oa/api/cold-chain-records`
  - `POST /v1/plugins/pharma_oa/api/cold-chain-records`
  - `GET /v1/plugins/pharma_oa/api/cold-chain-anomalies`
  - `GET /v1/plugins/pharma_oa/api/cold-chain-jobs`
  - `POST /v1/plugins/pharma_oa/api/cold-chain-jobs`
  - `POST /v1/plugins/pharma_oa/api/cold-chain-jobs/retry`
- Demo seed route contracts:
  - `GET /v1/plugins/pharma_oa/api/demo-seed/status`
  - `POST /v1/plugins/pharma_oa/api/demo-seed/apply`

The current employee module uses the core `/v1/pharma-oa/employees` API and integrated route `/skoll/pharma-oa/employees`. The product module uses the core `/v1/pharma-oa/products` API for drug master data, disable flow, and import validation. The supplier module uses `/v1/pharma-oa/suppliers` for supplier records, contacts, qualification attachment metadata, reminders, and purchase eligibility checks. The customer module uses `/v1/pharma-oa/customers` and `/skoll/pharma-oa/customers` for customer contacts, region ownership, qualification attachment metadata, organization/owner isolation, reminders, and sales eligibility checks. The warehouse module uses `/v1/pharma-oa/warehouses` for warehouse, area, location, temperature attributes, disable flow, and inbound/outbound movement location eligibility. The master data exchange module uses `/v1/pharma-oa/master-data/template`, `/v1/pharma-oa/master-data/import`, and `/v1/pharma-oa/master-data/export` for employee, product, supplier, and customer templates, row-level import reports, and export jobs. The purchase module uses `/v1/pharma-oa/purchase-requests` and `/v1/pharma-oa/purchase-orders` for supplier-qualified requests, workflow approval, rejection, and idempotent purchase-order generation. The sales module uses `/v1/pharma-oa/sales-orders`, `/v1/pharma-oa/sales-outbounds`, and `/skoll/pharma-oa/sales` for customer-qualified orders, batch outbound, stock deduction, and immutable ledger references. The inventory operation module uses `/v1/pharma-oa/stocktakes` for workflow-gated stocktake differences with idempotent ledger posting, and `/v1/pharma-oa/transfers` for atomic cross-warehouse movement with paired outbound and inbound ledger references. The inventory alert module runs retryable near-expiry, low-stock, and over-stock scans, writes deterministic reminders to the notification service, and links each alert to its warehouse, balance, and batch context.

## Lifecycle Smoke

The plugin is validated by `go test ./internal/plugin/...`. The dedicated test installs, enables, disables, checks permission/menu catalog effects, route extension registration, catalog audit events, and duplicate-install failure handling.
## Announcements and policy documents

- Announcement drafts target one or more organization IDs or role IDs before publication.
- Policy records require an attached document reference and share the same controlled publication lifecycle.
- Published records are visible only to matching audiences; read confirmation is idempotent and queryable.
- The host console route is `/skoll/pharma-oa/announcements`, backed by explicit read, create, publish, confirm, and receipt permissions.

## Contract archive

- Contract entries relate to supplier or customer master data and retain immutable file metadata references.
- Creation validates file access before launching an assigned approval workflow; approval and rejection remain auditable.
- Expiry scans create idempotent notification-center reminders that link back to the contract detail.

## Qualification management

- The unified ledger reads employee certificates plus supplier and customer qualifications without duplicating master data.
- Dynamic status classifies permanent, valid, expiring, and expired records using the selected reminder window.
- Expiry scans create idempotent owner/system notifications with direct links to the related master-data record.
- Expired supplier and customer qualifications block purchase and sales actions and append `pharma_oa.qualification.block` audit evidence.

## Quality complaints

- Complaint registration relates an active customer, active product, and matching inventory batch before workflow creation.
- Private evidence files are checked through the file access service and retained as immutable attachment metadata.
- A single assigned handler resolves or rejects the workflow with a required conclusion; terminal actions are idempotent and audited.
- The host console route is `/skoll/pharma-oa/quality-complaints`, with product-filtered batch selection and complete empty, error, no-permission, saving, destructive, detail, and responsive states.

## Drug recall

- Recall creation traces immutable sales outbound lines by inventory batch and groups affected quantity plus outbound evidence by customer.
- An optional source quality complaint must refer to the same product and batch before the recall can be created.
- Each affected customer becomes a tracked task with a required completion note; the recall closes automatically and idempotently after the final task.
- The host console route is `/skoll/pharma-oa/drug-recalls`, with batch scope preview, processing trace, permission-aware completion, and responsive states.

## Cold-chain records

- Readings are accepted only for positive inventory balances in enabled, temperature-controlled warehouse locations and retain the related product, batch, warehouse, area, and location evidence.
- Location temperature limits override area limits, which override warehouse limits; the effective limits are frozen into each immutable reading.
- Retryable scans evaluate the latest reading per stock balance against temperature and humidity limits, create idempotent reminders, and resolve reminders after a normal reading.
- Active and resolved anomalies expose risk, reasons, batch context, notification identity, and a direct `/skoll/pharma-oa/cold-chain` target for the compliance dashboard.

## Compliance audit dashboard

- The read-only dashboard aggregates active qualification, quality complaint, drug recall, and cold-chain risks without duplicating source-domain state.
- High and medium risks are filterable by source and keyword, retain source IDs plus batch/subject trace fields, and link directly to the owning console.
- Dashboard reads and CSV exports use the authenticated actor and append `pharma_oa.compliance_dashboard.view` or `.export` audit evidence with the applied filters and result counts.
- The host console route is `/skoll/pharma-oa/compliance-dashboard`, with loading, empty, error, no-permission, exporting, detail, and responsive states.
