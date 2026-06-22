# Skoll 重构父任务表

> 执行规则: 本文档是父任务表。实际执行与提交以 [work_items.md](work_items.md) 的 Work Item 为最小单元。每个 Work Item 验收通过后提交一次代码。父任务用于里程碑分组和验收聚合，不作为最小执行或提交单元。

状态枚举:

父任务表和 Work Item 表必须使用同一套状态枚举。

- `Todo`: 待执行
- `Doing`: 执行中
- `Review`: 待验收
- `Failed`: 验收失败，需返工
- `Done`: 验收通过且已提交
- `Blocked`: 外部条件阻塞

## M0: 架构章程与质量基线

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 1 | M0-01 | `skoll-refactor-governance` | 固化架构模型 | 架构决策文档 | 明确 Modular Monolith + Clean/Hexagonal + Tactical DDD；明确不做兼容方案 | 文档审阅 | Done |
| 2 | M0-02 | `skoll-refactor-governance` | 建立任务验收模板 | `acceptance_log.md` | 模板包含改动文件、命令结果、人工验收、提交信息 | 文档审阅 | Done |
| 3 | M0-03 | `skoll-quality-gate` | 建立当前质量基线 | 当前测试/构建/覆盖率记录 | `go test ./...`、覆盖率、前端 build/typecheck 有真实结果 | `go test ./...`; `go test ./... -coverprofile=coverage.out`; `cd web && npm run build` | Done |
| 4 | M0-04 | `skoll-refactor-governance` | 校准任务表 | 更新本文件 | 所有任务具备交付物、验收要求、验证命令和状态 | 文档审阅 | Done |
| 5 | M0-05 | `skoll-docs-writer` | 清理文档入口 | `docs/README.md`、`docs/refactor/README.md` | 当前计划入口唯一，旧计划进入 archive | `rg "docs/refactor|legacy-plans" docs/README.md` | Done |

里程碑验收:

- 当前重构入口可从 `docs/README.md` 找到。
- 所有旧计划文档不再作为执行入口。
- 质量基线有真实命令结果。
- M0 完成后提交一次里程碑文档提交。

## M1: 权限资源目录与菜单注册中心

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 6 | M1-01 | `skoll-permission-menu-refactor` | 定义 PermissionResource 模型 | `internal/domain/permission` | 支持 api/menu/button/data_scope/plugin；单测覆盖校验规则 | `go test ./internal/domain/permission` | Done |
| 7 | M1-02 | `skoll-permission-menu-refactor` | 定义 MenuNode 模型 | `internal/domain/menu` | 支持父子、路径、组件、图标、排序、显隐、授权字段 | `go test ./internal/domain/menu` | Done |
| 8 | M1-03 | `skoll-database-development` | 新增权限与菜单迁移 | MySQL/PostgreSQL migration | 可在空库执行；唯一约束和索引明确 | `go test ./internal/store/sql/...` | Done |
| 9 | M1-04 | `skoll-permission-menu-refactor` | 实现 catalog repository/store | repository + memory/sql store | service 不依赖 GORM；重复注册幂等 | `go test ./internal/repository/... ./internal/store/...` | Done |
| 10 | M1-05 | `skoll-permission-menu-refactor` | 实现 permission catalog service | service | 注册、列表、启停、来源过滤、diff 可用 | `go test ./internal/service/permission/...` | Done |
| 11 | M1-06 | `skoll-api-contracts` | 实现权限与菜单 API | HTTP handler + OpenAPI | API、OpenAPI、错误码一致；无 handler 直连 store | `go test ./internal/handler/http/v1/...` | Done |
| 12 | M1-07 | `skoll-plugin-platform` | 插件 manifest 权限/菜单导入 | plugin manager 集成 | 插件启用导入，禁用隐藏访问，权限进入 catalog | `go test ./internal/plugin/...` | Done |
| 13 | M1-08 | `skoll-vue-frontend` | 前端权限 store 与菜单 store | web store/client | sidebar、route guard、按钮权限同源 | `cd web && npm run build` | Done |
| 14 | M1-09 | `skoll-web-ui-design` | 权限矩阵和菜单管理页接入 catalog | Vue 页面 | 授权/撤销/排序/显隐后 UI 与后端一致 | `cd web && npm run build` | Done |
| 15 | M1-10 | `skoll-testing-automation` | M1 端到端验收 | smoke 或手工脚本 | 新角色授权后侧栏、路由、按钮、API 权限一致 | `go test ./...`; `cd web && npm run build` | Done |

里程碑验收:

- 新建角色只授予部分菜单时，侧栏、路由、按钮和 API 行为一致。
- 插件启用后菜单和权限进入目录，禁用后不可访问。
- OpenAPI、权限目录文档和 UI 均同步。
- 验收通过后提交一次 M1 完成提交。

## M2: 审计、登录日志、错误日志统一化

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 16 | M2-01 | `skoll-audit-observability-refactor` | 定义 AuditEvent 分类与 action 命名 | domain + 文档 | operation/login/error/plugin/security 类型明确 | `go test ./internal/domain/audit/...` | Done |
| 17 | M2-02 | `skoll-audit-observability-refactor` | 登录日志与错误日志模型 | domain/store | 记录 IP、UA、trace、结果、错误摘要 | `go test ./internal/store/...` | Done |
| 18 | M2-03 | `skoll-audit-observability-refactor` | 审计写入中间件 | middleware | 成功、失败、未授权均记录 | `go test ./internal/handler/middleware/...` | Done |
| 19 | M2-04 | `skoll-api-contracts` | 审计查询、详情和导出 API | handler + OpenAPI + 对齐记录 | 支持分类、actor、resource、action、时间、风险筛选；导出/list 过滤一致，OpenAPI/client 字段同步 | `go test ./internal/handler/http/v1/audit/...` | Done |
| 20 | M2-05 | `skoll-vue-frontend` | 审计页升级 | Vue 页面 + 审计页 UX baseline | tab、筛选、详情、导出、风险标签、空态、错误态、无权限态可用且有验收标准 | `cd web && npm run build` | Done |
| 21 | M2-06 | `skoll-testing-automation` | 审计 E2E 验收 | smoke 脚本/记录 + audit smoke fixture | 登录失败、权限拒绝、插件、菜单、导出固定样本可查询 | `powershell -ExecutionPolicy Bypass -File ./scripts/smoke-auth-audit.ps1` | Done |

里程碑验收:

- 审计列表、详情、导出可用。
- 高风险操作失败也有审计。
- Memory 和 SQL store 均有测试。

## M3: 文件与对象存储平台

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 22 | M3-01 | `skoll-file-storage-refactor` | 定义 FileObject 与 ObjectStore port | domain + port | 字段覆盖 key、name、size、mime、hash、owner、visibility、storage、status | `go test ./internal/domain/file/...` | Done |
| 23 | M3-02 | `skoll-file-storage-refactor` | 实现 local adapter | store adapter | 上传、下载、删除、路径穿越防护通过 | `go test ./internal/store/object/...` | Done |
| 24 | M3-03 | `skoll-database-development` | 文件元数据 store | SQL/memory store | 元数据与对象写入失败状态一致 | `go test ./internal/store/...` | Done |
| 25 | M3-04 | `skoll-api-contracts` | 文件 API | handler + OpenAPI | 上传、下载、删除、列表、权限校验完整 | `go test ./internal/handler/http/v1/file/...` | Done |
| 26 | M3-05 | `skoll-file-storage-refactor` | 分片上传 | service/API | init/upload/complete/abort 可用；hash 校验失败不可完成 | `go test ./internal/service/file/...` | Todo |
| 27 | M3-06 | `skoll-vue-frontend` | 文件管理页 | Vue 页面 | 上传进度、搜索、预览、下载、删除、错误提示可用 | `cd web && npm run build` | Todo |
| 28 | M3-07 | `skoll-security-hardening` | 文件安全测试 | tests | 未授权下载、路径穿越、非法 mime、超大文件被拒绝 | `go test ./...` | Todo |

里程碑验收:

- 本地存储可完整上传、下载、删除。
- 分片上传可完成和中止。
- 文件操作均有权限和审计。

## M4: 字典、配置、组织与数据范围

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 29 | M4-01 | `skoll-dictionary-organization-refactor` | 字典独立模型和迁移 | domain + migration | DictionaryType/DictionaryItem 独立表 | `go test ./internal/domain/system/... ./internal/store/...` | Todo |
| 30 | M4-02 | `skoll-cache-development` | 字典缓存策略 | cache/service | 更新字典后缓存失效 | `go test ./internal/cache/... ./internal/service/system/...` | Todo |
| 31 | M4-03 | `skoll-api-contracts` | 字典 API/UI 接入 | handler + Vue | 类型、条目、状态、排序、搜索完整 | `go test ./...`; `cd web && npm run build` | Todo |
| 32 | M4-04 | `skoll-data-dictionary-config` | 配置 schema 注册 | domain/service/UI | 系统与插件配置通过 schema 渲染和校验 | `go test ./internal/service/system/...`; `cd web && npm run build` | Todo |
| 33 | M4-05 | `skoll-dictionary-organization-refactor` | 组织/部门/岗位模型 | domain/store/API | 部门树、岗位、用户归属可用 | `go test ./...` | Todo |
| 34 | M4-06 | `skoll-permission-rbac` | 数据范围查询过滤 | service/store | 用户列表按 all/department/department_tree/self/custom 返回不同结果 | `go test ./internal/service/rbac/... ./internal/service/user/...` | Todo |
| 35 | M4-07 | `skoll-vue-frontend` | 组织管理与数据范围 UI | Vue 页面 | 授权时可选择数据范围，用户归属可编辑 | `cd web && npm run build` | Todo |

里程碑验收:

- 字典驱动表单选项。
- 用户列表按数据范围过滤。
- 组织和岗位在用户管理中可用。

## M5: 代码生成器 v1

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 36 | M5-01 | `skoll-generator-refactor` | GeneratorSpec 模型 | domain | 描述模块、表、字段、索引、校验、菜单、权限、页面 | `go test ./internal/domain/generator/...` | Todo |
| 37 | M5-02 | `skoll-generator-refactor` | 模板规范 | docs/templates | 输出路径、命名、冲突策略明确 | 文档审阅 | Todo |
| 38 | M5-03 | `skoll-generator-refactor` | dry-run 与 diff | service | 写入前展示文件清单和 diff | `go test ./internal/service/generator/...` | Todo |
| 39 | M5-04 | `skoll-code-generator` | 后端 CRUD 生成 | generator templates | 输出 domain/service/store/handler/migration/OpenAPI | `go test ./...` | Todo |
| 40 | M5-05 | `skoll-code-generator` | 前端 CRUD 生成 | Vue/API/store/templates | 输出列表、表单、API client、菜单 | `cd web && npm run build` | Todo |
| 41 | M5-06 | `skoll-generator-refactor` | 生成历史与回滚 | store/service | 记录批次、hash、spec；只回滚未被用户改动文件 | `go test ./internal/service/generator/...` | Todo |
| 42 | M5-07 | `skoll-testing-automation` | 生成器 golden 测试 | tests | 幂等、冲突检测、回滚测试通过 | `go test ./...` | Todo |
| 43 | M5-08 | `skoll-generator-refactor` | 示例模块生成验收 | examples/plugins | 单表 CRUD 生成后前后端跑通 | `go test ./...`; `cd web && npm run build` | Todo |

里程碑验收:

- 单表 CRUD 从 spec 到前后端页面完整跑通。
- 生成代码通过测试和构建。
- dry-run、历史、回滚可用。

## M6: 插件市场与生产生命周期

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 44 | M6-01 | `skoll-plugin-marketplace-refactor` | marketplace index schema | schema/docs | 定义 id、version、skoll_version、risk、signature、source、changelog | schema 校验 | Todo |
| 45 | M6-02 | `skoll-plugin-platform` | 本地市场 service/API | service/API | 展示本地插件、可安装包、签名、风险 | `go test ./internal/plugin/... ./internal/handler/http/v1/plugin/...` | Todo |
| 46 | M6-03 | `skoll-plugin-marketplace-refactor` | 插件安装预检 | service/UI | 展示权限、菜单、配置、资产、迁移、风险、签名 diff | `go test ./...`; `cd web && npm run build` | Todo |
| 47 | M6-04 | `skoll-plugin-platform` | 插件迁移 hook | plugin/service | install/upgrade/downgrade/uninstall 状态可追踪 | `go test ./internal/plugin/...` | Todo |
| 48 | M6-05 | `skoll-security-hardening` | 插件签名策略升级 | plugin/security | 签名覆盖 manifest、资产清单、入口和前端资产 hash | `go test ./internal/plugin/...` | Todo |
| 49 | M6-06 | `skoll-plugin-marketplace-refactor` | 灰度/回滚产品化 | service/frontend | 影响真实路由、菜单、功能可见性 | `go test ./...`; `cd web && npm run build` | Todo |
| 50 | M6-07 | `skoll-web-ui-design` | 插件安全报告 UI | Vue 页面 | 安装前后均可查看风险和权限影响 | `cd web && npm run build` | Todo |
| 51 | M6-08 | `skoll-testing-automation` | 插件生命周期验收 | tests | install/enable/disable/upgrade/rollback 全链路通过 | `go test ./...` | Todo |

里程碑验收:

- 插件安装前可看到风险、权限和迁移影响。
- 插件升级失败可回滚到上一可用版本。
- 生命周期操作均有审计。

## M7: 发布质量、文档与开源体验

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 52 | M7-01 | `skoll-quality-gate` | CI 质量门禁 | CI | gofmt、go test、前端 build、OpenAPI、manifest 校验进入 CI | CI 运行 | Todo |
| 53 | M7-02 | `skoll-testing-automation` | 覆盖率门槛 | CI/docs | 核心包覆盖率目标和例外名单明确 | `go test ./... -coverprofile=coverage.out` | Todo |
| 54 | M7-03 | `skoll-performance-scaling` | 性能基线 | tests/docs | 登录、用户列表、角色列表、插件列表、审计查询有 QPS/P95/P99 | k6 或 vegeta 脚本 | Todo |
| 55 | M7-04 | `skoll-docs-writer` | 快速开始和运行手册 | docs | 新环境 30 分钟内启动、登录、安装插件、生成模块 | 手工复现 | Todo |
| 56 | M7-05 | `skoll-open-source-framework` | 示例项目与发布检查清单 | examples/docs | 至少一个生成模块和一个插件示例可运行 | `go test ./...`; `cd web && npm run build` | Todo |

里程碑验收:

- 质量门禁稳定运行。
- 文档能指导新环境启动和操作。
- 示例模块和插件能证明框架能力。

## FE: 前端体验专项父任务

> FE 专项与 M1-M7 并行推进，实际提交仍以 [work_items.md](work_items.md) 中的 FE0-FE6 Work Item 为最小单元。

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 57 | FE0 | `skoll-frontend-design-refactor` / `skoll-frontend-testing-refactor` / `skoll-frontend-performance-refactor` | 建立前端体验、构建、类型和浏览器验收基线 | 前端基线记录与验收模板 | 当前页面体验、build、typecheck、bundle warning、浏览器检查项均被记录 | `cd web && npm run build`; `cd web && npm run typecheck` | Done |
| 58 | FE1 | `skoll-frontend-design-refactor` | 建立 Skoll Admin 设计系统与视觉规范 | 页面布局、表格、表单、状态、响应式规范 | 后台界面专业克制、可扫描；无嵌套卡片、营销页风格、装饰性渐变干扰 | 文档审阅 | Done |
| 59 | FE2 | `skoll-frontend-coding-refactor` | 建立前端架构与数据流规范 | API client、Pinia、Router、权限、SchemaForm、错误处理规范 | 前端数据流统一，菜单、路由、按钮、插件权限同源 | `rg -n "apiGet|defineStore|beforeEach|SchemaForm|StateBlock|v-permission|canAccess" web/src; cd web && npm run typecheck` | Done |
| 60 | FE3 | `skoll-frontend-design-refactor` / `skoll-frontend-testing-refactor` | 升级核心后台页面体验 | Dashboard/User/Role/Permission/Menu/Plugin/Audit/Setting 页面改造 | 每个核心页面具备正常、加载、空、错误、无权限、保存中和窄屏状态 | `cd web && npm run build`; 浏览器验收 | Done |
| 61 | FE4 | `skoll-frontend-design-refactor` / `skoll-frontend-performance-refactor` | 升级插件与开发者门户体验 | Plugin/Marketplace/Dev Portal 页面改造 | 插件权限、风险、配置、日志、发布、回滚、任务状态可读且可操作 | `cd web && npm run build`; 浏览器验收 | Done |
| 62 | FE5 | `skoll-frontend-testing-refactor` | 建立前端测试与验收体系 | typecheck/build/smoke/权限/响应式验收清单 | 前端任务不能只以 build 通过作为验收；关键状态必须被记录 | `cd web && npm run typecheck`; `cd web && npm run build` | Done |
| 63 | FE6 | `skoll-frontend-performance-refactor` | 建立前端性能与可观测基线 | bundle、懒加载、表格、请求数量、插件重面板性能记录 | 主要页面按需加载；大表格有降级策略；性能变化进入验收记录 | `cd web && npm run build` | Done |

## N0: 发布前冻结与质量门禁

| 顺序 | ID | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|
| 64 | N0 | `skoll-refactor-governance` / `skoll-quality-gate` / `skoll-docs-writer` | 发布前冻结与质量门禁 | 文档入口、质量门禁记录、范围冻结说明 | N0-01 到 N0-05 全部通过验收；不引入兼容方案或新范围扩散 | `rg -n "N0-01|N0-02|N0-03|N0-04|N0-05" docs/refactor/work_items.md docs/refactor/acceptance_log.md` | Done |
