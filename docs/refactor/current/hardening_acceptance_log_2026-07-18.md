# Skoll Hardening Acceptance Log

> Batch: `skoll-hardening-2026-07-18`
> Append evidence only after an individual Work Item reaches Review and passes its acceptance criteria.
> Failed acceptance must be recorded and returned through `Failed -> Doing` before retry.

## Acceptance Entry Template

```text
## <Work Item ID> <Title>

- Date:
- Status flow:
- Scope:

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |

### Verification Commands

Result:

### Next Step
```

## H1-00 创建强化批次正式任务表

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: 创建父任务表、Work Item 表和验收日志，并将 current 索引切换到新批次。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 正式文件 | Pass | `hardening_task_board_2026-07-18.md`、`hardening_work_items_2026-07-18.md`、`hardening_acceptance_log_2026-07-18.md` 均存在 |
| Work Item 结构 | Pass | 25 个 Work Item 均具备 ID、Skill、任务描述、依赖、交付物、验收标准、验证命令和状态 8 个字段 |
| 父子任务覆盖 | Pass | H1-H6 共 6 个父任务均至少包含一个正式 Work Item |
| 状态与历史边界 | Pass | Review 时 1 项 Review、24 项 Todo；`docs/refactor/old/` 无新增进度变更 |
| 文档链接 | Pass | `docs/refactor/README.md` 中 16 个本地链接全部可解析 |
| Diff 质量 | Pass | `git diff --check` 通过 |

### Verification Commands

Result: 表结构、父子覆盖、状态统计、本地链接、历史目录边界和 diff 检查全部通过。

### Next Step

领取 `H1-01`，定义插件路由权限描述符和解析器契约。

## H1-01 定义插件路由权限描述符与解析器契约

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: 新增插件路由权限描述符、标准化 registry、只读 resolver、冲突校验和中英文诊断。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 标准化与解析 | Pass | method、path、permission、audit action、source 均确定性标准化，未声明 method 和 nil resolver 默认拒绝 |
| 非法声明 | Pass | 非法 method、path、permission、audit action、source 均返回稳定错误码 |
| 路由冲突 | Pass | 标准化后的同 method/path 重复声明在 registry 构造阶段失败，并报告两个来源 |
| 多语言诊断 | Pass | `Error()` 默认返回中文，`Message("en-US")` 返回英文；错误 code 不依赖 locale |
| 真实插件契约 | Pass | 医药 OA manifest 的 119 条路由全部构建为权限描述符，无丢失或冲突 |
| 并发与静态检查 | Pass | `go test -race ./internal/plugin/...` 与 `go vet ./internal/plugin/...` 通过 |
| 影响面 | Pass | 未新增 HTTP endpoint；OpenAPI、migration/seed 和前端契约无需变更；CodeGraph 已同步 |

### Verification Commands

Result: `go test ./internal/plugin/...`、resolver 定向测试、race、vet、coverage、CodeGraph 和 `git diff --check` 全部通过；插件包覆盖率为 78.6%。

### Next Step

领取 `H1-02`，将插件路由权限 resolver 接入认证中间件并保持失败关闭。
