# FE2 Architecture Before FE3 Gate

日期: 2026-06-19  
范围: ADJ-FE-20260619-02  
状态: Accepted gate

## 目标

在 FE3 页面体验升级开始前，确认 API client、Pinia store、router/permission、SchemaForm 和页面组件边界已经有可执行标准。FE3 可以逐页改造，但不得绕过这些标准形成新的 page-local API、重复 store、散落权限判断或第二套表单生成器。

## 门禁结论

| 架构域 | 标准文档 | 状态 | FE3 约束 |
|---|---|---|---|
| API client | `docs/refactor/fe2_api_client_standard.md` | Passed | 页面请求先迁入 typed API client，再由页面/store 调用。 |
| Pinia store | `docs/refactor/fe2_pinia_store_standard.md` | Passed | 共享异步状态使用 loading/error/data/retry/refresh 合同。 |
| Router/permission | `docs/refactor/fe2_route_permission_standard.md` | Passed | 菜单、路由、按钮、插件入口使用同源权限规则。 |
| SchemaForm | `docs/refactor/fe2_schema_form_standard.md` | Passed | 插件配置、系统设置、生成器表单复用 SchemaForm，不新增表单引擎。 |
| FE1 UI baseline | `docs/refactor/fe1_component_acceptance_examples.md` | Passed | 页面升级必须覆盖列表、编辑、详情、空态、错误态、无权限态示例。 |

## FE3 页面开工规则

每个 FE3 页面任务开始前必须先回答:

1. 页面是否已有 typed API client；没有则先创建或补齐。
2. 页面状态是 page-local 还是 shared async store；shared 状态必须进入 Pinia store 或明确记录不进入的原因。
3. 页面 route、sidebar、按钮和插件入口权限是否同源；不得在模板中直接写 role/permission includes。
4. 页面是否需要 schema-driven 表单；需要则复用 `SchemaForm`。
5. 页面是否覆盖 FE1 组件验收示例中的对应状态。

## 禁止项

- 禁止新增直接散落在页面中的重复 endpoint 拼接。
- 禁止新增与现有 Pinia store 重叠的第二份状态源。
- 禁止新增旧 route alias、兼容 redirect 或 fallback UI 路径。
- 禁止页面直接判断 `super_admin`、`permissions.includes(...)` 或硬编码角色绕过。
- 禁止新增与 `SchemaForm` 并行的临时 schema 表单生成器。
- 禁止为了快速通过 FE3 而吞掉后端错误或显示假成功。

## FE3 验收附加项

FE3 单页验收除原 Work Item 命令外，至少补充:

```powershell
rg -n "apiGet|apiPost|apiPut|apiDelete|fetch\\(" web/src/views/<Page> web/src/<feature>
rg -n "BUTTON_ACCESS|useButtonAccess|v-permission|canAccess" web/src/views/<Page> web/src/permissions
rg -n "el-empty|el-result|loading|error|retry|refresh" web/src/views/<Page>
```

如果页面使用 SchemaForm，补充:

```powershell
rg -n "SchemaForm|update:valid|update:errors|PluginConfigSchema" web/src/views/<Page> web/src/components/Common/SchemaForm.vue
```

如果页面迁入 store，补充:

```powershell
rg -n "defineStore|syncStatus|lastError|lastQuery|retry|refresh" web/src/stores web/src/views/<Page>
```

## 当前仍开放的 FE2 后续项

| Work Item | 状态 | 说明 |
|---|---|---|
| FE2-05 | Todo | 统一确认动作 helper；影响删除、禁用、回滚、发布。 |
| FE2-06 | Todo | 统一前端错误展示；影响后端错误反馈一致性。 |
| FE2-07 | Todo | 统一导出/下载交互；影响 Audit 和后续列表页导出。 |
| FE2-08 | Todo | 统一空态/错误态/无权限态组件。 |
| FE2-09 | Todo | 前端架构 smoke 检查。 |

这些任务不阻塞 FE3 的架构边界理解，但在对应页面触及确认、错误、导出、状态组件时必须同步引用或提前完成。

## 验收结论

- API client boundary: Passed，FE2-01 已定义 JSON wrapper、typed client、query、response、error 和 export 边界。
- Store boundary: Passed，FE2-02 已定义 shared async state 和 page-local state 边界。
- Router/permission boundary: Passed，FE2-03 已定义 route/menu/button/plugin 权限同源规则。
- SchemaForm boundary: Passed，FE2-04 已定义 schema-driven 表单职责和宿主页面职责。
- FE3 gate: Passed，页面开工规则、禁止项和附加验收命令已明确。
