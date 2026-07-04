# Skoll 业务插件能力与医药 OA 目标规划 2026-07-04

> 目标: 让 Skoll 不只是能安装插件，而是能承载医药行业 OA、供应链、CRM、合规质量等真实业务插件。
> 当前规则: Skoll 处于开源基础建设阶段，不设计旧接口、旧数据结构、旧插件格式、旧页面路径兼容方案。
> 索引依据: 优先使用 CodeGraph。当前索引显示 482 files / 8,347 nodes / 25,294 edges，索引已同步。

## 当前判断

### 前端 Element Plus 统一情况

当前前端已经使用 Element Plus:

- `web/package.json` 依赖 `element-plus`。
- `web/vite.config.ts` 使用 `ElementPlusResolver`。
- `web/src/components/Common` 已有 Dialog、Form、SchemaForm、StateBlock、Table 等通用组件。
- 核心页面大量使用 `el-button`、`el-table`、`el-form`、`el-drawer`、`el-tabs`、`el-tag`、`ElMessage`、`ElMessageBox`。

但还不能认为风格已经完全统一:

- 页面级布局、工具栏、筛选区、详情抽屉、危险操作确认仍存在各自实现。
- 缺少稳定的 `PageShell`、`PageToolbar`、`FilterBar`、`DataTable`、`DetailDrawer`、`ConfirmAction` 设计模式。
- `Common/Table.vue` 只是表格外壳，还没有覆盖批量操作、列密度、空态、错误态、分页、行操作、移动端策略。

结论: Element Plus 已采用，但统一设计系统仍需要作为 F0/F6 的基础任务继续建设。

### 多主题支持情况

当前前端已有一套 CSS token:

- `web/src/styles/variables.scss` 定义 `:root` 颜色、圆角、阴影、侧栏色等变量。
- `global.scss` 和 `App.vue` 使用这些变量。

但尚未形成真正多主题系统:

- 未发现 theme store。
- 未发现 `data-theme`、dark/light/compact、多品牌 token。
- 未发现 Element Plus CSS variable 主题映射。
- 未发现 Header/Profile/Setting 中的主题切换入口。

结论: 当前是单主题 token 化，不是多主题系统。医药 OA 作为业务插件需要统一视觉基础，应新增 Theme Engine 和 Plugin UI Kit。

### 后端能力是否可提供给插件

当前插件平台已具备:

- 插件 manifest: `id`、`version`、`ui_mode`、`level`、`app_id`、`ui_menu`、`config_schema`、`permissions`、`signature` 等。
- 插件生命周期: install、enable、disable、uninstall、preflight、config、page、assets、logs。
- 扩展登记: route、middleware、event、menu、widget、setting snapshot。
- 前端插件挂载: sidebar/top_tab/app home、iframe 页面、locale/token bridge。
- 插件迁移 hook 和签名风险基础。

但对复杂业务插件仍缺关键能力:

- 缺少插件专属数据模型注册、迁移、回滚、审计和卸载策略。
- 缺少插件后端 API 自动注册到主路由、权限、OpenAPI、审计的完整闭环。
- 缺少宿主能力 SDK: 用户、组织、角色、权限、字典、文件、审计、配置、消息、工作流、报表、编号规则。
- 缺少插件间依赖、业务事件总线、跨插件能力调用契约。
- 缺少插件内菜单、页面、按钮、数据权限、业务对象权限的一体化声明。
- 缺少业务插件生成器，从领域模型生成后端、前端、权限、菜单、审计、迁移和测试。

结论: 当前平台可以支撑页面型、配置型、生命周期型插件；要承载医药 OA 这类复杂业务插件，还需要业务插件平台层。

## 医药 OA 完整功能蓝图

用户前面提到的员工管理、OA 审批、进出货管理、客户管理只是医药 OA 的第一层能力。若目标是可演示、可扩展、可二次开发的行业插件，建议按以下业务域规划。

| 业务域 | 必备功能 | 建议功能 | 依赖 Skoll 基建 |
| --- | --- | --- | --- |
| 组织与员工 | 员工档案、部门岗位、任职状态、证照资质、合同附件 | 入转调离、培训记录、健康证/执业证到期提醒、员工数据权限 | 组织、用户、角色、文件、字典、编号规则、审计、通知 |
| OA 审批 | 请假、报销、采购申请、合同审批、用章审批 | 流程设计器、条件分支、会签/或签、抄送、撤回、驳回、加签、转办 | Workflow Engine、Form Builder、Todo、Notification、Audit |
| 公告与协同 | 公告制度、待办中心、站内信、日程提醒 | 会议纪要、任务协同、制度签收、移动端待办入口 | 消息中心、待办中心、权限、附件、富文本 |
| 药品/商品主数据 | 药品档案、规格、剂型、生产厂家、批准文号、有效期规则 | 条码、分类、禁售/停用、温控属性、重点监管品类 | 字典、编码规则、导入导出、审计、数据权限 |
| 供应商管理 | 供应商档案、联系人、资质附件、合作状态 | 资质到期提醒、供应商评级、采购历史、黑名单 | 文件、通知、审计、数据权限、CRM 基础 |
| 客户管理 | 客户档案、联系人、跟进记录、资质附件、合同记录 | 客户分级、销售机会、拜访计划、回款状态、区域归属 | CRM、文件、数据权限、报表、审计 |
| 采购管理 | 采购申请、采购订单、采购入库、退货 | 供应商资质校验、审批联动、到货提醒、采购价格历史 | 工作流、库存、供应商、编号规则、审计 |
| 销售管理 | 销售订单、出库单、销售退货、客户资质校验 | 合同联动、发票记录、回款跟踪、销售漏斗 | CRM、库存、审批、报表、数据权限 |
| 库存管理 | 入库、出库、库存流水、盘点、调拨 | 批号、效期、近效期预警、温控库位、库存上下限 | 库存领域、规则引擎、通知、审计、导入导出 |
| 质量与合规 | 资质管理、质量投诉、药品召回、审计追踪 | GSP/GMP 检查清单、不良反应记录、冷链温湿度记录 | 合规规则、审计追踪、文件、工作流、报表 |
| 合同与档案 | 合同台账、附件归档、审批关联、到期提醒 | 合同模板、变更记录、归档权限、电子签章预留 | 文件中心、审批、通知、审计、数据权限 |
| 财务协同 | 报销、应收、应付、付款申请 | 发票记录、回款计划、费用科目、对账报表 | 工作流、表单、报表、编号规则 |
| 报表看板 | 待办、库存预警、资质到期、采购/销售概览 | 经营分析、客户跟进、员工资质、质量风险 | Report Engine、Query Builder、Dashboard Widgets |

## 框架基建能力矩阵

| 基建能力 | 优先级 | 支撑业务 | 交付物 | 验收标准 |
| --- | --- | --- | --- | --- |
| Theme Engine | P0 | 所有业务插件 | theme store、CSS token、Element Plus variable mapping | light/dark/compact 可切换；刷新后保留；插件页面同步主题 |
| Plugin UI Kit | P0 | 所有业务插件 | PageShell、DataTable、SchemaForm、DetailDrawer、ConfirmAction | 主应用和插件页面视觉、密度、状态、交互一致 |
| Plugin Host SDK | P0 | 所有业务插件 | 前端 SDK + 后端 PluginContext | 插件可稳定调用 auth、user、org、dict、file、audit、config、permission、workflow、report |
| Plugin Data Module Contract | P0 | 主数据、库存、CRM、合规 | data manifest、migration runner、namespace 规则 | 插件可声明表结构、索引、迁移、卸载策略，安装预检可见 |
| Plugin API/OpenAPI Contract | P0 | 所有业务插件 | route registry、OpenAPI 聚合、权限绑定、审计动作声明 | 插件 API 自动进入权限、OpenAPI、审计和前端 client |
| Workflow Engine | P0 | OA 审批、采购、合同、质量 | 流程定义、实例、任务、动作、时间线 | 发起、审批、驳回、撤回、转办、抄送、审计全链路可跑通 |
| Form Builder | P0 | OA 表单、资质、单据 | schema、校验、布局、字典、附件、明细表 | 可配置审批表单、采购单、出入库单、客户资质表单 |
| Todo/Notification Center | P0 | 审批、预警、提醒 | 待办、已办、站内信、提醒规则 | 审批待办、资质到期、库存预警可跳转到目标业务对象 |
| Business Rule Engine | P0 | 医药合规、库存、资质 | 规则定义、规则执行、规则结果 | 批号效期、资质有效期、客户/供应商状态可拦截业务动作 |
| Numbering Engine | P1 | 单据、员工、客户、合同 | 编号规则配置和生成服务 | 采购单、入库单、合同、客户编号可按规则生成且并发安全 |
| Import/Export Engine | P1 | 主数据、库存、客户 | 模板、校验、错误回执、导出任务 | 大批量导入有错误行反馈；导出可审计、可异步 |
| Report/Query Engine | P1 | 看板、经营分析 | 指标定义、查询 API、图表组件 | 库存预警、资质到期、审批效率、客户跟进可配置查询 |
| Print/Template Engine | P2 | 单据、合同、审批 | 打印模板、PDF/HTML 输出 | 入库单、出库单、合同台账可按模板输出 |
| Business Event Bus | P0 | 插件联动 | 事件发布订阅、事务后事件、重试 | 入库完成、资质到期、审批完成可触发通知/报表/审计 |
| Business Plugin Generator | P0 | 快速开发 | 从领域模型生成 API/UI/权限/菜单/迁移/测试 | 生成一个业务对象后可直接 CRUD、授权、审计、构建通过 |

## 新增里程碑建议

当前 M0-M7 基础建设已完成，后续应以“医药 OA 可作为业务插件落地”为牵引，新增 F6-F12。F0-F5 仍保留，但执行顺序应围绕 F6-F12 调整。

| 里程碑 | 目标 | 优先级 | 依赖 | 交付物 | 里程碑验收 |
| --- | --- | --- | --- | --- | --- |
| F6: 业务插件平台底座 | 让插件能安全声明数据、API、权限、UI、宿主能力调用 | P0 | M0-M7,F0 | Host SDK、Data Contract、API Contract、UI Kit、Theme Engine | 一个 sample business plugin 可声明数据表、API、菜单、权限并被主应用加载 |
| F7: 工作流、表单、待办平台 | 支撑 OA 审批和业务单据流转 | P0 | F6 | Workflow Engine、Form Builder、Todo/Notification、审计时间线 | 请假、采购申请、合同审批 3 条流程端到端跑通 |
| F8: 医药主数据插件 | 建立医药 OA 的员工、药品、供应商、客户、仓库基础 | P0 | F6,F7 | pharma_oa 插件骨架和主数据模块 | 员工/药品/供应商/客户/仓库 CRUD、权限、审计、导入导出可验收 |
| F9: 医药进销存插件能力 | 建立采购、销售、库存、批号、效期、库存预警 | P0 | F8,F7 | 采购单、销售单、入库、出库、盘点、调拨、库存流水 | 采购入库到销售出库闭环跑通；批号效期和库存预警可触发 |
| F10: 医药 OA 协同与合规 | 建立公告、合同、资质、质量投诉、召回和合规审计 | P0 | F7,F8,F9 | 合规质量模块、合同档案、资质提醒、质量流程 | 供应商/客户/员工资质到期可提醒；质量投诉和召回可走审批 |
| F11: 医药 CRM 与经营看板 | 建立客户跟进、销售机会、回款、经营报表 | P1 | F8,F9,F10 | CRM 模块、报表指标、Dashboard widgets | 客户跟进、销售漏斗、库存预警、资质到期、审批效率可视化 |
| F12: 医药 OA 端到端样板验收 | 形成可演示、可二次开发、可作为开源示例的行业插件 | P0 | F6-F11 | demo 数据、验收脚本、E2E smoke、行业 README | 新员工入职、采购申请、采购入库、销售出库、资质预警、客户跟进全链路通过 |

## 方向结论

医药 OA 不应直接在主应用中散落实现页面，而应作为业务插件样板牵引 Skoll 框架升级。下一阶段最关键的顺序是:

1. F0/F6: 统一 UI、主题和业务插件底座。
2. F7: 建立工作流、动态表单、待办通知。
3. F8: 建立医药主数据插件。
4. F9: 建立进销存、批号、效期、库存预警。
5. F10: 建立合规质量、合同档案、资质提醒。
6. F11/F12: 建立 CRM、报表和端到端行业样板验收。

按这个路线，Skoll 的“超越 gin-vue-admin”不只是功能清单更多，而是形成可复用、可扩展、能承载行业业务插件的开源框架能力。
