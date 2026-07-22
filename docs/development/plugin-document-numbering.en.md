# Plugin Document Numbering

Skoll exposes tenant-safe document numbering through `HostServices.DocumentNumbers`. Plugins use this service for business identifiers such as purchase orders, sales orders, stock movements, approval requests, and customer records. Plugins must not implement independent counters.

## Rule

A `DocumentNumberRule` declares a lowercase document type, uppercase prefix, separator, UTC period, sequence width, starting value, and gap policy. The current platform supports `none`, `year`, `month`, and `day` periods and the `transactional` gap policy.

The sequence namespace is:

```text
plugin + tenant + document type + period
```

Once the first number is issued in a namespace, the complete rule is frozen for that period. A different prefix, separator, width, start, period, or gap policy returns a `conflict`. This prevents one active sequence from producing incompatible identifiers.

## Preview

`Preview` returns the next candidate without reserving it. It is suitable for forms and operator feedback, but it is never authoritative and may change before issuance.

```go
preview, err := host.DocumentNumbers.Preview(ctx, input)
```

## Issue

Formal issuance must run inside the host transaction that creates or changes the business document.

```go
err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
    issued, err := host.DocumentNumbers.Issue(tx.Context(), input)
    if err != nil {
        return err
    }
    return persistDocument(tx.Context(), issued.Number)
})
```

The `IdempotencyKey` is mandatory for issuance and identifies one business issuance attempt within a plugin, tenant, and document type. Repeating the same key and input returns the original number with `Duplicate` set to `true`. Reusing the key with different input returns `conflict`.

The transactional gap policy advances the sequence and writes the idempotency record in the caller's transaction. A rollback removes both changes, so the number can be reused. A committed document number is never reassigned.

## Isolation And Authorization

Every request names a tenant and a permission. The host resolves the authenticated user's trusted data scope and rejects requests whose tenant is outside that scope. Plugin identity comes from the managed host credential and cannot be supplied by the plugin request.

Sequences are independent across tenants, document types, plugins, and periods. Concurrent issuance is serialized by the database and protected by a unique sequence constraint.

## Errors

The public client preserves `DocumentNumberError` across the process boundary:

| Code | Meaning |
| --- | --- |
| `invalid_request` | Input or rule validation failed. |
| `forbidden` | The tenant is outside the authenticated data scope. |
| `conflict` | An idempotency key or active numbering rule conflicts. |
| `transaction_required` | Formal issuance was attempted outside a host transaction. |
| `exhausted` | The configured sequence width has no remaining values. |
| `unavailable` | The host could not complete the operation. |

Plugins should correct non-retryable errors. Only errors with `Retryable` set by the host may be retried automatically, using the same idempotency key.
