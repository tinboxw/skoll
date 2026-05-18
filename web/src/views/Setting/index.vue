<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import { useI18n } from "../../i18n";
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

const { t } = useI18n();

const loading = ref(false);
const saving = ref(false);
const resetting = ref(false);
const error = ref("");
const success = ref("");
const searchKey = ref("");
const allSettings = ref<SettingRecord[]>([]);
const editorKey = ref("");
const editorValue = ref("");
const editorEncrypted = ref(false);

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
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
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

async function resetSettings(): Promise<void> {
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
	void loadSettings();
});
</script>

<template>
	<section>
		<h2>{{ t("page.settings") }}</h2>
		<p>{{ t("settings.desc") }}</p>
		<p v-if="error" class="error">{{ error }}</p>
		<p v-if="success" class="success">{{ success }}</p>
		<div class="toolbar">
			<input v-model="searchKey" type="text" :placeholder="t('settings.searchPlaceholder')" :disabled="loading || saving || resetting" />
			<div class="actions">
				<button type="button" :disabled="loading || saving || resetting" @click="loadSettings">{{ loading ? t("common.loading") : t("common.refresh") }}</button>
				<button type="button" :disabled="loading || saving || resetting" @click="resetSettings">{{ resetting ? t("common.loading") : t("settings.reset") }}</button>
			</div>
		</div>

		<section class="editor">
			<h3>{{ t("settings.editor") }}</h3>
			<div class="editor-grid">
				<label>
					<span>{{ t("table.key") }}</span>
					<input v-model="editorKey" type="text" :disabled="saving || resetting" />
				</label>
				<label>
					<span>{{ t("table.value") }}</span>
					<textarea v-model="editorValue" rows="5" :disabled="saving || resetting" />
				</label>
				<label class="toggle">
					<input v-model="editorEncrypted" type="checkbox" :disabled="saving || resetting" />
					<span>{{ t("settings.encrypted") }}</span>
				</label>
			</div>
			<div class="actions">
				<button type="button" :disabled="saving || resetting || editorKey.trim() === ''" @click="fetchByKey">{{ t("settings.fetchByKey") }}</button>
				<button type="button" :disabled="saving || resetting || editorKey.trim() === ''" @click="saveSetting">{{ saving ? t("common.loading") : t("common.save") }}</button>
			</div>
		</section>

		<section class="raw-list">
			<h3>{{ t("settings.rawList") }}</h3>
			<div v-if="groupedSettings.length > 0" class="group-list">
				<section v-for="[groupName, items] in groupedSettings" :key="groupName" class="group-panel">
					<h4>{{ groupName }}</h4>
					<table>
						<thead>
							<tr>
								<th>{{ t("table.key") }}</th>
								<th>{{ t("table.value") }}</th>
								<th>{{ t("table.status") }}</th>
								<th>{{ t("table.actions") }}</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="item in items" :key="item.id">
								<td>{{ item.key }}</td>
								<td>{{ item.value }}</td>
								<td>{{ item.encrypted ? t("settings.encrypted") : t("settings.plain") }}</td>
								<td><button type="button" :disabled="saving || resetting" @click="selectSetting(item)">{{ t("common.edit") }}</button></td>
							</tr>
						</tbody>
					</table>
				</section>
			</div>
			<p v-else>{{ loading ? t("common.loading") : t("common.empty") }}</p>
		</section>
	</section>
</template>

<style scoped>
.toolbar {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 8px;
	margin: 10px 0;
}

label {
	display: grid;
	gap: 6px;
	color: var(--color-text-muted);
}

input[type="text"],
textarea {
	padding: 8px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface-soft);
}

textarea {
	resize: vertical;
	min-height: 120px;
}

.actions {
	display: flex;
	gap: 8px;
}

button {
	padding: 8px 12px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface-soft);
	cursor: pointer;
}

.toggle {
	grid-template-columns: auto 1fr;
	align-items: center;
	column-gap: 8px;
}

.editor {
	margin-top: 10px;
	padding: 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.editor h3 {
	margin: 0 0 10px;
}

.editor-grid {
	display: grid;
	gap: 8px;
	margin-bottom: 8px;
}

.raw-list {
	margin-top: 12px;
}

.group-list {
	display: grid;
	gap: 10px;
}

.group-panel {
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	padding: 8px;
}

.group-panel h4 {
	margin: 0 0 8px;
	text-transform: uppercase;
	font-size: 0.85rem;
	color: var(--color-text-muted);
}

table {
	width: 100%;
	border-collapse: collapse;
}

th,
td {
	text-align: left;
	padding: 8px;
	border-bottom: 1px solid var(--color-border);
}

.error {
	color: var(--color-danger);
}

.success {
	color: var(--color-success);
}

@media (max-width: 900px) {
	.toolbar {
		flex-direction: column;
		align-items: stretch;
	}
}
</style>

