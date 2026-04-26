# Plugin Upgrade Safety Baseline

## Overview

This document defines E5 baseline safety controls for plugin package installation and online upgrade.

## Feature Baseline

- package signature verification contract
- dependency and version precheck contract
- failed-upgrade rollback workflow

## API Endpoints

- `POST /admin/v1/plugins/packages/install`
- `POST /admin/v1/plugins/{name}/upgrade`

## Signature Contract

- payload field: `signature`
- verification rule: `signature == "sig:" + package_hash`
- invalid signatures fail package install/upgrade verification path

## Dependency Precheck Contract

Dependencies are passed as:

```json
"dependencies": [
  {"name": "core-ext", "min_version": "1.0.0"}
]
```

Precheck rules:

- dependency plugin must be installed
- dependency plugin version must be `>= min_version`

## Rollback Contract

Upgrade response contains rollout safety result:

```json
{
  "name": "audit-ext",
  "previous_version": "1.1.0",
  "target_version": "1.2.0",
  "succeeded": false,
  "rolled_back": true,
  "reason": "plugin signature verification failed"
}
```

- failed upgrade preserves previous version state.
- successful upgrade returns `succeeded=true` and `rolled_back=false`.
