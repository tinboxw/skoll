<script setup lang="ts">
import { CircleAlert, CircleCheck, FileQuestion, LoaderCircle, ShieldAlert, TriangleAlert } from "lucide-vue-next";
import { computed } from "vue";

import { businessMessages } from "./messages";
import type { BusinessLocale, BusinessState } from "./types";

const props = withDefaults(defineProps<{
	state: Exclude<BusinessState, "ready">;
	title?: string;
	description?: string;
	locale?: BusinessLocale;
}>(), {
	title: "",
	description: "",
	locale: "zh-CN"
});

const copy = computed(() => businessMessages(props.locale).states[props.state]);
const icon = computed(() => {
	if (props.state === "loading") return LoaderCircle;
	if (props.state === "empty") return FileQuestion;
	if (props.state === "success") return CircleCheck;
	if (props.state === "forbidden") return ShieldAlert;
	if (props.state === "conflict" || props.state === "destructive") return TriangleAlert;
	return CircleAlert;
});
const role = computed(() => props.state === "error" || props.state === "forbidden" || props.state === "conflict" || props.state === "destructive" ? "alert" : "status");
</script>

<template>
	<section class="business-state" :class="`business-state--${state}`" :role="role" :aria-busy="state === 'loading'">
		<component :is="icon" :size="24" aria-hidden="true" />
		<div>
			<h3>{{ title || copy.title }}</h3>
			<p>{{ description || copy.description }}</p>
			<div v-if="$slots.actions" class="business-state__actions"><slot name="actions" /></div>
		</div>
	</section>
</template>

<style scoped>
.business-state {
	display: grid;
	grid-template-columns: 28px minmax(0, 1fr);
	gap: 12px;
	align-items: start;
	padding: 18px;
	border: 1px solid var(--color-border);
	border-left: 3px solid var(--color-info);
	border-radius: var(--radius-sm, 4px);
	background: var(--color-surface-soft);
	color: var(--color-text);
}
.business-state h3,
.business-state p { margin: 0; }
.business-state h3 { font-size: 0.98rem; }
.business-state p { margin-top: 4px; color: var(--color-text-muted); }
.business-state__actions { display: flex; gap: 8px; margin-top: 12px; flex-wrap: wrap; }
.business-state--error,
.business-state--destructive { border-left-color: var(--color-danger); }
.business-state--forbidden,
.business-state--conflict { border-left-color: var(--color-warning); }
.business-state--success { border-left-color: var(--color-success); }
.business-state--loading svg { animation: business-spin 1s linear infinite; }
@keyframes business-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .business-state--loading svg { animation: none; } }
</style>
