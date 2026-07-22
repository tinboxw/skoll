<script setup lang="ts">
import { computed } from "vue";

import type { DocumentFieldSchema, DocumentValue } from "./types";
import { emptyDocumentValue } from "./validation";

const props = withDefaults(defineProps<{
	field: DocumentFieldSchema;
	modelValue?: DocumentValue;
	disabled?: boolean;
	error?: string;
	showLabel?: boolean;
}>(), {
	modelValue: undefined,
	disabled: false,
	error: "",
	showLabel: true
});

const emit = defineEmits<{
	(e: "update:modelValue", value: DocumentValue): void;
}>();

const value = computed(() => props.modelValue?.type === props.field.type ? props.modelValue : emptyDocumentValue(props.field));
const textValue = computed(() => value.value.value || "");
const dateTimeValue = computed<Date | undefined>(() => {
	if (!textValue.value) return undefined;
	const parsed = new Date(textValue.value);
	return Number.isNaN(parsed.getTime()) ? undefined : parsed;
});

function update(patch: Partial<DocumentValue>): void {
	emit("update:modelValue", { ...value.value, ...patch, type: props.field.type });
}

function updateReference(patch: { id?: string; label?: string }): void {
	update({
		reference: {
			type: props.field.referenceType || "record",
			id: value.value.reference?.id || "",
			label: value.value.reference?.label || "",
			...patch
		}
	});
}
</script>

<template>
	<el-form-item
		class="document-field"
		:class="{ 'document-field--unlabeled': !showLabel }"
		:label="showLabel ? field.label : ''"
		:required="field.required"
		:error="error"
	>
		<el-switch
			v-if="field.type === 'boolean'"
			:model-value="textValue === 'true'"
			:disabled="disabled"
			@update:model-value="(next: boolean) => update({ value: String(next) })"
		/>
		<el-input
			v-else-if="field.type === 'text' || field.type === 'json'"
			:model-value="textValue"
			type="textarea"
			:rows="field.type === 'json' ? 5 : 3"
			:disabled="disabled"
			:input-style="field.type === 'json' ? { fontFamily: 'ui-monospace, SFMono-Regular, Consolas, monospace' } : undefined"
			@update:model-value="(next: string) => update({ value: next })"
		/>
		<el-date-picker
			v-else-if="field.type === 'date'"
			:model-value="textValue"
			type="date"
			value-format="YYYY-MM-DD"
			:disabled="disabled"
			@update:model-value="(next: string) => update({ value: next || '' })"
		/>
		<el-date-picker
			v-else-if="field.type === 'datetime'"
			:model-value="dateTimeValue"
			type="datetime"
			:disabled="disabled"
			@update:model-value="(next: Date | null) => update({ value: next ? next.toISOString() : '' })"
		/>
		<div v-else-if="field.type === 'money'" class="document-field__compound document-field__compound--money">
			<el-input
				:model-value="value.currency || ''"
				maxlength="3"
				:disabled="disabled"
				@update:model-value="(next: string) => update({ currency: next.toUpperCase() })"
			/>
			<el-input
				:model-value="textValue"
				inputmode="decimal"
				:disabled="disabled"
				@update:model-value="(next: string) => update({ value: next })"
			/>
		</div>
		<div v-else-if="field.type === 'quantity'" class="document-field__compound">
			<el-input
				:model-value="textValue"
				inputmode="decimal"
				:disabled="disabled"
				@update:model-value="(next: string) => update({ value: next })"
			/>
			<el-input
				:model-value="value.unit || ''"
				maxlength="32"
				:disabled="disabled"
				@update:model-value="(next: string) => update({ unit: next })"
			/>
		</div>
		<div v-else-if="field.type === 'reference'" class="document-field__compound">
			<el-input
				:model-value="value.reference?.id || ''"
				:placeholder="field.referenceType"
				:disabled="disabled"
				@update:model-value="(next: string) => updateReference({ id: next })"
			/>
			<el-input
				:model-value="value.reference?.label || ''"
				:disabled="disabled"
				@update:model-value="(next: string) => updateReference({ label: next })"
			/>
		</div>
		<el-input
			v-else
			:model-value="textValue"
			:inputmode="field.type === 'integer' || field.type === 'decimal' ? 'decimal' : 'text'"
			:disabled="disabled"
			@update:model-value="(next: string) => update({ value: next })"
		/>
	</el-form-item>
</template>

<style scoped>
.document-field {
	min-width: 0;
	margin-bottom: 0;
}

.document-field :deep(.el-form-item__content),
.document-field :deep(.el-date-editor) {
	width: 100%;
}

.document-field--unlabeled :deep(.el-form-item__label) {
	display: none;
}

.document-field__compound {
	display: grid;
	grid-template-columns: minmax(120px, 1fr) minmax(100px, 0.55fr);
	gap: 6px;
	width: 100%;
}

.document-field__compound--money {
	grid-template-columns: 84px minmax(120px, 1fr);
}

@media (max-width: 560px) {
	.document-field__compound {
		grid-template-columns: 1fr;
	}
}
</style>
