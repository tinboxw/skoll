# Skoll 后续细化 Work Items

> 日期: 2026-06-19
> 适用范围: 当前 `work_items.md` 已完成后的下一批 M3-M7 任务。
> 执行规则: 本文件是下一阶段候选最小任务池。进入开发前，按优先级搬入正式 `work_items.md` 或创建新的批次任务表；每完成一个 Work Item 且验收通过后提交一次代码。

## 基本原则

1. 当前批次 `work_items.md` 已完成 `170/170/0`，不再回写新功能任务破坏完成态。
2. 下一批任务继续遵守 Modular Monolith + Clean/Hexagonal boundaries + Tactical DDD。
3. Skoll 处于开源基础建设阶段，不做旧接口、旧数据结构、旧插件格式、旧页面路径兼容方案。
4. 单项任务粒度控制在 0.5-1 天，跨后端、前端、测试、文档的大包必须拆分。
5. 每项任务必须有交付物、验收标准和验证命令；验收不通过则任务退回重新执行。

## 任务状态

- `Todo`: 待执行
- `Doing`: 执行中
- `Review`: 待验收
- `Failed`: 验收失败，需返工
- `Done`: 验收通过且已提交
- `Blocked`: 外部条件阻塞

## 阶段建议

| 阶段 | 目标 | 建议顺序 | 里程碑验收 |
| --- | --- | --- | --- |
| N0 | 发布前冻结与质量门禁 | 先执行 | 文档入口、验收记录、质量命令一致 |
| N1/M3 | 文件与对象存储平台 | 第二 | 本地文件上传、下载、删除、权限、审计可用 |
| N2/M4 | 字典、配置、组织与数据范围 | 第三 | 字典驱动表单，用户列表受数据范围过滤 |
| N3/M5 | 代码生成器 v1 | 第四 | 单表 CRUD 从 spec 到前后端页面跑通 |
| N4/M6 | 插件市场与生命周期 | 第五 | 插件安装预检、签名、迁移、灰度、回滚可验收 |
| N5/M7 | 开源发布质量与体验 | 最后 | CI、文档、示例、发布检查清单可执行 |

## N0: 发布前冻结与质量门禁

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 1 | N0-01 | N0 | `skoll-refactor-governance` | 完成态一致性校验 | `docs/refactor/n0_completion_consistency_check.md` | `work_items.md` 当前 170 项全部 Done；`acceptance_log.md` 和 `git log` 覆盖 FE6 尾盘；M3-M7 Todo 被识别为后续路线图 | `git log --oneline -20`; 文档审阅 | Done |
| 2 | N0-02 | N0 | `skoll-docs-writer` | 文档入口复查 | 文档索引修正 | `docs/README.md`、`docs/refactor/README.md`、`source_map.md` 不指向已移除主目录文件 | `rg -n "docs/refactor/fe|progress_inspection_2026-06-19|task_adjustment" docs/README.md docs/refactor/README.md docs/refactor/source_map.md` | Todo |
| 3 | N0-03 | N0 | `skoll-quality-gate` | Go 全量测试 | 质量门禁记录 | `go test ./...` 结果记录，失败项列入返工任务 | `go test ./...` | Todo |
| 4 | N0-04 | N0 | `skoll-quality-gate` | 前端类型和构建门禁 | 质量门禁记录 | typecheck 与 build 结果记录，失败项列入返工任务 | `cd web && npm run typecheck`; `cd web && npm run build` | Todo |
| 5 | N0-05 | N0 | `skoll-open-source-framework` | 发布前范围冻结说明 | release scope note | 明确下一阶段只从本文件领取任务，不追加兼容方案 | 文档审阅 | Todo |

## N1/M3: 文件与对象存储平台

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 6 | M3-01-01 | M3-01 | `skoll-file-storage-refactor` | 定义 FileObject 值对象 | `internal/domain/file` | key/name/size/mime/hash/owner/visibility/storage/status 字段和校验完整 | `go test ./internal/domain/file/...` | Todo |
| 7 | M3-01-02 | M3-01 | `skoll-file-storage-refactor` | 定义 ObjectStore port | port interface | Put/Get/Delete/Stat/Presign 接口不依赖具体存储实现 | `go test ./internal/domain/file/...` | Todo |
| 8 | M3-01-03 | M3-01 | `skoll-file-storage-refactor` | 定义 multipart port | port interface | Init/UploadPart/Complete/Abort 支持 hash 校验上下文 | `go test ./internal/domain/file/...` | Todo |
| 9 | M3-02-01 | M3-02 | `skoll-file-storage-refactor` | 实现 local object adapter | local adapter | 上传、读取、删除、stat 可用 | `go test ./internal/store/object/...` | Todo |
| 10 | M3-02-02 | M3-02 | `skoll-security-hardening` | local adapter 路径安全 | tests | 路径穿越、非法 key、覆盖保护测试通过 | `go test ./internal/store/object/...` | Todo |
| 11 | M3-03-01 | M3-03 | `skoll-database-development` | 文件元数据 migration | MySQL/PostgreSQL SQL | 字段、索引、唯一约束、状态字段完整 | migration review | Todo |
| 12 | M3-03-02 | M3-03 | `skoll-database-development` | 文件元数据 store | memory/sql store | metadata 写入、查询、删除、状态更新契约一致 | `go test ./internal/store/...` | Todo |
| 13 | M3-04-01 | M3-04 | `skoll-file-storage-refactor` | 文件上传 service | service | 元数据和对象写入失败时状态一致，不产生幽灵记录 | `go test ./internal/service/file/...` | Todo |
| 14 | M3-04-02 | M3-04 | `skoll-permission-rbac` | 文件访问权限策略 | service tests | private/public/plugin_asset 访问规则明确，无权限不可下载 | `go test ./internal/service/file/... ./internal/service/rbac/...` | Todo |
| 15 | M3-04-03 | M3-04 | `skoll-observability-audit` | 文件审计事件 | audit integration | upload/download/delete/forbidden 均写审计 | `go test ./internal/service/file/... ./internal/service/audit/...` | Todo |
| 16 | M3-05-01 | M3-05 | `skoll-api-contracts` | 文件 API 契约 | OpenAPI | upload/download/delete/list/detail 错误码和响应字段明确 | OpenAPI review | Todo |
| 17 | M3-05-02 | M3-05 | `skoll-api-contracts` | 文件 API handler | handler tests | 上传、下载、删除、列表、权限拒绝路径测试通过 | `go test ./internal/handler/http/v1/file/...` | Todo |
| 18 | M3-06-01 | M3-06 | `skoll-file-storage-refactor` | 分片上传 service | service | init/upload/complete/abort 可用，hash 失败不可完成 | `go test ./internal/service/file/...` | Todo |
| 19 | M3-06-02 | M3-06 | `skoll-api-contracts` | 分片上传 API | handler + OpenAPI | 分片接口契约、错误码、清理策略明确 | `go test ./internal/handler/http/v1/file/...` | Todo |
| 20 | M3-07-01 | M3-07 | `skoll-vue-frontend` | 文件 API client/store | web client/store | 上传进度、错误、取消、刷新状态统一 | `cd web && npm run typecheck` | Todo |
| 21 | M3-07-02 | M3-07 | `skoll-frontend-design-refactor` | 文件管理页面 | Vue page | 搜索、预览、下载、删除、空态、错误态、无权限态完整 | `cd web && npm run build` | Todo |
| 22 | M3-08-01 | M3-08 | `skoll-testing-automation` | 文件安全 smoke | tests/scripts | 路径穿越、非法 mime、超大文件、未授权下载覆盖 | `go test ./...` | Todo |

## N2/M4: 字典、配置、组织与数据范围

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 23 | M4-01-01 | M4-01 | `skoll-dictionary-organization-refactor` | DictionaryType/Item domain | domain | 类型、条目、排序、状态、系统内置标记校验完整 | `go test ./internal/domain/system/...` | Todo |
| 24 | M4-01-02 | M4-01 | `skoll-database-development` | 字典 migration/store | migration + store | memory/sql 契约一致，唯一约束完整 | `go test ./internal/store/...` | Todo |
| 25 | M4-02-01 | M4-02 | `skoll-cache-development` | 字典缓存策略 | cache/service | 更新后缓存失效，错误不污染缓存 | `go test ./internal/cache/... ./internal/service/system/...` | Todo |
| 26 | M4-03-01 | M4-03 | `skoll-api-contracts` | 字典 API 契约与 handler | OpenAPI + handler | 类型、条目、状态、搜索、排序完整 | `go test ./internal/handler/http/v1/system/...` | Todo |
| 27 | M4-03-02 | M4-03 | `skoll-vue-frontend` | 字典管理 UI | Vue page | 类型/条目管理、启停、排序、空态、错误态完整 | `cd web && npm run build` | Todo |
| 28 | M4-04-01 | M4-04 | `skoll-data-dictionary-config` | 配置 schema registry | domain/service | 系统和插件配置均可用 schema 描述字段、校验、默认值 | `go test ./internal/service/system/...` | Done |
| 29 | M4-04-02 | M4-04 | `skoll-vue-frontend` | SchemaForm 配置接入 | Vue page | 系统设置和插件配置复用 SchemaForm | `cd web && npm run typecheck`; `cd web && npm run build` | Done |
| 30 | M4-05-01 | M4-05 | `skoll-dictionary-organization-refactor` | 组织/部门/岗位 domain | domain | 部门树、岗位、用户归属模型完整 | `go test ./internal/domain/organization/...` | Done |
| 31 | M4-05-02 | M4-05 | `skoll-database-development` | 组织 migration/store | migration + store | 部门树查询、岗位关联、用户归属可用 | `go test ./internal/store/...` | Done |
| 32 | M4-06-01 | M4-06 | `skoll-permission-rbac` | DataScope domain/service | domain/service | all/department/department_tree/self/custom 解析明确 | `go test ./internal/service/rbac/...` | Done |
| 33 | M4-06-02 | M4-06 | `skoll-permission-rbac` | 用户列表数据范围过滤 | service/store | 不同角色返回不同用户列表，super_admin 绕过明确 | `go test ./internal/service/user/... ./internal/service/rbac/...` | Done |
| 34 | M4-07-01 | M4-07 | `skoll-vue-frontend` | 组织管理 UI | Vue page | 部门树、岗位、用户归属编辑可用 | `cd web && npm run build` | Done |
| 35 | M4-07-02 | M4-07 | `skoll-frontend-testing-refactor` | 数据范围验收清单 | docs/smoke | admin、部门管理员、普通用户三类结果可复查 | 文档审阅; `cd web && npm run build` | Done |

## N3/M5: 代码生成器 v1

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 36 | M5-01-01 | M5-01 | `skoll-generator-refactor` | GeneratorSpec domain | domain | 模块、表、字段、索引、校验、权限、菜单、页面描述完整 | `go test ./internal/domain/generator/...` | Done |
| 37 | M5-01-02 | M5-01 | `skoll-generator-refactor` | GeneratorSpec validation | domain tests | 命名、字段类型、冲突、权限 key 校验覆盖 | `go test ./internal/domain/generator/...` | Done |
| 38 | M5-02-01 | M5-02 | `skoll-generator-refactor` | 模板输出规范 | docs/templates | 输出路径、命名、禁止覆盖用户改动策略明确 | 文档审阅 | Done |
| 39 | M5-03-01 | M5-03 | `skoll-generator-refactor` | dry-run 文件清单 | service | 生成前展示文件清单和变更摘要 | `go test ./internal/service/generator/...` | Done |
| 40 | M5-03-02 | M5-03 | `skoll-generator-refactor` | dry-run diff | service | 新增、修改、冲突文件 diff 可读 | `go test ./internal/service/generator/...` | Done |
| 41 | M5-04-01 | M5-04 | `skoll-code-generator` | 后端 domain/store 生成 | templates | domain、migration、store、repository 输出可编译 | `go test ./...` | Done |
| 42 | M5-04-02 | M5-04 | `skoll-code-generator` | 后端 service/handler/OpenAPI 生成 | templates | CRUD API、权限、审计、OpenAPI 同步 | `go test ./...` | Done |
| 43 | M5-05-01 | M5-05 | `skoll-code-generator` | 前端 API/store 生成 | templates | API client、Pinia store、错误状态生成 | `cd web && npm run typecheck` | Done |
| 44 | M5-05-02 | M5-05 | `skoll-code-generator` | 前端 list/form 页面生成 | templates | 列表、筛选、表单、状态组件、权限按钮生成 | `cd web && npm run build` | Done |
| 45 | M5-06-01 | M5-06 | `skoll-generator-refactor` | 生成历史记录 | store/service | 批次、hash、spec、操作者、文件清单可查 | `go test ./internal/service/generator/...` | Done |
| 46 | M5-06-02 | M5-06 | `skoll-generator-refactor` | 回滚机制 | service | 未修改文件可回滚，冲突文件给人工处理清单 | `go test ./internal/service/generator/...` | Done |
| 47 | M5-07-01 | M5-07 | `skoll-testing-automation` | generator golden tests | tests | 幂等、冲突、回滚、模板输出测试通过 | `go test ./...` | Done |
| 48 | M5-08-01 | M5-08 | `skoll-testing-automation` | demo_product 生成验收 | example | 单表 CRUD 从 spec 到前后端页面跑通 | `go test ./...`; `cd web && npm run build` | Done |

## N4/M6: 插件市场与生命周期

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 49 | M6-01-01 | M6-01 | `skoll-plugin-marketplace-refactor` | marketplace index schema | schema/docs | id/version/skoll_version/risk/signature/source/changelog 完整 | schema review | Done |
| 50 | M6-01-02 | M6-01 | `skoll-plugin-marketplace-refactor` | index 校验器 | service/tests | 无效索引、重复插件、版本冲突能报错 | `go test ./internal/plugin/...` | Done |
| 51 | M6-02-01 | M6-02 | `skoll-plugin-platform` | 本地市场 service/API | service/API | 本地插件、可安装包、签名、风险可展示 | `go test ./internal/plugin/... ./internal/handler/http/v1/plugin/...` | Done |
| 52 | M6-02-02 | M6-02 | `skoll-vue-frontend` | 市场列表 UI | Vue page | 搜索、筛选、风险、签名、版本、安装入口完整 | `cd web && npm run build` | Done |
| 53 | M6-03-01 | M6-03 | `skoll-plugin-marketplace-refactor` | 远程 index adapter | service | 远程失败不影响本地插件，错误可见 | `go test ./internal/plugin/...` | Done |
| 54 | M6-03-02 | M6-03 | `skoll-plugin-marketplace-refactor` | 安装预检 service | service | 权限 diff、菜单 diff、配置、资源、迁移、签名、风险完整 | `go test ./internal/plugin/...` | Done |
| 55 | M6-03-03 | M6-03 | `skoll-web-ui-design` | 安装预检 UI | Vue page | 高风险动作确认、阻断原因、审计线索可读 | `cd web && npm run build` | Done |
| 56 | M6-04-01 | M6-04 | `skoll-plugin-platform` | 插件迁移 hook | plugin/service | install/upgrade/downgrade/uninstall 状态可追踪 | `go test ./internal/plugin/...` | Done |
| 57 | M6-05-01 | M6-05 | `skoll-security-hardening` | 插件签名策略升级 | security/plugin | manifest、资产清单、后端入口、前端资产 hash 覆盖 | `go test ./internal/plugin/...` | Done |
| 58 | M6-06-01 | M6-06 | `skoll-plugin-marketplace-refactor` | 灰度策略 service | service | 路由、菜单、功能可见性受灰度策略影响 | `go test ./internal/plugin/...` | Done |
| 59 | M6-06-02 | M6-06 | `skoll-plugin-marketplace-refactor` | 回滚 service/UI | service/frontend | 回滚后版本、菜单、权限、配置、资产一致 | `go test ./...`; `cd web && npm run build` | Done |
| 60 | M6-07-01 | M6-07 | `skoll-testing-automation` | 插件生命周期验收 | tests | install/enable/disable/upgrade/rollback 全链路通过 | `go test ./...` | Done |
| 61 | M6-07-02 | M6-07 | `skoll-web-ui-design` | 插件安全报告 UI | Vue 页面 | 安装前后均可查看风险和权限影响 | `cd web && npm run build` | Done |

## N5/M7: 开源发布质量与体验

| 顺序 | Work Item | 父任务 | Skill | 任务 | 交付物 | 验收要求 | 验证命令 | 状态 |
|---:|---|---|---|---|---|---|---|---|
| 62 | M7-01-01 | M7-01 | `skoll-quality-gate` | CI Go 门禁 | CI | gofmt/go test/coverage 进入 CI | CI review | Done |
| 63 | M7-01-02 | M7-01 | `skoll-quality-gate` | CI 前端门禁 | CI | typecheck/build/browser smoke 最小集进入 CI 或文档化阻塞原因 | CI review | Done |
| 64 | M7-01-03 | M7-01 | `skoll-api-contracts` | OpenAPI/manifest 校验门禁 | CI | OpenAPI 和 plugin manifest schema 校验进入 CI | CI review | Done |
| 65 | M7-02-01 | M7-02 | `skoll-testing-automation` | 覆盖率目标和例外名单 | docs/CI | 核心包覆盖率目标、暂缓包、提升计划明确 | `go test ./... -coverprofile=coverage.out` | Done |
| 66 | M7-03-01 | M7-03 | `skoll-performance-scaling` | 后端性能基线脚本 | scripts/docs | login/user list/role list/plugin list/audit query 有 QPS/P95/P99 | k6 或 vegeta 脚本 | Done |
| 67 | M7-03-02 | M7-03 | `skoll-frontend-performance-refactor` | 前端性能采样记录 | docs | 首屏、路由切换、重表格、插件面板有基线 | `cd web && npm run build` | Todo |
| 68 | M7-04-01 | M7-04 | `skoll-docs-writer` | 快速开始手册 | docs | 新环境 30 分钟内启动、登录、执行核心流程 | 手工复现 | Todo |
| 69 | M7-04-02 | M7-04 | `skoll-docs-writer` | 运维手册 | docs | 配置、部署、备份、日志、常见故障路径明确 | 文档审阅 | Todo |
| 70 | M7-05-01 | M7-05 | `skoll-open-source-framework` | 示例模块 | examples | 至少一个生成模块可运行 | `go test ./...`; `cd web && npm run build` | Todo |
| 71 | M7-05-02 | M7-05 | `skoll-open-source-framework` | 示例插件 | examples | 至少一个插件可安装、启停、回滚 | `go test ./...`; `cd web && npm run build` | Todo |
| 72 | M7-06-01 | M7-06 | `skoll-community-governance` | 开源贡献模板 | docs/.github | issue/PR/security/contributing 模板完整 | 文档审阅 | Todo |
| 73 | M7-06-02 | M7-06 | `skoll-devops-deployment` | Release checklist | docs | 版本号、迁移、镜像、OpenAPI、示例、质量门禁清单完整 | 文档审阅 | Todo |

## 建议执行策略

1. 先执行 N0，确保当前完成态可发布、可追溯。
2. M3 和 M4 是平台基础能力，优先于生成器和插件市场。
3. M5 依赖 M1/M3/M4 的权限、文件、字典、配置能力，不宜提前实现大而全生成器。
4. M6 依赖 M1 权限目录、M2 审计、M3 文件资产、M4 配置 schema，建议在 M3/M4 后启动。
5. M7 贯穿执行，但正式收口放在 M3-M6 可运行后。
