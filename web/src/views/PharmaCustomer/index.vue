<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { BadgeAlert, Edit, Plus, RefreshCw, ShieldOff } from "lucide-vue-next";
import { ElMessage } from "element-plus";

import { ConfirmAction, DataScopeIndicator, DataTable, DetailDrawer, FilterBar, PageShell, PageToolbar, StateBlock, type DataTableColumn } from "../../components/Common";
import { useI18n } from "../../i18n";
import { useButtonAccess } from "../../permissions/button";
import { resolveAuthorizedDataScope, type AuthorizedDataScopeDecision } from "../../permissions/data-scope";
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
const { t } = useI18n();
const loading = ref(false);
const hasLoaded = ref(false);
const saving = ref(false);
const scopeLoading = ref(false);
const writeScopeLoading = ref(false);
const checking = ref("");
const error = ref("");
const scopeError = ref("");
const writeScopeError = ref("");
const keyword = ref("");
const statusFilter = ref("");
const regionFilter = ref("");
const customers = ref<PharmaCustomer[]>([]);
const reminders = ref<CustomerQualificationReminder[]>([]);
const currentPage = ref(1);
const pageSize = ref(50);
const total = ref(0);
const drawerOpen = ref(false);
const editing = ref<PharmaCustomer | null>(null);
const disableTarget = ref<PharmaCustomer | null>(null);
const disableDialogOpen = ref(false);
const disableReason = ref("");
const readScope = ref<AuthorizedDataScopeDecision | null>(null);
const writeScope = ref<AuthorizedDataScopeDecision | null>(null);
const writeScopeAction = ref<"create" | "update">("create");
let loadController: AbortController | null = null;
let filterTimer: ReturnType<typeof setTimeout> | null = null;

const form = reactive({
	code: "",
	name: "",
	region: "East",
	organizationId: "",
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
const profileOrganizationId = computed(() => userStore.profile?.organizationId?.trim() || "");
const allowedOrganizationIDs = computed(() => writeScope.value?.departmentIds ?? []);
const organizationRestricted = computed(() => writeScope.value?.scope === "department" || writeScope.value?.scope === "department_tree");
const ownerRestricted = computed(() => writeScope.value?.scope === "self");
const canSaveTarget = computed(() => {
	const decision = writeScope.value;
	if (!decision) return false;
	if (decision.scope === "all") return true;
	if (decision.scope === "self") return form.ownerId.trim() === actorId.value;
	return decision.departmentIds.includes(form.organizationId.trim());
});
const canSaveCustomer = computed(() => {
	const allowed = editing.value ? canUpdate.value : canCreate.value;
	return allowed && canSaveTarget.value && Boolean(form.code.trim() && form.name.trim() && form.region.trim() && form.organizationId.trim() && form.ownerId.trim());
});

const rows = computed<CustomerRow[]>(() => customers.value.map((item) => ({
	...item,
	contactSummary: item.contacts.length > 0 ? item.contacts.map((contact) => `${contact.name} ${contact.phone || contact.email || ""}`.trim()).join(", ") : "-",
	qualificationSummary: item.qualifications.length > 0 ? item.qualifications.map((qualification) => `${qualification.name} ${formatDate(qualification.expiresAt)}`).join(", ") : "-",
	scopeSummary: `${item.organizationId} / ${item.ownerId}`
})));

const summary = computed(() => ({
	total: total.value,
	active: customers.value.filter((item) => item.status === "active").length,
	disabled: customers.value.filter((item) => item.status === "disabled").length,
	reminders: reminders.value.length
}));

const columns = computed<DataTableColumn[]>(() => [
	{ key: "code", label: t("customer.code"), minWidth: 120 },
	{ key: "name", label: t("customer.name"), minWidth: 180 },
	{ key: "region", label: t("customer.region"), width: 120 },
	{ key: "scopeSummary", label: t("customer.organizationOwner"), minWidth: 180 },
	{ key: "status", label: t("customer.status"), width: 110 },
	{ key: "qualificationSummary", label: t("customer.qualifications"), minWidth: 220 }
]);

onMounted(() => {
	form.ownerId = actorId.value;
	form.organizationId = profileOrganizationId.value;
	disableReason.value = t("customer.disable.defaultReason");
	void loadPage();
	void refreshReminders();
});

onBeforeUnmount(() => {
	loadController?.abort();
	if (filterTimer) clearTimeout(filterTimer);
});

watch([keyword, statusFilter, regionFilter], () => {
	currentPage.value = 1;
	if (filterTimer) clearTimeout(filterTimer);
	filterTimer = setTimeout(() => void loadPage(), 250);
});

async function refresh(): Promise<void> {
	currentPage.value = 1;
	readScope.value = null;
	await Promise.all([loadPage(true), refreshReminders()]);
}

async function loadPage(force = false): Promise<void> {
	error.value = "";
	scopeError.value = "";
	if (!canRead.value) {
		return;
	}
	loadController?.abort();
	const controller = new AbortController();
	loadController = controller;
	loading.value = true;
	scopeLoading.value = readScope.value === null;
	try {
		const decision = readScope.value ?? await resolveAuthorizedDataScope("pharma_oa.customer", "read");
		const page = await listCustomers({
			keyword: keyword.value,
			status: statusFilter.value,
			region: regionFilter.value,
			offset: (currentPage.value - 1) * pageSize.value,
			limit: pageSize.value,
			signal: controller.signal,
			force
		});
		if (loadController !== controller) return;
		readScope.value = decision;
		customers.value = page.items;
		total.value = page.total;
		hasLoaded.value = true;
	} catch (e) {
		if (e instanceof DOMException && e.name === "AbortError") return;
		error.value = toErrorMessage(e);
		scopeError.value = error.value;
	} finally {
		if (loadController === controller) {
			loadController = null;
			loading.value = false;
			scopeLoading.value = false;
		}
	}
}

function changePage(page: number): void {
	currentPage.value = page;
	void loadPage();
}

function changePageSize(size: number): void {
	pageSize.value = size;
	currentPage.value = 1;
	void loadPage();
}

function openCreate(): void {
	editing.value = null;
	Object.assign(form, {
		code: "",
		name: "",
		region: "East",
		organizationId: profileOrganizationId.value,
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
	void loadWriteScope("create");
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
	void loadWriteScope("update");
}

async function loadWriteScope(action: "create" | "update"): Promise<void> {
	writeScopeAction.value = action;
	writeScopeLoading.value = true;
	writeScopeError.value = "";
	writeScope.value = null;
	try {
		const decision = await resolveAuthorizedDataScope("pharma_oa.customer", action);
		writeScope.value = decision;
		if (decision.scope === "self") {
			form.ownerId = actorId.value;
			if (!editing.value && profileOrganizationId.value) form.organizationId = profileOrganizationId.value;
		} else if ((decision.scope === "department" || decision.scope === "department_tree") && !decision.departmentIds.includes(form.organizationId.trim())) {
			form.organizationId = decision.departmentIds[0] ?? "";
		}
	} catch (cause) {
		writeScopeError.value = toErrorMessage(cause);
	} finally {
		writeScopeLoading.value = false;
	}
}

async function saveCustomer(): Promise<void> {
	if (!canSaveCustomer.value) {
		error.value = canSaveTarget.value ? t("error.forbidden") : t("dataScope.targetDenied");
		return;
	}
	saving.value = true;
	error.value = "";
	try {
		const body = buildCustomerRequest();
		if (editing.value) await updateCustomer(editing.value.id, body);
		else await createCustomer(body);
		await loadPage(true);
		drawerOpen.value = false;
		await refreshReminders();
		ElMessage.success(t("customer.saved"));
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
	reminders.value = await listCustomerQualificationReminders(30);
}

function openDisable(row: CustomerRow): void {
	disableTarget.value = row;
	disableReason.value = t("customer.disable.defaultReason");
	disableDialogOpen.value = true;
}

async function confirmDisable(): Promise<void> {
	if (!disableTarget.value || !canDisable.value) {
		return;
	}
	saving.value = true;
	error.value = "";
	try {
		await disableCustomer(disableTarget.value.id, disableReason.value);
		await loadPage(true);
		disableDialogOpen.value = false;
		await refreshReminders();
		ElMessage.success(t("customer.disabled"));
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
		const result = await validateCustomerSalesEligibility(row.id);
		ElMessage[result.allowed ? "success" : "warning"](result.allowed ? t("customer.sales.allowed") : result.reason || t("customer.sales.blocked"));
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
		organizationId: form.organizationId.trim(),
		ownerId: form.ownerId.trim() || actorId.value,
		rating: Number(form.rating) || 3,
		contacts,
		qualifications
	};
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
			:title="t('customer.title')"
			:description="t('customer.description')"
			:loading="loading && !hasLoaded"
			:error="error"
			:forbidden="!canRead"
			:forbidden-title="t('customer.noPermissionTitle')"
			:forbidden-description="t('customer.noPermissionDescription')"
		>
			<template #actions>
				<el-button :icon="RefreshCw" :loading="loading" @click="refresh">{{ t("common.refresh") }}</el-button>
				<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">{{ t("customer.new") }}</el-button>
			</template>
			<template v-if="canRead" #stateActions>
				<el-button type="primary" :loading="loading" @click="refresh">{{ t("common.retry") }}</el-button>
			</template>

			<DataScopeIndicator :decision="readScope" :loading="scopeLoading" :error="scopeError" @retry="refresh" />

			<section class="summary-grid">
				<div class="summary-tile"><span>{{ t("customer.summary.total") }}</span><strong>{{ summary.total }}</strong></div>
				<div class="summary-tile"><span>{{ t("customer.summary.pageActive") }}</span><strong>{{ summary.active }}</strong></div>
				<div class="summary-tile muted"><span>{{ t("customer.summary.pageDisabled") }}</span><strong>{{ summary.disabled }}</strong></div>
				<div class="summary-tile risk"><span>{{ t("customer.summary.expiring") }}</span><strong>{{ summary.reminders }}</strong></div>
			</section>

		<PageToolbar>
			<FilterBar>
					<el-input v-model="keyword" clearable :aria-label="t('customer.filter.keyword')" :placeholder="t('customer.filter.keyword')" />
					<el-select v-model="statusFilter" clearable :aria-label="t('customer.filter.allStatuses')" :placeholder="t('customer.filter.allStatuses')">
						<el-option :label="t('customer.status.active')" value="active" />
						<el-option :label="t('customer.status.disabled')" value="disabled" />
					</el-select>
					<el-select v-model="regionFilter" clearable :aria-label="t('customer.filter.allRegions')" :placeholder="t('customer.filter.allRegions')">
						<el-option :label="t('customer.region.east')" value="East" />
						<el-option :label="t('customer.region.south')" value="South" />
						<el-option :label="t('customer.region.west')" value="West" />
						<el-option :label="t('customer.region.north')" value="North" />
					</el-select>
			</FilterBar>
		</PageToolbar>

		<section class="reminder-banner" :class="{ active: reminders.length > 0 }">
			<BadgeAlert :size="18" />
				<span>{{ reminders.length }} {{ t("customer.reminder") }}</span>
		</section>

		<DataTable
			:rows="rows"
			:columns="columns"
				row-key="id"
				:loading="loading"
				:error="error"
				:height="520"
				virtualized
				:empty-text="keyword || statusFilter || regionFilter ? t('customer.empty.filtered') : t('customer.empty.title')"
		>
			<template #cell-status="{ row }">
					<el-tag :type="row.status === 'active' ? 'success' : 'info'" effect="light">{{ t(`customer.status.${row.status}`) }}</el-tag>
			</template>
			<template #actions="{ row }">
					<el-tooltip :content="t('customer.edit')"><el-button link :icon="Edit" :disabled="!canUpdate" :aria-label="t('customer.edit')" @click="openEdit(row as CustomerRow)" /></el-tooltip>
					<el-button link :loading="checking === row.id" :disabled="!canSales" :aria-label="t('customer.sales.check')" @click="checkSales(row as CustomerRow)">{{ t("customer.sales.short") }}</el-button>
					<el-tooltip :content="t('customer.disable.title')"><el-button link type="danger" :icon="ShieldOff" :disabled="!canDisable || row.status === 'disabled'" :aria-label="t('customer.disable.title')" @click="openDisable(row as CustomerRow)" /></el-tooltip>
				</template>
				<template #pagination>
					<el-pagination
						:current-page="currentPage"
						:page-size="pageSize"
						:page-sizes="[25, 50, 100]"
						:total="total"
						layout="total, sizes, prev, pager, next"
						@current-change="changePage"
						@size-change="changePageSize"
					/>
				</template>
			</DataTable>

			<DetailDrawer v-model="drawerOpen" :title="editing ? t('customer.edit') : t('customer.new')" size="48%">
				<el-skeleton v-if="writeScopeLoading" :rows="7" animated />
				<StateBlock v-else-if="writeScopeError" type="error" :title="t('dataScope.loadFailed')" :description="writeScopeError">
					<template #actions><el-button type="primary" @click="loadWriteScope(writeScopeAction)">{{ t("common.retry") }}</el-button></template>
				</StateBlock>
				<el-form v-else label-position="top">
					<DataScopeIndicator :decision="writeScope" />
					<el-alert v-if="writeScope && !canSaveTarget" class="target-warning" type="warning" :title="t('dataScope.targetDenied')" show-icon :closable="false" />
					<el-form-item :label="t('customer.code')" required>
						<el-input v-model="form.code" />
					</el-form-item>
					<el-form-item :label="t('customer.name')" required>
					<el-input v-model="form.name" />
				</el-form-item>
				<div class="form-grid">
						<el-form-item :label="t('customer.region')" required>
							<el-select v-model="form.region">
								<el-option :label="t('customer.region.east')" value="East" />
								<el-option :label="t('customer.region.south')" value="South" />
								<el-option :label="t('customer.region.west')" value="West" />
								<el-option :label="t('customer.region.north')" value="North" />
							</el-select>
						</el-form-item>
						<el-form-item :label="t('customer.rating')">
						<el-input-number v-model="form.rating" :min="1" :max="5" />
					</el-form-item>
				</div>
				<div class="form-grid">
						<el-form-item :label="t('customer.organization')" required>
							<el-select v-if="organizationRestricted" v-model="form.organizationId" filterable :placeholder="t('dataScope.selectOrganization')">
								<el-option v-for="id in allowedOrganizationIDs" :key="id" :label="id" :value="id" />
							</el-select>
							<el-input v-else v-model="form.organizationId" :disabled="ownerRestricted" />
						</el-form-item>
						<el-form-item :label="t('customer.owner')" required>
							<el-input v-model="form.ownerId" :disabled="ownerRestricted" />
					</el-form-item>
				</div>
				<div class="form-grid">
						<el-form-item :label="t('customer.contact')" required>
						<el-input v-model="form.contactName" />
					</el-form-item>
						<el-form-item :label="t('customer.phone')">
						<el-input v-model="form.contactPhone" />
					</el-form-item>
				</div>
					<el-form-item :label="t('customer.email')">
					<el-input v-model="form.contactEmail" />
				</el-form-item>
				<div class="form-grid">
						<el-form-item :label="t('customer.qualification')" required>
						<el-input v-model="form.qualificationName" />
					</el-form-item>
						<el-form-item :label="t('customer.qualificationNumber')">
						<el-input v-model="form.qualificationNumber" />
					</el-form-item>
				</div>
					<el-form-item :label="t('customer.expiresAt')" required>
					<el-date-picker v-model="form.qualificationExpiresAt" type="date" value-format="YYYY-MM-DD" />
				</el-form-item>
				<div class="form-grid">
						<el-form-item :label="t('customer.attachmentId')">
						<el-input v-model="form.attachmentFileId" />
					</el-form-item>
						<el-form-item :label="t('customer.attachmentName')">
						<el-input v-model="form.attachmentFileName" />
					</el-form-item>
				</div>
				</el-form>
				<template #footer>
					<el-button :disabled="saving" @click="drawerOpen = false">{{ t("common.cancel") }}</el-button>
					<el-button type="primary" :loading="saving" :disabled="writeScopeLoading || Boolean(writeScopeError) || !canSaveCustomer" @click="saveCustomer">{{ t("common.save") }}</el-button>
				</template>
			</DetailDrawer>

			<DetailDrawer v-model="disableDialogOpen" :title="t('customer.disable.title')" size="34%">
				<el-form label-position="top">
					<el-form-item :label="t('customer.disable.reason')">
					<el-input v-model="disableReason" type="textarea" :rows="3" />
				</el-form-item>
			</el-form>
			<template #footer>
					<el-button :disabled="saving" @click="disableDialogOpen = false">{{ t("common.cancel") }}</el-button>
					<ConfirmAction
						:label="t('customer.disable.confirm')"
						:message="t('customer.disable.message')"
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

.target-warning {
	margin: 12px 0;
}

@media (max-width: 720px) {
	.summary-grid,
	.form-grid {
		grid-template-columns: 1fr;
	}
}
</style>
