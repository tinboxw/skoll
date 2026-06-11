<script setup lang="ts">
import { computed, watch } from "vue";

import type { PluginConfigField, PluginConfigOption, PluginConfigSchema } from "../../plugins/types";
import type { Locale } from "../../i18n";

const props = defineProps<{
	schema?: PluginConfigSchema | null;
	modelValue: Record<string, unknown>;
	disabled?: boolean;
	locale?: Locale;
}>();

const emit = defineEmits<{
	(e: "update:modelValue", value: Record<string, unknown>): void;
	(e: "update:valid", value: boolean): void;
	(e: "update:errors", value: Record<string, string>): void;
}>();

const fields = computed(() => props.schema?.fields ?? []);
const validationErrors = computed(() => {
	const errors: Record<string, string> = {};
	for (const field of fields.value) {
		const message = validateField(field, fieldValue(field));
		if (message !== "") {
			errors[field.key] = message;
		}
	}
	return errors;
});
const isValid = computed(() => Object.keys(validationErrors.value).length === 0);

watch(validationErrors, (next) => emit("update:errors", next), { immediate: true });
watch(isValid, (next) => emit("update:valid", next), { immediate: true });

function localizedLabel(item: { label?: string; labelZhCN?: string; labelEnUS?: string }, fallback: string): string {
	if (props.locale === "en-US") {
		return item.labelEnUS || item.label || fallback;
	}
	return item.labelZhCN || item.label || fallback;
}

function fieldValue(field: PluginConfigField): unknown {
	const current = props.modelValue[field.key];
	if (current !== undefined) {
		return current;
	}
	return coerceValue(field.default, field.type);
}

function coerceValue(value: unknown, type?: PluginConfigField["type"]): unknown {
	if (type === "boolean") {
		return value === true || value === "true" || value === "1";
	}
	if (type === "number") {
		const parsed = Number(value ?? 0);
		return Number.isFinite(parsed) ? parsed : 0;
	}
	return value ?? "";
}

function updateField(field: PluginConfigField, value: unknown): void {
	emit("update:modelValue", {
		...props.modelValue,
		[field.key]: coerceValue(value, field.type)
	});
}

function optionLabel(option: PluginConfigOption): string {
	return localizedLabel(option, option.value);
}

function validateField(field: PluginConfigField, value: unknown): string {
	const label = localizedLabel(field, field.key);
	if (field.required && field.type !== "boolean" && String(value ?? "").trim() === "") {
		return props.locale === "en-US" ? `${label} is required` : `${label}不能为空`;
	}
	if (field.type === "number") {
		const numeric = Number(value);
		if (!Number.isFinite(numeric)) {
			return props.locale === "en-US" ? `${label} must be a number` : `${label}必须是数字`;
		}
		if (typeof field.min === "number" && numeric < field.min) {
			return props.locale === "en-US" ? `${label} must be >= ${field.min}` : `${label}不能小于 ${field.min}`;
		}
		if (typeof field.max === "number" && numeric > field.max) {
			return props.locale === "en-US" ? `${label} must be <= ${field.max}` : `${label}不能大于 ${field.max}`;
		}
	}
	if (field.type !== "number" && field.type !== "boolean") {
		const text = String(value ?? "");
		if (typeof field.minLength === "number" && text.length < field.minLength) {
			return props.locale === "en-US" ? `${label} length must be >= ${field.minLength}` : `${label}长度不能小于 ${field.minLength}`;
		}
		if (typeof field.maxLength === "number" && text.length > field.maxLength) {
			return props.locale === "en-US" ? `${label} length must be <= ${field.maxLength}` : `${label}长度不能大于 ${field.maxLength}`;
		}
		if (field.pattern) {
			try {
				if (!new RegExp(field.pattern).test(text)) {
					return props.locale === "en-US" ? `${label} format is invalid` : `${label}格式不正确`;
				}
			} catch {
				return props.locale === "en-US" ? `${label} pattern is invalid` : `${label}校验规则不正确`;
			}
		}
	}
	return "";
}
</script>

<template>
	<el-form v-if="fields.length > 0" label-position="top" class="schema-form">
		<el-form-item
			v-for="field in fields"
			:key="field.key"
			:label="localizedLabel(field, field.key)"
			:required="field.required"
			:error="validationErrors[field.key]"
		>
			<el-switch
				v-if="field.type === 'boolean'"
				:model-value="Boolean(fieldValue(field))"
				:disabled="disabled"
				@update:model-value="(value: boolean) => updateField(field, value)"
			/>
			<el-input-number
				v-else-if="field.type === 'number'"
				:model-value="Number(fieldValue(field))"
				:disabled="disabled"
				:min="field.min"
				:max="field.max"
				controls-position="right"
				@update:model-value="(value: number | undefined) => updateField(field, value ?? 0)"
			/>
			<el-select
				v-else-if="field.type === 'select'"
				:model-value="String(fieldValue(field))"
				:disabled="disabled"
				filterable
				@update:model-value="(value: string) => updateField(field, value)"
			>
				<el-option v-for="option in field.options ?? []" :key="option.value" :value="option.value" :label="optionLabel(option)" />
			</el-select>
			<el-input
				v-else-if="field.type === 'textarea'"
				:model-value="String(fieldValue(field))"
				type="textarea"
				:rows="4"
				:placeholder="field.placeholder"
				:maxlength="field.maxLength"
				:show-word-limit="typeof field.maxLength === 'number'"
				:disabled="disabled"
				@update:model-value="(value: string) => updateField(field, value)"
			/>
			<el-input
				v-else
				:model-value="String(fieldValue(field))"
				:placeholder="field.placeholder"
				:maxlength="field.maxLength"
				:show-word-limit="typeof field.maxLength === 'number'"
				:disabled="disabled"
				@update:model-value="(value: string) => updateField(field, value)"
			/>
			<p v-if="field.help" class="field-help">{{ field.help }}</p>
		</el-form-item>
	</el-form>
</template>

<style scoped>
.schema-form {
	display: grid;
	gap: 4px;
}

.schema-form :deep(.el-select),
.schema-form :deep(.el-input-number) {
	width: 100%;
}

.field-help {
	margin: 6px 0 0;
	color: var(--color-text-muted);
	font-size: 0.82rem;
}
</style>
