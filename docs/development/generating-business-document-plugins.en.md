# Generating Business Document Plugins

Skoll can generate an independent business-document plugin that uses only the public Go SDK and public Vue document package. The generated source does not import host `internal/` packages or host application pages.

## Generator Input

Set `Plugin.Enabled` and provide a `DocumentSpec`:

```go
input.Document = &generator.DocumentSpec{
    Enabled:      true,
    SchemaKey:    "purchase_request",
    SchemaName:   "Purchase Request",
    DefinitionID: "purchase-request-approval",
    NumberPrefix: "PR",
    TitleField:   "title",
}
```

`TitleField` must name a declared entity field. The generator maps non-primary entity fields to the public document schema and creates the `draft -> submitted -> approved/rejected` state model.

## Generated Boundary

The generated plugin contains:

- `plugin.yaml` with document routes, permissions, audit actions, data ownership, and lifecycle policy.
- `document-schema.json` as the reviewable business-document contract.
- `backend/main.go` using `pkg/pluginclient` and `pkg/pluginsdk` only.
- `web/src/document-schema.ts` and a Vue workspace using `@skoll/document-ui`.
- Plugin-owned migrations, package commands, and acceptance tests.

The backend creates drafts locally, then calls the host document service for submit, query, detail, approval, and export. Workflow definitions are created and published through the public workflow service. Formal document mutations run inside the public transaction service.

## Build And Package

From the generated plugin directory:

```powershell
./plugin.ps1 package
./plugin.ps1 verify
./plugin.ps1 install
```

On Unix-like systems:

```bash
sh ./plugin.sh package
sh ./plugin.sh verify
sh ./plugin.sh install
```

The source plugin is generated under `examples/plugins/<plugin-id>`. Its Go module resolves the repository's public SDK through the current repository layout, and its frontend resolves the public `packages/skoll-document-ui` package. The packaged runtime contains compiled assets and does not depend on host source paths.

## Acceptance

Run the contract test:

```powershell
go test ./internal/service/generator -run TestDocumentPluginTargetUsesOnlyPublicContracts -count=1
```

Run the full generated-package lifecycle:

```powershell
$env:SKOLL_GENERATOR_PLUGIN_E2E = "1"
go test ./internal/service/generator -run TestGeneratedPluginBuildPackageAndInstallWithoutSourceEdits -count=1 -v
```

The lifecycle gate must generate without manual edits, compile Go, typecheck and build Vue, package and verify the archive, install and enable the plugin, create and submit a document, approve it, schedule export, disable execution, roll back plugin-owned data according to policy, and uninstall.

Any failed gate returns the Work Item to `Doing`; fix the generated source template and rerun the complete lifecycle from generation.

## Current-Only Rule

Generated document plugins use the current manifest, SDK, UI package, and lifecycle contracts. Do not add legacy routes, alternate payloads, host-internal imports, dual implementations, compatibility adapters, or runtime fallbacks.
