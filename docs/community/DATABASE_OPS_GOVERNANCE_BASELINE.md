# Database Ops Governance Baseline

## Overview

This document defines E6 baseline governance for database operations.

## Covered APIs

- `POST /admin/v1/db/migrations/plan`
- `POST /admin/v1/db/migrations/drift-detect`
- `POST /admin/v1/db/backup`
- `GET /admin/v1/db/backups/catalog`
- `POST /admin/v1/db/restore`
- `POST /admin/v1/db/restore/drills`
- `GET /admin/v1/db/restore/drills`
- `POST /admin/v1/db/sql/execute`

## Governance Controls

- migration operations require explicit `from_version`, `to_version`, and steps.
- backup operations produce named backup records.
- restore requires explicit safeguard token: `confirm_token = I_UNDERSTAND`.
- restore drills produce evidence with RTO/RPO timing and data-check status.
- controlled SQL classes:
  - `read_only`: only read statements are allowed.
  - `write_guarded`: write statements require `confirm_token = I_UNDERSTAND`.
  - `destructive_confirmed`: destructive statements (`DROP`, `TRUNCATE`, `DELETE`) require:
    - `allow_dangerous=true`
    - `confirm_token = I_UNDERSTAND`
    - `confirm_token_dual = CONFIRM_DESTRUCTIVE_SQL`

## Audit Linkage

DB governance actions are written to audit pipeline (`actor=dbops`) with actions:

- `migration_plan`
- `migration_drift_detect`
- `backup`
- `restore`
- `restore_drill`
- `sql_execute`

## Operational Notes

- this baseline provides governance contract and guardrails, not direct DB execution.
- rollout can later plug this contract into real migration/backup engines.
