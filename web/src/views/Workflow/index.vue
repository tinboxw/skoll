<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { Check, ClipboardList, Eye, GitPullRequest, Plus, RefreshCw, Send, X } from "lucide-vue-next";
import { ElMessage } from "element-plus";

import { ConfirmAction, DataTable, DetailDrawer, FilterBar, PageShell, PageToolbar, type DataTableColumn } from "../../components/Common";
import { loadFormSchemas, type FormSchema } from "../../form-builder/types";
import { completeWorkflowNotifications, createWorkflowTodo, upsertNotification } from "../../notifications/types";
import { useButtonAccess } from "../../permissions/button";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";
import {
	approveWorkflowTask,
	copyWorkflowTask,
	createWorkflowDefinition,
	getWorkflowInstance,
	publishWorkflowDefinition,
	rejectWorkflowTask,
	startWorkflowInstance,
	transferWorkflowTask,
	withdrawWorkflowInstance,
	type WorkflowActor,
	type WorkflowDefinitionRequest,
	type WorkflowInstance,
	type WorkflowTask
} from "../../workflow/api";

type WorkflowRow = Record<string, unknown> & {
	id: string;
	title: string;
	status: string;
	business: string;
	starter: string;
	pendingAssignee: string;
	updatedAt: string;
	raw: WorkflowInstance;
};

type ViewMode = "pending" | "approved" | "initiated" | "copied";

const DEMO_DEFINITION_ID = "pharma-oa-demo-approval";
const STORAGE_KEY = "skoll.workflow.instanceIds";

const userStore = useUserStore();
const buttonAccess = useButtonAccess();
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const keyword = ref("");
const activeView = ref<ViewMode>("pending");
const instances = ref<WorkflowInstance[]>([]);
const selected = ref<WorkflowInstance | null>(null);
const drawerOpen = ref(false);
const launchOpen = ref(false);
const actionOpen = ref(false);
const actionMode = ref<"approve" | "reject" | "transfer" | "copy">("approve");
const actionInstance = ref<WorkflowInstance | null>(null);
const actionTask = ref<WorkflowTask | null>(null);
const formSchemas = ref<FormSchema[]>([]);

const launchForm = reactive({
	title: "Purchase approval",
	businessType: "oa.purchase",
	businessId: "purchase-demo-001",
	formSchemaId: ""
});

const actionForm = reactive({
	comment: "",
	targetId: "approver-2",
	targetName: "Approver 2"
});

const canRead = computed(() => buttonAccess.can("workflow.instance.read"));
const canStart = computed(() => buttonAccess.can("workflow.instance.start"));
const canAct = computed(() => buttonAccess.can("workflow.task.act"));
const currentActor = computed<WorkflowActor>(() => ({
	id: userStore.profile?.id?.trim() || "starter-1",
	name: userStore.profile?.name?.trim() || "Current User"
}));
const selectedLaunchSchema = computed(() => formSchemas.value.find((item) => item.id === launchForm.formSchemaId) || null);

const columns: DataTableColumn[] = [
	{ key: "title", label: "Title", minWidth: 190 },
	{ key: "business", label: "Business", minWidth: 150 },
	{ key: "status", label: "Status", width: 120 },
	{ key: "starter", label: "Starter", minWidth: 130 },
	{ key: "pendingAssignee", label: "Pending", minWidth: 130 },
	{ key: "updatedAt", label: "Updated", minWidth: 160 }
];

const rows = computed<WorkflowRow[]>(() => instances.value.map(toRow));
const filteredRows = computed<WorkflowRow[]>(() => {
	const q = keyword.value.trim().toLowerCase();
	return rows.value.filter((row) => {
		if (!matchesView(row.raw, activeView.value)) {
			return false;
		}
		if (!q) {
			return true;
		}
		return [row.title, row.business, row.status, row.starter, row.pendingAssignee].some((value) => value.toLowerCase().includes(q));
	});
});
const summary = computed(() => ({
	pending: rows.value.filter((row) => matchesView(row.raw, "pending")).length,
	approved: rows.value.filter((row) => matchesView(row.raw, "approved")).length,
	initiated: rows.value.filter((row) => matchesView(row.raw, "initiated")).length,
	copied: rows.value.filter((row) => matchesView(row.raw, "copied")).length
}));

onMounted(() => {
	loadAvailableForms();
	void refreshInstances();
});

async function refreshInstances(): Promise<void> {
	error.value = "";
	if (!canRead.value) {
		return;
	}
	loading.value = true;
	try {
		const ids = loadInstanceIds();
		const loaded = await Promise.allSettled(ids.map((id) => getWorkflowInstance(id)));
		const fulfilled = loaded
			.filter((item): item is PromiseFulfilledResult<WorkflowInstance> => item.status === "fulfilled")
			.map((item) => item.value);
		if (ids.length > 0 && fulfilled.length === 0) {
			error.value = "Workflow instances failed to refresh.";
			instances.value = [];
			return;
		}
		instances.value = fulfilled.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

async function launchWorkflow(): Promise<void> {
	error.value = "";
	if (!canStart.value) {
		error.value = "You do not have permission to start workflow instances.";
		return;
	}
	if (launchForm.title.trim() === "" || launchForm.businessType.trim() === "" || launchForm.businessId.trim() === "") {
		error.value = "Title, business type, and business ID are required.";
		return;
	}
	saving.value = true;
	try {
		await ensureDemoDefinition();
		const instance = await startWorkflowInstance({
			id: createID("wf"),
			definitionId: DEMO_DEFINITION_ID,
			businessType: launchForm.businessType.trim(),
			businessId: launchForm.businessId.trim(),
			title: launchForm.title.trim(),
			starter: currentActor.value
		});
		upsertInstance(instance);
		createWorkflowTodo({
			instanceId: instance.id,
			title: `Approve ${instance.title}`,
			actorId: currentActor.value.id,
			body: `${instance.businessType} / ${instance.businessId}`
		});
		launchOpen.value = false;
		activeView.value = "initiated";
		ElMessage.success("Workflow launched");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function submitAction(): Promise<void> {
	const instance = actionInstance.value;
	const task = actionTask.value;
	if (!instance || !task) {
		return;
	}
	if (!canAct.value) {
		error.value = "You do not have permission to act on workflow tasks.";
		return;
	}
	saving.value = true;
	error.value = "";
	try {
		const actor = currentActor.value;
		let updated: WorkflowInstance;
		if (actionMode.value === "approve") {
			updated = await approveWorkflowTask(instance.id, task.id, { actor, comment: actionForm.comment });
		} else if (actionMode.value === "reject") {
			updated = await rejectWorkflowTask(instance.id, task.id, { actor, comment: actionForm.comment });
		} else if (actionMode.value === "transfer") {
			updated = await transferWorkflowTask(instance.id, task.id, { actor, target: targetActor(), comment: actionForm.comment });
			createWorkflowTodo({
				instanceId: updated.id,
				title: `Transferred ${updated.title}`,
				actorId: actionForm.targetId.trim(),
				body: actionForm.comment || "Workflow task transferred to you."
			});
		} else {
			updated = await copyWorkflowTask(instance.id, task.id, { actor, target: targetActor(), comment: actionForm.comment });
			upsertNotification({
				id: `message-workflow-${updated.id}-${actionForm.targetId.trim()}`,
				category: "message",
				title: `Copied: ${updated.title}`,
				body: actionForm.comment || "Workflow task copied to you.",
				actorId: actionForm.targetId.trim(),
				target: { type: "workflow", id: updated.id, path: `/skoll/workflow?instance=${encodeURIComponent(updated.id)}` }
			});
		}
		if (actionMode.value === "approve" || actionMode.value === "reject" || actionMode.value === "transfer") {
			completeWorkflowNotifications(updated.id, actor.id);
		}
		upsertInstance(updated);
		actionOpen.value = false;
		selected.value = updated;
		ElMessage.success("Workflow updated");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function withdraw(row: WorkflowRow): Promise<void> {
	if (!canAct.value) {
		error.value = "You do not have permission to withdraw workflow instances.";
		return;
	}
	saving.value = true;
	try {
		const updated = await withdrawWorkflowInstance(row.id, { actor: currentActor.value, comment: "Withdrawn from workflow console" });
		upsertInstance(updated);
		ElMessage.success("Workflow withdrawn");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

function openTimeline(row: WorkflowRow): void {
	selected.value = row.raw;
	drawerOpen.value = true;
}

function loadAvailableForms(): void {
	try {
		formSchemas.value = loadFormSchemas();
	} catch {
		formSchemas.value = [];
	}
}

function applyLaunchSchema(): void {
	const schema = selectedLaunchSchema.value;
	if (!schema) {
		return;
	}
	launchForm.title = `${schema.name} approval`;
	launchForm.businessType = schema.businessType;
}

function openAction(mode: "approve" | "reject" | "transfer" | "copy", row: WorkflowRow): void {
	const task = row.raw.tasks.find((item) => item.status === "pending" && item.assignee.id === currentActor.value.id);
	if (!task) {
		error.value = "No pending task is assigned to you.";
		return;
	}
	actionMode.value = mode;
	actionInstance.value = row.raw;
	actionTask.value = task;
	actionForm.comment = "";
	actionForm.targetId = "approver-2";
	actionForm.targetName = "Approver 2";
	actionOpen.value = true;
}

function toRow(instance: WorkflowInstance): WorkflowRow {
	const pending = instance.tasks.filter((item) => item.status === "pending").map((item) => item.assignee.name || item.assignee.id).join(", ");
	return {
		id: instance.id,
		title: instance.title,
		status: instance.status,
		business: `${instance.businessType} / ${instance.businessId}`,
		starter: instance.starter.name || instance.starter.id,
		pendingAssignee: pending || "-",
		updatedAt: formatDate(instance.updatedAt),
		raw: instance
	};
}

function matchesView(instance: WorkflowInstance, mode: ViewMode): boolean {
	const actorID = currentActor.value.id;
	if (mode === "pending") {
		return instance.tasks.some((task) => task.status === "pending" && task.assignee.id === actorID);
	}
	if (mode === "approved") {
		return instance.status === "approved";
	}
	if (mode === "initiated") {
		return instance.starter.id === actorID;
	}
	return instance.tasks.some((task) => task.status === "copied" && task.assignee.id === actorID);
}

function upsertInstance(instance: WorkflowInstance): void {
	const next = instances.value.filter((item) => item.id !== instance.id);
	instances.value = [instance, ...next].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
	saveInstanceIds(instances.value.map((item) => item.id));
}

async function ensureDemoDefinition(): Promise<void> {
	const definition: WorkflowDefinitionRequest = {
		id: DEMO_DEFINITION_ID,
		key: "pharma.oa.demo.approval",
		name: "Pharma OA Demo Approval",
		version: 1,
		nodes: [
			{ id: "start", key: "start", name: "Start", type: "start" },
			{ id: "approval", key: "approval", name: "Approval", type: "approval", assignees: [currentActor.value.id] },
			{ id: "end", key: "end", name: "End", type: "end" }
		],
		transitions: [
			{ from: "start", to: "approval" },
			{ from: "approval", to: "end" }
		]
	};
	await createWorkflowDefinition(definition);
	await publishWorkflowDefinition(DEMO_DEFINITION_ID);
}

function targetActor(): WorkflowActor {
	const id = actionForm.targetId.trim();
	return { id, name: actionForm.targetName.trim() || id };
}

function loadInstanceIds(): string[] {
	try {
		const parsed = JSON.parse(localStorage.getItem(STORAGE_KEY) || "[]") as unknown;
		return Array.isArray(parsed) ? parsed.filter((item): item is string => typeof item === "string" && item.trim() !== "") : [];
	} catch {
		return [];
	}
}

function saveInstanceIds(ids: string[]): void {
	localStorage.setItem(STORAGE_KEY, JSON.stringify([...new Set(ids)].slice(0, 80)));
}

function createID(prefix: string): string {
	const random = typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;
	return `${prefix}-${random}`;
}

function formatDate(value: string): string {
	if (!value) {
		return "-";
	}
	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
</script>

<template>
	<PageShell
		title="Workflow"
		description="Pending approvals, launched instances, copied tasks, and completed approvals."
		:loading="loading"
		:error="error"
		:forbidden="!canRead"
		forbidden-title="Workflow unavailable"
		forbidden-description="Ask an administrator for workflow.instance.read permission."
		data-testid="workflow-page"
	>
		<template #actions>
			<el-button :loading="loading" :icon="RefreshCw" @click="refreshInstances">Refresh</el-button>
			<el-button type="primary" :icon="Plus" :disabled="!canStart" @click="launchOpen = true">Launch</el-button>
		</template>
		<template #stateActions>
			<el-button :icon="RefreshCw" @click="refreshInstances">Retry</el-button>
		</template>

		<el-segmented
			v-model="activeView"
			class="workflow-summary"
			aria-label="Workflow summary"
			:options="[
				{ label: `Pending ${summary.pending}`, value: 'pending' },
				{ label: `Approved ${summary.approved}`, value: 'approved' },
				{ label: `Initiated ${summary.initiated}`, value: 'initiated' },
				{ label: `Copied ${summary.copied}`, value: 'copied' }
			]"
		/>

		<PageToolbar>
			<FilterBar>
				<el-input v-model="keyword" clearable placeholder="Search title, business, starter, or assignee" />
			</FilterBar>
		</PageToolbar>

		<DataTable
			:rows="filteredRows"
			:columns="columns"
			row-key="id"
			:loading="loading"
			:error="error"
			empty-text="No workflow instances in this view"
			data-testid="workflow-table"
		>
			<template #cell-title="{ row }">
				<div class="title-cell">
					<ClipboardList class="cell-icon" aria-hidden="true" />
					<span>{{ row.title }}</span>
				</div>
			</template>
			<template #cell-status="{ value }">
				<el-tag :type="value === 'approved' ? 'success' : value === 'rejected' ? 'danger' : value === 'withdrawn' ? 'info' : 'warning'">
					{{ value }}
				</el-tag>
			</template>
			<template #actions="{ row }">
				<el-tooltip content="Timeline">
					<el-button :icon="Eye" circle @click="openTimeline(row)" />
				</el-tooltip>
				<el-tooltip content="Approve">
					<el-button :icon="Check" circle type="success" :disabled="!canAct || row.raw.status !== 'running'" @click="openAction('approve', row)" />
				</el-tooltip>
				<el-tooltip content="Reject">
					<el-button :icon="X" circle type="danger" :disabled="!canAct || row.raw.status !== 'running'" @click="openAction('reject', row)" />
				</el-tooltip>
			</template>
		</DataTable>

		<el-dialog v-model="launchOpen" title="Launch workflow" width="520px" class="workflow-dialog">
			<el-form label-position="top">
				<el-form-item label="Form schema">
					<el-select
						v-model="launchForm.formSchemaId"
						clearable
						placeholder="Select saved form"
						@change="applyLaunchSchema"
					>
						<el-option
							v-for="schema in formSchemas"
							:key="schema.id"
							:label="`${schema.name} / v${schema.version}`"
							:value="schema.id"
						/>
					</el-select>
				</el-form-item>
				<el-form-item label="Title">
					<el-input v-model="launchForm.title" />
				</el-form-item>
				<el-form-item label="Business type">
					<el-input v-model="launchForm.businessType" />
				</el-form-item>
				<el-form-item label="Business ID">
					<el-input v-model="launchForm.businessId" />
				</el-form-item>
			</el-form>
			<div v-if="selectedLaunchSchema" class="launch-form-preview" data-testid="workflow-form-preview">
				<div class="launch-form-preview__head">
					<strong>{{ selectedLaunchSchema.name }}</strong>
					<el-tag>v{{ selectedLaunchSchema.version }}</el-tag>
				</div>
				<div class="launch-form-preview__fields">
					<el-tag v-for="field in selectedLaunchSchema.fields" :key="field.key" :type="field.required ? 'danger' : 'info'">
						{{ field.label }}
					</el-tag>
				</div>
			</div>
			<template #footer>
				<el-button @click="launchOpen = false">Cancel</el-button>
				<el-button type="primary" :loading="saving" :icon="Send" @click="launchWorkflow">Launch</el-button>
			</template>
		</el-dialog>

		<el-dialog v-model="actionOpen" :title="`${actionMode} task`" width="520px" class="workflow-dialog">
			<el-form label-position="top">
				<el-form-item v-if="actionMode === 'transfer' || actionMode === 'copy'" label="Target user ID">
					<el-input v-model="actionForm.targetId" />
				</el-form-item>
				<el-form-item v-if="actionMode === 'transfer' || actionMode === 'copy'" label="Target name">
					<el-input v-model="actionForm.targetName" />
				</el-form-item>
				<el-form-item label="Comment">
					<el-input v-model="actionForm.comment" type="textarea" :rows="3" />
				</el-form-item>
			</el-form>
			<template #footer>
				<el-button @click="actionOpen = false">Cancel</el-button>
				<el-button v-if="actionMode === 'approve'" type="success" :loading="saving" :icon="Check" @click="submitAction">Approve</el-button>
				<ConfirmAction
					v-else-if="actionMode === 'reject'"
					label="Reject"
					message="Reject this workflow task?"
					:loading="saving"
					@confirm="submitAction"
				/>
				<el-button v-else type="primary" :loading="saving" :icon="GitPullRequest" @click="submitAction">{{ actionMode }}</el-button>
			</template>
		</el-dialog>

		<DetailDrawer v-model="drawerOpen" title="Workflow timeline" size="52%">
			<template v-if="selected">
				<div class="drawer-head">
					<h3>{{ selected.title }}</h3>
					<el-tag>{{ selected.status }}</el-tag>
				</div>
				<el-timeline>
					<el-timeline-item
						v-for="item in selected.timeline"
						:key="item.id"
						:timestamp="formatDate(item.createdAt)"
						:type="item.type === 'reject' ? 'danger' : item.type === 'approve' ? 'success' : 'primary'"
					>
						<strong>{{ item.type }}</strong>
						<p>{{ item.actor.name || item.actor.id }} <span v-if="item.target?.id">-> {{ item.target.name || item.target.id }}</span></p>
						<p v-if="item.comment">{{ item.comment }}</p>
					</el-timeline-item>
				</el-timeline>
				<div class="drawer-actions">
					<el-button :icon="GitPullRequest" :disabled="!canAct || selected.status !== 'running'" @click="openAction('transfer', toRow(selected))">Transfer</el-button>
					<el-button :icon="Send" :disabled="!canAct || selected.status !== 'running'" @click="openAction('copy', toRow(selected))">Copy</el-button>
					<ConfirmAction
						label="Withdraw"
						message="Withdraw this workflow instance?"
						:disabled="!canAct || selected.status !== 'running'"
						:loading="saving"
						@confirm="withdraw(toRow(selected))"
					/>
				</div>
			</template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.workflow-summary {
	width: 100%;
}

.title-cell,
.drawer-head,
.drawer-actions {
	display: flex;
	align-items: center;
	gap: 8px;
	min-width: 0;
}

.title-cell span {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.cell-icon {
	width: 16px;
	height: 16px;
	color: var(--color-primary);
	flex-shrink: 0;
}

.drawer-head {
	justify-content: space-between;
}

.drawer-head h3 {
	margin: 0;
	font-size: 1rem;
}

.drawer-actions {
	justify-content: flex-end;
	flex-wrap: wrap;
}

.launch-form-preview {
	display: grid;
	gap: 10px;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-bg);
}

.launch-form-preview__head,
.launch-form-preview__fields {
	display: flex;
	align-items: center;
	gap: 8px;
	flex-wrap: wrap;
}

.launch-form-preview__head {
	justify-content: space-between;
}

@media (max-width: 760px) {
	.workflow-summary :deep(.el-segmented__group) {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	:deep(.workflow-dialog) {
		width: 92% !important;
	}
}
</style>
