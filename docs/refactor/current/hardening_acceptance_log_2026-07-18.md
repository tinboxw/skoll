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
