# Admin Dynamic Route/Menu/Button Permission Contract

## Overview

This document defines the E2 baseline contract for dynamic route/menu/button permissions.

## Endpoints

- `PUT /admin/v1/roles/{id}/permission-contract`
- `GET /admin/v1/roles/{id}/permission-contract`
- `POST /admin/v1/roles/{id}/permission-contract/consistency-check`

## Contract Payload

Request (`PUT`):

```json
{
  "version": "v2",
  "items": [
    {
      "menu_id": 1,
      "route": "/system/users",
      "buttons": ["create", "delete"]
    }
  ]
}
```

Response (`GET`/`PUT`):

```json
{
  "role_id": 1,
  "version": "v2",
  "items": [
    {
      "menu_id": 1,
      "route": "/system/users",
      "buttons": ["create", "delete"]
    }
  ]
}
```

Consistency response (`POST .../consistency-check`):

```json
{
  "role_id": 1,
  "passed": true,
  "problems": []
}
```

## Versioning Rules

- Supported versions: `v1`, `v2`.
- Empty version defaults to `v1`.
- Unknown versions fall back to `v1` for backward compatibility.

## Normalization Rules

- `route` is required; empty route entries are dropped.
- `buttons` are deduplicated and sorted.
- duplicate `(menu_id, route, buttons)` records are deduplicated.

## Consistency Rules

- each `menu_id` in contract must be bound by role menu bindings.
- empty route entries are rejected during consistency checks.

## Rollout Notes

- frontend should treat unknown additive fields as forward-compatible.
- backend fallback to `v1` avoids hard failures during phased rollout.
