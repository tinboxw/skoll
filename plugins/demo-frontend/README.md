# demo-frontend

This example follows frontend-only plugin style:
- no backend Go handlers/services
- frontend static assets only
- ui_mode: frontend_only
- frontend_entry exposed for plugin page navigation

## Core modules
1. Release Feed (frontend)
- Input: keyword from filter box.
- Output: release note cards filtered by version/title/tag.
- Demo value: quickly show how frontend-only plugin can deliver useful read-only tooling.

2. Command Simulator (frontend)
- Input: operation button click (`validate/install/enable/disable`).
- Output: timestamped simulated execution log.
- Demo value: provide interaction flow without any backend dependency.

## Usage example
1. Open `plugins/demo-frontend/static/index.html` directly in browser.
2. Type `ux` or `0.2.0` in filter box to verify release-feed filtering.
3. Click command buttons and inspect output panel updates.
