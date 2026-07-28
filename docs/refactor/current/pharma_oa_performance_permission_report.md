# 医药 OA 性能与权限验收报告

> Work Item: `F12-04`
> 日期：2026-07-17
> 范围：大列表分页、常用查询、客户数据范围、审批权限、库存权限和风险清单。

## 性能样本

验收样本在进程内服务中创建 600 条员工和 600 条客户记录，验证：

- 默认分页为 50 条，单页上限为 200 条，负偏移归零。
- 员工关键字与状态组合查询返回唯一目标记录。
- 客户区域与负责人数据范围组合查询不泄露其他负责人的客户。
- 代表性查询组合耗时必须小于 2 秒；本次实测约 6.10 ms。

1000 条记录基准样本（Windows amd64，13th Gen Intel Core i5-13500H，100 次固定迭代）：

| 场景 | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| 员工分页关键字/状态查询 | 1,301,641 | 232,453 | 1,003 |
| 客户负责人/区域范围查询 | 1,094,226 | 172,257 | 1,504 |

这些数字是当前进程内实现的回归基线，不等同于生产数据库容量承诺。基准入口为 `tests/benchmark/pharma_oa_benchmark_test.go`。

## 权限矩阵

| 能力 | 无令牌 | 无权限用户 | 对应权限用户 | `super_admin` | 数据范围 |
| --- | --- | --- | --- | --- | --- |
| 插件管理与插件业务 API | `401` | 按路由策略判定 | 可访问 | 绕过 RBAC | 不适用 |
| 插件页面与静态资源 | 公开 | 公开 | 公开 | 公开 | 只允许 `page`/`assets` 路径 |
| 采购申请审批/驳回 | `401` | `403` | `pharma_oa.purchase.approve/reject` | 可访问 | actor 取 JWT subject |
| 合同审批/驳回 | `401` | `403` | `pharma_oa.contract.approve/reject` | 可访问 | actor 取 JWT subject |
| 质量投诉处理/驳回 | `401` | `403` | `pharma_oa.quality_complaint.resolve/reject` | 可访问 | actor 取 JWT subject |
| 采购入库 | `401` | `403` | `pharma_oa.inbound.read/create` | 可访问 | actor 取 JWT subject |
| 销售出库 | `401` | `403` | `pharma_oa.sales.outbound.read/create` | 可访问 | actor 取 JWT subject |
| 盘点审批/驳回 | `401` | `403` | `pharma_oa.stocktake.*` | 可访问 | actor 取 JWT subject |
| 库存调拨 | `401` | `403` | `pharma_oa.stock_transfer.read/create` | 可访问 | actor 取 JWT subject |
| 员工/客户列表与维护 | `401` | `403` | 对应 `pharma_oa.employee.*` / `pharma_oa.customer.*` | 可访问 | 普通用户强制绑定 JWT subject；超级管理员全量 |

权限拒绝继续写入 `system.security.deny` 审计事件；请求体或查询参数中的 `actorId`、`ownerId`、`organizationId`、`includeAll` 不能覆盖已认证身份。

## 风险清单

| 等级 | 状态 | 风险与处置 |
| --- | --- | --- |
| 高 | 已关闭 | `/v1/plugins` 公开前缀曾放行其下管理与业务 API；现仅插件清单、页面和静态资源公开，管理与业务 API 必须认证。 |
| 高 | 已关闭 | 审批、入出库、盘点和调拨曾只有前端权限声明；现主路由与插件别名路由均执行后端 RBAC。 |
| 高 | 已关闭 | 客户范围和 actor 可由请求参数冒用；现已认证请求以 JWT subject/role 为唯一身份来源。 |
| 高 | 待办 | 本项之外的医药 OA 非关键模块仍需逐步加入后端细粒度路由权限映射；在完成前不得将前端菜单可见性视为后端授权。 |
| 中 | 待办 | 当前列表使用进程内全量筛选与排序；迁移到持久化仓储时需为状态、编码、负责人、组织和区域建立索引并重新采样。 |
| 中 | 待办 | 当前 JWT 不携带组织树声明，普通用户客户范围按本人负责人收紧；组织级共享需先建立受信任的组织声明与数据范围策略。 |
| 低 | 已接受 | 插件 `page` 与 `assets` 是公开展示入口；不得在静态文件中嵌入密钥或受限业务数据。 |

## 验收命令

```powershell
.\scripts\smoke-pharma-oa-performance-permission.ps1
.\scripts\smoke-pharma-oa-performance-permission.ps1 -Locale en-US
go test ./...
go vet ./...
```

本项无 migration 或 seed 变更。OpenAPI 双份契约已同步分页参数、JWT 数据范围语义和新增 `403` 响应；前端现有权限 key 不变。
