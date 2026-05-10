<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";

import { useI18n } from "../../i18n";
import { type ApiResponse, apiDelete, apiGet } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type UserRecord = {
	id: string;
	account: string;
	name?: string;
	email: string;
	status?: string;
};

function normalizeUserRecord(item: unknown): UserRecord | null {
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
		account: String(row.account ?? row.Account ?? "").trim(),
		name: String(row.name ?? row.Name ?? "").trim(),
		email: String(row.email ?? row.Email ?? "").trim(),
		status: String(row.status ?? row.Status ?? "").trim()
	};
}

const { t } = useI18n();
const route = useRoute();

const loading = ref(false);
const error = ref("");
const operating = ref(false);
const rows = ref<UserRecord[]>([]);
const page = ref(1);
const pageSize = 10;
const canGoNext = ref(false);

const hasRows = computed(() => rows.value.length > 0);
const returnTo = computed(() => route.fullPath || "/user");

async function loadUsers(targetPage = page.value): Promise<void> {
	loading.value = true;
	error.value = "";
	try {
		const safePage = Math.max(1, targetPage);
		const offset = (safePage - 1) * pageSize;
		const payload = await apiGet<ApiResponse<UserRecord[]>>(`/v1/users?offset=${offset}&limit=${pageSize}`);
		const list = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeUserRecord(item)).filter((item): item is UserRecord => item !== null)
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

async function deleteUser(userID: string): Promise<void> {
	if (operating.value || loading.value) {
		return;
	}
	operating.value = true;
	error.value = "";
	try {
		await apiDelete<ApiResponse<unknown>>(`/v1/users/${userID}`);
		await loadUsers(page.value);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		operating.value = false;
	}
}

function prevPage(): void {
	if (loading.value || page.value <= 1) {
		return;
	}
	void loadUsers(page.value - 1);
}

function nextPage(): void {
	if (loading.value || !canGoNext.value) {
		return;
	}
	void loadUsers(page.value + 1);
}

void loadUsers(1);
</script>

<template>
	<section>
		<h2>{{ t("page.users") }}</h2>
		<p v-if="error" class="error">{{ error }}</p>
		<div class="toolbar">
			<div class="toolbar-actions">
				<router-link class="button-link" :to="{ path: '/user/add', query: { returnTo } }">{{ t("user.create") }}</router-link>
				<router-link class="button-link" :to="{ path: '/user/batch-add', query: { returnTo } }">{{ t("user.batchCreate") }}</router-link>
				<button type="button" :disabled="loading || operating" @click="loadUsers(page)">{{ loading ? t("common.loading") : t("common.refresh") }}</button>
			</div>
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
					<th>{{ t("table.email") }}</th>
					<th>{{ t("table.status") }}</th>
					<th>{{ t("table.actions") }}</th>
				</tr>
			</thead>
			<tbody v-if="hasRows">
				<tr v-for="item in rows" :key="item.id">
					<td>{{ item.id }}</td>
					<td>{{ item.name || item.account }}</td>
					<td>{{ item.email }}</td>
					<td>{{ item.status || "-" }}</td>
					<td class="actions">
						<router-link :to="{ path: `/user/${item.id}/edit`, query: { returnTo } }">{{ t("common.edit") }}</router-link>
						<button type="button" :disabled="operating || loading" @click="deleteUser(item.id)">{{ t("common.delete") }}</button>
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

.toolbar-actions {
	display: flex;
	gap: 8px;
	align-items: center;
}

.pager {
	display: flex;
	align-items: center;
	gap: 8px;
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

.button-link {
	display: inline-flex;
	height: 32px;
	padding: 0 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
	align-items: center;
	text-decoration: none;
	color: inherit;
}

.actions {
	display: flex;
	gap: 8px;
	align-items: center;
}

button:disabled {
	opacity: 0.6;
	cursor: default;
}

.error {
	color: var(--color-danger);
	margin: 4px 0;
}
</style>

