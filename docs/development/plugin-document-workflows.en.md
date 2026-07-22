# Plugin Document Workflow

`HostServices.Documents` is the current public service for binding a typed business document to a namespaced Skoll workflow and its collaboration history. Plugins use it for leave requests, expenses, procurement, contracts, quality reviews, stock operations, and other approval-backed records. Plugins must not update document state separately from workflow state or keep a second attachment, comment, or timeline store.

## Transaction Rule

`Submit`, `Act`, `AddAttachment`, `RemoveAttachment`, and `AddComment` require `HostServices.Transactions`. The document binding, document version, workflow instance/tasks/actions, collaboration records, timeline events, and idempotency result commit or roll back together.

```go
var submitted pluginsdk.DocumentWorkflowResult
err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
    var err error
    submitted, err = host.Documents.Submit(tx.Context(), input)
    return err
})
```

`Get`, `ListAttachments`, `ListComments`, and `Timeline` are read-only and do not require a transaction. Every operation carries a tenant and permission. The host constrains that tenant against the verified user scope before reading or writing.

## State And Workflow Binding

- `Submit` applies the schema's exact `submit` transition, creates document version 1, starts one workflow instance, and persists their binding.
- `approve`, `reject`, `withdraw`, and `cancel` must exist as schema actions from the current document state. A successful action increments the document version once.
- `delegate` transfers the pending workflow task and intentionally leaves document state and version unchanged.
- The actor comes only from the trusted host context. Request payloads cannot choose the submitter or decision actor.
- A workflow instance belongs to exactly one document in one plugin namespace. Plugin and tenant keys prevent cross-plugin and cross-tenant reads.

## Attachments, Comments, And Timeline

- `AddAttachment` accepts only a current host file ID. The host resolves that file through the plugin- and actor-scoped file service before it creates the link, so a plugin cannot attach another plugin's or user's private file.
- `RemoveAttachment` is a soft unlink. It records remover identity, time, and sequence while preserving the original file snapshot and add attribution for review.
- `AddComment` creates an attributed immutable comment. The public service intentionally has no comment update or delete operation.
- Every submit, workflow action, attachment add/remove, and comment add appends one immutable timeline event. Sequence values are unique and strictly increasing within one plugin, tenant, and document.
- Timeline reads use `afterSequence` cursor paging. Attachment and comment lists use bounded offset paging; all page sizes are limited to 200.
- A document can hold at most 200 active attachments and 1,000 comments. Comment bodies are trimmed, non-empty, and limited to 8,000 UTF-8 bytes.

```go
err = host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
    if _, err := host.Documents.AddAttachment(tx.Context(), attachmentInput); err != nil {
        return err
    }
    _, err := host.Documents.AddComment(tx.Context(), commentInput)
    return err
})
```

## Idempotency And Concurrency

Document submit and workflow actions carry an idempotency key. Reusing the key with identical actor and input returns the committed result with `duplicate=true`; changing any input returns `conflict`. Decisions also require `expectedVersion`. Collaboration writes use caller-assigned immutable resource IDs and reject duplicate IDs. The repository locks the current binding so document actions and collaboration events receive a serialized timeline position.

## Errors

`DocumentWorkflowError` exposes only current stable codes: `invalid_request`, `forbidden`, `not_found`, `conflict`, `transaction_required`, and `unavailable`. Both in-process and managed-process clients receive the same code, field, message, and retryability contract.

## Persistence

Migration 28 creates `sk_document_workflow_bindings` and `sk_document_workflow_actions` for MySQL and PostgreSQL. Migration 29 adds attributed attachment links, immutable comments, and append-only timeline events. Timeline rows have no update or delete lifecycle columns and enforce a unique document sequence. Memory mode migrates the same GORM models.

There is one current implementation. No legacy document state path, dual write, compatibility adapter, local plugin workflow, or fallback transaction exists.

## Verification

```powershell
go test ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/store/sql/gormrepo -run "DocumentWorkflow|DocumentCollaboration|HostServices|ThirdPartyPlugin" -count=1
go test -race ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin ./internal/plugin/hostservice ./internal/store/sql/gormrepo -run "DocumentWorkflow|DocumentCollaboration|HostServices|ThirdPartyPlugin" -count=1
```
