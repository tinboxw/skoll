<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import StateBlock from "../../components/Common/StateBlock.vue";
import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import { useButtonAccess } from "../../permissions/button";
import { type ApiResponse, apiGet, apiPut } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type DepartmentRecord = {
	id: string;
	name: string;
	parentId?: string;
	leader?: string;
	status: "enabled" | "disabled";
	order: number;
};

type DepartmentNode = DepartmentRecord & {
	children: DepartmentNode[];
};

type PositionRecord = {
	id: string;
	code: string;
	name: string;
	description?: string;
	status: "enabled" | "disabled";
	order: number;
};

type UserRecord = {
	id: string;
	account: string;
	name: string;
	email: string;
	status: string;
	departmentId: string;
	positionId: string;
};

type ListPayload<T> = {
	items?: T[];
	customized?: boolean;
};

const { t } = useI18n();
const buttonAccess = useButtonAccess();

const loading = ref(false);
const saving = ref(false);
const userLoading = ref(false);
const userSaving = ref(false);
const error = ref("");
const success = ref("");
const userError = ref("");
const userSuccess = ref("");
const departmentsCustomized = ref(false);
const positionsCustomized = ref(false);
const departments = ref<DepartmentRecord[]>([]);
const positions = ref<PositionRecord[]>([]);
const users = ref<UserRecord[]>([]);
const selectedDepartmentID = ref("");
const assignmentDrawerVisible = ref(false);
const assignmentForm = ref<UserRecord | null>(null);
const canGoNext = ref(false);
const userPage = ref(1);
const pageSize = 50;

const canReadOrganization = computed(() => buttonAccess.can("org.read"));
const canManageOrganization = computed(() => buttonAccess.can("org.manage"));
const canUpdateUser = computed(() => buttonAccess.can("user.update"));
const activeDepartments = computed(() => departments.value.filter((item) => item.status === "enabled").length);
const activePositions = computed(() => positions.value.filter((item) => item.status === "enabled").length);
const selectedDepartment = computed(() => departments.value.find((item) => item.id === selectedDepartmentID.value) ?? null);
const selectedDepartmentName = computed(() => selectedDepartment.value?.name ?? t("organization.allDepartments"));
const departmentTree = computed(() => buildDepartmentTree(departments.value));
const usersInSelectedDepartment = computed(() => {
	if (!selectedDepartmentID.value) {
		return users.value;
	}
	return users.value.filter((item) => item.departmentId === selectedDepartmentID.value);
});

function normalizeStatus(value: unknown): "enabled" | "disabled" {
	return value === "disabled" ? "disabled" : "enabled";
}

function normalizeDepartments(items: DepartmentRecord[]): DepartmentRecord[] {
	return items
		.map((item) => ({
			id: String(item.id || "").trim(),
			name: String(item.name || "").trim(),
			parentId: String(item.parentId || "").trim(),
			leader: String(item.leader || "").trim(),
			status: normalizeStatus(item.status),
			order: Number.isFinite(item.order) ? Number(item.order) : 0
		}))
		.filter((item) => item.id !== "" && item.name !== "")
		.sort((a, b) => a.order - b.order || a.id.localeCompare(b.id));
}

function normalizePositions(items: PositionRecord[]): PositionRecord[] {
	return items
		.map((item) => ({
			id: String(item.id || "").trim(),
			code: String(item.code || "").trim(),
			name: String(item.name || "").trim(),
			description: String(item.description || "").trim(),
			status: normalizeStatus(item.status),
			order: Number.isFinite(item.order) ? Number(item.order) : 0
		}))
		.filter((item) => item.id !== "" && item.code !== "" && item.name !== "")
		.sort((a, b) => a.order - b.order || a.code.localeCompare(b.code));
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
		email: String(row.email ?? row.Email ?? "").trim(),
		status: String(row.status ?? row.Status ?? "").trim(),
		departmentId: String(row.departmentId ?? row.DepartmentID ?? "").trim(),
		positionId: String(row.positionId ?? row.PositionID ?? "").trim()
	};
}

function buildDepartmentTree(items: DepartmentRecord[]): DepartmentNode[] {
	const byID = new Map<string, DepartmentNode>();
	for (const item of items) {
		byID.set(item.id, { ...item, children: [] });
	}
	const roots: DepartmentNode[] = [];
	for (const node of byID.values()) {
		if (node.parentId && byID.has(node.parentId) && node.parentId !== node.id) {
			byID.get(node.parentId)?.children.push(node);
		} else {
			roots.push(node);
		}
	}
	const sortNodes = (nodes: DepartmentNode[]): DepartmentNode[] => {
		nodes.sort((a, b) => a.order - b.order || a.id.localeCompare(b.id));
		for (const node of nodes) {
			sortNodes(node.children);
		}
		return nodes;
	};
	return sortNodes(roots);
}

function departmentParentOptions(row: DepartmentRecord): DepartmentRecord[] {
	return departments.value.filter((item) => item.id !== row.id);
}

function labelByID<T extends { id: string; name: string }>(items: T[], id: string): string {
	if (!id) {
		return "-";
	}
	const matched = items.find((item) => item.id === id);
	return matched ? matched.name : id;
}

async function loadOrganization(): Promise<void> {
	if (!canReadOrganization.value) {
		error.value = t("error.forbidden");
		return;
	}
	loading.value = true;
	error.value = "";
	try {
		const [departmentPayload, positionPayload] = await Promise.all([
			apiGet<ApiResponse<ListPayload<DepartmentRecord>>>("/v1/system/departments"),
			apiGet<ApiResponse<ListPayload<PositionRecord>>>("/v1/system/positions")
		]);
		departments.value = normalizeDepartments(departmentPayload.data?.items ?? []);
		positions.value = normalizePositions(positionPayload.data?.items ?? []);
		departmentsCustomized.value = Boolean(departmentPayload.data?.customized);
		positionsCustomized.value = Boolean(positionPayload.data?.customized);
		if (selectedDepartmentID.value && !departments.value.some((item) => item.id === selectedDepartmentID.value)) {
			selectedDepartmentID.value = "";
		}
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

async function loadUsers(targetPage = userPage.value): Promise<void> {
	if (!canReadOrganization.value) {
		return;
	}
	userLoading.value = true;
	userError.value = "";
	try {
		const safePage = Math.max(1, targetPage);
		const offset = (safePage - 1) * pageSize;
		const payload = await apiGet<ApiResponse<unknown[]>>(`/v1/users?offset=${offset}&limit=${pageSize}`);
		users.value = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeUserRecord(item)).filter((item): item is UserRecord => item !== null)
			: [];
		userPage.value = safePage;
		canGoNext.value = users.value.length >= pageSize;
	} catch (e) {
		userError.value = toErrorMessage(e);
		users.value = [];
		canGoNext.value = false;
	} finally {
		userLoading.value = false;
	}
}

async function refreshAll(): Promise<void> {
	await Promise.all([loadOrganization(), loadUsers(userPage.value)]);
}

async function saveOrganization(): Promise<void> {
	if (!canManageOrganization.value) {
		error.value = t("error.forbidden");
		return;
	}
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		const [departmentPayload, positionPayload] = await Promise.all([
			apiPut<ApiResponse<ListPayload<DepartmentRecord>>>("/v1/system/departments", { items: normalizeDepartments(departments.value) }),
			apiPut<ApiResponse<ListPayload<PositionRecord>>>("/v1/system/positions", { items: normalizePositions(positions.value) })
		]);
		departments.value = normalizeDepartments(departmentPayload.data?.items ?? []);
		positions.value = normalizePositions(positionPayload.data?.items ?? []);
		departmentsCustomized.value = Boolean(departmentPayload.data?.customized);
		positionsCustomized.value = Boolean(positionPayload.data?.customized);
		success.value = t("organization.saveDone");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

function addDepartment(parentId = ""): void {
	const next = departments.value.length + 1;
	departments.value.push({
		id: `dept-${next}`,
		name: `Department ${next}`,
		parentId,
		leader: "",
		status: "enabled",
		order: next * 10
	});
}

function addPosition(): void {
	const next = positions.value.length + 1;
	positions.value.push({
		id: `pos-${next}`,
		code: `position_${next}`,
		name: `Position ${next}`,
		description: "",
		status: "enabled",
		order: next * 10
	});
}

async function removeDepartment(index: number): Promise<void> {
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: t("organization.removeDepartmentConfirm"),
		confirmText: t("common.delete"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (!confirmed) {
		return;
	}
	const removed = departments.value[index];
	departments.value.splice(index, 1);
	if (removed) {
		for (const item of departments.value) {
			if (item.parentId === removed.id) {
				item.parentId = "";
			}
		}
		if (selectedDepartmentID.value === removed.id) {
			selectedDepartmentID.value = "";
		}
	}
}

async function removePosition(index: number): Promise<void> {
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: t("organization.removePositionConfirm"),
		confirmText: t("common.delete"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (confirmed) {
		positions.value.splice(index, 1);
	}
}

function selectDepartment(data: unknown): void {
	if (!data || typeof data !== "object") {
		return;
	}
	const row = data as Record<string, unknown>;
	selectedDepartmentID.value = String(row.id ?? "").trim();
}

function clearDepartmentSelection(): void {
	selectedDepartmentID.value = "";
}

function openAssignment(row: UserRecord): void {
	assignmentForm.value = { ...row };
	assignmentDrawerVisible.value = true;
	userError.value = "";
	userSuccess.value = "";
}

async function saveAssignment(): Promise<void> {
	if (!canUpdateUser.value) {
		userError.value = t("error.forbidden");
		return;
	}
	if (!assignmentForm.value) {
		return;
	}
	userSaving.value = true;
	userError.value = "";
	userSuccess.value = "";
	try {
		const form = assignmentForm.value;
		await apiPut<ApiResponse<UserRecord>>(`/v1/users/${form.id}`, {
			name: form.name.trim(),
			email: form.email.trim(),
			status: form.status.trim(),
			departmentId: form.departmentId.trim(),
			positionId: form.positionId.trim()
		});
		userSuccess.value = t("organization.assignmentSaved");
		assignmentDrawerVisible.value = false;
		await loadUsers(userPage.value);
	} catch (e) {
		userError.value = toErrorMessage(e);
	} finally {
		userSaving.value = false;
	}
}

function prevUsersPage(): void {
	if (userLoading.value || userPage.value <= 1) {
		return;
	}
	void loadUsers(userPage.value - 1);
}

function nextUsersPage(): void {
	if (userLoading.value || !canGoNext.value) {
		return;
	}
	void loadUsers(userPage.value + 1);
}

onMounted(() => {
	void refreshAll();
});
</script>

<template>
	<section class="organization-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.organization") }}</h2>
				<p>{{ t("organization.desc") }}</p>
			</div>
			<div class="header-actions">
				<el-button :loading="loading || userLoading" :disabled="saving || userSaving" @click="refreshAll">{{ t("common.refresh") }}</el-button>
				<el-button type="primary" :loading="saving" :disabled="loading || !canManageOrganization" @click="saveOrganization">{{ t("common.save") }}</el-button>
			</div>
		</header>

		<StateBlock v-if="!canReadOrganization" type="forbidden" :title="t('organization.noPermissionTitle')" :description="t('error.forbidden')" />

		<template v-else>
			<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
			<el-alert v-if="success" :title="success" type="success" show-icon :closable="false" />

			<section class="summary-grid">
				<article class="summary-card">
					<span>{{ t("organization.summary.departments") }}</span>
					<strong>{{ departments.length }}</strong>
					<small>{{ t("organization.summary.enabled") }} {{ activeDepartments }}</small>
				</article>
				<article class="summary-card">
					<span>{{ t("organization.summary.positions") }}</span>
					<strong>{{ positions.length }}</strong>
					<small>{{ t("organization.summary.enabled") }} {{ activePositions }}</small>
				</article>
				<article class="summary-card">
					<span>{{ t("organization.summary.users") }}</span>
					<strong>{{ usersInSelectedDepartment.length }}</strong>
					<small>{{ selectedDepartmentName }}</small>
				</article>
			</section>

			<section class="workspace-grid">
				<section class="panel department-tree-panel">
					<div class="section-header">
						<div class="title-row">
							<h3>{{ t("organization.departmentTree") }}</h3>
							<el-tag :type="departmentsCustomized ? 'success' : 'info'" effect="plain">
								{{ departmentsCustomized ? t("menu.editor.customized") : t("menu.editor.defaultSource") }}
							</el-tag>
						</div>
						<div class="inline-actions">
							<el-button text @click="clearDepartmentSelection">{{ t("organization.allDepartments") }}</el-button>
							<el-button type="primary" plain :disabled="!canManageOrganization" @click="addDepartment(selectedDepartmentID)">{{ t("organization.addDepartment") }}</el-button>
						</div>
					</div>
					<el-tree
						v-loading="loading"
						class="department-tree"
						:data="departmentTree"
						node-key="id"
						default-expand-all
						highlight-current
						:props="{ label: 'name', children: 'children' }"
						:empty-text="t('common.empty')"
						@node-click="selectDepartment"
					>
						<template #default="{ data }">
							<div class="tree-node">
								<span>{{ data.name }}</span>
								<el-tag size="small" effect="plain" :type="data.status === 'enabled' ? 'success' : 'info'">{{ data.status }}</el-tag>
							</div>
						</template>
					</el-tree>
				</section>

				<section class="panel">
					<div class="section-header">
						<div class="title-row">
							<h3>{{ t("organization.departmentList") }}</h3>
							<el-tag effect="plain">{{ selectedDepartmentName }}</el-tag>
						</div>
					</div>
					<el-table v-loading="loading" :data="departments" border row-key="id" :empty-text="t('common.empty')">
						<el-table-column :label="t('organization.departmentId')" min-width="150">
							<template #default="{ row }"><el-input v-model="row.id" :disabled="!canManageOrganization" /></template>
						</el-table-column>
						<el-table-column :label="t('table.name')" min-width="160">
							<template #default="{ row }"><el-input v-model="row.name" :disabled="!canManageOrganization" /></template>
						</el-table-column>
						<el-table-column :label="t('organization.parentId')" min-width="170">
							<template #default="{ row }">
								<el-select v-model="row.parentId" clearable filterable :disabled="!canManageOrganization">
									<el-option v-for="item in departmentParentOptions(row)" :key="item.id" :label="item.name" :value="item.id" />
								</el-select>
							</template>
						</el-table-column>
						<el-table-column :label="t('organization.leader')" min-width="140">
							<template #default="{ row }"><el-input v-model="row.leader" :disabled="!canManageOrganization" /></template>
						</el-table-column>
						<el-table-column :label="t('table.status')" width="130">
							<template #default="{ row }">
								<el-select v-model="row.status" :disabled="!canManageOrganization">
									<el-option :label="t('dictionary.enabled')" value="enabled" />
									<el-option :label="t('dictionary.disabled')" value="disabled" />
								</el-select>
							</template>
						</el-table-column>
						<el-table-column :label="t('menu.editor.order')" width="120">
							<template #default="{ row }"><el-input-number v-model="row.order" :step="10" controls-position="right" :disabled="!canManageOrganization" /></template>
						</el-table-column>
						<el-table-column :label="t('table.actions')" width="90" fixed="right">
							<template #default="{ $index }">
								<el-button link type="danger" :disabled="!canManageOrganization" @click="removeDepartment($index)">{{ t("common.delete") }}</el-button>
							</template>
						</el-table-column>
					</el-table>
				</section>
			</section>

			<section class="panel">
				<div class="section-header">
					<div class="title-row">
						<h3>{{ t("organization.positionList") }}</h3>
						<el-tag :type="positionsCustomized ? 'success' : 'info'" effect="plain">
							{{ positionsCustomized ? t("menu.editor.customized") : t("menu.editor.defaultSource") }}
						</el-tag>
					</div>
					<el-button type="primary" plain :disabled="!canManageOrganization" @click="addPosition">{{ t("organization.addPosition") }}</el-button>
				</div>
				<el-table v-loading="loading" :data="positions" border row-key="id" :empty-text="t('common.empty')">
					<el-table-column :label="t('organization.positionId')" min-width="140">
						<template #default="{ row }"><el-input v-model="row.id" :disabled="!canManageOrganization" /></template>
					</el-table-column>
					<el-table-column :label="t('organization.positionCode')" min-width="150">
						<template #default="{ row }"><el-input v-model="row.code" :disabled="!canManageOrganization" /></template>
					</el-table-column>
					<el-table-column :label="t('table.name')" min-width="160">
						<template #default="{ row }"><el-input v-model="row.name" :disabled="!canManageOrganization" /></template>
					</el-table-column>
					<el-table-column :label="t('table.description')" min-width="180">
						<template #default="{ row }"><el-input v-model="row.description" :disabled="!canManageOrganization" /></template>
					</el-table-column>
					<el-table-column :label="t('table.status')" width="130">
						<template #default="{ row }">
							<el-select v-model="row.status" :disabled="!canManageOrganization">
								<el-option :label="t('dictionary.enabled')" value="enabled" />
								<el-option :label="t('dictionary.disabled')" value="disabled" />
							</el-select>
						</template>
					</el-table-column>
					<el-table-column :label="t('menu.editor.order')" width="120">
						<template #default="{ row }"><el-input-number v-model="row.order" :step="10" controls-position="right" :disabled="!canManageOrganization" /></template>
					</el-table-column>
					<el-table-column :label="t('table.actions')" width="90" fixed="right">
						<template #default="{ $index }">
							<el-button link type="danger" :disabled="!canManageOrganization" @click="removePosition($index)">{{ t("common.delete") }}</el-button>
						</template>
					</el-table-column>
				</el-table>
			</section>

			<section class="panel">
				<div class="section-header">
					<div class="title-row">
						<h3>{{ t("organization.userAssignments") }}</h3>
						<el-tag effect="plain">{{ selectedDepartmentName }}</el-tag>
					</div>
					<div class="inline-actions">
						<el-button :disabled="userLoading || userPage <= 1" @click="prevUsersPage">{{ t("common.prev") }}</el-button>
						<span>{{ t("common.page") }} {{ userPage }}</span>
						<el-button :disabled="userLoading || !canGoNext" @click="nextUsersPage">{{ t("common.next") }}</el-button>
					</div>
				</div>
				<el-alert v-if="userError" :title="userError" type="error" show-icon :closable="false" />
				<el-alert v-if="userSuccess" :title="userSuccess" type="success" show-icon :closable="false" />
				<el-table v-loading="userLoading" :data="usersInSelectedDepartment" border row-key="id" :empty-text="t('common.empty')">
					<el-table-column prop="account" :label="t('user.account')" min-width="150" show-overflow-tooltip />
					<el-table-column prop="name" :label="t('table.name')" min-width="150" show-overflow-tooltip />
					<el-table-column :label="t('table.department')" min-width="150" show-overflow-tooltip>
						<template #default="{ row }">{{ labelByID(departments, row.departmentId) }}</template>
					</el-table-column>
					<el-table-column :label="t('table.position')" min-width="150" show-overflow-tooltip>
						<template #default="{ row }">{{ labelByID(positions, row.positionId) }}</template>
					</el-table-column>
					<el-table-column :label="t('table.status')" width="120">
						<template #default="{ row }">
							<el-tag :type="row.status === 'active' ? 'success' : 'info'" effect="plain">{{ row.status || "-" }}</el-tag>
						</template>
					</el-table-column>
					<el-table-column :label="t('table.actions')" width="120" fixed="right">
						<template #default="{ row }">
							<el-button link type="primary" :disabled="!canUpdateUser" @click="openAssignment(row)">{{ t("organization.editAssignment") }}</el-button>
						</template>
					</el-table-column>
				</el-table>
			</section>
		</template>

		<el-drawer v-model="assignmentDrawerVisible" :title="t('organization.editAssignment')" size="420px">
			<el-form v-if="assignmentForm" label-position="top" class="assignment-form" @submit.prevent="saveAssignment">
				<el-form-item :label="t('user.account')">
					<el-input :model-value="assignmentForm.account" disabled />
				</el-form-item>
				<el-form-item :label="t('table.name')">
					<el-input v-model="assignmentForm.name" disabled />
				</el-form-item>
				<el-form-item :label="t('table.email')">
					<el-input v-model="assignmentForm.email" disabled />
				</el-form-item>
				<el-form-item :label="t('table.status')">
					<el-input v-model="assignmentForm.status" disabled />
				</el-form-item>
				<el-form-item :label="t('table.department')">
					<el-select v-model="assignmentForm.departmentId" clearable filterable :disabled="userSaving || !canUpdateUser">
						<el-option v-for="item in departments" :key="item.id" :label="item.name" :value="item.id" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('table.position')">
					<el-select v-model="assignmentForm.positionId" clearable filterable :disabled="userSaving || !canUpdateUser">
						<el-option v-for="item in positions" :key="item.id" :label="item.name" :value="item.id" />
					</el-select>
				</el-form-item>
				<div class="drawer-actions">
					<el-button @click="assignmentDrawerVisible = false">{{ t("common.cancel") }}</el-button>
					<el-button type="primary" native-type="submit" :loading="userSaving" :disabled="!canUpdateUser">{{ t("common.save") }}</el-button>
				</div>
			</el-form>
		</el-drawer>
	</section>
</template>

<style scoped>
.organization-page {
	display: grid;
	gap: 14px;
}

.page-header,
.section-header,
.title-row,
.header-actions,
.inline-actions {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
}

.page-header {
	align-items: flex-start;
}

.page-header h2,
.panel h3 {
	margin: 0;
}

.page-header h2 {
	font-size: 1.35rem;
}

.page-header p {
	margin: 6px 0 0;
	color: var(--color-text-muted);
}

.summary-grid {
	display: grid;
	grid-template-columns: repeat(3, minmax(0, 1fr));
	gap: 12px;
}

.summary-card {
	display: grid;
	gap: 4px;
	padding: 14px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

.summary-card span,
.summary-card small {
	color: var(--color-text-muted);
}

.summary-card strong {
	font-size: 1.35rem;
}

.workspace-grid {
	display: grid;
	grid-template-columns: minmax(260px, 0.7fr) minmax(0, 1.3fr);
	gap: 14px;
	align-items: start;
}

.panel {
	display: grid;
	gap: 12px;
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.department-tree-panel {
	min-height: 340px;
}

.department-tree {
	min-height: 260px;
}

.tree-node {
	display: flex;
	align-items: center;
	justify-content: space-between;
	width: 100%;
	gap: 8px;
	padding-right: 8px;
}

.assignment-form :deep(.el-select) {
	width: 100%;
}

.drawer-actions {
	display: flex;
	justify-content: flex-end;
	gap: 8px;
}

@media (max-width: 1100px) {
	.workspace-grid,
	.summary-grid {
		grid-template-columns: 1fr;
	}
}

@media (max-width: 900px) {
	.page-header,
	.section-header,
	.header-actions,
	.inline-actions {
		display: grid;
		justify-content: stretch;
	}
}
</style>
