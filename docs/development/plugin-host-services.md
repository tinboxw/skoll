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

插件禁用、崩溃、卸载或宿主关闭时，凭证立即失效，活动事务回滚。重新启用会签发新凭证，旧凭证不会恢复。

## 用户身份与数据范围

宿主凭证证明“哪个插件正在调用”，不能代表终端用户。处理业务 HTTP 请求时，插件必须把 Skoll 转发的用户 access token 放入调用上下文：

```go
ctx := pluginclient.WithUserToken(r.Context(), userAccessToken)
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

## 安全边界

- 宿主网关只监听随机回环地址，只接受 `POST` 和受支持的 v1 operation。
- 每个请求同时校验插件生命周期凭证；数据范围调用还会校验用户 JWT。
- 请求与响应上限为 32 MiB；未知字段、未知能力、无效事务和非回环来源直接失败。
- 网关错误只返回稳定 code，不返回内部数据库、密钥或底层错误文本。
- 插件凭证、用户 token 和 secret 明文不得进入日志、审计详情或业务响应。
- 当前只有 host-service HTTP v1，不提供远程数据库、旧 token、备用 URL 或降级路径。

## 验证

```powershell
go test ./internal/plugin -run "TestHostGateway|TestManagedProcessLauncher" -count=1
go test -race ./internal/plugin -run "TestHostGateway|TestManagedProcessLauncher" -count=1
go list -deps ./pkg/pluginclient
```

契约测试必须覆盖八类服务、有效与无效用户身份、事务提交与回滚、凭证撤销和非回环拒绝。
