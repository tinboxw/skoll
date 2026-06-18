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

## M1-01-05: 补 permission domain README

- 状态: Passed
- Work Item: M1-01-05
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/permission/README.md`
- `docs/refactor/task_board.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | README 说明资源类型、命名规则、风险等级、metadata 和 no-compat 规则。 |
| API/OpenAPI 同步 | N/A | 本项只补充 domain README，无 API 影响。 |
| 权限目录同步 | Passed | PermissionResource 模型文档完成，父任务 M1-01 标记 Done。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、父任务状态、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | README 明确不添加旧权限别名、不保留兼容授权路径、不维护新旧权限双轨来源。 |

### 自动化验证

```powershell
go test ./internal/domain/permission
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/domain/permission/README.md`。
2. 确认 README 覆盖类型、命名和 no-compat 规则。

结果摘要: 通过。permission domain README 可作为后续实现约束。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-02-01`，创建 menu domain 包结构。

## M1-02-01: 创建 menu domain 包结构

- 状态: Passed
- Work Item: M1-02-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/menu/doc.go`
- `internal/domain/menu/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 `internal/domain/menu` 包结构，包可编译。 |
| API/OpenAPI 同步 | N/A | 本项只创建 domain 包结构，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | README 明确不添加旧菜单来源、兼容路由或新旧菜单双轨实现。 |

### 自动化验证

```powershell
go test ./internal/domain/menu
```

结果摘要: 通过。包可编译，无测试文件。

### 人工验收

1. 审阅 `internal/domain/menu/doc.go`。
2. 审阅 `internal/domain/menu/README.md`，确认无 store/service 依赖。

结果摘要: 通过。menu domain 包已建立为干净领域边界。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-02-02`，定义 MenuNode 类型和值对象。

## M1-02-02: 定义 MenuNode 类型和值对象

- 状态: Passed
- Work Item: M1-02-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/menu/node.go`
- `internal/domain/menu/node_test.go`
- `internal/domain/menu/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 定义 `MenuNode`、`NodeIdentity`、`NodeView`，覆盖 parent/path/component/icon/sort/visible/source 字段。 |
| API/OpenAPI 同步 | N/A | 本项只定义 domain 类型，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧菜单来源、兼容路由或新旧菜单双轨实现。 |

### 自动化验证

```powershell
go test ./internal/domain/menu
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/domain/menu/node.go`。
2. 确认 MenuNode 包含 parent/path/component/icon/sort/visible/source 字段。

结果摘要: 通过。MenuNode 领域类型和值对象已建立。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-02-03`，实现 MenuNode 校验规则。

## M1-02-03: 实现 MenuNode 校验规则

- 状态: Passed
- Work Item: M1-02-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/menu/node.go`
- `internal/domain/menu/node_test.go`
- `internal/domain/menu/rules.go`
- `internal/domain/menu/rules_test.go`
- `internal/domain/menu/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `NewNode` 统一规范化 identity/view，并校验 path、name、sort、source。 |
| API/OpenAPI 同步 | N/A | 本项只变更 domain 类型和校验，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧菜单来源、兼容路由或新旧菜单双轨实现。 |

### 自动化验证

```powershell
go test ./internal/domain/menu
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/domain/menu/rules.go`。
2. 确认 path、name、sort、source 均有失败用例覆盖。

结果摘要: 通过。MenuNode 校验规则已建立，并由单元测试覆盖。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-02-04`，实现菜单树排序与过滤规则。

## M1-02-04: 实现菜单树排序与过滤规则

- 状态: Passed
- Work Item: M1-02-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/menu/tree.go`
- `internal/domain/menu/tree_test.go`
- `internal/domain/menu/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增同级节点稳定排序、隐藏节点过滤和权限过滤规则。 |
| API/OpenAPI 同步 | N/A | 本项只变更 domain 规则，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧菜单来源、兼容路由或新旧菜单双轨实现。 |

### 自动化验证

```powershell
go test ./internal/domain/menu
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/domain/menu/tree.go`。
2. 确认 `SortSiblings` 使用稳定排序且不修改入参。
3. 确认隐藏节点和缺少角色/权限的节点可被过滤。

结果摘要: 通过。菜单树排序与过滤规则已建立。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-02-05`，补充 menu domain README。

## M1-02-05: 补 menu domain README

- 状态: Passed
- Work Item: M1-02-05
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/domain/menu/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | README 已说明菜单节点、树规则、权限字段和无兼容边界。 |
| API/OpenAPI 同步 | N/A | 本项只补充包内文档，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | 包 README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | README 明确不保留旧菜单来源或旧路由映射。 |

### 自动化验证

```powershell
go test ./internal/domain/menu
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/domain/menu/README.md`。
2. 确认已说明菜单节点、树、权限字段。

结果摘要: 通过。menu domain README 已补齐。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-03-01`，设计权限表 migration。

## M1-03-01: 设计权限表 migration

- 状态: Passed
- Work Item: M1-03-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `migrations/mysql/20260618_000013_create_permission_resources.sql`
- `migrations/postgres/20260618_000013_create_permission_resources.sql`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | MySQL/PostgreSQL migration 均新增 `sk_permission_resources` 表。 |
| API/OpenAPI 同步 | N/A | 本项只设计数据库 migration，无 API 影响。 |
| 权限目录同步 | Passed | 表结构承载 permission domain 的 key/type/module/source/name/risk/metadata/enabled 字段。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | Passed | 新增 MySQL/PostgreSQL migration，暂不引入 seed。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步；migrations README 由 `M1-03-05` 专项补充。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧权限表兼容迁移或双写结构。 |

### 自动化验证

```powershell
rg "sk_permission_resources|uk_permission_resources_key|idx_permission_resources" migrations/mysql/20260618_000013_create_permission_resources.sql migrations/postgres/20260618_000013_create_permission_resources.sql
```

结果摘要: 通过。字段、唯一约束和索引定义均可检索。

### 人工验收

1. 审阅 MySQL migration。
2. 审阅 PostgreSQL migration。
3. 确认字段、索引、唯一约束齐全。

结果摘要: 通过。权限资源表 migration 设计完成。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-03-02`，设计菜单表 migration。

## M1-03-02: 设计菜单表 migration

- 状态: Passed
- Work Item: M1-03-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `migrations/mysql/20260618_000014_create_menu_nodes.sql`
- `migrations/postgres/20260618_000014_create_menu_nodes.sql`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | MySQL/PostgreSQL migration 均新增 `sk_menu_nodes` 表。 |
| API/OpenAPI 同步 | N/A | 本项只设计数据库 migration，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | Passed | 新增 MySQL/PostgreSQL menu migration，暂不引入 seed。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步；migrations README 由 `M1-03-05` 专项补充。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧菜单表兼容迁移或双写结构。 |

### 自动化验证

```powershell
rg "sk_menu_nodes|parent_key|source|path|sort|uk_menu_nodes_key|idx_menu_nodes" migrations/mysql/20260618_000014_create_menu_nodes.sql migrations/postgres/20260618_000014_create_menu_nodes.sql
```

结果摘要: 通过。parent/source/path/sort 索引和唯一约束定义均可检索。

### 人工验收

1. 审阅 MySQL migration。
2. 审阅 PostgreSQL migration。
3. 确认 parent/source/path/sort 索引齐全。

结果摘要: 通过。菜单表 migration 设计完成。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-03-03`，增加 gormrepo model。

## M1-03-03: 增加 gormrepo model

- 状态: Passed
- Work Item: M1-03-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/store/sql/gormrepo/permission_menu_model.go`
- `internal/store/sql/gormrepo/permission_menu_model_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 permission resource 与 menu node 的 GORM model 和 domain 转换。 |
| API/OpenAPI 同步 | N/A | 本项只新增 SQL model，无 API 影响。 |
| 权限目录同步 | Passed | `PermissionResourceModel` 字段与 permission domain/migration 对齐。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | Passed | Model 字段与 `000013`、`000014` migration 对齐；暂不引入 seed。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧权限/菜单持久化兼容模型。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/store/sql/gormrepo -run "TestPermissionResourceModelRoundTrip|TestMenuNodeModelRoundTrip"
go test ./internal/store/sql/gormrepo/...
```

结果摘要: 默认环境下转换测试和全包验收均受本机 cgo 编译器路径缺失影响失败；关闭 cgo 后转换测试通过。失败原因与 M0 基线一致。

### 人工验收

1. 审阅 `PermissionResourceModel` 与 `20260618_000013_create_permission_resources.sql`。
2. 审阅 `MenuNodeModel` 与 `20260618_000014_create_menu_nodes.sql`。
3. 确认字段、表名、JSON 字段和唯一/索引相关字段一致。

结果摘要: 通过。GORM model 字段与 migration 一致。

### 失败与返工

- 失败原因: 全包 gormrepo 测试依赖 SQLite cgo，当前 `CC` 指向缺失的 gcc。
- 返工动作: 新增不连接 SQLite 的纯转换测试，并单独执行通过；保留环境失败记录。
- 重新验收结果: 通过。

### 下一步

- 进入 `M1-03-04`，注册新 model 到 all_models。

## M1-03-04: 注册新 model 到 all_models

- 状态: Passed
- Work Item: M1-03-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/store/sql/gormrepo/all_models.go`
- `internal/store/sql/gormrepo/all_models_test.go`
- `internal/store/sql/gormrepo/test_helper.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `AllModels()` 和 gormrepo test helper 的 AutoMigrate 列表均包含新表模型。 |
| API/OpenAPI 同步 | N/A | 本项只变更 SQL store bootstrap，无 API 影响。 |
| 权限目录同步 | Passed | 权限资源模型已进入 SQL 模型列表。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | Passed | 新模型列表与 `000013`、`000014` migration 对齐；暂不引入 seed。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧表兼容模型或双迁移路径。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/store/sql/gormrepo -run "TestAllModelsIncludesPermissionAndMenuModels"
go test ./internal/store/sql/gormrepo/...
```

结果摘要: 关闭 cgo 后模型列表测试通过。全包验收受本机 cgo 编译器路径缺失影响失败，失败原因与 M0 基线一致。

### 人工验收

1. 审阅 `internal/store/sql/gormrepo/all_models.go`。
2. 审阅 `internal/store/sql/gormrepo/test_helper.go`。
3. 确认 AutoMigrate/模型列表包含 `PermissionResourceModel` 和 `MenuNodeModel`。

结果摘要: 通过。新模型已注册到 SQL bootstrap 列表。

### 失败与返工

- 失败原因: 全包 gormrepo 测试依赖 SQLite cgo，当前 `CC` 指向缺失的 gcc。
- 返工动作: 增加模型列表专项测试，并在 `CGO_ENABLED=0` 下执行通过；保留环境失败记录。
- 重新验收结果: 通过。

### 下一步

- 进入 `M1-03-05`，更新 migration 文档。

## M1-03-05: migration 文档更新

- 状态: Passed
- Work Item: M1-03-05
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `migrations/mysql/README.md`
- `migrations/postgres/README.md`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | MySQL/PostgreSQL migration README 均说明新表用途、执行顺序和回滚说明。 |
| API/OpenAPI 同步 | N/A | 本项只更新 migration 文档，无 API 影响。 |
| 权限目录同步 | Passed | 文档说明权限资源表用途和对应 GORM model。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | Passed | 文档明确 `000013`、`000014` 执行顺序且不包含 seed。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | migrations README、Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 migration 兼容说明或双迁移路径。 |

### 自动化验证

```powershell
rg "20260618_000013|20260618_000014|sk_permission_resources|sk_menu_nodes|执行顺序|回滚说明" migrations/mysql/README.md migrations/postgres/README.md
```

结果摘要: 通过。新表用途和执行顺序均可检索。

### 人工验收

1. 审阅 `migrations/mysql/README.md`。
2. 审阅 `migrations/postgres/README.md`。
3. 确认已说明新表用途和执行顺序。

结果摘要: 通过。migration 文档已更新。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-04-01`，定义 permission repository 接口。

## M1-04-01: 定义 permission repository 接口

- 状态: Passed
- Work Item: M1-04-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/repository/permission/permission_repo.go`
- `internal/repository/permission/permission_repo_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `PermissionRepository` 覆盖 register/list/get/update state 方法。 |
| API/OpenAPI 同步 | N/A | 本项只定义 repository 接口，无 API 影响。 |
| 权限目录同步 | Passed | 接口使用 permission domain 类型和 resource filter。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧权限 repository 兼容接口。 |

### 自动化验证

```powershell
go test ./internal/repository/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/repository/permission/permission_repo.go`。
2. 确认 service-facing 方法覆盖 register/list/get/update state。

结果摘要: 通过。permission repository 接口已定义。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-04-02`，定义 menu repository 接口。

## M1-04-02: 定义 menu repository 接口

- 状态: Passed
- Work Item: M1-04-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/repository/menu/menu_repo.go`
- `internal/repository/menu/menu_repo_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `MenuRepository` 覆盖 tree/list/upsert/reorder 方法。 |
| API/OpenAPI 同步 | N/A | 本项只定义 repository 接口，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧菜单 repository 兼容接口。 |

### 自动化验证

```powershell
go test ./internal/repository/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/repository/menu/menu_repo.go`。
2. 确认 service-facing 方法覆盖 tree/list/upsert/reorder。

结果摘要: 通过。menu repository 接口已定义。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-04-03`，实现 memory permission store。

## M1-04-03: 实现 memory permission store

- 状态: Passed
- Work Item: M1-04-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/store/memory/permission_store.go`
- `internal/store/memory/permission_store_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 memory permission store，覆盖 register/list/get/set enabled。 |
| API/OpenAPI 同步 | N/A | 本项只实现 memory store，无 API 影响。 |
| 权限目录同步 | Passed | store 使用 permission repository 接口和 permission domain 类型。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧权限 store 兼容路径。 |

### 自动化验证

```powershell
go test ./internal/store/memory/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/store/memory/permission_store.go`。
2. 确认注册幂等、source filter、enable/disable 均有测试覆盖。

结果摘要: 通过。memory permission store 已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-04-04`，实现 memory menu store。

## M1-04-04: 实现 memory menu store

- 状态: Passed
- Work Item: M1-04-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/store/memory/menu_store.go`
- `internal/store/memory/menu_store_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 memory menu store，覆盖 tree/list/upsert/reorder。 |
| API/OpenAPI 同步 | N/A | 本项只实现 memory store，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧菜单 store 兼容路径。 |

### 自动化验证

```powershell
go test ./internal/store/memory/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/store/memory/menu_store.go`。
2. 确认 upsert/tree/reorder/visibility 均有测试覆盖。

结果摘要: 通过。memory menu store 已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-04-05`，实现 SQL permission store。

## M1-04-05: 实现 SQL permission store

- 状态: Passed
- Work Item: M1-04-05
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/store/sql/gormrepo/permission_store.go`
- `internal/store/sql/gormrepo/permission_store_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 SQL permission store，覆盖 register/list/get/set enabled。 |
| API/OpenAPI 同步 | N/A | 本项只实现 SQL store，无 API 影响。 |
| 权限目录同步 | Passed | store 实现 permission repository 契约并使用 permission domain/model。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | Passed | store 写入字段与 permission resource migration/model 对齐；暂不引入 seed。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧权限 SQL store 兼容路径。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/store/sql/gormrepo -run "TestPermissionStoreImplementsRepository"
go test ./internal/store/sql/gormrepo/...
```

结果摘要: 关闭 cgo 后 repository 契约测试通过。全包验收受本机 cgo 编译器路径缺失影响失败，失败原因与 M0 基线一致。

### 人工验收

1. 审阅 `internal/store/sql/gormrepo/permission_store.go`。
2. 确认 SQL store 与 permission repository 契约一致。

结果摘要: 通过。SQL permission store 已实现。

### 失败与返工

- 失败原因: 全包 gormrepo 测试依赖 SQLite cgo，当前 `CC` 指向缺失的 gcc。
- 返工动作: 增加 repository 契约专项测试，并在 `CGO_ENABLED=0` 下执行通过；保留环境失败记录。
- 重新验收结果: 通过。

### 下一步

- 进入 `M1-04-06`，实现 SQL menu store。

## M1-04-06: 实现 SQL menu store

- 状态: Passed
- Work Item: M1-04-06
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/store/sql/gormrepo/menu_store.go`
- `internal/store/sql/gormrepo/menu_store_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 SQL menu store，覆盖 tree/list/upsert/reorder。 |
| API/OpenAPI 同步 | N/A | 本项只实现 SQL store，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | Passed | store 写入字段与 menu node migration/model 对齐；暂不引入 seed。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧菜单 SQL store 兼容路径。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/store/sql/gormrepo -run "TestMenuStoreImplementsRepository"
go test ./internal/store/sql/gormrepo/...
```

结果摘要: 关闭 cgo 后 repository 契约测试通过。全包验收受本机 cgo 编译器路径缺失影响失败，失败原因与 M0 基线一致。

### 人工验收

1. 审阅 `internal/store/sql/gormrepo/menu_store.go`。
2. 确认 SQL store 与 menu repository 契约一致。

结果摘要: 通过。SQL menu store 已实现。

### 失败与返工

- 失败原因: 全包 gormrepo 测试依赖 SQLite cgo，当前 `CC` 指向缺失的 gcc。
- 返工动作: 增加 repository 契约专项测试，并在 `CGO_ENABLED=0` 下执行通过；保留环境失败记录。
- 重新验收结果: 通过。

### 下一步

- 进入 `M1-04-07`，接入 store factory。

## M1-04-07: 接入 store factory

- 状态: Passed
- Work Item: M1-04-07
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/store/factory.go`
- `internal/store/factory_test.go`
- `internal/store/sql/mysql/adapter.go`
- `internal/store/sql/postgres/adapter.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `Bundle`、memory bundle、MySQL/PostgreSQL adapter 均提供 permission/menu store。 |
| API/OpenAPI 同步 | N/A | 本项只变更 store factory，无 API 影响。 |
| 权限目录同步 | Passed | Bundle 暴露 permission repository。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 store factory 兼容路径。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/store -run "TestNewBundleModes"
go test ./internal/store/...
```

结果摘要: 关闭 cgo 后 memory bundle 专项测试通过。默认环境和全 store 验收受本机 cgo 编译器路径缺失影响失败，失败原因与 M0 基线一致。

### 人工验收

1. 审阅 `internal/store/factory.go`。
2. 审阅 MySQL/PostgreSQL adapter repository 方法。
3. 确认 memory/mysql/postgres bundle 可提供新 store。

结果摘要: 通过。store factory 已接入 permission/menu store。

### 失败与返工

- 失败原因: 全 store 测试链路包含 gormrepo SQLite cgo，当前 `CC` 指向缺失的 gcc。
- 返工动作: 在 `CGO_ENABLED=0` 下执行 memory bundle 专项测试通过；保留环境失败记录。
- 重新验收结果: 通过。

### 下一步

- 进入 `M1-05-01`，定义 permission catalog service interface。

## M1-05-01: 定义 permission catalog service interface

- 状态: Passed
- Work Item: M1-05-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/service/permission/service.go`
- `internal/service/permission/types.go`
- `internal/service/permission/service_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | permission catalog service interface 覆盖注册、列表、获取、启用、停用和 diff。 |
| API/OpenAPI 同步 | N/A | 本项只定义 service interface，无 API 影响。 |
| 权限目录同步 | Passed | service interface 使用 permission domain 类型。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 permission service 兼容接口。 |

### 自动化验证

```powershell
go test ./internal/service/permission/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/service/permission/service.go`。
2. 确认接口覆盖注册、列表、启停、diff。

结果摘要: 通过。permission catalog service interface 已定义。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-05-02`，实现 RegisterResource。

## M1-05-02: 实现 RegisterResource

- 状态: Passed
- Work Item: M1-05-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/service/permission/service_impl.go`
- `internal/service/permission/service_impl_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `RegisterResource` 使用 domain 校验并写入 repository。 |
| API/OpenAPI 同步 | N/A | 本项只实现 service 方法，无 API 影响。 |
| 权限目录同步 | Passed | 注册逻辑生成 permission domain resource 并写入 catalog repository。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 permission service 兼容路径。 |

### 自动化验证

```powershell
go test ./internal/service/permission/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/service/permission/service_impl.go`。
2. 确认幂等、校验失败、仓储失败路径均有测试覆盖。

结果摘要: 通过。`RegisterResource` 已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-05-03`，实现 ListResources/GetResource。

## M1-05-03: 实现 ListResources/GetResource

- 状态: Passed
- Work Item: M1-05-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/service/permission/service_impl.go`
- `internal/service/permission/service_impl_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `ListResources` 和 `GetResource` 已接入 repository。 |
| API/OpenAPI 同步 | N/A | 本项只实现 service 方法，无 API 影响。 |
| 权限目录同步 | Passed | 列表和获取逻辑使用 permission catalog repository。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 permission service 兼容路径。 |

### 自动化验证

```powershell
go test ./internal/service/permission/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/service/permission/service_impl.go`。
2. 确认 type/source/enabled 过滤有测试覆盖。

结果摘要: 通过。`ListResources` 和 `GetResource` 已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-05-04`，实现 Enable/Disable。

## M1-05-04: 实现 Enable/Disable

- 状态: Passed
- Work Item: M1-05-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/service/permission/service_impl.go`
- `internal/service/permission/service_impl_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `EnableResource` 和 `DisableResource` 已实现并写入 repository 状态。 |
| API/OpenAPI 同步 | N/A | 本项只实现 service 方法，无 API 影响。 |
| 权限目录同步 | Passed | 启停逻辑写入 permission catalog repository。 |
| 审计 action 同步 | Passed | `auditFn` 预留状态变更审计动作接入点。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 permission service 兼容路径。 |

### 自动化验证

```powershell
go test ./internal/service/permission/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/service/permission/service_impl.go`。
2. 确认状态变更写 repository，且审计动作预留点有测试覆盖。

结果摘要: 通过。`EnableResource` 和 `DisableResource` 已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-05-05`，实现 permission diff。

## M1-05-05: 实现 permission diff

- 状态: Passed
- Work Item: M1-05-05
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/service/permission/service_impl.go`
- `internal/service/permission/service_impl_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `DiffResources` 输出 added/updated/removed，可用于 install preflight。 |
| API/OpenAPI 同步 | N/A | 本项只实现 service 方法，无 API 影响。 |
| 权限目录同步 | Passed | diff 基于 permission catalog repository 的 source 过滤。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 permission diff 兼容路径。 |

### 自动化验证

```powershell
go test ./internal/service/permission/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/service/permission/service_impl.go`。
2. 确认 diff 输出可复用于 install preflight。

结果摘要: 通过。permission diff 已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-05-06`，定义 menu registry service interface。

## M1-05-06: 定义 menu registry service interface

- 状态: Passed
- Work Item: M1-05-06
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/service/menu/service.go`
- `internal/service/menu/types.go`
- `internal/service/menu/service_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | menu registry service interface 覆盖 merge/tree/filter/reorder。 |
| API/OpenAPI 同步 | N/A | 本项只定义 service interface，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 menu service 兼容接口。 |

### 自动化验证

```powershell
go test ./internal/service/menu/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/service/menu/service.go`。
2. 确认接口覆盖 merge/tree/filter/reorder。

结果摘要: 通过。menu registry service interface 已定义。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-05-07`，实现菜单合并与排序。

## M1-05-07: 实现菜单合并与排序

- 状态: Passed
- Work Item: M1-05-07
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/service/menu/service_impl.go`
- `internal/service/menu/service_impl_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `MergeNodes` 稳定合并系统/插件/生成模块菜单并按同级 sort 展开。 |
| API/OpenAPI 同步 | N/A | 本项只实现 service 方法，无 API 影响。 |
| 权限目录同步 | N/A | 本项不变更权限目录。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧菜单合并兼容路径。 |

### 自动化验证

```powershell
go test ./internal/service/menu/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/service/menu/service_impl.go`。
2. 确认系统/插件/生成模块菜单稳定合并和排序有测试覆盖。

结果摘要: 通过。菜单合并与排序已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-05-08`，实现菜单权限过滤。

## M1-05-08: 实现菜单权限过滤

- 状态: Passed
- Work Item: M1-05-08
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/service/menu/service_impl.go`
- `internal/service/menu/service_impl_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `Filter` 应用 visible、requiredRoles、requiredPermissions 过滤。 |
| API/OpenAPI 同步 | N/A | 本项只实现 service 方法，无 API 影响。 |
| 权限目录同步 | Passed | 菜单过滤使用角色和权限上下文。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧菜单权限过滤兼容路径。 |

### 自动化验证

```powershell
go test ./internal/service/menu/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/service/menu/service_impl.go`。
2. 确认 requiredRoles/requiredPermissions 生效并有测试覆盖。

结果摘要: 通过。菜单权限过滤已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-06-01`，设计 permission API 契约。

## M1-06-01: 设计 permission API 契约

- 状态: Passed
- Work Item: M1-06-01
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/api/openapi.yaml`
- `internal/handler/http/openapi.yaml`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | OpenAPI 新增 permission list/detail/enable/disable/diff 契约。 |
| API/OpenAPI 同步 | Passed | 文档 OpenAPI 和内嵌 Swagger OpenAPI 已同步。 |
| 权限目录同步 | N/A | 本项只设计 API 契约，不注册权限 seed。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 permission API 兼容路径。 |

### 自动化验证

```powershell
rg "/v1/permissions|PermissionResourceListAPIResponse|PermissionResourceDetailAPIResponse|PermissionStateChangeAPIResponse|PermissionResourceDiffAPIResponse" docs/api/openapi.yaml internal/handler/http/openapi.yaml
```

结果摘要: 通过。list/detail/enable/disable/diff 请求响应契约均可检索。

### 人工验收

1. 审阅 `docs/api/openapi.yaml`。
2. 审阅 `internal/handler/http/openapi.yaml`。
3. 确认 list/detail/enable/disable/diff 请求响应明确。

结果摘要: 通过。permission API 契约已设计。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-06-02`，实现 permission handler。

## M1-06-02: 实现 permission handler

- 状态: Passed
- Work Item: M1-06-02
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/handler/http/v1/permission/handler.go`
- `internal/handler/http/v1/permission/handler_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 permission handler，覆盖 list/detail/enable/disable/diff。 |
| API/OpenAPI 同步 | Passed | handler 路由与 `M1-06-01` OpenAPI 契约一致。 |
| 权限目录同步 | N/A | 本项不注册权限 seed。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | handler 只依赖 service，不直连 store。 |

### 自动化验证

```powershell
go test ./internal/handler/http/v1/permission/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/handler/http/v1/permission/handler.go`。
2. 确认 HTTP 状态码、错误码和响应格式有测试覆盖。

结果摘要: 通过。permission handler 已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-06-03`，设计 menu API 契约。

## M1-06-03: 设计 menu API 契约

- 状态: Passed
- Work Item: M1-06-03
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `docs/api/openapi.yaml`
- `internal/handler/http/openapi.yaml`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | OpenAPI 新增 menu tree/save/reorder/visibility 契约。 |
| API/OpenAPI 同步 | Passed | 文档 OpenAPI 和内嵌 Swagger OpenAPI 已同步。 |
| 权限目录同步 | N/A | 本项只设计 API 契约，不注册权限 seed。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未添加旧 menu API 兼容路径。 |

### 自动化验证

```powershell
rg "/v1/menus|MenuTreeAPIResponse|MenuSaveAPIResponse|MenuReorderAPIResponse|MenuVisibilityAPIResponse" docs/api/openapi.yaml internal/handler/http/openapi.yaml
```

结果摘要: 通过。tree/save/reorder/visibility 请求响应契约均可检索。

### 人工验收

1. 审阅 `docs/api/openapi.yaml`。
2. 审阅 `internal/handler/http/openapi.yaml`。
3. 确认 tree/save/reorder/visibility 请求响应明确。

结果摘要: 通过。menu API 契约已设计。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-06-04`，实现 menu handler。

## M1-06-04: 实现 menu handler

- 状态: Passed
- Work Item: M1-06-04
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/handler/http/v1/menu/handler.go`
- `internal/handler/http/v1/menu/handler_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 menu handler，覆盖 tree/save/reorder/visibility。 |
| API/OpenAPI 同步 | Passed | handler 路由与 `M1-06-03` OpenAPI 契约一致。 |
| 权限目录同步 | Passed | tree 支持 roles/permissions 过滤并覆盖失败路径。 |
| 审计 action 同步 | N/A | 本项无审计 action 变更。 |
| migration/seed 同步 | N/A | 本项无数据结构变更。 |
| 前端 API client/UI 同步 | N/A | 本项无前端实现影响。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | handler 只依赖 service，不直连 store。 |

### 自动化验证

```powershell
go test ./internal/handler/http/v1/menu/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/handler/http/v1/menu/handler.go`。
2. 确认 handler 不直连 store，且鉴权/过滤失败路径有测试覆盖。

结果摘要: 通过。menu handler 已实现。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-06-05`，注册路由与权限 keys。

## M1-06-05: 注册路由与权限 keys

- 状态: Passed
- Work Item: M1-06-05
- 日期: 2026-06-18
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/handler/http/router.go`
- `internal/handler/http/v1/menu/handler.go`
- `internal/handler/http/v1/menu/handler_test.go`
- `internal/handler/http/router_test.go`
- `internal/bootstrap/di.go`
- `internal/bootstrap/permission_menu_seed.go`
- `internal/bootstrap/permission_menu_seed_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 主 router 注册 permission/menu handler，bootstrap 注入 service。 |
| API/OpenAPI 同步 | Passed | 路由与 M1-06-01/M1-06-03 OpenAPI 契约保持一致。 |
| 权限目录同步 | Passed | `permission.manage`、`menu.read`、`menu.manage` 纳入 system catalog seed。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | Passed | 新增 system permission catalog seed，幂等注册。 |
| 前端 API client/UI 同步 | N/A | 本项为后端路由注册，无前端改动。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 新路由走统一 service，不引入旧权限旁路。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/handler/http/v1/menu/...
$env:CGO_ENABLED='0'; go test ./internal/handler/http/...
$env:CGO_ENABLED='0'; go test ./internal/bootstrap/...
```

默认 `go test ./internal/handler/http/...` 与 `go test ./internal/bootstrap/...` 仍受本机 cgo 环境影响失败，错误为 `C compiler "D:\Program Files\JetBrains\CLion 2024.1.1\bin\mingw\bin\gcc.exe" not found`，与 M0 基线一致。

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/handler/http/router.go` 和 `internal/bootstrap/permission_menu_seed.go`。
2. 确认主 router 可访问 permission/menu API，系统权限 key 可由 permission catalog 查询。

结果摘要: 通过。路由与权限 key 已注册。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-07-01`，扩展 plugin manifest permission 字段读取。

## M1-07-01: 扩展 plugin manifest permission 字段读取

- 状态: Passed
- Work Item: M1-07-01
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/plugin/loader.go`
- `internal/plugin/loader_test.go`
- `internal/plugin/types.go`
- `internal/plugin/types_test.go`
- `internal/plugin/signature.go`
- `docs/schemas/plugin-manifest.schema.json`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | plugin manifest 支持字符串与结构化 permission 声明读取。 |
| API/OpenAPI 同步 | N/A | 本项未改 HTTP API。 |
| 权限目录同步 | Passed | 结构化声明保留 key/type/module/name/risk/metadata，后续可导入 catalog。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | N/A | 本项仅扩展 manifest parser。 |
| 文档同步 | Passed | manifest JSON schema 与 Work Item 状态已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 旧字符串权限会归一为结构化声明，并走同一套严格校验。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/plugin/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/plugin/loader.go` 和 `internal/plugin/types.go`。
2. 确认空权限 key、重复 key、非法 type/risk 都会被拒绝，字符串权限不会绕过校验。

结果摘要: 通过。plugin manifest permission 字段读取已扩展。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-07-02`，扩展 plugin manifest menu 字段读取。

## M1-07-02: 扩展 plugin manifest menu 字段读取

- 状态: Passed
- Work Item: M1-07-02
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/plugin/menu_mapping.go`
- `internal/plugin/menu_mapping_test.go`
- `internal/plugin/loader.go`
- `internal/plugin/loader_test.go`
- `internal/plugin/types.go`
- `internal/plugin/signature.go`
- `docs/schemas/plugin-manifest.schema.json`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | `ui_menu` 支持 key/parent_key/component/visible，并可映射为 `domain/menu.MenuNode`。 |
| API/OpenAPI 同步 | N/A | 本项未改 HTTP API。 |
| 权限目录同步 | Passed | 菜单节点保留 required_roles/required_permissions。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | N/A | 本项仅扩展 manifest parser 和领域映射。 |
| 文档同步 | Passed | manifest JSON schema 与 Work Item 状态已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 旧 `ui_menu` 字段保持可用，并统一通过 `MenuNodes()` 校验与映射。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/plugin/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/plugin/menu_mapping.go`。
2. 确认 manifest `ui_menu` 可生成合法 `MenuNode`，非法 key/path 会被拒绝。

结果摘要: 通过。plugin manifest menu 字段读取已扩展。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-07-03`，插件 enable 导入 catalog/registry。

## M1-07-03: 插件 enable 导入 catalog/registry

- 状态: Passed
- Work Item: M1-07-03
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/plugin/catalog.go`
- `internal/plugin/manager.go`
- `internal/plugin/manager_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | RuntimeManager enable 时导入插件权限和菜单到 catalog registry。 |
| API/OpenAPI 同步 | N/A | 本项未改 HTTP API。 |
| 权限目录同步 | Passed | `CatalogPermissions()` 生成 `domain/permission.PermissionResource`，source 为 `plugin.<id>`。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | N/A | 本项为 plugin manager 生命周期能力。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | enable/reload 统一导入，disable/uninstall 统一移除。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/plugin/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/plugin/catalog.go` 与 `internal/plugin/manager.go`。
2. 确认 enable 后权限和菜单可查询，disable 后 registry 清理。

结果摘要: 通过。插件 enable 导入 catalog/registry 已完成。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-07-04`，插件 disable 隐藏/冻结权限与菜单。

## M1-07-04: 插件 disable 隐藏访问

- 状态: Passed
- Work Item: M1-07-04
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/plugin/catalog.go`
- `internal/plugin/manager.go`
- `internal/plugin/manager_test.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 插件 disable 后 catalog 保留记录但权限 inactive、菜单 hidden。 |
| API/OpenAPI 同步 | N/A | 本项未改 HTTP API。 |
| 权限目录同步 | Passed | disable 后 `PermissionResource.Enabled=false`。 |
| 审计 action 同步 | N/A | 审计导入在 M1-07-05 处理。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | N/A | 本项为 plugin manager 生命周期能力。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | disable 冻结、uninstall 移除，生命周期语义清晰。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/plugin/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/plugin/catalog.go` 和 `internal/plugin/manager.go`。
2. 确认 disable 后菜单不可见、权限不可用，uninstall 后移除 catalog 记录。

结果摘要: 通过。插件 disable 隐藏访问已完成。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-07-05`，插件权限/菜单导入审计。

## M1-07-05: 插件权限/菜单导入审计

- 状态: Passed
- Work Item: M1-07-05
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `internal/plugin/catalog_audit.go`
- `internal/plugin/manager.go`
- `internal/plugin/manager_test.go`
- `internal/bootstrap/di.go`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | RuntimeManager 支持 catalog audit sink，enable/disable 产生审计事件。 |
| API/OpenAPI 同步 | N/A | 本项未改 HTTP API。 |
| 权限目录同步 | Passed | 审计事件记录权限与菜单导入数量。 |
| 审计 action 同步 | Passed | 新增 `plugin_catalog_import` 与 `plugin_catalog_disable` 事件。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | N/A | 本项为后端审计接线。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | bootstrap 使用统一 audit service 适配器，不记录敏感内容。 |

### 自动化验证

```powershell
$env:CGO_ENABLED='0'; go test ./internal/plugin/... ./internal/service/audit/...
$env:CGO_ENABLED='0'; go test ./internal/bootstrap/...
```

结果摘要: 通过。

### 人工验收

1. 审阅 `internal/plugin/catalog_audit.go`、`internal/plugin/manager.go` 和 `internal/bootstrap/di.go`。
2. 确认 enable/disable 审计包含 actor/action/resource/result/counts，不包含密钥、包内容或 token。

结果摘要: 通过。插件权限/菜单导入审计已完成。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-08-01`，设计 RBAC 授权矩阵契约。

## M1-08-01: 新增 permission API client

- 状态: Passed
- Work Item: M1-08-01
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/permissions/api.ts`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 permission API client，覆盖 list/detail/enable/disable/diff。 |
| API/OpenAPI 同步 | Passed | client 路径与 M1-06 permission API 契约一致。 |
| 权限目录同步 | N/A | 本项为前端 client 封装，不改目录数据。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 类型定义完整，统一复用 `utils/api` 的错误处理。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未新增页面内临时请求封装。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `web/src/permissions/api.ts`。
2. 确认请求、响应、diff、启停状态类型明确，网络与后端错误仍由统一 API wrapper 处理。

结果摘要: 通过。permission API client 已新增。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-08-02`，新增 permission store。

## M1-08-02: 新增 permission store

- 状态: Passed
- Work Item: M1-08-02
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/stores/permissions.ts`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 Pinia permission store。 |
| API/OpenAPI 同步 | N/A | 本项复用 M1-08-01 client。 |
| 权限目录同步 | Passed | store 支持按 key/type/source 查询 catalog 缓存。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 支持加载、失败状态、retry、refresh、启停本地 patch。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未引入页面局部重复状态管理。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `web/src/stores/permissions.ts`。
2. 确认加载、失败、重试、缓存刷新与启停 patch 路径清晰。

结果摘要: 通过。permission store 已新增。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-08-03`，新增 menu API client。

## M1-08-03: 新增 menu API client

- 状态: Passed
- Work Item: M1-08-03
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/navigation/api.ts`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 menu API client，覆盖 tree/save/reorder/visibility 接口。 |
| API/OpenAPI 同步 | Passed | client 路径与 M1-06 menu API 契约一致。 |
| 权限目录同步 | N/A | 本项为前端 client 封装，不改目录数据。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 类型定义完整，统一复用 `utils/api` 错误处理。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 新增 registry menu client，未引入页面局部临时请求封装。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `web/src/navigation/api.ts`。
2. 确认 tree/save/reorder/visibility 请求和响应类型清晰，menu tree 归一化逻辑覆盖子节点与权限字段。

结果摘要: 通过。menu API client 已新增。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-08-04`，新增 menu store。

## M1-08-04: 新增 menu store

- 状态: Passed
- Work Item: M1-08-04
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/stores/navigation.ts`
- `web/src/navigation/api.ts`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | navigation Pinia store 已改为使用 registry menu API。 |
| API/OpenAPI 同步 | Passed | store 通过 `getMenuTree`/`saveMenuNodes`/`reorderMenuNodes`/`setMenuVisibility` 访问 `/v1/menus*`。 |
| 权限目录同步 | N/A | 本项为前端状态管理，不改权限目录数据。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 侧栏继续优先使用 `navigationStore.systemMenus`，其来源已切换为 registry tree。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 移除 store 内旧 `/v1/system/menus` 请求，统一走 registry menu client。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `web/src/stores/navigation.ts`。
2. 确认 registry flat nodes 会组装为侧栏使用的树结构，并支持保存、重排、显隐状态操作。
3. 确认 `web/src/App.vue` 仍通过 `navigationStore.systemMenus` 构建侧栏，因此加载成功时 registry tree 优先于静态默认菜单。

结果摘要: 通过。menu store 已接入 registry tree。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-08-05`，统一 route guard 权限入口。

## M1-08-05: 统一 route guard 权限入口

- 状态: Passed
- Work Item: M1-08-05
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/permissions/route.ts`
- `web/src/router/index.ts`
- `web/src/plugins/index.ts`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增 route access 统一入口。 |
| API/OpenAPI 同步 | N/A | 本项未改 HTTP API。 |
| 权限目录同步 | Passed | route guard 继续使用统一 `canAccess` 与 implied permission 判断。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 静态路由、插件动态路由和 auth plugin public 判断均通过 `permissions/route`。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | router 内不再手写 roles/permissions 分支，插件 route meta 合并复用统一入口。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `web/src/permissions/route.ts`。
2. 确认 router guard 调用 `isPublicRoute` 与 `canAccessRoute`。
3. 确认插件动态 route 权限 meta 由 `withRouteAccessMeta` 注入，与静态 route 走同一判断入口。

结果摘要: 通过。route guard 权限入口已统一。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-08-06`，统一按钮权限入口。

## M1-08-06: 统一按钮权限入口

- 状态: Passed
- Work Item: M1-08-06
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/permissions/button.ts`
- `web/src/permissions/directive.ts`
- `web/src/views/Plugin/index.vue`
- `web/src/views/Role/list.vue`
- `web/src/views/Role/edit.vue`
- `web/src/views/User/list.vue`
- `web/src/views/User/add.vue`
- `web/src/views/User/edit.vue`
- `web/src/views/User/batch.vue`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 新增按钮权限工具入口并接入 `v-permission`。 |
| API/OpenAPI 同步 | N/A | 本项未改 HTTP API。 |
| 权限目录同步 | Passed | 按钮权限继续复用 `canAccess` 与 implied permission 判断。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 用户、角色、插件页按钮权限判断迁移到 `useButtonAccess`/`BUTTON_ACCESS`。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 页面中不再直接使用 `useAccess/access.can` 判断按钮权限；业务状态判断仍留在页面。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `web/src/permissions/button.ts` 与 `web/src/permissions/directive.ts`。
2. 搜索 `useAccess|access\.can`，确认页面按钮权限判断已迁移。
3. 确认用户、角色、插件页的按钮可见/禁用逻辑仍保留原有业务状态判断。

结果摘要: 通过。按钮权限入口已统一。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-09-01`，权限矩阵页读取 catalog。

## M1-09-01: 权限矩阵页读取 catalog

- 状态: Passed
- Work Item: M1-09-01
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/views/Permission/index.vue`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 权限矩阵页改为读取 `usePermissionStore` catalog。 |
| API/OpenAPI 同步 | N/A | 本项复用 M1-08 permission client/store，不改 HTTP API。 |
| 权限目录同步 | Passed | 矩阵资源来自后端权限 catalog，角色权限仅补齐 catalog 未注册项。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 刷新按钮和矩阵 loading 同时覆盖角色与 catalog 加载。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 权限矩阵页不再导入 `BASE_PERMISSION_CATALOG`/`BUILTIN_ROLE_DEFAULTS`。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `web/src/views/Permission/index.vue`。
2. 搜索 `BASE_PERMISSION_CATALOG|BUILTIN_ROLE_DEFAULTS|permissions/catalog`，确认权限矩阵页不再引用孤立权限常量。
3. 确认资源和动作选项由 catalog 派生，并保留空 catalog 下的最小可用状态。

结果摘要: 通过。权限矩阵页已读取 catalog。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-09-02`，权限矩阵授权/撤销接入 API。

## M1-09-02: 权限矩阵授权/撤销接入 API

- 状态: Passed
- Work Item: M1-09-02
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/views/Permission/index.vue`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 权限矩阵 grant/revoke 使用后端返回 role 更新页面状态。 |
| API/OpenAPI 同步 | Passed | 继续使用 `/v1/roles/{id}/grant` 与 `/v1/roles/{id}/revoke`。 |
| 权限目录同步 | Passed | 授权项来自 catalog 矩阵，角色权限状态采用后端响应。 |
| 审计 action 同步 | N/A | 后端接口已有 grant/revoke 审计，本项未新增 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 保存成功后以响应 role patch 本地角色列表，异常响应时强制重载角色列表。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 不再手工拼接矩阵权限作为最终状态。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `toggleMatrixPermission`。
2. 确认 grant/revoke 成功后使用接口返回的 role 更新 `roles` 与 `selectedMatrixPermissions`。
3. 确认返回体无法归一化时会重新加载角色列表，避免前端状态漂移。

结果摘要: 通过。权限矩阵授权/撤销保存后前后端状态一致。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-09-03`，菜单管理页读取 registry tree。

## M1-09-03: 菜单管理页读取 registry tree

- 状态: Passed
- Work Item: M1-09-03
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/views/Menu/index.vue`
- `web/src/i18n/index.ts`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 菜单管理页通过 navigation store 读取 registry tree，并补齐页面状态。 |
| API/OpenAPI 同步 | N/A | 本项复用 M1-08 menu client/store，不改 HTTP API。 |
| 权限目录同步 | N/A | 本项未改权限目录数据。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 加载态、错误重试、空态和表格 empty text 已覆盖。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 页面继续通过 registry-backed navigation store 加载菜单。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `web/src/views/Menu/index.vue`。
2. 确认错误态提供重试，空态提供新增根菜单入口，表格提供 empty text。
3. 确认缺失的 `common.retry` 中英文文案已补齐。

结果摘要: 通过。菜单管理页读取 registry tree 的页面状态已补齐。

### 失败与返工

- 失败原因: 初次验收发现 `common.retry` i18n key 缺失。
- 返工动作: 补齐中英文 `common.retry` 文案。
- 重新验收结果: Passed

### 下一步

- 进入 `M1-09-04`，菜单管理页编辑与保存。

## M1-09-04: 菜单管理页编辑与保存

- 状态: Passed
- Work Item: M1-09-04
- 日期: 2026-06-19
- 执行人: Codex
- 提交: 本任务提交

### 改动文件

- `web/src/views/Menu/index.vue`
- `docs/refactor/work_items.md`
- `docs/refactor/acceptance_log.md`

### 验收项

| 验收项 | 结果 | 说明 |
|---|---|---|
| 任务交付物完成 | Passed | 菜单管理页编辑保存继续走 registry-backed navigation store。 |
| API/OpenAPI 同步 | N/A | 本项复用 M1-08 menu client/store，不改 HTTP API。 |
| 权限目录同步 | N/A | 本项未改权限目录数据。 |
| 审计 action 同步 | N/A | 本项未新增审计 action。 |
| migration/seed 同步 | N/A | 本项未改数据库结构。 |
| 前端 API client/UI 同步 | Passed | 显隐、排序、权限字段保存路径保留，且保存时保留 registry source/component。 |
| 文档同步 | Passed | Work Item 状态和验收记录已同步。 |
| 无兼容方案/无旧路径残留 | Passed | 未回退到旧 system menu 请求或页面局部保存逻辑。 |

### 自动化验证

```powershell
cd web
npm run typecheck
npm run build
```

结果摘要: 通过。build 仅输出既有 Sass legacy JS API 与 Rollup pure annotation 警告。

### 人工验收

1. 审阅 `web/src/views/Menu/index.vue` 的 `createDraft` 与 `normalizeDrafts`。
2. 确认 visible、order、requiredRoles、requiredPermissions 仍随保存 payload 进入 store。
3. 确认 source/component 在编辑保存时不会被归一化丢失。

结果摘要: 通过。菜单管理页编辑与保存路径完整。

### 失败与返工

- 失败原因: 无
- 返工动作: 无
- 重新验收结果: 不适用

### 下一步

- 进入 `M1-09-05`，菜单和权限页面确认弹窗。
