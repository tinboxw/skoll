<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";

import { useI18n } from "../../i18n";
import type { BackendPluginRecord } from "../../plugins/types";
import { getDefaultHomePath, resolvePluginEntryPath, setDefaultHomePath, usePluginStore } from "../../stores/plugins";
import type { ApiResponse } from "../../utils/api";
import { apiDelete, apiGet, apiPost } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

const router = useRouter();
const pluginStore = usePluginStore();
const { t } = useI18n();

const loading = ref(false);
const operating = ref(false);
const error = ref<string | null>(null);
const showOnlyEnabled = ref(false);
const debugText = ref("");
const logText = ref("");
const operationText = ref("");
const selectedPlugin = ref("");
const pluginPath = ref("");
const activeDefaultHome = ref(getDefaultHomePath());

const visiblePlugins = computed(() => {
	if (!showOnlyEnabled.value) {
		return pluginStore.items;
	}
	return pluginStore.items.filter((item) => item.enabled !== false);
});

async function refreshPlugins(): Promise<void> {
	loading.value = true;
	error.value = null;
	try {
		const payload = await apiGet<ApiResponse<BackendPluginRecord[]>>("/v1/plugins");
		const records = Array.isArray(payload.data) ? payload.data : [];
		pluginStore.setBackendRecords(
			records.map((item) => ({
				id: item.id,
				name: item.name,
				version: item.version,
				enabled: item.enabled,
				uiMode: item.uiMode,
				entryPath: item.frontendEntry,
				systemBuiltin: item.systemBuiltin
			}))
		);
		for (const item of records) {
			pluginStore.registerPlugin({
				id: item.id,
				name: item.name,
				version: item.version,
				enabled: item.enabled,
				uiMode: item.uiMode,
				entryPath: item.frontendEntry,
				systemBuiltin: item.systemBuiltin
			});
		}
		pluginStore.markSynced(null);
	} catch (e) {
		error.value = toErrorMessage(e);
		pluginStore.markSynced(error.value);
	} finally {
		loading.value = false;
	}
}

async function runAction(action: "enable" | "disable" | "uninstall", pluginID: string): Promise<void> {
	operating.value = true;
	error.value = null;
	try {
		if (action === "uninstall") {
			await apiDelete<ApiResponse<unknown>>(`/v1/plugins/${pluginID}`);
		} else {
			await apiPost<ApiResponse<unknown>>(`/v1/plugins/${pluginID}/${action}`);
		}
		await refreshPlugins();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		operating.value = false;
	}
}

async function openDebug(pluginID: string): Promise<void> {
	selectedPlugin.value = pluginID;
	error.value = null;
	try {
		const payload = await apiGet<ApiResponse<unknown>>(`/v1/plugins/${pluginID}/debug`);
		debugText.value = JSON.stringify(payload.data, null, 2);
	} catch (e) {
		error.value = toErrorMessage(e);
		debugText.value = "";
	}
}

async function openLogs(pluginID: string): Promise<void> {
	selectedPlugin.value = pluginID;
	error.value = null;
	try {
		const payload = await apiGet<ApiResponse<{ content?: string }>>(`/v1/plugins/${pluginID}/logs`);
		logText.value = payload.data?.content ?? "";
	} catch (e) {
		error.value = toErrorMessage(e);
		logText.value = "";
	}
}

async function validatePluginPath(): Promise<void> {
	if (!pluginPath.value.trim()) {
		error.value = t("plugin.pathRequired");
		return;
	}
	operating.value = true;
	error.value = null;
	try {
		const payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/validate", {
			path: pluginPath.value.trim()
		});
		operationText.value = JSON.stringify(payload.data, null, 2);
	} catch (e) {
		error.value = toErrorMessage(e);
		operationText.value = "";
	} finally {
		operating.value = false;
	}
}

async function installPluginPath(): Promise<void> {
	if (!pluginPath.value.trim()) {
		error.value = t("plugin.pathRequired");
		return;
	}
	operating.value = true;
	error.value = null;
	try {
		const payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/install", {
			path: pluginPath.value.trim()
		});
		operationText.value = JSON.stringify(payload.data, null, 2);
		await refreshPlugins();
	} catch (e) {
		error.value = toErrorMessage(e);
		operationText.value = "";
	} finally {
		operating.value = false;
	}
}

function pluginEntryPath(pluginID: string): string {
	const plugin = pluginStore.items.find((item) => item.id === pluginID);
	if (!plugin) {
		return "";
	}
	if (plugin.uiMode === "backend_only") {
		return "";
	}
	return resolvePluginEntryPath(plugin);
}

function getPluginRecord(pluginID: string) {
	return pluginStore.items.find((item) => item.id === pluginID);
}

function isSystemBuiltin(pluginID: string): boolean {
	const plugin = getPluginRecord(pluginID);
	return plugin?.systemBuiltin === true;
}

function canVisit(pluginID: string): boolean {
	const plugin = getPluginRecord(pluginID);
	if (!plugin || plugin.enabled === false) {
		return false;
	}
	return pluginEntryPath(pluginID) !== "";
}

function canSetDefault(pluginID: string): boolean {
	return canVisit(pluginID);
}

function canEnable(pluginID: string): boolean {
	const plugin = getPluginRecord(pluginID);
	return plugin?.enabled === false;
}

function canDisable(pluginID: string): boolean {
	const plugin = getPluginRecord(pluginID);
	if (!plugin || plugin.enabled === false) {
		return false;
	}
	return !isSystemBuiltin(pluginID);
}

function canUninstall(pluginID: string): boolean {
	return !isSystemBuiltin(pluginID);
}

async function visitPlugin(pluginID: string): Promise<void> {
	const entryPath = pluginEntryPath(pluginID);
	if (!entryPath) {
		return;
	}
	await router.push(entryPath);
}

function setAsDefaultHome(pluginID: string): void {
	const entryPath = pluginEntryPath(pluginID);
	if (!entryPath) {
		return;
	}
	setDefaultHomePath(entryPath);
	activeDefaultHome.value = entryPath;
}
</script>

<template>
	<section>
		<header class="topbar">
			<div>
				<h2>{{ t("plugin.title") }}</h2>
				<p>{{ t("plugin.desc") }}</p>
			</div>
			<div class="actions">
				<label>
					<input v-model="showOnlyEnabled" type="checkbox" />
					{{ t("plugin.onlyEnabled") }}
				</label>
				<button type="button" :disabled="loading || operating" @click="refreshPlugins">
					{{ loading ? t("plugin.refreshing") : t("plugin.refresh") }}
				</button>
			</div>
		</header>

		<p v-if="error" class="error">{{ error }}</p>

		<section class="install-panel">
			<h3>{{ t("plugin.installTitle") }}</h3>
			<p>{{ t("plugin.installDesc") }}</p>
			<div class="install-actions">
				<input v-model="pluginPath" type="text" :placeholder="t('plugin.pathPlaceholder')" :disabled="operating" />
				<button type="button" :disabled="operating" @click="validatePluginPath">{{ t("plugin.action.validate") }}</button>
				<button type="button" :disabled="operating" @click="installPluginPath">{{ t("plugin.action.install") }}</button>
			</div>
			<pre v-if="operationText">{{ operationText }}</pre>
		</section>

		<table class="plugin-table">
			<thead>
				<tr>
					<th>{{ t("plugin.table.id") }}</th>
					<th>{{ t("plugin.table.name") }}</th>
					<th>{{ t("plugin.table.version") }}</th>
					<th>{{ t("plugin.table.status") }}</th>
					<th>{{ t("plugin.table.actions") }}</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="item in visiblePlugins" :key="item.id">
					<td>{{ item.id }}</td>
					<td>{{ item.name }}</td>
					<td>{{ item.version }}</td>
					<td>
						<span :class="item.enabled === false ? 'disabled' : 'enabled'">
							{{ item.enabled === false ? t("plugin.status.disabled") : t("plugin.status.enabled") }}
						</span>
					</td>
					<td class="action-cell">
						<button v-if="canVisit(item.id)" type="button" :disabled="operating" @click="visitPlugin(item.id)">{{ t("plugin.action.visit") }}</button>
						<span v-else class="action-placeholder">{{ t("plugin.action.visit") }}</span>
						<button v-if="canSetDefault(item.id)" type="button" :disabled="operating" @click="setAsDefaultHome(item.id)">{{ t("plugin.action.setDefault") }}</button>
						<span v-else class="action-placeholder">{{ t("plugin.action.setDefault") }}</span>
						<button v-if="canEnable(item.id)" type="button" :disabled="operating" @click="runAction('enable', item.id)">{{ t("plugin.action.enable") }}</button>
						<span v-else class="action-placeholder">{{ t("plugin.action.enable") }}</span>
						<button v-if="canDisable(item.id)" type="button" :disabled="operating" @click="runAction('disable', item.id)">{{ t("plugin.action.disable") }}</button>
						<span v-else class="action-placeholder">{{ t("plugin.action.disable") }}</span>
						<button v-if="canUninstall(item.id)" type="button" :disabled="operating" @click="runAction('uninstall', item.id)">{{ t("plugin.action.uninstall") }}</button>
						<span v-else class="action-placeholder">{{ t("plugin.action.uninstall") }}</span>
						<button type="button" :disabled="operating" @click="openDebug(item.id)">{{ t("plugin.action.debug") }}</button>
						<button type="button" :disabled="operating" @click="openLogs(item.id)">{{ t("plugin.action.logs") }}</button>
						<span v-if="activeDefaultHome === pluginEntryPath(item.id) && pluginEntryPath(item.id)" class="default-home-badge">{{ t("plugin.defaultHomeActive") }}</span>
					</td>
				</tr>
			</tbody>
		</table>

		<section v-if="selectedPlugin" class="inspector">
			<h3>{{ t("plugin.inspector") }}: {{ selectedPlugin }}</h3>
			<div class="panes">
				<article>
					<h4>{{ t("plugin.debug") }}</h4>
					<pre>{{ debugText || t("plugin.noDebug") }}</pre>
				</article>
				<article>
					<h4>{{ t("plugin.logs") }}</h4>
					<pre>{{ logText || t("plugin.noLogs") }}</pre>
				</article>
			</div>
		</section>
	</section>
</template>

<style scoped>
.topbar {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 12px;
	margin-bottom: 12px;
}

.topbar p {
	margin: 4px 0 0;
	color: var(--color-text-muted);
}

.actions {
	display: flex;
	gap: 10px;
	align-items: center;
}

button {
	border: none;
	border-radius: var(--radius-md);
	padding: 7px 10px;
	background: var(--color-primary);
	color: var(--color-on-primary);
	cursor: pointer;
}

button:hover {
	background: var(--color-primary-strong);
}

button:disabled {
	opacity: 0.6;
	cursor: default;
}

.plugin-table {
	width: 100%;
	border-collapse: collapse;
	font-size: 0.92rem;
}

.install-panel {
	margin-bottom: 12px;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

.install-panel h3 {
	margin: 0;
}

.install-panel p {
	margin: 4px 0 10px;
	color: var(--color-text-muted);
}

.install-actions {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
	align-items: center;
}

.install-actions input {
	min-width: 300px;
	flex: 1;
	height: 34px;
	padding: 0 10px;
	border-radius: var(--radius-sm);
	border: 1px solid var(--color-border);
	background: var(--color-surface);
	color: var(--color-text);
}

.plugin-table th,
.plugin-table td {
	text-align: left;
	padding: 8px;
	border-bottom: 1px solid var(--color-border);
}

.action-cell {
	display: grid;
	grid-template-columns: repeat(7, minmax(86px, 1fr));
	gap: 6px;
	align-items: center;
}

.action-cell button,
.action-placeholder {
	width: 100%;
	text-align: center;
	white-space: nowrap;
}

.action-placeholder {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	height: 30px;
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
	border: 1px dashed var(--color-border);
	color: var(--color-text-muted);
	font-size: 0.82rem;
}

.default-home-badge {
	grid-column: 1 / -1;
}

.action-cell button {
	padding: 5px 8px;
	font-size: 0.78rem;
}

.default-home-badge {
	font-size: 0.75rem;
	padding: 2px 8px;
	border-radius: 999px;
	background: var(--color-tag-bg);
	color: var(--color-tag-text);
	border: 1px solid var(--color-border);
	align-self: center;
}

.enabled {
	color: var(--color-success);
	font-weight: 600;
}

.disabled {
	color: var(--color-danger);
	font-weight: 600;
}

.error {
	color: var(--color-danger);
	margin-bottom: 10px;
}

.inspector {
	margin-top: 14px;
	border-top: 1px solid var(--color-border);
	padding-top: 10px;
}

.panes {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: 10px;
}

pre {
	margin: 0;
	padding: 10px;
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
	border: 1px solid var(--color-border);
	white-space: pre-wrap;
	word-break: break-word;
	font-size: 0.78rem;
	max-height: 220px;
	overflow: auto;
}

@media (max-width: 980px) {
	.panes {
		grid-template-columns: 1fr;
	}
}
</style>

