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

## H1-02 在认证中间件执行插件路由权限

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: 将已启用插件 manifest 的路由权限 registry 接入认证中间件，限定公开页面与静态资源边界，并为拒绝结果写入安全审计。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| JWT 边界 | Pass | 插件业务 API 在全局认证关闭或命中额外 skip path 时仍要求 JWT；匿名真实路由返回 HTTP 401 |
| 动态 RBAC | Pass | 已启用插件 manifest 的 method/path/permission 在启动、重载、启用、禁用、卸载后重建只读 registry；权限检查使用声明的 resource/action |
| 失败关闭 | Pass | resolver 缺失、路由未声明、权限键非法均返回 HTTP 403；即使 `super_admin` 也不能绕过路由声明检查 |
| 公开边界 | Pass | 仅插件 `page` 与 `assets/*` 保持公开；隔离运行时的插件页面返回 HTTP 200 |
| 审计与多语言 | Pass | 权限拒绝写入安全审计及稳定 reason；错误消息默认中文，并支持 `Accept-Language: en-US` 英文响应 |
| 权限矩阵 | Pass | 普通用户允许/拒绝、super admin、缺失 checker、自定义 API 前缀、声明/未声明路由均有自动化覆盖 |
| 运行时冒烟 | Pass | memory 模式端口 18088：health 200、匿名插件 API 401、登录 200、声明路由穿过权限层、未声明路由 403；验收后进程已停止 |
| 影响面 | Pass | 未新增 HTTP endpoint/schema；无需 OpenAPI、migration/seed 或前端契约变更；权限、审计、配置文档与 CodeGraph 已同步 |

### Verification Commands

Result: `go test -race ./internal/bootstrap ./internal/handler/http/...`、`go vet ./internal/bootstrap ./internal/handler/http/...`、`go test ./...`、二进制构建、隔离运行时 HTTP 冒烟、`git diff --check`、历史目录边界检查和 CodeGraph 同步全部通过。

### Next Step

领取 `H1-03`，验证插件安装、启用、禁用过程中的权限快照与审计行为。
