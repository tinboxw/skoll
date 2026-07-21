# 插件宿主服务契约

本文说明进程内 Go 业务插件如何使用 Skoll 提供的事务、可信数据范围、文件、审计、配置与密钥能力。公开契约位于 `pkg/pluginsdk`，宿主适配器位于 `internal/plugin/hostservice`。

## 基本规则

1. 插件只依赖 `pkg/pluginsdk` 中的公开类型，不依赖 `internal/service`、`internal/repository` 或数据库适配器。
2. 多步写操作必须通过 `HostServices.Transactions.Within` 建立事务边界。
3. 数据访问范围必须通过 `HostServices.DataScopes.Resolve` 获取，不能使用请求参数中的用户、组织或租户信息作为授权依据。
4. 缺少宿主服务、事务回调、可信身份或权限资源时直接失败，不提供兼容路径或降级执行。
5. 请求筛选只能调用 `ScopePredicate.Constrain` 收窄可信范围，不能自行合并或扩大范围。
6. 配置不得承载密钥；密码、令牌、凭据、Cookie、会话和授权头必须通过 `HostServices.Secrets` 存取。

## 依赖注入

插件依赖对象应显式接收并校验 `pluginsdk.HostServices`：

```go
type Dependencies struct {
    Host pluginsdk.HostServices
}

func NewBackend(deps Dependencies) (http.Handler, error) {
    if err := deps.Host.Validate(); err != nil {
        return nil, err
    }
    // Build plugin handlers and services.
}
```

宿主负责把 Unit of Work、RBAC 服务和组织仓储适配成公开端口。插件不构造这些适配器。

## 事务服务

`Within` 将同一个事务上下文传给回调。插件仓储必须使用 `tx.Context()` 执行全部参与事务的数据库操作：

```go
err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
    if err := orders.Create(tx.Context(), order); err != nil {
        return err
    }
    return inventory.Reserve(tx.Context(), reservation)
})
```

任一操作返回错误时，整个回调回滚。嵌套事务复用当前事务上下文，不启动独立提交。

宿主 SQL 仓储从上下文解析事务连接；直接使用全局数据库连接会绕过事务，不能作为插件仓储实现。

## 可信数据范围

插件只传权限资源与动作，身份从宿主已经验证的请求上下文读取：

```go
scope, err := host.DataScopes.Resolve(ctx, pluginsdk.Permission{
    Resource: "pharma_oa.customer",
    Action:   "read",
})
if err != nil {
    return err
}
```

`ScopePredicate` 同时约束租户、数据所有者和组织。三个维度按 AND 关系判断，任何一个维度不匹配都拒绝访问。

请求中的查询条件只能收窄范围：

```go
scope = scope.Constrain(pluginsdk.ScopeFilter{
    OwnerIDs:        requestedOwnerIDs,
    OrganizationIDs: requestedOrganizationIDs,
})
if scope.Denied() {
    return errAccessDenied
}
```

`self` 范围固定为已认证主体；普通用户的 `all` 范围仍受可信租户根及其组织树约束；只有 `super_admin` 可以获得跨租户全范围。

## 文件服务

`HostServices.Files` 提供 `Store`、`List`、`Get`、`Download` 和 `Delete`。宿主固定以下边界，插件不能覆盖：

- 对象键统一写入 `plugins/{pluginID}/` 命名空间。
- 来源固定为当前插件，读取、下载和删除都会再次校验 `Source.PluginID`。
- 所有者与审计主体来自可信 JWT；后台任务使用绑定的插件主体。
- 宿主计算大小、MIME 与 SHA-256，不接受插件声明值。
- 只允许 `private` 和 `plugin_asset` 可见性；可执行文件、路径穿越和超限内容直接拒绝。

```go
object, err := host.Files.Store(ctx, pluginsdk.FileWrite{
    Key:        "reports/monthly.csv",
    Name:       "monthly.csv",
    Content:    content,
    Visibility: pluginsdk.FileVisibilityPrivate,
})
```

业务附件如果先通过宿主 HTTP 文件接口上传，必须声明当前插件的 `sourcePluginId`；否则插件无权读取该对象。

## 审计服务

`HostServices.Audit.Record` 自动绑定可信主体，并把动作和资源分别限制在 `plugin.{pluginID}.` 与 `plugin:{pluginID}:` 命名空间。业务请求中的用户 ID 只能作为附加证据，不能替代宿主身份。

审计详情限制为 64 KiB，并递归脱敏密码、密钥、令牌、凭据、授权头、Cookie 和会话字段。插件不得直接写宿主审计仓储，也不得清理审计历史。

## 配置与密钥

`HostServices.Config` 读取和整体替换当前插件配置。写入前必须通过当前 manifest 的 `configSchema` 校验；manifest 声明了来源但 Schema 无法加载时直接失败，不跳过校验。返回值始终执行递归脱敏。

`HostServices.Secrets` 仅操作当前插件的私有设置命名空间。密钥名称经哈希隔离，值使用 AES-256-GCM 加密保存；审计记录只包含密钥名称，不包含明文。宿主从主密钥派生插件密钥，主密钥不足安全长度时插件宿主构造失败。

```go
if err := host.Secrets.Set(ctx, "erp.api_token", token); err != nil {
    return err
}
token, err := host.Secrets.Get(ctx, "erp.api_token")
```

## 契约测试要求

每个业务插件至少覆盖以下场景：

- 多表写入中途失败后不存在残留记录。
- 请求伪造其他租户、所有者或组织时，`Constrain` 返回拒绝或只保留交集。
- 缺少 JWT 可信身份时数据范围解析失败。
- `self` 范围忽略服务决定或请求携带的其他用户 ID。
- 插件不能读取、下载或删除其他插件的文件，文件键不能逃逸插件命名空间。
- 审计主体来自可信上下文，详情中的嵌套密钥和授权数据均已脱敏。
- 配置中的敏感键被拒绝，manifest Schema 不可用时不保存配置。
- 密钥密文不等于明文，审计与配置读取结果不包含密钥值。
- 插件构造时缺少任一 Host Service 均失败。

参考测试：`internal/plugin/hostservice/*_test.go`、`internal/plugin/pharmaoa/host_adapters_test.go` 和 `pkg/pluginsdk/*_test.go`。
