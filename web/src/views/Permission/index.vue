<script setup lang="ts">
import { onMounted, ref } from "vue";

import { useI18n } from "../../i18n";
import { type ApiResponse, apiGet, apiPost, apiPut } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type RoleRecord = {
	id: string;
	name: string;
	key?: string;
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
		key: String(row.key ?? row.Key ?? "").trim()
	};
}

const { t } = useI18n();

const loading = ref(false);
const saving = ref(false);
const error = ref("");
const success = ref("");

const roles = ref<RoleRecord[]>([]);
const selectedRoleId = ref("");
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
			<div class="rule-list">
				<div v-for="item in rules" :key="item.id" class="rule-row">
					<select v-model="item.resource" :disabled="saving">
						<option v-for="resource in resourceOptions" :key="resource" :value="resource">{{ resource }}</option>
					</select>
					<select v-model="item.action" :disabled="saving">
						<option v-for="action in actionOptions" :key="action" :value="action">{{ action }}</option>
					</select>
					<select v-model="item.effect" :disabled="saving">
						<option v-for="effect in effectOptions" :key="effect" :value="effect">{{ effect }}</option>
					</select>
					<select v-model="item.scope" :disabled="saving">
						<option v-for="scope in scopeOptions" :key="scope" :value="scope">{{ scope }}</option>
					</select>
					<button type="button" :disabled="saving || rules.length <= 1" @click="removeRule(item.id)">{{ t("common.delete") }}</button>
				</div>
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

.rule-list {
	display: grid;
	gap: 8px;
}

.rule-row {
	display: grid;
	grid-template-columns: repeat(5, minmax(0, 1fr));
	gap: 8px;
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

