# 医药 OA 持久化契约

> 状态：H2-01 schema 基线。默认语言：简体中文。
> 实现入口：`internal/repository/pharmaoa/`。

## 目标与边界

本契约冻结 Pharma OA 从内存样板迁移到 SQL repository 时的领域归属、表命名、唯一约束、查询索引、审计列和事务边界。H2-01 不替换现有 service，不增加 HTTP API，也不实现旧数据结构兼容。

生产目标数据库为 MySQL 和 PostgreSQL；SQLite 用于 repository/schema 契约测试。三个方言必须共享同一份 `SchemaBaseline()`，不允许各自维护不同业务约束。

## Repository Ports

| 端口 | 聚合与职责 | 后续实现项 |
| --- | --- | --- |
| `EmployeeRepository` | 员工档案、状态、证照 JSON 与组织查询 | H2-02 |
| `ProductRepository` | 药品主数据、批准文号、温控属性 | H2-02 |
| `SupplierRepository` | 供应商、联系人、资质与附件 metadata | H2-02 |
| `CustomerRepository` | 客户、负责人/组织范围、资质与附件 metadata | H2-02 |
| `WarehouseRepository` | 仓库、区域和库位聚合 | H2-02 |
| `InventoryRepository` | 批次、余额、锁与不可变流水；提供事务回调 | H2-03 |
| `PurchaseRepository` / `PurchaseInboundRepository` | 采购申请、订单和入库 | H2-03 |
| `SalesRepository` | 销售订单和出库 | H2-03 |
| Contract/Complaint/Recall/CRM ports | 合同、投诉、召回、跟进与销售机会 | H2-04 |

Repository 接收 domain entity，不向 service 暴露 GORM model。列表统一使用 `ListFilter` 的分页、关键字、状态和可信组织范围；具体 SQL 实现不能从请求参数扩大组织范围。

## Schema 归属

所有表使用 `pharma_oa_` namespace。完整机器可读清单由 `SchemaBaseline()` 提供，当前覆盖 29 张聚合根、明细或作业表。

| 领域 | 核心表 | 关键约束与索引 |
| --- | --- | --- |
| 主数据 | employees, products, suppliers, customers, warehouses | code 唯一；药品批准文号唯一；客户按 organization/owner/status 查询 |
| 库存 | stock_batches, stock_balances, stock_ledger, stock_locks, stocktakes, transfers | 批次按 product+batch 唯一；余额按完整库位唯一；ledger idempotency 唯一且不可更新 |
| 采购 | purchase_requests, purchase_request_lines, purchase_orders, purchase_inbounds | 单号唯一；申请与订单一对一；入库 idempotency 唯一 |
| 销售 | sales_orders, sales_outbounds | 单号唯一；出库 idempotency 唯一；客户/组织范围有索引 |
| 协同合规 | announcements, contracts, qualifications, quality_complaints, drug_recalls, cold_chain_records | 合同/投诉/召回单号唯一；资质和合同到期时间可索引扫描；冷链记录不可变 |
| CRM 财务 | customer_follow_ups, sales_opportunities, payment_plans, invoice_records | owner+organization+status/stage 范围索引；机会/发票单号唯一 |
| 作业 | inventory_alert_jobs, report_export_jobs | idempotency key 唯一；status+created_at 支持重试与清理扫描 |

数组型附件只保存 file metadata ID；文件内容仍由 object store 管理。当前不需要独立查询的 contacts、attachments、stage history 和 task snapshot 使用 JSON，后续若出现可证明的查询需求再以新的当前 migration 正规化，不保留双写路径。

## 字段规则

- ID：使用 `VARCHAR(64)`/字符串 `shared.ID`，不引入数据库自增业务 ID。
- 金额：SQL 实现使用 `DECIMAL(18,2)`；禁止用浮点列保存最终金额。
- 数量：库存数量使用整数；采购数量沿当前 domain 语义使用 `DECIMAL(18,4)`。
- 时间：MySQL 使用 UTC `DATETIME(3)`，PostgreSQL 使用 `TIMESTAMPTZ`，SQLite 测试使用 UTC timestamp 文本/driver time。
- 审计：可变表包含 `created_at`、`updated_at`、`created_by`、`updated_by`；不可变表也保留创建时间和创建者，禁止 update/delete repository 方法。
- 并发：`stock_balances.version` 用于乐观锁；数据库实现还必须在余额更新事务中执行行锁或等价原子条件更新。

## 事务边界

| 事务组 | 必须同一事务提交的写入 |
| --- | --- |
| `inventory` | batch、balance、ledger，以及 inbound/outbound/stocktake/transfer 状态 |
| `purchase` | 采购申请审批状态与生成的采购订单 |
| `sales` | 销售订单状态与出库准备；实际扣减进入 inventory 事务 |
| `compliance` | 投诉/召回状态与任务快照；外部通知在提交后发布 |
| `jobs` | 作业 attempt/status/error 与 idempotency key |

领域事件、通知和审计副作用必须在事务提交后发布，或使用后续 outbox；不能在事务失败时留下“成功”事件。

## Migration 与回滚

| 项目 | 规则 |
| --- | --- |
| Source | 内存样板，无持久化业务数据 |
| Target | `pharma_oa` schema baseline v1 |
| Empty state | 按 `BuildMigrationPlan(dialect).Up` 顺序创建 |
| Existing core data | 只新增 `pharma_oa_` 表，不改 core 表 |
| Partially created | 实现脚本必须幂等；已存在对象需校验结构而不是静默忽略冲突 |
| Rollback | `Down` 逆序 drop 是破坏性步骤，只允许空环境或失败的首次安装 |
| Uninstall | 固定 `retain`；正常插件卸载不删除业务数据 |
| Production recovery | 已写入业务数据后使用前向修复 migration 或备份恢复，不执行 destructive down |
| Downtime | 初始建表不要求停机；后续大表索引按各数据库在线 DDL 能力另行计划 |

本批次不做旧表、旧字段或旧插件 migration 兼容。H2-02 至 H2-04 必须从此基线生成各方言 migration，并在接线前通过空库、部分状态和 restart 测试。

## 验证

```powershell
go test ./internal/repository/pharmaoa -count=1
go test ./internal/repository/pharmaoa -run TestMigrationPlanSupportsAllDialectsAndReversesOrder -count=1
go test ./internal/plugin -run TestPharmaOAPluginManifestCoversIndustrySkeleton -count=1
```
