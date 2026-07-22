# Plugin Business Document Contract

Skoll exposes the current business-document contract from `pkg/pluginsdk`. Independent plugins use it to describe and validate business records without importing host-internal form, workflow, database, or industry packages.

This contract defines document semantics only. Number reservation, persistence, workflow binding, attachments, search, export, and UI rendering are separate platform capabilities.

## Schema

A `DocumentSchema` declares:

- one lowercase plugin-local document key and positive schema version;
- typed header fields;
- zero or more named line groups with bounded row counts;
- a finite state set and one non-terminal initial state;
- explicit actions that move documents between states.

Every non-terminal state must have at least one outgoing action. Terminal states cannot be action sources, self-transitions are rejected, and state and action keys must be unique.

```go
schema := pluginsdk.DocumentSchema{
    Key:          "purchase_order",
    Name:         "Purchase order",
    Version:      1,
    InitialState: "draft",
    Header: []pluginsdk.DocumentFieldSchema{
        {Key: "supplier", Label: "Supplier", Type: pluginsdk.DocumentFieldReference, Required: true, ReferenceType: "supplier"},
        {Key: "total", Label: "Total", Type: pluginsdk.DocumentFieldMoney, Required: true},
        {Key: "needed_at", Label: "Needed at", Type: pluginsdk.DocumentFieldDateTime},
        {Key: "custom", Label: "Custom", Type: pluginsdk.DocumentFieldJSON},
    },
    Lines: []pluginsdk.DocumentLineSchema{{
        Key: "items", Name: "Items", MinItems: 1, MaxItems: 100,
        Fields: []pluginsdk.DocumentFieldSchema{
            {Key: "product", Label: "Product", Type: pluginsdk.DocumentFieldReference, Required: true, ReferenceType: "product"},
            {Key: "quantity", Label: "Quantity", Type: pluginsdk.DocumentFieldQuantity, Required: true},
            {Key: "unit_price", Label: "Unit price", Type: pluginsdk.DocumentFieldMoney, Required: true},
        },
    }},
    States: []pluginsdk.DocumentStateSchema{
        {Key: "draft", Name: "Draft"},
        {Key: "pending", Name: "Pending"},
        {Key: "approved", Name: "Approved", Terminal: true},
        {Key: "rejected", Name: "Rejected"},
    },
    Actions: []pluginsdk.DocumentActionSchema{
        {Key: "submit", Name: "Submit", From: []string{"draft", "rejected"}, To: "pending"},
        {Key: "approve", Name: "Approve", From: []string{"pending"}, To: "approved"},
        {Key: "reject", Name: "Reject", From: []string{"pending"}, To: "rejected", RequiresComment: true},
    },
}

if err := schema.Validate(); err != nil {
    return err
}
```

## Values

`DocumentValue` is an explicit tagged value. Plugins must not send Go or JSON floating-point numbers for business amounts or quantities.

| Type | Representation |
| --- | --- |
| `string`, `text` | UTF-8 `value` |
| `integer` | Canonical base-10 int64 in `value` |
| `decimal` | Plain base-10 decimal in `value`; exponent notation is rejected |
| `money` | Decimal `value` plus a three-letter uppercase `currency` |
| `quantity` | Decimal `value` plus a bounded `unit` |
| `boolean` | `true` or `false` in `value` |
| `date` | `YYYY-MM-DD` in `value` |
| `datetime` | Canonical UTC RFC3339Nano in `value` |
| `reference` | A typed `reference` containing `type`, `id`, and an optional display label |
| `json` | Bounded valid JSON in `value` for schema-declared custom data |

Attributes that do not belong to a value type are rejected. For example, a string cannot carry a currency and a money value cannot carry a unit.

Fields may declare deterministic `min`, `max`, `min_length`, `max_length`, and `pattern` rules. Numeric rules use exact rational comparison; they do not convert through `float64`.

## Records And Versions

A `DocumentRecord` binds one record to an exact schema key and version. It contains a number, title, state, positive optimistic version, typed header map, typed line groups, and auditable metadata.

`schema.ValidateRecord(record)` rejects:

- a mismatched type or schema version;
- an unknown or missing required field;
- an unknown line group, invalid row count, or duplicate line ID;
- a value type or validation-rule violation;
- an undeclared state or non-positive optimistic version;
- non-UTC, reversed, or unattributed metadata.

Map keys are sorted before unknown-field validation, so identical invalid payloads produce the same error path on every run.

`DocumentActionInput` carries a positive `expectedVersion`. Later document services must compare this version atomically before applying an action; no last-write-wins path is part of the contract.

## State Resolution

Use `schema.NextState(currentState, action, comment)` before applying an action. It verifies that the action is declared for the current state and enforces required comments. Approval workflow orchestration remains authoritative for who may execute the action; this contract only resolves the document transition.

## Errors

Validation returns `*pluginsdk.DocumentContractError` with stable `field` and `message` values. Callers should inspect the error with `errors.As` and bind `field` to the relevant form or line control.

## Current-Only Rule

Plugins use this single typed contract. Do not translate old dynamic-form records, accept untyped numeric values, infer missing value types, preserve undeclared fields, or add a fallback state transition path.

## Verification

From the repository root:

```powershell
go test ./pkg/pluginsdk -run Document -count=1
go test -race ./pkg/pluginsdk -run Document -count=1
go test ./... -count=1
go vet ./...
```
