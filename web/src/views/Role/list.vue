<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import { BUTTON_ACCESS, useButtonAccess } from "../../permissions/button";
import { type ApiResponse, apiDelete, apiGet, apiPost } from "../../utils/api";
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

const { t } = useI18n();
const buttonAccess = useButtonAccess();
const loading = ref(false);
const operating = ref(false);
const error = ref("");
const rows = ref<RoleRecord[]>([]);
const page = ref(1);
const pageSize = 10;
const canGoNext = ref(false);
const creating = ref(false);
const createError = ref("");
const createSuccess = ref("");
const createName = ref("");
const createKey = ref("");
const createDescription = ref("");

const canCreateRole = computed(() => buttonAccess.can(BUTTON_ACCESS.roleCreate));
const canUpdateRole = computed(() => buttonAccess.can(BUTTON_ACCESS.roleUpdate));
const canDeleteRole = computed(() => buttonAccess.can(BUTTON_ACCESS.roleDelete));

async function loadRoles(targetPage = page.value): Promise<void> {
	loading.value = true;
	error.value = "";
	try {
		const safePage = Math.max(1, targetPage);
		const offset = (safePage - 1) * pageSize;
		const payload = await apiGet<ApiResponse<RoleRecord[]>>(`/v1/roles?offset=${offset}&limit=${pageSize}`);
		const list = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeRoleRecord(item)).filter((item): item is RoleRecord => item !== null)
			: [];
		rows.value = list;
		page.value = safePage;
		canGoNext.value = list.length >= pageSize;
	} catch (e) {
		error.value = toErrorMessage(e);
		rows.value = [];
		canGoNext.value = false;
	} finally {
		loading.value = false;
	}
}

function prevPage(): void {
	if (loading.value || page.value <= 1) {
		return;
	}
	void loadRoles(page.value - 1);
}

function nextPage(): void {
	if (loading.value || !canGoNext.value) {
		return;
	}
	void loadRoles(page.value + 1);
}

async function createRole(): Promise<void> {
	if (!canCreateRole.value) {
		createError.value = t("error.forbidden");
		return;
	}
	createError.value = "";
	createSuccess.value = "";
	creating.value = true;
	try {
		await apiPost<ApiResponse<RoleRecord>>("/v1/roles", {
			name: createName.value.trim(),
			key: createKey.value.trim(),
			description: createDescription.value.trim(),
			permissions: [],
			builtIn: false
		});
		createSuccess.value = t("role.createDone");
		createName.value = "";
		createKey.value = "";
		createDescription.value = "";
		await loadRoles(1);
	} catch (e) {
		createError.value = toErrorMessage(e);
	} finally {
		creating.value = false;
	}
}

async function deleteRole(roleID: string): Promise<void> {
	if (!canDeleteRole.value) {
		error.value = t("error.forbidden");
		return;
	}
	if (operating.value || loading.value) {
		return;
	}
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: `${t("common.delete")}: ${roleID}`,
		confirmText: t("common.delete"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (!confirmed) {
		return;
	}
	operating.value = true;
	error.value = "";
	try {
		await apiDelete<ApiResponse<unknown>>(`/v1/roles/${roleID}`);
		await loadRoles(page.value);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		operating.value = false;
	}
}

onMounted(() => {
	void loadRoles(1);
});
</script>

<template>
	<section class="role-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.roles") }}</h2>
				<p>{{ t("permission.matrixHint") }}</p>
			</div>
			<el-button :loading="loading" :disabled="operating" @click="loadRoles(page)">{{ t("common.refresh") }}</el-button>
		</header>

		<section v-if="canCreateRole" class="create-panel">
			<h3>{{ t("role.createTitle") }}</h3>
			<el-form class="create-form" @submit.prevent="createRole">
				<el-input v-model="createName" :placeholder="t('role.createNamePlaceholder')" :disabled="creating" />
				<el-input v-model="createKey" :placeholder="t('role.createKeyPlaceholder')" :disabled="creating" />
				<el-input v-model="createDescription" :placeholder="t('role.createDescPlaceholder')" :disabled="creating" />
				<el-button type="primary" native-type="submit" :loading="creating" :disabled="creating || createName.trim() === '' || createKey.trim() === ''">{{ t("role.create") }}</el-button>
			</el-form>
			<el-alert v-if="createError" :title="createError" type="error" show-icon :closable="false" />
			<el-alert v-if="createSuccess" :title="createSuccess" type="success" show-icon :closable="false" />
		</section>
		<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

		<el-table v-loading="loading" :data="rows" border row-key="id" :empty-text="t('common.empty')">
			<el-table-column prop="id" :label="t('table.id')" min-width="190" show-overflow-tooltip />
			<el-table-column prop="name" :label="t('table.name')" min-width="160" />
			<el-table-column :label="t('table.key')" min-width="150">
				<template #default="{ row }">
					<el-tag effect="plain">{{ row.key || "-" }}</el-tag>
				</template>
			</el-table-column>
			<el-table-column :label="t('table.permissions')" width="120" align="center">
				<template #default="{ row }">{{ Array.isArray(row.permissions) ? row.permissions.length : 0 }}</template>
			</el-table-column>
			<el-table-column :label="t('table.actions')" width="170" fixed="right">
				<template #default="{ row }">
					<el-button v-if="canUpdateRole" link type="primary" tag="router-link" :to="{ name: 'role-edit', params: { id: row.id } }">{{ t("common.edit") }}</el-button>
					<el-button v-if="canDeleteRole" link type="danger" :disabled="loading || operating" @click="deleteRole(row.id)">{{ t("common.delete") }}</el-button>
					<span v-if="!canUpdateRole && !canDeleteRole" class="muted">{{ t("common.empty") }}</span>
				</template>
			</el-table-column>
		</el-table>

		<div class="pager">
			<el-button :disabled="loading || operating || page <= 1" @click="prevPage">{{ t("common.prev") }}</el-button>
			<span>{{ t("common.page") }} {{ page }}</span>
			<el-button :disabled="loading || operating || !canGoNext" @click="nextPage">{{ t("common.next") }}</el-button>
		</div>
	</section>
</template>

<style scoped>
.role-page {
	display: grid;
	gap: 14px;
}

.page-header {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
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
	gap: 10px;
	padding: 14px;
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
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 8px;
}

.pager {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 8px;
}

.muted {
	color: var(--color-text-muted);
}

@media (max-width: 900px) {
	.page-header,
	.pager {
		display: grid;
		justify-content: stretch;
	}

	.create-form {
		grid-template-columns: 1fr;
	}
}
</style>

