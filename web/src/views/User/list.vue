<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";

import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import { useAccess } from "../../permissions/access";
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
const access = useAccess();
const route = useRoute();

const loading = ref(false);
const error = ref("");
const operating = ref(false);
const rows = ref<UserRecord[]>([]);
const roles = ref<RoleRecord[]>([]);
const departments = ref<OrganizationOption[]>([]);
const positions = ref<OrganizationOption[]>([]);
const selectedRoleID = ref("");
const selectedUserIDs = ref<string[]>([]);
const opSuccess = ref("");
const page = ref(1);
const pageSize = 10;
const canGoNext = ref(false);

const hasSelectedRows = computed(() => selectedUserIDs.value.length > 0);
const returnTo = computed(() => route.fullPath || "/skoll/user");
const canCreateUser = computed(() => access.can("user.create"));
const canUpdateUser = computed(() => access.can("user.update"));
const canDeleteUser = computed(() => access.can("user.delete"));
const canAssignRole = computed(() => access.can("role.manage"));

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
		selectedUserIDs.value = [];
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
	try {
		const [deptPayload, positionPayload] = await Promise.all([
			apiGet<ApiResponse<{ items?: unknown[] }>>("/v1/system/departments"),
			apiGet<ApiResponse<{ items?: unknown[] }>>("/v1/system/positions")
		]);
		const deptItems = Array.isArray(deptPayload.data?.items) ? deptPayload.data.items : [];
		const positionItems = Array.isArray(positionPayload.data?.items) ? positionPayload.data.items : [];
		departments.value = deptItems.map((item) => normalizeOrganizationOption(item)).filter((item): item is OrganizationOption => item !== null);
		positions.value = positionItems.map((item) => normalizeOrganizationOption(item)).filter((item): item is OrganizationOption => item !== null);
	} catch {
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

		<section v-if="canAssignRole" class="batch-bar">
			<el-select v-model="selectedRoleID" :disabled="loading || operating || roles.length === 0" class="role-select">
				<el-option v-for="role in roles" :key="role.id" :value="role.id" :label="`${role.name} (${role.key || role.id})`" />
			</el-select>
			<el-button type="primary" :loading="operating" :disabled="loading || !hasSelectedRows || !selectedRoleID" @click="bulkAssignRole">{{ t("user.bulkAssign") }}</el-button>
		</section>

		<el-table
			v-loading="loading"
			:data="rows"
			border
			row-key="id"
			:empty-text="t('common.empty')"
			@selection-change="handleSelectionChange"
		>
			<el-table-column type="selection" width="48" :selectable="() => !operating" />
			<el-table-column prop="id" :label="t('table.id')" min-width="180" show-overflow-tooltip />
			<el-table-column :label="t('table.name')" min-width="160">
				<template #default="{ row }">{{ row.name || row.account }}</template>
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
					<span v-if="!canUpdateUser && !canDeleteUser" class="muted">{{ t("common.empty") }}</span>
				</template>
			</el-table-column>
		</el-table>

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

.batch-bar {
	display: flex;
	align-items: center;
	gap: 8px;
	justify-content: flex-start;
}

.role-select {
	width: 260px;
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
	.pager {
		display: grid;
		justify-content: stretch;
	}

	.role-select {
		width: 100%;
	}
}
</style>

