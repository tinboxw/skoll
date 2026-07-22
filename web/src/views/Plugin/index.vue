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
import MetricStrip, { type MetricStripItem } from "../../components/Common/MetricStrip.vue";
import PageShell from "../../components/Common/PageShell.vue";
import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import { BUTTON_ACCESS, useButtonAccess } from "../../permissions/button";
import { syncBackendPlugins } from "../../plugins";
import { applyConfigDefaults, normalizePluginConfigSchema } from "../../plugins/config-schema";
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
const activeHeavyPanel = ref<"inventory" | "marketplace" | "risk" | "devportal">("inventory");
const debugText = ref("");
const logText = ref("");
const configText = ref("{}");
const configForm = ref<Record<string, unknown>>({});
const configSchema = ref<PluginConfigSchema | null>(null);
const configFormValid = ref(true);
const operationText = ref("");
const installPreflight = ref<InstallPreflightResult | null>(null);
const validatedPluginPath = ref("");
const selectedPlugin = ref("");
const activeInspectorPanel = ref<"info" | "logs" | "config" | "">("");
const pluginPath = ref("");
const activeDefaultHome = ref(getDefaultHomePath());
const systemDefaultHome = getSystemDefaultHomePath();
const info = ref<string | null>(null);
type MarketplaceSignatureStatus = "signed" | "unsigned" | "incomplete" | "unknown";
type MarketplaceRiskLevel = "low" | "medium" | "high" | "critical" | "unknown";
type InstallPreflightStatus = "pass" | "blocked";
type InstallPreflightRiskLevel = "low" | "medium" | "high" | "critical";
type InstallPreflightResult = {
	status: InstallPreflightStatus;
	plugin: {
		id: string;
		name: string;
		version: string;
		description?: string;
		source: string;
	};
	blockers?: string[];
	warnings?: string[];
	permissions: InstallPreflightDiff<InstallPreflightPermission>;
	menus: InstallPreflightDiff<InstallPreflightMenu>;
	config: {
		hasSchema: boolean;
		fieldCount: number;
		requiredFields?: string[];
		defaultFields?: string[];
	};
	resources: {
		uiMode?: string;
		frontendEntry?: string;
		serviceBaseUrl?: string;
		serviceHealthUrl?: string;
		dependencies?: string[];
	};
	migration: {
		version?: string;
		pending?: InstallPreflightMigrationStep[];
		applied?: InstallPreflightMigrationStep[];
		error?: string;
	};
	signature: {
		status: MarketplaceSignatureStatus | "unsupported";
		algorithm?: string;
		vendorId?: string;
		signedAt?: string;
	};
	risk: {
		level: InstallPreflightRiskLevel;
		summary?: string[];
	};
};
type InstallPreflightDiff<T> = {
	add?: T[];
	update?: T[];
	conflict?: T[];
};
type InstallPreflightPermission = {
	key: string;
	type?: string;
	module?: string;
	name?: string;
	risk?: string;
	source?: string;
	existing?: string;
};
type InstallPreflightMenu = {
	key: string;
	parentKey?: string;
	path?: string;
	name?: string;
	source?: string;
	existing?: string;
	requiredRoles?: string[];
	requiredPermissions?: string[];
};
type InstallPreflightMigrationStep = {
	version: number;
	name: string;
	upPath?: string;
	downPath?: string;
};
type PluginRollbackCheckpoint = {
	name: string;
	status: string;
	before?: string;
	after?: string;
	message?: string;
};
type PluginRollbackPlan = {
	pluginId: string;
	status: string;
	fromVersion?: string;
	toVersion?: string;
	checkpoints?: PluginRollbackCheckpoint[];
};
type LocalMarketplaceItem = {
	id: string;
	name: string;
	version: string;
	description?: string;
	manifestPath?: string;
	packagePath?: string;
	packageDigest?: string;
	packageSizeBytes?: number;
	installable: boolean;
	signature: {
		status: MarketplaceSignatureStatus;
		algorithm?: string;
		vendorId?: string;
		signedAt?: string;
	};
	risk: {
		level: MarketplaceRiskLevel;
		permissions?: string[];
		migrations?: string[];
		network?: string[];
		assets?: string[];
	};
};
const marketplaceLoading = ref(false);
const marketplaceError = ref<string | null>(null);
const marketplaceKeyword = ref("");
const marketplaceRiskFilter = ref("");
const marketplaceSignatureFilter = ref("");
const marketplaceItems = ref<LocalMarketplaceItem[]>([]);
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
const devRollbackPlan = ref<PluginRollbackPlan | null>(null);
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
const summaryItems = computed<MetricStripItem[]>(() => [
	{
		label: t("header.plugins"),
		value: pluginStore.items.length,
		note: `${syncStatusText.value} · ${t("plugin.sync.attempts")} ${pluginStore.syncAttempts}`
	},
	{
		label: t("plugin.status.enabled"),
		value: enabledPluginCount.value,
		note: `${t("plugin.status.disabled")} ${disabledPluginCount.value}`
	},
	{
		label: t("plugin.appLevel"),
		value: appCount.value,
		note: `${t("plugin.systemLevel")} ${systemPlugins.value.length}`
	},
	{
		label: t("plugin.summary.risk"),
		value: riskPluginCount.value,
		note: `${t("plugin.summary.inaccessible")} ${inaccessiblePluginCount.value} · ${t("plugin.summary.failedTasks")} ${failedDevTaskCount.value}`,
		tone: riskPluginCount.value > 0 ? "warning" : "default"
	},
	{
		label: t("plugin.summary.configurable"),
		value: configSchemaCount.value,
		note: `${t("plugin.summary.filtered")} ${filteredPluginCount.value}`
	},
	{
		label: t("plugin.currentDefaultHome"),
		value: activeDefaultHome.value,
		note: t("plugin.defaultHomeHint"),
		wide: true
	}
]);
const hasActivePluginFilters = computed(() => showOnlyEnabled.value || pluginKeyword.value.trim() !== "" || pluginStatusFilter.value !== "");
const hasMarketplaceFilters = computed(() => marketplaceKeyword.value.trim() !== "" || marketplaceRiskFilter.value !== "" || marketplaceSignatureFilter.value !== "");
const filteredMarketplaceItems = computed(() => {
	const keyword = marketplaceKeyword.value.trim().toLowerCase();
	return marketplaceItems.value.filter((item) => {
		const matchesKeyword = keyword === "" || [
			item.id,
			item.name,
			item.version,
			item.description,
			item.manifestPath,
			item.packagePath,
			item.packageDigest,
			item.signature.algorithm,
			item.signature.vendorId,
			...(item.risk.permissions ?? []),
			...(item.risk.migrations ?? []),
			...(item.risk.network ?? []),
			...(item.risk.assets ?? [])
		].some((value) => String(value ?? "").toLowerCase().includes(keyword));
		const matchesRisk = marketplaceRiskFilter.value === "" || item.risk.level === marketplaceRiskFilter.value;
		const matchesSignature = marketplaceSignatureFilter.value === "" || item.signature.status === marketplaceSignatureFilter.value;
		return matchesKeyword && matchesRisk && matchesSignature;
	});
});
const marketplaceInstallableCount = computed(() => marketplaceItems.value.filter((item) => item.installable).length);
const marketplaceHighRiskCount = computed(() => marketplaceItems.value.filter((item) => ["high", "critical"].includes(item.risk.level)).length);
const marketplaceUnsignedCount = computed(() => marketplaceItems.value.filter((item) => item.signature.status !== "signed").length);
const canReadPlugins = computed(() => buttonAccess.can(BUTTON_ACCESS.pluginRead));
const canManagePlugins = computed(() => buttonAccess.can(BUTTON_ACCESS.pluginManage));
const installPreflightReady = computed(() => installPreflight.value !== null && installPreflight.value.status !== "blocked" && pluginPath.value.trim() === validatedPluginPath.value);
const pluginRiskRows = computed(() => pluginStore.items.map((plugin) => ({
	id: plugin.id,
	name: plugin.name,
	level: pluginRiskLevel(plugin),
	factors: pluginRiskFactors(plugin),
	permissions: pluginRequiredPermissions(plugin),
	configFields: plugin.configSchema?.fields?.length ?? 0,
	entryPath: plugin.entryPath || resolvePluginEntryPath(plugin),
	blockers: pluginRiskBlockers(plugin),
	audit: pluginRiskAuditTrail(plugin)
})));
const visibleRiskRows = computed(() => pluginRiskRows.value.filter((row) => row.level !== "low" || row.factors.length > 0));
const preInstallRiskRows = computed(() => filteredMarketplaceItems.value.map((item) => ({
	id: item.id,
	name: item.name,
	version: item.version,
	level: item.risk.level,
	signature: item.signature.status,
	installable: item.installable,
	permissions: item.risk.permissions ?? [],
	impacts: marketplaceRiskItems(item),
	source: marketplaceInstallPath(item),
	audit: "plugin.install.preflight"
})));
const visiblePreInstallRiskRows = computed(() => preInstallRiskRows.value.filter((row) => row.level !== "low" || row.signature !== "signed" || row.impacts.length > 0 || !row.installable));
const riskReportSummary = computed(() => ({
	preInstall: visiblePreInstallRiskRows.value.length,
	installed: visibleRiskRows.value.length,
	permissions: visibleRiskRows.value.reduce((total, row) => total + row.permissions.length, 0) + visiblePreInstallRiskRows.value.reduce((total, row) => total + row.permissions.length, 0)
}));
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
const devTaskSummary = computed(() => {
	const tasks = [...devReleaseTasks.value, ...devRolloutTasks.value];
	return {
		total: tasks.length,
		running: tasks.filter((item) => isDevTaskStatus(item, ["running", "pending", "queued"])).length,
		failed: tasks.filter((item) => isDevTaskStatus(item, ["failed", "error"])).length,
		success: tasks.filter((item) => isDevTaskStatus(item, ["success", "ok", "completed"])).length,
		rollback: devRolloutTasks.value.filter((item) => String(item.action || "").toLowerCase() === "rollback").length
	};
});
const devRollbackCheckpoints = computed(() => devRollbackPlan.value?.checkpoints ?? []);

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

function resetMarketplaceFilters(): void {
	marketplaceKeyword.value = "";
	marketplaceRiskFilter.value = "";
	marketplaceSignatureFilter.value = "";
}

function marketplaceRiskType(level: MarketplaceRiskLevel): "success" | "warning" | "danger" | "info" {
	if (level === "critical" || level === "high") {
		return "danger";
	}
	if (level === "medium") {
		return "warning";
	}
	if (level === "unknown") {
		return "info";
	}
	return "success";
}

function marketplaceSignatureType(status: MarketplaceSignatureStatus): "success" | "warning" | "danger" | "info" {
	if (status === "signed") {
		return "success";
	}
	if (status === "incomplete") {
		return "danger";
	}
	if (status === "unsigned") {
		return "warning";
	}
	return "info";
}

function marketplaceRiskItems(item: LocalMarketplaceItem): string[] {
	return [
		...(item.risk.permissions ?? []),
		...(item.risk.migrations ?? []),
		...(item.risk.network ?? []),
		...(item.risk.assets ?? [])
	];
}

function marketplaceInstallPath(item: LocalMarketplaceItem): string {
	return item.manifestPath || item.packagePath || "";
}

async function loadMarketplace(): Promise<void> {
	if (!ensurePluginAccess("plugin.read")) {
		return;
	}
	marketplaceLoading.value = true;
	marketplaceError.value = null;
	const query = new URLSearchParams();
	if (devPluginsRoot.value.trim()) {
		query.set("pluginsRoot", devPluginsRoot.value.trim());
	}
	try {
		const payload = await apiGet<ApiResponse<{ items?: LocalMarketplaceItem[] }>>(`/v1/plugins/marketplace/local?${query.toString()}`);
		marketplaceItems.value = payload.data?.items ?? [];
	} catch (e) {
		marketplaceError.value = toErrorMessage(e);
		marketplaceItems.value = [];
	} finally {
		marketplaceLoading.value = false;
	}
}

async function prepareMarketplaceInstall(item: LocalMarketplaceItem): Promise<void> {
	if (!ensurePluginAccess("plugin.manage")) {
		return;
	}
	const path = marketplaceInstallPath(item);
	if (!path) {
		error.value = t("plugin.advanced.marketPathMissing");
		return;
	}
	pluginPath.value = path;
	activeInspectorPanel.value = "";
	selectedPlugin.value = "";
	if (item.manifestPath) {
		await validatePluginPath();
		return;
	}
	info.value = t("plugin.advanced.marketPackageSelected");
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
		return t("plugin.advanced.risk.failedTaskBlocker");
	}
	if (plugin.enabled === false) {
		return t("plugin.advanced.risk.disabledBlocker");
	}
	if (!canVisit(plugin.id)) {
		return t("plugin.advanced.risk.inaccessibleBlocker");
	}
	if (pluginStore.degradedMode) {
		return t("plugin.advanced.risk.degradedBlocker");
	}
	if (pluginSignatureType(plugin) === "warning") {
		return t("plugin.advanced.risk.signatureBlocker");
	}
	return t("plugin.advanced.risk.noBlocker");
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

function isDevTaskStatus(item: Record<string, unknown>, statuses: string[]): boolean {
	return statuses.includes(String(item.taskStatus || item.status || "").toLowerCase());
}

function devTaskStatusType(status: unknown): "success" | "warning" | "danger" | "info" {
	const value = String(status || "").toLowerCase();
	if (["success", "ok", "completed"].includes(value)) {
		return "success";
	}
	if (["failed", "error"].includes(value)) {
		return "danger";
	}
	if (["running", "pending", "queued"].includes(value)) {
		return "warning";
	}
	return "info";
}

function canRetryDevTask(row: Record<string, unknown>): boolean {
	return isDevTaskStatus(row, ["failed", "error"]) && String(row.pluginId || "").trim() !== "";
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
		{ label: t("plugin.advanced.asset.entryPath"), value: pluginEntryPath(plugin.id) || "-" },
		{ label: t("plugin.advanced.asset.backendEndpoint"), value: plugin.backendEndpoint || "-" },
		{ label: t("plugin.advanced.asset.serviceHealthURL"), value: plugin.serviceHealthURL || "-" },
		{ label: t("plugin.advanced.asset.appId"), value: plugin.appId || "-" },
		{ label: t("plugin.advanced.asset.uiMode"), value: plugin.uiMode || "frontend" },
		{ label: t("plugin.advanced.asset.uiOpenMode"), value: plugin.uiOpenMode || "-" },
		{ label: t("plugin.advanced.asset.uiTabMode"), value: plugin.uiTabMode || "-" },
		{ label: t("plugin.advanced.asset.locales"), value: plugin.i18nLocales?.join(", ") || "-" }
	];
}

function preflightStatusType(status?: InstallPreflightStatus): "success" | "danger" | "info" {
	if (status === "pass") {
		return "success";
	}
	if (status === "blocked") {
		return "danger";
	}
	return "info";
}

function preflightRiskType(level?: InstallPreflightRiskLevel): "success" | "warning" | "danger" | "info" {
	if (level === "critical" || level === "high") {
		return "danger";
	}
	if (level === "medium") {
		return "warning";
	}
	if (level === "low") {
		return "success";
	}
	return "info";
}

function preflightSignatureType(status?: InstallPreflightResult["signature"]["status"]): "success" | "warning" | "danger" | "info" {
	if (status === "signed") {
		return "success";
	}
	if (status === "incomplete" || status === "unsupported") {
		return "danger";
	}
	if (status === "unsigned") {
		return "warning";
	}
	return "info";
}

function preflightDiffCount<T>(diff?: InstallPreflightDiff<T>): number {
	return (diff?.add?.length ?? 0) + (diff?.update?.length ?? 0) + (diff?.conflict?.length ?? 0);
}

function preflightDependencyText(): string {
	return installPreflight.value?.resources.dependencies?.join(", ") || "-";
}

function preflightMigrationCount(): number {
	return (installPreflight.value?.migration.pending?.length ?? 0) + (installPreflight.value?.migration.applied?.length ?? 0);
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
		configSchema.value = normalizePluginConfigSchema(payload.data?.configSchema) ?? normalizePluginConfigSchema(getPluginRecord(pluginID)?.configSchema) ?? null;
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
		const payload = await apiPost<ApiResponse<InstallPreflightResult>>("/v1/plugins/preflight", {
			path: pluginPath.value.trim()
		});
		installPreflight.value = payload.data ?? null;
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
		error.value = t("plugin.advanced.preflightRequired");
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
			loadMarketplace(),
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
		error.value = t("plugin.advanced.selectPlugin");
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
		captureDevRollbackPlan(payload, action);
		info.value = t("plugin.advanced.devSubmitted");
		await Promise.all([loadDevProjects(), loadDevReleaseOrders(), loadDevReleaseTasks(), loadDevRolloutTasks()]);
	} catch (e) {
		error.value = toErrorMessage(e);
		devResultText.value = "";
		devRollbackPlan.value = null;
	} finally {
		devLoading.value = false;
	}
}

async function confirmDevAction(action: "package" | "pipeline" | "rollout" | "rollback", pluginID: string): Promise<boolean> {
	const labelMap = {
		package: t("plugin.advanced.action.package"),
		pipeline: t("plugin.advanced.action.pipeline"),
		rollout: t("plugin.advanced.action.rollout"),
		rollback: t("plugin.advanced.action.rollback")
	};
	const target = action === "rollout" ? `${pluginID} / ${devTargetEnv.value.trim() || "production"}` : pluginID;
	return confirmAction({
		title: t("plugin.advanced.devConfirmTitle"),
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
		error.value = t("plugin.advanced.scaffoldRequired");
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
		info.value = t("plugin.advanced.scaffoldCreated");
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
		error.value = t("plugin.advanced.selectPlugin");
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
		info.value = t("plugin.advanced.releaseOrderCreated");
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
	const actionText = action === "approve" ? t("plugin.advanced.releaseApprove") : t("plugin.advanced.releaseReject");
	if (!(await confirmAction({
		title: t("plugin.advanced.releaseReviewTitle"),
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
		info.value = action === "approve" ? t("plugin.advanced.releaseApproved") : t("plugin.advanced.releaseRejected");
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
		error.value = t("plugin.advanced.releaseExecuteRequired");
		return;
	}
	if (!(await confirmAction({
		title: t("plugin.advanced.releaseExecuteTitle"),
		message: `${pluginID} -> ${devTargetEnv.value.trim() || "staging"} / ${devArtifactPath.value.trim()}`,
		confirmText: t("plugin.advanced.releaseExecute"),
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
		info.value = t("plugin.advanced.releaseTaskSubmitted");
		await Promise.all([loadDevReleaseOrders(), loadDevReleaseTasks()]);
	} catch (e) {
		error.value = toErrorMessage(e);
		devResultText.value = "";
	} finally {
		devLoading.value = false;
	}
}

async function retryDevTask(kind: "release" | "rollout", row: Record<string, unknown>): Promise<void> {
	const pluginID = String(row.pluginId || "").trim();
	if (!pluginID) {
		error.value = t("plugin.advanced.retryPluginMissing");
		return;
	}
	devSelectedPlugin.value = pluginID;
	if (kind === "release") {
		await runDevAction("pipeline");
		return;
	}
	const action = String(row.action || "").toLowerCase() === "rollback" ? "rollback" : "rollout";
	await runDevAction(action);
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

function captureDevRollbackPlan(payload: unknown, action: "validate" | "package" | "pipeline" | "rollout" | "rollback"): void {
	if (action !== "rollback") {
		devRollbackPlan.value = null;
		return;
	}
	const root = payload as { data?: Record<string, unknown> };
	const rollback = root?.data?.rollback as PluginRollbackPlan | undefined;
	if (rollback && Array.isArray(rollback.checkpoints)) {
		devRollbackPlan.value = rollback;
		return;
	}
	devRollbackPlan.value = null;
}

function rollbackCheckpointType(status?: string): "success" | "warning" | "danger" | "info" {
	if (status === "passed") {
		return "success";
	}
	if (status === "blocked") {
		return "danger";
	}
	if (status === "warning") {
		return "warning";
	}
	return "info";
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
	devTaskDrawerTitle.value = `${t(kind === "release" ? "plugin.advanced.releaseTask" : "plugin.advanced.rolloutTask")} ${taskID}`;
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
	<PageShell :title="t('plugin.title')" :description="t('plugin.desc')">
		<template #actions>
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
		</template>

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
			<MetricStrip :items="summaryItems" />

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
						<strong>{{ installPreflight.plugin.name || installPreflight.plugin.id }}</strong>
						<p>{{ installPreflight.plugin.id }}@{{ installPreflight.plugin.version }} · {{ validatedPluginPath }}</p>
					</div>
					<div class="preflight-tags">
						<el-tag :type="preflightStatusType(installPreflight.status)" effect="light">{{ installPreflight.status === "blocked" ? t("plugin.advanced.preflight.blocked") : t("plugin.advanced.preflight.installable") }}</el-tag>
						<el-tag :type="preflightRiskType(installPreflight.risk.level)" effect="light">{{ installPreflight.risk.level }}</el-tag>
						<el-tag :type="preflightSignatureType(installPreflight.signature.status)" effect="light">{{ installPreflight.signature.status }}</el-tag>
					</div>
				</div>
				<el-alert v-if="installPreflight.blockers?.length" class="page-alert" type="error" :title="t('plugin.advanced.preflight.blockedTitle')" show-icon :closable="false">
					<ul class="preflight-list">
						<li v-for="item in installPreflight.blockers" :key="item">{{ item }}</li>
					</ul>
				</el-alert>
				<el-alert v-if="installPreflight.warnings?.length" class="page-alert" type="warning" :title="t('plugin.advanced.preflight.reviewTitle')" show-icon :closable="false">
					<ul class="preflight-list">
						<li v-for="item in installPreflight.warnings" :key="item">{{ item }}</li>
					</ul>
				</el-alert>
				<el-descriptions :column="3" border>
					<el-descriptions-item :label="t('plugin.advanced.preflight.permissionDiff')">{{ preflightDiffCount(installPreflight.permissions) }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.menuDiff')">{{ preflightDiffCount(installPreflight.menus) }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.configFields')">{{ installPreflight.config.hasSchema ? installPreflight.config.fieldCount : t("plugin.advanced.preflight.noSchema") }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.dependencies')">{{ preflightDependencyText() }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.migrations')">{{ installPreflight.migration.error || t("plugin.advanced.preflight.itemCount", { count: preflightMigrationCount() }) }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.resources')">{{ installPreflight.resources.uiMode || "-" }} · {{ installPreflight.resources.frontendEntry || "-" }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.signature')">{{ installPreflight.signature.algorithm || "-" }} · {{ installPreflight.signature.vendorId || "-" }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.riskSummary')">{{ installPreflight.risk.summary?.join(", ") || t("plugin.advanced.preflight.noAdditionalRisk") }}</el-descriptions-item>
					<el-descriptions-item :label="t('plugin.advanced.preflight.auditTrail')">{{ t("plugin.advanced.detail.auditEvent") }}</el-descriptions-item>
				</el-descriptions>
				<div class="preflight-grid">
					<section class="preflight-section">
						<h4>{{ t("plugin.advanced.preflight.permissionImpact") }}</h4>
						<el-table :data="[...(installPreflight.permissions.conflict ?? []), ...(installPreflight.permissions.add ?? []), ...(installPreflight.permissions.update ?? [])]" size="small" max-height="220" :empty-text="t('plugin.advanced.preflight.noPermissionChanges')">
							<el-table-column prop="key" :label="t('plugin.advanced.preflight.key')" min-width="160" />
							<el-table-column prop="type" :label="t('plugin.advanced.preflight.type')" width="90" />
							<el-table-column prop="risk" :label="t('plugin.advanced.preflight.risk')" width="90">
								<template #default="{ row }">
									<el-tag :type="marketplaceRiskType(row.risk || 'low')" size="small">{{ row.risk || "low" }}</el-tag>
								</template>
							</el-table-column>
							<el-table-column prop="existing" :label="t('plugin.advanced.preflight.existing')" min-width="140" />
						</el-table>
					</section>
					<section class="preflight-section">
						<h4>{{ t("plugin.advanced.preflight.menuImpact") }}</h4>
						<el-table :data="[...(installPreflight.menus.conflict ?? []), ...(installPreflight.menus.add ?? []), ...(installPreflight.menus.update ?? [])]" size="small" max-height="220" :empty-text="t('plugin.advanced.preflight.noMenuChanges')">
							<el-table-column prop="key" :label="t('plugin.advanced.preflight.key')" min-width="150" />
							<el-table-column prop="path" :label="t('plugin.advanced.preflight.path')" min-width="180" />
							<el-table-column prop="existing" :label="t('plugin.advanced.preflight.existing')" min-width="140" />
						</el-table>
					</section>
					<section class="preflight-section">
						<h4>{{ t("plugin.advanced.preflight.configResources") }}</h4>
						<dl>
							<dt>{{ t("plugin.advanced.preflight.requiredFields") }}</dt>
							<dd>{{ installPreflight.config.requiredFields?.join(", ") || "-" }}</dd>
							<dt>{{ t("plugin.advanced.preflight.defaultFields") }}</dt>
							<dd>{{ installPreflight.config.defaultFields?.join(", ") || "-" }}</dd>
							<dt>{{ t("plugin.advanced.preflight.network") }}</dt>
							<dd>{{ installPreflight.resources.serviceBaseUrl || installPreflight.resources.serviceHealthUrl || "-" }}</dd>
						</dl>
					</section>
					<section class="preflight-section">
						<h4>{{ t("plugin.advanced.preflight.migrations") }}</h4>
						<el-table :data="installPreflight.migration.pending ?? []" size="small" max-height="180" :empty-text="t('plugin.advanced.preflight.noMigrations')">
							<el-table-column prop="version" :label="t('plugin.advanced.preflight.version')" width="80" />
							<el-table-column prop="name" :label="t('plugin.advanced.preflight.name')" min-width="180" />
						</el-table>
					</section>
				</div>
			</div>
			<pre v-if="operationText" class="code-block">{{ operationText }}</pre>
		</el-card>

		<div class="heavy-panel-tabs" :aria-label="t('plugin.advanced.panelLabel')">
			<el-button-group>
				<el-button :type="activeHeavyPanel === 'inventory' ? 'primary' : 'default'" @click="activeHeavyPanel = 'inventory'">{{ t("plugin.advanced.panel.inventory") }}</el-button>
				<el-button :type="activeHeavyPanel === 'marketplace' ? 'primary' : 'default'" @click="activeHeavyPanel = 'marketplace'">{{ t("plugin.advanced.panel.marketplace") }}</el-button>
				<el-button :type="activeHeavyPanel === 'risk' ? 'primary' : 'default'" @click="activeHeavyPanel = 'risk'">{{ t("plugin.advanced.panel.risk") }}</el-button>
				<el-button :type="activeHeavyPanel === 'devportal' ? 'primary' : 'default'" :disabled="!canManagePlugins" @click="activeHeavyPanel = 'devportal'">{{ t("plugin.advanced.dev.title") }}</el-button>
			</el-button-group>
		</div>

		<el-card v-if="activeHeavyPanel === 'marketplace'" shadow="never" class="marketplace-panel">
			<template #header>
				<div class="card-header">
					<div>
						<h3>{{ t("plugin.advanced.market.title") }}</h3>
						<p>{{ t("plugin.advanced.market.summary", { root: devPluginsRoot, total: marketplaceItems.length, installable: marketplaceInstallableCount, highRisk: marketplaceHighRiskCount, unsigned: marketplaceUnsignedCount }) }}</p>
					</div>
					<el-button type="primary" :icon="Refresh" :loading="marketplaceLoading" :disabled="!canReadPlugins" @click="loadMarketplace">{{ t("plugin.advanced.market.refresh") }}</el-button>
				</div>
			</template>
			<el-alert v-if="marketplaceError" class="page-alert" type="error" :title="marketplaceError" show-icon :closable="false" />
			<el-form label-position="top" class="marketplace-filters" @submit.prevent>
				<el-form-item :label="t('plugin.advanced.market.keyword')">
					<el-input v-model="marketplaceKeyword" clearable :placeholder="t('plugin.advanced.market.searchPlaceholder')" />
				</el-form-item>
				<el-form-item :label="t('plugin.advanced.market.risk')">
					<el-select v-model="marketplaceRiskFilter" clearable :placeholder="t('plugin.advanced.market.allRisks')">
						<el-option v-for="level in ['low', 'medium', 'high', 'critical', 'unknown']" :key="level" :label="t(`plugin.advanced.level.${level}`)" :value="level" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('plugin.advanced.market.signature')">
					<el-select v-model="marketplaceSignatureFilter" clearable :placeholder="t('plugin.advanced.market.allSignatures')">
						<el-option v-for="status in ['signed', 'unsigned', 'incomplete', 'unknown']" :key="status" :label="t(`plugin.advanced.signature.${status}`)" :value="status" />
					</el-select>
				</el-form-item>
				<el-form-item class="filter-actions">
					<el-button :disabled="!hasMarketplaceFilters" @click="resetMarketplaceFilters">{{ t("common.reset") }}</el-button>
				</el-form-item>
			</el-form>
			<StateBlock v-if="!marketplaceLoading && filteredMarketplaceItems.length === 0" type="empty" :description="t('plugin.advanced.market.empty')" />
			<el-table v-else v-loading="marketplaceLoading" :data="filteredMarketplaceItems" stripe border>
				<el-table-column prop="id" :label="t('plugin.table.id')" min-width="150" show-overflow-tooltip />
				<el-table-column prop="name" :label="t('plugin.table.name')" min-width="160" show-overflow-tooltip />
				<el-table-column prop="version" :label="t('plugin.table.version')" width="110" />
				<el-table-column :label="t('plugin.advanced.market.risk')" min-width="190">
					<template #default="{ row }">
						<div class="signal-stack">
							<el-tag :type="marketplaceRiskType(row.risk.level)" effect="light">{{ row.risk.level }}</el-tag>
							<span class="signal-meta">{{ t("plugin.advanced.preflight.itemCount", { count: marketplaceRiskItems(row).length }) }}</span>
						</div>
					</template>
				</el-table-column>
				<el-table-column :label="t('plugin.advanced.market.signature')" min-width="190">
					<template #default="{ row }">
						<div class="signal-stack">
							<el-tag :type="marketplaceSignatureType(row.signature.status)" effect="plain">{{ row.signature.status }}</el-tag>
							<span class="signal-meta">{{ row.signature.algorithm || "-" }}</span>
						</div>
					</template>
				</el-table-column>
				<el-table-column :label="t('plugin.advanced.market.package')" min-width="260" show-overflow-tooltip>
					<template #default="{ row }">
						<div class="package-cell">
							<strong>{{ row.installable ? t("plugin.advanced.package.ready") : t("plugin.advanced.package.source") }}</strong>
							<span>{{ row.packageDigest || row.manifestPath || "-" }}</span>
						</div>
					</template>
				</el-table-column>
				<el-table-column :label="t('plugin.advanced.preflight.path')" min-width="260" show-overflow-tooltip>
					<template #default="{ row }">
						{{ marketplaceInstallPath(row) || "-" }}
					</template>
				</el-table-column>
				<el-table-column :label="t('common.actions')" width="170" fixed="right">
					<template #default="{ row }">
						<el-button v-if="canManagePlugins" size="small" type="primary" :icon="Upload" :disabled="!marketplaceInstallPath(row) || operating" @click="prepareMarketplaceInstall(row)">
							{{ t("plugin.advanced.market.installEntry") }}
						</el-button>
						<el-tag v-else type="info" effect="plain">{{ t("plugin.advanced.market.readOnly") }}</el-tag>
					</template>
				</el-table-column>
			</el-table>
		</el-card>

		<el-card v-if="activeHeavyPanel === 'risk'" shadow="never" class="risk-report-panel plugin-risk-report">
			<template #header>
				<div class="card-header">
					<div>
						<h3>{{ t("plugin.advanced.riskReport.title") }}</h3>
						<p>{{ t("plugin.advanced.riskReport.description") }}</p>
					</div>
					<div class="risk-report-signals">
						<el-tag :type="riskReportSummary.preInstall > 0 ? 'warning' : 'success'" effect="light">{{ t("plugin.advanced.riskReport.preInstall") }} {{ riskReportSummary.preInstall }}</el-tag>
						<el-tag :type="riskReportSummary.installed > 0 ? 'warning' : 'success'" effect="light">{{ t("plugin.advanced.riskReport.installed") }} {{ riskReportSummary.installed }}</el-tag>
						<el-tag type="info" effect="plain">{{ t("plugin.advanced.riskReport.permissions") }} {{ riskReportSummary.permissions }}</el-tag>
					</div>
				</div>
			</template>
			<el-tabs>
				<el-tab-pane :label="t('plugin.advanced.riskReport.preInstall')">
					<StateBlock v-if="visiblePreInstallRiskRows.length === 0" type="empty" :description="t('plugin.advanced.riskReport.emptyPreInstall')" />
					<el-table v-else :data="visiblePreInstallRiskRows" stripe border>
						<el-table-column prop="id" :label="t('plugin.table.id')" min-width="150" show-overflow-tooltip />
						<el-table-column prop="version" :label="t('plugin.table.version')" width="100" />
						<el-table-column :label="t('plugin.advanced.market.risk')" width="110">
							<template #default="{ row }">
								<el-tag :type="marketplaceRiskType(row.level)" effect="light">{{ row.level }}</el-tag>
							</template>
						</el-table-column>
						<el-table-column :label="t('plugin.advanced.market.signature')" width="120">
							<template #default="{ row }">
								<el-tag :type="marketplaceSignatureType(row.signature)" effect="plain">{{ row.signature }}</el-tag>
							</template>
						</el-table-column>
						<el-table-column :label="t('plugin.advanced.riskReport.permissionImpact')" min-width="180">
							<template #default="{ row }">
								<el-tag v-for="item in row.permissions" :key="item" class="detail-tag" effect="plain">{{ item }}</el-tag>
								<span v-if="row.permissions.length === 0">-</span>
							</template>
						</el-table-column>
						<el-table-column :label="t('plugin.advanced.riskReport.riskFactors')" min-width="220">
							<template #default="{ row }">
								<el-tag v-for="item in row.impacts" :key="item" class="detail-tag" effect="plain">{{ item }}</el-tag>
								<span v-if="row.impacts.length === 0">-</span>
							</template>
						</el-table-column>
						<el-table-column prop="audit" :label="t('plugin.advanced.preflight.auditTrail')" min-width="180" show-overflow-tooltip />
					</el-table>
				</el-tab-pane>
				<el-tab-pane :label="t('plugin.advanced.riskReport.installed')">
					<StateBlock v-if="visibleRiskRows.length === 0" type="empty" :description="t('plugin.advanced.riskReport.emptyInstalled')" />
					<el-table v-else :data="visibleRiskRows" stripe border>
						<el-table-column prop="id" :label="t('plugin.table.id')" min-width="170" show-overflow-tooltip />
						<el-table-column :label="t('plugin.advanced.market.risk')" width="120">
							<template #default="{ row }">
								<el-tag :type="pluginRiskLevelType(row.level)" effect="light">{{ row.level }}</el-tag>
							</template>
						</el-table-column>
						<el-table-column :label="t('plugin.advanced.riskReport.permissionImpact')" min-width="180">
							<template #default="{ row }">
								<el-tag v-for="item in row.permissions" :key="item" class="detail-tag" effect="plain">{{ item }}</el-tag>
								<span v-if="row.permissions.length === 0">-</span>
							</template>
						</el-table-column>
						<el-table-column :label="t('plugin.advanced.riskReport.riskFactors')" min-width="220">
							<template #default="{ row }">
								<el-tag v-for="item in row.factors" :key="item" class="detail-tag" effect="plain">{{ item }}</el-tag>
								<span v-if="row.factors.length === 0">-</span>
							</template>
						</el-table-column>
						<el-table-column prop="blockers" :label="t('plugin.advanced.riskReport.blockers')" min-width="180" show-overflow-tooltip />
						<el-table-column prop="entryPath" :label="t('plugin.advanced.riskReport.entry')" min-width="180" show-overflow-tooltip />
						<el-table-column prop="audit" :label="t('plugin.advanced.preflight.auditTrail')" min-width="200" show-overflow-tooltip />
					</el-table>
				</el-tab-pane>
			</el-tabs>
		</el-card>

		<el-card v-if="activeHeavyPanel === 'devportal'" v-permission="'plugin.manage'" shadow="never" class="devportal-panel">
			<template #header>
				<div class="card-header">
					<div>
						<h3>{{ t("plugin.advanced.dev.title") }}</h3>
						<p>{{ t("plugin.advanced.dev.description") }}</p>
					</div>
					<el-button type="primary" :icon="Refresh" :loading="devLoading" @click="refreshDevPortal">{{ t("plugin.advanced.dev.refresh") }}</el-button>
				</div>
			</template>
			<div class="devportal-grid">
				<section class="devportal-controls">
					<el-form label-position="top">
						<el-form-item :label="t('plugin.advanced.dev.pluginsRoot')">
							<el-input v-model="devPluginsRoot" :placeholder="t('plugin.advanced.dev.pluginsRoot')" clearable />
						</el-form-item>
						<el-divider content-position="left">{{ t("plugin.advanced.dev.scaffold") }}</el-divider>
						<el-form-item :label="t('plugin.advanced.dev.newPluginId')">
							<el-input v-model="devScaffoldPluginID" :placeholder="t('plugin.advanced.dev.newPluginIdPlaceholder')" clearable />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.newPluginName')">
							<el-input v-model="devScaffoldPluginName" :placeholder="t('plugin.advanced.dev.newPluginNamePlaceholder')" clearable />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.appMode')">
							<div class="inline-fields">
								<el-input v-model="devScaffoldAppID" :placeholder="t('plugin.advanced.dev.optionalAppId')" clearable />
								<el-select v-model="devScaffoldMode">
									<el-option :label="t('plugin.advanced.dev.mode.workspace')" value="workspace" />
									<el-option :label="t('plugin.advanced.dev.mode.repository')" value="repository" />
								</el-select>
							</div>
						</el-form-item>
						<el-button :icon="Upload" :loading="devLoading" @click="scaffoldDevPlugin">{{ t("plugin.advanced.dev.createScaffold") }}</el-button>
						<el-divider content-position="left">{{ t("plugin.advanced.dev.buildRelease") }}</el-divider>
						<el-form-item :label="t('plugin.advanced.dev.plugin')">
							<el-select v-model="devSelectedPlugin" filterable clearable :placeholder="t('plugin.advanced.selectPlugin')" @change="refreshDevPortal">
								<el-option v-for="id in devPluginOptions" :key="id" :label="id" :value="id" />
							</el-select>
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.outputDir')">
							<el-input v-model="devOutputDir" :placeholder="t('plugin.advanced.dev.outputDirPlaceholder')" clearable />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.artifactPath')">
							<el-input v-model="devArtifactPath" :placeholder="t('plugin.advanced.dev.artifactPathPlaceholder')" clearable />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.releaseVersion')">
							<el-input v-model="devReleaseVersion" :placeholder="t('plugin.advanced.dev.releaseVersionPlaceholder')" clearable />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.changelog')">
							<el-input v-model="devChangelog" type="textarea" :rows="3" resize="vertical" :placeholder="t('plugin.advanced.dev.changelogPlaceholder')" />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.reviewComment')">
							<el-input v-model="devReviewComment" :placeholder="t('plugin.advanced.dev.reviewCommentPlaceholder')" clearable />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.targetEnv')">
							<el-segmented v-model="devTargetEnv" :options="['staging', 'production']" />
						</el-form-item>
						<el-button type="success" :icon="Document" :loading="devLoading" :disabled="!devSelectedPlugin" @click="createDevReleaseOrder">{{ t("plugin.advanced.dev.createReleaseOrder") }}</el-button>
						<el-divider content-position="left">{{ t("plugin.advanced.dev.rolloutRollback") }}</el-divider>
						<el-form-item :label="t('plugin.advanced.dev.rolloutStrategy')">
							<el-segmented v-model="devRolloutStrategy" :options="['percent', 'tag', 'canary']" />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.rolloutPercent')">
							<el-slider v-model="devRolloutPercent" :disabled="devRolloutStrategy !== 'percent'" :min="0" :max="100" show-input />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.tags')">
							<el-input v-model="devRolloutTags" :disabled="devRolloutStrategy !== 'tag'" :placeholder="t('plugin.advanced.dev.tagsPlaceholder')" clearable />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.canaryVersion')">
							<el-input v-model="devCanaryVersion" :disabled="devRolloutStrategy !== 'canary'" :placeholder="t('plugin.advanced.dev.canaryPlaceholder')" clearable />
						</el-form-item>
						<el-form-item :label="t('plugin.advanced.dev.rollbackPercent')">
							<el-input-number v-model="devRollbackPercent" :min="0" :max="100" controls-position="right" />
						</el-form-item>
					</el-form>
					<div class="devportal-actions">
						<el-button :icon="CircleCheck" :loading="devLoading" @click="runDevAction('validate')">{{ t("plugin.advanced.action.validate") }}</el-button>
						<el-button :icon="Upload" :loading="devLoading" :disabled="!devSelectedPlugin" @click="runDevAction('package')">{{ t("plugin.advanced.action.package") }}</el-button>
						<el-button type="primary" :icon="Tools" :loading="devLoading" :disabled="!devSelectedPlugin" @click="runDevAction('pipeline')">{{ t("plugin.advanced.action.pipeline") }}</el-button>
						<el-button type="warning" :icon="WarningFilled" :loading="devLoading" :disabled="!devSelectedPlugin" @click="runDevAction('rollout')">{{ t("plugin.advanced.action.rollout") }}</el-button>
						<el-button type="danger" :icon="Refresh" :loading="devLoading" :disabled="!devSelectedPlugin" @click="runDevAction('rollback')">{{ t("plugin.advanced.action.rollback") }}</el-button>
					</div>
					<el-descriptions v-if="devConfig" class="dev-config" :column="1" size="small" border>
						<el-descriptions-item :label="t('plugin.advanced.dev.enabled')">{{ String(devConfig.enabled ?? "") }}</el-descriptions-item>
						<el-descriptions-item :label="t('plugin.advanced.dev.defaultRoot')">{{ String(devConfig.defaultRoot ?? "") }}</el-descriptions-item>
						<el-descriptions-item :label="t('plugin.advanced.dev.allowedRoots')">{{ JSON.stringify(devConfig.allowedRoots ?? []) }}</el-descriptions-item>
					</el-descriptions>
				</section>

				<section class="devportal-data">
					<div class="dev-task-summary">
						<article class="dev-task-card">
							<span>{{ t("plugin.advanced.dev.totalTasks") }}</span>
							<strong>{{ devTaskSummary.total }}</strong>
						</article>
						<article class="dev-task-card">
							<span>{{ t("plugin.advanced.dev.running") }}</span>
							<strong>{{ devTaskSummary.running }}</strong>
						</article>
						<article class="dev-task-card" :class="{ 'is-danger': devTaskSummary.failed > 0 }">
							<span>{{ t("plugin.advanced.dev.failed") }}</span>
							<strong>{{ devTaskSummary.failed }}</strong>
						</article>
						<article class="dev-task-card">
							<span>{{ t("plugin.advanced.dev.rollbacks") }}</span>
							<strong>{{ devTaskSummary.rollback }}</strong>
						</article>
					</div>
					<el-tabs>
						<el-tab-pane :label="t('plugin.advanced.dev.projects')">
							<el-table :data="devProjects" stripe border max-height="260">
								<el-table-column prop="pluginId" :label="t('plugin.table.id')" min-width="160" show-overflow-tooltip />
								<el-table-column prop="version" :label="t('plugin.table.version')" width="100" />
								<el-table-column prop="status" :label="t('plugin.advanced.dev.status')" width="100" />
								<el-table-column prop="mode" :label="t('plugin.advanced.dev.mode')" width="110" />
								<el-table-column prop="path" :label="t('plugin.advanced.preflight.path')" min-width="220" show-overflow-tooltip />
							</el-table>
						</el-tab-pane>
						<el-tab-pane :label="t('plugin.advanced.dev.releaseOrders')">
							<el-table :data="devReleaseOrders" stripe border max-height="260">
								<el-table-column prop="orderId" :label="t('plugin.advanced.dev.orderId')" min-width="180" show-overflow-tooltip />
								<el-table-column prop="pluginId" :label="t('plugin.table.id')" min-width="140" />
								<el-table-column prop="releaseVersion" :label="t('plugin.table.version')" width="110" />
								<el-table-column prop="orderStatus" :label="t('plugin.advanced.dev.status')" width="110" />
								<el-table-column prop="createdAt" :label="t('plugin.advanced.dev.createdAt')" min-width="180" show-overflow-tooltip />
								<el-table-column :label="t('common.actions')" min-width="220" fixed="right">
									<template #default="{ row }">
										<div class="row-actions">
											<el-button size="small" type="success" :disabled="devLoading || !canManagePlugins || row.orderStatus !== 'pending'" @click="reviewDevReleaseOrder(row, 'approve')">{{ t("plugin.advanced.dev.approve") }}</el-button>
											<el-button size="small" type="warning" :disabled="devLoading || !canManagePlugins || row.orderStatus !== 'pending'" @click="reviewDevReleaseOrder(row, 'reject')">{{ t("plugin.advanced.dev.reject") }}</el-button>
											<el-button size="small" type="primary" :disabled="devLoading || !canManagePlugins || row.orderStatus !== 'approved' || !devArtifactPath" @click="executeDevReleaseOrder(row)">{{ t("plugin.advanced.dev.execute") }}</el-button>
										</div>
									</template>
								</el-table-column>
							</el-table>
						</el-tab-pane>
						<el-tab-pane :label="t('plugin.advanced.dev.releaseTasks')">
							<el-table :data="devReleaseTasks" stripe border max-height="260">
								<el-table-column prop="taskId" :label="t('plugin.advanced.dev.taskId')" min-width="180" show-overflow-tooltip />
								<el-table-column prop="pluginId" :label="t('plugin.table.id')" min-width="140" />
								<el-table-column prop="targetEnv" :label="t('plugin.advanced.dev.environment')" width="100" />
								<el-table-column :label="t('plugin.advanced.dev.status')" width="120">
									<template #default="{ row }">
										<el-tag :type="devTaskStatusType(row.taskStatus)" effect="light">{{ row.taskStatus || "-" }}</el-tag>
									</template>
								</el-table-column>
								<el-table-column prop="failureReason" :label="t('plugin.advanced.dev.failureReason')" min-width="180" show-overflow-tooltip />
								<el-table-column :label="t('plugin.advanced.dev.view')" min-width="210" fixed="right">
									<template #default="{ row }">
										<div class="row-actions">
											<el-button size="small" @click="openDevTaskDrawer('release', row, 'detail')">{{ t("plugin.advanced.dev.detail") }}</el-button>
											<el-button size="small" @click="openDevTaskDrawer('release', row, 'logs')">{{ t("plugin.advanced.dev.logs") }}</el-button>
											<el-button size="small" type="warning" :disabled="devLoading || !canRetryDevTask(row)" @click="retryDevTask('release', row)">{{ t("plugin.advanced.dev.retry") }}</el-button>
										</div>
									</template>
								</el-table-column>
							</el-table>
						</el-tab-pane>
						<el-tab-pane :label="t('plugin.advanced.dev.rolloutTasks')">
							<el-table :data="devRolloutTasks" stripe border max-height="260">
								<el-table-column prop="taskId" :label="t('plugin.advanced.dev.taskId')" min-width="180" show-overflow-tooltip />
								<el-table-column prop="pluginId" :label="t('plugin.table.id')" min-width="140" />
								<el-table-column prop="action" :label="t('plugin.advanced.dev.action')" width="100" />
								<el-table-column prop="rolloutPercent" :label="t('plugin.advanced.dev.percent')" width="100" />
								<el-table-column :label="t('plugin.advanced.dev.status')" width="120">
									<template #default="{ row }">
										<el-tag :type="devTaskStatusType(row.taskStatus)" effect="light">{{ row.taskStatus || "-" }}</el-tag>
									</template>
								</el-table-column>
								<el-table-column prop="failureReason" :label="t('plugin.advanced.dev.failureReason')" min-width="180" show-overflow-tooltip />
								<el-table-column :label="t('plugin.advanced.dev.view')" min-width="230" fixed="right">
									<template #default="{ row }">
										<div class="row-actions">
											<el-button size="small" @click="openDevTaskDrawer('rollout', row, 'detail')">{{ t("plugin.advanced.dev.detail") }}</el-button>
											<el-button size="small" @click="openDevTaskDrawer('rollout', row, 'logs')">{{ t("plugin.advanced.dev.logs") }}</el-button>
											<el-button size="small" type="warning" :disabled="devLoading || !canRetryDevTask(row)" @click="retryDevTask('rollout', row)">{{ t("plugin.advanced.dev.retry") }}</el-button>
											<el-button size="small" type="danger" :disabled="devLoading || !row.pluginId" @click="retryDevTask('rollout', { ...row, action: 'rollback' })">{{ t("plugin.advanced.action.rollback") }}</el-button>
										</div>
									</template>
								</el-table-column>
							</el-table>
						</el-tab-pane>
						<el-tab-pane :label="t('plugin.advanced.dev.response')">
							<div v-if="devRollbackPlan" class="rollback-plan-panel">
								<div class="rollback-plan-header">
									<div>
										<strong>{{ devRollbackPlan.pluginId }}</strong>
										<span>{{ devRollbackPlan.fromVersion || "-" }} → {{ devRollbackPlan.toVersion || "-" }}</span>
									</div>
									<el-tag :type="rollbackCheckpointType(devRollbackPlan.status)" effect="light">{{ devRollbackPlan.status }}</el-tag>
								</div>
								<el-table :data="devRollbackCheckpoints" size="small" border max-height="220" :empty-text="t('plugin.advanced.dev.noCheckpoints')">
									<el-table-column prop="name" :label="t('plugin.advanced.dev.checkpoint')" width="110" />
									<el-table-column :label="t('plugin.advanced.dev.status')" width="100">
										<template #default="{ row }">
											<el-tag :type="rollbackCheckpointType(row.status)" effect="light">{{ row.status || "-" }}</el-tag>
										</template>
									</el-table-column>
									<el-table-column prop="before" :label="t('plugin.advanced.dev.beforeRollback')" min-width="150" show-overflow-tooltip />
									<el-table-column prop="after" :label="t('plugin.advanced.dev.afterRollback')" min-width="150" show-overflow-tooltip />
									<el-table-column prop="message" :label="t('plugin.advanced.dev.message')" min-width="220" show-overflow-tooltip />
								</el-table>
							</div>
							<pre class="code-block">{{ devResultText || t("plugin.advanced.dev.noResult") }}</pre>
						</el-tab-pane>
					</el-tabs>
				</section>
			</div>
		</el-card>

		<el-drawer v-if="activeHeavyPanel === 'devportal'" v-model="devTaskDrawerOpen" :title="devTaskDrawerTitle" size="46%">
			<el-tabs>
				<el-tab-pane :label="t('plugin.advanced.dev.detail')">
					<pre class="code-block">{{ devTaskDetail ? JSON.stringify(devTaskDetail, null, 2) : t("plugin.advanced.dev.noDetail") }}</pre>
				</el-tab-pane>
				<el-tab-pane :label="t('plugin.advanced.dev.steps')">
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
					<el-empty v-else :description="t('plugin.advanced.dev.noSteps')" />
				</el-tab-pane>
				<el-tab-pane :label="t('plugin.advanced.dev.logs')">
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
					<el-empty v-else :description="t('plugin.advanced.dev.noLogs')" />
				</el-tab-pane>
			</el-tabs>
		</el-drawer>

		<el-drawer v-model="detailDrawerOpen" :title="detailPlugin ? `${detailPlugin.name} / ${detailPlugin.id}` : t('plugin.advanced.detail.title')" size="58%" class="detail-drawer">
			<template v-if="detailPlugin">
				<section class="detail-summary">
					<el-tag :type="pluginStatusType(detailPlugin.id)" effect="light">{{ pluginStatusText(detailPlugin.id) }}</el-tag>
					<el-tag :type="pluginRiskType(detailPlugin)" effect="plain">{{ t("plugin.signal.risk") }}: {{ pluginRiskText(detailPlugin) }}</el-tag>
					<el-tag :type="pluginHealthType(detailPlugin)" effect="plain">{{ t("plugin.signal.health") }}: {{ pluginHealthText(detailPlugin) }}</el-tag>
					<el-tag :type="pluginSignatureType(detailPlugin)" effect="plain">{{ t("plugin.signal.signature") }}: {{ pluginSignatureText(detailPlugin) }}</el-tag>
				</section>
				<el-tabs class="detail-tabs">
					<el-tab-pane :label="t('plugin.advanced.detail.permissions')" name="permissions">
						<div class="detail-section plugin-detail-permissions">
							<el-descriptions :column="2" border>
								<el-descriptions-item :label="t('plugin.advanced.detail.accessStatus')">{{ canVisit(detailPlugin.id) ? t("plugin.access.ready") : t("plugin.access.blocked") }}</el-descriptions-item>
								<el-descriptions-item :label="t('plugin.advanced.detail.managePermission')">{{ canManagePlugins ? t("plugin.advanced.detail.managePermissionValue") : "-" }}</el-descriptions-item>
								<el-descriptions-item :label="t('plugin.advanced.detail.menuPermission')">
									<el-tag v-for="item in pluginRequiredPermissions(detailPlugin)" :key="item" class="detail-tag" effect="plain">{{ item }}</el-tag>
									<span v-if="pluginRequiredPermissions(detailPlugin).length === 0">-</span>
								</el-descriptions-item>
								<el-descriptions-item :label="t('plugin.advanced.detail.requiredRoles')">
									<el-tag v-for="item in pluginRequiredRoles(detailPlugin)" :key="item" class="detail-tag" effect="plain">{{ item }}</el-tag>
									<span v-if="pluginRequiredRoles(detailPlugin).length === 0">-</span>
								</el-descriptions-item>
							</el-descriptions>
						</div>
					</el-tab-pane>
					<el-tab-pane :label="t('plugin.advanced.detail.menu')" name="menu">
						<div class="detail-section plugin-detail-menu">
							<el-descriptions :column="2" border>
								<el-descriptions-item :label="t('plugin.advanced.detail.label')">{{ detailPlugin.uiMenu?.label || detailPlugin.name }}</el-descriptions-item>
								<el-descriptions-item :label="t('plugin.advanced.detail.path')">{{ detailPlugin.uiMenu?.path || pluginEntryPath(detailPlugin.id) || "-" }}</el-descriptions-item>
								<el-descriptions-item :label="t('plugin.advanced.detail.icon')">{{ detailPlugin.uiMenu?.icon || "-" }}</el-descriptions-item>
								<el-descriptions-item :label="t('plugin.advanced.detail.order')">{{ detailPlugin.uiMenu?.order ?? "-" }}</el-descriptions-item>
							</el-descriptions>
						</div>
					</el-tab-pane>
					<el-tab-pane :label="t('plugin.advanced.detail.config')" name="config">
						<div class="detail-section plugin-detail-config">
							<el-descriptions :column="2" border>
								<el-descriptions-item :label="t('plugin.advanced.detail.schema')">{{ pluginConfigFieldCount(detailPlugin) }}</el-descriptions-item>
								<el-descriptions-item :label="t('plugin.advanced.detail.selected')">{{ selectedPlugin === detailPlugin.id ? activeInspectorPanel || "-" : "-" }}</el-descriptions-item>
							</el-descriptions>
							<el-empty v-if="pluginConfigFieldCount(detailPlugin) === 0" :description="t('plugin.advanced.detail.noConfigSchema')" />
							<el-table v-else :data="detailPlugin.configSchema?.fields ?? []" stripe border>
								<el-table-column prop="key" :label="t('plugin.advanced.preflight.key')" min-width="160" />
								<el-table-column prop="type" :label="t('plugin.advanced.preflight.type')" width="120" />
								<el-table-column prop="required" :label="t('plugin.advanced.detail.required')" width="120" />
								<el-table-column prop="help" :label="t('plugin.advanced.detail.help')" min-width="220" show-overflow-tooltip />
							</el-table>
						</div>
					</el-tab-pane>
					<el-tab-pane :label="t('plugin.advanced.detail.assets')" name="assets">
						<div class="detail-section plugin-detail-assets">
							<el-table :data="pluginAssetRows(detailPlugin)" stripe border>
								<el-table-column prop="label" :label="t('plugin.advanced.detail.asset')" width="180" />
								<el-table-column prop="value" :label="t('plugin.advanced.detail.value')" min-width="260" show-overflow-tooltip />
							</el-table>
						</div>
					</el-tab-pane>
					<el-tab-pane :label="t('plugin.advanced.dev.logs')" name="logs">
						<div class="detail-section plugin-detail-logs">
							<el-empty v-if="selectedPlugin !== detailPlugin.id || activeInspectorPanel !== 'logs'" :description="t('plugin.advanced.detail.logsHint')" />
							<pre v-else class="code-block">{{ logText || t("plugin.noLogs") }}</pre>
						</div>
					</el-tab-pane>
					<el-tab-pane :label="t('plugin.advanced.detail.releaseStatus')" name="release">
						<div class="detail-section plugin-detail-release">
							<el-descriptions :column="3" border>
								<el-descriptions-item :label="t('plugin.advanced.dev.releaseOrders')">{{ detailReleaseOrders.length }}</el-descriptions-item>
								<el-descriptions-item :label="t('plugin.advanced.dev.releaseTasks')">{{ detailReleaseTasks.length }}</el-descriptions-item>
								<el-descriptions-item :label="t('plugin.advanced.detail.rolloutRollback')">{{ detailRolloutTasks.length }}</el-descriptions-item>
							</el-descriptions>
							<el-table :data="[...detailReleaseTasks, ...detailRolloutTasks]" stripe border>
								<el-table-column prop="taskId" :label="t('plugin.advanced.dev.taskId')" min-width="180" show-overflow-tooltip />
								<el-table-column prop="action" :label="t('plugin.advanced.dev.action')" width="110" />
								<el-table-column prop="targetEnv" :label="t('plugin.advanced.detail.env')" width="110" />
								<el-table-column prop="taskStatus" :label="t('plugin.advanced.dev.status')" width="120" />
								<el-table-column prop="failureReason" :label="t('plugin.advanced.detail.failure')" min-width="220" show-overflow-tooltip />
							</el-table>
						</div>
					</el-tab-pane>
				</el-tabs>
			</template>
			<el-empty v-else :description="t('plugin.advanced.selectPlugin')" />
		</el-drawer>

		<div v-if="activeHeavyPanel === 'inventory'" class="content-grid">
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
										{{ t("plugin.advanced.dev.detail") }}
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
												{{ t("plugin.advanced.dev.detail") }}
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
								<el-collapse-item :title="t('plugin.advanced.detail.json')" name="json">
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
	</PageShell>
</template>

<style scoped>
.card-header h3,
.table-header h3 {
	margin: 0;
}

.card-header p {
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

.install-panel {
	border: 1px solid var(--color-border);
}

.risk-report-panel {
	border: 1px solid var(--color-border);
}

.risk-report-signals {
	display: flex;
	flex-wrap: wrap;
	justify-content: flex-end;
	gap: 8px;
}

.marketplace-panel {
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

.marketplace-filters {
	display: grid;
	grid-template-columns: minmax(240px, 1fr) minmax(150px, 190px) minmax(150px, 190px) auto;
	gap: 12px;
	align-items: end;
	margin-bottom: 12px;
}

.marketplace-filters :deep(.el-form-item) {
	margin-bottom: 0;
}

.marketplace-filters :deep(.el-select),
.marketplace-filters :deep(.el-input) {
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

.preflight-tags {
	display: flex;
	flex-wrap: wrap;
	justify-content: flex-end;
	gap: 6px;
}

.preflight-list {
	margin: 6px 0 0;
	padding-left: 18px;
}

.preflight-grid {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 12px;
}

.preflight-section {
	min-width: 0;
	border: 1px solid var(--color-border);
	border-radius: 8px;
	padding: 12px;
	background: var(--color-surface);
}

.preflight-section h4 {
	margin: 0 0 10px;
}

.preflight-section dl {
	display: grid;
	grid-template-columns: 88px minmax(0, 1fr);
	gap: 8px 10px;
	margin: 0;
}

.preflight-section dt {
	color: var(--color-text-muted);
}

.preflight-section dd {
	margin: 0;
	word-break: break-all;
}

.heavy-panel-tabs {
	display: flex;
	justify-content: flex-start;
}

.heavy-panel-tabs :deep(.el-button-group) {
	display: flex;
	flex-wrap: wrap;
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

.dev-task-summary {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 10px;
	margin-bottom: 12px;
}

.dev-task-card {
	border: 1px solid var(--color-border);
	border-radius: 8px;
	padding: 10px 12px;
	background: var(--color-surface-soft);
}

.dev-task-card span {
	display: block;
	color: var(--color-text-muted);
	font-size: 0.78rem;
}

.dev-task-card strong {
	display: block;
	margin-top: 4px;
	font-size: 1.45rem;
	line-height: 1;
}

.dev-task-card.is-danger {
	border-color: var(--color-danger);
	background: var(--color-danger-soft);
}

.rollback-plan-panel {
	display: grid;
	gap: 10px;
	margin-bottom: 12px;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: 8px;
	background: var(--color-surface);
}

.rollback-plan-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
}

.rollback-plan-header div {
	display: grid;
	gap: 2px;
}

.rollback-plan-header strong {
	color: var(--color-text);
	font-size: 0.9rem;
}

.rollback-plan-header span {
	color: var(--color-text-muted);
	font-size: 0.78rem;
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

.package-cell {
	display: grid;
	gap: 3px;
	min-width: 0;
}

.package-cell strong {
	font-size: 0.78rem;
	text-transform: uppercase;
	color: var(--color-primary-strong);
}

.package-cell span {
	overflow: hidden;
	color: var(--color-text-muted);
	font-size: 0.78rem;
	text-overflow: ellipsis;
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
	background: var(--color-code-surface);
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
	.plugin-filters {
		grid-template-columns: 1fr 1fr;
	}

	.marketplace-filters {
		grid-template-columns: 1fr 1fr;
	}

	.preflight-grid {
		grid-template-columns: 1fr;
	}

	.content-grid {
		grid-template-columns: 1fr;
	}

	.devportal-grid {
		grid-template-columns: 1fr;
	}

	.dev-task-summary {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	.inspector-card {
		position: static;
	}
}

@media (max-width: 720px) {
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

	.marketplace-filters {
		grid-template-columns: 1fr;
	}

	.header-actions {
		justify-content: flex-start;
	}

}
</style>

