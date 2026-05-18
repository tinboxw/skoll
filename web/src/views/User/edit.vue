<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";

import { useI18n } from "../../i18n";
import { type ApiResponse, apiDelete, apiGet, apiPost, apiPut } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type UserRecord = {
	id: string;
	account: string;
	name?: string;
	email: string;
	status?: string;
};

type RoleRecord = {
	id: string;
	name: string;
	key?: string;
};

type BindingRecord = {
	id: string;
	roleId: string;
	scope: string;
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

function normalizeBindingRecord(item: unknown): BindingRecord | null {
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
		roleId: String(row.roleId ?? row.RoleID ?? "").trim(),
		scope: String(row.scope ?? row.Scope ?? "").trim()
	};
}

const route = useRoute();
const { t } = useI18n();

const userId = computed(() => String(route.params.id ?? ""));
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const success = ref("");
const account = ref("");
const name = ref("");
const status = ref("");
const email = ref("");
const roles = ref<RoleRecord[]>([]);
const selectedRoleID = ref("");
const binding = ref(false);
const bindError = ref("");
const bindSuccess = ref("");
const bindings = ref<BindingRecord[]>([]);
const selectedBindingIDs = ref<string[]>([]);

async function loadUser(): Promise<void> {
	if (!userId.value) {
		error.value = "missing user id";
		return;
	}
	loading.value = true;
	error.value = "";
	try {
		const payload = await apiGet<ApiResponse<UserRecord>>(`/v1/users/${userId.value}`);
		const data = normalizeUserRecord(payload.data);
		if (!data) {
			throw new Error("invalid user payload");
		}
		account.value = data?.account ?? "";
		name.value = data?.name ?? "";
		status.value = data?.status ?? "";
		email.value = data?.email ?? "";
	} catch (e) {
		error.value = toErrorMessage(e);
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
		bindError.value = toErrorMessage(e);
	}
}

async function loadBindings(): Promise<void> {
	if (!userId.value) {
		return;
	}
	try {
		const payload = await apiGet<ApiResponse<BindingRecord[]>>(`/v1/rbac/bindings?subjectType=user&subjectId=${encodeURIComponent(userId.value)}`);
		bindings.value = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeBindingRecord(item)).filter((item): item is BindingRecord => item !== null)
			: [];
		selectedBindingIDs.value = [];
	} catch (e) {
		bindError.value = toErrorMessage(e);
	}
}

async function submit(): Promise<void> {
	if (!userId.value) {
		return;
	}
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		await apiPut<ApiResponse<UserRecord>>(`/v1/users/${userId.value}`, {
			name: name.value.trim(),
			email: email.value.trim(),
			status: status.value.trim()
		});
		success.value = t("user.updateDone");
		await loadUser();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function bindRole(): Promise<void> {
	if (!userId.value || !selectedRoleID.value) {
		return;
	}
	binding.value = true;
	bindError.value = "";
	bindSuccess.value = "";
	try {
		await apiPost<ApiResponse<unknown>>("/v1/rbac/bind", {
			subjectType: "user",
			subjectId: userId.value,
			roleId: selectedRoleID.value,
			scope: "all"
		});
		bindSuccess.value = t("user.roleAssignDone");
		await loadBindings();
	} catch (e) {
		bindError.value = toErrorMessage(e);
	} finally {
		binding.value = false;
	}
}

function roleNameByID(roleID: string): string {
	const matched = roles.value.find((item) => item.id === roleID);
	return matched ? `${matched.name} (${matched.key || matched.id})` : roleID;
}

function toggleBinding(bindingID: string): void {
	if (selectedBindingIDs.value.includes(bindingID)) {
		selectedBindingIDs.value = selectedBindingIDs.value.filter((id) => id !== bindingID);
		return;
	}
	selectedBindingIDs.value = [...selectedBindingIDs.value, bindingID];
}

async function unbindBindings(targetIDs: string[]): Promise<void> {
	if (targetIDs.length === 0) {
		return;
	}
	binding.value = true;
	bindError.value = "";
	bindSuccess.value = "";
	let successCount = 0;
	for (const id of targetIDs) {
		try {
			await apiDelete<ApiResponse<unknown>>(`/v1/rbac/bindings/${id}`);
		} catch {
			continue;
		}
		successCount += 1;
	}
	bindSuccess.value = `${t("user.roleUnbindDone")}: ${successCount}/${targetIDs.length}`;
	await loadBindings();
	binding.value = false;
}

async function unbindOne(bindingID: string): Promise<void> {
	await unbindBindings([bindingID]);
}

async function unbindSelected(): Promise<void> {
	await unbindBindings([...selectedBindingIDs.value]);
}

onMounted(() => {
	void loadUser();
	void loadRoles();
	void loadBindings();
});
</script>

<template>
	<section>
		<h2>{{ t("page.userEdit") }}</h2>
		<p>{{ t("user.editing") }}: {{ userId }}</p>
		<form @submit.prevent="submit">
			<label>
				<span>{{ t("user.account") }}</span>
				<input :value="account" type="text" disabled />
			</label>
			<label>
				<span>{{ t("table.name") }}</span>
				<input v-model="name" type="text" required :disabled="loading || saving" />
			</label>
			<label>
				<span>{{ t("table.status") }}</span>
				<select v-model="status" :disabled="loading || saving">
					<option value="active">active</option>
					<option value="disabled">disabled</option>
				</select>
			</label>
			<label>
				<span>{{ t("table.email") }}</span>
				<input v-model="email" type="email" required :disabled="loading || saving" />
			</label>
			<p v-if="error" class="error">{{ error }}</p>
			<p v-if="success" class="success">{{ success }}</p>
			<button type="submit" :disabled="loading || saving">{{ saving ? t("common.loading") : t("common.save") }}</button>
		</form>
		<section class="role-bind">
			<h3>{{ t("user.roleAssignTitle") }}</h3>
			<label>
				<span>{{ t("role.field") }}</span>
				<select v-model="selectedRoleID" :disabled="binding || roles.length === 0">
					<option v-for="role in roles" :key="role.id" :value="role.id">{{ role.name }} ({{ role.key || role.id }})</option>
				</select>
			</label>
			<p v-if="bindError" class="error">{{ bindError }}</p>
			<p v-if="bindSuccess" class="success">{{ bindSuccess }}</p>
			<button type="button" :disabled="binding || roles.length === 0 || !selectedRoleID" @click="bindRole">
				{{ binding ? t("common.loading") : t("user.assignRole") }}
			</button>
			<p class="hint">{{ t("user.roleAssignHint") }}</p>
			<section class="binding-list">
				<div class="binding-actions">
					<button type="button" :disabled="binding || selectedBindingIDs.length === 0" @click="unbindSelected">{{ t("user.unbindSelected") }}</button>
				</div>
				<table>
					<thead>
						<tr>
							<th></th>
							<th>{{ t("table.role") }}</th>
							<th>{{ t("table.scope") }}</th>
							<th>{{ t("table.actions") }}</th>
						</tr>
					</thead>
					<tbody v-if="bindings.length > 0">
						<tr v-for="item in bindings" :key="item.id">
							<td><input type="checkbox" :checked="selectedBindingIDs.includes(item.id)" :disabled="binding" @change="toggleBinding(item.id)" /></td>
							<td>{{ roleNameByID(item.roleId) }}</td>
							<td>{{ item.scope || "-" }}</td>
							<td>
								<button type="button" :disabled="binding" @click="unbindOne(item.id)">{{ t("user.unbind") }}</button>
							</td>
						</tr>
					</tbody>
					<tbody v-else>
						<tr>
							<td colspan="4">{{ t("common.empty") }}</td>
						</tr>
					</tbody>
				</table>
			</section>
		</section>
	</section>
</template>

<style scoped>
p,
span {
	color: var(--color-text-muted);
}

form {
	display: grid;
	gap: 8px;
	max-width: 420px;
	margin-bottom: 12px;
}

label {
	display: grid;
	gap: 6px;
}

select,
input,
button {
	padding: 8px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

button {
	background: var(--color-primary);
	color: var(--color-on-primary);
	border: none;
	cursor: pointer;
}

.role-bind {
	display: grid;
	gap: 8px;
	max-width: 420px;
}

.role-bind h3 {
	margin: 8px 0 0;
}

.error {
	margin: 0;
	color: var(--color-danger);
}

.success {
	margin: 0;
	color: var(--color-success);
}

.hint {
	font-size: 13px;
}

.binding-list {
	margin-top: 8px;
}

.binding-actions {
	margin-bottom: 6px;
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

