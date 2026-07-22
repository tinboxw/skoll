# Skoll Plugin Runtime Batch Closeout

> Batch: `skoll-plugin-runtime-2026-07-21`
> Closed: 2026-07-22
> Result: 35/35 Work Items accepted; PR0-PR5 complete.
> Scope: plugin-platform functionality and product experience. This is not a release or regulatory-readiness declaration.

## Outcome

Skoll now executes installable business plugins as supervised external processes through one current manifest and lifecycle contract. Public host services provide identity, scope, files, audit, config, secrets, workflow, durable jobs, and transaction boundaries. The generator produces a full-stack plugin that builds and runs without hand edits, and the host console enforces a shared Element Plus, theme, locale, accessibility, responsive, visual, and performance system.

The independent equipment-maintenance plugin proves that a second industry can deliver assets, work orders, approval events, inspections, spare inventory, dashboards, persistence, jobs, and a complete frontend without plugin-specific host production changes.

## Final Acceptance

| Area | Result | Evidence |
| --- | --- | --- |
| Work Item completion | Pass | 35 of 35 official Work Items are `Done` |
| External runtime | Pass | Package install, managed-process start, readiness, execution, restart, disable, stop, and uninstall pass in one real E2E |
| Platform services | Pass | Lifecycle-bound external clients exercise data scope, transaction, audit, workflow, job, file, config, and secret contracts |
| Generator | Pass | Generated backend/frontend/package installs and executes CRUD with zero source edits |
| Independent industry | Pass | Equipment maintenance uses public contracts and has zero plugin-specific host production diff |
| Product experience | Pass | Element Plus, semantic themes, compact density, bilingual copy, accessibility, responsive behavior, visual checks, and budgets pass |
| Backend quality | Pass | `go test ./... -count=1`, lifecycle race tests, and `go vet ./...` pass |
| Frontend quality | Pass | `npm run typecheck`, `npm run test:components`, and `npm run build` pass |
| Repository hygiene | Pass | `git diff --check` and CodeGraph synchronization pass |

## Platform Boundary

The batch proves a usable plugin platform, but it does not make every future business system cheap to implement yet. The equipment proof stores business records in an atomic JSON file below the host-owned plugin data directory while its declared SQL migrations validate lifecycle behavior. Public external-process contracts do not yet expose a general transactional relational data/query service.

For production-scale medical OA plugins, the next functionality batch should add one first-class plugin datastore contract. It must provide tenant-scoped schema ownership, typed parameters, transactions, pagination, migrations, query limits, audit attribution, restart durability, and destructive uninstall behavior without exposing host internals. This is the highest-priority platform gap.

## Residual Risks

| Priority | Risk | Required response |
| --- | --- | --- |
| P0 | External plugins lack a general relational datastore/query capability | Build and prove a public plugin datastore service before expanding high-volume OA modules |
| P1 | Medical OA needs reusable business-document, form, numbering, approval, attachment, and timeline primitives | Add schema-driven document/workflow infrastructure before implementing each module independently |
| P1 | Runtime diagnostics are technically available but not yet a complete operator-facing plugin workspace | Add health, process, capability, job, audit, and data-policy views to the plugin console |
| P2 | Root and proof-plugin production bundles still carry known large dependency chunks | Continue route/component splitting and enforce per-route budgets during the next UI work |

## Next Functional Direction

1. Public plugin datastore and scoped query service.
2. Schema-driven business document, numbering, form, workflow, attachment, and timeline engine.
3. Plugin runtime and datastore console experience with complete operational states.
4. Medical OA vertical plugins covering organization and employees, approval, customers, suppliers, products, batches, inventory, purchasing, sales, quality, finance coordination, contracts, documents, assets, and reporting.

The next batch must remain functionality and UI focused. Release packaging, deployment, operations, and documentation-only milestones stay outside its critical path.
