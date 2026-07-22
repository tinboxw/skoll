# 插件数据存储当前契约

> 状态：BF1-01 已定义 `pkg/pluginsdk` 公共类型与校验，BF1-02 已建立宿主侧 Schema 注册表；宿主适配器和外部进程客户端尚未接入，分别由 BF1-05、BF1-06 完成。
> 规则：只有当前结构化契约。插件不能获取原始 SQL、GORM、数据库连接、宿主表名或 `internal/` 实现。

## 设计边界

`DataStoreService` 只提供两个操作：

```go
type DataStoreService interface {
    Query(context.Context, DataQuery) (DataPage, error)
    Mutate(context.Context, DataMutation) (DataMutationResult, error)
}
```

- 插件提交逻辑表、显式字段、结构化过滤、稳定排序和游标分页。
- 插件提交权限和希望收窄的租户、组织、所有者范围；宿主从可信身份解析最终范围。
- `tenant_id`、`organization_id`、`owner_id` 由宿主注入，不能出现在插件变更主键、值或普通过滤条件中。
- 多次变更通过 `TransactionService.Within` 的回调上下文组成一个宿主事务，不暴露事务 ID。
- 表、字段和索引是否属于当前插件，由 BF1-02 的 schema registry 决定。

## Schema 与命名空间

插件只声明逻辑表、业务字段、主键和查询索引。宿主根据插件 ID 生成固定且不可由插件指定的 `skp_<hash>` 命名空间，并将逻辑表映射为不超过 63 字符的物理表名。插件不能读取或访问其他插件的命名空间。

每张表由宿主追加以下字段，插件不能重复声明或写入：

| 字段 | 用途 |
| --- | --- |
| `tenant_id` | 可信租户作用域 |
| `organization_id` | 可信组织作用域 |
| `owner_id` | 可信数据所有者作用域 |
| `version` | 乐观并发控制 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

当前 Schema 限制如下：

- 每个插件最多 128 张表，每张表最多 128 个业务字段和 32 个索引。
- 逻辑表名最长 42 字符；表名和字段名必须是小写 snake_case，且不能使用宿主、系统数据库或保留前缀。
- 主键必须由 1 至 4 个插件字段组成，字段必须非空且不可变。
- 单个索引最多包含 8 个已声明且可查询的字段；JSON 和二进制字段不能用于过滤、排序或索引。
- 注册结果使用防篡改副本；注销后，表与命名空间解析立即失效。

BF1-02 只负责 Schema 所有权和访问白名单。Schema 与插件清单、安装迁移、禁用及卸载策略的绑定由 BF1-07 完成，插件当前不能绕过生命周期直接注册数据库结构。

## 类型化值

所有值使用 `{type, value}`，避免 JSON number 和 SQL 方言造成精度或类型漂移。

| `type` | `value` 规则 |
| --- | --- |
| `null` | 必须为空 |
| `string` | UTF-8 字符串，最大 1 MiB |
| `integer` | canonical base-10 `int64`，不接受 `01` |
| `decimal` | 普通十进制定点字符串，不接受指数格式 |
| `boolean` | 只能是 `true` 或 `false` |
| `timestamp` | RFC3339Nano |
| `bytes` | 标准 Base64，解码后最大 1 MiB |
| `json` | 有效 JSON 文本，最大 1 MiB |

## 查询

查询必须显式选择字段、至少提供一个稳定排序，并使用 `1..200` 的 limit。字段和表标识只允许小写字母开头的 snake_case。

```go
page, err := store.Query(ctx, pluginsdk.DataQuery{
    Table:  "products",
    Fields: []string{"id", "code", "name", "updated_at"},
    Scope: pluginsdk.DataScopeIntent{
        Permission: pluginsdk.Permission{Resource: "medical_oa.product", Action: "read"},
        Filter: pluginsdk.ScopeFilter{TenantIDs: []string{"tenant-1"}},
    },
    Filter: &pluginsdk.DataFilter{
        Field: "name", Operator: pluginsdk.DataOperatorContains,
        Value: &pluginsdk.DataValue{Type: pluginsdk.DataValueString, Value: "tablet"},
    },
    Sort: []pluginsdk.DataSort{{Field: "updated_at", Direction: pluginsdk.DataSortDescending}},
    Page: pluginsdk.DataPageRequest{Limit: 50},
})
```

过滤器每个节点只能是一个 leaf、`all` 或 `any`。支持 `eq`、`ne`、`lt`、`lte`、`gt`、`gte`、`in`、`not_in`、`contains`、`prefix`、`is_null`、`not_null`；最大深度 8、节点 64、集合值 100。

## 变更

每次变更只处理一条记录，并强制主键与幂等键。`insert`、`update`、`upsert` 需要 values；`delete` 禁止 values。`expectedVersion` 用于 update、upsert、delete 的乐观并发控制。

```go
result, err := store.Mutate(ctx, pluginsdk.DataMutation{
    Table:     "products",
    Operation: pluginsdk.DataMutationInsert,
    Scope: pluginsdk.DataScopeIntent{
        Permission: pluginsdk.Permission{Resource: "medical_oa.product", Action: "create"},
    },
    Key: map[string]pluginsdk.DataValue{
        "id": {Type: pluginsdk.DataValueString, Value: "product-1"},
    },
    Values: map[string]pluginsdk.DataValue{
        "name": {Type: pluginsdk.DataValueString, Value: "Tablet"},
    },
    Returning:      []string{"id", "name", "updated_at"},
    IdempotencyKey: "product-1.create",
})
```

插件不能把主键同时放入 values，也不能写入宿主作用域字段。单次变更的 values 最多 128 个，复合主键最多 4 个字段。

## 错误

`DataStoreError` 提供稳定 `code`、字段路径、消息和 `retryable`：

| Code | 含义 |
| --- | --- |
| `invalid_request` | 类型、标识、过滤、分页或操作语义无效 |
| `forbidden` | 身份、权限、作用域或插件 namespace 被拒绝 |
| `not_found` | 当前作用域内记录不存在 |
| `conflict` | 幂等、唯一约束或乐观版本冲突 |
| `limit_exceeded` | 请求、结果或资源超过当前限制 |
| `unsupported` | schema 或数据库方言不支持该结构化表达式 |
| `unavailable` | 当前数据服务不可用，可依据 `retryable` 决定重试 |

调用前可执行 `DataQuery.Validate` 或 `DataMutation.Validate`；服务端仍必须重复校验，不能信任插件进程。

## 验证

```powershell
go test ./pkg/pluginsdk -count=1
go test -race ./pkg/pluginsdk -count=1
go test ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin -count=1
```

当前工程不提供旧数据 API、原始 SQL 逃生口、双 wire 格式或回退存储。
