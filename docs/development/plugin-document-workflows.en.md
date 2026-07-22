# Plugin Document Workflow

`HostServices.Documents` is the current public service for binding a typed business document to a namespaced Skoll workflow. Plugins use it for leave requests, expenses, procurement, contracts, quality reviews, stock operations, and other approval-backed records. Plugins must not update document state separately from workflow state.

## Transaction Rule

`Submit` and `Act` require `HostServices.Transactions`. The document binding, document version, workflow instance/tasks/actions, and idempotency result commit or roll back together.

```go
var submitted pluginsdk.DocumentWorkflowResult
err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
    var err error
    submitted, err = host.Documents.Submit(tx.Context(), input)
    return err
})
```

`Get` is read-only and does not require a transaction. Every operation carries a tenant and permission. The host constrains that tenant against the verified user scope before reading or writing.

## State And Workflow Binding

- `Submit` applies the schema's exact `submit` transition, creates document version 1, starts one workflow instance, and persists their binding.
- `approve`, `reject`, `withdraw`, and `cancel` must exist as schema actions from the current document state. A successful action increments the document version once.
- `delegate` transfers the pending workflow task and intentionally leaves document state and version unchanged.
- The actor comes only from the trusted host context. Request payloads cannot choose the submitter or decision actor.
- A workflow instance belongs to exactly one document in one plugin namespace. Plugin and tenant keys prevent cross-plugin and cross-tenant reads.

## Idempotency And Concurrency

Every write carries an idempotency key. Reusing the key with identical actor and input returns the committed result with `duplicate=true`; changing any input returns `conflict`. Decisions also require `expectedVersion`. The repository locks the current binding, rechecks idempotency after lock acquisition, and rejects stale versions.

## Errors

`DocumentWorkflowError` exposes only current stable codes: `invalid_request`, `forbidden`, `not_found`, `conflict`, `transaction_required`, and `unavailable`. Both in-process and managed-process clients receive the same code, field, message, and retryability contract.

## Persistence

Migration 28 creates `sk_document_workflow_bindings` and `sk_document_workflow_actions` for MySQL and PostgreSQL. The first table stores the validated schema snapshot, current document snapshot, workflow identity, state, and version. The second stores immutable idempotency results and is cascade-bound to its document. Memory mode migrates the same GORM models.

There is one current implementation. No legacy document state path, dual write, compatibility adapter, local plugin workflow, or fallback transaction exists.

## Verification

```powershell
go test ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/store/sql/gormrepo -run "DocumentWorkflow|HostServices|ThirdPartyPlugin" -count=1
go test -race ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/store/sql/gormrepo -run "DocumentWorkflow|HostServices|ThirdPartyPlugin" -count=1
```
