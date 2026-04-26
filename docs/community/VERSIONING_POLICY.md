# Versioning Policy

## Scheme

Skoll uses Semantic Versioning: `MAJOR.MINOR.PATCH`.

- MAJOR: incompatible API or behavior changes.
- MINOR: backward-compatible features.
- PATCH: backward-compatible fixes.

## Pre-release Tags

Use pre-release identifiers for milestone builds:

- `vX.Y.Z-alpha.N`
- `vX.Y.Z-beta.N`
- `vX.Y.Z-rc.N`

## Compatibility Rules

1. Keep probe and metrics endpoints stable across patch/minor releases.
2. Treat production auth-mode behavior changes as compatibility-sensitive.
3. Document migration actions for any config behavior changes.

## Release Checklist

Before tagging a release:

1. Validate fmt/test/race/bench gates.
2. Update changelog with user-visible changes.
3. Confirm README CN/EN parity.
4. Confirm milestone/release notes are updated.
