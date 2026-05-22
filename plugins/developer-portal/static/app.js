(function () {
  const TOKEN_KEY = "skoll.auth.token";
  const SUPPORTED_LOCALES = ["zh-CN", "en-US"];

  const i18n = {
    "zh-CN": {
      "page.title": "Skoll 开发者门户",
      "hero.eyebrow": "Skoll 开发工作台",
      "hero.title": "开发者门户",
      "hero.subtitle": "与管理后台风格统一。通过插件模块演进开发能力，避免与核心管理页面耦合。",
      "hero.tag.pluginPowered": "插件化承载",
      "hero.tag.removable": "生产可移除",
      "module.aria": "开发模块",
      "module.pluginDev.title": "插件开发",
      "module.pluginDev.desc": "在白名单目录快速创建插件，并批量校验清单。",
      "module.apiTools.title": "API 工具",
      "module.apiTools.desc": "携带当前登录态快速调试后端接口。",
      "module.opsLab.title": "运维实验室",
      "module.opsLab.desc": "预留用于环境诊断与部署辅助。",
      "pluginDev.flowHint": "推荐流程：先创建插件，再从列表选择插件进入配置编辑，最后执行构建与发布操作。",
      "step.scaffold": "1. 创建插件",
      "step.select": "2. 选择插件",
      "step.configure": "3. 配置编辑",
      "step.release": "4. 构建发布",
      "step.state.notStarted": "未开始",
      "step.state.inProgress": "进行中",
      "step.state.completed": "已完成",
      "pluginDev.scaffold.title": "创建插件",
      "pluginDev.scaffold.hint": "调用 <code>POST /skoll/v1/plugins/dev/scaffold</code>，要求 <strong>super_admin</strong>。",
      "field.pluginsRoot": "pluginsRoot",
      "field.pluginId": "pluginId",
      "field.pluginName": "pluginName",
      "field.appId": "appId（可选）",
      "field.mode": "mode",
      "action.createScaffold": "创建插件",
      "pluginDev.validate.title": "批量校验",
      "pluginDev.validate.hint": "调用 <code>POST /skoll/v1/plugins/dev/validate-all</code>，使用同一目录。",
      "action.validateAll": "批量校验",
      "pluginDev.projects.title": "开发中插件",
      "pluginDev.projects.hint": "支持独立仓库目录的项目扫描、预览、打包、删除（可选删目录）。",
      "action.loadProjects": "加载项目列表",
      "field.projectSearch": "搜索插件",
      "field.projectStatus": "状态筛选",
      "filter.all": "全部",
      "filter.ok": "正常",
      "filter.error": "异常",
      "tab.pluginManifest": "配置编辑",
      "tab.pluginAutomation": "发布与自动化",
      "tab.manifestVisual": "可视化",
      "tab.manifestYaml": "YAML 文本",
      "action.selectPlugin": "选择",
      "action.editManifestFromList": "编辑配置",
      "action.preview": "预览",
      "action.package": "打包",
      "action.remove": "删除",
      "action.removeAndDelete": "删除并清理目录",
      "status.loadingProjects": "正在加载项目列表...",
      "status.projectsLoaded": "项目列表已更新",
      "status.packaging": "正在打包...",
      "status.packaged": "打包完成",
      "status.removing": "正在删除项目...",
      "status.removed": "项目删除完成",
      "status.removeFailed": "删除失败",
      "status.packFailed": "打包失败",
      "status.projectHints": "构建/发布建议已写入结果面板",
      "status.loadingManifest": "正在加载 manifest...",
      "status.manifestLoaded": "manifest 已加载",
      "status.validatingManifest": "正在校验 manifest...",
      "status.manifestValid": "manifest 校验通过",
      "status.savingManifest": "正在保存 manifest...",
      "status.manifestSaved": "manifest 保存成功",
      "status.manifestFormSynced": "表单已生成 manifest 文本",
      "status.manifestFormParsed": "manifest 文本已填充到表单",
      "status.manifestFormParseFailed": "manifest 文本解析失败，请检查格式",
      "status.manifestInvalidRequired": "manifest 必填字段不完整（id/name/version）",
      "status.manifestInvalidApiVersion": "api_version 格式无效，应为 v数字（例如 v1）",
      "status.manifestInvalidMigration": "migration_version 格式无效，应为 v1.2.3 或 1.2.3",
      "status.manifestInvalidCompat": "api_version 已设置时 compatibility_skoll 不能为空",
      "status.manifestInvalidLevel": "level 与 app_id 不匹配：system 不能填 app_id，app 必须提供合法 app_id",
      "status.manifestInvalidOpenMode": "standalone 模式下必须 ui_nav_position=none 且 ui_tab_mode=disabled",
      "status.manifestInvalidLocales": "非 backend_only 模式必须至少一个合法 i18n_locales（如 zh-CN）",
      "status.unsavedManifestConfirm": "当前 Manifest 有未保存修改，确定放弃并继续吗？",
      "status.runningPipeline": "正在执行流水线...",
      "status.pipelineDone": "流水线执行完成",
      "status.rolloutApplying": "正在应用灰度...",
      "status.rolloutApplied": "灰度已应用",
      "status.rollbacking": "正在回滚...",
      "status.rollbackDone": "回滚完成",
      "status.creatingReleaseOrder": "正在创建发布单...",
      "status.releaseOrderCreated": "发布单创建成功",
      "status.listingReleaseOrders": "正在查询发布单...",
      "status.releaseOrdersListed": "发布单查询完成",
      "status.approvingReleaseOrder": "正在审批发布单...",
      "status.releaseOrderApproved": "发布单已审批通过",
      "status.rejectingReleaseOrder": "正在驳回发布单...",
      "status.releaseOrderRejected": "发布单已驳回",
      "table.plugin": "插件",
      "table.root": "目录",
      "table.mode": "模式",
      "table.status": "状态",
      "table.actions": "操作",
      "project.mode.workspace": "工作区",
      "project.mode.repository": "独立仓库",
      "project.status.ok": "正常",
      "project.status.invalid": "无效",
      "project.status.failed": "失败",
      "project.status.error": "异常",
      "project.status.unknown": "未知",
      "pluginDev.manifest.title": "Manifest 编辑器",
      "pluginDev.manifest.hint": "针对当前选中插件编辑 plugin.yaml，并支持保存前实时校验。",
      "pluginDev.selected.title": "当前插件",
      "pluginDev.selected.empty": "尚未选择插件，请先在列表中选择。",
      "pluginDev.selected.meta": "{id} @ {root} [{mode}]",
      "pluginDev.automation.title": "自动化操作",
      "pluginDev.automation.hint": "针对当前选中插件执行流水线、灰度和发布审批操作。",
      "pluginDev.pipeline.title": "一键流水线",
      "pluginDev.pipeline.hint": "执行 manifest 校验 + 制品打包。",
      "pluginDev.rollout.title": "灰度与回滚",
      "pluginDev.rollout.hint": "设置 rollout 百分比并支持回滚到上一个值。",
      "field.manifestPluginId": "manifest pluginId",
      "field.manifestYaml": "plugin.yaml",
      "manifestVisual.basic.title": "基础信息",
      "manifestVisual.ui.title": "UI 配置",
      "manifestVisual.id": "id",
      "manifestVisual.name": "name",
      "manifestVisual.nameZhCN": "name_zh_cn",
      "manifestVisual.nameEnUS": "name_en_us",
      "manifestVisual.version": "version",
      "manifestVisual.description": "description",
      "manifestVisual.apiVersion": "api_version",
      "manifestVisual.compat": "compatibility_skoll",
      "manifestVisual.migration": "migration_version",
      "manifestVisual.level": "level",
      "manifestVisual.appId": "app_id",
      "manifestVisual.uiMode": "ui_mode",
      "manifestVisual.mountPolicy": "mount_policy",
      "manifestVisual.navPosition": "ui_nav_position",
      "manifestVisual.openMode": "ui_open_mode",
      "manifestVisual.tabMode": "ui_tab_mode",
      "manifestVisual.frontendEntry": "frontend_entry",
      "manifestVisual.locales": "i18n_locales（逗号分隔）",
      "manifestVisual.permissions": "permissions（可勾选）",
      "manifestVisual.permissionsHint": "支持筛选与多来源合并：当前插件、框架内置、其他插件与自定义。",
      "manifestVisual.permissionsSource": "选项来源：当前插件 + 框架内置 + 其他插件 + 自定义",
      "manifestVisual.permissionsMeta": "来源：动态聚合目录。勾选项会写入 permissions，自定义框用于补充目录外权限。",
      "manifestVisual.permissionsSearch": "筛选权限",
      "manifestVisual.permissionsSearchPlaceholder": "例如 plugin.",
      "manifestVisual.permissionsGroupSelf": "当前插件",
      "manifestVisual.permissionsGroupFramework": "框架内置",
      "manifestVisual.permissionsGroupPlugins": "其他插件",
      "manifestVisual.permissionsGroupPluginPrefix": "插件",
      "manifestVisual.permissionsGroupEmpty": "当前筛选条件下无可选权限。",
      "status.permissionsCatalogLoadFailed": "权限目录加载失败，已保留当前勾选与自定义输入",
      "manifestVisual.permissionsMeaning": "权限含义",
      "manifestVisual.permissionsMeaningFallback": "暂无可展示权限，加载或输入后会在此显示。",
      "manifestVisual.permissionsEnableCustom": "需要补充自定义权限",
      "manifestVisual.permissionsCustom": "自定义权限（逗号分隔）",
      "field.pipelinePluginId": "pipeline pluginId",
      "field.rolloutPluginId": "rollout pluginId",
      "field.rolloutPercent": "rolloutPercent (0-100)",
      "action.loadManifest": "加载",
      "action.validateManifest": "校验",
      "action.saveManifest": "保存",
      "action.manifestParseToForm": "文本填充表单",
      "action.manifestApplyForm": "表单生成文本",
      "action.runPipeline": "执行流水线",
      "action.applyRollout": "应用灰度",
      "action.rollbackRollout": "回滚",
      "pluginDev.releaseOrder.title": "发布审批流",
      "pluginDev.releaseOrder.hint": "创建发布单并执行审批/驳回，返回审计字段用于追踪。",
      "field.releasePluginId": "release pluginId",
      "field.releaseVersion": "releaseVersion",
      "field.releaseOrderId": "orderId",
      "field.releaseChangelog": "changelog / review comment",
      "action.createReleaseOrder": "创建发布单",
      "action.listReleaseOrders": "查询发布单",
      "action.approveReleaseOrder": "审批通过",
      "action.rejectReleaseOrder": "审批驳回",
      "apiTools.title": "API 工具",
      "apiTools.hint": "在插件页直接携带当前 token 发起请求。",
      "field.method": "method",
      "field.path": "path",
      "field.jsonBody": "JSON body（GET 可留空）",
      "action.sendRequest": "发送请求",
      "opsLab.title": "运维实验室",
      "opsLab.placeholder": "规划中：部署检查清单、就绪性检测、回滚预演。",
      "result.title": "结果",
      "diagnostics.aria": "语言诊断",
      "diagnostics.title": "语言链路诊断",
      "diagnostics.queryLocale": "query.locale",
      "diagnostics.hostLocale": "宿主 locale",
      "diagnostics.storageLocale": "localStorage",
      "diagnostics.htmlLang": "html lang",
      "diagnostics.allowedLocales": "插件支持语言",
      "diagnostics.activeLocale": "当前生效",
      "diagnostics.lastSource": "最后来源",
      "status.ready": "就绪。",
      "status.onlySuperAdmin": "仅 super_admin 可使用开发操作。",
      "status.requirePluginIdAndName": "pluginId 和 pluginName 不能为空",
      "status.creatingScaffold": "正在创建插件...",
      "status.scaffoldCreated": "插件创建成功",
      "status.scaffoldCreatedAt": "插件创建成功，目录：",
      "status.validating": "正在校验清单...",
      "status.validateDone": "校验完成",
      "status.devPortalDisabled": "开发者脚手架接口不可用（404）。请在后端设置 SKOLL_DEV_PORTAL_ENABLED=true 后重启服务。",
      "status.pathRequired": "path 不能为空",
      "status.invalidJson": "JSON body 格式无效",
      "status.sending": "正在发送",
      "status.requestSuccess": "请求成功",
      "placeholder.pluginId": "例如 crm-order",
      "placeholder.projectSearch": "例如 developer-portal",
      "placeholder.pluginName": "例如 CRM 订单",
      "placeholder.appId": "例如 crm",
      "placeholder.apiBody": "例如 {\"key\":\"value\"}",
      "action.refreshDiagnostics": "刷新诊断",
      "action.expandResult": "展开结果",
      "action.collapseResult": "收起结果",
      "common.empty": "暂无数据",
      "common.emptyFiltered": "筛选后无匹配数据"
    },
    "en-US": {
      "page.title": "Skoll Developer Portal",
      "hero.eyebrow": "Skoll Developer Workspace",
      "hero.title": "Developer Portal",
      "hero.subtitle": "Unified with Skoll Admin style. Add and evolve developer capabilities through plugin modules without coupling to core management pages.",
      "hero.tag.pluginPowered": "Plugin Powered",
      "hero.tag.removable": "Production Removable",
      "module.aria": "Developer Modules",
      "module.pluginDev.title": "Plugin Dev",
      "module.pluginDev.desc": "Create plugins in allowlisted roots and validate manifests in bulk.",
      "module.apiTools.title": "API Tools",
      "module.apiTools.desc": "Request playground with auth token passthrough for rapid backend verification.",
      "module.opsLab.title": "Ops Lab",
      "module.opsLab.desc": "Reserved for env diagnostics and deployment helpers.",
      "pluginDev.flowHint": "Recommended flow: create plugin first, select plugin from list, edit configuration, then run build and release operations.",
      "step.scaffold": "1. Scaffold",
      "step.select": "2. Select Plugin",
      "step.configure": "3. Configure",
      "step.release": "4. Release",
      "step.state.notStarted": "Not started",
      "step.state.inProgress": "In progress",
      "step.state.completed": "Completed",
      "pluginDev.scaffold.title": "Create Plugin",
      "pluginDev.scaffold.hint": "Calls <code>POST /skoll/v1/plugins/dev/scaffold</code>. Requires <strong>super_admin</strong>.",
      "field.pluginsRoot": "pluginsRoot",
      "field.pluginId": "pluginId",
      "field.pluginName": "pluginName",
      "field.appId": "appId (optional)",
      "field.mode": "mode",
      "action.createScaffold": "Create Plugin",
      "pluginDev.validate.title": "Validate All",
      "pluginDev.validate.hint": "Calls <code>POST /skoll/v1/plugins/dev/validate-all</code> against the same root.",
      "action.validateAll": "Validate All",
      "pluginDev.projects.title": "In-Progress Plugins",
      "pluginDev.projects.hint": "Scan independent repo roots and perform preview/package/remove actions.",
      "action.loadProjects": "Load Projects",
      "field.projectSearch": "Search Plugin",
      "field.projectStatus": "Status Filter",
      "filter.all": "All",
      "filter.ok": "Healthy",
      "filter.error": "Error",
      "tab.pluginManifest": "Config Editor",
      "tab.pluginAutomation": "Release & Automation",
      "tab.manifestVisual": "Visual",
      "tab.manifestYaml": "YAML",
      "action.selectPlugin": "Select",
      "action.editManifestFromList": "Edit Config",
      "action.preview": "Preview",
      "action.package": "Package",
      "action.remove": "Remove",
      "action.removeAndDelete": "Remove + Delete Dir",
      "status.loadingProjects": "Loading projects...",
      "status.projectsLoaded": "Projects refreshed",
      "status.packaging": "Packaging...",
      "status.packaged": "Package completed",
      "status.removing": "Removing project...",
      "status.removed": "Project removed",
      "status.removeFailed": "Remove failed",
      "status.packFailed": "Package failed",
      "status.projectHints": "Build/publish hints written to result panel",
      "status.loadingManifest": "Loading manifest...",
      "status.manifestLoaded": "Manifest loaded",
      "status.validatingManifest": "Validating manifest...",
      "status.manifestValid": "Manifest valid",
      "status.savingManifest": "Saving manifest...",
      "status.manifestSaved": "Manifest saved",
      "status.manifestFormSynced": "Manifest text generated from form",
      "status.manifestFormParsed": "Manifest text parsed into form",
      "status.manifestFormParseFailed": "Failed to parse manifest text",
      "status.manifestInvalidRequired": "Manifest required fields are missing (id/name/version)",
      "status.manifestInvalidApiVersion": "Invalid api_version format; expected v<number> (e.g. v1)",
      "status.manifestInvalidMigration": "Invalid migration_version; expected v1.2.3 or 1.2.3",
      "status.manifestInvalidCompat": "compatibility_skoll is required when api_version is set",
      "status.manifestInvalidLevel": "level and app_id mismatch: system must not have app_id, app requires valid app_id",
      "status.manifestInvalidOpenMode": "standalone requires ui_nav_position=none and ui_tab_mode=disabled",
      "status.manifestInvalidLocales": "Non-backend_only mode requires at least one valid i18n locale (e.g. zh-CN)",
      "status.unsavedManifestConfirm": "Unsaved manifest changes detected. Discard and continue?",
      "status.runningPipeline": "Running pipeline...",
      "status.pipelineDone": "Pipeline completed",
      "status.rolloutApplying": "Applying rollout...",
      "status.rolloutApplied": "Rollout applied",
      "status.rollbacking": "Rolling back...",
      "status.rollbackDone": "Rollback completed",
      "status.creatingReleaseOrder": "Creating release order...",
      "status.releaseOrderCreated": "Release order created",
      "status.listingReleaseOrders": "Listing release orders...",
      "status.releaseOrdersListed": "Release orders loaded",
      "status.approvingReleaseOrder": "Approving release order...",
      "status.releaseOrderApproved": "Release order approved",
      "status.rejectingReleaseOrder": "Rejecting release order...",
      "status.releaseOrderRejected": "Release order rejected",
      "table.plugin": "Plugin",
      "table.root": "Root",
      "table.mode": "Mode",
      "table.status": "Status",
      "table.actions": "Actions",
      "project.mode.workspace": "Workspace",
      "project.mode.repository": "Repository",
      "project.status.ok": "Healthy",
      "project.status.invalid": "Invalid",
      "project.status.failed": "Failed",
      "project.status.error": "Error",
      "project.status.unknown": "Unknown",
      "pluginDev.manifest.title": "Manifest Editor",
      "pluginDev.manifest.hint": "Edit plugin.yaml for the selected plugin with save-time validation.",
      "pluginDev.selected.title": "Selected Plugin",
      "pluginDev.selected.empty": "No plugin selected. Choose one from the list first.",
      "pluginDev.selected.meta": "{id} @ {root} [{mode}]",
      "pluginDev.automation.title": "Automation",
      "pluginDev.automation.hint": "Run pipeline, rollout, and release approvals for the selected plugin.",
      "pluginDev.pipeline.title": "One-Click Pipeline",
      "pluginDev.pipeline.hint": "Run manifest validation and package artifact.",
      "pluginDev.rollout.title": "Gray Rollout & Rollback",
      "pluginDev.rollout.hint": "Set rollout percentage and rollback to previous value.",
      "field.manifestPluginId": "manifest pluginId",
      "field.manifestYaml": "plugin.yaml",
      "manifestVisual.basic.title": "Basic",
      "manifestVisual.ui.title": "UI",
      "manifestVisual.id": "id",
      "manifestVisual.name": "name",
      "manifestVisual.nameZhCN": "name_zh_cn",
      "manifestVisual.nameEnUS": "name_en_us",
      "manifestVisual.version": "version",
      "manifestVisual.description": "description",
      "manifestVisual.apiVersion": "api_version",
      "manifestVisual.compat": "compatibility_skoll",
      "manifestVisual.migration": "migration_version",
      "manifestVisual.level": "level",
      "manifestVisual.appId": "app_id",
      "manifestVisual.uiMode": "ui_mode",
      "manifestVisual.mountPolicy": "mount_policy",
      "manifestVisual.navPosition": "ui_nav_position",
      "manifestVisual.openMode": "ui_open_mode",
      "manifestVisual.tabMode": "ui_tab_mode",
      "manifestVisual.frontendEntry": "frontend_entry",
      "manifestVisual.locales": "i18n_locales (comma-separated)",
      "manifestVisual.permissions": "permissions (selectable)",
      "manifestVisual.permissionsHint": "Supports filtering and merged sources: current plugin, framework defaults, other plugins, and custom values.",
      "manifestVisual.permissionsSource": "Source: current plugin + framework defaults + other plugins + custom",
      "manifestVisual.permissionsMeta": "Source: dynamic catalog aggregation. Checked items are written to permissions; custom input appends out-of-catalog values.",
      "manifestVisual.permissionsSearch": "Filter Permissions",
      "manifestVisual.permissionsSearchPlaceholder": "e.g. plugin.",
      "manifestVisual.permissionsGroupSelf": "Current Plugin",
      "manifestVisual.permissionsGroupFramework": "Framework Defaults",
      "manifestVisual.permissionsGroupPlugins": "Other Plugins",
      "manifestVisual.permissionsGroupPluginPrefix": "Plugin",
      "manifestVisual.permissionsGroupEmpty": "No permissions match current filter.",
      "status.permissionsCatalogLoadFailed": "Failed to load permission catalog. Existing selections and custom values are kept.",
      "manifestVisual.permissionsMeaning": "Permission Meaning",
      "manifestVisual.permissionsMeaningFallback": "No permissions to display yet. Load manifest or input permissions to see details.",
      "manifestVisual.permissionsEnableCustom": "Add custom permissions",
      "manifestVisual.permissionsCustom": "custom permissions (comma-separated)",
      "field.pipelinePluginId": "pipeline pluginId",
      "field.rolloutPluginId": "rollout pluginId",
      "field.rolloutPercent": "rolloutPercent (0-100)",
      "action.loadManifest": "Load",
      "action.validateManifest": "Validate",
      "action.saveManifest": "Save",
      "action.manifestParseToForm": "Parse Text To Form",
      "action.manifestApplyForm": "Apply Form To Text",
      "action.runPipeline": "Run Pipeline",
      "action.applyRollout": "Apply Rollout",
      "action.rollbackRollout": "Rollback",
      "pluginDev.releaseOrder.title": "Release Approval Flow",
      "pluginDev.releaseOrder.hint": "Create release orders, then approve/reject with audit fields.",
      "field.releasePluginId": "release pluginId",
      "field.releaseVersion": "releaseVersion",
      "field.releaseOrderId": "orderId",
      "field.releaseChangelog": "changelog / review comment",
      "action.createReleaseOrder": "Create Order",
      "action.listReleaseOrders": "List Orders",
      "action.approveReleaseOrder": "Approve",
      "action.rejectReleaseOrder": "Reject",
      "apiTools.title": "API Tools",
      "apiTools.hint": "Send authenticated requests from this plugin page using current login token.",
      "field.method": "method",
      "field.path": "path",
      "field.jsonBody": "JSON body (optional for GET)",
      "action.sendRequest": "Send Request",
      "opsLab.title": "Ops Lab",
      "opsLab.placeholder": "Planned: deployment checklist, readiness checks, and rollback previews.",
      "result.title": "Result",
      "diagnostics.aria": "Locale Diagnostics",
      "diagnostics.title": "Locale Diagnostics",
      "diagnostics.queryLocale": "query.locale",
      "diagnostics.hostLocale": "host locale",
      "diagnostics.storageLocale": "localStorage",
      "diagnostics.htmlLang": "html lang",
      "diagnostics.allowedLocales": "plugin locales",
      "diagnostics.activeLocale": "active locale",
      "diagnostics.lastSource": "last source",
      "status.ready": "Ready.",
      "status.onlySuperAdmin": "Only super_admin can use developer actions.",
      "status.requirePluginIdAndName": "pluginId and pluginName are required",
      "status.creatingScaffold": "Creating plugin...",
      "status.scaffoldCreated": "Plugin created",
      "status.scaffoldCreatedAt": "Plugin created at:",
      "status.validating": "Validating manifests...",
      "status.validateDone": "Validation completed",
      "status.devPortalDisabled": "Developer scaffold endpoints are unavailable (404). Set SKOLL_DEV_PORTAL_ENABLED=true and restart backend.",
      "status.pathRequired": "path is required",
      "status.invalidJson": "JSON body is invalid",
      "status.sending": "Sending",
      "status.requestSuccess": "Request success",
      "placeholder.pluginId": "e.g. crm-order",
      "placeholder.projectSearch": "e.g. developer-portal",
      "placeholder.pluginName": "e.g. CRM Order",
      "placeholder.appId": "e.g. crm",
      "placeholder.apiBody": "e.g. {\"key\":\"value\"}",
      "action.refreshDiagnostics": "Refresh Diagnostics",
      "action.expandResult": "Expand Result",
      "action.collapseResult": "Collapse Result",
      "common.empty": "No data",
      "common.emptyFiltered": "No items match current filters"
    }
  };

  let currentLocale = "zh-CN";

  const statusEl = document.getElementById("status");
  const resultEl = document.getElementById("result");
  const resultAsideEl = document.getElementById("resultAside");
  const resultSummaryEl = document.getElementById("resultSummary");
  const btnResultToggle = document.getElementById("btnResultToggle");
  const rootEl = document.getElementById("pluginsRoot");
  const pluginIdEl = document.getElementById("pluginId");
  const pluginNameEl = document.getElementById("pluginName");
  const appIdEl = document.getElementById("appId");
  const scaffoldModeEl = document.getElementById("scaffoldMode");
  const rootOptionsEl = document.getElementById("pluginsRootOptions");
  const btnScaffold = document.getElementById("btnScaffold");
  const btnValidateAll = document.getElementById("btnValidateAll");
  const btnLoadProjects = document.getElementById("btnLoadProjects");
  const projectSearchEl = document.getElementById("projectSearch");
  const projectStatusFilterEl = document.getElementById("projectStatusFilter");
  const projectRowsEl = document.getElementById("projectRows");
  const flowStepButtons = document.querySelectorAll(".flow-step-btn");
  const flowToggleButtons = document.querySelectorAll(".flow-toggle");
  const btnDevTabManifest = document.getElementById("btnDevTabManifest");
  const btnDevTabAutomation = document.getElementById("btnDevTabAutomation");
  const devPaneManifest = document.getElementById("devPaneManifest");
  const devPaneAutomation = document.getElementById("devPaneAutomation");
  const selectedPluginMetaEl = document.getElementById("selectedPluginMeta");
  const btnSelectedEditManifest = document.getElementById("btnSelectedEditManifest");
  const btnSelectedRunPipeline = document.getElementById("btnSelectedRunPipeline");
  const btnSelectedListReleaseOrders = document.getElementById("btnSelectedListReleaseOrders");
  const manifestPluginIdEl = document.getElementById("manifestPluginId");
  const btnManifestTabVisual = document.getElementById("btnManifestTabVisual");
  const btnManifestTabYaml = document.getElementById("btnManifestTabYaml");
  const manifestPaneVisual = document.getElementById("manifestPaneVisual");
  const manifestPaneYaml = document.getElementById("manifestPaneYaml");
  const manifestYamlEl = document.getElementById("manifestYaml");
  const manifestFieldIdEl = document.getElementById("manifestFieldId");
  const manifestFieldNameEl = document.getElementById("manifestFieldName");
  const manifestFieldNameZhCNEl = document.getElementById("manifestFieldNameZhCN");
  const manifestFieldNameEnUSEl = document.getElementById("manifestFieldNameEnUS");
  const manifestFieldVersionEl = document.getElementById("manifestFieldVersion");
  const manifestFieldDescriptionEl = document.getElementById("manifestFieldDescription");
  const manifestFieldApiVersionEl = document.getElementById("manifestFieldApiVersion");
  const manifestFieldCompatEl = document.getElementById("manifestFieldCompat");
  const manifestFieldMigrationEl = document.getElementById("manifestFieldMigration");
  const manifestFieldLevelEl = document.getElementById("manifestFieldLevel");
  const manifestFieldAppIdEl = document.getElementById("manifestFieldAppId");
  const manifestFieldUiModeEl = document.getElementById("manifestFieldUiMode");
  const manifestFieldMountPolicyEl = document.getElementById("manifestFieldMountPolicy");
  const manifestFieldNavPositionEl = document.getElementById("manifestFieldNavPosition");
  const manifestFieldOpenModeEl = document.getElementById("manifestFieldOpenMode");
  const manifestFieldTabModeEl = document.getElementById("manifestFieldTabMode");
  const manifestFieldFrontendEntryEl = document.getElementById("manifestFieldFrontendEntry");
  const manifestFieldLocalesEl = document.getElementById("manifestFieldLocales");
  const manifestPermissionSearchEl = document.getElementById("manifestPermissionSearch");
  const manifestPermissionPresetEl = document.getElementById("manifestPermissionPreset");
  const manifestPermissionEnableCustomEl = document.getElementById("manifestPermissionEnableCustom");
  const manifestPermissionsCustomWrapEl = document.getElementById("manifestPermissionsCustomWrap");
  const manifestFieldPermissionsCustomEl = document.getElementById("manifestFieldPermissionsCustom");
  const manifestPermissionsMetaEl = document.getElementById("manifestPermissionsMeta");
  const manifestPermissionCatalogEl = document.getElementById("manifestPermissionCatalog");
  const btnManifestLoad = document.getElementById("btnManifestLoad");
  const btnManifestValidate = document.getElementById("btnManifestValidate");
  const btnManifestSave = document.getElementById("btnManifestSave");
  const manifestSaveFeedbackEl = document.getElementById("manifestSaveFeedback");
  const pipelinePluginIdEl = document.getElementById("pipelinePluginId");
  const btnRunPipeline = document.getElementById("btnRunPipeline");
  const rolloutPluginIdEl = document.getElementById("rolloutPluginId");
  const rolloutPercentEl = document.getElementById("rolloutPercent");
  const btnApplyRollout = document.getElementById("btnApplyRollout");
  const btnRollbackRollout = document.getElementById("btnRollbackRollout");
  const releasePluginIdEl = document.getElementById("releasePluginId");
  const releaseVersionEl = document.getElementById("releaseVersion");
  const releaseOrderIdEl = document.getElementById("releaseOrderId");
  const releaseChangelogEl = document.getElementById("releaseChangelog");
  const btnCreateReleaseOrder = document.getElementById("btnCreateReleaseOrder");
  const btnListReleaseOrders = document.getElementById("btnListReleaseOrders");
  const btnApproveReleaseOrder = document.getElementById("btnApproveReleaseOrder");
  const btnRejectReleaseOrder = document.getElementById("btnRejectReleaseOrder");
  const btnQuickHealth = document.getElementById("btnQuickHealth");
  const btnQuickMe = document.getElementById("btnQuickMe");
  const apiMethodEl = document.getElementById("apiMethod");
  const apiPathEl = document.getElementById("apiPath");
  const apiBodyEl = document.getElementById("apiBody");
  const btnApiSend = document.getElementById("btnApiSend");
  const btnRefreshDiagnostics = document.getElementById("btnRefreshDiagnostics");
  const diagLocaleQueryEl = document.getElementById("diag-locale-query");
  const diagLocaleHostEl = document.getElementById("diag-locale-host");
  const diagLocaleStorageEl = document.getElementById("diag-locale-storage");
  const diagLocaleHtmlEl = document.getElementById("diag-locale-html");
  const diagLocaleAllowedEl = document.getElementById("diag-locale-allowed");
  const diagLocaleActiveEl = document.getElementById("diag-locale-active");
  const diagLocaleSourceEl = document.getElementById("diag-locale-source");
  let allowActions = false;
  let projectItems = [];
  let selectedProjectKey = "";
  let projectSearchText = "";
  let projectStatusFilter = "all";
  let activeDevTab = "manifest";
  let activeManifestTab = "visual";
  let manifestDirty = false;
  let suspendDirtyTracking = false;
  let suspendManifestSync = false;
  let permissionSearchText = "";
  let permissionCatalogData = {
    self: [],
    framework: [],
    plugins: []
  };
  let yamlSyncTimer = null;
  let formSyncTimer = null;
  let resultExpanded = false;
  const PERMISSION_DESCRIPTIONS = {
    "oa.read": {
      "zh-CN": "读取 OA 应用相关数据",
      "en-US": "Read data under OA application"
    },
    "oa.write": {
      "zh-CN": "写入 OA 应用相关数据",
      "en-US": "Write data under OA application"
    },
    "plugin.read": {
      "zh-CN": "查看插件信息与清单",
      "en-US": "View plugin metadata and manifests"
    },
    "plugin.manage": {
      "zh-CN": "安装、启停、配置插件",
      "en-US": "Install, enable/disable, and configure plugins"
    },
    "plugin.write": {
      "zh-CN": "修改插件相关内容",
      "en-US": "Modify plugin-related content"
    },
    "release.manage": {
      "zh-CN": "创建和推进发布流程",
      "en-US": "Create and operate release workflows"
    },
    "release.approve": {
      "zh-CN": "审批发布单",
      "en-US": "Approve release orders"
    },
    "admin.read": {
      "zh-CN": "读取管理后台敏感数据",
      "en-US": "Read sensitive admin-console data"
    },
    "admin.write": {
      "zh-CN": "修改管理后台关键配置",
      "en-US": "Modify critical admin-console configuration"
    }
  };

  const flowProgress = {
    scaffold: "not-started",
    select: "not-started",
    configure: "not-started",
    release: "not-started"
  };

  function updateResultDrawer() {
    if (!resultAsideEl) {
      return;
    }
    resultAsideEl.classList.toggle("collapsed", !resultExpanded);
    if (btnResultToggle) {
      btnResultToggle.textContent = resultExpanded ? t("action.collapseResult") : t("action.expandResult");
    }
  }

  function setFlowStepState(stepKey, state) {
    const normalizedState = state === "completed" ? "completed" : (state === "in-progress" ? "in-progress" : "not-started");
    flowProgress[stepKey] = normalizedState;
    const stateI18nKey = normalizedState === "completed"
      ? "step.state.completed"
      : (normalizedState === "in-progress" ? "step.state.inProgress" : "step.state.notStarted");
    flowStepButtons.forEach(function (btn) {
      if (btn.getAttribute("data-step-key") !== stepKey) {
        return;
      }
      btn.classList.remove("state-not-started", "state-in-progress", "state-completed");
      btn.classList.add("state-" + normalizedState);
      btn.title = t(stateI18nKey);
      const originalText = String(btn.textContent || "").trim();
      btn.setAttribute("aria-label", originalText ? (originalText + " - " + t(stateI18nKey)) : t(stateI18nKey));
    });
  }

  function refreshFlowProgressFromState() {
    if (Array.isArray(projectItems) && projectItems.length > 0) {
      setFlowStepState("select", selectedProjectKey ? "completed" : "in-progress");
    } else {
      setFlowStepState("select", "not-started");
    }
    if (manifestPluginIdEl && String(manifestPluginIdEl.value || "").trim()) {
      setFlowStepState("configure", manifestDirty ? "in-progress" : "completed");
    } else {
      setFlowStepState("configure", "not-started");
    }
  }

  function setFlowSectionCollapsed(sectionEl, collapsed) {
    if (!sectionEl) {
      return;
    }
    sectionEl.classList.toggle("collapsed", Boolean(collapsed));
    const btn = sectionEl.querySelector(".flow-toggle");
    if (btn) {
      btn.textContent = collapsed ? "+" : "-";
    }
  }

  function expandFlowSectionById(sectionId) {
    const section = document.getElementById(sectionId);
    if (!section) {
      return;
    }
    setFlowSectionCollapsed(section, false);
    if (section.scrollIntoView) {
      section.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }

  function resolveLocale(rawLocale) {
    const normalized = String(rawLocale || "").trim();
    if (SUPPORTED_LOCALES.indexOf(normalized) >= 0) {
      return normalized;
    }
    const lower = normalized.toLowerCase();
    if (lower.startsWith("zh")) {
      return "zh-CN";
    }
    return "en-US";
  }

  function detectInitialLocale() {
    try {
      const fromQuery = new URLSearchParams(window.location.search).get("locale");
      if (fromQuery) {
        return resolveLocale(fromQuery);
      }
    } catch (err) {
      // Ignore malformed URL search params.
    }
    if (typeof window !== "undefined" && typeof window.__SKOLL_LOCALE === "string") {
      return resolveLocale(window.__SKOLL_LOCALE);
    }
    try {
      const persisted = (localStorage.getItem("skoll.ui.locale") || "").trim();
      if (persisted) {
        return resolveLocale(persisted);
      }
    } catch (err) {
      // Ignore localStorage errors in restricted environments.
    }
    const htmlLang = (document.documentElement.getAttribute("lang") || "").trim();
    if (htmlLang) {
      return resolveLocale(htmlLang);
    }
    return "zh-CN";
  }

  function detectLocaleQueryRaw() {
    try {
      return (new URLSearchParams(window.location.search).get("locale") || "").trim();
    } catch (err) {
      return "";
    }
  }

  function detectHostLocaleRaw() {
    if (typeof window === "undefined") {
      return "";
    }
    return typeof window.__SKOLL_LOCALE === "string" ? window.__SKOLL_LOCALE.trim() : "";
  }

  function detectStorageLocaleRaw() {
    try {
      return (localStorage.getItem("skoll.ui.locale") || "").trim();
    } catch (err) {
      return "";
    }
  }

  function detectHtmlLangRaw() {
    return (document.documentElement.getAttribute("lang") || "").trim();
  }

  function detectAllowedLocalesRaw() {
    if (typeof window === "undefined" || !Array.isArray(window.__SKOLL_LOCALES)) {
      return SUPPORTED_LOCALES.slice();
    }
    return window.__SKOLL_LOCALES.map(function (item) {
      return String(item || "").trim();
    }).filter(function (item) {
      return item !== "";
    });
  }

  function updateLocaleDiagnostics(source) {
    const queryLocale = detectLocaleQueryRaw();
    const hostLocale = detectHostLocaleRaw();
    const storageLocale = detectStorageLocaleRaw();
    const htmlLang = detectHtmlLangRaw();
    const allowedLocales = detectAllowedLocalesRaw();
    if (diagLocaleQueryEl) {
      diagLocaleQueryEl.textContent = queryLocale || "(empty)";
    }
    if (diagLocaleHostEl) {
      diagLocaleHostEl.textContent = hostLocale || "(empty)";
    }
    if (diagLocaleStorageEl) {
      diagLocaleStorageEl.textContent = storageLocale || "(empty)";
    }
    if (diagLocaleHtmlEl) {
      diagLocaleHtmlEl.textContent = htmlLang || "(empty)";
    }
    if (diagLocaleAllowedEl) {
      diagLocaleAllowedEl.textContent = allowedLocales.length > 0 ? allowedLocales.join(", ") : "(empty)";
    }
    if (diagLocaleActiveEl) {
      diagLocaleActiveEl.textContent = currentLocale;
    }
    if (diagLocaleSourceEl) {
      diagLocaleSourceEl.textContent = source || "unknown";
    }
  }

  function t(key) {
    const table = i18n[currentLocale] || i18n["en-US"];
    return table[key] || i18n["en-US"][key] || key;
  }

  function applyI18nToDOM() {
    document.documentElement.lang = currentLocale;
    document.querySelectorAll("[data-i18n]").forEach(function (node) {
      const key = node.getAttribute("data-i18n");
      if (!key) {
        return;
      }
      const value = t(key);
      if (/<[a-z][\s\S]*>/i.test(value)) {
        node.innerHTML = value;
      } else {
        node.textContent = value;
      }
    });
    document.querySelectorAll("[data-i18n-placeholder]").forEach(function (node) {
      const key = node.getAttribute("data-i18n-placeholder");
      if (!key) {
        return;
      }
      node.setAttribute("placeholder", t(key));
    });
    document.querySelectorAll("[data-i18n-aria-label]").forEach(function (node) {
      const key = node.getAttribute("data-i18n-aria-label");
      if (!key) {
        return;
      }
      node.setAttribute("aria-label", t(key));
    });
    document.title = t("page.title");
  }

  function escapeHtml(value) {
    return String(value || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function checkedPresetPermissionSet() {
    const selected = {};
    if (!manifestPermissionPresetEl) {
      return selected;
    }
    manifestPermissionPresetEl.querySelectorAll("input[type=checkbox]:checked").forEach(function (node) {
      const code = String(node.value || "").trim();
      if (code) {
        selected[code] = true;
      }
    });
    return selected;
  }

  function normalizePermissionCatalogData(raw) {
    const data = raw || {};
    const normalizeList = function (items) {
      if (!Array.isArray(items)) {
        return [];
      }
      const seen = {};
      return items.map(function (item) {
        return String(item || "").trim();
      }).filter(function (item) {
        if (!item || seen[item]) {
          return false;
        }
        seen[item] = true;
        return true;
      });
    };

    const plugins = Array.isArray(data.plugins) ? data.plugins.map(function (item) {
      const pluginId = String((item && item.pluginId) || "").trim();
      const pluginName = String((item && item.pluginName) || "").trim();
      return {
        pluginId: pluginId,
        pluginName: pluginName,
        permissions: normalizeList((item && item.permissions) || [])
      };
    }).filter(function (item) {
      return item.pluginId && item.permissions.length > 0;
    }) : [];

    return {
      self: normalizeList(data.self),
      framework: normalizeList(data.framework),
      plugins: plugins
    };
  }

  function buildPermissionCatalogGroups() {
    const groups = [];
    if (permissionCatalogData.self.length > 0) {
      groups.push({
        title: t("manifestVisual.permissionsGroupSelf"),
        source: "self",
        permissions: permissionCatalogData.self
      });
    }
    if (permissionCatalogData.framework.length > 0) {
      groups.push({
        title: t("manifestVisual.permissionsGroupFramework"),
        source: "framework",
        permissions: permissionCatalogData.framework
      });
    }
    if (permissionCatalogData.plugins.length > 0) {
      permissionCatalogData.plugins.forEach(function (item) {
        const titleSuffix = item.pluginName || item.pluginId;
        groups.push({
          title: t("manifestVisual.permissionsGroupPluginPrefix") + " · " + titleSuffix,
          source: "plugin",
          sourcePluginId: item.pluginId,
          permissions: item.permissions
        });
      });
    }
    return groups;
  }

  function bindPermissionPresetEvents() {
    if (!manifestPermissionPresetEl) {
      return;
    }
    manifestPermissionPresetEl.querySelectorAll("input[type=checkbox]").forEach(function (node) {
      node.addEventListener("change", function () {
        markManifestDirty();
        renderPermissionCatalog();
        scheduleSyncFromForm();
      });
    });
  }

  function renderPermissionPresetOptions() {
    if (!manifestPermissionPresetEl) {
      return;
    }
    const selectedMap = checkedPresetPermissionSet();
    const groups = buildPermissionCatalogGroups();
    const keyword = String(permissionSearchText || "").trim().toLowerCase();

    const blocks = groups.map(function (group) {
      const rows = group.permissions.filter(function (code) {
        return !keyword || code.toLowerCase().indexOf(keyword) >= 0;
      }).map(function (code) {
        const checked = selectedMap[code] ? " checked" : "";
        return "<label class=\"manifest-permission-row\">"
          + "<input type=\"checkbox\" value=\"" + escapeHtml(code) + "\" data-source=\"" + escapeHtml(group.source) + "\"" + checked + " />"
          + "<span>" + escapeHtml(code) + "</span>"
          + "</label>";
      });

      if (rows.length === 0) {
        return "";
      }

      return "<section class=\"manifest-permission-group\">"
        + "<div class=\"manifest-permission-group-title\">" + escapeHtml(group.title) + "</div>"
        + "<div class=\"manifest-permission-group-grid\">" + rows.join("") + "</div>"
        + "</section>";
    }).filter(function (block) {
      return block !== "";
    });

    if (blocks.length === 0) {
      manifestPermissionPresetEl.innerHTML = "<div class=\"manifest-permission-note\">" + escapeHtml(t("manifestVisual.permissionsGroupEmpty")) + "</div>";
      return;
    }

    manifestPermissionPresetEl.innerHTML = blocks.join("");
    bindPermissionPresetEvents();
  }

  async function loadPermissionCatalog(pluginId) {
    const normalizedPluginId = String(pluginId || "").trim();
    let path = "/skoll/v1/plugins/dev/permission-catalog?pluginsRoot="
      + encodeURIComponent((rootEl.value || "").trim());
    if (normalizedPluginId) {
      path += "&pluginId=" + encodeURIComponent(normalizedPluginId);
    }

    try {
      const payload = await callAPIGet(path);
      const data = payload && payload.data ? payload.data : payload;
      permissionCatalogData = normalizePermissionCatalogData(data);
      renderPermissionPresetOptions();
      renderPermissionCatalog();
    } catch (err) {
      permissionCatalogData = { self: [], framework: [], plugins: [] };
      renderPermissionPresetOptions();
      setStatus(t("status.permissionsCatalogLoadFailed") + " - " + formatAPIError(err, path), true);
    }
  }

  function collectCurrentPermissions() {
    const list = [];
    const seen = {};
    if (manifestPermissionPresetEl) {
      manifestPermissionPresetEl.querySelectorAll("input[type=checkbox]:checked").forEach(function (node) {
        const code = String(node.value || "").trim();
        if (code && !seen[code]) {
          seen[code] = true;
          list.push(code);
        }
      });
    }
    String((manifestFieldPermissionsCustomEl && manifestFieldPermissionsCustomEl.value) || "")
      .split(/[\r\n,]+/)
      .map(function (item) { return item.trim(); })
      .filter(function (item) { return item !== ""; })
      .forEach(function (code) {
        if (!seen[code]) {
          seen[code] = true;
          list.push(code);
        }
      });
    return list;
  }

  function renderPermissionCatalog() {
    if (manifestPermissionsMetaEl) {
      manifestPermissionsMetaEl.textContent = t("manifestVisual.permissionsMeta");
    }
    if (!manifestPermissionCatalogEl) {
      return;
    }
    const permissions = collectCurrentPermissions();
    if (!Array.isArray(permissions) || permissions.length === 0) {
      manifestPermissionCatalogEl.innerHTML = "<div class=\"manifest-permission-note\">" + escapeHtml(t("manifestVisual.permissionsMeaningFallback")) + "</div>";
      return;
    }
    manifestPermissionCatalogEl.innerHTML = "<div class=\"manifest-permission-note\">" + escapeHtml(t("manifestVisual.permissionsMeaning")) + "</div>"
      + permissions.map(function (code) {
        const desc = PERMISSION_DESCRIPTIONS[code];
        const note = desc ? (desc[currentLocale] || desc["en-US"] || code) : code;
        return "<div class=\"manifest-permission-item\">"
          + "<div class=\"manifest-permission-code\">" + escapeHtml(code) + "</div>"
          + "<div class=\"manifest-permission-note\">" + escapeHtml(note) + "</div>"
          + "</div>";
      }).join("");
  }

  function applyLocale(rawLocale, source) {
    currentLocale = resolveLocale(rawLocale);
    applyI18nToDOM();
    renderProjects();
    updateSelectedPluginMeta(projectByActionID(selectedProjectKey));
    renderPermissionPresetOptions();
    renderPermissionCatalog();
    updateLocaleDiagnostics(source || "applyLocale");
    updateResultDrawer();
  }

  function setStatus(msg, isError) {
    statusEl.textContent = msg;
    statusEl.classList.toggle("error", Boolean(isError));
    if (resultSummaryEl) {
      resultSummaryEl.textContent = String(msg || "");
      resultSummaryEl.classList.toggle("error", Boolean(isError));
    }
  }

  function setManifestDirty(nextDirty) {
    manifestDirty = Boolean(nextDirty);
    if (manifestPluginIdEl) {
      manifestPluginIdEl.classList.toggle("dirty", manifestDirty);
    }
    refreshFlowProgressFromState();
  }

  function markManifestDirty() {
    if (!suspendDirtyTracking) {
      setManifestDirty(true);
    }
  }

  function setManifestFeedback(msg, isError) {
    if (!manifestSaveFeedbackEl) {
      return;
    }
    manifestSaveFeedbackEl.textContent = String(msg || "");
    manifestSaveFeedbackEl.classList.toggle("error", Boolean(isError));
  }

  function setCustomPermissionVisibility(show) {
    const visible = Boolean(show);
    if (manifestPermissionsCustomWrapEl) {
      manifestPermissionsCustomWrapEl.classList.toggle("hidden", !visible);
    }
    if (manifestPermissionEnableCustomEl) {
      manifestPermissionEnableCustomEl.checked = visible;
    }
  }

  function confirmDiscardUnsavedManifest() {
    if (!manifestDirty) {
      return true;
    }
    return window.confirm(t("status.unsavedManifestConfirm"));
  }

  function setDevTab(tabId) {
    activeDevTab = tabId === "automation" ? "automation" : "manifest";
    if (btnDevTabManifest) {
      btnDevTabManifest.classList.toggle("active", activeDevTab === "manifest");
    }
    if (btnDevTabAutomation) {
      btnDevTabAutomation.classList.toggle("active", activeDevTab === "automation");
    }
    if (devPaneManifest) {
      devPaneManifest.classList.toggle("active", activeDevTab === "manifest");
    }
    if (devPaneAutomation) {
      devPaneAutomation.classList.toggle("active", activeDevTab === "automation");
    }
  }

  function setManifestTab(tabId) {
    const nextTab = tabId === "yaml" ? "yaml" : "visual";
    if (nextTab === activeManifestTab) {
      return;
    }
    if (nextTab === "yaml") {
      applyFormToManifestText();
    } else {
      try {
        parseManifestTextToForm();
      } catch (err) {
        setStatus(t("status.manifestFormParseFailed"), true);
      }
    }
    activeManifestTab = nextTab;
    if (btnManifestTabVisual) {
      btnManifestTabVisual.classList.toggle("active", activeManifestTab === "visual");
    }
    if (btnManifestTabYaml) {
      btnManifestTabYaml.classList.toggle("active", activeManifestTab === "yaml");
    }
    if (manifestPaneVisual) {
      manifestPaneVisual.classList.toggle("active", activeManifestTab === "visual");
    }
    if (manifestPaneYaml) {
      manifestPaneYaml.classList.toggle("active", activeManifestTab === "yaml");
    }
  }

  function filteredProjects() {
    return projectItems.filter(function (item) {
      const pluginId = String(item.pluginId || "").toLowerCase();
      const localizedName = String(resolveProjectName(item) || "").toLowerCase();
      const fallbackName = String(item.name || "").toLowerCase();
      const status = String(item.status || "").toLowerCase();
      const keyword = projectSearchText.toLowerCase();
      const hitSearch = !keyword || pluginId.indexOf(keyword) >= 0 || localizedName.indexOf(keyword) >= 0 || fallbackName.indexOf(keyword) >= 0;
      const hitStatus = projectStatusFilter === "all" || status === projectStatusFilter;
      return hitSearch && hitStatus;
    });
  }

  function formatI18n(templateKey, vars) {
    const template = String(t(templateKey) || "");
    return Object.keys(vars || {}).reduce(function (acc, key) {
      return acc.replace(new RegExp("\\{" + key + "\\}", "g"), String(vars[key] || ""));
    }, template);
  }

  function resolveProjectName(item) {
    if (!item) {
      return "";
    }
    const zhName = String(item.nameZhCN || "").trim();
    const enName = String(item.nameEnUS || "").trim();
    const defaultName = String(item.name || item.pluginId || "").trim();
    if (currentLocale === "zh-CN" && zhName) {
      return zhName;
    }
    if (currentLocale === "en-US" && enName) {
      return enName;
    }
    return defaultName;
  }

  function resolveProjectModeLabel(mode) {
    const normalized = String(mode || "workspace").trim().toLowerCase();
    if (normalized === "repository") {
      return t("project.mode.repository");
    }
    return t("project.mode.workspace");
  }

  function resolveProjectStatusLabel(status) {
    const normalized = String(status || "unknown").trim().toLowerCase();
    if (normalized === "ok" || normalized === "invalid" || normalized === "error" || normalized === "failed") {
      return t("project.status." + normalized);
    }
    return t("project.status.unknown");
  }

  function getToken() {
    const fromHost = String(window.__SKOLL_TOKEN || "").trim();
    if (fromHost) {
      return fromHost;
    }
    return (localStorage.getItem(TOKEN_KEY) || "").trim();
  }

  function resolveRequestURL(path) {
    const raw = String(path || "").trim();
    if (!raw) {
      return raw;
    }
    if (/^https?:\/\//i.test(raw)) {
      return raw;
    }
    let origin = "";
    try {
      if (window.top && window.top.location && window.top.location.origin) {
        origin = window.top.location.origin;
      }
    } catch (err) {
      origin = "";
    }
    if (!origin) {
      origin = window.location.origin || "";
    }
    if (!origin) {
      return raw;
    }
    const normalized = raw.startsWith("/") ? raw : ("/" + raw);
    return origin + normalized;
  }

  async function callAPI(path, body) {
    const token = getToken();
    const headers = {
      "Content-Type": "application/json"
    };
    if (token) {
      headers.Authorization = token.toLowerCase().startsWith("bearer ") ? token : "Bearer " + token;
    }

    const resp = await fetch(resolveRequestURL(path), {
      method: "POST",
      headers,
      body: JSON.stringify(body || {})
    });

    let payload = null;
    let rawText = "";
    try {
      rawText = await resp.text();
      payload = rawText ? JSON.parse(rawText) : null;
    } catch (e) {
      payload = null;
    }

    if (!resp.ok) {
      const err = new Error((payload && payload.message) || rawText || ("request failed: " + resp.status));
      err.status = resp.status;
      err.path = path;
      throw err;
    }

    return payload;
  }

  async function callAPIGet(path) {
    const token = getToken();
    const headers = {};
    if (token) {
      headers.Authorization = token.toLowerCase().startsWith("bearer ") ? token : "Bearer " + token;
    }
    const resp = await fetch(resolveRequestURL(path), { method: "GET", headers: headers });
    let payload = null;
    let rawText = "";
    try {
      rawText = await resp.text();
      payload = rawText ? JSON.parse(rawText) : null;
    } catch (e) {
      payload = null;
    }
    if (!resp.ok) {
      const err = new Error((payload && payload.message) || rawText || ("request failed: " + resp.status));
      err.status = resp.status;
      throw err;
    }
    return payload;
  }

  function formatAPIError(err, path) {
    const status = Number(err && err.status) || 0;
    if (status === 404 && String(path || "").indexOf("/v1/plugins/dev/") >= 0) {
      return t("status.devPortalDisabled");
    }
    return String((err && err.message) || err || "request failed");
  }

  async function callAnyAPI(method, path, body) {
    const token = getToken();
    const headers = {
      "Content-Type": "application/json"
    };
    if (token) {
      headers.Authorization = token.toLowerCase().startsWith("bearer ") ? token : "Bearer " + token;
    }

    const init = {
      method: method,
      headers: headers
    };
    if (body !== undefined) {
      init.body = JSON.stringify(body);
    }

    const resp = await fetch(resolveRequestURL(path), init);
    const text = await resp.text();
    let payload = text;
    try {
      payload = JSON.parse(text);
    } catch (e) {
      payload = text;
    }
    if (!resp.ok) {
      throw new Error((payload && payload.message) || ("request failed: " + resp.status));
    }
    return {
      status: resp.status,
      data: payload
    };
  }

  async function ensureSuperAdmin() {
    try {
      const token = getToken();
      const headers = {};
      if (token) {
        headers.Authorization = token.toLowerCase().startsWith("bearer ") ? token : "Bearer " + token;
      }
      const resp = await fetch(resolveRequestURL("/skoll/v1/auth/me"), { headers: headers });
      if (!resp.ok) {
        throw new Error("failed to resolve current user");
      }
      const payload = await resp.json();
      const role = (((payload || {}).data || {}).role || "").toString().trim().toLowerCase();
      allowActions = role === "super_admin";
    } catch (err) {
      allowActions = false;
    }

    if (!allowActions) {
      btnScaffold.disabled = true;
      btnValidateAll.disabled = true;
      if (btnLoadProjects) {
        btnLoadProjects.disabled = true;
      }
      if (btnManifestLoad) {
        btnManifestLoad.disabled = true;
      }
      if (btnManifestValidate) {
        btnManifestValidate.disabled = true;
      }
      if (btnManifestSave) {
        btnManifestSave.disabled = true;
      }
      if (btnRunPipeline) {
        btnRunPipeline.disabled = true;
      }
      if (btnApplyRollout) {
        btnApplyRollout.disabled = true;
      }
      if (btnRollbackRollout) {
        btnRollbackRollout.disabled = true;
      }
      if (btnCreateReleaseOrder) {
        btnCreateReleaseOrder.disabled = true;
      }
      if (btnListReleaseOrders) {
        btnListReleaseOrders.disabled = true;
      }
      if (btnApproveReleaseOrder) {
        btnApproveReleaseOrder.disabled = true;
      }
      if (btnRejectReleaseOrder) {
        btnRejectReleaseOrder.disabled = true;
      }
      if (btnSelectedEditManifest) {
        btnSelectedEditManifest.disabled = true;
      }
      if (btnSelectedRunPipeline) {
        btnSelectedRunPipeline.disabled = true;
      }
      if (btnSelectedListReleaseOrders) {
        btnSelectedListReleaseOrders.disabled = true;
      }
      setStatus(t("status.onlySuperAdmin"), true);
    }
  }

  async function handleScaffold() {
    const pluginId = (pluginIdEl.value || "").trim();
    const pluginName = (pluginNameEl.value || "").trim();
    if (!allowActions) {
      setStatus(t("status.onlySuperAdmin"), true);
      return;
    }
    if (!pluginId || !pluginName) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }

    setStatus(t("status.creatingScaffold"), false);
    setFlowStepState("scaffold", "in-progress");
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/scaffold", {
        pluginsRoot: (rootEl.value || "").trim(),
        pluginId: pluginId,
        pluginName: pluginName,
        appId: (appIdEl.value || "").trim(),
        mode: (scaffoldModeEl.value || "workspace").trim()
      });
      const result = payload && payload.data ? payload.data : payload;
      const createdDir = String((result && result.pluginDir) || "").trim();
      setStatus(createdDir ? (t("status.scaffoldCreatedAt") + " " + createdDir) : t("status.scaffoldCreated"), false);
      setFlowStepState("scaffold", "completed");
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
      void loadProjects();
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/scaffold"), true);
      resultEl.textContent = "{}";
    }
  }

  async function handleValidateAll() {
    if (!allowActions) {
      setStatus(t("status.onlySuperAdmin"), true);
      return;
    }
    setStatus(t("status.validating"), false);
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/validate-all", {
        pluginsRoot: (rootEl.value || "").trim()
      });
      setStatus(t("status.validateDone"), false);
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/validate-all"), true);
      resultEl.textContent = "{}";
    }
  }

  function renderProjects() {
    if (!projectRowsEl) {
      return;
    }
    const rows = filteredProjects();
    if (!Array.isArray(rows) || rows.length === 0) {
      const emptyKey = (projectSearchText || projectStatusFilter !== "all") ? "common.emptyFiltered" : "common.empty";
      projectRowsEl.innerHTML = "<tr><td colspan=\"5\" style=\"padding:8px; color:#6b7280;\">" + t(emptyKey) + "</td></tr>";
      updateSelectedPluginMeta(null);
      refreshFlowProgressFromState();
      return;
    }
    projectRowsEl.innerHTML = rows.map(function (item, idx) {
      const pluginId = String(item.pluginId || "").trim();
      const name = resolveProjectName(item) || pluginId || "-";
      const root = String(item.pluginsRoot || "-").trim();
      const mode = resolveProjectModeLabel(item.mode);
      const status = resolveProjectStatusLabel(item.status);
      const sourceIdx = projectItems.indexOf(item);
      const idAttr = "p" + sourceIdx;
      const cellStyle = "padding:6px; border-bottom:1px solid #eef2f7;";
      const rowCls = selectedProjectKey === idAttr ? "project-row selected" : "project-row";
      return "<tr class=\"" + rowCls + "\" data-row-id=\"" + idAttr + "\">"
        + "<td style=\"" + cellStyle + "\">" + name + "<br><small style=\"color:#64748b;\">" + pluginId + "</small></td>"
        + "<td style=\"" + cellStyle + "\">" + root + "</td>"
        + "<td style=\"" + cellStyle + "\">" + mode + "</td>"
        + "<td style=\"" + cellStyle + "\">" + status + "</td>"
        + "<td style=\"" + cellStyle + "\">"
        + "<button type=\"button\" class=\"table-action\" data-act=\"preview\" data-id=\"" + idAttr + "\">" + t("action.preview") + "</button> "
        + "<button type=\"button\" class=\"table-action\" data-act=\"package\" data-id=\"" + idAttr + "\">" + t("action.package") + "</button> "
        + "<button type=\"button\" class=\"table-action\" data-act=\"remove\" data-id=\"" + idAttr + "\">" + t("action.removeAndDelete") + "</button>"
        + "</td>"
        + "</tr>";
    }).join("");
  }

  function updateSelectedPluginMeta(item) {
    if (!selectedPluginMetaEl) {
      return;
    }
    if (!item) {
      selectedPluginMetaEl.textContent = t("pluginDev.selected.empty");
      return;
    }
    const pluginId = String(item.pluginId || "").trim();
    const root = String(item.pluginsRoot || "-").trim();
    const mode = resolveProjectModeLabel(item.mode);
    selectedPluginMetaEl.textContent = formatI18n("pluginDev.selected.meta", {
      id: pluginId,
      root: root,
      mode: mode
    });
  }

  function applyProjectSelection(item, actionID) {
    if (!item) {
      return;
    }
    const pluginId = String(item.pluginId || "").trim();
    if (!pluginId) {
      return;
    }
    selectedProjectKey = String(actionID || "").trim();
    if (manifestPluginIdEl) {
      manifestPluginIdEl.value = pluginId;
    }
    if (pipelinePluginIdEl) {
      pipelinePluginIdEl.value = pluginId;
    }
    if (rolloutPluginIdEl) {
      rolloutPluginIdEl.value = pluginId;
    }
    if (releasePluginIdEl) {
      releasePluginIdEl.value = pluginId;
    }
    updateSelectedPluginMeta(item);
    renderProjects();
    setDevTab("manifest");
    setFlowStepState("select", "completed");
    setFlowStepState("configure", "in-progress");
  }

  async function loadDevConfig() {
    if (!allowActions) {
      return;
    }
    try {
      const payload = await callAPIGet("/skoll/v1/plugins/dev/config");
      const data = payload && payload.data ? payload.data : payload;
      const roots = Array.isArray(data && data.allowedRoots) ? data.allowedRoots : [];
      const defaultRoot = String((data && data.defaultRoot) || "").trim();
      if (defaultRoot && !String(rootEl.value || "").trim()) {
        rootEl.value = defaultRoot;
      }
      if (rootOptionsEl) {
        rootOptionsEl.innerHTML = roots.map(function (item) {
          return "<option value=\"" + String(item).replace(/\"/g, "&quot;") + "\"></option>";
        }).join("");
      }
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/config"), true);
    }
  }

  function projectByActionID(actionID) {
    if (!String(actionID || "").startsWith("p")) {
      return null;
    }
    const idx = Number(String(actionID).slice(1));
    if (!Number.isInteger(idx) || idx < 0 || idx >= projectItems.length) {
      return null;
    }
    return projectItems[idx];
  }

  async function loadProjects() {
    if (!allowActions) {
      setStatus(t("status.onlySuperAdmin"), true);
      return;
    }
    setStatus(t("status.loadingProjects"), false);
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/projects", {
        pluginsRoot: (rootEl.value || "").trim()
      });
      const data = payload && payload.data ? payload.data : payload;
      projectItems = Array.isArray(data && data.projects) ? data.projects : [];
      renderProjects();
      syncPluginFieldsFromProjects();
      if (!String((manifestFieldIdEl && manifestFieldIdEl.value) || "").trim() && selectedPluginId()) {
        void loadManifest(true);
      }
      refreshFlowProgressFromState();
      setStatus(t("status.projectsLoaded"), false);
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/projects"), true);
    }
  }

  async function packageProject(item) {
    setStatus(t("status.packaging"), false);
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/package", {
        pluginsRoot: item.pluginsRoot,
        pluginId: item.pluginId
      });
      setStatus(t("status.packaged"), false);
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(t("status.packFailed") + " - " + formatAPIError(err, "/skoll/v1/plugins/dev/package"), true);
    }
  }

  async function removeProject(item) {
    setStatus(t("status.removing"), false);
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/remove", {
        pluginsRoot: item.pluginsRoot,
        pluginId: item.pluginId,
        removeFiles: true
      });
      setStatus(t("status.removed"), false);
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
      void loadProjects();
    } catch (err) {
      setStatus(t("status.removeFailed") + " - " + formatAPIError(err, "/skoll/v1/plugins/dev/remove"), true);
    }
  }

  function previewProject(item) {
    const url = resolveRequestURL(item.previewUrl || ("/skoll/plugins/" + item.pluginId));
    window.open(url, "_blank", "noopener,noreferrer");
    const hints = {
      pluginId: item.pluginId,
      buildHint: item.buildHint || "",
      packageHint: item.packageHint || "",
      publishHint: item.publishHint || ""
    };
    resultEl.textContent = JSON.stringify(hints, null, 2);
    setStatus(t("status.projectHints"), false);
  }

  function selectedPluginId() {
    const fromManifest = String((manifestPluginIdEl && manifestPluginIdEl.value) || "").trim();
    if (fromManifest) {
      return fromManifest;
    }
    return String((pluginIdEl && pluginIdEl.value) || "").trim();
  }

  function parseManifestScalar(raw) {
    const value = String(raw || "").trim();
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      return value.substring(1, value.length - 1);
    }
    return value;
  }

  function parseManifestText(text) {
    const manifest = {
      id: "",
      name: "",
      name_zh_cn: "",
      name_en_us: "",
      version: "0.1.0",
      description: "",
      api_version: "v1",
      compatibility_skoll: ">=1.0.0 <2.0.0",
      migration_version: "v0.1.0",
      ui_mode: "separated",
      level: "system",
      app_id: "",
      mount_policy: "admin",
      ui_nav_position: "none",
      ui_open_mode: "integrated",
      ui_tab_mode: "optional",
      frontend_entry: "",
      i18n_locales: ["zh-CN", "en-US"],
      permissions: []
    };

    const lines = String(text || "").split(/\r?\n/);
    let listSection = "";
    lines.forEach(function (line) {
      const trimmed = line.trim();
      if (trimmed === "" || trimmed.startsWith("#")) {
        return;
      }
      if (trimmed === "i18n_locales:") {
        listSection = "i18n_locales";
        manifest.i18n_locales = [];
        return;
      }
      if (trimmed === "permissions:") {
        listSection = "permissions";
        manifest.permissions = [];
        return;
      }
      if (trimmed.startsWith("-")) {
        const item = parseManifestScalar(trimmed.slice(1));
        if (listSection === "i18n_locales" && item) {
          manifest.i18n_locales.push(item);
        }
        if (listSection === "permissions" && item) {
          manifest.permissions.push(item);
        }
        return;
      }

      const idx = trimmed.indexOf(":");
      if (idx <= 0) {
        return;
      }
      listSection = "";
      const key = trimmed.slice(0, idx).trim();
      const rawValue = trimmed.slice(idx + 1).trim();
      const value = parseManifestScalar(rawValue);
      if (Object.prototype.hasOwnProperty.call(manifest, key)) {
        manifest[key] = value;
      }
    });

    return manifest;
  }

  function applyManifestToForm(manifest) {
    suspendDirtyTracking = true;
    suspendManifestSync = true;
    if (!manifest) {
      suspendManifestSync = false;
      suspendDirtyTracking = false;
      return;
    }
    if (manifestFieldIdEl) { manifestFieldIdEl.value = String(manifest.id || ""); }
    if (manifestFieldNameEl) { manifestFieldNameEl.value = String(manifest.name || ""); }
    if (manifestFieldNameZhCNEl) { manifestFieldNameZhCNEl.value = String(manifest.name_zh_cn || ""); }
    if (manifestFieldNameEnUSEl) { manifestFieldNameEnUSEl.value = String(manifest.name_en_us || ""); }
    if (manifestFieldVersionEl) { manifestFieldVersionEl.value = String(manifest.version || ""); }
    if (manifestFieldDescriptionEl) { manifestFieldDescriptionEl.value = String(manifest.description || ""); }
    if (manifestFieldApiVersionEl) { manifestFieldApiVersionEl.value = String(manifest.api_version || ""); }
    if (manifestFieldCompatEl) { manifestFieldCompatEl.value = String(manifest.compatibility_skoll || ""); }
    if (manifestFieldMigrationEl) { manifestFieldMigrationEl.value = String(manifest.migration_version || ""); }
    if (manifestFieldLevelEl) { manifestFieldLevelEl.value = String(manifest.level || "system"); }
    if (manifestFieldAppIdEl) { manifestFieldAppIdEl.value = String(manifest.app_id || ""); }
    if (manifestFieldUiModeEl) { manifestFieldUiModeEl.value = String(manifest.ui_mode || "separated"); }
    if (manifestFieldMountPolicyEl) { manifestFieldMountPolicyEl.value = String(manifest.mount_policy || "admin"); }
    if (manifestFieldNavPositionEl) { manifestFieldNavPositionEl.value = String(manifest.ui_nav_position || "none"); }
    if (manifestFieldOpenModeEl) { manifestFieldOpenModeEl.value = String(manifest.ui_open_mode || "integrated"); }
    if (manifestFieldTabModeEl) { manifestFieldTabModeEl.value = String(manifest.ui_tab_mode || "optional"); }
    if (manifestFieldFrontendEntryEl) { manifestFieldFrontendEntryEl.value = String(manifest.frontend_entry || ""); }
    if (manifestFieldLocalesEl) { manifestFieldLocalesEl.value = Array.isArray(manifest.i18n_locales) ? manifest.i18n_locales.join(",") : ""; }
    const permissionSet = Array.isArray(manifest.permissions) ? manifest.permissions.map(function (item) {
      return String(item || "").trim();
    }).filter(function (item) {
      return item !== "";
    }) : [];
    const presetMap = {};
    if (manifestPermissionPresetEl) {
      manifestPermissionPresetEl.querySelectorAll("input[type=checkbox]").forEach(function (node) {
        const key = String(node.value || "").trim();
        if (key) {
          presetMap[key] = true;
          node.checked = permissionSet.indexOf(key) >= 0;
        }
      });
    }
    if (manifestFieldPermissionsCustomEl) {
      const customPermissions = permissionSet.filter(function (perm) {
        return !presetMap[perm];
      });
      manifestFieldPermissionsCustomEl.value = customPermissions.join(", ");
      setCustomPermissionVisibility(customPermissions.length > 0);
    }
    if (manifestPluginIdEl && !String(manifestPluginIdEl.value || "").trim()) {
      manifestPluginIdEl.value = String(manifest.id || "").trim();
    }
    renderPermissionCatalog();
    suspendManifestSync = false;
    suspendDirtyTracking = false;
  }

  function quoteManifestValue(value) {
    const text = String(value || "").trim();
    if (text === "") {
      return '""';
    }
    if (/^[a-z0-9_.-]+$/i.test(text)) {
      return text;
    }
    return '"' + text.replace(/"/g, '\\"') + '"';
  }

  function normalizeManifestFieldDependencies() {
    const openMode = String((manifestFieldOpenModeEl && manifestFieldOpenModeEl.value) || "integrated").trim();
    if (openMode === "standalone") {
      if (manifestFieldNavPositionEl) {
        manifestFieldNavPositionEl.value = "none";
      }
      if (manifestFieldTabModeEl) {
        manifestFieldTabModeEl.value = "disabled";
      }
    }
    const level = String((manifestFieldLevelEl && manifestFieldLevelEl.value) || "system").trim();
    if (level === "system" && manifestFieldAppIdEl) {
      manifestFieldAppIdEl.value = "";
    }
  }

  function validateManifestFormState() {
    const id = String((manifestFieldIdEl && manifestFieldIdEl.value) || "").trim();
    const name = String((manifestFieldNameEl && manifestFieldNameEl.value) || "").trim();
    const version = String((manifestFieldVersionEl && manifestFieldVersionEl.value) || "").trim();
    const apiVersion = String((manifestFieldApiVersionEl && manifestFieldApiVersionEl.value) || "").trim();
    const migration = String((manifestFieldMigrationEl && manifestFieldMigrationEl.value) || "").trim();
    const compat = String((manifestFieldCompatEl && manifestFieldCompatEl.value) || "").trim();
    const mode = String((manifestFieldUiModeEl && manifestFieldUiModeEl.value) || "backend_only").trim();
    const level = String((manifestFieldLevelEl && manifestFieldLevelEl.value) || "system").trim();
    const appId = String((manifestFieldAppIdEl && manifestFieldAppIdEl.value) || "").trim();
    const nav = String((manifestFieldNavPositionEl && manifestFieldNavPositionEl.value) || "none").trim();
    const openMode = String((manifestFieldOpenModeEl && manifestFieldOpenModeEl.value) || "integrated").trim();
    const tabMode = String((manifestFieldTabModeEl && manifestFieldTabModeEl.value) || "optional").trim();
    const locales = String((manifestFieldLocalesEl && manifestFieldLocalesEl.value) || "")
      .split(",")
      .map(function (item) { return item.trim(); })
      .filter(function (item) { return item !== ""; });

    if (!id || !name || !version) {
      return t("status.manifestInvalidRequired");
    }
    if (apiVersion && !/^v[0-9]+$/.test(apiVersion)) {
      return t("status.manifestInvalidApiVersion");
    }
    if (migration && !/^v?[0-9]+\.[0-9]+\.[0-9]+$/.test(migration)) {
      return t("status.manifestInvalidMigration");
    }
    if (apiVersion && !compat) {
      return t("status.manifestInvalidCompat");
    }
    if (level === "app") {
      if (!/^[a-z0-9][a-z0-9_-]{1,62}$/.test(appId) || appId === "skoll") {
        return t("status.manifestInvalidLevel");
      }
    }
    if (level !== "app" && appId) {
      return t("status.manifestInvalidLevel");
    }
    if (openMode === "standalone" && (nav !== "none" || tabMode !== "disabled")) {
      return t("status.manifestInvalidOpenMode");
    }
    if (mode !== "backend_only") {
      if (locales.length === 0) {
        return t("status.manifestInvalidLocales");
      }
      const localeSeen = {};
      for (let i = 0; i < locales.length; i += 1) {
        const locale = locales[i];
        if (!/^[a-z]{2}(?:-[A-Z]{2})?$/.test(locale) || localeSeen[locale]) {
          return t("status.manifestInvalidLocales");
        }
        localeSeen[locale] = true;
      }
    }
    return "";
  }

  function buildManifestFromForm() {
    normalizeManifestFieldDependencies();
    const id = String((manifestFieldIdEl && manifestFieldIdEl.value) || "").trim();
    const name = String((manifestFieldNameEl && manifestFieldNameEl.value) || "").trim();
    const nameZhCN = String((manifestFieldNameZhCNEl && manifestFieldNameZhCNEl.value) || "").trim();
    const nameEnUS = String((manifestFieldNameEnUSEl && manifestFieldNameEnUSEl.value) || "").trim();
    const version = String((manifestFieldVersionEl && manifestFieldVersionEl.value) || "0.1.0").trim();
    const description = String((manifestFieldDescriptionEl && manifestFieldDescriptionEl.value) || "").trim();
    const apiVersion = String((manifestFieldApiVersionEl && manifestFieldApiVersionEl.value) || "v1").trim();
    const compat = String((manifestFieldCompatEl && manifestFieldCompatEl.value) || ">=1.0.0 <2.0.0").trim();
    const migration = String((manifestFieldMigrationEl && manifestFieldMigrationEl.value) || "v0.1.0").trim();
    const uiMode = String((manifestFieldUiModeEl && manifestFieldUiModeEl.value) || "separated").trim();
    const level = String((manifestFieldLevelEl && manifestFieldLevelEl.value) || "system").trim();
    const appId = String((manifestFieldAppIdEl && manifestFieldAppIdEl.value) || "").trim();
    const mountPolicy = String((manifestFieldMountPolicyEl && manifestFieldMountPolicyEl.value) || "admin").trim();
    const navPosition = String((manifestFieldNavPositionEl && manifestFieldNavPositionEl.value) || "none").trim();
    const openMode = String((manifestFieldOpenModeEl && manifestFieldOpenModeEl.value) || "integrated").trim();
    const tabMode = String((manifestFieldTabModeEl && manifestFieldTabModeEl.value) || "optional").trim();
    const frontendEntry = String((manifestFieldFrontendEntryEl && manifestFieldFrontendEntryEl.value) || "").trim();
    const locales = String((manifestFieldLocalesEl && manifestFieldLocalesEl.value) || "")
      .split(",")
      .map(function (item) { return item.trim(); })
      .filter(function (item) { return item !== ""; });
    const permissions = [];
    const seenPermissions = {};
    if (manifestPermissionPresetEl) {
      manifestPermissionPresetEl.querySelectorAll("input[type=checkbox]:checked").forEach(function (node) {
        const permission = String(node.value || "").trim();
        if (!permission || seenPermissions[permission]) {
          return;
        }
        seenPermissions[permission] = true;
        permissions.push(permission);
      });
    }
    String((manifestFieldPermissionsCustomEl && manifestFieldPermissionsCustomEl.value) || "")
      .split(/[\r\n,]+/)
      .map(function (item) { return item.trim(); })
      .filter(function (item) { return item !== ""; })
      .forEach(function (permission) {
        if (seenPermissions[permission]) {
          return;
        }
        seenPermissions[permission] = true;
        permissions.push(permission);
      });

    const lines = [
      "id: " + quoteManifestValue(id),
      "name: " + quoteManifestValue(name)
    ];
    if (nameZhCN) {
      lines.push("name_zh_cn: " + quoteManifestValue(nameZhCN));
    }
    if (nameEnUS) {
      lines.push("name_en_us: " + quoteManifestValue(nameEnUS));
    }
    lines.push("version: " + quoteManifestValue(version));
    if (description) {
      lines.push("description: " + quoteManifestValue(description));
    }
    lines.push(
      "api_version: " + quoteManifestValue(apiVersion),
      "compatibility_skoll: " + quoteManifestValue(compat),
      "migration_version: " + quoteManifestValue(migration),
      "ui_mode: " + quoteManifestValue(uiMode),
      "level: " + quoteManifestValue(level),
      "mount_policy: " + quoteManifestValue(mountPolicy),
      "ui_nav_position: " + quoteManifestValue(navPosition),
      "ui_open_mode: " + quoteManifestValue(openMode),
      "ui_tab_mode: " + quoteManifestValue(tabMode)
    );
    if (frontendEntry) {
      lines.push("frontend_entry: " + quoteManifestValue(frontendEntry));
    }
    if (appId) {
      lines.push("app_id: " + quoteManifestValue(appId));
    }
    lines.push("i18n_locales:");
    (locales.length > 0 ? locales : ["zh-CN", "en-US"]).forEach(function (locale) {
      lines.push("  - " + quoteManifestValue(locale));
    });
    lines.push("", "permissions:");
    permissions.forEach(function (perm) {
      lines.push("  - " + quoteManifestValue(perm));
    });
    return lines.join("\n") + "\n";
  }

  function applyFormToManifestText() {
    if (suspendManifestSync) {
      return;
    }
    suspendManifestSync = true;
    try {
      const prev = String((manifestYamlEl && manifestYamlEl.value) || "");
      const manifest = buildManifestFromForm();
      if (manifestYamlEl) {
        suspendDirtyTracking = true;
        manifestYamlEl.value = manifest;
        suspendDirtyTracking = false;
      }
      if (manifestPluginIdEl && !String(manifestPluginIdEl.value || "").trim() && manifestFieldIdEl) {
        manifestPluginIdEl.value = String(manifestFieldIdEl.value || "").trim();
      }
      if (prev !== manifest) {
        markManifestDirty();
      }
    } finally {
      suspendManifestSync = false;
    }
  }

  function parseManifestTextToForm() {
    if (suspendManifestSync) {
      return;
    }
    suspendManifestSync = true;
    try {
      const raw = String((manifestYamlEl && manifestYamlEl.value) || "").trim();
      if (!raw) {
        return;
      }
      const parsed = parseManifestText(raw);
      applyManifestToForm(parsed);
    } finally {
      suspendManifestSync = false;
    }
  }

  function scheduleSyncFromForm() {
    if (suspendManifestSync) {
      return;
    }
    if (formSyncTimer) {
      clearTimeout(formSyncTimer);
    }
    formSyncTimer = setTimeout(function () {
      applyFormToManifestText();
    }, 180);
  }

  function scheduleSyncFromYaml() {
    if (suspendManifestSync) {
      return;
    }
    if (yamlSyncTimer) {
      clearTimeout(yamlSyncTimer);
    }
    yamlSyncTimer = setTimeout(function () {
      try {
        parseManifestTextToForm();
      } catch (err) {
        // Ignore transient YAML parse errors while user is typing.
      }
    }, 220);
  }

  function syncManifestForSubmission() {
    if (activeManifestTab === "yaml") {
      try {
        parseManifestTextToForm();
      } catch (err) {
        const msg = t("status.manifestFormParseFailed");
        setStatus(msg, true);
        setManifestFeedback(msg, true);
        return false;
      }
      return true;
    }
    applyFormToManifestText();
    return true;
  }

  function syncPluginFieldsFromProjects() {
    if (!Array.isArray(projectItems) || projectItems.length === 0) {
      updateSelectedPluginMeta(null);
      return;
    }
    const selected = projectByActionID(selectedProjectKey) || projectItems[0];
    const selectedID = projectByActionID(selectedProjectKey) ? selectedProjectKey : "p0";
    applyProjectSelection(selected, selectedID);
  }

  function focusManifestEditorFor(item, actionID) {
    if (!confirmDiscardUnsavedManifest()) {
      return;
    }
    setManifestDirty(false);
    applyProjectSelection(item, actionID);
    if (manifestFieldIdEl && !String(manifestFieldIdEl.value || "").trim()) {
      manifestFieldIdEl.value = String((item && item.pluginId) || "").trim();
    }
    setDevTab("manifest");
    setManifestTab("visual");
    void loadManifest(true);
    if (manifestYamlEl && manifestYamlEl.scrollIntoView) {
      manifestYamlEl.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }

  function selectedReleasePluginId() {
    const fromRelease = String((releasePluginIdEl && releasePluginIdEl.value) || "").trim();
    if (fromRelease) {
      return fromRelease;
    }
    return selectedPluginId();
  }

  async function loadManifest(skipUnsavedConfirm) {
    if (!skipUnsavedConfirm && !confirmDiscardUnsavedManifest()) {
      return;
    }
    const pluginId = selectedPluginId();
    if (!pluginId) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }
    setStatus(t("status.loadingManifest"), false);
    await loadPermissionCatalog(pluginId);
    try {
      const path = "/skoll/v1/plugins/dev/manifest?pluginsRoot=" + encodeURIComponent((rootEl.value || "").trim()) + "&pluginId=" + encodeURIComponent(pluginId);
      const payload = await callAPIGet(path);
      const data = payload && payload.data ? payload.data : payload;
      if (manifestYamlEl) {
        suspendDirtyTracking = true;
        manifestYamlEl.value = String((data && data.manifest) || "");
        suspendDirtyTracking = false;
      }
      parseManifestTextToForm();
      setManifestDirty(false);
      setManifestFeedback("", false);
      setFlowStepState("configure", "in-progress");
      setStatus(t("status.manifestLoaded"), false);
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/manifest"), true);
    }
  }

  async function validateManifestDraft() {
    const pluginId = selectedPluginId();
    if (!syncManifestForSubmission()) {
      return;
    }
    const validationErr = validateManifestFormState();
    if (validationErr) {
      setStatus(validationErr, true);
      setManifestFeedback(validationErr, true);
      return;
    }
    const manifest = String((manifestYamlEl && manifestYamlEl.value) || "").trim();
    if (!pluginId || !manifest) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }
    setStatus(t("status.validatingManifest"), false);
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/manifest/validate", {
        pluginsRoot: (rootEl.value || "").trim(),
        pluginId: pluginId,
        manifest: manifest
      });
      setStatus(t("status.manifestValid"), false);
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/manifest"), true);
    }
  }

  async function saveManifest() {
    const pluginId = selectedPluginId();
    if (!syncManifestForSubmission()) {
      return;
    }
    const validationErr = validateManifestFormState();
    if (validationErr) {
      setStatus(validationErr, true);
      setManifestFeedback(validationErr, true);
      resultExpanded = true;
      updateResultDrawer();
      return;
    }
    const manifest = String((manifestYamlEl && manifestYamlEl.value) || "").trim();
    if (!pluginId || !manifest) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }
    setStatus(t("status.savingManifest"), false);
    setManifestFeedback(t("status.savingManifest"), false);
    try {
      const payload = await callAnyAPI("PUT", "/skoll/v1/plugins/dev/manifest", {
        pluginsRoot: (rootEl.value || "").trim(),
        pluginId: pluginId,
        manifest: manifest
      });
      setManifestDirty(false);
      setStatus(t("status.manifestSaved"), false);
      setManifestFeedback(t("status.manifestSaved"), false);
      resultExpanded = true;
      updateResultDrawer();
      setFlowStepState("configure", "completed");
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
      void loadProjects();
    } catch (err) {
      const errMsg = formatAPIError(err, "/skoll/v1/plugins/dev/manifest");
      setStatus(errMsg, true);
      setManifestFeedback(errMsg, true);
      resultExpanded = true;
      updateResultDrawer();
    }
  }

  async function runPipeline() {
    const pluginId = String((pipelinePluginIdEl && pipelinePluginIdEl.value) || "").trim() || selectedPluginId();
    if (!pluginId) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }
    setStatus(t("status.runningPipeline"), false);
    setFlowStepState("release", "in-progress");
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/pipeline", {
        pluginsRoot: (rootEl.value || "").trim(),
        pluginId: pluginId
      });
      setStatus(t("status.pipelineDone"), false);
      setFlowStepState("release", "completed");
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/pipeline"), true);
    }
  }

  async function applyRollout() {
    const pluginId = String((rolloutPluginIdEl && rolloutPluginIdEl.value) || "").trim() || selectedPluginId();
    const rolloutPercent = Number(String((rolloutPercentEl && rolloutPercentEl.value) || "100").trim());
    if (!pluginId) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }
    setStatus(t("status.rolloutApplying"), false);
    setFlowStepState("release", "in-progress");
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/rollout", {
        pluginId: pluginId,
        rolloutPercent: rolloutPercent
      });
      setStatus(t("status.rolloutApplied"), false);
      setFlowStepState("release", "completed");
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/rollout"), true);
    }
  }

  async function rollbackRollout() {
    const pluginId = String((rolloutPluginIdEl && rolloutPluginIdEl.value) || "").trim() || selectedPluginId();
    if (!pluginId) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }
    setStatus(t("status.rollbacking"), false);
    setFlowStepState("release", "in-progress");
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/rollback", {
        pluginId: pluginId
      });
      setStatus(t("status.rollbackDone"), false);
      setFlowStepState("release", "completed");
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/rollback"), true);
    }
  }

  async function createReleaseOrder() {
    const pluginId = selectedReleasePluginId();
    const releaseVersion = String((releaseVersionEl && releaseVersionEl.value) || "").trim();
    const changelog = String((releaseChangelogEl && releaseChangelogEl.value) || "").trim();
    if (!pluginId) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }
    setStatus(t("status.creatingReleaseOrder"), false);
    setFlowStepState("release", "in-progress");
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/release-orders", {
        pluginsRoot: (rootEl.value || "").trim(),
        pluginId: pluginId,
        releaseVersion: releaseVersion,
        changelog: changelog
      });
      const data = payload && payload.data ? payload.data : payload;
      if (releaseOrderIdEl && data && data.order && data.order.orderId) {
        releaseOrderIdEl.value = String(data.order.orderId);
      }
      setStatus(t("status.releaseOrderCreated"), false);
      setFlowStepState("release", "completed");
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/release-orders"), true);
    }
  }

  async function listReleaseOrders() {
    const pluginId = selectedReleasePluginId();
    setStatus(t("status.listingReleaseOrders"), false);
    try {
      const query = "?pluginsRoot=" + encodeURIComponent((rootEl.value || "").trim()) + "&pluginId=" + encodeURIComponent(pluginId);
      const payload = await callAPIGet("/skoll/v1/plugins/dev/release-orders" + query);
      setStatus(t("status.releaseOrdersListed"), false);
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
      const data = payload && payload.data ? payload.data : payload;
      if (releaseOrderIdEl && data && Array.isArray(data.orders) && data.orders.length > 0) {
        releaseOrderIdEl.value = String(data.orders[0].orderId || "");
      }
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/release-orders"), true);
    }
  }

  async function approveReleaseOrder() {
    const pluginId = selectedReleasePluginId();
    const orderId = String((releaseOrderIdEl && releaseOrderIdEl.value) || "").trim();
    const comment = String((releaseChangelogEl && releaseChangelogEl.value) || "").trim();
    if (!pluginId || !orderId) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }
    setStatus(t("status.approvingReleaseOrder"), false);
    setFlowStepState("release", "in-progress");
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/release-orders/" + encodeURIComponent(orderId) + "/approve", {
        pluginsRoot: (rootEl.value || "").trim(),
        pluginId: pluginId,
        comment: comment
      });
      setStatus(t("status.releaseOrderApproved"), false);
      setFlowStepState("release", "completed");
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/release-orders"), true);
    }
  }

  async function rejectReleaseOrder() {
    const pluginId = selectedReleasePluginId();
    const orderId = String((releaseOrderIdEl && releaseOrderIdEl.value) || "").trim();
    const comment = String((releaseChangelogEl && releaseChangelogEl.value) || "").trim();
    if (!pluginId || !orderId) {
      setStatus(t("status.requirePluginIdAndName"), true);
      return;
    }
    setStatus(t("status.rejectingReleaseOrder"), false);
    setFlowStepState("release", "in-progress");
    try {
      const payload = await callAPI("/skoll/v1/plugins/dev/release-orders/" + encodeURIComponent(orderId) + "/reject", {
        pluginsRoot: (rootEl.value || "").trim(),
        pluginId: pluginId,
        comment: comment
      });
      setStatus(t("status.releaseOrderRejected"), false);
      setFlowStepState("release", "completed");
      resultEl.textContent = JSON.stringify(payload.data || payload, null, 2);
    } catch (err) {
      setStatus(formatAPIError(err, "/skoll/v1/plugins/dev/release-orders"), true);
    }
  }

  function activateModule(moduleId) {
    document.querySelectorAll(".module-card").forEach(function (node) {
      node.classList.toggle("active", node.getAttribute("data-module") === moduleId);
    });
    document.querySelectorAll(".panel").forEach(function (node) {
      node.classList.toggle("active", node.id === moduleId);
    });
  }

  async function sendApiRequest(pathOverride, methodOverride) {
    const method = (methodOverride || apiMethodEl.value || "GET").trim().toUpperCase();
    const path = (pathOverride || apiPathEl.value || "").trim();
    if (!path) {
      setStatus(t("status.pathRequired"), true);
      return;
    }

    let body;
    if (method !== "GET") {
      const rawBody = (apiBodyEl.value || "").trim();
      if (rawBody) {
        try {
          body = JSON.parse(rawBody);
        } catch (err) {
          setStatus(t("status.invalidJson"), true);
          return;
        }
      }
    }

    setStatus(t("status.sending") + " " + method + " " + path + " ...", false);
    try {
      const payload = await callAnyAPI(method, path, body);
      setStatus(t("status.requestSuccess"), false);
      resultEl.textContent = JSON.stringify(payload, null, 2);
    } catch (err) {
      setStatus(String(err.message || err), true);
      resultEl.textContent = "{}";
    }
  }

  document.querySelectorAll(".module-card").forEach(function (node) {
    node.addEventListener("click", function () {
      activateModule(node.getAttribute("data-module"));
    });
  });

  btnScaffold.addEventListener("click", function () {
    void handleScaffold();
  });
  btnValidateAll.addEventListener("click", function () {
    void handleValidateAll();
  });
  if (btnLoadProjects) {
    btnLoadProjects.addEventListener("click", function () {
      void loadProjects();
    });
  }
  if (projectSearchEl) {
    projectSearchEl.addEventListener("input", function () {
      projectSearchText = String(projectSearchEl.value || "").trim();
      renderProjects();
    });
  }
  if (projectStatusFilterEl) {
    projectStatusFilterEl.addEventListener("change", function () {
      projectStatusFilter = String(projectStatusFilterEl.value || "all").trim() || "all";
      renderProjects();
    });
  }
  if (btnDevTabManifest) {
    btnDevTabManifest.addEventListener("click", function () {
      setDevTab("manifest");
    });
  }
  if (btnDevTabAutomation) {
    btnDevTabAutomation.addEventListener("click", function () {
      setDevTab("automation");
    });
  }
  if (btnManifestTabVisual) {
    btnManifestTabVisual.addEventListener("click", function () {
      setManifestTab("visual");
    });
  }
  if (btnManifestTabYaml) {
    btnManifestTabYaml.addEventListener("click", function () {
      setManifestTab("yaml");
    });
  }
  if (btnManifestLoad) {
    btnManifestLoad.addEventListener("click", function () {
      void loadManifest(false);
    });
  }
  if (btnManifestValidate) {
    btnManifestValidate.addEventListener("click", function () {
      void validateManifestDraft();
    });
  }
  if (btnManifestSave) {
    btnManifestSave.addEventListener("click", function () {
      void saveManifest();
    });
  }
  if (btnRunPipeline) {
    btnRunPipeline.addEventListener("click", function () {
      void runPipeline();
    });
  }
  if (btnSelectedRunPipeline) {
    btnSelectedRunPipeline.addEventListener("click", function () {
      void runPipeline();
    });
  }
  if (btnApplyRollout) {
    btnApplyRollout.addEventListener("click", function () {
      void applyRollout();
    });
  }
  if (btnRollbackRollout) {
    btnRollbackRollout.addEventListener("click", function () {
      void rollbackRollout();
    });
  }
  if (btnCreateReleaseOrder) {
    btnCreateReleaseOrder.addEventListener("click", function () {
      void createReleaseOrder();
    });
  }
  if (btnListReleaseOrders) {
    btnListReleaseOrders.addEventListener("click", function () {
      void listReleaseOrders();
    });
  }
  if (btnSelectedListReleaseOrders) {
    btnSelectedListReleaseOrders.addEventListener("click", function () {
      void listReleaseOrders();
    });
  }
  if (btnApproveReleaseOrder) {
    btnApproveReleaseOrder.addEventListener("click", function () {
      void approveReleaseOrder();
    });
  }
  if (btnRejectReleaseOrder) {
    btnRejectReleaseOrder.addEventListener("click", function () {
      void rejectReleaseOrder();
    });
  }
  if (btnSelectedEditManifest) {
    btnSelectedEditManifest.addEventListener("click", function () {
      const item = projectByActionID(selectedProjectKey) || (projectItems.length > 0 ? projectItems[0] : null);
      if (!item) {
        setStatus(t("pluginDev.selected.empty"), true);
        return;
      }
      focusManifestEditorFor(item, selectedProjectKey || "p0");
    });
  }
  if (projectRowsEl) {
    projectRowsEl.addEventListener("click", function (event) {
      const target = event.target && event.target.closest ? event.target.closest("button[data-act]") : null;
      if (target) {
        const action = target.getAttribute("data-act");
        const item = projectByActionID(target.getAttribute("data-id"));
        if (!item) {
          return;
        }
        if (action === "preview") {
          previewProject(item);
          return;
        }
        if (action === "package") {
          void packageProject(item);
          return;
        }
        if (action === "remove") {
          void removeProject(item);
        }
        return;
      }

      const row = event.target && event.target.closest ? event.target.closest("tr[data-row-id]") : null;
      if (!row) {
        return;
      }
      const rowID = row.getAttribute("data-row-id") || "";
      const rowItem = projectByActionID(rowID);
      if (!rowItem) {
        return;
      }
      if (!confirmDiscardUnsavedManifest()) {
        return;
      }
      setManifestDirty(false);
      applyProjectSelection(rowItem, rowID);
      setManifestTab("visual");
      expandFlowSectionById("devPaneManifest");
      void loadManifest(true);
    });
  }

  flowStepButtons.forEach(function (btn) {
    btn.addEventListener("click", function () {
      const target = btn.getAttribute("data-step-target") || "";
      if (!target) {
        return;
      }
      expandFlowSectionById(target);
    });
  });

  flowToggleButtons.forEach(function (btn) {
    btn.addEventListener("click", function () {
      const targetID = btn.getAttribute("data-flow-toggle") || "";
      const section = document.getElementById(targetID);
      if (!section) {
        return;
      }
      const willCollapse = !section.classList.contains("collapsed");
      setFlowSectionCollapsed(section, willCollapse);
    });
  });
  btnApiSend.addEventListener("click", function () {
    void sendApiRequest();
  });
  btnQuickHealth.addEventListener("click", function () {
    activateModule("api-tools");
    void sendApiRequest("/skoll/health", "GET");
  });
  btnQuickMe.addEventListener("click", function () {
    activateModule("api-tools");
    void sendApiRequest("/skoll/v1/auth/me", "GET");
  });
  if (btnRefreshDiagnostics) {
    btnRefreshDiagnostics.addEventListener("click", function () {
      updateLocaleDiagnostics("manual-refresh");
    });
  }
  if (btnResultToggle) {
    btnResultToggle.addEventListener("click", function () {
      resultExpanded = !resultExpanded;
      updateResultDrawer();
    });
  }

  [
    manifestFieldIdEl,
    manifestFieldNameEl,
    manifestFieldNameZhCNEl,
    manifestFieldNameEnUSEl,
    manifestFieldVersionEl,
    manifestFieldDescriptionEl,
    manifestFieldApiVersionEl,
    manifestFieldCompatEl,
    manifestFieldMigrationEl,
    manifestFieldLevelEl,
    manifestFieldAppIdEl,
    manifestFieldUiModeEl,
    manifestFieldMountPolicyEl,
    manifestFieldNavPositionEl,
    manifestFieldOpenModeEl,
    manifestFieldTabModeEl,
    manifestFieldFrontendEntryEl,
    manifestFieldLocalesEl,
    manifestFieldPermissionsCustomEl,
    manifestYamlEl
  ].forEach(function (node) {
    if (!node) {
      return;
    }
    node.addEventListener("input", markManifestDirty);
    node.addEventListener("change", markManifestDirty);
  });

  bindPermissionPresetEvents();

  if (manifestPermissionSearchEl) {
    manifestPermissionSearchEl.addEventListener("input", function () {
      permissionSearchText = String(manifestPermissionSearchEl.value || "").trim();
      renderPermissionPresetOptions();
    });
  }

  if (manifestPermissionEnableCustomEl) {
    manifestPermissionEnableCustomEl.addEventListener("change", function () {
      setCustomPermissionVisibility(Boolean(manifestPermissionEnableCustomEl.checked));
      markManifestDirty();
      renderPermissionCatalog();
      scheduleSyncFromForm();
    });
  }

  if (manifestFieldPermissionsCustomEl) {
    manifestFieldPermissionsCustomEl.addEventListener("input", function () {
      renderPermissionCatalog();
      scheduleSyncFromForm();
    });
    manifestFieldPermissionsCustomEl.addEventListener("change", function () {
      renderPermissionCatalog();
      scheduleSyncFromForm();
    });
  }

  if (manifestYamlEl) {
    manifestYamlEl.addEventListener("input", scheduleSyncFromYaml);
    manifestYamlEl.addEventListener("change", function () {
      try {
        parseManifestTextToForm();
      } catch (err) {
        setStatus(t("status.manifestFormParseFailed"), true);
      }
    });
  }

  [
    manifestFieldIdEl,
    manifestFieldNameEl,
    manifestFieldNameZhCNEl,
    manifestFieldNameEnUSEl,
    manifestFieldVersionEl,
    manifestFieldDescriptionEl,
    manifestFieldApiVersionEl,
    manifestFieldCompatEl,
    manifestFieldMigrationEl,
    manifestFieldLevelEl,
    manifestFieldAppIdEl,
    manifestFieldUiModeEl,
    manifestFieldMountPolicyEl,
    manifestFieldNavPositionEl,
    manifestFieldOpenModeEl,
    manifestFieldTabModeEl,
    manifestFieldFrontendEntryEl,
    manifestFieldLocalesEl
  ].forEach(function (node) {
    if (!node) {
      return;
    }
    node.addEventListener("input", scheduleSyncFromForm);
    node.addEventListener("change", scheduleSyncFromForm);
  });

  if (manifestFieldOpenModeEl) {
    manifestFieldOpenModeEl.addEventListener("change", function () {
      normalizeManifestFieldDependencies();
      scheduleSyncFromForm();
    });
  }
  if (manifestFieldLevelEl) {
    manifestFieldLevelEl.addEventListener("change", function () {
      normalizeManifestFieldDependencies();
      scheduleSyncFromForm();
    });
  }

  window.addEventListener("beforeunload", function (event) {
    if (!manifestDirty) {
      return;
    }
    event.preventDefault();
    event.returnValue = "";
  });

  window.addEventListener("skoll:locale", function (event) {
    const detail = event && event.detail ? event.detail : {};
    applyLocale(detail.locale, "event:skoll:locale");
  });

  window.addEventListener("message", function (event) {
    const data = event && event.data ? event.data : {};
    if (data.type === "skoll:locale") {
      applyLocale(data.locale, "event:postMessage");
    }
  });

  applyLocale(detectInitialLocale(), "bootstrap");
  setFlowStepState("scaffold", "not-started");
  setFlowStepState("select", "not-started");
  setFlowStepState("configure", "not-started");
  setFlowStepState("release", "not-started");
  updateResultDrawer();
  renderPermissionPresetOptions();
  parseManifestTextToForm();
  setDevTab("manifest");
  setManifestTab("visual");
  renderPermissionCatalog();
  void loadPermissionCatalog(selectedPluginId());

  void ensureSuperAdmin().then(function () {
    void loadDevConfig();
    void loadProjects();
  });
})();
