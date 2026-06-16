<script setup lang="ts">
import { onMounted, ref } from "vue";

import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
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

type PositionRecord = {
	id: string;
	code: string;
	name: string;
	description?: string;
	status: "enabled" | "disabled";
	order: number;
};

type ListPayload<T> = {
	items?: T[];
	customized?: boolean;
};

const { t } = useI18n();

const loading = ref(false);
const saving = ref(false);
const error = ref("");
const success = ref("");
const departmentsCustomized = ref(false);
const positionsCustomized = ref(false);
const departments = ref<DepartmentRecord[]>([]);
const positions = ref<PositionRecord[]>([]);

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

async function loadOrganization(): Promise<void> {
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
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

async function saveOrganization(): Promise<void> {
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

function addDepartment(): void {
	const next = departments.value.length + 1;
	departments.value.push({
		id: `dept-${next}`,
		name: `Department ${next}`,
		parentId: "",
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
	if (confirmed) {
		departments.value.splice(index, 1);
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

onMounted(() => {
	void loadOrganization();
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
				<el-button :loading="loading" :disabled="saving" @click="loadOrganization">{{ t("common.refresh") }}</el-button>
				<el-button type="primary" :loading="saving" :disabled="loading" @click="saveOrganization">{{ t("common.save") }}</el-button>
			</div>
		</header>

		<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
		<el-alert v-if="success" :title="success" type="success" show-icon :closable="false" />

		<section class="panel">
			<div class="section-header">
				<div class="title-row">
					<h3>{{ t("organization.departmentList") }}</h3>
					<el-tag :type="departmentsCustomized ? 'success' : 'info'" effect="plain">
						{{ departmentsCustomized ? t("menu.editor.customized") : t("menu.editor.defaultSource") }}
					</el-tag>
				</div>
				<el-button type="primary" plain @click="addDepartment">{{ t("organization.addDepartment") }}</el-button>
			</div>
			<el-table v-loading="loading" :data="departments" border row-key="id" :empty-text="t('common.empty')">
				<el-table-column :label="t('organization.departmentId')" min-width="150">
					<template #default="{ row }"><el-input v-model="row.id" /></template>
				</el-table-column>
				<el-table-column :label="t('table.name')" min-width="160">
					<template #default="{ row }"><el-input v-model="row.name" /></template>
				</el-table-column>
				<el-table-column :label="t('organization.parentId')" min-width="150">
					<template #default="{ row }"><el-input v-model="row.parentId" /></template>
				</el-table-column>
				<el-table-column :label="t('organization.leader')" min-width="140">
					<template #default="{ row }"><el-input v-model="row.leader" /></template>
				</el-table-column>
				<el-table-column :label="t('table.status')" width="130">
					<template #default="{ row }">
						<el-select v-model="row.status">
							<el-option :label="t('dictionary.enabled')" value="enabled" />
							<el-option :label="t('dictionary.disabled')" value="disabled" />
						</el-select>
					</template>
				</el-table-column>
				<el-table-column :label="t('menu.editor.order')" width="120">
					<template #default="{ row }"><el-input-number v-model="row.order" :step="10" controls-position="right" /></template>
				</el-table-column>
				<el-table-column :label="t('table.actions')" width="90" fixed="right">
					<template #default="{ $index }">
						<el-button link type="danger" @click="removeDepartment($index)">{{ t("common.delete") }}</el-button>
					</template>
				</el-table-column>
			</el-table>
		</section>

		<section class="panel">
			<div class="section-header">
				<div class="title-row">
					<h3>{{ t("organization.positionList") }}</h3>
					<el-tag :type="positionsCustomized ? 'success' : 'info'" effect="plain">
						{{ positionsCustomized ? t("menu.editor.customized") : t("menu.editor.defaultSource") }}
					</el-tag>
				</div>
				<el-button type="primary" plain @click="addPosition">{{ t("organization.addPosition") }}</el-button>
			</div>
			<el-table v-loading="loading" :data="positions" border row-key="id" :empty-text="t('common.empty')">
				<el-table-column :label="t('organization.positionId')" min-width="140">
					<template #default="{ row }"><el-input v-model="row.id" /></template>
				</el-table-column>
				<el-table-column :label="t('organization.positionCode')" min-width="150">
					<template #default="{ row }"><el-input v-model="row.code" /></template>
				</el-table-column>
				<el-table-column :label="t('table.name')" min-width="160">
					<template #default="{ row }"><el-input v-model="row.name" /></template>
				</el-table-column>
				<el-table-column :label="t('table.description')" min-width="180">
					<template #default="{ row }"><el-input v-model="row.description" /></template>
				</el-table-column>
				<el-table-column :label="t('table.status')" width="130">
					<template #default="{ row }">
						<el-select v-model="row.status">
							<el-option :label="t('dictionary.enabled')" value="enabled" />
							<el-option :label="t('dictionary.disabled')" value="disabled" />
						</el-select>
					</template>
				</el-table-column>
				<el-table-column :label="t('menu.editor.order')" width="120">
					<template #default="{ row }"><el-input-number v-model="row.order" :step="10" controls-position="right" /></template>
				</el-table-column>
				<el-table-column :label="t('table.actions')" width="90" fixed="right">
					<template #default="{ $index }">
						<el-button link type="danger" @click="removePosition($index)">{{ t("common.delete") }}</el-button>
					</template>
				</el-table-column>
			</el-table>
		</section>
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
.header-actions {
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

.panel {
	display: grid;
	gap: 12px;
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

@media (max-width: 900px) {
	.page-header,
	.section-header,
	.header-actions {
		display: grid;
		justify-content: stretch;
	}
}
</style>
