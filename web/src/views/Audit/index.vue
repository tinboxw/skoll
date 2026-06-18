<script setup lang="ts">
import { computed, ref } from "vue";

import { exportAuditEvents, getAuditEvent, listAuditEvents, type AuditEvent, type AuditEventDetail, type AuditEventListQuery, type AuditEventRisk, type AuditEventType } from "../../audit/api";
import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import { ApiError, type ApiResponse, apiDelete } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type AuditRecord = AuditEvent;
type AuditTypeFilter = "all" | AuditEventType;

const AUDIT_TYPE_TABS: Array<{ value: AuditTypeFilter; labelKey: string }> = [
	{ value: "all", labelKey: "audit.type.all" },
	{ value: "operation", labelKey: "audit.type.operation" },
	{ value: "login", labelKey: "audit.type.login" },
	{ value: "error", labelKey: "audit.type.error" },
	{ value: "plugin", labelKey: "audit.type.plugin" },
	{ value: "security", labelKey: "audit.type.security" }
];

const AUDIT_RISK_OPTIONS: Array<{ value: AuditEventRisk; labelKey: string }> = [
	{ value: "low", labelKey: "audit.risk.low" },
	{ value: "medium", labelKey: "audit.risk.medium" },
	{ value: "high", labelKey: "audit.risk.high" },
	{ value: "critical", labelKey: "audit.risk.critical" }
];

const { t } = useI18n();

const ACTION_HISTORY_KEY = "skoll.audit.action.history";
const RESOURCE_HISTORY_KEY = "skoll.audit.resource.history";

function readHistory(key: string): string[] {
	if (typeof window === "undefined") {
		return [];
	}
	try {
		const raw = window.localStorage.getItem(key);
		if (!raw) {
			return [];
		}
		const parsed = JSON.parse(raw);
		if (!Array.isArray(parsed)) {
			return [];
		}
		return parsed
			.map((item) => String(item ?? "").trim())
			.filter((item) => item !== "")
			.slice(0, 12);
	} catch {
		return [];
	}
}

function writeHistory(key: string, values: string[]): void {
	if (typeof window === "undefined") {
		return;
	}
	const normalized = values
		.map((item) => item.trim())
		.filter((item) => item !== "")
		.slice(0, 12);
	window.localStorage.setItem(key, JSON.stringify(normalized));
}

function mergeHistory(existing: string[], incoming: string[]): string[] {
	const merged = [...incoming, ...existing]
		.map((item) => item.trim())
		.filter((item) => item !== "");
	const set = new Set<string>();
	const out: string[] = [];
	for (const item of merged) {
		if (set.has(item)) {
			continue;
		}
		set.add(item);
		out.push(item);
		if (out.length >= 12) {
			break;
		}
	}
	return out;
}

function readInitialQuery(): URLSearchParams {
	if (typeof window === "undefined") {
		return new URLSearchParams();
	}
	return new URLSearchParams(window.location.search);
}

function readQueryValue(params: URLSearchParams, key: string): string {
	return (params.get(key) ?? "").trim();
}

function normalizeAuditTypeFilter(value: string): AuditTypeFilter {
	return value === "operation" || value === "login" || value === "error" || value === "plugin" || value === "security"
		? value
		: "all";
}

function toDateTimeLocalInput(value: Date): string {
	const adjusted = new Date(value.getTime() - value.getTimezoneOffset() * 60000);
	return adjusted.toISOString().slice(0, 16);
}

function formatOccurredAtLocal(value: string): string {
	const raw = value.trim();
	if (raw === "") {
		return "";
	}
	const parsed = new Date(raw);
	if (Number.isNaN(parsed.getTime())) {
		return raw;
	}
	return parsed.toLocaleString();
}

function buildDetailPayload(item: AuditRecord | null): Record<string, unknown> | null {
	if (!item) {
		return null;
	}
	return {
		...item,
		occurredAtLocal: formatOccurredAtLocal(item.occurredAt)
	};
}

function displayActor(item: AuditRecord): string {
	const name = (item.actor.name ?? "").trim();
	if (name !== "") {
		return name;
	}
	return item.actor.id.trim() || "-";
}

function displayResource(item: AuditRecord): string {
	const name = (item.resource.name ?? "").trim();
	if (name !== "") {
		return name;
	}
	return item.resource.type.trim() || item.resource.id.trim() || "-";
}

function defaultTimeRange(): { from: string; to: string } {
	const now = new Date();
	const monthAgo = new Date(now);
	monthAgo.setMonth(monthAgo.getMonth() - 1);
	return {
		from: toDateTimeLocalInput(monthAgo),
		to: toDateTimeLocalInput(now)
	};
}

const initialRange = defaultTimeRange();
const initialQuery = readInitialQuery();
const initialActorName = readQueryValue(initialQuery, "actorName");
const initialAction = readQueryValue(initialQuery, "action");
const initialResource = readQueryValue(initialQuery, "resource");
const initialResourceId = readQueryValue(initialQuery, "resourceId");
const initialType = normalizeAuditTypeFilter(readQueryValue(initialQuery, "type"));
const initialRisk = readQueryValue(initialQuery, "risk") as AuditEventRisk | "";
const initialFrom = readQueryValue(initialQuery, "fromLocal");
const initialTo = readQueryValue(initialQuery, "toLocal");
const hasExplicitRangeInQuery = initialFrom !== "" || initialTo !== "";
const autoRangeByDefault = !hasExplicitRangeInQuery;

const loading = ref(false);
const operating = ref(false);
const error = ref("");
const success = ref("");
const forbidden = ref(false);
const actorName = ref(initialActorName);
const action = ref(initialAction);
const resource = ref(initialResource);
const resourceId = ref(initialResourceId);
const selectedType = ref<AuditTypeFilter>(initialType);
const risk = ref<AuditEventRisk | "">(AUDIT_RISK_OPTIONS.some((item) => item.value === initialRisk) ? initialRisk : "");
const from = ref(initialFrom || initialRange.from);
const to = ref(initialTo || initialRange.to);
const limit = ref(Number.parseInt(readQueryValue(initialQuery, "limit") || "50", 10) || 50);
const page = ref(1);
const pageSize = ref(Number.parseInt(readQueryValue(initialQuery, "pageSize") || "10", 10) || 10);
const items = ref<AuditRecord[]>([]);
const selected = ref<AuditRecord | null>(null);
const selectedDetail = ref<AuditEventDetail | null>(null);
const detailLoading = ref(false);
const autoRangeEnabled = ref(autoRangeByDefault);
const actionHistory = ref<string[]>(readHistory(ACTION_HISTORY_KEY));
const resourceHistory = ref<string[]>(readHistory(RESOURCE_HISTORY_KEY));

const hasRows = computed(() => items.value.length > 0);
const totalPages = computed(() => Math.max(1, Math.ceil(items.value.length / pageSize.value)));
const pagedItems = computed(() => {
	const start = (page.value - 1) * pageSize.value;
	return items.value.slice(start, start + pageSize.value);
});
const quickActors = computed(() => {
	const actorSet = new Set<string>();
	for (const item of items.value) {
		const actor = displayActor(item);
		if (actor !== "") {
			actorSet.add(actor);
		}
		if (actorSet.size >= 8) {
			break;
		}
	}
	return Array.from(actorSet);
});
function syncQueryToURL(): void {
	if (typeof window === "undefined") {
		return;
	}
	const params = new URLSearchParams();
	if (selectedType.value !== "all") {
		params.set("type", selectedType.value);
	}
	if (actorName.value.trim() !== "") {
		params.set("actorName", actorName.value.trim());
	}
	if (action.value.trim() !== "") {
		params.set("action", action.value.trim());
	}
	if (resource.value.trim() !== "") {
		params.set("resource", resource.value.trim());
	}
	if (resourceId.value.trim() !== "") {
		params.set("resourceId", resourceId.value.trim());
	}
	if (risk.value !== "") {
		params.set("risk", risk.value);
	}
	if (!autoRangeEnabled.value && from.value.trim() !== "") {
		params.set("fromLocal", from.value.trim());
	}
	if (!autoRangeEnabled.value && to.value.trim() !== "") {
		params.set("toLocal", to.value.trim());
	}
	params.set("limit", String(Math.max(1, limit.value || 50)));
	params.set("pageSize", String(Math.max(1, pageSize.value || 10)));
	const query = params.toString();
	const target = query === "" ? window.location.pathname : `${window.location.pathname}?${query}`;
	window.history.replaceState({}, "", target);
}

function refreshAutoRange(): void {
	if (!autoRangeEnabled.value) {
		return;
	}
	const range = defaultTimeRange();
	from.value = range.from;
	to.value = range.to;
}

function handleFromInput(): void {
	autoRangeEnabled.value = false;
}

function handleToInput(): void {
	autoRangeEnabled.value = false;
}

function updateSuggestionHistory(): void {
	const nextAction = mergeHistory(actionHistory.value, [...quickActions.value, action.value]);
	const nextResource = mergeHistory(resourceHistory.value, [...quickResources.value, resource.value]);
	actionHistory.value = nextAction;
	resourceHistory.value = nextResource;
	writeHistory(ACTION_HISTORY_KEY, nextAction);
	writeHistory(RESOURCE_HISTORY_KEY, nextResource);
}
const quickActions = computed(() => {
	const actionSet = new Set<string>();
	for (const item of items.value) {
		const value = item.action.trim();
		if (value !== "") {
			actionSet.add(value);
		}
		if (actionSet.size >= 8) {
			break;
		}
	}
	return Array.from(actionSet);
});
const quickResources = computed(() => {
	const resourceSet = new Set<string>();
	for (const item of items.value) {
		const value = item.resource.type.trim();
		if (value !== "") {
			resourceSet.add(value);
		}
		if (resourceSet.size >= 8) {
			break;
		}
	}
	return Array.from(resourceSet);
});
const actionSuggestions = computed(() => mergeHistory(actionHistory.value, quickActions.value));
const resourceSuggestions = computed(() => mergeHistory(resourceHistory.value, quickResources.value));
const detailPayload = computed(() => buildDetailPayload(selectedDetail.value ?? selected.value));

function handleTypeChange(): void {
	void loadAuditLogs();
}

function buildQuery(): string {
	const params = new URLSearchParams();
	if (selectedType.value !== "all") {
		params.set("type", selectedType.value);
	}
	if (actorName.value.trim() !== "") {
		params.set("actorId", actorName.value.trim());
	}
	if (action.value.trim() !== "") {
		params.set("action", action.value.trim());
	}
	if (resource.value.trim() !== "") {
		params.set("resourceType", resource.value.trim());
	}
	if (resourceId.value.trim() !== "") {
		params.set("resourceId", resourceId.value.trim());
	}
	if (risk.value !== "") {
		params.set("risk", risk.value);
	}
	if (from.value.trim() !== "") {
		params.set("from", new Date(from.value).toISOString());
	}
	if (to.value.trim() !== "") {
		const end = new Date(to.value);
		end.setSeconds(59, 999);
		params.set("to", end.toISOString());
	}
	params.set("limit", String(Math.max(1, limit.value || 50)));
	const query = params.toString();
	return query === "" ? "" : `?${query}`;
}

function buildAuditEventQuery(): AuditEventListQuery {
	const query: AuditEventListQuery = {
		actorId: actorName.value.trim(),
		action: action.value.trim(),
		resourceType: resource.value.trim(),
		resourceId: resourceId.value.trim(),
		risk: risk.value || undefined,
		limit: Math.max(1, limit.value || 50)
	};
	if (selectedType.value !== "all") {
		query.type = selectedType.value;
	}
	if (from.value.trim() !== "") {
		query.from = new Date(from.value).toISOString();
	}
	if (to.value.trim() !== "") {
		const end = new Date(to.value);
		end.setSeconds(59, 999);
		query.to = end.toISOString();
	}
	return query;
}

async function loadAuditLogs(): Promise<void> {
	loading.value = true;
	error.value = "";
	success.value = "";
	refreshAutoRange();
	syncQueryToURL();
	try {
		const payload = await listAuditEvents(buildAuditEventQuery());
		items.value = payload.items.filter((item) => item.id.trim() !== "");
		forbidden.value = false;
		page.value = 1;
		selected.value = null;
		updateSuggestionHistory();
	} catch (e) {
		error.value = toErrorMessage(e);
		forbidden.value = e instanceof ApiError && e.status === 403;
		items.value = [];
		selected.value = null;
	} finally {
		loading.value = false;
	}
}

function prevPage(): void {
	if (page.value <= 1) {
		return;
	}
	page.value -= 1;
}

function nextPage(): void {
	if (page.value >= totalPages.value) {
		return;
	}
	page.value += 1;
}

function setQuickActor(actor: string): void {
	actorName.value = actor;
	void loadAuditLogs();
}

function setQuickAction(value: string): void {
	action.value = value;
	void loadAuditLogs();
}

function setQuickResource(value: string): void {
	resource.value = value;
	void loadAuditLogs();
}

function clearQuickActor(): void {
	actorName.value = "";
	void loadAuditLogs();
}

function clearActionFilter(): void {
	action.value = "";
	void loadAuditLogs();
}

function clearResourceFilter(): void {
	resource.value = "";
	resourceId.value = "";
	void loadAuditLogs();
}

async function openDetail(item: AuditRecord): Promise<void> {
	selected.value = item;
	selectedDetail.value = null;
	detailLoading.value = true;
	try {
		selectedDetail.value = await getAuditEvent(item.id);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		detailLoading.value = false;
	}
}

function closeDetail(): void {
	selected.value = null;
	selectedDetail.value = null;
	detailLoading.value = false;
}

async function exportCSV(): Promise<void> {
	operating.value = true;
	error.value = "";
	success.value = "";
	try {
		const blob = await exportAuditEvents(buildAuditEventQuery());
		const url = URL.createObjectURL(blob);
		const a = document.createElement("a");
		a.href = url;
		a.download = "audit_events.csv";
		document.body.appendChild(a);
		a.click();
		a.remove();
		URL.revokeObjectURL(url);
		success.value = t("audit.exported");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		operating.value = false;
	}
}

async function clearByRange(): Promise<void> {
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: `${t("audit.clear")}: ${from.value || "-"} - ${to.value || "-"}`,
		confirmText: t("audit.clear"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (!confirmed) {
		return;
	}
	operating.value = true;
	error.value = "";
	success.value = "";
	try {
		const payload = await apiDelete<ApiResponse<{ deleted?: number }>>(`/v1/audit${buildQuery()}`);
		success.value = `${t("audit.deleted")}: ${payload.data?.deleted ?? 0}`;
		await loadAuditLogs();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		operating.value = false;
	}
}

void loadAuditLogs();
</script>

<template>
	<section class="audit-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.audit") }}</h2>
				<p>{{ t("audit.desc") }}</p>
			</div>
			<div class="header-actions">
				<el-button :loading="loading" :disabled="operating" @click="loadAuditLogs">{{ t("common.refresh") }}</el-button>
				<el-button :loading="operating" :disabled="loading" @click="exportCSV">{{ t("audit.export") }}</el-button>
				<el-button type="danger" plain :loading="operating" :disabled="loading" @click="clearByRange">{{ t("audit.clear") }}</el-button>
			</div>
		</header>

		<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
		<el-alert v-if="success" :title="success" type="success" show-icon :closable="false" />

		<section class="panel">
			<el-tabs v-model="selectedType" class="audit-tabs" @tab-change="handleTypeChange">
				<el-tab-pane
					v-for="tab in AUDIT_TYPE_TABS"
					:key="tab.value"
					:name="tab.value"
					:label="t(tab.labelKey)"
				/>
			</el-tabs>

			<el-form label-position="top" class="filters" @submit.prevent="loadAuditLogs">
				<el-form-item :label="t('audit.actorId')">
					<el-input v-model="actorName" clearable :disabled="loading || operating" />
				</el-form-item>
				<el-form-item :label="t('audit.action')">
					<el-select v-model="action" filterable clearable allow-create default-first-option :disabled="loading || operating">
						<el-option v-for="value in actionSuggestions" :key="`action-option-${value}`" :value="value" :label="value" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('audit.resource')">
					<el-select v-model="resource" filterable clearable allow-create default-first-option :disabled="loading || operating">
						<el-option v-for="value in resourceSuggestions" :key="`resource-option-${value}`" :value="value" :label="value" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('audit.resourceId')">
					<el-input v-model="resourceId" clearable :disabled="loading || operating" />
				</el-form-item>
				<el-form-item :label="t('audit.risk')">
					<el-select v-model="risk" clearable :disabled="loading || operating">
						<el-option v-for="item in AUDIT_RISK_OPTIONS" :key="item.value" :value="item.value" :label="t(item.labelKey)" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('audit.from')">
					<el-date-picker
						v-model="from"
						type="datetime"
						value-format="YYYY-MM-DDTHH:mm"
						format="YYYY-MM-DD HH:mm"
						:disabled="loading || operating"
						@change="handleFromInput"
					/>
				</el-form-item>
				<el-form-item :label="t('audit.to')">
					<el-date-picker
						v-model="to"
						type="datetime"
						value-format="YYYY-MM-DDTHH:mm"
						format="YYYY-MM-DD HH:mm"
						:disabled="loading || operating"
						@change="handleToInput"
					/>
				</el-form-item>
				<el-form-item :label="t('audit.limit')">
					<el-input-number v-model="limit" :min="1" :max="500" :disabled="loading || operating" controls-position="right" />
				</el-form-item>
				<div class="filter-actions">
					<el-button type="primary" native-type="submit" :loading="loading" :disabled="operating">{{ t("common.refresh") }}</el-button>
					<el-button v-if="actorName.trim() !== ''" :disabled="loading || operating" @click="clearQuickActor">{{ t("audit.clearActor") }}</el-button>
					<el-button v-if="action.trim() !== ''" :disabled="loading || operating" @click="clearActionFilter">{{ t("audit.clearAction") }}</el-button>
					<el-button v-if="resource.trim() !== '' || resourceId.trim() !== ''" :disabled="loading || operating" @click="clearResourceFilter">{{ t("audit.clearResource") }}</el-button>
				</div>
			</el-form>

			<div v-if="quickActors.length > 0" class="quick-row">
				<span>{{ t("audit.quickActors") }}</span>
				<el-tag
					v-for="actor in quickActors"
					:key="actor"
					:type="actorName === actor ? 'primary' : 'info'"
					effect="plain"
					class="quick-tag"
					@click="setQuickActor(actor)"
				>
					{{ actor }}
				</el-tag>
			</div>

			<div v-if="quickActions.length > 0" class="quick-row">
				<span>{{ t("audit.quickActions") }}</span>
				<el-tag
					v-for="value in quickActions"
					:key="`action-${value}`"
					:type="action === value ? 'primary' : 'info'"
					effect="plain"
					class="quick-tag"
					@click="setQuickAction(value)"
				>
					{{ value }}
				</el-tag>
			</div>

			<div v-if="quickResources.length > 0" class="quick-row">
				<span>{{ t("audit.quickResources") }}</span>
				<el-tag
					v-for="value in quickResources"
					:key="`resource-${value}`"
					:type="resource === value ? 'primary' : 'info'"
					effect="plain"
					class="quick-tag"
					@click="setQuickResource(value)"
				>
					{{ value }}
				</el-tag>
			</div>
		</section>

		<section class="panel">
			<el-result
				v-if="forbidden"
				icon="warning"
				:title="t('audit.forbiddenTitle')"
				:sub-title="t('audit.forbiddenDesc')"
			/>
			<el-empty
				v-else-if="!loading && !hasRows"
				:description="t('audit.empty')"
			/>
			<el-table
				v-else
				v-loading="loading"
				:data="pagedItems"
				border
				row-key="id"
				highlight-current-row
				:empty-text="t('common.empty')"
				@row-click="openDetail"
			>
				<el-table-column prop="id" :label="t('table.id')" min-width="190" show-overflow-tooltip />
				<el-table-column :label="t('table.actor')" min-width="160">
					<template #default="{ row }">{{ displayActor(row) }}</template>
				</el-table-column>
				<el-table-column :label="t('table.action')" min-width="150">
					<template #default="{ row }">
						<el-tag effect="plain">{{ row.action || "-" }}</el-tag>
					</template>
				</el-table-column>
				<el-table-column :label="t('table.resource')" min-width="160">
					<template #default="{ row }">{{ displayResource(row) }}</template>
				</el-table-column>
				<el-table-column :label="t('table.occurredAt')" min-width="190">
					<template #default="{ row }">{{ formatOccurredAtLocal(row.occurredAt) || "-" }}</template>
				</el-table-column>
			</el-table>

			<div v-if="hasRows" class="pager">
				<el-pagination
					v-model:current-page="page"
					v-model:page-size="pageSize"
					background
					layout="prev, pager, next, sizes, total"
					:page-sizes="[10, 20, 50]"
					:total="items.length"
				/>
				<span class="page-total">{{ t("common.page") }} {{ page }} / {{ totalPages }}</span>
			</div>
		</section>

		<el-drawer :model-value="selected !== null" :title="t('audit.detail')" size="min(560px, 92vw)" @close="closeDetail">
			<el-skeleton v-if="detailLoading" :rows="8" animated />
			<div v-else-if="detailPayload" class="detail-drawer">
				<el-descriptions :column="1" border>
					<el-descriptions-item :label="t('table.id')">{{ detailPayload.id }}</el-descriptions-item>
					<el-descriptions-item :label="t('audit.type')">{{ detailPayload.type }}</el-descriptions-item>
					<el-descriptions-item :label="t('table.actor')">{{ displayActor(detailPayload as AuditRecord) }}</el-descriptions-item>
					<el-descriptions-item :label="t('table.action')">{{ detailPayload.action }}</el-descriptions-item>
					<el-descriptions-item :label="t('table.resource')">{{ displayResource(detailPayload as AuditRecord) }}</el-descriptions-item>
					<el-descriptions-item :label="t('audit.risk')">{{ detailPayload.risk }}</el-descriptions-item>
					<el-descriptions-item :label="t('table.occurredAt')">{{ detailPayload.occurredAtLocal }}</el-descriptions-item>
				</el-descriptions>

				<section class="detail-section">
					<h3>{{ t("audit.trace") }}</h3>
					<pre>{{ JSON.stringify(detailPayload.trace ?? {}, null, 2) }}</pre>
				</section>

				<section class="detail-section">
					<h3>{{ t("audit.metadata") }}</h3>
					<pre>{{ JSON.stringify(detailPayload.metadata ?? {}, null, 2) }}</pre>
				</section>

				<section class="detail-section">
					<h3>{{ t("audit.diff") }}</h3>
					<pre>{{ JSON.stringify(detailPayload.diff ?? {}, null, 2) }}</pre>
				</section>

				<section class="detail-section">
					<h3>{{ t("audit.sourceData") }}</h3>
					<pre>{{ JSON.stringify(detailPayload.sourceData ?? {}, null, 2) }}</pre>
				</section>
			</div>
		</el-drawer>
	</section>
</template>

<style scoped>
.audit-page {
	display: grid;
	gap: 14px;
}

.page-header {
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
	flex-wrap: wrap;
	justify-content: flex-end;
}

.panel {
	display: grid;
	gap: 12px;
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.filters {
	display: grid;
	grid-template-columns: repeat(3, minmax(0, 1fr));
	gap: 4px 14px;
}

.filters :deep(.el-date-editor),
.filters :deep(.el-select),
.filters :deep(.el-input-number) {
	width: 100%;
}

.filter-actions {
	grid-column: 1 / -1;
	display: flex;
	align-items: center;
	gap: 8px;
	flex-wrap: wrap;
}

.quick-row {
	display: flex;
	align-items: center;
	gap: 6px;
	flex-wrap: wrap;
}

.quick-row span {
	color: var(--color-text-muted);
	font-size: 0.88rem;
}

.quick-tag {
	cursor: pointer;
}

.pager {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
	flex-wrap: wrap;
}

.page-total {
	color: var(--color-text-muted);
}

.detail-drawer {
	display: grid;
	gap: 14px;
}

.detail-section {
	display: grid;
	gap: 8px;
}

.detail-section h3 {
	margin: 0;
	font-size: 0.95rem;
	font-weight: 700;
}

pre {
	margin: 0;
	padding: 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
	white-space: pre-wrap;
	word-break: break-word;
}

@media (max-width: 960px) {
	.page-header,
	.header-actions,
	.filters {
		display: grid;
		justify-content: stretch;
	}

	.filter-actions {
		grid-column: auto;
	}
}
</style>
