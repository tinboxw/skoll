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

## Manifest Grants

Every managed backend declares the exact host operations it needs. The current manifest has no wildcard, group alias, compatibility name, or implicit grant:

```yaml
host_capabilities:
  - transactions.within
  - datastore.query
  - datastore.mutate
  - events.publish
```

An absent or empty list means deny all. Unknown and duplicated operations make the manifest invalid. The issued process credential captures an immutable grant snapshot, so changing a manifest or an in-memory descriptor cannot expand a running process. A grant authorizes only gateway dispatch; plugin identity, user permission, trusted data scope, schema ownership, transaction, and service-specific policies are still enforced.

Install preflight and the plugin control center publish this exact operation list for operator review. Plugin lifecycle control is deliberately not a host capability: a plugin process cannot install, enable, disable, or uninstall itself.

## User Identity And Scope

The plugin credential identifies the plugin, not an end user. A business request must attach the forwarded Skoll access token:

```go
ctx, err := pluginclient.BindRequestContext(r.Context(), userAccessToken, r.Header)
if err != nil {
    return err
}
scope, err := host.DataScopes.Resolve(ctx, pluginsdk.Permission{
    Resource: "equipment_maintenance.asset",
    Action:   "read",
})
```

The host verifies signature, expiry, subject, role, and organization claims again. User, tenant, or organization fields in a plugin request body are never authorization evidence.

## Operation Correlation

The platform assigns one trusted correlation identity before dispatching every plugin route. `BindRequestContext` keeps that identity with the verified user token, and `pkg/pluginclient` sends it on every host call. Remote transaction callbacks and their detached finish request retain the same identity. Events override caller-supplied correlation with the trusted identity, jobs persist it through leases, retries, dead letters, and restart, and host audits store it with request and trace references.

Operation evidence contains only the capability, plugin owner, stage, stable error code, and retryability. Request bodies, event payloads, configuration values, tokens, and secrets are never copied into generic host-call evidence. Plugin diagnostics expose the durable audit or job evidence reference so an operator can search one identity across request, transaction, event, job, workflow, and audit records.

## Capabilities

| Service | Current capability |
| --- | --- |
| `Transactions` | Execute host calls in one bounded remote transaction; commit on success and roll back on error, timeout, or credential revocation |
| `DataScopes` | Resolve trusted tenant, owner, and organization scope for the verified user |
| `DataStore` | Query and mutate declared plugin-owned relational tables through typed, scoped contracts |
| `DocumentNumbers` | Preview and atomically issue tenant-safe business identifiers |
| `Documents` | Submit business documents and apply workflow-bound approval actions atomically |
| `Files` | Store, list, get, download, and delete files in the plugin namespace |
| `Audit` | Record redacted evidence bound to plugin and trusted caller identity |
| `Config` | Read or replace Manifest-Schema-validated plugin configuration |
| `Secrets` | Read or set values in a private encrypted plugin namespace |
| `Workflows` | Execute namespaced conditional and parallel approvals with quorum decisions, task-scoped delegation, absence substitution, and durable escalation timers |
| `Jobs` | Schedule, lease, complete, fail, and query namespaced durable jobs |

The gateway validates public contracts again, binds the credential to one plugin identity, injects trusted scope, and maps failures to stable HTTP status and error fields. `pkg/pluginclient` validates requests and responses, restores public typed errors, and propagates transaction, cancellation, and deadline contexts. Plugin business code never uses private HTTP paths. Schema declaration and lifecycle rules are defined in [Plugin Datastore Schema And Lifecycle](plugin-datastore-schema.en.md); document approval rules are defined in [Plugin Document Workflow](plugin-document-workflows.en.md).

## Transactions

`Within` starts a credential-bound host transaction with a 30-second maximum lifetime. Every participating host call must use `tx.Context()`. Callback error, timeout, disconnect, credential revocation, or host shutdown rolls the transaction back. Nested transactions are rejected and calls within one transaction are serialized.

## Resource Governance

One shared controller applies an independent token bucket and concurrency boundary to every plugin. The governed resources are `request`, `host_call`, `query`, `mutation`, `event`, `job`, `export`, `storage`, and `process`. A busy plugin consumes only its own bucket and slots; another plugin keeps its own full allowance.

Persistent work also has capacity boundaries. Event publication reserves pending-outbox capacity, job scheduling reserves pending-job capacity, and file writes reserve both object-size and total plugin-storage capacity before mutation. Idempotent event and job replay remains readable at capacity and does not allocate a second record. Reservations are released after success or failure, so capacity recovers without a fallback queue.

Business routes enforce request bytes, response bytes, processing timeout, rate, and concurrency before returning data. Host calls enforce a separate rate and concurrency policy. A rejected call returns HTTP `429`, code `plugin_quota_exceeded`, `retryable: true`, `Retry-After`, and `X-Skoll-Quota-Resource`. Clients must honor the retry signal with bounded backoff; they must not switch to a private host path, direct database access, or an ungoverned local queue.

The managed process receives host-owned runtime limits:

| Variable | Meaning |
| --- | --- |
| `SKOLL_PLUGIN_MEMORY_LIMIT_BYTES` | Effective process memory ceiling |
| `SKOLL_PLUGIN_MAX_PROCS` | Effective process-count/Go scheduler ceiling |
| `GOMEMLIMIT` | Go runtime memory target derived from the host policy |
| `GOMAXPROCS` | Go scheduler parallelism derived from the host policy |

Windows applies a Job Object process-count and memory limit. Linux applies `RLIMIT_AS`; an unsupported operating system fails process startup. A crashed, unhealthy, disabled, or stopped process releases its process lease, allowing a later governed restart.

The plugin diagnostics endpoint and Element Plus control-center view expose rate, burst, active concurrency, available tokens, reserved capacity, rejection count, and last rejection time for every resource. Rejections also appear as retryable `quota` diagnostic errors with durable evidence. Operators can therefore tune the explicit `plugin.quota` configuration without granting a plugin an alternate execution path.

## Security Boundary

- The gateway binds a random loopback address and accepts only declared v1 `POST` operations.
- The gateway rejects an undeclared operation before decoding or dispatching its business payload.
- Every request authenticates the lifecycle credential; data scope also verifies the user JWT.
- Plugin business request and response limits come from `plugin.quota`; host-protocol frames remain bounded to 32 MiB. Unknown fields, operations, transactions, and non-loopback sources fail closed.
- Secrets are encrypted and keyed by plugin identity; another plugin cannot read the value, key material, or plaintext audit detail.
- Install, enable, disable, and uninstall require separate high-risk RBAC actions: `plugin.install`, `plugin.enable`, `plugin.disable`, and `plugin.uninstall`.
- Datastore failures expose only stable `code`, `field`, `message`, and `retryable` fields, never database, secret, or infrastructure details.
- There is one current host-service HTTP v1 path, with no remote database, old token, alternate URL, or fallback mode.

## Verification

```powershell
go test ./internal/plugin ./internal/plugin/quota ./internal/plugin/hostservice -run "Quota|HostGateway|ManagedProcessLauncher" -count=1
go test -race ./internal/plugin ./internal/plugin/quota ./internal/plugin/hostservice -run "Quota|HostGateway|ManagedProcessLauncher" -count=1
go list -deps ./pkg/pluginclient
```
