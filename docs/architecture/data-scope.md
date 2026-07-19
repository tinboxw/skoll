# 可信组织数据范围

## 决策来源

数据范围由服务端组合以下可信来源生成：

1. JWT 中签名的用户 ID、当前组织 ID、组织路径和角色集合。
2. RBAC Binding 与 PolicyRule 对当前资源和动作的授权范围。
3. OrganizationRepository 中的当前组织及其后代组织。

HTTP 查询参数或请求体中的 `ownerId`、`organizationId`、`includeAll` 不参与授权决策。业务字段仍可用于描述记录归属，但创建和更新后的归属也必须落在已解析的数据范围内。

## 范围矩阵

| RBAC 范围 | 仓储谓词 | 写入约束 |
| --- | --- | --- |
| `self` | `owner_id = JWT.sub` | 记录负责人必须是当前用户 |
| `department` | `organization_id = JWT.organizationId` | 记录组织必须是当前组织 |
| `department_tree` | `organization_id IN (当前组织及全部后代)` | 记录组织必须位于当前组织树 |
| `all` | 无范围谓词 | 允许操作全部记录；超级管理员显式使用此范围 |

`custom` 不属于当前正式范围。若 RBAC 授予无法解析的范围，解析器按拒绝处理。

## 失败关闭

- 缺少签名身份、资源、动作或所需组织时拒绝解析。
- RBAC 未授权或命中 deny 规则时拒绝解析。
- 令牌中的组织已删除时拒绝组织或组织树范围。
- 组织树仅从组织仓储计算，不接受调用方提供后代 ID。
- 列表查询与写入目标使用同一范围判断，防止读取受限但更新可越界。

## 实现位置

- `internal/service/rbac/service_impl.go`：可信范围解析和组织树展开。
- `internal/service/user/service_impl.go`：用户仓储范围谓词。
- `internal/service/pharmaoa/customer_service.go`：客户读写范围判断。
- `internal/repository/pharmaoa`、`internal/store/sql/gormrepo`：内存和 SQL 多组织谓词。

`GET /v1/rbac/data-scope?resource=<resource>&action=<action>` 仅返回当前 JWT 主体经服务端解析后的有效范围，供前端展示范围标识并收敛可选组织。该接口不接受主体、负责人、组织或全量开关，因此不能扩大授权。

该自省接口必须携带有效 JWT，但不额外要求 `permission.read` 管理权限；处理器会按请求中的目标资源和动作执行 RBAC 范围解析，未获目标权限时返回拒绝结果。

H3-03 新增上述只读 HTTP/OpenAPI 契约和中英文前端资源；不新增权限键、审计动作、migration 或 seed。客户归属字段保持业务语义，不再具有授权语义。
