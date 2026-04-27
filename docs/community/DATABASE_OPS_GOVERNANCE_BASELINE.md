# Database Ops Governance Baseline

## Overview

This document defines E6 baseline governance for database operations.

## Covered APIs

- `POST /admin/v1/db/migrations/plan`
- `POST /admin/v1/db/backup`
- `POST /admin/v1/db/restore`
- `POST /admin/v1/db/sql/execute`

## Governance Controls

- migration operations require explicit `from_version`, `to_version`, and steps.
- backup operations produce named backup records.
- restore requires explicit safeguard token: `confirm_token = I_UNDERSTAND`.
- dangerous SQL statements (`DROP`, `TRUNCATE`, `DELETE`) are blocked unless:
  - `allow_dangerous=true`
  - `confirm_token = I_UNDERSTAND`

## Audit Linkage

DB governance actions are written to audit pipeline (`actor=dbops`) with actions:

- `migration_plan`
- `backup`
- `restore`
- `sql_execute`

## Operational Notes

- this baseline provides governance contract and guardrails, not direct DB execution.
- rollout can later plug this contract into real migration/backup engines.
