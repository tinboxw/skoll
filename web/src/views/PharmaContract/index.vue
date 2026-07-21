<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, reactive, ref } from "vue";
import { AlarmClock, Check, Eye, FileText, Plus, RefreshCw, Upload, X } from "lucide-vue-next";
import { ElMessage, ElMessageBox, type UploadFile } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { uploadFile } from "../../files/api";
import { useButtonAccess } from "../../permissions/button";
import { approveContract, createContract, listContracts, listCustomers, listSuppliers, rejectContract, scanContractExpiry, type ContractPartyType, type ContractStatus, type PharmaContract, type PharmaCustomer, type PharmaSupplier } from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

type ContractRow = Record<string, unknown> & PharmaContract & { partyText: string; amountText: string; expiryText: string };
const access = useButtonAccess();
const userStore = useUserStore();
const loading = ref(false), saving = ref(false), scanning = ref(false), actionId = ref("");
const error = ref(""), createOpen = ref(false), detailOpen = ref(false), keyword = ref("");
const partyFilter = ref<ContractPartyType | "">(""), statusFilter = ref<ContractStatus | "">("");
const items = ref<PharmaContract[]>([]), suppliers = ref<PharmaSupplier[]>([]), customers = ref<PharmaCustomer[]>([]), selected = ref<PharmaContract | null>(null);
const selectedFile = ref<File | null>(null), uploadedFileId = ref("");
const canRead = computed(() => access.can("pharma_oa.contract.read"));
const canCreate = computed(() => access.can("pharma_oa.contract.create"));
const canApprove = computed(() => access.can("pharma_oa.contract.approve"));
const canReject = computed(() => access.can("pharma_oa.contract.reject"));
const canScan = computed(() => access.can("pharma_oa.contract.expiry.run"));
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const form = reactive({ number: "", title: "", partyType: "supplier" as ContractPartyType, partyId: "", approverId: "", amount: 0, currency: "CNY", effectiveAt: "", expiresAt: "" });
const partyOptions = computed(() => form.partyType === "supplier" ? suppliers.value.filter((item) => item.status === "active") : customers.value.filter((item) => item.status === "active"));
const rows = computed<ContractRow[]>(() => items.value.map((item) => ({ ...item, partyText: `${valueLabel(item.partyType)}: ${item.partyName}`, amountText: `${item.currency} ${item.amount.toLocaleString()}`, expiryText: new Date(item.expiresAt).toLocaleDateString() })));
const columns: DataTableColumn[] = [{ key: "number", label: t("pharma.contract.contract"), minWidth: 150 }, { key: "title", label: t("pharma.contract.titleColumn"), minWidth: 220 }, { key: "partyText", label: t("pharma.contract.party"), minWidth: 220 }, { key: "amountText", label: t("pharma.contract.amount"), width: 150 }, { key: "expiryText", label: t("pharma.contract.expires"), width: 130 }, { key: "status", label: t("pharma.contract.status"), width: 150 }];
const pendingCount = computed(() => items.value.filter((item) => item.status === "pending_approval").length);
const expiringCount = computed(() => items.value.filter((item) => item.status !== "rejected" && new Date(item.expiresAt).getTime() <= Date.now() + 30 * 86400000).length);

onMounted(() => void refresh());

async function refresh() {
	if (!canRead.value) return;
	loading.value = true; error.value = "";
	try {
		const [contracts, supplierPage, customerPage] = await Promise.all([listContracts({ keyword: keyword.value, partyType: partyFilter.value, status: statusFilter.value }), listSuppliers({ status: "active", limit: 200 }), listCustomers({ status: "active", limit: 200 })]);
		items.value = contracts; suppliers.value = supplierPage.items; customers.value = customerPage.items;
		const target = new URLSearchParams(window.location.search).get("contractId");
		if (target) { const item = contracts.find((entry) => entry.id === target); if (item) openDetail(item); }
	} catch (cause) { error.value = toErrorMessage(cause); }
	finally { loading.value = false; }
}

function openCreate() {
	const today = new Date();
	const nextYear = new Date(today); nextYear.setFullYear(today.getFullYear() + 1);
	Object.assign(form, { number: "", title: "", partyType: "supplier", partyId: "", approverId: "", amount: 0, currency: "CNY", effectiveAt: today.toISOString().slice(0, 10), expiresAt: nextYear.toISOString().slice(0, 10) });
	selectedFile.value = null; uploadedFileId.value = ""; createOpen.value = true;
}

function onPartyTypeChange() { form.partyId = ""; }
function onFileSelected(file: UploadFile) { selectedFile.value = file.raw || null; uploadedFileId.value = ""; }

async function save() {
	if (!canCreate.value || !form.number.trim() || !form.title.trim() || !form.partyId || !form.approverId.trim() || !form.effectiveAt || !form.expiresAt || !selectedFile.value) return;
	saving.value = true; error.value = "";
	try {
		if (!uploadedFileId.value) {
			const file = await uploadFile({ file: selectedFile.value, name: selectedFile.value.name, visibility: "private", ownerType: "user", ownerId: actorId.value, sourceModule: "pharma_oa", sourcePluginId: "pharma_oa", metadata: { resource: "contract", contractNumber: form.number.trim() } });
			uploadedFileId.value = file.id;
		}
		const item = await createContract({ ...form, number: form.number.trim(), title: form.title.trim(), ownerId: actorId.value, approverId: form.approverId.trim(), amount: Number(form.amount), attachmentIds: [uploadedFileId.value] });
		items.value.unshift(item); createOpen.value = false; ElMessage.success(t("pharma.contract.sentForApproval"));
	} catch (cause) { error.value = toErrorMessage(cause); }
	finally { saving.value = false; }
}

function openDetail(item: PharmaContract) { selected.value = item; detailOpen.value = true; }
function replaceItem(item: PharmaContract) { items.value = items.value.map((current) => current.id === item.id ? item : current); if (selected.value?.id === item.id) selected.value = item; }

async function approve(item: PharmaContract) {
	if (!canApprove.value || item.status !== "pending_approval") return;
	await ElMessageBox.confirm(`${t("pharma.contract.approveContract")}: ${item.number}?`, t("pharma.contract.approveContract"), { type: "warning", confirmButtonText: t("pharma.contract.approve"), cancelButtonText: t("common.cancel") });
	actionId.value = item.id; error.value = "";
	try { replaceItem(await approveContract(item.id, actorId.value, "Approved in contract archive")); ElMessage.success(t("pharma.contract.approved")); }
	catch (cause) { error.value = toErrorMessage(cause); }
	finally { actionId.value = ""; }
}

async function reject(item: PharmaContract) {
	if (!canReject.value || item.status !== "pending_approval") return;
	const result = await ElMessageBox.prompt(t("pharma.contract.recordTheRejectionReason"), `${t("pharma.contract.reject")}: ${item.number}`, { type: "warning", confirmButtonText: t("pharma.contract.reject"), cancelButtonText: t("common.cancel"), inputValidator: (value) => Boolean(value.trim()) || t("pharma.common.reasonRequired") });
	actionId.value = item.id; error.value = "";
	try { replaceItem(await rejectContract(item.id, actorId.value, result.value)); ElMessage.success(t("pharma.contract.rejected")); }
	catch (cause) { error.value = toErrorMessage(cause); }
	finally { actionId.value = ""; }
}

async function scanExpiry() {
	if (!canScan.value) return;
	await ElMessageBox.confirm(t("pharma.contract.scanActiveContractsExpiringWithin30DaysAndCreateReminders"), t("pharma.contract.runExpiryScan"), { type: "warning", confirmButtonText: t("pharma.contract.runExpiryScan"), cancelButtonText: t("common.cancel") });
	scanning.value = true; error.value = "";
	try { const result = await scanContractExpiry(30, actorId.value); await refresh(); ElMessage.success(`${result.createdCount} ${t("pharma.common.createdReminders")}`); }
	catch (cause) { error.value = toErrorMessage(cause); }
	finally { scanning.value = false; }
}

function statusType(status: ContractStatus): "success" | "warning" | "danger" | "info" { return status === "active" ? "success" : status === "pending_approval" ? "warning" : status === "expired" ? "danger" : "info"; }
function formatBytes(size: number) { return size < 1024 ? `${size} B` : `${(size / 1024).toFixed(1)} KB`; }
</script>

<template>
	<PageShell :title="t('pharma.contract.title')" :description="t('pharma.contract.description')" :loading="loading" :error="error" :forbidden="!canRead" :forbidden-title="t('pharma.contract.noContractAccess')" :forbidden-description="t('pharma.contract.thisPageRequiresPharmaOaContractReadPermission')">
		<template #actions>
			<el-input v-model="keyword" clearable :placeholder="t('pharma.contract.search')" class="filter-control" @keyup.enter="refresh" />
			<el-select v-model="partyFilter" clearable :placeholder="t('pharma.contract.allParties')" class="filter-control" @change="refresh"><el-option :label="t('pharma.contract.suppliers')" value="supplier" /><el-option :label="t('pharma.contract.customers')" value="customer" /></el-select>
			<el-select v-model="statusFilter" clearable :placeholder="t('pharma.contract.allStatuses')" class="filter-control" @change="refresh"><el-option :label="t('pharma.contract.pendingApproval')" value="pending_approval" /><el-option :label="t('pharma.contract.active')" value="active" /><el-option :label="t('pharma.contract.statusRejected')" value="rejected" /><el-option :label="t('pharma.contract.expired')" value="expired" /></el-select>
			<el-tooltip :content="t('pharma.contract.refresh')"><el-button :icon="RefreshCw" circle :aria-label="t('pharma.contract.refresh')" :loading="loading" @click="refresh" /></el-tooltip>
			<el-button :icon="AlarmClock" :loading="scanning" :disabled="!canScan" @click="scanExpiry">{{ t("pharma.contract.expiryScan") }}</el-button>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">{{ t("pharma.contract.newContract") }}</el-button>
		</template>
		<section class="summary-band"><div><span>{{ t("pharma.contract.total") }}</span><strong>{{ items.length }}</strong></div><div><span>{{ t("pharma.contract.pending") }}</span><strong>{{ pendingCount }}</strong></div><div><span>{{ t("pharma.contract.dueIn30Days") }}</span><strong>{{ expiringCount }}</strong></div></section>
		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.contract.empty')">
			<template #cell-number="{ row }"><strong>{{ row.number }}</strong></template>
			<template #cell-status="{ row }"><el-tag :type="statusType(row.status as ContractStatus)">{{ valueLabel(row.status) }}</el-tag></template>
			<template #actions="{ row }"><el-tooltip :content="t('pharma.contract.viewDetails')"><el-button circle :icon="Eye" :aria-label="t('pharma.contract.viewDetails')" @click="openDetail(row as PharmaContract)" /></el-tooltip><el-tooltip v-if="row.status === 'pending_approval'" :content="t('pharma.contract.approve')"><el-button circle type="success" :icon="Check" :aria-label="t('pharma.contract.approve')" :loading="actionId === row.id" :disabled="!canApprove" @click="approve(row as PharmaContract)" /></el-tooltip><el-tooltip v-if="row.status === 'pending_approval'" :content="t('pharma.contract.reject')"><el-button circle type="danger" :icon="X" :aria-label="t('pharma.contract.reject')" :loading="actionId === row.id" :disabled="!canReject" @click="reject(row as PharmaContract)" /></el-tooltip></template>
		</DataTable>

		<DetailDrawer v-model="createOpen" :title="t('pharma.contract.newContract')" size="52%">
			<el-form label-position="top">
				<div class="form-grid"><el-form-item :label="t('pharma.contract.contractNumber')" required><el-input v-model="form.number" maxlength="64" /></el-form-item><el-form-item :label="t('pharma.contract.titleColumn')" required><el-input v-model="form.title" maxlength="160" /></el-form-item></div>
				<div class="form-grid"><el-form-item :label="t('pharma.contract.partyType')" required><el-segmented v-model="form.partyType" :options="[{ label: valueLabel('supplier'), value: 'supplier' }, { label: valueLabel('customer'), value: 'customer' }]" @change="onPartyTypeChange" /></el-form-item><el-form-item :label="t('pharma.contract.party')" required><el-select v-model="form.partyId" filterable :placeholder="t('pharma.contract.selectParty')"><el-option v-for="party in partyOptions" :key="party.id" :label="`${party.code} · ${party.name}`" :value="party.id" /></el-select></el-form-item></div>
				<div class="form-grid"><el-form-item :label="t('pharma.contract.approverUserId')" required><el-input v-model="form.approverId" /></el-form-item><el-form-item :label="t('pharma.contract.amount')" required><el-input-number v-model="form.amount" :min="0" :precision="2" controls-position="right" /></el-form-item></div>
				<div class="form-grid"><el-form-item :label="t('pharma.contract.currency')" required><el-select v-model="form.currency"><el-option :label="t('pharma.contract.cny')" value="CNY" /><el-option :label="t('pharma.contract.usd')" value="USD" /><el-option :label="t('pharma.contract.eur')" value="EUR" /></el-select></el-form-item><el-form-item :label="t('pharma.contract.effectiveDate')" required><el-date-picker v-model="form.effectiveAt" type="date" value-format="YYYY-MM-DD" /></el-form-item><el-form-item :label="t('pharma.contract.expiryDate')" required><el-date-picker v-model="form.expiresAt" type="date" value-format="YYYY-MM-DD" /></el-form-item></div>
				<el-form-item :label="t('pharma.contract.signedContractFile')" required><el-upload :auto-upload="false" :limit="1" :on-change="onFileSelected" accept=".pdf,.doc,.docx,.png,.jpg,.jpeg"><el-button :icon="Upload">{{ t("pharma.contract.selectFile") }}</el-button></el-upload></el-form-item>
			</el-form>
			<template #footer><el-button :disabled="saving" @click="createOpen = false">{{ t("pharma.contract.cancel") }}</el-button><el-button type="primary" :loading="saving" :disabled="!form.number.trim() || !form.title.trim() || !form.partyId || !form.approverId.trim() || !form.effectiveAt || !form.expiresAt || !selectedFile" @click="save">{{ t("pharma.contract.createAndSubmit") }}</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="selected ? `${selected.number} · ${selected.title}` : t('pharma.contract.title')" size="50%">
			<template v-if="selected"><div class="detail-status"><el-tag :type="statusType(selected.status)">{{ valueLabel(selected.status) }}</el-tag><span>{{ selected.partyName }}</span></div><dl class="detail-grid"><div><dt>{{ t("pharma.contract.party") }}</dt><dd>{{ valueLabel(selected.partyType) }} · {{ selected.partyId }}</dd></div><div><dt>{{ t("pharma.contract.amount") }}</dt><dd>{{ selected.currency }} {{ selected.amount.toLocaleString() }}</dd></div><div><dt>{{ t("pharma.contract.effective") }}</dt><dd>{{ new Date(selected.effectiveAt).toLocaleDateString() }}</dd></div><div><dt>{{ t("pharma.contract.expires") }}</dt><dd>{{ new Date(selected.expiresAt).toLocaleDateString() }}</dd></div><div><dt>{{ t("pharma.contract.owner") }}</dt><dd>{{ selected.ownerId }}</dd></div><div><dt>{{ t("pharma.contract.approver") }}</dt><dd>{{ selected.approverId }}</dd></div><div><dt>{{ t("pharma.contract.workflow") }}</dt><dd>{{ selected.workflowInstanceId }}</dd></div><div><dt>{{ t("pharma.contract.reminder") }}</dt><dd>{{ selected.reminderNotificationId || t('pharma.contract.notCreated') }}</dd></div></dl><section class="files"><h3>{{ t("pharma.contract.files") }}</h3><div v-for="file in selected.attachments" :key="file.fileId" class="file-row"><FileText :size="18" /><div><strong>{{ file.fileName }}</strong><small>{{ file.mime }} · {{ formatBytes(file.size) }} · {{ file.fileId }}</small></div></div></section></template>
			<template #footer><el-button @click="detailOpen = false">{{ t("pharma.contract.close") }}</el-button><el-button v-if="selected?.status === 'pending_approval'" type="danger" :icon="X" :disabled="!canReject" :loading="actionId === selected.id" @click="reject(selected)">{{ t("pharma.contract.reject") }}</el-button><el-button v-if="selected?.status === 'pending_approval'" type="success" :icon="Check" :disabled="!canApprove" :loading="actionId === selected.id" @click="approve(selected)">{{ t("pharma.contract.approve") }}</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.filter-control{width:180px}.summary-band{display:flex;gap:32px;padding:4px 0 16px;border-bottom:1px solid var(--el-border-color-lighter)}.summary-band div{display:grid;gap:3px}.summary-band span{color:var(--el-text-color-secondary);font-size:13px}.summary-band strong{font-size:22px}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.detail-status{display:flex;align-items:center;gap:10px;margin-bottom:18px}.detail-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;margin:0}.detail-grid div{min-width:0}.detail-grid dt{color:var(--el-text-color-secondary);font-size:12px}.detail-grid dd{margin:5px 0 0;overflow-wrap:anywhere}.files{margin-top:24px;border-top:1px solid var(--el-border-color-lighter)}.files h3{font-size:15px}.file-row{display:flex;align-items:center;gap:10px;padding:10px 0}.file-row div{display:grid;min-width:0}.file-row small{color:var(--el-text-color-secondary);overflow-wrap:anywhere}@media(max-width:760px){.filter-control{width:100%}.summary-band{gap:18px;justify-content:space-between}.form-grid,.detail-grid{grid-template-columns:1fr}.summary-band strong{font-size:19px}}
</style>
