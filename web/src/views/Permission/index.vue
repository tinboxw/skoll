<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import { useI18n } from "../../i18n";
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
			? (row.permissions ?? row.Permissions as unknown[]).filter((it): it is string => typeof it === "string").map((it) => it.trim()).filter((it) => it !== "")
			: [],
		builtIn: Boolean(row.builtIn ?? row.BuiltIn)
	};
}

const BUILTIN_ROLE_DEFAULTS: Record<string, string[]> = {
	super_admin: ["*", "user.read", "user.create", "user.update", "user.delete", "role.read", "role.create", "role.update", "role.delete", "permission.manage"],
	dept_admin: ["user.read", "user.update", "role.read"],
	operator: ["user.read", "role.read"],
	user: ["user.read"]
};

const { t } = useI18n();

const loading = ref(false);
const saving = ref(false);
const error = ref("");
const success = ref("");

const roles = ref<RoleRecord[]>([]);
const selectedRoleId = ref("");
const rules = ref<RuleRow[]>([]);

const roleDefaultRows = computed(() => {
	return roles.value.map((item) => {
		const key = String(item.key || "").trim().toLowerCase();
		const fromRole = Array.isArray(item.permissions) ? item.permissions.filter((it) => it.trim() !== "") : [];
		const fallback = BUILTIN_ROLE_DEFAULTS[key] || [];
		const defaults = fromRole.length > 0 ? fromRole : fallback;
		return {
			...item,
			defaultPermissions: defaults
		};
	});
});

const resourceOptions = ["user", "role", "rbac", "plugin", "system", "audit"];
const actionOptions = ["create", "read", "update", "delete", "manage", "*"];
const effectOptions: Array<"allow" | "deny"> = ["allow", "deny"];
const scopeOptions: Array<RuleRow["scope"]> = ["self", "dept", "dept_tree", "all", "custom"];

const checkSubjectType = ref("user");
const checkSubjectId = ref("");
const checkResource = ref(resourceOptions[0]);
const checkAction = ref(actionOptions[1]);
const checkResult = ref<"allowed" | "denied" | "">("");

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
	try {
		const payload = await apiPost<ApiResponse<PermissionCheckResult>>("/v1/rbac/check", {
			subjectType: checkSubjectType.value,
			subjectId: checkSubjectId.value.trim(),
			resource: `${checkResource.value.trim()}.${checkAction.value.trim()}`,
			action: checkAction.value.trim()
		});
		checkResult.value = payload.data?.allowed ? "allowed" : "denied";
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
	<section>
		<h2>{{ t("page.permissions") }}</h2>
		<p>{{ t("permission.desc") }}</p>

		<p v-if="error" class="error">{{ error }}</p>
		<p v-if="success" class="success">{{ success }}</p>

		<section class="panel">
			<h3>{{ t("permission.defaultsTitle") }}</h3>
			<p class="hint">{{ t("permission.defaultsHint") }}</p>
			<div class="defaults-wrap">
				<table class="defaults-table">
					<thead>
						<tr>
							<th>{{ t("table.name") }}</th>
							<th>{{ t("table.key") }}</th>
							<th>{{ t("permission.defaultPermissionList") }}</th>
						</tr>
					</thead>
					<tbody v-if="roleDefaultRows.length > 0">
						<tr v-for="item in roleDefaultRows" :key="item.id">
							<td>{{ item.name || item.id }}</td>
							<td>{{ item.key || "-" }}</td>
							<td>
								<div v-if="item.defaultPermissions.length > 0" class="perm-tags">
									<span v-for="perm in item.defaultPermissions" :key="item.id + ':' + perm" class="perm-tag">{{ perm }}</span>
								</div>
								<span v-else class="muted">{{ t("permission.noDefaults") }}</span>
							</td>
						</tr>
					</tbody>
					<tbody v-else>
						<tr>
							<td colspan="3">{{ t("common.empty") }}</td>
						</tr>
					</tbody>
				</table>
			</div>
		</section>

		<section class="panel">
			<h3>{{ t("permission.policyTitle") }}</h3>
			<label>
				<span>{{ t("permission.role") }}</span>
				<select v-model="selectedRoleId" :disabled="loading || roles.length === 0">
					<option v-for="item in roles" :key="item.id" :value="item.id">{{ item.name }} ({{ item.key || item.id }})</option>
				</select>
			</label>
			<label>
				<span>{{ t("permission.rulesSelect") }}</span>
			</label>
			<div class="rule-list-wrap">
				<table class="rule-table">
					<thead>
						<tr>
							<th>resource</th>
							<th>action</th>
							<th>effect</th>
							<th>scope</th>
							<th class="op-col">{{ t("table.actions") }}</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="item in rules" :key="item.id">
							<td>
								<select v-model="item.resource" :disabled="saving">
									<option v-for="resource in resourceOptions" :key="resource" :value="resource">{{ resource }}</option>
								</select>
							</td>
							<td>
								<select v-model="item.action" :disabled="saving">
									<option v-for="action in actionOptions" :key="action" :value="action">{{ action }}</option>
								</select>
							</td>
							<td>
								<select v-model="item.effect" :disabled="saving">
									<option v-for="effect in effectOptions" :key="effect" :value="effect">{{ effect }}</option>
								</select>
							</td>
							<td>
								<select v-model="item.scope" :disabled="saving">
									<option v-for="scope in scopeOptions" :key="scope" :value="scope">{{ scope }}</option>
								</select>
							</td>
							<td class="op-col">
								<button type="button" :disabled="saving || rules.length <= 1" @click="removeRule(item.id)">{{ t("common.delete") }}</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
			<button type="button" :disabled="saving" @click="addRule">{{ t("permission.addRule") }}</button>
			<button type="button" :disabled="saving || !selectedRoleId" @click="savePolicies">
				{{ saving ? t("common.loading") : t("permission.savePolicies") }}
			</button>
		</section>

		<section class="panel">
			<h3>{{ t("permission.checkTitle") }}</h3>
			<div class="grid">
				<label>
					<span>{{ t("permission.subjectType") }}</span>
					<select v-model="checkSubjectType" :disabled="saving">
						<option value="user">user</option>
						<option value="role">role</option>
					</select>
				</label>
				<label>
					<span>{{ t("permission.subjectId") }}</span>
					<input v-model="checkSubjectId" type="text" :disabled="saving" />
				</label>
				<label>
					<span>{{ t("permission.resource") }}</span>
					<select v-model="checkResource" :disabled="saving">
						<option v-for="resource in resourceOptions" :key="resource" :value="resource">{{ resource }}</option>
					</select>
				</label>
				<label>
					<span>{{ t("permission.action") }}</span>
					<select v-model="checkAction" :disabled="saving">
						<option v-for="action in actionOptions" :key="action" :value="action">{{ action }}</option>
					</select>
				</label>
			</div>
			<button type="button" :disabled="saving" @click="checkPermission">{{ t("permission.checkNow") }}</button>
			<p v-if="checkResult" class="result">{{ t(`permission.result.${checkResult}`) }}</p>
		</section>
	</section>
</template>

<style scoped>
p {
	color: var(--color-text-muted);
}

.panel {
	margin-top: 12px;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	display: grid;
	gap: 8px;
}

.panel h3 {
	margin: 0;
}

.hint {
	margin: 0;
	color: var(--color-text-muted);
	font-size: 13px;
}

.defaults-wrap {
	overflow: auto;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.defaults-table {
	width: 100%;
	border-collapse: collapse;
}

.defaults-table th,
.defaults-table td {
	text-align: left;
	padding: 8px;
	border-bottom: 1px solid var(--color-border);
}

.defaults-table th {
	font-size: 12px;
	color: var(--color-text-muted);
	background: var(--color-surface-soft);
}

.defaults-table tr:last-child td {
	border-bottom: 0;
}

.perm-tags {
	display: flex;
	flex-wrap: wrap;
	gap: 6px;
}

.perm-tag {
	padding: 2px 8px;
	border: 1px solid var(--color-border);
	border-radius: 999px;
	font-size: 12px;
	background: var(--color-surface-soft);
}

.muted {
	color: var(--color-text-muted);
}

.grid {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 8px;
}

label {
	display: grid;
	gap: 6px;
}

input,
select,
button {
	padding: 8px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
}

.rule-list-wrap {
	overflow: auto;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

.rule-table {
	width: 100%;
	border-collapse: collapse;
}

.rule-table th,
.rule-table td {
	padding: 8px;
	border-bottom: 1px solid var(--color-border);
	text-align: left;
}

.rule-table th {
	font-size: 12px;
	font-weight: 600;
	color: var(--color-text-muted);
	background: var(--color-surface);
}

.rule-table tr:last-child td {
	border-bottom: 0;
}

.rule-table td select {
	width: 100%;
}

.op-col {
	white-space: nowrap;
}

button {
	width: fit-content;
	background: var(--color-surface-soft);
	cursor: pointer;
}

.error {
	color: var(--color-danger);
}

.success {
	color: var(--color-success);
}

.result {
	margin: 0;
	font-weight: 600;
}
</style>

