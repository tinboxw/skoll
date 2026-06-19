# M2 Export/List Filter Parity Check

> Work Item: ADJ-20260619-01  
> Date: 2026-06-19  
> Scope: `/v1/audit` list and `/v1/audit/export`

## Conclusion

Passed. The audit event list API and CSV export API share the same filter parser and therefore use the same filtering semantics for type, actor, resource, action, result, risk, time range, offset, and limit.

## Shared Filter Entry Point

Both paths use `parseEventFilter` in `internal/handler/http/v1/audit/handler.go`.

| API path | Handler path | Service method | Filter source |
| --- | --- | --- | --- |
| `GET /v1/audit` | `queryEvents` | `EventService.ListEvents` | `parseEventFilter` |
| `GET /v1/audit/export` | `export` | `EventService.ExportEventSourceData` | `parseEventFilter` |

## Filter Fields

| Query field | List behavior | Export behavior | Status |
| --- | --- | --- | --- |
| `type` | Validated by `domainaudit.ParseEventType` | Same parser | Passed |
| `actorId` | Trimmed into `EventFilter.ActorID` | Same parser | Passed |
| `action` | Validated by `domainaudit.ParseAuditAction` | Same parser | Passed |
| `resourceType` | Trimmed into `EventFilter.ResourceType` | Same parser | Passed |
| `resourceId` | Trimmed into `EventFilter.ResourceID` | Same parser | Passed |
| `result` | Lower-cased and validated | Same parser | Passed |
| `risk` | Lower-cased and validated | Same parser | Passed |
| `from` / `to` | RFC3339 pair required | Same parser | Passed |
| `offset` | Non-negative, default `0` | Same parser | Passed |
| `limit` | Bounded to max `200`, default `50` | Same parser | Passed |

## Test Coverage

`internal/handler/http/v1/audit/handler_test.go` includes:

- `TestAuditHandlerListEventsWithFilters`: asserts list filter mapping for type, actor, action, resource, result, risk, time, offset, and bounded limit.
- `TestAuditHandlerExportEventsWithFilters`: asserts export filter mapping for the same filter family and stable CSV columns.
- `TestAuditHandlerListEventsRejectsInvalidFilter`: asserts invalid action rejects list requests.
- `TestAuditHandlerExportEventsRejectsInvalidFilter`: asserts invalid action rejects export requests.

## Verification

```powershell
rg "parseEventFilter" internal/handler/http/v1/audit
rg "ExportEventSourceData" internal/handler/http/v1/audit
rg "ListEvents" internal/handler/http/v1/audit
go test ./internal/handler/http/v1/audit/...
```

Result: Passed.
