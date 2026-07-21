# H5-01 MySQL 查询与索引基准

> 默认中文；Work Item `H5-01`；验收日期 2026-07-21。

## 环境与数据集

| 项目 | 值 |
| --- | --- |
| 数据库 | MySQL 5.7.26 / `skoll` |
| 基础夹具规模 | customers 10,000；stock_balances 20,000；inventory_alerts 10,000；report_export_jobs 10,000 |
| 总夹具 | 50,000 行 |
| 迭代 | 每个查询预热 10 次，计时 100 次 |
| 写入边界 | 仅允许 `h5-bench-*` ID，运行前后自动清理；最终残留 0 行 |
| 结果文件 | `mysql-query-benchmark.json` |

## 查询预算

| 类别 | 场景 | 命中索引 | Access | 估算行 | P50 ms | P95 ms | P99 ms | P95 预算 ms |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: |
| common_list | customer_common_list | idx_pharma_customers_scope_status | ref | 100 | 1.734 | 4.915 | 5.479 | 25 |
| dashboard | inventory_alert_dashboard | idx_pharma_inventory_alerts_status_type_seen | ref | 7,482 | 12.943 | 20.285 | 22.424 | 40 |
| inventory | stock_balance_position_list | uk_pharma_balance_position | ref | 200 | 1.633 | 5.024 | 5.794 | 25 |
| scope | customer_organization_scope | idx_pharma_customers_org_status_code | ref | 1,000 | 1.197 | 2.564 | 3.123 | 25 |
| scope | customer_owner_scope | idx_pharma_customers_owner_status_code | ref | 100 | 1.206 | 2.640 | 2.834 | 25 |
| report | report_export_owner_queue | idx_pharma_export_jobs_owner_status_created | ref | 24 | 2.305 | 4.618 | 5.428 | 25 |

六个查询全部使用命名索引且 P95 低于预算。预算是本机回归门禁，不是跨硬件生产 SLA；H5-04 将汇总容量边界和正式回归阈值。

## 索引调整

- 客户范围新增 `(organization_id, status, code)` 与 `(owner_id, status, code)`，分别服务组织范围和本人范围。
- 库存告警新增 `(status, type, last_seen_at)` 覆盖索引，将聚合 P95 从约 89ms 降至 20.285ms。
- 报表队列新增 `(owner_id, status, created_at)`，避免 owner 单列过滤后的额外排序。
- GORM model、`SchemaBaseline()`、MySQL/PostgreSQL 干净建表脚本与持久化文档保持一致。

## 复现

```powershell
$env:SKOLL_BENCHMARK_MYSQL_DSN = "root:root@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=true&loc=Local"
.\scripts\benchmark-pharma-oa-mysql.ps1
.\scripts\benchmark-pharma-oa-mysql.ps1 -Locale en-US
```

基准命令必须显式传入 `--allow-write-fixtures`；未授权写入会立即失败。脚本不会输出 DSN，失败时仍尝试清理已提交的前缀夹具。
