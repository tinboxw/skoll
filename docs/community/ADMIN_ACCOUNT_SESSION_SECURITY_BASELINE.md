# Admin Account and Session Security Baseline

## Overview

This document defines the E3 baseline for account/session hardening in admin APIs.

## Account Hardening Endpoints

- `POST /admin/v1/users/{id}/password/rotate`
- `POST /admin/v1/users/{id}/login-failures`
- `POST /admin/v1/users/{id}/lock/reset`
- `POST /admin/v1/users/{id}/mfa`

## Session Hardening Endpoints

- `POST /admin/v1/sessions/revoke`
- `GET /admin/v1/sessions/{session_id}/status`
- `POST /admin/v1/sessions/anomalies`

## Baseline Rules

- password rotation supports minimal interval guard (`min_interval_minutes`).
- failed login events increment counters and lock user once threshold is reached.
- MFA is controlled by an extension-friendly provider field (`provider`).
- session revoke is explicit and queryable via status API.
- session anomaly events are recorded and counted.

## Audit Linkage

The baseline writes security events into audit pipeline using `actor=security` actions:

- `password_rotate`
- `login_failure`
- `lock_reset`
- `mfa_update`
- `session_revoke`
- `session_anomaly`

## Rollout Notes

- this baseline is additive and backward-compatible with existing user CRUD APIs.
- downstream consumers should tolerate additive fields in security responses.
