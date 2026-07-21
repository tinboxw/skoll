# 插件业务事件契约

本文定义 Skoll 宿主向已启用外部插件投递业务事件的唯一当前协议。

## 清单声明

插件只会收到 `plugin.yaml` 中明确声明的事件。处理器名称是插件服务内部的分发键，不是 URL。

```yaml
events:
  subscriptions:
    - name: approval-completed
      handler: onApprovalCompleted
      retry_policy: standard
```

`retry_policy` 只允许以下值：

| 策略 | 最大尝试次数 | 重试间隔 | 适用场景 |
| --- | ---: | ---: | --- |
| `none` | 1 | 不重试 | 确定性失败或不允许自动重放的处理器 |
| `standard` | 3 | 1 秒 | 默认业务集成 |
| `aggressive` | 5 | 250 毫秒 | 短暂故障且处理器可安全幂等 |

## 投递协议

宿主向插件 `service_base_url` 下的固定端点发送请求：

```http
POST /_skoll/events
Content-Type: application/json
Idempotency-Key: business-delivery-...
X-Skoll-Plugin-ID: reports
X-Skoll-Event-Handler: onApprovalCompleted
```

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

插件返回任意 `2xx` 表示处理成功。网络错误、重定向和非 `2xx` 状态均视为失败；宿主不会读取或记录响应正文。

## 幂等与生命周期

- 每个宿主事件必须提供稳定且非空的 `eventId`。
- `deliveryId` 由 `eventId + pluginId + handler` 确定，同一投递的初次发送与所有重试保持不变。
- 插件必须用 `Idempotency-Key` 或 `deliveryId` 防止自身事务重复执行；宿主也会原子认领投递并跳过重复发布。
- 只有已启用且仍声明该订阅的插件会收到事件。禁用、卸载或重载会注销旧订阅并等待在途投递结束。
- 失败、重试成功和死信状态可通过宿主业务事件记录查询。达到最大尝试次数后不会继续自动投递。

本协议不提供旧端点、旧信封或双路径投递。
