# 文档吸收与归档记录

> 目的: 说明旧文档和阶段过程文档中哪些内容进入当前执行文档，哪些只保留归档。归档文档不再作为任务执行入口。

## 当前事实来源

| 主题 | 当前文档 |
| --- | --- |
| 文档入口 | `docs/README.md`、`docs/refactor/README.md` |
| 架构模型与治理原则 | `docs/refactor/architecture_and_execution_plan.md` |
| 可执行任务顺序 | `docs/refactor/task_board.md` |
| 最小 Work Item | `docs/refactor/work_items.md` |
| 验收记录 | `docs/refactor/acceptance_log.md` |
| 质量基线 | `docs/refactor/quality_baseline.md` |
| 前端基础规范 | `docs/refactor/frontend-foundation.md` |
| 前端质量与性能 | `docs/refactor/frontend-quality-performance.md` |
| 里程碑与尾盘收口 | `docs/refactor/milestone-closeout.md` |

## 归档目录

| 目录 | 用途 |
| --- | --- |
| `docs/archive/legacy-plans/` | 旧计划、旧路线图、旧安全设计，仅作历史参考 |
| `docs/archive/refactor-2026-06-19/` | 本轮重构拆分出的阶段过程文档、FE0-FE6 细节文档、M2 验收材料、巡检和尾盘记录 |

## 合并映射

| 原始材料 | 已合并到 | 保留内容 |
| --- | --- | --- |
| `archive/legacy-plans/quality_baseline_2026-06-09.md`、`archive/m15_quality_gate_report.md` | `quality_baseline.md`、`milestone-closeout.md` | 质量门禁、覆盖率缺口、前后端命令基线 |
| `archive/audit-e2e-checklist.md`、`m2_*` 文档 | `milestone-closeout.md`、`acceptance_log.md` | 审计 smoke、导出、OpenAPI/client 同步、M2 手工验收 |
| `archive/plugin_platform_m12_m14_roadmap.md`、`archive/legacy-plans/PLUGIN_SECURITY_DESIGN.md` | `architecture_and_execution_plan.md`、`task_board.md` | 插件发布、安全、签名、风险、生命周期和审计方向 |
| `archive/legacy-plans/long_term_roadmap.md`、`archive/legacy-plans/open_admin_baseline_plan.md` | `architecture_and_execution_plan.md`、`task_board.md` | Admin 骨架、生成器、插件平台、运维可观测和开源生态阶段 |
| `archive/legacy-plans/iteration_plan.md`、`archive/legacy-plans/feature_implementation_status.md` | `work_items.md`、`task_board.md` | 分阶段推进节奏、短周期拆分、缺失能力盘点 |
| `archive/legacy-plans/plugin_ui_optimization_plan.md`、`frontend_experience_plan.md`、`fe0_*` 到 `fe4_*` | `frontend-foundation.md` | 前端体验、视觉规范、页面架构、核心页面、插件门户体验 |
| `fe5_*`、`fe6_*` | `frontend-quality-performance.md` | 前端测试、浏览器验收、响应式、性能基线和回归清单 |
| `progress_inspection_2026-06-19.md`、`task_adjustment_proposals_2026-06-19.md`、`tail_threshold_closeout_2026-06-19.md` | `milestone-closeout.md` | 进度巡检、尾盘阈值、任务微调和发布前收口建议 |
| `gin-vue-admin_analysis.md` | `architecture_and_execution_plan.md`、`task_board.md`、`work_items.md` | GVA 功能面: 权限、菜单、审计、文件、字典、生成器、插件、质量 |

## 不再采用

1. 旧接口、旧数据结构、旧插件格式、旧页面路径的兼容方案。
2. 旧计划中的阶段编号、完成率和过期状态。
3. 静态 UI 草案作为验收通过依据。
4. GVA 实现细节或许可证受限内容的直接复制。

## 维护规则

1. 主目录只保留当前有效文档。
2. 过程文档先吸收摘要，再归档原文。
3. 若归档文档仍需被执行，必须先把有效内容摘录到当前主文档。
4. `acceptance_log.md` 是历史证据，内部引用当时文件路径时不批量改写；新任务应引用当前主文档。
