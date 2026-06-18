<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";

import { useI18n } from "../../i18n";
import { BUTTON_ACCESS, useButtonAccess } from "../../permissions/button";
import { BASE_PERMISSION_OPTIONS } from "../../permissions/catalog";
import { type ApiResponse, apiGet, apiPost, apiPut } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type RoleRecord = {
	id: string;
	name: string;
	key?: string;
	permissions?: string[];
};

type UserRecord = {
	id: string;
	account: string;
	name?: string;
	email?: string;
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
		email: String(row.email ?? row.Email ?? "").trim()
	};
}

const route = useRoute();
const { t } = useI18n();
const buttonAccess = useButtonAccess();

const permissionOptions = BASE_PERMISSION_OPTIONS;

const roleId = computed(() => String(route.params.id ?? ""));
const roleName = ref("");
const roleKey = ref("");
const roleDescription = ref("");
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const success = ref("");
const selectedPermission = ref(permissionOptions[0] ?? "");
const permissions = ref<string[]>([]);
const users = ref<UserRecord[]>([]);
const usersLoading = ref(false);
const usersError = ref("");
const canUpdateRole = computed(() => buttonAccess.can(BUTTON_ACCESS.roleUpdate));
const canManagePermissions = computed(() => buttonAccess.can({ permissions: [BUTTON_ACCESS.roleManage, BUTTON_ACCESS.permissionManage], mode: "any" }));
const canUpdateUser = computed(() => buttonAccess.can(BUTTON_ACCESS.userUpdate));

const availablePermissionOptions = computed(() => permissionOptions.filter((item) => !permissions.value.includes(item)));

async function loadRole(): Promise<void> {
	if (!roleId.value) {
		error.value = "missing role id";
		return;
	}
	loading.value = true;
	error.value = "";
	try {
		const payload = await apiGet<ApiResponse<RoleRecord>>(`/v1/roles/${roleId.value}`);
		const target = normalizeRoleRecord(payload.data);
		if (!target) {
			throw new Error("role not found");
		}
		roleName.value = target.name || target.key || roleId.value;
		roleKey.value = target.key || "";
		roleDescription.value = String((payload.data as Record<string, unknown>)?.description ?? (payload.data as Record<string, unknown>)?.Description ?? "").trim();
		permissions.value = Array.isArray(target.permissions) ? [...target.permissions] : [];
		if (!availablePermissionOptions.value.includes(selectedPermission.value)) {
			selectedPermission.value = availablePermissionOptions.value[0] ?? "";
		}
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

async function saveRole(): Promise<void> {
	if (!canUpdateRole.value) {
		error.value = t("error.forbidden");
		return;
	}
	if (!roleId.value) {
		return;
	}
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		await apiPut<ApiResponse<RoleRecord>>(`/v1/roles/${roleId.value}`, {
			name: roleName.value.trim(),
			key: roleKey.value.trim(),
			description: roleDescription.value.trim(),
			permissions: permissions.value
		});
		success.value = t("role.updateDone");
		await loadRole();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function loadRoleUsers(): Promise<void> {
	if (!roleId.value) {
		return;
	}
	usersLoading.value = true;
	usersError.value = "";
	try {
		const payload = await apiGet<ApiResponse<UserRecord[]>>(`/v1/roles/${roleId.value}/users`);
		users.value = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeUserRecord(item)).filter((item): item is UserRecord => item !== null)
			: [];
	} catch (e) {
		usersError.value = toErrorMessage(e);
		users.value = [];
	} finally {
		usersLoading.value = false;
	}
}

async function grantPermission(): Promise<void> {
	if (!canManagePermissions.value) {
		error.value = t("error.forbidden");
		return;
	}
	const permission = selectedPermission.value.trim().toLowerCase();
	if (!roleId.value || !permission) {
		return;
	}
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		await apiPost<ApiResponse<unknown>>(`/v1/roles/${roleId.value}/grant`, { permission });
		if (!permissions.value.includes(permission)) {
			permissions.value.push(permission);
		}
		selectedPermission.value = availablePermissionOptions.value[0] ?? "";
		success.value = t("role.permissionGranted");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function revokePermission(permission: string): Promise<void> {
	if (!canManagePermissions.value) {
		error.value = t("error.forbidden");
		return;
	}
	if (!roleId.value || !permission.trim()) {
		return;
	}
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		await apiPost<ApiResponse<unknown>>(`/v1/roles/${roleId.value}/revoke`, { permission });
		permissions.value = permissions.value.filter((item) => item !== permission);
		if (!selectedPermission.value) {
			selectedPermission.value = availablePermissionOptions.value[0] ?? "";
		}
		success.value = t("role.permissionRevoked");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

onMounted(() => {
	void loadRole();
	void loadRoleUsers();
});
</script>

<template>
	<section class="role-edit-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.roleEdit") }}</h2>
				<p>{{ roleId }}</p>
			</div>
			<el-button :loading="loading" :disabled="saving" @click="loadRole">{{ t("common.refresh") }}</el-button>
		</header>

		<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
		<el-alert v-if="success" :title="success" type="success" show-icon :closable="false" />

		<section class="panel">
			<h3>{{ t("table.description") }}</h3>
			<el-form label-position="top" class="edit-form" @submit.prevent="saveRole">
				<el-form-item :label="t('table.name')" required>
					<el-input v-model="roleName" :disabled="loading || saving || !canUpdateRole" />
				</el-form-item>
				<el-form-item :label="t('table.key')" required>
					<el-input v-model="roleKey" :disabled="loading || saving || !canUpdateRole" />
				</el-form-item>
				<el-form-item :label="t('table.description')">
					<el-input v-model="roleDescription" type="textarea" :rows="2" :disabled="loading || saving || !canUpdateRole" />
				</el-form-item>
				<div class="form-actions">
					<el-button type="primary" native-type="submit" :loading="saving" :disabled="loading || !canUpdateRole">
						{{ t("common.save") }}
					</el-button>
				</div>
			</el-form>
		</section>

		<section class="panel">
			<div class="section-header">
				<div>
					<h3>{{ t("table.permissions") }}</h3>
					<p>{{ t("permission.matrixHint") }}</p>
				</div>
				<el-form v-if="canManagePermissions" class="permission-actions" @submit.prevent="grantPermission">
					<el-select v-model="selectedPermission" filterable :disabled="loading || saving || availablePermissionOptions.length === 0">
						<el-option v-for="item in availablePermissionOptions" :key="item" :value="item" :label="item" />
					</el-select>
					<el-button type="primary" native-type="submit" :loading="saving" :disabled="loading || !selectedPermission.trim()">
						{{ t("role.addPermission") }}
					</el-button>
				</el-form>
			</div>
			<el-table v-loading="loading" :data="permissions" border :empty-text="t('common.empty')">
				<el-table-column :label="t('table.permissions')" min-width="220">
					<template #default="{ row }">
						<el-tag effect="plain">{{ row }}</el-tag>
					</template>
				</el-table-column>
				<el-table-column v-if="canManagePermissions" :label="t('table.actions')" width="150" fixed="right">
					<template #default="{ row }">
						<el-button link type="danger" :disabled="saving" @click="revokePermission(row)">{{ t("role.revokePermission") }}</el-button>
					</template>
				</el-table-column>
			</el-table>
		</section>

		<section class="panel">
			<div class="section-header">
				<div>
					<h3>{{ t("role.usersTitle") }}</h3>
					<p>{{ t("user.roleAssignHint") }}</p>
				</div>
				<el-button :loading="usersLoading" :disabled="saving" @click="loadRoleUsers">{{ t("common.refresh") }}</el-button>
			</div>
			<el-alert v-if="usersError" :title="usersError" type="error" show-icon :closable="false" />
			<el-table v-loading="usersLoading" :data="users" border row-key="id" :empty-text="t('common.empty')">
				<el-table-column prop="id" :label="t('table.id')" min-width="190" show-overflow-tooltip />
				<el-table-column :label="t('table.name')" min-width="160">
					<template #default="{ row }">{{ row.name || row.account || "-" }}</template>
				</el-table-column>
				<el-table-column prop="email" :label="t('table.email')" min-width="210" show-overflow-tooltip />
				<el-table-column :label="t('table.actions')" width="120" fixed="right">
					<template #default="{ row }">
						<el-button v-if="canUpdateUser" link type="primary" tag="router-link" :to="{ name: 'user-edit', params: { id: row.id } }">
							{{ t("common.edit") }}
						</el-button>
						<span v-else class="muted">{{ t("common.empty") }}</span>
					</template>
				</el-table-column>
			</el-table>
		</section>
	</section>
</template>

<style scoped>
.role-edit-page {
	display: grid;
	gap: 14px;
}

.page-header,
.section-header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 16px;
}

.page-header h2 {
	margin: 0;
	font-size: 1.35rem;
}

.page-header p,
.section-header p {
	margin: 6px 0 0;
	color: var(--color-text-muted);
}

.panel {
	display: grid;
	gap: 12px;
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.panel h3 {
	margin: 0;
	font-size: 1rem;
}

.edit-form {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 4px 14px;
}

.edit-form :deep(.el-form-item:nth-child(3)) {
	grid-column: 1 / -1;
}

.form-actions {
	display: flex;
	align-items: flex-end;
	padding-top: 22px;
}

.permission-actions {
	display: flex;
	align-items: center;
	gap: 8px;
	min-width: 420px;
}

.permission-actions :deep(.el-select) {
	flex: 1;
}

.muted {
	color: var(--color-text-muted);
}

@media (max-width: 900px) {
	.page-header,
	.section-header,
	.permission-actions,
	.edit-form {
		display: grid;
		min-width: 0;
	}

	.edit-form :deep(.el-form-item:nth-child(3)) {
		grid-column: auto;
	}
}
</style>
