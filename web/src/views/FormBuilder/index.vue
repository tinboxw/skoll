<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { Copy, Eye, FileText, Pencil, Plus, RefreshCw, Save, Trash2 } from "lucide-vue-next";
import { ElMessage } from "element-plus";

import { ConfirmAction, DataTable, DetailDrawer, FilterBar, PageShell, PageToolbar, type DataTableColumn } from "../../components/Common";
import { useI18n } from "../../i18n";
import { useButtonAccess } from "../../permissions/button";
import {
	FORM_FIELD_TYPE_OPTIONS,
	cloneFormSchema,
	createBlankSchema,
	createFormID,
	createPurchaseRequestSchema,
	deleteFormSchema,
	loadFormSchemas,
	upsertFormSchema,
	validateFormSchema,
	type FormField,
	type FormFieldType,
	type FormSchema
} from "../../form-builder/types";
import { toErrorMessage } from "../../utils/common";

type SchemaRow = Record<string, unknown> & {
	id: string;
	name: string;
	key: string;
	version: string;
	businessType: string;
	fields: string;
	updatedAt: string;
	raw: FormSchema;
};

type FieldRow = Record<string, unknown> & {
	key: string;
	label: string;
	type: string;
	required: string;
	raw: FormField;
	index: number;
};

const { t } = useI18n();
const buttonAccess = useButtonAccess();
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const keyword = ref("");
const schemas = ref<FormSchema[]>([]);
const selectedId = ref("");
const fieldDrawerOpen = ref(false);
const editingFieldIndex = ref(-1);
const activePane = ref<"design" | "preview" | "versions">("design");

const draft = reactive<FormSchema>(createBlankSchema());
const fieldForm = reactive({
	key: "",
	label: "",
	type: "string" as FormFieldType,
	required: false,
	dictionary: "",
	optionsText: "",
	maxFiles: 3,
	maxSizeMB: 10,
	acceptText: ".pdf, .jpg, .png",
	minRows: 1,
	maxRows: 20,
	columnsText: t("formBuilder.columnsPlaceholder")
});

const canManage = computed(() => buttonAccess.can("form.schema.manage"));
const selectedSchema = computed(() => schemas.value.find((item) => item.id === selectedId.value) || null);
const validationErrors = computed(() => validateFormSchema(draft));
const summary = computed(() => ({
	schemas: schemas.value.length,
	fields: draft.fields.length,
	versions: schemas.value.filter((item) => item.key === draft.key).length,
	required: draft.fields.filter((item) => item.required).length
}));

const schemaColumns = computed<DataTableColumn[]>(() => [
	{ key: "name", label: t("formBuilder.column.name"), minWidth: 180 },
	{ key: "key", label: t("formBuilder.column.key"), minWidth: 180 },
	{ key: "version", label: t("formBuilder.column.version"), width: 90 },
	{ key: "businessType", label: t("formBuilder.column.business"), minWidth: 150 },
	{ key: "fields", label: t("formBuilder.column.fields"), width: 90 },
	{ key: "updatedAt", label: t("formBuilder.column.updated"), minWidth: 160 }
]);

const fieldColumns = computed<DataTableColumn[]>(() => [
	{ key: "label", label: t("formBuilder.column.label"), minWidth: 160 },
	{ key: "key", label: t("formBuilder.column.key"), minWidth: 140 },
	{ key: "type", label: t("formBuilder.column.type"), minWidth: 120 },
	{ key: "required", label: t("formBuilder.column.required"), width: 110 }
]);
const fieldTypeOptions = computed(() => FORM_FIELD_TYPE_OPTIONS.map((item) => ({
	...item,
	label: t(`formBuilder.fieldType.${item.value}`)
})));

const schemaRows = computed<SchemaRow[]>(() => {
	const q = keyword.value.trim().toLowerCase();
	return schemas.value
		.map(toSchemaRow)
		.filter((row) => !q || [row.name, row.key, row.businessType].some((item) => item.toLowerCase().includes(q)));
});
const fieldRows = computed<FieldRow[]>(() => draft.fields.map((field, index) => toFieldRow(field, index)));

onMounted(() => {
	refreshSchemas();
});

function refreshSchemas(): void {
	error.value = "";
	if (!canManage.value) {
		return;
	}
	loading.value = true;
	try {
		schemas.value = loadFormSchemas();
		if (!selectedId.value && schemas.value.length > 0) {
			selectSchema(schemas.value[0]);
		}
	} catch (e) {
		error.value = toErrorMessage(e);
		schemas.value = [];
	} finally {
		loading.value = false;
	}
}

function selectSchema(schema: FormSchema): void {
	selectedId.value = schema.id;
	const copy = cloneFormSchema(schema);
	Object.assign(draft, copy);
	activePane.value = "design";
}

function newSchema(): void {
	const blank = createBlankSchema();
	Object.assign(draft, blank);
	selectedId.value = "";
	activePane.value = "design";
}

function usePurchaseTemplate(): void {
	const template = createPurchaseRequestSchema();
	Object.assign(draft, template);
	selectedId.value = "";
	activePane.value = "preview";
}

function duplicateVersion(): void {
	const copy = cloneFormSchema(draft);
	copy.id = createFormID("form");
	copy.version += 1;
	copy.name = `${copy.name} v${copy.version}`;
	copy.createdAt = new Date().toISOString();
	copy.updatedAt = copy.createdAt;
	Object.assign(draft, copy);
	selectedId.value = "";
	activePane.value = "design";
}

async function saveDraft(): Promise<void> {
	error.value = "";
	if (!canManage.value) {
		error.value = t("error.forbidden");
		return;
	}
	if (validationErrors.value.length > 0) {
		error.value = validationErrors.value[0];
		return;
	}
	saving.value = true;
	try {
		await waitForSavingState();
		const saved = upsertFormSchema(draft);
		schemas.value = loadFormSchemas();
		selectSchema(saved);
		ElMessage.success(t("formBuilder.saved"));
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

function removeSelected(): void {
	if (!selectedId.value) {
		return;
	}
	deleteFormSchema(selectedId.value);
	schemas.value = loadFormSchemas();
	if (schemas.value.length > 0) {
		selectSchema(schemas.value[0]);
	} else {
		newSchema();
	}
	ElMessage.success(t("formBuilder.deleted"));
}

function openFieldEditor(row?: FieldRow): void {
	if (row) {
		const field = row.raw;
		editingFieldIndex.value = row.index;
		fieldForm.key = field.key;
		fieldForm.label = field.label;
		fieldForm.type = field.type;
		fieldForm.required = Boolean(field.required);
		fieldForm.dictionary = field.dictionary || "";
		fieldForm.optionsText = (field.options || []).map((item) => `${item.label}=${item.value}`).join("\n");
		fieldForm.maxFiles = field.attachment?.maxFiles || 3;
		fieldForm.maxSizeMB = field.attachment?.maxSizeMB || 10;
		fieldForm.acceptText = (field.attachment?.accept || [".pdf", ".jpg", ".png"]).join(", ");
		fieldForm.minRows = field.detailTable?.minRows || 1;
		fieldForm.maxRows = field.detailTable?.maxRows || 20;
		fieldForm.columnsText = (field.detailTable?.columns || []).map((item) => `${item.key}:${item.label}:${item.type}`).join("\n") || t("formBuilder.columnsPlaceholder");
	} else {
		editingFieldIndex.value = -1;
		fieldForm.key = "";
		fieldForm.label = "";
		fieldForm.type = "string";
		fieldForm.required = false;
		fieldForm.dictionary = "";
		fieldForm.optionsText = "";
		fieldForm.maxFiles = 3;
		fieldForm.maxSizeMB = 10;
		fieldForm.acceptText = ".pdf, .jpg, .png";
		fieldForm.minRows = 1;
		fieldForm.maxRows = 20;
		fieldForm.columnsText = t("formBuilder.columnsPlaceholder");
	}
	fieldDrawerOpen.value = true;
}

function saveField(): void {
	const field = buildFieldFromForm();
	const index = editingFieldIndex.value;
	if (index >= 0) {
		draft.fields.splice(index, 1, field);
	} else {
		draft.fields.push({ ...field, displayOrder: (draft.fields.length + 1) * 10 });
	}
	draft.fields = draft.fields.map((item, order) => ({ ...item, displayOrder: (order + 1) * 10 }));
	fieldDrawerOpen.value = false;
}

function removeField(row: FieldRow): void {
	draft.fields.splice(row.index, 1);
	draft.fields = draft.fields.map((item, order) => ({ ...item, displayOrder: (order + 1) * 10 }));
}

function moveField(row: FieldRow, direction: -1 | 1): void {
	const target = row.index + direction;
	if (target < 0 || target >= draft.fields.length) {
		return;
	}
	const copy = [...draft.fields];
	const [item] = copy.splice(row.index, 1);
	copy.splice(target, 0, item);
	draft.fields = copy.map((field, order) => ({ ...field, displayOrder: (order + 1) * 10 }));
}

function buildFieldFromForm(): FormField {
	const base: FormField = {
		key: fieldForm.key.trim(),
		label: fieldForm.label.trim(),
		type: fieldForm.type,
		required: fieldForm.required
	};
	if (fieldForm.type === "select" || fieldForm.type === "multi_select") {
		base.options = parseOptions(fieldForm.optionsText);
	}
	if (fieldForm.type === "dictionary") {
		base.dictionary = fieldForm.dictionary.trim();
	}
	if (fieldForm.type === "attachment") {
		base.attachment = {
			maxFiles: Math.max(1, Number(fieldForm.maxFiles)),
			maxSizeMB: Math.max(1, Number(fieldForm.maxSizeMB)),
			accept: fieldForm.acceptText.split(",").map((item) => item.trim()).filter(Boolean),
			required: fieldForm.required
		};
	}
	if (fieldForm.type === "detail_table") {
		base.detailTable = {
			minRows: Math.max(0, Number(fieldForm.minRows)),
			maxRows: Math.max(1, Number(fieldForm.maxRows)),
			columns: parseColumns(fieldForm.columnsText)
		};
	}
	return base;
}

function parseOptions(value: string) {
	return value
		.split(/\r?\n|,/)
		.map((line) => line.trim())
		.filter(Boolean)
		.map((line) => {
			const [label, rawValue] = line.includes("=") ? line.split("=", 2) : [line, line];
			return { label: label.trim(), value: rawValue.trim() };
		})
		.filter((item) => item.label && item.value);
}

function parseColumns(value: string): FormField[] {
	return value
		.split(/\r?\n/)
		.map((line) => line.trim())
		.filter(Boolean)
		.map((line, index) => {
			const [key = "", label = "", type = "string"] = line.split(":").map((item) => item.trim());
			const normalizedType = FORM_FIELD_TYPE_OPTIONS.some((item) => item.value === type) && type !== "detail_table" ? type as FormFieldType : "string";
			return { key, label: label || key, type: normalizedType, displayOrder: (index + 1) * 10 };
		});
}

function toSchemaRow(schema: FormSchema): SchemaRow {
	return {
		id: schema.id,
		name: schema.name,
		key: schema.key,
		version: `v${schema.version}`,
		businessType: schema.businessType,
		fields: String(schema.fields.length),
		updatedAt: formatDate(schema.updatedAt),
		raw: schema
	};
}

function toFieldRow(field: FormField, index: number): FieldRow {
	return {
		key: field.key,
		label: field.label,
		type: field.type,
		required: field.required ? t("common.yes") : t("common.no"),
		raw: field,
		index
	};
}

function fieldPlaceholder(field: FormField): string {
	if (field.type === "attachment") {
		return t("formBuilder.preview.attachment", { accept: field.attachment?.accept?.join(", ") || "*", max: field.attachment?.maxFiles || 1 });
	}
	if (field.type === "detail_table") {
		return t("formBuilder.preview.detailTable", { columns: field.detailTable?.columns.length || 0, min: field.detailTable?.minRows || 0, max: field.detailTable?.maxRows || "n" });
	}
	if (field.type === "dictionary") {
		return field.dictionary || "dictionary.code";
	}
	if (field.type === "user") {
		return t("formBuilder.preview.selectUser");
	}
	if (field.type === "department") {
		return t("formBuilder.preview.selectDepartment");
	}
	return field.type;
}

function formatDate(value: string): string {
	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function waitForSavingState(): Promise<void> {
	return new Promise((resolve) => window.setTimeout(resolve, 120));
}
</script>

<template>
	<PageShell
		:title="t('formBuilder.title')"
		:description="t('formBuilder.description')"
		:loading="loading"
		:error="error"
		:forbidden="!canManage"
		:forbidden-title="t('formBuilder.unavailable')"
		:forbidden-description="t('formBuilder.permissionRequired')"
		data-testid="form-builder-page"
	>
		<template #actions>
			<el-button :icon="RefreshCw" :loading="loading" @click="refreshSchemas">{{ t("formBuilder.refresh") }}</el-button>
			<el-button :icon="Plus" @click="newSchema">{{ t("formBuilder.new") }}</el-button>
			<el-button :icon="FileText" @click="usePurchaseTemplate">{{ t("formBuilder.template") }}</el-button>
			<el-button :icon="Copy" :disabled="draft.fields.length === 0" @click="duplicateVersion">{{ t("formBuilder.newVersion") }}</el-button>
			<ConfirmAction
				:label="t('formBuilder.delete')"
				:message="t('formBuilder.deleteConfirm')"
				:disabled="!selectedId"
				@confirm="removeSelected"
			/>
			<el-button type="primary" :icon="Save" :loading="saving" @click="saveDraft">{{ t("formBuilder.save") }}</el-button>
		</template>
		<template #stateActions>
			<el-button :icon="RefreshCw" @click="refreshSchemas">{{ t("formBuilder.retry") }}</el-button>
		</template>

		<section class="form-summary" :aria-label="t('formBuilder.summary')">
			<div class="summary-tile">
				<span>{{ t("formBuilder.summary.schemas") }}</span>
				<strong>{{ summary.schemas }}</strong>
			</div>
			<div class="summary-tile">
				<span>{{ t("formBuilder.summary.currentFields") }}</span>
				<strong>{{ summary.fields }}</strong>
			</div>
			<div class="summary-tile">
				<span>{{ t("formBuilder.summary.versions") }}</span>
				<strong>{{ summary.versions }}</strong>
			</div>
			<div class="summary-tile">
				<span>{{ t("formBuilder.summary.required") }}</span>
				<strong>{{ summary.required }}</strong>
			</div>
		</section>

		<div class="builder-layout">
			<section class="schema-panel" :aria-label="t('formBuilder.schemas')">
				<PageToolbar>
					<FilterBar>
						<el-input v-model="keyword" clearable :placeholder="t('formBuilder.searchPlaceholder')" />
					</FilterBar>
				</PageToolbar>
				<DataTable
					:rows="schemaRows"
					:columns="schemaColumns"
					row-key="id"
					:empty-text="t('formBuilder.emptySchemas')"
					data-testid="form-schema-table"
				>
					<template #cell-name="{ row }">
						<el-button link class="schema-link" @click="selectSchema(row.raw)">
							<FileText class="cell-icon" aria-hidden="true" />
							<span>{{ row.name }}</span>
						</el-button>
					</template>
					<template #actions="{ row }">
						<el-tooltip :content="t('formBuilder.editSchema')">
							<el-button :icon="Pencil" circle :aria-label="t('formBuilder.editSchema')" @click="selectSchema(row.raw)" />
						</el-tooltip>
					</template>
				</DataTable>
			</section>

			<section class="designer-panel" :aria-label="t('formBuilder.designer')">
				<el-tabs v-model="activePane">
					<el-tab-pane :label="t('formBuilder.tab.design')" name="design">
						<el-form class="schema-form" label-position="top">
							<el-form-item :label="t('formBuilder.formName')">
								<el-input v-model="draft.name" />
							</el-form-item>
							<el-form-item :label="t('formBuilder.formKey')">
								<el-input v-model="draft.key" />
							</el-form-item>
							<el-form-item :label="t('formBuilder.version')">
								<el-input-number v-model="draft.version" :min="1" :step="1" />
							</el-form-item>
							<el-form-item :label="t('formBuilder.businessType')">
								<el-input v-model="draft.businessType" />
							</el-form-item>
							<el-form-item class="wide" :label="t('formBuilder.descriptionField')">
								<el-input v-model="draft.description" type="textarea" :rows="2" />
							</el-form-item>
						</el-form>

						<PageToolbar>
							<template #left>
								<strong>{{ t("formBuilder.fields") }}</strong>
							</template>
							<el-button type="primary" :icon="Plus" @click="openFieldEditor()">{{ t("formBuilder.addField") }}</el-button>
						</PageToolbar>
						<DataTable
							:rows="fieldRows"
							:columns="fieldColumns"
							row-key="key"
							:empty-text="t('formBuilder.emptyFields')"
							data-testid="form-field-table"
						>
							<template #cell-required="{ row, value }">
								<el-tag :type="row.raw.required ? 'danger' : 'info'">{{ value }}</el-tag>
							</template>
							<template #actions="{ row }">
								<el-tooltip :content="t('formBuilder.moveUp')">
									<el-button text :disabled="row.index === 0" @click="moveField(row, -1)">{{ t("formBuilder.moveUp") }}</el-button>
								</el-tooltip>
								<el-tooltip :content="t('formBuilder.moveDown')">
									<el-button text :disabled="row.index === fieldRows.length - 1" @click="moveField(row, 1)">{{ t("formBuilder.moveDown") }}</el-button>
								</el-tooltip>
								<el-tooltip :content="t('formBuilder.editField')">
									<el-button :icon="Pencil" circle :aria-label="t('formBuilder.editField')" @click="openFieldEditor(row)" />
								</el-tooltip>
								<el-tooltip :content="t('formBuilder.deleteField')">
									<el-button :icon="Trash2" circle type="danger" :aria-label="t('formBuilder.deleteField')" @click="removeField(row)" />
								</el-tooltip>
							</template>
						</DataTable>
					</el-tab-pane>

					<el-tab-pane :label="t('formBuilder.tab.preview')" name="preview">
						<div class="preview-head">
							<div>
								<h3>{{ draft.name || t("formBuilder.untitled") }}</h3>
								<p>{{ t("formBuilder.previewMeta", { key: draft.key, version: draft.version, businessType: draft.businessType }) }}</p>
							</div>
							<el-tag :type="validationErrors.length === 0 ? 'success' : 'warning'">
								{{ validationErrors.length === 0 ? t("formBuilder.ready") : t("formBuilder.needsReview") }}
							</el-tag>
						</div>
						<div v-if="draft.fields.length === 0" class="preview-empty">{{ t("formBuilder.previewEmpty") }}</div>
						<div v-else class="preview-grid" data-testid="form-preview">
							<div v-for="field in draft.fields" :key="field.key" class="preview-field" :class="{ wide: field.type === 'textarea' || field.type === 'detail_table' || field.type === 'attachment' }">
								<label>
									{{ field.label }}
									<span v-if="field.required">*</span>
								</label>
								<div class="preview-control">
									<Eye class="cell-icon" aria-hidden="true" />
									<span>{{ fieldPlaceholder(field) }}</span>
								</div>
								<div v-if="field.type === 'detail_table'" class="detail-columns">
									<el-tag v-for="column in field.detailTable?.columns || []" :key="column.key">{{ column.label }} / {{ column.type }}</el-tag>
								</div>
							</div>
						</div>
						<el-alert
							v-if="validationErrors.length > 0"
							type="warning"
							:closable="false"
							:title="validationErrors[0]"
							class="validation-alert"
						/>
					</el-tab-pane>

					<el-tab-pane :label="t('formBuilder.tab.versions')" name="versions">
						<DataTable
							:rows="schemas.filter((item) => item.key === draft.key).map(toSchemaRow)"
							:columns="schemaColumns"
							row-key="id"
							:empty-text="t('formBuilder.emptyVersions')"
							data-testid="form-version-table"
						>
							<template #actions="{ row }">
								<el-tooltip :content="t('formBuilder.openVersion')">
									<el-button :icon="Eye" circle :aria-label="t('formBuilder.openVersion')" @click="selectSchema(row.raw)" />
								</el-tooltip>
							</template>
						</DataTable>
					</el-tab-pane>
				</el-tabs>
			</section>
		</div>

		<DetailDrawer v-model="fieldDrawerOpen" :title="t('formBuilder.fieldEditor')" size="46%">
			<el-form class="field-form" label-position="top">
				<el-form-item :label="t('formBuilder.label')">
					<el-input v-model="fieldForm.label" />
				</el-form-item>
				<el-form-item :label="t('formBuilder.key')">
					<el-input v-model="fieldForm.key" />
				</el-form-item>
				<el-form-item :label="t('formBuilder.type')">
					<el-select v-model="fieldForm.type">
						<el-option v-for="item in fieldTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('formBuilder.required')">
					<el-switch v-model="fieldForm.required" />
				</el-form-item>
				<el-form-item v-if="fieldForm.type === 'dictionary'" class="wide" :label="t('formBuilder.dictionaryCode')">
					<el-input v-model="fieldForm.dictionary" :placeholder="t('formBuilder.dictionaryPlaceholder')" />
				</el-form-item>
				<el-form-item v-if="fieldForm.type === 'select' || fieldForm.type === 'multi_select'" class="wide" :label="t('formBuilder.options')">
					<el-input v-model="fieldForm.optionsText" type="textarea" :rows="4" :placeholder="t('formBuilder.optionsPlaceholder')" />
				</el-form-item>
				<template v-if="fieldForm.type === 'attachment'">
					<el-form-item :label="t('formBuilder.maxFiles')">
						<el-input-number v-model="fieldForm.maxFiles" :min="1" :max="20" />
					</el-form-item>
					<el-form-item :label="t('formBuilder.maxSizeMB')">
						<el-input-number v-model="fieldForm.maxSizeMB" :min="1" :max="200" />
					</el-form-item>
					<el-form-item class="wide" :label="t('formBuilder.acceptedExtensions')">
						<el-input v-model="fieldForm.acceptText" />
					</el-form-item>
				</template>
				<template v-if="fieldForm.type === 'detail_table'">
					<el-form-item :label="t('formBuilder.minRows')">
						<el-input-number v-model="fieldForm.minRows" :min="0" :max="50" />
					</el-form-item>
					<el-form-item :label="t('formBuilder.maxRows')">
						<el-input-number v-model="fieldForm.maxRows" :min="1" :max="200" />
					</el-form-item>
					<el-form-item class="wide" :label="t('formBuilder.columns')">
						<el-input v-model="fieldForm.columnsText" type="textarea" :rows="5" :placeholder="t('formBuilder.columnsPlaceholder')" />
					</el-form-item>
				</template>
			</el-form>
			<template #footer>
				<el-button @click="fieldDrawerOpen = false">{{ t("formBuilder.cancel") }}</el-button>
				<el-button type="primary" :icon="Save" @click="saveField">{{ t("formBuilder.saveField") }}</el-button>
			</template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.form-summary {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 12px;
}

.summary-tile {
	display: grid;
	gap: 6px;
	min-height: 78px;
	padding: 14px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.summary-tile span,
.preview-head p {
	color: var(--color-text-muted);
}

.summary-tile strong {
	font-size: 1.25rem;
}

.builder-layout {
	display: grid;
	grid-template-columns: minmax(320px, 0.88fr) minmax(0, 1.5fr);
	gap: var(--layout-gap);
	align-items: start;
}

.schema-panel,
.designer-panel {
	display: grid;
	gap: 12px;
	min-width: 0;
}

.designer-panel {
	padding: 14px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.schema-link {
	display: inline-flex;
	align-items: center;
	gap: 8px;
	max-width: 100%;
	color: var(--color-text);
}

.schema-link span {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.cell-icon {
	width: 16px;
	height: 16px;
	color: var(--color-primary);
	flex-shrink: 0;
}

.schema-form,
.field-form {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 4px 12px;
}

.schema-form .wide,
.field-form .wide {
	grid-column: 1 / -1;
}

.preview-head {
	display: flex;
	justify-content: space-between;
	gap: 12px;
	margin-bottom: 12px;
}

.preview-head h3 {
	margin: 0;
	font-size: 1.05rem;
}

.preview-grid {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 12px;
}

.preview-field {
	display: grid;
	gap: 6px;
	min-width: 0;
}

.preview-field.wide {
	grid-column: 1 / -1;
}

.preview-field label {
	font-weight: 700;
}

.preview-field label span {
	color: var(--color-danger);
}

.preview-control,
.preview-empty {
	display: flex;
	align-items: center;
	gap: 8px;
	min-height: 38px;
	padding: 8px 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
	background: var(--color-bg);
	color: var(--color-text-muted);
}

.preview-control span {
	min-width: 0;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.detail-columns {
	display: flex;
	gap: 6px;
	flex-wrap: wrap;
}

.validation-alert {
	margin-top: 12px;
}

@media (max-width: 1080px) {
	.builder-layout {
		grid-template-columns: 1fr;
	}
}

@media (max-width: 760px) {
	.form-summary,
	.schema-form,
	.field-form,
	.preview-grid {
		grid-template-columns: 1fr;
	}
}
</style>
