<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useI18n } from "../../i18n";
import { useAccess } from "../../permissions/access";
import { type ApiResponse, apiGet, apiPost } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type UserRecord = {
	id: string;
	account: string;
	name?: string;
	email: string;
};

type OrganizationOption = {
	id: string;
	name: string;
};

const router = useRouter();
const route = useRoute();
const { t } = useI18n();
const access = useAccess();

const account = ref("");
const name = ref("");
const email = ref("");
const password = ref("");
const departmentId = ref("");
const positionId = ref("");
const departments = ref<OrganizationOption[]>([]);
const positions = ref<OrganizationOption[]>([]);
const loading = ref(false);
const error = ref("");
const success = ref("");
const canCreateUser = computed(() => access.can("user.create"));

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

function resolveReturnTo(): string {
	const value = route.query.returnTo;
	if (typeof value === "string" && value.trim() !== "") {
		return value;
	}
	return "/skoll/user";
}

async function backToList(): Promise<void> {
	await router.push(resolveReturnTo());
}

function validateForm(): string {
	if (account.value.trim() === "") {
		return t("user.accountRequired");
	}
	if (email.value.trim() === "" || !email.value.includes("@")) {
		return t("user.emailInvalid");
	}
	if (password.value.trim().length < 8) {
		return t("user.passwordInvalid");
	}
	return "";
}

async function submit(): Promise<void> {
	error.value = "";
	success.value = "";
	if (!canCreateUser.value) {
		error.value = t("error.forbidden");
		return;
	}
	const validationMessage = validateForm();
	if (validationMessage !== "") {
		error.value = validationMessage;
		return;
	}
	loading.value = true;
	try {
		const payload = await apiPost<ApiResponse<UserRecord>>("/v1/users", {
			account: account.value.trim(),
			name: name.value.trim() || account.value.trim(),
			email: email.value.trim(),
			passwordHash: password.value.trim(),
			departmentId: departmentId.value.trim(),
			positionId: positionId.value.trim()
		});
		const created = payload.data as unknown as Record<string, unknown> | undefined;
		const createdName = String(created?.account ?? created?.Account ?? account.value).trim();
		success.value = `${t("user.createDone")}: ${createdName}`;
		setTimeout(() => {
			void router.push(resolveReturnTo());
		}, 300);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

void loadOrganizationOptions();
</script>

<template>
	<section class="user-add-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.userAdd") }}</h2>
				<p>{{ t("user.addHint") }}</p>
			</div>
			<el-button :disabled="loading" @click="backToList">{{ t("common.backToList") }}</el-button>
		</header>

		<section class="create-panel">
			<h3>{{ t("user.create") }}</h3>
			<el-form label-position="top" class="create-form" @submit.prevent="submit">
				<el-form-item :label="t('user.account')" required>
					<el-input v-model="account" :placeholder="t('user.account')" :disabled="loading || !canCreateUser" @blur="error = ''" />
				</el-form-item>
				<el-form-item :label="t('table.name')">
					<el-input v-model="name" :placeholder="t('table.name')" :disabled="loading || !canCreateUser" />
				</el-form-item>
				<el-form-item :label="t('table.email')" required>
					<el-input v-model="email" type="email" :placeholder="t('table.email')" :disabled="loading || !canCreateUser" />
				</el-form-item>
				<el-form-item :label="t('table.department')">
					<el-select v-model="departmentId" clearable filterable :placeholder="t('user.departmentPlaceholder')" :disabled="loading || !canCreateUser">
						<el-option v-for="item in departments" :key="item.id" :label="item.name" :value="item.id" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('table.position')">
					<el-select v-model="positionId" clearable filterable :placeholder="t('user.positionPlaceholder')" :disabled="loading || !canCreateUser">
						<el-option v-for="item in positions" :key="item.id" :label="item.name" :value="item.id" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('profile.newPassword')" required>
					<el-input v-model="password" type="password" show-password minlength="8" :placeholder="t('profile.newPassword')" :disabled="loading || !canCreateUser" />
				</el-form-item>
				<div class="form-actions">
					<el-button type="primary" native-type="submit" :loading="loading" :disabled="!canCreateUser">{{ t("user.create") }}</el-button>
				</div>
			</el-form>
			<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
			<el-alert v-if="success" :title="success" type="success" show-icon :closable="false" />
		</section>
	</section>
</template>

<style scoped>
.user-add-page {
	display: grid;
	gap: 14px;
}

.page-header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
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

.create-panel {
	display: grid;
	gap: 12px;
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.create-panel h3 {
	margin: 0;
	font-size: 1rem;
}

.create-form {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 4px 14px;
}

.form-actions {
	display: flex;
	align-items: flex-end;
	padding-top: 22px;
}

@media (max-width: 860px) {
	.page-header,
	.create-form {
		display: grid;
	}
}
</style>

