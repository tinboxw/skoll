# Skoll 医药 OA 里程碑与原子任务规划 2026-07-04

> 目标: 以医药 OA 行业插件为牵引，补齐 Skoll 业务插件开发、界面体验、工作流、表单、进销存、CRM、合规质量等能力。
> 当前规则: 不做旧接口、旧数据结构、旧插件格式、旧页面路径兼容方案。
> 当前状态: M0-M7 基础建设任务已完成；本文件是后续可插队到 task-board/work-items 的候选任务池。

## 进度距离评估

CodeGraph 抽查结果显示:

- `Workflow`: 未发现业务域实现。
- `Customer`: 未发现业务域实现。
- `Employee`: 未发现业务域实现。
- `Inventory`: 仅发现插件页面中的 UI 面板标识，不是库存业务域。

因此，Skoll 当前更接近“框架底座已完成”，距离“医药 OA 可交付插件”还需要 F6-F12 七个里程碑。重点不是单纯补页面，而是先补齐业务插件基建，再按行业业务域落地。

## 里程碑总览

| 里程碑 | 目标 | 优先级 | 依赖 | 交付物 | 里程碑验收 |
| --- | --- | --- | --- | --- | --- |
| F6: 业务插件平台底座 | 插件可声明数据、API、权限、菜单、页面和宿主能力调用 | P0 | M0-M7,F0 | Host SDK、Data Contract、API Contract、Theme/UI Kit | sample business plugin 可被安装、启用、授权、访问 API、渲染页面 |
| F7: 工作流与动态表单平台 | 支撑 OA 审批、业务单据、待办通知 | P0 | F6 | Workflow Engine、Form Builder、Todo/Notification | 请假、采购申请、合同审批 3 条流程端到端跑通 |
| F8: 医药主数据插件 | 建立员工、药品、供应商、客户、仓库等基础业务对象 | P0 | F6,F7 | pharma_oa 插件骨架和主数据模块 | 主数据 CRUD、权限、审计、导入导出、附件可验收 |
| F9: 医药进销存能力 | 建立采购、销售、库存、批号、效期、库存预警 | P0 | F8,F7 | 采购/销售/库存模块 | 采购入库到销售出库闭环跑通，库存流水和批号效期准确 |
| F10: 医药 OA 协同与合规 | 建立公告、合同、资质、质量投诉、召回、合规审计 | P0 | F7,F8,F9 | 协同与合规模块 | 资质到期、质量投诉、药品召回可提醒、可审批、可审计 |
| F11: 医药 CRM 与经营看板 | 建立客户跟进、销售机会、回款、经营指标 | P1 | F8,F9,F10 | CRM 模块、Dashboard widgets、报表 | 客户跟进、销售漏斗、库存预警、资质到期、审批效率可视化 |
| F12: 医药 OA 样板验收 | 形成可演示、可二次开发、可复制的行业插件样板 | P0 | F6-F11 | demo 数据、E2E smoke、行业 README | 新员工入职、采购申请、入库、销售出库、资质预警、客户跟进全链路通过 |

## F6: 业务插件平台底座

| Work Item | 优先级 | 依赖 | 预期交付物 | 验收标准 | Skills |
| --- | --- | --- | --- | --- | --- |
| F6-01 Theme Engine | P0 | F0 | theme store、CSS tokens、Element Plus variable mapping | light/dark/compact 可切换，刷新后保留，插件 iframe/页面同步主题 | skoll-frontend-design-refactor, skoll-vue-frontend |
| F6-02 Plugin UI Kit | P0 | F6-01 | PageShell、DataTable、SchemaForm、DetailDrawer、ConfirmAction | 主应用和插件页面视觉、密度、状态、危险操作一致 | skoll-frontend-design-refactor, skoll-vue-frontend |
| F6-03 Plugin Host SDK | P0 | M6 | 前端 SDK、后端 PluginContext、SDK 使用示例 | 插件可调用 auth、user、org、dict、file、audit、config、permission 基础能力 | skoll-plugin-platform, skoll-api-contracts |
| F6-04 Plugin Data Contract | P0 | F6-03 | data manifest、migration runner、namespace 规则 | 插件可声明表、索引、迁移、卸载策略，安装预检展示风险 | skoll-plugin-platform, skoll-database-development |
| F6-05 Plugin API/OpenAPI Contract | P0 | F6-03,F6-04 | route registry、OpenAPI 聚合、权限绑定、审计动作声明 | 插件 API 自动进入权限、OpenAPI、审计和前端 client | skoll-plugin-platform, skoll-api-contracts |
| F6-06 Business Event Bus | P0 | F6-03 | 事件模型、发布订阅、事务后事件、重试记录 | 插件可订阅审批完成、入库完成、资质到期等事件 | skoll-plugin-platform, skoll-workflow-jobs |
| F6-07 Business Plugin Generator | P0 | F6-02..F6-05 | 业务对象生成模板 | 从一个业务对象 spec 生成 API、UI、权限、菜单、迁移、测试并构建通过 | skoll-code-generator, skoll-generator-refactor |

## F7: 工作流与动态表单平台

| Work Item | 优先级 | 依赖 | 预期交付物 | 验收标准 | Skills |
| --- | --- | --- | --- | --- | --- |
| F7-01 Workflow Domain | P0 | F6-04,F6-05 | 流程定义、实例、节点、任务、动作模型 | 发起、审批、驳回、撤回、转办、抄送动作具备单元测试 | skoll-workflow-jobs, skoll-go-backend |
| F7-02 Workflow API | P0 | F7-01 | OpenAPI、handler、service、错误码 | 3 条流程通过 API 跑通，权限和审计记录完整 | skoll-api-contracts, skoll-workflow-jobs |
| F7-03 Workflow UI | P0 | F7-02,F6-02 | 流程列表、发起页、审批页、时间线抽屉 | 待审批、已审批、我发起、抄送视图完整，状态不重叠 | skoll-vue-frontend, skoll-frontend-testing-refactor |
| F7-04 Form Builder Schema | P0 | F6-04 | 表单 schema、字段类型、校验、明细表、附件字段 | 可表达请假、报销、采购申请、入库单、客户资质表单 | skoll-data-dictionary-config, skoll-api-contracts |
| F7-05 Form Builder UI | P0 | F7-04,F6-02 | 表单设计器、预览、保存、版本管理 | 用户可配置并预览表单，保存后流程发起使用该表单 | skoll-vue-frontend, skoll-frontend-design-refactor |
| F7-06 Todo/Notification Center | P0 | F7-02 | 待办、已办、站内信、提醒规则 | 审批待办和业务提醒可从通知跳转到目标对象 | skoll-workflow-jobs, skoll-vue-frontend |
| F7-07 Workflow Audit Fixtures | P0 | F7-02..F7-06 | fixtures、smoke、回放记录 | 审批全链路可审计、可回放、可查询 | skoll-testing-automation, skoll-observability-audit |

## F8: 医药主数据插件

| Work Item | 优先级 | 依赖 | 预期交付物 | 验收标准 | Skills |
| --- | --- | --- | --- | --- | --- |
| F8-01 pharma_oa 插件骨架 | P0 | F6 | manifest、菜单、权限、路由、demo 数据入口 | 插件可安装、启用、禁用，主菜单出现医药 OA 分组 | skoll-plugin-platform, skoll-open-source-framework |
| F8-02 员工管理 | P0 | F8-01,F6-07 | 员工档案、部门岗位、证照附件、状态流转 | 员工新增、编辑、离职、证照到期提醒、审计完整 | skoll-go-backend, skoll-vue-frontend |
| F8-03 药品/商品档案 | P0 | F8-01 | 药品档案、规格、剂型、厂家、批准文号、温控属性 | 药品档案 CRUD、导入、停用、审计完整 | skoll-data-dictionary-config, skoll-go-backend |
| F8-04 供应商管理 | P0 | F8-01 | 供应商档案、联系人、资质附件、评级、状态 | 资质到期提醒可触发通知，停用供应商不能用于采购 | skoll-file-storage, skoll-security-hardening |
| F8-05 客户管理 | P0 | F8-01 | 客户档案、联系人、资质附件、区域归属 | 客户数据按组织/负责人隔离，资质过期可拦截销售 | skoll-permission-rbac, skoll-vue-frontend |
| F8-06 仓库与库位 | P0 | F8-01 | 仓库、库区、库位、温控属性 | 出入库只能选择启用仓库和有效库位 | skoll-database-development, skoll-go-backend |
| F8-07 主数据导入导出 | P1 | F8-02..F8-06 | Excel 模板、导入校验、错误回执、导出任务 | 员工、药品、供应商、客户可批量导入，错误行清晰 | skoll-file-storage, skoll-testing-automation |

## F9: 医药进销存能力

| Work Item | 优先级 | 依赖 | 预期交付物 | 验收标准 | Skills |
| --- | --- | --- | --- | --- | --- |
| F9-01 库存领域模型 | P0 | F8-03,F8-06 | 库存余额、库存流水、批号、效期、锁定库存模型 | 入库、出库、盘点、调拨均产生不可篡改流水 | skoll-database-development, skoll-go-backend |
| F9-02 采购申请与采购订单 | P0 | F7,F8-04 | 采购申请、审批、采购订单 | 采购申请审批通过后生成采购订单，供应商资质校验生效 | skoll-workflow-jobs, skoll-api-contracts |
| F9-03 采购入库 | P0 | F9-01,F9-02 | 入库单、批号、效期、附件 | 入库后库存余额增加，批号效期写入，审计记录完整 | skoll-go-backend, skoll-vue-frontend |
| F9-04 销售订单与销售出库 | P0 | F9-01,F8-05 | 销售订单、出库单、客户资质校验 | 客户资质过期不能出库，出库后库存减少且流水准确 | skoll-security-hardening, skoll-vue-frontend |
| F9-05 盘点与调拨 | P1 | F9-01 | 盘点单、调拨单、差异处理 | 盘点差异需审批，调拨跨仓产生双向流水 | skoll-workflow-jobs, skoll-go-backend |
| F9-06 库存预警 | P0 | F9-01 | 近效期、低库存、超库存预警任务 | 预警进入待办/通知中心，点击可定位库存明细 | skoll-workflow-jobs, skoll-observability-audit |
| F9-07 进销存端到端 smoke | P0 | F9-02..F9-06 | smoke 数据和验收脚本 | 采购申请、入库、销售订单、出库、库存预警全链路通过 | skoll-testing-automation, skoll-quality-gate |

## F10: 医药 OA 协同与合规

| Work Item | 优先级 | 依赖 | 预期交付物 | 验收标准 | Skills |
| --- | --- | --- | --- | --- | --- |
| F10-01 公告制度 | P1 | F7 | 公告、制度、阅读确认 | 公告可按组织/角色发布，阅读确认可查询 | skoll-vue-frontend, skoll-go-backend |
| F10-02 合同档案 | P0 | F7,F8-04,F8-05 | 合同台账、附件、审批关联、到期提醒 | 合同可关联客户/供应商和审批流程，到期可提醒 | skoll-file-storage, skoll-workflow-jobs |
| F10-03 资质管理 | P0 | F8-02,F8-04,F8-05 | 员工/供应商/客户资质台账 | 资质过期可提醒并拦截采购/销售关键动作 | skoll-security-hardening, skoll-observability-audit |
| F10-04 质量投诉 | P0 | F7,F8-03,F8-05 | 投诉登记、处理流程、附件、结论 | 投诉可发起流程，处理过程可审计，可关联药品批号 | skoll-workflow-jobs, skoll-file-storage |
| F10-05 药品召回 | P0 | F9-01,F10-04 | 召回单、影响批次、客户范围、处理跟踪 | 按批号定位出库客户，召回任务可跟踪完成状态 | skoll-go-backend, skoll-vue-frontend |
| F10-06 冷链/温湿度记录 | P1 | F8-06,F9-01 | 温湿度记录、异常提醒 | 异常记录进入风险看板，关联库存批号 | skoll-observability-audit, skoll-workflow-jobs |
| F10-07 合规审计看板 | P0 | F10-02..F10-06 | 资质、投诉、召回、冷链风险看板 | 高风险事项可筛选、处理、追踪，审计记录完整 | skoll-frontend-design-refactor, skoll-observability-audit |

## F11: 医药 CRM 与经营看板

| Work Item | 优先级 | 依赖 | 预期交付物 | 验收标准 | Skills |
| --- | --- | --- | --- | --- | --- |
| F11-01 客户跟进 | P1 | F8-05 | 跟进记录、拜访计划、附件 | 销售人员只能查看自己或授权范围客户跟进 | skoll-permission-rbac, skoll-vue-frontend |
| F11-02 销售机会 | P1 | F8-05,F9-04 | 机会阶段、预计金额、关联客户和商品 | 销售机会可推进阶段并进入看板统计 | skoll-go-backend, skoll-vue-frontend |
| F11-03 回款与发票记录 | P1 | F9-04 | 回款计划、发票记录、附件 | 销售订单可关联回款与发票，逾期提醒可触发 | skoll-file-storage, skoll-workflow-jobs |
| F11-04 经营指标 API | P0 | F9,F10,F11-01 | 指标聚合 API | 库存预警、资质到期、审批效率、客户跟进、销售趋势可查询 | skoll-api-contracts, skoll-performance-scaling |
| F11-05 医药 OA Dashboard | P0 | F11-04,F6-02 | dashboard widgets | 关键指标响应快，空态/错误态/无权限态完整 | skoll-frontend-performance-refactor, skoll-vue-frontend |
| F11-06 报表导出 | P1 | F11-04 | 报表导出任务 | 常用报表可异步导出并留下审计记录 | skoll-file-storage, skoll-observability-audit |

## F12: 医药 OA 样板验收

| Work Item | 优先级 | 依赖 | 预期交付物 | 验收标准 | Skills |
| --- | --- | --- | --- | --- | --- |
| F12-01 Demo 数据集 | P0 | F8-F11 | 员工、药品、供应商、客户、仓库、库存、流程 demo 数据 | 一键初始化后可演示完整医药 OA 场景 | skoll-database-development, skoll-testing-automation |
| F12-02 端到端验收脚本 | P0 | F8-F11 | E2E smoke 脚本 | 新员工入职、采购申请、入库、销售出库、资质预警、客户跟进均通过 | skoll-testing-automation, skoll-quality-gate |
| F12-03 行业插件 README | P1 | F12-01 | 插件说明、安装、功能边界、二开指南 | 新开发者可按文档安装、运行、理解业务域 | skoll-docs-writer, skoll-open-source-framework |
| F12-04 性能与权限验收 | P0 | F12-02 | 性能采样、权限矩阵、风险清单 | 大列表分页、常用查询、数据权限、审批权限、库存权限通过验收 | skoll-performance-scaling, skoll-security-hardening |
| F12-05 里程碑关闭报告 | P0 | F12-01..F12-04 | 验收报告、问题清单、后续任务建议 | 所有 P0 验收通过，未通过项回流到 work_items 重新执行 | skoll-refactor-governance, skoll-quality-gate |

## 执行建议

1. 不要先写医药 OA 页面。先完成 F6/F7，否则业务插件会绕开框架基建，形成新的内聚风险。
2. F8-F10 必须坚持“真实 API、真实权限、真实审计、真实状态”，不能使用静态假数据作为验收。
3. 医药 OA 是样板插件，不应侵入主应用核心目录；主应用只提供宿主 SDK、主题、权限、菜单、路由、审计、工作流等平台能力。
4. 每个 Work Item 完成后必须独立验收并提交一次代码；验收不通过则回到该 Work Item 重新执行，不进入下一项。
5. 当前正式 `work_items.md` 已全部完成，本文件中的任务应由研发主管按优先级拆入新的 task-board/work-items 执行批次。
