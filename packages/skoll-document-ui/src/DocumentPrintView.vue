<script setup lang="ts">
import { Printer } from "lucide-vue-next";
import { computed } from "vue";

import { documentStateTone, formatDocumentDate, formatDocumentValue } from "./format";
import { documentKitMessages } from "./messages";
import type { DocumentKitMessages, DocumentLocale, DocumentPrintPayload } from "./types";

const props = withDefaults(defineProps<{
	payload: DocumentPrintPayload;
	locale?: DocumentLocale;
	messages?: Partial<DocumentKitMessages>;
	showCommand?: boolean;
}>(), {
	locale: "zh-CN",
	messages: () => ({}),
	showCommand: true
});

const emit = defineEmits<{
	(e: "print"): void;
}>();

const text = computed(() => documentKitMessages(props.locale, props.messages));
const redacted = computed(() => new Set(props.payload.redactedFields || []));
const stateName = computed(() => props.payload.schema.states.find((state) => state.key === props.payload.document.state)?.name || props.payload.document.state);
</script>

<template>
	<article class="document-print" :aria-label="payload.document.title">
		<header class="document-print__header">
			<div>
				<p>{{ payload.schema.name }}</p>
				<h1>{{ payload.document.title }}</h1>
				<span>{{ payload.document.number }}</span>
			</div>
			<div class="document-print__status">
				<el-tag :type="documentStateTone(payload.document.state)">{{ stateName }}</el-tag>
				<el-button v-if="showCommand" type="primary" @click="emit('print')"><Printer :size="16" />{{ text.print }}</el-button>
			</div>
		</header>
		<p v-if="redacted.size" class="document-print__notice">{{ text.sensitiveHidden }}</p>

		<dl class="document-print__fields">
			<div v-for="field in payload.schema.header" :key="field.key">
				<dt>{{ field.label }}</dt>
				<dd>{{ redacted.has(`header.${field.key}`) ? text.redacted : formatDocumentValue(field, payload.document.header[field.key], text) }}</dd>
			</div>
		</dl>

		<section v-for="lineSchema in payload.schema.lines" :key="lineSchema.key" class="document-print__lines">
			<h2>{{ lineSchema.name }}</h2>
			<table>
				<thead><tr><th v-for="field in lineSchema.fields" :key="field.key">{{ field.label }}</th></tr></thead>
				<tbody>
					<tr v-for="line in payload.document.lines[lineSchema.key] || []" :key="line.id">
						<td v-for="field in lineSchema.fields" :key="field.key">
							{{ redacted.has(`lines.${lineSchema.key}.${field.key}`) ? text.redacted : formatDocumentValue(field, line.values[field.key], text) }}
						</td>
					</tr>
				</tbody>
			</table>
		</section>

		<footer>
			<span>{{ text.createdBy }}: {{ payload.document.metadata.createdBy }}</span>
			<span>{{ text.generatedAt }}: {{ formatDocumentDate(payload.generatedAt, locale) }}</span>
		</footer>
	</article>
</template>

<style scoped>
.document-print {
	display: grid;
	gap: 20px;
	max-width: 1040px;
	margin: 0 auto;
	padding: clamp(18px, 3vw, 36px);
	color: var(--color-text);
	background: var(--color-surface);
}

.document-print__header,
.document-print__status,
.document-print footer {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
}

.document-print__header p,
.document-print__header h1 {
	margin: 0;
}

.document-print__header p,
.document-print__header span,
.document-print footer {
	color: var(--color-text-muted);
}

.document-print__header h1 {
	font-size: 1.7rem;
}

.document-print__notice {
	margin: 0;
	padding: 10px 12px;
	border: 1px solid var(--color-warning-border);
	background: var(--color-warning-soft);
	color: var(--color-warning-text);
}

.document-print__fields {
	display: grid;
	grid-template-columns: repeat(3, minmax(0, 1fr));
	gap: 14px;
	margin: 0;
}

.document-print__fields div {
	display: grid;
	gap: 3px;
}

.document-print dt {
	color: var(--color-text-muted);
	font-size: 0.78rem;
}

.document-print dd {
	margin: 0;
	white-space: pre-wrap;
	word-break: break-word;
}

.document-print__lines h2 {
	font-size: 1rem;
}

.document-print table {
	width: 100%;
	border-collapse: collapse;
}

.document-print th,
.document-print td {
	padding: 8px;
	border: 1px solid var(--color-border);
	text-align: left;
}

.document-print th {
	background: var(--color-surface-soft);
	font-size: 0.8rem;
}

.document-print footer {
	padding-top: 12px;
	border-top: 1px solid var(--color-border);
	font-size: 0.8rem;
}

@media (max-width: 680px) {
	.document-print__header,
	.document-print footer {
		align-items: flex-start;
		flex-direction: column;
	}

	.document-print__fields {
		grid-template-columns: 1fr;
	}

	.document-print__status {
		width: 100%;
	}
}

@media print {
	.document-print {
		max-width: none;
		padding: 0;
		color: CanvasText;
		background: Canvas;
	}

	.document-print__status :deep(.el-button) {
		display: none;
	}

	.document-print__fields div,
	.document-print__lines {
		break-inside: avoid;
	}
}
</style>
