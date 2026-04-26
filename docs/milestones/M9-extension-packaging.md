# M9-extension-packaging

## Summary

This milestone slice extends plugin lifecycle with packaging-oriented install flow and version-check capability, preparing a marketplace-ready extension operation baseline.

## Delivered

- Extended plugin manifest model with package metadata fields:
  - `package_url`
  - `package_hash`
- Added plugin manager capabilities:
  - `InstallPackage(name, version, packageURL, packageHash, hooks)`
  - `CheckVersion(name, latestVersion)`
- Added admin API endpoints:
  - `POST /admin/v1/plugins/packages/install`
  - `POST /admin/v1/plugins/{name}/version-check`
- Added validation rules:
  - numeric-dot version format only (`v` prefix accepted)
  - `package_url` and `package_hash` must be provided together
- Expanded tests:
  - plugin manager unit tests
  - admin route tests
  - storage adapter contract tests

## Validation

- `go fmt ./...`
- `go test ./...`
- `go test -race ./...`

## Risk Notes

- Package integrity is metadata-level only; no archive fetch/signature verification in this slice.
- Version comparison currently targets numeric segments and does not cover pre-release/build metadata.

## Next

- M9-ecosystem-docs (developer guide + compatibility policy)
