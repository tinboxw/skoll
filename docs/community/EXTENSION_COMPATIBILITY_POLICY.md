# Extension Compatibility Policy

## Scope

This policy defines backward-compatibility expectations for Skoll extension APIs and manifest schema.

## Versioning Baseline

- Extension `version` follows numeric-dot format (example: `1.2.3`).
- Optional `v` prefix is accepted in API inputs.
- Version check compares numeric segments from left to right.

## Compatibility Levels

1. Compatible:
   - Existing manifest fields retain semantics.
   - Existing admin extension endpoints keep path and request contracts.
2. Conditionally compatible:
   - New optional manifest fields may be introduced.
   - New endpoints may be added without changing old behavior.
3. Breaking:
   - Removing or renaming existing manifest fields.
   - Changing endpoint paths or required payload fields.

## Deprecation Policy

- Breaking changes require:
  - roadmap note,
  - migration guidance,
  - at least one milestone notice before removal.
- Deprecated fields/endpoints should emit clear release note warnings.

## Runtime Compatibility Notes

- Current lifecycle hooks are declaration-only; no runtime ABI contract is guaranteed yet.
- Plugin package integrity is metadata-level in this milestone; future signature policy updates may tighten requirements.

## Recommended Provider Practices

- Keep extension release notes and migration steps per version.
- Avoid reusing package URLs for different contents.
- Test enable/disable and version-check flows against supported Skoll versions.
