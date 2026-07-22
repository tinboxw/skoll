<script setup lang="ts">
import { Plus, Save, Send, Trash2, X } from "lucide-vue-next";
import { computed, watch } from "vue";

import DocumentFieldInput from "./DocumentFieldInput.vue";
import { documentKitMessages } from "./messages";
import type {
	DocumentDraft,
	DocumentFormErrors,
	DocumentKitMessages,
	DocumentLine,
	DocumentLineSchema,
	DocumentLocale,
	DocumentSchema,
	DocumentValue
} from "./types";
import { emptyDocumentValue, validateDocumentDraft } from "./validation";

const props = withDefaults(defineProps<{
	schema: DocumentSchema;
	modelValue: DocumentDraft;
	disabled?: boolean;
	busy?: boolean;
	locale?: DocumentLocale;
	messages?: Partial<DocumentKitMessages>;
	showSave?: boolean;
	showSubmit?: boolean;
	showCancel?: boolean;
}>(), {
	disabled: false,
	busy: false,
	locale: "zh-CN",
	messages: () => ({}),
	showSave: true,
	showSubmit: true,
	showCancel: true
});

const emit = defineEmits<{
	(e: "update:modelValue", value: DocumentDraft): void;
	(e: "update:valid", value: boolean): void;
	(e: "update:errors", value: DocumentFormErrors): void;
	(e: "save" | "submit" | "cancel", value: DocumentDraft): void;
}>();

const text = computed(() => documentKitMessages(props.locale, props.messages));
const errors = computed(() => validateDocumentDraft(props.schema, props.modelValue, props.locale, props.messages));
const valid = computed(() => Object.keys(errors.value).length === 0);

watch(errors, (value) => emit("update:errors", value), { immediate: true });
watch(valid, (value) => emit("update:valid", value), { immediate: true });

function updateDraft(patch: Partial<DocumentDraft>): void {
	emit("update:modelValue", { ...props.modelValue, ...patch });
}

function updateHeader(key: string, value: DocumentValue): void {
	updateDraft({ header: { ...props.modelValue.header, [key]: value } });
}

function updateLineValue(lineKey: string, rowIndex: number, fieldKey: string, value: DocumentValue): void {
	const rows = [...(props.modelValue.lines[lineKey] || [])];
	rows[rowIndex] = { ...rows[rowIndex], values: { ...rows[rowIndex].values, [fieldKey]: value } };
	updateDraft({ lines: { ...props.modelValue.lines, [lineKey]: rows } });
}

function addLine(schema: DocumentLineSchema): void {
	const rows = [...(props.modelValue.lines[schema.key] || [])];
	if (rows.length >= schema.maxItems) return;
	const line: DocumentLine = {
		id: `${schema.key}-${Date.now()}-${rows.length + 1}`,
		values: Object.fromEntries(schema.fields.map((field) => [field.key, emptyDocumentValue(field)]))
	};
	updateDraft({ lines: { ...props.modelValue.lines, [schema.key]: [...rows, line] } });
}

function removeLine(schema: DocumentLineSchema, index: number): void {
	const rows = [...(props.modelValue.lines[schema.key] || [])];
	if (rows.length <= schema.minItems) return;
	rows.splice(index, 1);
	updateDraft({ lines: { ...props.modelValue.lines, [schema.key]: rows } });
}
</script>

<template>
	<el-form class="document-form" label-position="top" @submit.prevent>
		<section class="document-form__band" :aria-label="text.documentInfo">
			<div class="document-form__identity">
				<el-form-item :label="text.number" required :error="errors.number">
					<el-input :model-value="modelValue.number" :disabled="disabled" @update:model-value="(value: string) => updateDraft({ number: value })" />
				</el-form-item>
				<el-form-item :label="text.title" required :error="errors.title">
					<el-input :model-value="modelValue.title" :disabled="disabled" @update:model-value="(value: string) => updateDraft({ title: value })" />
				</el-form-item>
			</div>
			<div class="document-form__fields">
				<DocumentFieldInput
					v-for="field in schema.header"
					:key="field.key"
					:field="field"
					:model-value="modelValue.header[field.key]"
					:error="errors[`header.${field.key}`]"
					:disabled="disabled"
					@update:model-value="(value) => updateHeader(field.key, value)"
				/>
			</div>
		</section>

		<section v-for="lineSchema in schema.lines" :key="lineSchema.key" class="document-form__band document-form__lines">
			<header class="document-form__section-header">
				<div>
					<h3>{{ lineSchema.name }}</h3>
					<p v-if="errors[`lines.${lineSchema.key}`]" role="alert">{{ errors[`lines.${lineSchema.key}`] }}</p>
				</div>
				<el-button
					:disabled="disabled || (modelValue.lines[lineSchema.key]?.length || 0) >= lineSchema.maxItems"
					@click="addLine(lineSchema)"
				><Plus :size="16" />{{ text.addLine }}</el-button>
			</header>
			<div class="document-form__table-wrap">
				<table class="document-form__table">
					<thead>
						<tr>
							<th v-for="field in lineSchema.fields" :key="field.key" scope="col">{{ field.label }}<span v-if="field.required" aria-hidden="true"> *</span></th>
							<th scope="col" class="document-form__action-column">{{ text.actions }}</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="(line, rowIndex) in modelValue.lines[lineSchema.key] || []" :key="line.id">
							<td v-for="field in lineSchema.fields" :key="field.key">
								<DocumentFieldInput
									:field="field"
									:model-value="line.values[field.key]"
									:error="errors[`lines.${lineSchema.key}.${rowIndex}.${field.key}`]"
									:disabled="disabled"
									:show-label="false"
									@update:model-value="(value) => updateLineValue(lineSchema.key, rowIndex, field.key, value)"
								/>
							</td>
							<td class="document-form__row-actions">
								<el-tooltip :content="text.removeLine">
									<el-button
										link
										type="danger"
										:aria-label="text.removeLine"
										:disabled="disabled || (modelValue.lines[lineSchema.key]?.length || 0) <= lineSchema.minItems"
										@click="removeLine(lineSchema, rowIndex)"
									><Trash2 :size="17" /></el-button>
								</el-tooltip>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</section>

		<footer class="document-form__footer">
			<el-button v-if="showCancel" :disabled="busy" @click="emit('cancel', modelValue)"><X :size="16" />{{ text.cancel }}</el-button>
			<el-button v-if="showSave" :loading="busy" :disabled="disabled" @click="emit('save', modelValue)"><Save :size="16" />{{ text.saveDraft }}</el-button>
			<el-button v-if="showSubmit" type="primary" :loading="busy" :disabled="disabled || !valid" @click="emit('submit', modelValue)"><Send :size="16" />{{ text.submit }}</el-button>
		</footer>
	</el-form>
</template>

<style scoped>
.document-form {
	display: grid;
	gap: var(--layout-gap);
	min-width: 0;
}

.document-form__band {
	display: grid;
	gap: 16px;
	min-width: 0;
	padding: var(--content-padding);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.document-form__identity,
.document-form__fields {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 14px;
}

.document-form__identity :deep(.el-form-item) {
	margin-bottom: 0;
}

.document-form__section-header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 12px;
}

.document-form__section-header h3,
.document-form__section-header p {
	margin: 0;
}

.document-form__section-header h3 {
	font-size: 1rem;
}

.document-form__section-header p {
	margin-top: 4px;
	color: var(--color-danger);
	font-size: 0.82rem;
}

.document-form__table-wrap {
	width: 100%;
	max-width: 100%;
	min-width: 0;
	overflow-x: auto;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
}

.document-form__table {
	width: 100%;
	min-width: 720px;
	border-collapse: collapse;
}

.document-form__table th,
.document-form__table td {
	padding: 9px;
	border-bottom: 1px solid var(--color-border);
	text-align: left;
	vertical-align: top;
}

.document-form__table th {
	background: var(--color-surface-soft);
	color: var(--color-text-muted);
	font-size: 0.82rem;
	font-weight: 600;
}

.document-form__table tbody tr:last-child td {
	border-bottom: 0;
}

.document-form__action-column,
.document-form__row-actions {
	width: 64px;
	text-align: center !important;
}

.document-form__footer {
	position: sticky;
	bottom: 0;
	z-index: 2;
	display: flex;
	justify-content: flex-end;
	gap: 8px;
	padding: 12px var(--content-padding);
	border-top: 1px solid var(--color-border);
	background: color-mix(in srgb, var(--color-surface) 94%, transparent);
	backdrop-filter: blur(10px);
}

@media (max-width: 760px) {
	.document-form__identity,
	.document-form__fields {
		grid-template-columns: 1fr;
	}

	.document-form__footer {
		position: static;
		justify-content: stretch;
		flex-wrap: wrap;
	}

	.document-form__footer :deep(.el-button) {
		flex: 1 1 130px;
	}
}

@media print {
	.document-form__footer,
	.document-form__section-header :deep(.el-button),
	.document-form__action-column,
	.document-form__row-actions {
		display: none;
	}

	.document-form__band {
		break-inside: avoid;
		border-color: var(--color-border-strong);
	}
}
</style>
