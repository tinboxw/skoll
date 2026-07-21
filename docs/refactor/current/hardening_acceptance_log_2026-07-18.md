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

## H1-04 Retry Log

| 尝试 | 状态 | 失败证据 | 重试动作 |
| --- | --- | --- | --- |
| 1 | Failed | 聚合 OpenAPI 定向测试可解析 YAML，但启用插件测试路由未进入 `paths` | 检查 route descriptor 过滤条件，修复聚合输入后将任务恢复为 `Doing` 并重跑同一验收 |
| 2 | Failed | race、全量测试与 vet 的组合命令超过 120 秒被终止，未取得完整退出码 | 拆分门禁并延长超时，逐项取得明确结果后再验收 |
| 3 | Failed | 隔离 memory 进程中安装并启用 `pharma_oa` 后，`/docs/openapi.yaml` 未出现声明路由 | 检查 bootstrap 到 Router 的 Manager/snapshot provider 依赖注入并补真实依赖链测试 |

- First retry finding: Windows 工作区中的嵌入 OpenAPI 使用 CRLF，聚合插入点仅匹配 LF；route descriptor 校验和渲染内容正常。
- First retry action: 聚合器同时识别 CRLF 与 LF，状态已恢复为 `Doing`。
- Second retry action: 门禁已拆分并提高超时，状态已恢复为 `Doing`。
- Third retry finding: 运行时 debug 已暴露 119 条 `pharma_oa` route snapshot；PowerShell 将 YAML 响应作为 `byte[]` 返回，验收脚本直接执行字符串 `Contains` 导致误报。
- Third retry action: 按 UTF-8 解码响应字节后重跑运行时文档断言，状态已恢复为 `Doing`。

## H1-04 同步插件路由权限 OpenAPI 与开发指南

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 为启用插件的当前 manifest 路由聚合 OpenAPI JWT、权限、审计与来源元数据，并同步双份基线规范和中英文开发文档。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 聚合路由 | Pass | `/docs/openapi.yaml` 按当前启用插件的 route snapshot 动态聚合；只接受通过既有 route permission registry 校验的声明 |
| 认证与权限 | Pass | 每个聚合 operation 声明 `security: bearerAuth`、`x-skoll-permission`、plugin id/source，并在 manifest 有声明时附带 `x-skoll-audit-action` |
| 生命周期 | Pass | 单元测试与隔离 memory 进程均验证 Enabled 展示、Disabled 移除、重新 Enabled 恢复；无效权限声明不进入 OpenAPI |
| OpenAPI 基线 | Pass | `docs/api/openapi.yaml` 与 `internal/handler/http/openapi.yaml` 完全一致，均定义 HTTP Bearer JWT `bearerAuth`；引用解析通过 |
| 双语文档 | Pass | `plugin-api-contract.md` 以中文为默认入口，英文版内容对齐；开发索引和插件教程均可进入，示例只使用当前 `plugin.yaml` 格式 |
| 文档链接 | Pass | 四份受影响开发文档的相对 Markdown 链接全部存在 |
| 质量门禁 | Pass | 定向 HTTP/plugin 测试、focused race、`go test ./...`、`go vet ./...`、运行时矩阵、diff/边界检查均通过 |
| 影响面 | Pass | 未新增 HTTP handler、permission key、audit action、migration/seed 或前端页面；仅同步已声明插件路由的运行时文档契约 |
| CodeGraph | Pass | 索引已同步到 669 files / 13,955 nodes / 41,856 edges，状态为 up to date；未修改 `docs/refactor/old/` |

### Verification Commands

```powershell
go test ./internal/handler/http/... ./internal/plugin/... -count=1
go test -race ./internal/handler/http ./internal/plugin/... -count=1
go test ./... -count=1
go vet ./...
Compare-Object (Get-Content -Encoding UTF8 docs/api/openapi.yaml) (Get-Content -Encoding UTF8 internal/handler/http/openapi.yaml)
codegraph sync .
codegraph status .
git diff --check
```

### Runtime Evidence

隔离 memory 实例使用当前 `plugins/pharma_oa/plugin.yaml`：启用时员工路由及 `bearerAuth`、permission、audit 元数据存在；禁用后路径移除；重新启用后恢复。验收进程已停止。

### Next Step

H1 父任务全部完成。按正式 Work Item 顺序领取 `H2-01`，定义 Pharma OA 持久化契约与 schema 基线。

## H2-01 定义医药 OA 持久化契约与 Schema 基线

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: 冻结医药 OA repository ports、29 张表的机器可读 schema、索引与唯一约束、审计字段、事务边界，以及三种受支持数据库的迁移和破坏性回滚策略。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Repository 边界 | Pass | 主数据、采购、销售、库存、合规与 CRM ports 接收 domain entity；库存写入通过显式事务回调聚合 batch、balance、ledger 与 lock |
| Schema 所有权 | Pass | 29 张表全部使用 `pharma_oa_` namespace，并声明领域 owner、主键、事务组和审计列 |
| 唯一约束与索引 | Pass | 药品批准文号、业务单号、库存位置、批次与关键 idempotency key 均有机器可读唯一索引；常用状态、范围、到期和时间查询有组合索引 |
| 不可变记录 | Pass | 库存流水和冷链记录标记为 immutable，仅包含 `created_at`/`created_by` 创建审计列，不提供 update/delete repository 方法 |
| 多数据库迁移 | Pass | MySQL、PostgreSQL、SQLite 共享同一 schema 基线；Up 保留依赖顺序，Down 严格逆序且每步标记 destructive，不支持方言失败关闭 |
| 卸载与恢复 | Pass | 正常卸载固定 `retain`；destructive down 仅允许空环境或首次安装失败，生产数据使用前向修复或备份恢复 |
| 文档与多语言 | Pass | 中文为默认架构契约，数据库和架构索引已接入；三份相关文档的本地 Markdown 链接全部可解析 |
| 质量门禁 | Pass | schema/迁移定向测试、repository/plugin 测试、`go test ./...`、`go vet ./...` 与 `git diff --check` 全部通过 |
| 影响面 | Pass | 本项未新增 HTTP API、权限键、审计 action、seed 或前端页面；无需 OpenAPI/i18n 资源变更，不实现旧结构兼容 |
| CodeGraph | Pass | 索引同步后为 672 files / 14,034 nodes / 42,074 edges，`SchemaBaseline` 可查询；未修改 `docs/refactor/old/` |

### Verification Commands

```powershell
go test ./internal/repository/pharmaoa -count=1
go test ./internal/repository/pharmaoa -run TestMigrationPlanSupportsAllDialectsAndReversesOrder -count=1
go test ./internal/plugin -run TestPharmaOAPluginManifestCoversIndustrySkeleton -count=1
go test ./internal/repository/... ./internal/plugin/... -count=1
go test ./...
go vet ./...
codegraph sync .
codegraph query "SchemaBaseline"
git diff --check
```

### Next Step

父任务 H2 保持 `Doing`。领取 `H2-02`，实现员工、药品、供应商、客户和仓库/库位的 SQL repository 与多数据库契约测试。

## H2-05 Retry Log

| 尝试 | 状态 | 失败证据 | 重试动作 |
| --- | --- | --- | --- |
| 1 | Failed | fresh-install migration 覆盖检查显示 `SchemaBaseline()` 的 30 张表中仅 27 张存在建表脚本，缺少 `pharma_oa_announcements`、`pharma_oa_qualifications`、`pharma_oa_cold_chain_records` | 当前资质已归属员工/供应商/客户聚合 JSON，因此删除未实现的独立资质候选表；为真实存在的公告和不可变冷链记录补 GORM model 与 retain-only 双数据库 migration，并加入全覆盖门禁；状态经 `Failed -> Doing` 恢复后重跑 |
| 2 | Failed | `smoke-pharma-oa-database.ps1` 在 Windows PowerShell 5 中按本地代码页读取 UTF-8 无 BOM 中文字符串，导致字符串终止符解析失败，smoke 未启动 | 可执行 PowerShell 脚本改为 ASCII 输出以兼容 Windows PowerShell 5/PowerShell 7；中文默认说明保留在 UTF-8 验收文档，状态经 `Failed -> Doing` 恢复后重跑同一脚本 |
| 3 | Failed | Docker Desktop 启动后 engine 在创建隔离 MySQL/PostgreSQL 容器时对 v1.53 API 返回 HTTP 500，MySQL 容器未创建，严格实库矩阵未运行 | 固定兼容 Docker API 版本并复查 engine；若 engine 恢复则重建隔离容器并执行 `-RequireExternalDatabases`，状态经 `Failed -> Doing` 恢复 |
| 4 | Blocked | 固定 `DOCKER_API_VERSION=1.43` 后 engine 仍返回 HTTP 500；`docker desktop restart` 超时，最终 `docker desktop status`/`docker version` 健康检查也在 30 秒内无响应 | Work Item 转为 `Blocked`，不提交、不进入 H3-01；由本机 Docker Desktop 或独立 MySQL/PostgreSQL 环境恢复后转回 `Doing`，执行严格实库矩阵 |
| 5 | Failed | 首次 `-Full` 验收被命令执行器 120 秒上限终止，没有完整退出码 | 将执行器超时提高到 600 秒，重跑同一脚本，不复用中断结果 |
| 6 | Failed | 提高超时后的全仓回归在链接 `workflow.test.exe` 时因系统盘仅剩约 0.6 GB 报 `No space left on device` | 仅执行 `go clean -cache -testcache` 清理可再生成的 Go 缓存，空间恢复到约 3.9 GB；再次执行同一 `-Full` 门禁后全仓测试和 vet 通过 |
| 7 | Failed | 隔离 MySQL 8.0 已就绪，但首轮实库 contract 在链接测试程序前因系统盘空间再次耗尽，未进入数据库断言 | 将 `GOCACHE`、`GOTMPDIR` 迁移到空间充足的 `D:\skoll-h2-05-db`，清理可再生成的默认 Go 缓存后按 `Failed -> Doing` 重跑同一 MySQL contract |
| 8 | Failed | 官方 PostgreSQL 16 unattended 安装器触发系统级授权，启动操作被取消，未产生可用 15432 实例 | 不重复请求系统级安装；尝试只解包已校验的官方介质，在 `D:\skoll-h2-05-db` 使用 `initdb`/`pg_ctl` 启动用户态隔离实例，随后按 `Failed -> Doing` 重跑 PostgreSQL contract |
| 9 | Failed | EDB 官方 PostgreSQL 16.14 ZIP 已按预期字节数下载，但 Windows `Expand-Archive` 在 300 秒执行上限内未完成 | 保留已下载 ZIP 与部分解压内容，改用 Windows `tar` 覆盖解压同一包；确认 `postgres.exe --version` 后再恢复 `Doing` |
| 10 | Failed | 严格矩阵先执行真实 PostgreSQL migration，再运行主数据 contract 时，`AutoMigrate` 因客户代码唯一索引名与 model 默认约束名不一致，报 `constraint uni_pharma_oa_customers_code does not exist` | 为主数据 model 的业务唯一键补齐与 schema/migration 一致的显式索引名，检查同类隐式命名，恢复 `Doing` 后重跑 migration -> repository 严格路径 |

## H2-05 数据库生命周期验收（Done）

| Gate | Result | Evidence |
| --- | --- | --- |
| Schema 与 migration 完整性 | Pass | 最终基线为 29 张真实业务表；MySQL/PostgreSQL `000019` 至 `000022` 在专用空库中各执行两遍，表、列与命名索引完整，生产 migration 无 `DROP`/`TRUNCATE`。 |
| fresh install 与 upgrade | Pass | SQLite fresh install 和保留既有主数据的 schema upgrade 通过；MySQL 8.0.31/PostgreSQL 16.14 真实 migration fresh install 通过。 |
| seed、重启与备份恢复 | Pass | 文件 SQLite 完整 seed 后关闭重开，再次 seed 复用相同业务 ID 且无重复库存/流水；备份后写入额外数据，再恢复备份，额外数据消失且 seed 仍可复用。 |
| repository contract | Pass | SQLite、MySQL 8.0.31、PostgreSQL 16.14 的主数据、订单/库存、工作流业务记录契约均实际通过。 |
| 严格支持数据库矩阵 | Pass | 隔离 MySQL 运行于 13306，EDB PostgreSQL 16.14 便携实例运行于 15432；`-RequireExternalDatabases -Full` 整体退出码 0，无外部数据库 SKIP。 |
| API/权限/审计/i18n | Pass | 未新增 HTTP endpoint、权限或审计动作；migration/seed 影响已同步。可执行脚本保持 ASCII 兼容 Windows PowerShell，中文默认说明位于验收报告。 |
| 质量门禁 | Pass | 严格 smoke、全仓 Go 回归、`go vet`、变更后定向 race、build、diff、文档链接和 CodeGraph 全部通过；缺少 DSN 时严格模式 fail-closed。 |

详细环境、失败重试、复现命令和实库版本见 `pharma_oa_database_acceptance_2026-07-18.md`。H2-05 已按 `Blocked -> Doing -> Failed -> Doing -> Review -> Done` 完成；父任务 H2 同步完成，提交后领取 H3-01。

## H2-04 医药 OA 工作流关联业务记录持久化

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 将合同、质量投诉、召回、客户跟进、销售机会、付款计划、发票、付款提醒、库存告警和报表导出作业接入统一 repository，并持久化完整聚合快照、查询列、重试日志与幂等引用。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Repository 与 DI | Pass | memory、MySQL、PostgreSQL store bundle 与 bootstrap 均注入 10 组 repository port，原 service 构造器保持可选 repository 注入以支持既有单元测试。 |
| 重启持久化 | Pass | SQLite 文件库关闭并重开后，11 张表完整 round-trip；工作流 ID、参与人、附件、召回任务、机会阶段历史、回款、通知引用、作业日志和时间均保留。 |
| 重试与幂等 | Pass | 付款提醒失败作业恢复后重试成功且 retry count/log 保留；库存告警作业保留唯一幂等键；报表 service 实例重建后，相同显式查询窗口复用既有成功作业和文件，不重复上传。 |
| Schema 与 migration | Pass | `SchemaBaseline()` 更新为 30 张表；11 个 H2-04 GORM model 的列和命名索引逐项匹配；MySQL/PostgreSQL `000021` migration 均为 retain-only，不含 destructive down。 |
| 数据库矩阵 | Pass with environment gate | SQLite 契约已实际运行；MySQL/PostgreSQL 同一 round-trip 用例已编译并分别由 `SKOLL_TEST_MYSQL_DSN`、`SKOLL_TEST_POSTGRES_DSN` 启用。本机未配置 DSN，两项明确为 SKIP，不记作实库通过，H2-05 必须复验。 |
| API/权限/审计/migration/seed | Pass | 未新增或修改 HTTP endpoint、OpenAPI schema、permission key、audit action 或 seed；现有业务审计 action 与权限链保持，新增影响仅为 repository、DI、schema 和 migration。 |
| 多语言与兼容 | Pass | 无新增前端或用户可见文案，locale 资源无需变更；架构与迁移说明以简体中文为默认入口；未实现旧字段、旧表或双写兼容层。 |
| 质量门禁 | Pass | 聚焦测试、定向 race、`go test ./... -count=1`、`go vet ./...`、`go build` 与 `git diff --check` 全部通过。 |
| CodeGraph | Pass | 影响面覆盖 service、handler、bootstrap 与双数据库 adapter；同步后为 691 files / 14,758 nodes / 45,228 edges，状态 up to date；未修改 `docs/refactor/old/`。 |

### Verification Commands

```powershell
go test ./internal/store/sql/gormrepo -run 'TestPharmaWorkflowRecord' -count=1 -v
go test ./internal/service/pharmaoa -run 'TestReportExportServiceRestart|TestReportExportFailedJobRetries|TestInventoryAlert' -count=1 -v
go test ./internal/repository/pharmaoa ./internal/store/sql/gormrepo ./internal/store ./internal/service/pharmaoa ./internal/bootstrap -count=1
go test -race ./internal/repository/pharmaoa ./internal/store/sql/gormrepo ./internal/store ./internal/service/pharmaoa ./internal/bootstrap -count=1
go test ./... -count=1
go vet ./...
go build -o "$env:TEMP\skoll-h2-04.exe" ./cmd/skoll
codegraph impact "ReportExportJobRepository"
codegraph impact "ContractService"
codegraph sync .
codegraph status .
git diff --check
```

### Next Step

父任务 H2 保持 `Doing`。按正式 Work Item 顺序领取 `H2-05`，验证 migration、seed 幂等、重启、备份夹具和 MySQL/PostgreSQL 支持矩阵。

## H2-04 Retry Log

| 尝试 | 状态 | 失败证据 | 重试动作 |
| --- | --- | --- | --- |
| 1 | Failed | 首轮工作流记录 repository 定向编译失败：内存销售机会筛选器使用了领域实体不存在的 `Number` 字段，说明候选 schema 与当前领域形状不一致 | 改用实际 `Title`、客户字段进行筛选，并在 H2-04 migration/model 中同步校正销售机会列；恢复 `Doing` 后重跑 repository/domain 测试 |
| 2 | Failed | 服务重启幂等验收使用未指定时间窗的报表请求；服务按当前时刻补齐默认结束时间后，两次请求形成不同业务窗口与幂等键，测试错误期待复用旧作业 | 使用明确且相同的 `From`/`To` 查询窗口重试，继续验证服务实例重建后不会重复上传或创建作业；状态经 `Failed -> Doing` 恢复 |

## H2-03 医药 OA 库存与业务单据原子持久化

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 将采购、采购入库、销售、销售出库、批次、余额、不可变流水、盘点和调拨接入统一 repository；库存余额与流水在同一事务内提交，多行库存变更任一失败时整体回滚。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| Repository 与重启持久化 | Pass | memory 与 GORM 实现覆盖采购、销售、批次、余额、流水、盘点和调拨；SQLite 文件库关闭并重开后，库存与全部业务单据 round-trip 通过。 |
| 原子性与并发 | Pass | 20 个并发出库请求竞争 100 件库存时仅 10 个成功，余额为 0 且流水数正确；多行出库第二行库存不足时，第一行余额和流水同步回滚。 |
| 幂等与不可变流水 | Pass | 流水使用业务幂等键和确定性 ID；并发重复出库只扣减一次，重复调拨复用同一对流水，源/目标余额保持 60/40；repository 不提供流水更新或删除接口。 |
| 事务边界 | Pass | 库存事务锁定余额并使用 version 条件更新，余额与流水同事务提交；采购审批的申请状态与订单同事务提交；采购入库和销售出库使用多行原子库存事务。 |
| Schema 与 migration | Pass | 11 个新增 GORM model 与 `SchemaBaseline()`、MySQL/PostgreSQL `20260718_000020` migration 一致；金额统一为 `(18,2)`，脚本不含 destructive `DROP`，正常卸载保留业务数据。 |
| 多数据库契约 | Pass with environment gate | SQLite 契约已实际运行。MySQL/PostgreSQL 使用同一 repository contract，但本机未设置 `SKOLL_TEST_MYSQL_DSN`、`SKOLL_TEST_POSTGRES_DSN`，两项测试明确为 SKIP，不记作实库通过；H2-05 必须在可用实库环境复验。 |
| API/权限/审计/i18n | Pass | 本项未新增 HTTP endpoint、permission key、前端页面或 seed，无需改 OpenAPI/权限资源/i18n locale；现有库存、采购和销售 audit action 保持提交后记录，架构说明默认使用中文。 |
| 质量门禁 | Pass | 聚焦测试、库存专项验收、SQLite 重启、schema/migration 契约、定向 race、`go test ./... -count=1`、`go vet ./...`、构建和 `git diff --check` 全部通过。 |
| CodeGraph 与边界 | Pass | 已同步索引并复核 `InventoryService` 的 55 个受影响符号；调用方已纳入全仓回归，未修改 `docs/refactor/old/`。 |

### Verification Commands

```powershell
go test ./internal/store/sql/gormrepo -run 'TestPharma(OrderRepositoriesPersistAcrossSQLiteRestart|OrderRepositoriesMySQLContract|OrderRepositoriesPostgreSQLContract|TransactionModelsMatchSchemaAndMigrations)$' -count=1 -v
go test ./internal/service/pharmaoa -run 'TestInventory(ConcurrentOutboundCannotOversell|RetryIsIdempotent|TransferRetryUsesBusinessIdempotencyKey|BatchFailureRollsBackAllLines|SQLiteRestartPersistence)$' -count=1 -v
go test -race ./internal/repository/pharmaoa ./internal/store/sql/gormrepo ./internal/store ./internal/service/pharmaoa ./internal/bootstrap -count=1
go test ./... -count=1
go vet ./...
go build -o "$env:TEMP\skoll-h2-03.exe" ./cmd/skoll
codegraph impact "InventoryService"
codegraph sync .
codegraph status .
git diff --check
```

### Next Step

父任务 H2 保持 `Doing`。按正式 Work Item 顺序领取 `H2-04`，持久化工作流关联的 CRM、合规、提醒和报表作业记录，并验证重启与重试不会产生重复副作用。

## H2-03 Retry Log

| 尝试 | 状态 | 失败证据 | 重试动作 |
| --- | --- | --- | --- |
| 1 | Failed | 首轮库存事务接线定向测试失败：bootstrap 使用未定义的 `stores` 变量；召回与投诉契约依赖当前 `pharma-stock-batch-N` 实体 ID，新内容派生 ID 生效后无法定位关联批次 | 修正 DI 为当前 `bundle` 变量；保留现有批次 ID 规则，并在事务内按产品/批号复用批次、探测重启后的可用序号；恢复 `Doing` 后重跑同一测试集 |
| 2 | Failed | 采购入库 service 切换 repository 后残留一个多余的 `}`，`internal/service/pharmaoa` 编译失败 | 删除残留括号，恢复 `Doing`，重新执行 repository/GORM/service 定向测试 |
| 3 | Failed | 新增 SQLite 重启测试直接比较 `shared.ID` 与 `string`，测试包编译失败 | 使用 `shared.ID.String()` 比较实际批次 ID，恢复 `Doing` 并重跑同一验收集合 |
| 4 | Failed | 新增调拨业务幂等验收时误用不存在的 `StockBalance.Position` 字段，`internal/service/pharmaoa` 测试包编译失败 | 按当前领域模型改用平铺的库位字段，恢复 `Doing` 后重跑调拨幂等及库存事务验收集合 |

## H2-02 医药 OA 主数据持久化仓储

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 将员工、药品、供应商、客户、仓库/库位从 service 进程内 map 迁移到统一 repository port，并接入 memory、MySQL 和 PostgreSQL store bundle。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 主数据持久化 | Pass | 五类 aggregate 的创建、读取、编辑和业务停用全部经过 repository；SQLite 文件库关闭并重开后完整 round-trip 通过，资质、联系人、附件 metadata、温控与库位 JSON 均保留。 |
| 分页与作用域 | Pass | 代码排序、offset/limit、关键词、状态、区域、组织和负责人 OR scope 在 memory 与 GORM 实现中保持同一契约；客户无授权范围仍返回空集。 |
| 唯一性与并发创建 | Pass | 主键和业务代码使用 insert-only 创建；药品额外约束批准文号。服务重建及两个实例共享仓储时不会因 ID 计数器重置覆盖旧记录。 |
| Schema 与迁移 | Pass | 五个 GORM model 与 `SchemaBaseline()` 列和索引一致；MySQL/PostgreSQL migration 均为非破坏性建表及索引脚本，正常卸载保持 `retain`。 |
| 数据库契约 | Pass with environment gate | SQLite 契约已实际运行；MySQL/PostgreSQL 同一套 round-trip 用例已编译，并分别由 `SKOLL_TEST_MYSQL_DSN`、`SKOLL_TEST_POSTGRES_DSN` 启用。本机未配置 DSN，Docker engine 恢复失败，因此两项实库运行明确为 SKIP，不记作运行通过；H2-05 必须在可用实库环境复验。 |
| 质量门禁 | Pass | focused、定向 race、`go test ./... -count=1`、`go vet ./...`、构建、Markdown 链接、`git diff --check` 全部通过。 |
| API/权限/审计/i18n | Pass | 未新增 HTTP endpoint、permission key、audit action、seed 或前端页面，无 OpenAPI/权限资源变更；现有审计动作保持，架构说明以中文为默认入口，无旧数据结构兼容层。 |
| CodeGraph 与边界 | Pass | 索引已同步为 677 files / 14,223 nodes / 42,973 edges，状态 up to date；未修改 `docs/refactor/old/`。 |

### Verification Commands

```powershell
go test ./internal/store/sql/gormrepo -run 'TestPharmaMaster|TestSchemaBaseline' -count=1 -v
go test ./internal/repository/pharmaoa ./internal/store/sql/gormrepo ./internal/store ./internal/service/pharmaoa ./internal/bootstrap -count=1
go test -race ./internal/repository/pharmaoa ./internal/store/sql/gormrepo ./internal/service/pharmaoa ./internal/bootstrap -count=1
go test ./... -count=1
go vet ./...
go build -o "$env:TEMP\skoll-h2-02.exe" ./cmd/skoll
codegraph sync .
codegraph status .
git diff --check
```

### Next Step

父任务 H2 保持 `Doing`。按正式 Work Item 顺序领取 `H2-03`，实现采购、销售、批次、余额、不可变流水、盘点和调拨的原子事务持久化。

## H2-02 Retry Log

| 尝试 | 状态 | 失败证据 | 重试动作 |
| --- | --- | --- | --- |
| 1 | Failed | 首轮主数据 repository 契约测试编译失败：fixture 将供应商的 `Primary`、`MimeType` 字段误用于客户联系人/附件 | 按当前 customer domain 字段修正 fixture，将 Work Item 恢复为 `Doing`，重跑同一验收命令 |
| 2 | Failed | 尝试启动一次性 MySQL 容器时 Docker Desktop Linux engine `_ping` 返回 HTTP 500，容器未创建 | 检查并恢复 Docker engine；成功后继续 MySQL/PostgreSQL 实库契约，仍不可用则保留明确环境证据并由 H2-05 数据库环境门复验 |
| 3 | Failed | 加入批准文号唯一性后，既有产品导入 fixture 的第二行复用了相同批准文号，原断言仍期望创建两条 | 为不同药品使用不同批准文号，保留批准文号重复拒绝的新约束，恢复 `Doing` 后重跑相关服务与仓储测试 |

## H3-01 Retry Log

| 尝试 | 状态 | 失败证据 | 重试动作 |
| --- | --- | --- | --- |
| 1 | Failed | 首次将 JWT、认证、bootstrap、插件和整套 Pharma OA HTTP 包合并执行，命令在 120 秒内无结果并被超时终止，不能作为通过证据 | 按 `Failed -> Doing` 恢复任务，将认证专项、bootstrap、插件与 Pharma OA 调用方拆分为独立批次并延长必要批次的超时后重跑 |
| 2 | Failed | JWT 与独立认证中间件已通过，但限定 `TestBuiltinAuth|TestAuthGuardMiddleware` 的 bootstrap 包仍在 120 秒内未完成并被外部超时终止 | 保留已通过专项证据，按 `Failed -> Doing` 恢复任务，将 bootstrap 批次超时放宽到 5 分钟以区分冷编译耗时与测试挂起 |

## H3-01 认证契约受信任组织声明

- Date: 2026-07-18
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 将用户 ID、组织 ID、组织父链、主角色和全部角色写入服务端签发的 JWT，并同步认证响应、前端会话类型、双语错误和 OpenAPI。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| JWT 身份契约 | Pass | `JWTIdentity` 仅接收服务端解析结果；签名与解析覆盖 `sub`、`organizationId`、根到叶 `organizationPath`、`role` 和 `roles`，拒绝缺失用户、角色不一致及组织路径不闭合的声明，签名篡改测试通过。 |
| 受信任数据来源 | Pass | 登录仅解码账号和密码；用户 ID/部门来自用户仓储，组织路径逐级读取组织仓储并检测缺失和循环，角色集合来自 RBAC 绑定。携带伪造组织和角色字段的请求签发结果仍为仓储中的真实身份。 |
| 权限与审计 | Pass | 超级管理员旁路改为检查全部签名角色；插件开发入口、客户范围和公告角色受众读取签名角色；登录审计记录真实组织和角色集合。未新增 permission key、migration 或 seed。 |
| API 与前端 | Pass | 登录和 `/auth/me` 返回组织与角色声明；两份 OpenAPI 定义相同的请求/响应 Schema 且引用测试通过；Pinia 会话、登录页和内置认证插件持久化完整身份字段。 |
| 多语言错误 | Pass | `invalid_auth_request`、`invalid_credentials`、`invalid_organization`、`unauthorized` 使用稳定错误码；新增中文默认与英文翻译，两个新增 locale key 均恰好出现两次。 |
| 运行时 smoke | Pass | 隔离 memory 实例在 `127.0.0.1:18080` 接受带伪造组织/角色字段的真实登录请求；JWT 与响应一致且未采用伪造值，受保护接口无 Token 为 401、有 Token 通过；验收后进程已停止。 |
| 质量门禁 | Pass | 认证专项、bootstrap、插件/Pharma OA 调用方、定向 race、全仓 Go 测试、`go vet`、后端构建、前端 typecheck/build、脚本语法、OpenAPI 同步及 `git diff --check` 全部通过。 |
| CodeGraph 与边界 | Pass | 索引同步后为 694 files / 14,844 nodes / 45,599 edges；`JWTClaims` 影响面 130 个符号并由全仓测试覆盖；未修改 `docs/refactor/old/`，未纳入用户已有文档和本地环境变更。 |

### Verification Commands

```powershell
go test ./pkg/security ./internal/handler/middleware -count=1
go test ./internal/bootstrap -run 'TestBuiltinAuth|TestAuthGuardMiddleware' -count=1 -v
go test ./internal/handler/http/v1/plugin ./internal/handler/http/v1/pharmaoa -count=1
go test ./internal/handler/http -run 'OpenAPI|Docs' -count=1
go test -race ./pkg/security ./internal/handler/middleware ./internal/bootstrap -run 'TestSign|TestParseJWT|TestAuthMiddleware|TestBuiltinAuth|TestAuthGuardMiddleware' -count=1
go test ./... -count=1
go vet ./...
go build -o "$env:TEMP\skoll-h3-01.exe" ./cmd/skoll
npm --prefix web run typecheck
npm --prefix web run build
$env:SKOLL_API_BASE='http://127.0.0.1:18080'; npm --prefix web run smoke:auth
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
node --check web/scripts/auth-plugin-smoke.mjs
codegraph impact "JWTClaims"
codegraph sync .
codegraph status .
git diff --check
```

### Next Step

父任务 H3 保持 `Doing`。提交 H3-01 后按正式 Work Item 顺序领取 `H3-02`，基于受信任声明和授权实现本人、组织、组织树与全量数据范围策略，并拒绝请求参数扩大范围。

## H3-02 组织树数据范围策略

- Date: 2026-07-19
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 基于签名 JWT、RBAC Binding/PolicyRule 和组织仓储生成本人、当前组织、组织树与全量范围，并将同一范围用于用户列表、医药客户读写以及内存/GORM 仓储谓词。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 可信决策来源 | Pass | `ResolveDataScope` 只接收资源和动作；用户、当前组织和超级管理员角色来自签名 JWT，授权范围来自 RBAC，组织树后代来自 OrganizationRepository。缺失身份、组织、授权或不支持的 `custom` 范围均失败关闭。 |
| 范围矩阵 | Pass | 单元矩阵覆盖 `self`、`department`、`department_tree`、`all`；组织树仅包含当前组织及后代，不包含父级其他分支；超级管理员全量旁路保持显式。 |
| 仓储谓词与写入 | Pass | 用户仓储使用用户/组织过滤；医药 OA memory、GORM 主数据和 JSON 工作流仓储支持多组织 `IN` 与负责人 `OR`；客户创建、更新、禁用和校验共享同一范围判断，更新不能把记录移出授权树。 |
| 参数伪造 | Pass | 客户 HTTP handler 不再读取请求体或查询参数中的 scope、`ownerId`、`organizationId`、`includeAll` 作为授权输入；回归测试证明伪造参数不能读取其他负责人记录或执行跨范围写入。 |
| MySQL 契约 | Pass | 本机 MySQL 5.7 的工作流契约直接通过；默认 MyISAM 导致首次主数据迁移触发 1000-byte key limit，随后在隔离 `skoll_h302_acceptance` InnoDB 会话中主数据和工作流契约均通过，验收库已删除，原 `skoll` 数据未清理。 |
| API/权限/审计/migration/seed/i18n | Pass | 未新增 HTTP 路径、请求响应 Schema、权限键、审计动作、migration 或 seed；两份 OpenAPI 保持同步。403 继续由框架现有中英文 `error.forbidden` 映射展示，中文为默认语言。 |
| 质量门禁 | Pass | 聚焦测试、无缓存全仓 Go 测试、定向 race、`go vet ./...`、后端构建、前端 typecheck/build、OpenAPI 同步和运行时 smoke 全部通过。 |
| 运行时 | Pass | 隔离 memory 后端在 `127.0.0.1:18080` 健康检查为 `ok`；认证 smoke 通过；超级管理员访问客户和用户列表均为 200；验收进程已停止。 |

### Verification Commands

```powershell
go test ./internal/service/rbac ./internal/service/user ./internal/service/pharmaoa ./internal/repository/pharmaoa ./internal/store/sql/gormrepo ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/bootstrap
go test ./... -count=1
go test -race ./internal/service/rbac ./internal/service/user ./internal/service/pharmaoa ./internal/repository/pharmaoa ./internal/store/sql/gormrepo ./internal/handler/http/v1/pharmaoa ./internal/handler/http ./internal/bootstrap -count=1
$env:SKOLL_TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/skoll_h302_acceptance?charset=utf8mb4&parseTime=true&loc=UTC&default_storage_engine=InnoDB'; go test ./internal/store/sql/gormrepo -run 'TestPharmaMasterRepositoriesMySQLContract|TestPharmaWorkflowRecordRepositoriesMySQLContract' -count=1 -v
go vet ./...
go build -o "$env:TEMP\skoll-h3-02.exe" ./cmd/skoll
npm --prefix web run typecheck
npm --prefix web run build
git diff --no-index -- docs/api/openapi.yaml internal/handler/http/openapi.yaml
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 首轮聚焦测试发现旧 RBAC/用户测试仍传入调用方范围字段、客户 handler 测试未注入可信解析器，且一个内部客户夹具缺少 actor 或显式系统范围。 | 将测试改为签名身份/RBAC/组织矩阵，HTTP 测试注入确定性解析器，系统调用显式声明可信范围后重跑。 |
| 2 | Failed -> Doing | 本机 MySQL 5.7 工作流契约通过，但默认 MyISAM 的主数据 AutoMigrate 在进入范围断言前触发 error 1071。 | 创建隔离 InnoDB 验收库重跑同一 MySQL 契约，通过后删除隔离库，不改动原库。 |
| 3 | Failed -> Doing | 无缓存全仓测试发现用户 handler 假解析器仍读取旧字段，库存集成夹具以系统流程创建客户但未显式声明全量范围。 | 假解析器改为返回可信决策，系统夹具显式使用 `IncludeAll`，聚焦与全仓测试随后通过。 |

### Next Step

父任务 H3 保持 `Doing`。H3-02 提交后按正式 Work Item 顺序领取 `H3-03`，实现已授权范围的中英文前端 UX 与完整页面状态。

## H3-03 已授权数据范围前端体验

- Date: 2026-07-19
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 为用户与医药客户工作流提供服务端授权范围自省、范围指示器和受限选择器，移除前端可伪造范围参数，并补齐中英文与完整页面状态。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 可信范围自省 | Pass | 新增认证后的 `GET /v1/rbac/data-scope`，只接收业务 `resource`/`action`，实际用户、组织和 RBAC 范围继续来自签名身份与服务端策略；缺少参数返回 400，拒绝授权返回 403。 |
| 范围组件与选择器 | Pass | `DataScopeIndicator` 覆盖 loading、成功、错误和重试；客户创建/更新按 `self` 锁定负责人，按 `department`/`department_tree` 仅展示后端授权组织，`all` 保留业务选择。 |
| 参数收敛 | Pass | 删除客户 API 的 `scope`、`ownerId`、`organizationId`、`includeAll` 等授权参数及全部调用点；前端仅发送业务筛选和业务字段，不能扩大服务端授权范围。 |
| 页面状态 | Pass | 客户页覆盖 loading、中文 empty、error/retry、route no-permission、saving、两级 destructive confirm 和授权目标拒绝；用户页复用范围组件并在受限账号下展示可重试错误态。 |
| 多语言 | Pass | 中文默认；客户流程、范围名称/说明、错误、空状态、保存与停用操作均有 `zh-CN`/`en-US` 资源，浏览器切换后标题、范围、空态和错误文案即时更新。 |
| 响应式 | Pass | 390x844 下客户页与用户页均无横向文档溢出；移动端页头纵向排列，范围说明和授权 ID 可换行；恢复 1280x720 后桌面布局正常。 |
| 浏览器业务闭环 | Pass | 隔离 memory 环境以管理员创建并停用 `H303-ACCEPT-001`，统计、列表、成功提示和按钮状态同步；普通账号直达客户页被路由守卫拒绝，用户页范围请求失败时提供中英文错误和重试。 |
| API/权限/审计/migration/seed | Pass | 两份 OpenAPI 同步新增只读自省契约；未新增权限键、审计动作、migration 或 seed。端点复用既有认证和 RBAC 决策，业务写操作继续使用既有权限与审计。 |
| 质量门禁 | Pass | RBAC handler/service 聚焦测试、全仓 Go 测试、目标 race、`go vet`、后端构建、前端 typecheck/build、OpenAPI 哈希一致与 `git diff --check` 全部通过。 |
| CodeGraph 与边界 | Pass | 索引同步后为 697 files / 14,916 nodes / 45,800 edges；`resolveAuthorizedDataScope` 影响 11 个符号并由用户/客户浏览器与全仓回归覆盖；未修改 `docs/refactor/old/`。 |

### Verification Commands

```powershell
go test ./internal/handler/http/v1/rbac ./internal/handler/http ./internal/service/rbac ./internal/handler/http/v1/pharmaoa -count=1
go test ./... -count=1
go test -race ./internal/handler/http/v1/rbac ./internal/service/rbac ./internal/handler/http/v1/pharmaoa -count=1
go vet ./...
go build -o "$env:TEMP\skoll-h3-03-final.exe" ./cmd/skoll
npm --prefix web run typecheck
npm --prefix web run build
$a = Get-FileHash docs/api/openapi.yaml -Algorithm SHA256; $b = Get-FileHash internal/handler/http/openapi.yaml -Algorithm SHA256; if ($a.Hash -ne $b.Hash) { exit 1 }
codegraph impact resolveAuthorizedDataScope
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 首轮浏览器空状态显示 Element Plus 默认 `No data`；客户页向 `DataTable` 传入了组件不支持的 `empty-title`/`empty-description` 属性。 | 改用组件契约支持的 `empty-text`，分别映射中文默认空态和筛选无结果文案，重新 typecheck/build 并在浏览器确认 `暂无客户记录`。 |
| 2 | Failed -> Doing | 390x844 截图发现页头控制区仍保持桌面横排，范围说明和长授权 ID 在窄屏下缺少可靠换行策略。 | 页头在 860px 以下改为纵向排列并允许状态 chips 换行；范围说明和授权 ID 增加断行规则，复测客户/用户页无横向溢出。 |

### Next Step

父任务 H3 保持 `Doing`。按正式 Work Item 顺序领取 `H3-04`，建立本人、组织、组织树、全量和拒绝场景的 API/浏览器端到端矩阵，并验证跨组织访问被拒绝且写入审计完整。

## H3-04 组织数据范围端到端矩阵

- Date: 2026-07-19
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 建立默认关闭的本人、当前组织、组织树、全部数据和拒绝五类测试身份，贯通 RBAC、客户 API、拒绝审计、中英文浏览器与 MySQL 持久化验收。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 显式夹具 | Pass | 仅 `SKOLL_SCOPE_MATRIX_FIXTURES=true` 时创建五个测试账号、四级组织节点、角色/策略/绑定和四条客户数据；重复执行不产生重复记录，生产默认关闭并已写入配置文档。 |
| API 范围矩阵 | Pass | `self` 仅返回 `SCOPE-SELF`；`department` 返回本人和同组织客户；`department_tree` 再包含下级组织客户；`all` 返回四条；`denied` 的范围自省和列表均为 403。伪造 `includeAll`、组织和负责人参数不能扩大集合。 |
| 越权写入与审计 | Pass | 跨组织更新、禁用和销售资格校验返回 403；服务记录 `pharma_oa.customer.<operation>.denied`，包含 actor、目标组织/负责人、`data_scope` 原因和 denied 结果。集成矩阵验证审计证据与数据未被修改。 |
| 超级管理员与中间件 | Pass | 超级管理员全量旁路继续由签名角色显式触发；数据范围自省必须携带 JWT，但不误要求 `permission.read` 管理权限，目标资源/动作仍由 RBAC 解析并失败关闭。 |
| 浏览器双语矩阵 | Pass | 五类账号逐一真实登录；中文默认与英文切换均显示正确范围名称、授权目标和 1/2/3/4 条客户集合；拒绝账号直达客户页在两种语言下都被路由守卫送回仪表盘。 |
| 空集合契约 | Pass | 客户服务深拷贝保持 `contacts`、`qualifications` 和附件为空数组，JSON 固定输出 `[]`，避免合法空数据在 Vue 页面读取 `.length` 时崩溃。 |
| MySQL 隔离验收 | Pass | 本机 MySQL 隔离库 `skoll_h304_acceptance` 完成迁移、夹具和同一 API 矩阵；客户负责人使用 SQL 保存后的真实自增用户 ID，因此本人范围与内存实现一致。验收后停止进程并删除隔离库。 |
| API/权限/migration/seed/i18n | Pass | 未新增业务 HTTP 路径、权限键或 migration；新增拒绝审计动作和测试专用 seed 开关。OpenAPI 两份副本哈希一致；测试夹具业务文案不进入正式 locale，用户界面继续中文默认并具备英文资源。 |
| 质量门禁 | Pass | 全仓无缓存 Go 测试、目标 race、`go vet`、后端构建、前端 typecheck/build、脚本语法、运行时 memory/MySQL smoke、OpenAPI 一致性和 `git diff --check` 全部通过。 |
| CodeGraph 与边界 | Pass | 索引同步后为 701 files / 15,008 nodes / 46,090 edges；`ensureScopeMatrixFixtures` 影响 5 个符号并由 bootstrap、运行时和全仓测试覆盖；未修改 `docs/refactor/old/`。 |

### Verification Commands

```powershell
go test ./internal/bootstrap ./internal/service/rbac ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./tests/integration -count=1
go test ./... -count=1
go test -race ./internal/bootstrap ./internal/service/rbac ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa ./tests/integration -count=1
go vet ./...
go build -o "$env:TEMP\skoll-h3-04-final.exe" ./cmd/skoll
npm --prefix web run typecheck
npm --prefix web run build
$env:SKOLL_API_BASE='http://127.0.0.1:18080'; npm --prefix web run smoke:h3-data-scope
node --check web/scripts/h3-data-scope-smoke.mjs
codegraph impact ensureScopeMatrixFixtures
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 首轮集成矩阵存在未使用导入和上下文变量，无法编译。 | 删除残留变量，保持同一矩阵范围后重跑。 |
| 2 | Failed -> Doing | 本人范围创建返回 403；内存 RBAC 新绑定统一使用固定 ID `new`，后写绑定覆盖先前绑定。 | 为 RBAC 服务生成唯一绑定 ID并增加多绑定并存回归测试。 |
| 3 | Failed -> Doing | 运行时 `scope_self` 列表为 200，但数据范围自省为 403。 | 将自省端点从 `/v1/rbac` 管理权限前缀中精确排除，保留 JWT 认证并由目标资源 RBAC 决策。 |
| 4 | Failed -> Doing | 浏览器加载空联系人客户时，后端将空切片复制为 `nil`，Vue 读取 `.length` 发生渲染异常。 | 服务边界保持空集合为非 nil 切片，补 JSON 数组回归测试并重跑五账号双语矩阵。 |
| 5 | Failed -> Doing | MySQL 中用户字符串夹具 ID 被映射为自增主键，客户仍引用旧字符串，导致本人范围列表为空。 | 客户夹具反查已持久化账号的真实 ID，增加主键改写仓储替身测试，重建隔离库后矩阵通过。 |

### Next Step

父任务 H3 已完成。按正式 Work Item 顺序领取 `H4-01`，盘点 Pharma OA 硬编码文案并建立中文默认、英文完整和缺失 key 失败的 locale 基线。

## H4-01 Pharma OA 多语言基线盘点

- Date: 2026-07-19
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 盘点 15 个 Pharma OA 主页面、全局 locale 字典、静态翻译引用和插件双语桥接，建立中文默认、英文键集对等且可由 CI 强制执行的版本化基线。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 中文默认策略 | Pass | `DEFAULT_LOCALE` 显式声明为 `zh-CN`，无本地持久化偏好时统一回退到该常量；基线检查验证中文是首个受支持字典和默认 locale。 |
| 双语键完整性 | Pass | `zh-CN` 与 `en-US` 均为 650 个无重复键且键集完全一致；601 个静态 `t`/`translate` 引用全部存在，修正仪表盘无效引用 `page.plugins` 为 `menu.plugins`。 |
| Pharma OA 清单 | Pass | 15 个 `Pharma*` 主页面全部写入 `web/i18n/pharma-oa-baseline.json`，记录 namespace、迁移状态、静态翻译键数和用户可见硬编码计数；新增、删除或文案漂移均会失败。 |
| 插件桥接 | Pass | 检查 `plugins/pharma_oa/plugin.yaml` 的中英文插件名、菜单名及 `zh-CN`/`en-US` locale 声明，任一字段缺失都会失败。 |
| CI 失败策略 | Pass | `npm run typecheck` 先执行 `npm run check:i18n`，键集不对称、静态缺键、页面清单漂移、中文默认策略或插件桥接缺失均以非零状态阻断 CI。 |
| 质量门禁 | Pass | locale 检查、`vue-tsc --noEmit`、Vite 生产构建、Node 脚本语法、JSON 解析和 `git diff --check` 全部通过。 |
| API/权限/审计/migration/seed | Pass | 本项仅建立前端 locale 基线与清单，未新增 HTTP 契约、权限键、审计动作、migration 或 seed；无需修改 OpenAPI。 |
| CodeGraph 与边界 | Pass | 索引同步为 702 files / 15,053 nodes / 46,187 edges，状态 up to date；未修改 `docs/refactor/old/`，未纳入用户已有文档和本地环境变更。 |

### Verification Commands

```powershell
npm --prefix web run check:i18n
npm --prefix web run typecheck
npm --prefix web run build
node --check web/scripts/h4-locale-baseline-check.mjs
node -e "JSON.parse(require('fs').readFileSync('web/i18n/pharma-oa-baseline.json','utf8')); console.log('baseline json valid')"
codegraph sync .
codegraph status .
codegraph query "DEFAULT_LOCALE"
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 初始扫描器把带插值的模板字符串翻译调用当作静态键，且页面清单仍使用零值占位计数。 | 仅收集单引号/双引号静态键，保留动态状态映射为运行时逻辑，并写入 15 个页面的实际基线计数后重验。 |
| 2 | Failed -> Doing | 修正动态模板排除后，客户页真实静态键数为 58，与粗扫得到的 59 相差一项。 | 使用 SFC/TypeScript 解析结果校准客户页计数；不放宽键完整性和硬编码检查，随后同一验收命令通过。 |

### Next Step

父任务 H4 保持 `Doing`。H4-01 提交后按正式 Work Item 顺序领取 `H4-02`，依据版本化清单完成 Pharma OA 主流程中英文资源迁移和双语浏览器验收。

## H4-02 Pharma OA 主流程双语迁移

- Date: 2026-07-19
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 将 15 个 Pharma OA 主页面的主数据、协同、供应链、合规、CRM、财务和经营看板文案迁移为中文默认、英文完整的资源目录，并覆盖运行时枚举、页面状态、确认框、校验和响应式切换。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 主流程资源迁移 | Pass | `web/src/i18n/pharma.json` 为 15 个页面提供中英文资源；两种 locale 各 1,361 个键且键集一致，1,254 个静态引用全部存在，版本化清单记录每页实际引用数。 |
| 可见硬编码门禁 | Pass | SFC/TypeScript AST 检查覆盖模板文本、用户可见属性、插值分支、消息、确认提示及按钮、输入校验和错误赋值；15 个页面硬编码计数均为 0。 |
| 运行时业务值 | Pass | 状态、风险、阶段、渠道、来源、主体类型、合同主体和日/周/月粒度统一经 `valueLabel` 解析；未知值保持可读回退，不直接暴露下划线枚举。 |
| 状态切换一致性 | Pass | `PageShell` 与 `DataTable` 的 error/no-permission/empty 标题已双语化；已翻译错误在 locale 切换时按稳定错误键重新解析，消除中文错误详情滞留在英文页面的问题。 |
| 双语浏览器验收 | Pass | 桌面端抽检看板、合规、采购入库、销售出库、客户跟进、销售机会、付款发票、公告、合同和资质；中文/英文标题、动作与错误态正确切换，最终干净标签页无控制台错误。 |
| 响应式验收 | Pass | 375x844 中文销售页和英文看板截图均无文字重叠，文档 `scrollWidth` 等于 `clientWidth`；重置视口后 1280px 双语看板同样无横向溢出。 |
| 质量门禁 | Pass | locale 检查、`vue-tsc --noEmit`、Vite 生产构建、Node 语法、两份 JSON 解析和 `git diff --check` 全部通过。 |
| API/权限/审计/migration/seed | Pass | 本项只调整前端 locale、共享状态展示和检查脚本；未新增或修改 HTTP 契约、权限键、审计动作、migration 或 seed，无需修改 OpenAPI。 |
| CodeGraph 与边界 | Pass | 同步后索引为 702 files / 15,080 nodes / 46,281 edges，状态 up to date；未修改 `docs/refactor/old/`，未纳入用户已有文档与本地环境文件。 |

### Verification Commands

```powershell
npm --prefix web run check:i18n
npm --prefix web run typecheck
npm --prefix web run build
node --check web/scripts/h4-locale-baseline-check.mjs
node -e "JSON.parse(require('fs').readFileSync('web/i18n/pharma-oa-baseline.json','utf8')); JSON.parse(require('fs').readFileSync('web/src/i18n/pharma.json','utf8'))"
codegraph sync .
codegraph status .
codegraph query "localizeKnownError"
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 首次 `vue-tsc` 验收发现 `String.replaceAll` 超出项目 ES2020 目标。 | 改用全局正则替换下划线后重跑类型检查。 |
| 2 | Failed -> Doing | 英文浏览器模式下错误标题为 `Request failed`，但错误详情仍保留中文“服务暂时不可用”。 | 共享状态层按稳定错误键实时重翻译，并将错误、无权限和空状态标题纳入双语资源。 |
| 3 | Failed -> Doing | 中文销售页主按钮仍显示 `New order`，原检查器未解析插值条件分支。 | 扩展 AST 门禁并清理由其进一步发现的抽屉标题、确认框、校验和成功提示，校准页面引用基线后完成双语桌面/移动复验。 |

### Next Step

父任务 H4 保持 `Doing`。按正式 Work Item 顺序领取 `H4-03`，强化键盘操作、焦点恢复、ARIA、对比度、错误关联、减少动画和破坏性确认。

## H4-03 Pharma OA 可访问交互加固

- Date: 2026-07-19
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 加固 Pharma OA 15 个主页面及共享页面壳、表格、状态块和详情抽屉的键盘操作、焦点恢复、ARIA 名称、表单关联、减少动画和破坏性确认，并建立可重复执行的静态门禁。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 自动化可访问性门禁 | Pass | Vue SFC 与 TypeScript AST 检查覆盖 15 个页面，确认 46 个图标按钮具备可访问名称、14 个破坏性确认同时具备本地化确认/取消动作，且不存在正数 `tabindex`、无替代文本图片或缺失键盘语义的可点击原生元素。 |
| 键盘与焦点 | Pass | 桌面端仅用键盘打开和关闭客户抽屉、执行资质到期扫描；抽屉初始焦点位于对话框内，Escape 关闭后焦点分别恢复到“新建客户”和“到期扫描”触发按钮。 |
| ARIA 与表单关联 | Pass | `PageShell` 标题通过 `aria-labelledby` 关联页面区域，数据表具备可访问名称和本地化操作列表头，空/无权限状态使用 polite status、错误状态使用 assertive alert；抽检 12 个表单项的 label `for` 与控件 id 全部匹配。 |
| 可见焦点、对比度与减少动画 | Pass | 全局 `:focus-visible` 使用 3px 主题色轮廓并保留组件自身边界对比；`prefers-reduced-motion: reduce` 下滚动改为即时，动画和过渡缩短到近即时，同时不隐藏焦点或状态反馈。 |
| 破坏性操作 | Pass | 14 个确认/输入流程均显式提供中文默认、可切换英文的确认与取消文案；资质扫描对话框可由 Escape 取消，关闭后焦点恢复且未触发操作。 |
| 响应式验收 | Pass | 390x844 客户页无页面横向溢出，动作区正常换行；359px 抽屉没有页脚溢出，初始焦点位于对话框，Escape 关闭后返回触发按钮。 |
| 质量门禁 | Pass | `check:a11y`、locale 清单、`vue-tsc --noEmit`、Vite 生产构建、Node 语法和 `git diff --check` 全部通过；构建仅保留上游 Sass/Rollup 提示。 |
| API/权限/审计/migration/seed | Pass | 本项仅调整前端交互语义、共享组件、样式和检查脚本；未修改 HTTP 契约、权限键、审计动作、migration 或 seed，无需更新 OpenAPI。 |
| CodeGraph 与边界 | Pass | 同步后索引为 703 files / 15,114 nodes / 46,357 edges，状态 up to date；未修改 `docs/refactor/old/`，未纳入用户已有文档、本地索引或运行数据。 |

### Verification Commands

```powershell
npm --prefix web run check:a11y
node --check web/scripts/h4-accessibility-check.mjs
npm --prefix web run typecheck
npm --prefix web run build
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 首次静态门禁发现 14 个确认框缺少显式、本地化的取消按钮文案。 | 为全部确认和输入流程补充 `common.cancel`，再次扫描后通过 15 页面 / 46 图标按钮 / 14 确认流程门禁。 |
| 2 | Failed -> Doing | 首次类型验收发现 9 个页面因新增 `common.cancel` 引用与版本化 locale 基线各差 1。 | 按页面实际 AST 引用更新精确基线，重跑 locale 检查、`vue-tsc` 和生产构建后通过。 |

### Next Step

父任务 H4 保持 `Doing`。按正式 Work Item 顺序领取 `H4-04`，验证中英文 loading、empty、error、no-permission、saving、destructive 和 responsive 状态矩阵。

## H4-04 Pharma OA 双语响应式浏览器矩阵

- Date: 2026-07-21
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 为 Pharma OA 建立可重复执行的 Selenium 真浏览器矩阵，覆盖中文默认、英文切换、1440x1000 桌面和 390x844 窄屏，以及 loading、empty、error、no-permission、saving、destructive、responsive 七类状态；补齐路由级可见拒绝页并固化截图证据。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 双语状态矩阵 | Pass | `scripts/h4-browser-matrix.py` 使用真实 Chrome、admin 与 dept_admin 会话执行 2 locales x 2 viewports x 7 states，共 28/28 场景通过；中文为默认 locale，英文使用同一业务流程复验。 |
| Loading / empty / error | Pass | API 定向延迟显示页面骨架，无匹配客户筛选显示本地化空态，定向网络失败显示 assertive error；四组环境均保持页面横向溢出 0px、关键文本溢出 0。 |
| No-permission | Pass | 新增认证后可访问的 `/skoll/forbidden` 双语拒绝页；受限账号访问客户路由会保留 `from` 查询并显示 polite status 与仪表盘动作，不再静默跳回首页。 |
| Saving / destructive | Pass | 资质到期扫描在中英文桌面/窄屏均显示本地化确认与取消动作；仅延迟扫描 API 后，触发按钮呈现稳定 loading 状态，截图后取消未完成请求，未写入业务数据。 |
| Responsive / visual review | Pass | 四份 contact sheet 覆盖中英文桌面和 390x844；工具栏在窄屏纵向排列，表格维持内部滚动边界，确认、错误与拒绝状态无文字遮挡或页面横向滚动。 |
| 可重复证据 | Pass | `docs/refactor/current/evidence/h4-04/` 保存 4 张矩阵拼图、28 行 `matrix.json` 和中文默认复现说明；Selenium/Pillow 依赖固定在 `scripts/requirements-browser.txt`。 |
| 质量门禁 | Pass | Python 语法、28 场景浏览器矩阵、locale/accessibility 门禁、`vue-tsc --noEmit`、Vite 生产构建、JSON 解析和 `git diff --check` 全部通过；构建仅保留既有上游提示。 |
| API/权限/审计/migration/seed | Pass | 本项未修改 HTTP/OpenAPI、后端权限键、审计动作、migration 或 seed；仅调整前端权限守卫的拒绝呈现，继续使用既有路由权限元数据和会话权限。 |
| CodeGraph 与边界 | Pass | 同步后索引为 705 files / 15,165 nodes / 46,483 edges，状态 up to date；未修改 `docs/refactor/old/`，未纳入用户已有文档、本地索引、IDE 或运行数据。 |

### Verification Commands

```powershell
python -m pip install -r scripts/requirements-browser.txt
python -m py_compile scripts/h4-browser-matrix.py
python scripts/h4-browser-matrix.py --base-url http://127.0.0.1:5174
npm --prefix web run typecheck
npm --prefix web run build
Get-Content docs/refactor/current/evidence/h4-04/matrix.json -Raw | ConvertFrom-Json
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | `dept_admin` 访问 Pharma OA 客户路由时被静默重定向到仪表盘，无法验收可见 no-permission 状态。 | 新增双语 `/skoll/forbidden` 页面，权限守卫保留来源路径并展示明确拒绝状态；真实受限账号桌面/窄屏复验通过。 |
| 2 | Failed -> Doing | 首次完整矩阵用全局网络节流模拟 loading，同时延迟了 Vite 动态模块，客户页主体未挂载并最终超时。 | 改为页面启动时只定向延迟或拒绝目标 API，增加逐场景阶段输出；重跑 28 个状态全部通过并生成四份拼图。 |

### Next Step

父任务 H4 完成。按正式 Work Item 顺序领取 `H5-01`，建立数据库代表性数据集、查询计划、P95/P99 与慢查询预算。

## H5-01 数据库查询与索引基准

- Date: 2026-07-21
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 为 Pharma OA 的常用列表、仪表盘、库存、组织/负责人范围和报表队列建立 MySQL 代表性数据集、查询计划、P50/P95/P99 与慢查询预算；同步 GORM、MySQL/PostgreSQL 基线 migration、机器可读 schema 和持久化文档中的索引契约。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 代表性数据集 | Pass | `cmd/skoll-db-benchmark` 在 MySQL 5.7.26 中生成 50,000 行隔离夹具：客户 10,000、库存余额 20,000、库存预警 10,000、报表任务 10,000；仅允许显式 `--allow-write-fixtures` 写入，运行前后按 `h5-bench-` 前缀清理，最终四表残留均为 0。 |
| 查询计划与索引 | Pass | 6/6 查询使用 `ref` 访问并命中约定索引：客户常用列表、预警仪表盘、库存位置、组织范围、负责人范围和报表队列分别命中 `idx_pharma_customers_scope_status`、`idx_pharma_inventory_alerts_status_type_seen`、`uk_pharma_balance_position`、`idx_pharma_customers_org_status_code`、`idx_pharma_customers_owner_status_code`、`idx_pharma_export_jobs_owner_status_created`。 |
| P95/P99 与预算 | Pass | 每个查询预热 10 次并计时 100 次；P95 依次为 4.9154、20.2846、5.0241、2.5645、2.6405、4.6185 ms，均低于 25 ms 或仪表盘 40 ms 预算；P99 依次为 5.4790、22.4239、5.7937、3.1228、2.8340、5.4282 ms。完整机器可读结果保存在 `docs/refactor/current/evidence/h5-01/mysql-query-benchmark.json`。 |
| Schema 与 migration | Pass | 新增四个复合索引契约，并同步 GORM model、repository schema、MySQL/PostgreSQL 干净基线 migration 和 `docs/architecture/pharma-oa-persistence.md`；真实 MySQL migration 合同连续执行两次通过，验证建表和幂等升级路径。 |
| 安全与可重复性 | Pass | 基准工具要求 DSN 明确选择数据库，不输出 DSN/凭据；PowerShell 包装脚本使用框架一致的 `zh-CN` 默认中文并支持英文提示，参数限制为 5,000-100,000 基础行和 20-1,000 次迭代。 |
| 质量门禁 | Pass | 聚焦 Go 测试、真实 MySQL migration 合同、`go test ./...`、`go vet ./...`、证据 JSON 逐查询断言和 `git diff --check` 全部通过；同步修复供应商资质测试的固定日期漂移，使其继续验证 30 天有效期与 60 天提醒语义。 |
| API/权限/审计/migration/seed | Pass | 本项未改变 HTTP/OpenAPI 契约、权限键或审计动作；migration 与 schema 索引已同步，未新增 seed。基准夹具仅在显式写入模式创建，验收后无残留。 |
| CodeGraph 与边界 | Pass | 同步后索引为 706 files / 15,205 nodes / 46,609 edges，状态 up to date；未修改 `docs/refactor/old/`，未纳入用户已有文档、本地索引、IDE 或运行数据。 |

### Verification Commands

```powershell
$env:SKOLL_BENCHMARK_MYSQL_DSN = 'root:root@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local'
go run ./cmd/skoll-db-benchmark --allow-write-fixtures --rows 10000 --iterations 100 --output docs/refactor/current/evidence/h5-01/mysql-query-benchmark.json
./scripts/benchmark-pharma-oa-mysql.ps1 -Rows 10000 -Iterations 100 -OutputPath tmp/h5-01-wrapper-final.json
$env:SKOLL_TEST_MYSQL_DSN = 'root:root@tcp(127.0.0.1:3306)/skoll_acceptance?charset=utf8mb4&parseTime=True&loc=Local'
go test ./internal/store/sql/gormrepo -run '^TestPharmaMigrationSQLMySQL$' -count=1 -v
go test ./cmd/skoll-db-benchmark ./internal/repository/pharmaoa ./internal/store/sql/gormrepo ./internal/handler/http/v1/pharmaoa
go test ./...
go vet ./...
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 首次造数在第 5,000 行触发库存位置唯一键冲突。 | 将库存批次 ID 改为全数据集唯一值，清理夹具后重新造数。 |
| 2 | Failed -> Doing | 基线计划中预警仪表盘 P95 为 87.898 ms，范围与报表查询缺少规定索引。 | 补充复合索引并同步 model、schema、双数据库 migration 与文档。 |
| 3 | Failed -> Doing | 首轮调优后仪表盘仍选择旧索引且 P95 为 89.306 ms，范围查询中的 OR 条件阻碍复合索引。 | 调整预警索引列顺序，将组织和负责人范围拆为独立基准。 |
| 4 | Failed -> Doing | 组织范围使用 IN 条件时优化器仍选择客户编码索引。 | 按实际单组织数据范围改为等值条件，复验命中 `idx_pharma_customers_org_status_code`。 |
| 5 | Failed -> Doing | 全量 Go 测试发现供应商资质夹具中的固定日期已随当前日期失效。 | 使用相对当前时间的 30 天有效期，保留 60 天到期提醒语义后重跑全量测试。 |
| 6 | Failed -> Doing | 包装脚本使用 1,000 行小样本时优化器选择旧预警索引，不能代表生产查询计划。 | 将最小基础数据集提升到 5,000 行，并以 10,000 行、100 次迭代完成正式复验。 |
| 7 | Failed -> Doing | 最终夹具核对首次使用了不存在的无前缀表名。 | 读取实际 migration 表名，改用 `pharma_oa_*` 四张表重试。 |
| 8 | Failed -> Doing | 第二次夹具核对错误引用库存余额不存在的 `batch_no` 字段。 | 与基准工具的清理实现统一按 `id LIKE 'h5-bench-%'` 断言，四表均返回 0。 |

### Next Step

父任务 H5 进入 `Doing`。按正式 Work Item 顺序领取 `H5-02`，实现服务端分页与前端大列表性能基线。

## H5-02 服务端分页与前端大列表 UX

- Date: 2026-07-21
- Status flow: `Todo -> Doing -> (Failed -> Doing) x 12 -> Review -> Done`
- Scope: 为 Pharma OA 员工、客户、产品、供应商和仓库主数据建立统一的 `total/cursor/offset/limit` 服务端分页契约；员工与客户页面改用可取消请求、短期页面缓存和虚拟表格，并在默认中文、英文、桌面与窄屏下完成真实浏览器验收。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| API 与存储分页 | Pass | 五个主数据列表统一返回 `items/offset/limit/total/hasMore/nextCursor/sort`；cursor 使用不透明 Base64 offset 且优先于显式 offset。GORM repository 使用同一过滤条件执行 `COUNT` 与有界 `ORDER BY code ASC LIMIT/OFFSET`，memory repository 保持相同 contract。 |
| 契约与真实 MySQL | Pass | handler 覆盖分页元数据、非法 cursor 400、cursor 往返及空证书数组；公开与内嵌 OpenAPI 完全一致，167 个 path 可解析，五个端点均声明分页参数和响应 schema；`TestPharmaMasterRepositoriesMySQLContract` 在本地 `skoll_acceptance` 实库通过。 |
| 前端大列表 | Pass | 员工与客户使用 250ms debounce、`AbortController`、10 秒页面缓存、25/50/100 页大小和 Element Plus 虚拟表格；写操作显式失效对应缓存，lookup 调用限制为最多 200 条，不再对完整结果集做浏览器内过滤。 |
| 多语言与响应式 | Pass | `el-config-provider` 使 Element Plus 跟随框架 locale，默认 `zh-CN`；中文显示“共 125 条”，英文显示 “Total 125”。桌面 1440x1000 与移动 390x844 页面及分页控件横向溢出均为 0。 |
| 浏览器矩阵 | Pass | `scripts/h5-large-list-browser.py` 使用真实 Chrome 完成 2 locales x 2 viewports x 2 pages，共 8/8 页面检查；每个 50 行页面仅挂载 24 个可见行，表格高度 520px，第二页稳定从 `0051` 开始，返回第一页命中缓存。中文桌面额外验证 `H5E0 -> H5E01` 取消旧请求，最终 26 条且无错误态。 |
| 可重复夹具与证据 | Pass | `scripts/h5-large-list-fixtures.ps1` 默认中文、支持英文，仅写入 `h5-list-*` 员工/客户并在 seed 前及验收后清理；正式矩阵前后计数为 `125/125 -> 0/0`。`docs/refactor/current/evidence/h5-02/` 保存 README、机器可读 JSON 和 8 张截图。 |
| 质量门禁与 bundle | Pass | `go test ./...`、`go vet ./...`、真实 MySQL 契约、OpenAPI 断言、Python 编译、locale/accessibility/large-list 门禁、`vue-tsc --noEmit`、Vite production build 和 `git diff --check` 全部通过。构建转换 3612 个模块；DataTable chunk 48.93 kB / gzip 16.24 kB，pagination chunk 11.80 kB / gzip 4.07 kB；仅保留既有 Sass 与 VueUse 上游警告。 |
| API/权限/审计/migration/seed | Pass | 本项仅改变列表响应 contract 并同步两份 OpenAPI；继续复用既有读取权限与审计边界，未新增或变更权限键、审计动作、migration 或产品 seed。验收 SQL 是显式可清理的临时夹具，不属于正式 seed。 |
| CodeGraph 与边界 | Pass | 同步后索引为 710 files / 15,319 nodes / 46,984 edges，状态 up to date；未修改 `docs/refactor/old/`，未纳入用户已有 `docs/README.md`、`docs/collaboration.md`、`.codegraph/`、`.vscode/`、`AGENTS.md` 或 `data/`。 |

### Verification Commands

```powershell
go test ./...
go vet ./...
$env:SKOLL_TEST_MYSQL_DSN = 'root:root@tcp(127.0.0.1:3306)/skoll_acceptance?charset=utf8mb4&parseTime=True&loc=Local'
go test ./internal/store/sql/gormrepo -run '^TestPharmaMasterRepositoriesMySQLContract$' -count=1 -v
npm --prefix web run typecheck
npm --prefix web run build
python -m py_compile scripts/h5-large-list-browser.py
$env:Path = 'D:\workspace\phpEnv\server\mysql\mysql-8.0\bin;' + $env:Path
./scripts/h5-large-list-fixtures.ps1 -Action seed -Rows 125
try { python -u ./scripts/h5-large-list-browser.py --base-url http://127.0.0.1:5176 } finally { ./scripts/h5-large-list-fixtures.ps1 -Action cleanup }
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 泛型 memory page 在错误分支返回 `nil`，无法编译。 | 返回类型化空 page，重跑 repository/service/handler 聚焦测试。 |
| 2 | Failed -> Doing | 首版 paged interface 扩散到全部 workflow repository，造成非主数据实现不满足接口。 | 将分页能力收敛到五个主数据 repository，保持其他 aggregate contract 不变。 |
| 3 | Failed -> Doing | 浏览器脚本拼接 base URL 时重复路径，无法进入目标页面。 | 将 base URL 与 `/skoll/...` 路径分离并加入显式参数。 |
| 4 | Failed -> Doing | 真实员工列表返回 `certificates: null`，前端证书单元格读取长度时报错并停留 loading。 | 修复 employee clone 使空证书稳定序列化为 `[]`，新增 handler 回归断言。 |
| 5 | Failed -> Doing | 分页按钮重绘导致 stale element，且仅等待总数会误把首屏相同总数当作筛选完成。 | 增加 idle/stale 重试和精确 keyword resource 请求等待。 |
| 6 | Failed -> Doing | 二次筛选时全页 skeleton 卸载输入框，无法触发取消旧请求。 | 仅首次加载显示 PageShell skeleton，后续查询保持 toolbar 挂载并由表格呈现 loading。 |
| 7 | Failed -> Doing | 取消场景预期总数误写为 25，且浏览器将已知 favicon 404 计为业务错误。 | 按 `H5E01` 实际集合修正为 26，仅忽略明确的 favicon 404，其他 severe 仍失败。 |
| 8 | Failed -> Doing | 一次复验在夹具已清理时启动，空总数文本触发转换异常。 | 固化 seed/browser/finally-cleanup 生命周期，并让空文本返回等待哨兵值。 |
| 9 | Failed -> Doing | 完整四组合矩阵超过首次 4 分钟外层上限，进程被验收 shell 中止。 | 单跑中文桌面确认功能后将外层上限调整为 10 分钟，完整矩阵 230.8 秒通过。 |
| 10 | Failed -> Doing | 首次全库测试命令误用 1 秒外层窗口，被工具中止。 | 使用 5 分钟上限重跑，随后暴露并处理真实契约问题。 |
| 11 | Failed -> Doing | `TestOpenAPIContractFilesStayInSync` 发现公开 OpenAPI 与运行时内嵌副本不一致。 | 将五端点分页参数和两个 schema 同步到内嵌文件，契约测试及全库测试通过。 |
| 12 | Failed -> Doing | 首次 OpenAPI 自定义断言被 PowerShell 展开 `$ref`，第二次引号转义仍不正确。 | 改用 here-string 传入 Python，双文件 167 paths / 5 endpoints 断言通过。 |

### Next Step

父任务 H5 保持 `Doing`。按正式 Work Item 顺序领取 `H5-03`，验证长任务、重试、死信、资源稳定性和监控阈值。

## H5-03 长任务、重试与资源稳定性

- Date: 2026-07-21
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 为主数据导入导出、逾期提醒、异步报表导出和插件业务事件建立可重复 race/soak 验收；补齐事件重试耗尽或处理器卸载后的可观测 `dead_letter` 终态，并输出 P50/P95/P99、堆与 goroutine 增量、重试、死信和重复副作用阈值证据。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 导入导出 soak | Pass | 主数据导入、重复提交与 XLSX 导出执行 100 次，100/100 成功；P50/P95/P99 为 11.1640/16.6587/19.9975 ms，堆增长 78,360 B，goroutine 增长 0，重复副作用 0。 |
| 提醒与报表重试 | Pass | 逾期提醒和真实异步报表各执行 100 次“首次失败 -> 重试成功 -> 幂等重放”；两类任务均 100/100 成功、各记录 100 次重试、重复通知/文件为 0，goroutine 增长均为 0；报表 P95/P99 为 17.1745/27.0794 ms。 |
| 插件事件与死信 | Pass | 插件事件执行 100 次临时失败与重试，100/100 成功，P95/P99 为 0.6801/0.9548 ms；永久失败在第 3 次尝试后进入 1 条 `dead_letter`，处理器卸载同样进入死信并向调用者返回错误，后续不会再次调度。 |
| 指标与告警阈值 | Pass | `internal/testing/jobsoak` 统一计算延迟分位数、堆/goroutine 增量、失败、重试、死信和重复副作用；独立阈值单测证明超限会使报告失败，`h5-job-metrics-audit.ps1` 强制校验 schema、`zh-CN`、race 标记及全部阈值。 |
| 可重复脚本与证据 | Pass | `scripts/h5-job-soak.ps1 -Iterations 100` 在 Windows PowerShell 下完成两组 race 测试和指标审计；`docs/refactor/current/evidence/h5-03/` 保存三份机器可读 JSON、中文 README 与英文摘要。 |
| 质量门禁 | Pass | `go test ./...`、`go vet ./...`、医药任务/事件 100 次 race/soak、采集器 race、指标审计、CodeGraph 同步和 `git diff --check` 全部通过。 |
| API/权限/审计/migration/seed | Pass | 本项未改变 HTTP/OpenAPI、权限键、菜单或数据库结构，未新增 migration/seed；继续复用提醒和报表既有审计动作，事件死信仅扩展内部内存重试记录状态。 |
| 多语言与边界 | Pass | 脚本和 JSON 固定记录默认 locale `zh-CN`，证据说明中文优先并含英文摘要；未修改 `docs/refactor/old/`，未纳入用户已有文档、IDE、CodeGraph 目录或运行数据。 |
| CodeGraph | Pass | 同步后索引为 713 files / 15,370 nodes / 47,181 edges，状态 up to date。 |

### Verification Commands

```powershell
./scripts/h5-job-soak.ps1 -Iterations 100
./scripts/h5-job-metrics-audit.ps1 -ReportPath @(
  './docs/refactor/current/evidence/h5-03/pharma-job-soak.json',
  './docs/refactor/current/evidence/h5-03/business-event-soak.json'
)
go test -race ./internal/testing/jobsoak -count=1
go test ./...
go vet ./...
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 首轮聚焦测试中，类型化 `nil *reportAuditFixture` 装入接口后仍被视为非空，报表执行审计时触发空指针。 | 使用显式审计夹具实例，重跑聚焦测试通过。 |
| 2 | Failed -> Doing | 首次正式 race/soak 的业务测试均通过，但 Windows PowerShell 5 按默认代码页解析 UTF-8 中文脚本，指标审计脚本出现字符串终止符错误。 | 将脚本控制台与错误文案改为 ASCII，JSON 和文档继续保留中文默认内容。 |
| 3 | Failed -> Doing | 第二次正式运行的 race 测试仍通过，但 PowerShell 默认代码页读取 UTF-8 JSON 后破坏中文场景名，`ConvertFrom-Json` 失败。 | 使用 `.NET ReadAllText(..., UTF8)` 强制解码两份报告，独立指标审计与完整脚本复验通过。 |
| 4 | Failed -> Doing | 将报表 soak 从同步夹具增强为真实异步派发时，首轮编译引用了不存在的 `ReportExportStatus` 别名。 | 改用现有 `ReportExportJobStatus`，重跑聚焦测试及 100 次正式 race/soak，异步任务全部回收。 |

### Next Step

父任务 H5 保持 `Doing`。按正式 Work Item 顺序领取 `H5-04`，发布中文默认、含英文摘要的容量与性能基线。

## H5-04 容量与性能基线

- Date: 2026-07-21
- Status flow: `Todo -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 将 H5-01 数据库查询、H5-02 服务端分页与前端大列表、H5-03 长任务稳定性汇总为中文默认、含英文摘要的正式基线；补充 allocation benchmark、关键 bundle 原始/gzip 大小、明确容量限制和自动回归阈值。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 环境与数据集 | Pass | 正式报告记录 Windows 10.0.26200、i5-13500H/16 logical processors、Go 1.24.1、Node 22.14.0、Vite 5.4.21、MySQL 5.7.26、本地 Chrome，以及 50,000 行数据库、125+125 浏览器、1,000 条内存 benchmark 和每场景 100 次 soak 数据规模。 |
| 数据库 P50/P95/P99 | Pass | 汇总 6 个索引查询，P95 为 2.5645-20.2846 ms，均满足普通查询 25 ms、仪表盘 40 ms 预算；规定索引和 `ref` 访问类型纳入回归失败条件。 |
| 分页与大列表 | Pass | 固化页面大小不超过 100、50 条页面挂载不超过 30 行、横向溢出为 0、缓存和取消请求必须通过；引用 H5-02 的 2 locales x 2 viewports x 2 pages 机器证据。 |
| 长任务稳定性 | Pass | 汇总 4 类 100 次 race/soak 的 P50/P95/P99、堆与 goroutine 增长、重试、死信和重复副作用；全部成功、重复副作用为 0、goroutine 增长为 0，永久插件失败进入死信。 |
| Allocation | Pass | `go test -benchmem` 每项 200 次、3 个样本：cache 最大 2,199 ns/238 B/3 allocs，员工内存分页最大 28,737,660 ns/1,605,958 B/20,070 allocs，客户范围最大 19,392,334 ns/1,158,208 B/14,620 allocs，均低于独立阈值。 |
| Bundle | Pass | 重新构建 3,612 个模块；DataTable 为 48,925 B / gzip 16,223 B，Pagination 为 11,799 B / gzip 4,071 B，分别满足 64/20 KiB 和 16/6 KiB 阈值。仅保留既有 Sass 与 VueUse 上游警告。 |
| 报告与限制 | Pass | `performance_capacity_baseline_2026-07-21.md` 中文优先并含英文摘要，明确单机、数据量、浏览器、内存夹具、外部服务、顺序任务、后台 worker、1,000 条进程内导出等限制，不将基线表述为生产并发承诺。 |
| 可重复证据 | Pass | `scripts/h5-performance-baseline.ps1` 验证上游三项证据，运行 benchmark/build，计算 gzip 并生成 `evidence/h5-04/performance-baseline.json`；`h5-performance-doc-audit.ps1` 校验文档结构与证据 schema/locale/pass 状态。完整复现 115.4 秒通过。 |
| 质量门禁 | Pass | 统一复现、文档审计、`go test ./...`、`go vet ./...`、前端 i18n/a11y/large-list/typecheck、production build、CodeGraph up to date 和 `git diff --check` 全部通过。 |
| API/权限/审计/migration/seed | Pass | 本项仅新增报告、证据和验收脚本，不改变 HTTP/OpenAPI、权限、菜单、审计动作、migration 或 seed。 |
| CodeGraph 与边界 | Pass | 索引保持 713 files / 15,370 nodes / 47,181 edges，状态 up to date；未修改 `docs/refactor/old/`，未纳入用户已有文档、IDE、CodeGraph 目录或运行数据。 |

### Verification Commands

```powershell
./scripts/h5-performance-baseline.ps1 -BenchmarkCount 3 -BenchmarkIterations 200
./scripts/h5-performance-doc-audit.ps1
go test ./...
go vet ./...
npm --prefix web run typecheck
npm --prefix web run build
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | 首次文档审计中，Windows PowerShell 5 按默认代码页解析脚本里的 UTF-8 中文标题字面量，产生字符串终止符错误。 | 保持脚本源码纯 ASCII，改为按章节数量和 `zh-CN`、P50/P95/P99、B/op、allocs/op、Bundle、dead-letter、复现参数等稳定标记审计；JSON 继续强制 UTF-8 读取，重跑通过。 |

### Next Step

父任务 H5 完成。按正式 Work Item 顺序领取 `H6-01`，演练部署、备份、恢复、升级与回滚。

## H6-01 部署、升级与恢复演练

- Date: 2026-07-21
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 修复并验证 Docker Compose、Docker 和 Kubernetes 部署资产；新增 MySQL 校验备份、隔离恢复、版本区间升级和恢复点回滚脚本；增加公开 readiness 契约；在本地 MySQL 隔离库完成可重复全链路演练并发布中英双语证据。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 部署清单 | Pass | Compose 配置可解析并包含 MySQL、应用数据、日志、插件包四个持久卷；Kustomize 渲染 Deployment、Service、ConfigMap 和两个 PVC，容器以非 root、只读根文件系统运行，`/app/data` 与 `/app/plugins` 可写。 |
| 干净部署与探针 | Pass | 当前二进制在随机 MySQL 5.7 隔离库完成干净启动并创建 47 张表；`GET /skoll/health` 与新增 `GET /skoll/ready` 均返回 200，运行时和公开 OpenAPI 两份契约同步。 |
| 备份与恢复 | Pass | 单事务备份生成 91,705 B SQL、SHA-256 元数据；恢复脚本校验摘要并重建指定隔离库，业务标记恢复为 `clean-v1`。系统库名和缺少 `-AllowRecreate` 均失败关闭。 |
| 前向升级 | Pass | 从已安装的 21 号基线执行 `20260718_000022_complete_pharma_oa_schema.sql`，目标表由 0 增至 2；升级前自动生成恢复点，升级后服务 health/readiness 通过。缺少 `-AllowUpgrade` 失败关闭。 |
| 回滚 | Pass | 将升级前恢复点还原到另一隔离库，两张 22 号目标表均不存在，原业务标记 `source-v1` 保持；不执行 down migration 或旧结构兼容。 |
| 可重复演练 | Pass | `h6-deployment-recovery-smoke.ps1` 必须显式传入 `-AllowDatabaseLifecycle`，最终机器报告耗时 54.95 秒且 `passed=true`；结束后四个随机 `skoll_h6_*` 数据库和临时文件全部清理。 |
| 文档与多语言 | Pass | `deployment_recovery_rehearsal_2026-07-21.md`、用户部署文档及三类部署 README 默认中文并含英文摘要；清理当前文档中的旧 JWT 变量、旧 API 前缀配置和旧探针路径。 |
| API/权限/审计/migration/seed | Pass | 新增公开 `/ready` 并同步 OpenAPI；探针不授予业务权限且不产生业务审计。未修改产品 migration 或 seed，只新增显式确认保护的运维脚本。 |
| 质量门禁 | Pass | `go test ./...`、`go vet ./...`、聚焦路由/OpenAPI 测试、Compose/Kustomize、PowerShell 语法、确认开关、证据契约、链接、diff 和历史目录边界全部通过。 |
| CodeGraph | Pass | 同步后索引为 716 files / 15,372 nodes / 47,187 edges，状态 up to date；未修改 `docs/refactor/old/`，未纳入用户已有文档、IDE、CodeGraph 目录或运行数据。 |

### Verification Commands

```powershell
$env:SKOLL_H6_MYSQL_PASSWORD = '<local-test-password>'
./scripts/h6-deployment-recovery-smoke.ps1 -AllowDatabaseLifecycle
go test ./...
go vet ./...
go test ./internal/handler/http ./internal/bootstrap -run 'TestRouterHealthAndReadinessRoutes|TestOpenAPIContractFilesStayInSync|TestHealth|TestRunner' -count=1
docker compose -f deploy/compose/docker-compose.yaml config --quiet
kubectl kustomize deploy/k8s | Out-Null
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | MySQL 8 `mysqldump` 对 MySQL 5.7 查询不存在的 `COLUMN_STATISTICS`。 | 检测客户端能力并加入 `--skip-column-statistics`。 |
| 2 | Failed -> Doing | migration 文件名解析未组合日期与六位序号，未选中脚本。 | 按 `YYYYMMDD_NNNNNN` 生成 14 位版本号。 |
| 3 | Failed -> Doing | 本地 MySQL 默认 MyISAM，历史建表索引超过 1000 字节限制。 | 在每个 migration 输入前设置会话默认引擎为 InnoDB。 |
| 4 | Failed -> Doing | Windows `Start-Process` 拆分带空格的 `--init-command`。 | 生成临时 UTF-8 SQL 输入并在 finally 中清理。 |
| 5 | Failed -> Doing | 历史 migration 与当前 GORM 表名不能从空库连续回放。 | 按真实升级路径从已安装基线恢复后只执行目标增量。 |
| 6 | Failed -> Doing | Docker daemon 未启动时，环境探测被 PowerShell 提升为异常并阻断证据写入。 | 改用进程退出码记录 `daemon-unavailable`，不将清单校验误报为容器运行。 |

### Next Step

父任务 H6 保持 `Doing`。按正式 Work Item 顺序领取 `H6-02`，补齐示例、监管、许可、数据安全和支持边界。

## H6-02 样板、监管、许可证与支持边界

- Date: 2026-07-21
- Status flow: `Todo -> Doing -> Review -> Done`
- Scope: 建立中文默认、英文对照的公开发布与运营边界；统一医药 OA 样板定位、虚构演示数据、持久化模式、监管验证、安全密钥、MIT/第三方许可证和社区支持范围；将边界纳入插件指南与发布清单。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 样板与监管定位 | Pass | `release-boundaries.md` 和英文对照明确 `pharma_oa` 是行业样板，不是经监管验证的 GSP/GMP、医疗器械、临床或药品追溯系统；功能名称不构成合规保证、医疗建议或监管认证。 |
| 演示与数据安全 | Pass | 明确所有 `DEMO-*` 人员、机构、药品、证照和交易均为虚构测试数据，禁止在生产应用 demo seed 或将真实个人/健康/商业秘密写入仓库、Issue、日志和证据；列明最小化、权限、加密、留存删除和备份恢复责任。 |
| 密钥与安全报告 | Pass | 要求更换默认账号密码和 `SKOLL_SECURITY_JWT_SECRET`，凭据进入密钥管理系统；公开文档链接 `SECURITY.md` 私密报告流程，并明确维护者不持有采用者生产密钥。 |
| 许可证 | Pass | 明确项目根 `LICENSE` 为 MIT、分发时保留声明、软件按原样提供；第三方组件保留自身许可证，发布者必须基于实际 Go/Node/容器构建生成并审查 SBOM 和第三方许可证清单。 |
| 支持边界 | Pass | 社区支持按维护者可用时间尽力提供，不承诺商业支持、响应时限、可用性、数据恢复、监管验证或安全值守 SLA；生产采用者负责容量、监控、值班、回滚和灾难恢复。 |
| 中英文一致 | Pass | 中文默认与英文文档均有 7 个对应章节，双向链接并包含 GSP/GMP、demo、密钥、MIT、SBOM、数据卷、支持和持久化模式等共同标记。 |
| 插件与发布指南 | Pass | 医药 OA 两份 README 引用正式边界，修正已过期的“仓储仅进程内”和“主流程未本地化”表述；发布清单新增许可证、监管、数据、安全渠道、支持和双语边界门禁，并停止指示更新 `old/`。 |
| 自动审计与链接 | Pass | `h6-release-boundary-audit.ps1` 使用 UTF-8 读取并校验双语结构、稳定标记、根许可证、过期表述、当前发布记录规则及六份文档的全部本地链接。 |
| API/权限/审计/migration/seed | Pass | 本项只修改公开文档和审计脚本，不改变 HTTP/OpenAPI、权限、审计动作、migration 或 seed；文档只说明现有持久化与 demo seed 行为。 |
| 边界与 CodeGraph | Pass | 插件定向测试、`git diff --check`、归档目录边界和 CodeGraph up to date 均通过；未修改 `docs/refactor/old/`，未纳入用户已有文档、IDE、CodeGraph 目录或运行数据。 |

### Verification Commands

```powershell
./scripts/h6-release-boundary-audit.ps1
go test ./internal/plugin -run TestPharmaOA -count=1
codegraph status .
git diff --check
```

### Next Step

父任务 H6 保持 `Doing`。按正式 Work Item 顺序领取 `H6-03`，执行最终强化发布门禁并关闭批次。

## H6-03 最终强化发布门禁与批次结项

- Date: 2026-07-21
- Status flow: `Todo -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Failed -> Doing -> Review -> Done`
- Scope: 复验 H1-H6 全部代码、前端、多语言、插件、数据库、长任务、性能、部署恢复、文档和运行时证据；发布最终问题清单、发布检查快照与下一批建议；关闭 `skoll-hardening-2026-07-18`。

### Acceptance Result

| Gate | Result | Evidence |
| --- | --- | --- |
| 批次一致性 | Pass | `h6-hardening-closeout-audit.ps1` 识别 25 个 Work Item、6 个父任务和 25 个独立验收条目；全部为 `Done`，归档目录无变更。 |
| Go 与契约 | Pass | tracked Go 文件格式正确；`go test ./... -count=1`、`go vet ./...`、OpenAPI 同步和插件 manifest 定向测试通过。 |
| 前端与多语言 | Pass | 1,365 个 locale keys、1,256 个引用、15 个医药 OA 视图、可访问性、大列表、typecheck 和 3,612 模块生产构建通过；默认 `zh-CN`。 |
| 插件与业务闭环 | Pass | 中英文插件生命周期和医药 OA 端到端 smoke 通过；权限、菜单、审计、失败状态、分页与性能检查通过。 |
| 数据库 | Pass | 隔离 `skoll_acceptance` 上 MySQL migration、seed、重启、备份恢复和仓储契约通过；SQLite 契约通过；本机 PostgreSQL 未配置并如实记录为发布前限制。 |
| 长任务与容量 | Pass | H5-03 两份 soak 指标审计及 H5-04 容量文档审计通过，无重复副作用或静默失败证据。 |
| 部署与恢复 | Pass | 随机隔离 MySQL 数据库完成 47 表干净部署、health/readiness、备份恢复、前向升级和恢复点回滚，临时资源自动清理。 |
| 隔离运行时 | Pass | 随机 loopback 端口上的内存模式实例完成 health、readiness、公开 OpenAPI、登录/token/profile 和 `pharma_oa` 发现审查。 |
| 文档与边界 | Pass | 中文默认结项报告含英文摘要；中英文发布边界、许可证、示例/监管、数据安全、密钥和支持限制审计通过。 |
| 问题清单 | Pass | P0 为 0；现场 PostgreSQL、Docker daemon、真实 Kubernetes、tag/SBOM/signing 等发布时工作明确列为非虚构限制。 |
| API/权限/审计/migration/seed | Pass | H6-03 不修改运行时 API、OpenAPI、权限、审计动作、migration 或 seed；只执行现有契约与运行时复验并新增结项脚本、报告和证据。 |
| CodeGraph 与边界 | Pass | 最终同步后索引 up to date；`git diff --check` 通过；未修改 `docs/refactor/old/`，未纳入用户已有文档、IDE、CodeGraph 目录或运行数据。 |

### Verification Commands

```powershell
go test ./... -count=1
go vet ./...
go test ./internal/handler/http -run TestOpenAPIContractFilesStayInSync -count=1
go test ./internal/plugin -run TestValidatePluginManifestsUnderPluginsDir -count=1
npm --prefix web run typecheck
npm --prefix web run build
./scripts/smoke-pharma-oa-plugin.ps1
./scripts/smoke-pharma-oa-plugin.ps1 -Locale en-US
./scripts/smoke-pharma-oa-e2e.ps1
./scripts/smoke-pharma-oa-e2e.ps1 -Locale en-US
./scripts/smoke-pharma-oa-performance-permission.ps1
$env:SKOLL_TEST_MYSQL_DSN = 'root:<password>@tcp(127.0.0.1:3306)/skoll_acceptance?parseTime=true'
./scripts/smoke-pharma-oa-database.ps1
./scripts/h5-job-metrics-audit.ps1 -ReportPath @('./docs/refactor/current/evidence/h5-03/pharma-job-soak.json','./docs/refactor/current/evidence/h5-03/business-event-soak.json')
./scripts/h5-performance-doc-audit.ps1
./scripts/h6-release-boundary-audit.ps1
$env:SKOLL_H6_MYSQL_PASSWORD = '<local-test-password>'
./scripts/h6-deployment-recovery-smoke.ps1 -AllowDatabaseLifecycle
./scripts/h6-runtime-review.ps1
./scripts/h6-hardening-closeout-audit.ps1 -ExpectedWorkItemStatus Done -ExpectedParentStatus Done
codegraph sync .
codegraph status .
git diff --check
```

### Retry Log

| Attempt | Status | Failure evidence | Retry action |
| --- | --- | --- | --- |
| 1 | Failed -> Doing | Runtime review异常路径在子进程退出前读取锁定的 stderr。 | 先停止进程再读取诊断，重跑。 |
| 2 | Failed -> Doing | Windows PowerShell 将 OpenAPI 响应内容作为 `byte[]`，误报缺少 `/ready`。 | 显式 UTF-8 解码响应，重跑。 |
| 3 | Failed -> Doing | 前端构建与全部 smoke 的组合命令超过外层 124 秒限制，无法得到完整结论。 | 拆分构建和 smoke、分别设置时限，两组均重跑通过。 |
| 4 | Failed -> Doing | 数据库验收拒绝重置普通 `skoll` 库。 | 改用受保护的 `skoll_acceptance` 隔离库，不触碰现有应用库，重跑通过。 |

### Next Step

父任务 H6 与强化批次全部完成。当前正式 Work Item 表没有剩余 `Todo`；后续实现必须先在 `docs/refactor/current/` 创建新的父任务表、Work Item 表和验收表。
