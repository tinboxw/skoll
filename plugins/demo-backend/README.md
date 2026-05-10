# demo-backend

This example follows backend-only plugin style:
- no frontend static assets
- ui_mode: backend_only
- plugin page must not show visit/default-home actions

## Core modules
1. Metrics Snapshot
- Endpoint: `GET /demo-backend/metrics`
- Query parameters:
	- `api`: api route hit count
	- `jobs`: background job hit count
	- `errors`: failed request count
- Value: produce availability score and aggregate traffic.

2. Audit Digest
- Endpoint: `GET /demo-backend/audit/report`
- Query parameters:
	- `events`: comma-separated event list, e.g. `user.create,user.update`
	- `limit`: top-N event categories
- Value: summarize top audit categories for operations dashboards.

## Usage example
- Start standalone demo service: `go run ./plugins/demo-backend`
- Test metrics snapshot:
	- `curl "http://127.0.0.1:18082/demo-backend/metrics?api=120&jobs=34&errors=3"`
- Test audit digest:
	- `curl "http://127.0.0.1:18082/demo-backend/audit/report?events=user.create,user.create,role.bind&limit=2"`
