# M2 Manual Acceptance Steps

> Work Item: M2-06-02  
> Date: 2026-06-19  
> Scope: audit event query, detail, export, UI states, and smoke logs

## Preconditions

1. Backend is running with audit event service enabled.
2. Frontend is running and can reach the same API prefix.
3. An admin user can log in.
4. Optional fixed fixture scenarios from `docs/refactor/fixtures/m2_audit_smoke_events.json` are seeded when strict fixture validation is required.

## API Smoke

| Step | Command | Expected result |
| --- | --- | --- |
| Basic smoke | `powershell -ExecutionPolicy Bypass -File ./scripts/smoke-auth-audit.ps1` | Health, login, current user, audit list, export, and runtime `login_failed` scenario pass. |
| Strict fixture smoke | `powershell -ExecutionPolicy Bypass -File ./scripts/smoke-auth-audit.ps1 -StrictFixtureAssertions` | All fixed fixture scenarios are found after seed: `login_failed`, `forbidden`, `plugin`, `menu`, `export`. |
| Backend audit tests | `go test ./internal/handler/http/v1/audit/...` | List, detail, filter, invalid filter, and export tests pass. |

## UI Query Flow

| Step | Action | Expected result |
| --- | --- | --- |
| Open audit page | Log in as admin and open `/audit`. | Audit page loads with type tabs, filters, refresh/export/clear actions, and table or empty state. |
| Query login failures | Select `login`, set action `auth.session.login_failed`, set result/risk through available filters where applicable, refresh. | Matching login failure events are visible or empty state clearly reports no match. |
| Query security denial | Select `security`, use action `system.security.deny`, refresh. | Forbidden events are visible when fixture or real denial events exist. |
| Query plugin events | Select `plugin`, use action `plugin.lifecycle.enable`, refresh. | Plugin lifecycle events are visible when fixture or real plugin events exist. |
| Query menu events | Select `operation`, use action `menu.node.update`, resource type `menu`, refresh. | Menu change events are visible when fixture or real menu events exist. |

## Detail Flow

| Step | Action | Expected result |
| --- | --- | --- |
| Open detail | Click an audit row. | Detail drawer opens. |
| Inspect summary | Review id, type, actor, action, resource, risk, occurred time. | Summary fields match the row. |
| Inspect trace | Review trace JSON. | method, path, ip, userAgent/request context are readable when present. |
| Inspect source data | Review source data JSON. | Original payload is present for event source exports. |
| Inspect diff | Review diff JSON for menu-like samples. | before/after/summary are visible when source data contains them. |

## Export Flow

| Step | Action | Expected result |
| --- | --- | --- |
| Export current filter | Apply a filter and click export. | CSV download starts and success alert is visible. |
| Inspect CSV | Open the exported file. | Header is `eventId,sourceData`; rows use compact JSON source data. |
| Verify filter parity | Compare exported rows with list filter. | Export uses the current list filter family. |

## State Checks

| State | How to check | Expected result |
| --- | --- | --- |
| Loading | Refresh audit list on a slow backend or throttled browser. | Table loading state is visible and controls are disabled where needed. |
| Empty | Use a filter that returns no rows. | Empty state is visible and stale rows are removed. |
| Error | Stop backend or force a failing API response. | Error alert is visible and retry controls remain usable. |
| No permission | Use a restricted role or strict fixture denial path. | No-permission result is visible for 403, not a misleading empty table. |
| Narrow viewport | Check at 390px width. | Header actions and filters stack without overlap. |

## Acceptance Record Template

Use this block in `docs/refactor/acceptance_log.md` for manual runs:

```text
Browser:
Backend mode:
Fixture seeded: yes/no
Smoke command:
Smoke result:
UI query result:
Detail result:
Export file:
State checks:
Known warnings:
```

## Pass Criteria

M2 manual acceptance passes only when:

1. Basic smoke command passes.
2. Strict fixture smoke passes when fixed fixture data is seeded.
3. Audit page query/detail/export flows are manually checked.
4. Empty, error, no-permission, and narrow viewport states are recorded.
5. Any warning is either resolved or explicitly carried into the next Work Item.
