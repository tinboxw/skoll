<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { ArrowDown, ArrowUp, Delete, Plus, Refresh, Switch, Upload } from "@element-plus/icons-vue";

import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import type { SystemMenuRecord } from "../../stores/navigation";
import { useNavigationStore } from "../../stores/navigation";

type MenuDraft = SystemMenuRecord & {
	rolesText: string;
	permissionsText: string;
	children: MenuDraft[];
};

const { t } = useI18n();
const navigationStore = useNavigationStore();
const saving = ref(false);
const loading = ref(false);
const drafts = ref<MenuDraft[]>([]);

const hasCustomizedMenus = computed(() => navigationStore.customized);
const syncStatus = computed(() => navigationStore.syncStatus);

function createDraft(seed?: Partial<SystemMenuRecord>): MenuDraft {
	return {
		id: seed?.id ?? "",
		label: seed?.label ?? "",
		path: seed?.path ?? "/skoll/",
		icon: seed?.icon ?? "settings",
		order: Number.isFinite(seed?.order) ? Number(seed?.order) : 100,
		visible: seed?.visible !== false,
		requiredRoles: seed?.requiredRoles ?? [],
		requiredPermissions: seed?.requiredPermissions ?? [],
		rolesText: (seed?.requiredRoles ?? []).join(", "),
		permissionsText: (seed?.requiredPermissions ?? []).join(", "),
		children: Array.isArray(seed?.children) ? seed.children.map((item) => createDraft(item)) : []
	};
}

function cloneToDrafts(items: SystemMenuRecord[]): MenuDraft[] {
	return items.map((item) => createDraft(item));
}

function splitCSV(value: string): string[] {
	return value.split(",").map((item) => item.trim()).filter((item) => item !== "");
}

function normalizeDrafts(items: MenuDraft[]): SystemMenuRecord[] {
	return items
		.map((item) => ({
			id: item.id.trim(),
			label: item.label.trim(),
			path: item.path.trim(),
			icon: item.icon?.trim() ?? "",
			order: Number.isFinite(item.order) ? Number(item.order) : 0,
			visible: item.visible !== false,
			requiredRoles: splitCSV(item.rolesText),
			requiredPermissions: splitCSV(item.permissionsText),
			children: normalizeDrafts(item.children ?? [])
		}))
		.filter((item) => item.id !== "" && item.label !== "" && item.path.startsWith("/"));
}

async function loadMenus(): Promise<void> {
	loading.value = true;
	try {
		await navigationStore.loadSystemMenus();
		drafts.value = cloneToDrafts(navigationStore.systemMenus);
	} finally {
		loading.value = false;
	}
}

function addRootMenu(): void {
	const nextOrder = drafts.value.length > 0 ? Math.max(...drafts.value.map((item) => item.order)) + 10 : 10;
	drafts.value.push(createDraft({
		id: `custom-${Date.now()}`,
		label: t("menu.editor.newMenu"),
		path: "/skoll/dashboard",
		icon: "settings",
		order: nextOrder,
		visible: true
	}));
}

function addChildMenu(row: MenuDraft): void {
	row.children.push(createDraft({
		id: `${row.id || "menu"}-child-${Date.now()}`,
		label: t("menu.editor.newChild"),
		path: row.path,
		icon: row.icon || "settings",
		order: row.children.length > 0 ? Math.max(...row.children.map((item) => item.order)) + 10 : 10,
		visible: true
	}));
}

function removeMenu(target: MenuDraft, pool = drafts.value): boolean {
	const index = pool.indexOf(target);
	if (index >= 0) {
		pool.splice(index, 1);
		return true;
	}
	return pool.some((item) => removeMenu(target, item.children));
}

function findSiblingPool(target: MenuDraft, pool = drafts.value): MenuDraft[] | null {
	if (pool.includes(target)) {
		return pool;
	}
	for (const item of pool) {
		const found = findSiblingPool(target, item.children);
		if (found) {
			return found;
		}
	}
	return null;
}

function resequenceOrders(pool: MenuDraft[]): void {
	pool.forEach((item, index) => {
		item.order = (index + 1) * 10;
	});
}

function canMove(row: MenuDraft, direction: -1 | 1): boolean {
	const pool = findSiblingPool(row);
	if (!pool) {
		return false;
	}
	const index = pool.indexOf(row);
	return direction < 0 ? index > 0 : index >= 0 && index < pool.length - 1;
}

function moveMenu(row: MenuDraft, direction: -1 | 1): void {
	const pool = findSiblingPool(row);
	if (!pool) {
		return;
	}
	const index = pool.indexOf(row);
	const nextIndex = index + direction;
	if (index < 0 || nextIndex < 0 || nextIndex >= pool.length) {
		return;
	}
	const [item] = pool.splice(index, 1);
	pool.splice(nextIndex, 0, item);
	resequenceOrders(pool);
}

async function confirmRemove(row: MenuDraft): Promise<void> {
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: t("menu.editor.removeConfirm"),
		confirmText: t("common.delete"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (!confirmed) {
		return;
	}
	removeMenu(row);
}

async function saveMenus(): Promise<void> {
	const items = normalizeDrafts(drafts.value);
	if (items.length === 0) {
		ElMessage.warning(t("menu.editor.emptyWarning"));
		return;
	}
	saving.value = true;
	try {
		await navigationStore.saveSystemMenus(items);
		drafts.value = cloneToDrafts(navigationStore.systemMenus);
		ElMessage.success(t("menu.editor.saveDone"));
	} finally {
		saving.value = false;
	}
}

function resetDrafts(): void {
	drafts.value = cloneToDrafts(navigationStore.systemMenus);
}

onMounted(() => {
	void loadMenus();
});
</script>

<template>
	<section class="menu-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.menus") }}</h2>
				<p>{{ t("menu.editor.desc") }}</p>
			</div>
			<div class="toolbar">
				<el-tag :type="hasCustomizedMenus ? 'success' : 'info'" effect="plain">
					{{ hasCustomizedMenus ? t("menu.editor.customized") : t("menu.editor.defaultSource") }}
				</el-tag>
				<el-button :icon="Refresh" :loading="loading" @click="loadMenus">{{ t("common.refresh") }}</el-button>
				<el-button :icon="Switch" :disabled="saving || loading" @click="resetDrafts">{{ t("common.reset") }}</el-button>
				<el-button type="primary" :icon="Upload" :loading="saving" @click="saveMenus">{{ t("common.save") }}</el-button>
			</div>
		</header>

		<el-alert
			v-if="navigationStore.lastError"
			class="status-alert"
			:title="navigationStore.lastError"
			type="error"
			show-icon
			:closable="false"
		/>

		<div class="table-actions">
			<el-button type="primary" plain :icon="Plus" @click="addRootMenu">{{ t("menu.editor.addRoot") }}</el-button>
			<span>{{ t("menu.editor.status") }}: {{ syncStatus }}</span>
		</div>

		<el-table
			v-loading="loading"
			:data="drafts"
			row-key="id"
			border
			default-expand-all
			:tree-props="{ children: 'children' }"
		>
			<el-table-column :label="t('menu.editor.label')" min-width="170">
				<template #default="{ row }">
					<el-input v-model="row.label" />
				</template>
			</el-table-column>
			<el-table-column :label="t('menu.editor.id')" min-width="150">
				<template #default="{ row }">
					<el-input v-model="row.id" />
				</template>
			</el-table-column>
			<el-table-column :label="t('menu.editor.path')" min-width="210">
				<template #default="{ row }">
					<el-input v-model="row.path" />
				</template>
			</el-table-column>
			<el-table-column :label="t('menu.editor.icon')" width="130">
				<template #default="{ row }">
					<el-select v-model="row.icon">
						<el-option label="dashboard" value="dashboard" />
						<el-option label="users" value="users" />
						<el-option label="roles" value="roles" />
						<el-option label="permissions" value="permissions" />
						<el-option label="menus" value="menus" />
						<el-option label="audit" value="audit" />
						<el-option label="plugins" value="plugins" />
						<el-option label="settings" value="settings" />
					</el-select>
				</template>
			</el-table-column>
			<el-table-column :label="t('menu.editor.order')" width="110">
				<template #default="{ row }">
					<el-input-number v-model="row.order" :min="0" :step="10" controls-position="right" />
				</template>
			</el-table-column>
			<el-table-column :label="t('menu.editor.visible')" width="90" align="center">
				<template #default="{ row }">
					<el-switch v-model="row.visible" />
				</template>
			</el-table-column>
			<el-table-column :label="t('menu.editor.permissions')" min-width="220">
				<template #default="{ row }">
					<el-input v-model="row.permissionsText" placeholder="role.manage, audit.read" />
				</template>
			</el-table-column>
			<el-table-column :label="t('menu.editor.roles')" min-width="160">
				<template #default="{ row }">
					<el-input v-model="row.rolesText" placeholder="admin" />
				</template>
			</el-table-column>
			<el-table-column :label="t('menu.editor.reorder')" width="132" align="center">
				<template #default="{ row }">
					<el-button-group>
						<el-button :icon="ArrowUp" :disabled="!canMove(row, -1)" :title="t('common.moveUp')" @click="moveMenu(row, -1)" />
						<el-button :icon="ArrowDown" :disabled="!canMove(row, 1)" :title="t('common.moveDown')" @click="moveMenu(row, 1)" />
					</el-button-group>
				</template>
			</el-table-column>
			<el-table-column :label="t('common.actions')" width="170" fixed="right">
				<template #default="{ row }">
					<el-button link type="primary" :icon="Plus" @click="addChildMenu(row)">{{ t("common.add") }}</el-button>
					<el-button link type="danger" :icon="Delete" @click="confirmRemove(row)">{{ t("common.delete") }}</el-button>
				</template>
			</el-table-column>
		</el-table>
	</section>
</template>

<style scoped>
.menu-page {
	display: grid;
	gap: 16px;
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

.toolbar,
.table-actions {
	display: flex;
	align-items: center;
	gap: 10px;
	flex-wrap: wrap;
}

.table-actions {
	justify-content: space-between;
}

.status-alert {
	margin-bottom: 0;
}

@media (max-width: 900px) {
	.page-header {
		display: grid;
	}
}
</style>
