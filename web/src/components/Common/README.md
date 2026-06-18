# web/src/components/Common

## 功能说明
前端通用组件。当前底层使用 Element Plus，保留轻量封装以承接业务页面迁移。

## 文件组织规范
- 采用小写与下划线命名，按职责拆分文件。
- 接口定义与实现分离，避免单文件过大。
- 变更时同步补充测试与文档。

## 当前规划文件
- `Table.vue`: 兼容原生 table slot，用 Element Plus Card 提供统一外壳。
- `Form.vue`: 基于 `el-form` 的默认纵向表单容器。
- `Dialog.vue`: 基于 `el-dialog` 的弹窗封装。
- `StateBlock.vue`: 统一 empty、error、forbidden 数据区状态，并提供 actions slot。

## 后续待补充实现
- [ ] 根据页面迁移情况补充更明确的 props（分页、加载态、表单校验规则等）。
- [ ] 补充单元测试与必要的集成测试。
- [ ] 完善示例、边界条件与错误处理说明。

