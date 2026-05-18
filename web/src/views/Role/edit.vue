<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";

import { useI18n } from "../../i18n";
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

const permissionOptions = [
	"user.read",
	"user.write",
	"role.read",
	"role.write",
	"rbac.manage",
	"plugin.manage",
	"system.manage"
];

const roleId = computed(() => String(route.params.id ?? ""));
const roleName = ref("");
const roleKey = ref("");
const roleDescription = ref("");
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const success = ref("");
const selectedPermission = ref(permissionOptions[0]);
const permissions = ref<string[]>([]);
const users = ref<UserRecord[]>([]);
const usersLoading = ref(false);
const usersError = ref("");

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
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

async function saveRole(): Promise<void> {
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
		success.value = t("role.permissionGranted");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function revokePermission(permission: string): Promise<void> {
	if (!roleId.value || !permission.trim()) {
		return;
	}
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		await apiPost<ApiResponse<unknown>>(`/v1/roles/${roleId.value}/revoke`, { permission });
		permissions.value = permissions.value.filter((item) => item !== permission);
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
	<section>
		<h2>{{ t("page.roleEdit") }}</h2>
		<p>{{ roleId }}</p>
		<p v-if="error" class="error">{{ error }}</p>
		<p v-if="success" class="success">{{ success }}</p>
		<form class="edit-form" @submit.prevent="saveRole">
			<label>
				<span>{{ t("table.name") }}</span>
				<input v-model="roleName" type="text" required :disabled="loading || saving" />
			</label>
			<label>
				<span>{{ t("table.key") }}</span>
				<input v-model="roleKey" type="text" required :disabled="loading || saving" />
			</label>
			<label>
				<span>{{ t("table.description") }}</span>
				<input v-model="roleDescription" type="text" :disabled="loading || saving" />
			</label>
			<button type="submit" :disabled="loading || saving">{{ saving ? t("common.loading") : t("common.save") }}</button>
		</form>
		<form class="add-form" @submit.prevent="grantPermission">
			<select v-model="selectedPermission" :disabled="loading || saving">
				<option v-for="item in permissionOptions" :key="item" :value="item">{{ item }}</option>
			</select>
			<button type="submit" :disabled="loading || saving || !selectedPermission.trim()">
				{{ saving ? t("common.loading") : t("role.addPermission") }}
			</button>
		</form>
		<ul>
			<li v-for="item in permissions" :key="item">
				<label class="row">
					<span>{{ item }}</span>
					<button type="button" :disabled="saving" @click="revokePermission(item)">{{ t("role.revokePermission") }}</button>
				</label>
			</li>
			<li v-if="permissions.length === 0" class="empty">{{ t("common.empty") }}</li>
		</ul>
		<section class="role-users">
			<div class="role-users-header">
				<h3>{{ t("role.usersTitle") }}</h3>
				<button type="button" :disabled="usersLoading || saving" @click="loadRoleUsers">{{ usersLoading ? t("common.loading") : t("common.refresh") }}</button>
			</div>
			<p v-if="usersError" class="error">{{ usersError }}</p>
			<table>
				<thead>
					<tr>
						<th>{{ t("table.id") }}</th>
						<th>{{ t("table.name") }}</th>
						<th>{{ t("table.email") }}</th>
						<th>{{ t("table.actions") }}</th>
					</tr>
				</thead>
				<tbody v-if="users.length > 0">
					<tr v-for="item in users" :key="item.id">
						<td>{{ item.id }}</td>
						<td>{{ item.name || item.account || "-" }}</td>
						<td>{{ item.email || "-" }}</td>
						<td>
							<router-link :to="`/skoll/user/${item.id}/edit`">{{ t("common.edit") }}</router-link>
						</td>
					</tr>
				</tbody>
				<tbody v-else>
					<tr>
						<td colspan="4">{{ usersLoading ? t("common.loading") : t("common.empty") }}</td>
					</tr>
				</tbody>
			</table>
		</section>
	</section>
</template>

<style scoped>
ul {
	padding: 0;
	list-style: none;
	display: grid;
	gap: 8px;
}

.row {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 8px;
	color: var(--color-text-muted);
}

.add-form {
	display: flex;
	gap: 8px;
	margin-bottom: 12px;
}

.edit-form {
	display: grid;
	gap: 8px;
	max-width: 460px;
	margin-bottom: 12px;
}

.edit-form label {
	display: grid;
	gap: 6px;
}

select,
input,
button {
	height: 32px;
	padding: 0 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

button {
	background: var(--color-surface-soft);
	cursor: pointer;
}

.error {
	color: var(--color-danger);
}

.success {
	color: var(--color-success);
}

.empty {
	color: var(--color-text-muted);
}

.role-users {
	margin-top: 12px;
}

.role-users-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-bottom: 8px;
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
</style>

