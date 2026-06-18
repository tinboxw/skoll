<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import {
	CircleCheck,
	Delete,
	Document,
	HomeFilled,
	Link,
	Refresh,
	Setting,
	Star,
	StarFilled,
	SwitchButton,
	Tickets,
	Tools,
	Upload,
	View,
	WarningFilled
} from "@element-plus/icons-vue";

import SchemaForm from "../../components/Common/SchemaForm.vue";
import StateBlock from "../../components/Common/StateBlock.vue";
import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import { BUTTON_ACCESS, useButtonAccess } from "../../permissions/button";
import { syncBackendPlugins } from "../../plugins";
import type { FrontendPluginManifest, PluginConfigSchema } from "../../plugins/types";
import { clearDefaultHomePath, createDefaultHomeTarget, getDefaultHomePath, getSystemDefaultHomePath, isDefaultHomePlugin, resolvePluginEntryPath, setDefaultHomeTarget, usePluginStore } from "../../stores/plugins";
import { useTabsStore } from "../../stores/tabs";
import { ApiError, type ApiResponse } from "../../utils/api";
import { API_BASE_PREFIX } from "../../utils/api-base-prefix";
import { apiDelete, apiGet, apiPost, apiPut } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

const router = useRouter();
const pluginStore = usePluginStore();
const tabsStore = useTabsStore();
const { t, locale } = useI18n();
const buttonAccess = useButtonAccess();

const loading = ref(false);
const operating = ref(false);
const error = ref<string | null>(null);
const showOnlyEnabled = ref(false);
const pluginKeyword = ref("");
const pluginStatusFilter = ref("");
const debugText = ref("");
const logText = ref("");
const configText = ref("{}");
const configForm = ref<Record<string, unknown>>({});
const configSchema = ref<PluginConfigSchema | null>(null);
const configFormValid = ref(true);
const operationText = ref("");
const installPreflight = ref<Record<string, unknown> | null>(null);
const validatedPluginPath = ref("");
const selectedPlugin = ref("");
const activeInspectorPanel = ref<"info" | "logs" | "config" | "">("");
const pluginPath = ref("");
const activeDefaultHome = ref(getDefaultHomePath());
const systemDefaultHome = getSystemDefaultHomePath();
const info = ref<string | null>(null);
const devLoading = ref(false);
const devPluginsRoot = ref("plugins");
const devSelectedPlugin = ref("");
const devOutputDir = ref("");
const devTargetEnv = ref("staging");
const devArtifactPath = ref("");
const devReleaseVersion = ref("");
const devChangelog = ref("");
const devReviewComment = ref("");
const devScaffoldPluginID = ref("");
const devScaffoldPluginName = ref("");
const devScaffoldAppID = ref("");
const devScaffoldMode = ref("workspace");
const devRolloutStrategy = ref<"percent" | "tag" | "canary">("percent");
const devRolloutPercent = ref(10);
const devRolloutTags = ref("");
const devCanaryVersion = ref("");
const devRollbackPercent = ref<number | null>(null);
const devConfig = ref<Record<string, unknown> | null>(null);
const devProjects = ref<Array<Record<string, unknown>>>([]);
const devReleaseOrders = ref<Array<Record<string, unknown>>>([]);
const devReleaseTasks = ref<Array<Record<string, unknown>>>([]);
const devRolloutTasks = ref<Array<Record<string, unknown>>>([]);
const devResultText = ref("");
const devTaskDrawerOpen = ref(false);
const devTaskDrawerTitle = ref("");
const devTaskDetail = ref<Record<string, unknown> | null>(null);
const devTaskLogs = ref<Array<Record<string, unknown>>>([]);
const detailDrawerOpen = ref(false);
const detailPluginID = ref("");

const visiblePlugins = computed(() => {
	if (!showOnlyEnabled.value) {
		return pluginStore.items;
	}
	return pluginStore.items.filter((item) => item.enabled !== false);
});

const filteredPlugins = computed(() => {
	const keyword = pluginKeyword.value.trim().toLowerCase();
	return visiblePlugins.value.filter((item) => {
		const matchesKeyword = keyword === "" || pluginMatchesKeyword(item, keyword);
		const matchesStatus = pluginMatchesStatus(item, pluginStatusFilter.value);
		return matchesKeyword && matchesStatus;
	});
});

const systemPlugins = computed(() => {
	return filteredPlugins.value.filter((item) => item.level !== "app");
});

const appPlugins = computed(() => {
	return filteredPlugins.value.filter((item) => item.level === "app");
});

const appPluginsGrouped = computed(() => {
	const grouped: Record<string, typeof appPlugins.value> = {};
	for (const plugin of appPlugins.value) {
		const appId = plugin.appId || "unknown";
		if (!grouped[appId]) {
			grouped[appId] = [];
		}
		grouped[appId].push(plugin);
	}
	return grouped;
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

const enabledPluginCount = computed(() => pluginStore.items.filter((item) => item.enabled !== false).length);
const disabledPluginCount = computed(() => pluginStore.items.length - enabledPluginCount.value);
const appCount = computed(() => Object.keys(appPluginsGrouped.value).length);
const filteredPluginCount = computed(() => filteredPlugins.value.length);
const inaccessiblePluginCount = computed(() => pluginStore.items.filter((item) => !canVisit(item.id)).length);
const configSchemaCount = computed(() => pluginStore.items.filter((item) => (item.configSchema?.fields?.length ?? 0) > 0).length);
const failedDevTaskCount = computed(() => [
	...devReleaseTasks.value,
	...devRolloutTasks.value
].filter((item) => String(item.taskStatus || "").toLowerCase() === "failed").length);
const riskPluginCount = computed(() => disabledPluginCount.value + inaccessiblePluginCount.value + failedDevTaskCount.value + (pluginStore.degradedMode ? 1 : 0));
const hasActivePluginFilters = computed(() => showOnlyEnabled.value || pluginKeyword.value.trim() !== "" || pluginStatusFilter.value !== "");
const canReadPlugins = computed(() => buttonAccess.can(BUTTON_ACCESS.pluginRead));
const canManagePlugins = computed(() => buttonAccess.can(BUTTON_ACCESS.pluginManage));
const installPreflightReady = computed(() => installPreflight.value !== null && pluginPath.value.trim() === validatedPluginPath.value);
const pluginRiskRows = computed(() => pluginStore.items.map((plugin) => ({
	id: plugin.id,
	name: plugin.name,
	level: pluginRiskLevel(plugin),
	factors: pluginRiskFactors(plugin),
	blockers: pluginRiskBlockers(plugin),
	audit: pluginRiskAuditTrail(plugin)
})));
const visibleRiskRows = computed(() => pluginRiskRows.value.filter((row) => row.level !== "low" || row.factors.length > 0));
const detailPlugin = computed(() => getPluginRecord(detailPluginID.value));
const detailReleaseOrders = computed(() => devReleaseOrders.value.filter((item) => taskBelongsToPlugin(item, detailPluginID.value)));
const detailReleaseTasks = computed(() => devReleaseTasks.value.filter((item) => taskBelongsToPlugin(item, detailPluginID.value)));
const detailRolloutTasks = computed(() => devRolloutTasks.value.filter((item) => taskBelongsToPlugin(item, detailPluginID.value)));
const devPluginOptions = computed(() => {
	const ids = new Set<string>();
	for (const item of pluginStore.items) {
		if (item.id) {
			ids.add(item.id);
		}
	}
	for (const item of devProjects.value) {
		const id = String(item.pluginId || "");
		if (id) {
			ids.add(id);
		}
	}
	return Array.from(ids).sort();
});
const devPendingOrders = computed(() => devReleaseOrders.value.filter((item) => String(item.orderStatus || "") === "pending"));
const devApprovedOrders = computed(() => devReleaseOrders.value.filter((item) => String(item.orderStatus || "") === "approved"));

function pluginMatchesKeyword(plugin: FrontendPluginManifest, keyword: string): boolean {
	return [
		plugin.id,
		plugin.name,
		plugin.nameZhCN,
		plugin.nameEnUS,
		plugin.version,
		plugin.description,
		plugin.appId,
		plugin.level,
		plugin.uiMode,
		plugin.mountPolicy,
		plugin.uiOpenMode,
		plugin.uiTabMode,
		plugin.entryPath,
		plugin.backendEndpoint,
		plugin.serviceHealthURL,
		plugin.healthStatus,
		typeof plugin.health === "string" ? plugin.health : plugin.health?.status,
		plugin.signatureStatus,
		plugin.signature?.algorithm,
		plugin.signature?.vendorId,
		plugin.vendor,
		plugin.uiMenu?.label,
		plugin.uiMenu?.labelZhCN,
		plugin.uiMenu?.labelEnUS,
		plugin.uiMenu?.path
	].some((value) => String(value ?? "").toLowerCase().includes(keyword));
}

function pluginMatchesStatus(plugin: FrontendPluginManifest, status: string): boolean {
	if (status === "enabled") {
		return plugin.enabled !== false;
	}
	if (status === "disabled") {
		return plugin.enabled === false;
	}
	if (status === "system") {
		return plugin.level !== "app";
	}
	if (status === "app") {
		return plugin.level === "app";
	}
	if (status === "config") {
		return (plugin.configSchema?.fields?.length ?? 0) > 0;
	}
	if (status === "inaccessible") {
		return !canVisit(plugin.id);
	}
	return true;
}

function resetPluginFilters(): void {
	showOnlyEnabled.value = false;
	pluginKeyword.value = "";
	pluginStatusFilter.value = "";
}

function pluginStatusType(pluginID: string): "success" | "info" {
	const plugin = getPluginRecord(pluginID);
	return plugin?.enabled === false ? "info" : "success";
}

function pluginStatusText(pluginID: string): string {
	const plugin = getPluginRecord(pluginID);
	return plugin?.enabled === false ? t("plugin.status.disabled") : t("plugin.status.enabled");
}

function pluginFailedTasks(pluginID: string): number {
	return [
		...devReleaseTasks.value,
		...devRolloutTasks.value
	].filter((item) => String(item.pluginId || "") === pluginID && String(item.taskStatus || "").toLowerCase() === "failed").length;
}

function pluginRiskType(plugin: FrontendPluginManifest): "success" | "warning" | "danger" | "info" {
	if (pluginFailedTasks(plugin.id) > 0) {
		return "danger";
	}
	if (plugin.enabled === false || !canVisit(plugin.id) || pluginStore.degradedMode) {
		return "warning";
	}
	return "success";
}

function pluginRiskText(plugin: FrontendPluginManifest): string {
	const failedTasks = pluginFailedTasks(plugin.id);
	if (failedTasks > 0) {
		return `${t("plugin.risk.failedTasks")}: ${failedTasks}`;
	}
	if (plugin.enabled === false) {
		return t("plugin.risk.disabled");
	}
	if (!canVisit(plugin.id)) {
		return t("plugin.risk.inaccessible");
	}
	if (pluginStore.degradedMode) {
		return t("plugin.risk.degraded");
	}
	return t("plugin.risk.clear");
}

function pluginRiskLevel(plugin: FrontendPluginManifest): "critical" | "high" | "medium" | "low" {
	if (pluginFailedTasks(plugin.id) > 0) {
		return "critical";
	}
	if (plugin.enabled === false || !canVisit(plugin.id)) {
		return "high";
	}
	if (pluginStore.degradedMode || pluginSignatureType(plugin) === "warning") {
		return "medium";
	}
	return "low";
}

function pluginRiskLevelType(level: "critical" | "high" | "medium" | "low"): "success" | "warning" | "danger" | "info" {
	if (level === "critical" || level === "high") {
		return "danger";
	}
	if (level === "medium") {
		return "warning";
	}
	return "success";
}

function pluginRiskFactors(plugin: FrontendPluginManifest): string[] {
	const factors: string[] = [];
	if (plugin.enabled === false) {
		factors.push(t("plugin.risk.disabled"));
	}
	if (!canVisit(plugin.id)) {
		factors.push(t("plugin.risk.inaccessible"));
	}
	if (pluginStore.degradedMode) {
		factors.push(t("plugin.risk.degraded"));
	}
	if (pluginFailedTasks(plugin.id) > 0) {
		factors.push(`${t("plugin.risk.failedTasks")} ${pluginFailedTasks(plugin.id)}`);
	}
	if (pluginSignatureType(plugin) === "warning") {
		factors.push(t("plugin.signal.signature"));
	}
	return factors;
}

function pluginRiskBlockers(plugin: FrontendPluginManifest): string {
	if (pluginFailedTasks(plugin.id) > 0) {
		return "存在失败发布/灰度任务";
	}
	if (plugin.enabled === false) {
		return "插件已停用";
	}
	if (!canVisit(plugin.id)) {
		return "入口或权限不可访问";
	}
	if (pluginStore.degradedMode) {
		return "后端同步降级";
	}
	if (pluginSignatureType(plugin) === "warning") {
		return "签名未验证";
	}
	return "无阻断";
}

function pluginRiskAuditTrail(plugin: FrontendPluginManifest): string {
	if (pluginFailedTasks(plugin.id) > 0) {
		return "plugin.dev.release / plugin.dev.rollout";
	}
	if (plugin.enabled === false) {
		return "plugin.lifecycle.disable";
	}
	if (!canVisit(plugin.id)) {
		return "system.security.deny";
	}
	return "plugin.lifecycle";
}

function pluginHealthType(plugin: FrontendPluginManifest): "success" | "warning" | "danger" | "info" {
	const status = pluginHealthStatus(plugin);
	if (["error", "failed", "unhealthy", "down"].includes(status)) {
		return "danger";
	}
	if (["degraded", "warning", "warn"].includes(status)) {
		return "warning";
	}
	if (["healthy", "ok", "ready", "success", "online"].includes(status)) {
		return "success";
	}
	if (plugin.enabled === false) {
		return "info";
	}
	return canVisit(plugin.id) ? "success" : "warning";
}

function pluginHealthText(plugin: FrontendPluginManifest): string {
	const status = pluginHealthStatus(plugin);
	if (status !== "") {
		return status;
	}
	if (plugin.enabled === false) {
		return t("plugin.health.disabled");
	}
	if (plugin.serviceHealthURL) {
		return t("plugin.health.endpoint");
	}
	return canVisit(plugin.id) ? t("plugin.health.ready") : t("plugin.health.blocked");
}

function pluginHealthStatus(plugin: FrontendPluginManifest): string {
	if (typeof plugin.healthStatus === "string" && plugin.healthStatus.trim() !== "") {
		return plugin.healthStatus.trim().toLowerCase();
	}
	if (typeof plugin.health === "string" && plugin.health.trim() !== "") {
		return plugin.health.trim().toLowerCase();
	}
	const health = plugin.health;
	if (health && typeof health !== "string" && typeof health.status === "string") {
		return health.status.trim().toLowerCase();
	}
	return "";
}

function pluginSignatureType(plugin: FrontendPluginManifest): "success" | "warning" | "danger" | "info" {
	const status = pluginSignatureStatus(plugin);
	if (["invalid", "failed", "untrusted", "error"].includes(status)) {
		return "danger";
	}
	if (["verified", "valid", "signed", "trusted"].includes(status) || plugin.signature?.value || plugin.systemBuiltin) {
		return "success";
	}
	if (["missing", "unsigned", "unknown"].includes(status)) {
		return "warning";
	}
	return "info";
}

function pluginSignatureText(plugin: FrontendPluginManifest): string {
	const status = pluginSignatureStatus(plugin);
	if (status) {
		return status;
	}
	if (plugin.signature?.algorithm) {
		return plugin.signature.algorithm;
	}
	if (plugin.systemBuiltin) {
		return t("plugin.signature.builtin");
	}
	return t("plugin.signature.unsigned");
}

function pluginSignatureStatus(plugin: FrontendPluginManifest): string {
	if (typeof plugin.signatureStatus === "string" && plugin.signatureStatus.trim() !== "") {
		return plugin.signatureStatus.trim().toLowerCase();
	}
	if (plugin.signature?.value) {
		return "signed";
	}
	return "";
}

function taskBelongsToPlugin(item: Record<string, unknown>, pluginID: string): boolean {
	return pluginID !== "" && String(item.pluginId || item.pluginID || "") === pluginID;
}

function openPluginDetail(pluginID: string): void {
	if (!ensurePluginAccess("plugin.read")) {
		return;
	}
	detailPluginID.value = pluginID;
	detailDrawerOpen.value = true;
}

function pluginRequiredPermissions(plugin: FrontendPluginManifest): string[] {
	return plugin.uiMenu?.requiredPermissions ?? [];
}

function pluginRequiredRoles(plugin: FrontendPluginManifest): string[] {
	return plugin.uiMenu?.requiredRoles ?? [];
}

function pluginConfigFieldCount(plugin: FrontendPluginManifest): number {
	return plugin.configSchema?.fields?.length ?? 0;
}

function pluginAssetRows(plugin: FrontendPluginManifest): Array<{ label: string; value: string }> {
	return [
		{ label: "entryPath", value: pluginEntryPath(plugin.id) || "-" },
		{ label: "backendEndpoint", value: plugin.backendEndpoint || "-" },
		{ label: "serviceHealthURL", value: plugin.serviceHealthURL || "-" },
		{ label: "appId", value: plugin.appId || "-" },
		{ label: "uiMode", value: plugin.uiMode || "frontend" },
		{ label: "uiOpenMode", value: plugin.uiOpenMode || "-" },
		{ label: "uiTabMode", value: plugin.uiTabMode || "-" },
		{ label: "locales", value: plugin.i18nLocales?.join(", ") || "-" }
	];
}

function preflightValue(key: string): string {
	const value = installPreflight.value?.[key];
	if (value === undefined || value === null || value === "") {
		return "-";
	}
	return String(value);
}

function preflightCount(key: string): number {
	const value = installPreflight.value?.[key];
	return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function preflightRiskType(): "success" | "warning" {
	return preflightCount("permissions") > 0 || preflightCount("dependencies") > 0 ? "warning" : "success";
}

function handlePluginCommand(command: string): void {
	const separator = command.indexOf(":");
	if (separator < 0) {
		return;
	}
	const action = command.slice(0, separator);
	const pluginID = command.slice(separator + 1);
	if (!pluginID) {
		return;
	}
	if (action === "config") {
		if (!ensurePluginAccess("plugin.read")) {
			return;
		}
		void openConfig(pluginID);
		return;
	}
	if (action === "debug") {
		if (!ensurePluginAccess("plugin.read")) {
			return;
		}
		void openDebug(pluginID);
		return;
	}
	if (action === "logs") {
		if (!ensurePluginAccess("plugin.read")) {
			return;
		}
		void openLogs(pluginID);
		return;
	}
	if (action === "enable" || action === "disable" || action === "uninstall") {
		if (!ensurePluginAccess("plugin.manage")) {
			return;
		}
		void runAction(action, pluginID);
	}
}

function ensurePluginAccess(permission: "plugin.read" | "plugin.manage"): boolean {
	const rule = permission === "plugin.manage" ? BUTTON_ACCESS.pluginManage : BUTTON_ACCESS.pluginRead;
	if (buttonAccess.can(rule)) {
		return true;
	}
	error.value = t("error.forbidden");
	return false;
}

async function refreshPlugins(): Promise<void> {
	if (!ensurePluginAccess("plugin.read")) {
		return;
	}
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
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	if ((action === "disable" || action === "uninstall") && !(await confirmPluginAction(action, pluginID))) {
		return;
	}
	operating.value = true;
	error.value = null;
	info.value = null;
	const beforeActionEntryPath = pluginEntryPath(pluginID);
	const beforeActionPlugin = getPluginRecord(pluginID);
	try {
		if (action === "uninstall") {
			await apiDelete<ApiResponse<unknown>>(`/v1/plugins/${pluginID}`);
		} else {
			await apiPost<ApiResponse<unknown>>(`/v1/plugins/${pluginID}/${action}`);
		}
		if ((action === "disable" || action === "uninstall") && beforeActionEntryPath && beforeActionPlugin && isDefaultHomePlugin(beforeActionPlugin)) {
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

async function confirmPluginAction(action: "disable" | "uninstall", pluginID: string): Promise<boolean> {
	const actionText = action === "disable" ? t("plugin.action.disable") : t("plugin.action.uninstall");
	return confirmAction({
		title: t("plugin.table.actions"),
		message: `${actionText}: ${pluginID}`,
		confirmText: actionText,
		cancelText: t("common.cancel"),
		type: action === "uninstall" ? "error" : "warning",
		danger: true
	});
}

async function openDebug(pluginID: string): Promise<void> {
	if (!ensurePluginAccess("plugin.read")) {
		return;
	}
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
	if (!ensurePluginAccess("plugin.read")) {
		return;
	}
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

async function openConfig(pluginID: string): Promise<void> {
	if (!ensurePluginAccess("plugin.read")) {
		return;
	}
	selectedPlugin.value = pluginID;
	activeInspectorPanel.value = "config";
	error.value = null;
	info.value = null;
	try {
		const payload = await apiGet<ApiResponse<{ pluginId?: string; config?: Record<string, unknown>; configSchema?: PluginConfigSchema | null }>>(`/v1/plugins/${pluginID}/config`);
		const config = payload.data?.config ?? {};
		configSchema.value = payload.data?.configSchema ?? getPluginRecord(pluginID)?.configSchema ?? null;
		configForm.value = applyConfigDefaults(config, configSchema.value);
		configText.value = JSON.stringify(configForm.value, null, 2);
	} catch (e) {
		error.value = toErrorMessage(e);
		configSchema.value = null;
		configForm.value = {};
		configText.value = "{}";
	}
}

async function saveConfig(): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	if (!selectedPlugin.value) {
		return;
	}
	operating.value = true;
	error.value = null;
	info.value = null;
	try {
		const parsed = configSchema.value ? configForm.value : JSON.parse(configText.value || "{}") as Record<string, unknown>;
		await apiPut<ApiResponse<unknown>>(`/v1/plugins/${selectedPlugin.value}/config`, {
			config: parsed
		});
		configText.value = JSON.stringify(parsed, null, 2);
		info.value = t("plugin.config.saved");
	} catch (e) {
		if (e instanceof SyntaxError) {
			error.value = t("plugin.config.invalidJson");
		} else {
			error.value = toErrorMessage(e);
		}
	} finally {
		operating.value = false;
	}
}

async function validatePluginPath(): Promise<void> {
	if (!ensurePluginAccess("plugin.read")) {
		return;
	}
	if (!pluginPath.value.trim()) {
		error.value = t("plugin.pathRequired");
		return;
	}
	operating.value = true;
	error.value = null;
	info.value = null;
	installPreflight.value = null;
	validatedPluginPath.value = "";
	try {
		const payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/validate", {
			path: pluginPath.value.trim()
		});
		installPreflight.value = payload.data && typeof payload.data === "object" ? payload.data as Record<string, unknown> : {};
		validatedPluginPath.value = pluginPath.value.trim();
		operationText.value = JSON.stringify(payload.data, null, 2);
	} catch (e) {
		error.value = toErrorMessage(e);
		operationText.value = "";
	} finally {
		operating.value = false;
	}
}

async function installPluginPath(): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	if (!pluginPath.value.trim()) {
		error.value = t("plugin.pathRequired");
		return;
	}
	if (!installPreflightReady.value) {
		error.value = "请先完成当前路径的安装预检。";
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
		installPreflight.value = null;
		validatedPluginPath.value = "";
		await refreshPlugins();
	} catch (e) {
		error.value = toErrorMessage(e);
		operationText.value = "";
	} finally {
		operating.value = false;
	}
}

function applyConfigDefaults(config: Record<string, unknown>, schema: PluginConfigSchema | null): Record<string, unknown> {
	const out = { ...config };
	for (const field of schema?.fields ?? []) {
		if (!field.key || out[field.key] !== undefined) {
			continue;
		}
		if (field.type === "boolean") {
			out[field.key] = field.default === "true" || field.default === "1";
		} else if (field.type === "number") {
			const parsed = Number(field.default ?? 0);
			out[field.key] = Number.isFinite(parsed) ? parsed : 0;
		} else {
			out[field.key] = field.default ?? "";
		}
	}
	return out;
}

function handleConfigFormUpdate(next: Record<string, unknown>): void {
	configForm.value = next;
	configText.value = JSON.stringify(next, null, 2);
}

function handleConfigFormValid(next: boolean): void {
	configFormValid.value = next;
}

async function refreshDevPortal(): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	devLoading.value = true;
	error.value = null;
	try {
		const configPayload = await apiGet<ApiResponse<Record<string, unknown>>>("/v1/plugins/dev/config");
		devConfig.value = configPayload.data ?? {};
		const defaultRoot = String(devConfig.value.defaultRoot || "").trim();
		if (defaultRoot && devPluginsRoot.value.trim() === "plugins") {
			devPluginsRoot.value = defaultRoot;
		}
		await Promise.all([
			loadDevProjects(),
			loadDevReleaseOrders(),
			loadDevReleaseTasks(),
			loadDevRolloutTasks()
		]);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		devLoading.value = false;
	}
}

async function loadDevProjects(): Promise<void> {
	const payload = await apiPost<ApiResponse<{ projects?: Array<Record<string, unknown>> }>>("/v1/plugins/dev/projects", {
		pluginsRoot: devPluginsRoot.value.trim()
	});
	devProjects.value = payload.data?.projects ?? [];
	if (!devSelectedPlugin.value && devPluginOptions.value.length > 0) {
		devSelectedPlugin.value = devPluginOptions.value[0];
	}
}

async function loadDevReleaseOrders(): Promise<void> {
	const query = new URLSearchParams();
	query.set("pluginsRoot", devPluginsRoot.value.trim());
	if (devSelectedPlugin.value) {
		query.set("pluginId", devSelectedPlugin.value);
	}
	const payload = await apiGet<ApiResponse<{ orders?: Array<Record<string, unknown>> }>>(`/v1/plugins/dev/release-orders?${query.toString()}`);
	devReleaseOrders.value = payload.data?.orders ?? [];
}

async function loadDevReleaseTasks(): Promise<void> {
	const query = new URLSearchParams();
	query.set("pluginsRoot", devPluginsRoot.value.trim());
	if (devSelectedPlugin.value) {
		query.set("pluginId", devSelectedPlugin.value);
	}
	const payload = await apiGet<ApiResponse<{ tasks?: Array<Record<string, unknown>> }>>(`/v1/plugins/dev/release-tasks?${query.toString()}`);
	devReleaseTasks.value = payload.data?.tasks ?? [];
}

async function loadDevRolloutTasks(): Promise<void> {
	const query = new URLSearchParams();
	if (devSelectedPlugin.value) {
		query.set("pluginId", devSelectedPlugin.value);
	}
	const suffix = query.toString();
	const payload = await apiGet<ApiResponse<{ tasks?: Array<Record<string, unknown>> }>>(`/v1/plugins/dev/rollout-tasks${suffix ? `?${suffix}` : ""}`);
	devRolloutTasks.value = payload.data?.tasks ?? [];
}

async function runDevAction(action: "validate" | "package" | "pipeline" | "rollout" | "rollback"): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	if ((action === "package" || action === "pipeline" || action === "rollout" || action === "rollback") && !devSelectedPlugin.value) {
		error.value = "请选择插件。";
		return;
	}
	if (action !== "validate" && !(await confirmDevAction(action, devSelectedPlugin.value))) {
		return;
	}
	devLoading.value = true;
	error.value = null;
	info.value = null;
	try {
		let payload: unknown;
		if (action === "validate") {
			payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/dev/validate-all", {
				pluginsRoot: devPluginsRoot.value.trim()
			});
		} else if (action === "package") {
			payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/dev/package", {
				pluginsRoot: devPluginsRoot.value.trim(),
				pluginId: devSelectedPlugin.value,
				outputDir: devOutputDir.value.trim()
			});
		} else if (action === "pipeline") {
			payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/dev/pipeline", {
				pluginsRoot: devPluginsRoot.value.trim(),
				pluginId: devSelectedPlugin.value,
				outputDir: devOutputDir.value.trim()
			});
		} else if (action === "rollout") {
			payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/dev/rollout", {
				pluginId: devSelectedPlugin.value,
				strategyType: devRolloutStrategy.value,
				targetEnv: devTargetEnv.value.trim() || "production",
				percent: devRolloutStrategy.value === "percent" ? devRolloutPercent.value : undefined,
				tags: devRolloutStrategy.value === "tag" ? devRolloutTags.value.split(",").map((item) => item.trim()).filter(Boolean) : undefined,
				canaryVersion: devRolloutStrategy.value === "canary" ? devCanaryVersion.value.trim() : undefined
			});
		} else {
			payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/dev/rollback", {
				pluginId: devSelectedPlugin.value,
				toPercent: devRollbackPercent.value === null ? undefined : devRollbackPercent.value
			});
		}
		devResultText.value = JSON.stringify(payload, null, 2);
		captureDevArtifactPath(payload);
		info.value = "DevPortal 操作已提交。";
		await Promise.all([loadDevProjects(), loadDevReleaseOrders(), loadDevReleaseTasks(), loadDevRolloutTasks()]);
	} catch (e) {
		error.value = toErrorMessage(e);
		devResultText.value = "";
	} finally {
		devLoading.value = false;
	}
}

async function confirmDevAction(action: "package" | "pipeline" | "rollout" | "rollback", pluginID: string): Promise<boolean> {
	const labelMap = {
		package: "打包",
		pipeline: "流水线",
		rollout: "灰度",
		rollback: "回滚"
	};
	const target = action === "rollout" ? `${pluginID} / ${devTargetEnv.value.trim() || "production"}` : pluginID;
	return confirmAction({
		title: "DevPortal 操作确认",
		message: `${labelMap[action]}: ${target}`,
		confirmText: labelMap[action],
		cancelText: t("common.cancel"),
		type: action === "rollback" ? "error" : action === "rollout" ? "warning" : "info",
		danger: action === "rollback" || action === "rollout"
	});
}

async function scaffoldDevPlugin(): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	if (!devScaffoldPluginID.value.trim() || !devScaffoldPluginName.value.trim()) {
		error.value = "请输入插件 ID 和插件名称。";
		return;
	}
	devLoading.value = true;
	error.value = null;
	info.value = null;
	try {
		const payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/dev/scaffold", {
			pluginsRoot: devPluginsRoot.value.trim(),
			pluginId: devScaffoldPluginID.value.trim(),
			pluginName: devScaffoldPluginName.value.trim(),
			appId: devScaffoldAppID.value.trim(),
			mode: devScaffoldMode.value
		});
		devResultText.value = JSON.stringify(payload, null, 2);
		devSelectedPlugin.value = devScaffoldPluginID.value.trim();
		info.value = "插件脚手架已创建。";
		await loadDevProjects();
	} catch (e) {
		error.value = toErrorMessage(e);
		devResultText.value = "";
	} finally {
		devLoading.value = false;
	}
}

async function createDevReleaseOrder(): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	if (!devSelectedPlugin.value) {
		error.value = "请选择插件。";
		return;
	}
	devLoading.value = true;
	error.value = null;
	info.value = null;
	try {
		const payload = await apiPost<ApiResponse<unknown>>("/v1/plugins/dev/release-orders", {
			pluginsRoot: devPluginsRoot.value.trim(),
			pluginId: devSelectedPlugin.value,
			releaseVersion: devReleaseVersion.value.trim(),
			changelog: devChangelog.value.trim()
		});
		devResultText.value = JSON.stringify(payload, null, 2);
		info.value = "发布单已创建。";
		await loadDevReleaseOrders();
	} catch (e) {
		error.value = toErrorMessage(e);
		devResultText.value = "";
	} finally {
		devLoading.value = false;
	}
}

async function reviewDevReleaseOrder(order: Record<string, unknown>, action: "approve" | "reject"): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	const orderID = String(order.orderId || "");
	const pluginID = String(order.pluginId || devSelectedPlugin.value);
	if (!orderID || !pluginID) {
		return;
	}
	const actionText = action === "approve" ? "通过发布单" : "拒绝发布单";
	if (!(await confirmAction({
		title: "发布审批确认",
		message: `${actionText}: ${orderID} / ${pluginID}`,
		confirmText: actionText,
		cancelText: t("common.cancel"),
		type: action === "approve" ? "warning" : "info",
		danger: action === "approve"
	}))) {
		return;
	}
	devLoading.value = true;
	error.value = null;
	info.value = null;
	try {
		const payload = await apiPost<ApiResponse<unknown>>(`/v1/plugins/dev/release-orders/${encodeURIComponent(orderID)}/${action}`, {
			pluginsRoot: devPluginsRoot.value.trim(),
			pluginId: pluginID,
			comment: devReviewComment.value.trim()
		});
		devResultText.value = JSON.stringify(payload, null, 2);
		info.value = action === "approve" ? "发布单已审批通过。" : "发布单已拒绝。";
		await loadDevReleaseOrders();
	} catch (e) {
		error.value = toErrorMessage(e);
		devResultText.value = "";
	} finally {
		devLoading.value = false;
	}
}

async function executeDevReleaseOrder(order: Record<string, unknown>): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	const orderID = String(order.orderId || "");
	const pluginID = String(order.pluginId || devSelectedPlugin.value);
	if (!orderID || !pluginID || !devArtifactPath.value.trim()) {
		error.value = "执行发布需要发布单、插件和 artifactPath。";
		return;
	}
	if (!(await confirmAction({
		title: "发布执行确认",
		message: `${pluginID} -> ${devTargetEnv.value.trim() || "staging"} / ${devArtifactPath.value.trim()}`,
		confirmText: "执行发布",
		cancelText: t("common.cancel"),
		type: "warning",
		danger: true
	}))) {
		return;
	}
	devLoading.value = true;
	error.value = null;
	info.value = null;
	try {
		const payload = await apiPost<ApiResponse<unknown>>(`/v1/plugins/dev/release-orders/${encodeURIComponent(orderID)}/execute`, {
			pluginsRoot: devPluginsRoot.value.trim(),
			pluginId: pluginID,
			targetEnv: devTargetEnv.value.trim() || "staging",
			artifactPath: devArtifactPath.value.trim()
		});
		devResultText.value = JSON.stringify(payload, null, 2);
		info.value = "发布任务已提交。";
		await Promise.all([loadDevReleaseOrders(), loadDevReleaseTasks()]);
	} catch (e) {
		error.value = toErrorMessage(e);
		devResultText.value = "";
	} finally {
		devLoading.value = false;
	}
}

function captureDevArtifactPath(payload: unknown): void {
	const root = payload as { data?: Record<string, unknown> };
	const data = root?.data;
	if (!data) {
		return;
	}
	const direct = String(data.artifactPath || "").trim();
	if (direct) {
		devArtifactPath.value = direct;
		return;
	}
	const steps = Array.isArray(data.steps) ? data.steps : [];
	for (const step of steps) {
		const artifact = String((step as Record<string, unknown>).artifactPath || "").trim();
		if (artifact) {
			devArtifactPath.value = artifact;
		}
	}
}

async function openDevTaskDrawer(kind: "release" | "rollout", task: Record<string, unknown>, view: "detail" | "logs"): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	const taskID = String(task.taskId || "");
	if (!taskID) {
		return;
	}
	devLoading.value = true;
	error.value = null;
	devTaskDrawerOpen.value = true;
	devTaskDrawerTitle.value = `${kind === "release" ? "发布任务" : "灰度任务"} ${taskID}`;
	devTaskDetail.value = null;
	devTaskLogs.value = [];
	try {
		const base = kind === "release" ? "/v1/plugins/dev/release-tasks" : "/v1/plugins/dev/rollout-tasks";
		if (view === "detail") {
			const query = kind === "release" ? `?${new URLSearchParams({ pluginsRoot: devPluginsRoot.value.trim() }).toString()}` : "";
			const payload = await apiGet<ApiResponse<{ task?: Record<string, unknown> }>>(`${base}/${encodeURIComponent(taskID)}${query}`);
			devTaskDetail.value = payload.data?.task ?? null;
			devTaskLogs.value = Array.isArray(devTaskDetail.value?.logs) ? devTaskDetail.value.logs as Array<Record<string, unknown>> : [];
		} else {
			const query = kind === "release" ? `?${new URLSearchParams({ pluginsRoot: devPluginsRoot.value.trim() }).toString()}` : "";
			const payload = await apiGet<ApiResponse<{ logs?: Array<Record<string, unknown>> }>>(`${base}/${encodeURIComponent(taskID)}/logs${query}`);
			devTaskDetail.value = task;
			devTaskLogs.value = payload.data?.logs ?? [];
		}
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		devLoading.value = false;
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
	if (pluginOpenMode(pluginID) === "standalone") {
		return plugin.uiMode !== "backend_only";
	}
	return pluginEntryPath(pluginID) !== "";
}

function pluginOpenMode(pluginID: string): "integrated" | "standalone" {
	const plugin = getPluginRecord(pluginID);
	if (plugin?.uiOpenMode === "standalone") {
		return "standalone";
	}
	return "integrated";
}

function pluginStandalonePageURL(pluginID: string): string {
	const rawLocale = (localStorage.getItem("skoll.ui.locale") || "").trim();
	const locale = rawLocale === "en-US" || rawLocale === "zh-CN" ? rawLocale : "zh-CN";
	return `${window.location.origin}${API_BASE_PREFIX}/v1/plugins/${encodeURIComponent(pluginID)}/page?locale=${encodeURIComponent(locale)}`;
}

function pluginTabMode(pluginID: string): "optional" | "fixed" | "disabled" {
	const plugin = getPluginRecord(pluginID);
	if (plugin?.uiTabMode === "fixed") {
		return "fixed";
	}
	if (plugin?.uiTabMode === "disabled") {
		return "disabled";
	}
	return "optional";
}

function canSetDefault(pluginID: string): boolean {
	return canVisit(pluginID) && pluginOpenMode(pluginID) !== "standalone";
}

function canTogglePin(pluginID: string): boolean {
	return canVisit(pluginID) && pluginTabMode(pluginID) === "optional";
}

function isPinned(pluginID: string): boolean {
	const path = pluginEntryPath(pluginID);
	if (!path) {
		return false;
	}
	if (pluginTabMode(pluginID) === "disabled") {
		return false;
	}
	if (pluginTabMode(pluginID) === "fixed") {
		return true;
	}
	return tabsStore.isPinned(path);
}

function canVisitApp(appId: string, plugins: Array<{ id: string; enabled?: boolean }>): boolean {
	if (appId.trim() === "") {
		return false;
	}
	return plugins.some((item) => canVisit(item.id));
}

async function visitApp(appId: string): Promise<void> {
	if (appId.trim() === "") {
		return;
	}
	await router.push(`/${appId}`);
}

async function visitSystemConsole(): Promise<void> {
	await router.push("/skoll/dashboard");
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
	if (pluginOpenMode(pluginID) === "standalone") {
		window.open(pluginStandalonePageURL(pluginID), "_blank", "noopener,noreferrer");
		return;
	}
	const entryPath = pluginEntryPath(pluginID);
	if (!entryPath) {
		return;
	}
	if (pluginTabMode(pluginID) === "fixed") {
		const plugin = getPluginRecord(pluginID);
		if (plugin) {
			tabsStore.pinPluginTab({
				id: plugin.id,
				label: plugin.name,
				path: entryPath
			});
		}
	}
	await router.push(entryPath);
}

function togglePinTab(pluginID: string): void {
	const plugin = getPluginRecord(pluginID);
	if (!plugin) {
		return;
	}
	if (pluginTabMode(pluginID) !== "optional") {
		return;
	}
	const entryPath = pluginEntryPath(pluginID);
	if (!entryPath) {
		return;
	}
	if (tabsStore.isPinned(entryPath)) {
		tabsStore.unpinByPath(entryPath);
		info.value = t("plugin.tab.unpinned");
		return;
	}
	tabsStore.pinPluginTab({
		id: plugin.id,
		label: plugin.name,
		path: entryPath
	});
	info.value = t("plugin.tab.pinned");
}

function setAsDefaultHome(pluginID: string): void {
	const plugin = getPluginRecord(pluginID);
	if (!plugin) {
		return;
	}
	const target = createDefaultHomeTarget(plugin);
	if (!target) {
		return;
	}
	setDefaultHomeTarget(target);
	activeDefaultHome.value = target.path;
	info.value = t("plugin.defaultHomeActive");
}

function resetDefaultHome(): void {
	clearDefaultHomePath();
	activeDefaultHome.value = getDefaultHomePath(systemDefaultHome);
	info.value = t("plugin.defaultHomeResetDone");
}
</script>

<template>
	<section class="plugin-page">
		<header class="page-header">
			<div class="title-block">
				<h2>{{ t("plugin.title") }}</h2>
				<p>{{ t("plugin.desc") }}</p>
			</div>
			<div class="header-actions">
				<el-switch v-model="showOnlyEnabled" :active-text="t('plugin.onlyEnabled')" />
				<el-button
					:icon="HomeFilled"
					:disabled="loading || operating || activeDefaultHome === systemDefaultHome"
					@click="resetDefaultHome"
				>
					{{ t("plugin.action.resetDefault") }}
				</el-button>
				<el-button
					type="primary"
					:icon="Refresh"
					:loading="loading || pluginStore.isSyncing"
					:disabled="operating || !canReadPlugins"
					@click="refreshPlugins"
				>
					{{ loading ? t("plugin.refreshing") : t("plugin.refresh") }}
				</el-button>
			</div>
		</header>

		<el-alert v-if="error" class="page-alert" type="error" :title="error" show-icon :closable="false" />
		<el-alert v-if="info" class="page-alert" type="success" :title="info" show-icon :closable="false" />
		<el-alert
			v-if="pluginStore.degradedMode"
			class="page-alert"
			type="warning"
			:title="t('plugin.sync.degraded')"
			show-icon
			:closable="false"
		/>
		<StateBlock v-if="!canReadPlugins" type="forbidden" :title="t('plugin.noPermissionTitle')" :description="t('error.forbidden')" />

		<template v-else>
			<section class="summary-grid">
				<el-card shadow="never" class="summary-card">
					<div class="summary-label">{{ t("header.plugins") }}</div>
					<div class="summary-value">{{ pluginStore.items.length }}</div>
					<div class="summary-note">{{ syncStatusText }} · {{ t("plugin.sync.attempts") }} {{ pluginStore.syncAttempts }}</div>
				</el-card>
				<el-card shadow="never" class="summary-card">
					<div class="summary-label">{{ t("plugin.status.enabled") }}</div>
					<div class="summary-value">{{ enabledPluginCount }}</div>
					<div class="summary-note">{{ t("plugin.status.disabled") }} {{ disabledPluginCount }}</div>
				</el-card>
				<el-card shadow="never" class="summary-card">
					<div class="summary-label">{{ t("plugin.appLevel") }}</div>
					<div class="summary-value">{{ appCount }}</div>
					<div class="summary-note">{{ t("plugin.systemLevel") }} {{ systemPlugins.length }}</div>
				</el-card>
				<el-card shadow="never" class="summary-card" :class="{ 'summary-card--warning': riskPluginCount > 0 }">
					<div class="summary-label">{{ t("plugin.summary.risk") }}</div>
					<div class="summary-value">{{ riskPluginCount }}</div>
					<div class="summary-note">{{ t("plugin.summary.inaccessible") }} {{ inaccessiblePluginCount }} · {{ t("plugin.summary.failedTasks") }} {{ failedDevTaskCount }}</div>
				</el-card>
				<el-card shadow="never" class="summary-card">
					<div class="summary-label">{{ t("plugin.summary.configurable") }}</div>
					<div class="summary-value">{{ configSchemaCount }}</div>
					<div class="summary-note">{{ t("plugin.summary.filtered") }} {{ filteredPluginCount }}</div>
				</el-card>
				<el-card shadow="never" class="summary-card is-wide">
					<div class="summary-label">{{ t("plugin.currentDefaultHome") }}</div>
					<div class="summary-path">{{ activeDefaultHome }}</div>
					<div class="summary-note">{{ t("plugin.defaultHomeHint") }}</div>
				</el-card>
			</section>

			<el-card shadow="never" class="filter-panel">
				<el-form label-position="top" class="plugin-filters" @submit.prevent>
					<el-form-item :label="t('plugin.filter.keyword')">
						<el-input v-model="pluginKeyword" clearable :placeholder="t('plugin.filter.keywordPlaceholder')" />
					</el-form-item>
					<el-form-item :label="t('plugin.filter.status')">
						<el-select v-model="pluginStatusFilter" clearable :placeholder="t('plugin.filter.allStatus')">
							<el-option :label="t('plugin.status.enabled')" value="enabled" />
							<el-option :label="t('plugin.status.disabled')" value="disabled" />
							<el-option :label="t('plugin.filter.systemOnly')" value="system" />
							<el-option :label="t('plugin.filter.appOnly')" value="app" />
							<el-option :label="t('plugin.filter.configurable')" value="config" />
							<el-option :label="t('plugin.filter.inaccessible')" value="inaccessible" />
						</el-select>
					</el-form-item>
					<el-form-item class="filter-actions">
						<el-button :disabled="!hasActivePluginFilters" @click="resetPluginFilters">{{ t("common.reset") }}</el-button>
					</el-form-item>
				</el-form>
			</el-card>

		<el-card shadow="never" class="install-panel">
			<template #header>
				<div class="card-header">
					<div>
						<h3>{{ t("plugin.installTitle") }}</h3>
						<p>{{ t("plugin.installDesc") }}</p>
					</div>
					<el-icon><Upload /></el-icon>
				</div>
			</template>
			<div class="install-actions">
				<el-input v-model="pluginPath" :placeholder="t('plugin.pathPlaceholder')" :disabled="operating" clearable />
				<el-button v-permission="'plugin.read'" :icon="CircleCheck" :disabled="operating" @click="validatePluginPath">{{ t("plugin.action.validate") }}</el-button>
				<el-button v-permission="'plugin.manage'" type="primary" :icon="Upload" :disabled="operating || !installPreflightReady" @click="installPluginPath">{{ t("plugin.action.install") }}</el-button>
			</div>
			<div v-if="installPreflight" class="preflight-panel plugin-install-preflight">
				<div class="preflight-header">
					<div>
						<strong>安装预检</strong>
						<p>{{ validatedPluginPath }}</p>
					</div>
					<el-tag :type="preflightRiskType()" effect="light">{{ preflightRiskType() === "warning" ? "需要复核" : "低风险" }}</el-tag>
				</div>
				<el-descriptions :column="3" border>
					<el-descriptions-item label="插件">{{ preflightValue("id") }}</el-descriptions-item>
					<el-descriptions-item label="版本">{{ preflightValue("version") }}</el-descriptions-item>
					<el-descriptions-item label="依赖">{{ preflightCount("dependencies") }}</el-descriptions-item>
					<el-descriptions-item label="权限 diff">{{ preflightCount("permissions") > 0 ? `${preflightCount("permissions")} 项新增/复核` : "无新增权限" }}</el-descriptions-item>
					<el-descriptions-item label="菜单 diff">当前接口未返回菜单变更，安装前需在详情/菜单注册中复核。</el-descriptions-item>
					<el-descriptions-item label="签名">当前 validate 响应未返回签名结果，按未验证处理。</el-descriptions-item>
					<el-descriptions-item label="迁移影响">当前 validate 响应未返回 migration 信息，安装前按未知影响复核。</el-descriptions-item>
					<el-descriptions-item label="风险">{{ preflightRiskType() === "warning" ? "权限或依赖存在变更" : "未发现权限/依赖风险" }}</el-descriptions-item>
					<el-descriptions-item label="安装门禁">{{ installPreflightReady ? "当前路径已预检" : "路径变化后需重新预检" }}</el-descriptions-item>
				</el-descriptions>
			</div>
			<pre v-if="operationText" class="code-block">{{ operationText }}</pre>
		</el-card>

		<el-card shadow="never" class="risk-report-panel plugin-risk-report">
			<template #header>
				<div class="card-header">
					<div>
						<h3>插件风险报告</h3>
						<p>汇总停用、不可访问、降级、签名和发布任务失败等风险信号。</p>
					</div>
					<el-tag :type="visibleRiskRows.length > 0 ? 'warning' : 'success'" effect="light">{{ visibleRiskRows.length }}</el-tag>
				</div>
			</template>
			<StateBlock v-if="visibleRiskRows.length === 0" type="empty" description="暂无插件风险。" />
			<el-table v-else :data="visibleRiskRows" stripe border>
				<el-table-column prop="id" label="插件" min-width="170" show-overflow-tooltip />
				<el-table-column label="风险等级" width="120">
					<template #default="{ row }">
						<el-tag :type="pluginRiskLevelType(row.level)" effect="light">{{ row.level }}</el-tag>
					</template>
				</el-table-column>
				<el-table-column label="风险因子" min-width="220">
					<template #default="{ row }">
						<el-tag v-for="item in row.factors" :key="item" class="detail-tag" effect="plain">{{ item }}</el-tag>
						<span v-if="row.factors.length === 0">-</span>
					</template>
				</el-table-column>
				<el-table-column prop="blockers" label="阻断原因" min-width="180" show-overflow-tooltip />
				<el-table-column prop="audit" label="审计线索" min-width="200" show-overflow-tooltip />
			</el-table>
		</el-card>

		<el-card v-permission="'plugin.manage'" shadow="never" class="devportal-panel">
			<template #header>
				<div class="card-header">
					<div>
						<h3>DevPortal 工作台</h3>
						<p>面向插件开发、打包、发布和灰度回滚的操作入口。</p>
					</div>
					<el-button type="primary" :icon="Refresh" :loading="devLoading" @click="refreshDevPortal">刷新工作台</el-button>
				</div>
			</template>
			<div class="devportal-grid">
				<section class="devportal-controls">
					<el-form label-position="top">
						<el-form-item label="插件根目录">
							<el-input v-model="devPluginsRoot" placeholder="plugins" clearable />
						</el-form-item>
						<el-divider content-position="left">脚手架</el-divider>
						<el-form-item label="新插件 ID">
							<el-input v-model="devScaffoldPluginID" placeholder="例如: report-center" clearable />
						</el-form-item>
						<el-form-item label="新插件名称">
							<el-input v-model="devScaffoldPluginName" placeholder="例如: Report Center" clearable />
						</el-form-item>
						<el-form-item label="App ID / 模式">
							<div class="inline-fields">
								<el-input v-model="devScaffoldAppID" placeholder="可选 appId" clearable />
								<el-select v-model="devScaffoldMode">
									<el-option label="workspace" value="workspace" />
									<el-option label="repository" value="repository" />
								</el-select>
							</div>
						</el-form-item>
						<el-button :icon="Upload" :loading="devLoading" @click="scaffoldDevPlugin">创建脚手架</el-button>
						<el-divider content-position="left">构建与发布</el-divider>
						<el-form-item label="插件">
							<el-select v-model="devSelectedPlugin" filterable clearable placeholder="选择插件" @change="refreshDevPortal">
								<el-option v-for="id in devPluginOptions" :key="id" :label="id" :value="id" />
							</el-select>
						</el-form-item>
						<el-form-item label="输出目录">
							<el-input v-model="devOutputDir" placeholder="默认 plugins/_dist" clearable />
						</el-form-item>
						<el-form-item label="制品路径">
							<el-input v-model="devArtifactPath" placeholder="打包或流水线成功后自动填充" clearable />
						</el-form-item>
						<el-form-item label="发布版本">
							<el-input v-model="devReleaseVersion" placeholder="留空则读取 manifest version" clearable />
						</el-form-item>
						<el-form-item label="变更说明">
							<el-input v-model="devChangelog" type="textarea" :rows="3" resize="vertical" placeholder="发布说明 / 风险点 / 回滚说明" />
						</el-form-item>
						<el-form-item label="审批意见">
							<el-input v-model="devReviewComment" placeholder="审批或拒绝时写入" clearable />
						</el-form-item>
						<el-form-item label="目标环境">
							<el-segmented v-model="devTargetEnv" :options="['staging', 'production']" />
						</el-form-item>
						<el-button type="success" :icon="Document" :loading="devLoading" :disabled="!devSelectedPlugin" @click="createDevReleaseOrder">创建发布单</el-button>
						<el-divider content-position="left">灰度与回滚</el-divider>
						<el-form-item label="灰度策略">
							<el-segmented v-model="devRolloutStrategy" :options="['percent', 'tag', 'canary']" />
						</el-form-item>
						<el-form-item label="灰度比例">
							<el-slider v-model="devRolloutPercent" :disabled="devRolloutStrategy !== 'percent'" :min="0" :max="100" show-input />
						</el-form-item>
						<el-form-item label="标签（逗号分隔）">
							<el-input v-model="devRolloutTags" :disabled="devRolloutStrategy !== 'tag'" placeholder="beta, internal, tenant-a" clearable />
						</el-form-item>
						<el-form-item label="金丝雀版本">
							<el-input v-model="devCanaryVersion" :disabled="devRolloutStrategy !== 'canary'" placeholder="例如: 1.2.0-canary.1" clearable />
						</el-form-item>
						<el-form-item label="回滚比例（可选）">
							<el-input-number v-model="devRollbackPercent" :min="0" :max="100" controls-position="right" />
						</el-form-item>
					</el-form>
					<div class="devportal-actions">
						<el-button :icon="CircleCheck" :loading="devLoading" @click="runDevAction('validate')">全量校验</el-button>
						<el-button :icon="Upload" :loading="devLoading" :disabled="!devSelectedPlugin" @click="runDevAction('package')">打包</el-button>
						<el-button type="primary" :icon="Tools" :loading="devLoading" :disabled="!devSelectedPlugin" @click="runDevAction('pipeline')">流水线</el-button>
						<el-button type="warning" :icon="WarningFilled" :loading="devLoading" :disabled="!devSelectedPlugin" @click="runDevAction('rollout')">灰度</el-button>
						<el-button type="danger" :icon="Refresh" :loading="devLoading" :disabled="!devSelectedPlugin" @click="runDevAction('rollback')">回滚</el-button>
					</div>
					<el-descriptions v-if="devConfig" class="dev-config" :column="1" size="small" border>
						<el-descriptions-item label="启用">{{ String(devConfig.enabled ?? "") }}</el-descriptions-item>
						<el-descriptions-item label="默认根目录">{{ String(devConfig.defaultRoot ?? "") }}</el-descriptions-item>
						<el-descriptions-item label="允许根目录">{{ JSON.stringify(devConfig.allowedRoots ?? []) }}</el-descriptions-item>
					</el-descriptions>
				</section>

				<section class="devportal-data">
					<el-tabs>
						<el-tab-pane label="项目">
							<el-table :data="devProjects" stripe border max-height="260">
								<el-table-column prop="pluginId" label="Plugin ID" min-width="160" show-overflow-tooltip />
								<el-table-column prop="version" label="版本" width="100" />
								<el-table-column prop="status" label="状态" width="100" />
								<el-table-column prop="mode" label="模式" width="110" />
								<el-table-column prop="path" label="路径" min-width="220" show-overflow-tooltip />
							</el-table>
						</el-tab-pane>
						<el-tab-pane label="发布单">
							<el-table :data="devReleaseOrders" stripe border max-height="260">
								<el-table-column prop="orderId" label="Order ID" min-width="180" show-overflow-tooltip />
								<el-table-column prop="pluginId" label="插件" min-width="140" />
								<el-table-column prop="releaseVersion" label="版本" width="110" />
								<el-table-column prop="orderStatus" label="状态" width="110" />
								<el-table-column prop="createdAt" label="创建时间" min-width="180" show-overflow-tooltip />
								<el-table-column label="操作" min-width="220" fixed="right">
									<template #default="{ row }">
										<div class="row-actions">
											<el-button size="small" type="success" :disabled="devLoading || !canManagePlugins || row.orderStatus !== 'pending'" @click="reviewDevReleaseOrder(row, 'approve')">通过</el-button>
											<el-button size="small" type="warning" :disabled="devLoading || !canManagePlugins || row.orderStatus !== 'pending'" @click="reviewDevReleaseOrder(row, 'reject')">拒绝</el-button>
											<el-button size="small" type="primary" :disabled="devLoading || !canManagePlugins || row.orderStatus !== 'approved' || !devArtifactPath" @click="executeDevReleaseOrder(row)">执行</el-button>
										</div>
									</template>
								</el-table-column>
							</el-table>
						</el-tab-pane>
						<el-tab-pane label="发布任务">
							<el-table :data="devReleaseTasks" stripe border max-height="260">
								<el-table-column prop="taskId" label="Task ID" min-width="180" show-overflow-tooltip />
								<el-table-column prop="pluginId" label="插件" min-width="140" />
								<el-table-column prop="targetEnv" label="环境" width="100" />
								<el-table-column prop="taskStatus" label="状态" width="110" />
								<el-table-column prop="failureReason" label="失败原因" min-width="180" show-overflow-tooltip />
								<el-table-column label="查看" width="150" fixed="right">
									<template #default="{ row }">
										<div class="row-actions">
											<el-button size="small" @click="openDevTaskDrawer('release', row, 'detail')">详情</el-button>
											<el-button size="small" @click="openDevTaskDrawer('release', row, 'logs')">日志</el-button>
										</div>
									</template>
								</el-table-column>
							</el-table>
						</el-tab-pane>
						<el-tab-pane label="灰度任务">
							<el-table :data="devRolloutTasks" stripe border max-height="260">
								<el-table-column prop="taskId" label="Task ID" min-width="180" show-overflow-tooltip />
								<el-table-column prop="pluginId" label="插件" min-width="140" />
								<el-table-column prop="action" label="动作" width="100" />
								<el-table-column prop="rolloutPercent" label="比例" width="100" />
								<el-table-column prop="taskStatus" label="状态" width="110" />
								<el-table-column prop="failureReason" label="失败原因" min-width="180" show-overflow-tooltip />
								<el-table-column label="查看" width="150" fixed="right">
									<template #default="{ row }">
										<div class="row-actions">
											<el-button size="small" @click="openDevTaskDrawer('rollout', row, 'detail')">详情</el-button>
											<el-button size="small" @click="openDevTaskDrawer('rollout', row, 'logs')">日志</el-button>
										</div>
									</template>
								</el-table-column>
							</el-table>
						</el-tab-pane>
						<el-tab-pane label="响应">
							<pre class="code-block">{{ devResultText || "暂无操作结果。" }}</pre>
						</el-tab-pane>
					</el-tabs>
				</section>
			</div>
		</el-card>

		<el-drawer v-model="devTaskDrawerOpen" :title="devTaskDrawerTitle" size="46%">
			<el-tabs>
				<el-tab-pane label="详情">
					<pre class="code-block">{{ devTaskDetail ? JSON.stringify(devTaskDetail, null, 2) : "暂无详情。" }}</pre>
				</el-tab-pane>
				<el-tab-pane label="步骤">
					<el-timeline v-if="Array.isArray(devTaskDetail?.steps) && devTaskDetail.steps.length > 0">
						<el-timeline-item
							v-for="(step, index) in devTaskDetail.steps"
							:key="index"
							:type="String(step.status || '') === 'success' || String(step.status || '') === 'ok' ? 'success' : String(step.status || '') === 'failed' ? 'danger' : 'primary'"
							:timestamp="String(step.startedAt || '')"
						>
							<strong>{{ step.name }}</strong>
							<p>{{ step.status }} {{ step.durationMs ? `· ${step.durationMs}ms` : "" }}</p>
							<p v-if="step.message">{{ step.message }}</p>
						</el-timeline-item>
					</el-timeline>
					<el-empty v-else description="暂无步骤。" />
				</el-tab-pane>
				<el-tab-pane label="日志">
					<el-timeline v-if="devTaskLogs.length > 0">
						<el-timeline-item
							v-for="(log, index) in devTaskLogs"
							:key="index"
							:type="String(log.level || '') === 'error' ? 'danger' : 'primary'"
							:timestamp="String(log.timestamp || '')"
						>
							<strong>{{ log.level }}</strong>
							<p>{{ log.step ? `[${log.step}] ` : "" }}{{ log.message }}</p>
						</el-timeline-item>
					</el-timeline>
					<el-empty v-else description="暂无日志。" />
				</el-tab-pane>
			</el-tabs>
		</el-drawer>

		<el-drawer v-model="detailDrawerOpen" :title="detailPlugin ? `${detailPlugin.name} / ${detailPlugin.id}` : 'Plugin detail'" size="58%" class="detail-drawer">
			<template v-if="detailPlugin">
				<section class="detail-summary">
					<el-tag :type="pluginStatusType(detailPlugin.id)" effect="light">{{ pluginStatusText(detailPlugin.id) }}</el-tag>
					<el-tag :type="pluginRiskType(detailPlugin)" effect="plain">{{ t("plugin.signal.risk") }}: {{ pluginRiskText(detailPlugin) }}</el-tag>
					<el-tag :type="pluginHealthType(detailPlugin)" effect="plain">{{ t("plugin.signal.health") }}: {{ pluginHealthText(detailPlugin) }}</el-tag>
					<el-tag :type="pluginSignatureType(detailPlugin)" effect="plain">{{ t("plugin.signal.signature") }}: {{ pluginSignatureText(detailPlugin) }}</el-tag>
				</section>
				<el-tabs class="detail-tabs">
					<el-tab-pane label="权限" name="permissions">
						<div class="detail-section plugin-detail-permissions">
							<el-descriptions :column="2" border>
								<el-descriptions-item label="访问状态">{{ canVisit(detailPlugin.id) ? t("plugin.access.ready") : t("plugin.access.blocked") }}</el-descriptions-item>
								<el-descriptions-item label="管理权限">{{ canManagePlugins ? "plugin.manage" : "-" }}</el-descriptions-item>
								<el-descriptions-item label="菜单权限">
									<el-tag v-for="item in pluginRequiredPermissions(detailPlugin)" :key="item" class="detail-tag" effect="plain">{{ item }}</el-tag>
									<span v-if="pluginRequiredPermissions(detailPlugin).length === 0">-</span>
								</el-descriptions-item>
								<el-descriptions-item label="角色要求">
									<el-tag v-for="item in pluginRequiredRoles(detailPlugin)" :key="item" class="detail-tag" effect="plain">{{ item }}</el-tag>
									<span v-if="pluginRequiredRoles(detailPlugin).length === 0">-</span>
								</el-descriptions-item>
							</el-descriptions>
						</div>
					</el-tab-pane>
					<el-tab-pane label="菜单" name="menu">
						<div class="detail-section plugin-detail-menu">
							<el-descriptions :column="2" border>
								<el-descriptions-item label="label">{{ detailPlugin.uiMenu?.label || detailPlugin.name }}</el-descriptions-item>
								<el-descriptions-item label="path">{{ detailPlugin.uiMenu?.path || pluginEntryPath(detailPlugin.id) || "-" }}</el-descriptions-item>
								<el-descriptions-item label="icon">{{ detailPlugin.uiMenu?.icon || "-" }}</el-descriptions-item>
								<el-descriptions-item label="order">{{ detailPlugin.uiMenu?.order ?? "-" }}</el-descriptions-item>
							</el-descriptions>
						</div>
					</el-tab-pane>
					<el-tab-pane label="配置" name="config">
						<div class="detail-section plugin-detail-config">
							<el-descriptions :column="2" border>
								<el-descriptions-item label="schema">{{ pluginConfigFieldCount(detailPlugin) }}</el-descriptions-item>
								<el-descriptions-item label="selected">{{ selectedPlugin === detailPlugin.id ? activeInspectorPanel || "-" : "-" }}</el-descriptions-item>
							</el-descriptions>
							<el-empty v-if="pluginConfigFieldCount(detailPlugin) === 0" description="暂无配置 Schema。" />
							<el-table v-else :data="detailPlugin.configSchema?.fields ?? []" stripe border>
								<el-table-column prop="key" label="Key" min-width="160" />
								<el-table-column prop="type" label="Type" width="120" />
								<el-table-column prop="required" label="Required" width="120" />
								<el-table-column prop="help" label="Help" min-width="220" show-overflow-tooltip />
							</el-table>
						</div>
					</el-tab-pane>
					<el-tab-pane label="资产" name="assets">
						<div class="detail-section plugin-detail-assets">
							<el-table :data="pluginAssetRows(detailPlugin)" stripe border>
								<el-table-column prop="label" label="Asset" width="180" />
								<el-table-column prop="value" label="Value" min-width="260" show-overflow-tooltip />
							</el-table>
						</div>
					</el-tab-pane>
					<el-tab-pane label="日志" name="logs">
						<div class="detail-section plugin-detail-logs">
							<el-empty v-if="selectedPlugin !== detailPlugin.id || activeInspectorPanel !== 'logs'" description="可从操作菜单加载该插件日志。" />
							<pre v-else class="code-block">{{ logText || t("plugin.noLogs") }}</pre>
						</div>
					</el-tab-pane>
					<el-tab-pane label="发布状态" name="release">
						<div class="detail-section plugin-detail-release">
							<el-descriptions :column="3" border>
								<el-descriptions-item label="发布单">{{ detailReleaseOrders.length }}</el-descriptions-item>
								<el-descriptions-item label="发布任务">{{ detailReleaseTasks.length }}</el-descriptions-item>
								<el-descriptions-item label="灰度/回滚">{{ detailRolloutTasks.length }}</el-descriptions-item>
							</el-descriptions>
							<el-table :data="[...detailReleaseTasks, ...detailRolloutTasks]" stripe border>
								<el-table-column prop="taskId" label="Task ID" min-width="180" show-overflow-tooltip />
								<el-table-column prop="action" label="Action" width="110" />
								<el-table-column prop="targetEnv" label="Env" width="110" />
								<el-table-column prop="taskStatus" label="Status" width="120" />
								<el-table-column prop="failureReason" label="Failure" min-width="220" show-overflow-tooltip />
							</el-table>
						</div>
					</el-tab-pane>
				</el-tabs>
			</template>
			<el-empty v-else description="请选择插件。" />
		</el-drawer>

		<div class="content-grid">
			<div class="table-stack">
				<el-card shadow="never">
					<template #header>
						<div class="table-header">
							<h3>{{ t("plugin.systemLevel") }}</h3>
							<el-button size="small" :icon="Link" :disabled="operating" @click="visitSystemConsole">{{ t("plugin.action.visit") }}</el-button>
						</div>
					</template>
					<StateBlock v-if="systemPlugins.length === 0" type="empty" :description="hasActivePluginFilters ? t('plugin.filter.empty') : t('plugin.empty.system')" />
					<el-table v-else :data="systemPlugins" stripe border>
						<el-table-column prop="id" :label="t('plugin.table.id')" min-width="180" show-overflow-tooltip />
						<el-table-column prop="name" :label="t('plugin.table.name')" min-width="160" show-overflow-tooltip />
						<el-table-column prop="version" :label="t('plugin.table.version')" width="110" />
						<el-table-column :label="t('plugin.table.mode')" width="130">
							<template #default="{ row }">
								<el-tag effect="plain">{{ row.uiMode || "frontend" }}</el-tag>
							</template>
						</el-table-column>
						<el-table-column :label="t('plugin.table.signals')" min-width="260">
							<template #default="{ row }">
								<div class="signal-stack">
									<el-tag :type="pluginRiskType(row)" effect="plain">{{ t("plugin.signal.risk") }}: {{ pluginRiskText(row) }}</el-tag>
									<el-tag :type="pluginHealthType(row)" effect="plain">{{ t("plugin.signal.health") }}: {{ pluginHealthText(row) }}</el-tag>
									<el-tag :type="pluginSignatureType(row)" effect="plain">{{ t("plugin.signal.signature") }}: {{ pluginSignatureText(row) }}</el-tag>
									<span class="signal-meta">{{ t("plugin.table.version") }} {{ row.version || "-" }}</span>
								</div>
							</template>
						</el-table-column>
						<el-table-column :label="t('plugin.table.status')" width="110">
							<template #default="{ row }">
								<el-tag :type="pluginStatusType(row.id)" effect="light">{{ pluginStatusText(row.id) }}</el-tag>
							</template>
						</el-table-column>
						<el-table-column :label="t('plugin.table.access')" min-width="170">
							<template #default="{ row }">
								<el-tag :type="canVisit(row.id) ? 'success' : 'warning'" effect="plain">
									{{ canVisit(row.id) ? t("plugin.access.ready") : t("plugin.access.blocked") }}
								</el-tag>
							</template>
						</el-table-column>
						<el-table-column :label="t('plugin.table.actions')" min-width="320" fixed="right">
							<template #default="{ row }">
								<div class="row-actions">
									<el-button size="small" type="primary" plain :icon="View" :disabled="operating || !canVisit(row.id)" @click="visitPlugin(row.id)">
										{{ t("plugin.action.visit") }}
									</el-button>
									<el-button size="small" :icon="Document" :disabled="operating" @click="openPluginDetail(row.id)">
										详情
									</el-button>
									<el-button size="small" :icon="isPinned(row.id) ? StarFilled : Star" :disabled="operating || !canTogglePin(row.id)" @click="togglePinTab(row.id)">
										{{ isPinned(row.id) ? t("plugin.action.unpinTab") : t("plugin.action.pinTab") }}
									</el-button>
									<el-button size="small" :icon="HomeFilled" :disabled="operating || !canSetDefault(row.id)" @click="setAsDefaultHome(row.id)">
										{{ t("plugin.action.setDefault") }}
									</el-button>
									<el-dropdown trigger="click" :disabled="operating" @command="handlePluginCommand">
										<el-button size="small" :icon="Tools">{{ t("plugin.table.actions") }}</el-button>
										<template #dropdown>
											<el-dropdown-menu>
												<el-dropdown-item v-permission="'plugin.read'" :icon="Setting" :command="`config:${row.id}`">{{ t("plugin.action.config") }}</el-dropdown-item>
												<el-dropdown-item v-permission="'plugin.read'" :icon="Document" :command="`debug:${row.id}`">{{ t("plugin.action.debug") }}</el-dropdown-item>
												<el-dropdown-item v-permission="'plugin.read'" :icon="Tickets" :command="`logs:${row.id}`">{{ t("plugin.action.logs") }}</el-dropdown-item>
												<el-dropdown-item v-if="canEnable(row.id)" v-permission="'plugin.manage'" :icon="SwitchButton" :command="`enable:${row.id}`">{{ t("plugin.action.enable") }}</el-dropdown-item>
												<el-dropdown-item v-if="canDisable(row.id)" v-permission="'plugin.manage'" :icon="WarningFilled" :command="`disable:${row.id}`">{{ t("plugin.action.disable") }}</el-dropdown-item>
												<el-dropdown-item v-if="canUninstall(row.id)" v-permission="'plugin.manage'" divided :icon="Delete" :command="`uninstall:${row.id}`">{{ t("plugin.action.uninstall") }}</el-dropdown-item>
											</el-dropdown-menu>
										</template>
									</el-dropdown>
									<el-tag v-if="activeDefaultHome === pluginEntryPath(row.id) && pluginEntryPath(row.id)" type="success" effect="plain">
										{{ t("plugin.defaultHomeActive") }}
									</el-tag>
								</div>
							</template>
						</el-table-column>
					</el-table>
				</el-card>

				<el-card shadow="never">
					<template #header>
						<div class="table-header">
							<h3>{{ t("plugin.appLevel") }}</h3>
							<el-tag effect="plain">{{ appCount }}</el-tag>
						</div>
					</template>
					<StateBlock v-if="Object.keys(appPluginsGrouped).length === 0" type="empty" :description="hasActivePluginFilters ? t('plugin.filter.empty') : t('plugin.empty.app')" />
					<el-collapse v-else>
						<el-collapse-item v-for="(plugins, appId) in appPluginsGrouped" :key="appId" :name="String(appId)">
							<template #title>
								<div class="app-title">
									<strong>{{ appId }}</strong>
									<el-tag size="small" effect="plain">{{ plugins.length }}</el-tag>
								</div>
							</template>
							<div class="app-toolbar">
								<el-button
									size="small"
									:icon="Link"
									:disabled="operating || !canVisitApp(String(appId), plugins)"
									@click="visitApp(String(appId))"
								>
									{{ t("plugin.action.visit") }} /{{ appId }}
								</el-button>
							</div>
							<el-table :data="plugins" stripe border>
								<el-table-column prop="id" :label="t('plugin.table.id')" min-width="180" show-overflow-tooltip />
								<el-table-column prop="name" :label="t('plugin.table.name')" min-width="160" show-overflow-tooltip />
								<el-table-column prop="version" :label="t('plugin.table.version')" width="110" />
								<el-table-column prop="appId" :label="t('plugin.table.appId')" min-width="120" show-overflow-tooltip />
								<el-table-column :label="t('plugin.table.signals')" min-width="260">
									<template #default="{ row }">
										<div class="signal-stack">
											<el-tag :type="pluginRiskType(row)" effect="plain">{{ t("plugin.signal.risk") }}: {{ pluginRiskText(row) }}</el-tag>
											<el-tag :type="pluginHealthType(row)" effect="plain">{{ t("plugin.signal.health") }}: {{ pluginHealthText(row) }}</el-tag>
											<el-tag :type="pluginSignatureType(row)" effect="plain">{{ t("plugin.signal.signature") }}: {{ pluginSignatureText(row) }}</el-tag>
											<span class="signal-meta">{{ t("plugin.table.version") }} {{ row.version || "-" }}</span>
										</div>
									</template>
								</el-table-column>
								<el-table-column :label="t('plugin.table.status')" width="110">
									<template #default="{ row }">
										<el-tag :type="pluginStatusType(row.id)" effect="light">{{ pluginStatusText(row.id) }}</el-tag>
									</template>
								</el-table-column>
								<el-table-column :label="t('plugin.table.access')" min-width="170">
									<template #default="{ row }">
										<el-tag :type="canVisit(row.id) ? 'success' : 'warning'" effect="plain">
											{{ canVisit(row.id) ? t("plugin.access.ready") : t("plugin.access.blocked") }}
										</el-tag>
									</template>
								</el-table-column>
								<el-table-column :label="t('plugin.table.actions')" min-width="300" fixed="right">
									<template #default="{ row }">
										<div class="row-actions">
											<el-button size="small" :icon="Document" :disabled="operating" @click="openPluginDetail(row.id)">
												详情
											</el-button>
											<el-button size="small" :icon="isPinned(row.id) ? StarFilled : Star" :disabled="operating || !canTogglePin(row.id)" @click="togglePinTab(row.id)">
												{{ isPinned(row.id) ? t("plugin.action.unpinTab") : t("plugin.action.pinTab") }}
											</el-button>
											<el-button size="small" :icon="HomeFilled" :disabled="operating || !canSetDefault(row.id)" @click="setAsDefaultHome(row.id)">
												{{ t("plugin.action.setDefault") }}
											</el-button>
											<el-dropdown trigger="click" :disabled="operating" @command="handlePluginCommand">
												<el-button size="small" :icon="Tools">{{ t("plugin.table.actions") }}</el-button>
												<template #dropdown>
													<el-dropdown-menu>
														<el-dropdown-item v-permission="'plugin.read'" :icon="Setting" :command="`config:${row.id}`">{{ t("plugin.action.config") }}</el-dropdown-item>
														<el-dropdown-item v-permission="'plugin.read'" :icon="Document" :command="`debug:${row.id}`">{{ t("plugin.action.debug") }}</el-dropdown-item>
														<el-dropdown-item v-permission="'plugin.read'" :icon="Tickets" :command="`logs:${row.id}`">{{ t("plugin.action.logs") }}</el-dropdown-item>
														<el-dropdown-item v-if="canEnable(row.id)" v-permission="'plugin.manage'" :icon="SwitchButton" :command="`enable:${row.id}`">{{ t("plugin.action.enable") }}</el-dropdown-item>
														<el-dropdown-item v-if="canDisable(row.id)" v-permission="'plugin.manage'" :icon="WarningFilled" :command="`disable:${row.id}`">{{ t("plugin.action.disable") }}</el-dropdown-item>
														<el-dropdown-item v-if="canUninstall(row.id)" v-permission="'plugin.manage'" divided :icon="Delete" :command="`uninstall:${row.id}`">{{ t("plugin.action.uninstall") }}</el-dropdown-item>
													</el-dropdown-menu>
												</template>
											</el-dropdown>
											<el-tag v-if="isDefaultHomePlugin(row)" type="success" effect="plain">{{ t("plugin.defaultHomeActive") }}</el-tag>
										</div>
									</template>
								</el-table-column>
							</el-table>
						</el-collapse-item>
					</el-collapse>
				</el-card>
			</div>

			<el-card shadow="never" class="inspector-card">
				<template #header>
					<div class="table-header">
						<h3>{{ t("plugin.inspector") }}</h3>
						<el-tag v-if="selectedPlugin" effect="plain">{{ selectedPlugin }}</el-tag>
					</div>
				</template>
				<el-empty v-if="!selectedPlugin" :description="t('plugin.noDebug')" />
				<template v-else>
					<article v-if="activeInspectorPanel === 'info'">
						<h4>{{ t("plugin.debug") }}</h4>
						<pre class="code-block">{{ debugText || t("plugin.noDebug") }}</pre>
					</article>
					<article v-else-if="activeInspectorPanel === 'config'">
						<h4>{{ t("plugin.action.config") }}</h4>
						<template v-if="configSchema && (configSchema.fields?.length ?? 0) > 0">
							<SchemaForm
								:model-value="configForm"
								:schema="configSchema"
								:locale="locale"
								:disabled="operating || !canManagePlugins"
								@update:model-value="handleConfigFormUpdate"
								@update:valid="handleConfigFormValid"
							/>
							<el-collapse class="config-preview">
								<el-collapse-item title="JSON" name="json">
									<pre class="code-block">{{ configText }}</pre>
								</el-collapse-item>
							</el-collapse>
						</template>
						<el-input v-else v-model="configText" type="textarea" :disabled="operating" :rows="14" resize="vertical" />
						<div class="config-actions">
							<el-button v-permission="'plugin.manage'" type="primary" :icon="Setting" :disabled="operating || !configFormValid" @click="saveConfig">{{ t("plugin.action.saveConfig") }}</el-button>
						</div>
					</article>
					<article v-else-if="activeInspectorPanel === 'logs'">
						<h4>{{ t("plugin.logs") }}</h4>
						<pre class="code-block">{{ logText || t("plugin.noLogs") }}</pre>
					</article>
				</template>
			</el-card>
		</div>
		</template>
	</section>
</template>

<style scoped>
.plugin-page {
	display: grid;
	gap: 16px;
}

.page-header {
	display: flex;
	justify-content: space-between;
	gap: 16px;
	align-items: flex-start;
}

.title-block h2,
.card-header h3,
.table-header h3 {
	margin: 0;
}

.title-block p,
.card-header p,
.summary-note {
	margin: 4px 0 0;
	color: var(--color-text-muted);
	font-size: 0.88rem;
}

.header-actions,
.install-actions,
.row-actions,
.table-header,
.card-header,
.app-toolbar {
	display: flex;
	align-items: center;
	gap: 10px;
}

.header-actions {
	flex-wrap: wrap;
	justify-content: flex-end;
}

.page-alert {
	margin: 0;
}

.summary-grid {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 12px;
}

.summary-card {
	min-height: 116px;
}

.summary-card--warning {
	border-color: var(--color-warning);
	background: var(--color-warning-soft);
}

.summary-card.is-wide {
	grid-column: span 1;
}

.summary-label {
	color: var(--color-text-muted);
	font-size: 0.86rem;
}

.summary-value {
	margin-top: 8px;
	font-size: 2rem;
	font-weight: 700;
	line-height: 1;
	color: var(--color-primary-strong);
}

.summary-path {
	margin-top: 8px;
	font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, Courier New, monospace;
	font-size: 0.92rem;
	font-weight: 700;
	word-break: break-all;
}

.install-panel {
	border: 1px solid var(--color-border);
}

.risk-report-panel {
	border: 1px solid var(--color-border);
}

.filter-panel {
	border: 1px solid var(--color-border);
}

.plugin-filters {
	display: grid;
	grid-template-columns: minmax(220px, 1fr) minmax(180px, 240px) auto;
	gap: 12px;
	align-items: end;
}

.plugin-filters :deep(.el-form-item) {
	margin-bottom: 0;
}

.plugin-filters :deep(.el-select),
.plugin-filters :deep(.el-input) {
	width: 100%;
}

.filter-actions {
	justify-content: flex-end;
}

.card-header,
.table-header {
	justify-content: space-between;
}

.install-actions {
	align-items: stretch;
}

.preflight-panel {
	display: grid;
	gap: 12px;
	margin-top: 12px;
}

.preflight-header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 12px;
}

.preflight-header p {
	margin: 4px 0 0;
	color: var(--color-text-muted);
	font-size: 0.86rem;
	word-break: break-all;
}

.content-grid {
	display: grid;
	grid-template-columns: minmax(0, 1fr) minmax(320px, 420px);
	gap: 16px;
	align-items: start;
}

.devportal-panel {
	border: 1px solid var(--color-border);
}

.devportal-grid {
	display: grid;
	grid-template-columns: minmax(280px, 360px) minmax(0, 1fr);
	gap: 16px;
	align-items: start;
}

.devportal-controls {
	display: grid;
	gap: 12px;
}

.devportal-actions {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}

.inline-fields {
	display: grid;
	grid-template-columns: minmax(0, 1fr) 150px;
	gap: 8px;
	width: 100%;
}

.dev-config {
	margin-top: 4px;
}

.devportal-data {
	min-width: 0;
}

.table-stack {
	display: grid;
	gap: 16px;
	min-width: 0;
}

.row-actions {
	flex-wrap: wrap;
}

.signal-stack {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: 6px;
}

.signal-meta {
	color: var(--color-text-muted);
	font-size: 0.78rem;
	white-space: nowrap;
}

.app-title {
	display: inline-flex;
	align-items: center;
	gap: 8px;
	width: 100%;
}

.app-toolbar {
	justify-content: flex-end;
	margin-bottom: 10px;
}

.inspector-card {
	position: sticky;
	top: 12px;
}

.code-block {
	margin: 12px 0 0;
	padding: 12px;
	border-radius: 8px;
	background: #f6f8fb;
	border: 1px solid var(--color-border);
	white-space: pre-wrap;
	word-break: break-word;
	font-size: 0.78rem;
	max-height: 360px;
	overflow: auto;
}

.config-actions {
	display: flex;
	justify-content: flex-end;
	margin-top: 10px;
}

.detail-summary {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
	margin-bottom: 12px;
}

.detail-tabs {
	min-width: 0;
}

.detail-section {
	display: grid;
	gap: 12px;
}

.detail-tag {
	margin-right: 6px;
	margin-bottom: 4px;
}

:deep(.el-card__header) {
	padding: 14px 16px;
}

:deep(.el-card__body) {
	padding: 16px;
}

:deep(.el-table) {
	--el-table-header-bg-color: var(--color-surface-soft);
}

@media (max-width: 1180px) {
	.summary-grid {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	.plugin-filters {
		grid-template-columns: 1fr 1fr;
	}

	.content-grid {
		grid-template-columns: 1fr;
	}

	.devportal-grid {
		grid-template-columns: 1fr;
	}

	.inspector-card {
		position: static;
	}
}

@media (max-width: 720px) {
	.page-header,
	.install-actions {
		flex-direction: column;
		align-items: stretch;
	}

	.inline-fields {
		grid-template-columns: 1fr;
	}

	.plugin-filters {
		grid-template-columns: 1fr;
	}

	.header-actions {
		justify-content: flex-start;
	}

	.summary-grid {
		grid-template-columns: 1fr;
	}
}
</style>

