# Generator Form Schema and Template Compatibility Policy

## Overview

This document defines E4 baseline policy for module generator form schema support and template compatibility.

## API Contract

`POST /admin/v1/generator/modules` supports additive fields:

- `template_version`: `v1` or `v2`
- `form_schema`: optional form schema payload

Example:

```json
{
  "module": "billing",
  "template_version": "v2",
  "form_schema": {
    "version": "v2",
    "fields": [
      {"name": "name", "type": "string", "required": true},
      {"name": "status", "type": "select", "required": false}
    ]
  }
}
```

## Compatibility Rules

- default `template_version` is `v1`.
- supported template versions: `v1`, `v2`.
- unsupported template version fails fast with explicit error.
- form schema versions follow same rule (`v1`, `v2`).
- duplicate form fields are deduplicated by field name.
- unknown future fields should be treated as additive and ignored where safe.

## Governance

Generator responses include compatibility metadata:

- `compatibility.template_version`
- `compatibility.compatibility_level`
- `compatibility.policy`

This metadata is used by frontend and plugin templates to decide migration behavior.
