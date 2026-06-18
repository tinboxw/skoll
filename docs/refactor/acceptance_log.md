# Skoll 重构验收记录

> 规则: 一任务一提交。每个任务通过验收后追加一条记录，并在同一次任务提交中包含该记录。验收失败也要记录失败原因，修复后重新验收。

## 记录模板

```markdown
## <task-id>: <task-title>

- 状态: Passed | Failed | Blocked
- Work Item:
- 日期:
- 执行人:
- 提交:

### 改动文件

- 

### 验收项

以下边界同步项为默认检查项；无影响时填写 `N/A` 并说明原因，有影响时必须同步对应交付物。

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 |  |  |
| API/OpenAPI 同步 |  |  |
| 权限目录同步 |  |  |
| 审计 action 同步 |  |  |
| migration/seed 同步 |  |  |
| 前端 API client/UI 同步 |  |  |
| 文档同步 |  |  |
| 无兼容方案/无旧路径残留 |  |  |

### 自动化验证

```powershell

```

结果摘要:

### 人工验收

1. 

结果摘要:

### 失败与返工

验收失败时必须保留本区内容；修复后追加返工动作和重新验收结果，不覆盖失败原因。

- 失败原因:
- 返工动作:
- 重新验收结果:

### 下一步

- 
```

## 验收记录

## M0-01-01: 校准架构决策文档标题、状态、适用范围

- 状态: Passed
- Work Item: M0-01-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/architecture_and_execution_plan.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 架构文档标题、状态、适用范围已明确。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 架构计划与 Work Item 状态已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 明确旧计划只作为历史归档材料，不作为任务来源、架构依据或执行入口。 |

### 自动化验证

```powershell
rg -n "唯一架构|旧计划|执行入口|当前状态|适用范围" docs\refactor\architecture_and_execution_plan.md docs\refactor\README.md docs\README.md
```

结果摘要: 通过。执行入口与旧计划归档说明可在当前文档入口中定位，架构计划已声明唯一生效。

### 人工验收

1. 审阅 `docs/refactor/architecture_and_execution_plan.md` 顶部元信息。
2. 审阅 `docs/refactor/README.md` 和 `docs/README.md` 中旧计划归档说明。

结果摘要: 通过。当前架构决策、适用范围和旧计划非执行入口的关系明确。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-01-02`，固化不做兼容方案规则。

## M0-01-02: 固化不做兼容方案规则

- 状态: Passed
- Work Item: M0-01-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/architecture_and_execution_plan.md`
- `docs/refactor/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | “不做兼容方案”规则已写入治理原则和执行入口。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 架构计划、执行入口、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 明确不做兼容方案，不保留旧接口、旧数据结构、旧插件格式和旧页面路径兼容层。 |

### 自动化验证

```powershell
rg "不做旧接口|不做兼容方案" docs/refactor docs/README.md
```

结果摘要: 通过。执行入口与治理原则均可检索到规则。

### 人工验收

1. 审阅 `docs/refactor/architecture_and_execution_plan.md` 的项目治理原则。
2. 审阅 `docs/refactor/README.md` 的执行原则。

结果摘要: 通过。规则位置明确，后续任务可直接引用。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-01-03`，固化分层依赖规则。

## M0-01-03: 固化分层依赖规则

- 状态: Passed
- Work Item: M0-01-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/architecture_and_execution_plan.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 架构计划新增分层边界验收口径。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 架构计划、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只固化边界规则，未引入兼容路径。 |

### 自动化验证

```powershell
rg "domain|service|repository|store|handler|web|plugin" docs/refactor/architecture_and_execution_plan.md
```

结果摘要: 通过。关键分层名称均可在架构计划中定位。

### 人工验收

1. 审阅 `docs/refactor/architecture_and_execution_plan.md` 的依赖规则。
2. 审阅新增分层边界验收口径表。

结果摘要: 通过。各层职责和禁止事项可作为后续 Work Item 的验收依据。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-01-04`，关联重构专用 skills。

## M0-01-04: 关联重构专用 skills

- 状态: Passed
- Work Item: M0-01-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/architecture_and_execution_plan.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 架构计划明确 Work Item 的 Skill 字段作为任务路由来源。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 架构计划、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只关联任务执行 skills，未引入兼容路径。 |

### 自动化验证

```powershell
rg "skoll-.*refactor" docs/refactor/architecture_and_execution_plan.md
```

结果摘要: 通过。M0-M6 及前端专项可在重构专用 skills 表中定位到对应 skill。

### 人工验收

1. 审阅 `docs/refactor/architecture_and_execution_plan.md` 的重构专用 Skills 表。
2. 确认 M0-M6 均能找到对应的 `skoll-*-refactor` skill。

结果摘要: 通过。后续任务可按 Work Item 的 Skill 字段读取对应 skill。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-02-01`，建立验收记录模板字段。

## M0-02-01: 建立验收记录模板字段

- 状态: Passed
- Work Item: M0-02-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/acceptance_log.md`
- `docs/refactor/work_items.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 验收记录模板包含状态、Work Item、日期、执行人、提交、改动文件和验收项。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 验收模板、历史记录字段和 Work Item 状态已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只补齐验收字段，未引入兼容路径。 |

### 自动化验证

```powershell
rg "状态:|Work Item:|日期:|执行人:|提交:|### 改动文件|### 验收项" docs/refactor/acceptance_log.md
```

结果摘要: 通过。模板必备字段均可检索到。

### 人工验收

1. 审阅 `docs/refactor/acceptance_log.md` 的记录模板。
2. 确认模板包含状态、日期、执行人、提交、改动文件、验收项。

结果摘要: 通过。模板字段满足后续任务验收记录要求。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-02-02`，补齐边界同步验收项。

## M0-02-02: 补齐边界同步验收项

- 状态: Passed
- Work Item: M0-02-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/acceptance_log.md`
- `docs/refactor/work_items.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 验收模板明确边界同步项是默认检查项。 |
| API/OpenAPI 同步 | Passed | 模板包含 API/OpenAPI 同步检查项。 |
| 权限目录同步 | Passed | 模板包含权限目录同步检查项。 |
| 审计 action 同步 | Passed | 模板包含审计 action 同步检查项。 |
| migration/seed 同步 | Passed | 模板包含 migration/seed 同步检查项。 |
| 前端 API client/UI 同步 | Passed | 模板包含前端 API client/UI 同步检查项。 |
| 文档同步 | Passed | 模板包含文档同步检查项，Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 模板保留无兼容方案/无旧路径残留检查项。 |

### 自动化验证

```powershell
rg "API/OpenAPI|权限目录|审计 action|migration/seed|前端 API client/UI|文档同步" docs/refactor/acceptance_log.md
```

结果摘要: 通过。边界同步验收项均可检索到。

### 人工验收

1. 审阅 `docs/refactor/acceptance_log.md` 的验收项表。
2. 确认 API/OpenAPI、权限、审计、migration、前端、文档均有检查项。

结果摘要: 通过。模板满足跨边界任务同步检查要求。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-02-03`，补齐失败返工记录区。

## M0-02-03: 补齐失败返工记录区

- 状态: Passed
- Work Item: M0-02-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/acceptance_log.md`
- `docs/refactor/work_items.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 模板包含失败原因、返工动作、重新验收结果字段，并要求保留失败记录。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 验收模板、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只补齐返工记录区，未引入兼容路径。 |

### 自动化验证

```powershell
rg "失败原因|返工动作|重新验收结果|不覆盖失败原因" docs/refactor/acceptance_log.md
```

结果摘要: 通过。失败返工记录字段和保留规则均可检索到。

### 人工验收

1. 审阅 `docs/refactor/acceptance_log.md` 的失败与返工模板。
2. 确认失败原因、返工动作、重新验收结果字段齐全。

结果摘要: 通过。模板支持失败、返工、重验收的闭环记录。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-02-04`，补齐一任务一提交说明。

## M0-02-04: 补齐一任务一提交说明

- 状态: Passed
- Work Item: M0-02-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/README.md`
- `docs/refactor/acceptance_log.md`
- `docs/refactor/work_items.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 执行入口和验收记录规则均明确一任务一提交。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 执行入口、验收记录、Work Item 状态已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只补齐提交规则，未引入兼容路径。 |

### 自动化验证

```powershell
rg "提交一次代码|一任务一提交" docs/refactor
```

结果摘要: 通过。执行入口、验收记录和任务表均可检索到提交规则。

### 人工验收

1. 审阅 `docs/refactor/README.md` 的提交规则。
2. 审阅 `docs/refactor/acceptance_log.md` 顶部规则。

结果摘要: 通过。一任务一提交规则明确，验收失败不得提交完成状态。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-03-01`，运行 Go 全量测试。

## M0-03-01: 运行 Go 全量测试

- 状态: Passed
- Work Item: M0-03-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/quality_baseline.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `go test ./...` 已执行，命令、结果、失败包和失败原因已记录。 |
| API/OpenAPI 同步 | N/A | 仅质量基线记录任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅质量基线记录任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅质量基线记录任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅质量基线记录任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅质量基线记录任务，无前端实现影响。 |
| 文档同步 | Passed | 质量基线记录、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只记录真实测试基线，未引入兼容路径。 |

### 自动化验证

```powershell
go test ./...
go env CGO_ENABLED CC
Test-Path 'D:\Program Files\JetBrains\CLion 2024.1.1\bin\mingw\bin\gcc.exe'
```

结果摘要: `go test ./...` 失败。失败原因是 `CGO_ENABLED=1` 且 `CC` 指向的 gcc 路径不存在；失败包和重跑命令已记录在 `docs/refactor/quality_baseline.md`。

### 人工验收

1. 审阅 `docs/refactor/quality_baseline.md` 的 M0-03-01 记录。
2. 确认记录包含命令、结果、失败包、失败原因和重跑命令。

结果摘要: 通过。本项验收目标是建立真实 Go 全量测试基线，当前基线为失败且原因明确。

### 失败与返工

- 失败原因: `go test ./...` 因本机 cgo 编译器路径不存在而失败。
- 返工动作: 记录失败包、失败原因、环境核对结果和重跑命令；不在本项中修改本机工具链。
- 重新验收结果: 基线记录验收通过；测试失败作为后续质量修复输入。

### 下一步

- 进入 `M0-03-02`，运行 Go 覆盖率统计。

## M0-03-02: 运行 Go 覆盖率统计

- 状态: Passed
- Work Item: M0-03-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/quality_baseline.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 覆盖率命令已执行，失败原因、`coverage.out` 状态、部分包覆盖率和重跑命令已记录。 |
| API/OpenAPI 同步 | N/A | 仅质量基线记录任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅质量基线记录任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅质量基线记录任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅质量基线记录任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅质量基线记录任务，无前端实现影响。 |
| 文档同步 | Passed | 质量基线记录、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只记录真实覆盖率基线，未引入兼容路径。 |

### 自动化验证

```powershell
go test ./... -coverprofile=coverage.out
if (Test-Path coverage.out) { go tool cover -func=coverage.out } else { Write-Output 'coverage.out missing' }
```

结果摘要: 覆盖率命令失败，`coverage.out` 未生成。失败原因是本机 cgo 编译器路径不存在；部分包覆盖率和重跑命令已记录在 `docs/refactor/quality_baseline.md`。

### 人工验收

1. 审阅 `docs/refactor/quality_baseline.md` 的 M0-03-02 记录。
2. 确认记录包含命令、结果、覆盖率文件状态、失败原因、部分包覆盖率和重跑命令。

结果摘要: 通过。本项验收目标是建立真实覆盖率基线，当前完整总覆盖率不可用且限制明确。

### 失败与返工

- 失败原因: `go test ./... -coverprofile=coverage.out` 因本机 cgo 编译器路径不存在而失败，未生成 `coverage.out`。
- 返工动作: 记录失败原因、覆盖率文件状态、可观察到的部分包覆盖率和重跑命令；不在本项中修改本机工具链。
- 重新验收结果: 覆盖率基线记录验收通过；覆盖率命令失败作为后续质量修复输入。

### 下一步

- 进入 `M0-03-03`，运行前端构建。

## M0-03-03: 运行前端构建

- 状态: Passed
- Work Item: M0-03-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/quality_baseline.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 前端 build 已执行，结果和 warning 已记录。 |
| API/OpenAPI 同步 | N/A | 仅质量基线记录任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅质量基线记录任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅质量基线记录任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅质量基线记录任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 本项只运行构建并记录结果，无前端代码改动。 |
| 文档同步 | Passed | 质量基线记录、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只记录真实构建基线，未引入兼容路径。 |

### 自动化验证

```powershell
cd web
npm run build
```

结果摘要: 通过。Vite 构建成功，3485 modules transformed，built in 9.01s；Sass deprecation、Rollup PURE annotation 和 npm notice 已记录在 `docs/refactor/quality_baseline.md`。

### 人工验收

1. 审阅 `docs/refactor/quality_baseline.md` 的 M0-03-03 记录。
2. 确认 build 结果、主要产物尺寸和 warning 已记录。

结果摘要: 通过。前端构建基线可复现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-03-04`，检查前端 typecheck 命令。

## M0-03-04: 检查前端 typecheck 命令

- 状态: Passed
- Work Item: M0-03-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/quality_baseline.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | typecheck 脚本存在，命令已执行并记录。 |
| API/OpenAPI 同步 | N/A | 仅质量基线记录任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅质量基线记录任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅质量基线记录任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅质量基线记录任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 本项只运行 typecheck 并记录结果，无前端代码改动。 |
| 文档同步 | Passed | 质量基线记录、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只记录真实 typecheck 基线，未引入兼容路径。 |

### 自动化验证

```powershell
cd web
npm run typecheck
```

结果摘要: 通过。`vue-tsc --noEmit` 未输出类型错误。

### 人工验收

1. 审阅 `web/package.json` 的 `typecheck` 脚本。
2. 审阅 `docs/refactor/quality_baseline.md` 的 M0-03-04 记录。

结果摘要: 通过。前端 typecheck 基线可复现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-03-05`，记录测试副作用。

## M0-03-05: 记录测试副作用

- 状态: Passed
- Work Item: M0-03-05
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/quality_baseline.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `git status --short` 输出已记录，污染文件 `coverage` 已列入并清理。 |
| API/OpenAPI 同步 | N/A | 仅质量基线记录任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅质量基线记录任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅质量基线记录任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅质量基线记录任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 本项只记录测试副作用，无前端代码改动。 |
| 文档同步 | Passed | 质量基线记录、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只记录并清理测试副作用，未引入兼容路径。 |

### 自动化验证

```powershell
git status --short
```

结果摘要: 首次输出 `?? coverage`；确认该文件为本轮覆盖率命令副作用后已清理，再次执行输出为空。

### 人工验收

1. 审阅 `docs/refactor/quality_baseline.md` 的 M0-03-05 记录。
2. 确认污染文件、修复动作和清理后状态均已记录。

结果摘要: 通过。质量门禁副作用已记录并清理。

### 失败与返工

- 失败原因: 无
- 返工动作: 已清理未跟踪覆盖率副作用文件 `coverage`。
- 重新验收结果: 通过，清理后 `git status --short` 输出为空。

### 下一步

- 进入 `M0-04-01`，校准父任务表状态枚举。

## M0-04-01: 校准父任务表状态枚举

- 状态: Passed
- Work Item: M0-04-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/task_board.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 父任务表明确必须与 Work Item 表使用同一套状态枚举。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 父任务表、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只校准任务状态枚举，未引入兼容路径。 |

### 自动化验证

```powershell
Select-String -Path docs\refactor\task_board.md,docs\refactor\work_items.md -Pattern "Todo|Doing|Review|Failed|Done|Blocked"
```

结果摘要: 通过。父任务表与 Work Item 表均包含同一套状态枚举。

### 人工验收

1. 审阅 `docs/refactor/task_board.md` 的状态枚举。
2. 对照 `docs/refactor/work_items.md` 的状态枚举。

结果摘要: 通过。两张任务表状态枚举一致。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-04-02`，校准父任务与 Work Item 关系。

## M0-04-02: 校准父任务与 Work Item 关系

- 状态: Passed
- Work Item: M0-04-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/task_board.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 父任务表明确父任务只用于分组和聚合，Work Item 才是最小执行与提交单元。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 父任务表、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只校准任务表关系，未引入兼容路径。 |

### 自动化验证

```powershell
rg "Work Item|最小执行" docs/refactor
```

结果摘要: 通过。当前执行入口、父任务表和 Work Item 表均说明最小执行单元。

### 人工验收

1. 审阅 `docs/refactor/task_board.md` 顶部执行规则。
2. 审阅 `docs/refactor/work_items.md` 顶部执行规则。

结果摘要: 通过。父任务与 Work Item 的职责边界明确。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-04-03`，校准 M0-M2 Work Item 覆盖率。

## M0-04-03: 校准 M0-M2 Work Item 覆盖率

- 状态: Passed
- Work Item: M0-04-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | M0-M2 每个父任务均至少有一个 Work Item，覆盖校验结果已写入 `work_items.md`。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | Work Item 覆盖说明、状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只校准任务覆盖率，未引入兼容路径。 |

### 自动化验证

```powershell
foreach ($id in @('M0-01','M0-02','M0-03','M0-04','M0-05','M1-01','M1-02','M1-03','M1-04','M1-05','M1-06','M1-07','M1-08','M1-09','M1-10','M2-01','M2-02','M2-03','M2-04','M2-05','M2-06')) {
  $count=(Select-String -Path docs\refactor\work_items.md -SimpleMatch "| $id" | Measure-Object).Count
  "$id $count"
}
```

结果摘要: 通过。M0-M2 每个父任务均有 Work Item，计数范围为 4-8。

### 人工验收

1. 审阅 `docs/refactor/work_items.md` 中 M0、M1、M2 Work Items。
2. 确认父任务 M0-01 至 M2-06 均有对应 Work Item。

结果摘要: 通过。M0-M2 父任务覆盖完整。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-04-04`，设置 M3-M7 滚动拆分规则。

## M0-04-04: 设置 M3-M7 滚动拆分规则

- 状态: Passed
- Work Item: M0-04-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `work_items.md` 明确 M3-M7 在 M2 验收完成后滚动拆分，拆分前父任务只作为路线图。 |
| API/OpenAPI 同步 | N/A | 仅文档治理任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档治理任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档治理任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档治理任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档治理任务，无前端实现影响。 |
| 文档同步 | Passed | 滚动拆分规则、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只设置拆分规则，未引入兼容路径。 |

### 自动化验证

```powershell
rg "M3-M7|滚动拆分|M2 验收完成" docs/refactor/work_items.md
```

结果摘要: 通过。M3-M7 滚动拆分规则可检索到。

### 人工验收

1. 审阅 `docs/refactor/work_items.md` 顶部执行规则。
2. 确认 M3-M7 在 M2 验收完成前不作为直接执行单元。

结果摘要: 通过。滚动拆分规则明确。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-05-01`，更新 `docs/refactor/README.md` 文档结构。

## M0-05-01: 更新 docs/refactor/README.md 文档结构

- 状态: Passed
- Work Item: M0-05-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `docs/refactor/README.md` 文档结构列出 `work_items.md`，并补充新增的 `quality_baseline.md`。 |
| API/OpenAPI 同步 | N/A | 仅文档入口任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档入口任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档入口任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档入口任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档入口任务，无前端实现影响。 |
| 文档同步 | Passed | 重构入口结构、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只更新当前入口结构，未引入旧计划执行入口。 |

### 自动化验证

```powershell
rg "work_items.md" docs/refactor/README.md
```

结果摘要: 通过。`work_items.md` 可从重构入口结构中定位。

### 人工验收

1. 审阅 `docs/refactor/README.md` 的文档结构。
2. 确认当前重构入口列出 `work_items.md` 和 `quality_baseline.md`。

结果摘要: 通过。重构入口结构与当前文件集合一致。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-05-02`，更新顶层 `docs/README.md` 当前入口。

## M0-05-02: 更新顶层 docs/README.md 当前入口

- 状态: Passed
- Work Item: M0-05-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 顶层文档入口包含 `work_items.md`，并补充新增的 `quality_baseline.md`。 |
| API/OpenAPI 同步 | N/A | 仅文档入口任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档入口任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档入口任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档入口任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档入口任务，无前端实现影响。 |
| 文档同步 | Passed | 顶层文档入口、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 本项只更新当前入口索引，未引入旧计划执行入口。 |

### 自动化验证

```powershell
rg "work_items.md" docs/README.md
```

结果摘要: 通过。`work_items.md` 可从顶层文档入口定位。

### 人工验收

1. 审阅 `docs/README.md` 当前执行入口表。
2. 确认顶层入口包含 `work_items.md` 和 `quality_baseline.md`。

结果摘要: 通过。顶层文档入口与当前重构文档结构同步。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-05-03`，校验旧计划文档归档。

## M0-05-03: 校验旧计划文档归档

- 状态: Passed
- Work Item: M0-05-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/source_map.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 旧计划类文档位于 `docs/archive/legacy-plans`，`source_map.md` 补齐吸收说明。 |
| API/OpenAPI 同步 | N/A | 仅文档归档任务，无 API 影响。 |
| 权限目录同步 | N/A | 仅文档归档任务，无权限影响。 |
| 审计 action 同步 | N/A | 仅文档归档任务，无审计 action 影响。 |
| migration/seed 同步 | N/A | 仅文档归档任务，无数据结构影响。 |
| 前端 API client/UI 同步 | N/A | 仅文档归档任务，无前端实现影响。 |
| 文档同步 | Passed | 旧文档吸收记录、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 旧计划只作为历史参考，不作为当前执行源。 |

### 自动化验证

```powershell
Get-ChildItem docs/archive/legacy-plans
```

结果摘要: 通过。归档目录包含旧计划类文档，`source_map.md` 记录其吸收去向和不再采用内容。

### 人工验收

1. 审阅 `docs/refactor/source_map.md` 的已归档目录和吸收内容。
2. 确认当前事实来源不指向旧计划作为执行源。

结果摘要: 通过。旧计划归档状态和当前事实来源清晰。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M0-05-04`，补充 M0 文档验收记录。

## M0-05-04: M0 文档验收记录

- 状态: Passed
- Work Item: M0-05-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/refactor/task_board.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | M0-01 至 M0-05 均已有验收记录，父任务表 M0 状态已标记 Done。 |
| API/OpenAPI 同步 | N/A | M0 文档整理阶段未变更 API。 |
| 权限目录同步 | N/A | M0 文档整理阶段未变更权限目录。 |
| 审计 action 同步 | N/A | M0 文档整理阶段未变更审计 action。 |
| migration/seed 同步 | N/A | M0 文档整理阶段未变更数据结构。 |
| 前端 API client/UI 同步 | N/A | M0 文档整理阶段未变更前端实现。 |
| 文档同步 | Passed | 架构计划、任务表、Work Item、质量基线、文档入口、旧文档归档记录和验收日志已同步。 |
| 无兼容方案/无旧路径残留 | Passed | M0 规则明确不做兼容方案，旧计划仅作为历史归档。 |

### 自动化验证

```powershell
rg "M0-01-01|M0-02-01|M0-03-01|M0-04-01|M0-05-01|M0-05-04" docs/refactor/acceptance_log.md
rg "work_items.md" docs/README.md docs/refactor/README.md
Get-ChildItem docs/archive/legacy-plans
```

结果摘要: 通过。M0 文档整理任务均有验收记录；顶层和重构入口均可定位 Work Items；旧计划归档目录可访问。

### 人工验收

1. 审阅 `docs/refactor/acceptance_log.md` 中 M0-01 至 M0-05 记录。
2. 审阅 `docs/refactor/task_board.md` 的 M0 父任务状态。
3. 审阅 `docs/refactor/quality_baseline.md` 的质量基线真实结果。

结果摘要: 通过。M0 文档整理闭环完成；Go 全量测试和覆盖率当前因本机 cgo 编译器路径缺失失败，已作为质量基线限制记录。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-01-01`，创建 permission domain 包结构。

## M1-01-01: 创建 permission domain 包结构

- 状态: Passed
- Work Item: M1-01-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/permission/doc.go`
- `internal/domain/permission/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 `internal/domain/permission` 包结构，包可编译。 |
| API/OpenAPI 同步 | N/A | 本项只创建 domain 包结构，无 API 影响。 |
| 权限目录同步 | N/A | 权限目录模型尚未定义，后续 M1-01-02 起补齐。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | README 明确不添加旧权限别名、兼容授权路径或新旧权限双轨实现。 |

### 自动化验证

```powershell
go test ./internal/domain/permission
```

结果摘要: 通过。包可编译，无测试文件。

### 人工验收

1. 审阅 `internal/domain/permission/doc.go`。
2. 审阅 `internal/domain/permission/README.md`，确认无 store/service 依赖。

结果摘要: 通过。permission domain 包已建立为干净领域边界。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-01-02`，定义 PermissionResource 类型和值对象。

## M1-01-02: 定义 PermissionResource 类型和值对象

- 状态: Passed
- Work Item: M1-01-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/permission/resource.go`
- `internal/domain/permission/resource_test.go`
- `internal/domain/permission/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 定义 `PermissionResource`、`ResourceIdentity` 和 api/menu/button/data_scope/plugin 类型枚举。 |
| API/OpenAPI 同步 | N/A | 本项只定义 domain 类型，无 API 影响。 |
| 权限目录同步 | Passed | 权限目录领域类型开始建立，尚未接入持久化或服务。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧权限别名、兼容授权路径或新旧权限双轨实现。 |

### 自动化验证

```powershell
go test ./internal/domain/permission
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/domain/permission/resource.go`。
2. 确认支持 `api`、`menu`、`button`、`data_scope`、`plugin` 五类权限资源。

结果摘要: 通过。PermissionResource 领域类型和值对象已建立。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-01-03`，实现 PermissionResource 校验规则。

## M1-01-03: 实现 PermissionResource 校验规则

- 状态: Passed
- Work Item: M1-01-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/permission/resource.go`
- `internal/domain/permission/resource_test.go`
- `internal/domain/permission/rules.go`
- `internal/domain/permission/rules_test.go`
- `internal/domain/permission/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | key/name/module/source 必填和 type 枚举校验已实现并测试覆盖。 |
| API/OpenAPI 同步 | N/A | 本项只修改 domain 校验规则，无 API 影响。 |
| 权限目录同步 | Passed | 权限目录领域校验规则已建立，尚未接入持久化或服务。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 校验规则只接受当前类型枚举，未添加旧权限别名或兼容路径。 |

### 自动化验证

```powershell
go test ./internal/domain/permission
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/domain/permission/rules.go`。
2. 审阅 `internal/domain/permission/rules_test.go` 的失败路径覆盖。

结果摘要: 通过。PermissionResource 校验规则覆盖必填字段和类型枚举。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-01-04`，实现 permission risk/source metadata。

## M1-01-04: 实现 permission risk/source metadata

- 状态: Passed
- Work Item: M1-01-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/permission/metadata.go`
- `internal/domain/permission/metadata_test.go`
- `internal/domain/permission/resource.go`
- `internal/domain/permission/resource_test.go`
- `internal/domain/permission/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 实现 risk 枚举、metadata 规范化/校验和 JSON 序列化测试；source 校验沿用 M1-01-03 规则。 |
| API/OpenAPI 同步 | N/A | 本项只修改 domain 类型，无 API 影响。 |
| 权限目录同步 | Passed | 权限目录领域模型包含风险和 metadata。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 风险等级只支持当前枚举，未添加旧风险别名或兼容路径。 |

### 自动化验证

```powershell
go test ./internal/domain/permission
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/domain/permission/metadata.go`。
2. 审阅 `internal/domain/permission/metadata_test.go` 的 risk、metadata 和序列化覆盖。

结果摘要: 通过。PermissionResource 风险与 metadata 能校验并序列化。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-01-05`，补 permission domain README。
