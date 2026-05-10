<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";

import { useI18n } from "../../i18n";
import { syncBackendPlugins } from "../../plugins";
import { clearDefaultHomePath, getDefaultHomePath, getSystemDefaultHomePath, isDefaultHomePath, resolvePluginEntryPath, setDefaultHomePath, usePluginStore } from "../../stores/plugins";
import { ApiError, type ApiResponse } from "../../utils/api";
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
const activeInspectorPanel = ref<"info" | "logs" | "">("");
const pluginPath = ref("");
const activeDefaultHome = ref(getDefaultHomePath());
const systemDefaultHome = getSystemDefaultHomePath();
const info = ref<string | null>(null);

const visiblePlugins = computed(() => {
	if (!showOnlyEnabled.value) {
		return pluginStore.items;
	}
	return pluginStore.items.filter((item) => item.enabled !== false);
});

const syncStatusText = computed(() => {
	if (pluginStore.syncStatus === "loading") {
		return t("plugin.sync.loading");
	}
	if (pluginStore.syncStatus === "error") {
		return t("plugin.sync.error");
	}
	if (pluginStore.syncStatus === "success") {
		return t("plugin.sync.success");
	}
	return t("plugin.sync.idle");
});

async function refreshPlugins(): Promise<void> {
	loading.value = true;
	error.value = null;
	info.value = null;
	try {
		await syncBackendPlugins(router, pluginStore);
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
	info.value = null;
	const beforeActionEntryPath = pluginEntryPath(pluginID);
	try {
		if (action === "uninstall") {
			await apiDelete<ApiResponse<unknown>>(`/v1/plugins/${pluginID}`);
		} else {
			await apiPost<ApiResponse<unknown>>(`/v1/plugins/${pluginID}/${action}`);
		}
		if ((action === "disable" || action === "uninstall") && beforeActionEntryPath && isDefaultHomePath(beforeActionEntryPath, systemDefaultHome)) {
			clearDefaultHomePath();
			activeDefaultHome.value = getDefaultHomePath(systemDefaultHome);
			info.value = t("plugin.defaultHomeResetAfterAction");
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
	activeInspectorPanel.value = "info";
	error.value = null;
	info.value = null;
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
	activeInspectorPanel.value = "logs";
	error.value = null;
	info.value = null;
	try {
		const payload = await apiGet<ApiResponse<{ content?: string }>>(`/v1/plugins/${pluginID}/logs`);
		logText.value = payload.data?.content ?? "";
	} catch (e) {
		if (e instanceof ApiError && e.status === 404) {
			logText.value = "";
			return;
		}
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
	info.value = null;
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
	info.value = null;
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
	info.value = t("plugin.defaultHomeActive");
}

function resetDefaultHome(): void {
	clearDefaultHomePath();
	activeDefaultHome.value = getDefaultHomePath(systemDefaultHome);
	info.value = t("plugin.defaultHomeResetDone");
}
</script>

<template>
	<section>
		<header class="topbar">
			<div>
				<h2>{{ t("plugin.title") }}</h2>
				<p>{{ t("plugin.desc") }}</p>
				<p class="sync-status">{{ syncStatusText }} · {{ t("plugin.sync.attempts") }} {{ pluginStore.syncAttempts }}</p>
				<p v-if="pluginStore.degradedMode" class="sync-degraded">{{ t("plugin.sync.degraded") }}</p>
			</div>
			<div class="actions">
				<label>
					<input v-model="showOnlyEnabled" type="checkbox" />
					{{ t("plugin.onlyEnabled") }}
				</label>
				<button type="button" :disabled="loading || operating || activeDefaultHome === systemDefaultHome" @click="resetDefaultHome">
					{{ t("plugin.action.resetDefault") }}
				</button>
				<button type="button" :disabled="loading || operating || pluginStore.isSyncing" @click="refreshPlugins">
					{{ loading ? t("plugin.refreshing") : t("plugin.refresh") }}
				</button>
			</div>
		</header>

		<p v-if="error" class="error">{{ error }}</p>
		<p v-if="info" class="success">{{ info }}</p>

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
						<button v-else type="button" class="is-placeholder" disabled>{{ t("plugin.action.visit") }}</button>
						<button v-if="canSetDefault(item.id)" type="button" :disabled="operating" @click="setAsDefaultHome(item.id)">{{ t("plugin.action.setDefault") }}</button>
						<button v-else type="button" class="is-placeholder" disabled>{{ t("plugin.action.setDefault") }}</button>
						<button v-if="canEnable(item.id)" type="button" :disabled="operating" @click="runAction('enable', item.id)">{{ t("plugin.action.enable") }}</button>
						<button v-else type="button" class="is-placeholder" disabled>{{ t("plugin.action.enable") }}</button>
						<button v-if="canDisable(item.id)" type="button" :disabled="operating" @click="runAction('disable', item.id)">{{ t("plugin.action.disable") }}</button>
						<button v-else type="button" class="is-placeholder" disabled>{{ t("plugin.action.disable") }}</button>
						<button v-if="canUninstall(item.id)" type="button" :disabled="operating" @click="runAction('uninstall', item.id)">{{ t("plugin.action.uninstall") }}</button>
						<button v-else type="button" class="is-placeholder" disabled>{{ t("plugin.action.uninstall") }}</button>
						<button type="button" :disabled="operating" @click="openDebug(item.id)">{{ t("plugin.action.debug") }}</button>
						<button type="button" :disabled="operating" @click="openLogs(item.id)">{{ t("plugin.action.logs") }}</button>
						<span v-if="activeDefaultHome === pluginEntryPath(item.id) && pluginEntryPath(item.id)" class="default-home-badge">{{ t("plugin.defaultHomeActive") }}</span>
					</td>
				</tr>
			</tbody>
		</table>

		<section v-if="selectedPlugin" class="inspector">
			<h3>{{ t("plugin.inspector") }}: {{ selectedPlugin }}</h3>
			<article v-if="activeInspectorPanel === 'info'">
				<h4>{{ t("plugin.debug") }}</h4>
				<pre>{{ debugText || t("plugin.noDebug") }}</pre>
			</article>
			<article v-else-if="activeInspectorPanel === 'logs'">
				<h4>{{ t("plugin.logs") }}</h4>
				<pre>{{ logText || t("plugin.noLogs") }}</pre>
			</article>
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

.sync-status {
	font-size: 12px;
}

.sync-degraded {
	font-size: 12px;
	color: #b45309;
}

.success {
	color: var(--color-success);
	margin: 4px 0;
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
	display: flex;
	flex-wrap: wrap;
	gap: 6px;
	align-items: center;
}

.action-cell button {
	width: auto;
	text-align: center;
	white-space: nowrap;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	padding: 7px 10px;
	line-height: 1;
	box-sizing: border-box;
}

.action-cell button.is-placeholder {
	background: var(--color-surface-soft);
	color: var(--color-text-muted);
	box-shadow: inset 0 0 0 1px var(--color-border);
	cursor: not-allowed;
	opacity: 0.9;
}

.default-home-badge {
	margin-left: 2px;
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
</style>

