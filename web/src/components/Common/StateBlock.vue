<script setup lang="ts">
const props = withDefaults(defineProps<{
	type?: "empty" | "error" | "forbidden";
	title?: string;
	description: string;
}>(), {
	type: "empty",
	title: ""
});

const resultIcon = props.type === "forbidden" ? "warning" : "error";
</script>

<template>
	<div class="state-block" :class="`state-block--${type}`">
		<el-empty v-if="type === 'empty'" :description="description">
			<template v-if="$slots.actions" #default>
				<div class="state-actions">
					<slot name="actions" />
				</div>
			</template>
		</el-empty>
		<el-result
			v-else
			:icon="resultIcon"
			:title="title || description"
			:sub-title="title ? description : ''"
		>
			<template v-if="$slots.actions" #extra>
				<div class="state-actions">
					<slot name="actions" />
				</div>
			</template>
		</el-result>
	</div>
</template>

<style scoped>
.state-block {
	display: grid;
	place-items: center;
	min-height: 180px;
	padding: 18px;
	border: 1px dashed var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

.state-block--forbidden {
	background: var(--color-warning-soft);
}

.state-actions {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 8px;
	flex-wrap: wrap;
}
</style>
