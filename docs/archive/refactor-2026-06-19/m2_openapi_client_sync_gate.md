# M2 OpenAPI/Client Sync Gate

> Work Item: ADJ-20260619-02  
> Date: 2026-06-19  
> Scope: `/v1/audit` list, `/v1/audit/{id}` detail, and `/v1/audit/export`

## Conclusion

Passed. The audit handler DTOs, public OpenAPI files, and frontend API client are aligned for list, detail, and CSV export contracts.

## Contract Sources

| Contract surface | Source | Status |
| --- | --- | --- |
| HTTP routes | `internal/handler/http/v1/audit/handler.go` | Passed |
| Public OpenAPI | `docs/api/openapi.yaml` | Passed |
| Embedded handler OpenAPI | `internal/handler/http/openapi.yaml` | Passed |
| Frontend client | `web/src/audit/api.ts` | Passed |
| Audit page integration | `web/src/views/Audit/index.vue` | Passed |

## List Contract

| Field family | Handler | OpenAPI | Frontend client | Status |
| --- | --- | --- | --- | --- |
| Filters | `type`, `actorId`, `action`, `resourceType`, `resourceId`, `result`, `risk`, `from`, `to`, `offset`, `limit` | Same query parameters on `/v1/audit` | `AuditEventListQuery` and `buildAuditQuery` | Passed |
| Envelope | `code/message/data` via `WriteJSON` | `AuditEventListAPIResponse` | `ApiResponse<AuditEventListResult>` | Passed |
| Data | `items`, `offset`, `limit` | `AuditEventListData` with optional `total` | `AuditEventListResult` with optional `total` | Passed |

## Detail Contract

| Field family | Handler | OpenAPI | Frontend client | Status |
| --- | --- | --- | --- | --- |
| Route | `GET /v1/audit/{id}` | `/v1/audit/{id}` | `getAuditEvent(id)` | Passed |
| Envelope | `auditEventDetailData{item}` | `AuditEventDetailAPIResponse` | `ApiResponse<AuditEventDetailResult>` | Passed |
| Detail fields | `sourceData`, `diff.before`, `diff.after`, `diff.summary` | `AuditEventDetail` + `AuditEventDiff` | `AuditEventDetail` + `AuditEventDiff` | Passed |
| Errors | forbidden and not_found are explicit | 403 and 404 documented | `ApiError` is surfaced by page state | Passed |

## Export Contract

| Field family | Handler | OpenAPI | Frontend client | Status |
| --- | --- | --- | --- | --- |
| Route | `GET /v1/audit/export` | `/v1/audit/export` | `exportAuditEvents(query)` | Passed |
| Filters | Shared `parseEventFilter` | Same parameter set as list | Reuses `AuditEventListQuery` | Passed |
| Response | CSV attachment | `text/csv` | `Blob` | Passed |
| CSV columns | `eventId,sourceData` | Example and description match | Download uses returned blob unchanged | Passed |

## Verification

```powershell
rg "AuditEventListQuery|AuditEventDetail|eventId,sourceData" web/src/audit/api.ts docs/api/openapi.yaml internal/handler/http/openapi.yaml
Compare-Object (Get-Content -Encoding utf8 docs/api/openapi.yaml) (Get-Content -Encoding utf8 internal/handler/http/openapi.yaml)
go test ./internal/handler/http/v1/audit/...
cd web
npm run typecheck
npm run build
```

Result: Passed.
