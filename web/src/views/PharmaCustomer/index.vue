<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { BadgeAlert, Edit, Plus, RefreshCw, ShieldOff } from "lucide-vue-next";
import { ElMessage } from "element-plus";

import { ConfirmAction, DataTable, DetailDrawer, FilterBar, PageShell, PageToolbar, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import {
	createCustomer,
	disableCustomer,
	listCustomerQualificationReminders,
	listCustomers,
	updateCustomer,
	validateCustomerSalesEligibility,
	type CustomerContact,
	type CustomerQualification,
	type CustomerQualificationReminder,
	type CustomerScope,
	type PharmaCustomer
} from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

type CustomerRow = Record<string, unknown> & PharmaCustomer & {
	contactSummary: string;
	qualificationSummary: string;
	scopeSummary: string;
};

const buttonAccess = useButtonAccess();
const userStore = useUserStore();
const loading = ref(false);
const saving = ref(false);
const checking = ref("");
const error = ref("");
const keyword = ref("");
const statusFilter = ref("");
const regionFilter = ref("");
const customers = ref<PharmaCustomer[]>([]);
const reminders = ref<CustomerQualificationReminder[]>([]);
const drawerOpen = ref(false);
const editing = ref<PharmaCustomer | null>(null);
const disableTarget = ref<PharmaCustomer | null>(null);
const disableDialogOpen = ref(false);
const disableReason = ref("Qualification expired");

const form = reactive({
	code: "",
	name: "",
	region: "East",
	organizationId: "default",
	ownerId: "",
	rating: 3,
	contactName: "",
	contactPhone: "",
	contactEmail: "",
	qualificationName: "Business License",
	qualificationNumber: "",
	qualificationExpiresAt: nextDate(30),
	attachmentFileId: "",
	attachmentFileName: ""
});

const canRead = computed(() => buttonAccess.can("pharma_oa.customer.read"));
const canCreate = computed(() => buttonAccess.can("pharma_oa.customer.create"));
const canUpdate = computed(() => buttonAccess.can("pharma_oa.customer.update"));
const canDisable = computed(() => buttonAccess.can("pharma_oa.customer.disable"));
const canSales = computed(() => buttonAccess.can("pharma_oa.customer.sales"));
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const currentScope = computed<CustomerScope>(() => ({
	ownerId: actorId.value,
	organizationId: form.organizationId.trim() || "default"
}));

const rows = computed<CustomerRow[]>(() => customers.value.map((item) => ({
	...item,
	contactSummary: item.contacts.length > 0 ? item.contacts.map((contact) => `${contact.name} ${contact.phone || contact.email || ""}`.trim()).join(", ") : "-",
	qualificationSummary: item.qualifications.length > 0 ? item.qualifications.map((qualification) => `${qualification.name} ${formatDate(qualification.expiresAt)}`).join(", ") : "-",
	scopeSummary: `${item.organizationId} / ${item.ownerId}`
})));

const filteredRows = computed(() => {
	const q = keyword.value.trim().toLowerCase();
	return rows.value.filter((row) => {
		if (statusFilter.value && row.status !== statusFilter.value) {
			return false;
		}
		if (regionFilter.value && row.region !== regionFilter.value) {
			return false;
		}
		if (!q) {
			return true;
		}
		return [row.code, row.name, row.region, row.organizationId, row.ownerId, row.contactSummary].some((value) => String(value || "").toLowerCase().includes(q));
	});
});

const summary = computed(() => ({
	total: customers.value.length,
	active: customers.value.filter((item) => item.status === "active").length,
	disabled: customers.value.filter((item) => item.status === "disabled").length,
	reminders: reminders.value.length
}));

const columns: DataTableColumn[] = [
	{ key: "code", label: "Code", minWidth: 120 },
	{ key: "name", label: "Customer", minWidth: 180 },
	{ key: "region", label: "Region", width: 120 },
	{ key: "scopeSummary", label: "Org / Owner", minWidth: 180 },
	{ key: "status", label: "Status", width: 110 },
	{ key: "qualificationSummary", label: "Qualifications", minWidth: 220 }
];

onMounted(() => {
	form.ownerId = actorId.value;
	void refresh();
});

async function refresh(): Promise<void> {
	error.value = "";
	if (!canRead.value) {
		return;
	}
	loading.value = true;
	try {
		const scope = currentScope.value;
		const [items, reminderItems] = await Promise.all([
			listCustomers({ scope }),
			listCustomerQualificationReminders(30, scope)
		]);
		customers.value = items;
		reminders.value = reminderItems;
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

function openCreate(): void {
	editing.value = null;
	Object.assign(form, {
		code: "",
		name: "",
		region: "East",
		organizationId: form.organizationId || "default",
		ownerId: actorId.value,
		rating: 3,
		contactName: "",
		contactPhone: "",
		contactEmail: "",
		qualificationName: "Business License",
		qualificationNumber: "",
		qualificationExpiresAt: nextDate(30),
		attachmentFileId: "",
		attachmentFileName: ""
	});
	drawerOpen.value = true;
}

function openEdit(row: CustomerRow): void {
	editing.value = row;
	const contact = row.contacts[0];
	const qualification = row.qualifications[0];
	const attachment = qualification?.attachments?.[0];
	Object.assign(form, {
		code: row.code,
		name: row.name,
		region: row.region,
		organizationId: row.organizationId,
		ownerId: row.ownerId,
		rating: row.rating,
		contactName: contact?.name ?? "",
		contactPhone: contact?.phone ?? "",
		contactEmail: contact?.email ?? "",
		qualificationName: qualification?.name ?? "Business License",
		qualificationNumber: qualification?.number ?? "",
		qualificationExpiresAt: qualification?.expiresAt ? formatInputDate(qualification.expiresAt) : nextDate(30),
		attachmentFileId: attachment?.fileId ?? "",
		attachmentFileName: attachment?.fileName ?? ""
	});
	drawerOpen.value = true;
}

async function saveCustomer(): Promise<void> {
	if (editing.value ? !canUpdate.value : !canCreate.value) {
		error.value = "No permission";
		return;
	}
	saving.value = true;
	error.value = "";
	try {
		const body = buildCustomerRequest();
		const saved = editing.value
			? await updateCustomer(editing.value.id, body)
			: await createCustomer(body);
		upsert(saved);
		drawerOpen.value = false;
		await refreshReminders();
		ElMessage.success("Customer saved");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function refreshReminders(): Promise<void> {
	if (!canRead.value) {
		return;
	}
	reminders.value = await listCustomerQualificationReminders(30, currentScope.value);
}

function openDisable(row: CustomerRow): void {
	disableTarget.value = row;
	disableReason.value = "Qualification expired";
	disableDialogOpen.value = true;
}

async function confirmDisable(): Promise<void> {
	if (!disableTarget.value || !canDisable.value) {
		return;
	}
	saving.value = true;
	error.value = "";
	try {
		const updated = await disableCustomer(disableTarget.value.id, disableReason.value, actorId.value, currentScope.value);
		upsert(updated);
		disableDialogOpen.value = false;
		await refreshReminders();
		ElMessage.success("Customer disabled");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

async function checkSales(row: CustomerRow): Promise<void> {
	if (!canSales.value) {
		return;
	}
	checking.value = row.id;
	error.value = "";
	try {
		const result = await validateCustomerSalesEligibility(row.id, currentScope.value);
		ElMessage[result.allowed ? "success" : "warning"](result.allowed ? "Customer can be used for sales" : result.reason || "Customer is blocked");
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		checking.value = "";
	}
}

function buildCustomerRequest() {
	const contacts: CustomerContact[] = form.contactName.trim()
		? [{
			id: "contact-1",
			name: form.contactName.trim(),
			phone: form.contactPhone.trim(),
			email: form.contactEmail.trim(),
			title: "Primary"
		}]
		: [];
	const qualifications: CustomerQualification[] = form.qualificationName.trim()
		? [{
			id: "qualification-1",
			name: form.qualificationName.trim(),
			number: form.qualificationNumber.trim(),
			expiresAt: form.qualificationExpiresAt,
			attachments: form.attachmentFileId.trim() && form.attachmentFileName.trim()
				? [{ fileId: form.attachmentFileId.trim(), fileName: form.attachmentFileName.trim(), size: 0 }]
				: []
		}]
		: [];
	return {
		code: form.code.trim(),
		name: form.name.trim(),
		region: form.region.trim(),
		organizationId: form.organizationId.trim() || "default",
		ownerId: form.ownerId.trim() || actorId.value,
		rating: Number(form.rating) || 3,
		contacts,
		qualifications,
		actorId: actorId.value,
		scope: currentScope.value
	};
}

function upsert(item: PharmaCustomer): void {
	const index = customers.value.findIndex((current) => current.id === item.id);
	if (index >= 0) {
		customers.value[index] = item;
	} else {
		customers.value.unshift(item);
	}
}

function nextDate(days: number): string {
	const date = new Date();
	date.setDate(date.getDate() + days);
	return formatInputDate(date.toISOString());
}

function formatDate(value: string): string {
	if (!value) {
		return "-";
	}
	return formatInputDate(value);
}

function formatInputDate(value: string): string {
	return value.slice(0, 10);
}
</script>

<template>
	<PageShell
		title="Customer Management"
		description="Maintain Pharma OA customer ownership, regions, contacts, qualification files, and sales eligibility."
		:loading="loading"
		:error="error"
		:no-permission="!canRead"
		no-permission-title="No customer access"
		no-permission-description="Customer records require pharma_oa.customer.read permission."
	>
		<template #actions>
			<el-button :icon="RefreshCw" :loading="loading" @click="refresh">Refresh</el-button>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">New customer</el-button>
		</template>

		<section class="summary-grid">
			<div class="summary-tile"><span>Total</span><strong>{{ summary.total }}</strong></div>
			<div class="summary-tile"><span>Active</span><strong>{{ summary.active }}</strong></div>
			<div class="summary-tile muted"><span>Disabled</span><strong>{{ summary.disabled }}</strong></div>
			<div class="summary-tile risk"><span>Expiring quals</span><strong>{{ summary.reminders }}</strong></div>
		</section>

		<PageToolbar>
			<FilterBar>
				<el-input v-model="keyword" clearable placeholder="Search code, name, region, organization, owner, or contact" />
				<el-select v-model="statusFilter" clearable placeholder="All statuses">
					<el-option label="Active" value="active" />
					<el-option label="Disabled" value="disabled" />
				</el-select>
				<el-select v-model="regionFilter" clearable placeholder="All regions">
					<el-option label="East" value="East" />
					<el-option label="South" value="South" />
					<el-option label="West" value="West" />
					<el-option label="North" value="North" />
				</el-select>
			</FilterBar>
		</PageToolbar>

		<section class="reminder-banner" :class="{ active: reminders.length > 0 }">
			<BadgeAlert :size="18" />
			<span>{{ reminders.length }} customer qualification reminder{{ reminders.length > 1 ? "s" : "" }} due in 30 days.</span>
		</section>

		<DataTable
			:rows="filteredRows"
			:columns="columns"
			row-key="id"
			:loading="loading"
			:error="error"
			:empty-title="keyword || statusFilter || regionFilter ? 'No customers match filters' : 'No customer records yet'"
			empty-description="Create a customer to validate region ownership and sales qualification controls."
		>
			<template #cell-status="{ row }">
				<el-tag :type="row.status === 'active' ? 'success' : 'info'" effect="light">{{ row.status }}</el-tag>
			</template>
			<template #actions="{ row }">
				<el-button link :icon="Edit" :disabled="!canUpdate" aria-label="Edit customer" @click="openEdit(row as CustomerRow)" />
				<el-button link :loading="checking === row.id" :disabled="!canSales" aria-label="Validate sales eligibility" @click="checkSales(row as CustomerRow)">Sales</el-button>
				<el-button link type="danger" :icon="ShieldOff" :disabled="!canDisable || row.status === 'disabled'" aria-label="Disable customer" @click="openDisable(row as CustomerRow)" />
			</template>
		</DataTable>

		<DetailDrawer v-model="drawerOpen" :title="editing ? 'Edit customer' : 'New customer'" size="48%">
			<el-form label-position="top">
				<el-form-item label="Code" required>
					<el-input v-model="form.code" />
				</el-form-item>
				<el-form-item label="Name" required>
					<el-input v-model="form.name" />
				</el-form-item>
				<div class="form-grid">
					<el-form-item label="Region" required>
						<el-select v-model="form.region">
							<el-option label="East" value="East" />
							<el-option label="South" value="South" />
							<el-option label="West" value="West" />
							<el-option label="North" value="North" />
						</el-select>
					</el-form-item>
					<el-form-item label="Rating">
						<el-input-number v-model="form.rating" :min="1" :max="5" />
					</el-form-item>
				</div>
				<div class="form-grid">
					<el-form-item label="Organization" required>
						<el-input v-model="form.organizationId" />
					</el-form-item>
					<el-form-item label="Owner" required>
						<el-input v-model="form.ownerId" />
					</el-form-item>
				</div>
				<div class="form-grid">
					<el-form-item label="Contact" required>
						<el-input v-model="form.contactName" />
					</el-form-item>
					<el-form-item label="Phone">
						<el-input v-model="form.contactPhone" />
					</el-form-item>
				</div>
				<el-form-item label="Email">
					<el-input v-model="form.contactEmail" />
				</el-form-item>
				<div class="form-grid">
					<el-form-item label="Qualification" required>
						<el-input v-model="form.qualificationName" />
					</el-form-item>
					<el-form-item label="Qualification number">
						<el-input v-model="form.qualificationNumber" />
					</el-form-item>
				</div>
				<el-form-item label="Expires at" required>
					<el-date-picker v-model="form.qualificationExpiresAt" type="date" value-format="YYYY-MM-DD" />
				</el-form-item>
				<div class="form-grid">
					<el-form-item label="Attachment file ID">
						<el-input v-model="form.attachmentFileId" />
					</el-form-item>
					<el-form-item label="Attachment file name">
						<el-input v-model="form.attachmentFileName" />
					</el-form-item>
				</div>
			</el-form>
			<template #footer>
				<el-button @click="drawerOpen = false">Cancel</el-button>
				<el-button type="primary" :loading="saving" @click="saveCustomer">Save</el-button>
			</template>
		</DetailDrawer>

		<DetailDrawer v-model="disableDialogOpen" title="Disable customer" size="34%">
			<el-form label-position="top">
				<el-form-item label="Reason">
					<el-input v-model="disableReason" type="textarea" :rows="3" />
				</el-form-item>
			</el-form>
			<template #footer>
				<el-button @click="disableDialogOpen = false">Cancel</el-button>
				<ConfirmAction
					label="Disable"
					message="Disable this customer? Sales eligibility checks will reject it."
					:loading="saving"
					danger
					@confirm="confirmDisable"
				/>
			</template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.summary-grid {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 12px;
}

.summary-tile {
	border: 1px solid var(--color-border);
	border-radius: 8px;
	padding: 12px;
	background: var(--color-surface);
}

.summary-tile span {
	display: block;
	color: var(--color-text-secondary);
	font-size: 0.86rem;
}

.summary-tile strong {
	display: block;
	margin-top: 4px;
	font-size: 1.35rem;
}

.summary-tile.muted strong {
	color: var(--color-text-secondary);
}

.summary-tile.risk strong,
.reminder-banner.active {
	color: var(--el-color-warning);
}

.reminder-banner {
	display: flex;
	align-items: center;
	gap: 8px;
	padding: 10px 12px;
	border: 1px solid var(--color-border);
	border-radius: 8px;
	background: var(--color-surface);
	color: var(--color-text-secondary);
}

.form-grid {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 12px;
}

@media (max-width: 720px) {
	.summary-grid,
	.form-grid {
		grid-template-columns: 1fr;
	}
}
</style>
