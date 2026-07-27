# @skoll/business-ui

Skoll 插件业务工作台的 Element Plus 组合组件。该包提供工作台、命令栏、筛选、通用列表和完整状态组件，并复用 `@skoll/document-ui` 的表单、明细行、详情、审批、时间线和附件能力。

业务插件直接组合这些组件，不在宿主 Web 中新增业务页面。

## 使用

```vue
<script setup lang="ts">
import {
	BusinessCommandBar,
	BusinessFilterBar,
	BusinessList,
	BusinessWorkspace
} from "@skoll/business-ui";
</script>

<template>
	<BusinessWorkspace title="采购订单">
		<template #actions><BusinessCommandBar :commands="commands" /></template>
		<template #filters><BusinessFilterBar v-model="filters" :fields="fields" /></template>
		<BusinessList :items="orders" :columns="columns" />
	</BusinessWorkspace>
</template>
```

公共导出包括工作台、命令栏、筛选栏、列表、状态反馈，以及 `BusinessDocumentForm`、
`BusinessDocumentDetail`、`BusinessWorkflowPanel`、`BusinessTimeline`、`BusinessFilePanel`
等单据工作流组件。所有数据由插件传入，组件不依赖宿主页面或具体业务模块。
