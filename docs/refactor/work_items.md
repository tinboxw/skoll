# Skoll 重构细粒度 Work Items

> 执行规则: Work Item 是最小执行和提交单元。每完成一个 Work Item 并通过验收后，提交一次代码。  
> 当前拆分范围: M0-M2 与 FE0-FE6。M3-M7 在 M2 验收完成后滚动拆分；滚动拆分完成前，M3-M7 父任务只作为路线图，不作为直接执行单元。
> 覆盖校验: M0-M2 每个父任务至少有一个 Work Item；当前校验结果为 M0-01 至 M0-05、M1-01 至 M1-10、M2-01 至 M2-06 均已覆盖。

状态枚举:

- `Todo`: 待执行
- `Doing`: 执行中
- `Review`: 待验收
- `Failed`: 验收失败，需返工
- `Done`: 验收通过且已提交
- `Blocked`: 外部条件阻塞

## 拆分原则

1. 每个 Work Item 尽量控制在 0.5-1 天内可完成。
2. 每个 Work Item 应尽量只涉及一个边界: domain、store、service、handler、frontend、test、docs。
3. 跨边界任务必须拆分，不允许一个 Work Item 同时完成后端、前端、文档和测试的大包。
4. 每个 Work Item 必须有验收要求和验证命令。
5. 验收通过后提交一次代码，提交信息使用 `<work-item-id>: <summary>`。
6. 不做旧接口、旧数据结构、旧插件格式、旧页面路径的兼容方案。

## M0 Work Items: 架构章程与质量基线

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 1 | M0-01-01 | M0-01 | `skoll-refactor-governance` | 校准架构决策文档标题、状态、适用范围 | `architecture_and_execution_plan.md` | 明确当前是唯一架构决策；无旧计划入口引用 | 文档审阅 | Done |
| 2 | M0-01-02 | M0-01 | `skoll-refactor-governance` | 固化不做兼容方案规则 | `architecture_and_execution_plan.md`、`README.md` | 规则同时出现在执行入口和治理原则中 | `rg "不做旧接口\|不做兼容方案" docs/refactor docs/README.md` | Done |
| 3 | M0-01-03 | M0-01 | `skoll-refactor-governance` | 固化分层依赖规则 | `architecture_and_execution_plan.md` | domain/service/repository/store/handler/web/plugin 边界清晰 | 文档审阅 | Done |
| 4 | M0-01-04 | M0-01 | `skoll-refactor-governance` | 关联重构专用 skills | `architecture_and_execution_plan.md` | M0-M6 均能找到对应 skill | `rg "skoll-.*refactor" docs/refactor/architecture_and_execution_plan.md` | Done |
| 5 | M0-02-01 | M0-02 | `skoll-refactor-governance` | 建立验收记录模板字段 | `acceptance_log.md` | 包含状态、日期、执行人、提交、改动文件、验收项 | 文档审阅 | Done |
| 6 | M0-02-02 | M0-02 | `skoll-refactor-governance` | 补齐边界同步验收项 | `acceptance_log.md` | API/OpenAPI、权限、审计、migration、前端、文档均有检查项 | 文档审阅 | Done |
| 7 | M0-02-03 | M0-02 | `skoll-refactor-governance` | 补齐失败返工记录区 | `acceptance_log.md` | 失败原因、返工动作、重新验收结果字段齐全 | 文档审阅 | Done |
| 8 | M0-02-04 | M0-02 | `skoll-refactor-governance` | 补齐一任务一提交说明 | `README.md`、`acceptance_log.md` | 明确验收通过后提交一次代码 | `rg "提交一次代码\|一任务一提交" docs/refactor` | Done |
| 9 | M0-03-01 | M0-03 | `skoll-quality-gate` | 运行 Go 全量测试 | 质量基线记录 | 记录命令、结果、失败包、失败原因 | `go test ./...` | Done |
| 10 | M0-03-02 | M0-03 | `skoll-quality-gate` | 运行 Go 覆盖率统计 | `coverage.out` 或报告 | 记录总覆盖率和低覆盖包清单 | `go test ./... -coverprofile=coverage.out`; `go tool cover -func=coverage.out` | Done |
| 11 | M0-03-03 | M0-03 | `skoll-quality-gate` | 运行前端构建 | 质量基线记录 | 记录 build 结果和 warning | `cd web && npm run build` | Done |
| 12 | M0-03-04 | M0-03 | `skoll-quality-gate` | 检查前端 typecheck 命令 | 质量基线记录 | 若脚本存在则执行并记录；不存在则记录缺口 | `cd web && npm run typecheck` | Done |
| 13 | M0-03-05 | M0-03 | `skoll-quality-gate` | 记录测试副作用 | 质量基线记录 | 运行门禁后 `git status --short` 输出被记录；污染文件列入修复项 | `git status --short` | Done |
| 14 | M0-04-01 | M0-04 | `skoll-refactor-governance` | 校准父任务表状态枚举 | `task_board.md` | 状态枚举与 work item 状态一致 | 文档审阅 | Done |
| 15 | M0-04-02 | M0-04 | `skoll-refactor-governance` | 校准父任务与 work item 关系 | `task_board.md`、`work_items.md` | task_board 是父任务，work_items 是最小提交单元 | `rg "Work Item\|最小执行" docs/refactor` | Done |
| 16 | M0-04-03 | M0-04 | `skoll-refactor-governance` | 校准 M0-M2 Work Item 覆盖率 | `work_items.md` | M0、M1、M2 每个父任务至少有一个 work item | 文档审阅 | Done |
| 17 | M0-04-04 | M0-04 | `skoll-refactor-governance` | 设置 M3-M7 滚动拆分规则 | `work_items.md` | 明确 M2 验收后再拆 M3-M7 | 文档审阅 | Done |
| 18 | M0-05-01 | M0-05 | `skoll-docs-writer` | 更新 `docs/refactor/README.md` 文档结构 | `README.md` | 列出 `work_items.md` | `rg "work_items.md" docs/refactor/README.md` | Done |
| 19 | M0-05-02 | M0-05 | `skoll-docs-writer` | 更新顶层 `docs/README.md` 当前入口 | `docs/README.md` | 当前执行入口包含 work items | `rg "work_items.md" docs/README.md` | Done |
| 20 | M0-05-03 | M0-05 | `skoll-docs-writer` | 校验旧计划文档归档 | `source_map.md` | 旧计划类文档在 legacy-plans 中，当前入口不引用其为执行源 | `Get-ChildItem docs/archive/legacy-plans` | Done |
| 21 | M0-05-04 | M0-05 | `skoll-docs-writer` | M0 文档验收记录 | `acceptance_log.md` | M0 文档整理任务有验收记录 | 文档审阅 | Done |

## M1 Work Items: 权限资源目录与菜单注册中心

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 22 | M1-01-01 | M1-01 | `skoll-permission-menu-refactor` | 创建 permission domain 包结构 | `internal/domain/permission` | 包可编译，无 store/service 依赖 | `go test ./internal/domain/permission` | Done |
| 23 | M1-01-02 | M1-01 | `skoll-permission-menu-refactor` | 定义 PermissionResource 类型和值对象 | domain code | 支持 api/menu/button/data_scope/plugin 类型 | `go test ./internal/domain/permission` | Done |
| 24 | M1-01-03 | M1-01 | `skoll-permission-menu-refactor` | 实现 PermissionResource 校验规则 | domain code + tests | key/name/module/source 必填，type 枚举校验 | `go test ./internal/domain/permission` | Done |
| 25 | M1-01-04 | M1-01 | `skoll-permission-menu-refactor` | 实现 permission risk/source metadata | domain code + tests | risk/source/metadata 可序列化并校验 | `go test ./internal/domain/permission` | Done |
| 26 | M1-01-05 | M1-01 | `skoll-permission-menu-refactor` | 补 permission domain README | docs in package | 说明类型、命名和 no-compat 规则 | 文档审阅 | Done |
| 27 | M1-02-01 | M1-02 | `skoll-permission-menu-refactor` | 创建 menu domain 包结构 | `internal/domain/menu` | 包可编译，无 store/service 依赖 | `go test ./internal/domain/menu` | Done |
| 28 | M1-02-02 | M1-02 | `skoll-permission-menu-refactor` | 定义 MenuNode 类型和值对象 | domain code | parent/path/component/icon/sort/visible/source 字段完整 | `go test ./internal/domain/menu` | Done |
| 29 | M1-02-03 | M1-02 | `skoll-permission-menu-refactor` | 实现 MenuNode 校验规则 | domain code + tests | path、name、sort、source 校验通过 | `go test ./internal/domain/menu` | Done |
| 30 | M1-02-04 | M1-02 | `skoll-permission-menu-refactor` | 实现菜单树排序与过滤规则 | domain code + tests | 同级排序稳定；隐藏节点和无权节点可过滤 | `go test ./internal/domain/menu` | Done |
| 31 | M1-02-05 | M1-02 | `skoll-permission-menu-refactor` | 补 menu domain README | docs in package | 说明菜单节点、树、权限字段 | 文档审阅 | Done |
| 32 | M1-03-01 | M1-03 | `skoll-database-development` | 设计权限表 migration | MySQL/PostgreSQL SQL | 字段、索引、唯一约束齐全 | migration review | Done |
| 33 | M1-03-02 | M1-03 | `skoll-database-development` | 设计菜单表 migration | MySQL/PostgreSQL SQL | parent/source/path/sort 索引齐全 | migration review | Done |
| 34 | M1-03-03 | M1-03 | `skoll-database-development` | 增加 gormrepo model | SQL models | model 字段与 migration 一致 | `go test ./internal/store/sql/gormrepo/...` | Done |
| 35 | M1-03-04 | M1-03 | `skoll-database-development` | 注册新 model 到 all_models | store bootstrap | AutoMigrate/模型列表包含新表 | `go test ./internal/store/sql/gormrepo/...` | Done |
| 36 | M1-03-05 | M1-03 | `skoll-database-development` | migration 文档更新 | migrations README | 说明新表用途和执行顺序 | 文档审阅 | Done |
| 37 | M1-04-01 | M1-04 | `skoll-permission-menu-refactor` | 定义 permission repository 接口 | repository code | service-facing 方法覆盖 register/list/get/update state | `go test ./internal/repository/...` | Done |
| 38 | M1-04-02 | M1-04 | `skoll-permission-menu-refactor` | 定义 menu repository 接口 | repository code | service-facing 方法覆盖 tree/list/upsert/reorder | `go test ./internal/repository/...` | Done |
| 39 | M1-04-03 | M1-04 | `skoll-permission-menu-refactor` | 实现 memory permission store | memory store + tests | 注册幂等、source filter、enable/disable 通过 | `go test ./internal/store/memory/...` | Done |
| 40 | M1-04-04 | M1-04 | `skoll-permission-menu-refactor` | 实现 memory menu store | memory store + tests | upsert/tree/reorder/visibility 通过 | `go test ./internal/store/memory/...` | Done |
| 41 | M1-04-05 | M1-04 | `skoll-database-development` | 实现 SQL permission store | gormrepo store + tests | 与 repository 契约一致 | `go test ./internal/store/sql/gormrepo/...` | Done |
| 42 | M1-04-06 | M1-04 | `skoll-database-development` | 实现 SQL menu store | gormrepo store + tests | 与 repository 契约一致 | `go test ./internal/store/sql/gormrepo/...` | Done |
| 43 | M1-04-07 | M1-04 | `skoll-database-development` | 接入 store factory | factory code | memory/mysql/postgres bundle 可提供新 store | `go test ./internal/store/...` | Done |
| 44 | M1-05-01 | M1-05 | `skoll-permission-menu-refactor` | 定义 permission catalog service interface | service code | 接口覆盖注册、列表、启停、diff | `go test ./internal/service/permission/...` | Done |
| 45 | M1-05-02 | M1-05 | `skoll-permission-menu-refactor` | 实现 RegisterResource | service code + tests | 幂等、校验失败、仓储失败路径覆盖 | `go test ./internal/service/permission/...` | Done |
| 46 | M1-05-03 | M1-05 | `skoll-permission-menu-refactor` | 实现 ListResources/GetResource | service code + tests | 支持 type/source/enabled 过滤 | `go test ./internal/service/permission/...` | Done |
| 47 | M1-05-04 | M1-05 | `skoll-permission-menu-refactor` | 实现 Enable/Disable | service code + tests | 状态变更写审计动作预留 | `go test ./internal/service/permission/...` | Done |
| 48 | M1-05-05 | M1-05 | `skoll-permission-menu-refactor` | 实现 permission diff | service code + tests | install preflight 可复用 diff 输出 | `go test ./internal/service/permission/...` | Done |
| 49 | M1-05-06 | M1-05 | `skoll-permission-menu-refactor` | 定义 menu registry service interface | service code | 接口覆盖 merge/tree/filter/reorder | `go test ./internal/service/menu/...` | Done |
| 50 | M1-05-07 | M1-05 | `skoll-permission-menu-refactor` | 实现菜单合并与排序 | service code + tests | 系统/插件/生成模块菜单稳定合并 | `go test ./internal/service/menu/...` | Done |
| 51 | M1-05-08 | M1-05 | `skoll-permission-menu-refactor` | 实现菜单权限过滤 | service code + tests | requiredRoles/requiredPermissions 生效 | `go test ./internal/service/menu/...` | Done |
| 52 | M1-06-01 | M1-06 | `skoll-api-contracts` | 设计 permission API 契约 | OpenAPI draft | list/detail/enable/disable/diff 请求响应明确 | OpenAPI review | Done |
| 53 | M1-06-02 | M1-06 | `skoll-api-contracts` | 实现 permission handler | handler code + tests | HTTP 状态码、错误码、响应格式一致 | `go test ./internal/handler/http/v1/permission/...` | Done |
| 54 | M1-06-03 | M1-06 | `skoll-api-contracts` | 设计 menu API 契约 | OpenAPI draft | tree/save/reorder/visibility 请求响应明确 | OpenAPI review | Done |
| 55 | M1-06-04 | M1-06 | `skoll-api-contracts` | 实现 menu handler | handler code + tests | 不直连 store；鉴权失败路径覆盖 | `go test ./internal/handler/http/v1/menu/...` | Done |
| 56 | M1-06-05 | M1-06 | `skoll-api-contracts` | 注册路由与权限 keys | router/OpenAPI/permission seed | API 路由可访问，权限 key 纳入 catalog | `go test ./internal/handler/http/...` | Done |
| 57 | M1-07-01 | M1-07 | `skoll-plugin-platform` | 扩展 plugin manifest permission 字段读取 | plugin parser | 字段校验严格，无旧格式旁路 | `go test ./internal/plugin/...` | Done |
| 58 | M1-07-02 | M1-07 | `skoll-plugin-platform` | 扩展 plugin manifest menu 字段读取 | plugin parser | ui_menu 进入统一 MenuNode 映射 | `go test ./internal/plugin/...` | Done |
| 59 | M1-07-03 | M1-07 | `skoll-plugin-platform` | 插件 enable 导入 catalog/registry | plugin manager | enable 后权限和菜单可查询 | `go test ./internal/plugin/...` | Done |
| 60 | M1-07-04 | M1-07 | `skoll-plugin-platform` | 插件 disable 隐藏访问 | plugin manager | disable 后菜单过滤且权限 inactive | `go test ./internal/plugin/...` | Done |
| 61 | M1-07-05 | M1-07 | `skoll-plugin-platform` | 插件权限/菜单导入审计 | plugin/service tests | enable/disable 产生审计事件 | `go test ./internal/plugin/... ./internal/service/audit/...` | Done |
| 62 | M1-08-01 | M1-08 | `skoll-vue-frontend` | 新增 permission API client | web client | 类型定义完整，错误处理统一 | `cd web && npm run build` | Done |
| 63 | M1-08-02 | M1-08 | `skoll-vue-frontend` | 新增 permission store | Pinia store | 支持加载、失败、重试、缓存刷新 | `cd web && npm run build` | Done |
| 64 | M1-08-03 | M1-08 | `skoll-vue-frontend` | 新增 menu API client | web client | tree/save/reorder 接口封装 | `cd web && npm run build` | Done |
| 65 | M1-08-04 | M1-08 | `skoll-vue-frontend` | 新增 menu store | Pinia store | 侧栏优先使用 registry tree | `cd web && npm run build` | Done |
| 66 | M1-08-05 | M1-08 | `skoll-vue-frontend` | 统一 route guard 权限入口 | router/permissions | 静态、动态、插件页面同源判断 | `cd web && npm run build` | Done |
| 67 | M1-08-06 | M1-08 | `skoll-vue-frontend` | 统一按钮权限入口 | directive/utils | 按钮权限不再页面散落判断 | `cd web && npm run build` | Done |
| 68 | M1-09-01 | M1-09 | `skoll-web-ui-design` | 权限矩阵页读取 catalog | Vue page | 不再使用孤立权限常量 | `cd web && npm run build` | Done |
| 69 | M1-09-02 | M1-09 | `skoll-web-ui-design` | 权限矩阵授权/撤销接入 API | Vue page | 保存后前后端一致 | `cd web && npm run build` | Done |
| 70 | M1-09-03 | M1-09 | `skoll-web-ui-design` | 菜单管理页读取 registry tree | Vue page | 加载、空态、错误态完整 | `cd web && npm run build` | Done |
| 71 | M1-09-04 | M1-09 | `skoll-web-ui-design` | 菜单管理页编辑与保存 | Vue page | 显隐、排序、权限字段可保存 | `cd web && npm run build` | Done |
| 72 | M1-09-05 | M1-09 | `skoll-web-ui-design` | 菜单和权限页面确认弹窗 | Vue page | 高风险保存有确认和错误反馈 | `cd web && npm run build` | Done |
| 73 | M1-10-01 | M1-10 | `skoll-testing-automation` | 编写 M1 后端集成测试 | tests | 角色授权、菜单过滤、API 拒绝路径覆盖 | `go test ./...` | Done |
| 74 | M1-10-02 | M1-10 | `skoll-testing-automation` | 编写 M1 前端 smoke 步骤 | docs/scripts | 可手工或脚本验证侧栏/路由/按钮一致性 | `cd web && npm run build` | Done |
| 75 | M1-10-03 | M1-10 | `skoll-docs-writer` | 更新权限与菜单架构文档 | docs | 说明 catalog/registry、权限命名、菜单来源 | 文档审阅 | Done |
| 76 | M1-10-04 | M1-10 | `skoll-refactor-governance` | M1 里程碑验收记录 | `acceptance_log.md` | 记录命令结果、人工验收、提交 hash | 文档审阅 | Done |

## M2 Work Items: 审计、登录日志、错误日志统一化

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 77 | M2-01-01 | M2-01 | `skoll-audit-observability-refactor` | 定义 AuditEventType | domain code | operation/login/error/plugin/security 枚举校验 | `go test ./internal/domain/audit/...` | Done |
| 78 | M2-01-02 | M2-01 | `skoll-audit-observability-refactor` | 定义 AuditAction 命名规则 | domain code + docs | `module.resource.action` 校验通过 | `go test ./internal/domain/audit/...` | Done |
| 79 | M2-01-03 | M2-01 | `skoll-audit-observability-refactor` | 定义 AuditEvent 统一结构 | domain code | actor/resource/result/trace/risk/metadata 字段完整 | `go test ./internal/domain/audit/...` | Done |
| 80 | M2-01-04 | M2-01 | `skoll-audit-observability-refactor` | 补审计 action catalog 文档 | docs/package README | 覆盖 user/role/rbac/plugin/menu/file/system | 文档审阅 | Done |
| 81 | M2-02-01 | M2-02 | `skoll-audit-observability-refactor` | 定义 LoginLog 模型 | domain/store model | 账号、结果、IP、UA、失败原因、session id 完整 | `go test ./internal/domain/audit/...` | Done |
| 82 | M2-02-02 | M2-02 | `skoll-audit-observability-refactor` | 定义 ErrorLog 模型 | domain/store model | trace、request、错误码、等级、摘要完整 | `go test ./internal/domain/audit/...` | Done |
| 83 | M2-02-03 | M2-02 | `skoll-database-development` | 新增 audit/log migration | MySQL/PostgreSQL SQL | operation/login/error 表或统一表设计明确 | migration review | Done |
| 84 | M2-02-04 | M2-02 | `skoll-database-development` | 实现 memory audit log store | memory store + tests | append/query/detail/export source data 通过 | `go test ./internal/store/memory/...` | Done |
| 85 | M2-02-05 | M2-02 | `skoll-database-development` | 实现 SQL audit log store | gormrepo store + tests | 按类型、actor、action、时间过滤通过 | `go test ./internal/store/sql/gormrepo/...` | Done |
| 86 | M2-02-06 | M2-02 | `skoll-audit-observability-refactor` | 接入 audit service 查询模型 | service code + tests | service 层不暴露 store 细节 | `go test ./internal/service/audit/...` | Done |
| 87 | M2-03-01 | M2-03 | `skoll-audit-observability-refactor` | 梳理现有 middleware 写审计路径 | 代码清单 | 找出旧分散写入点并列入替换清单 | `rg "audit|operation|login" internal/handler internal/service` | Done |
| 88 | M2-03-02 | M2-03 | `skoll-audit-observability-refactor` | 实现统一请求审计 middleware | middleware code + tests | 成功/失败/未授权均记录 | `go test ./internal/handler/middleware/...` | Done |
| 89 | M2-03-03 | M2-03 | `skoll-audit-observability-refactor` | 接入登录成功/失败审计 | auth handler/service | login 和 login_failed 可查询 | `go test ./internal/bootstrap/... ./internal/handler/http/...` | Done |
| 90 | M2-03-04 | M2-03 | `skoll-audit-observability-refactor` | 接入权限拒绝审计 | middleware/service | forbidden 事件记录 actor/resource/action | `go test ./internal/handler/middleware/...` | Done |
| 91 | M2-03-05 | M2-03 | `skoll-audit-observability-refactor` | 接入错误日志捕获 | middleware/logging | panic/handler error 记录 trace | `go test ./internal/handler/middleware/...` | Done |
| 92 | M2-04-01 | M2-04 | `skoll-api-contracts` | 设计审计列表 API 契约 | OpenAPI | filter/page/response 字段稳定 | OpenAPI review | Done |
| 93 | M2-04-02 | M2-04 | `skoll-api-contracts` | 实现审计列表 API | handler + tests | 分类、actor、resource、action、时间、风险筛选 | `go test ./internal/handler/http/v1/audit/...` | Done |
| 94 | M2-04-03 | M2-04 | `skoll-api-contracts` | 设计审计详情 API 契约 | OpenAPI | metadata/diff/trace 字段明确 | OpenAPI review | Done |
| 95 | M2-04-04 | M2-04 | `skoll-api-contracts` | 实现审计详情 API | handler + tests | 不存在返回 not_found，权限失败返回 forbidden | `go test ./internal/handler/http/v1/audit/...` | Done |
| 96 | M2-04-05 | M2-04 | `skoll-api-contracts` | 实现审计导出 API | handler + tests | CSV 字段稳定，过滤条件与列表一致 | `go test ./internal/handler/http/v1/audit/...` | Done |
| 97 | M2-04-06 | M2-04 | `skoll-api-contracts` | 更新 OpenAPI 文件 | `docs/api/openapi.yaml` | 审计 list/detail/export 契约同步 | OpenAPI lint/review | Done |
| 98 | M2-05-01 | M2-05 | `skoll-vue-frontend` | 新增审计 API client 类型 | web client | list/detail/export 类型完整 | `cd web && npm run build` | Done |
| 99 | M2-05-02 | M2-05 | `skoll-vue-frontend` | 审计页 tab 分类 | Vue page | operation/login/error/plugin/security tab 可切换 | `cd web && npm run build` | Done |
| 100 | M2-05-03 | M2-05 | `skoll-vue-frontend` | 审计页筛选区升级 | Vue page | actor/resource/action/time/risk 筛选可用 | `cd web && npm run build` | Done |
| 101 | M2-05-04 | M2-05 | `skoll-vue-frontend` | 审计详情抽屉升级 | Vue page | metadata、diff、trace 展示清楚 | `cd web && npm run build` | Done |
| 102 | M2-05-05 | M2-05 | `skoll-vue-frontend` | 审计导出按钮接入 | Vue page | 导出带当前过滤条件，错误可见 | `cd web && npm run build` | Done |
| 103 | M2-05-06 | M2-05 | `skoll-web-ui-design` | 审计页空态/错误态/无权限态 | Vue page | 专业后台交互状态完整 | `cd web && npm run build` | Done |
| 104 | ADJ-20260619-01 | M2-04 | `skoll-api-contracts` | M2 export/list filter parity check | 一致性记录 | CSV 导出使用与 list API 相同的分类、actor、resource、action、时间、风险筛选 | `rg "parseEventFilter" internal/handler/http/v1/audit; rg "ExportEventSourceData" internal/handler/http/v1/audit; rg "ListEvents" internal/handler/http/v1/audit` | Done |
| 105 | ADJ-20260619-02 | M2-04 | `skoll-api-contracts` / `skoll-vue-frontend` | M2 OpenAPI/client sync gate | 对齐记录 | list/detail/export API、OpenAPI、frontend client 字段一致 | `rg "AuditEventListQuery|AuditEventDetail|eventId,sourceData" web/src/audit/api.ts docs/api/openapi.yaml internal/handler/http/openapi.yaml` | Done |
| 106 | ADJ-20260619-03 | M2-05 | `skoll-frontend-design-refactor` / `skoll-frontend-testing-refactor` | FE0 audit page UX baseline | 审计页体验验收模板 | tab、筛选、详情、导出、空态、错误态、无权限态均有验收标准 | 文档审阅 | Done |
| 107 | ADJ-20260619-04 | M2-06 | `skoll-testing-automation` | M2 audit smoke fixture | smoke fixture | login_failed、forbidden、plugin、menu、export 均有固定样本 | 文档审阅 | Done |
| 108 | M2-06-01 | M2-06 | `skoll-testing-automation` | 更新 smoke-auth-audit 脚本场景 | script | 覆盖 login_failed、forbidden、plugin、menu、export | `powershell -ExecutionPolicy Bypass -File ./scripts/smoke-auth-audit.ps1` | Done |
| 109 | M2-06-02 | M2-06 | `skoll-testing-automation` | 编写 M2 手工验收步骤 | docs | UI 查询、详情、导出、日志检查步骤完整 | 文档审阅 | Done |
| 110 | M2-06-03 | M2-06 | `skoll-testing-automation` | M2 全量回归 | test run record | go test 和前端 build 通过或失败项记录 | `go test ./...`; `cd web && npm run build` | Done |
| 111 | M2-06-04 | M2-06 | `skoll-refactor-governance` | M2 里程碑验收记录 | `acceptance_log.md` | 记录命令结果、人工验收、提交 hash | 文档审阅 | Done |

## FE0 Work Items: 前端体验基线

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 112 | FE0-01 | FE0 | `skoll-frontend-design-refactor` | 盘点当前前端页面体验 | 页面体验清单 | Dashboard/User/Role/Permission/Menu/Plugin/Audit/Setting 状态、问题、优先级清楚 | 文档审阅 | Done |
| 113 | FE0-02 | FE0 | `skoll-frontend-coding-refactor` | 盘点前端架构与复用点 | 架构清单 | API client、stores、router、permissions、SchemaForm、Common 组件边界清楚 | `rg "defineStore\|SchemaForm\|canAccess\|v-permission" web/src` | Todo |
| 114 | FE0-03 | FE0 | `skoll-frontend-performance-refactor` | 建立前端构建基线 | 构建记录 | build 输出、chunk、warning 和明显依赖风险被记录 | `cd web && npm run build` | Todo |
| 115 | FE0-04 | FE0 | `skoll-frontend-testing-refactor` | 建立前端 typecheck 基线 | 检查记录 | typecheck 存在则记录结果，不存在则列入任务 | `cd web && npm run typecheck` | Todo |
| 116 | FE0-05 | FE0 | `skoll-frontend-testing-refactor` | 建立浏览器人工验收模板 | 前端验收模板 | 包含正常、加载、空态、错误、无权限、窄屏、危险操作 | 文档审阅 | Todo |

## FE1 Work Items: 设计系统与视觉规范

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 117 | FE1-01 | FE1 | `skoll-frontend-design-refactor` | 定义 Skoll Admin 视觉原则 | `frontend_experience_plan.md` 或 UI 文档 | 明确专业、克制、可扫、插件优先、不做营销页风格 | 文档审阅 | Todo |
| 118 | FE1-02 | FE1 | `skoll-frontend-design-refactor` | 定义页面布局标准 | UI 文档/样例 | 页面标题、工具栏、筛选、表格、详情、抽屉结构统一 | 文档审阅 | Todo |
| 119 | FE1-03 | FE1 | `skoll-frontend-design-refactor` | 定义表格体验标准 | UI 文档/组件要求 | 列稳定、状态标签、批量动作、紧凑行操作标准明确 | 文档审阅 | Todo |
| 120 | FE1-04 | FE1 | `skoll-frontend-design-refactor` | 定义表单和弹窗标准 | UI 文档/组件要求 | 校验、保存中、错误、确认、抽屉/弹窗适用边界明确 | 文档审阅 | Todo |
| 121 | FE1-05 | FE1 | `skoll-frontend-design-refactor` | 定义状态组件标准 | UI 文档/组件要求 | loading/empty/error/no-permission/success/failed 统一 | 文档审阅 | Todo |
| 122 | FE1-06 | FE1 | `skoll-frontend-coding-refactor` | 抽查全局样式与变量 | 样式清单 | 颜色、间距、字号、状态色来源清楚，无一页一套 | `rg "#[0-9A-Fa-f]{3,6}\|var\\(" web/src/styles web/src/views` | Todo |
| 123 | FE1-07 | FE1 | `skoll-frontend-design-refactor` | 定义响应式最低标准 | UI 文档 | 窄屏无重叠、无按钮溢出、表格有降级策略 | 文档审阅 | Todo |

## FE2 Work Items: 前端架构与数据流

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 124 | FE2-01 | FE2 | `skoll-frontend-coding-refactor` | 统一 API client 规范 | API client 文档/代码规范 | 请求、响应、错误、分页、导出封装标准明确 | 文档审阅 | Todo |
| 125 | FE2-02 | FE2 | `skoll-frontend-coding-refactor` | 统一 Pinia store 规范 | store 文档/代码规范 | loading/error/data/retry/refresh 模式明确 | 文档审阅 | Todo |
| 126 | FE2-03 | FE2 | `skoll-frontend-coding-refactor` | 统一 route guard 规范 | router/permission 文档 | 菜单、路由、按钮、插件权限同源 | `rg "canAccess\|beforeEach\|v-permission" web/src` | Todo |
| 127 | FE2-04 | FE2 | `skoll-frontend-coding-refactor` | 统一 SchemaForm 使用规范 | SchemaForm 文档/示例 | 插件配置、系统设置、生成器表单复用边界明确 | 文档审阅 | Todo |
| 128 | FE2-05 | FE2 | `skoll-frontend-coding-refactor` | 统一确认动作 helper | helper 规范/代码 | 删除、禁用、回滚、发布等危险动作使用统一确认 | `rg "useConfirmAction\|ElMessageBox" web/src` | Todo |
| 129 | FE2-06 | FE2 | `skoll-frontend-coding-refactor` | 统一前端错误展示 | helper 规范/代码 | 后端错误不会被吞掉或显示假成功 | `rg "catch\|ElMessage.error\|throw" web/src` | Todo |
| 130 | FE2-07 | FE2 | `skoll-frontend-coding-refactor` | 统一导出/下载交互 | helper 规范/代码 | 导出使用当前筛选条件，失败可见 | 文档审阅 | Todo |

## FE3 Work Items: 核心页面体验升级

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 131 | FE3-01 | FE3 | `skoll-frontend-design-refactor` | Dashboard 体验升级方案 | 页面方案/改动 | 信息密度、系统状态、插件状态、快捷入口合理 | `cd web && npm run build` | Todo |
| 132 | FE3-02 | FE3 | `skoll-frontend-design-refactor` | User 页面体验升级 | 页面改动 | 筛选、批量、编辑、角色分配、组织归属状态完整 | `cd web && npm run build` | Todo |
| 133 | FE3-03 | FE3 | `skoll-frontend-design-refactor` | Role 页面体验升级 | 页面改动 | 角色列表、授权入口、关联用户、危险操作清晰 | `cd web && npm run build` | Todo |
| 134 | FE3-04 | FE3 | `skoll-frontend-design-refactor` | Permission 页面体验升级 | 页面改动 | 权限矩阵可扫、可筛选、保存反馈明确 | `cd web && npm run build` | Todo |
| 135 | FE3-05 | FE3 | `skoll-frontend-design-refactor` | Menu 页面体验升级 | 页面改动 | 树表、排序、显隐、权限字段、保存确认完整 | `cd web && npm run build` | Todo |
| 136 | FE3-06 | FE3 | `skoll-frontend-design-refactor` | Plugin 页面体验升级 | 页面改动 | 状态、风险、权限、配置、访问、生命周期动作清晰 | `cd web && npm run build` | Todo |
| 137 | FE3-07 | FE3 | `skoll-frontend-design-refactor` | Audit 页面体验升级 | 页面改动 | tab、筛选、详情、导出、风险标签、trace 展示完整 | `cd web && npm run build` | Todo |
| 138 | FE3-08 | FE3 | `skoll-frontend-design-refactor` | Setting 页面体验升级 | 页面改动 | SchemaForm、分组、敏感项、保存反馈、审计提示清晰 | `cd web && npm run build` | Todo |
| 139 | FE3-09 | FE3 | `skoll-frontend-testing-refactor` | 核心页面状态验收 | 验收记录 | 每页正常/空态/错误/无权限/窄屏至少抽查 | 浏览器验收 | Todo |

## FE4 Work Items: 插件与开发者门户体验

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 140 | FE4-01 | FE4 | `skoll-frontend-design-refactor` | 插件列表信息架构升级 | Plugin 页面 | enabled、risk、signature、version、health、actions 可扫 | `cd web && npm run build` | Todo |
| 141 | FE4-02 | FE4 | `skoll-frontend-design-refactor` | 插件详情抽屉/详情页设计 | Plugin 页面 | 权限、菜单、配置、资产、日志、发布状态分区清楚 | `cd web && npm run build` | Todo |
| 142 | FE4-03 | FE4 | `skoll-frontend-design-refactor` | 插件安装预检体验 | Plugin/Marketplace UI | 权限 diff、菜单 diff、风险、签名、迁移影响清楚 | `cd web && npm run build` | Todo |
| 143 | FE4-04 | FE4 | `skoll-frontend-design-refactor` | 插件风险报告体验 | Plugin UI | 风险等级、因子、阻断原因、审计记录可读 | `cd web && npm run build` | Todo |
| 144 | FE4-05 | FE4 | `skoll-frontend-design-refactor` | Dev Portal 发布任务体验 | Developer Portal | 任务状态、步骤、日志、失败原因、重试/回滚清楚 | `cd web && npm run build` | Todo |
| 145 | FE4-06 | FE4 | `skoll-frontend-performance-refactor` | 插件重面板按需加载 | Plugin/Dev Portal | 日志、风险、发布历史不阻塞首屏 | `cd web && npm run build` | Todo |

## FE5 Work Items: 前端测试与验收体系

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 146 | FE5-01 | FE5 | `skoll-frontend-testing-refactor` | 固化 typecheck 门禁 | package script/CI 文档 | typecheck 可运行或缺口明确 | `cd web && npm run typecheck` | Todo |
| 147 | FE5-02 | FE5 | `skoll-frontend-testing-refactor` | 固化 build 门禁 | package script/CI 文档 | build 是所有前端任务必跑项 | `cd web && npm run build` | Todo |
| 148 | FE5-03 | FE5 | `skoll-frontend-testing-refactor` | 建立核心页面 smoke 清单 | docs/scripts | Dashboard/User/Role/Permission/Menu/Plugin/Audit/Setting 有步骤 | 文档审阅 | Todo |
| 149 | FE5-04 | FE5 | `skoll-frontend-testing-refactor` | 建立权限态验收清单 | docs/scripts | admin 和 restricted role 验收步骤明确 | 文档审阅 | Todo |
| 150 | FE5-05 | FE5 | `skoll-frontend-testing-refactor` | 建立响应式验收清单 | docs/scripts | 至少桌面和窄屏检查项明确 | 文档审阅 | Todo |
| 151 | FE5-06 | FE5 | `skoll-frontend-testing-refactor` | 评估 Playwright/browser 自动化 | 测试方案 | 是否引入、覆盖范围、执行命令明确 | 文档审阅 | Todo |
| 152 | FE5-07 | FE5 | `skoll-frontend-testing-refactor` | 前端验收记录接入 acceptance log | `acceptance_log.md` | 前端状态、浏览器、截图/路径字段可记录 | 文档审阅 | Todo |

## FE6 Work Items: 前端性能与可观测

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 153 | FE6-01 | FE6 | `skoll-frontend-performance-refactor` | 建立 bundle 基线 | 性能记录 | build chunk 和 warning 被记录 | `cd web && npm run build` | Todo |
| 154 | FE6-02 | FE6 | `skoll-frontend-performance-refactor` | 检查路由懒加载覆盖 | router 清单 | 主要业务页面均懒加载 | `rg "component:\\s*\\(\\)\\s*=>" web/src/router web/src` | Todo |
| 155 | FE6-03 | FE6 | `skoll-frontend-performance-refactor` | 建立表格性能规则 | 性能文档/组件要求 | 服务端分页、大列表策略、稳定尺寸明确 | 文档审阅 | Todo |
| 156 | FE6-04 | FE6 | `skoll-frontend-performance-refactor` | 共享数据缓存策略 | store 文档/代码 | 菜单、权限、字典缓存与失效规则明确 | 文档审阅 | Todo |
| 157 | FE6-05 | FE6 | `skoll-frontend-performance-refactor` | 插件/Dev Portal 性能检查 | 性能记录 | 重面板按需加载，请求数量可控 | 浏览器验收 | Todo |
| 158 | FE6-06 | FE6 | `skoll-frontend-performance-refactor` | 前端性能验收模板 | `acceptance_log.md` | 记录 bundle 变化、路由加载、请求数量、页面卡顿观察 | 文档审阅 | Todo |
