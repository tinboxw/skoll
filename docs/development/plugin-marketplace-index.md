# Plugin Marketplace Index

The marketplace index is the canonical catalog consumed by Skoll marketplace services. It lists installable plugin releases and the verification material needed before a package can enter install preflight.

Schema: `docs/schemas/plugin-marketplace-index.schema.json`

## Shape

```json
{
  "schema_version": "v1",
  "generated_at": "2026-06-29T00:00:00Z",
  "publisher": {
    "id": "skoll",
    "name": "Skoll"
  },
  "plugins": [
    {
      "id": "demo",
      "name": "Demo Separated Plugin",
      "version": "0.2.0",
      "skoll_version": ">=1.0.0 <2.0.0",
      "manifest": {
        "path": "plugin.yaml",
        "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      },
      "risk": {
        "level": "medium",
        "summary": "Declares menu and route permissions, no destructive migration.",
        "permissions": { "level": "medium", "items": ["menu.read", "route.read"] },
        "migrations": { "level": "low", "items": [] },
        "network": { "level": "low", "items": [] },
        "assets": { "level": "low", "items": ["frontend dist assets"] }
      },
      "signature": {
        "required": true,
        "algorithm": "RSA-SHA256",
        "public_key_id": "skoll-marketplace-2026",
        "signed_at": "2026-06-29T00:00:00Z",
        "digest": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
        "value": "base64-signature"
      },
      "source": {
        "type": "zip",
        "url": "https://marketplace.example.com/plugins/demo-0.2.0.zip",
        "digest": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
        "size_bytes": 4096,
        "homepage": "https://marketplace.example.com/plugins/demo"
      },
      "changelog": {
        "summary": "Adds separated frontend and backend demo features.",
        "items": ["Adds plugin menu entry", "Adds demo route"],
        "breaking": false
      }
    }
  ]
}
```

## Required Release Fields

| Field | Purpose | Validation rule |
|---|---|---|
| `id` | Stable plugin identifier matching `plugin.yaml` | Lowercase stable id, unique with `version` in one index |
| `version` | Installable plugin release version | Semver, accepts optional leading `v` |
| `skoll_version` | Skoll core compatibility expression | Same intent as manifest `compatibility_skoll` |
| `risk` | Preflight risk report seed | Must include permission, migration, network, and asset sections |
| `signature` | Verification material for the indexed release | Required and pinned to `RSA-SHA256` for v1 |
| `source` | Download location and package digest | Must include HTTPS URL, size, and sha256 digest |
| `changelog` | User-facing release delta | Must include summary and at least one item |

## Index Rules

- The index is release-oriented: each `plugins[]` entry represents one plugin id and one version.
- No compatibility mode is defined for old plugin formats or source-injected plugins.
- `id`, `name`, `version`, and `skoll_version` must agree with the release manifest before install.
- `source.digest` verifies the package archive; `manifest.digest` verifies the manifest inside the package; `signature.digest` identifies the signed payload.
- `risk.level=critical` must block normal install until a privileged approval path exists.
- The next validator task must reject duplicate `(id, version)` pairs, malformed compatibility expressions, missing signatures, invalid digests, and HTTPS source violations.

## Preflight Handoff

The marketplace index does not install or mutate plugin state. It hands the following material to install preflight:

- release identity: `id`, `version`, `skoll_version`
- package provenance: `source`, `signature`, `manifest`
- user-facing review data: `risk`, `changelog`, `description`
- registry diff seeds: permissions, migrations, network, assets from `risk`
