# M2 Audit Smoke Fixture

> Work Item: ADJ-20260619-04  
> Date: 2026-06-19  
> Fixture: `docs/refactor/fixtures/m2_audit_smoke_events.json`

## Purpose

This fixture defines fixed audit samples for the M2 smoke script. It keeps the next script task focused on execution mechanics instead of inventing ad hoc event data.

## Scenario Coverage

| Scenario | Event type | Action | Result | Risk | Query purpose | Status |
| --- | --- | --- | --- | --- | --- | --- |
| `login_failed` | `login` | `auth.session.login_failed` | `failure` | `medium` | Login failure can be queried and exported. | Passed |
| `forbidden` | `security` | `system.security.deny` | `denied` | `high` | Permission denial can be queried and exported. | Passed |
| `plugin` | `plugin` | `plugin.lifecycle.enable` | `success` | `high` | Plugin lifecycle event can be queried and exported. | Passed |
| `menu` | `operation` | `menu.node.update` | `success` | `medium` | Menu registry change can be queried and exported. | Passed |
| `export` | `operation` | `audit.event.export` | `success` | `low` | Export output can assert fixed `eventId,sourceData` columns. | Passed |

## Fixture Rules

1. Scenario IDs stay stable for scripts and acceptance logs.
2. Actions use the current `module.resource.action` rule; old bare actions such as `login_failed` are not used as event actions.
3. Every sample has `metadata.scenario` matching the scenario ID.
4. Every sample has `sourceData`, because `/v1/audit/export` exports `eventId,sourceData`.
5. Every sample defines `expectedQueries.list` and `expectedQueries.exportContains` for script assertions.
6. Samples are deterministic and use the same timestamp day so script time-window filters can be fixed.

## Script Consumption Notes

`M2-06-01` should load this fixture and seed or assert equivalent records before running smoke checks. The script should verify:

| Check | Expected behavior |
| --- | --- |
| List query | Each `expectedQueries.list` returns the matching `event.id`. |
| Detail query | Each matching event detail includes `sourceData` and trace context. |
| Export query | CSV response includes header `eventId,sourceData` and all `exportContains` tokens for the scenario. |
| Forbidden UI/API state | Restricted audit access produces the no-permission path and a `system.security.deny` event. |

## Verification

```powershell
rg -n 'login_failed|forbidden|plugin|menu|export|eventId|sourceData' docs/refactor/fixtures/m2_audit_smoke_events.json
rg -n "auth.session.login_failed|system.security.deny|plugin.lifecycle.enable|menu.node.update|audit.event.export" docs/refactor/fixtures/m2_audit_smoke_events.json docs/refactor/m2_audit_smoke_fixture.md
$fixture = Get-Content -Raw -Encoding utf8 docs/refactor/fixtures/m2_audit_smoke_events.json | ConvertFrom-Json
$fixture.scenarios.Count
```

Result: Passed.
