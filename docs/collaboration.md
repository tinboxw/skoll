# Skoll 极简协作通信机制

> 适用范围: Skoll 开源基础建设与后续 M3-M7 功能推进  
> 目标: 让研发主管、开发工程师、产品、测试可以低成本协作，信息不过载，任务可追踪，验收可复查。

## 团队职能

当前最小团队:

| 职能 | 负责人 | 核心职责 | 必须输出 |
| --- | --- | --- | --- |
| 研发主管 | Lead | 架构边界、任务拆分、优先级、风险决策、最终验收 | 任务顺序、验收结论、阻塞决策 |
| 开发工程师 | Dev | 后端、前端、测试、文档同步实现 | 代码、测试结果、验收记录、提交 |
| 产品 | Product | 用户场景、范围边界、验收口径、优先级输入 | 需求说明、验收样例、取舍建议 |
| 测试 | QA | 测试策略、手工验收、回归、失败复现 | 测试记录、缺陷清单、复验结论 |

建议补齐的职能不一定新增专人，可以先由现有成员兼任:

| 补充职能 | 建议归属 | 何时必须出现 | 输出 |
| --- | --- | --- | --- |
| 前端体验/设计 owner | Product + Dev 兼任 | 页面、插件门户、生成器、复杂表单进入开发前 | 页面验收清单、状态覆盖、响应式要求 |
| DevOps/Release owner | Lead + Dev 兼任 | CI、镜像、部署、发布前 | CI 结果、发布清单、回滚说明 |
| Security owner | Lead + QA 兼任 | 文件、插件、权限、签名、上传下载 | 安全检查清单、风险阻断结论 |
| Docs/Community owner | Product + Lead 兼任 | 对外发布、贡献者进入项目时 | README、贡献说明、变更说明 |

原则: 先用“角色帽子”补齐职能，不急着增加管理层级。

## 单一事实来源

| 信息 | 位置 |
| --- | --- |
| 当前入口 | `docs/README.md`、`docs/refactor/README.md` |
| 已完成任务 | `docs/refactor/work_items.md` |
| 下一批任务 | `docs/refactor/next_work_items.md` |
| 验收记录 | `docs/refactor/acceptance_log.md` |
| 架构与边界 | `docs/refactor/architecture_and_execution_plan.md` |
| 前端体验规范 | `docs/refactor/frontend-foundation.md` |
| 前端质量与性能 | `docs/refactor/frontend-quality-performance.md` |

所有沟通结论最终必须落到这些文档、Issue、PR 或提交记录里。聊天只用于快速同步，不作为最终事实来源。

## 工作节奏

| 节奏 | 参与者 | 时间成本 | 输出 |
| --- | --- | ---: | --- |
| 每日异步站会 | 全员 | 每人 3 行 | 昨天完成、今天计划、阻塞 |
| 任务开工确认 | Lead + Dev + QA，必要时 Product | 5-10 分钟 | 任务范围、验收标准、验证命令确认 |
| PR/提交前自检 | Dev | 5 分钟 | 命令结果、影响面、验收记录 |
| 验收复核 | QA + Lead | 10-20 分钟 | Passed/Failed，失败则返工 |
| 每周收口 | 全员 | 30 分钟 | 进度、风险、下周优先级 |

## 任务流转

```text
Todo -> Doing -> Review -> Done
              -> Failed -> Doing
              -> Blocked
```

规则:

1. `Todo`: 任务必须来自 `next_work_items.md` 或正式任务表。
2. `Doing`: 开工前必须确认验收标准和验证命令。
3. `Review`: 代码、测试、文档、验收记录已同步。
4. `Done`: QA/Lead 验收通过，并已提交。
5. `Failed`: 写明失败原因、复现步骤、返工动作。
6. `Blocked`: 写明阻塞人、阻塞条件、下一次检查时间。

## 极简通信模板

### 每日同步

```text
日期:
昨天:
今天:
阻塞:
需要谁响应:
```

### 任务开工

```text
任务:
目标:
范围内:
范围外:
验收标准:
验证命令:
风险:
```

### 验收结果

```text
任务:
结论: Passed / Failed / Blocked
验证命令:
手工验收:
问题:
下一步:
```

## 决策规则

| 决策类型 | Owner | 需要记录到 |
| --- | --- | --- |
| 架构边界 | Lead | `architecture_and_execution_plan.md` 或任务验收记录 |
| 需求范围 | Product + Lead | 任务描述、验收标准 |
| 测试通过/失败 | QA | `acceptance_log.md` |
| 发布/回滚 | Lead + DevOps owner | 发布清单、提交记录 |
| 安全阻断 | Security owner + Lead | 风险说明、失败验收记录 |

硬规则:

1. 不做旧接口、旧数据结构、旧插件格式、旧页面路径兼容方案。
2. API 变更必须同步 OpenAPI、权限、审计、前端 client。
3. 前端页面不能只交静态 UI，必须覆盖 loading、empty、error、no-permission、保存中、窄屏状态。
4. 验收失败不能标记 Done。
5. 每个通过验收的 Work Item 提交一次代码。

## 会议最小化

默认异步优先。只有以下情况开短会:

1. 任务范围不清，且文档无法在 10 分钟内澄清。
2. 验收失败超过一次。
3. 影响架构边界、数据结构、权限、插件生命周期或发布。
4. Product、Dev、QA 对验收结论不一致。

会议必须产出一个结论:

```text
决定:
影响:
责任人:
截止:
更新到:
```

## 推荐补齐的下一步

1. 为 `next_work_items.md` 的 N0 任务创建 Issue 或内部任务卡。
2. 给每个职能指定当前 owner，允许一人多帽。
3. 先跑一轮 N0 发布前冻结与质量门禁，验证这套协作机制是否足够轻。
4. 从 M3 开始，每个 Work Item 使用本文档模板沟通和验收。
