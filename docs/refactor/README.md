# Skoll 重构执行入口

> 状态: 当前重构批次已收口  
> 最后更新: 2026-06-19  
> 适用范围: Skoll 开源基础建设重构

## 执行原则

1. `docs/refactor` 是当前重构批次的唯一执行入口。
2. Skoll 当前处于开源基础建设阶段，不做旧接口、旧数据结构、旧插件格式、旧页面路径的兼容方案。
3. 架构模型采用 Modular Monolith + Clean/Hexagonal boundaries + Tactical DDD，普通 CRUD 保持简单、可生成。
4. 每个 Work Item 必须有验收要求；验收失败必须返工。
5. 每完成一个任务并通过验收后，提交一次代码。
6. 主目录只保留当前有效的规范、索引、任务和验收文档；过程文档归档到 `docs/archive/refactor-2026-06-19/`。

## 当前主目录

```text
docs/refactor/
├── README.md                         # 本文档，重构执行入口
├── architecture_and_execution_plan.md # 架构模型、治理原则、里程碑计划
├── task_board.md                     # 父任务表和里程碑级状态
├── work_items.md                     # 最小执行、验收、提交单元
├── acceptance_log.md                 # 验收记录
├── quality_baseline.md               # M0 质量基线
├── source_map.md                     # 旧文档吸收与归档说明
├── frontend-foundation.md            # FE0-FE4 前端体验、视觉、架构、页面与插件门户规范
├── frontend-quality-performance.md   # FE5-FE6 测试、浏览器验收、响应式、性能基线
└── milestone-closeout.md             # M2、FE0-FE6、尾盘优化和发布前收口摘要
```

## 执行与验收入口

| 文档 | 用途 |
| --- | --- |
| [architecture_and_execution_plan.md](architecture_and_execution_plan.md) | 架构选择、治理规则、里程碑设计 |
| [task_board.md](task_board.md) | 父任务和阶段状态 |
| [work_items.md](work_items.md) | 最小执行单元，当前批次为 `170/170/0` |
| [next_work_items.md](next_work_items.md) | 下一批 M3-M7 候选最小任务池 |
| [acceptance_log.md](acceptance_log.md) | 每项任务的验收记录 |
| [quality_baseline.md](quality_baseline.md) | 质量门禁和基线记录 |
| [source_map.md](source_map.md) | 旧文档内容吸收与归档说明 |

## 专题合并文档

| 文档 | 合并范围 | 后续用途 |
| --- | --- | --- |
| [frontend-foundation.md](frontend-foundation.md) | 前端专项计划、FE0-FE4 页面体验、设计系统、架构规范、插件门户材料 | 后续前端页面和交互实现的主规范 |
| [frontend-quality-performance.md](frontend-quality-performance.md) | FE5 测试验收体系、FE6 性能基线、browser smoke、响应式和回归清单 | 后续前端质量门禁和性能验收入口 |
| [milestone-closeout.md](milestone-closeout.md) | M2 验收记录、进度巡检、尾盘阈值、任务微调建议 | 发布前收口与项目 leader 巡检入口 |

## 单任务执行流程

1. 从 [work_items.md](work_items.md) 选择状态为 `Todo` 的最前 Work Item。
2. 读取任务对应 skill。
3. 实现任务范围内的代码、测试、文档。
4. 执行 Work Item 中的验证命令。
5. 在 [acceptance_log.md](acceptance_log.md) 追加验收记录。
6. 验收通过后提交代码。
7. 将 Work Item 状态更新为 `Done`，进入下一个 Work Item。

当前批次已全部完成。后续只允许处理发布前质量门禁失败、文档索引缺口和验收记录不一致问题，不再扩散新功能范围。

下一批功能建设从 [next_work_items.md](next_work_items.md) 领取任务。进入开发前，应先从 N0 发布前冻结与质量门禁开始，再按 M3 到 M7 顺序搬入正式执行表。

## 提交规则

提交信息使用:

```text
<task-id>: <short summary>
```

发布前至少确认:

- `work_items.md`、`acceptance_log.md`、`git log` 对已完成任务一致。
- OpenAPI、权限、审计、迁移、前端 API client 已同步。
- `go test ./...`、`cd web && npm run typecheck`、`cd web && npm run build` 已执行并记录。
- `git diff` 中没有无关文件。

## 归档说明

细碎过程文档已移动到 [../archive/refactor-2026-06-19](../archive/refactor-2026-06-19)。这些文件只作为证据和历史参考，不再作为执行入口。若后续发现归档文档仍有有效信息，应摘录到当前主目录专题文档，再保持归档状态。
