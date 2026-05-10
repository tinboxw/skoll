# demo-monolith

This example follows the non-separated plugin style:
- backend code and static resources in one plugin directory
- ui_mode: monolith
- frontend_entry exposed for plugin page navigation

## Core modules
1. Widget Data Service (backend)
- Endpoint: `GET /demo-monolith/widgets`
- Query parameters:
	- `visitors`: current visitor count
	- `error_rate`: current error percentage
- Value: returns widget list for monolith dashboard rendering.

2. Page Manifest Service (backend + frontend bridge)
- Endpoint: `GET /demo-monolith/manifest`
- Value: returns frontend entry path and static assets so host app can mount the plugin page.

## Usage example
- Start standalone demo service: `go run ./plugins/demo-monolith`
- Query widgets:
	- `curl "http://127.0.0.1:18083/demo-monolith/widgets?visitors=48&error_rate=1.7"`
- Query page manifest:
	- `curl "http://127.0.0.1:18083/demo-monolith/manifest"`
