# Skoll 重构执行入口

> 当前状态: 执行入口  
> 更新时间: 2026-06-18  
> 适用范围: Skoll 开源基础建设重构

## 1. 执行原则

1. 当前目录是重构唯一执行入口。
2. 旧计划文档只作为历史参考，不作为任务来源。
3. Skoll 当前处于开源基础建设阶段，不做兼容方案，不做旧接口、旧数据结构、旧插件格式、旧页面路径的兼容方案。
4. 每个任务必须有验收要求；验收失败则任务退回重新执行。
5. 每完成一个任务并通过验收后，提交一次代码。
6. 每个提交只包含一个任务的代码、测试和文档同步，不混入无关重构。

## 2. 当前文档结构

```text
docs/refactor/
├── README.md                         # 本文档，重构执行入口
├── architecture_and_execution_plan.md # 架构模型、治理原则、里程碑计划
├── task_board.md                     # 可执行任务表，按顺序执行和验收
├── work_items.md                     # 细粒度最小执行与提交单元
├── frontend_experience_plan.md       # 前端体验、视觉、编码、测试、性能专项计划
├── acceptance_log.md                 # 任务验收记录模板
├── quality_baseline.md               # M0 质量基线命令结果、失败原因和重跑命令
├── gin-vue-admin_analysis.md         # gin-vue-admin 对标分析
└── source_map.md                     # 旧文档吸收记录与归档说明
```

## 3. 推荐执行顺序

1. 先执行 `M0`，建立质量基线、任务验收模板和文档入口。
2. 再执行 `M1`，完成权限资源目录与菜单注册中心。
3. 接着执行 `M2` 和 `M3`，补齐审计和文件存储。
4. 然后执行 `M4`，完成字典、配置、组织和数据范围。
5. 最后执行 `M5` 和 `M6`，推进代码生成器和插件市场。
6. `M7` 质量、文档、示例和发布门禁贯穿所有里程碑。

## 4. 单任务执行流程

每个任务按以下流程推进:

1. 从 [work_items.md](work_items.md) 选择状态为 `Todo` 的最前 Work Item。
2. 读取任务对应 skill。
3. 实现任务范围内的代码、测试、文档。
4. 执行 Work Item 中的验证命令。
5. 在 [acceptance_log.md](acceptance_log.md) 追加验收记录。
6. 验收通过后提交代码。
7. 将 Work Item 状态更新为 `Done`，进入下一 Work Item。

验收失败时:

1. 将任务状态标记为 `Failed`。
2. 记录失败原因和失败命令。
3. 修复后重新执行验收。
4. 通过前不得提交完成状态。

## 5. 提交规则

一任务一提交: 每个 Work Item 验收通过后提交一次代码；验收未通过不得提交完成状态。

建议提交信息:

```text
<task-id>: <short summary>
```

示例:

```text
M1-01: add permission resource domain model
M3-03: implement local object storage adapter
M5-03: add generator dry-run diff
```

每次任务提交前至少确认:

- 任务验收项全部通过。
- OpenAPI、权限、审计、迁移、前端 API client 已同步。
- `git diff` 中没有无关文件。
- `docs/development_skills.md` 等用户已有改动未被混入。

## 6. 旧文档处理

已将旧计划类文档移动到 `docs/archive/legacy-plans/`。有效内容已吸收到:

- [architecture_and_execution_plan.md](architecture_and_execution_plan.md)
- [task_board.md](task_board.md)
- [work_items.md](work_items.md)
- [frontend_experience_plan.md](frontend_experience_plan.md)
- [quality_baseline.md](quality_baseline.md)
- [source_map.md](source_map.md)

后续如发现旧文档仍有有效信息，应摘录到当前执行文件，再保持旧文档归档状态。
