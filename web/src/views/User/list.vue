<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";

import { DataScopeIndicator, StateBlock } from "../../components/Common";
import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import { BUTTON_ACCESS, useButtonAccess } from "../../permissions/button";
import { resolveAuthorizedDataScope, type AuthorizedDataScopeDecision } from "../../permissions/data-scope";
import { type ApiResponse, apiDelete, apiGet, apiPost } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type UserRecord = {
	id: string;
	account: string;
	name?: string;
	email: string;
	status?: string;
	departmentId?: string;
	positionId?: string;
};

type RoleRecord = {
	id: string;
	name: string;
	key?: string;
};

type OrganizationOption = {
	id: string;
	name: string;
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
		status: String(row.status ?? row.Status ?? "").trim(),
		departmentId: String(row.departmentId ?? row.DepartmentID ?? "").trim(),
		positionId: String(row.positionId ?? row.PositionID ?? "").trim()
	};
}

function normalizeRoleRecord(item: unknown): RoleRecord | null {
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
		name: String(row.name ?? row.Name ?? "").trim(),
		key: String(row.key ?? row.Key ?? "").trim()
	};
}

function normalizeOrganizationOption(item: unknown): OrganizationOption | null {
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
		name: String(row.name ?? row.Name ?? id).trim() || id
	};
}

const { t } = useI18n();
const buttonAccess = useButtonAccess();
const route = useRoute();

const loading = ref(false);
const scopeLoading = ref(false);
const error = ref("");
const scopeError = ref("");
const readScope = ref<AuthorizedDataScopeDecision | null>(null);
const operating = ref(false);
const rows = ref<UserRecord[]>([]);
const roles = ref<RoleRecord[]>([]);
const departments = ref<OrganizationOption[]>([]);
const positions = ref<OrganizationOption[]>([]);
const organizationLoadError = ref("");
const selectedRoleID = ref("");
const selectedUserIDs = ref<string[]>([]);
const opSuccess = ref("");
const page = ref(1);
const pageSize = 10;
const canGoNext = ref(false);
const keyword = ref("");
const statusFilter = ref("");
const departmentFilter = ref("");
const positionFilter = ref("");

const hasSelectedRows = computed(() => selectedUserIDs.value.length > 0);
const filteredRows = computed(() => {
	const term = keyword.value.trim().toLowerCase();
	return rows.value.filter((item) => {
		const matchesTerm = term === "" || [item.id, item.account, item.name, item.email].some((value) => String(value ?? "").toLowerCase().includes(term));
		const matchesStatus = statusFilter.value === "" || item.status === statusFilter.value;
		const matchesDepartment = departmentFilter.value === "" || item.departmentId === departmentFilter.value;
		const matchesPosition = positionFilter.value === "" || item.positionId === positionFilter.value;
		return matchesTerm && matchesStatus && matchesDepartment && matchesPosition;
	});
});
const activeUserCount = computed(() => rows.value.filter((item) => item.status === "active").length);
const inactiveUserCount = computed(() => rows.value.filter((item) => item.status && item.status !== "active").length);
const hasActiveFilters = computed(() => keyword.value.trim() !== "" || statusFilter.value !== "" || departmentFilter.value !== "" || positionFilter.value !== "");
const returnTo = computed(() => route.fullPath || "/skoll/user");
const canReadUser = computed(() => buttonAccess.can("user.read"));
const canCreateUser = computed(() => buttonAccess.can(BUTTON_ACCESS.userCreate));
const canUpdateUser = computed(() => buttonAccess.can(BUTTON_ACCESS.userUpdate));
const canDeleteUser = computed(() => buttonAccess.can(BUTTON_ACCESS.userDelete));
const canAssignRole = computed(() => buttonAccess.can(BUTTON_ACCESS.roleManage));

async function loadUsers(targetPage = page.value): Promise<void> {
	if (!canReadUser.value) {
		error.value = t("error.forbidden");
		return;
	}
	loading.value = true;
	scopeLoading.value = true;
	error.value = "";
	scopeError.value = "";
	try {
		const safePage = Math.max(1, targetPage);
		const offset = (safePage - 1) * pageSize;
		const [payload, decision] = await Promise.all([
			apiGet<ApiResponse<UserRecord[]>>(`/v1/users?offset=${offset}&limit=${pageSize}`),
			resolveAuthorizedDataScope("user", "read")
		]);
		readScope.value = decision;
		const list = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeUserRecord(item)).filter((item): item is UserRecord => item !== null)
			: [];
		rows.value = list;
		selectedUserIDs.value = [];
		page.value = safePage;
		canGoNext.value = list.length >= pageSize;
	} catch (e) {
		error.value = toErrorMessage(e);
		scopeError.value = error.value;
		rows.value = [];
		canGoNext.value = false;
	} finally {
		loading.value = false;
		scopeLoading.value = false;
	}
}

async function loadRoles(): Promise<void> {
	try {
		const payload = await apiGet<ApiResponse<RoleRecord[]>>("/v1/roles?offset=0&limit=100");
		roles.value = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeRoleRecord(item)).filter((item): item is RoleRecord => item !== null)
			: [];
		if (!selectedRoleID.value && roles.value.length > 0) {
			selectedRoleID.value = roles.value[0].id;
		}
	} catch (e) {
		error.value = toErrorMessage(e);
	}
}

async function loadOrganizationOptions(): Promise<void> {
	organizationLoadError.value = "";
	try {
		const [deptPayload, positionPayload] = await Promise.all([
			apiGet<ApiResponse<{ items?: unknown[] }>>("/v1/system/departments"),
			apiGet<ApiResponse<{ items?: unknown[] }>>("/v1/system/positions")
		]);
		const deptItems = Array.isArray(deptPayload.data?.items) ? deptPayload.data.items : [];
		const positionItems = Array.isArray(positionPayload.data?.items) ? positionPayload.data.items : [];
		departments.value = deptItems.map((item) => normalizeOrganizationOption(item)).filter((item): item is OrganizationOption => item !== null);
		positions.value = positionItems.map((item) => normalizeOrganizationOption(item)).filter((item): item is OrganizationOption => item !== null);
	} catch (e) {
		organizationLoadError.value = toErrorMessage(e);
		departments.value = [];
		positions.value = [];
	}
}

function labelByID(options: OrganizationOption[], id?: string): string {
	const value = String(id ?? "").trim();
	if (!value) {
		return "-";
	}
	const matched = options.find((item) => item.id === value);
	return matched ? matched.name : value;
}

async function deleteUser(userID: string): Promise<void> {
	if (!canDeleteUser.value) {
		error.value = t("error.forbidden");
		return;
	}
	if (operating.value || loading.value) {
		return;
	}
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: `${t("common.delete")}: ${userID}`,
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

function handleSelectionChange(selection: UserRecord[]): void {
	selectedUserIDs.value = selection.map((item) => item.id);
}

function resetFilters(): void {
	keyword.value = "";
	statusFilter.value = "";
	departmentFilter.value = "";
	positionFilter.value = "";
}

async function bulkAssignRole(): Promise<void> {
	if (!canAssignRole.value) {
		error.value = t("error.forbidden");
		return;
	}
	if (!hasSelectedRows.value || !selectedRoleID.value) {
		return;
	}
	operating.value = true;
	error.value = "";
	opSuccess.value = "";
	let successCount = 0;
	for (const userID of selectedUserIDs.value) {
		try {
			await apiPost<ApiResponse<unknown>>("/v1/rbac/bind", {
				subjectType: "user",
				subjectId: userID,
				roleId: selectedRoleID.value,
				scope: "all"
			});
			successCount += 1;
		} catch {
			// Best-effort mode keeps the batch running when one user fails.
		}
	}
	opSuccess.value = `${t("user.bulkAssignDone")}: ${successCount}/${selectedUserIDs.value.length}`;
	operating.value = false;
}

void loadUsers(1);
void loadRoles();
void loadOrganizationOptions();
</script>

<template>
	<section class="user-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.users") }}</h2>
				<p>{{ t("user.roleAssignHint") }}</p>
			</div>
			<div class="toolbar-actions">
				<el-button v-if="canCreateUser" type="primary" tag="router-link" :to="{ name: 'user-add', query: { returnTo } }">{{ t("user.create") }}</el-button>
				<el-button v-if="canCreateUser" tag="router-link" :to="{ name: 'user-batch-add', query: { returnTo } }">{{ t("user.batchCreate") }}</el-button>
				<el-button :loading="loading" :disabled="operating" @click="loadUsers(page)">{{ t("common.refresh") }}</el-button>
			</div>
		</header>

		<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
		<el-alert v-if="opSuccess" :title="opSuccess" type="success" show-icon :closable="false" />
		<el-alert v-if="organizationLoadError" :title="`${t('user.organizationLoadFailed')}: ${organizationLoadError}`" type="warning" show-icon :closable="false" />

		<StateBlock v-if="!canReadUser" type="forbidden" :title="t('user.noPermissionTitle')" :description="t('error.forbidden')" />

		<DataScopeIndicator v-if="canReadUser" :decision="readScope" :loading="scopeLoading" :error="scopeError" @retry="loadUsers(page)" />

		<section v-if="canReadUser" class="summary-grid">
			<article class="summary-card">
				<span>{{ t("user.summary.currentPage") }}</span>
				<strong>{{ rows.length }}</strong>
				<small>{{ t("user.summary.filtered") }} {{ filteredRows.length }}</small>
			</article>
			<article class="summary-card">
				<span>{{ t("user.summary.active") }}</span>
				<strong>{{ activeUserCount }}</strong>
				<small>{{ t("user.summary.inactive") }} {{ inactiveUserCount }}</small>
			</article>
			<article class="summary-card">
				<span>{{ t("user.summary.selected") }}</span>
				<strong>{{ selectedUserIDs.length }}</strong>
				<small>{{ t("user.summary.roles") }} {{ roles.length }}</small>
			</article>
			<article class="summary-card">
				<span>{{ t("user.summary.organization") }}</span>
				<strong>{{ departments.length + positions.length }}</strong>
				<small>{{ t("table.department") }} {{ departments.length }} · {{ t("table.position") }} {{ positions.length }}</small>
			</article>
		</section>

		<template v-if="canReadUser">
			<el-form class="filters" label-position="top" @submit.prevent="loadUsers(1)">
				<el-form-item :label="t('user.filter.keyword')">
					<el-input v-model="keyword" clearable :placeholder="t('user.filter.keywordPlaceholder')" />
				</el-form-item>
				<el-form-item :label="t('table.status')">
					<el-select v-model="statusFilter" clearable :placeholder="t('user.filter.allStatus')">
						<el-option :label="t('user.status.active')" value="active" />
						<el-option :label="t('user.status.disabled')" value="disabled" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('table.department')">
					<el-select v-model="departmentFilter" clearable filterable :placeholder="t('user.filter.allDepartments')">
						<el-option v-for="item in departments" :key="item.id" :label="item.name" :value="item.id" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('table.position')">
					<el-select v-model="positionFilter" clearable filterable :placeholder="t('user.filter.allPositions')">
						<el-option v-for="item in positions" :key="item.id" :label="item.name" :value="item.id" />
					</el-select>
				</el-form-item>
				<el-form-item class="filter-actions" :label="t('common.actions')">
					<el-button type="primary" native-type="submit" :loading="loading" :disabled="operating">{{ t("common.refresh") }}</el-button>
					<el-button :disabled="!hasActiveFilters" @click="resetFilters">{{ t("common.reset") }}</el-button>
				</el-form-item>
			</el-form>

			<section v-if="canAssignRole" class="batch-bar">
				<div>
					<strong>{{ t("user.bulkAssign") }}</strong>
					<p>{{ t("user.bulkAssignHint") }}</p>
				</div>
				<div class="batch-actions">
					<el-select v-model="selectedRoleID" :disabled="loading || operating || roles.length === 0" class="role-select">
						<el-option v-for="role in roles" :key="role.id" :value="role.id" :label="`${role.name} (${role.key || role.id})`" />
					</el-select>
					<el-button type="primary" :loading="operating" :disabled="loading || !hasSelectedRows || !selectedRoleID" @click="bulkAssignRole">{{ t("user.bulkAssign") }}</el-button>
				</div>
			</section>

			<div class="table-shell">
				<el-table
					v-loading="loading"
					:data="filteredRows"
					border
					row-key="id"
					:empty-text="hasActiveFilters ? t('user.filter.empty') : t('common.empty')"
					@selection-change="handleSelectionChange"
				>
					<el-table-column type="selection" width="48" :selectable="() => !operating" />
					<el-table-column prop="id" :label="t('table.id')" min-width="180" show-overflow-tooltip />
					<el-table-column :label="t('table.name')" min-width="160">
						<template #default="{ row }">
							<div class="user-cell">
								<strong>{{ row.name || row.account }}</strong>
								<small>{{ row.account }}</small>
							</div>
						</template>
					</el-table-column>
					<el-table-column prop="email" :label="t('table.email')" min-width="210" show-overflow-tooltip />
					<el-table-column :label="t('table.department')" min-width="140" show-overflow-tooltip>
						<template #default="{ row }">{{ labelByID(departments, row.departmentId) }}</template>
					</el-table-column>
					<el-table-column :label="t('table.position')" min-width="140" show-overflow-tooltip>
						<template #default="{ row }">{{ labelByID(positions, row.positionId) }}</template>
					</el-table-column>
					<el-table-column :label="t('table.status')" width="120">
						<template #default="{ row }">
							<el-tag :type="row.status === 'active' ? 'success' : 'info'" effect="plain">{{ row.status || "-" }}</el-tag>
						</template>
					</el-table-column>
					<el-table-column :label="t('table.actions')" width="170" fixed="right">
						<template #default="{ row }">
							<el-button v-if="canUpdateUser" link type="primary" tag="router-link" :to="{ name: 'user-edit', params: { id: row.id }, query: { returnTo } }">{{ t("common.edit") }}</el-button>
							<el-button v-if="canDeleteUser" link type="danger" :disabled="operating || loading" @click="deleteUser(row.id)">{{ t("common.delete") }}</el-button>
							<span v-if="!canUpdateUser && !canDeleteUser" class="muted">{{ t("user.noRowActions") }}</span>
						</template>
					</el-table-column>
				</el-table>
			</div>
		</template>

		<div class="pager">
			<el-button :disabled="loading || operating || page <= 1" @click="prevPage">{{ t("common.prev") }}</el-button>
			<span>{{ t("common.page") }} {{ page }}</span>
			<el-button :disabled="loading || operating || !canGoNext" @click="nextPage">{{ t("common.next") }}</el-button>
		</div>
	</section>
</template>

<style scoped>
.user-page {
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

.toolbar-actions {
	display: flex;
	align-items: center;
	gap: 8px;
	flex-wrap: wrap;
	justify-content: flex-end;
}

.summary-grid {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
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
.batch-bar p,
.user-cell small {
	color: var(--color-text-muted);
}

.summary-card strong {
	font-size: 1.35rem;
}

.filters {
	display: grid;
	grid-template-columns: minmax(180px, 1.2fr) repeat(3, minmax(150px, 1fr)) minmax(170px, auto);
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

.filters :deep(.el-select),
.filters :deep(.el-input) {
	width: 100%;
}

.filter-actions :deep(.el-form-item__content) {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}

.batch-bar {
	display: flex;
	align-items: center;
	gap: 8px;
	justify-content: space-between;
	padding: 12px 14px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

.batch-bar p {
	margin: 4px 0 0;
}

.batch-actions {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 8px;
	flex-wrap: wrap;
}

.role-select {
	width: 260px;
}

.table-shell {
	overflow-x: auto;
}

.user-cell {
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

@media (max-width: 860px) {
	.page-header,
	.batch-bar,
	.batch-actions,
	.pager {
		display: grid;
		justify-content: stretch;
	}

	.summary-grid,
	.filters {
		grid-template-columns: 1fr;
	}

	.role-select {
		width: 100%;
	}
}

@media (min-width: 861px) and (max-width: 1180px) {
	.summary-grid {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	.filters {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
}
</style>

