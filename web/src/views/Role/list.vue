<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import { useI18n } from "../../i18n";
import { type ApiResponse, apiDelete, apiGet, apiPost } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type RoleRecord = {
	id: string;
	name: string;
	key?: string;
	permissions?: string[];
};

function normalizeRoleRecord(item: unknown): RoleRecord | null {
	if (!item || typeof item !== "object") {
		return null;
	}
	const row = item as Record<string, unknown>;
	const id = String(row.id ?? row.ID ?? "").trim();
	if (!id) {
		return null;
	}
	const permissionsRaw = row.permissions ?? row.Permissions;
	return {
		id,
		name: String(row.name ?? row.Name ?? "").trim(),
		key: String(row.key ?? row.Key ?? "").trim(),
		permissions: Array.isArray(permissionsRaw)
			? permissionsRaw.filter((item): item is string => typeof item === "string")
			: []
	};
}

const { t } = useI18n();
const loading = ref(false);
const operating = ref(false);
const error = ref("");
const rows = ref<RoleRecord[]>([]);
const page = ref(1);
const pageSize = 10;
const canGoNext = ref(false);
const creating = ref(false);
const createError = ref("");
const createSuccess = ref("");
const createName = ref("");
const createKey = ref("");
const createDescription = ref("");

const hasRows = computed(() => rows.value.length > 0);

async function loadRoles(targetPage = page.value): Promise<void> {
	loading.value = true;
	error.value = "";
	try {
		const safePage = Math.max(1, targetPage);
		const offset = (safePage - 1) * pageSize;
		const payload = await apiGet<ApiResponse<RoleRecord[]>>(`/v1/roles?offset=${offset}&limit=${pageSize}`);
		const list = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeRoleRecord(item)).filter((item): item is RoleRecord => item !== null)
			: [];
		rows.value = list;
		page.value = safePage;
		canGoNext.value = list.length >= pageSize;
	} catch (e) {
		error.value = toErrorMessage(e);
		rows.value = [];
		canGoNext.value = false;
	} finally {
		loading.value = false;
	}
}

function prevPage(): void {
	if (loading.value || page.value <= 1) {
		return;
	}
	void loadRoles(page.value - 1);
}

function nextPage(): void {
	if (loading.value || !canGoNext.value) {
		return;
	}
	void loadRoles(page.value + 1);
}

async function createRole(): Promise<void> {
	createError.value = "";
	createSuccess.value = "";
	creating.value = true;
	try {
		await apiPost<ApiResponse<RoleRecord>>("/v1/roles", {
			name: createName.value.trim(),
			key: createKey.value.trim(),
			description: createDescription.value.trim(),
			permissions: [],
			builtIn: false
		});
		createSuccess.value = t("role.createDone");
		createName.value = "";
		createKey.value = "";
		createDescription.value = "";
		await loadRoles(1);
	} catch (e) {
		createError.value = toErrorMessage(e);
	} finally {
		creating.value = false;
	}
}

async function deleteRole(roleID: string): Promise<void> {
	if (operating.value || loading.value) {
		return;
	}
	operating.value = true;
	error.value = "";
	try {
		await apiDelete<ApiResponse<unknown>>(`/v1/roles/${roleID}`);
		await loadRoles(page.value);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		operating.value = false;
	}
}

onMounted(() => {
	void loadRoles(1);
});
</script>

<template>
	<section>
		<h2>{{ t("page.roles") }}</h2>
		<section class="create-panel">
			<h3>{{ t("role.createTitle") }}</h3>
			<form class="create-form" @submit.prevent="createRole">
				<input v-model="createName" type="text" :placeholder="t('role.createNamePlaceholder')" required :disabled="creating" />
				<input v-model="createKey" type="text" :placeholder="t('role.createKeyPlaceholder')" required :disabled="creating" />
				<input v-model="createDescription" type="text" :placeholder="t('role.createDescPlaceholder')" :disabled="creating" />
				<button type="submit" :disabled="creating">{{ creating ? t("common.loading") : t("role.create") }}</button>
			</form>
			<p v-if="createError" class="error">{{ createError }}</p>
			<p v-if="createSuccess" class="success">{{ createSuccess }}</p>
		</section>
		<p v-if="error" class="error">{{ error }}</p>
		<div class="toolbar">
			<button type="button" :disabled="loading || operating" @click="loadRoles(page)">{{ loading ? t("common.loading") : t("common.refresh") }}</button>
			<div class="pager">
				<button type="button" :disabled="loading || operating || page <= 1" @click="prevPage">{{ t("common.prev") }}</button>
				<span>{{ t("common.page") }} {{ page }}</span>
				<button type="button" :disabled="loading || operating || !canGoNext" @click="nextPage">{{ t("common.next") }}</button>
			</div>
		</div>
		<table>
			<thead>
				<tr>
					<th>{{ t("table.id") }}</th>
					<th>{{ t("table.name") }}</th>
					<th>{{ t("table.key") }}</th>
					<th>{{ t("table.permissions") }}</th>
					<th>{{ t("table.actions") }}</th>
				</tr>
			</thead>
			<tbody v-if="hasRows">
				<tr v-for="item in rows" :key="item.id">
					<td>{{ item.id }}</td>
					<td>{{ item.name }}</td>
					<td>{{ item.key || "-" }}</td>
					<td>{{ Array.isArray(item.permissions) ? item.permissions.length : 0 }}</td>
					<td class="actions">
						<router-link :to="{ name: 'role-edit', params: { id: item.id } }">{{ t("common.edit") }}</router-link>
						<button type="button" :disabled="loading || operating" @click="deleteRole(item.id)">{{ t("common.delete") }}</button>
					</td>
				</tr>
			</tbody>
			<tbody v-else>
				<tr>
					<td colspan="5">{{ loading ? t("common.loading") : t("common.empty") }}</td>
				</tr>
			</tbody>
		</table>
	</section>
</template>

<style scoped>
.toolbar {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin: 8px 0 12px;
	gap: 8px;
}

.create-panel {
	margin: 12px 0;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.create-panel h3 {
	margin: 0 0 8px;
}

.create-form {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 8px;
}

.create-form input {
	height: 32px;
	padding: 0 8px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.pager {
	display: flex;
	align-items: center;
	gap: 8px;
}

.actions {
	display: flex;
	gap: 8px;
	align-items: center;
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

button {
	height: 32px;
	padding: 0 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
	cursor: pointer;
}

button:disabled {
	opacity: 0.6;
	cursor: default;
}

.error {
	color: var(--color-danger);
	margin: 4px 0;
}

.success {
	color: var(--color-success);
	margin: 4px 0;
}
</style>

