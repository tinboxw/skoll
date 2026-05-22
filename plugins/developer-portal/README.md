# Developer Portal Plugin

This plugin provides a dedicated developer hub page that is mounted through the plugin system.

## Goals

- Keep developer utilities decoupled from core console pages.
- Allow production environments to remove this plugin entirely.
- Provide a single entry for future dev tools.

## Install

Use plugin install API in Skoll:

- `POST /skoll/v1/plugins/install`
- body: `{ "path": "plugins/developer-portal" }`

Then enable and open from Plugin Management.

## Current Modules

- Plugin Dev: scaffold plugin and validate all plugin manifests.
	- Supports scaffold `mode`: `workspace` (default) or `repository`.
- API Tools: authenticated request playground for backend endpoints.
- Ops Lab: placeholder module for future developer operations.

## Environment Notes

Dev APIs are available only when backend Dev Portal is enabled:

- `SKOLL_DEV_PORTAL_ENABLED=true`
- `SKOLL_DEV_PLUGINS_ROOT=plugins;D:/workspace/skoll-apps` (allowlist roots)

For non-default roots, set `pluginsRoot` to one of allowlisted values.

Scaffold mode notes:

- `workspace`: only creates base directories and manifest files.
- `repository`: additionally creates backend/frontend starter files for a standalone plugin repo layout.

## Troubleshooting: Unstyled Plugin Page

If the plugin page looks like browser-default controls (unstyled text/buttons/inputs), check:

- Open via frontend plugin route, not backend raw page endpoint.
	- Use: `/skoll/plugins/developer-portal`
	- Avoid direct: `/skoll/v1/plugins/developer-portal/page`
- Plugin page runs in an iframe. Host admin CSS is isolated and does not style plugin content.
- Force refresh (`Ctrl+F5`) and ensure static assets include version query (for example `style.css?v=...`).
- Avoid broad global CSS resets in plugin styles (for example fixed `button` height for all buttons), which can break card/module layout.
