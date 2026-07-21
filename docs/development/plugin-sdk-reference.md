# 插件 SDK 当前契约

本文是进程内 Go 插件使用 `github.com/tinboxw/skoll/pkg/pluginsdk` 的当前公开契约。第三方插件业务代码只能依赖该包和 Go 标准库，不得导入 `internal/` 下的服务、领域、仓储、存储或安全实现。

当前基础建设不提供旧 SDK、旧 manifest、双路径或降级适配。`HostServices.Validate` 任一端口缺失即构造失败。

## HostServices

| 端口 | 用途 | 宿主强制边界 |
| --- | --- | --- |
| `Transactions` | 多资源原子写入 | 回调使用宿主事务上下文，错误整体回滚 |
| `DataScopes` | 可信数据范围 | 主体、租户和组织来自已验证 JWT，只允许 `Constrain` 收窄 |
| `Files` | 插件文件与附件 | 固定插件键前缀、来源、所有者、MIME、哈希和内容策略 |
| `Audit` | 业务审计 | 固定插件动作/资源 namespace，可信主体，递归脱敏 |
| `Config` | manifest Schema 配置 | 当前 Schema 校验，敏感键拒绝，失败关闭 |
| `Secrets` | 私有密钥 | 插件/键隔离，AES-256-GCM 加密，明文不进入审计 |
| `Workflows` | 定义、启动和处理审批 | 定义、实例、节点、任务和业务类型绑定插件 namespace；动作主体来自可信上下文 |
| `Jobs` | 持久调度、租约、重试与死信 | ID、namespace 和 worker 绑定插件；租约查询不会跨插件 |

插件入口必须先校验完整宿主契约：

```go
func New(host pluginsdk.HostServices) (*Plugin, error) {
    if err := host.Validate(); err != nil {
        return nil, err
    }
    return &Plugin{host: host}, nil
}
```

## 生命周期契约

插件目录必须使用当前 `plugin.yaml`，并通过以下单一路径运行：

1. `Install` 解析并校验 manifest、权限、数据、API 与事件声明。
2. `Enable` 完成 migration、服务健康检查、权限/菜单/路由/事件发布。
3. `Disable` 立即关闭业务路由、事件订阅和受管服务。
4. `Uninstall` 按 manifest 数据策略执行并清理运行时注册。

禁用插件不保留可执行能力。插件不得在核心 router 或依赖容器中另行注册业务入口。

## 工作流

插件向 `Workflows` 传递本地 ID 和本地 key，宿主保存时加入插件 namespace，返回时仍使用本地值。插件不能传入持久化前缀，也不能读取其他插件的定义或实例。

审批、拒绝、撤回、转交和抄送的执行者由宿主从可信上下文生成；请求体中的 actor 字段不是公开契约。审批节点的 `AssigneeIDs` 和转交目标仍由业务插件声明。

```go
definition, err := host.Workflows.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
    ID: "purchase_approval", Key: "purchase.approval", Name: "Purchase approval", Version: 1,
    Nodes: []pluginsdk.WorkflowNode{
        {ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
        {ID: "review", Key: "review", Name: "Review", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{approverID}},
        {ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
    },
    Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "review"}, {From: "review", To: "end"}},
})
```

插件应使用 `WorkflowDefinition*`、`WorkflowInstance*`、`WorkflowTask*` 和 `WorkflowAction*` 常量判断状态，不依赖宿主内部类型。

## 持久任务

`Jobs.Schedule` 的本地 ID 和 `IdempotencyKey` 在当前插件 namespace 内唯一。`LeaseDue` 强制指定 worker、数量和租约时长，宿主只租用当前插件的到期任务。完成或失败必须携带当前租约令牌；旧令牌和过期令牌被拒绝。

```go
job, err := host.Jobs.Schedule(ctx, pluginsdk.JobScheduleInput{
    ID: "daily_sync_20260722", Kind: "daily.sync",
    IdempotencyKey: "daily-sync-2026-07-22", Payload: payload, MaxAttempts: 3,
})
```

插件必须显式处理 `retry_wait` 与 `dead_letter`，不得把调度成功当作业务完成。Payload 与 Result 各限制为 1 MiB。

## 一致性验收

参考插件位于 `plugins/sdk-conformance/`。其 Go 代码只导入公开 SDK 和标准库，并覆盖：

- manifest 安装、启用、禁用、卸载；
- 事务回调与可信范围；
- 文件写入/读取、审计、配置与密钥；
- 工作流定义、发布、启动、审批；
- 任务调度、租约与完成。

开发或修改 SDK 后必须执行：

```powershell
go test ./internal/plugin/hostservice ./pkg/pluginsdk ./plugins/sdk-conformance -run "Conformance|HostServices|WorkflowService|JobService" -count=20
go test -race ./internal/plugin/hostservice ./pkg/pluginsdk ./plugins/sdk-conformance -run "Conformance|HostServices|WorkflowService|JobService" -count=1
go list -deps ./plugins/sdk-conformance
```

`go list -deps` 结果不得包含 `github.com/tinboxw/skoll/internal/`。完整宿主安全规则见 [插件宿主服务契约](plugin-host-services.md)，manifest 与 API 规则见 [插件开发教程](plugin-guide.md)。
