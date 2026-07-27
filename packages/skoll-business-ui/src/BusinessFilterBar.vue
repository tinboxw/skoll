<script setup lang="ts">
import { RotateCcw, Search } from "lucide-vue-next";
import { computed } from "vue";

import { businessMessages } from "./messages";
import type { BusinessFilterField, BusinessFilterModel, BusinessFilterValue, BusinessLocale } from "./types";

const props = withDefaults(defineProps<{
	fields: readonly BusinessFilterField[];
	modelValue: BusinessFilterModel;
	locale?: BusinessLocale;
	busy?: boolean;
}>(), {
	locale: "zh-CN",
	busy: false
});

const emit = defineEmits<{
	(event: "update:modelValue", value: BusinessFilterModel): void;
	(event: "search" | "reset"): void;
}>();
const copy = computed(() => businessMessages(props.locale));

function update(key: string, value: BusinessFilterValue): void {
	emit("update:modelValue", { ...props.modelValue, [key]: value });
}

function reset(): void {
	emit("update:modelValue", Object.fromEntries(props.fields.map((field) => [field.key, field.defaultValue])));
	emit("reset");
}
</script>

<template>
	<el-form class="business-filter-bar" label-position="top" @submit.prevent="emit('search')">
		<el-form-item v-for="field in fields" :key="field.key" :label="field.label">
			<el-input
				v-if="field.type === 'search' || field.type === 'text'"
				:model-value="String(modelValue[field.key] ?? '')"
				:placeholder="field.placeholder"
				:clearable="field.clearable !== false"
				@update:model-value="(value: string) => update(field.key, value)"
			>
				<template v-if="field.type === 'search'" #prefix><Search :size="16" aria-hidden="true" /></template>
			</el-input>
			<el-select
				v-else-if="field.type === 'select'"
				:model-value="modelValue[field.key]"
				:placeholder="field.placeholder"
				:clearable="field.clearable !== false"
				@update:model-value="(value: BusinessFilterValue) => update(field.key, value)"
			>
				<el-option v-for="option in field.options || []" :key="String(option.value)" :label="option.label" :value="option.value" />
			</el-select>
			<el-date-picker
				v-else
				:model-value="modelValue[field.key]"
				type="date"
				value-format="YYYY-MM-DD"
				:placeholder="field.placeholder"
				:clearable="field.clearable !== false"
				@update:model-value="(value: string) => update(field.key, value)"
			/>
		</el-form-item>
		<div class="business-filter-bar__actions">
			<el-button :icon="RotateCcw" :disabled="busy" @click="reset">{{ copy.reset }}</el-button>
			<el-button native-type="submit" type="primary" :icon="Search" :loading="busy">{{ copy.search }}</el-button>
		</div>
	</el-form>
</template>

<style scoped>
.business-filter-bar {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(min(220px, 100%), 1fr));
	gap: 10px;
	align-items: end;
}
.business-filter-bar :deep(.el-form-item) { min-width: 0; margin: 0; }
.business-filter-bar :deep(.el-form-item__content),
.business-filter-bar :deep(.el-select),
.business-filter-bar :deep(.el-date-editor) { width: 100%; }
.business-filter-bar__actions { display: flex; justify-self: end; gap: 8px; padding-bottom: 1px; }
@media (max-width: 680px) {
	.business-filter-bar { grid-template-columns: 1fr; }
	.business-filter-bar__actions { width: 100%; }
	.business-filter-bar__actions :deep(.el-button) { flex: 1; }
}
</style>
