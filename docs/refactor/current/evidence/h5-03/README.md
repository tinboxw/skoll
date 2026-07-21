# H5-03 长任务、重试与资源稳定性证据

## 验收结论

在 Windows/amd64、Go 1.24.1 和 race detector 开启的环境中，分别执行 100 次主数据导入导出、逾期提醒、异步报表导出和插件业务事件场景。全部场景成功，失败数、重复副作用与 goroutine 增量均为 0；预期的临时失败均可观测并成功重试，永久插件失败进入 1 条 `dead_letter` 终态。

## 指标摘要

| 场景 | 次数 | 重试 | P50 | P95 | P99 | 堆增长 | goroutine 增长 | 重复副作用 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 主数据导入导出与重复提交 | 100 | 0 | 11.1640 ms | 16.6587 ms | 19.9975 ms | 78,360 B | 0 | 0 |
| 逾期提醒失败重试与通知幂等 | 100 | 100 | 10.5494 ms | 13.1011 ms | 13.7909 ms | 38,208 B | 0 | 0 |
| 异步报表导出失败重试与文件幂等 | 100 | 100 | 13.1012 ms | 17.1745 ms | 27.0794 ms | 31,248 B | 0 | 0 |
| 插件业务事件重试与副作用幂等 | 100 | 100 | 0 ms | 0.6801 ms | 0.9548 ms | 3,520 B | 0 | 0 |

阈值：医药任务 P95 不超过 500 ms、P99 不超过 1,500 ms、堆增长不超过 64 MiB、goroutine 增长不超过 4；插件事件 P95 不超过 25 ms、P99 不超过 75 ms、堆增长不超过 16 MiB、goroutine 增长不超过 2。所有场景允许的失败和重复副作用均为 0。

## 证据文件

- `pharma-job-soak.json`：导入、导出、提醒和报表的机器可读指标。
- `business-event-soak.json`：插件业务事件重试、死信和资源指标。
- `h5-03-soak-summary.json`：两组报告的聚合摘要。

## 重现命令

```powershell
./scripts/h5-job-soak.ps1 -Iterations 100
./scripts/h5-job-metrics-audit.ps1 -ReportPath @(
  './docs/refactor/current/evidence/h5-03/pharma-job-soak.json',
  './docs/refactor/current/evidence/h5-03/business-event-soak.json'
)
```

## 影响声明

- API/OpenAPI：无 HTTP 契约变更。
- 权限：无权限键或菜单变更。
- 审计：继续复用提醒、报表既有审计动作；事件死信为内部重试记录状态，不新增审计 API。
- Migration/seed：无数据库结构、migration 或产品 seed 变更；soak 使用内存验收夹具。
- 多语言：脚本与证据固定记录默认 locale `zh-CN`；本项无前端可见文本。

## English Summary

All four 100-iteration race-enabled soak scenarios passed with zero failures, duplicate side effects, or goroutine growth. Transient reminder, report, and plugin-event failures retried successfully; a permanent plugin failure reached one observable `dead_letter` terminal record.
