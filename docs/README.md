# Skoll Documentation Index

> Last updated: 2026-07-04

Skoll is in open-source foundation construction. Documentation is organized by reader and purpose. Active planning stays easy to find; completed batches and detailed evidence are archived.

## Quick Links

| Entry | Purpose |
| --- | --- |
| [quick-start.md](quick-start.md) | Start a new environment, log in, and run core flows |
| [refactor/README.md](refactor/README.md) | Current refactor and pharma OA planning entry |
| [refactor/current/pharma_oa_milestone_plan_2026-07-04.md](refactor/current/pharma_oa_milestone_plan_2026-07-04.md) | F6-F12 pharma OA milestone and atomic task plan |
| [refactor/current/business_plugin_capability_plan_2026-07-04.md](refactor/current/business_plugin_capability_plan_2026-07-04.md) | Business-plugin capability and framework infrastructure plan |
| [refactor/governance/architecture_and_execution_plan.md](refactor/governance/architecture_and_execution_plan.md) | Architecture model and governance rules |

## Refactor Docs

| Directory | Purpose |
| --- | --- |
| [refactor/current](refactor/current) | Active plans and next-batch candidate work items |
| [refactor/governance](refactor/governance) | Architecture, execution principles, source map |
| [refactor/reference](refactor/reference) | Reusable implementation references |
| [refactor/old/completed-m0-m7-2026-07-04](refactor/old/completed-m0-m7-2026-07-04) | Completed M0-M7/FE/N0 task board, work items, acceptance logs, and quality evidence |

## Project Docs

| Document | Purpose |
| --- | --- |
| [configuration.md](configuration.md) | Configuration keys, defaults, and environment variables |
| [development_skills.md](development_skills.md) | Project-specific Codex skills and collaboration rules |
| [collaboration.md](collaboration.md) | Team roles, communication rhythm, task flow, and acceptance collaboration |
| [release-checklist.md](release-checklist.md) | Release quality gates, deployment, migration, examples, and rollback checklist |

## Architecture

| Document | Purpose |
| --- | --- |
| [architecture/README.md](architecture/README.md) | Architecture overview |
| [architecture/database.md](architecture/database.md) | Database design |
| [architecture/rbac.md](architecture/rbac.md) | RBAC permission model |

## API

| Document | Purpose |
| --- | --- |
| [api/README.md](api/README.md) | API overview, authentication, and response format |
| [api/openapi.yaml](api/openapi.yaml) | OpenAPI specification |

## Development

| Document | Purpose |
| --- | --- |
| [development/README.md](development/README.md) | Developer documentation entry |
| [development/getting-started.md](development/getting-started.md) | Developer quick start |
| [development/plugin-guide.md](development/plugin-guide.md) | Plugin development guide |
| [development/plugin_dev_tools.md](development/plugin_dev_tools.md) | Plugin development tools |

## User Docs

| Document | Purpose |
| --- | --- |
| [user/README.md](user/README.md) | User guide entry |
| [user/deployment.md](user/deployment.md) | Deployment and operations |
| [user/operations.md](user/operations.md) | Configuration, deployment, backup, logs, and troubleshooting |

## Smoke And Evidence

| Document | Purpose |
| --- | --- |
| [smoke/operations_reproduction_2026-07-04.md](smoke/operations_reproduction_2026-07-04.md) | Local reproduction record for Quick Start and operations docs |

## Schemas And Migrations

| Document | Purpose |
| --- | --- |
| [schemas/plugin-manifest.schema.json](schemas/plugin-manifest.schema.json) | Plugin manifest JSON Schema |
| [migrations/003_plugin_signature_support.sql](migrations/003_plugin_signature_support.sql) | Plugin signature migration script |

## Archive

| Directory | Purpose |
| --- | --- |
| [archive/legacy-plans](archive/legacy-plans) | Old plans kept only as historical reference |
| [archive/refactor-2026-06-19](archive/refactor-2026-06-19) | Earlier refactor process documents and evidence |
| [refactor/old/completed-m0-m7-2026-07-04](refactor/old/completed-m0-m7-2026-07-04) | Completed foundation-refactor evidence from the current refactor directory |

## Maintenance Rules

1. Keep active refactor execution under `docs/refactor/current/`.
2. Keep completed batches under `docs/refactor/old/<batch-name>/`.
3. Keep reusable technical references under `docs/refactor/reference/`.
4. Keep architecture and governance under `docs/refactor/governance/`.
5. Update this index whenever top-level or active refactor documents move.
6. Do not design legacy API, data, plugin, or page-route compatibility plans.
