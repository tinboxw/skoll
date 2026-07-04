<script setup lang="ts">
import StateBlock from "./StateBlock.vue";

withDefaults(defineProps<{
	title: string;
	description?: string;
	eyebrow?: string;
	loading?: boolean;
	error?: string;
	forbidden?: boolean;
	forbiddenTitle?: string;
	forbiddenDescription?: string;
}>(), {
	description: "",
	eyebrow: "",
	loading: false,
	error: "",
	forbidden: false,
	forbiddenTitle: "No permission",
	forbiddenDescription: "You do not have access to this page."
});
</script>

<template>
	<section class="page-shell" :aria-busy="loading ? 'true' : 'false'">
		<header class="page-shell__header">
			<div class="page-shell__title-block">
				<p v-if="eyebrow" class="page-shell__eyebrow">{{ eyebrow }}</p>
				<h2>{{ title }}</h2>
				<p v-if="description" class="page-shell__description">{{ description }}</p>
			</div>
			<div v-if="$slots.actions || $slots.meta" class="page-shell__actions">
				<slot name="meta" />
				<slot name="actions" />
			</div>
		</header>

		<StateBlock
			v-if="forbidden"
			type="forbidden"
			:title="forbiddenTitle"
			:description="forbiddenDescription"
		>
			<template v-if="$slots.stateActions" #actions>
				<slot name="stateActions" />
			</template>
		</StateBlock>
		<StateBlock v-else-if="error" type="error" title="Request failed" :description="error">
			<template v-if="$slots.stateActions" #actions>
				<slot name="stateActions" />
			</template>
		</StateBlock>
		<el-skeleton v-else-if="loading" class="page-shell__loading" :rows="8" animated />
		<div v-else class="page-shell__body">
			<slot />
		</div>
	</section>
</template>

<style scoped>
.page-shell {
	display: grid;
	gap: var(--layout-gap);
}

.page-shell__header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: var(--layout-gap);
}

.page-shell__title-block {
	min-width: 0;
}

.page-shell__eyebrow {
	margin: 0 0 4px;
	color: var(--color-primary);
	font-size: 0.76rem;
	font-weight: 800;
	text-transform: uppercase;
}

.page-shell h2 {
	margin: 0;
	font-size: 1.35rem;
	line-height: 1.25;
}

.page-shell__description {
	margin: 6px 0 0;
	color: var(--color-text-muted);
}

.page-shell__actions {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 8px;
	flex-wrap: wrap;
}

.page-shell__body {
	display: grid;
	gap: var(--layout-gap);
}

.page-shell__loading {
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

@media (max-width: 760px) {
	.page-shell__header,
	.page-shell__actions {
		display: grid;
		justify-content: stretch;
	}
}
</style>
