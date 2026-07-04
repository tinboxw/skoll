<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { Bell, Check, ExternalLink, Mail, RefreshCw, RotateCcw } from "lucide-vue-next";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";

import { ConfirmAction, DataTable, FilterBar, PageShell, PageToolbar, type DataTableColumn } from "../../components/Common";
import {
	loadNotifications,
	markNotificationDone,
	saveNotifications,
	seedNotificationDemo,
	type NotificationCategory,
	type NotificationItem,
	type NotificationStatus
} from "../../notifications/types";
import { useButtonAccess } from "../../permissions/button";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

type NotificationRow = Record<string, unknown> & {
	id: string;
	title: string;
	category: string;
	status: string;
	target: string;
	updatedAt: string;
	raw: NotificationItem;
};

type TabMode = "pending" | "done" | "message" | "reminder";

const router = useRouter();
const userStore = useUserStore();
const buttonAccess = useButtonAccess();
const loading = ref(false);
const error = ref("");
const keyword = ref("");
const activeTab = ref<TabMode>("pending");
const items = ref<NotificationItem[]>([]);

const currentActorId = computed(() => userStore.profile?.id?.trim() || "starter-1");
const canRead = computed(() => buttonAccess.can("notification.read"));
const canAct = computed(() => buttonAccess.can("notification.act"));

const columns: DataTableColumn[] = [
	{ key: "title", label: "Title", minWidth: 220 },
	{ key: "category", label: "Type", width: 110 },
	{ key: "status", label: "Status", width: 110 },
	{ key: "target", label: "Target", minWidth: 180 },
	{ key: "updatedAt", label: "Updated", minWidth: 160 }
];

const rows = computed<NotificationRow[]>(() => items.value.map(toRow));
const filteredRows = computed<NotificationRow[]>(() => {
	const q = keyword.value.trim().toLowerCase();
	return rows.value.filter((row) => {
		if (!matchesTab(row.raw, activeTab.value)) {
			return false;
		}
		if (!q) {
			return true;
		}
		return [row.title, row.category, row.status, row.target].some((value) => value.toLowerCase().includes(q));
	});
});
const summary = computed(() => ({
	pending: items.value.filter((item) => item.status === "pending" && item.category === "todo").length,
	done: items.value.filter((item) => item.status === "done").length,
	message: items.value.filter((item) => item.category === "message").length,
	reminder: items.value.filter((item) => item.category === "reminder").length
}));

onMounted(() => {
	refreshItems();
});

function refreshItems(): void {
	error.value = "";
	if (!canRead.value) {
		return;
	}
	loading.value = true;
	try {
		items.value = loadNotifications().filter((item) => item.actorId === currentActorId.value);
	} catch (e) {
		error.value = toErrorMessage(e);
		items.value = [];
	} finally {
		loading.value = false;
	}
}

function seedDemo(): void {
	seedNotificationDemo(currentActorId.value);
	refreshItems();
	ElMessage.success("Demo notifications added");
}

function clearItems(): void {
	const remaining = loadNotifications().filter((item) => item.actorId !== currentActorId.value);
	saveNotifications(remaining);
	refreshItems();
	ElMessage.success("Notifications cleared");
}

function markDone(row: NotificationRow): void {
	if (!canAct.value) {
		error.value = "You do not have permission to update notifications.";
		return;
	}
	const updated = markNotificationDone(row.id, currentActorId.value);
	if (!updated) {
		error.value = "Notification item was not found.";
		return;
	}
	refreshItems();
	ElMessage.success(updated.category === "message" ? "Message read" : "Todo completed");
}

async function jumpToTarget(row: NotificationRow): Promise<void> {
	const path = row.raw.target.path.trim();
	if (!path) {
		error.value = "Notification target path is empty.";
		return;
	}
	await router.push(path);
}

function matchesTab(item: NotificationItem, tab: TabMode): boolean {
	if (tab === "pending") {
		return item.category === "todo" && item.status === "pending";
	}
	if (tab === "done") {
		return item.status === "done" || item.status === "read";
	}
	return item.category === tab;
}

function toRow(item: NotificationItem): NotificationRow {
	return {
		id: item.id,
		title: item.title,
		category: item.category,
		status: item.status,
		target: `${item.target.type || "target"} / ${item.target.id || "-"}`,
		updatedAt: formatDate(item.updatedAt),
		raw: item
	};
}

function statusType(status: string): "success" | "warning" | "info" {
	if (status === "done" || status === "read") {
		return "success";
	}
	return status === "pending" ? "warning" : "info";
}

function categoryIcon(category: string) {
	if (category === "message") {
		return Mail;
	}
	return category === "reminder" ? Bell : Check;
}

function formatDate(value: string): string {
	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
</script>

<template>
	<PageShell
		title="Todo Center"
		description="Approval todos, completed items, in-app messages, and business reminders."
		:loading="loading"
		:error="error"
		:forbidden="!canRead"
		forbidden-title="Todo center unavailable"
		forbidden-description="Ask an administrator for notification.read permission."
		data-testid="todo-center-page"
	>
		<template #actions>
			<el-button :icon="RefreshCw" :loading="loading" @click="refreshItems">Refresh</el-button>
			<el-button :icon="Bell" @click="seedDemo">Demo reminder</el-button>
			<ConfirmAction label="Clear" message="Clear your notification center items?" @confirm="clearItems" />
		</template>
		<template #stateActions>
			<el-button :icon="RefreshCw" @click="refreshItems">Retry</el-button>
		</template>

		<section class="todo-summary" aria-label="Todo center summary">
			<button class="summary-tile" :class="{ active: activeTab === 'pending' }" type="button" @click="activeTab = 'pending'">
				<span>Pending</span>
				<strong>{{ summary.pending }}</strong>
			</button>
			<button class="summary-tile" :class="{ active: activeTab === 'done' }" type="button" @click="activeTab = 'done'">
				<span>Done</span>
				<strong>{{ summary.done }}</strong>
			</button>
			<button class="summary-tile" :class="{ active: activeTab === 'message' }" type="button" @click="activeTab = 'message'">
				<span>Messages</span>
				<strong>{{ summary.message }}</strong>
			</button>
			<button class="summary-tile" :class="{ active: activeTab === 'reminder' }" type="button" @click="activeTab = 'reminder'">
				<span>Reminders</span>
				<strong>{{ summary.reminder }}</strong>
			</button>
		</section>

		<PageToolbar>
			<FilterBar>
				<el-input v-model="keyword" clearable placeholder="Search title, type, status, or target" />
			</FilterBar>
		</PageToolbar>

		<DataTable
			:rows="filteredRows"
			:columns="columns"
			row-key="id"
			:loading="loading"
			empty-text="No items in this view"
			data-testid="todo-table"
		>
			<template #cell-title="{ row }">
				<div class="todo-title">
					<component :is="categoryIcon(row.category)" class="cell-icon" aria-hidden="true" />
					<div>
						<strong>{{ row.title }}</strong>
						<p>{{ row.raw.body }}</p>
					</div>
				</div>
			</template>
			<template #cell-status="{ value }">
				<el-tag :type="statusType(value)">{{ value }}</el-tag>
			</template>
			<template #cell-category="{ value }">
				<el-tag>{{ value }}</el-tag>
			</template>
			<template #actions="{ row }">
				<el-tooltip content="Open target">
					<el-button :icon="ExternalLink" circle aria-label="Open target" @click="jumpToTarget(row)" />
				</el-tooltip>
				<el-tooltip content="Mark done">
					<el-button :icon="Check" circle type="success" aria-label="Mark done" :disabled="!canAct || row.raw.status !== 'pending'" @click="markDone(row)" />
				</el-tooltip>
			</template>
		</DataTable>
	</PageShell>
</template>

<style scoped>
.todo-summary {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 12px;
}

.summary-tile {
	display: grid;
	gap: 6px;
	min-height: 84px;
	padding: 14px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
	color: var(--color-text);
	text-align: left;
	cursor: pointer;
}

.summary-tile strong {
	font-size: 1.35rem;
}

.summary-tile.active {
	border-color: var(--color-primary);
	box-shadow: inset 0 0 0 1px var(--color-primary);
}

.todo-title {
	display: flex;
	align-items: flex-start;
	gap: 8px;
	min-width: 0;
}

.todo-title strong,
.todo-title p {
	display: block;
	max-width: 100%;
	margin: 0;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.todo-title p {
	color: var(--color-text-muted);
	font-size: 0.84rem;
}

.cell-icon {
	width: 16px;
	height: 16px;
	color: var(--color-primary);
	flex-shrink: 0;
	margin-top: 2px;
}

@media (max-width: 760px) {
	.todo-summary {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
}
</style>
