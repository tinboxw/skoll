<script setup lang="ts">
withDefaults(defineProps<{
	modelValue: boolean;
	title: string;
	size?: string | number;
	loading?: boolean;
}>(), {
	size: "58%",
	loading: false
});

const emit = defineEmits<{
	(e: "update:modelValue", value: boolean): void;
	(e: "close"): void;
}>();

function closeDrawer(): void {
	emit("update:modelValue", false);
	emit("close");
}
</script>

<template>
	<el-drawer
		:model-value="modelValue"
		:title="title"
		:size="size"
		class="detail-drawer"
		@update:model-value="(value: boolean) => emit('update:modelValue', value)"
		@close="closeDrawer"
	>
		<template v-if="$slots.header" #header>
			<slot name="header" />
		</template>
		<el-skeleton v-if="loading" :rows="6" animated />
		<div v-else class="detail-drawer__body">
			<slot />
		</div>
		<template v-if="$slots.footer" #footer>
			<div class="detail-drawer__footer">
				<slot name="footer" />
			</div>
		</template>
	</el-drawer>
</template>

<style scoped>
.detail-drawer__body {
	display: grid;
	gap: var(--layout-gap);
}

.detail-drawer__footer {
	display: flex;
	justify-content: flex-end;
	gap: 8px;
	flex-wrap: wrap;
}

@media (max-width: 760px) {
	:deep(.el-drawer) {
		width: 92% !important;
	}
}
</style>
