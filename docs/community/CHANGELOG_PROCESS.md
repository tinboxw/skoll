# Changelog Process

## Scope

This process standardizes how Skoll updates `CHANGELOG.md` for open source releases.

## Update Rules

1. Every user-visible change must be recorded in `## [Unreleased]` before merge.
2. Group entries by type:
   - Added
   - Changed
   - Deprecated
   - Removed
   - Fixed
   - Security
3. Use concise, user-facing language and include affected file or feature references.

## Release Cut Procedure

1. Freeze `Unreleased` entries.
2. Create a new version section with date:
   - `## [X.Y.Z] - YYYY-MM-DD`
3. Move all relevant entries from `Unreleased` to the new section.
4. Keep an empty `Unreleased` section for subsequent development.

## Validation

Before release tag:

1. Ensure `CHANGELOG.md` includes current release changes.
2. Ensure versioning aligns with `docs/community/VERSIONING_POLICY.md`.
3. Ensure release note document references the same scope.
