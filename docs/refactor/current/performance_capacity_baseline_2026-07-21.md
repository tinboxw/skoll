# Skoll 容量与性能基线 2026-07-21

> 状态：H5-04 正式基线。默认语言：中文（`zh-CN`）。
> 适用范围：当前单体部署、Pharma OA 样板与业务插件基础能力的回归门禁，不是生产容量承诺或监管认证。

## 执行摘要

H5-01 至 H5-03 的数据库、分页、大列表和长任务证据已汇总为可重复执行的性能基线。当前单机环境下，50,000 行 MySQL 代表性数据集的 6 个查询均命中约定索引并满足 P95 预算；125 条浏览器夹具在每页 50 条时只挂载 24 行；4 类任务各运行 100 次，无失败、重复副作用或 goroutine 增长。allocation benchmark 与两个关键前端 chunk 也已设置明确回归阈值。

结论仅覆盖下文环境、数据规模和场景。增加数据量、并发、网络延迟或外部服务后必须重新测量，不得按线性比例推算生产容量。

## 验收环境

| 项目 | 环境 |
| --- | --- |
| 操作系统 | Microsoft Windows NT 10.0.26200.0, amd64 |
| CPU | 13th Gen Intel Core i5-13500H，Go 可见 16 个逻辑处理器 |
| Go | 1.24.1 windows/amd64 |
| Node.js | 22.14.0 |
| Vite | 5.4.21 |
| MySQL | 5.7.26，本机 loopback |
| 浏览器 | 本机 Chrome，桌面 1440x1000、移动 390x844 |
| race | H5-03 Go soak 全程启用 race detector |

## 数据集与容量

| 数据集 | 规模 | 用途 |
| --- | ---: | --- |
| Pharma OA 客户 | 10,000 | 常用列表、组织范围、负责人范围 |
| 库存余额 | 20,000 | 仓库/商品位置查询 |
| 库存预警 | 10,000 | 仪表盘聚合 |
| 报表任务 | 10,000 | 负责人队列查询 |
| 浏览器员工/客户 | 各 125 | 分页、虚拟表格、缓存、取消请求与响应式 |
| 内存服务 benchmark | 员工/客户各 1,000 | 开发/测试适配器分配与耗时边界 |
| 任务 soak | 每场景 100 次 | 导入导出、提醒、异步报表、插件事件 |

默认页面大小为 25/50/100，浏览器矩阵使用 50；lookup 调用最多读取 200 条。生产数据库列表必须使用服务端过滤、排序和分页，不能用浏览器或服务层全量扫描替代。

## 数据库查询基线

每个查询预热 10 次并执行 100 次。所有计划访问类型均为 `ref`，且命中规定索引。

| 查询 | P50 | P95 | P99 | P95 预算 | 索引 |
| --- | ---: | ---: | ---: | ---: | --- |
| 客户常用列表 | 1.7341 ms | 4.9154 ms | 5.4790 ms | 25 ms | `idx_pharma_customers_scope_status` |
| 库存预警仪表盘 | 12.9429 ms | 20.2846 ms | 22.4239 ms | 40 ms | `idx_pharma_inventory_alerts_status_type_seen` |
| 库存位置列表 | 1.6334 ms | 5.0241 ms | 5.7937 ms | 25 ms | `uk_pharma_balance_position` |
| 客户组织范围 | 1.1971 ms | 2.5645 ms | 3.1228 ms | 25 ms | `idx_pharma_customers_org_status_code` |
| 客户负责人范围 | 1.2060 ms | 2.6405 ms | 2.8340 ms | 25 ms | `idx_pharma_customers_owner_status_code` |
| 报表负责人队列 | 2.3050 ms | 4.6185 ms | 5.4282 ms | 25 ms | `idx_pharma_export_jobs_owner_status_created` |

慢查询回归定义：常用列表、库存、范围和报表队列 P95 超过 25 ms，或库存预警仪表盘 P95 超过 40 ms，即为失败；缺少规定索引或退化到 `ALL` 同样失败。

## 分页与大列表基线

- 员工、客户、产品、供应商和仓库统一使用 `items/offset/limit/total/hasMore/nextCursor/sort`。
- 125 条员工和客户数据在每页 50 条时，虚拟表格仅挂载 24 行，固定表格高度 520 px。
- 第二页稳定从第 51 条开始；返回第一页命中 10 秒缓存；快速筛选会取消旧请求。
- `zh-CN` 与 `en-US`、桌面与移动共 8 个页面检查全部通过，页面和分页控件横向溢出均为 0。
- 回归阈值：页面大小不得超过 100；50 条页面挂载行数不得超过 30；页面/分页横向溢出必须为 0；取消旧请求后不得出现错误态。

## 长任务稳定性基线

| 场景 | 成功/总数 | 重试 | P50 | P95 | P99 | 堆增长 | goroutine 增长 | 重复副作用 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 主数据导入导出 | 100/100 | 0 | 11.1640 ms | 16.6587 ms | 19.9975 ms | 78,360 B | 0 | 0 |
| 逾期提醒 | 100/100 | 100 | 10.5494 ms | 13.1011 ms | 13.7909 ms | 38,208 B | 0 | 0 |
| 异步报表导出 | 100/100 | 100 | 13.1012 ms | 17.1745 ms | 27.0794 ms | 31,248 B | 0 | 0 |
| 插件业务事件 | 100/100 | 100 | 0 ms | 0.6801 ms | 0.9548 ms | 3,520 B | 0 | 0 |

永久插件失败在第 3 次尝试后进入 1 条可观测 `dead_letter`；处理器卸载也会进入死信并返回错误。医药任务阈值为 P95 500 ms、P99 1,500 ms、堆增长 64 MiB、goroutine 增长 4；插件事件阈值为 P95 25 ms、P99 75 ms、堆增长 16 MiB、goroutine 增长 2。失败和重复副作用预算均为 0。

## 内存与分配基线

Go benchmark 使用 1,000 条内存适配器数据、每项 200 次迭代、3 个样本。表中耗时取样本中位数，内存和分配取最差样本。

| Benchmark | 中位 ns/op | 最大 ns/op | 最大 B/op | 最大 allocs/op | 回归阈值 |
| --- | ---: | ---: | ---: | ---: | --- |
| Local cache set/get | 1,428 | 2,199 | 238 | 3 | 5,000 ns；512 B；8 allocs |
| Employee paged query | 28,385,022 | 28,737,660 | 1,605,958 | 20,070 | 40 ms；2 MiB；25,000 allocs |
| Customer scoped query | 18,865,398 | 19,392,334 | 1,158,208 | 14,620 | 30 ms；1.5 MiB；18,000 allocs |

员工和客户 benchmark 测量的是内存开发/测试适配器，它会克隆和筛选 1,000 条对象；高分配值不是生产 MySQL 列表目标。生产路径以有界 SQL 查询和数据库分位数为准，但内存适配器也不得超过本表阈值，以防测试、演示和本地运行出现明显退化。

## 前端 Bundle 基线

Vite production build 转换 3,612 个模块。

| Chunk | 原始大小 | gzip | 阈值 |
| --- | ---: | ---: | --- |
| `DataTable-DYgsnTzm.js` | 48,925 B | 16,223 B | 原始 <= 64 KiB；gzip <= 20 KiB |
| `el-pagination-pSKIrJKT.js` | 11,799 B | 4,071 B | 原始 <= 16 KiB；gzip <= 6 KiB |

构建仍有既有的 Dart Sass legacy API 和 VueUse `PURE` 注释位置警告；它们不影响产物生成，但升级依赖时应重新验证。chunk 文件名含内容 hash，脚本按稳定前缀匹配。

## 已知限制

1. 本报告是单开发机、单进程、本地网络基线，没有测量多实例、容器限额、跨区网络、并发用户上限或吞吐饱和点。
2. MySQL 只覆盖 5.7.26 和 50,000 行代表性夹具；未覆盖百万级数据、PostgreSQL、慢磁盘、只读副本或连接池耗尽。
3. 浏览器矩阵只覆盖每类 125 条、Chrome 和两个视口；没有覆盖低端移动设备、弱网、Safari 或 Firefox。
4. 任务 soak 使用内存业务夹具；异步报表使用真实 goroutine，但没有接入远端对象存储、消息队列、Redis、ClickHouse 或外部通知服务。
5. 当前 soak 证明 100 次顺序任务和异步 worker 可回收，不等于 100 个并发任务容量。报表 worker 尚未形成可配置并发池，因此不得宣称无界并发安全。
6. 主数据 Excel 导出仍在进程内生成 Base64，服务层单次读取上限为 1,000 条；更大导出应转为持久化后台任务后重新定基线。
7. allocation 数值会随 Go 版本、CPU、编译器和依赖变化；升级工具链时先生成新证据，再审查阈值，不应直接放宽门禁。

## 回归阈值

| 类别 | 失败条件 |
| --- | --- |
| 数据库 | P95 超预算；未命中规定索引；访问类型退化为 `ALL` |
| 分页 | 页面大小超过 100；虚拟行超过 30/50；缓存/取消/响应式任一失败 |
| 任务 | 任一失败或重复副作用；P95/P99、堆或 goroutine 增长超阈值；永久失败未进入死信 |
| 分配 | 任一 benchmark 的最大 ns/op、B/op 或 allocs/op 超过表中阈值 |
| Bundle | DataTable 或 Pagination 的原始/gzip 大小超过表中阈值 |
| 文档 | 环境、数据集、分位数、内存、分配、bundle、限制、阈值或复现命令缺失 |

机器可读汇总位于 `evidence/h5-04/performance-baseline.json`；原始来源为 H5-01、H5-02 和 H5-03 的独立证据，禁止只修改汇总文件绕过上游验收。

## 复现命令

完整重现 allocation 与 bundle，并重新核对 H5-01 至 H5-03 证据：

```powershell
./scripts/h5-performance-baseline.ps1 -BenchmarkCount 3 -BenchmarkIterations 200
./scripts/h5-performance-doc-audit.ps1
```

上游数据库与任务基准：

```powershell
$env:SKOLL_BENCHMARK_MYSQL_DSN = 'root:root@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local'
go run ./cmd/skoll-db-benchmark --allow-write-fixtures --rows 10000 --iterations 100 --output docs/refactor/current/evidence/h5-01/mysql-query-benchmark.json
./scripts/h5-job-soak.ps1 -Iterations 100
```

常规质量门禁：

```powershell
go test ./...
go vet ./...
npm --prefix web run typecheck
npm --prefix web run build
codegraph sync .
codegraph status .
git diff --check
```

## English Summary

This baseline consolidates H5-01 through H5-03 on one Windows workstation. Six indexed MySQL queries over 50,000 representative rows passed their 25/40 ms P95 budgets. The bilingual large-list matrix passed with server pagination and 24 mounted rows per 50-row page. Four 100-iteration race-enabled job scenarios completed with zero failures, duplicate side effects, or goroutine growth; permanent plugin failure reached an observable dead letter. Allocation and DataTable/Pagination bundle budgets are enforced by `scripts/h5-performance-baseline.ps1`.

This is a regression baseline, not a production concurrency or throughput guarantee. Production sizing requires representative infrastructure, data volume, external services, network latency, concurrency, and sustained-load testing.
