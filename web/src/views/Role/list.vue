<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import StateBlock from "../../components/Common/StateBlock.vue";
import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import { BUTTON_ACCESS, useButtonAccess } from "../../permissions/button";
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
const buttonAccess = useButtonAccess();
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
const keyword = ref("");

const filteredRows = computed(() => {
	const term = keyword.value.trim().toLowerCase();
	if (!term) {
		return rows.value;
	}
	return rows.value.filter((item) => [item.id, item.name, item.key].some((value) => String(value ?? "").toLowerCase().includes(term)));
});
const permissionTotal = computed(() => rows.value.reduce((total, item) => total + (Array.isArray(item.permissions) ? item.permissions.length : 0), 0));
const guardedRoleCount = computed(() => rows.value.filter((item) => Array.isArray(item.permissions) && item.permissions.length > 0).length);
const hasActiveFilters = computed(() => keyword.value.trim() !== "");
const canReadRole = computed(() => buttonAccess.can("role.read"));
const canCreateRole = computed(() => buttonAccess.can(BUTTON_ACCESS.roleCreate));
const canUpdateRole = computed(() => buttonAccess.can(BUTTON_ACCESS.roleUpdate));
const canDeleteRole = computed(() => buttonAccess.can(BUTTON_ACCESS.roleDelete));

async function loadRoles(targetPage = page.value): Promise<void> {
	if (!canReadRole.value) {
		error.value = t("error.forbidden");
		return;
	}
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
	if (!canCreateRole.value) {
		createError.value = t("error.forbidden");
		return;
	}
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
	if (!canDeleteRole.value) {
		error.value = t("error.forbidden");
		return;
	}
	if (operating.value || loading.value) {
		return;
	}
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: `${t("common.delete")}: ${roleID}`,
		confirmText: t("common.delete"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (!confirmed) {
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

function resetFilters(): void {
	keyword.value = "";
}

onMounted(() => {
	void loadRoles(1);
});
</script>

<template>
	<section class="role-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.roles") }}</h2>
				<p>{{ t("permission.matrixHint") }}</p>
			</div>
			<el-button :loading="loading" :disabled="operating" @click="loadRoles(page)">{{ t("common.refresh") }}</el-button>
		</header>

		<StateBlock v-if="!canReadRole" type="forbidden" :title="t('role.noPermissionTitle')" :description="t('error.forbidden')" />

		<section v-else class="summary-grid">
			<article class="summary-card">
				<span>{{ t("role.summary.currentPage") }}</span>
				<strong>{{ rows.length }}</strong>
				<small>{{ t("role.summary.filtered") }} {{ filteredRows.length }}</small>
			</article>
			<article class="summary-card">
				<span>{{ t("role.summary.guarded") }}</span>
				<strong>{{ guardedRoleCount }}</strong>
				<small>{{ t("role.summary.permissions") }} {{ permissionTotal }}</small>
			</article>
			<article class="summary-card">
				<span>{{ t("role.summary.actions") }}</span>
				<strong>{{ canUpdateRole ? t("common.edit") : "-" }}</strong>
				<small>{{ canDeleteRole ? t("common.delete") : t("role.noRowActions") }}</small>
			</article>
		</section>

		<section v-if="canReadRole && canCreateRole" class="create-panel">
			<h3>{{ t("role.createTitle") }}</h3>
			<el-form class="create-form" @submit.prevent="createRole">
				<el-input v-model="createName" :placeholder="t('role.createNamePlaceholder')" :disabled="creating" />
				<el-input v-model="createKey" :placeholder="t('role.createKeyPlaceholder')" :disabled="creating" />
				<el-input v-model="createDescription" :placeholder="t('role.createDescPlaceholder')" :disabled="creating" />
				<el-button type="primary" native-type="submit" :loading="creating" :disabled="creating || createName.trim() === '' || createKey.trim() === ''">{{ t("role.create") }}</el-button>
			</el-form>
			<el-alert v-if="createError" :title="createError" type="error" show-icon :closable="false" />
			<el-alert v-if="createSuccess" :title="createSuccess" type="success" show-icon :closable="false" />
		</section>
		<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

		<template v-if="canReadRole">
			<el-form class="filters" label-position="top" @submit.prevent="loadRoles(1)">
				<el-form-item :label="t('role.filter.keyword')">
					<el-input v-model="keyword" clearable :placeholder="t('role.filter.keywordPlaceholder')" />
				</el-form-item>
				<el-form-item class="filter-actions" :label="t('common.actions')">
					<el-button type="primary" native-type="submit" :loading="loading" :disabled="operating">{{ t("common.refresh") }}</el-button>
					<el-button :disabled="!hasActiveFilters" @click="resetFilters">{{ t("common.reset") }}</el-button>
				</el-form-item>
			</el-form>

			<div class="table-shell">
				<el-table v-loading="loading" :data="filteredRows" border row-key="id" :empty-text="hasActiveFilters ? t('role.filter.empty') : t('common.empty')">
					<el-table-column prop="id" :label="t('table.id')" min-width="190" show-overflow-tooltip />
					<el-table-column :label="t('table.name')" min-width="170">
						<template #default="{ row }">
							<div class="role-cell">
								<strong>{{ row.name || row.key || row.id }}</strong>
								<small>{{ row.id }}</small>
							</div>
						</template>
					</el-table-column>
					<el-table-column :label="t('table.key')" min-width="150">
						<template #default="{ row }">
							<el-tag effect="plain">{{ row.key || "-" }}</el-tag>
						</template>
					</el-table-column>
					<el-table-column :label="t('table.permissions')" width="120" align="center">
						<template #default="{ row }">{{ Array.isArray(row.permissions) ? row.permissions.length : 0 }}</template>
					</el-table-column>
					<el-table-column :label="t('table.actions')" width="190" fixed="right">
						<template #default="{ row }">
							<el-button v-if="canUpdateRole" link type="primary" tag="router-link" :to="{ name: 'role-edit', params: { id: row.id } }">{{ t("common.edit") }}</el-button>
							<el-button v-if="canDeleteRole" link type="danger" :disabled="loading || operating" @click="deleteRole(row.id)">{{ t("common.delete") }}</el-button>
							<span v-if="!canUpdateRole && !canDeleteRole" class="muted">{{ t("role.noRowActions") }}</span>
						</template>
					</el-table-column>
				</el-table>
			</div>
		</template>

		<div v-if="canReadRole" class="pager">
			<el-button :disabled="loading || operating || page <= 1" @click="prevPage">{{ t("common.prev") }}</el-button>
			<span>{{ t("common.page") }} {{ page }}</span>
			<el-button :disabled="loading || operating || !canGoNext" @click="nextPage">{{ t("common.next") }}</el-button>
		</div>
	</section>
</template>

<style scoped>
.role-page {
	display: grid;
	gap: 14px;
}

.page-header {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
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

.summary-grid {
	display: grid;
	grid-template-columns: repeat(3, minmax(0, 1fr));
	gap: 12px;
}

.summary-card {
	display: grid;
	gap: 4px;
	padding: 14px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

.summary-card span,
.summary-card small,
.role-cell small {
	color: var(--color-text-muted);
}

.summary-card strong {
	font-size: 1.3rem;
}

.create-panel {
	display: grid;
	gap: 10px;
	padding: 14px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.create-panel h3 {
	margin: 0;
	font-size: 1rem;
}

.create-form {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 8px;
}

.filters {
	display: grid;
	grid-template-columns: minmax(220px, 1fr) minmax(170px, auto);
	gap: 12px;
	align-items: end;
	padding: 14px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.filters :deep(.el-form-item) {
	margin-bottom: 0;
}

.filter-actions :deep(.el-form-item__content) {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}

.table-shell {
	overflow-x: auto;
}

.role-cell {
	display: grid;
	gap: 2px;
}

.pager {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 8px;
}

.muted {
	color: var(--color-text-muted);
}

@media (max-width: 900px) {
	.page-header,
	.filters,
	.pager {
		display: grid;
		justify-content: stretch;
	}

	.summary-grid,
	.create-form {
		grid-template-columns: 1fr;
	}
}

</style>

