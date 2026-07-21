# 医药 OA 行业插件

[English](README.en.md) | 简体中文

`pharma_oa` 是 Skoll 的医药行业样板插件。它用一套可安装的插件契约组织主数据、审批、进销存、合规、CRM、经营分析和端到端演示能力，适合开发者了解复杂业务插件如何复用 Skoll 的权限、审计、工作流、通知、文件和前端宿主能力。

> 当前定位：可演示、可测试、可二次开发的行业样板，不是经过法规验证的生产级 GSP/GMP 系统。采用前必须阅读 [发布、监管与支持边界](../../docs/user/release-boundaries.md)。

## 快速开始

### 1. 准备环境

先按 [Skoll 快速开始](../../docs/quick-start.md) 安装 Go 1.24、Node.js 20 和项目依赖。请从仓库根目录执行后续命令。

### 2. 启动 Skoll

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/dev-up.ps1 -ForceRestart
```

后端启动时会扫描 `plugins/*/plugin.yaml`，自动安装并启用本目录中的插件。访问以下地址确认环境：

- 前端：`http://127.0.0.1:5173/skoll`
- 插件管理：登录后打开“插件”页面，确认“医药 OA”状态为已启用
- 医药 OA 首页：`http://127.0.0.1:5173/skoll/pharma-oa/employees`
- 经营看板：`http://127.0.0.1:5173/skoll/pharma-oa/dashboard`

开发环境默认账号为 `admin`，默认密码为 `Admin@123456`。持久环境首次登录后应立即修改密码。

### 3. 初始化演示数据

```powershell
$base = "http://127.0.0.1:8080/skoll"
$login = Invoke-RestMethod -Method Post -Uri "$base/v1/auth/login" `
  -ContentType "application/json" `
  -Body (@{ account = "admin"; password = "Admin@123456" } | ConvertTo-Json)
$headers = @{ Authorization = "Bearer $($login.data.token)" }

Invoke-RestMethod -Method Post `
  -Uri "$base/v1/pharma-oa/demo-seed/apply" `
  -Headers $headers
```

一次初始化会创建员工及资质、药品、合格供应商和客户、冷链仓库、采购审批、采购入库、销售出库及客户跟进。对同一运行实例重复执行会复用已有场景，不会重复写入库存流水。

### 4. 运行端到端验收

```powershell
.\scripts\smoke-pharma-oa-e2e.ps1
```

脚本默认输出中文；英文输出使用：

```powershell
.\scripts\smoke-pharma-oa-e2e.ps1 -Locale en-US
```

验收覆盖员工入职、采购申请审批、采购入库、销售出库、资质预警、客户跟进、库存证据和演示数据幂等性。

## 插件生命周期

仓库开发模式推荐依赖启动时自动发现。需要验证独立生命周期时运行：

```powershell
.\scripts\smoke-pharma-oa-plugin.ps1
```

该命令验证预检、安装、重复安装失败、启用、禁用、权限目录、菜单、路由、审计和失败状态。运行中的实例也可通过插件管理页面执行启用和禁用。

脚本默认输出中文；英文输出使用 `.\scripts\smoke-pharma-oa-plugin.ps1 -Locale en-US`。

若插件目录在服务启动后才加入，可先调用 `POST /skoll/v1/plugins/preflight`，再调用 `POST /skoll/v1/plugins/install` 和 `POST /skoll/v1/plugins/pharma_oa/enable`。请求体中的路径为 `plugins/pharma_oa`；不要对已经由启动扫描安装的实例重复安装。

## 功能范围

| 业务域 | 已实现能力 | 主要入口 |
| --- | --- | --- |
| 主数据 | 员工、药品、供应商、客户、仓库、资质、导入导出 | `/skoll/pharma-oa/employees`、`/customers` 及 `/v1/pharma-oa/*` |
| 工作流与采购 | 采购申请、审批/驳回、采购订单、采购入库 | `/skoll/pharma-oa/purchase-inbounds` 及采购 API |
| 销售与库存 | 销售订单、批次出库、盘点、调拨、库存流水、库存预警 | `/skoll/pharma-oa/sales` 及库存 API |
| 协同与合规 | 公告、合同、统一资质、质量投诉、药品召回、冷链、合规看板 | `/skoll/pharma-oa/announcements`、`/contracts`、`/qualifications`、`/quality-complaints`、`/drug-recalls`、`/cold-chain`、`/compliance-dashboard` |
| CRM 与财务协同 | 客户跟进、销售机会、回款计划、发票记录、逾期提醒 | `/skoll/pharma-oa/customer-follow-ups`、`/sales-opportunities`、`/payment-invoices` |
| 分析与演示 | 经营指标、响应式看板、异步 CSV 报表、完整演示数据 | `/skoll/pharma-oa/dashboard`、演示数据及报表 API |

完整 HTTP 契约以 [OpenAPI](../../docs/api/openapi.yaml) 为准。宿主前端使用 `/v1/pharma-oa/*`；`plugin.yaml` 中 `/v1/plugins/pharma_oa/api/*` 路由用于插件目录、权限和扩展注册，不应另建一套旧接口兼容层。

## 监管与数据边界

- “合规”“资质”“召回”“冷链”和“审计”是样板功能名称，不表示已满足任何地区的监管、验证、电子记录或质量体系要求。
- 内置 `DEMO-*` 人员、机构、药品、证照和交易均为虚构数据，不得替换或混入真实个人信息、患者数据、生产凭据或法定记录后继续作为演示 seed 使用。
- 生产采用者负责法规识别、系统验证、数据分类、最小权限、留存删除、备份恢复、事故响应和上线批准；维护者不提供监管认证或生产 SLA。
- 完整许可证、安全、数据和支持边界以 [中文默认说明](../../docs/user/release-boundaries.md) 及其 [English version](../../docs/user/release-boundaries.en.md) 为准。

## 运行边界

- `plugin.yaml` 是安装、菜单、配置、权限、路由、审计动作和事件声明的唯一插件契约。
- 当前业务服务由 Skoll 宿主在启动时装配，代码位于 `internal/domain/pharmaoa`、`internal/service/pharmaoa` 和 `internal/handler/http/v1/pharmaoa`。
- 当前业务前端是宿主集成页面，路由位于 `web/src/router/index.ts`，类型化客户端位于 `web/src/pharma-oa/api.ts`。`static/` 只用于插件页面和生命周期能力验证，不是完整业务控制台。
- `mysql` / `postgres` 模式使用 SQL 仓储持久化医药 OA 业务数据，演示 seed 可在重启后识别并复用；`memory` 模式仍是进程内开发夹具，重启即丢失。报表文件使用现有私有对象存储。
- 禁用插件会撤销插件目录中的菜单、权限和扩展注册，但不会热卸载已经由宿主启动的医药 OA 服务。不要据此声称具备进程级沙箱或热卸载能力。
- 本样板不提供旧 API、旧数据结构、旧插件格式或旧页面路径兼容方案。

## 多语言

- 插件声明 `zh-CN` 和 `en-US`，宿主未保存语言偏好时默认使用 `zh-CN`。
- 插件名称、菜单和配置项提供中英文 manifest 字段；宿主根据当前语言选择对应文本。
- 本文件为默认中文说明，[README.en.md](README.en.md) 提供英文说明。
- 新增前端业务文案时应使用宿主 `useI18n()` 和 `web/src/i18n/index.ts`，不能在组件中新增只支持单一语言的硬编码文本。
- 主流程可见文案已纳入中文默认与英文切换验收；新增文案必须同时补齐两种语言并保持状态与响应式覆盖。

## 扩展一个业务域

1. 在 `internal/domain/pharmaoa` 定义领域对象和不变量，在 `internal/service/pharmaoa` 实现用例，避免把业务规则放进 HTTP handler。
2. 在 `internal/handler/http/v1/pharmaoa` 增加 JWT 身份、权限、错误状态和审计覆盖，并同步两份 OpenAPI 契约。
3. 在 `plugin.yaml` 同步声明权限、路由、审计动作、菜单或事件；权限键统一使用 `pharma_oa.<resource>.<action>`。
4. 在 `web/src/pharma-oa/api.ts` 增加类型化客户端，在 `web/src/views/Pharma*` 增加宿主页，并在路由和菜单中接入。
5. 中文和英文文案同时进入框架 i18n 字典，默认中文；页面必须覆盖 loading、empty、error、no-permission、saving/destructive 和 responsive 状态。
6. 增加服务、HTTP、权限、审计、manifest、前端和端到端测试。若引入持久表，同一任务必须包含 migration/seed 影响和多数据库验证。

更通用的插件契约和开发流程见 [插件开发指南](../../docs/development/plugin-guide.md)。

## 验证命令

```powershell
go test ./internal/plugin -run TestPharmaOA -count=1
.\scripts\smoke-pharma-oa-plugin.ps1
.\scripts\smoke-pharma-oa-e2e.ps1
go test ./...
cd web
npm run typecheck
npm run build
```

## 常见问题

| 现象 | 检查项 |
| --- | --- |
| 插件列表没有医药 OA | 确认从仓库根目录启动，且 `plugins/pharma_oa/plugin.yaml` 可读取；查看后端启动日志 |
| 菜单不可见 | 确认插件已启用，并授予 `pharma_oa.menu.read`；`super_admin` 可直接验证 |
| API 返回 401/403 | 重新登录获取 JWT，并检查对应 `pharma_oa.*` 权限 |
| 演示数据消失 | 确认是否使用了 `memory` 模式；持久模式检查 DSN、migration 和启动日志，恢复前先验证备份 |
| 重复安装失败 | 启动扫描已经完成安装；改用列表、启用或禁用操作 |
| 找不到业务 API | 使用 `/skoll/v1/pharma-oa/*` 宿主路径，并以 OpenAPI 为准 |
