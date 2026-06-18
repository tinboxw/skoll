# 旧文档吸收记录

> 目的: 说明旧文档中哪些内容进入当前执行计划，哪些仅保留归档。旧文档不再作为任务执行入口。

## 已归档目录

旧计划类文档已移动到:

```text
docs/archive/legacy-plans/
```

## 吸收内容

| 来源文档 | 吸收到 | 保留内容 | 不再采用 |
|---|---|---|---|
| `archive/legacy-plans/quality_baseline_2026-06-09.md` | `task_board.md` M0/M7 | 质量门禁命令、覆盖率缺口、前端 build/typecheck、fixture 副作用 | 旧日期状态不作为当前结果 |
| `archive/m15_quality_gate_report.md` | `task_board.md` M0/M7 | go test、coverage、benchmark 采样方式和覆盖率提升方向 | 旧覆盖率数值不作为当前基线 |
| `archive/audit-e2e-checklist.md` | `task_board.md` M2 | 审计 smoke、操作场景、UI 验收、导出验收 | 旧 action 名称只作参考 |
| `archive/plugin_platform_m12_m14_roadmap.md` | `task_board.md` M6/M7 | 插件发布任务、灰度/回滚、门禁、观测、无旧路径约束 | 旧接口草案不直接作为最终契约 |
| `archive/legacy-plans/PLUGIN_SECURITY_DESIGN.md` | `task_board.md` M6 | 签名、风险分级、安装决策审计、安全报告 | 旧宽松策略、warn 后继续安装、旧行为保留方案均不采用 |
| `archive/legacy-plans/plugin_platform_baseline_v1.md` | `task_board.md` M6 | manifest、权限命名、审计、数据边界、插件独立交付 | 版本矩阵只作为发布说明素材，不作为基础建设任务 |
| `archive/legacy-plans/long_term_roadmap.md` | `README.md` 与 `task_board.md` | Admin 骨架、生成器、插件平台、运维可观测、开源生态阶段 | 旧 S0-S6 状态不作为当前执行状态 |
| `archive/legacy-plans/open_admin_baseline_plan.md` | `architecture_and_execution_plan.md` | 动态菜单、权限、字典、文件、代码生成、质量门禁主线 | 旧“下一步计划”不再单独执行 |
| `gin-vue-admin_analysis.md` | `task_board.md` M1-M7 | GVA 功能面: 权限、菜单、审计、文件、字典、生成器、插件、质量 | 不复制 GVA 实现和许可证受限内容 |

## 当前事实来源

| 主题 | 当前文档 |
|---|---|
| 架构模型与治理原则 | `docs/refactor/architecture_and_execution_plan.md` |
| 可执行任务顺序 | `docs/refactor/task_board.md` |
| 验收记录 | `docs/refactor/acceptance_log.md` |
| GVA 对标分析 | `docs/refactor/gin-vue-admin_analysis.md` |
| 文档入口 | `docs/refactor/README.md` |
