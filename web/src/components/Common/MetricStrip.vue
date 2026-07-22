<script setup lang="ts">
export interface MetricStripItem {
	label: string;
	value: string | number;
	note?: string;
	tone?: "default" | "warning";
	wide?: boolean;
}

defineProps<{
	items: MetricStripItem[];
}>();
</script>

<template>
	<dl class="metric-strip">
		<div
			v-for="item in items"
			:key="item.label"
			class="metric-strip__item"
			:class="{
				'metric-strip__item--warning': item.tone === 'warning',
				'metric-strip__item--wide': item.wide
			}"
		>
			<dt>{{ item.label }}</dt>
			<dd>{{ item.value }}</dd>
			<p v-if="item.note">{{ item.note }}</p>
		</div>
	</dl>
</template>

<style scoped>
.metric-strip {
	display: grid;
	grid-template-columns: repeat(5, minmax(0, 1fr));
	margin: 0;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
	overflow: hidden;
}

.metric-strip__item {
	display: grid;
	align-content: start;
	gap: 5px;
	min-width: 0;
	padding: 14px;
	border-right: 1px solid var(--color-border);
}

.metric-strip__item:last-child,
.metric-strip__item--wide {
	border-right: 0;
}

.metric-strip__item--wide {
	grid-column: 1 / -1;
	border-top: 1px solid var(--color-border);
}

.metric-strip__item--warning dd {
	color: var(--el-color-warning-dark-2);
}

.metric-strip dt,
.metric-strip dd,
.metric-strip p {
	margin: 0;
}

.metric-strip dt,
.metric-strip p {
	color: var(--color-text-muted);
}

.metric-strip dt {
	font-size: 0.78rem;
	font-weight: 700;
}

.metric-strip dd {
	min-width: 0;
	font-size: 1.45rem;
	font-weight: 800;
	line-height: 1.15;
	overflow-wrap: anywhere;
}

.metric-strip p {
	font-size: 0.78rem;
	line-height: 1.45;
}

@media (max-width: 980px) {
	.metric-strip {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	.metric-strip__item {
		border-bottom: 1px solid var(--color-border);
	}

	.metric-strip__item:nth-child(even) {
		border-right: 0;
	}

	.metric-strip__item--wide {
		border-bottom: 0;
	}
}

@media (max-width: 560px) {
	.metric-strip {
		grid-template-columns: 1fr;
	}

	.metric-strip__item {
		border-right: 0;
	}
}
</style>
