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

## H1-03 Retry Record

- Date: 2026-07-18
- Status flow: `Doing -> Failed`
- Failed gate: 首轮定向 Go 验收编译失败，插件路由成功审计断言误插入相邻旧测试，引用了未定义的 `events`。
- Retry rule: 将断言移动到自带审计事件仓储的插件权限矩阵用例，状态返回 Doing 后重跑同一组测试。
- Retry status: 已返回 Doing，进入同项重试。

- Second failed gate: 隔离运行时发现外部插件启用后的契约路由返回 HTTP 404；包装层只向路由器暴露内置插件 snapshot，未暴露已启用外部插件的 route snapshot。
- Second retry rule: 为已启用外部插件按当前 manifest 构造只读 extension snapshot；补状态测试后重建服务，复跑 enable 200、disable 403、re-enable 200 与审计查询。
- Second retry status: 已返回 Doing，进入同项重试。

- Third failed gate: Review 语义检查发现 lifecycle action 写入旧 `Record` 通道，未满足 action catalog 规定的 `EventTypePlugin`、`success/failure` 与 `high` risk。
- Third retry rule: lifecycle action 改接统一 AuditEvent sink，新增 uninstall catalog 条目与事件构造测试，保持其他既有审计写入点不变。
- Third retry status: 已返回 Doing，进入同项重试。

- Fourth failed gate: 统一事件通道测试发现连续生命周期操作在 Windows 上生成相同纳秒 ID，内存事件仓储覆盖记录，3 条事件只保留 1 条。
- Fourth retry rule: 生命周期审计 ID 使用时间戳加原子序列，补唯一性断言并运行 race 测试。
- Fourth retry status: 已返回 Doing，进入同项重试。

## H1-03 验证插件生命周期权限与审计行为

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 验证插件安装、启用、禁用、重新启用与卸载时的权限和 extension snapshot，并统一生命周期、路由成功与权限拒绝审计。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 生命周期 snapshot | Pass | 安装态不暴露 route permission/extension snapshot；启用后出现；禁用和卸载后移除；重新启用后恢复 |
| 外部路由执行 | Pass | 已启用外部插件按当前 manifest 暴露只读 extension snapshot，路由器可注册并执行声明路由 |
| 权限关闭 | Pass | 禁用插件立即从 route permission registry 移除，即使 super admin 访问原业务 API 也返回 403 |
| 路由审计 | Pass | 声明 `audit_action` 的授权请求写入 `EventTypePlugin`，携带 action、permission、source、HTTP status 与 trace；拒绝仍写安全审计 reason |
| 生命周期审计 | Pass | install/enable/disable/uninstall 的成功与失败均写统一 plugin event；结果为 `success/failure`、风险为 `high`，action catalog 已补 uninstall |
| 事件唯一性 | Pass | 生命周期事件 ID 使用时间戳加原子序列；定向 race 与连续操作测试未发生覆盖 |
| 失败状态 | Pass | 内置插件禁用/卸载失败返回 403，并生成 failure 审计；首轮编译、运行时 404、旧审计通道和 ID 冲突均按记录重试后通过 |
| 多语言 smoke | Pass | `smoke-pharma-oa-plugin.ps1` 在 `zh-CN` 与 `en-US` 下均通过，默认中文输出 |
| 运行时矩阵 | Pass | memory 模式 18089：启用 API 200、disable 200、禁用 API 403、enable 200、重启用 API 200；生命周期、路由成功和拒绝事件均可查询 |
| 影响面 | Pass | 未新增 manifest 路径、权限键、migration/seed 或前端契约；本项执行既有 manifest 路由，聚合 OpenAPI 安全元数据由依赖项 `H1-04` 紧接同步 |

### Verification Commands

Result: `go test ./internal/plugin/...`、插件 lifecycle smoke、相关包定向测试与 race、`go test ./...`、`go vet ./...`、双语 smoke、隔离 HTTP/审计矩阵、`git diff --check`、历史目录边界和 CodeGraph 同步全部通过。

### Next Step

领取 `H1-04`，同步插件路由权限的聚合 OpenAPI、安全元数据与中英文开发文档。
