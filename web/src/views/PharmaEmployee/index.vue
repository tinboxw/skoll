<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { BadgeAlert, Edit, Plus, RefreshCw, UserMinus } from "lucide-vue-next";
import { ElMessage } from "element-plus";

import { ConfirmAction, DataTable, DetailDrawer, FilterBar, PageShell, PageToolbar, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import {
	createEmployee,
	listEmployeeQualificationReminders,
	listEmployees,
	markEmployeeLeft,
	updateEmployee,
	type EmployeeCertificate,
	type PharmaEmployee,
	type QualificationReminder
} from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

type EmployeeRow = Record<string, unknown> & PharmaEmployee & {
	certificateSummary: string;
};

const buttonAccess = useButtonAccess();
const userStore = useUserStore();
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const keyword = ref("");
const statusFilter = ref("");
const employees = ref<PharmaEmployee[]>([]);
const reminders = ref<QualificationReminder[]>([]);
const drawerOpen = ref(false);
const editing = ref<PharmaEmployee | null>(null);
const leaveTarget = ref<PharmaEmployee | null>(null);
const leaveDialogOpen = ref(false);
const leaveReason = ref("Resigned");

const form = reactive({
	code: "",
	name: "",
	departmentId: "quality",
	positionId: "qa",
	phone: "",
	email: "",
	certificateName: "GSP",
	certificateNumber: "",
	certificateExpiresAt: nextDate(30)
});

const canRead = computed(() => buttonAccess.can("pharma_oa.employee.read"));
const canCreate = computed(() => buttonAccess.can("pharma_oa.employee.create"));
const canUpdate = computed(() => buttonAccess.can("pharma_oa.employee.update"));
const canLeave = computed(() => buttonAccess.can("pharma_oa.employee.leave"));
const actorId = computed(() => userStore.profile?.id?.trim() || "system");

const rows = computed<EmployeeRow[]>(() => employees.value.map((item) => ({
	...item,
	certificateSummary: item.certificates.length > 0 ? item.certificates.map((cert) => `${cert.name} ${formatDate(cert.expiresAt)}`).join(", ") : "-"
})));

const filteredRows = computed(() => {
	const q = keyword.value.trim().toLowerCase();
	return rows.value.filter((row) => {
		if (statusFilter.value && row.status !== statusFilter.value) {
			return false;
		}
		if (!q) {
			return true;
		}
		return [row.code, row.name, row.departmentId, row.positionId, row.email].some((value) => String(value || "").toLowerCase().includes(q));
	});
});

const summary = computed(() => ({
	total: employees.value.length,
	active: employees.value.filter((item) => item.status === "active").length,
	left: employees.value.filter((item) => item.status === "left").length,
	reminders: reminders.value.length
}));

const columns: DataTableColumn[] = [
	{ key: "code", label: "Code", minWidth: 120 },
	{ key: "name", label: "Name", minWidth: 150 },
	{ key: "departmentId", label: "Department", minWidth: 140 },
	{ key: "positionId", label: "Position", minWidth: 140 },
	{ key: "status", label: "Status", width: 110 },
	{ key: "certificateSummary", label: "Certificates", minWidth: 220 }
];

onMounted(() => {
	void refresh();
});

async function refresh(): Promise<void> {
	error.value = "";
	if (!canRead.value) {
		return;
	}
	loading.value = true;
	try {
		const [items, reminderItems] = await Promise.all([
			listEmployees(),
			listEmployeeQualificationReminders(30)
		]);
		employees.value = items;
		reminders.value = reminderItems;
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

function openCreate(): void {
	editing.value = null;
	resetForm();
	drawerOpen.value = true;
}

function openEdit(row: EmployeeRow): void {
	const item = employees.value.find((employee) => employee.id === row.id);
	if (!item) {
		return;
	}
	editing.value = item;
	form.code = item.code;
	form.name = item.name;
	form.departmentId = item.departmentId;
	form.positionId = item.positionId;
	form.phone = item.phone;
	form.email = item.email;
	const cert = item.certificates[0];
	form.certificateName = cert?.name || "GSP";
	form.certificateNumber = cert?.number || "";
	form.certificateExpiresAt = cert?.expiresAt ? cert.expiresAt.slice(0, 10) : nextDate(30);
	drawerOpen.value = true;
}

function openLeave(row: EmployeeRow): void {
	const item = employees.value.find((employee) => employee.id === row.id);
	if (!item) {
		return;
	}
	leaveTarget.value = item;
	leaveReason.value = "Resigned";
	leaveDialogOpen.value = true;
}

async function saveEmployee(): Promise<void> {
	if (form.code.trim() === "" || form.name.trim() === "" || form.departmentId.trim() === "" || form.positionId.trim() === "") {
		error.value = "Code, name, department, and position are required.";
		return;
	}
	if (editing.value && !canUpdate.value) {
		error.value = "You do not have permission to update employees.";
		return;
	}
	if (!editing.value && !canCreate.value) {
		error.value = "You do not have permission to create employees.";
		return;
	}
	saving.value = true;
	error.value = "";
	try {
		const body = {
			code: form.code.trim(),
			name: form.name.trim(),
			departmentId: form.departmentId.trim(),
			positionId: form.positionId.trim(),
			phone: form.phone.trim(),
			email: form.email.trim(),
			certificates: buildCertificates(),
			actorId: actorId.value
		};
		const saved = editing.value
			? await updateEmployee(editing.value.id, body)
			: await createEmployee(body);
		upsert(saved);
		drawerOpen.value = false;
		ElMessage.success(editing.value ? "Employee updated" : "Employee created");
		await refreshReminders();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function leaveEmployee(): Promise<void> {
	const target = leaveTarget.value;
	if (!target) {
		return;
	}
	if (!canLeave.value) {
		error.value = "You do not have permission to mark employees left.";
		return;
	}
	saving.value = true;
	error.value = "";
	try {
		const saved = await markEmployeeLeft(target.id, leaveReason.value, actorId.value);
		upsert(saved);
		leaveTarget.value = null;
		leaveDialogOpen.value = false;
		ElMessage.success("Employee status updated");
		await refreshReminders();
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function refreshReminders(): Promise<void> {
	try {
		reminders.value = await listEmployeeQualificationReminders(30);
	} catch {
		reminders.value = [];
	}
}

function upsert(item: PharmaEmployee): void {
	employees.value = [item, ...employees.value.filter((employee) => employee.id !== item.id)]
		.sort((a, b) => a.code.localeCompare(b.code));
}

function buildCertificates(): EmployeeCertificate[] {
	if (form.certificateName.trim() === "") {
		return [];
	}
	return [{
		id: "cert-primary",
		name: form.certificateName.trim(),
		number: form.certificateNumber.trim(),
		expiresAt: new Date(`${form.certificateExpiresAt}T00:00:00Z`).toISOString()
	}];
}

function resetForm(): void {
	form.code = "";
	form.name = "";
	form.departmentId = "quality";
	form.positionId = "qa";
	form.phone = "";
	form.email = "";
	form.certificateName = "GSP";
	form.certificateNumber = "";
	form.certificateExpiresAt = nextDate(30);
}

function statusType(status: unknown): "success" | "warning" | "info" {
	if (status === "active") {
		return "success";
	}
	if (status === "on_leave") {
		return "warning";
	}
	return "info";
}

function nextDate(days: number): string {
	const date = new Date();
	date.setDate(date.getDate() + days);
	return date.toISOString().slice(0, 10);
}

function formatDate(value: string): string {
	if (!value) {
		return "-";
	}
	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString();
}
</script>

<template>
	<PageShell
		title="Employee Management"
		description="Maintain Pharma OA employee records, departments, positions, certificates, and leave status."
		:loading="loading"
		:error="error"
		:forbidden="!canRead"
		forbidden-title="Employee management unavailable"
		forbidden-description="Ask an administrator for pharma_oa.employee.read permission."
		data-testid="pharma-employee-page"
	>
		<template #actions>
			<el-button :icon="RefreshCw" :loading="loading" @click="refresh">Refresh</el-button>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">New employee</el-button>
		</template>
		<template #stateActions>
			<el-button :icon="RefreshCw" @click="refresh">Retry</el-button>
		</template>

		<section class="summary-grid" aria-label="Employee summary">
			<div class="summary-tile"><span>Total</span><strong>{{ summary.total }}</strong></div>
			<div class="summary-tile"><span>Active</span><strong>{{ summary.active }}</strong></div>
			<div class="summary-tile"><span>Left</span><strong>{{ summary.left }}</strong></div>
			<div class="summary-tile risk"><span>Expiring certs</span><strong>{{ summary.reminders }}</strong></div>
		</section>

		<PageToolbar>
			<FilterBar>
				<el-input v-model="keyword" clearable placeholder="Search code, name, department, position, or email" />
				<el-select v-model="statusFilter" clearable placeholder="All statuses">
					<el-option label="Active" value="active" />
					<el-option label="On leave" value="on_leave" />
					<el-option label="Left" value="left" />
				</el-select>
			</FilterBar>
		</PageToolbar>

		<section v-if="reminders.length > 0" class="reminder-strip" data-testid="employee-reminders">
			<BadgeAlert aria-hidden="true" />
			<span>{{ reminders.length }} certificate reminder{{ reminders.length > 1 ? "s" : "" }} due in 30 days.</span>
		</section>

		<DataTable
			:rows="filteredRows"
			:columns="columns"
			row-key="id"
			:loading="loading"
			:error="error"
			empty-text="No employees match the current filters"
		>
			<template #cell-status="{ value }">
				<el-tag :type="statusType(value)">{{ value }}</el-tag>
			</template>
			<template #actions="{ row }">
				<el-tooltip content="Edit">
					<el-button :icon="Edit" circle :disabled="!canUpdate || row.status === 'left'" @click="openEdit(row)" />
				</el-tooltip>
				<el-tooltip content="Mark left">
					<el-button :icon="UserMinus" circle type="danger" :disabled="!canLeave || row.status === 'left'" @click="openLeave(row)" />
				</el-tooltip>
			</template>
		</DataTable>

		<DetailDrawer v-model="drawerOpen" :title="editing ? 'Edit employee' : 'New employee'" size="46%">
			<el-form label-position="top">
				<el-form-item label="Code" required>
					<el-input v-model="form.code" />
				</el-form-item>
				<el-form-item label="Name" required>
					<el-input v-model="form.name" />
				</el-form-item>
				<div class="form-grid">
					<el-form-item label="Department" required>
						<el-input v-model="form.departmentId" />
					</el-form-item>
					<el-form-item label="Position" required>
						<el-input v-model="form.positionId" />
					</el-form-item>
				</div>
				<div class="form-grid">
					<el-form-item label="Phone">
						<el-input v-model="form.phone" />
					</el-form-item>
					<el-form-item label="Email">
						<el-input v-model="form.email" />
					</el-form-item>
				</div>
				<div class="form-grid">
					<el-form-item label="Certificate">
						<el-input v-model="form.certificateName" />
					</el-form-item>
					<el-form-item label="Certificate no.">
						<el-input v-model="form.certificateNumber" />
					</el-form-item>
				</div>
				<el-form-item label="Expires at">
					<el-date-picker v-model="form.certificateExpiresAt" value-format="YYYY-MM-DD" type="date" />
				</el-form-item>
			</el-form>
			<div class="drawer-actions">
				<el-button @click="drawerOpen = false">Cancel</el-button>
				<el-button type="primary" :loading="saving" @click="saveEmployee">Save</el-button>
			</div>
		</DetailDrawer>

		<el-dialog v-model="leaveDialogOpen" title="Mark employee left" width="420px" @closed="leaveTarget = null">
			<el-form label-position="top">
				<el-form-item label="Employee">
					<el-input :model-value="leaveTarget ? `${leaveTarget.code} / ${leaveTarget.name}` : ''" disabled />
				</el-form-item>
				<el-form-item label="Reason">
					<el-input v-model="leaveReason" type="textarea" :rows="3" />
				</el-form-item>
			</el-form>
			<template #footer>
				<el-button @click="leaveDialogOpen = false">Cancel</el-button>
				<ConfirmAction
					label="Mark left"
					message="Mark this employee as left? This removes certificate reminders for the employee."
					:loading="saving"
					@confirm="leaveEmployee"
				/>
			</template>
		</el-dialog>
	</PageShell>
</template>

<style scoped>
.summary-grid {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 12px;
}

.summary-tile {
	background: var(--color-surface);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	display: grid;
	gap: 8px;
	min-height: 82px;
	padding: 14px;
}

.summary-tile span {
	color: var(--color-text-muted);
	font-size: 0.86rem;
}

.summary-tile strong {
	font-size: 1.4rem;
}

.summary-tile.risk {
	border-color: #f4c16a;
}

.reminder-strip {
	align-items: center;
	background: #fff8e6;
	border: 1px solid #f4d58a;
	border-radius: var(--radius-md);
	color: #7a4f00;
	display: flex;
	gap: 10px;
	padding: 12px 14px;
}

.reminder-strip svg {
	height: 18px;
	width: 18px;
}

.form-grid {
	display: grid;
	gap: 12px;
	grid-template-columns: repeat(2, minmax(0, 1fr));
}

.drawer-actions {
	display: flex;
	justify-content: flex-end;
	gap: 8px;
}

@media (max-width: 760px) {
	.summary-grid,
	.form-grid {
		grid-template-columns: 1fr;
	}

	:deep(.el-dialog) {
		width: 92% !important;
	}
}
</style>
