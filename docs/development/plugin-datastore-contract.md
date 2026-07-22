# 插件数据存储当前契约

> 状态：BF1-01 至 BF1-04 已完成公共契约、Schema 注册、可信查询规划与事务化变更执行；插件网关和外部进程客户端尚未接入，分别由 BF1-05、BF1-06 完成。
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

### 宿主查询规划

BF1-03 的宿主规划器在执行 SQL 前完成以下处理：

- 使用请求上下文和插件自有权限资源调用 `DataScopeService`，不接受插件进程传入的可信身份或最终作用域。
- 将插件请求的租户、组织和所有者 ID 与可信作用域取交集；任一维度越权或可信作用域不完整时直接拒绝。
- 根据 Schema 注册表校验表、选择字段、过滤字段和排序字段；标识符由宿主引用，值始终使用绑定参数。
- 对 SQLite、PostgreSQL 和 MySQL 生成各自的引用符及占位符，单次规划最多产生 900 个绑定参数。
- 自动把缺失的主键追加到排序末尾，并额外读取 `version` 与 cursor 所需字段；插件只收到其显式选择的字段。
- 只使用确定性的 keyset cursor。cursor 与插件、逻辑表、完整排序和值类型绑定，不能跨表或更换排序复用。
- 排序字段必须非空。当前契约不依赖三种数据库各不相同的 NULL 默认排序规则。

规划器输出由 BF1-05 的宿主 SQL 适配器执行。当前不提供 offset 分页、原始排序表达式、SQL 片段、插件指定物理表名或方言回退。

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

### 宿主变更执行

BF1-04 的宿主执行器对每次变更应用以下规则：

- `key` 必须与 Schema 中的完整主键一致；`values` 只能包含插件声明为可变的业务字段。
- 每次变更必须解析为唯一租户、组织和所有者。可信作用域本身包含多个 ID 或全量权限时，插件必须使用 `scope.filter` 明确收窄到一个 ID。
- `insert` 由宿主写入作用域、`version = 1`、`created_at` 和 `updated_at`；插件不能覆盖这些字段。
- `update`、`delete` 和已存在记录的 `upsert` 始终使用当前 `version` 作为 SQL 条件。提供的 `expectedVersion` 不匹配或执行期间版本变化时返回 `conflict`。
- `upsert` 不存在记录时执行插入；声明 `expectedVersion` 但记录不存在时返回 `conflict`。
- 幂等键按“插件 + key”持久化，请求哈希与结果和业务变更处于同一事务。同一请求重放返回原结果，同 key 不同请求返回 `conflict`。
- 成功变更自动写审计，审计只记录表、操作、影响行、版本、资源哈希和幂等键哈希，不记录业务字段值。
- 审计失败、结果持久化失败或外层 `TransactionService` 回调失败时，业务数据、幂等结果和审计一起回滚。

当前只提供单记录原子变更。批量业务操作必须在 `TransactionService.Within` 中组合多个当前变更，不提供非事务批量接口或失败后的部分提交模式。

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
go test ./internal/plugin/datastore -count=1
go test -race ./internal/plugin/datastore -count=1
go test ./pkg/pluginsdk ./pkg/pluginclient ./internal/plugin -count=1
```

当前工程不提供旧数据 API、原始 SQL 逃生口、双 wire 格式或回退存储。
