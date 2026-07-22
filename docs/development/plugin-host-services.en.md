# Plugin Host Services Contract

Independent plugin processes consume Skoll host capabilities only through `github.com/tinboxw/skoll/pkg/pluginclient`. They must not import `internal/`, read host database credentials, or construct host adapters.

## Initialization

```go
client, err := pluginclient.FromEnvironment()
if err != nil {
    return err
}
host, err := client.HostServices()
if err != nil {
    return err
}
```

The host injects `SKOLL_PLUGIN_ID`, loopback-only `SKOLL_PLUGIN_HOST_URL`, and lifecycle-only `SKOLL_PLUGIN_HOST_TOKEN`. These bootstrap values must never be logged or returned. Disable, crash, uninstall, and host shutdown revoke the token and roll back active transactions. Re-enable always receives a new token.

## User Identity And Scope

The plugin credential identifies the plugin, not an end user. A business request must attach the forwarded Skoll access token:

```go
ctx := pluginclient.WithUserToken(r.Context(), userAccessToken)
scope, err := host.DataScopes.Resolve(ctx, pluginsdk.Permission{
    Resource: "equipment_maintenance.asset",
    Action:   "read",
})
```

The host verifies signature, expiry, subject, role, and organization claims again. User, tenant, or organization fields in a plugin request body are never authorization evidence.

## Capabilities

| Service | Current capability |
| --- | --- |
| `Transactions` | Execute host calls in one bounded remote transaction; commit on success and roll back on error, timeout, or credential revocation |
| `DataScopes` | Resolve trusted tenant, owner, and organization scope for the verified user |
| `DataStore` | Query and mutate declared plugin-owned relational tables through typed, scoped contracts |
| `Files` | Store, list, get, download, and delete files in the plugin namespace |
| `Audit` | Record redacted evidence bound to plugin and trusted caller identity |
| `Config` | Read or replace Manifest-Schema-validated plugin configuration |
| `Secrets` | Read or set values in a private encrypted plugin namespace |
| `Workflows` | Create, publish, and execute namespaced approval workflows |
| `Jobs` | Schedule, lease, complete, fail, and query namespaced durable jobs |

The datastore gateway validates the public contract again, binds the credential to one plugin identity, injects trusted scope, and maps datastore failures to stable HTTP status and error fields. Client-side datastore error reconstruction and conformance coverage are completed by BF1-06; plugin business code never uses private HTTP paths.

## Transactions

`Within` starts a credential-bound host transaction with a 30-second maximum lifetime. Every participating host call must use `tx.Context()`. Callback error, timeout, disconnect, credential revocation, or host shutdown rolls the transaction back. Nested transactions are rejected and calls within one transaction are serialized.

## Security Boundary

- The gateway binds a random loopback address and accepts only declared v1 `POST` operations.
- Every request authenticates the lifecycle credential; data scope also verifies the user JWT.
- Requests and responses are limited to 32 MiB. Unknown fields, operations, transactions, and non-loopback sources fail closed.
- Datastore failures expose only stable `code`, `field`, `message`, and `retryable` fields, never database, secret, or infrastructure details.
- There is one current host-service HTTP v1 path, with no remote database, old token, alternate URL, or fallback mode.

## Verification

```powershell
go test ./internal/plugin -run "TestHostGateway|TestManagedProcessLauncher" -count=1
go test -race ./internal/plugin -run "TestHostGateway|TestManagedProcessLauncher" -count=1
go list -deps ./pkg/pluginclient
```
