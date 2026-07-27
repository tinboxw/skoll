<script setup lang="ts">
import type { BusinessLocale, BusinessState, BusinessTone } from "./types";
import BusinessStateBlock from "./BusinessState.vue";

withDefaults(defineProps<{
	title: string;
	description?: string;
	status?: string;
	statusTone?: BusinessTone;
	state?: BusinessState;
	stateTitle?: string;
	stateDescription?: string;
	locale?: BusinessLocale;
}>(), {
	description: "",
	status: "",
	statusTone: "info",
	state: "ready",
	stateTitle: "",
	stateDescription: "",
	locale: "zh-CN"
});
</script>

<template>
	<main class="business-workspace">
		<header class="business-workspace__header">
			<div class="business-workspace__identity">
				<div class="business-workspace__title">
					<h1>{{ title }}</h1>
					<el-tag v-if="status" :type="statusTone" effect="light">{{ status }}</el-tag>
				</div>
				<p v-if="description">{{ description }}</p>
			</div>
			<div v-if="$slots.actions" class="business-workspace__actions"><slot name="actions" /></div>
		</header>
		<section v-if="$slots.filters" class="business-workspace__filters"><slot name="filters" /></section>
		<BusinessStateBlock
			v-if="state !== 'ready'"
			:state="state"
			:title="stateTitle"
			:description="stateDescription"
			:locale="locale"
		>
			<template v-if="$slots.stateActions" #actions><slot name="stateActions" /></template>
		</BusinessStateBlock>
		<section v-else class="business-workspace__content"><slot /></section>
	</main>
</template>

<style scoped>
.business-workspace { display: grid; gap: var(--layout-gap); min-width: 0; }
.business-workspace__header,
.business-workspace__title,
.business-workspace__actions {
	display: flex;
	align-items: center;
	gap: 10px;
}
.business-workspace__header {
	justify-content: space-between;
	padding-bottom: 14px;
	border-bottom: 1px solid var(--color-border);
}
.business-workspace__identity { min-width: 0; }
.business-workspace__title h1,
.business-workspace__identity p { margin: 0; }
.business-workspace__title h1 { font-size: 1.45rem; line-height: 1.25; }
.business-workspace__identity p { margin-top: 4px; color: var(--color-text-muted); }
.business-workspace__actions { justify-content: flex-end; flex-wrap: wrap; }
.business-workspace__filters,
.business-workspace__content { min-width: 0; }
@media (max-width: 620px) {
	.business-workspace__header { align-items: flex-start; flex-direction: column; }
	.business-workspace__actions { width: 100%; justify-content: flex-start; }
}
</style>
