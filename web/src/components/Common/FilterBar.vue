<script setup lang="ts">
withDefaults(defineProps<{
	collapsed?: boolean;
}>(), {
	collapsed: false
});
</script>

<template>
	<form class="filter-bar" :class="{ 'filter-bar--collapsed': collapsed }" @submit.prevent>
		<div class="filter-bar__fields">
			<slot />
		</div>
		<div v-if="$slots.actions" class="filter-bar__actions">
			<slot name="actions" />
		</div>
	</form>
</template>

<style scoped>
.filter-bar {
	display: flex;
	align-items: flex-end;
	justify-content: space-between;
	gap: 12px;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.filter-bar__fields {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
	gap: 10px;
	flex: 1;
	min-width: 0;
}

.filter-bar__actions {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 8px;
	flex-wrap: wrap;
}

.filter-bar--collapsed .filter-bar__fields {
	grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
}

@media (max-width: 760px) {
	.filter-bar {
		display: grid;
	}

	.filter-bar__actions {
		justify-content: flex-start;
	}
}
</style>
