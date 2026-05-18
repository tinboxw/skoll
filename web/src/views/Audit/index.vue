<script setup lang="ts">
import { computed, ref } from "vue";

import { useI18n } from "../../i18n";
import { type ApiResponse, apiDelete, apiGet } from "../../utils/api";
import { getToken } from "../../utils/auth";
import { toErrorMessage } from "../../utils/common";
import { API_BASE_PREFIX } from "../../utils/api-base-prefix";

type AuditRecord = {
	id: string;
	actorId: string;
	action: string;
	resource: string;
	resourceId?: string;
	occurredAt: string;
};

function normalizeAuditRecord(item: unknown): AuditRecord | null {
	if (!item || typeof item !== "object") {
		return null;
	}
	const row = item as Record<string, unknown>;
	const id = String(row.id ?? row.ID ?? "").trim();
	if (!id) {
		return null;
	}
	return {
		id,
		actorId: String(row.actorId ?? row.ActorID ?? "").trim(),
		action: String(row.action ?? row.Action ?? "").trim(),
		resource: String(row.resource ?? row.Resource ?? "").trim(),
		resourceId: String(row.resourceId ?? row.ResourceID ?? "").trim(),
		occurredAt: String(row.occurredAt ?? row.OccurredAt ?? "").trim()
	};
}

const { t } = useI18n();

function toDateTimeLocalInput(value: Date): string {
	const adjusted = new Date(value.getTime() - value.getTimezoneOffset() * 60000);
	return adjusted.toISOString().slice(0, 16);
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

const loading = ref(false);
const operating = ref(false);
const error = ref("");
const success = ref("");
const actorId = ref("");
const from = ref(initialRange.from);
const to = ref(initialRange.to);
const limit = ref(50);
const page = ref(1);
const pageSize = ref(10);
const items = ref<AuditRecord[]>([]);
const selected = ref<AuditRecord | null>(null);

const hasRows = computed(() => items.value.length > 0);
const totalPages = computed(() => Math.max(1, Math.ceil(items.value.length / pageSize.value)));
const pagedItems = computed(() => {
	const start = (page.value - 1) * pageSize.value;
	return items.value.slice(start, start + pageSize.value);
});
const quickActors = computed(() => {
	const actorSet = new Set<string>();
	for (const item of items.value) {
		const actor = item.actorId.trim();
		if (actor !== "") {
			actorSet.add(actor);
		}
		if (actorSet.size >= 8) {
			break;
		}
	}
	return Array.from(actorSet);
});

function buildQuery(): string {
	const params = new URLSearchParams();
	if (actorId.value.trim() !== "") {
		params.set("actorId", actorId.value.trim());
	}
	if (from.value.trim() !== "") {
		params.set("from", new Date(from.value).toISOString());
	}
	if (to.value.trim() !== "") {
		params.set("to", new Date(to.value).toISOString());
	}
	params.set("limit", String(Math.max(1, limit.value || 50)));
	const query = params.toString();
	return query === "" ? "" : `?${query}`;
}

async function loadAuditLogs(): Promise<void> {
	loading.value = true;
	error.value = "";
	success.value = "";
	try {
		const payload = await apiGet<ApiResponse<AuditRecord[]>>(`/v1/audit${buildQuery()}`);
		items.value = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeAuditRecord(item)).filter((item): item is AuditRecord => item !== null)
			: [];
		page.value = 1;
		selected.value = items.value.length > 0 ? items.value[0] : null;
	} catch (e) {
		error.value = toErrorMessage(e);
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
	actorId.value = actor;
	void loadAuditLogs();
}

function clearQuickActor(): void {
	actorId.value = "";
	void loadAuditLogs();
}

function openDetail(item: AuditRecord): void {
	selected.value = item;
}

function closeDetail(): void {
	selected.value = null;
}

async function exportCSV(): Promise<void> {
	operating.value = true;
	error.value = "";
	success.value = "";
	try {
		const token = getToken().trim();
		const authValue = token.toLowerCase().startsWith("bearer ") ? token : `Bearer ${token}`;
		const resp = await fetch(`${API_BASE_PREFIX}/v1/audit/export${buildQuery()}`, {
			method: "GET",
			headers: {
				Authorization: authValue
			}
		});
		if (!resp.ok) {
			throw new Error(`export failed: ${resp.status}`);
		}
		const blob = await resp.blob();
		const url = URL.createObjectURL(blob);
		const a = document.createElement("a");
		a.href = url;
		a.download = "audit_logs.csv";
		document.body.appendChild(a);
		a.click();
		a.remove();
		URL.revokeObjectURL(url);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		operating.value = false;
	}
}

async function clearByRange(): Promise<void> {
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
	<section>
		<h2>{{ t("page.audit") }}</h2>
		<p>{{ t("audit.desc") }}</p>
		<p v-if="error" class="error">{{ error }}</p>
		<p v-if="success" class="success">{{ success }}</p>

		<div class="filters">
			<label>
				<span>{{ t("audit.actorId") }}</span>
				<input v-model="actorId" type="text" :disabled="loading || operating" />
			</label>
			<label>
				<span>{{ t("audit.from") }}</span>
				<input v-model="from" type="datetime-local" :disabled="loading || operating" />
			</label>
			<label>
				<span>{{ t("audit.to") }}</span>
				<input v-model="to" type="datetime-local" :disabled="loading || operating" />
			</label>
			<label>
				<span>{{ t("audit.limit") }}</span>
				<input v-model.number="limit" type="number" min="1" max="500" :disabled="loading || operating" />
			</label>
		</div>

		<div class="toolbar">
			<button type="button" :disabled="loading || operating" @click="loadAuditLogs">{{ loading ? t("common.loading") : t("common.refresh") }}</button>
			<button type="button" :disabled="loading || operating" @click="exportCSV">{{ t("audit.export") }}</button>
			<button type="button" :disabled="loading || operating" @click="clearByRange">{{ t("audit.clear") }}</button>
			<button v-if="actorId.trim() !== ''" type="button" :disabled="loading || operating" @click="clearQuickActor">{{ t("audit.clearActor") }}</button>
		</div>

		<div v-if="quickActors.length > 0" class="quick-actors">
			<span>{{ t("audit.quickActors") }}</span>
			<button
				v-for="actor in quickActors"
				:key="actor"
				type="button"
				class="chip"
				:class="{ active: actorId === actor }"
				:disabled="loading || operating"
				@click="setQuickActor(actor)"
			>
				{{ actor }}
			</button>
		</div>

		<table>
			<thead>
				<tr>
					<th>{{ t("table.id") }}</th>
					<th>{{ t("table.actor") }}</th>
					<th>{{ t("table.action") }}</th>
					<th>{{ t("table.resource") }}</th>
					<th>{{ t("table.occurredAt") }}</th>
				</tr>
			</thead>
			<tbody v-if="hasRows">
				<tr
					v-for="item in pagedItems"
					:key="item.id"
					:class="{ active: selected?.id === item.id }"
					@click="openDetail(item)"
				>
					<td>{{ item.id }}</td>
					<td>{{ item.actorId || "-" }}</td>
					<td>{{ item.action || "-" }}</td>
					<td>{{ item.resource || "-" }}</td>
					<td>{{ item.occurredAt || "-" }}</td>
				</tr>
			</tbody>
			<tbody v-else>
				<tr>
					<td colspan="5">{{ loading ? t("common.loading") : t("common.empty") }}</td>
				</tr>
			</tbody>
		</table>

		<div class="pager">
			<button type="button" :disabled="page <= 1" @click="prevPage">{{ t("common.prev") }}</button>
			<span>{{ t("common.page") }} {{ page }} / {{ totalPages }}</span>
			<button type="button" :disabled="page >= totalPages" @click="nextPage">{{ t("common.next") }}</button>
			<select v-model.number="pageSize">
				<option :value="10">10</option>
				<option :value="20">20</option>
				<option :value="50">50</option>
			</select>
		</div>

		<div v-if="selected" class="drawer-mask" @click.self="closeDetail">
			<aside class="drawer">
				<header>
					<h3>{{ t("audit.detail") }}</h3>
					<button type="button" @click="closeDetail">{{ t("common.close") }}</button>
				</header>
				<pre>{{ JSON.stringify(selected, null, 2) }}</pre>
			</aside>
		</div>
	</section>
</template>

<style scoped>
p {
	color: var(--color-text-muted);
}

.filters {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 8px;
	margin: 8px 0;
}

.filters label {
	display: grid;
	gap: 6px;
}

.toolbar {
	display: flex;
	gap: 8px;
	margin-bottom: 10px;
}

.quick-actors {
	display: flex;
	align-items: center;
	gap: 6px;
	margin-bottom: 8px;
	flex-wrap: wrap;
}

.quick-actors span {
	font-size: 0.88rem;
	color: var(--color-text-muted);
}

input,
button {
	height: 32px;
	padding: 0 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

button {
	cursor: pointer;
}

.chip {
	height: 28px;
	padding: 0 8px;
	font-size: 0.8rem;
}

.chip.active {
	background: var(--color-primary);
	color: var(--color-on-primary);
	border-color: transparent;
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

tbody tr {
	cursor: pointer;
}

tbody tr.active {
	background: var(--color-surface-soft);
}

.detail {
	margin-top: 12px;
}

.pager {
	display: flex;
	align-items: center;
	gap: 8px;
	margin-top: 10px;
}

.pager select {
	height: 32px;
	padding: 0 8px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

.drawer-mask {
	position: fixed;
	inset: 0;
	background: rgba(9, 13, 20, 0.35);
	display: flex;
	justify-content: flex-end;
	z-index: 60;
}

.drawer {
	width: min(560px, 92vw);
	height: 100%;
	background: var(--color-surface);
	border-left: 1px solid var(--color-border);
	padding: 12px;
	box-sizing: border-box;
	overflow: auto;
}

.drawer header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 8px;
}

.drawer h3 {
	margin: 0;
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

.error {
	color: var(--color-danger);
}

.success {
	color: var(--color-success);
}

@media (max-width: 960px) {
	.filters {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	.pager {
		flex-wrap: wrap;
	}
}
</style>
