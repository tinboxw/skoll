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
</style>

