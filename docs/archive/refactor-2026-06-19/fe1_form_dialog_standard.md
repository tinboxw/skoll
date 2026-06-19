# FE1 Skoll Admin Form And Dialog Standard

日期: 2026-06-19  
范围: FE1-04  
状态: Accepted baseline

## 目标

表单和弹窗必须支持精确输入、明确校验、可见保存状态和可恢复失败。Skoll Admin 的表单体验优先服务系统设置、插件配置、权限策略、菜单编辑、组织字典、Dev Portal 发布任务等高影响操作。

## 表单边界

| 场景 | 推荐承载 | 原因 |
|---|---|---|
| 筛选查询 | 页面内 filter panel | 查询条件需要和数据表保持同一上下文。 |
| 小型创建/编辑 | `el-dialog` 或页面内小表单 | 字段少、可一次完成、关闭影响小。 |
| 插件配置/系统设置 | `SchemaForm` 或页面内分组表单 | schema 驱动、校验和帮助文本可复用。 |
| 长表单/多区块编辑 | `el-drawer` 或独立页面 | 需要滚动、预览、日志、权限或审计提示。 |
| 审计/日志/trace 查看 | `el-drawer` | 长文本不能挤进 dialog 或表格 cell。 |
| 危险确认 | `confirmAction` | 删除、禁用、卸载、回滚、发布等必须阻断确认。 |

## 表单结构

- 默认使用 `label-position="top"`，让中文、英文和插件字段都能稳定换行。
- 字段按用户决策顺序排列: identity、basic info、status、permissions、risk、advanced。
- 同类字段分组，使用 section title、tabs、collapse 或 drawer tabs，不用嵌套卡片。
- 数字字段使用 `el-input-number`，布尔字段使用 switch/checkbox，枚举使用 select。
- 长 JSON、配置预览、日志和 diff 使用 code block 或 textarea，且允许换行和滚动。
- 只在必要处显示 help text；帮助文本必须解释字段影响，不写通用说明。

## 校验

- 必填、长度、范围、pattern 和 enum 必须在提交前可见。
- SchemaForm 字段使用 `update:valid` 和 `update:errors` 向宿主暴露校验状态。
- 保存按钮在表单无效、权限不足、保存中或依赖缺失时禁用。
- 后端错误必须展示在页面、表单或 drawer 内，不允许静默吞掉或显示假成功。
- 失败后保留用户输入，除非后端明确返回不可恢复状态。

## 保存中与结果反馈

- 保存、安装、发布、回滚、导出、校验等操作使用 `saving`、`operating` 或明确 loading state。
- 操作中禁用同一资源的重复提交按钮，并保留取消/关闭策略。
- 成功反馈要说明实际结果，例如 saved、deleted count、export started、task submitted。
- 失败反馈要保留原始错误摘要，并支持用户修改后重试。
- 保存成功后的刷新行为要明确: 刷新当前列表、更新当前详情、关闭 dialog，或保留当前上下文。

## Dialog 标准

Dialog 只承载短流程:

- 删除、禁用、卸载、回滚、发布、清空日志等危险动作使用统一 `confirmAction`。
- 小型表单 dialog 宽度默认 520px 左右，并在窄屏下不超过 viewport。
- Dialog 内容不超过一屏；超过一屏时改 drawer 或独立页面。
- 按钮顺序保持取消在前、确认在后；危险确认使用 danger/warning。
- Dialog 关闭不应丢失已经提交成功的状态，也不应隐藏失败原因。

## Drawer 标准

Drawer 用于较长、较复杂、可边看边操作的上下文:

- 插件配置、Dev Portal 任务详情、审计详情、日志、风险报告优先使用 drawer。
- Drawer 内表单保存中、加载中、失败、无权限状态必须在 drawer 内可见。
- 抽屉宽度使用 `min(560px, 92vw)` 或明确百分比，窄屏不溢出。
- Drawer 内可以使用 tabs，但 tabs 必须对应任务: detail、config、logs、steps、diff。
- 关闭 drawer 后应保留列表筛选和滚动上下文。

## 危险操作

- 危险操作必须明确对象 ID、对象名称或影响范围。
- 高影响动作需要使用 warning/error 类型确认，并使用 danger confirm button。
- 批量危险操作必须显示已选数量或筛选范围。
- 插件 enable/disable/uninstall、Dev Portal rollout/rollback、Audit clear、Menu reset/save 等都属于需要确认或明确保存反馈的动作。

## 当前代码锚点

- `web/src/components/Common/SchemaForm.vue`: schema 字段、校验错误、`update:valid`、`update:errors`。
- `web/src/composables/useConfirmAction.ts`: 统一 Element Plus 确认入口。
- `web/src/views/Audit/index.vue`: `operating`、危险清空确认、详情 drawer。
- `web/src/views/Plugin/index.vue`: 插件配置 SchemaForm、Dev Portal 发布/灰度/回滚确认、任务 drawer。
- `web/src/views/Menu/index.vue`、`Dictionary/index.vue`、`Organization/index.vue`、`Permission/index.vue`: 保存中、错误提示、确认动作的当前参考。

## FE1-04 验收结论

- Validation: Passed，定义了前端校验、SchemaForm valid/errors、保存按钮禁用和后端错误展示。
- Saving state: Passed，定义了 saving/operating/loading、重复提交锁定、成功和失败反馈。
- Error feedback: Passed，要求失败保留输入和错误摘要，不允许静默失败或假成功。
- Confirmation: Passed，定义了危险操作统一 confirmAction、对象和影响范围说明。
- Drawer/dialog boundary: Passed，明确短流程用 dialog，长流程和详情用 drawer 或独立页面。
