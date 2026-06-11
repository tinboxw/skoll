<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import SchemaForm from "../../components/Common/SchemaForm.vue";
import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import type { PluginConfigSchema } from "../../plugins/types";
import { type ApiResponse, apiGet, apiPost, apiPut } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type SettingRecord = {
	id: string;
	key: string;
	value: string;
	encrypted: boolean;
};

function normalizeSettingRecord(item: unknown): SettingRecord | null {
	if (!item || typeof item !== "object") {
		return null;
	}
	const row = item as Record<string, unknown>;
	const id = String(row.id ?? row.ID ?? "").trim();
	const key = String(row.key ?? row.Key ?? "").trim();
	if (!id || !key) {
		return null;
	}
	return {
		id,
		key,
		value: String(row.value ?? row.Value ?? ""),
		encrypted: Boolean(row.encrypted ?? row.Encrypted)
	};
}

const { t, locale } = useI18n();

const loading = ref(false);
const saving = ref(false);
const resetting = ref(false);
const schemaLoading = ref(false);
const error = ref("");
const success = ref("");
const searchKey = ref("");
const allSettings = ref<SettingRecord[]>([]);
const editorKey = ref("");
const editorValue = ref("");
const editorEncrypted = ref(false);
const schemaSaving = ref(false);
const schemaDraft = ref<Record<string, unknown>>({});
const schemaValid = ref(true);
const remoteSystemConfigSchema = ref<PluginConfigSchema | null>(null);

const fallbackSystemConfigSchema = computed<PluginConfigSchema>(() => ({
	titleZhCN: t("settings.schemaTitle"),
	titleEnUS: "Common Settings",
	description: t("settings.schemaDesc"),
	fields: [
		{
			key: "audit.retention_days",
			labelZhCN: t("settings.auditRetention"),
			labelEnUS: "Audit retention days",
			type: "number",
			default: "30",
			min: 1,
			max: 3650,
			help: t("settings.auditRetentionHelp")
		},
		{
			key: "plugin.auto_enable",
			labelZhCN: t("settings.autoEnable"),
			labelEnUS: "Auto-enable installed plugins",
			type: "boolean",
			default: "false"
		},
		{
			key: "plugin.dev_portal_enabled",
			labelZhCN: t("settings.devPortalEnabled"),
			labelEnUS: "Developer portal enabled",
			type: "boolean",
			default: "true"
		},
		{
			key: "skoll.menu.tree",
			labelZhCN: t("settings.menuTree"),
			labelEnUS: "Menu tree JSON",
			type: "textarea",
			default: "",
			placeholder: "[]"
		}
	]
}));
const systemConfigSchema = computed<PluginConfigSchema>(() => remoteSystemConfigSchema.value ?? fallbackSystemConfigSchema.value);

const filteredSettings = computed(() => {
	const keyword = searchKey.value.trim().toLowerCase();
	if (keyword === "") {
		return allSettings.value;
	}
	return allSettings.value.filter((item) => item.key.toLowerCase().includes(keyword));
});

const groupedSettings = computed(() => {
	const grouped: Record<string, SettingRecord[]> = {};
	for (const item of filteredSettings.value) {
		const namespace = item.key.includes(".") ? item.key.split(".")[0] : "misc";
		if (!grouped[namespace]) {
			grouped[namespace] = [];
		}
		grouped[namespace].push(item);
	}
	return Object.entries(grouped).sort((a, b) => a[0].localeCompare(b[0]));
});

const busy = computed(() => loading.value || saving.value || resetting.value || schemaLoading.value);
const schemaModel = computed(() => {
	if (Object.keys(schemaDraft.value).length > 0) {
		return schemaDraft.value;
	}
	const out: Record<string, unknown> = {};
	for (const field of systemConfigSchema.value.fields ?? []) {
		const item = allSettings.value.find((setting) => setting.key === field.key);
		out[field.key] = item ? coerceSettingValue(item.value, field.type) : coerceSettingValue(field.default ?? "", field.type);
	}
	return out;
});
const schemaEncryptedMap = computed(() => {
	const out: Record<string, boolean> = {};
	for (const item of allSettings.value) {
		out[item.key] = item.encrypted;
	}
	return out;
});

function coerceSettingValue(value: unknown, type?: string): unknown {
	if (type === "boolean") {
		return value === true || value === "true" || value === "1";
	}
	if (type === "number") {
		const parsed = Number(value ?? 0);
		return Number.isFinite(parsed) ? parsed : 0;
	}
	return value ?? "";
}

function serializeSettingValue(value: unknown): string {
	if (typeof value === "boolean") {
		return value ? "true" : "false";
	}
	if (value === null || value === undefined) {
		return "";
	}
	return String(value);
}

function normalizeSystemConfigSchema(schema: unknown): PluginConfigSchema | null {
	if (!schema || typeof schema !== "object") {
		return null;
	}
	const row = schema as PluginConfigSchema;
	const fields = Array.isArray(row.fields)
		? row.fields.filter((field) => field && typeof field === "object" && typeof field.key === "string" && field.key.trim() !== "")
		: [];
	if (fields.length === 0) {
		return null;
	}
	return {
		...row,
		fields
	};
}

async function loadSettingsSchema(): Promise<void> {
	schemaLoading.value = true;
	try {
		const payload = await apiGet<ApiResponse<PluginConfigSchema>>("/v1/system/settings/schema");
		remoteSystemConfigSchema.value = normalizeSystemConfigSchema(payload.data);
	} catch {
		remoteSystemConfigSchema.value = null;
	} finally {
		schemaLoading.value = false;
	}
}

async function loadSettings(): Promise<void> {
	loading.value = true;
	error.value = "";
	try {
		const payload = await apiGet<ApiResponse<SettingRecord[]>>("/v1/system/settings?offset=0&limit=500");
		allSettings.value = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeSettingRecord(item)).filter((item): item is SettingRecord => item !== null)
			: [];
		if (editorKey.value === "" && allSettings.value.length > 0) {
			selectSetting(allSettings.value[0]);
		}
		schemaDraft.value = buildSchemaModelFromSettings();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

function buildSchemaModelFromSettings(): Record<string, unknown> {
	const out: Record<string, unknown> = {};
	for (const field of systemConfigSchema.value.fields ?? []) {
		const item = allSettings.value.find((setting) => setting.key === field.key);
		out[field.key] = item ? coerceSettingValue(item.value, field.type) : coerceSettingValue(field.default ?? "", field.type);
	}
	return out;
}

function selectSetting(item: SettingRecord): void {
	editorKey.value = item.key;
	editorValue.value = item.value;
	editorEncrypted.value = item.encrypted;
}

async function fetchByKey(): Promise<void> {
	if (editorKey.value.trim() === "") {
		return;
	}
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		const payload = await apiGet<ApiResponse<SettingRecord>>(`/v1/system/settings/${encodeURIComponent(editorKey.value.trim())}`);
		const record = normalizeSettingRecord(payload.data);
		if (record) {
			selectSetting(record);
			success.value = t("settings.fetchDone");
		}
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function saveSetting(): Promise<void> {
	if (editorKey.value.trim() === "") {
		return;
	}
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		await apiPut<ApiResponse<SettingRecord>>(`/v1/system/settings/${encodeURIComponent(editorKey.value.trim())}`, {
			value: editorValue.value,
			encrypted: editorEncrypted.value
		});
		success.value = t("settings.saveDone");
		await loadSettings();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function saveSchemaSettings(next: Record<string, unknown>): Promise<void> {
	schemaDraft.value = next;
	schemaSaving.value = true;
	error.value = "";
	success.value = "";
	try {
		for (const field of systemConfigSchema.value.fields ?? []) {
			if (!field.key) {
				continue;
			}
			await apiPut<ApiResponse<SettingRecord>>(`/v1/system/settings/${encodeURIComponent(field.key)}`, {
				value: serializeSettingValue(next[field.key]),
				encrypted: schemaEncryptedMap.value[field.key] ?? false
			});
		}
		success.value = t("settings.saveDone");
		await loadSettings();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		schemaSaving.value = false;
	}
}

function updateSchemaDraft(next: Record<string, unknown>): void {
	schemaDraft.value = next;
}

function updateSchemaValid(next: boolean): void {
	schemaValid.value = next;
}

async function resetSettings(): Promise<void> {
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: t("settings.reset"),
		confirmText: t("common.reset"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (!confirmed) {
		return;
	}
	resetting.value = true;
	error.value = "";
	success.value = "";
	try {
		const payload = await apiPost<ApiResponse<{ reset?: number }>>("/v1/system/settings/reset");
		success.value = `${t("settings.resetDone")}: ${payload.data?.reset ?? 0}`;
		await loadSettings();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		resetting.value = false;
	}
}

onMounted(() => {
	void (async () => {
		await loadSettingsSchema();
		await loadSettings();
	})();
});
</script>

<template>
	<section class="setting-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.settings") }}</h2>
				<p>{{ t("settings.desc") }}</p>
			</div>
			<div class="header-actions">
				<el-button :loading="loading" :disabled="saving || resetting" @click="loadSettings">{{ t("common.refresh") }}</el-button>
				<el-button type="danger" plain :loading="resetting" :disabled="loading || saving" @click="resetSettings">{{ t("settings.reset") }}</el-button>
			</div>
		</header>

		<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
		<el-alert v-if="success" :title="success" type="success" show-icon :closable="false" />

		<section class="panel">
			<div class="section-header">
				<div>
					<h3>{{ t("settings.schemaTitle") }}</h3>
					<p>{{ t("settings.schemaDesc") }}</p>
				</div>
				<el-input v-model="searchKey" clearable class="search-input" :placeholder="t('settings.searchPlaceholder')" :disabled="busy" />
			</div>
			<SchemaForm
				:model-value="schemaModel"
				:schema="systemConfigSchema"
				:locale="locale"
				:disabled="busy || schemaSaving"
				@update:model-value="updateSchemaDraft"
				@update:valid="updateSchemaValid"
			/>
			<div class="form-actions">
				<el-button type="primary" :loading="schemaSaving" :disabled="busy || !schemaValid" @click="saveSchemaSettings(schemaModel)">{{ t("common.save") }}</el-button>
			</div>
		</section>

		<section class="panel">
			<h3>{{ t("settings.editor") }}</h3>
			<el-form label-position="top" class="editor-form" @submit.prevent="saveSetting">
				<el-form-item :label="t('table.key')" required>
					<el-input v-model="editorKey" :disabled="saving || resetting" />
				</el-form-item>
				<el-form-item :label="t('settings.encrypted')">
					<el-switch v-model="editorEncrypted" :disabled="saving || resetting" />
				</el-form-item>
				<el-form-item class="value-field" :label="t('table.value')">
					<el-input v-model="editorValue" type="textarea" :rows="5" :disabled="saving || resetting" />
				</el-form-item>
				<div class="form-actions">
					<el-button :disabled="saving || resetting || editorKey.trim() === ''" @click="fetchByKey">{{ t("settings.fetchByKey") }}</el-button>
					<el-button type="primary" native-type="submit" :loading="saving" :disabled="resetting || editorKey.trim() === ''">{{ t("common.save") }}</el-button>
				</div>
			</el-form>
		</section>

		<section class="panel">
			<h3>{{ t("settings.rawList") }}</h3>
			<el-empty v-if="!loading && groupedSettings.length === 0" :description="t('common.empty')" />
			<div v-else class="group-list">
				<section v-for="[groupName, items] in groupedSettings" :key="groupName" class="group-panel">
					<div class="group-title">
						<el-tag effect="plain">{{ groupName }}</el-tag>
						<span>{{ items.length }}</span>
					</div>
					<el-table v-loading="loading" :data="items" border row-key="id" :empty-text="t('common.empty')">
						<el-table-column prop="key" :label="t('table.key')" min-width="230" show-overflow-tooltip />
						<el-table-column prop="value" :label="t('table.value')" min-width="280" show-overflow-tooltip />
						<el-table-column :label="t('table.status')" width="120">
							<template #default="{ row }">
								<el-tag :type="row.encrypted ? 'warning' : 'info'" effect="plain">
									{{ row.encrypted ? t("settings.encrypted") : t("settings.plain") }}
								</el-tag>
							</template>
						</el-table-column>
						<el-table-column :label="t('table.actions')" width="110" fixed="right">
							<template #default="{ row }">
								<el-button link type="primary" :disabled="saving || resetting" @click="selectSetting(row)">{{ t("common.edit") }}</el-button>
							</template>
						</el-table-column>
					</el-table>
				</section>
			</div>
		</section>
	</section>
</template>

<style scoped>
.setting-page {
	display: grid;
	gap: 14px;
}

.page-header,
.section-header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 16px;
}

.page-header h2 {
	margin: 0;
	font-size: 1.35rem;
}

.page-header p {
	margin: 6px 0 0;
	color: var(--color-text-muted);
}

.header-actions {
	display: flex;
	align-items: center;
	gap: 8px;
}

.panel {
	display: grid;
	gap: 12px;
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.panel h3 {
	margin: 0;
	font-size: 1rem;
}

.search-input {
	width: min(380px, 100%);
}

.editor-form {
	display: grid;
	grid-template-columns: minmax(0, 1fr) 160px;
	gap: 4px 14px;
}

.value-field,
.form-actions {
	grid-column: 1 / -1;
}

.form-actions {
	display: flex;
	justify-content: flex-end;
	gap: 8px;
}

.group-list {
	display: grid;
	gap: 12px;
}

.group-panel {
	display: grid;
	gap: 8px;
}

.group-title {
	display: flex;
	align-items: center;
	gap: 8px;
	color: var(--color-text-muted);
}

@media (max-width: 820px) {
	.page-header,
	.section-header,
	.header-actions,
	.editor-form {
		display: grid;
	}

	.value-field,
	.form-actions {
		grid-column: auto;
	}

	.form-actions {
		justify-content: stretch;
	}
}
</style>
