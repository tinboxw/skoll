# Skoll 重构唯一架构决策与执行计划

> 日期: 2026-06-18  
> 当前状态: 生效中的唯一架构决策  
> 适用范围: Skoll 开源基础建设重构全程  
> 角色视角: 项目负责人 / 架构负责人  
> 目标: 在覆盖 gin-vue-admin 主干能力的基础上，把 Skoll 重构为可扩展、可测试、可插件化、可长期维护的开源 Admin 框架。

本文档是当前重构阶段唯一生效的架构决策和执行计划。旧计划文档只作为历史归档材料，不作为任务来源、架构依据或执行入口。

## 1. 先定框架模型

### 1.1 结论

Skoll 不建议采用“纯 DDD 项目”，也不建议回到传统 MVC/三层 CRUD。

推荐模型:

```text
模块化单体 Modular Monolith
+ Clean Architecture / Hexagonal Architecture 边界
+ 战术 DDD Tactical DDD
+ 插件化 Extension Platform
+ 契约优先 Contract-first
```

一句话: Skoll 的核心是“模块化单体 + 清晰依赖边界”，在复杂核心域使用 DDD 战术建模，在普通 CRUD 和工具型模块保持简单。

### 1.2 为什么不是纯 DDD

纯 DDD 对 Skoll 不合适，原因是:

1. Skoll 是开源 Admin 框架，不是单一业务系统。框架能力包含认证、菜单、权限、插件、文件、代码生成、配置、审计、部署，这些更像平台能力，不全是复杂业务域。
2. 纯 DDD 容易让简单 CRUD 过度建模，增加贡献者理解成本。
3. Skoll 需要强插件化和生成器，代码结构必须稳定、可约定、可生成。过度自由的 DDD 聚合会让生成器难以输出一致代码。
4. 开源框架第一价值是可用、可扩展、可维护。不是每个模块都需要 Entity、Value Object、Domain Service、Repository 全套仪式。

### 1.3 为什么不是传统 MVC / Service-DAO

传统 MVC / Service-DAO 也不够，原因是:

1. gin-vue-admin 的短板之一就是全局单例和源码注入式耦合，继续走轻边界模式会重复同类问题。
2. 权限、菜单、插件、生成器、审计、数据范围之间存在跨模块规则，不能只靠 handler/service 拼接。
3. 插件生命周期、签名、风险等级、迁移、灰度和回滚需要清晰端口和策略模型。
4. 后续要支持多数据库、多缓存、多对象存储、多部署形态，必须有 port/adapter 边界。

### 1.4 推荐分层

```text
cmd/skoll
  进程入口，只做启动。

internal/bootstrap
  配置、依赖装配、路由装配、生命周期管理。

internal/domain/<module>
  领域模型和业务规则。只放真正有规则的模块。
  例如 user、rbac、plugin、menu、permission、file、generator、audit。

internal/service/<module>
  应用服务。编排事务、权限检查、领域对象、仓储、事件、审计。

internal/repository/<module>
  面向 service 的仓储接口和组合能力。保持业务语义。

internal/store/<adapter>
  具体存储实现。MySQL/PostgreSQL/Memory/ClickHouse/Object/S3 等。

internal/handler/http/v1/<module>
  HTTP 适配器。只处理请求解析、响应、状态码和 OpenAPI 对齐。

web/src
  前端控制台。按视图、API client、store、permission、plugin extension 分层。

plugins/
  外部扩展。必须通过 manifest、权限、菜单、配置 schema、签名和生命周期接入。
```

### 1.5 领域复杂度分级

| 类型 | 使用模型 | 适用模块 | 规则 |
|------|----------|----------|------|
| 核心复杂域 | 战术 DDD | RBAC、Plugin、Generator、Menu/Permission、File、Audit | 需要 Entity/Value Object/Domain Service/Repository/Event |
| 平台支撑域 | Clean Service | Config、Dictionary、Organization、Workflow | 保留 domain，但模型简化 |
| 普通 CRUD | CRUD Module | 示例业务、生成模块 | 使用统一模板，不强行 DDD |
| 技术适配 | Port/Adapter | Cache、Store、Object Storage、Metrics、Logging | 不放业务规则，只实现接口 |

### 1.6 依赖规则

1. `domain` 不依赖 `service`、`handler`、`store`、`web`。
2. `service` 可以依赖 `domain`、`repository`、`event`、`pkg`，不能依赖 HTTP 框架细节。
3. `handler` 不能直接访问 `store` 或 GORM model。
4. `store` 不反向依赖 `service`。
5. 插件不能绕过 manifest 直接写入系统菜单、权限和路由。
6. 生成器输出必须遵守同样边界。

### 1.7 分层边界验收口径

后续每个涉及代码的 Work Item 都必须按以下边界自检:

| 边界 | 允许职责 | 禁止事项 |
|------|----------|----------|
| `domain` | 模型、值对象、领域规则、领域级校验 | 依赖 Gin、GORM、HTTP 请求、Vue、具体数据库适配 |
| `service` | 用例编排、事务边界、权限检查、审计事件、仓储调用 | 解析 HTTP、直接操作 GORM model、返回前端专用结构 |
| `repository` | 面向 service 的仓储接口、查询语义、组合能力 | 泄漏具体 SQL/GORM 细节给 service |
| `store` | Memory、SQL、Object、Cache 等具体 adapter 实现 | 反向调用 service 或承载业务编排 |
| `handler` | 请求解析、响应格式、状态码、OpenAPI 对齐 | 直接访问 store、绕过 service 实现业务规则 |
| `web` | Vue 页面、API client、Pinia store、权限和交互状态 | 猜测后端契约、维护孤立权限来源 |
| `plugin` | manifest、权限、菜单、配置、资产、生命周期 extension point | 绕过平台注册中心直接污染核心菜单、权限或路由 |

## 2. 项目治理原则

1. 每个任务必须有验收标准。验收失败时，任务退回“重新执行”，不得标记完成。
2. 每个里程碑必须有可运行成果。不能只交文档或半成品。
3. 所有新 API 必须同步 OpenAPI、权限目录、审计 action、前端 API client。
4. 所有涉及数据结构的任务必须有 migration、回滚说明和存储测试。
5. 所有插件能力必须经过 manifest schema、权限声明、风险评估和签名影响评估。
6. 所有生成能力必须支持 dry-run、diff、文件清单和失败回滚。
7. 前端页面不得只完成静态 UI，必须接入真实 API、权限状态、加载状态、空状态和错误状态。
8. 每个里程碑结束必须更新文档索引、功能状态和已知限制。
9. Skoll 当前处于开源基础建设阶段，不做兼容方案，不做旧接口、旧数据结构、旧插件格式、旧页面路径的兼容方案；需要破坏式调整时，直接更新当前契约、文档、示例和验收脚本。
10. `docs/refactor/task_board.md` 是父任务表；`docs/refactor/work_items.md` 是最小执行和提交单元。每完成一个 Work Item 并通过验收后，提交一次代码。
11. 前端是与后端同级的产品主线，不作为后端功能的附属验收项。所有用户可见能力必须同时满足体验、视觉、交互状态、权限状态、性能和可测试性要求。

## 2.1 重构专用 Skills

以下 skills 已写入 `C:\Users\tinbox.wu\.codex\skills`，后续执行重构任务时按任务类型触发:

| Skill | 适用任务 |
|------|----------|
| `skoll-refactor-governance` | M0、里程碑拆分、验收机制、架构决策、无兼容方案规则 |
| `skoll-permission-menu-refactor` | M1 权限资源目录、菜单注册中心、动态路由、按钮权限、权限矩阵 |
| `skoll-audit-observability-refactor` | M2 操作日志、登录日志、错误日志、审计 API、审计 UI |
| `skoll-file-storage-refactor` | M3 文件元数据、对象存储、上传下载、分片上传、插件资产 |
| `skoll-dictionary-organization-refactor` | M4 字典、配置 schema、组织部门、岗位、数据范围 |
| `skoll-generator-refactor` | M5 GeneratorSpec、CRUD 生成、dry-run、diff、生成历史 |
| `skoll-plugin-marketplace-refactor` | M6 插件市场、安装预检、签名、风险报告、灰度、回滚 |
| `skoll-frontend-design-refactor` | FE0-FE6 前端体验、视觉规范、交互状态、响应式与页面验收 |
| `skoll-frontend-coding-refactor` | FE2-FE4 Vue 3、TypeScript、API client、Pinia、Router、SchemaForm |
| `skoll-frontend-testing-refactor` | FE0/FE5 前端 typecheck、build、smoke、浏览器验收与验收记录 |
| `skoll-frontend-performance-refactor` | FE0/FE6 bundle、懒加载、表格性能、请求数量、首屏体验 |

## 3. 任务拆分

### M0: 架构章程与质量基线

| ID | 任务 | 粒度 | 验收要求 |
|----|------|------|----------|
| M0-01 | 确认架构模型 | 文档 | 本文档明确 Modular Monolith + Clean/Hexagonal + Tactical DDD；列出非目标；团队后续任务引用该决策 |
| M0-02 | 建立模块边界规则 | 文档 + 检查清单 | 输出依赖规则；新增/修改模块时 PR 必须说明 domain/service/store/handler 归属 |
| M0-03 | 当前质量基线 | 命令报告 | 记录 `go test ./...`、`go test ./... -coverprofile=coverage.out`、`cd web && npm run build` 结果；失败项有 owner 和重跑命令 |
| M0-04 | 当前功能基线 | 文档 | 对照 GVA 功能矩阵标注 Skoll 已有/部分/缺失；缺失项进入任务池 |
| M0-05 | 验收模板 | 文档 | 每项任务包含交付物、测试命令、人工验收、回滚要求、未通过处理 |

里程碑验收:

1. 架构决策文档被 `docs/README.md` 引用。
2. 当前测试、构建、覆盖率结果被记录。
3. 后续任务拆分均有验收标准。

### M1: 权限资源目录与菜单注册中心

| ID | 任务 | 粒度 | 验收要求 |
|----|------|------|----------|
| M1-01 | 定义 PermissionResource 模型 | 后端 domain | 支持 `api/menu/button/data_scope/plugin` 类型；包含 key、name、module、risk、source、enabled、metadata |
| M1-02 | 定义 MenuNode 模型 | 后端 domain | 支持 parent、path、component、icon、sort、visible、requiredRoles、requiredPermissions、source |
| M1-03 | 建立 permission repository 接口 | 后端 repository | Memory 和 SQL 均可实现；service 不直接依赖 GORM |
| M1-04 | 新增 SQL migration | 数据库 | MySQL/PostgreSQL migration 可重复执行；包含索引和唯一约束 |
| M1-05 | 实现 permission catalog service | 后端 service | 支持注册、列表、启停、按来源查询、权限 diff |
| M1-06 | 从路由/OpenAPI 同步 API 资源 | 后端 service | 至少能同步系统 API；重复同步不产生重复数据 |
| M1-07 | 插件 manifest 权限导入 | 插件 | 插件启用时导入权限，禁用时保留但标记 inactive |
| M1-08 | 菜单注册中心 service | 后端 service | 系统菜单、插件菜单、生成模块菜单统一合并、排序、权限过滤 |
| M1-09 | 菜单管理 API | 后端 handler | OpenAPI 更新；增删改查、排序、显隐、授权字段保存可用 |
| M1-10 | 前端权限目录 store | 前端 | 页面加载后能获取权限目录；失败有错误提示和重试 |
| M1-11 | 前端菜单管理页升级 | 前端 | Element Plus 树表编辑；支持排序、显隐、角色/权限字段 |
| M1-12 | 路由守卫统一权限判断 | 前端 | 静态页面、动态菜单、插件页面共用同一 `canAccess` 入口 |
| M1-13 | 权限矩阵页接入 catalog | 前端 | 不再维护孤立权限列表；授权/撤销后 UI 与后端一致 |
| M1-14 | 单元与集成测试 | 测试 | 覆盖注册、重复注册、插件导入、禁用、菜单过滤、无权限拒绝 |

里程碑验收:

1. 新建角色只授予部分菜单时，侧栏、路由、按钮和 API 权限表现一致。
2. 插件启用后菜单和权限进入 catalog，禁用后前端不可访问。
3. `go test ./...` 和前端 build 通过。
4. OpenAPI、权限目录文档和菜单管理页面均同步。

### M2: 审计、登录日志、错误日志统一化

| ID | 任务 | 粒度 | 验收要求 |
|----|------|------|----------|
| M2-01 | 定义 AuditEvent 分类 | domain | 支持 operation、login、error、plugin、security 类型 |
| M2-02 | 统一审计 action 命名 | domain 文档 | 命名格式为 `module.resource.action`，示例覆盖用户、角色、插件、菜单、文件 |
| M2-03 | 登录日志模型 | domain/store | 记录账号、结果、IP、UA、失败原因、会话 ID |
| M2-04 | 错误日志模型 | domain/store | 记录 trace ID、请求、堆栈摘要、错误码、等级 |
| M2-05 | 审计写入中间件 | handler middleware | 成功、失败、未授权均能按规则记录 |
| M2-06 | 审计查询 API | handler | 支持分类、actor、resource、action、时间、风险等级筛选 |
| M2-07 | 审计导出 | service/handler | CSV 导出字段稳定；大数据量有分页或流式限制 |
| M2-08 | 前端审计页升级 | 前端 | 三类日志 tab，筛选、详情、导出、风险标签 |
| M2-09 | 插件操作审计 | 插件/service | 安装、启用、禁用、发布、灰度、回滚均写审计 |
| M2-10 | 审计保留策略 | service | 支持按时间清理；清理自身也写审计 |

里程碑验收:

1. 登录失败、权限拒绝、插件启用、菜单修改、文件上传均能查到审计记录。
2. 审计列表、详情、导出均可用。
3. 至少覆盖 Memory 和 SQL store 测试。

### M3: 文件与对象存储平台

| ID | 任务 | 粒度 | 验收要求 |
|----|------|------|----------|
| M3-01 | 定义 FileObject 模型 | domain | 包含 key、name、size、mime、hash、owner、visibility、storage、status |
| M3-02 | 定义 ObjectStore port | store port | 支持 Put、Get、Delete、Presign、MultipartInit、MultipartUpload、MultipartComplete |
| M3-03 | local adapter | store adapter | 本地上传、下载、删除、路径穿越防护测试通过 |
| M3-04 | S3 协议对象存储 adapter | store adapter | MinIO/AWS/R2 可共用配置；无真实服务时有接口测试和 mock |
| M3-05 | 文件元数据 store | SQL/memory | 元数据与对象写入失败可回滚或标记失败 |
| M3-06 | 上传 API | handler | 单文件上传、下载、删除、列表；权限校验和 OpenAPI 完整 |
| M3-07 | 分片上传 API | handler/service | 支持 init/upload/complete/abort；hash 校验失败不可完成 |
| M3-08 | 文件权限策略 | service/rbac | private/public/plugin-asset 三类策略；无权不可下载 |
| M3-09 | 前端文件管理页 | 前端 | 上传、搜索、预览、下载、删除、错误提示 |
| M3-10 | 插件资产接入 | plugin | 插件可声明资产，安装后进入 file metadata 或资产目录 |
| M3-11 | 安全测试 | 测试 | 覆盖路径穿越、非法 mime、超大文件、未授权下载 |

里程碑验收:

1. 本地存储可完整上传、下载、删除。
2. 分片上传可完成和中止。
3. 文件操作均有权限和审计。
4. 前端文件页可真实操作。

### M4: 字典、配置、组织与数据范围

| ID | 任务 | 粒度 | 验收要求 |
|----|------|------|----------|
| M4-01 | 字典独立模型 | domain/store | DictionaryType 和 DictionaryItem 独立表，不再只依赖系统设置 |
| M4-02 | 字典缓存策略 | cache/service | 更新字典后缓存失效；读取命中有测试 |
| M4-03 | 字典 API/UI | handler/frontend | 类型、条目、状态、排序、搜索完整 |
| M4-04 | 配置 schema 注册 | domain/service | 系统与插件配置均通过 schema 描述字段、校验、默认值 |
| M4-05 | 配置变更审计 | service | 每次配置修改记录 before/after 摘要 |
| M4-06 | 组织/部门模型 | domain/store | 部门树、岗位、用户归属；迁移和测试完整 |
| M4-07 | 数据范围解析 | rbac/service | 支持 all、department、department_tree、self、custom |
| M4-08 | 用户列表数据过滤 | service/store | 按 data scope 过滤用户列表，super_admin 不受限 |
| M4-09 | 前端组织管理 | frontend | 部门树、岗位、用户归属编辑可用 |
| M4-10 | 权限矩阵展示 data scope | frontend | 授权时可选择数据范围，查询结果可解释 |

里程碑验收:

1. 字典不依赖硬编码，表单选项可从字典读取。
2. 用户列表按角色数据范围返回不同结果。
3. 组织和岗位能被用户管理页面使用。

### M5: 代码生成器 v1

| ID | 任务 | 粒度 | 验收要求 |
|----|------|------|----------|
| M5-01 | GeneratorSpec 模型 | domain | 描述模块、表、字段、索引、校验、菜单、权限、前端页面 |
| M5-02 | 模板规范 | 文档/模板 | 明确输出路径、命名、可重复生成策略、禁止覆盖用户改动策略 |
| M5-03 | dry-run 与 diff | service | 生成前展示文件清单、变更摘要和冲突 |
| M5-04 | 后端 CRUD 生成 | generator | 输出 domain/service/store/handler/migration/OpenAPI |
| M5-05 | 前端 CRUD 生成 | generator | 输出 Vue 列表、表单、API client、store、菜单注册 |
| M5-06 | 权限与审计生成 | generator | 自动生成权限 key、菜单、按钮权限、审计 action |
| M5-07 | 生成历史 | store/service | 记录批次、文件 hash、操作者、时间、spec |
| M5-08 | 回滚机制 | service | 未修改文件可自动回滚；冲突文件提示人工处理 |
| M5-09 | Dev Portal 页面 | frontend/plugin | 可填写模型、预览 diff、执行生成、查看历史 |
| M5-10 | 示例模块生成 | example | 生成 `demo_product` 或同等级示例，并通过测试 |
| M5-11 | 生成器测试 | 测试 | Golden file、幂等生成、冲突检测、回滚测试 |

里程碑验收:

1. 单表 CRUD 从 spec 到前后端页面可完整跑通。
2. 生成后的代码通过 `go test ./...` 和前端 build。
3. 生成历史可查看，dry-run 可预览，失败可回滚或给出明确人工处理清单。

### M6: 插件市场与插件生产生命周期

| ID | 任务 | 粒度 | 验收要求 |
|----|------|------|----------|
| M6-01 | Marketplace index schema | docs/schema | 定义插件 id、version、skoll_version、risk、signature、source、changelog |
| M6-02 | 本地市场页面 | frontend | 展示本地插件、可安装包、签名和风险等级 |
| M6-03 | 远端索引适配 | service | 支持配置远端 index URL；失败不影响本地插件 |
| M6-04 | 插件安装预检 | service | 展示权限 diff、菜单 diff、配置 schema、迁移、风险 |
| M6-05 | 插件迁移 hook | plugin | install/upgrade/downgrade/uninstall 迁移状态可追踪 |
| M6-06 | 插件签名策略升级 | security/plugin | 签名覆盖 manifest、资产清单、后端入口和前端资产 hash |
| M6-07 | 插件灰度产品化 | service/frontend | 灰度策略影响真实路由/菜单/功能可见性 |
| M6-08 | 插件回滚产品化 | service/frontend | 回滚后版本、菜单、权限、配置和资产一致 |
| M6-09 | 插件安全报告 | frontend/docs | 安装前和安装后均可查看安全报告 |
| M6-10 | 插件生命周期测试 | tests | 示例插件覆盖 install/enable/disable/upgrade/rollback |

里程碑验收:

1. 插件安装前可看到风险、权限和迁移影响。
2. 插件升级失败可回滚到上一可用版本。
3. 插件启停、灰度、回滚均有审计。

### M7: 发布质量、文档与开源体验

| ID | 任务 | 粒度 | 验收要求 |
|----|------|------|----------|
| M7-01 | CI 质量门禁 | CI | gofmt、go test、前端 build、OpenAPI 校验、manifest 校验进入 CI |
| M7-02 | 覆盖率门槛 | CI/docs | 核心包覆盖率目标和例外名单明确；低于门槛失败或阻断里程碑 |
| M7-03 | 性能基线 | tests/docs | 登录、用户列表、角色列表、插件列表、审计查询有 QPS/P95/P99 报告 |
| M7-04 | 公开契约策略 | docs | API、插件 manifest、配置 schema、migration 的当前版本和破坏式发布规则明确 |
| M7-05 | 快速开始更新 | docs | 新用户能按文档启动、登录、安装插件、生成模块 |
| M7-06 | 迁移指南 | docs | 从 gin-vue-admin/go-admin 迁移的概念映射和注意事项 |
| M7-07 | 示例项目 | examples/plugins | 最小后台、CRUD 插件、文件上传示例可运行 |
| M7-08 | 发布检查清单 | docs | release 前必须检查安全、测试、文档、迁移、回滚 |

里程碑验收:

1. 新环境按文档能在 30 分钟内启动并登录。
2. 质量门禁稳定运行。
3. 至少一个生成模块和一个插件示例可作为框架能力证明。

## 4. 验收与返工机制

每个任务执行完成后必须进入验收。

任务表格是必需品。没有任务表格，就无法保证顺序执行、验收失败返工和“一任务一提交”的节奏。

验收状态:

| 状态 | 含义 | 下一步 |
|------|------|--------|
| Passed | 全部验收项通过 | 进入下一任务 |
| Failed | 任一必选验收项失败 | 回到执行状态，修复后重新验收 |
| Blocked | 外部依赖阻塞 | 记录阻塞原因、临时方案和解除条件 |
| Deferred | 决策后延期 | 必须说明为什么不影响当前里程碑 |

任务验收记录必须包含:

1. 改动文件清单。
2. 自动化命令和结果。
3. 人工验证步骤。
4. OpenAPI、文档、迁移、权限、审计是否同步。
5. 已知限制。
6. 回滚方式。

## 5. 大项目重构还必须提前提出的细节

### 5.1 范围控制

1. 明确“复制 GVA 功能结果，不复制 GVA 实现方式”。
2. 每个里程碑只允许少量主线目标，避免权限、文件、生成器、插件市场同时半成品。
3. 新能力默认先做最小可运行闭环，再扩展高级选项。

### 5.2 数据迁移

1. 每张新表必须有 MySQL/PostgreSQL migration。
2. migration 必须说明是否可回滚。
3. 配置、菜单、权限、插件数据要有 seed 和升级策略。
4. 对已有数据结构的修改必须给出破坏式发布说明、数据重建方式和验收步骤。

### 5.3 API 与公开契约

1. API 路径、请求、响应和错误码必须稳定。
2. OpenAPI 是契约源之一，不允许后端改了前端猜。
3. 废弃字段不保留双轨实现；需要破坏式调整时同步更新 OpenAPI、前端 client、文档和示例。
4. 插件 manifest 必须 semver 化。

### 5.4 安全

1. 权限默认拒绝。
2. 插件安装默认最小权限。
3. 文件上传默认私有。
4. 配置项要区分普通配置和敏感配置。
5. 所有管理端高风险操作必须二次确认和审计。

### 5.5 测试策略

1. domain/service/store/handler 分层测试，不用一个 E2E 测所有逻辑。
2. 插件和生成器必须有 golden file 或 fixture。
3. 权限失败路径和数据范围路径必须测试。
4. 前端关键页面至少要有 smoke 脚本或手工验收脚本。

### 5.6 前端体验

1. 页面必须处理加载、空态、错误、无权限、保存中、删除确认。
2. 菜单、按钮、路由权限必须同源。
3. 表格、表单、弹窗、批量操作使用统一交互。
4. 低代码和生成器页面不能牺牲可读性和可调试性。

### 5.7 插件治理

1. 插件必须声明菜单、权限、配置、资产、迁移和风险。
2. 插件必须能禁用、卸载、升级、回滚。
3. 插件不能直接污染核心表，必须通过公开 extension point。
4. 插件示例要随框架版本持续维护。

### 5.8 工程协作

1. 每个 PR 标注所属里程碑和任务 ID。
2. 不允许混入无关重构。
3. 每次合并前跑对应最小门禁。
4. 失败验收必须形成返工记录，不靠口头记忆。

## 6. 推荐执行顺序

1. 先完成 M0。没有架构章程、质量基线和验收模板，不进入大规模代码改造。
2. 接着做 M1。权限资源目录和菜单注册中心是所有后台能力的底座。
3. 然后做 M2 和 M3。审计与文件管理是 GVA 高频能力，也能验证 Skoll 的分层和 adapter。
4. 再做 M4。字典、配置、组织、数据范围补齐企业后台真实使用场景。
5. 最后做 M5/M6。代码生成器和插件市场依赖前面的权限、菜单、审计、文件、配置基础。
6. M7 贯穿全程，但在每个里程碑结束时集中验收。

## 7. Leader 建议

1. Skoll 的定位要从“另一个后台脚手架”升级为“开源 Admin 平台内核”。这意味着核心不是页面数量，而是扩展协议、权限一致性、生成质量和插件生命周期。
2. 架构上保持克制。核心复杂域用 DDD，普通 CRUD 不要 DDD 化，技术适配只做 port/adapter。
3. 短期不要追求一次性超过 GVA 的全部功能。先把权限、菜单、审计、文件、字典、生成器这六条主链路做成可运行闭环。
4. 每个里程碑必须有一个演示路径。例如“创建角色 -> 授权菜单/API/按钮 -> 安装插件 -> 插件菜单出现 -> 操作被审计”。
5. 生成器是分水岭。生成器如果输出架构正确、可测试、可回滚的代码，Skoll 才真正超过传统 Admin 框架。
6. 插件市场必须晚于插件安全。没有签名、风险、权限 diff、迁移和回滚的市场只是下载页。
