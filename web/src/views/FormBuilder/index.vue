<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { Copy, Eye, FileText, Pencil, Plus, RefreshCw, Save, Trash2 } from "lucide-vue-next";
import { ElMessage } from "element-plus";

import { ConfirmAction, DataTable, DetailDrawer, FilterBar, PageShell, PageToolbar, type DataTableColumn } from "../../components/Common";
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
	columnsText: "name:Name:string\nquantity:Quantity:number"
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

const schemaColumns: DataTableColumn[] = [
	{ key: "name", label: "Name", minWidth: 180 },
	{ key: "key", label: "Key", minWidth: 180 },
	{ key: "version", label: "Version", width: 90 },
	{ key: "businessType", label: "Business", minWidth: 150 },
	{ key: "fields", label: "Fields", width: 90 },
	{ key: "updatedAt", label: "Updated", minWidth: 160 }
];

const fieldColumns: DataTableColumn[] = [
	{ key: "label", label: "Label", minWidth: 160 },
	{ key: "key", label: "Key", minWidth: 140 },
	{ key: "type", label: "Type", minWidth: 120 },
	{ key: "required", label: "Required", width: 110 }
];

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
		error.value = "You do not have permission to manage form schemas.";
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
		ElMessage.success("Form schema saved");
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
	ElMessage.success("Form schema deleted");
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
		fieldForm.columnsText = (field.detailTable?.columns || []).map((item) => `${item.key}:${item.label}:${item.type}`).join("\n") || "name:Name:string\nquantity:Quantity:number";
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
		fieldForm.columnsText = "name:Name:string\nquantity:Quantity:number";
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
		required: field.required ? "Yes" : "No",
		raw: field,
		index
	};
}

function fieldPlaceholder(field: FormField): string {
	if (field.type === "attachment") {
		return `Upload ${field.attachment?.accept?.join(", ") || "files"}; max ${field.attachment?.maxFiles || 1} file(s).`;
	}
	if (field.type === "detail_table") {
		return `${field.detailTable?.columns.length || 0} columns, ${field.detailTable?.minRows || 0}-${field.detailTable?.maxRows || "n"} rows`;
	}
	if (field.type === "dictionary") {
		return field.dictionary || "dictionary.code";
	}
	if (field.type === "user") {
		return "Select user";
	}
	if (field.type === "department") {
		return "Select department";
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
		title="Form Builder"
		description="Design workflow-ready form schemas, preview fields, and manage versions."
		:loading="loading"
		:error="error"
		:forbidden="!canManage"
		forbidden-title="Form builder unavailable"
		forbidden-description="Ask an administrator for form.schema.manage permission."
		data-testid="form-builder-page"
	>
		<template #actions>
			<el-button :icon="RefreshCw" :loading="loading" @click="refreshSchemas">Refresh</el-button>
			<el-button :icon="Plus" @click="newSchema">New</el-button>
			<el-button :icon="FileText" @click="usePurchaseTemplate">Template</el-button>
			<el-button :icon="Copy" :disabled="draft.fields.length === 0" @click="duplicateVersion">New version</el-button>
			<ConfirmAction
				label="Delete"
				message="Delete this form schema version?"
				:disabled="!selectedId"
				@confirm="removeSelected"
			/>
			<el-button type="primary" :icon="Save" :loading="saving" @click="saveDraft">Save</el-button>
		</template>
		<template #stateActions>
			<el-button :icon="RefreshCw" @click="refreshSchemas">Retry</el-button>
		</template>

		<section class="form-summary" aria-label="Form builder summary">
			<div class="summary-tile">
				<span>Schemas</span>
				<strong>{{ summary.schemas }}</strong>
			</div>
			<div class="summary-tile">
				<span>Current fields</span>
				<strong>{{ summary.fields }}</strong>
			</div>
			<div class="summary-tile">
				<span>Versions</span>
				<strong>{{ summary.versions }}</strong>
			</div>
			<div class="summary-tile">
				<span>Required</span>
				<strong>{{ summary.required }}</strong>
			</div>
		</section>

		<div class="builder-layout">
			<section class="schema-panel" aria-label="Form schemas">
				<PageToolbar>
					<FilterBar>
						<el-input v-model="keyword" clearable placeholder="Search schema, key, or business type" />
					</FilterBar>
				</PageToolbar>
				<DataTable
					:rows="schemaRows"
					:columns="schemaColumns"
					row-key="id"
					empty-text="No form schemas"
					data-testid="form-schema-table"
				>
					<template #cell-name="{ row }">
						<button class="schema-link" type="button" @click="selectSchema(row.raw)">
							<FileText class="cell-icon" aria-hidden="true" />
							<span>{{ row.name }}</span>
						</button>
					</template>
					<template #actions="{ row }">
						<el-tooltip content="Edit schema">
							<el-button :icon="Pencil" circle @click="selectSchema(row.raw)" />
						</el-tooltip>
					</template>
				</DataTable>
			</section>

			<section class="designer-panel" aria-label="Form designer">
				<el-tabs v-model="activePane">
					<el-tab-pane label="Design" name="design">
						<el-form class="schema-form" label-position="top">
							<el-form-item label="Form name">
								<el-input v-model="draft.name" />
							</el-form-item>
							<el-form-item label="Form key">
								<el-input v-model="draft.key" />
							</el-form-item>
							<el-form-item label="Version">
								<el-input-number v-model="draft.version" :min="1" :step="1" />
							</el-form-item>
							<el-form-item label="Business type">
								<el-input v-model="draft.businessType" />
							</el-form-item>
							<el-form-item class="wide" label="Description">
								<el-input v-model="draft.description" type="textarea" :rows="2" />
							</el-form-item>
						</el-form>

						<PageToolbar>
							<template #left>
								<strong>Fields</strong>
							</template>
							<el-button type="primary" :icon="Plus" @click="openFieldEditor()">Add field</el-button>
						</PageToolbar>
						<DataTable
							:rows="fieldRows"
							:columns="fieldColumns"
							row-key="key"
							empty-text="No fields in this form"
							data-testid="form-field-table"
						>
							<template #cell-required="{ value }">
								<el-tag :type="value === 'Yes' ? 'danger' : 'info'">{{ value }}</el-tag>
							</template>
							<template #actions="{ row }">
								<el-tooltip content="Move up">
									<el-button text :disabled="row.index === 0" @click="moveField(row, -1)">Up</el-button>
								</el-tooltip>
								<el-tooltip content="Move down">
									<el-button text :disabled="row.index === fieldRows.length - 1" @click="moveField(row, 1)">Down</el-button>
								</el-tooltip>
								<el-tooltip content="Edit field">
									<el-button :icon="Pencil" circle @click="openFieldEditor(row)" />
								</el-tooltip>
								<el-tooltip content="Delete field">
									<el-button :icon="Trash2" circle type="danger" @click="removeField(row)" />
								</el-tooltip>
							</template>
						</DataTable>
					</el-tab-pane>

					<el-tab-pane label="Preview" name="preview">
						<div class="preview-head">
							<div>
								<h3>{{ draft.name || "Untitled form" }}</h3>
								<p>{{ draft.key }} / v{{ draft.version }} / {{ draft.businessType }}</p>
							</div>
							<el-tag :type="validationErrors.length === 0 ? 'success' : 'warning'">
								{{ validationErrors.length === 0 ? "Ready" : "Needs review" }}
							</el-tag>
						</div>
						<div v-if="draft.fields.length === 0" class="preview-empty">Add fields to preview the form.</div>
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

					<el-tab-pane label="Versions" name="versions">
						<DataTable
							:rows="schemas.filter((item) => item.key === draft.key).map(toSchemaRow)"
							:columns="schemaColumns"
							row-key="id"
							empty-text="Save this schema to create the first version"
							data-testid="form-version-table"
						>
							<template #actions="{ row }">
								<el-tooltip content="Open version">
									<el-button :icon="Eye" circle @click="selectSchema(row.raw)" />
								</el-tooltip>
							</template>
						</DataTable>
					</el-tab-pane>
				</el-tabs>
			</section>
		</div>

		<DetailDrawer v-model="fieldDrawerOpen" title="Field editor" size="46%">
			<el-form class="field-form" label-position="top">
				<el-form-item label="Label">
					<el-input v-model="fieldForm.label" />
				</el-form-item>
				<el-form-item label="Key">
					<el-input v-model="fieldForm.key" />
				</el-form-item>
				<el-form-item label="Type">
					<el-select v-model="fieldForm.type">
						<el-option v-for="item in FORM_FIELD_TYPE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
					</el-select>
				</el-form-item>
				<el-form-item label="Required">
					<el-switch v-model="fieldForm.required" />
				</el-form-item>
				<el-form-item v-if="fieldForm.type === 'dictionary'" class="wide" label="Dictionary code">
					<el-input v-model="fieldForm.dictionary" placeholder="pharma.product.status" />
				</el-form-item>
				<el-form-item v-if="fieldForm.type === 'select' || fieldForm.type === 'multi_select'" class="wide" label="Options">
					<el-input v-model="fieldForm.optionsText" type="textarea" :rows="4" placeholder="Pending=pending&#10;Approved=approved" />
				</el-form-item>
				<template v-if="fieldForm.type === 'attachment'">
					<el-form-item label="Max files">
						<el-input-number v-model="fieldForm.maxFiles" :min="1" :max="20" />
					</el-form-item>
					<el-form-item label="Max size MB">
						<el-input-number v-model="fieldForm.maxSizeMB" :min="1" :max="200" />
					</el-form-item>
					<el-form-item class="wide" label="Accepted extensions">
						<el-input v-model="fieldForm.acceptText" />
					</el-form-item>
				</template>
				<template v-if="fieldForm.type === 'detail_table'">
					<el-form-item label="Min rows">
						<el-input-number v-model="fieldForm.minRows" :min="0" :max="50" />
					</el-form-item>
					<el-form-item label="Max rows">
						<el-input-number v-model="fieldForm.maxRows" :min="1" :max="200" />
					</el-form-item>
					<el-form-item class="wide" label="Columns">
						<el-input v-model="fieldForm.columnsText" type="textarea" :rows="5" placeholder="product:Product:string" />
					</el-form-item>
				</template>
			</el-form>
			<template #footer>
				<el-button @click="fieldDrawerOpen = false">Cancel</el-button>
				<el-button type="primary" :icon="Save" @click="saveField">Save field</el-button>
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
	border: 0;
	background: transparent;
	color: var(--color-text);
	cursor: pointer;
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
