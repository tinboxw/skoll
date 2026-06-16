<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { Check, Close, Refresh, Select, Setting } from "@element-plus/icons-vue";

import { useI18n } from "../../i18n";
import { BASE_PERMISSION_CATALOG, BUILTIN_ROLE_DEFAULTS, type PermissionCatalogItem } from "../../permissions/catalog";
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

const loading = ref(false);
const saving = ref(false);
const matrixSaving = ref(false);
const error = ref("");
const success = ref("");

const roles = ref<RoleRecord[]>([]);
const selectedRoleId = ref("");
const selectedMatrixPermissions = ref<string[]>([]);
const rules = ref<RuleRow[]>([]);

const resourceOptions = ["user", "role", "rbac", "plugin", "system", "audit"];
const actionOptions = ["create", "read", "update", "delete", "manage", "*"];
const effectOptions: Array<"allow" | "deny"> = ["allow", "deny"];
const scopeOptions: Array<RuleRow["scope"]> = ["self", "dept", "dept_tree", "all", "custom"];

const checkSubjectType = ref("user");
const checkSubjectId = ref("");
const checkResource = ref(resourceOptions[0]);
const checkAction = ref(actionOptions[1]);
const checkResult = ref<"allowed" | "denied" | "">("");
const checkScope = ref("");

const selectedRole = computed(() => roles.value.find((item) => item.id === selectedRoleId.value) ?? null);

const roleDefaultRows = computed(() => roles.value.map((item) => ({
	...item,
	defaultPermissions: resolveRolePermissions(item)
})));

const permissionCatalog = computed(() => {
	const map = new Map<string, PermissionCatalogItem>();
	for (const item of BASE_PERMISSION_CATALOG) {
		map.set(item.key, item);
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

function resolveRolePermissions(role: RoleRecord): string[] {
	const key = String(role.key || "").trim().toLowerCase();
	const fromRole = Array.isArray(role.permissions) ? role.permissions.filter((it) => it.trim() !== "") : [];
	return fromRole.length > 0 ? fromRole : BUILTIN_ROLE_DEFAULTS[key] || [];
}

function newRuleRow(): RuleRow {
	return {
		id: crypto.randomUUID(),
		resource: resourceOptions[0],
		action: actionOptions[1],
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
	matrixSaving.value = true;
	error.value = "";
	success.value = "";
	try {
		const endpoint = enabled ? "grant" : "revoke";
		await apiPost<ApiResponse<unknown>>(`/v1/roles/${selectedRoleId.value}/${endpoint}`, { permission });
		const next = new Set(selectedMatrixPermissions.value.filter((item) => item !== "*"));
		if (enabled) {
			next.add(permission);
		} else {
			next.delete(permission);
		}
		selectedMatrixPermissions.value = Array.from(next).sort();
		const role = selectedRole.value;
		if (role) {
			role.permissions = [...selectedMatrixPermissions.value];
		}
		success.value = enabled ? t("role.permissionGranted") : t("role.permissionRevoked");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		matrixSaving.value = false;
	}
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

async function savePolicies(): Promise<void> {
	if (!selectedRoleId.value) {
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
	void loadRoles();
});
</script>

<template>
	<section class="permission-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.permissions") }}</h2>
				<p>{{ t("permission.desc") }}</p>
			</div>
			<el-button :icon="Refresh" :loading="loading" @click="loadRoles">{{ t("common.refresh") }}</el-button>
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

			<el-table :data="matrixRows" border stripe>
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
