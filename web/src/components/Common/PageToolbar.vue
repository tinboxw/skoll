<script setup lang="ts">
withDefaults(defineProps<{
	title?: string;
	description?: string;
	dense?: boolean;
}>(), {
	title: "",
	description: "",
	dense: false
});
</script>

<template>
	<section class="page-toolbar" :class="{ 'page-toolbar--dense': dense }">
		<div v-if="title || description" class="page-toolbar__copy">
			<h3 v-if="title">{{ title }}</h3>
			<p v-if="description">{{ description }}</p>
		</div>
		<div v-if="$slots.default" class="page-toolbar__main">
			<slot />
		</div>
		<div v-if="$slots.actions" class="page-toolbar__actions">
			<slot name="actions" />
		</div>
	</section>
</template>

<style scoped>
.page-toolbar {
	display: grid;
	grid-template-columns: minmax(0, 1fr) auto;
	align-items: center;
	gap: 12px;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.page-toolbar--dense {
	padding: 8px 10px;
}

.page-toolbar__copy,
.page-toolbar__main,
.page-toolbar__actions {
	min-width: 0;
}

.page-toolbar__copy {
	display: grid;
	gap: 3px;
}

.page-toolbar__copy h3,
.page-toolbar__copy p {
	margin: 0;
}

.page-toolbar__copy h3 {
	font-size: 0.98rem;
}

.page-toolbar__copy p {
	color: var(--color-text-muted);
	font-size: 0.84rem;
}

.page-toolbar__actions {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 8px;
	flex-wrap: wrap;
}

@media (max-width: 760px) {
	.page-toolbar {
		grid-template-columns: 1fr;
	}

	.page-toolbar__actions {
		justify-content: flex-start;
	}
}
</style>
