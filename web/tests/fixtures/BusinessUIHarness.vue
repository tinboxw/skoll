<script setup lang="ts">
import { reactive } from "vue";
import { Plus, Trash2 } from "lucide-vue-next";

import BusinessCommandBar from "../../../packages/skoll-business-ui/src/BusinessCommandBar.vue";
import BusinessFilterBar from "../../../packages/skoll-business-ui/src/BusinessFilterBar.vue";
import BusinessList from "../../../packages/skoll-business-ui/src/BusinessList.vue";
import BusinessState from "../../../packages/skoll-business-ui/src/BusinessState.vue";
import BusinessWorkspace from "../../../packages/skoll-business-ui/src/BusinessWorkspace.vue";
import type { BusinessLocale } from "../../../packages/skoll-business-ui/src/types";

const props = defineProps<{ locale: BusinessLocale }>();
const english = props.locale === "en-US";
const filters = reactive({ keyword: "", status: "" });
const fields = [
	{ key: "keyword", label: english ? "Keyword" : "关键词", type: "search" as const },
	{
		key: "status",
		label: english ? "Status" : "状态",
		type: "select" as const,
		options: [
			{ label: english ? "Pending" : "待审批", value: "pending" },
			{ label: english ? "Approved" : "已通过", value: "approved" }
		]
	}
];
const columns = [
	{ key: "number", label: english ? "Number" : "单号", minWidth: 140 },
	{ key: "customer", label: english ? "Customer" : "客户", minWidth: 180 },
	{ key: "status", label: english ? "Status" : "状态", width: 120 }
];
const items = [
	{ id: "1", number: "SO-20260727-001", customer: english ? "Jiangnan Medical" : "江南医药", status: english ? "Pending" : "待审批" },
	{ id: "2", number: "SO-20260727-002", customer: english ? "Beichen Pharmacy" : "北辰药房", status: english ? "Approved" : "已通过" }
];
const commands = [
	{ id: "create", label: english ? "Create" : "新建", icon: Plus, tone: "primary" as const },
	{
		id: "delete",
		label: english ? "Delete" : "删除",
		icon: Trash2,
		destructive: true,
		confirmation: english ? "Delete the selected order?" : "确认删除所选订单？"
	}
];
</script>

<template>
	<div class="fixture-surface">
		<BusinessWorkspace
			:title="english ? 'Sales orders' : '销售订单'"
			:description="english ? 'Medical distribution order workspace' : '医药流通订单工作台'"
			:status="english ? '2 pending' : '2 项待处理'"
			status-tone="warning"
			:locale="locale"
		>
			<template #actions>
				<BusinessCommandBar :commands="commands" :locale="locale" />
			</template>
			<template #filters>
				<BusinessFilterBar v-model="filters" :fields="fields" :locale="locale" />
			</template>
			<BusinessList :items="items" :columns="columns" :locale="locale" can-next />
			<BusinessState state="success" :locale="locale" />
		</BusinessWorkspace>
	</div>
</template>

<style scoped>
.fixture-surface {
	min-height: 100vh;
	padding: var(--content-padding);
}
</style>
