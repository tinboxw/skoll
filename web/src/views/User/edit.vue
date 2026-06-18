<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";

import { useI18n } from "../../i18n";
import { BUTTON_ACCESS, useButtonAccess } from "../../permissions/button";
import { type ApiResponse, apiDelete, apiGet, apiPost, apiPut } from "../../utils/api";
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

type BindingRecord = {
	id: string;
	roleId: string;
	scope: string;
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

const route = useRoute();
const { t } = useI18n();
const buttonAccess = useButtonAccess();

const userId = computed(() => String(route.params.id ?? ""));
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const success = ref("");
const account = ref("");
const name = ref("");
const status = ref("");
const email = ref("");
const departmentId = ref("");
const positionId = ref("");
const departments = ref<OrganizationOption[]>([]);
const positions = ref<OrganizationOption[]>([]);
const roles = ref<RoleRecord[]>([]);
const selectedRoleID = ref("");
const binding = ref(false);
const bindError = ref("");
const bindSuccess = ref("");
const bindings = ref<BindingRecord[]>([]);
const selectedBindingIDs = ref<string[]>([]);
const canUpdateUser = computed(() => buttonAccess.can(BUTTON_ACCESS.userUpdate));
const canAssignRole = computed(() => buttonAccess.can(BUTTON_ACCESS.roleManage));

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
		departmentId.value = data?.departmentId ?? "";
		positionId.value = data?.positionId ?? "";
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
	if (!canUpdateUser.value) {
		error.value = t("error.forbidden");
		return;
	}
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
			status: status.value.trim(),
			departmentId: departmentId.value.trim(),
			positionId: positionId.value.trim()
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
	if (!canAssignRole.value) {
		bindError.value = t("error.forbidden");
		return;
	}
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

function handleBindingSelectionChange(selection: BindingRecord[]): void {
	selectedBindingIDs.value = selection.map((item) => item.id);
}

async function unbindBindings(targetIDs: string[]): Promise<void> {
	if (!canAssignRole.value) {
		bindError.value = t("error.forbidden");
		return;
	}
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
	void loadOrganizationOptions();
	void loadUser();
	void loadRoles();
	void loadBindings();
});
</script>

<template>
	<section class="user-edit-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.userEdit") }}</h2>
				<p>{{ t("user.editing") }}: {{ userId }}</p>
			</div>
		</header>

		<section class="panel">
			<h3>{{ t("table.name") }}</h3>
			<el-form label-position="top" class="edit-form" @submit.prevent="submit">
				<el-form-item :label="t('user.account')">
					<el-input :model-value="account" disabled />
				</el-form-item>
				<el-form-item :label="t('table.name')" required>
					<el-input v-model="name" :disabled="loading || saving || !canUpdateUser" />
				</el-form-item>
				<el-form-item :label="t('table.status')">
					<el-select v-model="status" :disabled="loading || saving || !canUpdateUser">
						<el-option label="active" value="active" />
						<el-option label="disabled" value="disabled" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('table.email')" required>
					<el-input v-model="email" type="email" :disabled="loading || saving || !canUpdateUser" />
				</el-form-item>
				<el-form-item :label="t('table.department')">
					<el-select v-model="departmentId" clearable filterable :placeholder="t('user.departmentPlaceholder')" :disabled="loading || saving || !canUpdateUser">
						<el-option v-for="item in departments" :key="item.id" :label="item.name" :value="item.id" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('table.position')">
					<el-select v-model="positionId" clearable filterable :placeholder="t('user.positionPlaceholder')" :disabled="loading || saving || !canUpdateUser">
						<el-option v-for="item in positions" :key="item.id" :label="item.name" :value="item.id" />
					</el-select>
				</el-form-item>
				<div class="form-actions">
					<el-button type="primary" native-type="submit" :loading="saving" :disabled="loading || !canUpdateUser">{{ t("common.save") }}</el-button>
				</div>
			</el-form>
			<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
			<el-alert v-if="success" :title="success" type="success" show-icon :closable="false" />
		</section>

		<section v-if="canAssignRole" class="panel">
			<div class="role-header">
				<div>
					<h3>{{ t("user.roleAssignTitle") }}</h3>
					<p>{{ t("user.roleAssignHint") }}</p>
				</div>
				<div class="role-actions">
					<el-select v-model="selectedRoleID" :disabled="binding || roles.length === 0" class="role-select">
						<el-option v-for="role in roles" :key="role.id" :value="role.id" :label="`${role.name} (${role.key || role.id})`" />
					</el-select>
					<el-button type="primary" :loading="binding" :disabled="roles.length === 0 || !selectedRoleID" @click="bindRole">{{ t("user.assignRole") }}</el-button>
				</div>
			</div>
			<el-alert v-if="bindError" :title="bindError" type="error" show-icon :closable="false" />
			<el-alert v-if="bindSuccess" :title="bindSuccess" type="success" show-icon :closable="false" />
			<div class="binding-actions">
				<el-button type="danger" plain :disabled="binding || selectedBindingIDs.length === 0" @click="unbindSelected">{{ t("user.unbindSelected") }}</el-button>
			</div>
			<el-table
				v-loading="binding"
				:data="bindings"
				border
				row-key="id"
				:empty-text="t('common.empty')"
				@selection-change="handleBindingSelectionChange"
			>
				<el-table-column type="selection" width="48" :selectable="() => !binding" />
				<el-table-column :label="t('table.role')" min-width="200">
					<template #default="{ row }">{{ roleNameByID(row.roleId) }}</template>
				</el-table-column>
				<el-table-column :label="t('table.scope')" width="130">
					<template #default="{ row }">
						<el-tag effect="plain">{{ row.scope || "-" }}</el-tag>
					</template>
				</el-table-column>
				<el-table-column :label="t('table.actions')" width="120" fixed="right">
					<template #default="{ row }">
						<el-button link type="danger" :disabled="binding" @click="unbindOne(row.id)">{{ t("user.unbind") }}</el-button>
					</template>
				</el-table-column>
			</el-table>
		</section>
	</section>
</template>

<style scoped>
.user-edit-page {
	display: grid;
	gap: 14px;
}

.page-header h2 {
	margin: 0;
	font-size: 1.35rem;
}

.page-header p,
.role-header p {
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

.form-actions {
	display: flex;
	align-items: flex-end;
	padding-top: 22px;
}

.role-header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 16px;
}

.role-actions {
	display: flex;
	align-items: center;
	gap: 8px;
}

.role-select {
	width: 280px;
}

.binding-actions {
	display: flex;
	justify-content: flex-end;
}

@media (max-width: 900px) {
	.edit-form,
	.role-header,
	.role-actions {
		display: grid;
	}

	.role-select {
		width: 100%;
	}
}
</style>

