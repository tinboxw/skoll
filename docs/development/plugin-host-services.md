# 插件宿主服务契约

> 默认语言：简体中文。English: [Plugin Host Services Contract](plugin-host-services.en.md)

独立插件进程只能通过 `github.com/tinboxw/skoll/pkg/pluginclient` 使用 Skoll 宿主能力。插件不得导入 `internal/`、读取宿主数据库凭证或自行构造宿主服务适配器。

## 初始化

宿主启动插件时注入当前生命周期的插件 ID、宿主网关地址和随机凭证。插件只需从环境创建客户端并校验完整服务集合：

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

环境变量只用于进程引导，不应写入日志、文件或业务响应：

| 变量 | 含义 |
| --- | --- |
| `SKOLL_PLUGIN_ID` | 当前 Manifest 插件 ID |
| `SKOLL_PLUGIN_HOST_URL` | 仅回环 HTTP 的宿主网关地址 |
| `SKOLL_PLUGIN_HOST_TOKEN` | 仅本次进程生命周期有效的宿主凭证 |
| `SKOLL_PLUGIN_MEMORY_LIMIT_BYTES` | 宿主下发的进程内存上限 |
| `SKOLL_PLUGIN_MAX_PROCS` | 宿主下发的进程数与 Go 调度并行度上限 |

插件禁用、崩溃、卸载或宿主关闭时，凭证立即失效，活动事务回滚。重新启用会签发新凭证，旧凭证不会恢复。

## 用户身份与数据范围

宿主凭证证明“哪个插件正在调用”，不能代表终端用户。处理业务 HTTP 请求时，插件必须把 Skoll 转发的用户 access token 放入调用上下文：

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

宿主会重新校验 JWT 签名、有效期、用户和组织声明。缺少或无效的用户 token 时，数据范围解析失败。插件请求体中的用户、租户和组织字段不能作为授权依据。

## 服务清单

| 服务 | 当前能力 |
| --- | --- |
| `Transactions` | 在一个有界远程事务会话中执行宿主调用；成功提交，错误、超时或凭证失效回滚 |
| `DataScopes` | 根据已验证用户和权限解析租户、所有者、组织范围 |
| `Files` | 在插件命名空间中存储、列出、读取、下载和删除文件 |
| `Audit` | 以插件与可信调用者身份记录脱敏审计证据 |
| `Config` | 读取或按当前 Manifest Schema 整体替换插件配置 |
| `Secrets` | 在插件私有加密命名空间中读取或设置密钥 |
| `Workflows` | 创建、发布和执行插件命名空间内的审批流程 |
| `Jobs` | 调度、租用、完成、失败和查询插件命名空间内的持久任务 |

所有方法沿用 `pkg/pluginsdk` 的公共类型与校验规则。`pkg/pluginclient` 实现完整的 `pluginsdk.HostServices`，业务代码无需接触 HTTP 路径或私有协议字段。

## 事务

`Within` 会启动一个绑定当前插件凭证、最长 30 秒的宿主事务会话。参与事务的宿主调用必须使用 `tx.Context()`：

```go
err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
    if _, err := host.Config.Replace(tx.Context(), values); err != nil {
        return err
    }
    _, err := host.Audit.Record(tx.Context(), receipt)
    return err
})
```

回调返回错误会回滚。事务不允许嵌套，同一事务内的调用串行执行；超时、失联、凭证撤销和宿主关闭都按失败处理。插件自己的独立数据事务由插件数据生命周期契约负责，不能把长事务会话当作进程数据库连接。

## 资源治理

平台使用一个共享控制器，为每个插件分别维护令牌桶、并发槽位和容量预留。当前治理资源为 `request`、`host_call`、`query`、`mutation`、`event`、`job`、`export`、`storage`、`process`。某个插件耗尽额度时，只拒绝该插件的对应资源，其他插件继续使用自己的完整额度。

持久任务在写入前预留容量：事件受待投递 outbox 数量约束，任务受待执行 job 数量约束，文件同时受单文件大小和插件总存储量约束。事件与任务的幂等重放不会重复占用容量。成功或失败后都会释放预留，系统按当前路径自动恢复，不存在备用队列或绕过治理的执行路径。

插件业务路由在执行前检查请求大小、响应大小、超时、速率和并发；宿主调用使用独立的速率与并发额度。额度拒绝统一返回 HTTP `429`、`plugin_quota_exceeded`、`retryable: true`、`Retry-After` 和 `X-Skoll-Quota-Resource`。插件应按该信号进行有界退避，禁止改用私有接口、直连宿主数据库或无治理的本地队列。

托管进程同时接收 `SKOLL_PLUGIN_MEMORY_LIMIT_BYTES`、`SKOLL_PLUGIN_MAX_PROCS`、`GOMEMLIMIT` 和 `GOMAXPROCS`。Windows 使用 Job Object 限制进程数与内存，Linux 使用 `RLIMIT_AS`；不支持的平台直接拒绝启动。崩溃、健康检查失败、禁用或停止都会释放进程额度，后续重启重新接受治理。

插件诊断 API 与控制中心会展示每类资源的速率、突发量、活动并发、可用令牌、预留容量、拒绝次数和最后拒绝时间。拒绝同时形成可重试的 `quota` 诊断错误和持久证据，运维人员可以据此调整唯一的 `plugin.quota` 配置。

## 安全边界

- 宿主网关只监听随机回环地址，只接受 `POST` 和受支持的 v1 operation。
- 每个请求同时校验插件生命周期凭证；数据范围调用还会校验用户 JWT。
- 插件业务请求与响应上限由 `plugin.quota` 配置；宿主协议帧仍限制为 32 MiB。未知字段、未知能力、无效事务和非回环来源直接失败。
- 网关错误只返回稳定 code，不返回内部数据库、密钥或底层错误文本。
- 插件凭证、用户 token 和 secret 明文不得进入日志、审计详情或业务响应。
- 当前只有 host-service HTTP v1，不提供远程数据库、旧 token、备用 URL 或降级路径。

## 验证

```powershell
go test ./internal/plugin ./internal/plugin/quota ./internal/plugin/hostservice -run "Quota|HostGateway|ManagedProcessLauncher" -count=1
go test -race ./internal/plugin ./internal/plugin/quota ./internal/plugin/hostservice -run "Quota|HostGateway|ManagedProcessLauncher" -count=1
go list -deps ./pkg/pluginclient
```

契约测试必须覆盖八类服务、有效与无效用户身份、事务提交与回滚、凭证撤销和非回环拒绝。
