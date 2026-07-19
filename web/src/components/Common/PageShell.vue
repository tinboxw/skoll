<script setup lang="ts">
import { computed, getCurrentInstance } from "vue";

import { localizeKnownError, useI18n } from "../../i18n";
import StateBlock from "./StateBlock.vue";

const props = withDefaults(defineProps<{
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
	forbiddenTitle: "",
	forbiddenDescription: ""
});

const { t } = useI18n();
const localizedError = computed(() => localizeKnownError(props.error));
const titleId = `page-shell-title-${getCurrentInstance()?.uid ?? "default"}`;
</script>

<template>
	<section class="page-shell" :aria-busy="loading ? 'true' : 'false'" :aria-labelledby="titleId">
		<header class="page-shell__header">
			<div class="page-shell__title-block">
				<p v-if="eyebrow" class="page-shell__eyebrow">{{ eyebrow }}</p>
				<h2 :id="titleId">{{ title }}</h2>
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
			:title="forbiddenTitle || t('state.noPermission')"
			:description="forbiddenDescription || t('state.noPagePermission')"
		>
			<template v-if="$slots.stateActions" #actions>
				<slot name="stateActions" />
			</template>
		</StateBlock>
		<StateBlock v-else-if="error" type="error" :title="t('state.requestFailed')" :description="localizedError">
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
