# Extension Developer Guide

## Goal

This guide defines the baseline process for developing and shipping Skoll extensions.

## Extension Lifecycle

1. Author a plugin manifest.
2. Install from manifest or package endpoint.
3. Enable/disable plugin through admin API.
4. Run version check for upgrade planning.

## Manifest Fields

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes | Unique plugin name. |
| `version` | string | Yes | Numeric-dot version. `v` prefix is accepted. |
| `hooks` | string[] | No | Declared lifecycle hooks. |
| `package_url` | string | No | Package source URL (set together with `package_hash`). |
| `package_hash` | string | No | Package integrity metadata (set together with `package_url`). |

## Admin APIs

- `POST /admin/v1/plugins/manifests`
- `POST /admin/v1/plugins/packages/install`
- `GET /admin/v1/plugins`
- `GET /admin/v1/plugins/{name}`
- `POST /admin/v1/plugins/{name}/enable`
- `POST /admin/v1/plugins/{name}/disable`
- `POST /admin/v1/plugins/{name}/version-check`

## Packaging Baseline

- Package metadata is persisted in plugin manifest records.
- `package_url` and `package_hash` must be provided together.
- Archive fetch, signature verification, and remote registry integration are out of this milestone scope.

## Hook Contract Baseline

Current baseline supports declarative hook names only. Runtime hook dispatch, sandboxing, and permission-scoped hook execution are planned in future milestones.

## Recommended Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Security Notes

- Prefer immutable package URLs and content-addressed hashes.
- Restrict package source domains via deployment policy.
- Keep admin APIs behind authentication and RBAC.
