# Plugin Business Event Contract

This document defines the only current protocol for delivering Skoll host business events to enabled external plugins.

## Manifest Declaration

A plugin receives only events explicitly declared in `plugin.yaml`. The handler is an internal dispatch key, not a URL.

```yaml
events:
  subscriptions:
    - name: approval-completed
      handler: onApprovalCompleted
      retry_policy: standard
```

| Policy | Maximum attempts | Retry delay |
| --- | ---: | ---: |
| `none` | 1 | no retry |
| `standard` | 3 | 1 second |
| `aggressive` | 5 | 250 milliseconds |

## Delivery Protocol

The host sends `POST /_skoll/events` below the plugin `service_base_url`, with `Content-Type: application/json`, `Idempotency-Key`, `X-Skoll-Plugin-ID`, and `X-Skoll-Event-Handler` headers.

```json
{
  "deliveryId": "business-delivery-...",
  "pluginId": "reports",
  "handler": "onApprovalCompleted",
  "eventId": "event-approval-20260722-0001",
  "eventName": "approval-completed",
  "source": "workflow",
  "subject": {"type": "approval", "id": "approval-1"},
  "payload": {},
  "metadata": {},
  "occurredAt": "2026-07-22T08:00:00Z"
}
```

Any `2xx` response succeeds. Network errors, redirects, and non-`2xx` responses fail; the host neither consumes nor records response bodies.

## Idempotency And Lifecycle

- Every host event has a stable, non-empty `eventId`.
- `deliveryId` is derived from `eventId + pluginId + handler` and remains stable across retries.
- Plugins enforce transaction idempotency with `Idempotency-Key` or `deliveryId`; the host also atomically claims deliveries and skips duplicate publications.
- Only enabled plugins with a current matching declaration receive events. Disable, uninstall, and reload remove old subscriptions and drain in-flight delivery.
- Failure, retry success, and dead-letter states remain observable. Delivery stops after the declared policy reaches its maximum attempts.

There is no legacy endpoint, envelope, or dual delivery path.
