# H5-04 容量与性能基线证据

- `performance-baseline.json` 汇总 H5-01 数据库查询、H5-02 大列表、H5-03 任务稳定性，并记录 3 个 allocation benchmark 和 2 个关键前端 chunk 的阈值结果。
- 正式中文报告与英文摘要位于 `../../performance_capacity_baseline_2026-07-21.md`。
- 复现：`./scripts/h5-performance-baseline.ps1 -BenchmarkCount 3 -BenchmarkIterations 200`。
- 文档审计：`./scripts/h5-performance-doc-audit.ps1`。

本证据是当前单机环境的回归基线，不代表生产并发或吞吐承诺。
