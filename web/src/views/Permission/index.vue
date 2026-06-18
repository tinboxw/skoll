<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { Check, Close, Refresh, Select, Setting } from "@element-plus/icons-vue";

import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import type { PermissionResource } from "../../permissions/api";
import { usePermissionStore } from "../../stores/permissions";
import { type ApiResponse, apiGet, apiPost, apiPut } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type RoleRecord = {
	id: string;
	name: string;
	key?: string;
	permissions?: string[];
	builtIn?: boolean;
};

type PolicyRule = {
	resource: string;
	action: string;
	effect: "allow" | "deny";
	scope: "self" | "dept" | "dept_tree" | "all" | "custom";
};

type RuleRow = PolicyRule & { id: string };

type PermissionCheckResult = {
	allowed?: boolean;
	scope?: PolicyRule["scope"];
};

type PermissionCatalogItem = {
	key: string;
	resource: string;
	action: string;
	label: string;
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
	return {
		id,
		name: String(row.name ?? row.Name ?? "").trim(),
		key: String(row.key ?? row.Key ?? "").trim(),
		permissions: Array.isArray(row.permissions ?? row.Permissions)
			? ((row.permissions ?? row.Permissions) as unknown[]).filter((it): it is string => typeof it === "string").map((it) => it.trim()).filter((it) => it !== "")
			: [],
		builtIn: Boolean(row.builtIn ?? row.BuiltIn)
	};
}

const { t } = useI18n();
const permissionStore = usePermissionStore();

const loading = ref(false);
const saving = ref(false);
const matrixSaving = ref(false);
const error = ref("");
const success = ref("");

const roles = ref<RoleRecord[]>([]);
const selectedRoleId = ref("");
const selectedMatrixPermissions = ref<string[]>([]);
const rules = ref<RuleRow[]>([]);

const effectOptions: Array<"allow" | "deny"> = ["allow", "deny"];
const scopeOptions: Array<RuleRow["scope"]> = ["self", "dept", "dept_tree", "all", "custom"];

const checkSubjectType = ref("user");
const checkSubjectId = ref("");
const checkResource = ref("");
const checkAction = ref("");
const checkResult = ref<"allowed" | "denied" | "">("");
const checkScope = ref("");

const selectedRole = computed(() => roles.value.find((item) => item.id === selectedRoleId.value) ?? null);

const roleDefaultRows = computed(() => roles.value.map((item) => ({
	...item,
	defaultPermissions: resolveRolePermissions(item)
})));

const permissionCatalog = computed(() => {
	const map = new Map<string, PermissionCatalogItem>();
	for (const item of permissionStore.enabledItems) {
		map.set(item.key, permissionResourceToCatalogItem(item));
	}
	for (const role of roleDefaultRows.value) {
		for (const permission of role.defaultPermissions) {
			if (permission === "*" || map.has(permission)) {
				continue;
			}
			const [resource = "custom", action = "manage"] = permission.split(".");
			map.set(permission, {
				key: permission,
				resource,
				action,
				label: permission
			});
		}
	}
	return Array.from(map.values()).sort((a, b) => a.resource.localeCompare(b.resource) || a.action.localeCompare(b.action));
});

const resourceOptions = computed(() => {
	const values = unique(permissionCatalog.value.map((item) => item.resource));
	return values.length > 0 ? values : ["system"];
});

const actionOptions = computed(() => {
	const values = unique(permissionCatalog.value.map((item) => item.action));
	return values.length > 0 ? values : ["read", "manage"];
});

const matrixRows = computed(() => {
	const grouped = new Map<string, PermissionCatalogItem[]>();
	for (const permission of permissionCatalog.value) {
		if (!grouped.has(permission.resource)) {
			grouped.set(permission.resource, []);
		}
		grouped.get(permission.resource)!.push(permission);
	}
	return Array.from(grouped.entries()).map(([resource, permissions]) => ({
		resource,
		permissions
	}));
});

watch(selectedRole, (role) => {
	selectedMatrixPermissions.value = role ? resolveRolePermissions(role) : [];
}, { immediate: true });

watch(resourceOptions, (options) => {
	if (!options.includes(checkResource.value)) {
		checkResource.value = options[0] ?? "";
	}
}, { immediate: true });

watch(actionOptions, (options) => {
	if (!options.includes(checkAction.value)) {
		checkAction.value = options.includes("read") ? "read" : options[0] ?? "";
	}
}, { immediate: true });

function resolveRolePermissions(role: RoleRecord): string[] {
	const fromRole = Array.isArray(role.permissions) ? role.permissions.filter((it) => it.trim() !== "") : [];
	return fromRole;
}

function newRuleRow(): RuleRow {
	return {
		id: crypto.randomUUID(),
		resource: resourceOptions.value[0] ?? "system",
		action: actionOptions.value.includes("read") ? "read" : actionOptions.value[0] ?? "manage",
		effect: "allow",
		scope: "all"
	};
}

function addRule(): void {
	rules.value.push(newRuleRow());
}

function removeRule(id: string): void {
	rules.value = rules.value.filter((item) => item.id !== id);
}

function hasMatrixPermission(permission: string): boolean {
	return selectedMatrixPermissions.value.includes("*") || selectedMatrixPermissions.value.includes(permission);
}

async function toggleMatrixPermission(permission: string, enabled: boolean): Promise<void> {
	if (!selectedRoleId.value || permission === "*") {
		return;
	}
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: enabled ? t("permission.grantConfirm") : t("permission.revokeConfirm"),
		confirmText: t("common.confirm"),
		cancelText: t("common.cancel"),
		danger: !enabled
	});
	if (!confirmed) {
		return;
	}
	matrixSaving.value = true;
	error.value = "";
	success.value = "";
	try {
		const endpoint = enabled ? "grant" : "revoke";
		const payload = await apiPost<ApiResponse<RoleRecord>>(`/v1/roles/${selectedRoleId.value}/${endpoint}`, { permission });
		const updatedRole = normalizeRoleRecord(payload.data);
		if (updatedRole) {
			patchRole(updatedRole);
			selectedMatrixPermissions.value = resolveRolePermissions(updatedRole);
		} else {
			await loadRoles();
		}
		success.value = enabled ? t("role.permissionGranted") : t("role.permissionRevoked");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		matrixSaving.value = false;
	}
}

function patchRole(role: RoleRecord): void {
	const index = roles.value.findIndex((item) => item.id === role.id);
	if (index >= 0) {
		roles.value[index] = role;
		return;
	}
	roles.value.push(role);
}

async function loadRoles(): Promise<void> {
	loading.value = true;
	error.value = "";
	try {
		const payload = await apiGet<ApiResponse<RoleRecord[]>>("/v1/roles?offset=0&limit=100");
		roles.value = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeRoleRecord(item)).filter((item): item is RoleRecord => item !== null)
			: [];
		if (!selectedRoleId.value && roles.value.length > 0) {
			selectedRoleId.value = roles.value[0].id;
		}
		if (rules.value.length === 0) {
			rules.value = [newRuleRow()];
		}
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

async function loadPermissionCatalog(): Promise<void> {
	try {
		await permissionStore.load({ limit: 500 }, { force: true });
	} catch (e) {
		error.value = toErrorMessage(e);
	}
}

async function loadPage(): Promise<void> {
	await Promise.all([
		loadRoles(),
		loadPermissionCatalog()
	]);
}

async function savePolicies(): Promise<void> {
	if (!selectedRoleId.value) {
		return;
	}
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: t("permission.savePoliciesConfirm"),
		confirmText: t("common.save"),
		cancelText: t("common.cancel"),
		type: "warning"
	});
	if (!confirmed) {
		return;
	}
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		const payloadRules: PolicyRule[] = rules.value.map((item) => ({
			resource: `${item.resource}.*`,
			action: item.action,
			effect: item.effect,
			scope: item.scope
		}));
		await apiPut<ApiResponse<unknown>>(`/v1/rbac/policies/${selectedRoleId.value}`, { rules: payloadRules });
		success.value = t("permission.saveDone");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function checkPermission(): Promise<void> {
	saving.value = true;
	error.value = "";
	checkResult.value = "";
	checkScope.value = "";
	try {
		const payload = await apiPost<ApiResponse<PermissionCheckResult>>("/v1/rbac/check", {
			subjectType: checkSubjectType.value,
			subjectId: checkSubjectId.value.trim(),
			resource: `${checkResource.value.trim()}.${checkAction.value.trim()}`,
			action: checkAction.value.trim()
		});
		checkResult.value = payload.data?.allowed ? "allowed" : "denied";
		checkScope.value = payload.data?.scope ?? "";
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

onMounted(() => {
	void loadPage();
});

function permissionResourceToCatalogItem(item: PermissionResource): PermissionCatalogItem {
	return {
		key: item.key,
		resource: item.metadata.resource || item.module || parsePermissionResource(item.key),
		action: item.metadata.action || parsePermissionAction(item.key) || item.type,
		label: item.name || item.key
	};
}

function parsePermissionResource(permission: string): string {
	const normalized = permission.trim();
	if (normalized.includes(".")) {
		return normalized.split(".")[0] || "custom";
	}
	if (normalized.includes(":")) {
		return normalized.split(":")[0] || "custom";
	}
	return "custom";
}

function parsePermissionAction(permission: string): string {
	const normalized = permission.trim();
	if (normalized.includes(".")) {
		return normalized.split(".").pop() || "manage";
	}
	if (normalized.includes(":")) {
		return normalized.split(":").pop() || "manage";
	}
	return "manage";
}

function unique(values: string[]): string[] {
	return Array.from(new Set(values.map((item) => item.trim()).filter((item) => item !== ""))).sort();
}
</script>

<template>
	<section class="permission-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.permissions") }}</h2>
				<p>{{ t("permission.desc") }}</p>
			</div>
			<el-button :icon="Refresh" :loading="loading || permissionStore.isLoading" @click="loadPage">{{ t("common.refresh") }}</el-button>
		</header>

		<el-alert v-if="error" type="error" :title="error" show-icon :closable="false" />
		<el-alert v-if="success" type="success" :title="success" show-icon :closable="false" />

		<el-card shadow="never">
			<template #header>
				<div class="card-header">
					<div>
						<h3>{{ t("permission.matrixTitle") }}</h3>
						<p>{{ t("permission.matrixHint") }}</p>
					</div>
					<el-select v-model="selectedRoleId" :disabled="loading || roles.length === 0" filterable class="role-select">
						<el-option v-for="item in roles" :key="item.id" :value="item.id" :label="`${item.name || item.id} (${item.key || item.id})`" />
					</el-select>
				</div>
			</template>

			<el-table v-loading="loading || permissionStore.isLoading" :data="matrixRows" border stripe>
				<el-table-column prop="resource" label="Resource" width="150" />
				<el-table-column label="Permissions" min-width="520">
					<template #default="{ row }">
						<div class="permission-switches">
							<el-check-tag
								v-for="permission in row.permissions"
								:key="permission.key"
								:checked="hasMatrixPermission(permission.key)"
								:disabled="matrixSaving || selectedMatrixPermissions.includes('*')"
								@click="toggleMatrixPermission(permission.key, !hasMatrixPermission(permission.key))"
							>
								{{ permission.key }}
							</el-check-tag>
						</div>
					</template>
				</el-table-column>
			</el-table>

			<div class="matrix-footer">
				<el-tag v-if="selectedMatrixPermissions.includes('*')" type="success" effect="plain">super permission *</el-tag>
				<el-tag v-else effect="plain">{{ selectedMatrixPermissions.length }} permissions</el-tag>
			</div>
		</el-card>

		<div class="content-grid">
			<el-card shadow="never">
				<template #header>
					<div class="card-header">
						<div>
							<h3>{{ t("permission.defaultsTitle") }}</h3>
							<p>{{ t("permission.defaultsHint") }}</p>
						</div>
					</div>
				</template>
				<el-table :data="roleDefaultRows" border stripe>
					<el-table-column prop="name" :label="t('table.name')" min-width="150" show-overflow-tooltip />
					<el-table-column prop="key" :label="t('table.key')" width="140" />
					<el-table-column :label="t('permission.defaultPermissionList')" min-width="260">
						<template #default="{ row }">
							<div v-if="row.defaultPermissions.length > 0" class="perm-tags">
								<el-tag v-for="perm in row.defaultPermissions" :key="row.id + ':' + perm" size="small" effect="plain">{{ perm }}</el-tag>
							</div>
							<span v-else class="muted">{{ t("permission.noDefaults") }}</span>
						</template>
					</el-table-column>
				</el-table>
			</el-card>

			<el-card shadow="never">
				<template #header>
					<div class="card-header">
						<div>
							<h3>{{ t("permission.checkTitle") }}</h3>
							<p>{{ t("permission.hint1") }}</p>
						</div>
					</div>
				</template>
				<el-form label-position="top">
					<div class="check-grid">
						<el-form-item :label="t('permission.subjectType')">
							<el-select v-model="checkSubjectType" :disabled="saving">
								<el-option label="user" value="user" />
								<el-option label="role" value="role" />
							</el-select>
						</el-form-item>
						<el-form-item :label="t('permission.subjectId')">
							<el-input v-model="checkSubjectId" :disabled="saving" clearable />
						</el-form-item>
						<el-form-item :label="t('permission.resource')">
							<el-select v-model="checkResource" :disabled="saving">
								<el-option v-for="resource in resourceOptions" :key="resource" :label="resource" :value="resource" />
							</el-select>
						</el-form-item>
						<el-form-item :label="t('permission.action')">
							<el-select v-model="checkAction" :disabled="saving">
								<el-option v-for="action in actionOptions" :key="action" :label="action" :value="action" />
							</el-select>
						</el-form-item>
					</div>
					<el-button type="primary" :icon="checkResult === 'allowed' ? Check : Select" :loading="saving" @click="checkPermission">{{ t("permission.checkNow") }}</el-button>
					<el-tag v-if="checkResult" class="check-result" :type="checkResult === 'allowed' ? 'success' : 'danger'" effect="light">
						<el-icon><component :is="checkResult === 'allowed' ? Check : Close" /></el-icon>
						{{ t(`permission.result.${checkResult}`) }}
					</el-tag>
					<el-tag v-if="checkScope" class="check-result" effect="plain">
						{{ t("permission.dataScope") }}: {{ checkScope }}
					</el-tag>
				</el-form>
			</el-card>
		</div>

		<el-card shadow="never">
			<template #header>
				<div class="card-header">
					<div>
						<h3>{{ t("permission.policyTitle") }}</h3>
						<p>{{ t("permission.hint2") }}</p>
					</div>
					<el-button :icon="Setting" :disabled="saving" @click="addRule">{{ t("permission.addRule") }}</el-button>
				</div>
			</template>
			<el-table :data="rules" border stripe>
				<el-table-column label="resource" min-width="150">
					<template #default="{ row }">
						<el-select v-model="row.resource" :disabled="saving">
							<el-option v-for="resource in resourceOptions" :key="resource" :label="resource" :value="resource" />
						</el-select>
					</template>
				</el-table-column>
				<el-table-column label="action" min-width="140">
					<template #default="{ row }">
						<el-select v-model="row.action" :disabled="saving">
							<el-option v-for="action in actionOptions" :key="action" :label="action" :value="action" />
						</el-select>
					</template>
				</el-table-column>
				<el-table-column label="effect" min-width="140">
					<template #default="{ row }">
						<el-select v-model="row.effect" :disabled="saving">
							<el-option v-for="effect in effectOptions" :key="effect" :label="effect" :value="effect" />
						</el-select>
					</template>
				</el-table-column>
				<el-table-column label="scope" min-width="150">
					<template #default="{ row }">
						<el-select v-model="row.scope" :disabled="saving">
							<el-option v-for="scope in scopeOptions" :key="scope" :label="scope" :value="scope" />
						</el-select>
					</template>
				</el-table-column>
				<el-table-column :label="t('table.actions')" width="120">
					<template #default="{ row }">
						<el-button type="danger" link :disabled="saving || rules.length <= 1" @click="removeRule(row.id)">{{ t("common.delete") }}</el-button>
					</template>
				</el-table-column>
			</el-table>
			<div class="policy-actions">
				<el-button type="primary" :loading="saving" :disabled="!selectedRoleId" @click="savePolicies">
					{{ saving ? t("common.loading") : t("permission.savePolicies") }}
				</el-button>
			</div>
		</el-card>
	</section>
</template>

<style scoped>
.permission-page {
	display: grid;
	gap: 16px;
}

.page-header,
.card-header,
.matrix-footer,
.policy-actions {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 12px;
}

.page-header h2,
.card-header h3 {
	margin: 0;
}

.page-header p,
.card-header p {
	margin: 4px 0 0;
	color: var(--color-text-muted);
	font-size: 0.9rem;
}

.role-select {
	width: min(360px, 100%);
}

.permission-switches,
.perm-tags {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}

.matrix-footer,
.policy-actions {
	justify-content: flex-end;
	margin-top: 12px;
}

.content-grid {
	display: grid;
	grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.8fr);
	gap: 16px;
	align-items: start;
}

.check-grid {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 12px;
}

.check-result {
	margin-left: 10px;
}

.muted {
	color: var(--color-text-muted);
}

:deep(.el-card__header) {
	padding: 14px 16px;
}

:deep(.el-card__body) {
	padding: 16px;
}

:deep(.el-table) {
	--el-table-header-bg-color: var(--color-surface-soft);
}

@media (max-width: 980px) {
	.page-header,
	.card-header {
		flex-direction: column;
	}

	.content-grid,
	.check-grid {
		grid-template-columns: 1fr;
	}
}
</style>
