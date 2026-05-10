<script setup lang="ts">
import { onMounted, ref } from "vue";

import { useI18n } from "../../i18n";
import { type ApiResponse, apiGet, apiPut } from "../../utils/api";
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

const SETTING_AUDIT_RETENTION_DAYS = "audit.retention.days";
const SETTING_PLUGIN_AUTO_ENABLE = "plugin.auto_enable";

const { t } = useI18n();

const loading = ref(false);
const saving = ref(false);
const error = ref("");
const success = ref("");
const auditRetentionDays = ref(90);
const pluginAutoEnable = ref(true);
const allSettings = ref<SettingRecord[]>([]);

async function loadSettings(): Promise<void> {
	loading.value = true;
	error.value = "";
	try {
		const payload = await apiGet<ApiResponse<SettingRecord[]>>("/v1/system/settings?offset=0&limit=200");
		const list = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeSettingRecord(item)).filter((item): item is SettingRecord => item !== null)
			: [];
		allSettings.value = list;

		const retention = list.find((item) => item.key === SETTING_AUDIT_RETENTION_DAYS);
		const autoEnable = list.find((item) => item.key === SETTING_PLUGIN_AUTO_ENABLE);

		auditRetentionDays.value = Number.parseInt(retention?.value ?? "90", 10) || 90;
		pluginAutoEnable.value = (autoEnable?.value ?? "true").toLowerCase() === "true";
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

async function saveSettings(): Promise<void> {
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		await apiPut<ApiResponse<SettingRecord>>(`/v1/system/settings/${SETTING_AUDIT_RETENTION_DAYS}`, {
			value: String(Math.max(1, auditRetentionDays.value)),
			encrypted: false
		});
		await apiPut<ApiResponse<SettingRecord>>(`/v1/system/settings/${SETTING_PLUGIN_AUTO_ENABLE}`, {
			value: String(pluginAutoEnable.value),
			encrypted: false
		});
		success.value = t("settings.saveDone");
		await loadSettings();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
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
		<div class="grid">
			<label>
				<span>{{ t("settings.auditRetention") }}</span>
				<input v-model.number="auditRetentionDays" type="number" min="1" :disabled="loading || saving" />
			</label>
			<label class="toggle">
				<input v-model="pluginAutoEnable" type="checkbox" :disabled="loading || saving" />
				<span>{{ t("settings.autoEnable") }}</span>
			</label>
			<div class="actions">
				<button type="button" :disabled="loading || saving" @click="loadSettings">{{ loading ? t("common.loading") : t("common.refresh") }}</button>
				<button type="button" :disabled="saving" @click="saveSettings">{{ saving ? t("common.loading") : t("common.save") }}</button>
			</div>
		</div>
		<section class="raw-list">
			<h3>{{ t("settings.rawList") }}</h3>
			<table>
				<thead>
					<tr>
						<th>{{ t("table.key") }}</th>
						<th>{{ t("table.value") }}</th>
						<th>{{ t("table.status") }}</th>
					</tr>
				</thead>
				<tbody v-if="allSettings.length > 0">
					<tr v-for="item in allSettings" :key="item.id">
						<td>{{ item.key }}</td>
						<td>{{ item.value }}</td>
						<td>{{ item.encrypted ? t("settings.encrypted") : t("settings.plain") }}</td>
					</tr>
				</tbody>
				<tbody v-else>
					<tr>
						<td colspan="3">{{ loading ? t("common.loading") : t("common.empty") }}</td>
					</tr>
				</tbody>
			</table>
		</section>
	</section>
</template>

<style scoped>
.grid {
	display: grid;
	gap: 12px;
	max-width: 380px;
}

label {
	display: grid;
	gap: 6px;
	color: var(--color-text-muted);
}

input[type="number"] {
	padding: 8px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface-soft);
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

.raw-list {
	margin-top: 12px;
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
</style>

